package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultConfigName    = "config.json"
	defaultServersDir    = "servers"
	defaultBackupsDir    = "backups"
	defaultRuntimesDir   = "runtimes"
	defaultDatabaseFile  = "manager.db"
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
)

type APIConfig struct {
	Host           string   `json:"host"`
	Port           int      `json:"port"`
	AllowedOrigins []string `json:"allowed_origins"`
}

type Config struct {
	ServersPath  string    `json:"servers_path"`
	BackupsPath  string    `json:"backups_path"`
	RuntimesPath string    `json:"runtimes_path"`
	DatabasePath string    `json:"database_path"`
	API          APIConfig `json:"api"`
	JWTSecret    string    `json:"-"`
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
	cfg.applyEnvOverrides()

	cfg.JWTSecret = LoadOrGenerateSecret(configDir)

	return &cfg, nil
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

	cfg.JWTSecret = LoadOrGenerateSecret(configDir)
	return &cfg, nil
}

func LoadOrGenerateSecret(configDir string) string {
	if envSecret := os.Getenv("NAVISERVER_SECRET_KEY"); envSecret != "" {
		return envSecret
	}

	secretPath := filepath.Join(configDir, ".naviserver_secret")

	data, err := os.ReadFile(secretPath)
	if err == nil {
		return string(data)
	}

	newSecret := make([]byte, 32)
	if _, err := rand.Read(newSecret); err != nil {
		return fmt.Sprintf("naviserver-secret-%d", time.Now().UnixNano())
	}

	secretStr := hex.EncodeToString(newSecret)

	_ = os.WriteFile(secretPath, []byte(secretStr), 0600)

	return secretStr
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
