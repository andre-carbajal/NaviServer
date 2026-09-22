package server

import (
	"fmt"
	"image"
	"image/png"
	"naviserver/internal/domain"
	"naviserver/internal/jvm"
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
	Java        JavaEnsurer
}

type JavaEnsurer interface {
	EnsureJava(version int) (string, error)
}

var (
	invalidFolderNameChars = regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
	validFolderName        = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
)

type serverFolderRename struct {
	from       string
	to         string
	folderName string
}

func NewManager(serversPath string, store *storage.GormStore, java JavaEnsurer) *Manager {
	return &Manager{
		ServersPath: serversPath,
		Store:       store,
		Java:        java,
	}
}

func (m *Manager) prepareLoaderOptions(loaderType string, downloader loader.ServerLoader, options loader.LoaderOptions, version string, configuredJava int) (loader.LoaderOptions, error) {
	if loaderType != "forge" && loaderType != "neoforge" {
		return options, nil
	}
	if m.Java == nil {
		return options, fmt.Errorf("managed Java runtime is required for %s installation", loaderType)
	}

	targetVersion := strings.TrimSpace(options.MCVersion)
	if targetVersion == "" {
		targetVersion = strings.TrimSpace(version)
	}
	if targetVersion == "" {
		versions, err := downloader.GetSupportedVersions(options)
		if err != nil {
			return options, fmt.Errorf("error getting %s Minecraft versions: %w", loaderType, err)
		}
		if len(versions) == 0 {
			return options, fmt.Errorf("no Minecraft versions available for %s", loaderType)
		}
		targetVersion = versions[0]
	}

	javaPath, err := m.Java.EnsureJava(jvm.ResolveJavaVersion(targetVersion, configuredJava))
	if err != nil {
		return options, fmt.Errorf("error preparing Java for %s: %w", targetVersion, err)
	}
	if !filepath.IsAbs(javaPath) {
		return options, fmt.Errorf("managed Java runtime returned a non-absolute path")
	}

	options.MCVersion = targetVersion
	options.JavaPath = javaPath
	return options, nil
}

func sanitizeFolderName(name string) string {
	name = strings.ReplaceAll(name, " ", "_")
	sanitized := invalidFolderNameChars.ReplaceAllString(name, "")
	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
	}
	return sanitized
}

func folderNameForRename(name string) (string, error) {
	name = strings.TrimSpace(name)
	folderName := strings.ReplaceAll(name, " ", "_")
	if name == "" || !validFolderName.MatchString(folderName) || strings.Contains(folderName, "..") || folderName == "." {
		return "", fmt.Errorf("invalid server name: use letters, numbers, spaces, '.', '_' or '-'")
	}

	folderName = sanitizeFolderName(name)
	if folderName == "" || folderName == "." || folderName == ".." {
		return "", fmt.Errorf("invalid server name: use letters, numbers, spaces, '.', '_' or '-'")
	}
	return folderName, nil
}

func (m *Manager) serverDirectory(srv *domain.Server) (string, error) {
	candidates := []string{srv.FolderName, srv.ID, sanitizeFolderName(srv.Name)}
	seen := make(map[string]struct{}, len(candidates))
	for _, folderName := range candidates {
		if folderName == "" {
			continue
		}
		if folderName == "." || folderName == ".." || strings.ContainsAny(folderName, `/\\`) || filepath.IsAbs(folderName) {
			continue
		}
		if _, ok := seen[folderName]; ok {
			continue
		}
		seen[folderName] = struct{}{}

		path := filepath.Join(m.ServersPath, folderName)
		info, err := os.Stat(path)
		if err == nil {
			if info.IsDir() {
				return path, nil
			}
			continue
		}
		if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("server directory is missing")
}

func (m *Manager) prepareServerFolderRename(srv *domain.Server, name string) (*serverFolderRename, error) {
	folderName, err := folderNameForRename(name)
	if err != nil {
		return nil, err
	}
	if srv.Status != "STOPPED" {
		return nil, fmt.Errorf("server must be stopped to rename")
	}

	from, err := m.serverDirectory(srv)
	if err != nil {
		return nil, err
	}
	to := filepath.Join(m.ServersPath, folderName)

	servers, err := m.Store.ListServers()
	if err != nil {
		return nil, err
	}
	for _, other := range servers {
		if other.ID == srv.ID {
			continue
		}
		otherFolder := other.FolderName
		if otherFolder == "" {
			otherFolder = other.ID
		}
		if strings.EqualFold(folderName, otherFolder) {
			return nil, fmt.Errorf("server folder already exists")
		}
	}

	fromInfo, err := os.Lstat(from)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(m.ServersPath)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !strings.EqualFold(entry.Name(), folderName) {
			continue
		}
		entryInfo, err := os.Lstat(filepath.Join(m.ServersPath, entry.Name()))
		if err != nil {
			return nil, err
		}
		if !os.SameFile(fromInfo, entryInfo) {
			return nil, fmt.Errorf("server folder already exists")
		}
	}

	return &serverFolderRename{from: from, to: to, folderName: folderName}, nil
}

