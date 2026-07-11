package loader

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"naviserver/internal/domain"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const BedrockDataBaseURL = "https://raw.githubusercontent.com/EndstoneMC/bedrock-server-data/v2"

var ErrBedrockPlatformUnsupported = errors.New("Bedrock Dedicated Server is only supported on 64-bit Linux and Windows")

type bedrockVersionChannel struct {
	Latest   string   `json:"latest"`
	Versions []string `json:"versions"`
}

type bedrockVersionRegistry struct {
	Release bedrockVersionChannel `json:"release"`
	Preview bedrockVersionChannel `json:"preview"`
}

type bedrockBinary struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

type bedrockVersionMetadata struct {
	Version string `json:"version"`
	Binary  struct {
		Windows bedrockBinary `json:"windows"`
		Linux   bedrockBinary `json:"linux"`
	} `json:"binary"`
}

type bedrockHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type BedrockLoader struct {
	client      bedrockHTTPClient
	dataBaseURL string
	goos        string
	goarch      string
}

func NewBedrockLoader() *BedrockLoader {
	return &BedrockLoader{
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
		dataBaseURL: BedrockDataBaseURL,
		goos:        runtime.GOOS,
		goarch:      runtime.GOARCH,
	}
}

func IsBedrockPlatformSupported(goos, goarch string) bool {
	return (goos == "linux" || goos == "windows") && goarch == "amd64"
}

func (l *BedrockLoader) GetSupportedVersions(options LoaderOptions) ([]string, error) {
	if err := l.validatePlatform(); err != nil {
		return nil, err
	}

	registry, err := l.fetchRegistry()
	if err != nil {
		return nil, err
	}

	return bedrockVersions(registry, options.IncludeSnapshots), nil
}

func (l *BedrockLoader) GetMetadata(options LoaderOptions) (*LoaderMetadata, error) {
	if err := l.validatePlatform(); err != nil {
		return nil, err
	}
	registry, err := l.fetchRegistry()
	if err != nil {
		return nil, err
	}

	return &LoaderMetadata{
		LatestVersion:     registry.Release.Latest,
		MinecraftVersions: bedrockVersions(registry, options.IncludeSnapshots),
	}, nil
}

func bedrockVersions(registry *bedrockVersionRegistry, includePreviews bool) []string {
	versions := append([]string(nil), registry.Release.Versions...)
	if includePreviews {
		versions = append(versions, registry.Preview.Versions...)
	}
	return versions
}

func (l *BedrockLoader) Load(options LoaderOptions, destDir string, progressChan chan<- domain.ProgressEvent) (string, error) {
	if err := l.validatePlatform(); err != nil {
		return "", err
	}

	registry, err := l.fetchRegistry()
	if err != nil {
		return "", fmt.Errorf("could not get Bedrock versions: %w", err)
	}

	version := strings.TrimSpace(options.MCVersion)
	if version == "" {
		version = registry.Release.Latest
	}

	channel := bedrockVersionChannelName(registry, version)
	if channel == "" {
		return "", fmt.Errorf("version %s not found in Bedrock server data", version)
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: fmt.Sprintf("Getting Bedrock %s metadata...", version)}
	}

	metadata, err := l.fetchMetadata(channel, version)
	if err != nil {
		return "", fmt.Errorf("could not get Bedrock %s metadata: %w", version, err)
	}
	if metadata.Version != version {
		return "", fmt.Errorf("Bedrock metadata version mismatch: requested %s, got %s", version, metadata.Version)
	}

	binary, err := l.binaryForPlatform(metadata)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(binary.URL) == "" || strings.TrimSpace(binary.SHA256) == "" {
		return "", fmt.Errorf("Bedrock metadata for %s does not include a complete %s binary", version, l.goos)
	}

	archivePath, err := l.downloadArchive(binary, version, progressChan)
	if err != nil {
		return "", err
	}
	defer os.Remove(archivePath)

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Extracting Bedrock Dedicated Server..."}
	}
	if err := l.installArchive(archivePath, destDir); err != nil {
		return "", fmt.Errorf("could not install Bedrock Dedicated Server: %w", err)
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Bedrock installation completed.", Progress: 100}
	}
	return version, nil
}

