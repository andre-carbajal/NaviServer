package addons

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"naviserver/internal/domain"
	"naviserver/internal/server"
	"naviserver/internal/storage"
)

const (
	modrinthBaseURL   = "https://api.modrinth.com/v2"
	curseForgeBaseURL = "https://api.curseforge.com/v1"

	minecraftGameID = 432
)

type AddonSource string

type AddonStatus string

type AddonType string

type ReleaseType string

const (
	AddonSourceModrinth   AddonSource = "modrinth"
	AddonSourceCurseForge AddonSource = "curseforge"
	AddonSourceManual     AddonSource = "manual"

	AddonStatusInstalled       AddonStatus = "installed"
	AddonStatusUpdateAvailable AddonStatus = "update_available"
	AddonStatusUnknownSource   AddonStatus = "unknown_source"

	AddonTypeMod    AddonType = "mod"
	AddonTypePlugin AddonType = "plugin"

	ReleaseTypeRelease ReleaseType = "release"
	ReleaseTypeBeta    ReleaseType = "beta"
	ReleaseTypeAlpha   ReleaseType = "alpha"
)

type Dependency struct {
	ProjectID   string `json:"projectId"`
	FileID      int64  `json:"fileId,omitempty"`
	Name        string `json:"name,omitempty"`
	Required    bool   `json:"required"`
	Source      string `json:"source"`
	Description string `json:"description,omitempty"`
}

type AddonVersion struct {
	VersionID    string      `json:"versionId"`
	VersionName  string      `json:"versionName"`
	VersionLabel string      `json:"versionLabel"`
	ReleaseType  ReleaseType `json:"releaseType"`
	PublishedAt  string      `json:"publishedAt,omitempty"`
	DownloadURL  string      `json:"downloadUrl"`
	Filename     string      `json:"filename"`
	Source       AddonSource `json:"source"`
	FileID       int64       `json:"fileId,omitempty"`
}

type Addon struct {
	ID                  string        `json:"id"`
	Name                string        `json:"name"`
	FileName            string        `json:"fileName"`
	Path                string        `json:"path"`
	IconURL             string        `json:"iconUrl,omitempty"`
	Source              AddonSource   `json:"source"`
	Type                AddonType     `json:"type"`
	Status              AddonStatus   `json:"status"`
	ProjectID           string        `json:"projectId,omitempty"`
	ProjectSlug         string        `json:"projectSlug,omitempty"`
	ProjectName         string        `json:"projectName,omitempty"`
	ProjectURL          string        `json:"projectUrl,omitempty"`
	VersionID           string        `json:"versionId,omitempty"`
	VersionName         string        `json:"versionName,omitempty"`
	VersionLabel        string        `json:"versionLabel,omitempty"`
	ReleaseType         ReleaseType   `json:"releaseType,omitempty"`
	HashSHA1            string        `json:"hashSha1,omitempty"`
	HashSHA512          string        `json:"hashSha512,omitempty"`
	CurseFingerprint    uint32        `json:"curseFingerprint,omitempty"`
	Size                int64         `json:"size"`
	ModifiedAt          string        `json:"modifiedAt"`
	Latest              *AddonVersion `json:"latest,omitempty"`
	MissingDependencies []Dependency  `json:"missingDependencies,omitempty"`
	Disabled            bool          `json:"disabled"`
}

type ListResponse struct {
	AddonType AddonType `json:"addonType"`
	Items     []Addon   `json:"items"`
}

