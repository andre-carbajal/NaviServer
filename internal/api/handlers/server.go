package handlers

import (
	"encoding/json"
	"fmt"
	"image"
	"net/http"

	"naviger/internal/domain"
	"naviger/internal/runner"
	"naviger/internal/server"
	"naviger/internal/storage"
	"naviger/internal/ws"
)

type ServerHandler struct {
	Manager    *server.Manager
	Supervisor *runner.Supervisor
	Store      *storage.GormStore
	HubManager *ws.HubManager
}

func NewServerHandler(
	manager *server.Manager,
	supervisor *runner.Supervisor,
	store *storage.GormStore,
	hubManager *ws.HubManager,
) *ServerHandler {
	return &ServerHandler{
		Manager:    manager,
		Supervisor: supervisor,
		Store:      store,
		HubManager: hubManager,
	}
}

func (h *ServerHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	if !CheckPermission(h.Store, r, id, func(p *domain.Permission) bool {
		return true // Read access
	}) {
		http.Error(w, "Forbidden", http.StatusForbidden)
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

	json.NewEncoder(w).Encode(srv)
}

func (h *ServerHandler) GetIcon(w http.ResponseWriter, r *http.Request) {
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

func (h *ServerHandler) UploadIcon(w http.ResponseWriter, r *http.Request) {
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

func (h *ServerHandler) Update(w http.ResponseWriter, r *http.Request) {
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

func (h *ServerHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

func (h *ServerHandler) List(w http.ResponseWriter, r *http.Request) {
	servers, err := h.Manager.ListServers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	claims, ok := GetUserClaims(r)
	if ok {
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
				allowed[p.ServerID] = true
				permsMap[p.ServerID] = p
			}

			var filtered []domain.Server
			for _, s := range servers {
				if allowed[s.ID] {
					perm := permsMap[s.ID]
					s.Permissions = &perm
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

func (h *ServerHandler) Create(w http.ResponseWriter, r *http.Request) {
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

func (h *ServerHandler) Start(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	if !CheckPermission(h.Store, r, id, func(p *domain.Permission) bool {
		return p.CanControlPower
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

func (h *ServerHandler) Stats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	if !CheckPermission(h.Store, r, id, func(p *domain.Permission) bool {
		return true
	}) {
		http.Error(w, "Forbidden", http.StatusForbidden)
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

func (h *ServerHandler) AllStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.Supervisor.GetAllServerStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *ServerHandler) Stop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if !CheckPermission(h.Store, r, id, func(p *domain.Permission) bool {
		return p.CanControlPower
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
