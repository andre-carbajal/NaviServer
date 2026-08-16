package addons

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	t.Cleanup(func() { _ = store.Close() })
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
	serverManager := server.NewManager(serversDir, store, nil)
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
	if !strings.Contains(endpoint, "modLoaderType=4") {
		t.Fatalf("expected Fabric mod loader filter, got %s", endpoint)
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
	if strings.Contains(endpoint, "modLoaderType=") {
		t.Fatalf("did not expect a mod loader filter for Paper plugins, got %s", endpoint)
	}
}

func TestCurseForgeSearchEndpointUsesOfficialModLoaderTypes(t *testing.T) {
	tests := []struct {
		loader string
		value  string
	}{
		{loader: "forge", value: "1"},
		{loader: "fabric", value: "4"},
		{loader: "neoforge", value: "6"},
	}

	for _, tt := range tests {
		t.Run(tt.loader, func(t *testing.T) {
			endpoint := curseForgeSearchEndpoint("", "1.21.1", tt.loader, 0, 20)
			if !strings.Contains(endpoint, "modLoaderType="+tt.value) {
				t.Fatalf("expected modLoaderType=%s for %s, got %s", tt.value, tt.loader, endpoint)
			}
		})
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

func TestIsCurseFileServerCompatibleEnvironmentPolicy(t *testing.T) {
	tests := []struct {
		name         string
		gameVersions []string
		want         bool
	}{
		{name: "not set", gameVersions: []string{"1.21.1"}, want: true},
		{name: "server", gameVersions: []string{"1.21.1", "Server"}, want: true},
		{name: "client and server separate tags", gameVersions: []string{"1.21.1", "Client", "Server"}, want: true},
		{name: "client and server", gameVersions: []string{"1.21.1", "Client & Server"}, want: true},
		{name: "client only", gameVersions: []string{"1.21.1", "Client"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := curseFile{GameVersions: tt.gameVersions}
			if got := isCurseFileServerCompatible(file, "fabric"); got != tt.want {
				t.Fatalf("expected server compatibility %v, got %v", tt.want, got)
			}
		})
	}
}

func TestListCompatibleCurseFilesAllowsEnvironmentOnlyServerTag(t *testing.T) {
	files := []curseFile{{
		ID:           101,
		GameVersions: []string{"1.21.1", "Server"},
	}}

	compatible := listCompatibleCurseFiles(files, "1.21.1", "fabric")
	if len(compatible) != 1 || compatible[0].ID != 101 {
		t.Fatalf("expected environment-only server file to be compatible, got %#v", compatible)
	}
}

func TestCurseForgeFingerprintResponseSupportsLatestFiles(t *testing.T) {
	m, _, _ := newTestAddonManager(t)
	if err := m.SetCustomCurseForgeAPIKey("test-key"); err != nil {
		t.Fatalf("failed to set CurseForge test key: %v", err)
	}
	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		if method != http.MethodPost || !strings.HasSuffix(url, "/fingerprints") {
			t.Fatalf("unexpected request: %s %s", method, url)
		}
		return 200, `{"latestFiles":[{"id":501,"modId":987,"fileName":"used-totem.jar","fileFingerprint":12345,"gameVersions":["1.21.1"]}],"installedFingerprints":[12345],"partialMatchFingerprints":{},"unmatchedFingerprints":[]}`
	})

	matches, err := m.curseForgeByFingerprints(context.Background(), []scannedAddonFile{{fingerprint: 12345}})
	if err != nil {
		t.Fatalf("fingerprint lookup failed: %v", err)
	}
	match, ok := matches[12345]
	if !ok || match.File.ModID != 987 || match.Mod.ID != 987 {
		t.Fatalf("expected latest file fingerprint match, got %#v", matches)
	}
}

func TestEnrichCurseForgeMatchesLoadsModLogo(t *testing.T) {
	m, _, _ := newTestAddonManager(t)
	if err := m.SetCustomCurseForgeAPIKey("test-key"); err != nil {
		t.Fatalf("failed to set CurseForge test key: %v", err)
	}
	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		if method != http.MethodPost || !strings.HasSuffix(url, "/mods") {
			t.Fatalf("unexpected request: %s %s", method, url)
		}
		return 200, `{"data":[{"id":987,"name":"UsedTotemMessage","slug":"used-totem-message","links":{"websiteUrl":"https://www.curseforge.com/minecraft/mc-mods/used-totem-message"},"logo":{"url":"https://media.forgecdn.net/logos/987/654/logo.png"}}]}`
	})

	matches := map[uint32]curseMatch{
		12345: {
			File: curseFile{ModID: 987},
			Mod:  curseMod{ID: 987},
		},
	}
	if err := m.enrichCurseForgeMatches(context.Background(), matches); err != nil {
		t.Fatalf("mod metadata lookup failed: %v", err)
	}
	match := matches[12345]
	if match.Mod.Name != "UsedTotemMessage" || match.Mod.iconURL() != "https://media.forgecdn.net/logos/987/654/logo.png" {
		t.Fatalf("expected CurseForge metadata and logo, got %#v", match.Mod)
	}
}

