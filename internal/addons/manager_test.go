package addons

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"naviserver/internal/domain"
	"naviserver/internal/server"
	"naviserver/internal/storage"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type testRoundTripper func(method, url string) (int, string)

func (rt testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	status, body := rt(req.Method, req.URL.String())
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

func testAddonHTTPClient(fn func(method, url string) (int, string)) *http.Client {
	return &http.Client{Transport: testRoundTripper(fn)}
}

func newTestAddonManager(t *testing.T) (*Manager, *domain.Server, string) {
	t.Helper()
	tempDir := t.TempDir()
	store, err := storage.NewGormStore(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	srv := &domain.Server{
		ID:         "srv-1",
		Name:       "Fabric",
		FolderName: "fabric-server",
		Version:    "1.21.1",
		Loader:     "fabric",
		Status:     "STOPPED",
		CreatedAt:  time.Now(),
	}
	if err := store.SaveServer(srv); err != nil {
		t.Fatalf("failed to save server: %v", err)
	}
	serversDir := filepath.Join(tempDir, "servers")
	serverManager := server.NewManager(serversDir, store)
	m := NewManager(serverManager, store)
	addonDir := filepath.Join(serversDir, srv.FolderName, "mods")
	if err := os.MkdirAll(addonDir, 0755); err != nil {
		t.Fatalf("failed to create addon dir: %v", err)
	}
	return m, srv, addonDir
}

func newTestAddonManagerForExistingStore(t *testing.T, source *Manager, addonDir string) (*Manager, *domain.Server, string) {
	t.Helper()
	srv, err := source.serverManager.GetServer("srv-1")
	if err != nil || srv == nil {
		t.Fatalf("failed to load source server: %v", err)
	}
	m := NewManager(source.serverManager, source.store)
	return m, srv, addonDir
}

func writeTestJar(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write jar: %v", err)
	}
}

func mustTestSHA1(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read jar: %v", err)
	}
	sum := sha1.Sum(content)
	return hex.EncodeToString(sum[:])
}

func writeTestPersistentCache(t *testing.T, addonDir string, cache persistentAddonCache) {
	t.Helper()
	payload, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("failed to marshal cache: %v", err)
	}
	path := addonCacheFilePath(addonDir)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}
	if err := os.WriteFile(path, payload, 0644); err != nil {
		t.Fatalf("failed to write cache: %v", err)
	}
}

func TestAddonScopeForLoader(t *testing.T) {
	tests := []struct {
		loader       string
		expectedType AddonType
		expectedPath string
		expectedOK   bool
	}{
		{loader: "paper", expectedType: AddonTypePlugin, expectedPath: "plugins", expectedOK: true},
		{loader: "fabric", expectedType: AddonTypeMod, expectedPath: "mods", expectedOK: true},
		{loader: "forge", expectedType: AddonTypeMod, expectedPath: "mods", expectedOK: true},
		{loader: "neoforge", expectedType: AddonTypeMod, expectedPath: "mods", expectedOK: true},
		{loader: "vanilla", expectedOK: false},
	}

	for _, tt := range tests {
		addonType, path, _, ok := addonScopeForLoader(tt.loader)
		if ok != tt.expectedOK {
			t.Fatalf("loader %s expected ok=%v got %v", tt.loader, tt.expectedOK, ok)
		}
		if !tt.expectedOK {
			continue
		}
		if addonType != tt.expectedType {
			t.Fatalf("loader %s expected type=%s got %s", tt.loader, tt.expectedType, addonType)
		}
		if path != tt.expectedPath {
			t.Fatalf("loader %s expected path=%s got %s", tt.loader, tt.expectedPath, path)
		}
	}
}

func TestCurseForgeFingerprintIgnoresWhitespace(t *testing.T) {
	base := []byte("abcDEF123")
	withWhitespace := []byte("a b\nc\tD\rE F123")

	baseHash := curseForgeFingerprint(base)
	wsHash := curseForgeFingerprint(withWhitespace)

	if baseHash != wsHash {
		t.Fatalf("expected equal fingerprints, got %d and %d", baseHash, wsHash)
	}
}

func TestTrimJarSuffix(t *testing.T) {
	if got := trimJarSuffix("example.jar"); got != "example" {
		t.Fatalf("unexpected value: %s", got)
	}
	if got := trimJarSuffix("example.jar.disabled"); got != "example" {
		t.Fatalf("unexpected disabled value: %s", got)
	}
	if got := trimJarSuffix("example"); got != "example" {
		t.Fatalf("unexpected value without suffix: %s", got)
	}
}

