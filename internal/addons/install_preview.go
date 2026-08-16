package addons

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// InstallPreviewRequest selects the same project version or file that the
// install endpoint will use, without downloading anything.
type InstallPreviewRequest struct {
	Source    AddonSource `json:"source"`
	ProjectID string      `json:"projectId"`
	VersionID string      `json:"versionId,omitempty"`
	FileID    int64       `json:"fileId,omitempty"`
}

type InstallPreviewDependency struct {
	Name         string      `json:"name"`
	Source       AddonSource `json:"source"`
	ProjectID    string      `json:"projectId"`
	IconURL      string      `json:"iconUrl,omitempty"`
	VersionID    string      `json:"versionId,omitempty"`
	FileID       int64       `json:"fileId,omitempty"`
	VersionLabel string      `json:"versionLabel,omitempty"`
	Filename     string      `json:"filename,omitempty"`
}

type InstallPreviewResponse struct {
	Dependencies []InstallPreviewDependency `json:"dependencies"`
}

// resolvedAddonFile is the catalog object selected for one node in an addon
// dependency graph. Keeping the source-specific object here lets preview and
// installation use the same resolution path while retaining the data needed
// by each downloader.
type resolvedAddonFile struct {
	Source       AddonSource
	ProjectID    string
	VersionID    string
	FileID       int64
	Name         string
	IconURL      string
	VersionLabel string
	Filename     string

	modrinth   *modrinthVersion
	curseForge *curseFile
}

func resolvedModrinthFile(projectID string, version modrinthVersion) resolvedAddonFile {
	projectID = strings.TrimSpace(projectID)
	if strings.TrimSpace(version.ProjectID) != "" {
		projectID = strings.TrimSpace(version.ProjectID)
	}
	name := strings.TrimSpace(version.ProjectTitle)
	if name == "" {
		name = projectID
	}
	return resolvedAddonFile{
		Source:       AddonSourceModrinth,
		ProjectID:    projectID,
		VersionID:    version.VersionID,
		Name:         name,
		VersionLabel: version.VersionNumber,
		Filename:     version.PrimaryFilename(),
		modrinth:     &version,
	}
}

func resolvedCurseForgeFile(projectID string, file curseFile) resolvedAddonFile {
	projectID = strings.TrimSpace(projectID)
	if file.ModID != 0 {
		projectID = strconv.FormatInt(file.ModID, 10)
	}
	versionLabel := strings.TrimSpace(file.DisplayName)
	if versionLabel == "" {
		versionLabel = file.FileName
	}
	return resolvedAddonFile{
		Source:       AddonSourceCurseForge,
		ProjectID:    projectID,
		VersionID:    strconv.FormatInt(file.ID, 10),
		FileID:       file.ID,
		Name:         projectID,
		VersionLabel: versionLabel,
		Filename:     file.FileName,
		curseForge:   &file,
	}
}

func (m *Manager) resolveModrinthAddonGraph(
	ctx context.Context,
	projectID,
	versionID string,
	loaders []string,
	mcVersion string,
	includeDependencies bool,
) (resolvedAddonFile, []resolvedAddonFile, error) {
	version, err := m.resolveModrinthVersion(ctx, projectID, versionID, loaders, mcVersion)
	if err != nil {
		return resolvedAddonFile{}, nil, err
	}
	if version == nil {
		return resolvedAddonFile{}, nil, fmt.Errorf("no compatible version found")
	}
	if version.PrimaryFileURL() == "" || version.PrimaryFilename() == "" {
		return resolvedAddonFile{}, nil, fmt.Errorf("version does not provide a downloadable file")
	}

	root := resolvedModrinthFile(projectID, *version)
	if !includeDependencies {
		return root, nil, nil
	}
	visited := map[string]struct{}{root.ProjectID: {}}
	dependencies, err := m.resolveModrinthDependencies(
		ctx,
		mcVersion,
		loaders,
		version.Dependencies,
		visited,
	)
	if err != nil {
		return resolvedAddonFile{}, nil, err
	}
	return root, dependencies, nil
}

