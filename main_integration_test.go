package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

type SonarrContainer struct {
	*testcontainers.DockerContainer
}

func NewSonarrContainer(ctx context.Context, networkName string) (*SonarrContainer, error) {
	ctr, err := testcontainers.Run(
		ctx, "lscr.io/linuxserver/sonarr:latest",
		testcontainers.WithFiles(
			testcontainers.ContainerFile{
				HostFilePath:      ".devcontainer/sonarr-config.xml",
				ContainerFilePath: "/defaults/config.xml",
				FileMode:          0o664,
			},
			testcontainers.ContainerFile{
				HostFilePath:      ".devcontainer/entrypoint.sh",
				ContainerFilePath: "/entrypoint.sh",
				FileMode:          0o755,
			},
		),
		testcontainers.WithEntrypoint("/entrypoint.sh"),
		testcontainers.WithEntrypointArgs("/defaults/config.xml", "/config/config.xml"),
		testcontainers.WithExposedPorts("8989/tcp"),
		testcontainers.WithWaitStrategy(
			wait.ForExec([]string{"curl", "-sf", "http://localhost:8989/ping"}),
		),
		testcontainers.CustomizeRequest(
			testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{networkName},
					NetworkAliases: map[string][]string{networkName: {"sonarr"}},
				},
			},
		),
	)
	if err != nil {
		return nil, err
	}
	return &SonarrContainer{DockerContainer: ctr}, nil
}

func (s *SonarrContainer) apiRequest(ctx context.Context, method string, path string, body io.Reader) (*http.Request, error) {
	endpoint, err := s.Endpoint(ctx, "http")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", endpoint, path), body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-Api-Key", "apikey")
	req.Header.Add("Content-Type", "application/json")
	return req, nil
}

func (s *SonarrContainer) findSchemaImplementation(ctx context.Context, schemaType string, implName string) (map[string]any, error) {
	req, err := s.apiRequest(ctx, "GET", fmt.Sprintf("/api/v3/%s/schema", schemaType), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	var schemas []map[string]any
	if err := json.Unmarshal(body, &schemas); err != nil {
		return nil, err
	}

	var result map[string]any
	for _, schema := range schemas {
		if schema["implementation"] == implName {
			result = schema
			break
		}
	}
	if result == nil {
		return nil, fmt.Errorf("No %s implementation found for %s", implName, schemaType)
	}
	return result, nil
}

func (s *SonarrContainer) AddIndexer(t *testing.T, ctx context.Context, name string, host string, port int) {
	t.Helper()
	schema, err := s.findSchemaImplementation(ctx, "indexer", "Torznab")
	require.NoError(t, err)

	schema["name"] = name
	schema["enable"] = true
	schema["enableRss"] = true
	schema["enableAutomaticSearch"] = true
	schema["enableInteractiveSearch"] = true

	fields, ok := schema["fields"].([]any)
	require.True(t, ok, "indexer schema missing fields")
	for _, f := range fields {
		field, ok := f.(map[string]any)
		if !ok {
			continue
		}
		switch field["name"] {
		case "baseUrl":
			field["value"] = fmt.Sprintf("http://%s:%d", host, port)
		case "apiPath":
			field["value"] = "/torznab"
		case "apiKey":
			field["value"] = "apikey"
		case "categories":
			field["value"] = []int{5000, 5030, 5040}
		}
	}

	payload, err := json.Marshal(schema)
	require.NoError(t, err)

	req, err := s.apiRequest(ctx, "POST", "/api/v3/indexer", bytes.NewReader(payload))
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Condition(t, func() bool {
		return resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK
	}, "unexpected status code: %d", resp.StatusCode)
}

func (s *SonarrContainer) Release(t *testing.T, ctx context.Context, seriesId int, season int) []map[string]any {
	t.Helper()
	req, err := s.apiRequest(ctx, "GET", fmt.Sprintf("/api/v3/release?seriesId=%d&seasonNumber=%d", seriesId, season), nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	var results []map[string]any
	require.NoError(t, json.Unmarshal(body, &results))
	return results
}

func (s *SonarrContainer) AddSeries(t *testing.T, ctx context.Context, tvdbId int, title string) int {
	t.Helper()
	body := map[string]any{
		"tvdbId":           tvdbId,
		"title":            title,
		"qualityProfileId": 1,
		"rootFolderPath":   "/downloads",
		"path":             fmt.Sprintf("/downloads/%s", title),
		"monitored":        true,
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := s.apiRequest(ctx, "POST", "/api/v3/series", bytes.NewReader(payload))
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Condition(t, func() bool {
		return resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK
	}, "unexpected status code: %d", resp.StatusCode)

	var created map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))

	id, ok := created["id"].(float64)
	require.True(t, ok, "series id missing from response")
	return int(id)
}

type AppContainer struct {
	testcontainers.Container
}

func NewAppContainer(ctx context.Context, networkName string) (*AppContainer, error) {
	ctr, err := testcontainers.GenericContainer(ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				FromDockerfile: testcontainers.FromDockerfile{
					Context:    ".",
					Dockerfile: "Containerfile",
				},
				Env: map[string]string{
					"SHAREARR_EMAIL":   "test@example.com",
					"SHAREARR_API_KEY": "apikey",
					"SHAREARR_DB":      "",
				},
				Networks:       []string{networkName},
				NetworkAliases: map[string][]string{networkName: {"sharearr"}},
				ExposedPorts:   []string{"8787/tcp"},
				WaitingFor:     wait.ForHTTP("/torznab?t=caps&apikey=apikey").WithPort("8787/tcp"),
			},
			Started: true,
		},
	)
	if err != nil {
		return nil, err
	}
	return &AppContainer{Container: ctr}, nil
}

