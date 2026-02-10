package handlers

import (
	"encoding/json"
	"net/http"

	"naviger/internal/backup"
	"naviger/internal/domain"
	"naviger/internal/ws"
)

type BackupHandler struct {
	BackupManager *backup.Manager
	HubManager    *ws.HubManager
}

func NewBackupHandler(backupManager *backup.Manager, hubManager *ws.HubManager) *BackupHandler {
	return &BackupHandler{
		BackupManager: backupManager,
		HubManager:    hubManager,
	}
}

func (h *BackupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "Missing backup name", http.StatusBadRequest)
		return
	}

	if err := h.BackupManager.DeleteBackup(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BackupHandler) Restore(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "Missing backup name", http.StatusBadRequest)
		return
	}

	var req struct {
		TargetServerID   string `json:"targetServerId"`
		NewServerName    string `json:"newServerName"`
		NewServerRAM     int    `json:"newServerRam"`
		NewServerLoader  string `json:"newServerLoader"`
		NewServerVersion string `json:"newServerVersion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.BackupManager.RestoreBackup(name, req.TargetServerID, req.NewServerName, req.NewServerRAM, req.NewServerLoader, req.NewServerVersion); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "restored"}`))
}

func (h *BackupHandler) ListByServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	backups, err := h.BackupManager.ListBackups(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backups)
}

func (h *BackupHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	backups, err := h.BackupManager.ListAllBackups()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backups)
}

func (h *BackupHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	h.BackupManager.CancelBackup(id)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "cancelled"}`))
}

func (h *BackupHandler) Create(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Name      string `json:"name,omitempty"`
		RequestID string `json:"requestId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	progressChan := make(chan domain.ProgressEvent)
	hubID := req.RequestID
	if hubID == "" {
		hubID = "backup-" + id
	}
	hub := h.HubManager.GetHub(hubID)

	go func() {
		for event := range progressChan {
			if event.ServerID == "" {
				event.ServerID = id
			}
			jsonBytes, _ := json.Marshal(event)
			hub.Broadcast(jsonBytes)
		}
	}()

	h.BackupManager.StartBackupJob(id, req.Name, req.RequestID, progressChan)

	response := map[string]string{
		"status": "creating",
		"id":     req.RequestID,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}
