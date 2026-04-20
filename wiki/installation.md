# Option 1: install from latest release (Linux/macOS)

```bash
./install.sh
```

The script:

- Detects OS and architecture.
- Downloads the latest release ZIP from GitHub.
- Lets you choose installation mode:
    - `Headless (Service/Daemon)`
    - `Desktop (App/Shortcut)`
- Installs binaries in `/opt/naviserver`.
- Installs `naviserver-cli` in `/usr/local/bin` when available.
- Configures background execution:
    - Linux: `systemd` service `naviserver`
    - macOS: `launchd` agent `com.naviserver.server`

# Option 2: build from source

```bash
./build.sh
```

Build individual binaries:

```bash
go build ./cmd/server
go build ./cmd/cli
```

# Uninstall (Linux/macOS)

```bash
./uninstall.sh
```

Useful flags:

- `--keep-data` or `-k`: preserve the data directory.
- `--yes` or `-y`: run non-interactively.

# Upgrade from NaviServer 1.x.x (legacy `naviger` data)

If you are coming from 1.x.x data/layout (`naviger`), run migration before first start of the new version.

Important notes:

- Run migration only once, before launching new NaviServer for the first time.
- The scripts create an automatic backup in your home directory.
- Data directory is moved from `naviger` to `naviserver`.
- Secret file is renamed from `.naviger_secret` to `.naviserver_secret`.
- If target `naviserver` directory already exists, data move is skipped to avoid overwrite.

Linux/macOS:

```bash
chmod +x migration/migrate.sh
./migration/migrate.sh
```

Windows:

```bat
migration\migrate.bat
```

After migration:

- Continue with normal installation (`./install.sh` on Linux/macOS or Windows installer).
- For headless mode, confirm service/agent is running after install.

