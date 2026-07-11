package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"naviserver/internal/loader"
)

type LoadersHandler struct {
	*BaseHandler
}

func (h *LoadersHandler) HandleGetLoaders(w http.ResponseWriter, r *http.Request) {
	loaders := loader.GetAvailableLoaders()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loaders)
}

func (h *LoadersHandler) HandleGetLoaderVersions(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "Missing loader name", http.StatusBadRequest)
		return
	}

	includeSnapshots, _ := strconv.ParseBool(r.URL.Query().Get("includeSnapshots"))
	includeUnstable, _ := strconv.ParseBool(r.URL.Query().Get("includeUnstable"))
	options := loader.LoaderOptions{
		MCVersion:        r.URL.Query().Get("mcVersion"),
		IncludeSnapshots: includeSnapshots,
		IncludeUnstable:  includeUnstable,
	}
	versions, err := loader.GetLoaderVersions(name, options)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, loader.ErrBedrockPlatformUnsupported) {
			status = http.StatusUnprocessableEntity
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versions)
}

func (h *LoadersHandler) HandleGetLoaderMetadata(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "Missing loader name", http.StatusBadRequest)
		return
	}

	includeSnapshots, _ := strconv.ParseBool(r.URL.Query().Get("includeSnapshots"))
	includeUnstable, _ := strconv.ParseBool(r.URL.Query().Get("includeUnstable"))
	options := loader.LoaderOptions{
		MCVersion:        r.URL.Query().Get("mcVersion"),
		IncludeSnapshots: includeSnapshots,
		IncludeUnstable:  includeUnstable,
		BuildVersion:     r.URL.Query().Get("buildVersion"),
		LoaderVersion:    r.URL.Query().Get("loaderVersion"),
		InstallerVersion: r.URL.Query().Get("installerVersion"),
	}
	md, err := loader.GetLoaderMetadata(name, options)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, loader.ErrBedrockPlatformUnsupported) {
			status = http.StatusUnprocessableEntity
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(md)
}
