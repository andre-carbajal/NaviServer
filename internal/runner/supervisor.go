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
	"strings"
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
	Cmd           *exec.Cmd
	Stdin         io.WriteCloser
	Cancel        context.CancelFunc
	StartedAt     time.Time
	Done          chan struct{}
	stopRequested bool
	killRequested bool
}

const (
	gracefulShutdownTimeout = 45 * time.Second
	processKillWaitTimeout  = 5 * time.Second
	processPollInterval     = 250 * time.Millisecond
)

func NewSupervisor(store *storage.GormStore, jvm *jvm.Manager, hubManager *ws.HubManager, serversPath string) *Supervisor {
	return &Supervisor{
		Store:       store,
		JVM:         jvm,
		HubManager:  hubManager,
		ServersPath: serversPath,
		processes:   make(map[string]*ActiveProcess),
	}
}

func (s *Supervisor) IsRunning(serverID string) bool {
	if s == nil {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.processes[serverID]
	return exists
}

func (s *Supervisor) activeServerIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(s.processes))
	for id := range s.processes {
		ids = append(ids, id)
	}
	return ids
}

func (s *Supervisor) hasActiveProcesses() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.processes) > 0
}

func (s *Supervisor) waitForProcesses(ctx context.Context) bool {
	if !s.hasActiveProcesses() {
		return true
	}

	ticker := time.NewTicker(processPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if !s.hasActiveProcesses() {
				return true
			}
		}
	}
}

func (s *Supervisor) waitForServerToStop(serverID string, timeout time.Duration) bool {
	if !s.IsRunning(serverID) {
		return true
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(processPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			return false
		case <-ticker.C:
			if !s.IsRunning(serverID) {
				return true
			}
		}
	}
}

// Shutdown asks every active server to stop cleanly, then forcefully terminates
// only the processes that did not exit before the shutdown deadline.
func (s *Supervisor) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, gracefulShutdownTimeout)
		defer cancel()
	}

	serverIDs := s.activeServerIDs()
	if len(serverIDs) == 0 {
		return nil
	}

	var shutdownErrors []error
	for _, id := range serverIDs {
		if err := s.requestStop(id, 0); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("stop server %s: %w", id, err))
		}
	}

	if !s.waitForProcesses(ctx) {
		for _, id := range s.activeServerIDs() {
			if err := s.KillServer(id); err != nil {
				shutdownErrors = append(shutdownErrors, fmt.Errorf("kill server %s: %w", id, err))
			}
		}
	}

	if s.hasActiveProcesses() {
		shutdownErrors = append(shutdownErrors, errors.New("some server processes did not stop"))
	}

	return errors.Join(shutdownErrors...)
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
		cancel()
		if killErr := cmd.Process.Kill(); killErr != nil {
			slog.Warn("could not clean up process after status update failure", "error", killErr)
		}
		_ = cmd.Wait()
		return fmt.Errorf("could not update status to RUNNING: %w", err)
	}

	done := make(chan struct{})
	s.processes[serverID] = &ActiveProcess{
		Cmd:       cmd,
		Stdin:     stdin,
		Cancel:    cancel,
		StartedAt: time.Now(),
		Done:      done,
	}

	go func(id string, c *exec.Cmd, cancelFunc context.CancelFunc) {
		err := c.Wait()
		cancelFunc()

		s.mu.Lock()
		delete(s.processes, id)
		s.mu.Unlock()

		hub := s.HubManager.GetHub(id)
		hub.ClearLogs()

		if err != nil {
			if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
				_ = exitErr.ExitCode()
			} else {
				slog.Warn("server process exited with an unexpected error", "server", id, "error", err)
			}
		}

		if uerr := s.Store.UpdateStatus(id, "STOPPED"); uerr != nil {
			slog.Warn("could not update status to STOPPED", "error", uerr)
		}
		close(done)
	}(serverID, cmd, cancel)

	return nil
}

func (s *Supervisor) StopServer(serverID string) error {
	return s.requestStop(serverID, gracefulShutdownTimeout)
}

func (s *Supervisor) requestStop(serverID string, fallbackTimeout time.Duration) error {
	s.mu.Lock()
	proc, exists := s.processes[serverID]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("server is not running")
	}
	if proc.stopRequested || proc.killRequested {
		s.mu.Unlock()
		return nil
	}
	proc.stopRequested = true
	s.mu.Unlock()

	if err := s.Store.UpdateStatus(serverID, "STOPPING"); err != nil {
		slog.Warn("could not update status to STOPPING", "error", err)
	}

	if proc.Stdin == nil {
		stopErr := fmt.Errorf("server process stdin is not available")
		if killErr := s.forceKillProcess(serverID, proc); killErr != nil {
			return errors.Join(stopErr, killErr)
		}
		slog.Warn("server process stdin is not available; forced process termination", "server", serverID)
		return nil
	}

	if _, err := io.WriteString(proc.Stdin, "stop\n"); err != nil {
		if killErr := s.forceKillProcess(serverID, proc); killErr != nil {
			return errors.Join(err, killErr)
		}
		slog.Warn("could not send stop command; forced process termination", "server", serverID, "error", err)
		return nil
	}

	if fallbackTimeout > 0 {
		s.watchStopTimeout(serverID, proc, fallbackTimeout)
	}

	return nil
}

