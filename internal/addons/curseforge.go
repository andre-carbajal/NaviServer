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

type curseLinks struct {
	WebsiteURL string `json:"websiteUrl"`
}

type curseMod struct {
	ID    int64      `json:"id"`
	Name  string     `json:"name"`
	Slug  string     `json:"slug"`
	Links curseLinks `json:"links"`
	Logo  struct {
		URL string `json:"url"`
	} `json:"logo"`
	LogoURL string `json:"logoUrl"`
}

type curseFileHash struct {
	Value string `json:"value"`
	Algo  int    `json:"algo"`
}

type curseFileDependency struct {
	ModID        int64 `json:"modId"`
	FileID       int64 `json:"fileId"`
	RelationType int   `json:"relationType"`
}

type curseFileIndex struct {
	GameVersion string `json:"gameVersion"`
	FileID      int64  `json:"fileId"`
	ModLoader   int    `json:"modLoader"`
}

type curseFile struct {
	ID           int64                 `json:"id"`
	GameID       int64                 `json:"gameId"`
	ModID        int64                 `json:"modId"`
	DisplayName  string                `json:"displayName"`
	FileName     string                `json:"fileName"`
	ReleaseType  int                   `json:"releaseType"`
	FileDate     string                `json:"fileDate"`
	DownloadURL  string                `json:"downloadUrl"`
	GameVersions []string              `json:"gameVersions"`
	Hashes       []curseFileHash       `json:"hashes"`
	Dependencies []curseFileDependency `json:"dependencies"`
	Fingerprint  uint32                `json:"fileFingerprint"`
}

type curseMatch struct {
	ID   int64     `json:"id"`
	File curseFile `json:"file"`
	Mod  curseMod  `json:"mod"`
}

type curseFingerprintResponse struct {
	Data struct {
		ExactMatches []curseMatch `json:"exactMatches"`
		LatestFiles  []curseFile  `json:"latestFiles"`
	} `json:"data"`
	LatestFiles []curseFile `json:"latestFiles"`
}

type curseModsResponse struct {
	Data []curseMod `json:"data"`
}

type curseSearchResponse struct {
	Data []struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		Slug          string `json:"slug"`
		Summary       string `json:"summary"`
		DownloadCount int64  `json:"downloadCount"`
		Categories    []struct {
			Name string `json:"name"`
			Slug string `json:"slug"`
		} `json:"categories"`
		Authors []struct {
			Name string `json:"name"`
		} `json:"authors"`
		Logo struct {
			URL string `json:"url"`
		} `json:"logo"`
		Links              curseLinks       `json:"links"`
		LatestFiles        []curseFile      `json:"latestFiles"`
		LatestFilesIndexes []curseFileIndex `json:"latestFilesIndexes"`
	} `json:"data"`
	Pagination struct {
		Index       int `json:"index"`
		PageSize    int `json:"pageSize"`
		ResultCount int `json:"resultCount"`
		TotalCount  int `json:"totalCount"`
	} `json:"pagination"`
}

type curseFilesResponse struct {
	Data       []curseFile `json:"data"`
	Pagination struct {
		Index       int `json:"index"`
		PageSize    int `json:"pageSize"`
		ResultCount int `json:"resultCount"`
		TotalCount  int `json:"totalCount"`
	} `json:"pagination"`
}

func mapCurseReleaseType(value int) ReleaseType {
	switch value {
	case 1:
		return ReleaseTypeRelease
	case 2:
		return ReleaseTypeBeta
	case 3:
		return ReleaseTypeAlpha
	default:
		return ReleaseTypeRelease
	}
}

func (m *Manager) curseForgeByFingerprints(ctx context.Context, items []scannedAddonFile) (map[uint32]curseMatch, error) {
	if strings.TrimSpace(m.resolveCurseForgeAPIKey()) == "" {
		return map[uint32]curseMatch{}, nil
	}
	fingerprints := make([]uint32, 0, len(items))
	for _, item := range items {
		fingerprints = append(fingerprints, item.fingerprint)
	}
	var response curseFingerprintResponse
	if err := m.doJSON(ctx, http.MethodPost, curseForgeBaseURL+"/fingerprints", map[string]any{
		"fingerprints": fingerprints,
	}, map[string]string{"x-api-key": m.resolveCurseForgeAPIKey()}, &response); err != nil {
		return nil, err
	}
	matches := make(map[uint32]curseMatch, len(response.Data.ExactMatches)+len(response.LatestFiles)+len(response.Data.LatestFiles))
	for _, match := range response.Data.ExactMatches {
		matches[match.File.Fingerprint] = match
	}
	for _, file := range append(response.Data.LatestFiles, response.LatestFiles...) {
		if file.Fingerprint == 0 || file.ModID == 0 {
			continue
		}
		if _, exists := matches[file.Fingerprint]; !exists {
			matches[file.Fingerprint] = curseMatch{
				File: file,
				Mod:  curseMod{ID: file.ModID},
			}
		}
	}
	return matches, nil
}

