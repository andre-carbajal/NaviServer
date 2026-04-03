#!/bin/bash

# NaviServer Migration Script (Linux & macOS)
# Migrates data from 'naviger' to 'naviserver'

set -e

# Helper for sudo
run_sudo() {
  if [ "${EUID:-$(id -u)}" -eq 0 ]; then "$@"; else sudo "$@"; fi
}

OS="$(uname -s)"
case "${OS}" in
    Linux*)     OS_TYPE="linux"; USER_CONFIG_DIR="${HOME}/.config";;
    Darwin*)    OS_TYPE="macos"; USER_CONFIG_DIR="${HOME}/Library/Application Support";;
    *)          echo "Unsupported OS: ${OS}"; exit 1;;
esac

OLD_APP="naviger"
NEW_APP="naviserver"
OLD_INSTALL_DIR="/opt/${OLD_APP}"
NEW_INSTALL_DIR="/opt/${NEW_APP}"
OLD_DATA_DIR="${USER_CONFIG_DIR}/${OLD_APP}"
NEW_DATA_DIR="${USER_CONFIG_DIR}/${NEW_APP}"

echo "--- Starting Migration: ${OLD_APP} -> ${NEW_APP} ---"

# 1. Stop old services
echo "Stopping old services..."
if [ "$OS_TYPE" = "linux" ]; then
    if systemctl is-active --quiet "${OLD_APP}" 2>/dev/null; then
        run_sudo systemctl stop "${OLD_APP}" || true
        run_sudo systemctl disable "${OLD_APP}" || true
    fi
elif [ "$OS_TYPE" = "macos" ]; then
    PLIST="${HOME}/Library/LaunchAgents/com.${OLD_APP}.server.plist"
    if [ -f "$PLIST" ]; then
        launchctl unload "$PLIST" 2>/dev/null || true
    fi
fi

# 2. Migrate Data Directory (Database, Servers, Backups)
if [ -d "$OLD_DATA_DIR" ]; then
    if [ -d "$NEW_DATA_DIR" ]; then
        echo "Warning: New data directory already exists at ${NEW_DATA_DIR}."
        echo "Skipping data move to avoid overwriting. Please merge manually if needed."
    else
        echo "Moving data from ${OLD_DATA_DIR} to ${NEW_DATA_DIR}..."
        mkdir -p "$(dirname "$NEW_DATA_DIR")"
        mv "$OLD_DATA_DIR" "$NEW_DATA_DIR"
        
        # Rename secret file to match new convention
        if [ -f "${NEW_DATA_DIR}/.naviger_secret" ]; then
            mv "${NEW_DATA_DIR}/.naviger_secret" "${NEW_DATA_DIR}/.naviserver_secret"
            echo "Secret file migrated successfully."
        fi
        
        echo "Data migrated successfully."
    fi
else
    echo "No old data directory found at ${OLD_DATA_DIR}. Skipping data migration."
fi

# 3. Handle Installation Directory
if [ -d "$OLD_INSTALL_DIR" ]; then
    if [ -d "$NEW_INSTALL_DIR" ]; then
        echo "New installation directory already exists. Removing old one..."
        run_sudo rm -rf "$OLD_INSTALL_DIR"
    else
        echo "Moving installation from ${OLD_INSTALL_DIR} to ${NEW_INSTALL_DIR}..."
        run_sudo mv "$OLD_INSTALL_DIR" "$NEW_INSTALL_DIR"
    fi
fi

# 4. Cleanup old symlinks
echo "Cleaning up old symlinks..."
run_sudo rm -f "/usr/local/bin/${OLD_APP}-server" "/usr/local/bin/${OLD_APP}-cli" 2>/dev/null || true

echo "--- Migration Finished ---"
echo "You can now install NaviServer and it will use your existing data."
echo "Note: If you were using a systemd service or launchd agent, the new installer will set up the new ones."
