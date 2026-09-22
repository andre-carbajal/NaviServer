package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"naviserver/internal/domain"
	"naviserver/internal/storage"
	"naviserver/internal/upload"
)

func requestWithUploadUser(req *http.Request, userID, role string) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), domain.UserContextKey, map[string]string{
		"id":   userID,
		"role": role,
	}))
}

func newUploadHandlerForTest(t *testing.T, permissions []domain.Permission) *UploadHandler {
	t.Helper()
	store, err := storage.NewGormStore(filepath.Join(t.TempDir(), "uploads.db"))
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if len(permissions) > 0 {
		if err := store.SetPermissions(permissions); err != nil {
			t.Fatalf("set permissions: %v", err)
		}
	}
	manager, err := upload.NewManager(nil, nil, nil)
	if err != nil {
		t.Fatalf("create upload manager: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })

	return &UploadHandler{
		BaseHandler:   &BaseHandler{Store: store},
		UploadManager: manager,
	}
}

func newCreateUploadRequest(t *testing.T, body map[string]any) *http.Request {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return httptest.NewRequest(http.MethodPost, "/uploads", bytes.NewReader(raw))
}

func createPermittedFileUpload(t *testing.T, handler *UploadHandler, totalBytes int64) upload.View {
	t.Helper()
	req := requestWithUploadUser(newCreateUploadRequest(t, map[string]any{
		"clientId":   t.Name(),
		"kind":       upload.KindServerFile,
		"serverId":   "server-1",
		"filename":   "server.properties",
		"totalBytes": totalBytes,
	}), "viewer", "viewer")
	recorder := httptest.NewRecorder()
	handler.HandleCreate(recorder, req)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected upload session, got %d", recorder.Code)
	}

	var view upload.View
	if err := json.NewDecoder(recorder.Body).Decode(&view); err != nil {
		t.Fatalf("decode upload session: %v", err)
	}
	return view
}

func newChunkTestServer(t *testing.T, handler *UploadHandler, uploadID string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = requestWithUploadUser(r, "viewer", "viewer")
		r.SetPathValue("id", uploadID)
		handler.HandleChunk(w, r)
	}))
}

func TestHandleCreateRequiresFilePermission(t *testing.T) {
	handler := newUploadHandlerForTest(t, []domain.Permission{{
		UserID:         "viewer",
		ServerID:       "server-1",
		CanViewConsole: false,
	}})
	req := requestWithUploadUser(newCreateUploadRequest(t, map[string]any{
		"clientId":   "client-1",
		"kind":       upload.KindServerFile,
		"serverId":   "server-1",
		"filename":   "server.properties",
		"totalBytes": 0,
	}), "viewer", "viewer")
	recorder := httptest.NewRecorder()

	handler.HandleCreate(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden status, got %d", recorder.Code)
	}
}

func TestHandleCreateAllowsPermittedFileAndGlobalBackup(t *testing.T) {
	handler := newUploadHandlerForTest(t, []domain.Permission{{
		UserID:          "viewer",
		ServerID:        "server-1",
		CanViewConsole:  true,
		CanControlPower: true,
	}})

	fileRequest := requestWithUploadUser(newCreateUploadRequest(t, map[string]any{
		"clientId":   "file-client",
		"kind":       upload.KindServerFile,
		"serverId":   "server-1",
		"filename":   "server.properties",
		"totalBytes": 0,
	}), "viewer", "viewer")
	fileRecorder := httptest.NewRecorder()
	handler.HandleCreate(fileRecorder, fileRequest)
	if fileRecorder.Code != http.StatusCreated {
		t.Fatalf("expected permitted file upload, got %d", fileRecorder.Code)
	}

	backupRequest := requestWithUploadUser(newCreateUploadRequest(t, map[string]any{
		"clientId":   "backup-client",
		"kind":       upload.KindBackup,
		"filename":   "backup.zip",
		"totalBytes": 0,
	}), "viewer", "viewer")
	backupRecorder := httptest.NewRecorder()
	handler.HandleCreate(backupRecorder, backupRequest)
	if backupRecorder.Code != http.StatusCreated {
		t.Fatalf("expected global backup upload, got %d", backupRecorder.Code)
	}
}

