package handlers

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"naviserver/internal/domain"
	"net/http"
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
		Name      string `json:"name"`
		Version   string `json:"version"`
		Loader    string `json:"loader"`
		RAM       int    `json:"ram"`
		RequestID string `json:"requestId"`
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

	h.Manager.StartCreateServerJob(req.Name, req.Loader, req.Version, req.RAM, progressChan)

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
