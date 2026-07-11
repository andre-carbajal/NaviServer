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
	Name                         string `json:"name"`
	RAM                          int    `json:"ram"`
	CustomArgs                   string `json:"customArgs"`
	Loader                       string `json:"loader"`
	Version                      string `json:"version"`
	Gamemode                     string `json:"gamemode"`
	Difficulty                   string `json:"difficulty"`
	MOTD                         string `json:"motd"`
	OnlineMode                   bool   `json:"onlineMode"`
	SpawnProtection              int    `json:"spawnProtection"`
	PvP                          bool   `json:"pvp"`
	AllowFlight                  bool   `json:"allowFlight"`
	EnableCommandBlock           bool   `json:"enableCommandBlock"`
	Hardcore                     bool   `json:"hardcore"`
	MaxPlayers                   int    `json:"maxPlayers"`
	ViewDistance                 int    `json:"viewDistance"`
	SimulationDistance           int    `json:"simulationDistance"`
	TickDistance                 int    `json:"tickDistance"`
	ForceGamemode                bool   `json:"forceGamemode"`
	AllowCheats                  bool   `json:"allowCheats"`
	AllowList                    bool   `json:"allowList"`
	LevelName                    string `json:"levelName"`
	DefaultPlayerPermissionLevel string `json:"defaultPlayerPermissionLevel"`
	TexturepackRequired          bool   `json:"texturepackRequired"`
	PlayerIdleTimeout            int    `json:"playerIdleTimeout"`
	MaxThreads                   int    `json:"maxThreads"`
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
		Name:                         srv.Name,
		RAM:                          srv.RAM,
		CustomArgs:                   srv.CustomArgs,
		Loader:                       srv.Loader,
		Version:                      srv.Version,
		Gamemode:                     "survival",
		Difficulty:                   "easy",
		MOTD:                         "A Minecraft Server",
		OnlineMode:                   true,
		SpawnProtection:              16,
		PvP:                          true,
		AllowFlight:                  false,
		EnableCommandBlock:           false,
		Hardcore:                     false,
		MaxPlayers:                   20,
		ViewDistance:                 10,
		SimulationDistance:           10,
		TickDistance:                 4,
		LevelName:                    "Bedrock level",
		DefaultPlayerPermissionLevel: "member",
		PlayerIdleTimeout:            30,
		MaxThreads:                   8,
	}
	if srv.Loader == "bedrock" {
		settings.MOTD = "Dedicated Server"
		settings.MaxPlayers = 10
		settings.ViewDistance = 32
	}

	if val, ok := props.Get("gamemode"); ok && val != "" {
		settings.Gamemode = strings.ToLower(val)
	}
	if val, ok := props.Get("difficulty"); ok && val != "" {
		settings.Difficulty = strings.ToLower(val)
	}
	messageKey := "motd"
	if srv.Loader == "bedrock" {
		messageKey = "server-name"
	}
	if val, ok := props.Get(messageKey); ok {
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
	if srv.Loader == "bedrock" {
		if val, ok := props.Get("tick-distance"); ok {
			settings.TickDistance = parseIntOrDefault(val, settings.TickDistance)
		}
		if val, ok := props.Get("force-gamemode"); ok {
			settings.ForceGamemode = parseBoolOrDefault(val, settings.ForceGamemode)
		}
		if val, ok := props.Get("allow-cheats"); ok {
			settings.AllowCheats = parseBoolOrDefault(val, settings.AllowCheats)
		}
		if val, ok := props.Get("allow-list"); ok {
			settings.AllowList = parseBoolOrDefault(val, settings.AllowList)
		}
		if val, ok := props.Get("level-name"); ok && strings.TrimSpace(val) != "" {
			settings.LevelName = val
		}
		if val, ok := props.Get("default-player-permission-level"); ok && strings.TrimSpace(val) != "" {
			settings.DefaultPlayerPermissionLevel = strings.ToLower(val)
		}
		if val, ok := props.Get("texturepack-required"); ok {
			settings.TexturepackRequired = parseBoolOrDefault(val, settings.TexturepackRequired)
		}
		if val, ok := props.Get("player-idle-timeout"); ok {
			settings.PlayerIdleTimeout = parseIntOrDefault(val, settings.PlayerIdleTimeout)
		}
		if val, ok := props.Get("max-threads"); ok {
			settings.MaxThreads = parseIntOrDefault(val, settings.MaxThreads)
		}
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

	if err := validateSettingsForLoader(next, srv.Loader); err != nil {
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

	values := map[string]string{
		"gamemode":      strings.ToLower(next.Gamemode),
		"difficulty":    strings.ToLower(next.Difficulty),
		"online-mode":   formatBool(next.OnlineMode),
		"max-players":   strconv.Itoa(next.MaxPlayers),
		"view-distance": strconv.Itoa(next.ViewDistance),
	}
	if srv.Loader == "bedrock" {
		values["server-name"] = next.MOTD
		values["force-gamemode"] = formatBool(next.ForceGamemode)
		values["allow-cheats"] = formatBool(next.AllowCheats)
		values["allow-list"] = formatBool(next.AllowList)
		values["level-name"] = strings.TrimSpace(next.LevelName)
		values["default-player-permission-level"] = strings.ToLower(strings.TrimSpace(next.DefaultPlayerPermissionLevel))
		values["texturepack-required"] = formatBool(next.TexturepackRequired)
		values["player-idle-timeout"] = strconv.Itoa(next.PlayerIdleTimeout)
		values["max-threads"] = strconv.Itoa(next.MaxThreads)
		values["tick-distance"] = strconv.Itoa(next.TickDistance)
	} else {
		values["motd"] = next.MOTD
		values["spawn-protection"] = strconv.Itoa(next.SpawnProtection)
		values["pvp"] = formatBool(next.PvP)
		values["allow-flight"] = formatBool(next.AllowFlight)
		values["enable-command-block"] = formatBool(next.EnableCommandBlock)
		values["hardcore"] = formatBool(next.Hardcore)
		values["simulation-distance"] = strconv.Itoa(next.SimulationDistance)
	}
	props.SetMany(values)

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
		IncludeSnapshots: srv.Loader == "bedrock" && strings.Contains(strings.ToLower(srv.Version), "-preview."),
		IncludeUnstable:  false,
	})
	if err != nil {
		return nil, err
	}

	return versions, nil
}

