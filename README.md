# ⛏️ NaviServer

![GitHub release](https://img.shields.io/github/v/release/andre-carbajal/NaviServer?style=flat-square)
![License](https://img.shields.io/github/license/andre-carbajal/NaviServer?style=flat-square)

NaviServer is a control panel and daemon for running Minecraft servers. It lets you create, configure, start, stop and
monitor multiple server instances from one place, without manually managing Java installations, server files or backups.

It is designed for both personal servers and small self-hosted environments. Use the browser dashboard for everyday
operations, the CLI for automation, or the interactive TUI when working directly from a terminal.

## What NaviServer does

NaviServer manages the complete lifecycle of each Minecraft server:

- Create instances for Vanilla, Paper, Fabric, Forge or NeoForge.
- Download and manage the required loader versions and Java runtimes.
- Start, stop, restart and monitor servers from the Web UI, CLI or TUI.
- Follow live server output and send console commands through WebSockets.
- Edit server files, upload/download files and manage icons.
- Install and synchronize addons from supported providers.
- Create, restore, upload and download backups, including automatic backups.
- Track CPU, memory and server status through the dashboard.
- Manage users, passwords and per-server permissions.
- Share selected servers through public links.

The application has two parts: a background daemon that owns the servers and data, and clients that connect to it. The
Web UI is served by the daemon, so a normal installation does not require a separate web server.

## Choose how to run it

| Use case                     | Recommended option                                |
|------------------------------|---------------------------------------------------|
| Local desktop use            | Native installer with the system tray application |
| Headless Linux or macOS host | Native headless service                           |
| Docker host, NAS or homelab  | Docker image or Compose                           |
| Automation and scripts       | `naviserver-cli`                                  |
| Terminal-only administration | `naviserver-cli tui`                              |

Start with the [installation guide](wiki/installation.md). It covers native installers, headless services, Docker,
volumes, ports, upgrades and uninstallation. The [wiki home](wiki/home.md) links to configuration, CLI, TUI and
migration documentation.

## Quick start with Docker

Docker is the quickest way to run NaviServer on a server. The image supports `linux/amd64` and `linux/arm64`.
The examples use the image published in GitHub Container Registry:
`ghcr.io/andre-carbajal/naviserver`. The equivalent image is also available from Docker Hub as
`anvian/naviserver`.

```bash
docker run -d --name naviserver --restart unless-stopped \
  -p 23008:23008 \
  -v naviserver-data:/data \
  ghcr.io/andre-carbajal/naviserver:latest
```

Open `http://localhost:23008` after the container starts. The `/data` volume contains the database, configuration,
secrets, server instances, backups and downloaded runtimes; keep it when replacing or upgrading the container.

For a Compose deployment, use the repository's [`compose.yml`](compose.yml):

```bash
docker compose up -d
```

The Web UI, API and WebSocket use TCP port `23008`. Minecraft server ports are separate and must be published explicitly
in Compose or with `docker run`. Keep the Minecraft port range away from `23008`; the default range is `25565-25600`.
See the [Docker installation documentation](wiki/installation.md#docker) for secrets, backups, port mappings and
version-pinned deployments.

## Native quick start

Download the appropriate asset from [GitHub Releases](https://github.com/andre-carbajal/NaviServer/releases), or use the
installation script on Linux/macOS:

```bash
curl -sSL https://raw.githubusercontent.com/andre-carbajal/NaviServer/main/install.sh | sh
```

For detailed native installation, headless service setup, data locations and migration from version 1.x, see the
[installation guide](wiki/installation.md).

## CLI administration

NaviServer can also be administered through `naviserver-cli`, which is useful for scripts, automation, remote
administration and terminal-only environments. The CLI covers server lifecycle operations, settings, backups, users,
permissions and shell completion. It also includes an interactive TUI for managing the application directly from a
terminal.

The [CLI documentation](wiki/cli.md) contains the complete command reference, authentication details, port
configuration, shell completion and TUI information.

## Supported platforms and loaders

- Native desktop/headless builds: Windows, macOS and Linux.
- Docker images: `linux/amd64` and `linux/arm64`.
- Minecraft loaders: Vanilla, Paper, Fabric, Forge and NeoForge.

## Documentation

- [Installation and Docker](wiki/installation.md)
- [Configuration and environment variables](wiki/configuration.md)
- [CLI reference](wiki/cli.md)
- [TUI guide](wiki/tui.md)
- [Migration from 1.x](wiki/migration-from-1x.md)
- [Project wiki home](wiki/home.md)

## Contributing and support

Bug reports and feature requests are welcome
through [GitHub Issues](https://github.com/andre-carbajal/NaviServer/issues).
Contributions can be submitted through pull requests. Before reporting an issue, include the NaviServer version, the
installation method, relevant logs and the operating system or container architecture.

## License

NaviServer is released under the [MIT License](LICENSE).