func (m *Manager) resolveModrinthDependencies(
	ctx context.Context,
	mcVersion string,
	loaders []string,
	dependencies []modrinthDependency,
	visited map[string]struct{},
) ([]resolvedAddonFile, error) {
	resolved := make([]resolvedAddonFile, 0)
	for _, dependency := range dependencies {
		if dependency.DependencyType != "required" {
			continue
		}
		projectID := strings.TrimSpace(dependency.ProjectID)
		if projectID == "" {
			return nil, fmt.Errorf("required Modrinth dependency has no project id")
		}
		if _, ok := visited[projectID]; ok {
			continue
		}
		visited[projectID] = struct{}{}

		version, err := m.resolveModrinthVersion(
			ctx,
			projectID,
			dependency.VersionID,
			loaders,
			mcVersion,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve required Modrinth dependency %s: %w", projectID, err)
		}
		if version == nil {
			return nil, fmt.Errorf("no compatible Modrinth dependency found for project %s", projectID)
		}
		if version.PrimaryFileURL() == "" || version.PrimaryFilename() == "" {
			return nil, fmt.Errorf("Modrinth dependency %s does not provide a downloadable file", projectID)
		}

		nested, err := m.resolveModrinthDependencies(
			ctx,
			mcVersion,
			loaders,
			version.Dependencies,
			visited,
		)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, nested...)
		resolved = append(resolved, resolvedModrinthFile(projectID, *version))
	}
	return resolved, nil
}

func (m *Manager) resolveCurseForgeAddonGraph(
	ctx context.Context,
	projectID string,
	fileID int64,
	mcVersion,
	loader string,
	includeDependencies bool,
) (resolvedAddonFile, []resolvedAddonFile, error) {
	file, rawDependencies, err := m.resolveCurseForgeFile(ctx, projectID, fileID, mcVersion, loader)
	if err != nil {
		return resolvedAddonFile{}, nil, err
	}
	if file == nil {
		return resolvedAddonFile{}, nil, fmt.Errorf("no compatible file found")
	}
	if strings.TrimSpace(file.FileName) == "" {
		return resolvedAddonFile{}, nil, fmt.Errorf("CurseForge file has no filename")
	}

	root := resolvedCurseForgeFile(projectID, *file)
	if !includeDependencies {
		return root, nil, nil
	}
	visited := map[int64]struct{}{file.ModID: {}}
	dependencies, err := m.resolveCurseForgeDependencies(
		ctx,
		mcVersion,
		loader,
		rawDependencies,
		visited,
	)
	if err != nil {
		return resolvedAddonFile{}, nil, err
	}
	return root, dependencies, nil
}

func (m *Manager) resolveCurseForgeDependencies(
	ctx context.Context,
	mcVersion,
	loader string,
	dependencies []Dependency,
	visited map[int64]struct{},
) ([]resolvedAddonFile, error) {
	resolved := make([]resolvedAddonFile, 0)
	for _, dependency := range dependencies {
		if !dependency.Required {
			continue
		}
		if dependency.Source != string(AddonSourceCurseForge) {
			return nil, fmt.Errorf("unsupported source for required CurseForge dependency %s", dependency.ProjectID)
		}
		projectID := strings.TrimSpace(dependency.ProjectID)
		modID, err := strconv.ParseInt(projectID, 10, 64)
		if err != nil || modID <= 0 {
			return nil, fmt.Errorf("invalid CurseForge dependency project id %q", dependency.ProjectID)
		}
		if _, ok := visited[modID]; ok {
			continue
		}
		visited[modID] = struct{}{}

		file, nestedDependencies, err := m.resolveCurseForgeFile(
			ctx,
			projectID,
			dependency.FileID,
			mcVersion,
			loader,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve CurseForge dependency %s: %w", projectID, err)
		}
		if file == nil {
			return nil, fmt.Errorf("no compatible CurseForge dependency found for project %s", projectID)
		}
		if strings.TrimSpace(file.FileName) == "" {
			return nil, fmt.Errorf("CurseForge dependency %s has no filename", projectID)
		}

		nested, err := m.resolveCurseForgeDependencies(
			ctx,
			mcVersion,
			loader,
			nestedDependencies,
			visited,
		)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, nested...)
		resolved = append(resolved, resolvedCurseForgeFile(projectID, *file))
	}
	return resolved, nil
}

func addonSourceProjectKey(source AddonSource, projectID string) string {
	return strings.ToLower(strings.TrimSpace(string(source))) + ":" + strings.TrimSpace(projectID)
}

func (m *Manager) installedAddonProjectKeys(ctx context.Context, serverID string) (map[string]struct{}, error) {
	list, err := m.listAddonsReadOnly(ctx, serverID)
	if err != nil {
		return nil, err
	}
	keys := make(map[string]struct{})
	for _, addon := range list.Items {
		if addon.ProjectID == "" || !validAddonProvenanceSource(addon.Source) {
			continue
		}
		keys[addonSourceProjectKey(addon.Source, addon.ProjectID)] = struct{}{}
	}
	return keys, nil
}

