package runner

import (
	"bufio"
	"context"
	"io"
	"naviserver/internal/domain"
	"naviserver/internal/storage"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	supervisorHelperEnv  = "NAVISERVER_SUPERVISOR_HELPER"
	supervisorHelperMode = "NAVISERVER_SUPERVISOR_HELPER_MODE"
)

type shutdownTestProcess struct {
	id     string
	cmd    *exec.Cmd
	done   chan struct{}
	writes *atomic.Int32
}

type countingWriteCloser struct {
	io.WriteCloser
	writes *atomic.Int32
}

func (w *countingWriteCloser) Write(data []byte) (int, error) {
	w.writes.Add(1)
	return w.WriteCloser.Write(data)
}

func TestSupervisorShutdownProcessHelper(t *testing.T) {
	if os.Getenv(supervisorHelperEnv) != "1" {
		return
	}

	if os.Getenv(supervisorHelperMode) == "graceful" || os.Getenv(supervisorHelperMode) == "ignore" {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			if strings.TrimSpace(scanner.Text()) == "stop" {
				if os.Getenv(supervisorHelperMode) == "graceful" {
					return
				}
			}
		}
	}

	select {}
}

func newShutdownTestSupervisor(t *testing.T, modes map[string]string) (*Supervisor, *storage.GormStore, []shutdownTestProcess) {
	t.Helper()

	tempDir := t.TempDir()
	store, err := storage.NewGormStore(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	supervisor := &Supervisor{
		Store:     store,
		processes: make(map[string]*ActiveProcess),
	}
	processes := make([]shutdownTestProcess, 0, len(modes))

	for id, mode := range modes {
		srv := &domain.Server{
			ID:         id,
			Name:       id,
			FolderName: id,
			Version:    "1.21.1",
			Loader:     "vanilla",
			Port:       25565,
			RAM:        1024,
			Status:     "RUNNING",
			CreatedAt:  time.Now(),
		}
		if err := store.SaveServer(srv); err != nil {
			t.Fatalf("failed to save test server %s: %v", id, err)
		}

		cmd := exec.Command(os.Args[0], "-test.run=TestSupervisorShutdownProcessHelper")
		cmd.Env = append(os.Environ(), supervisorHelperEnv+"=1", supervisorHelperMode+"="+mode)
		pipe, err := cmd.StdinPipe()
		if err != nil {
			t.Fatalf("failed to create stdin for %s: %v", id, err)
		}
		if err := cmd.Start(); err != nil {
			t.Fatalf("failed to start helper for %s: %v", id, err)
		}

		writes := &atomic.Int32{}
		stdin := &countingWriteCloser{WriteCloser: pipe, writes: writes}
		done := make(chan struct{})
		supervisor.processes[id] = &ActiveProcess{
			Cmd:   cmd,
			Stdin: stdin,
			Done:  done,
		}
		go func(serverID string, process *exec.Cmd, processDone chan struct{}) {
			_ = process.Wait()
			supervisor.mu.Lock()
			delete(supervisor.processes, serverID)
			supervisor.mu.Unlock()
			_ = store.UpdateStatus(serverID, "STOPPED")
			close(processDone)
		}(id, cmd, done)

		processes = append(processes, shutdownTestProcess{id: id, cmd: cmd, done: done, writes: writes})
	}

	t.Cleanup(func() {
		for _, process := range processes {
			if process.cmd.ProcessState == nil && process.cmd.Process != nil {
				_ = process.cmd.Process.Kill()
			}
			select {
			case <-process.done:
			case <-time.After(processKillWaitTimeout):
				t.Errorf("helper process %s did not exit during cleanup", process.id)
			}
		}
	})

	return supervisor, store, processes
}

func TestSupervisorShutdownWithoutActiveServers(t *testing.T) {
	supervisor := &Supervisor{processes: map[string]*ActiveProcess{}}

	if err := supervisor.Shutdown(context.Background()); err != nil {
		t.Fatalf("expected clean shutdown without active servers, got %v", err)
	}
}

func TestSupervisorShutdownStopsAllServersGracefully(t *testing.T) {
	supervisor, store, processes := newShutdownTestSupervisor(t, map[string]string{
		"server-one": "graceful",
		"server-two": "graceful",
	})

	if err := supervisor.Shutdown(context.Background()); err != nil {
		t.Fatalf("expected graceful shutdown, got %v", err)
	}
	if supervisor.hasActiveProcesses() {
		t.Fatal("expected all server processes to stop")
	}

	for _, process := range processes {
		if writes := process.writes.Load(); writes != 1 {
			t.Fatalf("expected one stop command for %s, got %d", process.id, writes)
		}
		server, err := store.GetServerByID(process.id)
		if err != nil {
			t.Fatalf("failed to reload %s: %v", process.id, err)
		}
		if server.Status != "STOPPED" {
			t.Fatalf("expected %s to be STOPPED, got %s", process.id, server.Status)
		}
	}
}

func TestSupervisorStopServerStopsGracefully(t *testing.T) {
	supervisor, store, processes := newShutdownTestSupervisor(t, map[string]string{
		"graceful-server": "graceful",
	})

	if err := supervisor.StopServer(processes[0].id); err != nil {
		t.Fatalf("expected graceful stop request to succeed, got %v", err)
	}
	if !supervisor.waitForServerToStop(processes[0].id, processKillWaitTimeout) {
		t.Fatal("expected graceful stop to remove the active process")
	}
	if writes := processes[0].writes.Load(); writes != 1 {
		t.Fatalf("expected one stop command, got %d", writes)
	}

	server, err := store.GetServerByID(processes[0].id)
	if err != nil {
		t.Fatalf("failed to reload server: %v", err)
	}
	if server.Status != "STOPPED" {
		t.Fatalf("expected server to be STOPPED, got %s", server.Status)
	}
}

func TestSupervisorStopServerIsIdempotentAndFallsBackToKill(t *testing.T) {
	supervisor, store, processes := newShutdownTestSupervisor(t, map[string]string{
		"unresponsive-server": "ignore",
	})

	if err := supervisor.requestStop(processes[0].id, 50*time.Millisecond); err != nil {
		t.Fatalf("expected first stop request to succeed, got %v", err)
	}
	if err := supervisor.requestStop(processes[0].id, 50*time.Millisecond); err != nil {
		t.Fatalf("expected repeated stop request to be idempotent, got %v", err)
	}
	if writes := processes[0].writes.Load(); writes != 1 {
		t.Fatalf("expected one stop command, got %d", writes)
	}

	if !supervisor.waitForServerToStop(processes[0].id, processKillWaitTimeout) {
		t.Fatal("expected fallback kill to stop the unresponsive process")
	}
	server, err := store.GetServerByID(processes[0].id)
	if err != nil {
		t.Fatalf("failed to reload server: %v", err)
	}
	if server.Status != "STOPPED" {
		t.Fatalf("expected server to be STOPPED, got %s", server.Status)
	}
}

func TestSupervisorStopServerKillsWhenStdinIsUnavailable(t *testing.T) {
	supervisor, store, processes := newShutdownTestSupervisor(t, map[string]string{
		"missing-stdin-server": "ignore",
	})

	supervisor.mu.Lock()
	supervisor.processes[processes[0].id].Stdin = nil
	supervisor.mu.Unlock()

	if err := supervisor.requestStop(processes[0].id, 0); err != nil {
		t.Fatalf("expected missing stdin to trigger a successful fallback kill, got %v", err)
	}
	if !supervisor.waitForServerToStop(processes[0].id, processKillWaitTimeout) {
		t.Fatal("expected process with missing stdin to stop")
	}

	server, err := store.GetServerByID(processes[0].id)
	if err != nil {
		t.Fatalf("failed to reload server: %v", err)
	}
	if server.Status != "STOPPED" {
		t.Fatalf("expected server to be STOPPED, got %s", server.Status)
	}
}

func TestSupervisorRestartKillsBeforeStartingReplacement(t *testing.T) {
	supervisor, store, processes := newShutdownTestSupervisor(t, map[string]string{
		"restart-server": "ignore",
	})
	if err := store.DeleteServer(processes[0].id); err != nil {
		t.Fatalf("failed to remove test server before restart: %v", err)
	}

	err := supervisor.restartServerWithTimeout(processes[0].id, 50*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "server not found") {
		t.Fatalf("expected replacement start to fail after old process stopped, got %v", err)
	}
	if supervisor.IsRunning(processes[0].id) {
		t.Fatal("expected old process to be gone before replacement start")
	}
}

func TestSupervisorShutdownKillsUnresponsiveServersAfterDeadline(t *testing.T) {
	supervisor, store, processes := newShutdownTestSupervisor(t, map[string]string{
		"unresponsive-server": "ignore",
		"graceful-server":     "graceful",
	})
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := supervisor.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("expected fallback kill to complete, got %v", err)
	}
	if supervisor.hasActiveProcesses() {
		t.Fatal("expected unresponsive server process to be removed")
	}

	for _, process := range processes {
		server, err := store.GetServerByID(process.id)
		if err != nil {
			t.Fatalf("failed to reload server %s: %v", process.id, err)
		}
		if server.Status != "STOPPED" {
			t.Fatalf("expected server %s to be STOPPED after fallback kill, got %s", process.id, server.Status)
		}
	}
}
