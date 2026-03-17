package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/anacrolix/torrent/types/infohash"
	"github.com/gin-gonic/gin"
)

const torznabMIME = "application/rss+xml; charset=utf-8"

var capsTmpl = template.Must(template.New("caps").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<caps>
  <limits default="100" max="100"/>
  <searching>
    <search available="yes" supportedParams="q"/>
    <tv-search available="yes" supportedParams="q,season,ep"/>
    <movie-search available="no" supportedParams=""/>
  </searching>
  <categories>
{{- range .}}
    <category id="{{.ID}}" name="{{.Name}}">
{{- range .Subcategories}}
      <subcat id="{{.ID}}" name="{{.Name}}"/>
{{- end}}
    </category>
{{- end}}
  </categories>
</caps>`))

var searchTmpl = template.Must(template.New("search").Funcs(template.FuncMap{
	"xmlescape": func(s string) string {
		var buf bytes.Buffer
		xml.EscapeText(&buf, []byte(s))
		return buf.String()
	},
}).Parse(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:torznab="http://torznab.com/schemas/2015/feed">
  <channel>
    <title>sharearr</title>
{{- range .Items}}
    <item>
      <title>{{xmlescape .Name}}</title>
      <guid>{{.GUID}}</guid>
      <link>{{.Link}}</link>
      <enclosure url="{{.Link}}" length="{{.SizeBytes}}" type="application/x-bittorrent" />
      <pubDate>{{.PubDate}}</pubDate>
{{- if $.Extended}}
      <torznab:attr name="infohash" value="{{.InfoHash}}"/>
      <torznab:attr name="size" value="{{.SizeBytes}}"/>
      <torznab:attr name="seeders" value="{{.Seeders}}"/>
      <torznab:attr name="leechers" value="{{.Leechers}}"/>
{{- range .Categories}}
      <torznab:attr name="category" value="{{.ID}}"/>
{{- end}}
{{- end}}
    </item>
{{- end}}
  </channel>
</rss>`))

type TorrentCategory struct {
	Torrent
	Categories []Category
	PeerCounts
}

type torznabItem struct {
	Name       string
	GUID       string
	Link       string
	SizeBytes  int64
	PubDate    string
	InfoHash   string
	Categories []Category
	PeerCounts
}

type torznabResponse struct {
	Items    []torznabItem
	Extended bool
}

type TorznabService struct {
	db       *sql.DB
	torrents *TorrentService
	peers    *PeerService
}

func NewTorznabService(db *sql.DB, torrents *TorrentService, peers *PeerService) *TorznabService {
	return &TorznabService{db: db, torrents: torrents, peers: peers}
}

func (s *TorznabService) Search(ctx context.Context, p TorznabQuery) ([]TorrentCategory, error) {
	ts := TorrentSearch{
		Query:       p.Q,
		CategoryIDs: p.Cat,
		Limit:       p.Limit,
		Offset:      p.Offset,
	}
	if p.TorznabTvQuery != nil {
		ts.TorrentTvSearch = &TorrentTvSearch{Season: p.Season, Episode: p.Episode}
	}
	results, err := s.torrents.Search(ctx, ts)
	if err != nil || !p.Extended || len(results) == 0 {
		return results, err
	}

	ids := make([]any, len(results))
	for i, tc := range results {
		ids[i] = tc.ID
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT ct.torrent_id, c.id, c.name, c.parent_id, c.created_at, c.updated_at
		 FROM category_torrents ct
		 JOIN categories c ON c.id = ct.category_id
		 WHERE ct.torrent_id IN (`+placeholders(len(ids))+`)`,
		ids...,
	)
	if err != nil {
		return nil, fmt.Errorf("lookup categories: %w", err)
	}
	defer rows.Close()

	catsByTorrent := make(map[int64][]Category)
	for rows.Next() {
		var torrentID int64
		var c Category
		if err := rows.Scan(&torrentID, &c.ID, &c.Name, &c.ParentID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		catsByTorrent[torrentID] = append(catsByTorrent[torrentID], c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}

	for i := range results {
		results[i].Categories = catsByTorrent[results[i].ID]
	}

	ihs := make([]infohash.T, len(results))
	for i, tc := range results {
		ihs[i] = tc.InfoHash
	}
	peerCounts, err := s.peers.CountByInfoHashes(ctx, ihs)
	if err != nil {
		return nil, err
	}
	for i := range results {
		if c, ok := peerCounts[results[i].InfoHash]; ok {
			results[i].PeerCounts = c
		}
	}
	return results, nil
}

type TorznabHandler struct {
	service    *TorznabService
	categories *CategoryService
}

func NewTorznabHandler(service *TorznabService, categories *CategoryService) *TorznabHandler {
	return &TorznabHandler{service: service, categories: categories}
}

func NewTorznabHandlerFromDB(db *sql.DB) *TorznabHandler {
	return NewTorznabHandler(
		NewTorznabService(db, NewTorrentService(NewTorrentRepository(db)), NewPeerServiceFromDB(db)),
		NewCategoryServiceFromDB(db),
	)
}

func (h *TorznabHandler) Handle(c *gin.Context) {
	p := TorznabQuery{Limit: 100}
	if err := c.ShouldBindQuery(&p); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	switch p.Type {
	case "caps":
		h.caps(c)
	case "search", "tvsearch":
		h.search(c, p)
	default:
		c.Status(http.StatusBadRequest)
	}
}

func (h *TorznabHandler) caps(c *gin.Context) {
	tree, err := h.categories.ListTree(c.Request.Context())
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Header("Content-Type", torznabMIME)
	if err := capsTmpl.Execute(c.Writer, tree); err != nil {
		log.Printf("torznab: render caps: %v", err)
	}
}

func newTorznabItem(t TorrentCategory, base string) torznabItem {
	return torznabItem{
		Name:       t.Name,
		GUID:       fmt.Sprintf("%s/torrent/%d", base, t.ID),
		Link:       fmt.Sprintf("%s/torrent/%d/download", base, t.ID),
		SizeBytes:  t.SizeBytes,
		PubDate:    t.CreatedAt.UTC().Format(time.RFC1123Z),
		InfoHash:   t.InfoHash.String(),
		Categories: t.Categories,
		PeerCounts: t.PeerCounts,
	}
}

func (h *TorznabHandler) search(c *gin.Context, p TorznabQuery) {
	torrents, err := h.service.Search(c.Request.Context(), p)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	base := fmt.Sprintf("%s://%s", requestScheme(c), c.Request.Host)
	items := make([]torznabItem, len(torrents))
	for i, t := range torrents {
		items[i] = newTorznabItem(t, base)
	}

	h.renderResponse(c, torznabResponse{Items: items, Extended: p.Extended}, p.Type)
}

func (h *TorznabHandler) renderResponse(c *gin.Context, resp torznabResponse, logCtx string) {
	c.Header("Content-Type", torznabMIME)
	if err := searchTmpl.Execute(c.Writer, resp); err != nil {
		log.Printf("torznab: render %s: %v", logCtx, err)
	}
}

func requestScheme(c *gin.Context) string {
	if c.Request.TLS != nil {
		return "https"
	}
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	return "http"
}
