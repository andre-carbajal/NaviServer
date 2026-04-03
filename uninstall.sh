#!/usr/bin/env bash

# NaviServer Uninstall Script for Linux and macOS
# Reverses install.sh: stops and disables service/agent, removes files and symlinks

set -euo pipefail

INSTALL_DIR="/opt/naviserver"
BIN_DIR="/usr/local/bin"
SERVICE_FILE="/etc/systemd/system/naviger.service"
PLIST_NAME="com.naviserver.server.plist"

usage() {
  cat <<EOF
Usage: $0 [--yes|-y] [--help|-h]

Options:
  -y, --yes    Don't prompt, run non-interactively
  -h, --help   Show this help message

This script will request sudo only when required to remove system files.
EOF
}

# Parse args
FORCE=no
for arg in "$@"; do
  case "$arg" in
    -y|--yes) FORCE=yes; shift || true;;
    -h|--help) usage; exit 0;;
    *) ;;
  esac
done

run_sudo() {
  if [ "${EUID:-$(id -u)}" -eq 0 ]; then
    "$@"
  else
    sudo "$@"
  fi
}

REAL_USER="${SUDO_USER:-$USER}"
if [ -z "$REAL_USER" ]; then
  REAL_USER="$USER"
fi

OS="$(uname -s)"
case "$OS" in
  Linux*) OS_TYPE=linux ;;
  Darwin*) OS_TYPE=macos ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

if [ "$FORCE" != "yes" ]; then
  echo "This will remove NaviServer installation at: ${INSTALL_DIR}"
  echo "It will also remove symlinks in ${BIN_DIR} and any service/agent configuration."
  read -r -p "Are you sure you want to continue? [y/N]: " confirm
  case "$confirm" in
    [yY]|[yY][eE][sS]) ;;
    *) echo "Aborted."; exit 0 ;;
  esac
fi

echo "Stopping and removing service/agent (if present)..."

if [ "$OS_TYPE" = "linux" ]; then
  if command -v systemctl >/dev/null 2>&1; then
    if systemctl is-active --quiet naviger; then
      echo "Stopping naviger service..."
      run_sudo systemctl stop naviger || true
    fi

    if systemctl is-enabled --quiet naviger 2>/dev/null; then
      echo "Disabling naviger service..."
      run_sudo systemctl disable naviger || true
    fi

    if [ -f "$SERVICE_FILE" ]; then
      echo "Removing systemd service file ${SERVICE_FILE}..."
      run_sudo rm -f "$SERVICE_FILE"
      echo "Reloading systemd daemon..."
      run_sudo systemctl daemon-reload || true
    fi

    # Also check for user service just in case
    if systemctl --user is-active --quiet naviger 2>/dev/null; then
        echo "Stopping user service..."
        systemctl --user stop naviger || true
        systemctl --user disable naviger || true
    fi
  else
    echo "systemctl not found; skipping systemd cleanup."
  fi

  # Remove desktop entry if it exists (user-owned, no sudo needed)
  DESKTOP_FILE="$HOME/.local/share/applications/naviger.desktop"
  if [ -f "$DESKTOP_FILE" ]; then
      echo "Removing desktop entry..."
      rm -f "$DESKTOP_FILE"
  fi

elif [ "$OS_TYPE" = "macos" ]; then
  USER_HOME="$HOME"
  PLIST_FILE="$USER_HOME/Library/LaunchAgents/${PLIST_NAME}"

  if [ -f "$PLIST_FILE" ]; then
    echo "Unloading launchd agent $PLIST_FILE..."
    launchctl unload "$PLIST_FILE" 2>/dev/null || true

    echo "Removing plist file..."
    rm -f "$PLIST_FILE"
  else
    echo "No launchd agent plist found at $PLIST_FILE"
  fi

  # Remove App Bundle from system Applications (needs sudo)
  APP_PATH="/Applications/NaviServer.app"
  if [ -d "$APP_PATH" ]; then
      echo "Removing NaviServer.app from /Applications (requires sudo)..."
      run_sudo rm -rf "$APP_PATH"
  fi

  # Remove App Bundle from user Applications (no sudo needed)
  USER_APP_PATH="$USER_HOME/Applications/NaviServer.app"
  if [ -d "$USER_APP_PATH" ]; then
      echo "Removing NaviServer.app from user Applications..."
      rm -rf "$USER_APP_PATH"
  fi

  # Remove naviserver-cli from /usr/local/bin (needs sudo)
  if [ -f "${BIN_DIR}/naviserver-cli" ] || [ -L "${BIN_DIR}/naviserver-cli" ]; then
      echo "Removing ${BIN_DIR}/naviserver-cli (requires sudo)..."
      run_sudo rm -f "${BIN_DIR}/naviserver-cli" || true
  fi

  # Remove PATH entry added by PKG (needs sudo)
  if [ -f "/etc/paths.d/naviger" ]; then
      echo "Removing /etc/paths.d/naviger (requires sudo)..."
      run_sudo rm -f "/etc/paths.d/naviger" || true
  fi

  # Forget PKG receipt (needs sudo)
  PKG_ID="com.naviserver.server"
  if pkgutil --pkg-info "$PKG_ID" >/dev/null 2>&1; then
      echo "Removing PKG receipt for ${PKG_ID} (requires sudo)..."
      run_sudo pkgutil --forget "$PKG_ID" || true
  fi
fi

# Remove symlinks in BIN_DIR (Linux only; needs sudo)
if [ "$OS_TYPE" = "linux" ]; then
  echo "Removing symlinks in ${BIN_DIR} (requires sudo)..."
  if [ -L "${BIN_DIR}/naviserver-cli" ] || [ -e "${BIN_DIR}/naviserver-cli" ]; then
    run_sudo rm -f "${BIN_DIR}/naviserver-cli" || true
    echo "Removed ${BIN_DIR}/naviserver-cli"
  fi
  if [ -L "${BIN_DIR}/naviserver-server" ] || [ -e "${BIN_DIR}/naviserver-server" ]; then
    run_sudo rm -f "${BIN_DIR}/naviserver-server" || true
    echo "Removed ${BIN_DIR}/naviserver-server"
  fi
fi

# Remove installation directory (needs sudo)
if [ -d "$INSTALL_DIR" ]; then
  echo "Removing installation directory ${INSTALL_DIR} (requires sudo)..."
  run_sudo rm -rf "$INSTALL_DIR" || true
  echo "Removed ${INSTALL_DIR}"
else
  echo "No installation directory found at ${INSTALL_DIR}"
fi

# Additional cleanup: logs in /tmp (user-owned, no sudo needed)
rm -f /tmp/naviger.out /tmp/naviger.err 2>/dev/null || true

echo "Uninstall complete."

exit 0
