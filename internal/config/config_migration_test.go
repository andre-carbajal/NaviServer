package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigMigratesMissingFieldsWithoutLosingCustomData(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "naviserver-config-migrate")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, defaultConfigName)
	initial := `{
  "servers_path": "/custom/servers",
  "custom_value": "keep-me",
  "api": {
    "port": 9999,
    "extra": true
  }
}`
	if err := os.WriteFile(configPath, []byte(initial), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.ServersPath != "/custom/servers" {
		t.Fatalf("servers_path overwritten, got %q", cfg.ServersPath)
	}
	if cfg.API.Port != 9999 {
		t.Fatalf("api.port overwritten, got %d", cfg.API.Port)
	}
	if cfg.API.Host == "" {
		t.Fatal("expected api.host to be backfilled")
	}

	migratedBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read migrated config: %v", err)
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(migratedBytes, &root); err != nil {
		t.Fatalf("failed to unmarshal migrated config: %v", err)
	}

	var customValue string
	if err := json.Unmarshal(root["custom_value"], &customValue); err != nil {
		t.Fatalf("missing custom_value: %v", err)
	}
	if customValue != "keep-me" {
		t.Fatalf("custom_value changed, got %q", customValue)
	}

	rawAPI, ok := root["api"]
	if !ok {
		t.Fatal("missing api object after migration")
	}

	var apiMap map[string]json.RawMessage
	if err := json.Unmarshal(rawAPI, &apiMap); err != nil {
		t.Fatalf("failed to unmarshal api object: %v", err)
	}

	if _, ok := apiMap["host"]; !ok {
		t.Fatal("expected api.host to be added")
	}
	if _, ok := apiMap["allowed_origins"]; !ok {
		t.Fatal("expected api.allowed_origins to be added")
	}
	if _, ok := apiMap["extra"]; !ok {
		t.Fatal("expected api.extra to be preserved")
	}
}

func TestLoadConfigEnvOverridesDoNotRewriteConfigFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "naviserver-config-env")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, defaultConfigName)
	initial := `{
  "api": {
    "host": "0.0.0.0",
    "port": 23008,
    "allowed_origins": []
  }
}`
	if err := os.WriteFile(configPath, []byte(initial), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	t.Setenv(envServerPort, "24000")

	cfg, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.API.Port != 24000 {
		t.Fatalf("expected env override port 24000, got %d", cfg.API.Port)
	}

	persistedBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read persisted config: %v", err)
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(persistedBytes, &root); err != nil {
		t.Fatalf("failed to unmarshal persisted config: %v", err)
	}

	var apiCfg APIConfig
	if err := json.Unmarshal(root["api"], &apiCfg); err != nil {
		t.Fatalf("failed to unmarshal api config: %v", err)
	}

	if apiCfg.Port != 23008 {
		t.Fatalf("env override should not be persisted, got %d", apiCfg.Port)
	}
}
