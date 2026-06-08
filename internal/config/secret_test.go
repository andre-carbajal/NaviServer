package config

import (
	"errors"
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

	secret1, err := LoadOrGenerateSecret(tempDir)
	if err != nil {
		t.Fatalf("LoadOrGenerateSecret failed: %v", err)
	}
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

	secret2, err := LoadOrGenerateSecret(tempDir)
	if err != nil {
		t.Fatalf("LoadOrGenerateSecret failed: %v", err)
	}
	if secret1 != secret2 {
		t.Errorf("Expected secret to persist. Got %s, want %s", secret2, secret1)
	}

	t.Setenv("NAVISERVER_SECRET_KEY", "custom-env-secret")

	secret3, err := LoadOrGenerateSecret(tempDir)
	if err != nil {
		t.Fatalf("LoadOrGenerateSecret failed: %v", err)
	}
	if secret3 != "custom-env-secret" {
		t.Errorf("Expected env var secret. Got %s, want custom-env-secret", secret3)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("random source unavailable")
}

func TestGenerateRandomHexReturnsErrorOnRandomFailure(t *testing.T) {
	secret, err := generateRandomHex(failingReader{}, 32)
	if err == nil {
		t.Fatal("expected random generation error")
	}
	if secret != "" {
		t.Fatalf("expected empty secret on error, got %q", secret)
	}
}

func TestLoadOrGenerateCLIToken(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "naviserver-cli-token-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	token1, err := LoadOrGenerateCLIToken(tempDir)
	if err != nil {
		t.Fatalf("LoadOrGenerateCLIToken failed: %v", err)
	}
	if len(token1) != 64 {
		t.Errorf("Expected 64 char hex token, got length %d", len(token1))
	}

	tokenPath := filepath.Join(tempDir, ".naviserver_cli_token")
	if info, err := os.Stat(tokenPath); err != nil {
		t.Fatalf("CLI token file was not created: %v", err)
	} else if info.Mode().Perm() != 0600 {
		t.Fatalf("expected CLI token file mode 0600, got %v", info.Mode().Perm())
	}

	token2, err := LoadOrGenerateCLIToken(tempDir)
	if err != nil {
		t.Fatalf("LoadOrGenerateCLIToken failed: %v", err)
	}
	if token1 != token2 {
		t.Errorf("Expected CLI token to persist. Got %s, want %s", token2, token1)
	}

	t.Setenv("NAVISERVER_CLI_TOKEN", "custom-cli-token")
	token3, err := LoadOrGenerateCLIToken(tempDir)
	if err != nil {
		t.Fatalf("LoadOrGenerateCLIToken failed: %v", err)
	}
	if token3 != "custom-cli-token" {
		t.Errorf("Expected env var CLI token. Got %s, want custom-cli-token", token3)
	}
}
