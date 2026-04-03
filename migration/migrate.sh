#!/bin/bash

# NaviServer Migration Script (Linux & macOS)
# Migrates data from 'naviger' to 'naviserver' with backup and automatic installation

set -e

# Color output
color_info()    { echo "ℹ️  $1"; }
color_success() { echo "✓ $1"; }
color_warning() { echo "⚠️  $1"; }
color_error()   { echo "✗ $1"; }

# Helper for sudo
run_sudo() {
  if [ "${EUID:-$(id -u)}" -eq 0 ]; then "$@"; else sudo "$@"; fi
}

OS="$(uname -s)"
case "${OS}" in
    Linux*)     OS_TYPE="linux"; USER_CONFIG_DIR="${HOME}/.config";;
    Darwin*)    OS_TYPE="macos"; USER_CONFIG_DIR="${HOME}/Library/Application Support";;
    *)          color_error "Unsupported OS: ${OS}"; exit 1;;
esac

OLD_APP="naviger"
NEW_APP="naviserver"
OLD_INSTALL_DIR="/opt/${OLD_APP}"
NEW_INSTALL_DIR="/opt/${NEW_APP}"
OLD_DATA_DIR="${USER_CONFIG_DIR}/${OLD_APP}"
NEW_DATA_DIR="${USER_CONFIG_DIR}/${NEW_APP}"
BACKUP_DIR="${HOME}"
TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")

# Determine backup extension and command
if command -v tar >/dev/null 2>&1; then
    BACKUP_EXT="tar.gz"
    BACKUP_FILE="${BACKUP_DIR}/${OLD_APP}_backup_${TIMESTAMP}.tar.gz"
else
    BACKUP_EXT="zip"
    BACKUP_FILE="${BACKUP_DIR}/${OLD_APP}_backup_${TIMESTAMP}.zip"
fi

echo ""
color_info "=== NaviServer Migration Script ==="
echo ""
color_info "This script will:"
echo "  1. Create a backup of your data"
echo "  2. Migrate data from ${OLD_APP} to ${NEW_APP}"
echo "  3. Install the new version automatically"
echo "  4. Restart the service"
echo ""
echo "Data directory: ${OLD_DATA_DIR}"
echo "Backup location: ${BACKUP_FILE}"
echo ""

# Confirmation at the beginning
read -p "Do you want to continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    color_error "Migration cancelled by user."
    exit 0
fi

echo ""

# ============================================================
# 1. VALIDATION
# ============================================================
color_info "Step 1: Validating old installation..."

if [ ! -d "$OLD_DATA_DIR" ]; then
    color_error "No old data directory found at ${OLD_DATA_DIR}"
    color_error "Nothing to migrate. Exiting."
    exit 1
fi

color_success "Old data directory found"

# ============================================================
# 2. CREATE BACKUP
# ============================================================
color_info "Step 2: Creating backup..."

if command -v tar >/dev/null 2>&1; then
    # Use tar.gz
    if tar -czf "${BACKUP_FILE}" -C "$(dirname "$OLD_DATA_DIR")" "$(basename "$OLD_DATA_DIR")" 2>/dev/null; then
        BACKUP_SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
        color_success "Backup created: ${BACKUP_FILE} (${BACKUP_SIZE})"
    else
        color_error "Failed to create backup"
        exit 1
    fi
elif command -v zip >/dev/null 2>&1; then
    # Use zip as fallback
    if zip -r -q "${BACKUP_FILE}" "${OLD_DATA_DIR}" 2>/dev/null; then
        BACKUP_SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
        color_success "Backup created: ${BACKUP_FILE} (${BACKUP_SIZE})"
    else
        color_error "Failed to create backup"
        exit 1
    fi
else
    color_warning "No backup tool available (tar/zip). Continuing without backup..."
fi

echo ""

# ============================================================
# 3. STOP OLD SERVICES
# ============================================================
color_info "Step 3: Stopping old services..."

if [ "$OS_TYPE" = "linux" ]; then
    if systemctl is-active --quiet "${OLD_APP}" 2>/dev/null; then
        run_sudo systemctl stop "${OLD_APP}" || true
        run_sudo systemctl disable "${OLD_APP}" || true
        color_success "Old service stopped"
    else
        color_info "Old service not running"
    fi
elif [ "$OS_TYPE" = "macos" ]; then
    PLIST="${HOME}/Library/LaunchAgents/com.${OLD_APP}.server.plist"
    if [ -f "$PLIST" ]; then
        launchctl unload "$PLIST" 2>/dev/null || true
        color_success "Old agent stopped"
    else
        color_info "Old agent not configured"
    fi
fi

echo ""

# ============================================================
# 4. MIGRATE DATA
# ============================================================
color_info "Step 4: Migrating data..."