func TestListAddonsLoadsCurseForgeModLogo(t *testing.T) {
	m, srv, addonDir := newTestAddonManager(t)
	if err := m.SetCustomCurseForgeAPIKey("test-key"); err != nil {
		t.Fatalf("failed to set CurseForge test key: %v", err)
	}
	jarPath := filepath.Join(addonDir, "used-totem.jar")
	writeTestJar(t, jarPath, "used-totem-content")
	_, _, fingerprint, err := computeJarMetadata(jarPath)
	if err != nil {
		t.Fatalf("failed to compute test jar metadata: %v", err)
	}

	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		switch {
		case method == http.MethodPost && strings.HasSuffix(url, "/fingerprints"):
			return 200, fmt.Sprintf(`{"data":{"exactMatches":[{"id":1,"file":{"id":501,"modId":987,"displayName":"UsedTotemMessage 1.0.0","fileName":"used-totem.jar","releaseType":1,"fileFingerprint":%d,"gameVersions":["1.21.1"]},"mod":{"id":987,"name":"UsedTotemMessage","slug":"used-totem-message","links":{"websiteUrl":"https://www.curseforge.com/minecraft/mc-mods/used-totem-message"}}}]}}`, fingerprint)
		case method == http.MethodPost && strings.HasSuffix(url, "/mods"):
			return 200, `{"data":[{"id":987,"name":"UsedTotemMessage","slug":"used-totem-message","logo":{"url":"https://media.forgecdn.net/logos/987/654/logo.png"}}]}`
		case strings.Contains(url, "/mods/987/files?"):
			return 200, `{"data":[],"pagination":{"index":0,"pageSize":50,"resultCount":0,"totalCount":0}}`
		default:
			return 404, `{}`
		}
	})

	response, err := m.ListAddons(context.Background(), srv.ID)
	if err != nil {
		t.Fatalf("listing addons failed: %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].Source != AddonSourceCurseForge {
		t.Fatalf("expected a CurseForge addon, got %#v", response.Items)
	}
	if response.Items[0].IconURL != "https://media.forgecdn.net/logos/987/654/logo.png" {
		t.Fatalf("expected CurseForge logo URL, got %#v", response.Items[0])
	}
}

func TestCachedCurseForgeAddonWithoutIconForcesMetadataRefresh(t *testing.T) {
	modifiedAt := time.Unix(123, 0)
	cache := &persistentAddonCache{
		Entries: map[string]persistentAddonEntry{
			"used-totem.jar": {
				Size:           10,
				ModifiedAtUnix: modifiedAt.UnixNano(),
				Addon: Addon{
					Source:    AddonSourceCurseForge,
					ProjectID: "987",
					IconURL:   "",
				},
			},
		},
	}
	item := scannedAddonFile{
		fileName:   "used-totem.jar",
		size:       10,
		modifiedAt: modifiedAt,
	}
	if _, ok := cachedAddonForItem(cache, item, AddonTypeMod); ok {
		t.Fatal("expected cached CurseForge addon without an icon to be refreshed")
	}
}

func TestFilterCurseSearchFilesUsesLoaderIndexes(t *testing.T) {
	files := []curseFile{
		{ID: 101, GameVersions: []string{"1.21.1", "Server"}},
		{ID: 102, GameVersions: []string{"1.21.1", "Server"}},
	}
	indexes := []curseFileIndex{
		{FileID: 101, GameVersion: "1.21.1", ModLoader: 4},
		{FileID: 102, GameVersion: "1.21.1", ModLoader: 1},
	}

	filtered := filterCurseSearchFiles(files, indexes, "1.21.1", "fabric")
	if len(filtered) != 1 || filtered[0].ID != 101 {
		t.Fatalf("expected only Fabric-indexed file, got %#v", filtered)
	}
}

func TestFilterCurseSearchFilesFallsBackWithoutVersionIndex(t *testing.T) {
	files := []curseFile{{ID: 101, GameVersions: []string{"1.21.1", "Server"}}}
	indexes := []curseFileIndex{{FileID: 101, GameVersion: "1.20.1", ModLoader: 1}}

	filtered := filterCurseSearchFiles(files, indexes, "1.21.1", "fabric")
	if len(filtered) != 1 || filtered[0].ID != 101 {
		t.Fatalf("expected files without a matching version index to be preserved, got %#v", filtered)
	}
}

func TestSearchCurseForgeIncludesServerEnvironmentFile(t *testing.T) {
	m, _, _ := newTestAddonManager(t)
	if err := m.SetCustomCurseForgeAPIKey("test-key"); err != nil {
		t.Fatalf("failed to set CurseForge test key: %v", err)
	}

	var requestedURL string
	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		requestedURL = url
		return 200, `{"data":[{"id":123,"name":"Server Mod","slug":"server-mod","summary":"Server-compatible mod","latestFiles":[{"id":101,"fileName":"server-mod.jar","gameVersions":["Client","Fabric","Server","1.21.1"],"releaseType":1},{"id":102,"fileName":"client-mod.jar","gameVersions":["Client","Fabric","1.21.1"],"releaseType":1}],"latestFilesIndexes":[{"gameVersion":"1.21.1","fileId":101,"modLoader":4},{"gameVersion":"1.21.1","fileId":102,"modLoader":4}]}],"pagination":{"index":0,"pageSize":20,"resultCount":1,"totalCount":1}}`
	})

	results, hasMore, err := m.searchCurseForge(context.Background(), "server", "1.21.1", "fabric", 0, 20)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if hasMore {
		t.Fatalf("expected no additional page")
	}
	if !strings.Contains(requestedURL, "modLoaderType=4") {
		t.Fatalf("expected Fabric loader query, got %s", requestedURL)
	}
	if len(results) != 1 || results[0].Latest == nil || results[0].Latest.FileID != 101 {
		t.Fatalf("expected Server file in search results, got %#v", results)
	}
	if len(results[0].Versions) != 1 || results[0].Versions[0].FileID != 101 {
		t.Fatalf("expected only Server file version, got %#v", results[0].Versions)
	}
}

func TestSearchCurseForgeFallsBackToFullFilesForMissingServerLatestFile(t *testing.T) {
	m, _, _ := newTestAddonManager(t)
	if err := m.SetCustomCurseForgeAPIKey("test-key"); err != nil {
		t.Fatalf("failed to set CurseForge test key: %v", err)
	}

	var requestedFilesURL string
	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		switch {
		case strings.Contains(url, "/mods/search?"):
			return 200, `{"data":[{"id":123,"name":"Server Mod","slug":"server-mod","summary":"Server-compatible mod","latestFiles":[{"id":101,"fileName":"client-mod.jar","gameVersions":["Client","Fabric","1.21.1"],"releaseType":1}],"latestFilesIndexes":[{"gameVersion":"1.21.1","fileId":101,"modLoader":4}]}],"pagination":{"index":0,"pageSize":20,"resultCount":1,"totalCount":1}}`
		case strings.Contains(url, "/mods/123/files?"):
			requestedFilesURL = url
			return 200, `{"data":[{"id":102,"modId":123,"fileName":"server-mod.jar","gameVersions":["Client","Fabric","Server","1.21.1"],"releaseType":1}],"pagination":{"index":0,"pageSize":50,"resultCount":1,"totalCount":1}}`
		default:
			return 404, `{}`
		}
	})

	results, _, err := m.searchCurseForge(context.Background(), "server", "1.21.1", "fabric", 0, 20)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if !strings.Contains(requestedFilesURL, "modLoaderType=4") {
		t.Fatalf("expected Fabric loader query in fallback, got %s", requestedFilesURL)
	}
	if len(results) != 1 || results[0].Latest == nil || results[0].Latest.FileID != 102 {
		t.Fatalf("expected fallback Server file in search results, got %#v", results)
	}
}

