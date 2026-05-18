package backup

import (
	"context"
	"naviserver/internal/domain"
	"naviserver/internal/storage"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestBackupManager(t *testing.T) (*Manager, *storage.GormStore, string, string) {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	serversPath := filepath.Join(tempDir, "servers")
	backupsPath := filepath.Join(tempDir, "backups")

	store, err := storage.NewGormStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if err := os.MkdirAll(serversPath, 0o755); err != nil {
		t.Fatalf("failed to create servers path: %v", err)
	}
	if err := os.MkdirAll(backupsPath, 0o755); err != nil {
		t.Fatalf("failed to create backups path: %v", err)
	}

	return NewManager(serversPath, backupsPath, store), store, serversPath, backupsPath
}

func createTestServerForBackups(t *testing.T, store *storage.GormStore, id string, maxBackups int, enabled bool, lastRunAt *time.Time) {
	t.Helper()

	srv := &domain.Server{
		ID:                      id,
		Name:                    "Server " + id,
		FolderName:              "server-" + id,
		Version:                 "1.21.1",
		Loader:                  "vanilla",
		Port:                    25565,
		RAM:                     2048,
		Status:                  "STOPPED",
		CreatedAt:               time.Now().UTC(),
		AutoBackupEnabled:       enabled,
		AutoBackupIntervalValue: 5,
		AutoBackupIntervalUnit:  "minute",
		AutoBackupMaxBackups:    maxBackups,
		AutoBackupLastRunAt:     lastRunAt,
	}

	if err := store.SaveServer(srv); err != nil {
		t.Fatalf("failed to save server: %v", err)
	}
}

func TestParseAutoBackupInterval(t *testing.T) {
	if _, err := ParseAutoBackupInterval(4, "minute"); err == nil {
		t.Fatalf("expected error for interval below minimum")
	}
	if _, err := ParseAutoBackupInterval(31, "day"); err == nil {
		t.Fatalf("expected error for interval above maximum")
	}
	if _, err := ParseAutoBackupInterval(1, "week"); err == nil {
		t.Fatalf("expected error for invalid interval unit")
	}

	got, err := ParseAutoBackupInterval(2, "hour")
	if err != nil {
		t.Fatalf("expected valid interval, got error: %v", err)
	}
	if got != 2*time.Hour {
		t.Fatalf("unexpected interval: %s", got)
	}
}

func TestApplyBackupLimitPrunesOldestBackups(t *testing.T) {
	manager, store, _, backupsPath := newTestBackupManager(t)
	createTestServerForBackups(t, store, "srv-1", 2, false, nil)

	createdAts := []time.Time{
		time.Now().Add(-3 * time.Hour),
		time.Now().Add(-2 * time.Hour),
		time.Now().Add(-1 * time.Hour),
	}

	for i := 0; i < 3; i++ {
		name := "backup-" + time.Date(2026, 1, i+1, 1, 0, 0, 0, time.UTC).Format("20060102150405") + ".zip"
		if err := os.WriteFile(filepath.Join(backupsPath, name), []byte("test"), 0o644); err != nil {
			t.Fatalf("failed writing backup file: %v", err)
		}
		err := store.SaveBackup(&domain.Backup{
			ID:        name,
			Name:      name,
			FileName:  name,
			ServerID:  "srv-1",
			Size:      4,
			CreatedAt: createdAts[i],
			CreatedBy: "test",
		})
		if err != nil {
			t.Fatalf("failed to save backup: %v", err)
		}
	}

	if err := manager.applyBackupLimit("srv-1"); err != nil {
		t.Fatalf("applyBackupLimit failed: %v", err)
	}

	backups, err := store.ListBackupsByServerID("srv-1")
	if err != nil {
		t.Fatalf("failed listing backups: %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("expected 2 backups after pruning, got %d", len(backups))
	}
}

func TestRunAutoBackupCycleCreatesBackupWhenDue(t *testing.T) {
	manager, store, serversPath, _ := newTestBackupManager(t)
	lastRun := time.Now().UTC().Add(-10 * time.Minute)
	createTestServerForBackups(t, store, "srv-2", 10, true, &lastRun)

	serverDir := filepath.Join(serversPath, "server-srv-2")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatalf("failed creating server dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "server.properties"), []byte("motd=test"), 0o644); err != nil {
		t.Fatalf("failed writing test server file: %v", err)
	}

	manager.runAutoBackupCycle(context.Background())

	deadline := time.Now().Add(3 * time.Second)
	for {
		backups, err := store.ListBackupsByServerID("srv-2")
		if err != nil {
			t.Fatalf("failed listing backups: %v", err)
		}
		if len(backups) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected auto backup to be created")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestAutoBackupLockPreventsOverlap(t *testing.T) {
	manager, _, _, _ := newTestBackupManager(t)

	if !manager.acquireAutoBackupLock("srv-1") {
		t.Fatalf("expected first lock acquisition to succeed")
	}
	if manager.acquireAutoBackupLock("srv-1") {
		t.Fatalf("expected second lock acquisition to fail")
	}
	manager.releaseAutoBackupLock("srv-1")
	if !manager.acquireAutoBackupLock("srv-1") {
		t.Fatalf("expected lock acquisition after release to succeed")
	}
}
