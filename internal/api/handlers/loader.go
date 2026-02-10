package handlers

import (
	"encoding/json"
	"net/http"

	"naviger/internal/loader"
)

type LoaderHandler struct{}

func NewLoaderHandler() *LoaderHandler {
	return &LoaderHandler{}
}

func (h *LoaderHandler) GetVersions(w http.ResponseWriter, r *http.Request) {
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

func (h *LoaderHandler) GetAvailable(w http.ResponseWriter, r *http.Request) {
	loaders := loader.GetAvailableLoaders()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loaders)
}
