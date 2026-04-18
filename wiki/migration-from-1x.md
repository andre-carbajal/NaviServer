If your current installation still uses legacy `naviger` paths/binaries, migrate first before running the new version.

# What changes in migration

- App name: `naviger` -> `naviserver`
- Binaries: `naviger-server` -> `naviserver-server`, `naviger-cli` -> `naviserver-cli`
- Data directories:
  - Linux: `~/.config/naviger` -> `~/.config/naviserver`
  - macOS: `~/Library/Application Support/naviger` -> `~/Library/Application Support/naviserver`
  - Windows: `%AppData%\naviger` -> `%AppData%\naviserver`
- Secret key file: `.naviger_secret` -> `.naviserver_secret`

# When to run migration

Run migration only once:
- after downloading the new version,
- before first launch of new NaviServer,
- and before running the new installer.

# Linux/macOS migration

From repository root:

```bash
chmod +x migration/migrate.sh
./migration/migrate.sh
```

What `migration/migrate.sh` does:
- Validates that old data exists.
- Creates automatic backup in your home directory (`naviger_backup_<timestamp>.tar.gz` or `.zip`).
- Stops old services/agents.
- Moves old data directory to `naviserver`.
- Renames secret file to `.naviserver_secret`.
- Cleans old symlinks/legacy install paths.
- Runs `install.sh` automatically and verifies/restarts service when applicable.

# Windows migration

Run:

```bat
migration\migrate.bat
```

What `migration/migrate.bat` does:
- Validates old data directory.
- Creates backup (`naviger_backup_<timestamp>.zip`) in user home.
- Stops old processes.
- Moves data and renames secret file.
- Downloads latest Windows installer and runs it.

# Safety behavior

- If target `naviserver` data directory already exists, scripts skip moving data to avoid accidental overwrite.
- If automatic installer step fails, migrated data and backup remain available.
- Keep the generated backup until you validate servers, users, and backups after upgrade.

# After migration

1. Complete installation (if not done automatically).
2. Start NaviServer in your preferred mode.
3. Validate expected data:
   - server list
   - user accounts/login
   - backups list
4. If headless, verify service/agent status (`systemctl` on Linux, `launchctl` on macOS).

