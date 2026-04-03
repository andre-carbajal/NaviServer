package domain

type ServerRepository interface {
	SaveServer(srv *Server) error
	UpdateServer(id string, name *string, ram *int, customArgs *string) error
	UpdateServerPort(id string, port int) error
	ListServers() ([]Server, error)
	GetServerByID(id string) (*Server, error)
	DeleteServer(id string) error
	UpdateStatus(id string, status string) error
}

type UserRepository interface {
	CreateUser(user *User) error
	GetUserByUsername(username string) (*User, error)
	GetUserByID(id string) (*User, error)
	ListUsers() ([]User, error)
	DeleteUser(id string) error
	SetPermissions(permissions []Permission) error
	GetPermissions(userID string) ([]Permission, error)
	UpdatePassword(userID string, hashedPassword string) error
}

type SettingRepository interface {
	GetSetting(key string) (string, error)
	SetSetting(key string, value string) error
	GetPortRange() (int, int, error)
	SetPortRange(start int, end int) error
}

type PublicLinkRepository interface {
	CreatePublicLink(link *PublicLink) error
	GetPublicLink(token string) (*PublicLink, error)
	GetPublicLinkByServerID(serverID string) (*PublicLink, error)
	DeletePublicLink(token string) error
}

type BackupRepository interface {
	SaveBackup(backup *Backup) error
	UpdateBackup(name string, serverID string) error
	ListBackups(serverID string, userID string, role string) ([]Backup, error)
	GetBackupByName(name string) (*Backup, error)
	DeleteBackup(name string) error
	ListAllBackups() ([]Backup, error)
}

type Repository interface {
	ServerRepository
	UserRepository
	SettingRepository
	PublicLinkRepository
	BackupRepository
}
