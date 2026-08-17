package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleDeleteServerRejectsRunningServer(t *testing.T) {
	handler, store, serversPath := newTestServerHandler(t)
	srv := saveTestServer(t, store, "RUNNING")
	serverDir := filepath.Join(serversPath, srv.FolderName)
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatalf("failed to create server directory: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+srv.ID, nil)
	req.SetPathValue("id", srv.ID)
	rec := httptest.NewRecorder()

	handler.HandleDeleteServer(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d (%s)", http.StatusConflict, rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(serverDir); err != nil {
		t.Fatal("test server directory should not be touched while the server is running")
	}
}

func TestHandleDeleteServerRemovesStoppedServer(t *testing.T) {
	handler, store, serversPath := newTestServerHandler(t)
	srv := saveTestServer(t, store, "STOPPED")
	serverDir := filepath.Join(serversPath, srv.FolderName)
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatalf("failed to create server directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "server.jar"), []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create server file: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+srv.ID, nil)
	req.SetPathValue("id", srv.ID)
	rec := httptest.NewRecorder()

	handler.HandleDeleteServer(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d (%s)", http.StatusNoContent, rec.Code, rec.Body.String())
	}
	deleted, err := store.GetServerByID(srv.ID)
	if err != nil {
		t.Fatalf("failed to query deleted server: %v", err)
	}
	if deleted != nil {
		t.Fatal("expected server record to be deleted")
	}
	if _, err := os.Stat(serverDir); !os.IsNotExist(err) {
		t.Fatal("expected server directory to be deleted")
	}
}
