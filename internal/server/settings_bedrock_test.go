package server

import (
	"naviserver/internal/domain"
	"naviserver/internal/storage"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newBedrockSettingsManager(t *testing.T) (*Manager, string) {
	t.Helper()
	tempDir := t.TempDir()
	store, err := storage.NewGormStore(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	manager := NewManager(filepath.Join(tempDir, "servers"), store)
	server := &domain.Server{
		ID: "bedrock-1", Name: "Bedrock", FolderName: "bedrock", Version: "1.26.33",
		Loader: "bedrock", Port: 19132, RAM: 4096, Status: "STOPPED", CreatedAt: time.Now(),
	}
	if err := store.SaveServer(server); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(manager.ServersPath, server.FolderName)
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	return manager, root
}

func TestBedrockSettingsReadAndWriteNativeProperties(t *testing.T) {
	manager, root := newBedrockSettingsManager(t)
	propertiesPath := filepath.Join(root, "server.properties")
	initial := strings.Join([]string{
		"server-name=Original Bedrock",
		"gamemode=creative",
		"difficulty=hard",
		"online-mode=true",
		"max-players=10",
		"view-distance=32",
		"tick-distance=4",
		"level-name=Bedrock level",
		"default-player-permission-level=member",
	}, "\n")
	if err := os.WriteFile(propertiesPath, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	settings, err := manager.GetServerSettings("bedrock-1")
	if err != nil {
		t.Fatal(err)
	}
	if settings.MOTD != "Original Bedrock" || settings.TickDistance != 4 || settings.LevelName != "Bedrock level" {
		t.Fatalf("unexpected Bedrock settings: %+v", settings)
	}

	settings.MOTD = "Updated Bedrock"
	settings.Gamemode = "survival"
	settings.TickDistance = 8
	settings.ViewDistance = 48
	settings.ForceGamemode = true
	settings.AllowCheats = true
	settings.AllowList = true
	settings.LevelName = "My World"
	settings.DefaultPlayerPermissionLevel = "operator"
	settings.TexturepackRequired = true
	settings.PlayerIdleTimeout = 15
	settings.MaxThreads = 0
	if err := manager.UpdateServerSettings("bedrock-1", *settings); err != nil {
		t.Fatal(err)
	}

	updated, err := os.ReadFile(propertiesPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(updated)
	for _, expected := range []string{
		"server-name=Updated Bedrock", "tick-distance=8", "view-distance=48",
		"force-gamemode=true", "allow-cheats=true", "allow-list=true",
		"level-name=My World", "default-player-permission-level=operator",
		"texturepack-required=true", "player-idle-timeout=15", "max-threads=0",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("properties missing %q:\n%s", expected, content)
		}
	}
	for _, javaOnly := range []string{"motd=", "spawn-protection=", "simulation-distance=", "hardcore="} {
		if strings.Contains(content, javaOnly) {
			t.Fatalf("Bedrock properties unexpectedly contain %q:\n%s", javaOnly, content)
		}
	}
}

func TestValidateBedrockSettingsRejectsUnsupportedValues(t *testing.T) {
	settings := ServerSettings{
		Name: "Bedrock", RAM: 4096, Gamemode: "spectator", Difficulty: "normal",
		MOTD: "Bedrock", MaxPlayers: 10, ViewDistance: 32, TickDistance: 4,
		LevelName: "Bedrock level", DefaultPlayerPermissionLevel: "member",
	}
	if err := validateSettingsForLoader(settings, "bedrock"); err == nil {
		t.Fatal("expected spectator validation error")
	}

	settings.Gamemode = "survival"
	settings.TickDistance = 13
	if err := validateSettingsForLoader(settings, "bedrock"); err == nil {
		t.Fatal("expected tick-distance validation error")
	}
}
