package server

import (
	"fmt"
	"naviserver/internal/domain"
	"naviserver/internal/loader"
	"path/filepath"
	"strconv"
	"strings"
)

var validGameModes = map[string]bool{
	"survival":  true,
	"creative":  true,
	"adventure": true,
	"spectator": true,
}

var validDifficulties = map[string]bool{
	"peaceful": true,
	"easy":     true,
	"normal":   true,
	"hard":     true,
}

type ServerSettings struct {
	Name               string `json:"name"`
	RAM                int    `json:"ram"`
	CustomArgs         string `json:"customArgs"`
	Loader             string `json:"loader"`
	Version            string `json:"version"`
	Gamemode           string `json:"gamemode"`
	Difficulty         string `json:"difficulty"`
	MOTD               string `json:"motd"`
	OnlineMode         bool   `json:"onlineMode"`
	SpawnProtection    int    `json:"spawnProtection"`
	PvP                bool   `json:"pvp"`
	AllowFlight        bool   `json:"allowFlight"`
	EnableCommandBlock bool   `json:"enableCommandBlock"`
	Hardcore           bool   `json:"hardcore"`
	MaxPlayers         int    `json:"maxPlayers"`
	ViewDistance       int    `json:"viewDistance"`
	SimulationDistance int    `json:"simulationDistance"`
}

func (m *Manager) serverRootByID(id string) (string, error) {
	srv, err := m.GetServer(id)
	if err != nil {
		return "", err
	}
	if srv == nil {
		return "", fmt.Errorf("server not found")
	}

	folderName := srv.FolderName
	if folderName == "" {
		folderName = id
	}
	return filepath.Join(m.ServersPath, folderName), nil
}

