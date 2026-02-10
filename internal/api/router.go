package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"naviger/internal/api/handlers"
	"naviger/internal/backup"
	"naviger/internal/config"
	"naviger/internal/runner"
	"naviger/internal/server"
	"naviger/internal/storage"
	"naviger/internal/ws"
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

	authHandler := handlers.NewAuthHandler(api.Store, api.Config)
	serverHandler := handlers.NewServerHandler(api.Manager, api.Supervisor, api.Store, api.HubManager)
	backupHandler := handlers.NewBackupHandler(api.BackupManager, api.HubManager)
	loaderHandler := handlers.NewLoaderHandler()
	settingsHandler := handlers.NewSettingsHandler(api.Store, api.HubManager)
	systemHandler := handlers.NewSystemHandler()
	wsHandler := handlers.NewWSHandler(api.HubManager)
	userHandler := handlers.NewUsersHandler(api.Store)
	fileHandler := handlers.NewFilesHandler(api.Manager, api.Store)
	linkHandler := handlers.NewPublicLinkHandler(api.Store, api.Manager, api.Supervisor)

	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("POST /auth/logout", authHandler.Logout)
	mux.HandleFunc("POST /auth/setup", authHandler.Setup)
	mux.HandleFunc("POST /public-links/{token}/access", linkHandler.Access)
	mux.HandleFunc("GET /public-links/{token}", linkHandler.GetServerInfo)
	mux.HandleFunc("DELETE /public-links/{token}", linkHandler.Delete)

	protect := func(h http.HandlerFunc, role string) http.Handler {
		return api.AuthMiddleware(h, role, api.Config.JWTSecret)
	}

	mux.Handle("GET /auth/me", protect(authHandler.Me, ""))

	mux.Handle("GET /loaders", protect(loaderHandler.GetAvailable, ""))
	mux.Handle("GET /loaders/{name}/versions", protect(loaderHandler.GetVersions, ""))

	mux.Handle("GET /servers", protect(serverHandler.List, ""))
	mux.Handle("GET /servers-stats", protect(serverHandler.AllStats, ""))
	mux.Handle("POST /servers", protect(serverHandler.Create, "admin"))

	mux.Handle("GET /servers/{id}", protect(serverHandler.Get, ""))
	mux.Handle("GET /servers/{id}/stats", protect(serverHandler.Stats, ""))
	mux.HandleFunc("GET /servers/{id}/icon", serverHandler.GetIcon)
	mux.Handle("POST /servers/{id}/icon", protect(serverHandler.UploadIcon, "admin"))
	mux.Handle("PUT /servers/{id}", protect(serverHandler.Update, "admin"))
	mux.Handle("DELETE /servers/{id}", protect(serverHandler.Delete, "admin"))

	mux.Handle("GET /servers/{id}/files", protect(fileHandler.List, ""))
	mux.Handle("GET /servers/{id}/files/content", protect(fileHandler.GetContent, ""))
	mux.Handle("PUT /servers/{id}/files/content", protect(fileHandler.SaveContent, ""))
	mux.Handle("POST /servers/{id}/files/directory", protect(fileHandler.CreateDirectory, ""))
	mux.Handle("DELETE /servers/{id}/files", protect(fileHandler.Delete, ""))
	mux.Handle("GET /servers/{id}/files/download", protect(fileHandler.Download, ""))
	mux.Handle("POST /servers/{id}/files/upload", protect(fileHandler.Upload, ""))

	mux.Handle("POST /servers/{id}/start", protect(serverHandler.Start, ""))
	mux.Handle("POST /servers/{id}/stop", protect(serverHandler.Stop, ""))
	mux.Handle("POST /servers/{id}/backup", protect(backupHandler.Create, ""))
	mux.Handle("GET /servers/{id}/backups", protect(backupHandler.ListByServer, ""))

	mux.Handle("GET /backups", protect(backupHandler.ListAll, "admin"))
	mux.Handle("DELETE /backups/{name}", protect(backupHandler.Delete, "admin"))
	mux.Handle("DELETE /backups/progress/{id}", protect(backupHandler.Cancel, "admin"))
	mux.Handle("POST /backups/{name}/restore", protect(backupHandler.Restore, "admin"))

	mux.Handle("GET /settings/port-range", protect(settingsHandler.GetPortRange, "admin"))
	mux.Handle("PUT /settings/port-range", protect(settingsHandler.SetPortRange, "admin"))
	mux.Handle("GET /settings/log-buffer-size", protect(settingsHandler.GetLogBufferSize, "admin"))
	mux.Handle("PUT /settings/log-buffer-size", protect(settingsHandler.SetLogBufferSize, "admin"))
	mux.Handle("GET /settings/public-ip", protect(settingsHandler.GetPublicIP, "admin"))
	mux.Handle("PUT /settings/public-ip", protect(settingsHandler.SetPublicIP, "admin"))

	mux.Handle("GET /system/interfaces", protect(systemHandler.GetNetworkInterfaces, "admin"))
	mux.Handle("POST /system/restart", protect(systemHandler.RestartDaemon, "admin"))
	mux.Handle("GET /updates", protect(settingsHandler.CheckUpdates, "admin"))

	mux.Handle("GET /ws/servers/{id}/console", protect(wsHandler.Console, ""))
	mux.Handle("GET /ws/progress/{id}", protect(wsHandler.Progress, ""))

	mux.Handle("GET /users", protect(userHandler.List, "admin"))
	mux.Handle("POST /users", protect(userHandler.Create, "admin"))
	mux.Handle("PUT /users/permissions", protect(userHandler.UpdatePermissions, "admin"))
	mux.Handle("GET /users/{id}/permissions", protect(userHandler.GetPermissions, "admin"))
	mux.Handle("DELETE /users/{id}", protect(userHandler.Delete, "admin"))
	mux.Handle("PUT /users/{id}/password", protect(userHandler.UpdatePassword, ""))

	mux.Handle("POST /public-links", protect(linkHandler.Create, "admin"))

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