func TestAddonDisabledFilenameHelpers(t *testing.T) {
	if !isAddonJarFile("example.jar") {
		t.Fatalf("expected .jar to be accepted")
	}
	if !isAddonJarFile("example.jar.disabled") {
		t.Fatalf("expected .jar.disabled to be accepted")
	}
	if isAddonJarFile("example.disabled") {
		t.Fatalf("did not expect non-jar disabled file to be accepted")
	}
	if !isAddonDisabledFile("example.jar.disabled") {
		t.Fatalf("expected disabled file to be detected")
	}
	if got := normalizeAddonFileName("example"); got != "example.jar" {
		t.Fatalf("unexpected normalized file name: %s", got)
	}
	if got := normalizeAddonFileName("example.jar.disabled"); got != "example.jar.disabled" {
		t.Fatalf("unexpected normalized disabled file name: %s", got)
	}
}

func TestModrinthSearchEndpointWithEmptyQueryUsesDownloadsIndex(t *testing.T) {
	endpoint := modrinthSearchEndpoint("", []string{"fabric"}, "1.21.1", 0, 20)

	if !strings.Contains(endpoint, "index=downloads") {
		t.Fatalf("expected downloads index for empty query, got %s", endpoint)
	}
	if !strings.Contains(endpoint, "query=") {
		t.Fatalf("expected query parameter in endpoint, got %s", endpoint)
	}
}

func TestModrinthSearchEndpointWithQueryUsesRelevance(t *testing.T) {
	endpoint := modrinthSearchEndpoint("fabric api", []string{"fabric"}, "1.21.1", 20, 20)

	if strings.Contains(endpoint, "index=downloads") {
		t.Fatalf("did not expect downloads index for explicit query, got %s", endpoint)
	}
	if !strings.Contains(endpoint, "query=fabric+api") {
		t.Fatalf("expected encoded query in endpoint, got %s", endpoint)
	}
}

func TestCurseForgeSearchEndpointWithEmptyQueryUsesPopularity(t *testing.T) {
	endpoint := curseForgeSearchEndpoint("", "1.21.1", "fabric", 0, 20)

	if !strings.Contains(endpoint, "sortField=2") {
		t.Fatalf("expected popularity sort field, got %s", endpoint)
	}
	if !strings.Contains(endpoint, "sortOrder=desc") {
		t.Fatalf("expected descending sort order, got %s", endpoint)
	}
	if !strings.Contains(endpoint, "classId=6") {
		t.Fatalf("expected mods class filter for fabric loader, got %s", endpoint)
	}
}

func TestCurseForgeSearchEndpointWithQueryUsesSearchFilter(t *testing.T) {
	endpoint := curseForgeSearchEndpoint("fabric api", "1.21.1", "fabric", 20, 20)

	if !strings.Contains(endpoint, "searchFilter=fabric+api") {
		t.Fatalf("expected encoded search filter, got %s", endpoint)
	}
	if strings.Contains(endpoint, "sortField=2") {
		t.Fatalf("did not expect popularity sort when query is present, got %s", endpoint)
	}
}

func TestCurseForgeSearchEndpointWithPaperUsesPluginClass(t *testing.T) {
	endpoint := curseForgeSearchEndpoint("", "1.21.1", "paper", 0, 20)
	if !strings.Contains(endpoint, "classId=5") {
		t.Fatalf("expected plugin class filter for paper loader, got %s", endpoint)
	}
}

func TestModrinthFacetsRestrictServerSideForFabric(t *testing.T) {
	facets := modrinthFacets([]string{"fabric"}, "1.21.1")
	if !strings.Contains(facets, "server_side:required") {
		t.Fatalf("expected required server_side facet, got %s", facets)
	}
	if !strings.Contains(facets, "server_side:optional") {
		t.Fatalf("expected optional server_side facet, got %s", facets)
	}
}

func TestModrinthFacetsDoNotRestrictServerSideForPaper(t *testing.T) {
	facets := modrinthFacets([]string{"paper", "spigot"}, "1.21.1")
	if strings.Contains(facets, "server_side:required") {
		t.Fatalf("did not expect server_side restriction for paper plugins, got %s", facets)
	}
}

func TestIsCurseFileServerCompatibleRejectsClientOnly(t *testing.T) {
	file := curseFile{
		GameVersions: []string{"1.21.1", "Fabric", "Client"},
	}
	if isCurseFileServerCompatible(file, "fabric") {
		t.Fatalf("expected client-only file to be rejected")
	}
}

func TestIsCurseFileServerCompatibleAllowsServerAndUnknown(t *testing.T) {
	serverFile := curseFile{
		GameVersions: []string{"1.21.1", "Fabric", "Server"},
	}
	if !isCurseFileServerCompatible(serverFile, "fabric") {
		t.Fatalf("expected server-tagged file to be accepted")
	}

	unknownFile := curseFile{
		GameVersions: []string{"1.21.1", "Fabric"},
	}
	if !isCurseFileServerCompatible(unknownFile, "fabric") {
		t.Fatalf("expected unknown-side file to be accepted")
	}
}

