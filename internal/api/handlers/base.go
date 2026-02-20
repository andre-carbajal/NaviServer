package handlers

import (
	"net/http"

	"naviger/internal/backup"
	"naviger/internal/config"
	"naviger/internal/domain"
	"naviger/internal/runner"
	"naviger/internal/server"
	"naviger/internal/storage"
	"naviger/internal/ws"
)

type BaseHandler struct {
	Manager       *server.Manager
	Supervisor    *runner.Supervisor
	Store         *storage.GormStore
	HubManager    *ws.HubManager
	BackupManager *backup.Manager
	Config        *config.Config
}

func NewBaseHandler(
	manager *server.Manager,
	supervisor *runner.Supervisor,
	store *storage.GormStore,
	hubManager *ws.HubManager,
	backupManager *backup.Manager,
	cfg *config.Config,
) *BaseHandler {
	return &BaseHandler{
		Manager:       manager,
		Supervisor:    supervisor,
		Store:         store,
		HubManager:    hubManager,
		BackupManager: backupManager,
		Config:        cfg,
	}
}

func (h *BaseHandler) checkPermission(r *http.Request, serverID string, check func(*domain.Permission) bool) bool {
	userCtx := r.Context().Value(domain.UserContextKey)
	if userCtx == nil {
		return false
	}
	claims := userCtx.(map[string]string)
	role := claims["role"]
	if role == "admin" {
		return true
	}

	userID := claims["id"]
	perms, err := h.Store.GetPermissions(userID)
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