func TestSearchAddonsPropagatesCurseForgeErrors(t *testing.T) {
	m, srv, _ := newTestAddonManager(t)
	if err := m.SetCustomCurseForgeAPIKey("test-key"); err != nil {
		t.Fatalf("failed to set CurseForge test key: %v", err)
	}
	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		return 400, `{"error":"invalid CurseForge request"}`
	})

	_, err := m.SearchAddons(context.Background(), srv.ID, "server", string(AddonSourceCurseForge), 0, 20)
	if err == nil || !strings.Contains(err.Error(), "request failed (400)") {
		t.Fatalf("expected CurseForge API error to be returned, got %v", err)
	}
}

func TestListAddonVersionsUsesLoaderAndEnvironmentFilter(t *testing.T) {
	m, srv, _ := newTestAddonManager(t)
	if err := m.SetCustomCurseForgeAPIKey("test-key"); err != nil {
		t.Fatalf("failed to set CurseForge test key: %v", err)
	}

	var requestedURL string
	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		requestedURL = url
		return 200, `{"data":[{"id":201,"fileName":"server-mod.jar","gameVersions":["1.21.1","Server"],"releaseType":1},{"id":202,"fileName":"client-mod.jar","gameVersions":["1.21.1","Client"],"releaseType":1}],"pagination":{"index":0,"pageSize":50,"resultCount":2,"totalCount":2}}`
	})

	result, err := m.ListAddonVersions(context.Background(), srv.ID, VersionsRequest{
		Source:    AddonSourceCurseForge,
		ProjectID: "123",
	})
	if err != nil {
		t.Fatalf("listing versions failed: %v", err)
	}
	if !strings.Contains(requestedURL, "modLoaderType=4") {
		t.Fatalf("expected Fabric loader query, got %s", requestedURL)
	}
	if len(result.Versions) != 1 || result.Versions[0].FileID != 201 {
		t.Fatalf("expected only Server version, got %#v", result.Versions)
	}
}

func TestInstallCurseForgeRejectsClientOnlyFile(t *testing.T) {
	m, srv, addonDir := newTestAddonManager(t)
	if err := m.SetCustomCurseForgeAPIKey("test-key"); err != nil {
		t.Fatalf("failed to set CurseForge test key: %v", err)
	}

	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		return 200, `{"data":[{"id":301,"fileName":"client-mod.jar","gameVersions":["1.21.1","Client"],"releaseType":1}],"pagination":{"index":0,"pageSize":50,"resultCount":1,"totalCount":1}}`
	})

	err := m.InstallAddon(context.Background(), srv.ID, InstallRequest{
		Source:    AddonSourceCurseForge,
		ProjectID: "123",
		FileID:    301,
	})
	if err == nil || !strings.Contains(err.Error(), "no compatible file found") {
		t.Fatalf("expected client-only file to be rejected, got %v", err)
	}
	entries, err := os.ReadDir(addonDir)
	if err != nil {
		t.Fatalf("failed to inspect addon directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no file to be installed, found %d entries", len(entries))
	}
}

func TestInstallCurseForgeAcceptsServerFileWithLoaderFilter(t *testing.T) {
	m, srv, addonDir := newTestAddonManager(t)
	if err := m.SetCustomCurseForgeAPIKey("test-key"); err != nil {
		t.Fatalf("failed to set CurseForge test key: %v", err)
	}

	var requestedFilesURL string
	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		switch {
		case strings.Contains(url, "/mods/123/files?"):
			requestedFilesURL = url
			return 200, `{"data":[{"id":401,"modId":123,"fileName":"server-mod.jar","downloadUrl":"https://example.test/server-mod.jar","gameVersions":["1.21.1","Server"],"releaseType":1}],"pagination":{"index":0,"pageSize":50,"resultCount":1,"totalCount":1}}`
		case url == "https://example.test/server-mod.jar":
			return 200, "jar-content"
		default:
			return 404, `{}`
		}
	})

	err := m.InstallAddon(context.Background(), srv.ID, InstallRequest{
		Source:    AddonSourceCurseForge,
		ProjectID: "123",
		FileID:    401,
	})
	if err != nil {
		t.Fatalf("expected Server file to install, got %v", err)
	}
	if !strings.Contains(requestedFilesURL, "modLoaderType=4") {
		t.Fatalf("expected Fabric loader query during installation, got %s", requestedFilesURL)
	}
	content, err := os.ReadFile(filepath.Join(addonDir, "server-mod.jar"))
	if err != nil {
		t.Fatalf("failed to read installed file: %v", err)
	}
	if string(content) != "jar-content" {
		t.Fatalf("unexpected installed file content: %q", content)
	}
}

