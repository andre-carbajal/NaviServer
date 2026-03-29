package api

import (
	"fmt"
	"naviger/internal/api/handlers"
	"naviger/internal/backup"
	"naviger/internal/config"
	"naviger/internal/runner"
	"naviger/internal/server"
	"naviger/internal/storage"
	"naviger/internal/ws"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	baseHandler := handlers.NewBaseHandler(api.Manager, api.Supervisor, api.Store, api.HubManager, api.BackupManager, api.Config)

	authHandler := &handlers.AuthHandler{BaseHandler: baseHandler}
	serverHandler := &handlers.ServerHandler{BaseHandler: baseHandler}
	filesHandler := &handlers.FilesHandler{BaseHandler: baseHandler}
	backupHandler := &handlers.BackupHandler{BaseHandler: baseHandler}
	settingsHandler := &handlers.SettingsHandler{BaseHandler: baseHandler}
	systemHandler := &handlers.SystemHandler{BaseHandler: baseHandler}
	usersHandler := &handlers.UsersHandler{BaseHandler: baseHandler}
	linksHandler := &handlers.LinksHandler{BaseHandler: baseHandler}
	wsHandler := &handlers.WSHandler{BaseHandler: baseHandler}
	loadersHandler := &handlers.LoadersHandler{BaseHandler: baseHandler}

	mux.HandleFunc("POST /auth/login", authHandler.HandleLogin)
	mux.HandleFunc("POST /auth/logout", authHandler.HandleLogout)
	mux.HandleFunc("POST /auth/setup", authHandler.HandleSetup)
	mux.HandleFunc("GET /auth/setup", authHandler.HandleCheckSetup)
	mux.HandleFunc("POST /public-links/{token}/access", linksHandler.HandleAccessPublicLink)
	mux.HandleFunc("GET /public-links/{token}", linksHandler.HandleGetPublicServerInfo)

	protect := func(h http.HandlerFunc, role string) http.Handler {
		return api.AuthMiddleware(h, role, api.Config.JWTSecret)
	}

	mux.Handle("DELETE /public-links/{token}", protect(linksHandler.HandleDeletePublicLink, ""))

	mux.Handle("GET /auth/me", protect(authHandler.HandleMe, ""))

	mux.Handle("GET /loaders", protect(loadersHandler.HandleGetLoaders, ""))
	mux.Handle("GET /loaders/{name}/versions", protect(loadersHandler.HandleGetLoaderVersions, ""))

	mux.Handle("GET /servers", protect(serverHandler.HandleListServers, ""))
	mux.Handle("GET /servers-stats", protect(serverHandler.HandleGetAllServerStats, ""))
	mux.Handle("POST /servers", protect(serverHandler.HandleCreateServer, "admin"))

	mux.Handle("GET /servers/{id}", protect(serverHandler.HandleGetServer, ""))
	mux.Handle("GET /servers/{id}/stats", protect(serverHandler.HandleGetServerStats, ""))
	mux.HandleFunc("GET /servers/{id}/icon", serverHandler.HandleGetServerIcon)
	mux.Handle("POST /servers/{id}/icon", protect(serverHandler.HandleUploadServerIcon, "admin"))
	mux.Handle("PUT /servers/{id}", protect(serverHandler.HandleUpdateServer, "admin"))
	mux.Handle("DELETE /servers/{id}", protect(serverHandler.HandleDeleteServer, "admin"))

	mux.Handle("GET /servers/{id}/files", protect(filesHandler.HandleListFiles, ""))
	mux.Handle("GET /servers/{id}/files/content", protect(filesHandler.HandleGetFileContent, ""))
	mux.Handle("PUT /servers/{id}/files/content", protect(filesHandler.HandleSaveFileContent, ""))
	mux.Handle("POST /servers/{id}/files/directory", protect(filesHandler.HandleCreateDirectory, ""))
	mux.Handle("DELETE /servers/{id}/files", protect(filesHandler.HandleDeleteFile, ""))
	mux.Handle("GET /servers/{id}/files/download", protect(filesHandler.HandleDownloadFile, ""))
	mux.Handle("POST /servers/{id}/files/upload", protect(filesHandler.HandleUploadFile, ""))

	mux.Handle("POST /servers/{id}/start", protect(serverHandler.HandleStartServer, ""))
	mux.Handle("POST /servers/{id}/stop", protect(serverHandler.HandleStopServer, ""))
	mux.Handle("POST /servers/{id}/backup", protect(serverHandler.HandleBackupServer, ""))
	mux.Handle("GET /servers/{id}/backups", protect(backupHandler.HandleListBackupsByServer, ""))

	mux.Handle("GET /backups", protect(backupHandler.HandleListAllBackups, ""))
	mux.Handle("POST /backups/upload", protect(backupHandler.HandleUploadBackup, ""))
	mux.Handle("PUT /backups/{name}", protect(backupHandler.HandleUpdateBackup, "admin"))
	mux.Handle("DELETE /backups/{name}", protect(backupHandler.HandleDeleteBackup, ""))
	mux.Handle("GET /backups/{name}/download", protect(backupHandler.HandleDownloadBackup, ""))
	mux.Handle("DELETE /backups/progress/{id}", protect(backupHandler.HandleCancelBackup, ""))
	mux.Handle("POST /backups/{name}/restore", protect(backupHandler.HandleRestoreBackup, ""))

	mux.Handle("GET /settings/port-range", protect(settingsHandler.HandleGetPortRange, "admin"))
	mux.Handle("PUT /settings/port-range", protect(settingsHandler.HandleSetPortRange, "admin"))
	mux.Handle("GET /settings/log-buffer-size", protect(settingsHandler.HandleGetLogBufferSize, "admin"))
	mux.Handle("PUT /settings/log-buffer-size", protect(settingsHandler.HandleSetLogBufferSize, "admin"))
	mux.Handle("GET /settings/public-ip", protect(settingsHandler.HandleGetPublicIP, ""))
	mux.Handle("PUT /settings/public-ip", protect(settingsHandler.HandleSetPublicIP, "admin"))

	mux.Handle("GET /system/interfaces", protect(systemHandler.HandleGetNetworkInterfaces, "admin"))
	mux.Handle("POST /system/restart", protect(systemHandler.HandleRestartDaemon, "admin"))
	mux.Handle("GET /updates", protect(systemHandler.HandleCheckUpdates, "admin"))
	mux.Handle("GET /version", protect(systemHandler.HandleGetVersion, ""))

	mux.Handle("GET /ws/servers/{id}/console", protect(wsHandler.HandleConsole, ""))
	mux.Handle("GET /ws/progress/{id}", protect(wsHandler.HandleProgress, ""))

	mux.Handle("GET /users", protect(usersHandler.HandleListUsers, "admin"))
	mux.Handle("POST /users", protect(usersHandler.HandleCreateUser, "admin"))
	mux.Handle("PUT /users/permissions", protect(usersHandler.HandleUpdatePermissions, "admin"))
	mux.Handle("GET /users/{id}/permissions", protect(usersHandler.HandleGetPermissions, "admin"))
	mux.Handle("DELETE /users/{id}", protect(usersHandler.HandleDeleteUser, "admin"))
	mux.Handle("PUT /users/{id}/password", protect(usersHandler.HandleUpdatePassword, ""))

	mux.Handle("POST /public-links", protect(linksHandler.HandleCreatePublicLink, ""))
	mux.Handle("GET /servers/{id}/public-link", protect(linksHandler.HandleGetPublicLink, ""))

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

func (api *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1") {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
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
