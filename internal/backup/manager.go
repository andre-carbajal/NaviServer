package backup

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"naviger/internal/domain"
	"naviger/internal/server"
	"naviger/internal/storage"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/go-unarr"
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

type Info struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func (m *Manager) UploadBackup(file multipart.File, filename string, serverID string, userID string) error {
	if !strings.HasSuffix(filename, ".zip") && !strings.HasSuffix(filename, ".rar") {
		return fmt.Errorf("invalid file type, only .zip and .rar are supported")
	}

	if err := os.MkdirAll(m.BackupsPath, 0755); err != nil {
		return fmt.Errorf("could not create backups directory: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "backup-upload-")
	if err != nil {
		return fmt.Errorf("could not create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	tempFile, err := os.CreateTemp(tempDir, "backup-*.tmp")
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}

	_, err = io.Copy(tempFile, file)
	if err != nil {
		tempFile.Close()
		return fmt.Errorf("could not save uploaded file: %w", err)
	}
	tempFilePath := tempFile.Name()
	tempFile.Close()

	backupFileName := sanitizeFileName(filename)
	ext := filepath.Ext(backupFileName)
	if ext == "" {
		if strings.HasSuffix(filename, ".rar") {
			backupFileName += ".rar"
		} else {
			backupFileName += ".zip"
		}
	}
	backupFilePath := filepath.Join(m.BackupsPath, backupFileName)

	if strings.HasSuffix(strings.ToLower(filename), ".zip") {
		if err := m.processZipUpload(tempFilePath, backupFilePath); err != nil {
			return err
		}
	} else {
		if err := m.processArchiveUploadGeneric(tempFilePath, backupFilePath); err != nil {
			return err
		}
	}

	info, err := os.Stat(backupFilePath)
	if err != nil {
		return err
	}

	backup := &domain.Backup{
		ID:        uuid.New().String(),
		Name:      backupFileName,
		FileName:  backupFileName,
		ServerID:  serverID,
		Size:      info.Size(),
		CreatedAt: time.Now(),
		CreatedBy: userID,
	}

	return m.Store.SaveBackup(backup)
}

func (m *Manager) processZipUpload(tempFilePath, targetPath string) error {
	r, err := zip.OpenReader(tempFilePath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer r.Close()

	var root string
	files := r.File
	if len(files) == 0 {
		return fmt.Errorf("empty zip file")
	}

	var validFiles []*zip.File
	for _, f := range files {
		if strings.HasPrefix(f.Name, "__MACOSX") || strings.HasSuffix(f.Name, ".DS_Store") {
			continue
		}
		validFiles = append(validFiles, f)
	}

	if len(validFiles) > 0 {
		first := validFiles[0].Name
		parts := strings.Split(filepath.ToSlash(first), "/")
		if len(parts) > 1 {
			candidateRoot := parts[0] + "/"
			isRoot := true
			for _, f := range validFiles {
				if !strings.HasPrefix(filepath.ToSlash(f.Name), candidateRoot) {
					isRoot = false
					break
				}
			}
			if isRoot {
				root = candidateRoot
			}
		}
	}

	if root == "" {
		src, err := os.Open(tempFilePath)
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer dst.Close()

		_, err = io.Copy(dst, src)
		return err
	}

	outFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)
	defer w.Close()

	for _, f := range validFiles {
		rc, err := f.Open()
		if err != nil {
			return err
		}

		newName := strings.TrimPrefix(filepath.ToSlash(f.Name), root)
		if newName == "" || newName == "/" {
			rc.Close()
			continue
		}

		header := f.FileHeader
		header.Name = newName

		if f.FileInfo().IsDir() {
			if !strings.HasSuffix(header.Name, "/") {
				header.Name += "/"
			}
		} else {
			header.Method = zip.Deflate
		}

		target, err := w.CreateHeader(&header)
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(target, rc)
		rc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) processArchiveUploadGeneric(tempFilePath, targetPath string) error {
	archive, err := unarr.NewArchive(tempFilePath)
	if err != nil {
		return fmt.Errorf("could not open archive: %w", err)
	}
	defer archive.Close()

	extractDir := filepath.Dir(tempFilePath)
	extractPath := filepath.Join(extractDir, "extracted")
	os.MkdirAll(extractPath, 0755)

	files, err := archive.List()
	if err != nil {
		return fmt.Errorf("could not list archive contents: %w", err)
	}

	var rootDir string
	isSingleDir := true
	if len(files) > 0 {
		for _, f := range files {
			parts := strings.Split(filepath.ToSlash(f), "/")
			if len(parts) > 0 {
				if rootDir == "" {
					if len(parts) > 1 {
						rootDir = parts[0]
					} else {
						isSingleDir = false
						break
					}
				}
				if !strings.HasPrefix(filepath.ToSlash(f), rootDir) {
					isSingleDir = false
					break
				}
			}
		}
	} else {
		isSingleDir = false
	}

	finalExtractPath := extractPath
	for _, f := range files {
		cleanName := filepath.Clean(f)
		if strings.Contains(cleanName, "..") || filepath.IsAbs(cleanName) {
			return fmt.Errorf("malicious archive entry: %s", f)
		}
	}

	if _, err := archive.Extract(extractPath); err != nil {
		return fmt.Errorf("could not extract archive: %w", err)
	}

	sourceDir := finalExtractPath
	if isSingleDir && rootDir != "" {
		sourceDir = filepath.Join(finalExtractPath, rootDir)
	}

	if !strings.HasSuffix(targetPath, ".zip") {
		targetPath = strings.TrimSuffix(targetPath, filepath.Ext(targetPath)) + ".zip"
	}

	newZipFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("could not create new zip file: %w", err)
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == sourceDir {
			return nil
		}
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
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
			fileToZip, err := os.Open(path)
			if err != nil {
				return err
			}
			defer fileToZip.Close()
			_, err = io.Copy(writer, fileToZip)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *Manager) DeleteBackup(name string) error {
	if strings.Contains(name, "..") {
		return fmt.Errorf("invalid backup name")
	}
	backupPath := filepath.Join(m.BackupsPath, name)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		_ = m.Store.DeleteBackup(name)
		return fmt.Errorf("backup file not found")
	}

	if err := os.Remove(backupPath); err != nil {
		return err
	}

	return m.Store.DeleteBackup(name)
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

func (m *Manager) ListAllBackups(userID string, role string) ([]domain.Backup, error) {
	return m.Store.ListBackups("", userID, role)
}

func (m *Manager) ListBackups(serverID string, userID string, role string) ([]domain.Backup, error) {
	return m.Store.ListBackups(serverID, userID, role)
}

func (m *Manager) SyncBackups() error {
	if err := os.MkdirAll(m.BackupsPath, 0755); err != nil {
		return err
	}

	files, err := os.ReadDir(m.BackupsPath)
	if err != nil {
		return err
	}

	dbBackups, err := m.Store.ListAllBackups()
	if err != nil {
		return err
	}

	fileMap := make(map[string]bool)
	for _, file := range files {
		if !file.IsDir() && !strings.HasSuffix(file.Name(), ".temp") {
			fileMap[file.Name()] = true
		}
	}

	dbMap := make(map[string]bool)
	for _, b := range dbBackups {
		dbMap[b.FileName] = true

		// Cleanup logic: If DB record exists but file is gone, delete from DB
		if !fileMap[b.FileName] {
			if err := m.Store.DeleteBackup(b.Name); err != nil {
				log.Printf("Failed to remove ghost backup record %s: %v", b.Name, err)
			}
		}
	}

	servers, err := m.Store.ListServers()
	if err != nil {
		return err
	}

	for fileName := range fileMap {
		if dbMap[fileName] {
			continue
		}

		// Discovery logic for new files
		fInfo, err := os.Stat(filepath.Join(m.BackupsPath, fileName))
		if err != nil {
			continue
		}

		var serverID string
		for _, srv := range servers {
			safeName := sanitizeFileName(srv.Name)
			if strings.HasPrefix(fileName, safeName) {
				serverID = srv.ID
				break
			}
		}

		backup := &domain.Backup{
			ID:        uuid.New().String(),
			Name:      fileName,
			FileName:  fileName,
			ServerID:  serverID,
			Size:      fInfo.Size(),
			CreatedAt: fInfo.ModTime(),
			CreatedBy: "system",
		}

		if err := m.Store.SaveBackup(backup); err != nil {
			log.Printf("Failed to sync backup %s: %v", fileName, err)
		}
	}

	return nil
}

func (m *Manager) StartBackupJob(serverID, backupName, requestID, userID string, progressChan chan<- domain.ProgressEvent) {
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

		_, err := m.CreateBackup(ctx, serverID, backupName, userID, progressChan)
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

func (m *Manager) CreateBackup(ctx context.Context, serverID string, backupName string, userID string, progressChan chan<- domain.ProgressEvent) (string, error) {
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

	info, _ := os.Stat(backupFilePath)
	backup := &domain.Backup{
		ID:        uuid.New().String(),
		Name:      backupFileName,
		FileName:  backupFileName,
		ServerID:  serverID,
		Size:      info.Size(),
		CreatedAt: time.Now(),
		CreatedBy: userID,
	}
	m.Store.SaveBackup(backup)

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

		folderName := sanitizeFileName(newServerName)
		targetDir = filepath.Join(m.ServersPath, folderName)

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

	if err := unarchive(backupPath, targetDir); err != nil {
		return fmt.Errorf("failed to unarchive backup: %w", err)
	}

	if err := server.UpdateServerProperties(targetDir, targetPort); err != nil {
		return fmt.Errorf("failed to update server properties: %w", err)
	}

	return nil
}

func unarchive(src, dest string) error {
	archive, err := unarr.NewArchive(src)
	if err != nil {
		return err
	}
	defer archive.Close()

	destClean := filepath.Clean(dest)
	if err := os.MkdirAll(destClean, 0755); err != nil {
		return err
	}

	for {
		err := archive.Entry()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		name := archive.Name()
		target := filepath.Join(destClean, name)

		if !strings.HasPrefix(target, destClean+string(os.PathSeparator)) && target != destClean {
			return fmt.Errorf("illegal archive entry: %s", name)
		}

		if strings.HasSuffix(name, "/") {
			os.MkdirAll(target, 0755)
			continue
		}

		os.MkdirAll(filepath.Dir(target), 0755)
		f, err := os.Create(target)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, archive)
		f.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) UpdateBackup(name string, serverID string) error {
	return m.Store.UpdateBackup(name, serverID)
}

func sanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, " ", "_")
	// Allow most Unicode characters but remove path-illegal and potentially dangerous ones.
	// Illegal in Windows/Unix: / \ : * ? " < > |
	reg := regexp.MustCompile(`[\\/:*?"<>|]`)
	sanitized := reg.ReplaceAllString(name, "")

	// Truncate but avoid cutting in the middle of a multi-byte character
	if len(sanitized) > 100 {
		runes := []rune(sanitized)
		if len(runes) > 50 {
			sanitized = string(runes[:50])
		}
	}
	return sanitized
}
