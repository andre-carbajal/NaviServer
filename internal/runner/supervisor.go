package runner

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"naviger/internal/jvm"
	"naviger/internal/runner/strategy"
	"naviger/internal/server"
	"naviger/internal/storage"
	"naviger/internal/ws"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"naviger/internal/domain"

	"github.com/shirou/gopsutil/v3/process"
)

type Supervisor struct {
	Store       *storage.GormStore
	JVM         *jvm.Manager
	HubManager  *ws.HubManager
	ServersPath string
	processes   map[string]*ActiveProcess
	mu          sync.Mutex
}

type ActiveProcess struct {
	Cmd    *exec.Cmd
	Stdin  io.WriteCloser
	Cancel context.CancelFunc
}

func NewSupervisor(store *storage.GormStore, jvm *jvm.Manager, hubManager *ws.HubManager, serversPath string) *Supervisor {
	return &Supervisor{
		Store:       store,
		JVM:         jvm,
		HubManager:  hubManager,
		ServersPath: serversPath,
		processes:   make(map[string]*ActiveProcess),
	}
}

func (s *Supervisor) StartServer(serverID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.processes[serverID]; exists {
		return fmt.Errorf("server is already running")
	}

	srv, err := s.Store.GetServerByID(serverID)
	if err != nil {
		return err
	}
	if srv == nil {
		return fmt.Errorf("server not found")
	}

	folderName := srv.FolderName
	if folderName == "" {
		folderName = srv.ID
	}

	serverDir := filepath.Join(s.ServersPath, folderName)
	absServerDir, err := filepath.Abs(serverDir)
	if err != nil {
		return fmt.Errorf("error getting absolute path for server: %w", err)
	}

	if err := checkPortAvailable(srv.Port); err != nil {
		slog.Info("Port is busy, attempting to allocate a new one", "port", srv.Port)
		newPort, err := server.AllocatePort(s.Store)
		if err != nil {
			return fmt.Errorf("failed to allocate new port: %w", err)
		}

		if err := s.Store.UpdateServerPort(srv.ID, newPort); err != nil {
			return fmt.Errorf("failed to update server port in database: %w", err)
		}
		srv.Port = newPort
		slog.Info("Reassigned server to new port", "server", srv.Name, "port", newPort)
	}

	configFile := filepath.Join(absServerDir, "server.properties")
	if err := ensurePortInProperties(configFile, srv.Port); err != nil {
		slog.Warn("Could not update server.properties", "error", err)
	}

	requiredJava := GetJavaVersionForMC(srv.Version)
	javaPath, err := s.JVM.EnsureJava(requiredJava)
	if err != nil {
		return fmt.Errorf("error preparing Java: %w", err)
	}

	runner := strategy.GetRunner(srv.Loader)
	cmd, err := runner.BuildCommand(javaPath, absServerDir, srv.RAM, srv.CustomArgs)
	if err != nil {
		return err
	}

	prepareCommand(cmd)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	hub := s.HubManager.GetHub(serverID)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				text := scanner.Text()
				hub.Broadcast([]byte(text))
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				text := scanner.Text()
				hub.Broadcast([]byte(text))
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case command, ok := <-hub.Commands:
				if !ok {
					return
				}
				_, err := stdin.Write(command)
				if err != nil {
					return
				}
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("failed to start: %w", err)
	}

	if err := s.Store.UpdateStatus(serverID, "RUNNING"); err != nil {
		slog.Warn("could not update status to RUNNING", "error", err)
	}

	s.processes[serverID] = &ActiveProcess{
		Cmd:    cmd,
		Stdin:  stdin,
		Cancel: cancel,
	}

	go func(id string, c *exec.Cmd, cancelFunc context.CancelFunc) {
		err := c.Wait()
		cancelFunc()

		s.mu.Lock()
		delete(s.processes, id)
		s.mu.Unlock()

		if err == nil {
			if uerr := s.Store.UpdateStatus(id, "STOPPED"); uerr != nil {
				slog.Warn("could not update status to STOPPED", "error", uerr)
			}
			return
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			_ = exitErr.ExitCode()
			if uerr := s.Store.UpdateStatus(id, "STOPPED"); uerr != nil {
				slog.Warn("could not update status to STOPPED", "error", uerr)
			}
		}

	}(serverID, cmd, cancel)

	return nil
}

