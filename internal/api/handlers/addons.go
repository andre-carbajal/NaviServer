package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"naviserver/internal/addons"
	"naviserver/internal/domain"
)

type AddonsHandler struct {
	*BaseHandler
}

func (h *AddonsHandler) ensureManager(w http.ResponseWriter) bool {
	if h.AddonsManager != nil {
		return true
	}
	http.Error(w, "Addon manager is not configured", http.StatusServiceUnavailable)
	return false
}

func (h *AddonsHandler) canManageAddons(r *http.Request, serverID string) bool {
	return h.checkPermission(r, serverID, func(p *domain.Permission) bool {
		return p.CanViewConsole
	})
}

func (h *AddonsHandler) HandleListAddons(w http.ResponseWriter, r *http.Request) {
	if !h.ensureManager(w) {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing server id", http.StatusBadRequest)
		return
	}
	if !h.canManageAddons(r, id) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	result, err := h.AddonsManager.ListAddons(r.Context(), id)
	if err != nil {
		h.writeAddonError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *AddonsHandler) HandleSyncAddons(w http.ResponseWriter, r *http.Request) {
	if !h.ensureManager(w) {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing server id", http.StatusBadRequest)
		return
	}
	if !h.canManageAddons(r, id) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	result, err := h.AddonsManager.SyncAddons(r.Context(), id)
	if err != nil {
		h.writeAddonError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *AddonsHandler) HandleSearchAddons(w http.ResponseWriter, r *http.Request) {
	if !h.ensureManager(w) {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing server id", http.StatusBadRequest)
		return
	}
	if !h.canManageAddons(r, id) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req addons.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	results, err := h.AddonsManager.SearchAddons(
		r.Context(),
		id,
		req.Query,
		req.Source,
		req.Offset,
		req.Limit,
	)
	if err != nil {
		h.writeAddonError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *AddonsHandler) HandleAddonVersions(w http.ResponseWriter, r *http.Request) {
	if !h.ensureManager(w) {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing server id", http.StatusBadRequest)
		return
	}
	if !h.canManageAddons(r, id) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req addons.VersionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	result, err := h.AddonsManager.ListAddonVersions(r.Context(), id, req)
	if err != nil {
		h.writeAddonError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *AddonsHandler) HandleInstallAddon(w http.ResponseWriter, r *http.Request) {
	if !h.ensureManager(w) {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing server id", http.StatusBadRequest)
		return
	}
	if !h.canManageAddons(r, id) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req addons.InstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.AddonsManager.InstallAddon(r.Context(), id, req); err != nil {
		h.writeAddonError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *AddonsHandler) HandleInstallPreview(w http.ResponseWriter, r *http.Request) {
	if !h.ensureManager(w) {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing server id", http.StatusBadRequest)
		return
	}
	if !h.canManageAddons(r, id) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req addons.InstallPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	result, err := h.AddonsManager.PreviewInstallAddon(r.Context(), id, req)
	if err != nil {
		h.writeAddonError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *AddonsHandler) HandleDeleteAddon(w http.ResponseWriter, r *http.Request) {
	if !h.ensureManager(w) {
		return
	}
	id := r.PathValue("id")
	addonID := r.PathValue("addonId")
	if id == "" || addonID == "" {
		http.Error(w, "Missing id", http.StatusBadRequest)
		return
	}
	if !h.canManageAddons(r, id) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.AddonsManager.DeleteAddon(id, addonID); err != nil {
		h.writeAddonError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AddonsHandler) HandleSetAddonDisabled(w http.ResponseWriter, r *http.Request) {
	if !h.ensureManager(w) {
		return
	}
	id := r.PathValue("id")
	addonID := r.PathValue("addonId")
	if id == "" || addonID == "" {
		http.Error(w, "Missing id", http.StatusBadRequest)
		return
	}
	if !h.canManageAddons(r, id) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var body struct {
		Disabled bool `json:"disabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.AddonsManager.SetAddonDisabled(id, addonID, body.Disabled); err != nil {
		h.writeAddonError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *AddonsHandler) HandleUpdateAddon(w http.ResponseWriter, r *http.Request) {
	if !h.ensureManager(w) {
		return
	}
	id := r.PathValue("id")
	addonID := r.PathValue("addonId")
	if id == "" || addonID == "" {
		http.Error(w, "Missing id", http.StatusBadRequest)
		return
	}
	if !h.canManageAddons(r, id) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var body struct {
		IncludeDependencies bool `json:"includeDependencies"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	if err := h.AddonsManager.UpdateAddon(r.Context(), id, addonID, body.IncludeDependencies); err != nil {
		h.writeAddonError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *AddonsHandler) HandleUpdateAllAddons(w http.ResponseWriter, r *http.Request) {
	if !h.ensureManager(w) {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing server id", http.StatusBadRequest)
		return
	}
	if !h.canManageAddons(r, id) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var body struct {
		IncludeDependencies bool `json:"includeDependencies"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	result, err := h.AddonsManager.UpdateAllAddons(r.Context(), id, body.IncludeDependencies)
	if err != nil {
		h.writeAddonError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *AddonsHandler) writeAddonError(w http.ResponseWriter, err error) {
	msg := strings.ToLower(err.Error())
	switch {
	case addons.IsCurseForgeKeyMissing(err):
		http.Error(w, err.Error(), http.StatusFailedDependency)
	case strings.Contains(msg, "forbidden"):
		http.Error(w, err.Error(), http.StatusForbidden)
	case strings.Contains(msg, "not found"):
		http.Error(w, err.Error(), http.StatusNotFound)
	case strings.Contains(msg, "must be stopped"):
		http.Error(w, err.Error(), http.StatusConflict)
	case strings.Contains(msg, "invalid") || strings.Contains(msg, "required"):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		var pathErr *json.SyntaxError
		if errors.As(err, &pathErr) {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