func (s *Supervisor) watchStopTimeout(serverID string, proc *ActiveProcess, timeout time.Duration) {
	go func() {
		timer := time.NewTimer(timeout)
		defer timer.Stop()

		select {
		case <-proc.Done:
			return
		case <-timer.C:
			if err := s.forceKillProcess(serverID, proc); err != nil {
				slog.Warn("failed to force stop server after graceful shutdown timeout", "server", serverID, "error", err)
			}
		}
	}()
}

func (s *Supervisor) forceKillProcess(serverID string, proc *ActiveProcess) error {
	s.mu.Lock()
	current, exists := s.processes[serverID]
	if !exists || current != proc {
		s.mu.Unlock()
		return nil
	}
	if proc.killRequested {
		s.mu.Unlock()
		return s.waitForProcessDone(proc)
	}
	proc.killRequested = true
	proc.stopRequested = true
	s.mu.Unlock()

	if err := s.Store.UpdateStatus(serverID, "STOPPING"); err != nil {
		slog.Warn("could not update status to STOPPING", "error", err)
	}

	if proc.Cancel != nil {
		proc.Cancel()
	}

	if proc.Cmd == nil || proc.Cmd.Process == nil {
		return fmt.Errorf("server process is not available")
	}

	if err := proc.Cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("failed to kill server process: %w", err)
	}

	return s.waitForProcessDone(proc)
}

func (s *Supervisor) waitForProcessDone(proc *ActiveProcess) error {
	if proc.Done == nil {
		return nil
	}

	timer := time.NewTimer(processKillWaitTimeout)
	defer timer.Stop()

	select {
	case <-proc.Done:
		return nil
	case <-timer.C:
		return fmt.Errorf("timed out waiting for server process to stop")
	}
}

func (s *Supervisor) RestartServer(serverID string) error {
	return s.restartServerWithTimeout(serverID, gracefulShutdownTimeout)
}

func (s *Supervisor) restartServerWithTimeout(serverID string, timeout time.Duration) error {
	if !s.IsRunning(serverID) {
		return s.StartServer(serverID)
	}

	if err := s.requestStop(serverID, 0); err != nil {
		return err
	}

	if !s.waitForServerToStop(serverID, timeout) {
		if err := s.KillServer(serverID); err != nil && s.IsRunning(serverID) {
			return fmt.Errorf("timeout waiting for server to stop before restart: %w", err)
		}
		if !s.waitForServerToStop(serverID, processKillWaitTimeout) {
			return fmt.Errorf("timeout waiting for server to stop before restart")
		}
	}

	return s.StartServer(serverID)
}

func (s *Supervisor) KillServer(serverID string) error {
	s.mu.Lock()
	proc, exists := s.processes[serverID]
	s.mu.Unlock()

	if !exists {
		return fmt.Errorf("server is not running")
	}

	return s.forceKillProcess(serverID, proc)
}

func (s *Supervisor) SendCommand(serverID, cmd string) error {
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
			mcServer, err := mcstatus.NewJavaServer(fmt.Sprintf("127.0.0.1:%d", srv.Port))
			if err == nil {
				status, err := mcServer.Status()
				if err == nil {
					if s, ok := status.(*mcstatus.JavaStatusResponse); ok {
						stats.OnlinePlayers = s.Players.Online
						stats.MaxPlayers = s.Players.Max
						players := make([]domain.Player, 0, len(s.Players.Sample))
						for _, player := range s.Players.Sample {
							players = append(players, domain.Player{
								Name: player.Name,
								ID:   player.ID,
							})
						}
						stats.Players = players
					}
				}
			}
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

	mcServer, err := mcstatus.NewJavaServer(fmt.Sprintf("127.0.0.1:%d", srv.Port))
	if err == nil {
		status, err := mcServer.Status()
		if err == nil {
			if s, ok := status.(*mcstatus.JavaStatusResponse); ok {
				stats.OnlinePlayers = s.Players.Online
				stats.MaxPlayers = s.Players.Max
				players := make([]domain.Player, 0, len(s.Players.Sample))
				for _, player := range s.Players.Sample {
					players = append(players, domain.Player{
						Name: player.Name,
						ID:   player.ID,
					})
				}
				stats.Players = players
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
