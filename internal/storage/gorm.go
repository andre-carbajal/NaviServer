package storage

import (
	"errors"
	"fmt"
	"log"
	"naviserver/internal/config"
	"naviserver/internal/domain"
	"os"
	"strconv"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Server struct {
	ID         string `gorm:"primaryKey"`
	Name       string
	FolderName string
	Version    string
	Loader     string
	Port       int
	RAM        int
	Status     string
	CustomArgs string
	CreatedAt  time.Time

	AutoBackupEnabled       bool   `gorm:"not null;default:false"`
	AutoBackupIntervalValue int    `gorm:"not null;default:24"`
	AutoBackupIntervalUnit  string `gorm:"not null;default:hour"`
	AutoBackupMaxBackups    int    `gorm:"not null;default:10"`
	AutoBackupLastRunAt     *time.Time
}

type Setting struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

type User struct {
	ID       string `gorm:"primaryKey"`
	Username string `gorm:"uniqueIndex"`
	Password string
	Role     string
}

type Permission struct {
	UserID          string `gorm:"primaryKey"`
	ServerID        string `gorm:"primaryKey"`
	CanViewConsole  bool
	CanControlPower bool
}

type PublicLink struct {
	Token    string `gorm:"primaryKey"`
	ServerID string
	Action   string
}

type Backup struct {
	ID         string `gorm:"primaryKey"`
	Name       string `gorm:"uniqueIndex"`
	FileName   string
	ServerID   string `gorm:"index"`
	ServerName string `gorm:"column:server_name;->"`
	Size       int64
	CreatedAt  time.Time
	CreatedBy  string
}

type GormStore struct {
	db *gorm.DB
}

func (s *GormStore) Close() error {
	db, err := s.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

const (
	defaultAutoBackupIntervalValue = 24
	defaultAutoBackupIntervalUnit  = "hour"
	defaultAutoBackupMaxBackups    = 10
)

func normalizeAutoBackupConfig(enabled bool, intervalValue int, intervalUnit string, maxBackups int) (bool, int, string, int) {
	normalizedUnit := intervalUnit
	switch normalizedUnit {
	case "minute", "hour", "day":
	default:
		normalizedUnit = defaultAutoBackupIntervalUnit
	}

	if intervalValue <= 0 {
		intervalValue = defaultAutoBackupIntervalValue
	}

	if maxBackups <= 0 {
		maxBackups = defaultAutoBackupMaxBackups
	}

	return enabled, intervalValue, normalizedUnit, maxBackups
}

func NewGormStore(path string) (*GormStore, error) {
	newLogger := gormlogger.New(
		log.New(os.Stdout, "", log.LstdFlags),
		gormlogger.Config{
			IgnoreRecordNotFoundError: true,
			LogLevel:                  gormlogger.Error,
		},
	)

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: newLogger})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&Server{}, &Setting{}, &User{}, &Permission{}, &PublicLink{}, &Backup{})
	if err != nil {
		return nil, fmt.Errorf("error migrating database: %w", err)
	}

	store := &GormStore{db: db}

	if err := store.initDefaultSettings(); err != nil {
		return nil, fmt.Errorf("error initializing settings: %w", err)
	}

	return store, nil
}

func (s *GormStore) initDefaultSettings() error {
	defaults := map[string]string{
		"port_range_start": "25565",
		"port_range_end":   "25600",
		"log_buffer_size":  strconv.Itoa(config.DefaultLogBufferSize),
	}

	for key, value := range defaults {
		var setting Setting
		result := s.db.First(&setting, "key = ?", key)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				if err := s.db.Create(&Setting{Key: key, Value: value}).Error; err != nil {
					return err
				}
			} else {
				return result.Error
			}
		}
	}

	return nil
}

