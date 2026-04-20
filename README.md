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

```bash
# Open interactive TUI
go run ./cmd/cli tui

# Servers
go run ./cmd/cli server list
go run ./cmd/cli server create --name <server-name> --loader <loader> --version <version> --ram <mb>
go run ./cmd/cli server create --name <server-name> --async
go run ./cmd/cli server start <server-id>
go run ./cmd/cli server stop <server-id>
go run ./cmd/cli server delete <server-id>
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
