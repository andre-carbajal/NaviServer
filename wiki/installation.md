# Installation

NaviServer publishes installers for Windows, macOS and Linux. Choose the method for your operating system below.

> Always download releases from the official [GitHub Releases](https://github.com/andre-carbajal/NaviServer/releases)
> page. The examples below use the latest release.

## Remote installation (Linux/macOS)

The same installer supports both Linux and macOS. It detects the operating system and architecture, downloads the latest
ZIP release, and asks whether to install desktop or headless mode:

```bash
curl -sSL https://raw.githubusercontent.com/andre-carbajal/NaviServer/main/install.sh | sh
```

The script may request `sudo` for `/opt/naviserver`, `/usr/local/bin`, and service configuration. On Linux, headless
mode creates a `systemd` service named `naviserver`; desktop mode creates a desktop entry.

For an auditable installation, download and inspect the script first:

```bash
curl -fsSL -o /tmp/naviserver-install.sh https://raw.githubusercontent.com/andre-carbajal/NaviServer/main/install.sh
less /tmp/naviserver-install.sh
bash /tmp/naviserver-install.sh
rm -f /tmp/naviserver-install.sh
```

## Linux

### Ubuntu and Debian (`.deb`)

Download the `NaviServer-<version>-linux.deb` asset from the releases page and install it:

```bash
sudo apt install ./NaviServer-<version>-linux.deb
```

The package installs the application and its desktop integration. Start NaviServer from the applications menu or run the
installed server binary as needed.

### Other Linux distributions (`.zip`)

Download `NaviServer-<version>-linux.zip`, extract it, and run the included binaries. For a system-wide installation,
the supported installer script installs into `/opt/naviserver` and exposes the CLI in `/usr/local/bin`.

### Linux requirements and data

- Official binaries target `x86_64`.
- The remote installer requires `curl` and `unzip`.
- Application files are installed in `/opt/naviserver`.
- The CLI is linked from `/usr/local/bin/naviserver-cli` when available.
- User data, servers, backups and secrets are stored under `~/.config/naviserver`.

## macOS

### Package installer (`.pkg`)

Download `NaviServer-<version>-macos.pkg` and open it. Follow the installer prompts. The package installs the
application and registers the required paths.

### Application archive (`.zip`)

Download `NaviServer-<version>-macos.zip`, extract `NaviServer.app`, and move it to `/Applications` or `~/Applications`.

### Headless mode

Use the remote installer and select `Headless (Service/Daemon)`. The installer creates the `launchd` agent
`com.naviserver.server` for the current user. Desktop mode installs the application bundle instead.

### macOS requirements and data

- Use a release matching the Mac architecture supported by the published asset.
- The remote installer requires `curl` and `unzip`.
- User data, servers, backups and secrets are stored under `~/Library/Application Support/naviserver`.

## Windows

### Installer (`.exe`)

Download `NaviServer-<version>-windows.exe` from the releases page and run it. Follow the wizard to select the
installation directory and optional shortcuts.

### Portable archive (`.zip`)

Download `NaviServer-<version>-windows.zip`, extract it to a directory, and run the included server or CLI executable.
Create your own shortcut or scheduled task if you need NaviServer to start automatically.

Windows does not use `install.sh`; use the `.exe` installer or the ZIP archive instead. The migration helper is
`migration\migrate.bat` and can be run from Command Prompt.

Windows user data, servers, backups and secrets are stored under `%AppData%\naviserver`.

## Uninstallation

Before uninstalling, stop active servers and create an independent backup if the data matters. The two modes below have
different effects.

### Option A: remove NaviServer but keep servers and data (recommended)

On Linux or macOS, run the repository's uninstall script with `--keep-data`:

```bash
curl -fsSL -o /tmp/naviserver-uninstall.sh https://raw.githubusercontent.com/andre-carbajal/NaviServer/main/uninstall.sh
bash /tmp/naviserver-uninstall.sh --keep-data
rm -f /tmp/naviserver-uninstall.sh
```

Or, from a local checkout:

```bash
bash uninstall.sh --keep-data
```

The script stops and removes the `systemd` service or `launchd` agent, removes application files and CLI links, but
preserves:

- Linux: `~/.config/naviserver`
- macOS: `~/Library/Application Support/naviserver`

This includes server instances, backups, configuration and secret files. Reinstall NaviServer using the normal
installation method and it will use the preserved data directory.

On Windows, uninstall NaviServer from **Settings > Apps** or run the uninstaller installed with the `.exe`. Do not
delete `%AppData%\naviserver`; that directory contains the servers and user data. If using the portable ZIP, delete only
the extracted application directory and keep `%AppData%\naviserver`.

### Option B: remove NaviServer and all servers/data

This option permanently removes the application and user data. Back up the data first if it may be needed later.

On Linux or macOS:

```bash
bash uninstall.sh
```

The script asks for confirmation and removes services/agents, application files, CLI links and the user data directory.
With `--yes`, confirmation is skipped and no automatic backup is created:

```bash
bash uninstall.sh --yes
```

On Windows:

1. Uninstall NaviServer from **Settings > Apps** or run the installed uninstaller.
2. After confirming that the data is no longer needed, remove `%AppData%\naviserver`.
3. For a portable installation, remove the extracted application directory as well.

Deleting the data directory removes server instances, backups, configuration and secrets. This cannot be undone without
a backup.

## Upgrade from NaviServer 1.x

If your installation still uses the legacy `naviger` layout, migrate before starting NaviServer for the first time:

- Linux/macOS: `bash migration/migrate.sh`
- Windows: `migration\migrate.bat`

The migration creates a backup, moves legacy data to the new `naviserver` location when the destination does not already
exist, and renames the legacy secret file. Run it only once before the new installation or first launch.
