package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
)

type Config struct {
	ServersPath  string `json:"servers_path"`
	BackupsPath  string `json:"backups_path"`
	RuntimesPath string `json:"runtimes_path"`
	DatabasePath string `json:"database_path"`
	JWTSecret    string `json:"-"`
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

	var cfg Config
	if err := json.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}

	cfg.JWTSecret = LoadOrGenerateSecret(configDir)

	return &cfg, nil
}

func createDefaultConfig(configPath, configDir string) (*Config, error) {
	cfg := Config{
		ServersPath:  filepath.Join(configDir, defaultServersDir),
		BackupsPath:  filepath.Join(configDir, defaultBackupsDir),
		RuntimesPath: filepath.Join(configDir, defaultRuntimesDir),
		DatabasePath: filepath.Join(configDir, defaultDatabaseFile),
	}

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
	if envSecret := os.Getenv("NAVIGER_SECRET_KEY"); envSecret != "" {
		return envSecret
	}

	secretPath := filepath.Join(configDir, ".naviger_secret")

	data, err := os.ReadFile(secretPath)
	if err == nil {
		return string(data)
	}

	newSecret := make([]byte, 32)
	if _, err := rand.Read(newSecret); err != nil {
		return fmt.Sprintf("naviger-secret-%d", time.Now().UnixNano())
	}

	secretStr := hex.EncodeToString(newSecret)

	_ = os.WriteFile(secretPath, []byte(secretStr), 0600)

	return secretStr
}

func IsDev() bool {
	val := os.Getenv("NAVIGER_DEV")
	return val == "true" || val == "1"
}

func GetPort() int {
	if IsDev() {
		return devPort
	}
	return defaultPort
}