func TestInstallCurseForgeDownloadsNestedRequiredDependenciesFromCurseForge(t *testing.T) {
	m, srv, addonDir := newTestAddonManager(t)
	if err := m.SetCustomCurseForgeAPIKey("test-key"); err != nil {
		t.Fatalf("failed to set CurseForge test key: %v", err)
	}

	requestedFiles := make([]string, 0)
	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		switch {
		case strings.Contains(url, "/mods/123/files?"):
			requestedFiles = append(requestedFiles, url)
			return 200, `{"data":[{"id":401,"modId":123,"fileName":"parent-mod.jar","downloadUrl":"https://example.test/parent-mod.jar","gameVersions":["1.21.1","Server"],"releaseType":1,"dependencies":[{"modId":456,"fileId":402,"relationType":3},{"modId":999,"relationType":2}]}],"pagination":{"index":0,"pageSize":50,"resultCount":1,"totalCount":1}}`
		case strings.Contains(url, "/mods/456/files?"):
			requestedFiles = append(requestedFiles, url)
			return 200, `{"data":[{"id":402,"modId":456,"fileName":"library-mod.jar","downloadUrl":"https://example.test/library-mod.jar","gameVersions":["1.21.1","Server"],"releaseType":1,"dependencies":[{"modId":789,"fileId":403,"relationType":3}]}],"pagination":{"index":0,"pageSize":50,"resultCount":1,"totalCount":1}}`
		case strings.Contains(url, "/mods/789/files?"):
			requestedFiles = append(requestedFiles, url)
			return 200, `{"data":[{"id":403,"modId":789,"fileName":"nested-library.jar","downloadUrl":"https://example.test/nested-library.jar","gameVersions":["1.21.1","Server"],"releaseType":1}],"pagination":{"index":0,"pageSize":50,"resultCount":1,"totalCount":1}}`
		case url == "https://example.test/parent-mod.jar":
			return 200, "parent-content"
		case url == "https://example.test/library-mod.jar":
			return 200, "library-content"
		case url == "https://example.test/nested-library.jar":
			return 200, "nested-library-content"
		default:
			return 404, `{}`
		}
	})

	err := m.InstallAddon(context.Background(), srv.ID, InstallRequest{
		Source:              AddonSourceCurseForge,
		ProjectID:           "123",
		FileID:              401,
		IncludeDependencies: true,
	})
	if err != nil {
		t.Fatalf("expected CurseForge dependencies to install, got %v", err)
	}
	for _, file := range []struct {
		name    string
		content string
	}{
		{name: "parent-mod.jar", content: "parent-content"},
		{name: "library-mod.jar", content: "library-content"},
		{name: "nested-library.jar", content: "nested-library-content"},
	} {
		content, err := os.ReadFile(filepath.Join(addonDir, file.name))
		if err != nil {
			t.Fatalf("expected %s to be installed: %v", file.name, err)
		}
		if string(content) != file.content {
			t.Fatalf("unexpected content for %s: %q", file.name, content)
		}
	}
	if len(requestedFiles) != 3 {
		t.Fatalf("expected parent and two CurseForge dependency file requests, got %d", len(requestedFiles))
	}
	for _, url := range requestedFiles {
		if !strings.Contains(url, "modLoaderType=4") {
			t.Fatalf("expected Fabric loader filter for dependency request, got %s", url)
		}
	}
	provenance, err := readAddonProvenance(addonDir)
	if err != nil {
		t.Fatalf("failed to read CurseForge provenance: %v", err)
	}
	for _, fileName := range []string{"parent-mod.jar", "library-mod.jar", "nested-library.jar"} {
		entry, ok := provenance.Entries[fileName]
		if !ok || entry.Source != AddonSourceCurseForge {
			t.Fatalf("expected %s to retain CurseForge provenance, got %#v", fileName, provenance.Entries)
		}
	}
}

func TestInstallModrinthDownloadsNestedRequiredDependenciesFromModrinth(t *testing.T) {
	m, srv, addonDir := newTestAddonManager(t)
	requestedURLs := make([]string, 0)
	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		requestedURLs = append(requestedURLs, url)
		switch {
		case url == "https://api.modrinth.com/v2/version/parent-version":
			return 200, `{"id":"parent-version","project_id":"parent","files":[{"url":"https://example.test/parent.jar","filename":"parent.jar","primary":true}],"dependencies":[{"version_id":"dep-version","project_id":"dep","dependency_type":"required"},{"version_id":"optional-version","project_id":"optional","dependency_type":"optional"}]}`
		case url == "https://api.modrinth.com/v2/version/dep-version":
			return 200, `{"id":"dep-version","project_id":"dep","files":[{"url":"https://example.test/dep.jar","filename":"dep.jar","primary":true}],"dependencies":[{"version_id":"nested-version","project_id":"nested","dependency_type":"required"}]}`
		case url == "https://api.modrinth.com/v2/version/nested-version":
			return 200, `{"id":"nested-version","project_id":"nested","files":[{"url":"https://example.test/nested.jar","filename":"nested.jar","primary":true}]}`
		case url == "https://example.test/parent.jar":
			return 200, "parent-content"
		case url == "https://example.test/dep.jar":
			return 200, "dep-content"
		case url == "https://example.test/nested.jar":
			return 200, "nested-content"
		default:
			t.Fatalf("unexpected Modrinth request: %s %s", method, url)
			return 404, `{}`
		}
	})

	err := m.InstallAddon(context.Background(), srv.ID, InstallRequest{
		Source:              AddonSourceModrinth,
		ProjectID:           "parent",
		VersionID:           "parent-version",
		IncludeDependencies: true,
	})
	if err != nil {
		t.Fatalf("expected Modrinth dependencies to install, got %v", err)
	}
	for _, fileName := range []string{"parent.jar", "dep.jar", "nested.jar"} {
		if _, err := os.Stat(filepath.Join(addonDir, fileName)); err != nil {
			t.Fatalf("expected %s to be installed: %v", fileName, err)
		}
	}
	if len(requestedURLs) != 6 {
		t.Fatalf("expected three version and three download requests, got %d: %v", len(requestedURLs), requestedURLs)
	}
	for _, url := range requestedURLs {
		if strings.Contains(url, "curseforge") {
			t.Fatalf("did not expect CurseForge request for Modrinth installation: %s", url)
		}
	}
	provenance, err := readAddonProvenance(addonDir)
	if err != nil {
		t.Fatalf("failed to read Modrinth provenance: %v", err)
	}
	for _, fileName := range []string{"parent.jar", "dep.jar", "nested.jar"} {
		entry, ok := provenance.Entries[fileName]
		if !ok || entry.Source != AddonSourceModrinth {
			t.Fatalf("expected %s to retain Modrinth provenance, got %#v", fileName, provenance.Entries)
		}
	}
}

