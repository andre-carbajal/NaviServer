#!/usr/bin/env bash

# NaviServer Uninstall Script for Linux and macOS
# Reverses install.sh: stops and disables service/agent, removes files and symlinks
# Optionally preserves data with --keep-data flag

set -euo pipefail

INSTALL_DIR="/opt/naviserver"
BIN_DIR="/usr/local/bin"
SERVICE_FILE="/etc/systemd/system/naviserver.service"
PLIST_NAME="com.naviserver.server.plist"

# Color output
color_info()    { echo "ℹ️  $1"; }
color_success() { echo "✓ $1"; }
color_warning() { echo "⚠️  $1"; }
color_error()   { echo "✗ $1"; }

usage() {
  cat <<EOF
Usage: $0 [options]

Options:
  --keep-data, -k  Preserve data directory (~/.config/naviserver/)
  --yes, -y        Don't prompt, run non-interactively
  --help, -h       Show this help message

This script will request sudo only when required to remove system files.

Without --keep-data, a backup will be created before uninstalling.
EOF
}

# Parse args
FORCE=no
KEEP_DATA=no

for arg in "$@"; do
  case "$arg" in
    -y|--yes) FORCE=yes; shift || true;;
    -k|--keep-data) KEEP_DATA=yes; shift || true;;
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
  *) color_error "Unsupported OS: $OS"; exit 1 ;;
esac

# Determine config directory
if [ "$OS_TYPE" = "linux" ]; then
    USER_CONFIG_DIR="${HOME}/.config"
else
    USER_CONFIG_DIR="${HOME}/Library/Application Support"
fi

DATA_DIR="${USER_CONFIG_DIR}/naviserver"

echo ""
color_info "=== NaviServer Uninstall Script ==="
echo ""
echo "Installation directory: ${INSTALL_DIR}"
echo "Data directory: ${DATA_DIR}"
echo ""

# ============================================================
# CONFIRMATION AND BACKUP
# ============================================================

if [ "$KEEP_DATA" = "yes" ]; then
    color_info "Using --keep-data flag"
    color_success "Your data will be preserved at: ${DATA_DIR}"
    echo ""
else
    echo "This will remove NaviServer installation at: ${INSTALL_DIR}"
    echo ""
    
    if [ "$FORCE" != "yes" ]; then
        color_warning "Your data will be DELETED unless backed up!"
        echo ""
        read -r -p "Do you want to create a backup before uninstalling? (y/n) " backup_confirm
        case "$backup_confirm" in
            [yY]|[yY][eE][sS])
                echo "Creating backup before uninstall..."
                ;;
            *)
                backup_confirm="no"
                ;;
        esac
    else
        backup_confirm="no"
    fi
    
    # Create backup if confirmed
    if [ "$backup_confirm" = "yes" ] || [ "$backup_confirm" != "no" ]; then
        TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")
        BACKUP_FILE="${HOME}/naviserver_uninstall_backup_${TIMESTAMP}.tar.gz"
        
        if command -v tar >/dev/null 2>&1; then
            if tar -czf "${BACKUP_FILE}" -C "$(dirname "$DATA_DIR")" "naviserver" 2>/dev/null; then
                BACKUP_SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
                color_success "Backup created: ${BACKUP_FILE} (${BACKUP_SIZE})"
            else
                color_warning "Could not create backup"
            fi
        else
            color_warning "tar not available - skipping backup"
        fi
        echo ""
    fi
fi

# Final confirmation
if [ "$FORCE" != "yes" ]; then
    if [ "$KEEP_DATA" = "yes" ]; then
        read -r -p "Continue uninstallation (keeping data)? [y/N]: " confirm
    else
        read -r -p "Continue uninstallation? [y/N]: " confirm
    fi
    case "$confirm" in
        [yY]|[yY][eE][sS]) ;;
        *) color_error "Aborted."; exit 0 ;;
    esac
fi

echo ""

# ============================================================
# UNINSTALL PROCESS
# ============================================================

color_info "Stopping and removing service/agent (if present)..."
echo ""

if [ "$OS_TYPE" = "linux" ]; then
  if command -v systemctl >/dev/null 2>&1; then
    if systemctl is-active --quiet naviserver; then
      echo "Stopping naviserver service..."
      run_sudo systemctl stop naviserver || true
    fi

    if systemctl is-enabled --quiet naviserver 2>/dev/null; then
      echo "Disabling naviserver service..."
      run_sudo systemctl disable naviserver || true
    fi

    if [ -f "$SERVICE_FILE" ]; then
      echo "Removing systemd service file ${SERVICE_FILE}..."
      run_sudo rm -f "$SERVICE_FILE"
      echo "Reloading systemd daemon..."
      run_sudo systemctl daemon-reload || true
    fi

    # Also check for user service just in case
    if systemctl --user is-active --quiet naviserver 2>/dev/null; then
        echo "Stopping user service..."
        systemctl --user stop naviserver || true
        systemctl --user disable naviserver || true
    fi
  else
    color_warning "systemctl not found; skipping systemd cleanup."
  fi

  # Remove desktop entry if it exists (user-owned, no sudo needed)
  DESKTOP_FILE="$HOME/.local/share/applications/naviserver.desktop"
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
  if [ -f "/etc/paths.d/naviserver" ]; then
      echo "Removing /etc/paths.d/naviserver (requires sudo)..."
      run_sudo rm -f "/etc/paths.d/naviserver" || true
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
  color_warning "No installation directory found at ${INSTALL_DIR}"
fi

# Remove data directory if not keeping it
if [ "$KEEP_DATA" != "yes" ]; then
    if [ -d "$DATA_DIR" ]; then
        echo "Removing data directory ${DATA_DIR}..."
        rm -rf "$DATA_DIR" || true
        echo "Removed ${DATA_DIR}"
    fi
fi

# Additional cleanup: logs in /tmp (user-owned, no sudo needed)
rm -f /tmp/naviserver.out /tmp/naviserver.err 2>/dev/null || true

echo ""
echo "═══════════════════════════════════════════════════════"
color_success "Uninstall complete."
echo "═══════════════════════════════════════════════════════"
echo ""

if [ "$KEEP_DATA" = "yes" ]; then
    color_success "Your data has been preserved at: ${DATA_DIR}"
else
    color_info "If you need to restore from backup, extract it with:"
    color_info "  tar -xzf ~/naviserver_uninstall_backup_*.tar.gz -C ${USER_CONFIG_DIR}/"
fi
echo ""

exit 0