type SearchRequest struct {
	Query  string `json:"query"`
	Source string `json:"source"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

type VersionsRequest struct {
	Source    AddonSource `json:"source"`
	ProjectID string      `json:"projectId"`
}

type VersionsResponse struct {
	Versions []AddonVersion `json:"versions"`
}

type SearchResponse struct {
	Items      []SearchResult `json:"items"`
	HasMore    bool           `json:"hasMore"`
	NextOffset int            `json:"nextOffset"`
}

type SearchResult struct {
	Source      AddonSource    `json:"source"`
	ProjectID   string         `json:"projectId"`
	ProjectSlug string         `json:"projectSlug,omitempty"`
	ProjectName string         `json:"projectName"`
	AuthorName  string         `json:"authorName,omitempty"`
	Description string         `json:"description,omitempty"`
	ProjectURL  string         `json:"projectUrl,omitempty"`
	IconURL     string         `json:"iconUrl,omitempty"`
	Downloads   int64          `json:"downloads,omitempty"`
	Latest      *AddonVersion  `json:"latest,omitempty"`
	Versions    []AddonVersion `json:"versions,omitempty"`
}

type InstallRequest struct {
	Source              AddonSource `json:"source"`
	ProjectID           string      `json:"projectId"`
	VersionID           string      `json:"versionId,omitempty"`
	FileID              int64       `json:"fileId,omitempty"`
	IncludeDependencies bool        `json:"includeDependencies"`
}

type UpdateAllResult struct {
	Updated []Addon `json:"updated"`
	Failed  []struct {
		ID    string `json:"id"`
		Error string `json:"error"`
	} `json:"failed"`
}

type Manager struct {
	serverManager *server.Manager
	store         *storage.GormStore
	httpClient    *http.Client
	userAgent     string
}

// BuildCurseForgeAPIKey is injected at build time via ldflags in CI.
var BuildCurseForgeAPIKey = ""

type scannedAddonFile struct {
	fileName    string
	fullPath    string
	relPath     string
	size        int64
	modifiedAt  time.Time
	sha1        string
	sha512      string
	fingerprint uint32
	disabled    bool
}

func NewManager(serverManager *server.Manager, store *storage.GormStore) *Manager {
	return &Manager{
		serverManager: serverManager,
		store:         store,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "andre-carbajal/naviserver",
	}
}

type CurseForgeKeyStatus struct {
	HasCustomKey    bool   `json:"hasCustomKey"`
	HasEmbeddedKey  bool   `json:"hasEmbeddedKey"`
	EffectiveSource string `json:"effectiveSource"`
}

func (m *Manager) resolveCurseForgeAPIKey() string {
	custom, err := m.getCustomCurseForgeAPIKey()
	if err == nil && strings.TrimSpace(custom) != "" {
		return strings.TrimSpace(custom)
	}
	return strings.TrimSpace(BuildCurseForgeAPIKey)
}

func (m *Manager) getCustomCurseForgeAPIKey() (string, error) {
	if m.store == nil {
		return "", fmt.Errorf("settings store not configured")
	}
	value, err := m.store.GetSetting("curseforge_api_key")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func (m *Manager) GetCurseForgeKeyStatus() CurseForgeKeyStatus {
	custom, _ := m.getCustomCurseForgeAPIKey()
	hasCustom := strings.TrimSpace(custom) != ""
	hasEmbedded := strings.TrimSpace(BuildCurseForgeAPIKey) != ""
	effective := "none"
	if hasCustom {
		effective = "custom"
	} else if hasEmbedded {
		effective = "embedded"
	}
	return CurseForgeKeyStatus{
		HasCustomKey:    hasCustom,
		HasEmbeddedKey:  hasEmbedded,
		EffectiveSource: effective,
	}
}

func (m *Manager) SetCustomCurseForgeAPIKey(key string) error {
	if m.store == nil {
		return fmt.Errorf("settings store not configured")
	}
	return m.store.SetSetting("curseforge_api_key", strings.TrimSpace(key))
}

func (m *Manager) ClearCustomCurseForgeAPIKey() error {
	if m.store == nil {
		return fmt.Errorf("settings store not configured")
	}
	return m.store.SetSetting("curseforge_api_key", "")
}

func (m *Manager) ListAddons(ctx context.Context, serverID string) (*ListResponse, error) {
	srv, root, addonType, addonDir, loaders, err := m.resolveServerAddonContext(serverID)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(addonDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &ListResponse{AddonType: addonType, Items: []Addon{}}, nil
		}
		return nil, fmt.Errorf("failed to read addons directory: %w", err)
	}

	scannedItems := make([]scannedAddonFile, 0)
	for _, entry := range entries {
		if entry.IsDir() || !isAddonJarFile(entry.Name()) {
			continue
		}
		entryName := entry.Name()
		fullPath := filepath.Join(addonDir, entryName)
		info, err := entry.Info()
		if err != nil {
			continue
		}
		sha1Hash, sha512Hash, fp, err := computeJarMetadata(fullPath)
		if err != nil {
			continue
		}
		relPath := strings.ReplaceAll(strings.TrimPrefix(fullPath, root), "\\", "/")
		if !strings.HasPrefix(relPath, "/") {
			relPath = "/" + relPath
		}
		scannedItems = append(scannedItems, scannedAddonFile{
			fileName:    entryName,
			fullPath:    fullPath,
			relPath:     relPath,
			size:        info.Size(),
			modifiedAt:  info.ModTime(),
			sha1:        sha1Hash,
			sha512:      sha512Hash,
			fingerprint: fp,
			disabled:    isAddonDisabledFile(entryName),
		})
	}

	if len(scannedItems) == 0 {
		return &ListResponse{AddonType: addonType, Items: []Addon{}}, nil
	}

	modrinthCurrent, _ := m.modrinthVersionsFromHashes(ctx, scannedItems)
	modrinthLatest, _ := m.modrinthLatestFromHashes(ctx, scannedItems, loaders, []string{srv.Version})
	curseMatches, _ := m.curseForgeByFingerprints(ctx, scannedItems)
	modrinthIconCache := map[string]string{}

	addons := make([]Addon, 0, len(scannedItems))
	for _, item := range scannedItems {
		_ = item.fullPath
		addon := Addon{
			ID:               item.fileName,
			Name:             trimJarSuffix(item.fileName),
			FileName:         item.fileName,
			Path:             item.relPath,
			Source:           AddonSourceManual,
			Type:             addonType,
			Status:           AddonStatusUnknownSource,
			HashSHA1:         item.sha1,
			HashSHA512:       item.sha512,
			CurseFingerprint: item.fingerprint,
			Size:             item.size,
			ModifiedAt:       item.modifiedAt.UTC().Format(time.RFC3339),
			Disabled:         item.disabled,
		}

		if current, ok := modrinthCurrent[item.sha1]; ok {
			addon.Source = AddonSourceModrinth
			addon.Status = AddonStatusInstalled
			addon.ProjectID = current.ProjectID
			addon.ProjectName = current.ProjectTitle
			addon.ProjectSlug = current.ProjectSlug
			addon.ProjectURL = "https://modrinth.com/project/" + current.ProjectID
			if current.ProjectID != "" {
				if cached, ok := modrinthIconCache[current.ProjectID]; ok {
					addon.IconURL = cached
				} else {
					iconURL, _ := m.getModrinthProjectIcon(ctx, current.ProjectID)
					addon.IconURL = iconURL
					modrinthIconCache[current.ProjectID] = iconURL
				}
			}
			addon.VersionID = current.VersionID
			addon.VersionName = current.VersionName
			addon.VersionLabel = current.VersionNumber
			addon.ReleaseType = ReleaseType(current.VersionType)

			if latest, hasLatest := modrinthLatest[item.sha1]; hasLatest && !item.disabled {
				if latest.VersionID != "" && latest.VersionID != current.VersionID {
					addon.Status = AddonStatusUpdateAvailable
					addon.Latest = &AddonVersion{
						VersionID:    latest.VersionID,
						VersionName:  latest.VersionName,
						VersionLabel: latest.VersionNumber,
						ReleaseType:  ReleaseType(latest.VersionType),
						PublishedAt:  latest.DatePublished,
						DownloadURL:  latest.PrimaryFileURL(),
						Filename:     latest.PrimaryFilename(),
						Source:       AddonSourceModrinth,
					}
				}
			}
		} else if match, ok := curseMatches[item.fingerprint]; ok {
			addon.Source = AddonSourceCurseForge
			addon.Status = AddonStatusInstalled
			addon.ProjectID = strconv.FormatInt(match.File.ModID, 10)
			addon.ProjectName = match.Mod.Name
			addon.ProjectSlug = match.Mod.Slug
			addon.ProjectURL = match.Mod.Links.WebsiteURL
			addon.IconURL = match.Mod.Logo.URL
			addon.VersionID = strconv.FormatInt(match.File.ID, 10)
			addon.VersionName = match.File.DisplayName
			addon.VersionLabel = match.File.FileName
			addon.ReleaseType = mapCurseReleaseType(match.File.ReleaseType)

			latest, deps := m.findLatestCurseFile(ctx, match.File.ModID, srv.Version, srv.Loader)
			if latest != nil && latest.ID != match.File.ID && !item.disabled {
				addon.Status = AddonStatusUpdateAvailable
				addon.Latest = &AddonVersion{
					VersionID:    strconv.FormatInt(latest.ID, 10),
					VersionName:  latest.DisplayName,
					VersionLabel: latest.FileName,
					ReleaseType:  mapCurseReleaseType(latest.ReleaseType),
					PublishedAt:  latest.FileDate,
					DownloadURL:  latest.DownloadURL,
					Filename:     latest.FileName,
					Source:       AddonSourceCurseForge,
					FileID:       latest.ID,
				}
			}
			addon.MissingDependencies = deps
		}

		addons = append(addons, addon)
	}

	sort.Slice(addons, func(i, j int) bool {
		if addons[i].Status != addons[j].Status {
			return addons[i].Status < addons[j].Status
		}
		return strings.ToLower(addons[i].Name) < strings.ToLower(addons[j].Name)
	})

	return &ListResponse{AddonType: addonType, Items: addons}, nil
}

func (m *Manager) SearchAddons(
	ctx context.Context,
	serverID,
	query,
	source string,
	offset,
	limit int,
) (*SearchResponse, error) {
	srv, _, _, _, loaders, err := m.resolveServerAddonContext(serverID)
	if err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	results := make([]SearchResult, 0, limit)
	allowModrinth := source == "" || source == "all" || source == string(AddonSourceModrinth)
	allowCurse := source == "" || source == "all" || source == string(AddonSourceCurseForge)
	hasMore := false

	if allowModrinth {
		items, modrinthHasMore, err := m.searchModrinth(
			ctx,
			query,
			loaders,
			srv.Version,
			offset,
			limit,
		)
		if err == nil {
			results = append(results, items...)
			hasMore = hasMore || modrinthHasMore
		}
	}

	if allowCurse {
		items, curseHasMore, err := m.searchCurseForge(
			ctx,
			query,
			srv.Version,
			srv.Loader,
			offset,
			limit,
		)
		if err != nil {
			if errors.Is(err, errCurseForgeKeyMissing) && allowModrinth {
				// Keep Modrinth results working even when CurseForge is not configured.
			} else if errors.Is(err, errCurseForgeKeyMissing) {
				return nil, err
			}
		} else {
			results = append(results, items...)
			hasMore = hasMore || curseHasMore
		}
	}

	nextOffset := offset + limit
	if !hasMore {
		nextOffset = offset + len(results)
	}

	return &SearchResponse{
		Items:      results,
		HasMore:    hasMore,
		NextOffset: nextOffset,
	}, nil
}

func (m *Manager) ListAddonVersions(
	ctx context.Context,
	serverID string,
	req VersionsRequest,
) (*VersionsResponse, error) {
	srv, _, _, _, loaders, err := m.resolveServerAddonContext(serverID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.ProjectID) == "" {
		return nil, fmt.Errorf("projectId is required")
	}

	switch req.Source {
	case AddonSourceModrinth:
		versions, err := m.listModrinthVersions(ctx, req.ProjectID, loaders, srv.Version)
		if err != nil {
			return nil, err
		}
		return &VersionsResponse{Versions: mapModrinthVersions(versions)}, nil
	case AddonSourceCurseForge:
		modID, err := strconv.ParseInt(req.ProjectID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid curseforge project id")
		}
		files, err := m.listCurseFiles(ctx, modID, srv.Version)
		if err != nil {
			return nil, err
		}
		return &VersionsResponse{
			Versions: mapCurseFiles(
				listCompatibleCurseFiles(files, srv.Version, srv.Loader),
			),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported source: %s", req.Source)
	}
}

func (m *Manager) InstallAddon(ctx context.Context, serverID string, req InstallRequest) error {
	srv, _, _, addonDir, loaders, err := m.resolveServerAddonContext(serverID)
	if err != nil {
		return err
	}
	if srv.Status != "STOPPED" {
		return fmt.Errorf("server must be stopped to modify addons")
	}
	if req.ProjectID == "" {
		return fmt.Errorf("projectId is required")
	}

	source := req.Source
	if source == "" {
		source = AddonSourceModrinth
	}

	switch source {
	case AddonSourceModrinth:
		version, err := m.resolveModrinthVersion(ctx, req.ProjectID, req.VersionID, loaders, srv.Version)
		if err != nil {
			return err
		}
		if version == nil {
			return fmt.Errorf("no compatible version found")
		}
		primaryURL := version.PrimaryFileURL()
		primaryName := version.PrimaryFilename()
		if primaryURL == "" || primaryName == "" {
			return fmt.Errorf("version does not provide a downloadable file")
		}
		targetPath := filepath.Join(addonDir, primaryName)
		if err := m.downloadToFile(ctx, primaryURL, targetPath, nil); err != nil {
			return err
		}
		if req.IncludeDependencies {
			for _, dep := range version.Dependencies {
				if dep.DependencyType != "required" || dep.ProjectID == "" {
					continue
				}
				depVersion, err := m.resolveModrinthVersion(ctx, dep.ProjectID, dep.VersionID, loaders, srv.Version)
				if err != nil || depVersion == nil {
					continue
				}
				depURL := depVersion.PrimaryFileURL()
				depName := depVersion.PrimaryFilename()
				if depURL == "" || depName == "" {
					continue
				}
				_ = m.downloadToFile(ctx, depURL, filepath.Join(addonDir, depName), nil)
			}
		}
		return nil
	case AddonSourceCurseForge:
		if strings.TrimSpace(m.resolveCurseForgeAPIKey()) == "" {
			return errCurseForgeKeyMissing
		}
		file, deps, err := m.resolveCurseForgeFile(ctx, req.ProjectID, req.FileID, srv.Version, srv.Loader)
		if err != nil {
			return err
		}
		if file == nil {
			return fmt.Errorf("no compatible file found")
		}
		downloadURL := file.DownloadURL
		if downloadURL == "" {
			downloadURL, err = m.getCurseFileDownloadURL(ctx, file.ModID, file.ID)
			if err != nil {
				return err
			}
		}
		if downloadURL == "" {
			return fmt.Errorf("file has no download url")
		}
		if err := m.downloadToFile(ctx, downloadURL, filepath.Join(addonDir, file.FileName), map[string]string{"x-api-key": m.resolveCurseForgeAPIKey()}); err != nil {
			return err
		}
		if req.IncludeDependencies {
			for _, dep := range deps {
				if !dep.Required || dep.ProjectID == "" {
					continue
				}
				depFile, _, err := m.resolveCurseForgeFile(ctx, dep.ProjectID, dep.FileID, srv.Version, srv.Loader)
				if err != nil || depFile == nil {
					continue
				}
				depURL := depFile.DownloadURL
				if depURL == "" {
					depURL, _ = m.getCurseFileDownloadURL(ctx, depFile.ModID, depFile.ID)
				}
				if depURL == "" {
					continue
				}
				_ = m.downloadToFile(ctx, depURL, filepath.Join(addonDir, depFile.FileName), map[string]string{"x-api-key": m.resolveCurseForgeAPIKey()})
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported source: %s", source)
	}
}

func (m *Manager) DeleteAddon(serverID, addonID string) error {
	srv, _, _, addonDir, _, err := m.resolveServerAddonContext(serverID)
	if err != nil {
		return err
	}
	if srv.Status != "STOPPED" {
		return fmt.Errorf("server must be stopped to modify addons")
	}
	if strings.TrimSpace(addonID) == "" {
		return fmt.Errorf("addon id is required")
	}

	base := normalizeAddonFileName(addonID)
	target := filepath.Join(addonDir, base)
	if err := os.Remove(target); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("addon not found")
		}
		return err
	}
	return nil
}

func (m *Manager) SetAddonDisabled(serverID, addonID string, disabled bool) error {
	srv, _, _, addonDir, _, err := m.resolveServerAddonContext(serverID)
	if err != nil {
		return err
	}
	if srv.Status != "STOPPED" {
		return fmt.Errorf("server must be stopped to modify addons")
	}
	if strings.TrimSpace(addonID) == "" {
		return fmt.Errorf("addon id is required")
	}

	base := normalizeAddonFileName(addonID)
	currentDisabled := isAddonDisabledFile(base)
	if currentDisabled == disabled {
		return nil
	}

	source := filepath.Join(addonDir, base)
	targetName := base + ".disabled"
	if !disabled {
		targetName = strings.TrimSuffix(base, ".disabled")
	}
	target := filepath.Join(addonDir, targetName)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("target addon already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(source, target); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("addon not found")
		}
		return err
	}
	return nil
}

func (m *Manager) UpdateAddon(ctx context.Context, serverID, addonID string, includeDependencies bool) error {
	list, err := m.ListAddons(ctx, serverID)
	if err != nil {
		return err
	}
	var addon *Addon
	for i := range list.Items {
		if list.Items[i].ID == addonID {
			addon = &list.Items[i]
			break
		}
	}
	if addon == nil {
		return fmt.Errorf("addon not found")
	}
	if addon.Status != AddonStatusUpdateAvailable || addon.Latest == nil {
		return fmt.Errorf("addon does not have an update available")
	}
	if addon.Source == AddonSourceModrinth {
		if err := m.InstallAddon(ctx, serverID, InstallRequest{
			Source:              AddonSourceModrinth,
			ProjectID:           addon.ProjectID,
			VersionID:           addon.Latest.VersionID,
			IncludeDependencies: includeDependencies,
		}); err != nil {
			return err
		}
		if addon.Latest.Filename != "" &&
			addon.FileName != "" &&
			addon.Latest.Filename != addon.FileName {
			_ = m.DeleteAddon(serverID, addon.FileName)
		}
		return nil
	}
	if addon.Source == AddonSourceCurseForge {
		if err := m.InstallAddon(ctx, serverID, InstallRequest{
			Source:              AddonSourceCurseForge,
			ProjectID:           addon.ProjectID,
			FileID:              addon.Latest.FileID,
			IncludeDependencies: includeDependencies,
		}); err != nil {
			return err
		}
		if addon.Latest.Filename != "" &&
			addon.FileName != "" &&
			addon.Latest.Filename != addon.FileName {
			_ = m.DeleteAddon(serverID, addon.FileName)
		}
		return nil
	}
	return fmt.Errorf("unsupported addon source for update")
}

func (m *Manager) UpdateAllAddons(ctx context.Context, serverID string, includeDependencies bool) (*UpdateAllResult, error) {
	list, err := m.ListAddons(ctx, serverID)
	if err != nil {
		return nil, err
	}
	result := &UpdateAllResult{Updated: []Addon{}}
	for _, addon := range list.Items {
		if addon.Status != AddonStatusUpdateAvailable {
			continue
		}
		if err := m.UpdateAddon(ctx, serverID, addon.ID, includeDependencies); err != nil {
			result.Failed = append(result.Failed, struct {
				ID    string `json:"id"`
				Error string `json:"error"`
			}{ID: addon.ID, Error: err.Error()})
			continue
		}
		result.Updated = append(result.Updated, addon)
	}
	return result, nil
}

func (m *Manager) resolveServerAddonContext(serverID string) (*domain.Server, string, AddonType, string, []string, error) {
	srv, err := m.serverManager.GetServer(serverID)
	if err != nil {
		return nil, "", "", "", nil, err
	}
	if srv == nil {
		return nil, "", "", "", nil, fmt.Errorf("server not found")
	}

	folderName := srv.FolderName
	if strings.TrimSpace(folderName) == "" {
		folderName = srv.ID
	}
	root := filepath.Join(m.serverManager.ServersPath, folderName)
	addonType, relativePath, loaders, ok := addonScopeForLoader(strings.ToLower(srv.Loader))
	if !ok {
		return nil, "", "", "", nil, fmt.Errorf("addon management is only supported for paper, fabric, forge and neoforge servers")
	}
	addonDir := filepath.Join(root, relativePath)
	if err := os.MkdirAll(addonDir, 0755); err != nil {
		return nil, "", "", "", nil, fmt.Errorf("failed to create addon directory: %w", err)
	}
	return srv, root, addonType, addonDir, loaders, nil
}

func addonScopeForLoader(loader string) (AddonType, string, []string, bool) {
	switch loader {
	case "paper":
		return AddonTypePlugin, "plugins", []string{"paper", "spigot", "bukkit", "purpur", "minecraft"}, true
	case "fabric":
		return AddonTypeMod, "mods", []string{"fabric"}, true
	case "forge":
		return AddonTypeMod, "mods", []string{"forge"}, true
	case "neoforge":
		return AddonTypeMod, "mods", []string{"neoforge"}, true
	default:
		return "", "", nil, false
	}
}

func trimJarSuffix(name string) string {
	name = strings.TrimSuffix(name, ".disabled")
	if strings.HasSuffix(strings.ToLower(name), ".jar") {
		return name[:len(name)-4]
	}
	return name
}

func isAddonJarFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".jar") ||
		strings.HasSuffix(lower, ".jar.disabled")
}

func isAddonDisabledFile(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".jar.disabled")
}

func normalizeAddonFileName(addonID string) string {
	base := filepath.Base(addonID)
	lower := strings.ToLower(base)
	if strings.HasSuffix(lower, ".jar") || strings.HasSuffix(lower, ".jar.disabled") {
		return base
	}
	return base + ".jar"
}

func computeJarMetadata(path string) (sha1Hash string, sha512Hash string, fingerprint uint32, err error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", "", 0, err
	}
	h1 := sha1.Sum(content)
	h512 := sha512.Sum512(content)
	return hex.EncodeToString(h1[:]), hex.EncodeToString(h512[:]), curseForgeFingerprint(content), nil
}

func (m *Manager) downloadToFile(ctx context.Context, url, targetPath string, extraHeaders map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("download failed (%d): %s", resp.StatusCode, string(body))
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
	tmp := targetPath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, targetPath)
}

func (m *Manager) doJSON(ctx context.Context, method, endpoint string, body any, headers map[string]string, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", m.userAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}

func modrinthFacets(loaders []string, mcVersion string) string {
	parts := make([]string, 0, len(loaders)+2)
	if mcVersion != "" {
		parts = append(parts, fmt.Sprintf("[\"versions:%s\"]", mcVersion))
	}
	if len(loaders) > 0 {
		loaderFacets := make([]string, 0, len(loaders))
		for _, loader := range loaders {
			loaderFacets = append(loaderFacets, fmt.Sprintf("\"categories:%s\"", loader))
		}
		parts = append(parts, "["+strings.Join(loaderFacets, ",")+"]")
	}
	if shouldRestrictToServerSide(loaders) {
		parts = append(parts, "[\"server_side:required\",\"server_side:optional\"]")
	}
	if len(parts) == 0 {
		return "[]"
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func shouldRestrictToServerSide(loaders []string) bool {
	for _, loader := range loaders {
		switch strings.ToLower(loader) {
		case "fabric", "forge", "neoforge":
			return true
		}
	}
	return false
}

var errCurseForgeKeyMissing = errors.New("curseforge api key is not configured (set CURSEFORGE_API_KEY)")

func encodeQuery(value string) string {
	return url.QueryEscape(value)
}

func IsCurseForgeKeyMissing(err error) bool {
	return errors.Is(err, errCurseForgeKeyMissing)
}