func TestIsLikelyClientOnlyCurseProject(t *testing.T) {
	categories := []struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}{
		{Name: "Shaders", Slug: "shaders"},
	}
	if !isLikelyClientOnlyCurseProject("Iris Shaders", "", categories) {
		t.Fatalf("expected shader project to be flagged as client-only")
	}
}

func TestIsLikelyClientOnlyCurseProjectAllowsCommonServerMods(t *testing.T) {
	categories := []struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}{
		{Name: "API and Library", Slug: "api-and-library"},
	}
	if isLikelyClientOnlyCurseProject("Bookshelf", "A common library mod", categories) {
		t.Fatalf("did not expect library mod to be flagged")
	}
}

func TestUpdateAddonsForServerVersionNoopsForVanilla(t *testing.T) {
	tempDir := t.TempDir()
	store, err := storage.NewGormStore(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	srv := &domain.Server{
		ID:         "srv-1",
		Name:       "Vanilla",
		FolderName: "vanilla",
		Version:    "1.21.1",
		Loader:     "vanilla",
		Status:     "STOPPED",
		CreatedAt:  time.Now(),
	}
	if err := store.SaveServer(srv); err != nil {
		t.Fatalf("failed to save server: %v", err)
	}

	serverManager := server.NewManager(filepath.Join(tempDir, "servers"), store)
	m := NewManager(serverManager, store)
	result, err := m.UpdateAddonsForServerVersion(nil, srv.ID, true)
	if err != nil {
		t.Fatalf("expected vanilla no-op, got %v", err)
	}
	if result == nil || len(result.Updated) != 0 || len(result.Disabled) != 0 || len(result.Failed) != 0 {
		t.Fatalf("expected empty no-op result, got %#v", result)
	}
}

func TestPersistentAddonCacheWritesAndReusesMetadata(t *testing.T) {
	m, srv, addonDir := newTestAddonManager(t)
	jarPath := filepath.Join(addonDir, "cached.jar")
	writeTestJar(t, jarPath, "cached-content")

	calls := 0
	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		calls++
		switch {
		case strings.Contains(url, "/version_files/update"):
			return 200, `{}`
		case strings.Contains(url, "/version_files"):
			sha1 := mustTestSHA1(t, jarPath)
			return 200, `{"` + sha1 + `":{"id":"ver-1","project_id":"project-1","project_title":"Cached Mod","project_slug":"cached-mod","version_number":"1.0.0","version_type":"release","files":[{"url":"https://example.test/cached.jar","filename":"cached.jar","primary":true}]}}`
		case strings.Contains(url, "/project/project-1"):
			return 200, `{"icon_url":"https://example.test/icon.png"}`
		default:
			return 404, `{}`
		}
	})

	first, err := m.ListAddons(t.Context(), srv.ID)
	if err != nil {
		t.Fatalf("first ListAddons failed: %v", err)
	}
	if calls == 0 {
		t.Fatalf("expected first load to call metadata APIs")
	}
	if len(first.Items) != 1 || first.Items[0].ProjectID != "project-1" || first.Items[0].IconURL == "" {
		t.Fatalf("unexpected first response: %#v", first.Items)
	}
	if _, err := os.Stat(addonCacheFilePath(addonDir)); err != nil {
		t.Fatalf("expected persistent cache file: %v", err)
	}

	m2, _, _ := newTestAddonManagerForExistingStore(t, m, addonDir)
	m2.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		t.Fatalf("did not expect HTTP request when persistent cache is valid: %s %s", method, url)
		return 500, `{}`
	})
	second, err := m2.ListAddons(t.Context(), srv.ID)
	if err != nil {
		t.Fatalf("second ListAddons failed: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].ProjectID != "project-1" || second.Items[0].IconURL != "https://example.test/icon.png" {
		t.Fatalf("unexpected cached response: %#v", second.Items)
	}
}

func TestPersistentAddonCacheInvalidatesChangedFile(t *testing.T) {
	m, srv, addonDir := newTestAddonManager(t)
	jarPath := filepath.Join(addonDir, "changed.jar")
	writeTestJar(t, jarPath, "old-content")
	cache := persistentAddonCache{
		SchemaVersion: 1,
		ServerID:      srv.ID,
		Loader:        srv.Loader,
		Version:       srv.Version,
		AddonType:     AddonTypeMod,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Entries: map[string]persistentAddonEntry{
			"changed.jar": {
				Size:           1,
				ModifiedAtUnix: 1,
				Addon: Addon{
					ID:        "changed.jar",
					FileName:  "changed.jar",
					Name:      "stale",
					ProjectID: "stale-project",
					Source:    AddonSourceModrinth,
				},
			},
		},
	}
	writeTestPersistentCache(t, addonDir, cache)

	calls := 0
	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		calls++
		return 200, `{}`
	})
	response, err := m.ListAddons(t.Context(), srv.ID)
	if err != nil {
		t.Fatalf("ListAddons failed: %v", err)
	}
	if calls == 0 {
		t.Fatalf("expected stale cache to trigger metadata refresh")
	}
	if len(response.Items) != 1 || response.Items[0].ProjectID == "stale-project" {
		t.Fatalf("expected stale cached metadata to be ignored, got %#v", response.Items)
	}
}

