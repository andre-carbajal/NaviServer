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
VERSION="${NAVIGER_VERSION:-}"
if [ -z "${VERSION}" ] && [ -f "internal/updater/updater.go" ]; then
    VERSION=$(grep -E 'CurrentVersion\s*=' internal/updater/updater.go | head -n 1 | awk -F'"' '{print $2}')
    VERSION="${VERSION#v}"
fi
if [ -z "${VERSION}" ]; then
    VERSION="dev"
fi

echo "Building Go backend with version: v${VERSION}"
LDFLAGS="-X 'github.com/andre-carbajal/Naviger/internal/updater.CurrentVersion=v${VERSION}'"

echo "Building server..."
if [[ "$OSTYPE" == "darwin"* ]]; then
    GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -v -o dist/naviger-server-amd64 ./cmd/server
    GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -v -o dist/naviger-server-arm64 ./cmd/server
    lipo -create -output dist/naviger-server dist/naviger-server-amd64 dist/naviger-server-arm64
    rm dist/naviger-server-amd64 dist/naviger-server-arm64
else
    go build -ldflags "${LDFLAGS}" -v -o dist/naviger-server ./cmd/server
fi

echo "Building CLI..."
if [[ "$OSTYPE" == "darwin"* ]]; then
    GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -v -o dist/naviger-cli-amd64 ./cmd/cli
    GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -v -o dist/naviger-cli-arm64 ./cmd/cli
    lipo -create -output dist/naviger-cli dist/naviger-cli-amd64 dist/naviger-cli-arm64
    rm dist/naviger-cli-amd64 dist/naviger-cli-arm64
else
    go build -ldflags "${LDFLAGS}" -v -o dist/naviger-cli ./cmd/cli
fi

if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "Creating macOS Application Bundle..."
    APP_NAME="Naviger"
    APP_DIR="dist/${APP_NAME}.app"
    CONTENTS_DIR="${APP_DIR}/Contents"
    MACOS_DIR="${CONTENTS_DIR}/MacOS"
    RESOURCES_DIR="${CONTENTS_DIR}/Resources"

    mkdir -p "${MACOS_DIR}"
    mkdir -p "${RESOURCES_DIR}"

    cp "dist/naviger-server" "${MACOS_DIR}/${APP_NAME}"

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
    <string>com.naviger.server</string>
    <key>CFBundleName</key>
    <string>${APP_NAME}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0.0</string>
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
        if [ -f "dist/naviger-cli" ]; then
            cp "dist/naviger-cli" "${BIN_DST}/naviger-cli"
            chmod +x "${BIN_DST}/naviger-cli"
        fi

        PKG_NAME="Naviger-${VERSION}-macos.pkg"
        echo "Creating PKG ${PKG_NAME}..."
        pkgbuild --root "${PKG_ROOT}" --install-location / --identifier "com.naviger.server" --version "${VERSION}" "dist/${PKG_NAME}"

        rm -rf "${PKG_ROOT}"
    fi

elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "Creating Linux Desktop Entry..."

    ICON_SRC="cmd/server/icon.png"
    if [ -f "$ICON_SRC" ]; then
        cp "$ICON_SRC" "dist/naviger.png"
    fi

    cat > "dist/naviger.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=Naviger
Comment=Naviger Server Manager
Exec=/opt/naviger/naviger-server
Icon=/usr/share/pixmaps/naviger.png
Terminal=false
Categories=Development;Server;
EOF

    chmod +x "dist/naviger.desktop"

    echo "Linux desktop files created in dist/"

    if command -v dpkg-deb >/dev/null 2>&1; then
        DEB_ROOT="dist/deb_root"
        rm -rf "${DEB_ROOT}"

        mkdir -p "${DEB_ROOT}/opt/naviger"
        mkdir -p "${DEB_ROOT}/usr/local/bin"
        mkdir -p "${DEB_ROOT}/usr/share/applications"
        mkdir -p "${DEB_ROOT}/usr/share/pixmaps"
        mkdir -p "${DEB_ROOT}/DEBIAN"

        cp "dist/naviger-server" "${DEB_ROOT}/opt/naviger/naviger-server"
        chmod +x "${DEB_ROOT}/opt/naviger/naviger-server"
        cp "dist/naviger-cli" "${DEB_ROOT}/usr/local/bin/naviger-cli"
        chmod +x "${DEB_ROOT}/usr/local/bin/naviger-cli"

        cp -r "dist/web_dist" "${DEB_ROOT}/opt/naviger/web_dist"

        cp "dist/naviger.png" "${DEB_ROOT}/usr/share/pixmaps/naviger.png"
        cp "dist/naviger.desktop" "${DEB_ROOT}/usr/share/applications/naviger.desktop"

        INSTALLED_SIZE=$(du -ks "${DEB_ROOT}" | cut -f1)

        cat > "${DEB_ROOT}/DEBIAN/control" <<EOF
Package: naviger
Version: ${VERSION}
Architecture: amd64
Maintainer: Andre Carbajal
Installed-Size: ${INSTALLED_SIZE}
Description: Modern Minecraft Server Manager
 Naviger is a lightweight, cross-platform Minecraft server manager
 with Web UI, CLI, and native integration.
Depends: libc6
Section: utils
Priority: optional
Homepage: https://github.com/andre-carbajal/Naviger
EOF

        cat > "${DEB_ROOT}/DEBIAN/postinst" <<'EOF'
#!/bin/bash
set -e
ln -sf /opt/naviger/naviger-server /usr/local/bin/naviger-server
EOF
        chmod 755 "${DEB_ROOT}/DEBIAN/postinst"

        cat > "${DEB_ROOT}/DEBIAN/postrm" <<'EOF'
#!/bin/bash
set -e
rm -f /usr/local/bin/naviger-server
EOF
        chmod 755 "${DEB_ROOT}/DEBIAN/postrm"

        DEB_NAME="Naviger-${VERSION}-linux.deb"
        echo "Creating DEB ${DEB_NAME}..."
        dpkg-deb --build "${DEB_ROOT}" "dist/${DEB_NAME}"

        rm -rf "${DEB_ROOT}"
        echo "DEB created at dist/${DEB_NAME}"
    else
        echo "dpkg-deb not found, skipping .deb creation."
    fi
fi

echo "Build finished successfully!"