func (s *GormStore) SaveServer(srv *domain.Server) error {
	enabled, intervalValue, intervalUnit, maxBackups := normalizeAutoBackupConfig(
		srv.AutoBackupEnabled,
		srv.AutoBackupIntervalValue,
		srv.AutoBackupIntervalUnit,
		srv.AutoBackupMaxBackups,
	)

	gormServer := &Server{
		ID:                      srv.ID,
		Name:                    srv.Name,
		FolderName:              srv.FolderName,
		Version:                 srv.Version,
		Loader:                  srv.Loader,
		Port:                    srv.Port,
		RAM:                     srv.RAM,
		Status:                  srv.Status,
		CustomArgs:              srv.CustomArgs,
		CreatedAt:               srv.CreatedAt,
		AutoBackupEnabled:       enabled,
		AutoBackupIntervalValue: intervalValue,
		AutoBackupIntervalUnit:  intervalUnit,
		AutoBackupMaxBackups:    maxBackups,
		AutoBackupLastRunAt:     srv.AutoBackupLastRunAt,
	}

	return s.db.Create(gormServer).Error
}

func (s *GormStore) UpdateServer(id string, name *string, ram *int, customArgs *string) error {
	if name == nil && ram == nil && customArgs == nil {
		return errors.New("no fields to update")
	}

	updates := make(map[string]interface{})
	if name != nil {
		updates["name"] = *name
	}
	if ram != nil {
		updates["ram"] = *ram
	}
	if customArgs != nil {
		updates["custom_args"] = *customArgs
	}

	return s.db.Model(&Server{}).Where("id = ?", id).Updates(updates).Error
}

func (s *GormStore) UpdateServerPort(id string, port int) error {
	return s.db.Model(&Server{}).Where("id = ?", id).Update("port", port).Error
}

func (s *GormStore) UpdateServerAutoBackupConfig(id string, enabled bool, intervalValue int, intervalUnit string, maxBackups int, lastRunAt *time.Time) error {
	_, normalizedIntervalValue, normalizedIntervalUnit, normalizedMaxBackups := normalizeAutoBackupConfig(
		enabled,
		intervalValue,
		intervalUnit,
		maxBackups,
	)

	updates := map[string]interface{}{
		"auto_backup_enabled":        enabled,
		"auto_backup_interval_value": normalizedIntervalValue,
		"auto_backup_interval_unit":  normalizedIntervalUnit,
		"auto_backup_max_backups":    normalizedMaxBackups,
		"auto_backup_last_run_at":    lastRunAt,
	}

	return s.db.Model(&Server{}).Where("id = ?", id).Updates(updates).Error
}

func (s *GormStore) UpdateServerAutoBackupLastRun(id string, lastRunAt time.Time) error {
	return s.db.Model(&Server{}).Where("id = ?", id).Update("auto_backup_last_run_at", &lastRunAt).Error
}

func (s *GormStore) UpdateServerVersion(id, version string) error {
	return s.db.Model(&Server{}).Where("id = ?", id).Update("version", version).Error
}

func (s *GormStore) ListServers() ([]domain.Server, error) {
	var gormServers []Server
	if err := s.db.Find(&gormServers).Error; err != nil {
		return nil, err
	}

	var servers []domain.Server
	for _, gs := range gormServers {
		enabled, intervalValue, intervalUnit, maxBackups := normalizeAutoBackupConfig(
			gs.AutoBackupEnabled,
			gs.AutoBackupIntervalValue,
			gs.AutoBackupIntervalUnit,
			gs.AutoBackupMaxBackups,
		)

		servers = append(servers, domain.Server{
			ID:                      gs.ID,
			Name:                    gs.Name,
			FolderName:              gs.FolderName,
			Version:                 gs.Version,
			Loader:                  gs.Loader,
			Port:                    gs.Port,
			RAM:                     gs.RAM,
			Status:                  gs.Status,
			CustomArgs:              gs.CustomArgs,
			CreatedAt:               gs.CreatedAt,
			AutoBackupEnabled:       enabled,
			AutoBackupIntervalValue: intervalValue,
			AutoBackupIntervalUnit:  intervalUnit,
			AutoBackupMaxBackups:    maxBackups,
			AutoBackupLastRunAt:     gs.AutoBackupLastRunAt,
		})
	}
	return servers, nil
}

