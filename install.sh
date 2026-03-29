#!/bin/bash

# Naviger Installation Script for Linux and macOS

set -e

REPO_OWNER="andre-carbajal"
REPO_NAME="Naviger"

# Determine OS and Architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

case "${OS}" in
    Linux*)     OS_TYPE="linux";;
    Darwin*)    OS_TYPE="macos";;
    *)          echo "Unsupported operating system: ${OS}"; exit 1;;
esac

if [ "$OS_TYPE" = "macos" ]; then
    ASSET_SUFFIX="macos"
else
    # Linux
    if [ "$ARCH" = "x86_64" ]; then
        ASSET_SUFFIX="linux"
    else
        echo "Warning: The official Linux build is optimized for x86_64. Your architecture is ${ARCH}."
        echo "Attempting to install anyway (it might not work)..."
        ASSET_SUFFIX="linux"
    fi
fi

echo "Detected System: ${OS} (${ARCH}) -> Asset: ${ASSET_SUFFIX}"

# Ask for installation mode
echo "Select installation mode:"
echo "1) Headless (Service/Daemon)"
echo "2) Desktop (App/Shortcut)"
read -p "Enter choice [1-2]: " INSTALL_MODE

if [ "$INSTALL_MODE" != "1" ] && [ "$INSTALL_MODE" != "2" ]; then
    echo "Invalid choice. Exiting."
    exit 1
fi

# Check for dependencies
command -v curl >/dev/null 2>&1 || { echo >&2 "curl is required but not installed. Aborting."; exit 1; }
command -v unzip >/dev/null 2>&1 || { echo >&2 "unzip is required but not installed. Aborting."; exit 1; }

# Helper: run command with sudo only if not already root
run_sudo() {
  if [ "${EUID:-$(id -u)}" -eq 0 ]; then
    "$@"
  else
    sudo "$@"
  fi
}

# Determine actual user
REAL_USER="${SUDO_USER:-$USER}"

# Define paths
INSTALL_DIR="/opt/naviger"
BIN_DIR="/usr/local/bin"

# Stop existing service if running
echo "Checking for existing installation..."
if [ "$OS_TYPE" = "linux" ]; then
    if systemctl is-active --quiet naviger; then
        echo "Stopping existing Naviger service..."
        run_sudo systemctl stop naviger
    fi
elif [ "$OS_TYPE" = "macos" ]; then
    USER_HOME="$HOME"
    PLIST_FILE="$USER_HOME/Library/LaunchAgents/com.naviger.server.plist"

    if [ -f "$PLIST_FILE" ]; then
        echo "Stopping existing Naviger agent..."
        launchctl unload "$PLIST_FILE" 2>/dev/null || true
    fi
fi

# Clean up previous installation
if [ -d "$INSTALL_DIR" ]; then
    echo "Removing previous installation at ${INSTALL_DIR}..."
    run_sudo rm -rf "${INSTALL_DIR}"
fi

# Fetch latest version
echo "Fetching latest release info..."
LATEST_URL=$(curl -Ls -o /dev/null -w %{url_effective} "https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/latest")
VERSION=$(basename "$LATEST_URL")
CLEAN_VERSION="${VERSION#v}"

if [ -z "$VERSION" ] || [ "$VERSION" = "latest" ]; then
    echo "Error: Could not determine latest version."
    exit 1
fi

echo "Latest version: ${VERSION}"

ASSET_NAME="Naviger-${CLEAN_VERSION}-${ASSET_SUFFIX}.zip"
DOWNLOAD_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}/${ASSET_NAME}"

# Download and Extract
TMP_DIR=$(mktemp -d)
echo "Downloading ${ASSET_NAME} from ${DOWNLOAD_URL}..."

if curl -L -o "${TMP_DIR}/${ASSET_NAME}" "${DOWNLOAD_URL}" --fail; then
    echo "Download successful."
else
    echo "Error: Failed to download release. Please check if the asset exists for your platform."
    rm -rf "${TMP_DIR}"
    exit 1
fi

echo "Extracting..."
unzip -q "${TMP_DIR}/${ASSET_NAME}" -d "${TMP_DIR}/extracted"

