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

func newSettingsManagerForTest(t *testing.T) (*Manager, string) {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	serversPath := filepath.Join(tempDir, "servers")

	store, err := storage.NewGormStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	manager := NewManager(serversPath, store)
	srv := &domain.Server{
		ID:         "srv-1",
		Name:       "Test Server",
		FolderName: "test-server",
		Version:    "1.21.1",
		Loader:     "vanilla",
		Port:       25565,
		RAM:        2048,
		Status:     "STOPPED",
		CreatedAt:  time.Now(),
	}

	if err := store.SaveServer(srv); err != nil {
		t.Fatalf("failed to save server: %v", err)
	}

	serverRoot := filepath.Join(serversPath, srv.FolderName)
	return manager, serverRoot
}

func TestGetServerSettingsOnlineModeDefaultsTrueWhenMissing(t *testing.T) {
	manager, serverRoot := newSettingsManagerForTest(t)
	propertiesPath := filepath.Join(serverRoot, "server.properties")
	content := strings.Join([]string{
		"motd=A Minecraft Server",
		"pvp=true",
	}, "\n")

	if err := os.MkdirAll(serverRoot, 0o755); err != nil {
		t.Fatalf("failed to create server root: %v", err)
	}
	if err := os.WriteFile(propertiesPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write properties file: %v", err)
	}

	settings, err := manager.GetServerSettings("srv-1")
	if err != nil {
		t.Fatalf("GetServerSettings failed: %v", err)
	}

	if !settings.OnlineMode {
		t.Fatalf("expected onlineMode default true when key is missing")
	}
	if settings.SpawnProtection != 16 {
		t.Fatalf("expected spawnProtection default 16 when key is missing, got %d", settings.SpawnProtection)
	}
}

func TestUpdateServerSettingsWritesOnlineMode(t *testing.T) {
	manager, serverRoot := newSettingsManagerForTest(t)
	propertiesPath := filepath.Join(serverRoot, "server.properties")
	if err := os.MkdirAll(serverRoot, 0o755); err != nil {
		t.Fatalf("failed to create server root: %v", err)
	}
	if err := os.WriteFile(propertiesPath, []byte("motd=Test\n"), 0o644); err != nil {
		t.Fatalf("failed to seed properties file: %v", err)
	}

	err := manager.UpdateServerSettings("srv-1", ServerSettings{
		Name:               "Test Server",
		RAM:                2048,
		CustomArgs:         "",
		Loader:             "vanilla",
		Version:            "1.21.1",
		Gamemode:           "survival",
		Difficulty:         "normal",
		MOTD:               "Test",
		OnlineMode:         false,
		SpawnProtection:    8,
		PvP:                true,
		AllowFlight:        false,
		EnableCommandBlock: false,
		Hardcore:           false,
		MaxPlayers:         20,
		ViewDistance:       10,
		SimulationDistance: 10,
	})
	if err != nil {
		t.Fatalf("UpdateServerSettings failed: %v", err)
	}

	updated, err := os.ReadFile(propertiesPath)
	if err != nil {
		t.Fatalf("failed to read properties file: %v", err)
	}

	if !strings.Contains(string(updated), "online-mode=false") {
		t.Fatalf("expected online-mode=false in properties file, got:\n%s", string(updated))
	}
	if !strings.Contains(string(updated), "spawn-protection=8") {
		t.Fatalf("expected spawn-protection=8 in properties file, got:\n%s", string(updated))
	}
}
