package sharearr

import (
	"context"
	"database/sql/driver"
	"fmt"
	"net/netip"
	"time"

	"github.com/jmoiron/sqlx"
)

type PeerID [20]byte

func (p *PeerID) Scan(src any) error {
	b, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("peerid: expected []byte, got %T", src)
	}
	copy(p[:], b)
	return nil
}

func (p PeerID) Value() (driver.Value, error) {
	return p[:], nil
}

type PeerAnnouncement struct {
	UserID     int64     `db:"user_id"`
	IP         string    `db:"ip"`
	Port       uint16    `db:"port"`
	InfoHash   InfoHash  `db:"info_hash"`
	PeerID     PeerID    `db:"peer_id"`
	Downloaded int64     `db:"downloaded"`
	Uploaded   int64     `db:"uploaded"`
	Left       int64     `db:"left"`
	UpdatedAt  time.Time `db:"updated_at"`
}

type PeerRepository struct {
	db *sqlx.DB
}

func NewPeerRepository(db *sqlx.DB) *PeerRepository {
	return &PeerRepository{db: db}
}

func (r *PeerRepository) Announce(ctx context.Context, pa PeerAnnouncement, currentTime time.Time) error {
	pa.UpdatedAt = currentTime
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO peers (torrent_id, user_id, peer_id, ip, port, downloaded, uploaded, left)
		SELECT id, :user_id, :peer_id, :ip, :port, :downloaded, :uploaded, :left FROM torrents WHERE info_hash = :info_hash
		ON CONFLICT (torrent_id, peer_id) DO UPDATE SET
			user_id    = excluded.user_id,
			ip         = excluded.ip,
			port       = excluded.port,
			downloaded = excluded.downloaded,
			uploaded   = excluded.uploaded,
			left       = excluded.left,
			updated_at = :updated_at`,
		pa,
	)
	if err != nil {
		return fmt.Errorf("announce peer: %w", err)
	}
	return nil
}

func (r *PeerRepository) DeleteStale(ctx context.Context, currentTime time.Time, olderThan time.Duration) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM peers WHERE updated_at < ?`, currentTime.Add(olderThan))
	if err != nil {
		return fmt.Errorf("delete stale peers: %w", err)
	}
	return nil
}

func (r *PeerRepository) Delete(ctx context.Context, infoHash InfoHash, peerID PeerID) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM peers
		WHERE torrent_id = (SELECT id FROM torrents WHERE info_hash = ?)
		  AND peer_id = ?`,
		infoHash, peerID,
	)
	if err != nil {
		return fmt.Errorf("delete peer: %w", err)
	}
	return nil
}

func (r *PeerRepository) ListAddrByInfoHash(ctx context.Context, infoHash InfoHash, maxCount uint) ([]netip.AddrPort, error) {
	query := `
		SELECT p.ip, p.port
		FROM peers p
		JOIN torrents t ON t.id = p.torrent_id
		WHERE t.info_hash = ?`
	args := []any{infoHash}
	if maxCount > 0 {
		query += ` LIMIT ?`
		args = append(args, maxCount)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list peer addrs: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var addrs []netip.AddrPort
	for rows.Next() {
		var ip string
		var port int
		if err := rows.Scan(&ip, &port); err != nil {
			return nil, fmt.Errorf("scan peer addr: %w", err)
		}
		addr, err := netip.ParseAddr(ip)
		if err != nil {
			continue
		}
		addrs = append(addrs, netip.AddrPortFrom(addr, uint16(port))) //nolint:gosec // port range enforced by DB CHECK constraint
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate peer addrs: %w", err)
	}
	return addrs, nil
}

type PeerCounts struct {
	Seeders  int32
	Leechers int32
}

func (r *PeerRepository) CountByInfoHash(ctx context.Context, infoHash InfoHash) (seeders, leechers int32, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(CASE WHEN p.left = 0 THEN 1 END),
			COUNT(CASE WHEN p.left > 0 THEN 1 END)
		FROM peers p
		JOIN torrents t ON t.id = p.torrent_id
		WHERE t.info_hash = ?`,
		infoHash,
	).Scan(&seeders, &leechers)
	if err != nil {
		return 0, 0, fmt.Errorf("count peers: %w", err)
	}
	return seeders, leechers, nil
}

func (r *PeerRepository) CountByInfoHashes(ctx context.Context, infoHashes []InfoHash) (map[InfoHash]PeerCounts, error) {
	query, args, err := sqlx.In(`
		SELECT t.info_hash,
			COUNT(CASE WHEN p.left = 0 THEN 1 END),
			COUNT(CASE WHEN p.left > 0 THEN 1 END)
		FROM peers p
		JOIN torrents t ON t.id = p.torrent_id
		WHERE t.info_hash IN (?)
		GROUP BY t.info_hash`,
		infoHashes,
	)
	if err != nil {
		return nil, fmt.Errorf("build count by info hashes query: %w", err)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("count peers by info hashes: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	counts := make(map[InfoHash]PeerCounts, len(infoHashes))
	for rows.Next() {
		var ih InfoHash
		var c PeerCounts
		if err := rows.Scan(&ih, &c.Seeders, &c.Leechers); err != nil {
			return nil, fmt.Errorf("scan peer counts: %w", err)
		}
		counts[ih] = c
	}
	return counts, rows.Err()
}

type PeerService struct {
	repo *PeerRepository
}

func NewPeerService(repo *PeerRepository) *PeerService {
	return &PeerService{repo: repo}
}

func NewPeerServiceFromDB(db *sqlx.DB) *PeerService {
	return &PeerService{repo: NewPeerRepository(db)}
}

func (s *PeerService) Announce(ctx context.Context, pa PeerAnnouncement) error {
	return s.repo.Announce(ctx, pa, time.Now())
}

func (s *PeerService) Delete(ctx context.Context, infoHash InfoHash, peerID PeerID) error {
	return s.repo.Delete(ctx, infoHash, peerID)
}

func (s *PeerService) DeleteStale(ctx context.Context) error {
	return s.repo.DeleteStale(ctx, time.Now(), -1*time.Hour)
}

func (s *PeerService) CountByInfoHash(ctx context.Context, infoHash InfoHash) (seeders, leechers int32, err error) {
	return s.repo.CountByInfoHash(ctx, infoHash)
}

func (s *PeerService) CountByInfoHashes(ctx context.Context, infoHashes []InfoHash) (map[InfoHash]PeerCounts, error) {
	return s.repo.CountByInfoHashes(ctx, infoHashes)
}

func (s *PeerService) ListAddrByInfoHash(ctx context.Context, infoHash InfoHash, maxCount uint) ([]netip.AddrPort, error) {
	return s.repo.ListAddrByInfoHash(ctx, infoHash, maxCount)
}
