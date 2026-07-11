package runner

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"naviserver/internal/jvm"
	"naviserver/internal/runner/strategy"
	"naviserver/internal/server"
	"naviserver/internal/storage"
	"naviserver/internal/ws"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"naviserver/internal/domain"

	"github.com/andre-carbajal/go-mcstatus"
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
	Cmd       *exec.Cmd
	Stdin     io.WriteCloser
	Cancel    context.CancelFunc
	StartedAt time.Time
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

	if err := checkPortAvailable(srv.Loader, srv.Port); err != nil {
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

	if err := server.UpdateServerPropertiesForLoader(absServerDir, srv.Port, srv.Loader); err != nil {
		slog.Warn("Could not update server.properties", "error", err)
	}

	javaPath := ""
	if srv.Loader != "bedrock" {
		requiredJava := GetJavaVersionForMC(srv.Version)
		javaPath, err = s.JVM.EnsureJava(requiredJava)
		if err != nil {
			return fmt.Errorf("error preparing Java: %w", err)
		}
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
		Cmd:       cmd,
		Stdin:     stdin,
		Cancel:    cancel,
		StartedAt: time.Now(),
	}

	go func(id string, c *exec.Cmd, cancelFunc context.CancelFunc) {
		err := c.Wait()
		cancelFunc()

		s.mu.Lock()
		delete(s.processes, id)
		s.mu.Unlock()

		hub := s.HubManager.GetHub(id)
		hub.ClearLogs()

		if err == nil {
			if uerr := s.Store.UpdateStatus(id, "STOPPED"); uerr != nil {
				slog.Warn("could not update status to STOPPED", "error", uerr)
			}
			return
		}

		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
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

func (s *Supervisor) RestartServer(serverID string) error {
	s.mu.Lock()
	_, exists := s.processes[serverID]
	s.mu.Unlock()

	if !exists {
		return s.StartServer(serverID)
	}

	if err := s.StopServer(serverID); err != nil {
		return err
	}

	timeout := time.After(45 * time.Second)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for server to stop before restart")
		case <-ticker.C:
			s.mu.Lock()
			_, stillRunning := s.processes[serverID]
			s.mu.Unlock()

			if !stillRunning {
				return s.StartServer(serverID)
			}
		}
	}
}

func (s *Supervisor) KillServer(serverID string) error {
	s.mu.Lock()
	proc, exists := s.processes[serverID]
	s.mu.Unlock()

	if !exists {
		return fmt.Errorf("server is not running")
	}

	if err := s.Store.UpdateStatus(serverID, "STOPPING"); err != nil {
		slog.Warn("could not update status to STOPPING", "error", err)
	}

	if proc.Cancel != nil {
		proc.Cancel()
	}

	if proc.Cmd == nil || proc.Cmd.Process == nil {
		return fmt.Errorf("server process is not available")
	}

	if err := proc.Cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill server process: %w", err)
	}

	if err := s.Store.UpdateStatus(serverID, "STOPPED"); err != nil {
		slog.Warn("could not update status to STOPPED after kill", "error", err)
	}

	return nil
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
		CPU:     0,
		RAM:     0,
		Disk:    0,
		Players: []domain.Player{},
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
		if srv != nil && srv.Status == "RUNNING" {
			populateMinecraftStatus(stats, srv)
		}
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

	if !proc.StartedAt.IsZero() {
		stats.UptimeSeconds = int64(time.Since(proc.StartedAt).Seconds())
	}

	populateMinecraftStatus(stats, srv)

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

func populateMinecraftStatus(stats *domain.ServerStats, srv *domain.Server) {
	address := fmt.Sprintf("127.0.0.1:%d", srv.Port)
	if srv.Loader == "bedrock" {
		mcServer, err := mcstatus.NewBedrockServer(address)
		if err != nil {
			return
		}
		status, err := mcServer.Status()
		if err != nil {
			return
		}
		if response, ok := status.(*mcstatus.BedrockStatusResponse); ok {
			stats.OnlinePlayers = response.Online
			stats.MaxPlayers = response.Max
		}
		return
	}

	mcServer, err := mcstatus.NewJavaServer(address)
	if err != nil {
		return
	}
	status, err := mcServer.Status()
	if err != nil {
		return
	}
	if response, ok := status.(*mcstatus.JavaStatusResponse); ok {
		stats.OnlinePlayers = response.Players.Online
		stats.MaxPlayers = response.Players.Max
		players := make([]domain.Player, 0, len(response.Players.Sample))
		for _, player := range response.Players.Sample {
			players = append(players, domain.Player{Name: player.Name, ID: player.ID})
		}
		stats.Players = players
	}
}

func checkPortAvailable(loaderType string, port int) error {
	network := "tcp"
	if loaderType == "bedrock" {
		network = "udp"
	}
	if network == "udp" {
		listener, err := net.ListenPacket(network, fmt.Sprintf(":%d", port))
		if err != nil {
			return fmt.Errorf("UDP port %d is not available: %w", port, err)
		}
		return listener.Close()
	}
	listener, err := net.Listen(network, fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("TCP port %d is not available: %w", port, err)
	}
	return listener.Close()
}
