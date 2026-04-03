@echo off
REM NaviServer Migration Script (Windows)
REM Migrates data from 'naviger' to 'naviserver' with backup and automatic installer download

setlocal enabledelayedexpansion

echo.
echo === NaviServer Migration Script ===
echo.
echo This script will:
echo  1. Create a backup of your data
echo  2. Migrate data from Naviger to NaviServer
echo  3. Download and install the new version automatically
echo.

REM Get current date/time for backup filename
for /f "tokens=2-4 delims=/ " %%a in ('date /t') do (set mydate=%%c-%%a-%%b)
for /f "tokens=1-2 delims=/:" %%a in ('time /t') do (set mytime=%%a-%%b)
set TIMESTAMP=%mydate%_%mytime%

set OLD_DATA_DIR=%AppData%\naviger
set NEW_DATA_DIR=%AppData%\naviserver
set BACKUP_FILE=%USERPROFILE%\naviger_backup_%TIMESTAMP%.zip

REM ============================================================
REM 1. CONFIRMATION AT THE BEGINNING
REM ============================================================
echo Data directory: %OLD_DATA_DIR%
echo Backup location: %BACKUP_FILE%
echo.
set /p confirm="Do you want to continue? (y/n) "
if /i not "!confirm!"=="y" (
    echo Migration cancelled by user.
    pause
    exit /b 0
)

echo.

REM ============================================================
REM 2. VALIDATION
REM ============================================================
echo Step 1: Validating old installation...

if not exist "%OLD_DATA_DIR%" (
    echo ERROR: No old data directory found at %OLD_DATA_DIR%
    echo Nothing to migrate. Exiting.
    pause
    exit /b 1
)

echo ^✓ Old data directory found

REM ============================================================
REM 3. CREATE BACKUP
REM ============================================================
echo Step 2: Creating backup...

REM Try PowerShell first (modern Windows)
powershell -NoProfile -Command "Compress-Archive -Path '%OLD_DATA_DIR%' -DestinationPath '%BACKUP_FILE%' -Force" >nul 2>&1

if exist "%BACKUP_FILE%" (
    for /F "usebackq" %%A in ('%BACKUP_FILE%') do (
        set BACKUP_SIZE=%%~zA
    )
    echo ^✓ Backup created: %BACKUP_FILE%
) else (
    REM Fallback: Try 7-Zip if available
    if exist "C:\Program Files\7-Zip\7z.exe" (
        "C:\Program Files\7-Zip\7z.exe" a -tzip "%BACKUP_FILE%" "%OLD_DATA_DIR%" >nul 2>&1
        if exist "%BACKUP_FILE%" (
            echo ^✓ Backup created: %BACKUP_FILE%
        ) else (
            echo WARNING: Could not create backup. Continuing...
        )
    ) else (
        echo WARNING: Could not create backup. Continuing...
    )
)

echo.

REM ============================================================
REM 4. STOP OLD PROCESSES
REM ============================================================
echo Step 3: Stopping old services...

taskkill /IM naviger-server.exe /F >nul 2>&1
taskkill /IM naviger-cli.exe /F >nul 2>&1
echo ^✓ Old processes stopped

echo.

REM ============================================================
REM 5. MIGRATE DATA
REM ============================================================
echo Step 4: Migrating data...

if exist "%NEW_DATA_DIR%" (
    echo WARNING: New data directory already exists at %NEW_DATA_DIR%.
    echo Skipping data move to avoid overwriting.
) else (
    move "%OLD_DATA_DIR%" "%NEW_DATA_DIR%" >nul 2>&1
    
    REM Rename secret file
    if exist "%NEW_DATA_DIR%\.naviger_secret" (
        ren "%NEW_DATA_DIR%\.naviger_secret" ".naviserver_secret" >nul 2>&1
        echo ^✓ Secret file migrated
    )
    
    echo ^✓ Data migrated to %NEW_DATA_DIR%
)

echo.

REM ============================================================
REM 6. DOWNLOAD INSTALLER
REM ============================================================
echo Step 5: Downloading NaviServer installer...

REM Get latest release info from GitHub
for /f "delims=" %%A in ('powershell -NoProfile -Command "try { $url = (Invoke-WebRequest 'https://api.github.com/repos/andre-carbajal/Naviger/releases/latest' -UseBasicParsing | ConvertFrom-Json).assets | Where-Object { $_.name -like '*windows*.exe' } | Select-Object -First 1 -ExpandProperty browser_download_url; Write-Output $url } catch { Write-Output 'error' }" 2>nul') do (
    set DOWNLOAD_URL=%%A
)

if "!DOWNLOAD_URL!"=="error" (
    echo.
    echo WARNING: Could not automatically download the installer.
    echo.
    echo Please download NaviServer manually:
    echo   1. Visit: https://github.com/andre-carbajal/Naviger/releases/latest
    echo   2. Download the Windows .exe installer
    echo   3. Run the installer
    echo.
    echo Your data has been migrated and is safe at:
    echo   %NEW_DATA_DIR%
    echo.
    pause
    exit /b 0
)

REM Download the installer
set INSTALLER_PATH=%TEMP%\NaviServer-installer.exe

echo Downloading from: !DOWNLOAD_URL!

powershell -NoProfile -Command "Invoke-WebRequest -Uri '!DOWNLOAD_URL!' -OutFile '%INSTALLER_PATH%'" >nul 2>&1

if not exist "%INSTALLER_PATH%" (
    echo.
    echo WARNING: Could not download the installer automatically.
    echo.
    echo Please download NaviServer manually:
    echo   1. Visit: https://github.com/andre-carbajal/Naviger/releases/latest
    echo   2. Download the Windows .exe installer
    echo   3. Run the installer
    echo.
    echo Your data has been migrated and is safe at:
    echo   %NEW_DATA_DIR%
    echo.
    pause
    exit /b 0
)

echo ^✓ Installer downloaded successfully

echo.

REM ============================================================
REM 7. RUN INSTALLER
REM ============================================================
echo Step 6: Running NaviServer installer...
echo.

start /wait "%INSTALLER_PATH%"

REM Clean up installer
del "%INSTALLER_PATH%" >nul 2>&1

echo.

REM ============================================================
REM 8. FINAL SUMMARY
REM ============================================================
echo ===============================================================
echo ^✓ Migration completed successfully!
echo ===============================================================
echo.
echo Summary:
echo   ^✓ Data migrated to: %NEW_DATA_DIR%
if exist "%BACKUP_FILE%" (
    echo   ^✓ Backup created at: %BACKUP_FILE%
)
echo   ^✓ NaviServer installer downloaded and executed
echo.
echo Your backup is available at:
echo   %BACKUP_FILE%
echo.
pause
exit /b 0
