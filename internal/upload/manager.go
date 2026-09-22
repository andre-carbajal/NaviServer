package upload

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"naviserver/internal/backup"
	"naviserver/internal/server"
	"naviserver/internal/ws"

	"github.com/google/uuid"
)

const ChunkSize int64 = 5 << 20

type Kind string

const (
	KindServerFile Kind = "server-file"
	KindBackup     Kind = "backup"
	KindServerIcon Kind = "server-icon"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusUploading  Status = "uploading"
	StatusReady      Status = "ready"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusError      Status = "error"
	StatusCancelled  Status = "cancelled"
)

var (
	ErrNotFound       = errors.New("upload not found")
	ErrForbidden      = errors.New("upload forbidden")
	ErrConflict       = errors.New("upload state conflict")
	ErrInvalidChunk   = errors.New("invalid upload chunk")
	ErrChunkTooLarge  = errors.New("upload chunk too large")
	ErrInvalidTarget  = errors.New("invalid upload target")
	ErrInvalidRequest = errors.New("invalid upload request")
)

type CreateInput struct {
	ClientID      string
	UserID        string
	Kind          Kind
	ServerID      string
	DirectoryPath string
	RelativePath  string
	Filename      string
	ContentType   string
	TotalBytes    int64
}

type View struct {
	ID            string `json:"id"`
	ClientID      string `json:"clientId"`
	Kind          Kind   `json:"kind"`
	ServerID      string `json:"serverId,omitempty"`
	Filename      string `json:"filename"`
	Status        Status `json:"status"`
	ReceivedBytes int64  `json:"receivedBytes"`
	TotalBytes    int64  `json:"totalBytes"`
	Progress      int    `json:"progress"`
	Message       string `json:"message,omitempty"`
	Error         string `json:"error,omitempty"`
}

type session struct {
	mu sync.Mutex
	View
	UserID        string
	DirectoryPath string
	RelativePath  string
	ContentType   string
	TempPath      string
}

type Manager struct {
	serverManager *server.Manager
	backupManager *backup.Manager
	hubManager    *ws.HubManager
	tempDir       string

	mu       sync.RWMutex
	sessions map[string]*session
	clients  map[string]string
}

func NewManager(
	serverManager *server.Manager,
	backupManager *backup.Manager,
	hubManager *ws.HubManager,
) (*Manager, error) {
	tempDir, err := os.MkdirTemp("", "naviserver-uploads-")
	if err != nil {
		return nil, fmt.Errorf("create upload temp directory: %w", err)
	}

	return &Manager{
		serverManager: serverManager,
		backupManager: backupManager,
		hubManager:    hubManager,
		tempDir:       tempDir,
		sessions:      make(map[string]*session),
		clients:       make(map[string]string),
	}, nil
}

func (m *Manager) Close() error {
	if m == nil || m.tempDir == "" {
		return nil
	}
	return os.RemoveAll(m.tempDir)
}

func (m *Manager) Create(input CreateInput) (View, error) {
	if err := validateCreateInput(input); err != nil {
		return View{}, err
	}

	clientKey := input.UserID + "\x00" + input.ClientID
	m.mu.Lock()
	if existingID, ok := m.clients[clientKey]; ok {
		if existing, exists := m.sessions[existingID]; exists {
			existing.mu.Lock()
			view := existing.viewLocked()
			existing.mu.Unlock()
			m.mu.Unlock()
			return view, nil
		}
		delete(m.clients, clientKey)
	}
	m.mu.Unlock()

	tempFile, err := os.CreateTemp(m.tempDir, "upload-*")
	if err != nil {
		return View{}, fmt.Errorf("create upload temp file: %w", err)
	}
	tempPath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return View{}, fmt.Errorf("close upload temp file: %w", err)
	}

	id := uuid.NewString()
	s := &session{
		View: View{
			ID:         id,
			ClientID:   input.ClientID,
			Kind:       input.Kind,
			ServerID:   input.ServerID,
			Filename:   input.Filename,
			Status:     StatusPending,
			TotalBytes: input.TotalBytes,
		},
		UserID:        input.UserID,
		DirectoryPath: input.DirectoryPath,
		RelativePath:  input.RelativePath,
		ContentType:   input.ContentType,
		TempPath:      tempPath,
	}

	m.mu.Lock()
	m.sessions[id] = s
	m.clients[clientKey] = id
	m.mu.Unlock()
	m.notify(s)

	return s.View, nil
}

