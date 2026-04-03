package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrGenerateSecret(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "naviserver-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	secret1 := LoadOrGenerateSecret(tempDir)
	if secret1 == "" {
		t.Error("Expected generated secret, got empty string")
	}
	if len(secret1) != 64 {
		t.Errorf("Expected 64 char hex string, got length %d", len(secret1))
	}

	secretPath := filepath.Join(tempDir, ".naviserver_secret")
	if _, err := os.Stat(secretPath); os.IsNotExist(err) {
		t.Error("Secret file was not created")
	}

	secret2 := LoadOrGenerateSecret(tempDir)
	if secret1 != secret2 {
		t.Errorf("Expected secret to persist. Got %s, want %s", secret2, secret1)
	}

	os.Setenv("NAVISERVER_SECRET_KEY", "custom-env-secret")
	defer os.Unsetenv("NAVISERVER_SECRET_KEY")

	secret3 := LoadOrGenerateSecret(tempDir)
	if secret3 != "custom-env-secret" {
		t.Errorf("Expected env var secret. Got %s, want custom-env-secret", secret3)
	}
}
