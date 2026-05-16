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

func (l *PaperLoader) GetSupportedVersions(options LoaderOptions) ([]string, error) {
	return l.getVersions()
}

func (l *PaperLoader) GetMetadata(options LoaderOptions) (*LoaderMetadata, error) {
	versions, err := l.getVersions()
	if err != nil {
		return nil, err
	}
	latest := ""
	if len(versions) > 0 {
		latest = versions[0]
	}
	md := &LoaderMetadata{
		LatestVersion:     latest,
		MinecraftVersions: versions,
	}
	targetVersion := options.MCVersion
	if targetVersion == "" && latest != "" {
		targetVersion = latest
	}
	if targetVersion != "" {
		builds, err := l.getBuilds(targetVersion)
		if err == nil {
			md.BuildVersions = builds
		}
	}
	return md, nil
}

func (l *PaperLoader) Load(options LoaderOptions, destDir string, progressChan chan<- domain.ProgressEvent) (string, error) {
	versionID := options.MCVersion
	if versionID == "" {
		versions, err := l.getVersions()
		if err != nil {
			return "", fmt.Errorf("error getting Paper versions: %w", err)
		}
		if len(versions) == 0 {
			return "", fmt.Errorf("no Paper versions found")
		}
		versionID = versions[0]
	}
	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: fmt.Sprintf("Searching for version %s...", versionID)}
	}

	versions, err := l.getVersions()
	if err != nil {
		return "", fmt.Errorf("error getting Paper versions: %w", err)
	}

	versionExists := false
	for _, v := range versions {
		if v == versionID {
			versionExists = true
			break
		}
	}

	if !versionExists {
		return "", fmt.Errorf("version %s not found in Paper", versionID)
	}

	selectedBuild := options.BuildVersion
	if selectedBuild == "" {
		if progressChan != nil {
			progressChan <- domain.ProgressEvent{Message: "Getting latest build..."}
		}
		latestBuild, err := l.getLatestBuild(versionID)
		if err != nil {
			return "", fmt.Errorf("error getting latest build: %w", err)
		}
		selectedBuild = fmt.Sprintf("%d", latestBuild)
	}
	buildInt := 0
	_, _ = fmt.Sscanf(selectedBuild, "%d", &buildInt)
	if buildInt <= 0 {
		return "", fmt.Errorf("invalid Paper build version: %s", selectedBuild)
	}

	downloadURL := fmt.Sprintf("%sversions/%s/builds/%d/downloads/paper-%s-%d.jar",
		PaperAPIURL, versionID, buildInt, versionID, buildInt)

	finalPath := filepath.Join(destDir, "server.jar")
	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: fmt.Sprintf("Downloading Paper server.jar from: %s", downloadURL)}
	}

	err = l.downloadFile(downloadURL, finalPath, progressChan)
	if err != nil {
		return "", err
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Installation completed.", Progress: 100}
	}
	return versionID, nil
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
	builds, err := l.getBuilds(version)
	if err != nil {
		return 0, err
	}
	if len(builds) == 0 {
		return 0, fmt.Errorf("no builds found for version %s", version)
	}
	latest := 0
	_, _ = fmt.Sscanf(builds[0], "%d", &latest)
	return latest, nil
}

func (l *PaperLoader) getBuilds(version string) ([]string, error) {
	url := fmt.Sprintf("%sversions/%s", PaperAPIURL, version)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API responded with status %d", resp.StatusCode)
	}

	var response PaperBuildsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if len(response.Builds) == 0 {
		return nil, fmt.Errorf("no builds found for version %s", version)
	}
	builds := make([]string, 0, len(response.Builds))
	for i := len(response.Builds) - 1; i >= 0; i-- {
		builds = append(builds, fmt.Sprintf("%d", response.Builds[i]))
	}
	return builds, nil
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
