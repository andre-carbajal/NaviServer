package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultConfigName    = "config.json"
	defaultServersDir    = "servers"
	defaultBackupsDir    = "backups"
	defaultRuntimesDir   = "runtimes"
	defaultDatabaseFile  = "manager.db"
	secretFileName       = ".naviserver_secret"
	cliTokenFileName     = ".naviserver_cli_token"
	defaultPort          = 23008
	devPort              = 23009
	DefaultLogBufferSize = 1000
	defaultAPIHost       = "0.0.0.0"
)

const (
	envServerHost      = "NAVISERVER_HOST"
	envServerPort      = "NAVISERVER_PORT"
	envAllowedOrigins  = "NAVISERVER_ALLOWED_ORIGINS"
	envAllowedOrigins2 = "NAVISERVER_CORS_ALLOWED_ORIGINS"
	envCurseForgeKey   = "CURSEFORGE_API_KEY"
	envCLIToken        = "NAVISERVER_CLI_TOKEN"
)

type APIConfig struct {
	Host           string   `json:"host"`
	Port           int      `json:"port"`
	AllowedOrigins []string `json:"allowed_origins"`
}

type Config struct {
	ServersPath      string    `json:"servers_path"`
	BackupsPath      string    `json:"backups_path"`
	RuntimesPath     string    `json:"runtimes_path"`
	DatabasePath     string    `json:"database_path"`
	API              APIConfig `json:"api"`
	CurseForgeAPIKey string    `json:"-"`
	JWTSecret        string    `json:"-"`
	CLIToken         string    `json:"-"`
}

func LoadConfig(configDir string) (*Config, error) {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, defaultConfigName)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return createDefaultConfig(configPath, configDir)
	}

	file, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	cfg := defaultConfig(configDir)
	if err := json.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}

	cfg.applyDefaults(configDir)
	if err := migrateConfigFileIfNeeded(configPath, file, cfg); err != nil {
		return nil, err
	}
	cfg.applyEnvOverrides()

	if err := cfg.loadRuntimeSecrets(configDir); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func migrateConfigFileIfNeeded(configPath string, file []byte, cfg Config) error {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(file, &root); err != nil {
		return err
	}
	if root == nil {
		root = make(map[string]json.RawMessage)
	}

	changed := false
	setIfMissing := func(key string, value any) error {
		if _, ok := root[key]; ok {
			return nil
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		root[key] = encoded
		changed = true
		return nil
	}

	if err := setIfMissing("servers_path", cfg.ServersPath); err != nil {
		return err
	}
	if err := setIfMissing("backups_path", cfg.BackupsPath); err != nil {
		return err
	}
	if err := setIfMissing("runtimes_path", cfg.RuntimesPath); err != nil {
		return err
	}
	if err := setIfMissing("database_path", cfg.DatabasePath); err != nil {
		return err
	}

	if rawAPI, ok := root["api"]; !ok || len(rawAPI) == 0 || string(rawAPI) == "null" {
		encoded, err := json.Marshal(cfg.API)
		if err != nil {
			return err
		}
		root["api"] = encoded
		changed = true
	} else {
		var apiMap map[string]json.RawMessage
		if err := json.Unmarshal(rawAPI, &apiMap); err != nil {
			encoded, marshalErr := json.Marshal(cfg.API)
			if marshalErr != nil {
				return marshalErr
			}
			root["api"] = encoded
			changed = true
		} else {
			if apiMap == nil {
				apiMap = make(map[string]json.RawMessage)
			}

			apiChanged := false
			setAPIIfMissing := func(key string, value any) error {
				if _, ok := apiMap[key]; ok {
					return nil
				}
				encoded, err := json.Marshal(value)
				if err != nil {
					return err
				}
				apiMap[key] = encoded
				apiChanged = true
				return nil
			}

			if err := setAPIIfMissing("host", cfg.API.Host); err != nil {
				return err
			}
			if err := setAPIIfMissing("port", cfg.API.Port); err != nil {
				return err
			}
			if err := setAPIIfMissing("allowed_origins", cfg.API.AllowedOrigins); err != nil {
				return err
			}

			if apiChanged {
				encoded, err := json.Marshal(apiMap)
				if err != nil {
					return err
				}
				root["api"] = encoded
				changed = true
			}
		}
	}

	if !changed {
		return nil
	}

	encoded, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, encoded, 0644)
}

func createDefaultConfig(configPath, configDir string) (*Config, error) {
	cfg := defaultConfig(configDir)
	cfg.applyEnvOverrides()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return nil, err
	}

	if err := cfg.loadRuntimeSecrets(configDir); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (cfg *Config) loadRuntimeSecrets(configDir string) error {
	jwtSecret, err := LoadOrGenerateSecret(configDir)
	if err != nil {
		return err
	}

	cliToken, err := LoadOrGenerateCLIToken(configDir)
	if err != nil {
		return err
	}

	cfg.JWTSecret = jwtSecret
	cfg.CLIToken = cliToken
	return nil
}

