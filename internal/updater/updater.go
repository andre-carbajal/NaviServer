package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

var (
	CurrentVersion = "dev"
	RepoOwner      = "andre-carbajal"
	RepoName       = "naviserver"
)

type Tag struct {
	Name string `json:"name"`
}

type UpdateInfo struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseURL      string `json:"release_url"`
}

func CheckForUpdates() (*UpdateInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/tags", RepoOwner, RepoName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "naviserver-updater")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch tags: %s", resp.Status)
	}

	var tags []Tag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}

	if len(tags) == 0 {
		return &UpdateInfo{
			CurrentVersion:  CurrentVersion,
			LatestVersion:   CurrentVersion,
			UpdateAvailable: false,
		}, nil
	}

	latestTag := tags[0].Name
	updateAvailable := compareVersions(latestTag, CurrentVersion) > 0
	releaseURL := fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", RepoOwner, RepoName, latestTag)

	return &UpdateInfo{
		CurrentVersion:  CurrentVersion,
		LatestVersion:   latestTag,
		UpdateAvailable: updateAvailable,
		ReleaseURL:      releaseURL,
	}, nil
}

func compareVersions(v1, v2 string) int {
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		n1, _ := strconv.Atoi(parts1[i])
		n2, _ := strconv.Atoi(parts2[i])
		if n1 > n2 {
			return 1
		}
		if n1 < n2 {
			return -1
		}
	}

	if len(parts1) > len(parts2) {
		return 1
	}
	if len(parts1) < len(parts2) {
		return -1
	}

	return 0
}
