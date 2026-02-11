package api

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"naviger/internal/backup"
	"naviger/internal/config"
	"naviger/internal/domain"
	"naviger/internal/loader"
	"naviger/internal/runner"
	"naviger/internal/server"
	"naviger/internal/storage"
	"naviger/internal/updater"
	"naviger/internal/ws"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

type Server struct {
	Manager       *server.Manager
	Supervisor    *runner.Supervisor
	Store         *storage.GormStore
	HubManager    *ws.HubManager
	BackupManager *backup.Manager
	Config        *config.Config
}

func NewAPIServer(
	manager *server.Manager,
	supervisor *runner.Supervisor,
	store *storage.GormStore,
	hubManager *ws.HubManager,
	backupManager *backup.Manager,
	cfg *config.Config,
) *Server {
	return &Server{
		Manager:       manager,
		Supervisor:    supervisor,
		Store:         store,
		HubManager:    hubManager,
		BackupManager: backupManager,
		Config:        cfg,
	}
}

func (api *Server) CreateHTTPServer(listenAddr string) *http.Server {
	mux := http.NewServeMux()

	ex, err := os.Executable()
	var webDistPath string
	if err == nil {
		exPath := filepath.Dir(ex)

		path1 := filepath.Join(exPath, "web_dist")
		if _, err := os.Stat(path1); err == nil {
			webDistPath = path1
		} else {
			path2 := filepath.Join(filepath.Dir(exPath), "Resources", "web_dist")
			if _, err := os.Stat(path2); err == nil {
				webDistPath = path2
			} else {
				webDistPath = "web_dist"
			}
		}
	} else {
		webDistPath = "web_dist"
	}

	fileServer := http.FileServer(http.Dir(webDistPath))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(webDistPath, r.URL.Path)
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(webDistPath, "index.html"))
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	mux.HandleFunc("POST /auth/login", api.handleLogin)
	mux.HandleFunc("POST /auth/logout", api.handleLogout)
	mux.HandleFunc("POST /auth/setup", api.handleSetup)
	mux.HandleFunc("POST /public-links/{token}/access", api.handleAccessPublicLink)
	mux.HandleFunc("GET /public-links/{token}", api.handleGetPublicServerInfo)
	mux.HandleFunc("DELETE /public-links/{token}", api.handleDeletePublicLink)

	protect := func(h http.HandlerFunc, role string) http.Handler {
		return api.AuthMiddleware(h, role, api.Config.JWTSecret)
	}

	mux.Handle("GET /auth/me", protect(api.handleMe, ""))

	mux.Handle("GET /loaders", protect(api.handleGetLoaders, ""))
	mux.Handle("GET /loaders/{name}/versions", protect(api.handleGetLoaderVersions, ""))

	mux.Handle("GET /servers", protect(api.handleListServers, ""))
	mux.Handle("GET /servers-stats", protect(api.handleGetAllServerStats, ""))
	mux.Handle("POST /servers", protect(api.handleCreateServer, "admin"))

	mux.Handle("GET /servers/{id}", protect(api.handleGetServer, ""))
	mux.Handle("GET /servers/{id}/stats", protect(api.handleGetServerStats, ""))
	mux.HandleFunc("GET /servers/{id}/icon", api.handleGetServerIcon)
	mux.Handle("POST /servers/{id}/icon", protect(api.handleUploadServerIcon, "admin"))
	mux.Handle("PUT /servers/{id}", protect(api.handleUpdateServer, "admin"))
	mux.Handle("DELETE /servers/{id}", protect(api.handleDeleteServer, "admin"))

	mux.Handle("GET /servers/{id}/files", protect(api.handleListFiles, ""))
	mux.Handle("GET /servers/{id}/files/content", protect(api.handleGetFileContent, ""))
	mux.Handle("PUT /servers/{id}/files/content", protect(api.handleSaveFileContent, ""))
	mux.Handle("POST /servers/{id}/files/directory", protect(api.handleCreateDirectory, ""))
	mux.Handle("DELETE /servers/{id}/files", protect(api.handleDeleteFile, ""))
	mux.Handle("GET /servers/{id}/files/download", protect(api.handleDownloadFile, ""))
	mux.Handle("POST /servers/{id}/files/upload", protect(api.handleUploadFile, ""))

	mux.Handle("POST /servers/{id}/start", protect(api.handleStartServer, ""))
	mux.Handle("POST /servers/{id}/stop", protect(api.handleStopServer, ""))
	mux.Handle("POST /servers/{id}/backup", protect(api.handleBackupServer, ""))
	mux.Handle("GET /servers/{id}/backups", protect(api.handleListBackupsByServer, ""))

	mux.Handle("GET /backups", protect(api.handleListAllBackups, "admin"))
	mux.Handle("DELETE /backups/{name}", protect(api.handleDeleteBackup, "admin"))
	mux.Handle("DELETE /backups/progress/{id}", protect(api.handleCancelBackup, "admin"))
	mux.Handle("POST /backups/{name}/restore", protect(api.handleRestoreBackup, "admin"))

	mux.Handle("GET /settings/port-range", protect(api.handleGetPortRange, "admin"))
	mux.Handle("PUT /settings/port-range", protect(api.handleSetPortRange, "admin"))
	mux.Handle("GET /settings/log-buffer-size", protect(api.handleGetLogBufferSize, "admin"))
	mux.Handle("PUT /settings/log-buffer-size", protect(api.handleSetLogBufferSize, "admin"))
	mux.Handle("GET /settings/public-ip", protect(api.handleGetPublicIP, "admin"))
	mux.Handle("PUT /settings/public-ip", protect(api.handleSetPublicIP, "admin"))

	mux.Handle("GET /system/interfaces", protect(api.handleGetNetworkInterfaces, "admin"))
	mux.Handle("POST /system/restart", protect(api.handleRestartDaemon, "admin"))
	mux.Handle("GET /updates", protect(api.handleCheckUpdates, "admin"))

	mux.Handle("GET /ws/servers/{id}/console", protect(api.handleConsole, ""))
	mux.Handle("GET /ws/progress/{id}", protect(api.handleProgress, ""))

	mux.Handle("GET /users", protect(api.handleListUsers, "admin"))
	mux.Handle("POST /users", protect(api.handleCreateUser, "admin"))
	mux.Handle("PUT /users/permissions", protect(api.handleUpdatePermissions, "admin"))
	mux.Handle("GET /users/{id}/permissions", protect(api.handleGetPermissions, "admin"))
	mux.Handle("DELETE /users/{id}", protect(api.handleDeleteUser, "admin"))
	mux.Handle("PUT /users/{id}/password", protect(api.handleUpdatePassword, ""))

	mux.Handle("POST /public-links", protect(api.handleCreatePublicLink, "admin"))

	handler := api.corsMiddleware(mux)

	return &http.Server{
		Addr:    listenAddr,
		Handler: handler,
	}
}