func (mod curseMod) iconURL() string {
	if iconURL := strings.TrimSpace(mod.Logo.URL); iconURL != "" {
		return iconURL
	}
	return strings.TrimSpace(mod.LogoURL)
}

func (m *Manager) enrichCurseForgeMatches(ctx context.Context, matches map[uint32]curseMatch) error {
	modIDs := make(map[int64]struct{})
	for fingerprint, match := range matches {
		modID := match.File.ModID
		if modID == 0 {
			modID = match.Mod.ID
		}
		if modID == 0 {
			continue
		}
		if strings.TrimSpace(match.Mod.Name) != "" && match.Mod.iconURL() != "" {
			continue
		}
		modIDs[modID] = struct{}{}
		if match.Mod.ID == 0 {
			match.Mod.ID = modID
			matches[fingerprint] = match
		}
	}
	if len(modIDs) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(modIDs))
	for modID := range modIDs {
		ids = append(ids, modID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	mods, err := m.fetchCurseForgeMods(ctx, ids)
	if err != nil {
		return err
	}

	for fingerprint, match := range matches {
		modID := match.File.ModID
		if modID == 0 {
			modID = match.Mod.ID
		}
		mod, ok := mods[strconv.FormatInt(modID, 10)]
		if !ok {
			continue
		}
		if strings.TrimSpace(mod.Name) == "" {
			mod.Name = match.Mod.Name
		}
		if strings.TrimSpace(mod.Slug) == "" {
			mod.Slug = match.Mod.Slug
		}
		if strings.TrimSpace(mod.Links.WebsiteURL) == "" {
			mod.Links = match.Mod.Links
		}
		if mod.iconURL() == "" {
			mod.Logo = match.Mod.Logo
			mod.LogoURL = match.Mod.LogoURL
		}
		matches[fingerprint] = curseMatch{ID: match.ID, File: match.File, Mod: mod}
	}
	return nil
}

func (m *Manager) fetchCurseForgeMods(ctx context.Context, ids []int64) (map[string]curseMod, error) {
	mods := make(map[string]curseMod)
	if len(ids) == 0 {
		return mods, nil
	}

	const batchSize = 50
	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		var response curseModsResponse
		if err := m.doJSON(ctx, http.MethodPost, curseForgeBaseURL+"/mods", map[string]any{
			"modIds": ids[start:end],
		}, map[string]string{"x-api-key": m.resolveCurseForgeAPIKey()}, &response); err != nil {
			return nil, err
		}
		for _, mod := range response.Data {
			if mod.ID != 0 {
				mods[strconv.FormatInt(mod.ID, 10)] = mod
			}
		}
	}
	return mods, nil
}

