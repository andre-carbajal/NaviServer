package domain

import "time"

type Server struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	FolderName  string      `json:"folderName"`
	Version     string      `json:"version"`
	Loader      string      `json:"loader"`
	Port        int         `json:"port"`
	RAM         int         `json:"ram"`
	Status      string      `json:"status"`
	CustomArgs  string      `json:"customArgs"`
	CreatedAt   time.Time   `json:"created_at"`
	Permissions *Permission `json:"permissions,omitempty"`
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
	CPU           float64 `json:"cpu"`
	RAM           uint64  `json:"ram"`
	Disk          int64   `json:"disk"`
	OnlinePlayers int     `json:"onlinePlayers"`
	MaxPlayers    int     `json:"maxPlayers"`
}
