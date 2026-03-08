package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"

	"naviger/internal/domain"
)

type FilesHandler struct {
	*BaseHandler
}

func (h *FilesHandler) HandleListFiles(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	if !h.checkPermission(r, id, func(p *domain.Permission) bool { return p.CanViewConsole }) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	files, err := h.Manager.ListFiles(id, path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func (h *FilesHandler) HandleGetFileContent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Missing path", http.StatusBadRequest)
		return
	}

	if !h.checkPermission(r, id, func(p *domain.Permission) bool { return p.CanViewConsole }) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	content, err := h.Manager.ReadFile(id, path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(content)
}

func (h *FilesHandler) HandleSaveFileContent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Missing path", http.StatusBadRequest)
		return
	}

	if !h.checkPermission(r, id, func(p *domain.Permission) bool { return p.CanViewConsole }) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	content, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}

	if err := h.Manager.WriteFile(id, path, content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *FilesHandler) HandleCreateDirectory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if !h.checkPermission(r, id, func(p *domain.Permission) bool { return p.CanViewConsole }) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.Manager.CreateDirectory(id, req.Path); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *FilesHandler) HandleDeleteFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Missing path", http.StatusBadRequest)
		return
	}

	if !h.checkPermission(r, id, func(p *domain.Permission) bool { return p.CanViewConsole }) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.Manager.DeleteFile(id, path); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *FilesHandler) HandleDownloadFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Missing path", http.StatusBadRequest)
		return
	}

	if !h.checkPermission(r, id, func(p *domain.Permission) bool { return p.CanViewConsole }) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	fileReader, err := h.Manager.DownloadFile(id, path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer fileReader.Close()

	_, filename := filepath.Split(path)
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", "application/octet-stream")

	io.Copy(w, fileReader)
}

func (h *FilesHandler) HandleUploadFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		dirPath = "/"
	}

	if !h.checkPermission(r, id, func(p *domain.Permission) bool { return p.CanViewConsole }) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	targetPath := filepath.Join(dirPath, header.Filename)
	relativePath := r.URL.Query().Get("relative_path")
	if relativePath != "" {
		targetPath = filepath.Join(dirPath, relativePath)
	}

	if err := h.Manager.UploadFile(id, targetPath, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
