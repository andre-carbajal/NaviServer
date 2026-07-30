package handlers

import (
	"bytes"
	"encoding/json"
	"naviserver/internal/backup"
	"naviserver/internal/domain"
	"naviserver/internal/server"
	"naviserver/internal/storage"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func newServerHandlerWithBackupManager(t *testing.T) (*ServerHandler, *storage.GormStore) {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	serversPath := filepath.Join(tempDir, "servers")
	backupsPath := filepath.Join(tempDir, "backups")

	store, err := storage.NewGormStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	manager := server.NewManager(serversPath, store)
	backupManager := backup.NewManager(serversPath, backupsPath, store)
	handler := &ServerHandler{
		BaseHandler: &BaseHandler{
			Manager:       manager,
			Store:         store,
			BackupManager: backupManager,
		},
	}

	return handler, store
}

func saveServerForAutoBackupTest(t *testing.T, store *storage.GormStore) {
	t.Helper()

	srv := &domain.Server{
		ID:         "srv-auto-1",
		Name:       "Auto Server",
		FolderName: "auto-server",
		Version:    "1.21.1",
		Loader:     "vanilla",
		Port:       25566,
		RAM:        4096,
		Status:     "STOPPED",
		CreatedAt:  time.Now().UTC(),
	}

	if err := store.SaveServer(srv); err != nil {
		t.Fatalf("failed to save server: %v", err)
	}
}

func TestHandleUpdateServerAutoBackupValidationError(t *testing.T) {
	handler, store := newServerHandlerWithBackupManager(t)
	saveServerForAutoBackupTest(t, store)

	reqBody := map[string]any{
		"enabled":       true,
		"intervalValue": 1,
		"intervalUnit":  "minute",
		"maxBackups":    10,
	}
	raw, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/servers/srv-auto-1/auto-backup", bytes.NewReader(raw))
	req.SetPathValue("id", "srv-auto-1")
	rec := httptest.NewRecorder()

	handler.HandleUpdateServerAutoBackup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d (%s)", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestHandleUpdateServerAutoBackupSuccess(t *testing.T) {
	handler, store := newServerHandlerWithBackupManager(t)
	saveServerForAutoBackupTest(t, store)

	reqBody := map[string]any{
		"enabled":       true,
		"intervalValue": 6,
		"intervalUnit":  "hour",
		"maxBackups":    12,
	}
	raw, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/servers/srv-auto-1/auto-backup", bytes.NewReader(raw))
	req.SetPathValue("id", "srv-auto-1")
	rec := httptest.NewRecorder()

	handler.HandleUpdateServerAutoBackup(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d (%s)", http.StatusOK, rec.Code, rec.Body.String())
	}

	srv, err := store.GetServerByID("srv-auto-1")
	if err != nil {
		t.Fatalf("failed to load server: %v", err)
	}
	if srv == nil {
		t.Fatalf("server not found after update")
	}
	if !srv.AutoBackupEnabled {
		t.Fatalf("expected auto backup enabled")
	}
	if srv.AutoBackupIntervalValue != 6 || srv.AutoBackupIntervalUnit != "hour" {
		t.Fatalf("unexpected interval config: %d %s", srv.AutoBackupIntervalValue, srv.AutoBackupIntervalUnit)
	}
	if srv.AutoBackupMaxBackups != 12 {
		t.Fatalf("unexpected max backups: %d", srv.AutoBackupMaxBackups)
	}
}
