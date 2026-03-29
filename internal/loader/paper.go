package loader

import (
	"encoding/json"
	"fmt"
	"io"
	"naviserver/internal/domain"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const PaperAPIURL = "https://api.papermc.io/v2/projects/paper/"

type PaperVersionsResponse struct {
	Versions []string `json:"versions"`
}

type PaperBuildsResponse struct {
	Builds []int `json:"builds"`
}

type PaperLoader struct{}

func NewPaperLoader() *PaperLoader {
	return &PaperLoader{}
}

func (l *PaperLoader) GetSupportedVersions() ([]string, error) {
	return l.getVersions()
}

func (l *PaperLoader) Load(versionID string, destDir string, progressChan chan<- domain.ProgressEvent) error {
	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: fmt.Sprintf("Searching for version %s...", versionID)}
	}

	versions, err := l.getVersions()
	if err != nil {
		return fmt.Errorf("error getting Paper versions: %w", err)
	}

	versionExists := false
	for _, v := range versions {
		if v == versionID {
			versionExists = true
			break
		}
	}

	if !versionExists {
		return fmt.Errorf("version %s not found in Paper", versionID)
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Getting latest build..."}
	}
	latestBuild, err := l.getLatestBuild(versionID)
	if err != nil {
		return fmt.Errorf("error getting latest build: %w", err)
	}

	downloadURL := fmt.Sprintf("%sversions/%s/builds/%d/downloads/paper-%s-%d.jar",
		PaperAPIURL, versionID, latestBuild, versionID, latestBuild)

	finalPath := filepath.Join(destDir, "server.jar")
	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: fmt.Sprintf("Downloading Paper server.jar from: %s", downloadURL)}
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

func (l *PaperLoader) getVersions() ([]string, error) {
	resp, err := http.Get(PaperAPIURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API responded with status %d", resp.StatusCode)
	}

	var response PaperVersionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	var filteredVersions []string
	for _, v := range response.Versions {
		if !strings.Contains(v, "-") {
			filteredVersions = append(filteredVersions, v)
		}
	}

	SortVersions(filteredVersions)
	return filteredVersions, nil
}

func (l *PaperLoader) getLatestBuild(version string) (int, error) {
	url := fmt.Sprintf("%sversions/%s", PaperAPIURL, version)
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API responded with status %d", resp.StatusCode)
	}

	var response PaperBuildsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, err
	}

	if len(response.Builds) == 0 {
		return 0, fmt.Errorf("no builds found for version %s", version)
	}

	return response.Builds[len(response.Builds)-1], nil
}

func (l *PaperLoader) downloadFile(url string, dest string, progressChan chan<- domain.ProgressEvent) error {
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
		Message:      "Downloading Paper server.jar",
	}

	_, err = io.Copy(out, progressReader)
	return err
}
