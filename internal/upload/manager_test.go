package upload

import (
	"bytes"
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"naviserver/internal/domain"
	"naviserver/internal/server"
	"naviserver/internal/storage"
)

func newUploadManagerForTest(t *testing.T) *Manager {
	t.Helper()
	manager, err := NewManager(nil, nil, nil)
	if err != nil {
		t.Fatalf("create upload manager: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	return manager
}

func validFileInput(clientID, userID string, totalBytes int64) CreateInput {
	return CreateInput{
		ClientID:   clientID,
		UserID:     userID,
		Kind:       KindServerFile,
		ServerID:   "server-1",
		Filename:   "config.txt",
		TotalBytes: totalBytes,
	}
}

func tempPathFor(t *testing.T, manager *Manager, id string) string {
	t.Helper()
	manager.mu.RLock()
	session := manager.sessions[id]
	manager.mu.RUnlock()
	if session == nil {
		t.Fatalf("session %q not found", id)
	}
	return session.TempPath
}

func TestAppendChunkRequiresConfirmedSequentialOffset(t *testing.T) {
	manager := newUploadManagerForTest(t)
	created, err := manager.Create(validFileInput("client-1", "user-1", 6))
	if err != nil {
		t.Fatalf("create upload: %v", err)
	}

	view, err := manager.AppendChunk(
		created.ID,
		"user-1",
		0,
		2,
		6,
		bytes.NewReader([]byte("abc")),
	)
	if err != nil {
		t.Fatalf("append first chunk: %v", err)
	}
	if view.ReceivedBytes != 3 || view.Status != StatusUploading {
		t.Fatalf("unexpected first chunk view: %#v", view)
	}

	_, err = manager.AppendChunk(
		created.ID,
		"user-1",
		4,
		5,
		6,
		bytes.NewReader([]byte("ef")),
	)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected out-of-order chunk conflict, got %v", err)
	}

	view, err = manager.AppendChunk(
		created.ID,
		"user-1",
		3,
		5,
		6,
		bytes.NewReader([]byte("def")),
	)
	if err != nil {
		t.Fatalf("append final chunk: %v", err)
	}
	if view.ReceivedBytes != 6 || view.Status != StatusReady || view.Progress != 100 {
		t.Fatalf("unexpected final chunk view: %#v", view)
	}

	content, err := os.ReadFile(tempPathFor(t, manager, created.ID))
	if err != nil {
		t.Fatalf("read temp upload: %v", err)
	}
	if string(content) != "abcdef" {
		t.Fatalf("unexpected temp content %q", content)
	}
}

func TestCancelRemovesSessionAndTemporaryFile(t *testing.T) {
	manager := newUploadManagerForTest(t)
	created, err := manager.Create(validFileInput("client-1", "user-1", 1))
	if err != nil {
		t.Fatalf("create upload: %v", err)
	}
	tempPath := tempPathFor(t, manager, created.ID)

	if err := manager.Cancel(created.ID, "user-1"); err != nil {
		t.Fatalf("cancel upload: %v", err)
	}
	if _, err := os.Stat(tempPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected temporary file removal, got %v", err)
	}
	if _, err := manager.Get(created.ID, "user-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected removed session, got %v", err)
	}
}

func TestCreateValidatesTraversalAndChunkBounds(t *testing.T) {
	manager := newUploadManagerForTest(t)
	invalid := validFileInput("client-1", "user-1", 1)
	invalid.RelativePath = "../outside.txt"
	if _, err := manager.Create(invalid); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("expected traversal rejection, got %v", err)
	}

	created, err := manager.Create(validFileInput("client-2", "user-1", ChunkSize+1))
	if err != nil {
		t.Fatalf("create large upload: %v", err)
	}
	_, err = manager.AppendChunk(
		created.ID,
		"user-1",
		0,
		ChunkSize,
		ChunkSize+1,
		bytes.NewReader(nil),
	)
	if !errors.Is(err, ErrChunkTooLarge) {
		t.Fatalf("expected oversized chunk rejection, got %v", err)
	}

	_, err = manager.AppendChunk(
		created.ID,
		"user-1",
		0,
		math.MaxInt64,
		math.MaxInt64,
		bytes.NewReader(nil),
	)
	if !errors.Is(err, ErrInvalidChunk) {
		t.Fatalf("expected overflowing range rejection, got %v", err)
	}
}

