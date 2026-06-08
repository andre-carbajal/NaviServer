package addons

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type modrinthDependency struct {
	VersionID      string `json:"version_id"`
	ProjectID      string `json:"project_id"`
	FileName       string `json:"file_name"`
	DependencyType string `json:"dependency_type"`
}

type modrinthFile struct {
	Hashes  map[string]string `json:"hashes"`
	URL     string            `json:"url"`
	Name    string            `json:"filename"`
	Primary bool              `json:"primary"`
}

type modrinthVersion struct {
	VersionID     string               `json:"id"`
	ProjectID     string               `json:"project_id"`
	VersionName   string               `json:"name"`
	VersionNumber string               `json:"version_number"`
	VersionType   string               `json:"version_type"`
	DatePublished string               `json:"date_published"`
	Files         []modrinthFile       `json:"files"`
	Dependencies  []modrinthDependency `json:"dependencies"`
	ProjectTitle  string               `json:"project_title"`
	ProjectSlug   string               `json:"project_slug"`
}

func (v modrinthVersion) PrimaryFileURL() string {
	for _, file := range v.Files {
		if file.Primary && file.URL != "" {
			return file.URL
		}
	}
	if len(v.Files) > 0 {
		return v.Files[0].URL
	}
	return ""
}

func (v modrinthVersion) PrimaryFilename() string {
	for _, file := range v.Files {
		if file.Primary && file.Name != "" {
			return file.Name
		}
	}
	if len(v.Files) > 0 {
		return v.Files[0].Name
	}
	return ""
}

type modrinthSearchResponse struct {
	Hits []struct {
		ProjectID   string   `json:"project_id"`
		Slug        string   `json:"slug"`
		Title       string   `json:"title"`
		Author      string   `json:"author"`
		Description string   `json:"description"`
		IconURL     string   `json:"icon_url"`
		Downloads   int64    `json:"downloads"`
		LatestID    string   `json:"latest_version"`
		ProjectType string   `json:"project_type"`
		Categories  []string `json:"categories"`
	} `json:"hits"`
	Offset    int `json:"offset"`
	Limit     int `json:"limit"`
	TotalHits int `json:"total_hits"`
}

func (m *Manager) modrinthVersionsFromHashes(ctx context.Context, items []scannedAddonFile) (map[string]modrinthVersion, error) {
	if len(items) == 0 {
		return map[string]modrinthVersion{}, nil
	}
	hashes := make([]string, 0, len(items))
	for _, item := range items {
		hashes = append(hashes, item.sha1)
	}
	body := map[string]any{
		"hashes":    hashes,
		"algorithm": "sha1",
	}
	response := map[string]modrinthVersion{}
	if err := m.doJSON(ctx, http.MethodPost, modrinthBaseURL+"/version_files", body, nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (m *Manager) modrinthLatestFromHashes(ctx context.Context, items []scannedAddonFile, loaders []string, versions []string) (map[string]modrinthVersion, error) {
	if len(items) == 0 {
		return map[string]modrinthVersion{}, nil
	}
	hashes := make([]string, 0, len(items))
	for _, item := range items {
		hashes = append(hashes, item.sha1)
	}
	body := map[string]any{
		"hashes":        hashes,
		"algorithm":     "sha1",
		"loaders":       loaders,
		"game_versions": versions,
	}
	response := map[string]modrinthVersion{}
	if err := m.doJSON(ctx, http.MethodPost, modrinthBaseURL+"/version_files/update", body, nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (m *Manager) resolveModrinthVersion(ctx context.Context, projectID, versionID string, loaders []string, mcVersion string) (*modrinthVersion, error) {
	if strings.TrimSpace(versionID) != "" {
		var version modrinthVersion
		if err := m.doJSON(ctx, http.MethodGet, modrinthBaseURL+"/version/"+versionID, nil, nil, &version); err != nil {
			return nil, err
		}
		return &version, nil
	}

	versions, err := m.listModrinthVersions(ctx, projectID, loaders, mcVersion)
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, nil
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].DatePublished > versions[j].DatePublished
	})
	return &versions[0], nil
}

func (m *Manager) listModrinthVersions(ctx context.Context, projectID string, loaders []string, mcVersion string) ([]modrinthVersion, error) {
	query := fmt.Sprintf(
		"%s/project/%s/version?loaders=%s&game_versions=%s",
		modrinthBaseURL,
		projectID,
		encodeQuery("["+quoteCSV(loaders)+"]"),
		encodeQuery(fmt.Sprintf("[\"%s\"]", mcVersion)),
	)
	versions := make([]modrinthVersion, 0)
	if err := m.doJSON(ctx, http.MethodGet, query, nil, nil, &versions); err != nil {
		return nil, err
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].DatePublished > versions[j].DatePublished
	})
	return versions, nil
}

