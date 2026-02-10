package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"naviger/internal/config"
	"naviger/internal/storage"
	"naviger/internal/updater"
	"naviger/internal/ws"
)

type SettingsHandler struct {
	Store      *storage.GormStore
	HubManager *ws.HubManager
}

func NewSettingsHandler(store *storage.GormStore, hubManager *ws.HubManager) *SettingsHandler {
	return &SettingsHandler{
		Store:      store,
		HubManager: hubManager,
	}
}

func (h *SettingsHandler) GetPortRange(w http.ResponseWriter, r *http.Request) {
	start, end, err := h.Store.GetPortRange()
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

func (h *SettingsHandler) SetPortRange(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Start int `json:"start"`
		End   int `json:"end"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.Store.SetPortRange(req.Start, req.End); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "updated"}`))
}

func (h *SettingsHandler) GetLogBufferSize(w http.ResponseWriter, r *http.Request) {
	val, err := h.Store.GetSetting("log_buffer_size")
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

func (h *SettingsHandler) SetLogBufferSize(w http.ResponseWriter, r *http.Request) {
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
	if err := h.Store.SetLogBufferSize(req.LogBufferSize); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if h.HubManager != nil {
		h.HubManager.SetDefaultHistorySize(req.LogBufferSize)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"updated"}`))
}

func (h *SettingsHandler) GetPublicIP(w http.ResponseWriter, r *http.Request) {
	val, err := h.Store.GetSetting("public_ip")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"public_ip": "localhost"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"public_ip": val})
}

func (h *SettingsHandler) SetPublicIP(w http.ResponseWriter, r *http.Request) {
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

	if err := h.Store.SetSetting("public_ip", req.PublicIP); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"updated"}`))
}

func (h *SettingsHandler) CheckUpdates(w http.ResponseWriter, r *http.Request) {
	updateInfo, err := updater.CheckForUpdates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updateInfo)
}