func (s *GormStore) GetServerByID(id string) (*domain.Server, error) {
	var gormServer Server
	result := s.db.First(&gormServer, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("error querying server: %w", result.Error)
	}

	enabled, intervalValue, intervalUnit, maxBackups := normalizeAutoBackupConfig(
		gormServer.AutoBackupEnabled,
		gormServer.AutoBackupIntervalValue,
		gormServer.AutoBackupIntervalUnit,
		gormServer.AutoBackupMaxBackups,
	)

	return &domain.Server{
		ID:                      gormServer.ID,
		Name:                    gormServer.Name,
		FolderName:              gormServer.FolderName,
		Version:                 gormServer.Version,
		Loader:                  gormServer.Loader,
		Port:                    gormServer.Port,
		RAM:                     gormServer.RAM,
		Status:                  gormServer.Status,
		CustomArgs:              gormServer.CustomArgs,
		CreatedAt:               gormServer.CreatedAt,
		AutoBackupEnabled:       enabled,
		AutoBackupIntervalValue: intervalValue,
		AutoBackupIntervalUnit:  intervalUnit,
		AutoBackupMaxBackups:    maxBackups,
		AutoBackupLastRunAt:     gormServer.AutoBackupLastRunAt,
	}, nil
}

func (s *GormStore) DeleteServer(id string) error {
	return s.db.Delete(&Server{}, "id = ?", id).Error
}

func (s *GormStore) UpdateStatus(id, status string) error {
	return s.db.Model(&Server{}).Where("id = ?", id).Update("status", status).Error
}

func (s *GormStore) GetSetting(key string) (string, error) {
	var setting Setting
	result := s.db.First(&setting, "key = ?", key)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("setting not found: %s", key)
		}
		return "", result.Error
	}
	return setting.Value, nil
}

func (s *GormStore) SetSetting(key, value string) error {
	var setting Setting
	result := s.db.First(&setting, "key = ?", key)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return s.db.Create(&Setting{Key: key, Value: value}).Error
		}
		return result.Error
	}

	return s.db.Model(&setting).Update("value", value).Error
}

func (s *GormStore) GetPortRange() (int, int, error) {
	startStr, err := s.GetSetting("port_range_start")
	if err != nil {
		return 0, 0, err
	}

	endStr, err := s.GetSetting("port_range_end")
	if err != nil {
		return 0, 0, err
	}

	start, err := strconv.Atoi(startStr)
	if err != nil {
		return 0, 0, fmt.Errorf("error parsing port_range_start: %w", err)
	}

	end, err := strconv.Atoi(endStr)
	if err != nil {
		return 0, 0, fmt.Errorf("error parsing port_range_end: %w", err)
	}

	return start, end, nil
}

func (s *GormStore) SetPortRange(start, end int) error {
	if start <= 0 || end <= 0 || start > end {
		return fmt.Errorf("invalid port range: %d-%d", start, end)
	}

	if err := s.SetSetting("port_range_start", fmt.Sprintf("%d", start)); err != nil {
		return err
	}

	if err := s.SetSetting("port_range_end", fmt.Sprintf("%d", end)); err != nil {
		return err
	}

	return nil
}

func (s *GormStore) CreateUser(user *domain.User) error {
	gormUser := &User{
		ID:       user.ID,
		Username: user.Username,
		Password: user.Password,
		Role:     user.Role,
	}
	return s.db.Create(gormUser).Error
}

func (s *GormStore) GetUserByUsername(username string) (*domain.User, error) {
	var gormUser User
	err := s.db.Where("username = ?", username).First(&gormUser).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &domain.User{
		ID:       gormUser.ID,
		Username: gormUser.Username,
		Password: gormUser.Password,
		Role:     gormUser.Role,
	}, nil
}

