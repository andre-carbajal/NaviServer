@echo off
setlocal

echo "Cleaning up previous build..."
if exist dist (
    rmdir /s /q dist
)

echo "Building web frontend..."
pushd web
call npm install
call npm run build
popd

mkdir dist\web_dist
xcopy /s /e /i /y web\dist\* dist\web_dist\

echo "Building Go backend..."
set VERSION=%NAVIGER_VERSION%
if "%VERSION%"=="" (
    set VERSION=dev
)
echo "Building version: v%VERSION%"
set LDFLAGS=-X "github.com/andre-carbajal/Naviger/internal/updater.CurrentVersion=v%VERSION%"

echo "Building server..."
call go build -ldflags "-H=windowsgui %LDFLAGS%" -v -o dist\naviger-server.exe .\cmd\server

echo "Building CLI..."
call go build -ldflags "%LDFLAGS%" -v -o dist\naviger-cli.exe .\cmd\cli

echo "Build finished successfully!"
endlocal
