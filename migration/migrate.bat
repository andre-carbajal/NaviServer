@echo off
setlocal enabledelayedexpansion

echo --- Starting Migration: Naviger -^> NaviServer ---

:: 1. Stop old process
echo Stopping Naviger process if running...
taskkill /IM naviger-server.exe /F >nul 2>&1
taskkill /IM naviger-cli.exe /F >nul 2>&1

:: 2. Migrate Data Directory (AppData)
set OLD_DATA_DIR=%AppData%\naviger
set NEW_DATA_DIR=%AppData%\naviserver

if exist "%OLD_DATA_DIR%" (
    if exist "%NEW_DATA_DIR%" (
        echo Warning: New data directory already exists at %NEW_DATA_DIR%.
        echo Skipping data move to avoid overwriting.
    ) else (
        echo Moving data from %OLD_DATA_DIR% to %NEW_DATA_DIR%...
        move "%OLD_DATA_DIR%" "%NEW_DATA_DIR%"
        
        :: Rename secret file
        if exist "%NEW_DATA_DIR%\.naviger_secret" (
            ren "%NEW_DATA_DIR%\.naviger_secret" ".naviserver_secret"
            echo Secret file migrated successfully.
        )
        
        echo Data migrated successfully.
    )
) else (
    echo No old data directory found at %OLD_DATA_DIR%. Skipping data migration.
)

:: 3. Handle Installation Directory (Common Program Files)
:: The installer usually handles its own directory, but we can help clean up if needed.
set "OLD_INSTALL_64=%ProgramFiles%\Naviger"
set "OLD_INSTALL_32=%ProgramFiles(x86)%\Naviger"

if exist "%OLD_INSTALL_64%" (
    echo Found old 64-bit installation. Note: Please uninstall the old version manually after migration.
)
if exist "%OLD_INSTALL_32%" (
    echo Found old 32-bit installation. Note: Please uninstall the old version manually after migration.
)

echo.
echo --- Migration Finished ---
echo You can now install NaviServer and it will use your existing data.
echo.
pause