func (l *BedrockLoader) validatePlatform() error {
	if IsBedrockPlatformSupported(l.goos, l.goarch) {
		return nil
	}
	return fmt.Errorf(
		"%w (current platform: %s/%s)",
		ErrBedrockPlatformUnsupported,
		l.goos,
		l.goarch,
	)
}

func (l *BedrockLoader) fetchRegistry() (*bedrockVersionRegistry, error) {
	var registry bedrockVersionRegistry
	if err := l.fetchJSON(l.dataBaseURL+"/versions.json", &registry); err != nil {
		return nil, err
	}
	if registry.Release.Latest == "" || len(registry.Release.Versions) == 0 {
		return nil, fmt.Errorf("Bedrock version registry does not contain releases")
	}
	return &registry, nil
}

func (l *BedrockLoader) fetchMetadata(channel, version string) (*bedrockVersionMetadata, error) {
	var metadata bedrockVersionMetadata
	url := fmt.Sprintf("%s/%s/%s/metadata.json", l.dataBaseURL, channel, version)
	if err := l.fetchJSON(url, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

func (l *BedrockLoader) fetchJSON(url string, target any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "NaviServer-Bedrock-Loader")

	resp, err := l.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request to %s returned %s", url, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("invalid JSON from %s: %w", url, err)
	}
	return nil
}

func (l *BedrockLoader) binaryForPlatform(metadata *bedrockVersionMetadata) (bedrockBinary, error) {
	switch l.goos {
	case "windows":
		return metadata.Binary.Windows, nil
	case "linux":
		return metadata.Binary.Linux, nil
	default:
		return bedrockBinary{}, fmt.Errorf("Bedrock has no binary for operating system %s", l.goos)
	}
}

func (l *BedrockLoader) downloadArchive(binary bedrockBinary, version string, progressChan chan<- domain.ProgressEvent) (string, error) {
	req, err := http.NewRequest(http.MethodGet, binary.URL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "NaviServer-Bedrock-Loader")

	resp, err := l.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not download Bedrock %s: %w", version, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Bedrock download returned %s", resp.Status)
	}

	tempFile, err := os.CreateTemp("", "naviserver-bedrock-*.zip")
	if err != nil {
		return "", err
	}
	tempPath := tempFile.Name()
	removeTemp := true
	defer func() {
		_ = tempFile.Close()
		if removeTemp {
			_ = os.Remove(tempPath)
		}
	}()

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: fmt.Sprintf("Downloading Bedrock Dedicated Server %s...", version)}
	}

	hash := sha256.New()
	reader := &ProgressReader{
		Reader:       resp.Body,
		Total:        resp.ContentLength,
		ProgressChan: progressChan,
		Message:      "Downloading Bedrock Dedicated Server",
	}
	if _, err := io.Copy(io.MultiWriter(tempFile, hash), reader); err != nil {
		return "", fmt.Errorf("could not save Bedrock archive: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return "", fmt.Errorf("could not close Bedrock archive: %w", err)
	}

	actualHash := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actualHash, strings.TrimSpace(binary.SHA256)) {
		return "", fmt.Errorf("Bedrock archive checksum mismatch: expected %s, got %s", binary.SHA256, actualHash)
	}

	removeTemp = false
	return tempPath, nil
}

