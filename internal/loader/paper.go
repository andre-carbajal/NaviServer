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

const PaperAPIURL = "https://fill.papermc.io/v3/projects/paper"

type PaperVersionsResponse struct {
	Versions map[string][]string `json:"versions"`
}

type paperBuildDownload struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type paperBuild struct {
	ID        int                           `json:"id"`
	Downloads map[string]paperBuildDownload `json:"downloads"`
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

	builds, err := l.getBuildDetails(versionID)
	if err != nil {
		return "", fmt.Errorf("error getting Paper builds: %w", err)
	}
	if len(builds) == 0 {
		return "", fmt.Errorf("no builds found for version %s", versionID)
	}

	selectedBuild := options.BuildVersion
	if selectedBuild == "" {
		if progressChan != nil {
			progressChan <- domain.ProgressEvent{Message: "Getting latest build..."}
		}
		selectedBuild = fmt.Sprintf("%d", builds[0].ID)
	}

	var build *paperBuild
	for i := range builds {
		if fmt.Sprintf("%d", builds[i].ID) == selectedBuild {
			build = &builds[i]
			break
		}
	}
	if build == nil {
		return "", fmt.Errorf("invalid Paper build version: %s", selectedBuild)
	}

	download, ok := build.Downloads["server:default"]
	if !ok || strings.TrimSpace(download.URL) == "" {
		return "", fmt.Errorf("paper build %s does not provide a server download", selectedBuild)
	}

	finalPath := filepath.Join(destDir, "server.jar")
	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: fmt.Sprintf("Downloading Paper server.jar from: %s", download.URL)}
	}

	err = l.downloadFile(download.URL, finalPath, progressChan)
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

	return stablePaperVersions(response.Versions), nil
}

func stablePaperVersions(groups map[string][]string) []string {
	seen := make(map[string]bool)
	versions := make([]string, 0)
	for _, groupedVersions := range groups {
		for _, version := range groupedVersions {
			if strings.Contains(version, "-") || seen[version] {
				continue
			}
			versions = append(versions, version)
			seen[version] = true
		}
	}
	SortVersions(versions)
	return versions
}

func (l *PaperLoader) getBuilds(version string) ([]string, error) {
	builds, err := l.getBuildDetails(version)
	if err != nil {
		return nil, err
	}
	if len(builds) == 0 {
		return nil, fmt.Errorf("no builds found for version %s", version)
	}
	buildVersions := make([]string, 0, len(builds))
	for _, build := range builds {
		buildVersions = append(buildVersions, fmt.Sprintf("%d", build.ID))
	}
	return buildVersions, nil
}

func (l *PaperLoader) getBuildDetails(version string) ([]paperBuild, error) {
	url := fmt.Sprintf("%s/versions/%s/builds", PaperAPIURL, version)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API responded with status %d", resp.StatusCode)
	}

	builds := make([]paperBuild, 0)
	if err := json.NewDecoder(resp.Body).Decode(&builds); err != nil {
		return nil, err
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