func applyServerFolderRename(rename *serverFolderRename) (func() error, error) {
	if filepath.Clean(rename.from) == filepath.Clean(rename.to) {
		return func() error { return nil }, nil
	}

	fromInfo, err := os.Lstat(rename.from)
	if err != nil {
		return nil, err
	}
	toInfo, toErr := os.Lstat(rename.to)
	caseOnlyRename := toErr == nil && os.SameFile(fromInfo, toInfo)
	if toErr != nil && !os.IsNotExist(toErr) {
		return nil, toErr
	}
	if toErr == nil && !caseOnlyRename {
		return nil, fmt.Errorf("server folder already exists")
	}

	if caseOnlyRename {
		temporaryPath := filepath.Join(filepath.Dir(rename.from), ".rename-"+uuid.NewString())
		if err := os.Rename(rename.from, temporaryPath); err != nil {
			return nil, err
		}
		if err := os.Rename(temporaryPath, rename.to); err != nil {
			if rollbackErr := os.Rename(temporaryPath, rename.from); rollbackErr != nil {
				return nil, fmt.Errorf("%w (failed to restore server folder: %v)", err, rollbackErr)
			}
			return nil, err
		}
	} else if err := os.Rename(rename.from, rename.to); err != nil {
		return nil, err
	}

	return func() error { return os.Rename(rename.to, rename.from) }, nil
}

func (m *Manager) StartCreateServerJob(name, loaderType string, options loader.LoaderOptions, version string, ram int, progressChan chan<- domain.ProgressEvent) {
	go func() {
		defer close(progressChan)
		srv, err := m.CreateServer(name, loaderType, options, version, ram, progressChan)
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

func (m *Manager) CreateServer(name string, loaderType string, options loader.LoaderOptions, version string, ram int, progressChan chan<- domain.ProgressEvent) (*domain.Server, error) {
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
	options, err = m.prepareLoaderOptions(loaderType, downloader, options, version, 0)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(serverDir, 0755); err != nil {
		return nil, fmt.Errorf("filesystem error: %w", err)
	}

	resolvedVersion, err := downloader.Load(options, serverDir, progressChan)
	if err != nil {
		os.RemoveAll(serverDir)
		return nil, fmt.Errorf("download error: %w", err)
	}
	if strings.TrimSpace(resolvedVersion) != "" {
		version = resolvedVersion
	}
	if version == "" {
		version = options.MCVersion
	}
	if version == "" {
		version = "latest"
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

func (m *Manager) UpdateServer(id string, name *string, ram *int, customArgs *string, javaVersion *int) error {
	if name == nil {
		return m.Store.UpdateServer(id, nil, ram, customArgs, javaVersion, nil)
	}

	srv, err := m.GetServer(id)
	if err != nil {
		return err
	}
	if srv == nil {
		return fmt.Errorf("server not found")
	}

	normalizedName := strings.TrimSpace(*name)
	if normalizedName == "" {
		return fmt.Errorf("name is required")
	}
	name = &normalizedName
	if normalizedName == srv.Name {
		return m.Store.UpdateServer(id, name, ram, customArgs, javaVersion, nil)
	}

	rename, err := m.prepareServerFolderRename(srv, normalizedName)
	if err != nil {
		return err
	}
	rollback, err := applyServerFolderRename(rename)
	if err != nil {
		return fmt.Errorf("failed to rename server folder: %w", err)
	}
	if err := m.Store.UpdateServer(id, name, ram, customArgs, javaVersion, &rename.folderName); err != nil {
		if rollbackErr := rollback(); rollbackErr != nil {
			return fmt.Errorf("%w (failed to restore server folder: %v)", err, rollbackErr)
		}
		return err
	}
	return nil
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
