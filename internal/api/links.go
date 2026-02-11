package api

import (
	"encoding/json"
	"net/http"

	"naviger/internal/domain"

	"github.com/google/uuid"
)

func (api *Server) handleCreatePublicLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ServerID string `json:"serverId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ServerID == "" {
		http.Error(w, "ServerID required", http.StatusBadRequest)
		return
	}

	existing, err := api.Store.GetPublicLinkByServerID(req.ServerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if existing != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(existing)
		return
	}

	token := uuid.NewString()
	link := &domain.PublicLink{
		Token:    token,
		ServerID: req.ServerID,
		Action:   "control",
	}

	if err := api.Store.CreatePublicLink(link); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(link)
}

func (api *Server) handleDeletePublicLink(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		http.Error(w, "Missing Token", http.StatusBadRequest)
		return
	}

	if err := api.Store.DeletePublicLink(token); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (api *Server) handleGetPublicServerInfo(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		http.Error(w, "Missing Token", http.StatusBadRequest)
		return
	}

	link, err := api.Store.GetPublicLink(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if link == nil {
		http.Error(w, "Invalid or expired link", http.StatusNotFound)
		return
	}

	srv, err := api.Manager.GetServer(link.ServerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if srv == nil {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	status := srv.Status

	response := map[string]interface{}{
		"name":    srv.Name,
		"version": srv.Version,
		"loader":  srv.Loader,
		"status":  status,
		"id":      srv.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (api *Server) handleAccessPublicLink(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		http.Error(w, "Missing Token", http.StatusBadRequest)
		return
	}

	var req struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	link, err := api.Store.GetPublicLink(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if link == nil {
		http.Error(w, "Invalid or expired link", http.StatusNotFound)
		return
	}

	if link.Action == "control" {
		if req.Action == "start" {
			if err := api.Supervisor.StartServer(link.ServerID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else if req.Action == "stop" {
			if err := api.Supervisor.StopServer(link.ServerID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "Unsupported action", http.StatusBadRequest)
			return
		}
	} else if link.Action == "start" {
		if req.Action == "start" {
			if err := api.Supervisor.StartServer(link.ServerID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "This link only allows starting", http.StatusForbidden)
			return
		}
	} else {
		http.Error(w, "Invalid link configuration", http.StatusForbidden)
		return
	}

	w.Write([]byte(`{"status": "executed"}`))
}