func (api *Server) Start(listenAddr string) error {
	httpServer := api.CreateHTTPServer(listenAddr)
	fmt.Printf("API listening on http://localhost%s\n", listenAddr)
	return httpServer.ListenAndServe()
}

func (api *Server) handleRestartDaemon(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status": "restarting"}`))
	go func() {
		os.Exit(0)
	}()
}

func (api *Server) handleCheckUpdates(w http.ResponseWriter, r *http.Request) {
	updateInfo, err := updater.CheckForUpdates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updateInfo)
}

func (api *Server) handleGetLoaderVersions(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "Missing loader name", http.StatusBadRequest)
		return
	}

	versions, err := loader.GetLoaderVersions(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versions)
}

func (api *Server) handleGetLoaders(w http.ResponseWriter, r *http.Request) {
	loaders := loader.GetAvailableLoaders()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loaders)
}

func (api *Server) handleDeleteBackup(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "Missing backup name", http.StatusBadRequest)
		return
	}

	if err := api.BackupManager.DeleteBackup(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (api *Server) handleRestoreBackup(w http.ResponseWriter, r *http.Request) {
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

	if err := api.BackupManager.RestoreBackup(name, req.TargetServerID, req.NewServerName, req.NewServerRAM, req.NewServerLoader, req.NewServerVersion); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "restored"}`))
}

func (api *Server) handleListBackupsByServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	backups, err := api.BackupManager.ListBackups(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backups)
}

func (api *Server) handleListAllBackups(w http.ResponseWriter, r *http.Request) {
	backups, err := api.BackupManager.ListAllBackups()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backups)
}

func (api *Server) handleGetServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	srv, err := api.Manager.GetServer(id)
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

func (api *Server) handleGetServerIcon(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	iconPath, err := api.Manager.GetServerIconPath(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, iconPath)
}

func (api *Server) handleUploadServerIcon(w http.ResponseWriter, r *http.Request) {
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

	if err := api.Manager.SaveServerIcon(id, img); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (api *Server) handleUpdateServer(w http.ResponseWriter, r *http.Request) {
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

	if err := api.Store.UpdateServer(id, req.Name, req.RAM, req.CustomArgs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (api *Server) handleDeleteServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	if err := api.Manager.DeleteServer(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	api.HubManager.RemoveHub(id)

	w.WriteHeader(http.StatusNoContent)
}

func (api *Server) handleListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := api.Manager.ListServers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userCtx := r.Context().Value(UserContextKey)
	if userCtx != nil {
		claims := userCtx.(map[string]string)
		role := claims["role"]
		userID := claims["id"]

		if role != "admin" {
			perms, err := api.Store.GetPermissions(userID)
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

func (api *Server) handleCreateServer(w http.ResponseWriter, r *http.Request) {
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
	hub := api.HubManager.GetHub(hubID)

	go func() {
		for event := range progressChan {
			if event.ServerID == "" {
				event.ServerID = "new-server"
			}
			jsonBytes, _ := json.Marshal(event)
			hub.Broadcast(jsonBytes)
		}
	}()

	api.Manager.StartCreateServerJob(req.Name, req.Loader, req.Version, req.RAM, progressChan)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	response := map[string]string{
		"status": "creating",
		"id":     req.RequestID,
	}
	json.NewEncoder(w).Encode(response)
}

func (api *Server) handleStartServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	if !api.checkPermission(r, id, func(p *domain.Permission) bool {
		return p.CanControlPower || p.CanViewConsole
	}) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := api.Supervisor.StartServer(id); err != nil {
		http.Error(w, fmt.Sprintf("Error starting: %v", err), http.StatusBadRequest)
		return
	}

	w.Write([]byte(`{"status": "started"}`))
}

func (api *Server) handleGetServerStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	stats, err := api.Supervisor.GetServerStats(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (api *Server) handleGetAllServerStats(w http.ResponseWriter, r *http.Request) {
	stats, err := api.Supervisor.GetAllServerStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (api *Server) handleStopServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if !api.checkPermission(r, id, func(p *domain.Permission) bool {
		return p.CanControlPower || p.CanViewConsole
	}) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := api.Supervisor.StopServer(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(`{"status": "stopping"}`))
}

func (api *Server) handleBackupServer(w http.ResponseWriter, r *http.Request) {
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
	hub := api.HubManager.GetHub(hubID)

	go func() {
		for event := range progressChan {
			if event.ServerID == "" {
				event.ServerID = id
			}
			jsonBytes, _ := json.Marshal(event)
			hub.Broadcast(jsonBytes)
		}
	}()

	api.BackupManager.StartBackupJob(id, req.Name, req.RequestID, progressChan)

	response := map[string]string{
		"status": "creating",
		"id":     req.RequestID,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

func (api *Server) handleGetPortRange(w http.ResponseWriter, r *http.Request) {
	start, end, err := api.Store.GetPortRange()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]int{
		"start": start,
		"end":   end,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (api *Server) handleSetPortRange(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Start int `json:"start"`
		End   int `json:"end"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := api.Store.SetPortRange(req.Start, req.End); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "updated"}`))
}

func (api *Server) handleGetLogBufferSize(w http.ResponseWriter, r *http.Request) {
	val, err := api.Store.GetSetting("log_buffer_size")
	if err != nil {
		response := map[string]int{"log_buffer_size": config.DefaultLogBufferSize}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		http.Error(w, "invalid stored value for log_buffer_size", http.StatusInternalServerError)
		return
	}
	response := map[string]int{"log_buffer_size": n}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (api *Server) handleSetLogBufferSize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LogBufferSize int `json:"log_buffer_size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.LogBufferSize < 0 {
		http.Error(w, "log_buffer_size must be >= 0", http.StatusBadRequest)
		return
	}
	if err := api.Store.SetLogBufferSize(req.LogBufferSize); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if api.HubManager != nil {
		api.HubManager.SetDefaultHistorySize(req.LogBufferSize)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"updated"}`))
}

func (api *Server) handleGetNetworkInterfaces(w http.ResponseWriter, r *http.Request) {
	var addresses []string

	interfaces, err := net.Interfaces()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			if ip.To4() != nil {
				addresses = append(addresses, ip.String())
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{"interfaces": addresses})
}

func (api *Server) handleGetPublicIP(w http.ResponseWriter, r *http.Request) {
	val, err := api.Store.GetSetting("public_ip")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"public_ip": "localhost"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"public_ip": val})
}

func (api *Server) handleSetPublicIP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PublicIP string `json:"public_ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.PublicIP == "" {
		http.Error(w, "public_ip cannot be empty", http.StatusBadRequest)
		return
	}

	if err := api.Store.SetSetting("public_ip", req.PublicIP); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"updated"}`))
}
func (api *Server) handleConsole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	hub := api.HubManager.GetHub(id)
	hub.ServeWs(w, r)
}

func (api *Server) handleProgress(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}
	hub := api.HubManager.GetHub(id)
	hub.ServeWs(w, r)
}

func (api *Server) handleCancelBackup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	api.BackupManager.CancelBackup(id)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "cancelled"}`))
}

func (api *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (api *Server) checkPermission(r *http.Request, serverID string, check func(*domain.Permission) bool) bool {
	userCtx := r.Context().Value(UserContextKey)
	if userCtx == nil {
		return false
	}
	claims := userCtx.(map[string]string)
	role := claims["role"]
	if role == "admin" {
		return true
	}

	userID := claims["id"]
	perms, err := api.Store.GetPermissions(userID)
	if err != nil {
		return false
	}

	for _, p := range perms {
		if p.ServerID == serverID {
			return check(&p)
		}
	}
	return false
}
