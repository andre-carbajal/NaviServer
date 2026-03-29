#!/bin/bash

set -e

echo "Cleaning up previous build..."
rm -rf dist

echo "Building web frontend..."
cd web || exit
npm install
npm run build
cd ..

mkdir -p dist/web_dist
cp -r web/dist/* dist/web_dist/

# Calculate version for injection
VERSION="${NAVISERVER_VERSION:-}"
if [ -z "${VERSION}" ] && [ -f "internal/updater/updater.go" ]; then
    VERSION=$(grep -E 'CurrentVersion\s*=' internal/updater/updater.go | head -n 1 | awk -F'"' '{print $2}')
    VERSION="${VERSION#v}"
fi
if [ -z "${VERSION}" ]; then
    VERSION="dev"
fi

echo "Building Go backend with version: v${VERSION}"
LDFLAGS="-X 'naviserver/internal/updater.CurrentVersion=v${VERSION}'"

echo "Building server..."
if [[ "$OSTYPE" == "darwin"* ]]; then
    GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -v -o dist/naviserver-server-amd64 ./cmd/server
    GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -v -o dist/naviserver-server-arm64 ./cmd/server
    lipo -create -output dist/naviserver-server dist/naviserver-server-amd64 dist/naviserver-server-arm64
    rm dist/naviserver-server-amd64 dist/naviserver-server-arm64
else
    go build -ldflags "${LDFLAGS}" -v -o dist/naviserver-server ./cmd/server
fi

echo "Building CLI..."
if [[ "$OSTYPE" == "darwin"* ]]; then
    GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -v -o dist/naviserver-cli-amd64 ./cmd/cli
    GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -v -o dist/naviserver-cli-arm64 ./cmd/cli
    lipo -create -output dist/naviserver-cli dist/naviserver-cli-amd64 dist/naviserver-cli-arm64
    rm dist/naviserver-cli-amd64 dist/naviserver-cli-arm64
else
    go build -ldflags "${LDFLAGS}" -v -o dist/naviserver-cli ./cmd/cli
fi

if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "Creating macOS Application Bundle..."
    APP_NAME="NaviServer"
    APP_DIR="dist/${APP_NAME}.app"
    CONTENTS_DIR="${APP_DIR}/Contents"
    MACOS_DIR="${CONTENTS_DIR}/MacOS"
    RESOURCES_DIR="${CONTENTS_DIR}/Resources"

    mkdir -p "${MACOS_DIR}"
    mkdir -p "${RESOURCES_DIR}"

    cp "dist/naviserver-server" "${MACOS_DIR}/${APP_NAME}"

    cp -r "dist/web_dist" "${MACOS_DIR}/"

    cat > "${CONTENTS_DIR}/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>${APP_NAME}</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>CFBundleIdentifier</key>
    <string>com.naviserver.server</string>
    <key>CFBundleName</key>
    <string>${APP_NAME}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>2.0.0</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSHumanReadableCopyright</key>
    <string>Universal binary</string>
</dict>
</plist>
EOF

    if command -v sips >/dev/null 2>&1; then
        ICON_SRC="cmd/server/icon.png"
        if [ -f "$ICON_SRC" ]; then
            echo "Generating AppIcon.icns..."
            ICONSET="${RESOURCES_DIR}/AppIcon.iconset"
            mkdir -p "${ICONSET}"

            sips -z 16 16     "$ICON_SRC" --out "${ICONSET}/icon_16x16.png" > /dev/null 2>&1
            sips -z 32 32     "$ICON_SRC" --out "${ICONSET}/icon_16x16@2x.png" > /dev/null 2>&1
            sips -z 32 32     "$ICON_SRC" --out "${ICONSET}/icon_32x32.png" > /dev/null 2>&1
            sips -z 64 64     "$ICON_SRC" --out "${ICONSET}/icon_32x32@2x.png" > /dev/null 2>&1
            sips -z 128 128   "$ICON_SRC" --out "${ICONSET}/icon_128x128.png" > /dev/null 2>&1
            sips -z 256 256   "$ICON_SRC" --out "${ICONSET}/icon_128x128@2x.png" > /dev/null 2>&1
            sips -z 256 256   "$ICON_SRC" --out "${ICONSET}/icon_256x256.png" > /dev/null 2>&1
            sips -z 512 512   "$ICON_SRC" --out "${ICONSET}/icon_256x256@2x.png" > /dev/null 2>&1
            sips -z 512 512   "$ICON_SRC" --out "${ICONSET}/icon_512x512.png" > /dev/null 2>&1

            if command -v iconutil >/dev/null 2>&1; then
                iconutil -c icns "${ICONSET}" -o "${RESOURCES_DIR}/AppIcon.icns"
                rm -rf "${ICONSET}"
            else
                rm -rf "${ICONSET}"
            fi
        fi
    fi

    echo "macOS App Bundle created at ${APP_DIR}"

    if command -v pkgbuild >/dev/null 2>&1; then
        PKG_ROOT="dist/pkg_root"
        APP_DST="${PKG_ROOT}/Applications"
        BIN_DST="${PKG_ROOT}/usr/local/bin"

        rm -rf "${PKG_ROOT}"
        mkdir -p "${APP_DST}" "${BIN_DST}"

        cp -R "${APP_DIR}" "${APP_DST}/"
        if [ -f "dist/naviserver-cli" ]; then
            cp "dist/naviserver-cli" "${BIN_DST}/naviserver-cli"
            chmod +x "${BIN_DST}/naviserver-cli"
        fi

        PKG_NAME="NaviServer-${VERSION}-macos.pkg"
        echo "Creating PKG ${PKG_NAME}..."
        pkgbuild --root "${PKG_ROOT}" --install-location / --identifier "com.naviserver.server" --version "${VERSION}" "dist/${PKG_NAME}"

        rm -rf "${PKG_ROOT}"
    fi

elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "Creating Linux Desktop Entry..."

    ICON_SRC="cmd/server/icon.png"
    if [ -f "$ICON_SRC" ]; then
        cp "$ICON_SRC" "dist/naviserver.png"
    fi

    cat > "dist/naviserver.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=NaviServer
Comment=NaviServer Server Manager
Exec=/opt/naviserver/naviserver-server
Icon=/usr/share/pixmaps/naviserver.png
Terminal=false
Categories=Development;Server;
EOF

    chmod +x "dist/naviserver.desktop"

    echo "Linux desktop files created in dist/"

    if command -v dpkg-deb >/dev/null 2>&1; then
        DEB_ROOT="dist/deb_root"
        rm -rf "${DEB_ROOT}"

        mkdir -p "${DEB_ROOT}/opt/naviserver"
        mkdir -p "${DEB_ROOT}/usr/local/bin"
        mkdir -p "${DEB_ROOT}/usr/share/applications"
        mkdir -p "${DEB_ROOT}/usr/share/pixmaps"
        mkdir -p "${DEB_ROOT}/DEBIAN"

        cp "dist/naviserver-server" "${DEB_ROOT}/opt/naviserver/naviserver-server"
        chmod +x "${DEB_ROOT}/opt/naviserver/naviserver-server"
        cp "dist/naviserver-cli" "${DEB_ROOT}/usr/local/bin/naviserver-cli"
        chmod +x "${DEB_ROOT}/usr/local/bin/naviserver-cli"

        cp -r "dist/web_dist" "${DEB_ROOT}/opt/naviserver/web_dist"

        cp "dist/naviserver.png" "${DEB_ROOT}/usr/share/pixmaps/naviserver.png"
        cp "dist/naviserver.desktop" "${DEB_ROOT}/usr/share/applications/naviserver.desktop"

        INSTALLED_SIZE=$(du -ks "${DEB_ROOT}" | cut -f1)

        cat > "${DEB_ROOT}/DEBIAN/control" <<EOF
Package: naviserver
Version: ${VERSION}
Architecture: amd64
Maintainer: Andre Carbajal
Installed-Size: ${INSTALLED_SIZE}
Description: Modern Minecraft Server Manager
 NaviServer is a lightweight, cross-platform Minecraft server manager
 with Web UI, CLI, and native integration.
Depends: libc6
Section: utils
Priority: optional
Homepage: https://github.com/andre-carbajal/NaviServer
EOF

        cat > "${DEB_ROOT}/DEBIAN/postinst" <<'EOF'
#!/bin/bash
set -e
ln -sf /opt/naviserver/naviserver-server /usr/local/bin/naviserver-server
EOF
        chmod 755 "${DEB_ROOT}/DEBIAN/postinst"

        cat > "${DEB_ROOT}/DEBIAN/postrm" <<'EOF'
#!/bin/bash
set -e
rm -f /usr/local/bin/naviserver-server
EOF
        chmod 755 "${DEB_ROOT}/DEBIAN/postrm"

        DEB_NAME="NaviServer-${VERSION}-linux.deb"
        echo "Creating DEB ${DEB_NAME}..."
        dpkg-deb --build "${DEB_ROOT}" "dist/${DEB_NAME}"

        rm -rf "${DEB_ROOT}"
        echo "DEB created at dist/${DEB_NAME}"
    else
        echo "dpkg-deb not found, skipping .deb creation."
    fi
fi

echo "Build finished successfully!"
