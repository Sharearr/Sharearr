package sharearr

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/types/infohash"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/mattn/go-sqlite3"
)

type InfoHash struct{ infohash.T }

func (ih *InfoHash) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("infohash: expected string, got %T", src)
	}
	ih.T = infohash.FromHexString(s)
	return nil
}

func (ih InfoHash) Value() (driver.Value, error) {
	return ih.T.String(), nil
}

func (ih InfoHash) String() string {
	return ih.T.String()
}

var ErrTorrentNotFound = errors.New("torrent not found")
var ErrTorrentAlreadyExists = errors.New("torrent already exists")
var ErrInvalidTorrent = errors.New("invalid torrent")

type Torrent struct {
	ID        int64         `db:"id"`
	InfoHash  InfoHash      `db:"info_hash"`
	Name      string        `db:"name"`
	SizeBytes int64         `db:"size_bytes"`
	File      []byte        `db:"file"`
	UserID    sql.NullInt64 `db:"user_id"`
	CreatedAt time.Time     `db:"created_at"`
	UpdatedAt time.Time     `db:"updated_at"`
}

type torrentResponse struct {
	ID        int64     `json:"id"`
	InfoHash  string    `json:"info_hash"`
	Name      string    `json:"name"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

func newTorrentResponse(t *Torrent) torrentResponse {
	return torrentResponse{
		ID:        t.ID,
		InfoHash:  t.InfoHash.String(),
		Name:      t.Name,
		SizeBytes: t.SizeBytes,
		CreatedAt: t.CreatedAt,
	}
}

type TorrentRepository struct {
	db *sqlx.DB
}

func NewTorrentRepository(db *sqlx.DB) *TorrentRepository {
	return &TorrentRepository{db: db}
}

func (r *TorrentRepository) Create(ctx context.Context, t *Torrent, categoryIDs []int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	res, err := tx.ExecContext(ctx,
		`INSERT INTO torrents (info_hash, name, size_bytes, file, user_id)
		 VALUES (?, ?, ?, ?, ?)`,
		t.InfoHash, t.Name, t.SizeBytes, t.File, t.UserID,
	)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return ErrTorrentAlreadyExists
		}
		return fmt.Errorf("create torrent: %w", err)
	}
	t.ID, err = res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last insert id: %w", err)
	}

	if len(categoryIDs) > 0 {
		q, args, err := sqlx.In(
			`INSERT INTO category_torrents (category_id, torrent_id)
			 SELECT id, ? FROM categories WHERE id IN (?)`,
			t.ID, categoryIDs,
		)
		if err != nil {
			return fmt.Errorf("build category link query: %w", err)
		}
		if _, err := tx.ExecContext(ctx, q, args...); err != nil {
			return fmt.Errorf("link categories: %w", err)
		}
	}

	return tx.Commit()
}

func (r *TorrentRepository) GetByInfoHash(ctx context.Context, ih InfoHash) (*Torrent, error) {
	t := &Torrent{}
	err := r.db.GetContext(ctx, t,
		`SELECT id, info_hash, name, size_bytes, file, user_id, created_at, updated_at
		 FROM torrents WHERE info_hash = ?`, ih,
	)
	if err != nil {
		return nil, fmt.Errorf("get torrent by info_hash: %w", err)
	}
	return t, nil
}

func (r *TorrentRepository) GetByID(ctx context.Context, id int64) (*Torrent, error) {
	t := &Torrent{}
	err := r.db.GetContext(ctx, t,
		`SELECT id, info_hash, name, size_bytes, file, user_id, created_at, updated_at
		 FROM torrents WHERE id = ?`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("get torrent by id: %w", err)
	}
	return t, nil
}

type TorrentSearch struct {
	Query       string
	CategoryIDs []int64
	Extended    bool
	Limit       int
	Offset      int
	Season      string
	Episode     string
}

func normalizedTVParams(season, episode string) string {
	return normalizeTVParam(season, "S") + normalizeTVParam(episode, "E")
}

func normalizeTVParam(val, prefix string) string {
	val = strings.TrimPrefix(strings.ToUpper(val), prefix)
	n, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s%02d", prefix, n)
}

func sanitizeFtsQuery(query string) string {
	return strings.Join(strings.FieldsFunc(query, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), " ")
}

func (r *TorrentRepository) Search(ctx context.Context, ts TorrentSearch) ([]TorrentCategory, error) {
	var args []any
	var where []string

	from := "FROM torrents t "
	if ts.Query != "" || ts.Season != "" || ts.Episode != "" {
		from = "FROM torrents_fts JOIN torrents t ON t.id = torrents_fts.rowid "
		term := sanitizeFtsQuery(ts.Query) + " " + normalizedTVParams(ts.Season, ts.Episode)
		where = append(where, "torrents_fts MATCH ?")
		args = append(args, term)
	}

	if len(ts.CategoryIDs) > 0 {
		where = append(where, "t.id IN (SELECT torrent_id FROM category_torrents WHERE category_id IN (?))")
		args = append(args, ts.CategoryIDs)
	}

	q := "SELECT t.id, t.info_hash, t.name, t.size_bytes, t.user_id, t.created_at, t.updated_at " + from
	if len(where) > 0 {
		q += "WHERE " + strings.Join(where, " AND ") + " "
	}
	if ts.Limit > 0 {
		q += "LIMIT ? "
		args = append(args, ts.Limit)
	}
	if ts.Offset > 0 {
		q += "OFFSET ? "
		args = append(args, ts.Offset)
	}

	q, args, err := sqlx.In(q, args...)
	if err != nil {
		return nil, fmt.Errorf("build search query: %w", err)
	}
	rows, err := r.db.QueryxContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("search torrents: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var results []TorrentCategory
	for rows.Next() {
		var tc TorrentCategory
		if err := rows.StructScan(&tc); err != nil {
			return nil, fmt.Errorf("scan torrent: %w", err)
		}
		results = append(results, tc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate torrents: %w", err)
	}
	return results, nil
}

type TorrentService struct {
	repo *TorrentRepository
}

func NewTorrentService(repo *TorrentRepository) *TorrentService {
	return &TorrentService{repo: repo}
}

func (s *TorrentService) Create(ctx context.Context, userID int64, categoryIDs []int64, file []byte) (*Torrent, error) {
	mi, err := metainfo.Load(bytes.NewReader(file))
	if err != nil {
		return nil, ErrInvalidTorrent
	}

	info, err := mi.UnmarshalInfo()
	if err != nil {
		return nil, ErrInvalidTorrent
	}

	private := true
	info.Private = &private
	mi.InfoBytes, err = bencode.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("marshal info: %w", err)
	}

	mi.Announce = ""
	mi.AnnounceList = nil

	var buf bytes.Buffer
	if err := mi.Write(&buf); err != nil {
		return nil, fmt.Errorf("encode torrent: %w", err)
	}

	t := &Torrent{
		InfoHash:  InfoHash{mi.HashInfoBytes()},
		Name:      info.BestName(),
		SizeBytes: info.TotalLength(),
		File:      buf.Bytes(),
		UserID:    sql.NullInt64{Int64: userID, Valid: true},
	}

	if err := s.repo.Create(ctx, t, categoryIDs); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TorrentService) GetByInfoHash(ctx context.Context, infoHash InfoHash) (*Torrent, error) {
	t, err := s.repo.GetByInfoHash(ctx, infoHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTorrentNotFound
	}
	return t, err
}

func (s *TorrentService) GetByID(ctx context.Context, id int64) (*Torrent, error) {
	t, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTorrentNotFound
	}
	return t, err
}

func (s *TorrentService) Search(ctx context.Context, ts TorrentSearch) ([]TorrentCategory, error) {
	return s.repo.Search(ctx, ts)
}

type TorrentHandler struct {
	torrent  *TorrentService
	category *CategoryService
}

func NewTorrentHandler(torrent *TorrentService, category *CategoryService) *TorrentHandler {
	return &TorrentHandler{torrent: torrent, category: category}
}

func NewTorrentHandlerFromDB(db *sqlx.DB) *TorrentHandler {
	return NewTorrentHandler(NewTorrentService(NewTorrentRepository(db)), NewCategoryServiceFromDB(db))
}

func (h *TorrentHandler) Download(c *gin.Context) {
	var p IDParam
	if err := c.ShouldBindUri(&p); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	t, err := h.torrent.GetByID(c.Request.Context(), p.ID)
	if errors.Is(err, ErrTorrentNotFound) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	u, _ := userFromContext(c.Request.Context())
	announceURL := fmt.Sprintf("%s://%s/announce/%s", requestScheme(c), c.Request.Host, u.ApiKey)

	mi, err := metainfo.Load(bytes.NewReader(t.File))
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	mi.Announce = announceURL

	var buf bytes.Buffer
	if err := mi.Write(&buf); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.torrent"`, t.Name))
	c.Data(http.StatusOK, "application/x-bittorrent", buf.Bytes())
}

func (h *TorrentHandler) Upload(c *gin.Context) {
	var p CreateTorrent
	if err := c.ShouldBind(&p); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	f, err := p.File.Open()
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	defer f.Close() //nolint:errcheck

	fileBytes, err := io.ReadAll(f)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	categoryIDs := p.CategoryIDs

	if catName := c.Param("cat"); catName != "" {
		cat, err := h.category.GetRootByName(c.Request.Context(), catName)
		if errors.Is(err, ErrCategoryNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		categoryIDs = append(categoryIDs, cat.ID)
	}

	u, _ := userFromContext(c.Request.Context())

	t, err := h.torrent.Create(c.Request.Context(), u.ID, categoryIDs, fileBytes)
	if errors.Is(err, ErrInvalidTorrent) {
		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}
	if errors.Is(err, ErrTorrentAlreadyExists) {
		c.AbortWithStatus(http.StatusConflict)
		return
	}
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, newTorrentResponse(t))
}
