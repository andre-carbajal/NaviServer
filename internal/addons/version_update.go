package addons

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type AddonUpdateFailure struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Reason string `json:"reason"`
}

type VersionAddonUpdateResult struct {
	Updated  []Addon              `json:"updated"`
	Disabled []Addon              `json:"disabled"`
	Failed   []AddonUpdateFailure `json:"failed"`
}

func (m *Manager) UpdateAddonsForServerVersion(ctx context.Context, serverID string, includeDependencies bool) (*VersionAddonUpdateResult, error) {
	srv, _, _, _, loaders, err := m.resolveServerAddonContext(serverID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "addon management is only supported") {
			return &VersionAddonUpdateResult{Updated: []Addon{}, Disabled: []Addon{}, Failed: []AddonUpdateFailure{}}, nil
		}
		return nil, err
	}
	if srv.Status != "STOPPED" {
		return nil, fmt.Errorf("server must be stopped to modify addons")
	}

	list, err := m.ListAddons(ctx, serverID)
	if err != nil {
		return nil, err
	}

	result := &VersionAddonUpdateResult{
		Updated:  []Addon{},
		Disabled: []Addon{},
		Failed:   []AddonUpdateFailure{},
	}

	for _, addon := range list.Items {
		if addon.Disabled {
			continue
		}

		switch addon.Source {
		case AddonSourceModrinth:
			m.updateModrinthAddonForServerVersion(ctx, serverID, addon, loaders, srv.Version, includeDependencies, result)
		case AddonSourceCurseForge:
			m.updateCurseForgeAddonForServerVersion(ctx, serverID, addon, srv.Version, srv.Loader, includeDependencies, result)
		default:
			m.disableAddonForVersionUpdate(serverID, addon, "addon source is unknown")
			result.Disabled = append(result.Disabled, addon)
		}
	}

	return result, nil
}

func (m *Manager) updateModrinthAddonForServerVersion(
	ctx context.Context,
	serverID string,
	addon Addon,
	loaders []string,
	mcVersion string,
	includeDependencies bool,
	result *VersionAddonUpdateResult,
) {
	versions, err := m.listModrinthVersions(ctx, addon.ProjectID, loaders, mcVersion)
	if err != nil {
		m.disableAddonAfterVersionUpdateFailure(serverID, addon, fmt.Sprintf("failed checking compatible Modrinth versions: %v", err), result)
		return
	}
	if len(versions) == 0 {
		m.disableAddonForVersionUpdate(serverID, addon, "no compatible Modrinth version found")
		result.Disabled = append(result.Disabled, addon)
		return
	}

	latest := versions[0]
	if latest.VersionID == addon.VersionID {
		return
	}

	if err := m.InstallAddon(ctx, serverID, InstallRequest{
		Source:              AddonSourceModrinth,
		ProjectID:           addon.ProjectID,
		VersionID:           latest.VersionID,
		IncludeDependencies: includeDependencies,
	}); err != nil {
		m.disableAddonAfterVersionUpdateFailure(serverID, addon, fmt.Sprintf("failed installing compatible Modrinth version: %v", err), result)
		return
	}

	latestName := latest.PrimaryFilename()
	if latestName != "" && addon.FileName != "" && latestName != addon.FileName {
		_ = m.DeleteAddon(serverID, addon.FileName)
	}
	result.Updated = append(result.Updated, addon)
}

func (m *Manager) updateCurseForgeAddonForServerVersion(
	ctx context.Context,
	serverID string,
	addon Addon,
	mcVersion string,
	loader string,
	includeDependencies bool,
	result *VersionAddonUpdateResult,
) {
	modID, err := strconv.ParseInt(addon.ProjectID, 10, 64)
	if err != nil {
		m.disableAddonAfterVersionUpdateFailure(serverID, addon, "invalid CurseForge project id", result)
		return
	}
	files, err := m.listCurseFiles(ctx, modID, mcVersion, loader)
	if err != nil {
		m.disableAddonAfterVersionUpdateFailure(serverID, addon, fmt.Sprintf("failed checking compatible CurseForge files: %v", err), result)
		return
	}
	latest := chooseLatestCurseFile(files, mcVersion, loader)
	if latest == nil {
		m.disableAddonForVersionUpdate(serverID, addon, "no compatible CurseForge file found")
		result.Disabled = append(result.Disabled, addon)
		return
	}

	currentFileID, _ := strconv.ParseInt(addon.VersionID, 10, 64)
	if latest.ID == currentFileID {
		return
	}

	if err := m.InstallAddon(ctx, serverID, InstallRequest{
		Source:              AddonSourceCurseForge,
		ProjectID:           addon.ProjectID,
		FileID:              latest.ID,
		IncludeDependencies: includeDependencies,
	}); err != nil {
		m.disableAddonAfterVersionUpdateFailure(serverID, addon, fmt.Sprintf("failed installing compatible CurseForge file: %v", err), result)
		return
	}

	if latest.FileName != "" && addon.FileName != "" && latest.FileName != addon.FileName {
		_ = m.DeleteAddon(serverID, addon.FileName)
	}
	result.Updated = append(result.Updated, addon)
}

func (m *Manager) disableAddonAfterVersionUpdateFailure(serverID string, addon Addon, reason string, result *VersionAddonUpdateResult) {
	m.disableAddonForVersionUpdate(serverID, addon, reason)
	result.Disabled = append(result.Disabled, addon)
	result.Failed = append(result.Failed, AddonUpdateFailure{
		ID:     addon.ID,
		Name:   addon.ProjectName,
		Reason: reason,
	})
}

func (m *Manager) disableAddonForVersionUpdate(serverID string, addon Addon, reason string) {
	if err := m.SetAddonDisabled(serverID, addon.ID, true); err != nil {
		return
	}
}
