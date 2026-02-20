package handlers

import (
	"net/http"
)

type WSHandler struct {
	*BaseHandler
}

func (h *WSHandler) HandleConsole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	hub := h.HubManager.GetHub(id)
	hub.ServeWs(w, r)
}

func (h *WSHandler) HandleProgress(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}
	hub := h.HubManager.GetHub(id)
	hub.ServeWs(w, r)
}
