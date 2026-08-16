package handlers

import (
	"encoding/json"
	"fmt"
	"image"
	// Register the JPEG decoder for image.Decode.
	_ "image/jpeg"
	// Register the PNG decoder for image.Decode.
	_ "image/png"
	"naviserver/internal/domain"
	"naviserver/internal/loader"
	"naviserver/internal/server"
	"net/http"
	"path/filepath"
	"strings"
)

type ServerHandler struct {
	*BaseHandler
}

func (h *ServerHandler) HandleListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := h.Manager.ListServers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userCtx := r.Context().Value(domain.UserContextKey)
	if userCtx != nil {
		claims := userCtx.(map[string]string)
		role := claims["role"]
		userID := claims["id"]

		if role != "admin" {
			perms, err := h.Store.GetPermissions(userID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			allowed := make(map[string]bool)
			permsMap := make(map[string]domain.Permission)
			for _, p := range perms {
				if p.CanViewConsole || p.CanControlPower {
					allowed[p.ServerID] = true
					permsMap[p.ServerID] = p
				}
			}

			var filtered []domain.Server
			for _, s := range servers {
				if allowed[s.ID] {
					s.Permissions = new(permsMap[s.ID])
					filtered = append(filtered, s)
				}
			}
			servers = filtered
		} else {
			adminPerm := domain.Permission{
				CanViewConsole:  true,
				CanControlPower: true,
			}
			for i := range servers {
				servers[i].Permissions = &adminPerm
			}
		}
	}

	json.NewEncoder(w).Encode(servers)
}

func (h *ServerHandler) HandleCreateServer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string               `json:"name"`
		Version       string               `json:"version"`
		Loader        string               `json:"loader"`
		LoaderOptions loader.LoaderOptions `json:"loaderOptions"`
		RAM           int                  `json:"ram"`
		RequestID     string               `json:"requestId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	progressChan := make(chan domain.ProgressEvent)
	hubID := "progress"
	if req.RequestID != "" {
		hubID = req.RequestID
	}
	hub := h.HubManager.GetHub(hubID)

	go func() {
		for event := range progressChan {
			if event.ServerID == "" {
				event.ServerID = "new-server"
			}
			jsonBytes, _ := json.Marshal(event)
			hub.Broadcast(jsonBytes)
		}
	}()

	if req.LoaderOptions.MCVersion == "" && req.Version != "" {
		req.LoaderOptions.MCVersion = req.Version
	}
	h.Manager.StartCreateServerJob(req.Name, req.Loader, req.LoaderOptions, req.Version, req.RAM, progressChan)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	response := map[string]string{
		"status": "creating",
		"id":     req.RequestID,
	}
	json.NewEncoder(w).Encode(response)
}

func (h *ServerHandler) HandleGetServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	srv, err := h.Manager.GetServer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if srv == nil {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	userCtx := r.Context().Value(domain.UserContextKey)
	if userCtx != nil {
		claims := userCtx.(map[string]string)
		role := claims["role"]
		userID := claims["id"]

		if role == "admin" {
			srv.Permissions = &domain.Permission{
				UserID:          userID,
				ServerID:        srv.ID,
				CanViewConsole:  true,
				CanControlPower: true,
			}
		} else {
			perms, err := h.Store.GetPermissions(userID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var userPerm *domain.Permission
			for _, p := range perms {
				if p.ServerID == srv.ID {
					userPerm = new(p)
					break
				}
			}

			if userPerm == nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			srv.Permissions = userPerm
		}
	}

	json.NewEncoder(w).Encode(srv)
}

func (h *ServerHandler) HandleUpdateServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Name       *string `json:"name"`
		RAM        *int    `json:"ram"`
		CustomArgs *string `json:"customArgs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.Store.UpdateServer(id, req.Name, req.RAM, req.CustomArgs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ServerHandler) HandleUpdateServerAutoBackup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Enabled       bool   `json:"enabled"`
		IntervalValue int    `json:"intervalValue"`
		IntervalUnit  string `json:"intervalUnit"`
		MaxBackups    int    `json:"maxBackups"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.BackupManager.UpdateAutoBackupConfig(
		id,
		req.Enabled,
		req.IntervalValue,
		req.IntervalUnit,
		req.MaxBackups,
	); err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if strings.Contains(msg, "invalid") ||
			strings.Contains(msg, "must be") ||
			strings.Contains(msg, "at least") ||
			strings.Contains(msg, "cannot exceed") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updatedServer, err := h.Manager.GetServer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if updatedServer == nil {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedServer)
}

func (h *ServerHandler) HandleGetServerSettings(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	settings, err := h.Manager.GetServerSettings(id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (h *ServerHandler) HandleUpdateServerSettings(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	var req server.ServerSettings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.Manager.UpdateServerSettings(id, req); err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if strings.Contains(msg, "must be stopped") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if strings.Contains(msg, "invalid") ||
			strings.Contains(msg, "must be") ||
			strings.Contains(msg, "required") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ServerHandler) HandleGetVersionOptions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	versions, err := h.Manager.GetVersionOptions(id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{
		"versions": versions,
	})
}

func (h *ServerHandler) HandleUpdateServerVersion(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}
	var req struct {
		Version             string `json:"version"`
		IncludeDependencies *bool  `json:"includeDependencies,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	includeDependencies := true
	if req.IncludeDependencies != nil {
		includeDependencies = *req.IncludeDependencies
	}

	srv, version, err := h.Manager.ValidateServerVersionUpdate(id, req.Version)
	if err != nil {
		h.writeVersionUpdateError(w, err)
		return
	}
	if h.BackupManager == nil {
		http.Error(w, "Backup manager is not configured", http.StatusServiceUnavailable)
		return
	}

	userID := "system"
	if userCtx := r.Context().Value(domain.UserContextKey); userCtx != nil {
		claims := userCtx.(map[string]string)
		if claims["id"] != "" {
			userID = claims["id"]
		}
	}

	backupLabel := fmt.Sprintf("pre-update-%s-%s-to-%s", srv.Name, srv.Version, version)
	backupPath, err := h.BackupManager.CreateBackup(r.Context(), id, backupLabel, userID, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create pre-update backup: %v", err), http.StatusInternalServerError)
		return
	}
	backupName := filepath.Base(backupPath)

	resolvedVersion, err := h.Manager.ApplyServerVersionUpdate(id, version)
	if err != nil {
		restoreErr := h.BackupManager.RestoreBackup(backupName, id, "", 0, "", "")
		if restoreErr != nil {
			http.Error(
				w,
				fmt.Sprintf("version update failed: %v; backup %s restore failed: %v", err, backupName, restoreErr),
				http.StatusInternalServerError,
			)
			return
		}
		http.Error(
			w,
			fmt.Sprintf("version update failed: %v; restored backup %s", err, backupName),
			http.StatusInternalServerError,
		)
		return
	}

	var addonsResult any
	if h.AddonsManager != nil {
		result, err := h.AddonsManager.UpdateAddonsForServerVersion(r.Context(), id, includeDependencies)
		if err != nil {
			addonsResult = map[string]any{
				"updated":  []string{},
				"disabled": []string{},
				"failed": []map[string]string{{
					"id":     "addons",
					"reason": err.Error(),
				}},
			}
		} else {
			addonsResult = result
		}
	}

	response := map[string]any{
		"backupName":    backupName,
		"restored":      false,
		"serverUpdated": true,
		"version":       resolvedVersion,
		"addons":        addonsResult,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *ServerHandler) writeVersionUpdateError(w http.ResponseWriter, err error) {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "not found") {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if strings.Contains(msg, "must be stopped") {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if strings.Contains(msg, "version") && (strings.Contains(msg, "required") ||
		strings.Contains(msg, "not available") ||
		strings.Contains(msg, "must be greater") ||
		strings.Contains(msg, "unable to compare")) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func (h *ServerHandler) HandleDeleteServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	if err := h.Manager.DeleteServer(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.HubManager.RemoveHub(id)

	w.WriteHeader(http.StatusNoContent)
}

func (h *ServerHandler) HandleStartServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	if !h.checkPermission(r, id, func(p *domain.Permission) bool {
		return p.CanControlPower || p.CanViewConsole
	}) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.Supervisor.StartServer(id); err != nil {
		http.Error(w, fmt.Sprintf("Error starting: %v", err), http.StatusBadRequest)
		return
	}

	w.Write([]byte(`{"status": "started"}`))
}

func (h *ServerHandler) HandleStopServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	if !h.checkPermission(r, id, func(p *domain.Permission) bool {
		return p.CanControlPower || p.CanViewConsole
	}) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.Supervisor.StopServer(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(`{"status": "stopping"}`))
}

