# Changelog

- **Project Rebranding: Naviger is now NaviServer**:
  - Renamed all application occurrences, binaries, and identifiers from `Naviger` to `NaviServer` across the entire codebase.
  - Updated Go module name to `naviserver` and refreshed all internal imports and build configurations.
  - Migrated environment variables (e.g., `NAVIGER_DEV` -> `NAVISERVER_DEV`, `NAVIGER_SECRET_KEY` -> `NAVISERVER_SECRET_KEY`) and system paths.
  - **IMPORTANT MIGRATION NOTE**: Users of previous versions MUST run the provided `migration/migrate.sh` (Linux/macOS) or `migration/migrate.bat` (Windows) *before* installing the new version. These scripts ensure that existing servers, backups, and the database are moved from `naviger` to `naviserver` data directories automatically to prevent data loss or duplicate installations.
  - Added dedicated migration scripts (in `migration/` directory) to handle service termination, data movement, and cleanup of legacy symlinks and installations.

- Database-driven backup management and enhanced security:
    - Implemented a new `Backup` model in the database to track files, server associations, and user ownership.
    - Added automatic synchronization in `BackupManager` to discover existing files and register them in the database on
      startup.
    - Restructured API handlers in `internal/api/handlers/backup.go` to enforce server-based permissions for users.
    - Users can now only manage backups for servers they are authorized to handle; orphaned backups are now restricted
      to administrators.
    - Introduced a new `UploadBackupModal` in the frontend to allow associating uploaded files with specific servers.
    - Updated the backup creation process to calculate and store the final compressed file size in the database.
- Added real-time Minecraft server player monitoring:
    - Integrated [go-mcstatus](https://github.com/andre-carbajal/go-mcstatus) library to query server status.
    - Displaying online and max player counts in the server list.
    - Added a dedicated "Players" stat card to the server detail view (console page).
    - Updated API and frontend types to support player statistics.
- Unified project versioning:
    - Transitioned `internal/updater.CurrentVersion` from a constant to a variable to support dynamic version injection
      during compilation.
    - Implemented version injection using Go `ldflags` in both `build.sh` and `build.bat`.
    - Set the default version in the source code to `"dev"` to clearly distinguish development builds from official
      releases.
    - Updated GitHub Actions to automatically inject the release tag version into the binary, ensuring consistency
      across all platforms.
- Improve error handling and type definitions across components
- Improved `SyncBackups` to perform bidirectional synchronization, removing ghost database records when physical files
  are deleted externally.