func TestListAddonsUsesRecordedCurseForgeSourceWhenCatalogsOverlap(t *testing.T) {
	m, srv, addonDir := newTestAddonManager(t)
	if err := m.SetCustomCurseForgeAPIKey("test-key"); err != nil {
		t.Fatalf("failed to set CurseForge test key: %v", err)
	}
	jarPath := filepath.Join(addonDir, "shared.jar")
	writeTestJar(t, jarPath, "shared-content")
	_, sha512Hash, fingerprint, err := computeJarMetadata(jarPath)
	if err != nil {
		t.Fatalf("failed to compute jar metadata: %v", err)
	}
	if err := m.recordAddonProvenance(addonDir, "shared.jar", addonProvenanceEntry{
		Source:    AddonSourceCurseForge,
		ProjectID: "987",
		VersionID: "501",
		FileID:    501,
	}); err != nil {
		t.Fatalf("failed to record test provenance: %v", err)
	}
	info, err := os.Stat(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	writeTestPersistentCache(t, addonDir, persistentAddonCache{
		SchemaVersion: 1,
		ServerID:      srv.ID,
		Loader:        srv.Loader,
		Version:       srv.Version,
		AddonType:     AddonTypeMod,
		Entries: map[string]persistentAddonEntry{
			"shared.jar": {
				Size:           info.Size(),
				ModifiedAtUnix: info.ModTime().UnixNano(),
				Addon: Addon{
					ID:        "shared.jar",
					FileName:  "shared.jar",
					Source:    AddonSourceModrinth,
					ProjectID: "mr-project",
					IconURL:   "https://example.test/modrinth.png",
				},
			},
		},
	})

	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		switch {
		case method == http.MethodPost && strings.HasSuffix(url, "/fingerprints"):
			return 200, fmt.Sprintf(`{"data":{"exactMatches":[{"file":{"id":501,"modId":987,"displayName":"Shared CurseForge","fileName":"shared.jar","releaseType":1,"fileFingerprint":%d,"gameVersions":["1.21.1"]},"mod":{"id":987,"name":"Shared CurseForge","slug":"shared-curseforge","links":{"websiteUrl":"https://example.test/curseforge"},"logo":{"url":"https://example.test/curseforge.png"}}}]}}`, fingerprint)
		case method == http.MethodPost && strings.HasSuffix(url, "/version_files/update"):
			return 200, fmt.Sprintf(`{"%s":{"id":"mr-version","project_id":"mr-project","project_title":"Shared Modrinth","project_slug":"shared-modrinth","version_number":"1.0.0","version_type":"release"}}`, mustTestSHA1(t, jarPath))
		case method == http.MethodPost && strings.HasSuffix(url, "/version_files"):
			return 200, fmt.Sprintf(`{"%s":{"id":"mr-version","project_id":"mr-project","project_title":"Shared Modrinth","project_slug":"shared-modrinth","version_number":"1.0.0","version_type":"release"}}`, mustTestSHA1(t, jarPath))
		case strings.Contains(url, "/project/mr-project"):
			return 200, `{"icon_url":"https://example.test/modrinth.png"}`
		case strings.Contains(url, "/mods/987/files?"):
			return 200, `{"data":[],"pagination":{"index":0,"pageSize":50,"resultCount":0,"totalCount":0}}`
		default:
			t.Fatalf("unexpected overlapping-catalog request: %s %s", method, url)
			return 404, `{}`
		}
	})

	response, err := m.ListAddons(context.Background(), srv.ID)
	if err != nil {
		t.Fatalf("listing addons failed: %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].Source != AddonSourceCurseForge {
		t.Fatalf("expected recorded CurseForge source to win, got %#v", response.Items)
	}
	if response.Items[0].HashSHA512 != sha512Hash {
		t.Fatalf("expected SHA-512 metadata to remain available, got %#v", response.Items[0])
	}

	m2, _, _ := newTestAddonManagerForExistingStore(t, m, addonDir)
	m2.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		t.Fatalf("did not expect catalog request after persisted source resolution: %s %s", method, url)
		return 500, `{}`
	})
	second, err := m2.ListAddons(context.Background(), srv.ID)
	if err != nil {
		t.Fatalf("listing persisted addons failed: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].Source != AddonSourceCurseForge {
		t.Fatalf("expected persisted CurseForge source, got %#v", second.Items)
	}
}

func TestListAddonsUsesRecordedModrinthSourceWhenCatalogsOverlap(t *testing.T) {
	m, srv, addonDir := newTestAddonManager(t)
	jarPath := filepath.Join(addonDir, "shared.jar")
	writeTestJar(t, jarPath, "shared-modrinth-content")
	_, _, fingerprint, err := computeJarMetadata(jarPath)
	if err != nil {
		t.Fatalf("failed to compute jar metadata: %v", err)
	}
	if err := m.recordAddonProvenance(addonDir, "shared.jar", addonProvenanceEntry{
		Source:    AddonSourceModrinth,
		ProjectID: "mr-project",
		VersionID: "mr-version",
	}); err != nil {
		t.Fatalf("failed to record test provenance: %v", err)
	}
	info, err := os.Stat(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	writeTestPersistentCache(t, addonDir, persistentAddonCache{
		SchemaVersion: 1,
		ServerID:      srv.ID,
		Loader:        srv.Loader,
		Version:       srv.Version,
		AddonType:     AddonTypeMod,
		Entries: map[string]persistentAddonEntry{
			"shared.jar": {
				Size:           info.Size(),
				ModifiedAtUnix: info.ModTime().UnixNano(),
				Addon: Addon{
					ID:        "shared.jar",
					FileName:  "shared.jar",
					Source:    AddonSourceCurseForge,
					ProjectID: "987",
					IconURL:   "https://example.test/curseforge.png",
				},
			},
		},
	})

	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		switch {
		case method == http.MethodPost && strings.HasSuffix(url, "/version_files/update"):
			return 200, fmt.Sprintf(`{"%s":{"id":"mr-version","project_id":"mr-project","project_title":"Shared Modrinth","project_slug":"shared-modrinth","version_number":"1.0.0","version_type":"release"}}`, mustTestSHA1(t, jarPath))
		case method == http.MethodPost && strings.HasSuffix(url, "/version_files"):
			return 200, fmt.Sprintf(`{"%s":{"id":"mr-version","project_id":"mr-project","project_title":"Shared Modrinth","project_slug":"shared-modrinth","version_number":"1.0.0","version_type":"release"}}`, mustTestSHA1(t, jarPath))
		case strings.Contains(url, "/project/mr-project"):
			return 200, `{"icon_url":"https://example.test/modrinth.png"}`
		default:
			t.Fatalf("unexpected Modrinth-overlap request: %s %s", method, url)
			return 404, `{}`
		}
	})

	response, err := m.ListAddons(context.Background(), srv.ID)
	if err != nil {
		t.Fatalf("listing addons failed: %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].Source != AddonSourceModrinth {
		t.Fatalf("expected recorded Modrinth source to win, got %#v", response.Items)
	}
	if response.Items[0].CurseFingerprint != fingerprint {
		t.Fatalf("expected CurseForge fingerprint metadata to remain available, got %#v", response.Items[0])
	}
}