func (m *Manager) searchCurseForge(
	ctx context.Context,
	query,
	mcVersion,
	loader string,
	offset,
	limit int,
) ([]SearchResult, bool, error) {
	if strings.TrimSpace(m.resolveCurseForgeAPIKey()) == "" {
		return nil, false, errCurseForgeKeyMissing
	}
	endpoint := curseForgeSearchEndpoint(query, mcVersion, loader, offset, limit)
	var response curseSearchResponse
	if err := m.doJSON(ctx, http.MethodGet, endpoint, nil, map[string]string{"x-api-key": m.resolveCurseForgeAPIKey()}, &response); err != nil {
		return nil, false, err
	}

	sort.SliceStable(response.Data, func(i, j int) bool {
		leftScore := curseSearchRelevanceScore(query, response.Data[i].Name, response.Data[i].Slug)
		rightScore := curseSearchRelevanceScore(query, response.Data[j].Name, response.Data[j].Slug)
		if leftScore != rightScore {
			return leftScore > rightScore
		}
		return response.Data[i].DownloadCount > response.Data[j].DownloadCount
	})

	results := make([]SearchResult, 0, len(response.Data))
	for _, item := range response.Data {
		if isLikelyClientOnlyCurseProject(item.Name, item.Summary, item.Categories) {
			continue
		}
		compatible, err := m.compatibleCurseSearchFiles(
			ctx,
			item.ID,
			item.LatestFiles,
			item.LatestFilesIndexes,
			mcVersion,
			loader,
		)
		if err != nil {
			return nil, false, err
		}
		if len(compatible) == 0 {
			continue
		}
		latest := chooseLatestCurseFile(compatible, mcVersion, loader)
		result := SearchResult{
			Source:      AddonSourceCurseForge,
			ProjectID:   strconv.FormatInt(item.ID, 10),
			ProjectSlug: item.Slug,
			ProjectName: item.Name,
			Description: item.Summary,
			ProjectURL:  item.Links.WebsiteURL,
			IconURL:     item.Logo.URL,
			Downloads:   item.DownloadCount,
		}
		if len(item.Authors) > 0 {
			result.AuthorName = item.Authors[0].Name
		}
		if latest != nil {
			mapped := mapCurseFile(*latest)
			result.Latest = &mapped
		}
		for i, mapped := range mapCurseFiles(compatible) {
			result.Versions = append(result.Versions, mapped)
			if i >= 14 {
				break
			}
		}
		results = append(results, result)
	}

	hasMore := false
	if response.Pagination.TotalCount > 0 {
		hasMore = response.Pagination.Index+response.Pagination.ResultCount <
			response.Pagination.TotalCount
	} else {
		hasMore = len(response.Data) >= limit
	}
	return results, hasMore, nil
}

func (m *Manager) compatibleCurseSearchFiles(
	ctx context.Context,
	modID int64,
	latestFiles []curseFile,
	latestFilesIndexes []curseFileIndex,
	mcVersion,
	loader string,
) ([]curseFile, error) {
	candidateFiles := filterCurseSearchFiles(latestFiles, latestFilesIndexes, mcVersion, loader)
	compatible := listCompatibleCurseFiles(candidateFiles, mcVersion, loader)
	if len(compatible) > 0 {
		return compatible, nil
	}

	// The search response only contains a project's latest files. A compatible
	// server file can be omitted even when the project has one, so use the full
	// files endpoint as the authoritative fallback before hiding the project.
	files, err := m.listCurseFiles(ctx, modID, mcVersion, loader)
	if err != nil {
		return nil, fmt.Errorf("failed to load CurseForge files for mod %d: %w", modID, err)
	}
	return listCompatibleCurseFiles(files, mcVersion, loader), nil
}

func curseForgeSearchEndpoint(
	query,
	mcVersion,
	loader string,
	offset,
	limit int,
) string {
	values := url.Values{}
	values.Set("gameId", strconv.Itoa(minecraftGameID))
	values.Set("gameVersion", mcVersion)
	values.Set("pageSize", strconv.Itoa(limit))
	values.Set("index", strconv.Itoa(offset))
	switch strings.ToLower(strings.TrimSpace(loader)) {
	case "paper":
		// Minecraft Bukkit/Spigot plugins class.
		values.Set("classId", "5")
	case "fabric", "forge", "neoforge":
		// Minecraft Mods class; excludes shader packs/resource packs/datapacks.
		values.Set("classId", "6")
		if modLoaderType, ok := curseForgeModLoaderType(loader); ok {
			values.Set("modLoaderType", strconv.Itoa(modLoaderType))
		}
	}

	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		// Popular defaults when there is no search text.
		values.Set("sortField", "2")
		values.Set("sortOrder", "desc")
	} else {
		values.Set("searchFilter", trimmed)
	}

	return fmt.Sprintf("%s/mods/search?%s", curseForgeBaseURL, values.Encode())
}

func curseForgeModLoaderType(loader string) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(loader)) {
	case "forge":
		return 1, true
	case "fabric":
		return 4, true
	case "neoforge":
		return 6, true
	default:
		return 0, false
	}
}

