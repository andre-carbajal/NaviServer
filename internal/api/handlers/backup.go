package handlers

import (
	"encoding/json"
	"fmt"
	"naviserver/internal/domain"
	"net/http"
)

type BackupHandler struct {
	*BaseHandler
}

func (h *BackupHandler) HandleUploadBackup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("backup")
	if err != nil {
		http.Error(w, "Error Retrieving the File", http.StatusBadRequest)
		return
	}
	defer file.Close()

	serverID := r.FormValue("serverId")

	if serverID != "" {
		if !h.checkPermission(r, serverID, func(p *domain.Permission) bool {
			return p.CanControlPower
		}) {
			http.Error(w, "Forbidden: No permission to manage backups for this server", http.StatusForbidden)
			return
		}
	}

	var userID string
	userCtx := r.Context().Value(domain.UserContextKey)
	if userCtx != nil {
		claims := userCtx.(map[string]string)
		userID = claims["id"]
	}

	if err := h.BackupManager.UploadBackup(file, handler.Filename, serverID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *BackupHandler) HandleDownloadBackup(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "Missing backup name", http.StatusBadRequest)
		return
	}

	userCtx := r.Context().Value(domain.UserContextKey)
	if userCtx != nil {
		claims := userCtx.(map[string]string)
		role := claims["role"]
		userID := claims["id"]

		if role != "admin" {
			backup, err := h.Store.GetBackupByName(name)
			if err != nil || backup == nil {
				http.Error(w, "Backup not found", http.StatusNotFound)
				return
			}

			if backup.ServerID == "" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			perms, err := h.Store.GetPermissions(userID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			allowed := false
			for _, p := range perms {
				if p.ServerID == backup.ServerID && p.CanControlPower {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, "Forbidden: No permission to manage backups for this server", http.StatusForbidden)
				return
			}
		}
	}

	path, err := h.BackupManager.GetBackupFilePath(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))
	http.ServeFile(w, r, path)
}

func (h *BackupHandler) HandleDeleteBackup(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "Missing backup name", http.StatusBadRequest)
		return
	}

	userCtx := r.Context().Value(domain.UserContextKey)
	if userCtx != nil {
		claims := userCtx.(map[string]string)
		role := claims["role"]
		userID := claims["id"]

		if role != "admin" {
			backup, err := h.Store.GetBackupByName(name)
			if err != nil || backup == nil {
				http.Error(w, "Backup not found", http.StatusNotFound)
				return
			}

			if backup.ServerID == "" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			perms, err := h.Store.GetPermissions(userID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			allowed := false
			for _, p := range perms {
				if p.ServerID == backup.ServerID && p.CanControlPower {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, "Forbidden: No permission to manage backups for this server", http.StatusForbidden)
				return
			}
		}
	}

	if err := h.BackupManager.DeleteBackup(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BackupHandler) HandleRestoreBackup(w http.ResponseWriter, r *http.Request) {
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

	userCtx := r.Context().Value(domain.UserContextKey)
	if userCtx != nil {
		claims := userCtx.(map[string]string)
		role := claims["role"]
		userID := claims["id"]

		if role != "admin" {
			backup, err := h.Store.GetBackupByName(name)
			if err != nil || backup == nil {
				http.Error(w, "Backup not found", http.StatusNotFound)
				return
			}

			if backup.ServerID == "" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			perms, err := h.Store.GetPermissions(userID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			allowed := false
			for _, p := range perms {
				if p.ServerID == backup.ServerID && p.CanControlPower {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, "Forbidden: No permission for this backup", http.StatusForbidden)
				return
			}

			if req.TargetServerID != "" {
				targetAllowed := false
				for _, p := range perms {
					if p.ServerID == req.TargetServerID && p.CanControlPower {
						targetAllowed = true
						break
					}
				}
				if !targetAllowed {
					http.Error(w, "Forbidden: No permission for target server", http.StatusForbidden)
					return
				}
			} else {
				http.Error(w, "Only administrators can create new servers from backups", http.StatusForbidden)
				return
			}
		}
	}

	if err := h.BackupManager.RestoreBackup(name, req.TargetServerID, req.NewServerName, req.NewServerRAM, req.NewServerLoader, req.NewServerVersion); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "restored"}`))
}

func (h *BackupHandler) HandleListBackupsByServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	userCtx := r.Context().Value(domain.UserContextKey)
	userID := ""
	role := ""
	if userCtx != nil {
		claims := userCtx.(map[string]string)
		role = claims["role"]
		userID = claims["id"]
	}

	backups, err := h.BackupManager.ListBackups(id, userID, role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backups)
}

func (h *BackupHandler) HandleListAllBackups(w http.ResponseWriter, r *http.Request) {
	userCtx := r.Context().Value(domain.UserContextKey)
	userID := ""
	role := ""
	if userCtx != nil {
		claims := userCtx.(map[string]string)
		role = claims["role"]
		userID = claims["id"]
	}

	backups, err := h.BackupManager.ListAllBackups(userID, role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backups)
}

func (h *BackupHandler) HandleCancelBackup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	h.BackupManager.CancelBackup(id)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "cancelled"}`))
}

func (h *BackupHandler) HandleUpdateBackup(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "Missing backup name", http.StatusBadRequest)
		return
	}

	var req struct {
		ServerID string `json:"serverId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.BackupManager.UpdateBackup(name, req.ServerID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "updated"}`))
}