func (s *GormStore) GetUserByID(id string) (*domain.User, error) {
	var gormUser User
	err := s.db.Where("id = ?", id).First(&gormUser).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &domain.User{
		ID:       gormUser.ID,
		Username: gormUser.Username,
		Password: gormUser.Password,
		Role:     gormUser.Role,
	}, nil
}

func (s *GormStore) ListUsers() ([]domain.User, error) {
	var gormUsers []User
	if err := s.db.Find(&gormUsers).Error; err != nil {
		return nil, err
	}
	var users []domain.User
	for _, u := range gormUsers {
		users = append(users, domain.User{
			ID:       u.ID,
			Username: u.Username,
			Role:     u.Role,
		})
	}
	return users, nil
}

func (s *GormStore) DeleteUser(id string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&User{}, "id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Delete(&Permission{}, "user_id = ?", id).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *GormStore) UpdatePassword(userID, hashedPassword string) error {
	return s.db.Model(&User{}).Where("id = ?", userID).Update("password", hashedPassword).Error
}

func (s *GormStore) SetPermissions(permissions []domain.Permission) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if len(permissions) == 0 {
			return nil
		}
		userID := permissions[0].UserID
		if err := tx.Delete(&Permission{}, "user_id = ?", userID).Error; err != nil {
			return err
		}

		for _, p := range permissions {
			gormPerm := Permission{
				UserID:          p.UserID,
				ServerID:        p.ServerID,
				CanViewConsole:  p.CanViewConsole,
				CanControlPower: p.CanControlPower,
			}
			if err := tx.Save(&gormPerm).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *GormStore) GetPermissions(userID string) ([]domain.Permission, error) {
	var gormPerms []Permission
	if err := s.db.Where("user_id = ?", userID).Find(&gormPerms).Error; err != nil {
		return nil, err
	}
	var perms []domain.Permission
	for _, p := range gormPerms {
		perms = append(perms, domain.Permission{
			UserID:          p.UserID,
			ServerID:        p.ServerID,
			CanViewConsole:  p.CanViewConsole,
			CanControlPower: p.CanControlPower,
		})
	}
	return perms, nil
}

func (s *GormStore) CreatePublicLink(link *domain.PublicLink) error {
	return s.db.Create(&PublicLink{
		Token:    link.Token,
		ServerID: link.ServerID,
		Action:   link.Action,
	}).Error
}

func (s *GormStore) GetPublicLink(token string) (*domain.PublicLink, error) {
	var l PublicLink
	if err := s.db.Where("token = ?", token).First(&l).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &domain.PublicLink{
		Token:    l.Token,
		ServerID: l.ServerID,
		Action:   l.Action,
	}, nil
}

func (s *GormStore) GetPublicLinkByServerID(serverID string) (*domain.PublicLink, error) {
	var l PublicLink
	if err := s.db.Where("server_id = ?", serverID).First(&l).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &domain.PublicLink{
		Token:    l.Token,
		ServerID: l.ServerID,
		Action:   l.Action,
	}, nil
}

func (s *GormStore) DeletePublicLink(token string) error {
	return s.db.Delete(&PublicLink{}, "token = ?", token).Error
}

func (s *GormStore) SetLogBufferSize(size int) error {
	if size < 0 {
		return fmt.Errorf("invalid log buffer size: %d", size)
	}
	return s.SetSetting("log_buffer_size", fmt.Sprintf("%d", size))
}

func (s *GormStore) SaveBackup(backup *domain.Backup) error {
	gormBackup := &Backup{
		ID:        backup.ID,
		Name:      backup.Name,
		FileName:  backup.FileName,
		ServerID:  backup.ServerID,
		Size:      backup.Size,
		CreatedAt: backup.CreatedAt,
		CreatedBy: backup.CreatedBy,
	}
	return s.db.Save(gormBackup).Error
}