func (h *ServerHandler) HandleRestartServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	if !h.checkPermission(r, id, func(p *domain.Permission) bool {
		return p.CanControlPower || p.CanViewConsole
	}) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.Supervisor.RestartServer(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(`{"status": "restarting"}`))
}

func (h *ServerHandler) HandleKillServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	if !h.checkPermission(r, id, func(p *domain.Permission) bool {
		return p.CanControlPower || p.CanViewConsole
	}) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.Supervisor.KillServer(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(`{"status": "killed"}`))
}

func (h *ServerHandler) HandleGetServerStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	stats, err := h.Supervisor.GetServerStats(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *ServerHandler) HandleGetAllServerStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.Supervisor.GetAllServerStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *ServerHandler) HandleGetServerIcon(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	iconPath, err := h.Manager.GetServerIconPath(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, iconPath)
}

func (h *ServerHandler) HandleUploadServerIcon(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("icon")
	if err != nil {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		http.Error(w, "Invalid image format", http.StatusBadRequest)
		return
	}

	if err := h.Manager.SaveServerIcon(id, img); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ServerHandler) HandleBackupServer(w http.ResponseWriter, r *http.Request) {
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

	userCtx := r.Context().Value(domain.UserContextKey)
	var userID string
	if userCtx != nil {
		claims := userCtx.(map[string]string)
		userID = claims["id"]
	}

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

	h.BackupManager.StartBackupJob(id, req.Name, req.RequestID, userID, progressChan)

	response := map[string]string{
		"status": "creating",
		"id":     req.RequestID,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}