func TestHandleCreateRejectsNonAdminIcon(t *testing.T) {
	handler := newUploadHandlerForTest(t, nil)
	req := requestWithUploadUser(newCreateUploadRequest(t, map[string]any{
		"clientId":   "icon-client",
		"kind":       upload.KindServerIcon,
		"serverId":   "server-1",
		"filename":   "icon.png",
		"totalBytes": 0,
	}), "viewer", "viewer")
	recorder := httptest.NewRecorder()

	handler.HandleCreate(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected non-admin icon upload to be forbidden, got %d", recorder.Code)
	}
}

func TestParseContentRangeRejectsInvalidBounds(t *testing.T) {
	for _, value := range []string{
		"",
		"bytes 5-4/10",
		"bytes -1-4/10",
		"bytes 0-10/10",
		"bytes 0-4/0",
		"bytes 0-4/*",
	} {
		if _, _, _, err := parseContentRange(value); err == nil {
			t.Fatalf("expected invalid Content-Range %q to fail", value)
		}
	}

	start, end, total, err := parseContentRange("bytes 5-9/10")
	if err != nil {
		t.Fatalf("valid Content-Range rejected: %v", err)
	}
	if start != 5 || end != 9 || total != 10 {
		t.Fatalf("unexpected parsed range: %d-%d/%d", start, end, total)
	}
}

func TestHandleChunkAcceptsHTTPBodyAndTracksSequentialOffset(t *testing.T) {
	handler := newUploadHandlerForTest(t, []domain.Permission{{
		UserID:         "viewer",
		ServerID:       "server-1",
		CanViewConsole: true,
	}})
	created := createPermittedFileUpload(t, handler, 6)
	server := newChunkTestServer(t, handler, created.ID)
	defer server.Close()

	sendChunk := func(body, contentRange string) upload.View {
		t.Helper()
		req, err := http.NewRequest(
			http.MethodPut,
			server.URL+"/uploads/"+created.ID+"/chunk",
			bytes.NewReader([]byte(body)),
		)
		if err != nil {
			t.Fatalf("create chunk request: %v", err)
		}
		req.Header.Set("Content-Range", contentRange)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("send chunk request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected chunk status 200, got %d", resp.StatusCode)
		}
		var view upload.View
		if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
			t.Fatalf("decode chunk response: %v", err)
		}
		return view
	}

	first := sendChunk("abc", "bytes 0-2/6")
	if first.ReceivedBytes != 3 {
		t.Fatalf("expected first confirmed offset 3, got %d", first.ReceivedBytes)
	}
	second := sendChunk("def", "bytes 3-5/6")
	if second.ReceivedBytes != 6 || second.Progress != 100 {
		t.Fatalf("expected completed chunked body, got %#v", second)
	}
}

func TestHandleChunkRejectsShortExtraAndOutOfOrderBodies(t *testing.T) {
	handler := newUploadHandlerForTest(t, []domain.Permission{{
		UserID:         "viewer",
		ServerID:       "server-1",
		CanViewConsole: true,
	}})
	created := createPermittedFileUpload(t, handler, 6)
	server := newChunkTestServer(t, handler, created.ID)
	defer server.Close()

	send := func(body, contentRange string) int {
		t.Helper()
		req, err := http.NewRequest(
			http.MethodPut,
			server.URL+"/uploads/"+created.ID+"/chunk",
			bytes.NewReader([]byte(body)),
		)
		if err != nil {
			t.Fatalf("create chunk request: %v", err)
		}
		req.Header.Set("Content-Range", contentRange)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("send chunk request: %v", err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}

	if status := send("def", "bytes 3-5/6"); status != http.StatusConflict {
		t.Fatalf("expected out-of-order chunk to return 409, got %d", status)
	}
	if status := send("ab", "bytes 0-2/6"); status != http.StatusBadRequest {
		t.Fatalf("expected short chunk to return 400, got %d", status)
	}
	if status := send("abcd", "bytes 0-2/6"); status != http.StatusBadRequest {
		t.Fatalf("expected extra chunk to return 400, got %d", status)
	}
}