func (s *GormStore) UpdateBackup(name, serverID string) error {
	return s.db.Model(&Backup{}).Where("name = ?", name).Update("server_id", serverID).Error
}

func (s *GormStore) ListBackups(serverID, userID, role string) ([]domain.Backup, error) {
	var gormBackups []Backup
	query := s.db.Model(&Backup{}).
		Select("backups.*, servers.name as server_name").
		Joins("left join servers on backups.server_id = servers.id")

	if role != "admin" {
		var permittedServerIDs []string
		s.db.Model(&Permission{}).Where("user_id = ? AND can_control_power = ?", userID, true).Pluck("server_id", &permittedServerIDs)

		if serverID != "" {
			isPermitted := false
			for _, id := range permittedServerIDs {
				if id == serverID {
					isPermitted = true
					break
				}
			}
			if !isPermitted {
				return []domain.Backup{}, nil
			}
			query = query.Where("backups.server_id = ?", serverID)
		} else {
			query = query.Where("backups.server_id IN ?", permittedServerIDs)
		}
	} else {
		if serverID != "" {
			query = query.Where("backups.server_id = ?", serverID)
		}
	}

	query = query.Order("backups.created_at desc")

	if err := query.Scan(&gormBackups).Error; err != nil {
		return nil, err
	}

	var backups []domain.Backup
	for _, b := range gormBackups {
		backups = append(backups, domain.Backup{
			ID:         b.ID,
			Name:       b.Name,
			FileName:   b.FileName,
			ServerID:   b.ServerID,
			ServerName: b.ServerName,
			Size:       b.Size,
			CreatedAt:  b.CreatedAt,
			CreatedBy:  b.CreatedBy,
		})
	}
	return backups, nil
}

func (s *GormStore) GetBackupByName(name string) (*domain.Backup, error) {
	var b Backup
	if err := s.db.Where("name = ?", name).First(&b).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &domain.Backup{
		ID:        b.ID,
		Name:      b.Name,
		FileName:  b.FileName,
		ServerID:  b.ServerID,
		Size:      b.Size,
		CreatedAt: b.CreatedAt,
		CreatedBy: b.CreatedBy,
	}, nil
}

func (s *GormStore) DeleteBackup(name string) error {
	return s.db.Delete(&Backup{}, "name = ?", name).Error
}

func (s *GormStore) ListAllBackups() ([]domain.Backup, error) {
	var gormBackups []Backup
	err := s.db.Model(&Backup{}).
		Select("backups.*, servers.name as server_name").
		Joins("left join servers on backups.server_id = servers.id").
		Order("backups.created_at desc").
		Scan(&gormBackups).Error

	if err != nil {
		return nil, err
	}
	var backups []domain.Backup
	for _, b := range gormBackups {
		backups = append(backups, domain.Backup{
			ID:         b.ID,
			Name:       b.Name,
			FileName:   b.FileName,
			ServerID:   b.ServerID,
			ServerName: b.ServerName,
			Size:       b.Size,
			CreatedAt:  b.CreatedAt,
			CreatedBy:  b.CreatedBy,
		})
	}
	return backups, nil
}

func (s *GormStore) ListBackupsByServerID(serverID string) ([]domain.Backup, error) {
	var gormBackups []Backup
	if err := s.db.
		Where("server_id = ?", serverID).
		Order("created_at asc").
		Find(&gormBackups).Error; err != nil {
		return nil, err
	}

	backups := make([]domain.Backup, 0, len(gormBackups))
	for _, b := range gormBackups {
		backups = append(backups, domain.Backup{
			ID:         b.ID,
			Name:       b.Name,
			FileName:   b.FileName,
			ServerID:   b.ServerID,
			ServerName: b.ServerName,
			Size:       b.Size,
			CreatedAt:  b.CreatedAt,
			CreatedBy:  b.CreatedBy,
		})
	}

	return backups, nil
}
