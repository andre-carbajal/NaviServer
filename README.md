# ⛏️ NaviServer: A Modern Minecraft Server Manager

![GitHub release](https://img.shields.io/github/v/release/andre-carbajal/NaviServer?style=flat-square)

NaviServer is a modern, lightweight, and cross-platform Minecraft server manager designed to make managing multiple server
instances easy through an intuitive web interface, a powerful CLI, and clean operating system integration.

---

## 📦 Supported Loaders

- Vanilla
- Paper
- Fabric
- Forge
- NeoForge

---

## ✨ Features

- Dual Interface:
    - Web UI: Modern control panel developed in React and Vite.
    - CLI + TUI: Script-friendly CLI plus interactive terminal interface based on Bubble Tea.
- Backup Management: Complete system for creating, listing, and restoring backups.
- Real-Time Console: Communication via WebSockets for monitoring and live commands.
- Daemon with System Integration: Background application with icon in the system tray (systray) for quick access and
  status control.
- Automatic Runtime Management (JVM): Downloads and organizes the necessary Java versions for each server automatically.
- Performance Statistics: Monitoring of CPU and RAM usage per server.
- Cross-Platform: Runs on Windows, macOS, and Linux.

## 🧩 CLI Quick Commands

Assuming `naviserver-cli` is installed and available in your `PATH`.
For local development, you can run the same commands with `go run ./cmd/cli`.

```bash
# Open interactive TUI
naviserver-cli tui

# Servers
naviserver-cli server list
naviserver-cli server create --name <server-name> --loader <loader> --version <version> --ram <mb>
naviserver-cli server create --name <server-name> --async
naviserver-cli server start <server-id>
naviserver-cli server stop <server-id>
naviserver-cli server delete <server-id>

# Completion
naviserver-cli completion zsh > ~/.zfunc/_naviserver-cli

# Settings
naviserver-cli settings port-range get
naviserver-cli settings port-range set --start 23008 --end 23108
naviserver-cli settings public-ip get
naviserver-cli settings public-ip set --value localhost
naviserver-cli settings interfaces list
naviserver-cli settings log-buffer get
naviserver-cli settings log-buffer set --lines 1200

# Users
naviserver-cli user list
naviserver-cli user create --username dev --password 'change-me'
naviserver-cli user password set <user-id> --password 'new-password'
naviserver-cli user permissions get <user-id>
naviserver-cli user permissions set <user-id> --server <server-id> --power true --console false

```

`server create` is synchronous by default (waits for progress completion over WebSocket).
Use `--async` for fire-and-return behavior.

## 🐞 Bugs & Feedback

If you encounter any issues or have suggestions for improvements, please visit
our [GitHub Issues](https://github.com/andre-carbajal/NaviServer/issues) page to report bugs or provide feedback.
We appreciate your input to help us enhance NaviServer!

## 📋 Contributing

Contributions are welcome! If you'd like to contribute to NaviServer, please fork the repository and create a pull request
with your changes.

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