func TestSyncAddonsBypassesPersistentCache(t *testing.T) {
	m, srv, addonDir := newTestAddonManager(t)
	jarPath := filepath.Join(addonDir, "sync.jar")
	writeTestJar(t, jarPath, "sync-content")
	info, err := os.Stat(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	cache := persistentAddonCache{
		SchemaVersion: 1,
		ServerID:      srv.ID,
		Loader:        srv.Loader,
		Version:       srv.Version,
		AddonType:     AddonTypeMod,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Entries: map[string]persistentAddonEntry{
			"sync.jar": {
				Size:           info.Size(),
				ModifiedAtUnix: info.ModTime().UnixNano(),
				Addon: Addon{
					ID:        "sync.jar",
					FileName:  "sync.jar",
					Name:      "cached",
					ProjectID: "cached-project",
					Source:    AddonSourceModrinth,
					Size:      info.Size(),
				},
			},
		},
	}
	writeTestPersistentCache(t, addonDir, cache)

	calls := 0
	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		calls++
		switch {
		case strings.Contains(url, "/version_files/update"):
			return 200, `{}`
		case strings.Contains(url, "/version_files"):
			sha1 := mustTestSHA1(t, jarPath)
			return 200, `{"` + sha1 + `":{"id":"ver-2","project_id":"fresh-project","project_title":"Fresh Mod","project_slug":"fresh-mod","version_number":"2.0.0","version_type":"release","files":[{"url":"https://example.test/sync.jar","filename":"sync.jar","primary":true}]}}`
		case strings.Contains(url, "/project/fresh-project"):
			return 200, `{"icon_url":"https://example.test/fresh.png"}`
		default:
			return 404, `{}`
		}
	})
	response, err := m.SyncAddons(t.Context(), srv.ID)
	if err != nil {
		t.Fatalf("SyncAddons failed: %v", err)
	}
	if calls == 0 {
		t.Fatalf("expected SyncAddons to call metadata APIs")
	}
	if len(response.Items) != 1 || response.Items[0].ProjectID != "fresh-project" {
		t.Fatalf("expected sync to refresh cached metadata, got %#v", response.Items)
	}
}

func TestPersistentAddonCacheCleansDeletedFilesAndIgnoresCacheDirectory(t *testing.T) {
	m, srv, addonDir := newTestAddonManager(t)
	keepPath := filepath.Join(addonDir, "keep.jar")
	writeTestJar(t, keepPath, "keep-content")
	info, err := os.Stat(keepPath)
	if err != nil {
		t.Fatal(err)
	}
	cacheDir := filepath.Join(addonDir, ".cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeTestJar(t, filepath.Join(cacheDir, "ignored.jar"), "ignored-content")
	cache := persistentAddonCache{
		SchemaVersion: 1,
		ServerID:      srv.ID,
		Loader:        srv.Loader,
		Version:       srv.Version,
		AddonType:     AddonTypeMod,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Entries: map[string]persistentAddonEntry{
			"keep.jar": {
				Size:           info.Size(),
				ModifiedAtUnix: info.ModTime().UnixNano(),
				Addon:          Addon{ID: "keep.jar", FileName: "keep.jar", Name: "keep", Source: AddonSourceManual, Size: info.Size()},
			},
			"deleted.jar": {
				Size:           10,
				ModifiedAtUnix: 10,
				Addon:          Addon{ID: "deleted.jar", FileName: "deleted.jar", Name: "deleted", Source: AddonSourceManual, Size: 10},
			},
		},
	}
	writeTestPersistentCache(t, addonDir, cache)

	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		t.Fatalf("did not expect HTTP request for valid cached file: %s %s", method, url)
		return 500, `{}`
	})
	response, err := m.ListAddons(t.Context(), srv.ID)
	if err != nil {
		t.Fatalf("ListAddons failed: %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].FileName != "keep.jar" {
		t.Fatalf("expected only keep.jar, got %#v", response.Items)
	}
	updated, err := readPersistentAddonCache(addonDir, srv, AddonTypeMod)
	if err != nil {
		t.Fatalf("failed to read rewritten cache: %v", err)
	}
	if len(updated.Entries) != 1 {
		t.Fatalf("expected deleted entries to be cleaned, got %#v", updated.Entries)
	}
	if _, ok := updated.Entries["ignored.jar"]; ok {
		t.Fatalf("expected .cache jar to be ignored")
	}
}
