package loader

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestBedrockPlatformSupport(t *testing.T) {
	tests := []struct {
		goos      string
		goarch    string
		supported bool
	}{
		{goos: "linux", goarch: "amd64", supported: true},
		{goos: "windows", goarch: "amd64", supported: true},
		{goos: "darwin", goarch: "amd64", supported: false},
		{goos: "linux", goarch: "arm64", supported: false},
	}

	for _, test := range tests {
		if got := IsBedrockPlatformSupported(test.goos, test.goarch); got != test.supported {
			t.Fatalf("support for %s/%s = %v, want %v", test.goos, test.goarch, got, test.supported)
		}
	}
}

func TestAvailableLoadersHideBedrockOnUnsupportedPlatforms(t *testing.T) {
	macLoaders := availableLoadersForPlatform("darwin", "amd64")
	if containsVersion(macLoaders, "bedrock") {
		t.Fatalf("Bedrock unexpectedly available on macOS: %v", macLoaders)
	}
	linuxLoaders := availableLoadersForPlatform("linux", "amd64")
	if !containsVersion(linuxLoaders, "bedrock") {
		t.Fatalf("Bedrock missing on Linux amd64: %v", linuxLoaders)
	}
}

func TestBedrockLoaderReturnsTypedPlatformError(t *testing.T) {
	loader := newTestBedrockLoader("https://example.invalid", "darwin")
	_, err := loader.GetSupportedVersions(LoaderOptions{})
	if !errors.Is(err, ErrBedrockPlatformUnsupported) {
		t.Fatalf("expected typed platform error, got %v", err)
	}
}

