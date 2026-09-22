package server

import (
	"naviserver/internal/domain"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUpdateServerMovesFolderAndPersistsFolderName(t *testing.T) {
	manager, oldRoot := newSettingsManagerForTest(t)
	if err := os.MkdirAll(oldRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(oldRoot, "world.dat")
	if err := os.WriteFile(marker, []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}

	name := "Renamed Server"
	if err := manager.UpdateServer("srv-1", &name, nil, nil, nil); err != nil {
		t.Fatalf("UpdateServer failed: %v", err)
	}

	newRoot := filepath.Join(manager.ServersPath, "Renamed_Server")
	if _, err := os.Stat(oldRoot); !os.IsNotExist(err) {
		t.Fatalf("expected old server folder to be gone, got err %v", err)
	}
	if contents, err := os.ReadFile(filepath.Join(newRoot, "world.dat")); err != nil || string(contents) != "world" {
		t.Fatalf("expected server files to move intact, contents=%q err=%v", contents, err)
	}

	updated, err := manager.GetServer("srv-1")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != name || updated.FolderName != "Renamed_Server" {
		t.Fatalf("expected name and folder to be updated, got name=%q folder=%q", updated.Name, updated.FolderName)
	}
}

func TestUpdateServerRenameRejectsInvalidNameCollisionAndRunningServer(t *testing.T) {
	tests := []struct {
		name       string
		newName    string
		status     string
		targetName string
		addTarget  bool
		wantError  string
	}{
		{name: "invalid characters", newName: "Bad/Name", wantError: "invalid"},
		{name: "existing folder", newName: "Taken Server", targetName: "Taken_Server", wantError: "already exists"},
		{name: "case insensitive folder collision", newName: "Taken Server", targetName: "taken_server", wantError: "already exists"},
		{name: "folder used by another server", newName: "Taken Server", addTarget: true, wantError: "already exists"},
		{name: "running server", newName: "Renamed Server", status: "RUNNING", wantError: "must be stopped"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, oldRoot := newSettingsManagerForTest(t)
			if err := os.MkdirAll(oldRoot, 0o755); err != nil {
				t.Fatal(err)
			}
			if tt.status != "" {
				if err := manager.Store.UpdateStatus("srv-1", tt.status); err != nil {
					t.Fatal(err)
				}
			}
			if tt.targetName != "" {
				target := filepath.Join(manager.ServersPath, tt.targetName)
				if err := os.MkdirAll(target, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if tt.addTarget {
				if err := manager.Store.SaveServer(&domain.Server{
					ID: "srv-2", Name: "Taken Server", FolderName: "Taken_Server", Status: "STOPPED", CreatedAt: time.Now(),
				}); err != nil {
					t.Fatal(err)
				}
			}

			name := tt.newName
			err := manager.UpdateServer("srv-1", &name, nil, nil, nil)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("expected error containing %q, got %v", tt.wantError, err)
			}
			updated, err := manager.GetServer("srv-1")
			if err != nil {
				t.Fatal(err)
			}
			if updated.Name != "Test Server" || updated.FolderName != "test-server" {
				t.Fatalf("failed rename changed database state: name=%q folder=%q", updated.Name, updated.FolderName)
			}
			if _, err := os.Stat(oldRoot); err != nil {
				t.Fatalf("failed rename moved or removed the source folder: %v", err)
			}
		})
	}
}

func TestUpdateServerRenameRejectsMissingSourceDirectory(t *testing.T) {
	manager, _ := newSettingsManagerForTest(t)
	name := "Renamed Server"
	if err := manager.UpdateServer("srv-1", &name, nil, nil, nil); err == nil {
		t.Fatal("expected missing source directory to fail")
	}
	updated, err := manager.GetServer("srv-1")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Test Server" || updated.FolderName != "test-server" {
		t.Fatalf("missing directory changed database state: name=%q folder=%q", updated.Name, updated.FolderName)
	}
}

func TestUpdateServerSettingsRenamesLegacyDirectory(t *testing.T) {
	manager, _ := newSettingsManagerForTest(t)
	emptyFolderName := ""
	if err := manager.Store.UpdateServer("srv-1", nil, nil, nil, nil, &emptyFolderName); err != nil {
		t.Fatal(err)
	}

	legacyRoot := filepath.Join(manager.ServersPath, "Test_Server")
	if err := os.MkdirAll(legacyRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyRoot, "server.properties"), []byte("motd=Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := manager.UpdateServerSettings("srv-1", ServerSettings{
		Name:               "Renamed Server",
		RAM:                2048,
		Loader:             "vanilla",
		Version:            "1.21.1",
		JavaVersion:        8,
		Gamemode:           "survival",
		Difficulty:         "normal",
		MOTD:               "Test",
		OnlineMode:         true,
		SpawnProtection:    16,
		PvP:                true,
		MaxPlayers:         20,
		ViewDistance:       10,
		SimulationDistance: 10,
	})
	if err != nil {
		t.Fatalf("UpdateServerSettings failed: %v", err)
	}

	newRoot := filepath.Join(manager.ServersPath, "Renamed_Server")
	if _, err := os.Stat(legacyRoot); !os.IsNotExist(err) {
		t.Fatalf("expected legacy folder to be moved, got err %v", err)
	}
	if _, err := os.Stat(filepath.Join(newRoot, "server.properties")); err != nil {
		t.Fatalf("expected server.properties in renamed folder: %v", err)
	}
	updated, err := manager.GetServer("srv-1")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Renamed Server" || updated.FolderName != "Renamed_Server" {
		t.Fatalf("expected legacy folder metadata updated, got name=%q folder=%q", updated.Name, updated.FolderName)
	}
}
