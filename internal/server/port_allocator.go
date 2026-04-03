package server

import (
	"fmt"
	"naviserver/internal/storage"
	"net"
)

func AllocatePort(store *storage.GormStore) (int, error) {
	startPort, endPort, err := store.GetPortRange()
	if err != nil {
		return 0, fmt.Errorf("error getting port range: %w", err)
	}

	servers, err := store.ListServers()
	if err != nil {
		return 0, err
	}

	usedPorts := make(map[int]bool)
	for _, s := range servers {
		usedPorts[s.Port] = true
	}

	for port := startPort; port <= endPort; port++ {
		if usedPorts[port] {
			continue
		}

		if isPortAvailable(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no free ports in range %d-%d", startPort, endPort)
}

func isPortAvailable(port int) bool {
	conn, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