func TestAddonProvenanceFollowsDisableAndDeleteLifecycle(t *testing.T) {
	m, srv, addonDir := newTestAddonManager(t)
	fileName := "lifecycle.jar"
	writeTestJar(t, filepath.Join(addonDir, fileName), "lifecycle-content")
	if err := m.recordAddonProvenance(addonDir, fileName, addonProvenanceEntry{
		Source:    AddonSourceModrinth,
		ProjectID: "lifecycle-project",
		VersionID: "lifecycle-version",
	}); err != nil {
		t.Fatalf("failed to record lifecycle provenance: %v", err)
	}
	m.storeCachedAddonList(srv.ID, "lifecycle", &ListResponse{Items: []Addon{{FileName: fileName}}})

	if err := m.SetAddonDisabled(srv.ID, fileName, true); err != nil {
		t.Fatalf("failed to disable addon: %v", err)
	}
	if _, ok := m.getCachedAddonList(srv.ID, "lifecycle"); ok {
		t.Fatal("expected disabling an addon to invalidate its list cache")
	}
	provenance, err := readAddonProvenance(addonDir)
	if err != nil {
		t.Fatalf("failed to read disabled provenance: %v", err)
	}
	if _, ok := provenance.Entries[fileName]; ok {
		t.Fatalf("did not expect old filename in provenance: %#v", provenance.Entries)
	}
	if _, ok := provenance.Entries[fileName+".disabled"]; !ok {
		t.Fatalf("expected disabled filename in provenance: %#v", provenance.Entries)
	}

	if err := m.SetAddonDisabled(srv.ID, fileName+".disabled", false); err != nil {
		t.Fatalf("failed to re-enable addon: %v", err)
	}
	provenance, err = readAddonProvenance(addonDir)
	if err != nil {
		t.Fatalf("failed to read re-enabled provenance: %v", err)
	}
	if _, ok := provenance.Entries[fileName]; !ok {
		t.Fatalf("expected enabled filename in provenance: %#v", provenance.Entries)
	}

	if err := m.DeleteAddon(srv.ID, fileName); err != nil {
		t.Fatalf("failed to delete addon: %v", err)
	}
	provenance, err = readAddonProvenance(addonDir)
	if err != nil {
		t.Fatalf("failed to read deleted provenance: %v", err)
	}
	if len(provenance.Entries) != 0 {
		t.Fatalf("expected deleted addon provenance to be removed: %#v", provenance.Entries)
	}
}

