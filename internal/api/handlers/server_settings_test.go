package handlers

import (
	"bytes"
	"encoding/json"
	"naviserver/internal/domain"
	"naviserver/internal/server"
	"naviserver/internal/storage"
	"net/http"
	"net/http/httptest"
	"os"
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

	manager := server.NewManager(serversPath, store, nil)
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

func TestHandleUpdateServerSettingsValidationErrorJavaVersion(t *testing.T) {
	handler, store, _ := newTestServerHandler(t)
	srv := saveTestServer(t, store, "STOPPED")

	reqBody := map[string]any{
		"name":               srv.Name,
		"ram":                4096,
		"customArgs":         "",
		"loader":             "vanilla",
		"version":            srv.Version,
		"javaVersion":        11,
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

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d (%s)", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestHandleUpdateServerRenamesFolder(t *testing.T) {
	handler, store, serversPath := newTestServerHandler(t)
	srv := saveTestServer(t, store, "STOPPED")
	oldRoot := filepath.Join(serversPath, srv.FolderName)
	if err := os.MkdirAll(oldRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldRoot, "world.dat"), []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPut, "/servers/srv-1", bytes.NewBufferString(`{"name":"Renamed Server"}`))
	req.SetPathValue("id", srv.ID)
	rec := httptest.NewRecorder()
	handler.HandleUpdateServer(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d (%s)", http.StatusOK, rec.Code, rec.Body.String())
	}

	newRoot := filepath.Join(serversPath, "Renamed_Server")
	if _, err := os.Stat(oldRoot); !os.IsNotExist(err) {
		t.Fatalf("expected old folder to be gone, got err %v", err)
	}
	if contents, err := os.ReadFile(filepath.Join(newRoot, "world.dat")); err != nil || string(contents) != "world" {
		t.Fatalf("expected file to be preserved, contents=%q err=%v", contents, err)
	}
	updated, err := store.GetServerByID(srv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Renamed Server" || updated.FolderName != "Renamed_Server" {
		t.Fatalf("expected name and folder metadata updated, got name=%q folder=%q", updated.Name, updated.FolderName)
	}
}

func TestHandleUpdateServerSettingsRenamesFolder(t *testing.T) {
	handler, store, serversPath := newTestServerHandler(t)
	srv := saveTestServer(t, store, "STOPPED")
	oldRoot := filepath.Join(serversPath, srv.FolderName)
	if err := os.MkdirAll(oldRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldRoot, "server.properties"), []byte("motd=Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	body := map[string]any{
		"name": "Renamed Server", "ram": 4096, "customArgs": "", "loader": "vanilla",
		"version": "1.21.1", "javaVersion": 8, "gamemode": "survival", "difficulty": "normal",
		"motd": "Test", "onlineMode": true, "spawnProtection": 16, "pvp": true,
		"allowFlight": false, "enableCommandBlock": false, "hardcore": false,
		"maxPlayers": 20, "viewDistance": 10, "simulationDistance": 10,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPut, "/servers/srv-1/settings", bytes.NewReader(raw))
	req.SetPathValue("id", srv.ID)
	rec := httptest.NewRecorder()
	handler.HandleUpdateServerSettings(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d (%s)", http.StatusOK, rec.Code, rec.Body.String())
	}

	newRoot := filepath.Join(serversPath, "Renamed_Server")
	if _, err := os.Stat(oldRoot); !os.IsNotExist(err) {
		t.Fatalf("expected old folder to be gone, got err %v", err)
	}
	if _, err := os.Stat(filepath.Join(newRoot, "server.properties")); err != nil {
		t.Fatalf("expected server.properties in renamed folder: %v", err)
	}
	updated, err := store.GetServerByID(srv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Renamed Server" || updated.FolderName != "Renamed_Server" {
		t.Fatalf("expected name and folder metadata updated, got name=%q folder=%q", updated.Name, updated.FolderName)
	}
}

func TestHandleUpdateServerRenameValidationStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		status     string
		makeTarget bool
		wantStatus int
	}{
		{name: "invalid name", serverName: "Bad/Name", status: "STOPPED", wantStatus: http.StatusBadRequest},
		{name: "folder collision", serverName: "Taken Server", status: "STOPPED", makeTarget: true, wantStatus: http.StatusConflict},
		{name: "running server", serverName: "Renamed Server", status: "RUNNING", wantStatus: http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, store, serversPath := newTestServerHandler(t)
			srv := saveTestServer(t, store, tt.status)
			if err := os.MkdirAll(filepath.Join(serversPath, srv.FolderName), 0o755); err != nil {
				t.Fatal(err)
			}
			if tt.makeTarget {
				if err := os.MkdirAll(filepath.Join(serversPath, "Taken_Server"), 0o755); err != nil {
					t.Fatal(err)
				}
			}

			raw, err := json.Marshal(map[string]string{"name": tt.serverName})
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPut, "/servers/srv-1", bytes.NewReader(raw))
			req.SetPathValue("id", srv.ID)
			rec := httptest.NewRecorder()
			handler.HandleUpdateServer(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d (%s)", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}
