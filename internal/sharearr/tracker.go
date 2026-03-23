package sharearr

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/anacrolix/generics"
	"github.com/anacrolix/torrent/tracker"
	httpTrackerServer "github.com/anacrolix/torrent/tracker/http/server"
	trackerServer "github.com/anacrolix/torrent/tracker/server"
	"github.com/anacrolix/torrent/tracker/udp"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type DBTracker struct {
	peers *PeerService
}

func NewDBTracker(peers *PeerService) *DBTracker {
	return &DBTracker{peers: peers}
}

func NewDBTrackerFromDB(db *sqlx.DB) *DBTracker {
	return NewDBTracker(NewPeerService(NewPeerRepository(db)))
}

func (t *DBTracker) TrackAnnounce(ctx context.Context, req udp.AnnounceRequest, addr netip.AddrPort) error {
	u, ok := userFromContext(ctx)
	if !ok {
		return fmt.Errorf("missing user in context")
	}

	if req.Event == tracker.Stopped {
		return t.peers.Delete(ctx, InfoHash{req.InfoHash}, PeerID(req.PeerId))
	}

	return t.peers.Announce(ctx, PeerAnnouncement{
		UserID:     u.ID,
		IP:         addr.Addr().String(),
		Port:       req.Port,
		InfoHash:   InfoHash{req.InfoHash},
		PeerID:     PeerID(req.PeerId),
		Downloaded: req.Downloaded,
		Uploaded:   req.Uploaded,
		Left:       req.Left,
	})
}

func (t *DBTracker) GetPeers(ctx context.Context, infoHash trackerServer.InfoHash, opts trackerServer.GetPeersOpts, requester netip.AddrPort) trackerServer.ServerAnnounceResult {
	var maxCount uint
	if opts.MaxCount.Ok {
		maxCount = opts.MaxCount.Value
	}

	addrs, err := t.peers.ListAddrByInfoHash(ctx, InfoHash{infoHash}, maxCount)
	if err != nil {
		return trackerServer.ServerAnnounceResult{Err: err}
	}

	seeders, leechers, err := t.peers.CountByInfoHash(ctx, InfoHash{infoHash})
	if err != nil {
		return trackerServer.ServerAnnounceResult{Err: err}
	}

	result := make([]trackerServer.PeerInfo, len(addrs))
	for i, addr := range addrs {
		result[i] = trackerServer.PeerInfo{AnnounceAddr: addr}
	}

	return trackerServer.ServerAnnounceResult{
		Peers:    result,
		Seeders:  generics.Some(seeders),
		Leechers: generics.Some(leechers),
		Interval: generics.Some[int32](1800),
	}
}

func (t *DBTracker) Scrape(ctx context.Context, infoHashes []trackerServer.InfoHash) ([]udp.ScrapeInfohashResult, error) {
	ihs := make([]InfoHash, len(infoHashes))
	for i, ih := range infoHashes {
		ihs[i] = InfoHash{ih}
	}
	counts, err := t.peers.CountByInfoHashes(ctx, ihs)
	if err != nil {
		return nil, err
	}

	results := make([]udp.ScrapeInfohashResult, len(infoHashes))
	for i, ih := range ihs {
		if c, ok := counts[ih]; ok {
			results[i].Seeders = c.Seeders
			results[i].Leechers = c.Leechers
		}
	}
	return results, nil
}

type TrackerHandler struct {
	Announce gin.HandlerFunc
}

func NewTrackerHandler(announce gin.HandlerFunc) *TrackerHandler {
	return &TrackerHandler{Announce: announce}
}

func NewTrackerHandlerFromDB(db *sqlx.DB) *TrackerHandler {
	return NewTrackerHandler(
		gin.WrapH(
			&httpTrackerServer.Handler{
				Announce: &trackerServer.AnnounceHandler{
					AnnounceTracker: NewDBTrackerFromDB(db),
				},
			},
		),
	)
}