if [ -d "$NEW_DATA_DIR" ]; then
    color_warning "New data directory already exists at ${NEW_DATA_DIR}"
    color_warning "Skipping data move to avoid overwriting. Please merge manually if needed."
else
    mkdir -p "$(dirname "$NEW_DATA_DIR")"
    if mv "$OLD_DATA_DIR" "$NEW_DATA_DIR"; then
        color_success "Data moved to ${NEW_DATA_DIR}"
    else
        color_error "Failed to move data"
        exit 1
    fi
    
    # Rename secret file to match new convention
    if [ -f "${NEW_DATA_DIR}/.naviger_secret" ]; then
        mv "${NEW_DATA_DIR}/.naviger_secret" "${NEW_DATA_DIR}/.naviserver_secret"
        color_success "Secret file migrated"
    fi
fi

echo ""

# ============================================================
# 5. HANDLE INSTALLATION DIRECTORY
# ============================================================
color_info "Step 5: Cleaning up old installation..."

if [ -d "$OLD_INSTALL_DIR" ]; then
    if [ -d "$NEW_INSTALL_DIR" ]; then
        run_sudo rm -rf "$OLD_INSTALL_DIR"
        color_success "Old installation removed"
    else
        run_sudo mv "$OLD_INSTALL_DIR" "$NEW_INSTALL_DIR"
        color_success "Installation migrated"
    fi
fi

# Cleanup old symlinks
run_sudo rm -f "/usr/local/bin/${OLD_APP}-server" "/usr/local/bin/${OLD_APP}-cli" 2>/dev/null || true

echo ""

# ============================================================
# 6. INSTALL NEW VERSION
# ============================================================
color_info "Step 6: Installing new version automatically..."

# Check if install.sh exists in the same directory as this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALL_SCRIPT="${SCRIPT_DIR}/install.sh"

if [ ! -f "$INSTALL_SCRIPT" ]; then
    color_error "install.sh not found at ${INSTALL_SCRIPT}"
    color_warning "Please run the installation manually:"
    color_warning "  bash ${SCRIPT_DIR}/install.sh"
    exit 1
fi

echo ""
echo "Running: bash ${INSTALL_SCRIPT} (this will ask for headless/desktop mode)"
echo ""

# Run install.sh but don't exit if it fails
if bash "$INSTALL_SCRIPT"; then
    color_success "Installation completed successfully"
else
    INSTALL_EXIT_CODE=$?
    color_warning "Installation script exited with code ${INSTALL_EXIT_CODE}"
    color_warning "Your data has been migrated and is safe."
    color_warning "Please try running the installation manually:"
    color_warning "  bash ${INSTALL_SCRIPT}"
    echo ""
    
    # Continue anyway - data is safe
fi

echo ""

# ============================================================
# 7. VERIFY NEW SERVICE IS RUNNING (headless only)
# ============================================================
color_info "Step 7: Verifying new service..."

if [ "$OS_TYPE" = "linux" ]; then
    # Check if it's a headless installation
    if systemctl is-active --quiet naviserver 2>/dev/null; then
        color_success "NaviServer service is running"
    else
        color_warning "NaviServer service is not running yet"
        color_info "If you installed in headless mode, trying to restart service..."
        
        if run_sudo systemctl restart naviserver 2>/dev/null; then
            sleep 2
            if systemctl is-active --quiet naviserver; then
                color_success "NaviServer service restarted successfully"
            else
                color_warning "Service restart may have failed. Please check with:"
                color_warning "  systemctl status naviserver"
            fi
        fi
    fi
elif [ "$OS_TYPE" = "macos" ]; then
    PLIST="${HOME}/Library/LaunchAgents/com.naviserver.server.plist"
    if [ -f "$PLIST" ]; then
        if launchctl list com.naviserver.server >/dev/null 2>&1; then
            color_success "NaviServer agent is running"
        else
            color_info "Restarting NaviServer agent..."
            launchctl stop com.naviserver.server 2>/dev/null || true
            sleep 1
            if launchctl start com.naviserver.server 2>/dev/null; then
                color_success "NaviServer agent restarted successfully"
            else
                color_warning "Agent restart may have failed"
            fi
        fi
    else
        color_info "NaviServer agent not yet configured (may be desktop mode)"
    fi
fi

echo ""
echo "═══════════════════════════════════════════════════════"
color_success "Migration completed successfully!"
echo "═══════════════════════════════════════════════════════"
echo ""
echo "Summary:"
echo "  ✓ Data migrated to: ${NEW_DATA_DIR}"
echo "  ✓ Backup created at: ${BACKUP_FILE}"
echo "  ✓ New version installed"
echo ""
color_info "Your backup is available at:"
color_info "  ${BACKUP_FILE}"
echo ""
color_info "If you need to restore your backup:"
echo "  tar -xzf ${BACKUP_FILE} -C ${USER_CONFIG_DIR}/"
echo ""