func quoteCSV(items []string) string {
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		quoted = append(quoted, fmt.Sprintf("\"%s\"", item))
	}
	return strings.Join(quoted, ",")
}

func (m *Manager) searchModrinth(
	ctx context.Context,
	query string,
	loaders []string,
	mcVersion string,
	offset,
	limit int,
) ([]SearchResult, bool, error) {
	trimmedQuery := strings.TrimSpace(query)
	endpoint := modrinthSearchEndpoint(query, loaders, mcVersion, offset, limit)
	var response modrinthSearchResponse
	if err := m.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &response); err != nil {
		return nil, false, err
	}
	results := make([]SearchResult, 0, len(response.Hits))
	for _, hit := range response.Hits {
		if hit.ProjectType != "mod" && hit.ProjectType != "plugin" {
			continue
		}
		searchResult := SearchResult{
			Source:      AddonSourceModrinth,
			ProjectID:   hit.ProjectID,
			ProjectSlug: hit.Slug,
			ProjectName: hit.Title,
			AuthorName:  hit.Author,
			Description: hit.Description,
			ProjectURL:  "https://modrinth.com/project/" + hit.Slug,
			IconURL:     hit.IconURL,
			Downloads:   hit.Downloads,
		}
		if trimmedQuery == "" && hit.LatestID != "" {
			searchResult.Latest = &AddonVersion{
				VersionID:    hit.LatestID,
				VersionName:  "Latest compatible",
				VersionLabel: "latest",
				ReleaseType:  ReleaseTypeRelease,
				Source:       AddonSourceModrinth,
			}
			results = append(results, searchResult)
			continue
		}

		versions, _ := m.listModrinthVersions(ctx, hit.ProjectID, loaders, mcVersion)
		for i, mapped := range mapModrinthVersions(versions) {
			if i == 0 {
				searchResult.Latest = &mapped
			}
			searchResult.Versions = append(searchResult.Versions, mapped)
			if i >= 14 {
				break
			}
		}
		results = append(results, searchResult)
	}
	hasMore := false
	if response.TotalHits > 0 {
		hasMore = response.Offset+len(response.Hits) < response.TotalHits
	} else {
		hasMore = len(response.Hits) >= limit
	}
	return results, hasMore, nil
}

func modrinthSearchEndpoint(
	query string,
	loaders []string,
	mcVersion string,
	offset,
	limit int,
) string {
	values := url.Values{}
	values.Set("limit", strconv.Itoa(limit))
	values.Set("offset", strconv.Itoa(offset))
	values.Set("facets", modrinthFacets(loaders, mcVersion))

	if strings.TrimSpace(query) == "" {
		values.Set("query", "")
		values.Set("index", "downloads")
	} else {
		values.Set("query", query)
	}

	return fmt.Sprintf("%s/search?%s", modrinthBaseURL, values.Encode())
}

func mapModrinthVersions(versions []modrinthVersion) []AddonVersion {
	mapped := make([]AddonVersion, 0, len(versions))
	for _, version := range versions {
		mapped = append(mapped, AddonVersion{
			VersionID:    version.VersionID,
			VersionName:  version.VersionName,
			VersionLabel: version.VersionNumber,
			ReleaseType:  ReleaseType(version.VersionType),
			PublishedAt:  version.DatePublished,
			DownloadURL:  version.PrimaryFileURL(),
			Filename:     version.PrimaryFilename(),
			Source:       AddonSourceModrinth,
		})
	}
	return mapped
}

func (m *Manager) getModrinthProjectIcon(ctx context.Context, projectID string) (string, error) {
	var response struct {
		IconURL string `json:"icon_url"`
	}
	if err := m.doJSON(
		ctx,
		http.MethodGet,
		modrinthBaseURL+"/project/"+projectID,
		nil,
		nil,
		&response,
	); err != nil {
		return "", err
	}
	return response.IconURL, nil
}
