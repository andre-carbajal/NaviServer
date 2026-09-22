package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"naviserver/internal/domain"
	"naviserver/internal/upload"
)

type UploadHandler struct {
	*BaseHandler
	UploadManager *upload.Manager
}

type createUploadRequest struct {
	ClientID      string      `json:"clientId"`
	Kind          upload.Kind `json:"kind"`
	ServerID      string      `json:"serverId"`
	DirectoryPath string      `json:"directoryPath"`
	RelativePath  string      `json:"relativePath"`
	Filename      string      `json:"filename"`
	ContentType   string      `json:"contentType"`
	TotalBytes    int64       `json:"totalBytes"`
}

func (h *UploadHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req createUploadRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		h.writeUploadError(w, upload.ErrInvalidRequest)
		return
	}

	userID, role, ok := uploadUser(r)
	if !ok {
		h.writeUploadError(w, upload.ErrForbidden)
		return
	}
	if err := h.authorize(req.Kind, req.ServerID, role, r); err != nil {
		h.writeUploadError(w, err)
		return
	}

	view, err := h.UploadManager.Create(upload.CreateInput{
		ClientID:      req.ClientID,
		UserID:        userID,
		Kind:          req.Kind,
		ServerID:      req.ServerID,
		DirectoryPath: req.DirectoryPath,
		RelativePath:  req.RelativePath,
		Filename:      req.Filename,
		ContentType:   req.ContentType,
		TotalBytes:    req.TotalBytes,
	})
	if err != nil {
		h.writeUploadError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, view)
}

func (h *UploadHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := uploadUser(r)
	if !ok {
		h.writeUploadError(w, upload.ErrForbidden)
		return
	}

	view, err := h.UploadManager.Get(r.PathValue("id"), userID)
	if err != nil {
		h.writeUploadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *UploadHandler) HandleChunk(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := uploadUser(r)
	if !ok {
		h.writeUploadError(w, upload.ErrForbidden)
		return
	}

	start, end, total, err := parseContentRange(r.Header.Get("Content-Range"))
	if err != nil {
		h.writeUploadError(w, err)
		return
	}
	chunkLength := end - start + 1
	if chunkLength > upload.ChunkSize {
		h.writeUploadError(w, upload.ErrChunkTooLarge)
		return
	}
	if r.ContentLength >= 0 && r.ContentLength != chunkLength {
		h.writeUploadError(w, upload.ErrInvalidChunk)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, chunkLength+1)
	view, err := h.UploadManager.AppendChunk(
		r.PathValue("id"),
		userID,
		start,
		end,
		total,
		io.LimitReader(r.Body, chunkLength+1),
	)
	if err != nil {
		h.writeUploadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *UploadHandler) HandleComplete(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := uploadUser(r)
	if !ok {
		h.writeUploadError(w, upload.ErrForbidden)
		return
	}

	view, err := h.UploadManager.Complete(r.PathValue("id"), userID)
	if err != nil {
		h.writeUploadError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, view)
}

func (h *UploadHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := uploadUser(r)
	if !ok {
		h.writeUploadError(w, upload.ErrForbidden)
		return
	}

	if err := h.UploadManager.Cancel(r.PathValue("id"), userID); err != nil {
		h.writeUploadError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *UploadHandler) authorize(kind upload.Kind, serverID, role string, r *http.Request) error {
	switch kind {
	case upload.KindServerFile:
		if serverID == "" || !h.checkPermission(r, serverID, func(p *domain.Permission) bool {
			return p.CanViewConsole
		}) {
			return upload.ErrForbidden
		}
	case upload.KindBackup:
		if serverID != "" {
			if !h.checkPermission(r, serverID, func(p *domain.Permission) bool {
				return p.CanControlPower
			}) {
				return upload.ErrForbidden
			}
		}
	case upload.KindServerIcon:
		if role != "admin" || serverID == "" {
			return upload.ErrForbidden
		}
	default:
		return upload.ErrInvalidTarget
	}
	return nil
}

func uploadUser(r *http.Request) (string, string, bool) {
	value := r.Context().Value(domain.UserContextKey)
	claims, ok := value.(map[string]string)
	if !ok {
		return "", "", false
	}
	userID := claims["id"]
	role := claims["role"]
	return userID, role, userID != ""
}

func parseContentRange(value string) (int64, int64, int64, error) {
	parts := strings.Fields(value)
	if len(parts) != 2 || parts[0] != "bytes" {
		return 0, 0, 0, upload.ErrInvalidChunk
	}
	rangeAndTotal := strings.Split(parts[1], "/")
	if len(rangeAndTotal) != 2 {
		return 0, 0, 0, upload.ErrInvalidChunk
	}
	bounds := strings.Split(rangeAndTotal[0], "-")
	if len(bounds) != 2 || rangeAndTotal[1] == "*" {
		return 0, 0, 0, upload.ErrInvalidChunk
	}
	start, err := strconv.ParseInt(bounds[0], 10, 64)
	if err != nil {
		return 0, 0, 0, upload.ErrInvalidChunk
	}
	end, err := strconv.ParseInt(bounds[1], 10, 64)
	if err != nil {
		return 0, 0, 0, upload.ErrInvalidChunk
	}
	total, err := strconv.ParseInt(rangeAndTotal[1], 10, 64)
	if err != nil {
		return 0, 0, 0, upload.ErrInvalidChunk
	}
	if start < 0 || end < start || total <= 0 || end >= total {
		return 0, 0, 0, upload.ErrInvalidChunk
	}
	return start, end, total, nil
}

func (h *UploadHandler) writeUploadError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, upload.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, upload.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, upload.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, upload.ErrChunkTooLarge):
		status = http.StatusRequestEntityTooLarge
	default:
		if !errors.Is(err, upload.ErrInvalidChunk) &&
			!errors.Is(err, upload.ErrInvalidRequest) &&
			!errors.Is(err, upload.ErrInvalidTarget) {
			status = http.StatusInternalServerError
		}
	}
	http.Error(w, fmt.Sprintf("%v", err), status)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
