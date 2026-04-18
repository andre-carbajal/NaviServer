package handlers

import (
	"encoding/json"
	"naviserver/internal/updater"
	"net"
	"net/http"
	"os"
	"time"
)

type SystemHandler struct {
	*BaseHandler
}

func (h *SystemHandler) HandleGetNetworkInterfaces(w http.ResponseWriter, r *http.Request) {
	var addresses []string

	interfaces, err := net.Interfaces()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			if ip.To4() != nil {
				addresses = append(addresses, ip.String())
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{"interfaces": addresses})
}

func (h *SystemHandler) HandleRestartDaemon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status": "restarting"}`))
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	go func() {
		time.Sleep(300 * time.Millisecond)
		os.Exit(1)
	}()
}

func (h *SystemHandler) HandleCheckUpdates(w http.ResponseWriter, r *http.Request) {
	updateInfo, err := updater.CheckForUpdates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updateInfo)
}

func (h *SystemHandler) HandleGetVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"version": updater.CurrentVersion,
	})
}
