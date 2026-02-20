package main

import (
	"context"
	"net/netip"
	"sync"
	"time"

	"github.com/anacrolix/generics"
	"github.com/anacrolix/torrent/tracker"
	trackerServer "github.com/anacrolix/torrent/tracker/server"
	"github.com/anacrolix/torrent/tracker/udp"
)

type peerEntry struct {
	left     int64
	lastSeen time.Time
}

// MemTracker is an in-memory implementation of the AnnounceTracker interface.
type MemTracker struct {
	mu    sync.Mutex
	peers map[trackerServer.InfoHash]map[netip.AddrPort]peerEntry
}

func NewMemTracker() *MemTracker {
	return &MemTracker{
		peers: make(map[trackerServer.InfoHash]map[netip.AddrPort]peerEntry),
	}
}

func (t *MemTracker) TrackAnnounce(_ context.Context, req udp.AnnounceRequest, addr netip.AddrPort) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.peers[req.InfoHash] == nil {
		t.peers[req.InfoHash] = make(map[netip.AddrPort]peerEntry)
	}
	if req.Event == tracker.Stopped {
		delete(t.peers[req.InfoHash], addr)
	} else {
		t.peers[req.InfoHash][addr] = peerEntry{
			left:     req.Left,
			lastSeen: time.Now(),
		}
	}
	return nil
}

func (t *MemTracker) GetPeers(_ context.Context, infoHash trackerServer.InfoHash, opts trackerServer.GetPeersOpts, _ netip.AddrPort) trackerServer.ServerAnnounceResult {
	t.mu.Lock()
	defer t.mu.Unlock()
	swarm := t.peers[infoHash]
	var peers []trackerServer.PeerInfo
	var seeders, leechers int32
	for addr, entry := range swarm {
		if entry.left == 0 {
			seeders++
		} else {
			leechers++
		}
		if opts.MaxCount.Ok && uint(len(peers)) >= opts.MaxCount.Value {
			continue
		}
		peers = append(peers, trackerServer.PeerInfo{AnnounceAddr: addr})
	}
	return trackerServer.ServerAnnounceResult{
		Peers:    peers,
		Seeders:  generics.Some(seeders),
		Leechers: generics.Some(leechers),
		Interval: generics.Some[int32](1800),
	}
}

func (t *MemTracker) Scrape(_ context.Context, infoHashes []trackerServer.InfoHash) ([]udp.ScrapeInfohashResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	results := make([]udp.ScrapeInfohashResult, len(infoHashes))
	for i, ih := range infoHashes {
		for _, entry := range t.peers[ih] {
			if entry.left == 0 {
				results[i].Seeders++
			} else {
				results[i].Leechers++
			}
		}
	}
	return results, nil
}
