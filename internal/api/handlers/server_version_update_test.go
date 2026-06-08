package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"naviserver/internal/backup"
	"naviserver/internal/domain"
	"naviserver/internal/loader"
	"naviserver/internal/server"
	"naviserver/internal/storage"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeVersionLoader struct {
	fail bool
}

func (l fakeVersionLoader) Load(options loader.LoaderOptions, destDir string, progressChan chan<- domain.ProgressEvent) (string, error) {
	if err := os.WriteFile(filepath.Join(destDir, "server.jar"), []byte("updated"), 0644); err != nil {
		return "", err
	}
	if l.fail {
		return "", fmt.Errorf("fake loader failure")
	}
	return options.MCVersion, nil
}

func (l fakeVersionLoader) GetSupportedVersions(options loader.LoaderOptions) ([]string, error) {
	return []string{"1.21.1", "1.21.2"}, nil
}

func (l fakeVersionLoader) GetMetadata(options loader.LoaderOptions) (*loader.LoaderMetadata, error) {
	return &loader.LoaderMetadata{MinecraftVersions: []string{"1.21.1", "1.21.2"}}, nil
}

func newVersionUpdateHandler(t *testing.T, failLoader bool) (*ServerHandler, *storage.GormStore, string, string) {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	serversPath := filepath.Join(tempDir, "servers")
	backupsPath := filepath.Join(tempDir, "backups")

	store, err := storage.NewGormStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	restore := loader.RegisterLoaderForTest("test-loader", func() loader.ServerLoader {
		return fakeVersionLoader{fail: failLoader}
	})
	t.Cleanup(restore)

	srv := &domain.Server{
		ID:         "srv-1",
		Name:       "Test Server",
		FolderName: "test-server",
		Version:    "1.21.1",
		Loader:     "test-loader",
		Port:       25565,
		RAM:        4096,
		Status:     "STOPPED",
		CreatedAt:  time.Now(),
	}
	if err := store.SaveServer(srv); err != nil {
		t.Fatalf("failed to save server: %v", err)
	}

	serverDir := filepath.Join(serversPath, srv.FolderName)
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		t.Fatalf("failed to create server dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "server.jar"), []byte("original"), 0644); err != nil {
		t.Fatalf("failed to write server jar: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "server.properties"), []byte("server-port=25565\n"), 0644); err != nil {
		t.Fatalf("failed to write server properties: %v", err)
	}

	manager := server.NewManager(serversPath, store)
	backupManager := backup.NewManager(serversPath, backupsPath, store)
	handler := &ServerHandler{
		BaseHandler: &BaseHandler{
			Manager:       manager,
			Store:         store,
			BackupManager: backupManager,
		},
	}
	return handler, store, serverDir, backupsPath
}

func TestHandleUpdateServerVersionCreatesBackupAndUpdatesVersion(t *testing.T) {
	handler, store, serverDir, backupsPath := newVersionUpdateHandler(t, false)

	raw, _ := json.Marshal(map[string]string{"version": "1.21.2"})
	req := httptest.NewRequest(http.MethodPost, "/servers/srv-1/version-update", bytes.NewReader(raw))
	req.SetPathValue("id", "srv-1")
	rec := httptest.NewRecorder()

	handler.HandleUpdateServerVersion(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d (%s)", http.StatusOK, rec.Code, rec.Body.String())
	}

	updated, err := store.GetServerByID("srv-1")
	if err != nil {
		t.Fatalf("failed to reload server: %v", err)
	}
	if updated.Version != "1.21.2" {
		t.Fatalf("expected version 1.21.2, got %s", updated.Version)
	}
	jar, _ := os.ReadFile(filepath.Join(serverDir, "server.jar"))
	if string(jar) != "updated" {
		t.Fatalf("expected updated jar, got %q", string(jar))
	}
	backups, err := os.ReadDir(backupsPath)
	if err != nil {
		t.Fatalf("failed to read backups: %v", err)
	}
	if len(backups) != 1 || !strings.Contains(backups[0].Name(), "pre-update-Test_Server-1.21.1-to-1.21.2") {
		t.Fatalf("expected pre-update backup, got %#v", backups)
	}
}

func TestHandleUpdateServerVersionRestoresBackupWhenLoaderFails(t *testing.T) {
	handler, store, serverDir, _ := newVersionUpdateHandler(t, true)

	raw, _ := json.Marshal(map[string]string{"version": "1.21.2"})
	req := httptest.NewRequest(http.MethodPost, "/servers/srv-1/version-update", bytes.NewReader(raw))
	req.SetPathValue("id", "srv-1")
	rec := httptest.NewRecorder()

	handler.HandleUpdateServerVersion(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d (%s)", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "restored backup") {
		t.Fatalf("expected restore message, got %s", rec.Body.String())
	}

	updated, err := store.GetServerByID("srv-1")
	if err != nil {
		t.Fatalf("failed to reload server: %v", err)
	}
	if updated.Version != "1.21.1" {
		t.Fatalf("expected version to remain 1.21.1, got %s", updated.Version)
	}
	jar, _ := os.ReadFile(filepath.Join(serverDir, "server.jar"))
	if string(jar) != "original" {
		t.Fatalf("expected restored jar, got %q", string(jar))
	}
}

func TestHandleUpdateServerVersionConflictWhenRunningWithoutBackupManager(t *testing.T) {
	handler, store, _ := newTestServerHandler(t)
	srv := saveTestServer(t, store, "RUNNING")

	raw, _ := json.Marshal(map[string]string{"version": "1.21.2"})
	req := httptest.NewRequest(http.MethodPost, "/servers/srv-1/version-update", bytes.NewReader(raw))
	req.SetPathValue("id", srv.ID)
	rec := httptest.NewRecorder()

	handler.HandleUpdateServerVersion(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d (%s)", http.StatusConflict, rec.Code, rec.Body.String())
	}
}
