# Changelog

- Database-driven backup management and enhanced security:
  - Implemented a new `Backup` model in the database to track files, server associations, and user ownership.
  - Added automatic synchronization in `BackupManager` to discover existing files and register them in the database on startup.
  - Restructured API handlers in `internal/api/handlers/backup.go` to enforce server-based permissions for users.
  - Users can now only manage backups for servers they are authorized to handle; orphaned backups are now restricted to administrators.
  - Introduced a new `UploadBackupModal` in the frontend to allow associating uploaded files with specific servers.
  - Updated the backup creation process to calculate and store the final compressed file size in the database.
- Added real-time Minecraft server player monitoring:
  - Integrated [go-mcstatus](https://github.com/andre-carbajal/go-mcstatus) library to query server status.
  - Displaying online and max player counts in the server list.
  - Added a dedicated "Players" stat card to the server detail view (console page).
  - Updated API and frontend types to support player statistics.
- Unified project versioning:
  - Transitioned `internal/updater.CurrentVersion` from a constant to a variable to support dynamic version injection during compilation.
  - Implemented version injection using Go `ldflags` in both `build.sh` and `build.bat`.
  - Set the default version in the source code to `"dev"` to clearly distinguish development builds from official releases.
  - Updated GitHub Actions to automatically inject the release tag version into the binary, ensuring consistency across all platforms.
- Improve error handling and type definitions across components