func (m *Manager) enrichCurseForgeDependencyNames(ctx context.Context, dependencies []resolvedAddonFile) error {
	ids := make([]int64, 0, len(dependencies))
	seen := make(map[int64]struct{})
	for _, dependency := range dependencies {
		modID, err := strconv.ParseInt(dependency.ProjectID, 10, 64)
		if err != nil || modID <= 0 {
			continue
		}
		if _, ok := seen[modID]; ok {
			continue
		}
		seen[modID] = struct{}{}
		ids = append(ids, modID)
	}
	mods, err := m.fetchCurseForgeMods(ctx, ids)
	if err != nil {
		return err
	}
	for index := range dependencies {
		if mod, ok := mods[dependencies[index].ProjectID]; ok {
			if strings.TrimSpace(mod.Name) != "" {
				dependencies[index].Name = strings.TrimSpace(mod.Name)
			}
			if strings.TrimSpace(dependencies[index].IconURL) == "" {
				dependencies[index].IconURL = strings.TrimSpace(mod.iconURL())
			}
		}
		if strings.TrimSpace(dependencies[index].Name) == "" {
			dependencies[index].Name = dependencies[index].ProjectID
		}
	}
	return nil
}

func (m *Manager) enrichModrinthDependencyNames(ctx context.Context, dependencies []resolvedAddonFile) {
	for index := range dependencies {
		if dependencies[index].Name != dependencies[index].ProjectID && dependencies[index].IconURL != "" {
			continue
		}

		var project struct {
			Title   string `json:"title"`
			IconURL string `json:"icon_url"`
		}
		if err := m.doJSON(
			ctx,
			http.MethodGet,
			modrinthBaseURL+"/project/"+url.PathEscape(dependencies[index].ProjectID),
			nil,
			nil,
			&project,
		); err == nil {
			if dependencies[index].Name == dependencies[index].ProjectID && strings.TrimSpace(project.Title) != "" {
				dependencies[index].Name = strings.TrimSpace(project.Title)
			}
			if strings.TrimSpace(dependencies[index].IconURL) == "" {
				dependencies[index].IconURL = strings.TrimSpace(project.IconURL)
			}
		}
	}
}

func (m *Manager) PreviewInstallAddon(
	ctx context.Context,
	serverID string,
	req InstallPreviewRequest,
) (*InstallPreviewResponse, error) {
	srv, _, _, _, loaders, err := m.resolveServerAddonContextWithOptions(serverID, false)
	if err != nil {
		return nil, err
	}
	projectID := strings.TrimSpace(req.ProjectID)
	if projectID == "" {
		return nil, fmt.Errorf("projectId is required")
	}

	source := req.Source
	if source == "" {
		source = AddonSourceModrinth
	}
	var dependencies []resolvedAddonFile
	switch source {
	case AddonSourceModrinth:
		_, dependencies, err = m.resolveModrinthAddonGraph(
			ctx,
			projectID,
			req.VersionID,
			loaders,
			srv.Version,
			true,
		)
	case AddonSourceCurseForge:
		if strings.TrimSpace(m.resolveCurseForgeAPIKey()) == "" {
			return nil, errCurseForgeKeyMissing
		}
		_, dependencies, err = m.resolveCurseForgeAddonGraph(
			ctx,
			projectID,
			req.FileID,
			srv.Version,
			srv.Loader,
			true,
		)
	default:
		return nil, fmt.Errorf("unsupported source: %s", source)
	}
	if err != nil {
		return nil, err
	}

	if source == AddonSourceCurseForge && len(dependencies) > 0 {
		if err := m.enrichCurseForgeDependencyNames(ctx, dependencies); err != nil {
			return nil, err
		}
	}
	if source == AddonSourceModrinth && len(dependencies) > 0 {
		m.enrichModrinthDependencyNames(ctx, dependencies)
	}

	installed, err := m.installedAddonProjectKeys(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect installed addons: %w", err)
	}

	response := &InstallPreviewResponse{Dependencies: make([]InstallPreviewDependency, 0, len(dependencies))}
	seen := make(map[string]struct{})
	for _, dependency := range dependencies {
		key := addonSourceProjectKey(dependency.Source, dependency.ProjectID)
		if _, ok := installed[key]; ok {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		name := strings.TrimSpace(dependency.Name)
		if name == "" {
			name = dependency.ProjectID
		}
		response.Dependencies = append(response.Dependencies, InstallPreviewDependency{
			Name:         name,
			Source:       dependency.Source,
			ProjectID:    dependency.ProjectID,
			IconURL:      dependency.IconURL,
			VersionID:    dependency.VersionID,
			FileID:       dependency.FileID,
			VersionLabel: dependency.VersionLabel,
			Filename:     dependency.Filename,
		})
	}
	return response, nil
}
