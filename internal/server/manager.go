package server

import (
	"fmt"
	"image"
	"image/png"
	"naviserver/internal/domain"
	"naviserver/internal/loader"
	"naviserver/internal/storage"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/image/draw"
)

type Manager struct {
	ServersPath string
	Store       *storage.GormStore
}

func NewManager(serversPath string, store *storage.GormStore) *Manager {
	return &Manager{
		ServersPath: serversPath,
		Store:       store,
	}
}

func sanitizeFolderName(name string) string {
	name = strings.ReplaceAll(name, " ", "_")
	reg := regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
	sanitized := reg.ReplaceAllString(name, "")
	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
	}
	return sanitized
}

func (m *Manager) StartCreateServerJob(name, loaderType, version string, ram int, progressChan chan<- domain.ProgressEvent) {
	go func() {
		defer close(progressChan)
		srv, err := m.CreateServer(name, loaderType, version, ram, progressChan)
		if err != nil {
			fmt.Printf("Error creating server: %v\n", err)
			event := domain.ProgressEvent{
				ServerID: "error",
				Message:  fmt.Sprintf("Error: %v", err),
				Progress: 0,
			}
			progressChan <- event
			return
		}

		event := domain.ProgressEvent{
			ServerID: srv.ID,
			Message:  "Server created successfully",
			Progress: 100,
		}
		progressChan <- event
	}()
}

func (m *Manager) CreateServer(name string, loaderType string, version string, ram int, progressChan chan<- domain.ProgressEvent) (*domain.Server, error) {
	if strings.ContainsAny(name, "\\/:*?\"<>|") || strings.Contains(name, "..") {
		return nil, fmt.Errorf("invalid server name: contains forbidden characters")
	}

	id := uuid.New().String()
	folderName := sanitizeFolderName(name)
	serverDir := filepath.Join(m.ServersPath, folderName)

	if _, err := os.Stat(serverDir); !os.IsNotExist(err) {
		folderName = fmt.Sprintf("%s-%s", folderName, id[:8])
		serverDir = filepath.Join(m.ServersPath, folderName)
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Allocating port..."}
	}
	assignedPort, err := AllocatePort(m.Store)
	if err != nil {
		return nil, fmt.Errorf("error allocating port: %w", err)
	}
	fmt.Printf("Port allocated for '%s': %d\n", name, assignedPort)

	downloader, err := loader.GetLoader(loaderType)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(serverDir, 0755); err != nil {
		return nil, fmt.Errorf("filesystem error: %w", err)
	}

	if err := downloader.Load(version, serverDir, progressChan); err != nil {
		os.RemoveAll(serverDir)
		return nil, fmt.Errorf("download error: %w", err)
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Configuring server..."}
	}
	os.WriteFile(filepath.Join(serverDir, "eula.txt"), []byte("eula=true"), 0644)

	if err := UpdateServerProperties(serverDir, assignedPort); err != nil {
		fmt.Printf("Warning: Could not write server.properties: %v\n", err)
	}

	newServer := &domain.Server{
		ID:         id,
		Name:       name,
		FolderName: folderName,
		Version:    version,
		Loader:     loaderType,
		Port:       assignedPort,
		RAM:        ram,
		Status:     "STOPPED",
		CreatedAt:  time.Now(),
	}

	if err := m.Store.SaveServer(newServer); err != nil {
		os.RemoveAll(serverDir)
		return nil, fmt.Errorf("DB error: %w", err)
	}

	return newServer, nil
}

func (m *Manager) GetServer(id string) (*domain.Server, error) {
	return m.Store.GetServerByID(id)
}

func (m *Manager) ListServers() ([]domain.Server, error) {
	return m.Store.ListServers()
}

func (m *Manager) DeleteServer(id string) error {
	srv, err := m.Store.GetServerByID(id)
	if err != nil || srv == nil {
		return fmt.Errorf("server not found in DB")
	}

	folderName := srv.FolderName
	if folderName == "" {
		folderName = id
		if _, err := os.Stat(filepath.Join(m.ServersPath, folderName)); os.IsNotExist(err) {
			folderName = sanitizeFolderName(srv.Name)
		}
	}

	serverDir := filepath.Join(m.ServersPath, folderName)

	if err := os.RemoveAll(serverDir); err != nil {
		return fmt.Errorf("error deleting server files: %w", err)
	}

	if err := m.Store.DeleteServer(id); err != nil {
		return fmt.Errorf("error deleting server from database: %w", err)
	}

	return nil
}

func (m *Manager) SaveServerIcon(id string, img image.Image) error {
	srv, err := m.GetServer(id)
	if err != nil {
		return err
	}
	if srv == nil {
		return fmt.Errorf("server not found")
	}

	folderName := srv.FolderName
	if folderName == "" {
		folderName = id
	}
	serverDir := filepath.Join(m.ServersPath, folderName)

	if _, err := os.Stat(serverDir); os.IsNotExist(err) {
		return fmt.Errorf("server directory not found")
	}

	dst := image.NewRGBA(image.Rect(0, 0, 64, 64))
	draw.BiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)

	outFile, err := os.Create(filepath.Join(serverDir, "server-icon.png"))
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	if err := png.Encode(outFile, dst); err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}
	return nil
}

func (m *Manager) GetServerIconPath(id string) (string, error) {
	srv, err := m.GetServer(id)
	if err != nil {
		return "", err
	}
	if srv == nil {
		return "", fmt.Errorf("server not found")
	}

	folderName := srv.FolderName
	if folderName == "" {
		folderName = id
	}

	iconPath := filepath.Join(m.ServersPath, folderName, "server-icon.png")
	if _, err := os.Stat(iconPath); os.IsNotExist(err) {
		return "", fmt.Errorf("icon not found")
	}
	return iconPath, nil
}