func (m *Manager) Get(id, userID string) (View, error) {
	s, err := m.getSession(id, userID)
	if err != nil {
		return View{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.viewLocked(), nil
}

func (m *Manager) AppendChunk(
	id, userID string,
	start, end, total int64,
	body io.Reader,
) (View, error) {
	if start < 0 || end < start || total < 0 || end >= total {
		return View{}, ErrInvalidChunk
	}
	chunkLength := end - start + 1
	if chunkLength <= 0 || chunkLength > ChunkSize {
		return View{}, ErrChunkTooLarge
	}

	s, err := m.getSession(id, userID)
	if err != nil {
		return View{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status != StatusPending && s.Status != StatusUploading {
		return s.viewLocked(), ErrConflict
	}
	if total != s.TotalBytes || start != s.ReceivedBytes {
		return s.viewLocked(), ErrConflict
	}

	chunk := make([]byte, chunkLength)
	if _, err := io.ReadFull(body, chunk); err != nil {
		return s.viewLocked(), fmt.Errorf("%w: %v", ErrInvalidChunk, err)
	}
	var extra [1]byte
	if n, err := body.Read(extra[:]); n > 0 || (err != nil && !errors.Is(err, io.EOF)) {
		return s.viewLocked(), ErrInvalidChunk
	}

	file, err := os.OpenFile(s.TempPath, os.O_WRONLY, 0600)
	if err != nil {
		return s.viewLocked(), fmt.Errorf("open upload temp file: %w", err)
	}
	_, writeErr := file.WriteAt(chunk, start)
	closeErr := file.Close()
	if writeErr != nil {
		return s.viewLocked(), fmt.Errorf("write upload chunk: %w", writeErr)
	}
	if closeErr != nil {
		return s.viewLocked(), fmt.Errorf("close upload temp file: %w", closeErr)
	}

	s.ReceivedBytes = end + 1
	s.Status = StatusUploading
	if s.ReceivedBytes == s.TotalBytes {
		s.Status = StatusReady
	}
	s.Progress = progress(s.ReceivedBytes, s.TotalBytes)
	view := s.viewLocked()
	m.notifyLocked(s)
	return view, nil
}

func (m *Manager) Complete(id, userID string) (View, error) {
	s, err := m.getSession(id, userID)
	if err != nil {
		return View{}, err
	}

	s.mu.Lock()
	if s.Status == StatusCompleted || s.Status == StatusError {
		view := s.viewLocked()
		s.mu.Unlock()
		return view, nil
	}
	if s.Status == StatusProcessing {
		view := s.viewLocked()
		s.mu.Unlock()
		return view, nil
	}
	if s.ReceivedBytes != s.TotalBytes {
		view := s.viewLocked()
		s.mu.Unlock()
		return view, ErrConflict
	}
	s.Status = StatusProcessing
	s.Message = "Processing upload..."
	view := s.viewLocked()
	s.mu.Unlock()
	m.notify(s)

	go m.finalize(s)
	return view, nil
}

func (m *Manager) Cancel(id, userID string) error {
	s, err := m.getSession(id, userID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	if s.Status == StatusProcessing {
		s.mu.Unlock()
		return ErrConflict
	}
	s.Status = StatusCancelled
	s.Message = "Upload cancelled"
	s.mu.Unlock()
	m.notify(s)
	m.removeSession(id)
	if err := os.Remove(s.TempPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func validateCreateInput(input CreateInput) error {
	if input.UserID == "" || input.ClientID == "" || input.Filename == "" || input.TotalBytes < 0 {
		return ErrInvalidRequest
	}
	if filepath.Base(input.Filename) != input.Filename || strings.ContainsAny(input.Filename, `/\\`) {
		return ErrInvalidTarget
	}
	if !validRelativePath(input.RelativePath) || !validServerPath(input.DirectoryPath) {
		return ErrInvalidTarget
	}
	switch input.Kind {
	case KindServerFile:
		if input.ServerID == "" {
			return ErrInvalidTarget
		}
	case KindBackup:
		ext := strings.ToLower(filepath.Ext(input.Filename))
		if ext != ".zip" && ext != ".rar" {
			return ErrInvalidTarget
		}
	case KindServerIcon:
		if input.ServerID == "" {
			return ErrInvalidTarget
		}
	default:
		return ErrInvalidTarget
	}
	return nil
}

func validServerPath(path string) bool {
	if path == "" {
		return true
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	return strings.HasPrefix(path, "/") && clean != ".." && !strings.HasPrefix(clean, ".."+string(os.PathSeparator))
}

func validRelativePath(path string) bool {
	if path == "" {
		return true
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	return !filepath.IsAbs(clean) && clean != ".." && !strings.HasPrefix(clean, ".."+string(os.PathSeparator))
}

func (m *Manager) getSession(id, userID string) (*session, error) {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	if s.UserID != userID {
		return nil, ErrForbidden
	}
	return s, nil
}

func (s *session) viewLocked() View {
	view := s.View
	return view
}

func progress(received, total int64) int {
	if total <= 0 {
		return 0
	}
	return int((received * 100) / total)
}

func (m *Manager) finalize(s *session) {
	err := m.finalizeTarget(s)

	s.mu.Lock()
	if err != nil {
		s.Status = StatusError
		s.Error = err.Error()
		s.Message = "Upload failed"
	} else {
		s.Status = StatusCompleted
		s.Progress = 100
		s.Message = "Upload completed"
	}
	s.mu.Unlock()
	m.notify(s)
	_ = os.Remove(s.TempPath)
}

func (m *Manager) finalizeTarget(s *session) error {
	file, err := os.Open(s.TempPath)
	if err != nil {
		return fmt.Errorf("open completed upload: %w", err)
	}
	defer file.Close()

	s.mu.Lock()
	kind := s.Kind
	serverID := s.ServerID
	directoryPath := s.DirectoryPath
	relativePath := s.RelativePath
	filename := s.Filename
	userID := s.UserID
	s.mu.Unlock()

	switch kind {
	case KindServerFile:
		targetPath := filepath.Join(directoryPath, filename)
		if relativePath != "" {
			targetPath = filepath.Join(directoryPath, filepath.FromSlash(relativePath))
		}
		return m.serverManager.UploadFile(serverID, targetPath, file)
	case KindBackup:
		return m.backupManager.UploadBackup(file, filename, serverID, userID)
	case KindServerIcon:
		img, _, err := image.Decode(file)
		if err != nil {
			return fmt.Errorf("invalid image format: %w", err)
		}
		return m.serverManager.SaveServerIcon(serverID, img)
	default:
		return ErrInvalidTarget
	}
}

func (m *Manager) notify(s *session) {
	s.mu.Lock()
	view := s.viewLocked()
	s.mu.Unlock()
	m.notifyView(view)
}

func (m *Manager) notifyLocked(s *session) {
	m.notifyView(s.viewLocked())
}

func (m *Manager) notifyView(view View) {
	if m.hubManager == nil {
		return
	}
	message, err := json.Marshal(view)
	if err != nil {
		return
	}
	m.hubManager.GetHub(view.ID).Broadcast(message)
}

func (m *Manager) removeSession(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return
	}
	delete(m.sessions, id)
	delete(m.clients, s.UserID+"\x00"+s.ClientID)
}
