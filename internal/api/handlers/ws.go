package handlers

import (
	"net/http"
)

type WSHandler struct {
	*BaseHandler
}

func (h *WSHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	hub := h.HubManager.GetHub(id)
	hub.ServeWs(w, r)
}
