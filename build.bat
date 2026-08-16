@echo off
setlocal

echo "Cleaning up previous build..."
if exist dist (
    rmdir /s /q dist
)

echo "Building web frontend..."
pushd web
call npm install --ignore-scripts
call npm run build
popd

mkdir dist\web_dist
xcopy /s /e /i /y web\dist\* dist\web_dist\

echo "Building Go backend..."
set VERSION=%NAVISERVER_VERSION%
if "%VERSION%"=="" (
    set VERSION=dev
)
echo "Building version: v%VERSION%"
set CURSEFORGE_KEY=%NAVISERVER_CURSEFORGE_API_KEY%
set LDFLAGS=-X "naviserver/internal/updater.CurrentVersion=v%VERSION%" -X "naviserver/internal/addons.BuildCurseForgeAPIKey=%CURSEFORGE_KEY%"

echo "Building server..."
call go build -ldflags "-H=windowsgui %LDFLAGS%" -v -o dist\naviserver-server.exe .\cmd\server

echo "Building CLI..."
call go build -ldflags "%LDFLAGS%" -v -o dist\naviserver-cli.exe .\cmd\cli

echo "Build finished successfully!"
endlocal