func (m *Manager) ValidateServerVersionUpdate(id string, targetVersion string) (*domain.Server, string, error) {
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

func (m *Manager) ApplyServerVersionUpdate(id string, version string) (string, error) {
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

	resolvedVersion, err := downloader.Load(loader.LoaderOptions{
		MCVersion: version,
	}, root, nil)
	if err != nil {
		return "", fmt.Errorf("version update failed: %w", err)
	}
	if err := UpdateServerPropertiesForLoader(root, srv.Port, srv.Loader); err != nil {
		return "", fmt.Errorf("version update configuration failed: %w", err)
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
	return validateSettingsForLoader(next, "vanilla")
}

func validateSettingsForLoader(next ServerSettings, loaderType string) error {
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
	if loaderType == "bedrock" && gamemode == "spectator" {
		return fmt.Errorf("spectator gamemode is not supported by Bedrock Dedicated Server")
	}
	difficulty := strings.ToLower(strings.TrimSpace(next.Difficulty))
	if !validDifficulties[difficulty] {
		return fmt.Errorf("invalid difficulty")
	}
	if loaderType != "bedrock" && next.SpawnProtection < 0 {
		return fmt.Errorf("spawn-protection must be greater than or equal to 0")
	}

	if next.MaxPlayers < 1 || next.MaxPlayers > 1000 {
		return fmt.Errorf("max-players must be between 1 and 1000")
	}
	if loaderType == "bedrock" {
		if next.ViewDistance < 5 {
			return fmt.Errorf("view-distance must be at least 5 for Bedrock")
		}
		if next.TickDistance < 4 || next.TickDistance > 12 {
			return fmt.Errorf("tick-distance must be between 4 and 12 for Bedrock")
		}
		if strings.Contains(next.MOTD, ";") {
			return fmt.Errorf("Bedrock server name cannot contain semicolons")
		}
		levelName := strings.TrimSpace(next.LevelName)
		if levelName == "" || levelName == "." || levelName == ".." || filepath.Base(levelName) != levelName || strings.ContainsAny(levelName, "\\/:*?\"<>|") {
			return fmt.Errorf("invalid Bedrock level name")
		}
		permission := strings.ToLower(strings.TrimSpace(next.DefaultPlayerPermissionLevel))
		if permission != "visitor" && permission != "member" && permission != "operator" {
			return fmt.Errorf("invalid default Bedrock player permission level")
		}
		if next.PlayerIdleTimeout < 0 {
			return fmt.Errorf("player-idle-timeout must be greater than or equal to 0")
		}
		if next.MaxThreads < 0 {
			return fmt.Errorf("max-threads must be greater than or equal to 0")
		}
	} else {
		if next.ViewDistance < 2 || next.ViewDistance > 32 {
			return fmt.Errorf("view-distance must be between 2 and 32")
		}
		if next.SimulationDistance < 2 || next.SimulationDistance > 32 {
			return fmt.Errorf("simulation-distance must be between 2 and 32")
		}
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

type parsedMinecraftVersion struct {
	parts        []int
	isPreview    bool
	previewBuild int
}

func parseMinecraftVersion(value string) (parsedMinecraftVersion, bool) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	normalized = strings.TrimPrefix(normalized, "v")
	if normalized == "" {
		return parsedMinecraftVersion{}, false
	}

	result := parsedMinecraftVersion{}
	if previewIndex := strings.Index(normalized, "-preview."); previewIndex >= 0 {
		build := strings.TrimSpace(normalized[previewIndex+len("-preview."):])
		parsedBuild, err := strconv.Atoi(build)
		if err != nil {
			return parsedMinecraftVersion{}, false
		}
		result.isPreview = true
		result.previewBuild = parsedBuild
		normalized = normalized[:previewIndex]
	}

	parts := strings.Split(normalized, ".")
	parsed := make([]int, 0, len(parts))
	for _, part := range parts {
		digits := leadingDigits(part)
		if digits == "" {
			return parsedMinecraftVersion{}, false
		}
		num, err := strconv.Atoi(digits)
		if err != nil {
			return parsedMinecraftVersion{}, false
		}
		parsed = append(parsed, num)
	}
	result.parts = parsed
	return result, true
}

func leadingDigits(value string) string {
	for i, ch := range value {
		if ch < '0' || ch > '9' {
			return value[:i]
		}
	}
	return value
}

func compareVersions(left string, right string) (int, bool) {
	lv, ok := parseMinecraftVersion(left)
	if !ok {
		return 0, false
	}
	rv, ok := parseMinecraftVersion(right)
	if !ok {
		return 0, false
	}

	maxLen := len(lv.parts)
	if len(rv.parts) > maxLen {
		maxLen = len(rv.parts)
	}

	for i := 0; i < maxLen; i++ {
		l := 0
		r := 0
		if i < len(lv.parts) {
			l = lv.parts[i]
		}
		if i < len(rv.parts) {
			r = rv.parts[i]
		}
		if l > r {
			return 1, true
		}
		if l < r {
			return -1, true
		}
	}
	if lv.isPreview != rv.isPreview {
		if lv.isPreview {
			return -1, true
		}
		return 1, true
	}
	if lv.isPreview {
		if lv.previewBuild > rv.previewBuild {
			return 1, true
		}
		if lv.previewBuild < rv.previewBuild {
			return -1, true
		}
	}

	return 0, true
}

func isFutureVersion(candidate string, current string) (bool, bool) {
	cmp, ok := compareVersions(candidate, current)
	if !ok {
		return false, false
	}
	return cmp > 0, true
}
