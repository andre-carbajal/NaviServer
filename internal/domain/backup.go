package domain

import "time"

type Backup struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	FileName   string    `json:"fileName"`
	ServerID   string    `json:"serverId"`
	ServerName string    `json:"serverName"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"createdAt"`
	CreatedBy  string    `json:"createdBy"`
}