func TestUpdateAddonPreservesCurseForgeProvenance(t *testing.T) {
	m, srv, addonDir := newTestAddonManager(t)
	if err := m.SetCustomCurseForgeAPIKey("test-key"); err != nil {
		t.Fatalf("failed to set CurseForge test key: %v", err)
	}
	oldFileName := "old.jar"
	writeTestJar(t, filepath.Join(addonDir, oldFileName), "old-content")
	if err := m.recordAddonProvenance(addonDir, oldFileName, addonProvenanceEntry{
		Source:    AddonSourceCurseForge,
		ProjectID: "123",
		VersionID: "401",
		FileID:    401,
	}); err != nil {
		t.Fatalf("failed to record old addon provenance: %v", err)
	}
	_, _, oldFingerprint, err := computeJarMetadata(filepath.Join(addonDir, oldFileName))
	if err != nil {
		t.Fatalf("failed to compute old addon fingerprint: %v", err)
	}

	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		switch {
		case method == http.MethodPost && strings.HasSuffix(url, "/fingerprints"):
			return 200, fmt.Sprintf(`{"data":{"exactMatches":[{"file":{"id":401,"modId":123,"displayName":"Old","fileName":"old.jar","releaseType":1,"fileFingerprint":%d,"gameVersions":["1.21.1"]},"mod":{"id":123,"name":"Updateable CurseForge","slug":"updateable","links":{"websiteUrl":"https://example.test/updateable"},"logo":{"url":"https://example.test/updateable.png"}}}]}}`, oldFingerprint)
		case strings.Contains(url, "/mods/123/files?"):
			return 200, `{"data":[{"id":401,"modId":123,"displayName":"Old","fileName":"old.jar","releaseType":1,"fileDate":"2025-01-01T00:00:00Z","gameVersions":["1.21.1","Server"]},{"id":402,"modId":123,"displayName":"New","fileName":"new.jar","downloadUrl":"https://example.test/new.jar","releaseType":1,"fileDate":"2026-01-01T00:00:00Z","gameVersions":["1.21.1","Server"]}],"pagination":{"index":0,"pageSize":50,"resultCount":2,"totalCount":2}}`
		case url == "https://example.test/new.jar":
			return 200, "new-content"
		default:
			return 404, `{}`
		}
	})

	if err := m.UpdateAddon(context.Background(), srv.ID, oldFileName, false); err != nil {
		t.Fatalf("expected CurseForge update to succeed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(addonDir, "new.jar")); err != nil {
		t.Fatalf("expected updated file to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(addonDir, oldFileName)); !os.IsNotExist(err) {
		t.Fatalf("expected old file to be removed, got err=%v", err)
	}
	provenance, err := readAddonProvenance(addonDir)
	if err != nil {
		t.Fatalf("failed to read updated provenance: %v", err)
	}
	if _, ok := provenance.Entries[oldFileName]; ok {
		t.Fatalf("did not expect old provenance after update: %#v", provenance.Entries)
	}
	if entry, ok := provenance.Entries["new.jar"]; !ok || entry.Source != AddonSourceCurseForge || entry.FileID != 402 {
		t.Fatalf("expected updated file to retain CurseForge provenance: %#v", provenance.Entries)
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
	t.Cleanup(func() { _ = store.Close() })
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

	serverManager := server.NewManager(filepath.Join(tempDir, "servers"), store, nil)
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

func TestPreviewModrinthIncludesNestedRequiredDependenciesAndSkipsOptional(t *testing.T) {
	m, srv, addonDir := newTestAddonManager(t)
	downloadRequests := 0
	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		if strings.HasPrefix(url, "https://example.test/") {
			downloadRequests++
			return 200, "should-not-be-downloaded"
		}
		switch url {
		case "https://api.modrinth.com/v2/version/root-version":
			return 200, `{"id":"root-version","project_id":"root","files":[{"url":"https://example.test/root.jar","filename":"root.jar","primary":true}],"dependencies":[{"version_id":"dep-version","project_id":"dep","dependency_type":"required"},{"version_id":"optional-version","project_id":"optional","dependency_type":"optional"}]}`
		case "https://api.modrinth.com/v2/version/dep-version":
			return 200, `{"id":"dep-version","project_id":"dep","project_title":"Dependency One","version_number":"2.0.0","files":[{"url":"https://example.test/dep.jar","filename":"dep.jar","primary":true}],"dependencies":[{"version_id":"nested-version","project_id":"nested","dependency_type":"required"}]}`
		case "https://api.modrinth.com/v2/version/nested-version":
			return 200, `{"id":"nested-version","project_id":"nested","project_title":"Nested Dependency","version_number":"1.2.3","files":[{"url":"https://example.test/nested.jar","filename":"nested.jar","primary":true}]}`
		case "https://api.modrinth.com/v2/project/dep":
			return 200, `{"title":"Dependency One","icon_url":"https://cdn.example.test/dep.png"}`
		case "https://api.modrinth.com/v2/project/nested":
			return 200, `{"title":"Nested Dependency","icon_url":"https://cdn.example.test/nested.png"}`
		default:
			t.Fatalf("unexpected preview request: %s %s", method, url)
			return 404, `{}`
		}
	})

	preview, err := m.PreviewInstallAddon(context.Background(), srv.ID, InstallPreviewRequest{
		Source:    AddonSourceModrinth,
		ProjectID: "root",
		VersionID: "root-version",
	})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if downloadRequests != 0 {
		t.Fatalf("preview downloaded %d files", downloadRequests)
	}
	if len(preview.Dependencies) != 2 {
		t.Fatalf("expected two required dependencies, got %#v", preview.Dependencies)
	}
	if preview.Dependencies[0].ProjectID != "nested" || preview.Dependencies[1].ProjectID != "dep" {
		t.Fatalf("expected nested dependency order, got %#v", preview.Dependencies)
	}
	if preview.Dependencies[0].Name != "Nested Dependency" || preview.Dependencies[1].Name != "Dependency One" {
		t.Fatalf("expected dependency names, got %#v", preview.Dependencies)
	}
	if preview.Dependencies[0].IconURL != "https://cdn.example.test/nested.png" || preview.Dependencies[1].IconURL != "https://cdn.example.test/dep.png" {
		t.Fatalf("expected Modrinth dependency icons, got %#v", preview.Dependencies)
	}
	for _, dependency := range preview.Dependencies {
		if dependency.Source != AddonSourceModrinth || dependency.Filename == "" || dependency.VersionLabel == "" {
			t.Fatalf("expected complete Modrinth dependency metadata, got %#v", dependency)
		}
	}
	if _, err := os.Stat(addonDir); err != nil {
		t.Fatalf("expected addon directory to remain available: %v", err)
	}
}

func TestPreviewCurseForgeIncludesNestedRequiredDependenciesWithNames(t *testing.T) {
	m, srv, _ := newTestAddonManager(t)
	if err := m.SetCustomCurseForgeAPIKey("test-key"); err != nil {
		t.Fatalf("failed to set CurseForge test key: %v", err)
	}
	downloadRequests := 0
	var fileRequests []string
	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		if strings.HasPrefix(url, "https://example.test/") {
			downloadRequests++
			return 200, "should-not-be-downloaded"
		}
		switch {
		case strings.Contains(url, "/mods/100/files?"):
			fileRequests = append(fileRequests, url)
			return 200, `{"data":[{"id":1001,"modId":100,"displayName":"Root 1.0","fileName":"root.jar","gameVersions":["1.21.1","Client","Server"],"downloadUrl":"https://example.test/root.jar","dependencies":[{"modId":200,"fileId":2001,"relationType":3},{"modId":999,"fileId":9991,"relationType":2}]}],"pagination":{"index":0,"pageSize":50,"resultCount":1,"totalCount":1}}`
		case strings.Contains(url, "/mods/200/files?"):
			fileRequests = append(fileRequests, url)
			return 200, `{"data":[{"id":2001,"modId":200,"displayName":"Dependency 2.0","fileName":"dependency.jar","gameVersions":["1.21.1","Server"],"downloadUrl":"https://example.test/dependency.jar","dependencies":[{"modId":300,"fileId":3001,"relationType":3}]}],"pagination":{"index":0,"pageSize":50,"resultCount":1,"totalCount":1}}`
		case strings.Contains(url, "/mods/300/files?"):
			fileRequests = append(fileRequests, url)
			return 200, `{"data":[{"id":3001,"modId":300,"displayName":"Nested 3.0","fileName":"nested.jar","gameVersions":["1.21.1","Server"],"downloadUrl":"https://example.test/nested.jar"}],"pagination":{"index":0,"pageSize":50,"resultCount":1,"totalCount":1}}`
		case method == http.MethodPost && strings.HasSuffix(url, "/v1/mods"):
			return 200, `{"data":[{"id":200,"name":"Library Project","logo":{"url":"https://cdn.example.test/library.png"}},{"id":300,"name":"Nested Project","logo":{"url":"https://cdn.example.test/nested-project.png"}}]}`
		default:
			t.Fatalf("unexpected CurseForge preview request: %s %s", method, url)
			return 404, `{}`
		}
	})

	preview, err := m.PreviewInstallAddon(context.Background(), srv.ID, InstallPreviewRequest{
		Source:    AddonSourceCurseForge,
		ProjectID: "100",
		FileID:    1001,
	})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if downloadRequests != 0 {
		t.Fatalf("preview downloaded %d files", downloadRequests)
	}
	if len(preview.Dependencies) != 2 {
		t.Fatalf("expected two required dependencies, got %#v", preview.Dependencies)
	}
	if preview.Dependencies[0].ProjectID != "300" || preview.Dependencies[1].ProjectID != "200" {
		t.Fatalf("expected nested dependency order, got %#v", preview.Dependencies)
	}
	if preview.Dependencies[0].Name != "Nested Project" || preview.Dependencies[1].Name != "Library Project" {
		t.Fatalf("expected grouped CurseForge names, got %#v", preview.Dependencies)
	}
	if preview.Dependencies[0].IconURL != "https://cdn.example.test/nested-project.png" || preview.Dependencies[1].IconURL != "https://cdn.example.test/library.png" {
		t.Fatalf("expected CurseForge dependency icons, got %#v", preview.Dependencies)
	}
	for _, url := range fileRequests {
		if !strings.Contains(url, "modLoaderType=4") {
			t.Fatalf("expected Fabric loader filter in %s", url)
		}
	}
}