# Installation Logic
if [ "$INSTALL_MODE" = "1" ]; then
    # HEADLESS MODE
    echo "Installing Headless Mode..."

    run_sudo mkdir -p "${INSTALL_DIR}"
    run_sudo rm -rf "${INSTALL_DIR}/*"

    # Check if extracted content is inside a folder (common with zips) or flat
    if [ -d "${TMP_DIR}/extracted/Naviger.app" ]; then
         # macOS app bundle case - we need the binary inside
         run_sudo cp -r "${TMP_DIR}/extracted/Naviger.app/Contents/MacOS/Naviger" "${INSTALL_DIR}/naviger-server"
         # Also copy web_dist if it exists inside Resources or MacOS
         if [ -d "${TMP_DIR}/extracted/Naviger.app/Contents/MacOS/web_dist" ]; then
             run_sudo cp -r "${TMP_DIR}/extracted/Naviger.app/Contents/MacOS/web_dist" "${INSTALL_DIR}/"
         fi
    else
         run_sudo cp -r "${TMP_DIR}/extracted/"* "${INSTALL_DIR}/"
    fi

    # Ensure CLI is installed if it was outside the app bundle
    if [ -f "${TMP_DIR}/extracted/naviger-cli" ]; then
        run_sudo cp "${TMP_DIR}/extracted/naviger-cli" "${INSTALL_DIR}/"
    fi

    # Cleanup
    rm -rf "${TMP_DIR}"

    # Set permissions
    run_sudo chmod +x "${INSTALL_DIR}/naviger-server"
    if [ -f "${INSTALL_DIR}/naviger-cli" ]; then
        run_sudo chmod +x "${INSTALL_DIR}/naviger-cli"
        run_sudo ln -sf "${INSTALL_DIR}/naviger-cli" "${BIN_DIR}/naviger-cli"
    fi
    run_sudo chown -R "$REAL_USER" "$INSTALL_DIR"

    # Service Configuration
    if [ "$OS_TYPE" = "linux" ]; then
        SERVICE_FILE="/etc/systemd/system/naviger.service"
        echo "Setting up systemd service..."

        run_sudo bash -c "cat > ${SERVICE_FILE}" <<EOF
[Unit]
Description=Naviger Server Daemon
After=network.target

[Service]
Type=simple
User=$REAL_USER
ExecStart=${INSTALL_DIR}/naviger-server --headless
WorkingDirectory=${INSTALL_DIR}
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

        echo "Reloading systemd daemon..."
        run_sudo systemctl daemon-reload
        echo "Enabling and starting naviger service..."
        run_sudo systemctl enable naviger
        run_sudo systemctl start naviger
        echo "Naviger service installed and started (Headless)."

    elif [ "$OS_TYPE" = "macos" ]; then
        USER_HOME="$HOME"
        PLIST_FILE="$USER_HOME/Library/LaunchAgents/com.naviger.server.plist"

        echo "Setting up launchd agent at $PLIST_FILE..."

        # Ensure the LaunchAgents directory exists (user-owned, no sudo needed)
        mkdir -p "$(dirname "$PLIST_FILE")"

        cat > "${PLIST_FILE}" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.naviger.server</string>
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/naviger-server</string>
        <string>--headless</string>
    </array>
    <key>WorkingDirectory</key>
    <string>${INSTALL_DIR}</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>StandardOutPath</key>
    <string>/tmp/naviger.out</string>
    <key>StandardErrorPath</key>
    <string>/tmp/naviger.err</string>
</dict>
</plist>
EOF

        echo "Loading launchd agent..."
        launchctl unload "$PLIST_FILE" 2>/dev/null || true
        launchctl load "$PLIST_FILE"
        echo "Naviger agent installed and loaded (Headless)."
    fi

else
    # DESKTOP MODE
    echo "Installing Desktop Mode..."

    if [ "$OS_TYPE" = "macos" ]; then
        APP_DEST="/Applications/Naviger.app"
        echo "Installing to ${APP_DEST} (requires sudo)..."

        # Look for Naviger.app in extracted files
        if [ -d "${TMP_DIR}/extracted/Naviger.app" ]; then
            run_sudo rm -rf "${APP_DEST}"
            run_sudo cp -r "${TMP_DIR}/extracted/Naviger.app" "/Applications/"
            echo "Naviger.app installed to /Applications."
        else
            echo "Error: Naviger.app not found in the downloaded package."
        fi

    elif [ "$OS_TYPE" = "linux" ]; then
        # Install binaries to /opt/naviger (needs sudo)
        run_sudo mkdir -p "${INSTALL_DIR}"
        run_sudo rm -rf "${INSTALL_DIR}/*"
        run_sudo cp -r "${TMP_DIR}/extracted/"* "${INSTALL_DIR}/"

        # Set permissions
        run_sudo chmod +x "${INSTALL_DIR}/naviger-server"
        run_sudo chown -R "$REAL_USER" "$INSTALL_DIR"

        # Install Desktop Entry
        DESKTOP_FILE="${INSTALL_DIR}/naviger.desktop"
        ICON_FILE="${INSTALL_DIR}/naviger.png"

        if [ -f "$DESKTOP_FILE" ]; then
             run_sudo cp "$DESKTOP_FILE" "/usr/share/applications/naviger.desktop"
             echo "Desktop entry installed."
        else
             echo "Warning: naviger.desktop not found in package. Creating one..."
             cat > naviger.desktop <<EOF
[Desktop Entry]
Type=Application
Name=Naviger
Comment=Naviger Server Manager
Exec=${INSTALL_DIR}/naviger-server
Icon=${ICON_FILE}
Terminal=false
Categories=Development;Server;
EOF
             run_sudo mv naviger.desktop "/usr/share/applications/"
        fi

        echo "Naviger installed in Desktop mode."
    fi

    # Install CLI if available (needs sudo)
    if [ -f "${TMP_DIR}/extracted/naviger-cli" ]; then
        echo "Installing CLI tool..."
        run_sudo cp "${TMP_DIR}/extracted/naviger-cli" "${BIN_DIR}/naviger-cli"
        run_sudo chmod +x "${BIN_DIR}/naviger-cli"
    fi

    # Cleanup
    rm -rf "${TMP_DIR}"
fi

echo "Installation complete!"