func TestCreateIsIdempotentPerUserAndClient(t *testing.T) {
	manager := newUploadManagerForTest(t)
	first, err := manager.Create(validFileInput("client-1", "user-1", 0))
	if err != nil {
		t.Fatalf("create first upload: %v", err)
	}
	second, err := manager.Create(validFileInput("client-1", "user-1", 0))
	if err != nil {
		t.Fatalf("create duplicate upload: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected same session for duplicate client, got %q and %q", first.ID, second.ID)
	}

	otherUser, err := manager.Create(validFileInput("client-1", "user-2", 0))
	if err != nil {
		t.Fatalf("create other-user upload: %v", err)
	}
	if first.ID == otherUser.ID {
		t.Fatal("expected client id to be scoped to user")
	}
}

func TestAppendChunkRejectsShortBodyWithoutAdvancingOffset(t *testing.T) {
	manager := newUploadManagerForTest(t)
	created, err := manager.Create(validFileInput("client-1", "user-1", 3))
	if err != nil {
		t.Fatalf("create upload: %v", err)
	}

	_, err = manager.AppendChunk(
		created.ID,
		"user-1",
		0,
		2,
		3,
		bytes.NewReader([]byte("ab")),
	)
	if !errors.Is(err, ErrInvalidChunk) {
		t.Fatalf("expected short body rejection, got %v", err)
	}

	view, err := manager.Get(created.ID, "user-1")
	if err != nil {
		t.Fatalf("get upload: %v", err)
	}
	if view.ReceivedBytes != 0 {
		t.Fatalf("short body advanced offset to %d", view.ReceivedBytes)
	}
}

func TestCompleteFinalizesServerFileAsynchronously(t *testing.T) {
	root := t.TempDir()
	store, err := storage.NewGormStore(filepath.Join(root, "servers.db"))
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.SaveServer(&domain.Server{
		ID:         "server-1",
		Name:       "Test Server",
		FolderName: "test-server",
		Version:    "1.21.1",
		Loader:     "vanilla",
		Port:       25565,
		RAM:        1024,
		Status:     "STOPPED",
	}); err != nil {
		t.Fatalf("save server: %v", err)
	}

	manager, err := NewManager(
		server.NewManager(filepath.Join(root, "servers"), store, nil),
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("create upload manager: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })

	created, err := manager.Create(validFileInput("client-1", "user-1", 3))
	if err != nil {
		t.Fatalf("create upload: %v", err)
	}
	if _, err := manager.AppendChunk(
		created.ID,
		"user-1",
		0,
		2,
		3,
		bytes.NewReader([]byte("abc")),
	); err != nil {
		t.Fatalf("append upload: %v", err)
	}
	if _, err := manager.Complete(created.ID, "user-1"); err != nil {
		t.Fatalf("complete upload: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		view, err := manager.Get(created.ID, "user-1")
		if err != nil {
			t.Fatalf("get completed upload: %v", err)
		}
		if view.Status == StatusCompleted {
			content, err := os.ReadFile(filepath.Join(root, "servers", "test-server", "config.txt"))
			if err != nil {
				t.Fatalf("read finalized file: %v", err)
			}
			if string(content) != "abc" {
				t.Fatalf("unexpected finalized content %q", content)
			}
			return
		}
		if view.Status == StatusError {
			t.Fatalf("upload finalization failed: %s", view.Error)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for upload finalization")
}