func (s *Supervisor) StopServer(serverID string) error {
	s.mu.Lock()
	proc, exists := s.processes[serverID]
	s.mu.Unlock()

	if !exists {
		return fmt.Errorf("server is not running")
	}

	if err := s.Store.UpdateStatus(serverID, "STOPPING"); err != nil {
		slog.Warn("could not update status to STOPPING", "error", err)
	}
	_, err := io.WriteString(proc.Stdin, "stop\n")
	return err
}

func (s *Supervisor) SendCommand(serverID string, cmd string) error {
	s.mu.Lock()
	proc, exists := s.processes[serverID]
	s.mu.Unlock()

	if !exists {
		return fmt.Errorf("server is not running")
	}

	_, err := io.WriteString(proc.Stdin, cmd+"\n")
	return err
}

func (s *Supervisor) GetServerStats(serverID string) (*domain.ServerStats, error) {
	s.mu.Lock()
	proc, exists := s.processes[serverID]
	s.mu.Unlock()

	stats := &domain.ServerStats{
		CPU:  0,
		RAM:  0,
		Disk: 0,
	}

	srv, err := s.Store.GetServerByID(serverID)
	if err == nil && srv != nil {
		folderName := srv.FolderName
		if folderName == "" {
			folderName = srv.ID
		}
		serverDir := filepath.Join(s.ServersPath, folderName)
		var size int64
		_ = filepath.Walk(serverDir, func(_ string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				size += info.Size()
			}
			return nil
		})
		stats.Disk = size
	}

	if !exists {
		return stats, nil
	}

	if proc.Cmd != nil && proc.Cmd.Process != nil {
		p, err := process.NewProcess(int32(proc.Cmd.Process.Pid))
		if err == nil {
			if cpu, err := p.CPUPercent(); err == nil {
				stats.CPU = cpu
			}
			if mem, err := p.MemoryInfo(); err == nil {
				stats.RAM = mem.RSS
			}
		}
	}

	return stats, nil
}

func (s *Supervisor) GetAllServerStats() (map[string]domain.ServerStats, error) {
	servers, err := s.Store.ListServers()
	if err != nil {
		return nil, err
	}

	result := make(map[string]domain.ServerStats)

	for _, srv := range servers {
		stats, err := s.GetServerStats(srv.ID)
		if err == nil && stats != nil {
			result[srv.ID] = *stats
		} else {
			result[srv.ID] = domain.ServerStats{}
		}
	}

	return result, nil
}

func (s *Supervisor) ResetRunningStates() error {
	servers, err := s.Store.ListServers()
	if err != nil {
		return err
	}

	for _, srv := range servers {
		if srv.Status == "RUNNING" || srv.Status == "STARTING" || srv.Status == "STOPPING" {
			if err := s.Store.UpdateStatus(srv.ID, "STOPPED"); err != nil {
				slog.Error("Failed to reset status for server", "server", srv.Name, "error", err)
			} else {
				slog.Info("Reset server status to STOPPED", "server", srv.Name)
			}
		}
	}
	return nil
}

func ensurePortInProperties(path string, port int) error {
	props := make(map[string]string)
	var lines []string

	if file, err := os.Open(path); err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			lines = append(lines, line)

			if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				props[key] = val
			}
		}
		file.Close()
	}

	portStr := fmt.Sprintf("%d", port)
	if currentVal, ok := props["server-port"]; ok && currentVal == portStr {
		return nil
	}

	var newContent []string
	portUpdated := false

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "server-port=") || strings.HasPrefix(strings.TrimSpace(line), "server-port =") {
			newContent = append(newContent, fmt.Sprintf("server-port=%s", portStr))
			portUpdated = true
		} else {
			newContent = append(newContent, line)
		}
	}

	if !portUpdated {
		newContent = append(newContent, fmt.Sprintf("server-port=%s", portStr))
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range newContent {
		writer.WriteString(line + "\n")
	}
	return writer.Flush()
}

func checkPortAvailable(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("port %d is not available: %w", port, err)
	}
	_ = ln.Close()
	return nil
}
