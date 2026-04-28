package sdk

import "time"

type Server struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Loader    string    `json:"loader"`
	Port      int       `json:"port"`
	RAM       int       `json:"ram"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type BackupInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type ProgressEvent struct {
	ServerID     string  `json:"serverId"`
	Message      string  `json:"message"`
	Progress     float64 `json:"progress"`
	CurrentBytes int64   `json:"currentBytes"`
	TotalBytes   int64   `json:"totalBytes"`
}

type ServerStats struct {
	CPU  float64 `json:"cpu"`
	RAM  uint64  `json:"ram"`
	Disk int64   `json:"disk"`
}

type UpdateInfo struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseURL      string `json:"release_url"`
}

type PortRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type CreateServerRequest struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Loader    string `json:"loader"`
	Ram       int    `json:"ram"`
	RequestID string `json:"requestId"`
}

type RestoreBackupRequest struct {
	TargetServerID   string `json:"targetServerId,omitempty"`
	NewServerName    string `json:"newServerName,omitempty"`
	NewServerVersion string `json:"newServerVersion,omitempty"`
	NewServerLoader  string `json:"newServerLoader,omitempty"`
	NewServerRam     int    `json:"newServerRam,omitempty"`
}

type LogBufferSettings struct {
	LogBufferSize int `json:"log_buffer_size"`
}

type PublicAddressSettings struct {
	PublicIP string `json:"public_ip"`
}

type NetworkInterfaces struct {
	Interfaces []string `json:"interfaces"`
}

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type Permission struct {
	UserID          string `json:"userId"`
	ServerID        string `json:"serverId"`
	CanViewConsole  bool   `json:"canViewConsole"`
	CanControlPower bool   `json:"canControlPower"`
}