func TestBedrockVersionsIncludePreviewsOnRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/versions.json" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(bedrockVersionRegistry{
			Release: bedrockVersionChannel{Latest: "1.2.0", Versions: []string{"1.2.0", "1.1.0"}},
			Preview: bedrockVersionChannel{Latest: "1.3.0-preview.1", Versions: []string{"1.3.0-preview.1"}},
		})
	}))
	defer server.Close()

	loader := newTestBedrockLoader(server.URL, "linux")
	releases, err := loader.GetSupportedVersions(LoaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(releases, ",") != "1.2.0,1.1.0" {
		t.Fatalf("unexpected releases: %v", releases)
	}

	allVersions, err := loader.GetSupportedVersions(LoaderOptions{IncludeSnapshots: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(allVersions, ",") != "1.2.0,1.1.0,1.3.0-preview.1" {
		t.Fatalf("unexpected versions with previews: %v", allVersions)
	}

	metadata, err := loader.GetMetadata(LoaderOptions{IncludeSnapshots: true})
	if err != nil {
		t.Fatal(err)
	}
	if metadata.LatestVersion != "1.2.0" {
		t.Fatalf("latest stable = %s, want 1.2.0", metadata.LatestVersion)
	}
}

func TestBedrockLoadVerifiesAndInstallsLinuxArchive(t *testing.T) {
	archive := makeBedrockArchive(t, map[string]string{
		"bedrock_server":    "linux-binary",
		"server.properties": "server-name=Dedicated Server\n",
		"allowlist.json":    "[]",
	})
	server := newBedrockDataServer(t, archive, false)
	defer server.Close()

	loader := newTestBedrockLoader(server.URL, "linux")
	dest := t.TempDir()
	version, err := loader.Load(LoaderOptions{MCVersion: "1.2.0"}, dest, nil)
	if err != nil {
		t.Fatal(err)
	}
	if version != "1.2.0" {
		t.Fatalf("resolved version = %s", version)
	}

	binaryPath := filepath.Join(dest, "bedrock_server")
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "linux-binary" {
		t.Fatalf("unexpected binary contents: %q", data)
	}
	info, err := os.Stat(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0111 == 0 {
		t.Fatalf("bedrock_server is not executable: %s", info.Mode())
	}
}

func TestBedrockLoadInstallsWindowsExecutable(t *testing.T) {
	archive := makeBedrockArchive(t, map[string]string{
		"bedrock_server.exe": "windows-binary",
		"server.properties":  "server-name=Dedicated Server\n",
	})
	server := newBedrockDataServer(t, archive, false)
	defer server.Close()

	loader := newTestBedrockLoader(server.URL, "windows")
	dest := t.TempDir()
	if _, err := loader.Load(LoaderOptions{MCVersion: "1.2.0"}, dest, nil); err != nil {
		t.Fatal(err)
	}
	assertTestFile(t, filepath.Join(dest, "bedrock_server.exe"), "windows-binary")
}

func TestBedrockUpdatePreservesMutableData(t *testing.T) {
	archive := makeBedrockArchive(t, map[string]string{
		"bedrock_server":           "new-binary",
		"server.properties":        "server-name=Archive Default\n",
		"allowlist.json":           "[]",
		"permissions.json":         "[]",
		"worlds/Bedrock level/db":  "new-world",
		"resource_packs/base.json": "new-base-pack",
	})
	server := newBedrockDataServer(t, archive, false)
	defer server.Close()

	dest := t.TempDir()
	writeTestFile(t, filepath.Join(dest, "bedrock_server"), "old-binary")
	writeTestFile(t, filepath.Join(dest, "server.properties"), "server-name=My Server\n")
	writeTestFile(t, filepath.Join(dest, "allowlist.json"), "[\"player\"]")
	writeTestFile(t, filepath.Join(dest, "permissions.json"), "[{\"xuid\":\"1\"}]")
	writeTestFile(t, filepath.Join(dest, "worlds/Bedrock level/db"), "my-world")
	writeTestFile(t, filepath.Join(dest, "resource_packs/custom.json"), "custom-pack")

	loader := newTestBedrockLoader(server.URL, "linux")
	if _, err := loader.Load(LoaderOptions{MCVersion: "1.2.0"}, dest, nil); err != nil {
		t.Fatal(err)
	}

	assertTestFile(t, filepath.Join(dest, "bedrock_server"), "new-binary")
	assertTestFile(t, filepath.Join(dest, "server.properties"), "server-name=My Server\n")
	assertTestFile(t, filepath.Join(dest, "allowlist.json"), "[\"player\"]")
	assertTestFile(t, filepath.Join(dest, "permissions.json"), "[{\"xuid\":\"1\"}]")
	assertTestFile(t, filepath.Join(dest, "worlds/Bedrock level/db"), "my-world")
	assertTestFile(t, filepath.Join(dest, "resource_packs/base.json"), "new-base-pack")
	assertTestFile(t, filepath.Join(dest, "resource_packs/custom.json"), "custom-pack")
}

func TestBedrockLoadRejectsChecksumMismatch(t *testing.T) {
	archive := makeBedrockArchive(t, map[string]string{"bedrock_server": "binary"})
	server := newBedrockDataServer(t, archive, true)
	defer server.Close()

	loader := newTestBedrockLoader(server.URL, "linux")
	_, err := loader.Load(LoaderOptions{MCVersion: "1.2.0"}, t.TempDir(), nil)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
}

func TestExtractBedrockZipRejectsTraversal(t *testing.T) {
	archive := makeBedrockArchive(t, map[string]string{"../outside": "bad"})
	archivePath := filepath.Join(t.TempDir(), "bad.zip")
	if err := os.WriteFile(archivePath, archive, 0600); err != nil {
		t.Fatal(err)
	}

	err := extractBedrockZip(archivePath, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "illegal zip entry") {
		t.Fatalf("expected traversal error, got %v", err)
	}
}

func newTestBedrockLoader(baseURL, goos string) *BedrockLoader {
	return &BedrockLoader{
		client:      http.DefaultClient,
		dataBaseURL: baseURL,
		goos:        goos,
		goarch:      "amd64",
	}
}

func newBedrockDataServer(t *testing.T, archive []byte, invalidHash bool) *httptest.Server {
	t.Helper()
	hash := sha256.Sum256(archive)
	hashString := hex.EncodeToString(hash[:])
	if invalidHash {
		hashString = strings.Repeat("0", 64)
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions.json":
			_ = json.NewEncoder(w).Encode(bedrockVersionRegistry{
				Release: bedrockVersionChannel{Latest: "1.2.0", Versions: []string{"1.2.0"}},
				Preview: bedrockVersionChannel{Latest: "1.3.0-preview.1", Versions: []string{"1.3.0-preview.1"}},
			})
		case "/release/1.2.0/metadata.json":
			metadata := bedrockVersionMetadata{Version: "1.2.0"}
			metadata.Binary.Linux = bedrockBinary{URL: server.URL + "/bedrock.zip", SHA256: hashString}
			metadata.Binary.Windows = metadata.Binary.Linux
			_ = json.NewEncoder(w).Encode(metadata)
		case "/bedrock.zip":
			w.Header().Set("Content-Type", "application/zip")
			w.Header().Set("Content-Length", fmtInt(len(archive)))
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	return server
}

func makeBedrockArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, contents := range files {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0644)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(entry, contents); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertTestFile(t *testing.T, path, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != expected {
		t.Fatalf("%s = %q, want %q", path, data, expected)
	}
}

func fmtInt(value int) string {
	return strconv.Itoa(value)
}

func containsVersion(versions []string, expected string) bool {
	for _, version := range versions {
		if version == expected {
			return true
		}
	}
	return false
}
