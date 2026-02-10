package handlers

import (
	"net/http"

	"naviger/internal/ws"
)

type WSHandler struct {
	HubManager *ws.HubManager
}

func NewWSHandler(hubManager *ws.HubManager) *WSHandler {
	return &WSHandler{
		HubManager: hubManager,
	}
}

func (h *WSHandler) Console(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	hub := h.HubManager.GetHub(id)
	hub.ServeWs(w, r)
}

func (h *WSHandler) Progress(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}
	hub := h.HubManager.GetHub(id)
	hub.ServeWs(w, r)
}
