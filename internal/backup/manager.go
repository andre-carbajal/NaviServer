package backup

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"naviger/internal/domain"
	"naviger/internal/server"
	"naviger/internal/storage"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Manager struct {
	ServersPath string
	BackupsPath string
	Store       *storage.GormStore

	activeBackups   map[string]context.CancelFunc
	activeBackupsMu sync.Mutex
}

func NewManager(serversPath, backupsPath string, store *storage.GormStore) *Manager {
	return &Manager{
		ServersPath:   serversPath,
		BackupsPath:   backupsPath,
		Store:         store,
		activeBackups: make(map[string]context.CancelFunc),
	}
}

type BackupInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func (m *Manager) DeleteBackup(name string) error {
	if strings.Contains(name, "..") {
		return fmt.Errorf("invalid backup name")
	}
	backupPath := filepath.Join(m.BackupsPath, name)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup not found")
	}
	return os.Remove(backupPath)
}

func (m *Manager) GetBackupFilePath(name string) (string, error) {
	if strings.Contains(name, "..") {
		return "", fmt.Errorf("invalid backup name")
	}
	backupPath := filepath.Join(m.BackupsPath, name)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return "", fmt.Errorf("backup not found")
	}
	return backupPath, nil
}

func (m *Manager) ListAllBackups() ([]BackupInfo, error) {
	files, err := os.ReadDir(m.BackupsPath)
	if err != nil {
		return nil, fmt.Errorf("could not read backups directory: %w", err)
	}

	var backups []BackupInfo
	for _, file := range files {
		if file.IsDir() || strings.HasSuffix(file.Name(), ".temp") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}
		backups = append(backups, BackupInfo{
			Name: file.Name(),
			Size: info.Size(),
		})
	}

	return backups, nil
}

func (m *Manager) ListBackups(serverID string) ([]BackupInfo, error) {
	srv, err := m.Store.GetServerByID(serverID)
	if err != nil {
		return nil, fmt.Errorf("could not get server info: %w", err)
	}
	if srv == nil {
		return nil, fmt.Errorf("server with ID '%s' not found in database", serverID)
	}

	safeName := sanitizeFileName(srv.Name)

	files, err := os.ReadDir(m.BackupsPath)
	if err != nil {
		return nil, fmt.Errorf("could not read backups directory: %w", err)
	}

	var backups []BackupInfo
	for _, file := range files {
		if file.IsDir() || strings.HasSuffix(file.Name(), ".temp") {
			continue
		}

		if strings.HasPrefix(file.Name(), safeName) {
			info, err := file.Info()
			if err != nil {
				continue
			}
			backups = append(backups, BackupInfo{
				Name: file.Name(),
				Size: info.Size(),
			})
		}
	}

	return backups, nil
}

func (m *Manager) StartBackupJob(serverID, backupName, requestID string, progressChan chan<- domain.ProgressEvent) {
	ctx, cancel := context.WithCancel(context.Background())

	m.activeBackupsMu.Lock()
	m.activeBackups[requestID] = cancel
	m.activeBackupsMu.Unlock()

	go func() {
		defer close(progressChan)
		defer func() {
			m.activeBackupsMu.Lock()
			delete(m.activeBackups, requestID)
			m.activeBackupsMu.Unlock()
		}()

		_, err := m.CreateBackup(ctx, serverID, backupName, progressChan)
		if err != nil {
			event := domain.ProgressEvent{
				ServerID: serverID,
				Message:  fmt.Sprintf("Error: %v", err),
				Progress: -1,
			}
			progressChan <- event
			return
		}

		event := domain.ProgressEvent{
			ServerID: serverID,
			Message:  "Backup created successfully",
			Progress: 100,
		}
		progressChan <- event
	}()
}

func (m *Manager) CancelBackup(requestID string) {
	m.activeBackupsMu.Lock()
	cancel, ok := m.activeBackups[requestID]
	if ok {
		delete(m.activeBackups, requestID)
	}
	m.activeBackupsMu.Unlock()

	if ok {
		cancel()
	}
}

