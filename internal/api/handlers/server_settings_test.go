package handlers

import (
	"bytes"
	"encoding/json"
	"naviserver/internal/domain"
	"naviserver/internal/server"
	"naviserver/internal/storage"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func newTestServerHandler(t *testing.T) (*ServerHandler, *storage.GormStore, string) {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	serversPath := filepath.Join(tempDir, "servers")

	store, err := storage.NewGormStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	manager := server.NewManager(serversPath, store)
	handler := &ServerHandler{
		BaseHandler: &BaseHandler{
			Manager: manager,
			Store:   store,
		},
	}

	return handler, store, serversPath
}

func saveTestServer(t *testing.T, store *storage.GormStore, status string) *domain.Server {
	t.Helper()

	srv := &domain.Server{
		ID:         "srv-1",
		Name:       "Test Server",
		FolderName: "test-server",
		Version:    "1.21.1",
		Loader:     "vanilla",
		Port:       25565,
		RAM:        4096,
		Status:     status,
		CreatedAt:  time.Now(),
	}

	if err := store.SaveServer(srv); err != nil {
		t.Fatalf("failed to save server: %v", err)
	}

	return srv
}

func TestHandleUpdateServerSettingsConflictWhenRunning(t *testing.T) {
	handler, store, _ := newTestServerHandler(t)
	srv := saveTestServer(t, store, "RUNNING")

	reqBody := map[string]any{
		"name":               srv.Name,
		"ram":                4096,
		"customArgs":         "",
		"loader":             "vanilla",
		"version":            srv.Version,
		"gamemode":           "survival",
		"difficulty":         "normal",
		"motd":               "Test",
		"onlineMode":         true,
		"spawnProtection":    16,
		"pvp":                true,
		"allowFlight":        false,
		"enableCommandBlock": false,
		"hardcore":           false,
		"maxPlayers":         20,
		"viewDistance":       10,
		"simulationDistance": 10,
	}
	raw, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/servers/srv-1/settings", bytes.NewReader(raw))
	req.SetPathValue("id", srv.ID)
	rec := httptest.NewRecorder()

	handler.HandleUpdateServerSettings(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d (%s)", http.StatusConflict, rec.Code, rec.Body.String())
	}
}

func TestHandleUpdateServerSettingsValidationError(t *testing.T) {
	handler, store, _ := newTestServerHandler(t)
	srv := saveTestServer(t, store, "STOPPED")

	reqBody := map[string]any{
		"name":               srv.Name,
		"ram":                4096,
		"customArgs":         "",
		"loader":             "vanilla",
		"version":            srv.Version,
		"gamemode":           "builder",
		"difficulty":         "normal",
		"motd":               "Test",
		"onlineMode":         true,
		"spawnProtection":    16,
		"pvp":                true,
		"allowFlight":        false,
		"enableCommandBlock": false,
		"hardcore":           false,
		"maxPlayers":         20,
		"viewDistance":       10,
		"simulationDistance": 10,
	}
	raw, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/servers/srv-1/settings", bytes.NewReader(raw))
	req.SetPathValue("id", srv.ID)
	rec := httptest.NewRecorder()

	handler.HandleUpdateServerSettings(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d (%s)", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestHandleUpdateServerSettingsValidationErrorSpawnProtection(t *testing.T) {
	handler, store, _ := newTestServerHandler(t)
	srv := saveTestServer(t, store, "STOPPED")

	reqBody := map[string]any{
		"name":               srv.Name,
		"ram":                4096,
		"customArgs":         "",
		"loader":             "vanilla",
		"version":            srv.Version,
		"gamemode":           "survival",
		"difficulty":         "normal",
		"motd":               "Test",
		"onlineMode":         true,
		"spawnProtection":    -1,
		"pvp":                true,
		"allowFlight":        false,
		"enableCommandBlock": false,
		"hardcore":           false,
		"maxPlayers":         20,
		"viewDistance":       10,
		"simulationDistance": 10,
	}
	raw, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/servers/srv-1/settings", bytes.NewReader(raw))
	req.SetPathValue("id", srv.ID)
	rec := httptest.NewRecorder()

	handler.HandleUpdateServerSettings(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d (%s)", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}
