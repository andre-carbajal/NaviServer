package loader

import (
	"encoding/json"
	"fmt"
	"io"
	"naviserver/internal/domain"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

const QuiltMetaURL = "https://meta.quiltmc.org/v3/versions/"

type QuiltLoader struct{}

func NewQuiltLoader() *QuiltLoader { return &QuiltLoader{} }

func (l *QuiltLoader) GetSupportedVersions(options LoaderOptions) ([]string, error) {
	return l.getGameVersions(options.IncludeUnstable)
}

func (l *QuiltLoader) GetMetadata(options LoaderOptions) (*LoaderMetadata, error) {
	game, err := l.getGameVersions(options.IncludeUnstable)
	if err != nil {
		return nil, err
	}
	md := &LoaderMetadata{MinecraftVersions: game}
	if len(game) > 0 {
		md.LatestVersion = game[0]
	}
	loaderVersions, _ := l.getLoaderVersions(options.IncludeUnstable)
	installerVersions, _ := l.getInstallerVersions(options.IncludeUnstable)
	md.LoaderVersions = loaderVersions
	md.InstallerVersions = installerVersions
	return md, nil
}

func (l *QuiltLoader) Load(options LoaderOptions, destDir string, progressChan chan<- domain.ProgressEvent) (string, error) {
	gameVersions, err := l.getGameVersions(options.IncludeUnstable)
	if err != nil || len(gameVersions) == 0 {
		return "", fmt.Errorf("error getting Quilt versions: %w", err)
	}
	mc := options.MCVersion
	if mc == "" {
		mc = gameVersions[0]
	}

	loaderVersions, err := l.getLoaderVersions(options.IncludeUnstable)
	if err != nil || len(loaderVersions) == 0 {
		return "", fmt.Errorf("error getting Quilt loader versions: %w", err)
	}
	loaderVersion := loaderVersions[0]
	if options.LoaderVersion != "" {
		loaderVersion = options.LoaderVersion
	}

	installerVersions, err := l.getInstallerVersions(options.IncludeUnstable)
	if err != nil || len(installerVersions) == 0 {
		return "", fmt.Errorf("error getting Quilt installer versions: %w", err)
	}
	installerVersion := installerVersions[0]
	if options.InstallerVersion != "" {
		installerVersion = options.InstallerVersion
	}

	downloadURL := fmt.Sprintf("https://maven.quiltmc.org/repository/release/org/quiltmc/quilt-installer/%s/quilt-installer-%s.jar", installerVersion, installerVersion)
	installerPath := filepath.Join(destDir, "quilt-installer.jar")
	if err := l.downloadFile(downloadURL, installerPath, progressChan); err != nil {
		return "", err
	}

	cmd := exec.Command("java", "-jar", "quilt-installer.jar", "install", "server", mc, loaderVersion, "--download-server")
	cmd.Dir = destDir
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error running Quilt installer: %w", err)
	}
	_ = os.Remove(installerPath)

	if _, err := os.Stat(filepath.Join(destDir, "quilt-server-launch.jar")); err == nil {
		_ = os.Rename(filepath.Join(destDir, "quilt-server-launch.jar"), filepath.Join(destDir, "server.jar"))
	}
	return mc, nil
}

func (l *QuiltLoader) getGameVersions(includeUnstable bool) ([]string, error) {
	return l.fetchVersionList("game", includeUnstable)
}

func (l *QuiltLoader) getLoaderVersions(includeUnstable bool) ([]string, error) {
	return l.fetchVersionList("loader", includeUnstable)
}

func (l *QuiltLoader) getInstallerVersions(includeUnstable bool) ([]string, error) {
	// Installer availability in Quilt Meta can mark many entries as non-stable.
	// To avoid empty installer dropdowns for release MC versions, always return
	// the full installer list and let users pick.
	return l.fetchVersionList("installer", true)
}

func (l *QuiltLoader) fetchVersionList(kind string, includeUnstable bool) ([]string, error) {
	resp, err := http.Get(QuiltMetaURL + kind)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Quilt endpoints may return either:
	// - []{version, stable}
	// - {versions: []{version, stable}}
	// and some entries may not carry `stable`.
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var direct []map[string]any
	if err := json.Unmarshal(rawBody, &direct); err == nil {
		return normalizeQuiltVersions(direct, includeUnstable), nil
	}

	var wrapped struct {
		Versions []map[string]any `json:"versions"`
	}
	if err := json.Unmarshal(rawBody, &wrapped); err == nil {
		return normalizeQuiltVersions(wrapped.Versions, includeUnstable), nil
	}

	return nil, fmt.Errorf("unexpected Quilt %s response format", kind)
}

func normalizeQuiltVersions(entries []map[string]any, includeUnstable bool) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		rawVersion, ok := e["version"].(string)
		if !ok || rawVersion == "" {
			continue
		}

		stable := true
		if rawStable, hasStable := e["stable"]; hasStable {
			if stableBool, ok := rawStable.(bool); ok {
				stable = stableBool
			}
		}

		if includeUnstable || stable {
			out = append(out, rawVersion)
		}
	}
	return out
}

func (l *QuiltLoader) downloadFile(url string, dest string, progressChan chan<- domain.ProgressEvent) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	progressReader := &ProgressReader{
		Reader:       resp.Body,
		Total:        resp.ContentLength,
		ProgressChan: progressChan,
		Message:      "Downloading Quilt installer.jar",
	}
	_, err = io.Copy(out, progressReader)
	return err
}