func (m *Manager) CreateBackup(ctx context.Context, serverID string, backupName string, progressChan chan<- domain.ProgressEvent) (string, error) {
	srv, err := m.Store.GetServerByID(serverID)
	if err != nil {
		return "", fmt.Errorf("could not get server info: %w", err)
	}
	if srv == nil {
		return "", fmt.Errorf("server with ID '%s' not found in database", serverID)
	}

	folderName := srv.FolderName
	if folderName == "" {
		folderName = srv.ID
	}
	serverDir := filepath.Join(m.ServersPath, folderName)

	if _, err := os.Stat(serverDir); os.IsNotExist(err) {
		return "", fmt.Errorf("server directory for ID '%s' does not exist", serverID)
	}

	if backupName == "" {
		backupName = srv.Name
	}

	safeName := sanitizeFileName(backupName)
	timestamp := time.Now().Format("20060102-150405")
	backupFileName := fmt.Sprintf("%s-%s.zip", safeName, timestamp)
	backupFilePath := filepath.Join(m.BackupsPath, backupFileName)
	tempBackupFilePath := backupFilePath + ".temp"

	if err := os.MkdirAll(m.BackupsPath, 0755); err != nil {
		return "", fmt.Errorf("could not create backups directory: %w", err)
	}

	var totalSize int64
	filepath.Walk(serverDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	backupFile, err := os.Create(tempBackupFilePath)
	if err != nil {
		return "", fmt.Errorf("could not create backup file: %w", err)
	}

	zipWriter := zip.NewWriter(backupFile)

	var processedSize int64
	var lastProgress int

	err = filepath.Walk(serverDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		relPath, err := filepath.Rel(serverDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
			if err != nil {
				return err
			}

			processedSize += info.Size()
			if totalSize > 0 && progressChan != nil {
				percentage := (float64(processedSize) / float64(totalSize)) * 100
				progressInt := int(percentage)

				if progressInt > lastProgress {
					lastProgress = progressInt
					progressChan <- domain.ProgressEvent{
						Message:      fmt.Sprintf("Backing up... %d%%", progressInt),
						Progress:     percentage,
						CurrentBytes: processedSize,
						TotalBytes:   totalSize,
					}
				}
			}
		}
		return err
	})

	zipErr := zipWriter.Close()
	fileErr := backupFile.Close()

	if err != nil || zipErr != nil || fileErr != nil {
		os.Remove(tempBackupFilePath)
		if err != nil {
			return "", fmt.Errorf("error creating backup: %w", err)
		}
		return "", fmt.Errorf("error closing files: %v, %v", zipErr, fileErr)
	}

	if err := os.Rename(tempBackupFilePath, backupFilePath); err != nil {
		return "", fmt.Errorf("error renaming temp file: %w", err)
	}

	return backupFilePath, nil
}

func (m *Manager) RestoreBackup(backupName string, targetServerID string, newServerName string, newServerRAM int, newServerLoader, newServerVersion string) error {
	backupPath := filepath.Join(m.BackupsPath, backupName)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup not found")
	}

	var targetDir string
	var targetPort int

	if targetServerID != "" {
		srv, err := m.Store.GetServerByID(targetServerID)
		if err != nil {
			return err
		}
		if srv == nil {
			return fmt.Errorf("server not found")
		}
		if srv.Status != "STOPPED" {
			return fmt.Errorf("server must be stopped to restore backup")
		}

		folderName := srv.FolderName
		if folderName == "" {
			folderName = srv.ID
		}
		targetDir = filepath.Join(m.ServersPath, folderName)
		targetPort = srv.Port

		files, err := os.ReadDir(targetDir)
		if err != nil {
			return err
		}
		for _, file := range files {
			os.RemoveAll(filepath.Join(targetDir, file.Name()))
		}

	} else {
		if newServerName == "" {
			return fmt.Errorf("server name is required for new server")
		}

		id := uuid.New().String()

		// Sanitize folder name for new server
		folderName := sanitizeFileName(newServerName)
		targetDir = filepath.Join(m.ServersPath, folderName)

		// Check for collision and append ID if needed
		if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
			folderName = fmt.Sprintf("%s-%s", folderName, id[:8])
			targetDir = filepath.Join(m.ServersPath, folderName)
		}

		port, err := server.AllocatePort(m.Store)
		if err != nil {
			return err
		}
		targetPort = port

		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}

		newServer := &domain.Server{
			ID:         id,
			Name:       newServerName,
			FolderName: folderName,
			Version:    newServerVersion,
			Loader:     newServerLoader,
			Port:       targetPort,
			RAM:        newServerRAM,
			Status:     "STOPPED",
			CreatedAt:  time.Now(),
		}

		if err := m.Store.SaveServer(newServer); err != nil {
			os.RemoveAll(targetDir)
			return err
		}
	}

	if err := unzip(backupPath, targetDir); err != nil {
		return fmt.Errorf("failed to unzip backup: %w", err)
	}

	if err := server.UpdateServerProperties(targetDir, targetPort); err != nil {
		return fmt.Errorf("failed to update server properties: %w", err)
	}

	return nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: illegal file path", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func sanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, " ", "_")
	reg := regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
	sanitized := reg.ReplaceAllString(name, "")
	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
	}
	return sanitized
}
