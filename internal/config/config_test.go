package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadConfigCreatesDefaultAPIConfig(t *testing.T) {
	tempDir := t.TempDir()

	cfg, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.API.Host != defaultAPIHost {
		t.Fatalf("unexpected api.host: got %q want %q", cfg.API.Host, defaultAPIHost)
	}
	if cfg.API.Port != GetPort() {
		t.Fatalf("unexpected api.port: got %d want %d", cfg.API.Port, GetPort())
	}
	if len(cfg.API.AllowedOrigins) != 0 {
		t.Fatalf("unexpected allowed origins: %#v", cfg.API.AllowedOrigins)
	}

	configPath := filepath.Join(tempDir, defaultConfigName)
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read generated config: %v", err)
	}

	var persisted Config
	if err := json.Unmarshal(content, &persisted); err != nil {
		t.Fatalf("failed to decode generated config: %v", err)
	}
	if persisted.API.Port != cfg.API.Port {
		t.Fatalf("persisted api.port mismatch: got %d want %d", persisted.API.Port, cfg.API.Port)
	}
}

func TestLoadConfigAppliesEnvOverrides(t *testing.T) {
	t.Setenv(envServerHost, "127.0.0.1")
	t.Setenv(envServerPort, "24000")
	t.Setenv(envAllowedOrigins, "http://a.local, http://b.local, http://a.local")

	tempDir := t.TempDir()
	seed := Config{
		ServersPath:  filepath.Join(tempDir, "servers"),
		BackupsPath:  filepath.Join(tempDir, "backups"),
		RuntimesPath: filepath.Join(tempDir, "runtimes"),
		DatabasePath: filepath.Join(tempDir, "manager.db"),
		API: APIConfig{
			Host:           "0.0.0.0",
			Port:           23008,
			AllowedOrigins: []string{"http://from-config.local"},
		},
	}

	data, err := json.Marshal(seed)
	if err != nil {
		t.Fatalf("marshal seed config failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, defaultConfigName), data, 0644); err != nil {
		t.Fatalf("write seed config failed: %v", err)
	}

	cfg, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.API.Host != "127.0.0.1" {
		t.Fatalf("unexpected api.host: got %q want %q", cfg.API.Host, "127.0.0.1")
	}
	if cfg.API.Port != 24000 {
		t.Fatalf("unexpected api.port: got %d want %d", cfg.API.Port, 24000)
	}
	wantOrigins := []string{"http://a.local", "http://b.local"}
	if !reflect.DeepEqual(cfg.API.AllowedOrigins, wantOrigins) {
		t.Fatalf("unexpected origins: got %#v want %#v", cfg.API.AllowedOrigins, wantOrigins)
	}
}
