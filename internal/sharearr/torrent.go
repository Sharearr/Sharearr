package sharearr

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/types/infohash"
	"github.com/gin-gonic/gin"
	sqlite3 "github.com/mattn/go-sqlite3"
)

var ErrTorrentNotFound = errors.New("torrent not found")
var ErrTorrentAlreadyExists = errors.New("torrent already exists")
var ErrInvalidTorrent = errors.New("invalid torrent")

type Torrent struct {
	ID        int64
	InfoHash  infohash.T
	Name      string
	SizeBytes int64
	File      []byte
	UserID    int64
	CreatedAt time.Time
	UpdatedAt time.Time
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
	db *sql.DB
}

func NewTorrentRepository(db *sql.DB) *TorrentRepository {
	return &TorrentRepository{db: db}
}

func (r *TorrentRepository) Create(ctx context.Context, t *Torrent, categoryIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO torrents (info_hash, name, size_bytes, file, user_id)
		 VALUES (?, ?, ?, ?, ?)`,
		t.InfoHash.String(), t.Name, t.SizeBytes, t.File, t.UserID,
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

	for _, catID := range categoryIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO category_torrents (category_id, torrent_id) VALUES (?, ?)`,
			catID, t.ID,
		); err != nil {
			return fmt.Errorf("link category %d: %w", catID, err)
		}
	}

	return tx.Commit()
}

func (r *TorrentRepository) GetByInfoHash(ctx context.Context, ih infohash.T) (*Torrent, error) {
	t := &Torrent{}
	var ihStr string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, info_hash, name, size_bytes, file, user_id, created_at, updated_at
		 FROM torrents WHERE info_hash = ?`, ih.String(),
	).Scan(&t.ID, &ihStr, &t.Name, &t.SizeBytes, &t.File, &t.UserID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get torrent by info_hash: %w", err)
	}
	t.InfoHash = infohash.FromHexString(ihStr)
	return t, nil
}

func (r *TorrentRepository) GetByID(ctx context.Context, id int64) (*Torrent, error) {
	t := &Torrent{}
	var ihStr string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, info_hash, name, size_bytes, file, user_id, created_at, updated_at
		 FROM torrents WHERE id = ?`, id,
	).Scan(&t.ID, &ihStr, &t.Name, &t.SizeBytes, &t.File, &t.UserID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get torrent by id: %w", err)
	}
	t.InfoHash = infohash.FromHexString(ihStr)
	return t, nil
}

type TorrentSearch struct {
	Query       string
	CategoryIDs []int
	Extended    bool
	Limit       int
	Offset      int
	*TorrentTvSearch
}

type TorrentTvSearch struct {
	Season  string
	Episode string
}

func (ts *TorrentTvSearch) normalizedTVParams() string {
	s := normalizeTVParam(ts.Season, "S")
	e := normalizeTVParam(ts.Episode, "E")
	switch {
	case s != "" && e != "":
		return s + e
	case s != "":
		return s
	case e != "":
		return e
	}
	return ""
}

func normalizeTVParam(val, prefix string) string {
	val = strings.TrimPrefix(strings.ToUpper(val), prefix)
	n, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s%02d", prefix, n)
}

func (r *TorrentRepository) Search(ctx context.Context, ts TorrentSearch) ([]TorrentCategory, error) {
	var args []any
	builder := strings.Builder{}

	builder.WriteString("SELECT t.id, t.info_hash, t.name, t.size_bytes, t.user_id, t.created_at, t.updated_at ")
	catSubquery := "t.id IN (SELECT torrent_id FROM category_torrents WHERE category_id IN (" + placeholders(len(ts.CategoryIDs)) + "))"

	if ts.Query != "" || ts.TorrentTvSearch != nil {
		builder.WriteString(
			`FROM torrents_fts
		 	 JOIN torrents t ON t.id = torrents_fts.rowid `,
		)
		var terms []string
		terms = append(terms, ts.Query)
		if ts.TorrentTvSearch != nil {
			terms = append(terms, ts.normalizedTVParams())
		}
		builder.WriteString("WHERE torrents_fts MATCH ? ")
		args = append(args, strings.Join(terms, " "))
		if len(ts.CategoryIDs) > 0 {
			builder.WriteString("AND " + catSubquery + " ")
			for _, id := range ts.CategoryIDs {
				args = append(args, id)
			}
		}
	} else {
		builder.WriteString("FROM torrents t ")
		if len(ts.CategoryIDs) > 0 {
			builder.WriteString("WHERE " + catSubquery + " ")
			for _, id := range ts.CategoryIDs {
				args = append(args, id)
			}
		}
	}
	if ts.Limit > 0 {
		builder.WriteString("LIMIT ? ")
		args = append(args, ts.Limit)
	}
	if ts.Offset > 0 {
		builder.WriteString("OFFSET ? ")
		args = append(args, ts.Offset)
	}
	rows, err := r.db.QueryContext(ctx, builder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("search torrents: %w", err)
	}
	defer rows.Close()

	var results []TorrentCategory
	for rows.Next() {
		var tc TorrentCategory
		var ihStr string
		if err := rows.Scan(&tc.ID, &ihStr, &tc.Name, &tc.SizeBytes, &tc.UserID, &tc.CreatedAt, &tc.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan torrent: %w", err)
		}
		tc.InfoHash = infohash.FromHexString(ihStr)
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
		InfoHash:  mi.HashInfoBytes(),
		Name:      info.BestName(),
		SizeBytes: info.TotalLength(),
		File:      buf.Bytes(),
		UserID:    userID,
	}

	if err := s.repo.Create(ctx, t, categoryIDs); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TorrentService) GetByInfoHash(ctx context.Context, infoHash infohash.T) (*Torrent, error) {
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

func NewTorrentHandlerFromDB(db *sql.DB) *TorrentHandler {
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
	announceURL := fmt.Sprintf("%s://%s/announce/%s", requestScheme(c), c.Request.Host, u.APIKey)

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
	defer f.Close()

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