func TestPreviewExcludesInstalledDependencyBySourceAndProject(t *testing.T) {
	m, srv, addonDir := newTestAddonManager(t)
	installedPath := filepath.Join(addonDir, "dependency.jar")
	writeTestJar(t, installedPath, "installed-dependency")
	if err := m.recordAddonProvenance(addonDir, "dependency.jar", addonProvenanceEntry{
		Source:    AddonSourceModrinth,
		ProjectID: "dep",
		VersionID: "old-version",
	}); err != nil {
		t.Fatalf("failed to record provenance: %v", err)
	}

	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		switch url {
		case "https://api.modrinth.com/v2/version/root-version":
			return 200, `{"id":"root-version","project_id":"root","files":[{"url":"https://example.test/root.jar","filename":"root.jar","primary":true}],"dependencies":[{"version_id":"dep-version","project_id":"dep","dependency_type":"required"},{"version_id":"other-version","project_id":"other","dependency_type":"required"}]}`
		case "https://api.modrinth.com/v2/version/dep-version":
			return 200, `{"id":"dep-version","project_id":"dep","files":[{"url":"https://example.test/dep.jar","filename":"dep.jar","primary":true}]}`
		case "https://api.modrinth.com/v2/version/other-version":
			return 200, `{"id":"other-version","project_id":"other","files":[{"url":"https://example.test/other.jar","filename":"other.jar","primary":true}]}`
		case "https://api.modrinth.com/v2/project/dep":
			return 200, `{"title":"Dependency"}`
		case "https://api.modrinth.com/v2/project/other":
			return 200, `{"title":"Other dependency"}`
		case "https://api.modrinth.com/v2/version_files":
			return 200, `{}`
		case "https://api.modrinth.com/v2/version_files/update":
			return 200, `{}`
		case "https://api.curseforge.com/v1/fingerprints":
			return 200, `{}`
		default:
			t.Fatalf("unexpected installed-dependency preview request: %s %s", method, url)
			return 404, `{}`
		}
	})

	preview, err := m.PreviewInstallAddon(context.Background(), srv.ID, InstallPreviewRequest{
		Source:    AddonSourceModrinth,
		ProjectID: "root",
		VersionID: "root-version",
	})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if len(preview.Dependencies) != 1 || preview.Dependencies[0].ProjectID != "other" {
		t.Fatalf("expected only uninstalled dependency, got %#v", preview.Dependencies)
	}
}

func TestPreviewCurseForgeFallsBackToProjectIDWhenMetadataMissing(t *testing.T) {
	m, srv, _ := newTestAddonManager(t)
	if err := m.SetCustomCurseForgeAPIKey("test-key"); err != nil {
		t.Fatalf("failed to set CurseForge test key: %v", err)
	}
	m.httpClient = testAddonHTTPClient(func(method, url string) (int, string) {
		switch {
		case strings.Contains(url, "/mods/100/files?"):
			return 200, `{"data":[{"id":1001,"modId":100,"fileName":"root.jar","gameVersions":["1.21.1","Server"],"dependencies":[{"modId":200,"fileId":2001,"relationType":3}]}],"pagination":{"index":0,"pageSize":50,"resultCount":1,"totalCount":1}}`
		case strings.Contains(url, "/mods/200/files?"):
			return 200, `{"data":[{"id":2001,"modId":200,"fileName":"dependency.jar","gameVersions":["1.21.1","Server"]}],"pagination":{"index":0,"pageSize":50,"resultCount":1,"totalCount":1}}`
		case method == http.MethodPost && strings.HasSuffix(url, "/v1/mods"):
			return 200, `{"data":[]}`
		default:
			t.Fatalf("unexpected fallback preview request: %s %s", method, url)
			return 404, `{}`
		}
	})

	preview, err := m.PreviewInstallAddon(context.Background(), srv.ID, InstallPreviewRequest{
		Source:    AddonSourceCurseForge,
		ProjectID: "100",
		FileID:    1001,
	})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if len(preview.Dependencies) != 1 || preview.Dependencies[0].Name != "200" {
		t.Fatalf("expected project ID fallback, got %#v", preview.Dependencies)
	}
}