func curseSearchRelevanceScore(query, name, slug string) int {
	q := strings.ToLower(strings.TrimSpace(query))
	n := strings.ToLower(strings.TrimSpace(name))
	s := strings.ToLower(strings.TrimSpace(slug))
	if q == "" {
		return 0
	}
	switch {
	case n == q || s == q:
		return 500
	case strings.HasPrefix(n, q) || strings.HasPrefix(s, q):
		return 400
	case strings.Contains(n, q) || strings.Contains(s, q):
		return 300
	}

	score := 0
	parts := strings.Fields(q)
	for _, part := range parts {
		if part == "" {
			continue
		}
		if strings.HasPrefix(n, part) || strings.HasPrefix(s, part) {
			score += 80
		} else if strings.Contains(n, part) || strings.Contains(s, part) {
			score += 30
		}
	}
	return score
}

func (m *Manager) findLatestCurseFile(ctx context.Context, modID int64, mcVersion, loader string) (*curseFile, []Dependency) {
	files, err := m.listCurseFiles(ctx, modID, mcVersion, loader)
	if err != nil {
		return nil, nil
	}
	latest := chooseLatestCurseFile(files, mcVersion, loader)
	if latest == nil {
		return nil, nil
	}
	deps := make([]Dependency, 0)
	for _, dep := range latest.Dependencies {
		required := dep.RelationType == 3
		deps = append(deps, Dependency{
			ProjectID: strconv.FormatInt(dep.ModID, 10),
			FileID:    dep.FileID,
			Required:  required,
			Source:    string(AddonSourceCurseForge),
		})
	}
	return latest, deps
}

func (m *Manager) resolveCurseForgeFile(ctx context.Context, projectID string, fileID int64, mcVersion, loader string) (*curseFile, []Dependency, error) {
	modID, err := strconv.ParseInt(projectID, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid curseforge project id")
	}
	files, err := m.listCurseFiles(ctx, modID, mcVersion, loader)
	if err != nil {
		return nil, nil, err
	}
	compatible := listCompatibleCurseFiles(files, mcVersion, loader)
	var selected *curseFile
	if fileID > 0 {
		for i := range compatible {
			if compatible[i].ID == fileID {
				selected = &compatible[i]
				break
			}
		}
	} else {
		if len(compatible) > 0 {
			selected = &compatible[0]
		}
	}
	if selected == nil {
		return nil, nil, nil
	}
	if selected.ModID == 0 {
		selected.ModID = modID
	}
	deps := make([]Dependency, 0)
	for _, dep := range selected.Dependencies {
		required := dep.RelationType == 3
		deps = append(deps, Dependency{
			ProjectID: strconv.FormatInt(dep.ModID, 10),
			FileID:    dep.FileID,
			Required:  required,
			Source:    string(AddonSourceCurseForge),
		})
	}
	return selected, deps, nil
}

func (m *Manager) getCurseFileDownloadURL(ctx context.Context, modID int64, fileID int64) (string, error) {
	var response struct {
		Data string `json:"data"`
	}
	endpoint := fmt.Sprintf("%s/mods/%d/files/%d/download-url", curseForgeBaseURL, modID, fileID)
	if err := m.doJSON(ctx, http.MethodGet, endpoint, nil, map[string]string{"x-api-key": m.resolveCurseForgeAPIKey()}, &response); err != nil {
		return "", err
	}
	return response.Data, nil
}

func (m *Manager) listCurseFiles(ctx context.Context, modID int64, mcVersion, loader string) ([]curseFile, error) {
	const pageSize = 50
	files := make([]curseFile, 0)
	for index := 0; ; index += pageSize {
		values := url.Values{}
		values.Set("gameVersion", mcVersion)
		values.Set("pageSize", strconv.Itoa(pageSize))
		values.Set("index", strconv.Itoa(index))
		if modLoaderType, ok := curseForgeModLoaderType(loader); ok {
			values.Set("modLoaderType", strconv.Itoa(modLoaderType))
		}
		endpoint := fmt.Sprintf("%s/mods/%d/files?%s", curseForgeBaseURL, modID, values.Encode())
		var response curseFilesResponse
		if err := m.doJSON(ctx, http.MethodGet, endpoint, nil, map[string]string{"x-api-key": m.resolveCurseForgeAPIKey()}, &response); err != nil {
			return nil, err
		}
		files = append(files, response.Data...)
		if response.Pagination.TotalCount > 0 {
			if response.Pagination.Index+response.Pagination.ResultCount >= response.Pagination.TotalCount {
				break
			}
		} else if len(response.Data) < pageSize {
			break
		}
	}
	return files, nil
}