func (a *AppContainer) UploadTorrent(t *testing.T, ctx context.Context, torrentFile []byte, name string, cat string) torrentResponse {
	t.Helper()
	endpoint, err := a.Endpoint(ctx, "http")
	require.NoError(t, err)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, err := w.CreateFormFile("file", name+".torrent")
	require.NoError(t, err)
	_, err = io.Copy(fw, bytes.NewReader(torrentFile))
	require.NoError(t, err)
	w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/torrent/%s?apikey=apikey", endpoint, cat), &body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "UploadTorrent response: %s", respBody)

	var result torrentResponse
	require.NoError(t, json.Unmarshal(respBody, &result))
	return result
}

func makeTorrent(t *testing.T, name string, size int64) []byte {
	t.Helper()
	info := metainfo.Info{
		Name:        name,
		PieceLength: size,
		Length:      size,
		Pieces:      make([]byte, 20),
	}
	infoBytes, err := bencode.Marshal(info)
	require.NoError(t, err)
	mi := metainfo.MetaInfo{InfoBytes: infoBytes}
	var buf bytes.Buffer
	require.NoError(t, mi.Write(&buf))
	return buf.Bytes()
}

type TestEnv struct {
	App    *AppContainer
	Sonarr *SonarrContainer
}

func setup(t *testing.T, ctx context.Context) (*TestEnv, func()) {
	t.Helper()

	net, err := network.New(ctx)
	require.NoError(t, err)

	app, err := NewAppContainer(ctx, net.Name)
	require.NoError(t, err)

	sonarr, err := NewSonarrContainer(ctx, net.Name)
	require.NoError(t, err)

	cleanup := func() {
		testcontainers.CleanupContainer(t, app)
		testcontainers.CleanupContainer(t, sonarr)
		net.Remove(ctx)
	}
	return &TestEnv{App: app, Sonarr: sonarr}, cleanup
}

func TestWithApp(t *testing.T) {
	ctx := context.Background()
	env, cleanup := setup(t, ctx)
	defer cleanup()

	name := "the.beverly.hillbillies.s01"
	env.App.UploadTorrent(t, ctx, makeTorrent(t, name, 100*1024), name, "tv")

	env.Sonarr.AddIndexer(t, ctx, "sharearr_indexer", "sharearr", 8787)
	seriesId := env.Sonarr.AddSeries(t, ctx, 71471, "The Beverly Hillbillies")
	results := env.Sonarr.Release(t, ctx, seriesId, 1)

	assert.NotEmpty(t, results, "No results returned for search")
}
