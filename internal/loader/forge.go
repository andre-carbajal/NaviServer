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

const ForgeAPIURL = "https://bmclapi2.bangbang93.com/forge/"

type ForgeLoader struct{}

func NewForgeLoader() *ForgeLoader {
	return &ForgeLoader{}
}

func (l *ForgeLoader) GetSupportedVersions() ([]string, error) {
	resp, err := http.Get(ForgeAPIURL + "minecraft")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API responded with status %d", resp.StatusCode)
	}

	var versions []string
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, err
	}

	SortVersions(versions)
	return versions, nil
}

func (l *ForgeLoader) getLoaderVersions(minecraftVersion string) ([]string, error) {
	resp, err := http.Get(ForgeAPIURL + "minecraft/" + minecraftVersion)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API responded with status %d", resp.StatusCode)
	}

	type forgeLoaderVersion struct {
		Version string `json:"version"`
	}

	var loaderInfo []forgeLoaderVersion
	if err := json.NewDecoder(resp.Body).Decode(&loaderInfo); err != nil {
		return nil, err
	}

	var versions []string
	for _, v := range loaderInfo {
		versions = append(versions, v.Version)
	}

	SortVersions(versions)
	return versions, nil
}

func (l *ForgeLoader) Load(versionID string, destDir string, progressChan chan<- domain.ProgressEvent) error {
	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: fmt.Sprintf("Searching for version %s...", versionID)}
	}

	supportedVersions, err := l.GetSupportedVersions()
	if err != nil {
		return fmt.Errorf("error getting Forge versions: %w", err)
	}

	versionExists := false
	for _, v := range supportedVersions {
		if v == versionID {
			versionExists = true
			break
		}
	}

	if !versionExists {
		return fmt.Errorf("version %s not found in Forge", versionID)
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Getting loader versions..."}
	}
	loaderVersions, err := l.getLoaderVersions(versionID)
	if err != nil {
		return fmt.Errorf("error getting Forge loader versions: %w", err)
	}
	if len(loaderVersions) == 0 {
		return fmt.Errorf("no loader versions found for Forge on minecraft version %s", versionID)
	}
	latestLoaderVersion := loaderVersions[0]

	forgeVersion := fmt.Sprintf("%s-%s", versionID, latestLoaderVersion)
	downloadURL := fmt.Sprintf("https://maven.minecraftforge.net/net/minecraftforge/forge/%s/forge-%s-installer.jar", forgeVersion, forgeVersion)

	installerPath := filepath.Join(destDir, "installer.jar")
	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: fmt.Sprintf("Downloading Forge installer.jar from: %s", downloadURL)}
	}

	err = l.downloadFile(downloadURL, installerPath, progressChan)
	if err != nil {
		return err
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Running Forge installer..."}
	}
	cmd := exec.Command("java", "-jar", "installer.jar", "--installServer")
	cmd.Dir = destDir
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running Forge installer: %w", err)
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Cleaning up installation files..."}
	}
	if err := os.Remove(installerPath); err != nil {
		return fmt.Errorf("error removing installer: %w", err)
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Forge installation completed.", Progress: 100}
	}
	return nil
}

func (l *ForgeLoader) downloadFile(url string, dest string, progressChan chan<- domain.ProgressEvent) error {
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
		Message:      "Downloading Forge installer.jar",
	}

	_, err = io.Copy(out, progressReader)
	return err
}