func (l *BedrockLoader) installArchive(archivePath, destDir string) error {
	parent := filepath.Dir(filepath.Clean(destDir))
	stagingDir, err := os.MkdirTemp(parent, ".naviserver-bedrock-install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stagingDir)

	if err := extractBedrockZip(archivePath, stagingDir); err != nil {
		return err
	}

	binaryName := "bedrock_server"
	if l.goos == "windows" {
		binaryName = "bedrock_server.exe"
	}
	stagedBinary := filepath.Join(stagingDir, binaryName)
	info, err := os.Stat(stagedBinary)
	if err != nil || info.IsDir() {
		return fmt.Errorf("archive does not contain %s", binaryName)
	}

	isUpdate := bedrockInstallationExists(destDir)
	if err := mergeBedrockInstallation(stagingDir, destDir, isUpdate); err != nil {
		return err
	}
	if l.goos == "linux" {
		if err := os.Chmod(filepath.Join(destDir, binaryName), 0755); err != nil {
			return fmt.Errorf("could not mark %s executable: %w", binaryName, err)
		}
	}
	return nil
}

func bedrockVersionChannelName(registry *bedrockVersionRegistry, version string) string {
	for _, candidate := range registry.Release.Versions {
		if candidate == version {
			return "release"
		}
	}
	for _, candidate := range registry.Preview.Versions {
		if candidate == version {
			return "preview"
		}
	}
	return ""
}

func extractBedrockZip(src, dest string) error {
	archive, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer archive.Close()

	destClean := filepath.Clean(dest)
	for _, entry := range archive.File {
		name := filepath.Clean(filepath.FromSlash(entry.Name))
		if name == "." {
			continue
		}
		if filepath.IsAbs(name) {
			return fmt.Errorf("illegal zip entry: %s", entry.Name)
		}
		target := filepath.Join(destClean, name)
		if target != destClean && !strings.HasPrefix(target, destClean+string(os.PathSeparator)) {
			return fmt.Errorf("illegal zip entry: %s", entry.Name)
		}

		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		reader, err := entry.Open()
		if err != nil {
			return err
		}
		mode := entry.Mode().Perm()
		if mode == 0 {
			mode = 0644
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			reader.Close()
			return err
		}
		_, copyErr := io.Copy(output, reader)
		closeErr := output.Close()
		reader.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func bedrockInstallationExists(destDir string) bool {
	for _, name := range []string{"bedrock_server", "bedrock_server.exe", "server.properties"} {
		if _, err := os.Stat(filepath.Join(destDir, name)); err == nil {
			return true
		}
	}
	return false
}

func mergeBedrockInstallation(src, dest string, preserveData bool) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(src, path)
		if err != nil || relative == "." {
			return err
		}
		relative = filepath.Clean(relative)
		if preserveData && isMutableBedrockPath(relative) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		target := filepath.Join(dest, relative)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		input, err := os.Open(path)
		if err != nil {
			return err
		}
		temp, err := os.CreateTemp(filepath.Dir(target), ".bedrock-file-*")
		if err != nil {
			input.Close()
			return err
		}
		tempName := temp.Name()
		_, copyErr := io.Copy(temp, input)
		input.Close()
		closeErr := temp.Close()
		if copyErr != nil {
			_ = os.Remove(tempName)
			return copyErr
		}
		if closeErr != nil {
			_ = os.Remove(tempName)
			return closeErr
		}
		if err := os.Chmod(tempName, info.Mode().Perm()); err != nil {
			_ = os.Remove(tempName)
			return err
		}
		if err := replaceBedrockFile(tempName, target); err != nil {
			_ = os.Remove(tempName)
			return err
		}
		return nil
	})
}

func replaceBedrockFile(tempName, target string) error {
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return os.Rename(tempName, target)
	} else if err != nil {
		return err
	}

	backupFile, err := os.CreateTemp(filepath.Dir(target), ".bedrock-previous-*")
	if err != nil {
		return err
	}
	backupName := backupFile.Name()
	if err := backupFile.Close(); err != nil {
		_ = os.Remove(backupName)
		return err
	}
	if err := os.Remove(backupName); err != nil {
		return err
	}
	if err := os.Rename(target, backupName); err != nil {
		return err
	}
	if err := os.Rename(tempName, target); err != nil {
		_ = os.Rename(backupName, target)
		return err
	}
	_ = os.Remove(backupName)
	return nil
}

func isMutableBedrockPath(relative string) bool {
	relative = filepath.ToSlash(filepath.Clean(relative))
	switch relative {
	case "server.properties", "allowlist.json", "permissions.json":
		return true
	}
	return relative == "worlds" || strings.HasPrefix(relative, "worlds/")
}