func LoadOrGenerateSecret(configDir string) (string, error) {
	return loadOrGenerateSecretFile(configDir, secretFileName, "NAVISERVER_SECRET_KEY")
}

func LoadOrGenerateCLIToken(configDir string) (string, error) {
	return loadOrGenerateSecretFile(configDir, cliTokenFileName, envCLIToken)
}

func loadOrGenerateSecretFile(configDir, fileName, envName string) (string, error) {
	if envSecret := strings.TrimSpace(os.Getenv(envName)); envSecret != "" {
		return envSecret, nil
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	secretPath := filepath.Join(configDir, fileName)

	data, err := os.ReadFile(secretPath)
	if err == nil {
		return strings.TrimSpace(string(data)), nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read %s: %w", fileName, err)
	}

	secretStr, err := generateRandomHex(rand.Reader, 32)
	if err != nil {
		return "", fmt.Errorf("failed to generate secure %s: %w", fileName, err)
	}

	if err := os.WriteFile(secretPath, []byte(secretStr), 0600); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", fileName, err)
	}

	return secretStr, nil
}

func generateRandomHex(reader io.Reader, byteCount int) (string, error) {
	secret := make([]byte, byteCount)
	if _, err := io.ReadFull(reader, secret); err != nil {
		return "", err
	}
	return hex.EncodeToString(secret), nil
}

func ResolveConfigDir() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	appName := "naviserver"
	if IsDev() {
		appName = "naviserver-dev"
	}

	return filepath.Join(userConfigDir, appName), nil
}

func IsDev() bool {
	val := os.Getenv("NAVISERVER_DEV")
	return val == "true" || val == "1"
}

func GetPort() int {
	if IsDev() {
		return devPort
	}
	return defaultPort
}

func defaultConfig(configDir string) Config {
	return Config{
		ServersPath:  filepath.Join(configDir, defaultServersDir),
		BackupsPath:  filepath.Join(configDir, defaultBackupsDir),
		RuntimesPath: filepath.Join(configDir, defaultRuntimesDir),
		DatabasePath: filepath.Join(configDir, defaultDatabaseFile),
		API: APIConfig{
			Host:           defaultAPIHost,
			Port:           GetPort(),
			AllowedOrigins: []string{},
		},
	}
}

func (cfg *Config) applyDefaults(configDir string) {
	defaults := defaultConfig(configDir)

	if strings.TrimSpace(cfg.ServersPath) == "" {
		cfg.ServersPath = defaults.ServersPath
	}
	if strings.TrimSpace(cfg.BackupsPath) == "" {
		cfg.BackupsPath = defaults.BackupsPath
	}
	if strings.TrimSpace(cfg.RuntimesPath) == "" {
		cfg.RuntimesPath = defaults.RuntimesPath
	}
	if strings.TrimSpace(cfg.DatabasePath) == "" {
		cfg.DatabasePath = defaults.DatabasePath
	}
	if strings.TrimSpace(cfg.API.Host) == "" {
		cfg.API.Host = defaults.API.Host
	}
	if cfg.API.Port <= 0 {
		cfg.API.Port = defaults.API.Port
	}
	cfg.API.AllowedOrigins = normalizeOrigins(cfg.API.AllowedOrigins)
}

func (cfg *Config) applyEnvOverrides() {
	if hostRaw := strings.TrimSpace(os.Getenv(envServerHost)); hostRaw != "" {
		cfg.API.Host = hostRaw
	}

	if portRaw := strings.TrimSpace(os.Getenv(envServerPort)); portRaw != "" {
		if port, err := strconv.Atoi(portRaw); err == nil && port > 0 {
			cfg.API.Port = port
		}
	}

	if originsRaw := strings.TrimSpace(os.Getenv(envAllowedOrigins)); originsRaw != "" {
		cfg.API.AllowedOrigins = splitOrigins(originsRaw)
		return
	}

	if originsRaw := strings.TrimSpace(os.Getenv(envAllowedOrigins2)); originsRaw != "" {
		cfg.API.AllowedOrigins = splitOrigins(originsRaw)
	}

	if key := strings.TrimSpace(os.Getenv(envCurseForgeKey)); key != "" {
		cfg.CurseForgeAPIKey = key
	}
}

func splitOrigins(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}

	return normalizeOrigins(strings.Split(raw, ","))
}

func normalizeOrigins(origins []string) []string {
	if len(origins) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(origins))
	normalized := make([]string, 0, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	return normalized
}
