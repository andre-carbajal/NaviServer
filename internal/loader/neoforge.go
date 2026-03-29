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
	"strings"
)

const NeoForgeAPIURL = "https://maven.neoforged.net/api/maven/versions/releases/net%2Fneoforged%2Fneoforge"

type NeoForgeLoader struct{}

func NewNeoForgeLoader() *NeoForgeLoader {
	return &NeoForgeLoader{}
}

type NeoForgeVersionsResponse struct {
	Versions []string `json:"versions"`
}

func (l *NeoForgeLoader) GetSupportedVersions() ([]string, error) {
	resp, err := http.Get(NeoForgeAPIURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API responded with status %d", resp.StatusCode)
	}

	var response NeoForgeVersionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	var versionsList []string
	seen := make(map[string]bool)

	for _, version := range response.Versions {
		if strings.HasPrefix(version, "0.") || strings.Contains(version, "snapshot") || strings.Contains(version, "alpha") {
			continue
		}

		parts := strings.Split(version, ".")
		if len(parts) >= 2 {
			majorNum := parts[0]
			var formatted string

			if majorNum == "20" || majorNum == "21" {
				formatted = fmt.Sprintf("1.%s.%s", majorNum, parts[1])
			} else {
				if len(parts) >= 3 {
					formatted = fmt.Sprintf("%s.%s.%s", majorNum, parts[1], parts[2])
				} else {
					formatted = fmt.Sprintf("%s.%s", majorNum, parts[1])
				}
			}

			if !seen[formatted] {
				versionsList = append(versionsList, formatted)
				seen[formatted] = true
			}
		}
	}

	SortVersions(versionsList)
	return versionsList, nil
}

func (l *NeoForgeLoader) getLoaderVersions(minecraftVersion string) ([]string, error) {
	resp, err := http.Get(NeoForgeAPIURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API responded with status %d", resp.StatusCode)
	}

	var response NeoForgeVersionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	var loaderVersionsList []string
	parts := strings.Split(minecraftVersion, ".")

	if len(parts) >= 3 {
		if parts[0] == "1" && (parts[1] == "20" || parts[1] == "21") {
			versionPrefix := parts[1] + "." + parts[2] + "."

			for _, version := range response.Versions {
				if strings.HasPrefix(version, versionPrefix) {
					loaderVersionsList = append(loaderVersionsList, version)
				}
			}
		} else {
			versionPrefix := parts[0] + "." + parts[1] + "." + parts[2] + "."

			for _, version := range response.Versions {
				if strings.HasPrefix(version, versionPrefix) {
					loaderVersionsList = append(loaderVersionsList, version)
				}
			}
		}
	} else if len(parts) == 2 {
		if parts[0] == "1" && (parts[1] == "20" || parts[1] == "21") {
			versionPrefix := parts[1] + ".0."

			for _, version := range response.Versions {
				if strings.HasPrefix(version, versionPrefix) {
					loaderVersionsList = append(loaderVersionsList, version)
				}
			}
		} else {
			versionPrefix := parts[0] + "." + parts[1] + "."

			for _, version := range response.Versions {
				if strings.HasPrefix(version, versionPrefix) {
					loaderVersionsList = append(loaderVersionsList, version)
				}
			}
		}
	}

	SortVersions(loaderVersionsList)
	return loaderVersionsList, nil
}

func (l *NeoForgeLoader) Load(versionID string, destDir string, progressChan chan<- domain.ProgressEvent) error {
	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: fmt.Sprintf("Searching for version %s...", versionID)}
	}

	supportedVersions, err := l.GetSupportedVersions()
	if err != nil {
		return fmt.Errorf("error getting NeoForge versions: %w", err)
	}

	versionExists := false
	for _, v := range supportedVersions {
		if v == versionID {
			versionExists = true
			break
		}
	}

	if !versionExists {
		return fmt.Errorf("version %s not found in NeoForge", versionID)
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Getting loader versions..."}
	}
	loaderVersions, err := l.getLoaderVersions(versionID)
	if err != nil {
		return fmt.Errorf("error getting NeoForge loader versions: %w", err)
	}
	if len(loaderVersions) == 0 {
		return fmt.Errorf("no loader versions found for NeoForge on minecraft version %s", versionID)
	}

	latestLoaderVersion := loaderVersions[0]

	downloadURL := fmt.Sprintf("https://maven.neoforged.net/releases/net/neoforged/neoforge/%s/neoforge-%s-installer.jar", latestLoaderVersion, latestLoaderVersion)

	installerPath := filepath.Join(destDir, "installer.jar")
	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: fmt.Sprintf("Downloading NeoForge installer.jar from: %s", downloadURL)}
	}

	err = l.downloadFile(downloadURL, installerPath, progressChan)
	if err != nil {
		return err
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Running NeoForge installer..."}
	}
	cmd := exec.Command("java", "-jar", "installer.jar", "--installServer")
	cmd.Dir = destDir
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running NeoForge installer: %w", err)
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Cleaning up installation files..."}
	}
	if err := os.Remove(installerPath); err != nil {
		return fmt.Errorf("error removing installer: %w", err)
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "NeoForge installation completed.", Progress: 100}
	}
	return nil
}

func (l *NeoForgeLoader) downloadFile(url string, dest string, progressChan chan<- domain.ProgressEvent) error {
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
		Message:      "Downloading NeoForge installer.jar",
	}

	_, err = io.Copy(out, progressReader)
	return err
}
