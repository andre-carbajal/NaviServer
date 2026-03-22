package server

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type FileEntry struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	IsDirectory  bool      `json:"isDirectory"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
}

func (m *Manager) sanitizePath(serverID, requestPath string) (string, error) {
	srv, err := m.GetServer(serverID)
	if err != nil {
		return "", err
	}
	if srv == nil {
		return "", fmt.Errorf("server not found")
	}

	folderName := srv.FolderName
	if folderName == "" {
		folderName = serverID
	}
	serverRoot := filepath.Join(m.ServersPath, folderName)

	cleanRequestPath := filepath.Clean(requestPath)

	cleanRequestPath = strings.TrimPrefix(cleanRequestPath, "/")
	cleanRequestPath = strings.TrimPrefix(cleanRequestPath, "\\")

	fullPath := filepath.Join(serverRoot, cleanRequestPath)

	if !strings.HasPrefix(fullPath, serverRoot+string(os.PathSeparator)) && fullPath != serverRoot {
		return "", fmt.Errorf("access denied: path outside server directory")
	}

	return fullPath, nil
}

func (m *Manager) ListFiles(serverID, requestPath string) ([]FileEntry, error) {
	fullPath, err := m.sanitizePath(serverID, requestPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	files := make([]FileEntry, 0)
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		relPath := filepath.Join(requestPath, entry.Name())
		relPath = strings.ReplaceAll(relPath, "\\", "/")
		if !strings.HasPrefix(relPath, "/") {
			relPath = "/" + relPath
		}

		files = append(files, FileEntry{
			Name:         entry.Name(),
			Path:         relPath,
			IsDirectory:  entry.IsDir(),
			Size:         info.Size(),
			LastModified: info.ModTime(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDirectory != files[j].IsDirectory {
			return files[i].IsDirectory
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	return files, nil
}

func (m *Manager) ReadFile(serverID, requestPath string) ([]byte, error) {
	fullPath, err := m.sanitizePath(serverID, requestPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("cannot read directory")
	}
	if info.Size() > 10*1024*1024 {
		return nil, fmt.Errorf("file too large to read (max 10MB)")
	}

	return os.ReadFile(fullPath)
}

func (m *Manager) WriteFile(serverID, requestPath string, content []byte) error {
	fullPath, err := m.sanitizePath(serverID, requestPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	return os.WriteFile(fullPath, content, 0644)
}

func (m *Manager) CreateDirectory(serverID, requestPath string) error {
	fullPath, err := m.sanitizePath(serverID, requestPath)
	if err != nil {
		return err
	}
	return os.MkdirAll(fullPath, 0755)
}

func (m *Manager) DeleteFile(serverID, requestPath string) error {
	fullPath, err := m.sanitizePath(serverID, requestPath)
	if err != nil {
		return err
	}
	return os.RemoveAll(fullPath)
}

func (m *Manager) RenameFile(serverID, oldPath, newPath string) error {
	fullOldPath, err := m.sanitizePath(serverID, oldPath)
	if err != nil {
		return err
	}
	fullNewPath, err := m.sanitizePath(serverID, newPath)
	if err != nil {
		return err
	}
	return os.Rename(fullOldPath, fullNewPath)
}

func (m *Manager) DownloadFile(serverID, requestPath string) (io.ReadCloser, error) {
	fullPath, err := m.sanitizePath(serverID, requestPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("cannot download directory")
	}

	return os.Open(fullPath)
}

func (m *Manager) DownloadDirectory(serverID, requestPath string, w io.Writer) error {
	fullPath, err := m.sanitizePath(serverID, requestPath)
	if err != nil {
		return err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory")
	}

	archive := zip.NewWriter(w)
	defer archive.Close()

	return filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(fullPath, path)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		zipFile, err := archive.Create(relPath)
		if err != nil {
			return err
		}

		_, err = io.Copy(zipFile, f)
		return err
	})
}

func (m *Manager) UploadFile(serverID, requestPath string, content io.Reader) error {
	fullPath, err := m.sanitizePath(serverID, requestPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, content)
	return err
}
