package server

import (
	"fmt"
	"naviserver/internal/domain"
	"naviserver/internal/storage"
	"net"
	"path/filepath"
	"testing"
	"time"
)

func newPortAllocatorStore(t *testing.T) *storage.GormStore {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "allocator.db")
	store, err := storage.NewGormStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	return store
}

func reserveTCPPort(t *testing.T) (int, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to reserve port: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	return port, func() {
		_ = listener.Close()
	}
}

func reserveUDPPort(t *testing.T) (int, func()) {
	t.Helper()

	listener, err := net.ListenPacket("udp", ":0")
	if err != nil {
		t.Fatalf("failed to reserve UDP port: %v", err)
	}

	port := listener.LocalAddr().(*net.UDPAddr).Port
	return port, func() {
		_ = listener.Close()
	}
}

func findAvailablePort(t *testing.T, start, end int) int {
	t.Helper()

	for port := start; port <= end; port++ {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			_ = listener.Close()
			return port
		}
	}

	t.Fatalf("no available port found in range %d-%d", start, end)
	return 0
}

func TestAllocatePortSkipsPortsUsedByServers(t *testing.T) {
	store := newPortAllocatorStore(t)
	occupiedPort, release := reserveTCPPort(t)
	defer release()

	rangeEnd := occupiedPort + 10
	freePortToUse := findAvailablePort(t, occupiedPort+1, rangeEnd)

	if err := store.SetPortRange(occupiedPort, rangeEnd); err != nil {
		t.Fatalf("failed to set port range: %v", err)
	}

	err := store.SaveServer(&domain.Server{
		ID:         "srv-1",
		Name:       "Used Server",
		FolderName: "used-server",
		Version:    "1.21.1",
		Loader:     "vanilla",
		Port:       occupiedPort,
		RAM:        4096,
		Status:     "STOPPED",
		CreatedAt:  time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to save server: %v", err)
	}

	allocated, err := AllocatePort(store)
	if err != nil {
		t.Fatalf("expected free port, got error: %v", err)
	}

	if allocated != freePortToUse {
		t.Fatalf("expected port %d, got %d", freePortToUse, allocated)
	}
}

func TestAllocatePortReturnsErrorWhenRangeIsExhausted(t *testing.T) {
	store := newPortAllocatorStore(t)
	occupiedPort, release := reserveTCPPort(t)
	defer release()

	if err := store.SetPortRange(occupiedPort, occupiedPort); err != nil {
		t.Fatalf("failed to set port range: %v", err)
	}

	allocated, err := AllocatePort(store)
	if err == nil {
		t.Fatalf("expected exhaustion error, got port %d", allocated)
	}
}

func TestAllocatePortSkipsUDPPorts(t *testing.T) {
	store := newPortAllocatorStore(t)
	occupiedPort, release := reserveUDPPort(t)
	defer release()

	if err := store.SetPortRange(occupiedPort, occupiedPort); err != nil {
		t.Fatalf("failed to set port range: %v", err)
	}

	allocated, err := AllocatePort(store)
	if err == nil {
		t.Fatalf("expected UDP exhaustion error, got port %d", allocated)
	}
}