func (m *Manager) GetServerSettings(id string) (*ServerSettings, error) {
	srv, err := m.GetServer(id)
	if err != nil {
		return nil, err
	}
	if srv == nil {
		return nil, fmt.Errorf("server not found")
	}

	root, err := m.serverRootByID(id)
	if err != nil {
		return nil, err
	}

	props, err := parsePropertiesFile(filepath.Join(root, "server.properties"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse server.properties: %w", err)
	}

	settings := &ServerSettings{
		Name:               srv.Name,
		RAM:                srv.RAM,
		CustomArgs:         srv.CustomArgs,
		Loader:             srv.Loader,
		Version:            srv.Version,
		Gamemode:           "survival",
		Difficulty:         "easy",
		MOTD:               "A Minecraft Server",
		OnlineMode:         true,
		SpawnProtection:    16,
		PvP:                true,
		AllowFlight:        false,
		EnableCommandBlock: false,
		Hardcore:           false,
		MaxPlayers:         20,
		ViewDistance:       10,
		SimulationDistance: 10,
	}

	if val, ok := props.Get("gamemode"); ok && val != "" {
		settings.Gamemode = strings.ToLower(val)
	}
	if val, ok := props.Get("difficulty"); ok && val != "" {
		settings.Difficulty = strings.ToLower(val)
	}
	if val, ok := props.Get("motd"); ok {
		settings.MOTD = val
	}
	if val, ok := props.Get("online-mode"); ok {
		settings.OnlineMode = parseBoolOrDefault(val, settings.OnlineMode)
	}
	if val, ok := props.Get("spawn-protection"); ok {
		settings.SpawnProtection = parseIntOrDefault(val, settings.SpawnProtection)
	}
	if val, ok := props.Get("pvp"); ok {
		settings.PvP = parseBoolOrDefault(val, settings.PvP)
	}
	if val, ok := props.Get("allow-flight"); ok {
		settings.AllowFlight = parseBoolOrDefault(val, settings.AllowFlight)
	}
	if val, ok := props.Get("enable-command-block"); ok {
		settings.EnableCommandBlock = parseBoolOrDefault(
			val,
			settings.EnableCommandBlock,
		)
	}
	if val, ok := props.Get("hardcore"); ok {
		settings.Hardcore = parseBoolOrDefault(val, settings.Hardcore)
	}
	if val, ok := props.Get("max-players"); ok {
		settings.MaxPlayers = parseIntOrDefault(val, settings.MaxPlayers)
	}
	if val, ok := props.Get("view-distance"); ok {
		settings.ViewDistance = parseIntOrDefault(val, settings.ViewDistance)
	}
	if val, ok := props.Get("simulation-distance"); ok {
		settings.SimulationDistance = parseIntOrDefault(
			val,
			settings.SimulationDistance,
		)
	}

	return settings, nil
}

func (m *Manager) UpdateServerSettings(id string, next ServerSettings) error {
	srv, err := m.GetServer(id)
	if err != nil {
		return err
	}
	if srv == nil {
		return fmt.Errorf("server not found")
	}
	if srv.Status != "STOPPED" {
		return fmt.Errorf("server must be stopped to update settings")
	}

	if err := validateSettings(next); err != nil {
		return err
	}

	root, err := m.serverRootByID(id)
	if err != nil {
		return err
	}
	path := filepath.Join(root, "server.properties")

	props, err := parsePropertiesFile(path)
	if err != nil {
		return fmt.Errorf("failed to parse server.properties: %w", err)
	}

	props.SetMany(map[string]string{
		"gamemode":             strings.ToLower(next.Gamemode),
		"difficulty":           strings.ToLower(next.Difficulty),
		"motd":                 next.MOTD,
		"online-mode":          formatBool(next.OnlineMode),
		"spawn-protection":     strconv.Itoa(next.SpawnProtection),
		"pvp":                  formatBool(next.PvP),
		"allow-flight":         formatBool(next.AllowFlight),
		"enable-command-block": formatBool(next.EnableCommandBlock),
		"hardcore":             formatBool(next.Hardcore),
		"max-players":          strconv.Itoa(next.MaxPlayers),
		"view-distance":        strconv.Itoa(next.ViewDistance),
		"simulation-distance":  strconv.Itoa(next.SimulationDistance),
	})

	if err := props.Write(path); err != nil {
		return fmt.Errorf("failed to write server.properties: %w", err)
	}

	name := strings.TrimSpace(next.Name)
	customArgs := strings.TrimSpace(next.CustomArgs)
	if err := m.Store.UpdateServer(id, &name, &next.RAM, &customArgs); err != nil {
		return err
	}

	return nil
}

func (m *Manager) GetVersionOptions(id string) ([]string, error) {
	srv, err := m.GetServer(id)
	if err != nil {
		return nil, err
	}
	if srv == nil {
		return nil, fmt.Errorf("server not found")
	}

	versions, err := loader.GetLoaderVersions(srv.Loader, loader.LoaderOptions{
		IncludeSnapshots: false,
		IncludeUnstable:  false,
	})
	if err != nil {
		return nil, err
	}

	return versions, nil
}

func (m *Manager) ValidateServerVersionUpdate(id, targetVersion string) (*domain.Server, string, error) {
	srv, err := m.GetServer(id)
	if err != nil {
		return nil, "", err
	}
	if srv == nil {
		return nil, "", fmt.Errorf("server not found")
	}
	if srv.Status != "STOPPED" {
		return nil, "", fmt.Errorf("server must be stopped to update version")
	}

	version := strings.TrimSpace(targetVersion)
	if version == "" {
		return nil, "", fmt.Errorf("version is required")
	}
	isFuture, comparable := isFutureVersion(version, srv.Version)
	if !comparable {
		return nil, "", fmt.Errorf("unable to compare versions: current=%s target=%s", srv.Version, version)
	}
	if !isFuture {
		return nil, "", fmt.Errorf("target version must be greater than current version")
	}

	versions, err := m.GetVersionOptions(id)
	if err != nil {
		return nil, "", err
	}

	versionFound := false
	for _, candidate := range versions {
		if candidate == version {
			versionFound = true
			break
		}
	}
	if !versionFound {
		return nil, "", fmt.Errorf("version %s is not available for loader %s", version, srv.Loader)
	}

	return srv, version, nil
}

func (m *Manager) ApplyServerVersionUpdate(id, version string) (string, error) {
	srv, err := m.GetServer(id)
	if err != nil {
		return "", err
	}
	if srv == nil {
		return "", fmt.Errorf("server not found")
	}

	root, err := m.serverRootByID(id)
	if err != nil {
		return "", err
	}

	downloader, err := loader.GetLoader(srv.Loader)
	if err != nil {
		return "", err
	}

	loadOptions, err := m.prepareLoaderOptions(srv.Loader, downloader, loader.LoaderOptions{
		MCVersion: version,
	}, version)
	if err != nil {
		return "", fmt.Errorf("version update failed: %w", err)
	}

	resolvedVersion, err := downloader.Load(loadOptions, root, nil)
	if err != nil {
		return "", fmt.Errorf("version update failed: %w", err)
	}

	if strings.TrimSpace(resolvedVersion) == "" {
		resolvedVersion = version
	}
	if err := m.Store.UpdateServerVersion(id, resolvedVersion); err != nil {
		return "", err
	}
	return resolvedVersion, nil
}

func validateSettings(next ServerSettings) error {
	name := strings.TrimSpace(next.Name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if next.RAM < 512 || next.RAM > 262144 {
		return fmt.Errorf("ram must be between 512 and 262144 MB")
	}

	gamemode := strings.ToLower(strings.TrimSpace(next.Gamemode))
	if !validGameModes[gamemode] {
		return fmt.Errorf("invalid gamemode")
	}
	difficulty := strings.ToLower(strings.TrimSpace(next.Difficulty))
	if !validDifficulties[difficulty] {
		return fmt.Errorf("invalid difficulty")
	}
	if next.SpawnProtection < 0 {
		return fmt.Errorf("spawn-protection must be greater than or equal to 0")
	}

	if next.MaxPlayers < 1 || next.MaxPlayers > 1000 {
		return fmt.Errorf("max-players must be between 1 and 1000")
	}
	if next.ViewDistance < 2 || next.ViewDistance > 32 {
		return fmt.Errorf("view-distance must be between 2 and 32")
	}
	if next.SimulationDistance < 2 || next.SimulationDistance > 32 {
		return fmt.Errorf("simulation-distance must be between 2 and 32")
	}
	return nil
}

func formatBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func parseBoolOrDefault(value string, fallback bool) bool {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if normalized == "true" {
		return true
	}
	if normalized == "false" {
		return false
	}
	return fallback
}

func parseIntOrDefault(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func parseVersionParts(value string) ([]int, bool) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	normalized = strings.TrimPrefix(normalized, "v")
	if normalized == "" {
		return nil, false
	}

	parts := strings.Split(normalized, ".")
	parsed := make([]int, 0, len(parts))
	for _, part := range parts {
		digits := leadingDigits(part)
		if digits == "" {
			return nil, false
		}
		num, err := strconv.Atoi(digits)
		if err != nil {
			return nil, false
		}
		parsed = append(parsed, num)
	}
	return parsed, true
}

func leadingDigits(value string) string {
	for i, ch := range value {
		if ch < '0' || ch > '9' {
			return value[:i]
		}
	}
	return value
}

func compareVersions(left, right string) (int, bool) {
	lv, ok := parseVersionParts(left)
	if !ok {
		return 0, false
	}
	rv, ok := parseVersionParts(right)
	if !ok {
		return 0, false
	}

	maxLen := len(lv)
	if len(rv) > maxLen {
		maxLen = len(rv)
	}

	for i := 0; i < maxLen; i++ {
		l := 0
		r := 0
		if i < len(lv) {
			l = lv[i]
		}
		if i < len(rv) {
			r = rv[i]
		}
		if l > r {
			return 1, true
		}
		if l < r {
			return -1, true
		}
	}

	return 0, true
}

func isFutureVersion(candidate, current string) (bool, bool) {
	cmp, ok := compareVersions(candidate, current)
	if !ok {
		return false, false
	}
	return cmp > 0, true
}
