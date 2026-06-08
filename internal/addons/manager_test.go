package addons

import (
	"naviserver/internal/domain"
	"naviserver/internal/server"
	"naviserver/internal/storage"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