func mapCurseFile(file curseFile) AddonVersion {
	return AddonVersion{
		VersionID:    strconv.FormatInt(file.ID, 10),
		VersionName:  file.DisplayName,
		VersionLabel: file.FileName,
		ReleaseType:  mapCurseReleaseType(file.ReleaseType),
		PublishedAt:  file.FileDate,
		DownloadURL:  file.DownloadURL,
		Filename:     file.FileName,
		Source:       AddonSourceCurseForge,
		FileID:       file.ID,
	}
}

func mapCurseFiles(files []curseFile) []AddonVersion {
	mapped := make([]AddonVersion, 0, len(files))
	for _, file := range files {
		mapped = append(mapped, mapCurseFile(file))
	}
	return mapped
}

func chooseLatestCurseFile(files []curseFile, mcVersion, loader string) *curseFile {
	compatible := listCompatibleCurseFiles(files, mcVersion, loader)
	if len(compatible) == 0 {
		return nil
	}
	return &compatible[0]
}

func listCompatibleCurseFiles(files []curseFile, mcVersion, loader string) []curseFile {
	compatible := make([]curseFile, 0)
	for _, file := range files {
		if !containsString(file.GameVersions, mcVersion) {
			continue
		}
		if !isCurseFileServerCompatible(file, loader) {
			continue
		}
		compatible = append(compatible, file)
	}
	sort.Slice(compatible, func(i, j int) bool {
		if compatible[i].ReleaseType != compatible[j].ReleaseType {
			return compatible[i].ReleaseType < compatible[j].ReleaseType
		}
		return compatible[i].FileDate > compatible[j].FileDate
	})
	return compatible
}

func isCurseFileServerCompatible(file curseFile, loader string) bool {
	loader = strings.ToLower(strings.TrimSpace(loader))
	if loader == "paper" {
		// Paper plugins are server-side by nature.
		return true
	}
	hasServer := false
	hasClient := false
	for _, gameVersion := range file.GameVersions {
		client, server := curseFileSideTags(gameVersion)
		hasClient = hasClient || client
		hasServer = hasServer || server
	}
	if hasServer {
		return true
	}
	// Explicit client-only should be excluded.
	if hasClient {
		return false
	}
	// Most CurseForge files don't tag side explicitly; keep them.
	return true
}

func curseFileSideTags(value string) (hasClient, hasServer bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return false, false
	}

	parts := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == '&' || r == '/' || r == ','
	})
	if len(parts) == 0 {
		parts = []string{normalized}
	}
	for _, part := range parts {
		switch strings.TrimSpace(part) {
		case "client":
			hasClient = true
		case "server":
			hasServer = true
		}
	}
	return hasClient, hasServer
}

func filterCurseSearchFiles(files []curseFile, indexes []curseFileIndex, mcVersion, loader string) []curseFile {
	modLoaderType, ok := curseForgeModLoaderType(loader)
	if !ok || len(indexes) == 0 {
		return files
	}

	allowedFileIDs := make(map[int64]struct{})
	hasVersionIndex := false
	for _, index := range indexes {
		if index.GameVersion != mcVersion {
			continue
		}
		hasVersionIndex = true
		if index.ModLoader != 0 && index.ModLoader != modLoaderType {
			continue
		}
		allowedFileIDs[index.FileID] = struct{}{}
	}

	if !hasVersionIndex {
		return files
	}
	if len(allowedFileIDs) == 0 {
		return files
	}
	filtered := make([]curseFile, 0, len(allowedFileIDs))
	for _, file := range files {
		if _, ok := allowedFileIDs[file.ID]; ok {
			filtered = append(filtered, file)
		}
	}
	if len(filtered) == 0 {
		// latestFilesIndexes can reference a file that is not included in the
		// abbreviated latestFiles payload. Let the full files endpoint decide.
		return files
	}
	return filtered
}

func isLikelyClientOnlyCurseProject(
	name, summary string,
	categories []struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	},
) bool {
	for _, category := range categories {
		categoryValue := strings.ToLower(strings.TrimSpace(category.Slug + " " + category.Name))
		if strings.Contains(categoryValue, "shader") ||
			strings.Contains(categoryValue, "resource-pack") ||
			strings.Contains(categoryValue, "texture-pack") {
			return true
		}
	}

	text := strings.ToLower(strings.TrimSpace(name + " " + summary))
	if strings.Contains(text, "shader") || strings.Contains(text, "resource pack") {
		return true
	}
	return false
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
