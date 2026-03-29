package loader

import (
	"encoding/json"
	"fmt"
	"io"
	"naviserver/internal/domain"
	"net/http"
	"os"
	"path/filepath"
)

const FabricAPIURL = "https://meta.fabricmc.net/v2/versions/"

type FabricGameVersion struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

type FabricLoaderVersion struct {
	Version string `json:"version"`
}

type FabricInstallerVersion struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

type FabricLoader struct{}

func NewFabricLoader() *FabricLoader {
	return &FabricLoader{}
}

func (l *FabricLoader) GetSupportedVersions() ([]string, error) {
	return l.getGameVersions()
}

func (l *FabricLoader) Load(versionID string, destDir string, progressChan chan<- domain.ProgressEvent) error {
	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: fmt.Sprintf("Searching for version %s...", versionID)}
	}

	gameVersions, err := l.getGameVersions()
	if err != nil {
		return fmt.Errorf("error getting Fabric versions: %w", err)
	}

	versionExists := false
	for _, v := range gameVersions {
		if v == versionID {
			versionExists = true
			break
		}
	}

	if !versionExists {
		return fmt.Errorf("version %s not found in Fabric", versionID)
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Getting loader versions..."}
	}
	loaderVersions, err := l.getLoaderVersions()
	if err != nil {
		return fmt.Errorf("error getting Fabric loader versions: %w", err)
	}
	if len(loaderVersions) == 0 {
		return fmt.Errorf("no loader versions found for Fabric")
	}
	latestLoaderVersion := loaderVersions[0]

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Getting latest installer version..."}
	}
	installerVersion, err := l.getLatestInstallerVersion()
	if err != nil {
		return fmt.Errorf("error getting latest installer version: %w", err)
	}

	downloadURL := fmt.Sprintf("%sloader/%s/%s/%s/server/jar",
		FabricAPIURL, versionID, latestLoaderVersion, installerVersion)

	finalPath := filepath.Join(destDir, "server.jar")
	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: fmt.Sprintf("Downloading Fabric server.jar from: %s", downloadURL)}
	}

	err = l.downloadFile(downloadURL, finalPath, progressChan)
	if err != nil {
		return err
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Installation completed.", Progress: 100}
	}
	return nil
}

func (l *FabricLoader) getGameVersions() ([]string, error) {
	resp, err := http.Get(FabricAPIURL + "game")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API responded with status %d", resp.StatusCode)
	}

	var versions []FabricGameVersion
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, err
	}

	var stableVersions []string
	for _, v := range versions {
		if v.Stable {
			stableVersions = append(stableVersions, v.Version)
		}
	}

	return stableVersions, nil
}

func (l *FabricLoader) getLoaderVersions() ([]string, error) {
	resp, err := http.Get(FabricAPIURL + "loader")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API responded with status %d", resp.StatusCode)
	}

	var versions []FabricLoaderVersion
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, err
	}

	var loaderVersions []string
	for _, v := range versions {
		loaderVersions = append(loaderVersions, v.Version)
	}

	return loaderVersions, nil
}

func (l *FabricLoader) getLatestInstallerVersion() (string, error) {
	resp, err := http.Get(FabricAPIURL + "installer")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API responded with status %d", resp.StatusCode)
	}

	var versions []FabricInstallerVersion
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return "", err
	}

	for _, v := range versions {
		if v.Stable {
			return v.Version, nil
		}
	}

	return "", fmt.Errorf("no stable installer version found")
}

func (l *FabricLoader) downloadFile(url string, dest string, progressChan chan<- domain.ProgressEvent) error {
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error downloading file: status %d", resp.StatusCode)
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Starting download..."}
	}

	progressReader := &ProgressReader{
		Reader:       resp.Body,
		Total:        resp.ContentLength,
		ProgressChan: progressChan,
		Message:      "Downloading Fabric server.jar",
	}

	_, err = io.Copy(out, progressReader)
	return err
}
