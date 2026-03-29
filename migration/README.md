# NaviServer Migration Guide

This directory contains scripts to help you migrate from **Naviger** to **NaviServer**.

## Purpose

Since the application has been rebranded, several things have changed:
1. **Application Name:** Naviger -> NaviServer
2. **Binary Names:** `naviger-server` -> `naviserver-server`, `naviger-cli` -> `naviserver-cli`
3. **Data Directories:** 
   - Linux: `~/.config/naviger` -> `~/.config/naviserver`
   - macOS: `~/Library/Application Support/naviger` -> `~/Library/Application Support/naviserver`
   - Windows: `%AppData%\naviger` -> `%AppData%\naviserver`
4. **Secret Key File:** `.naviger_secret` -> `.naviserver_secret`

If you are an existing user, you **must** run these migration scripts before installing the new version to ensure your servers, backups, and user accounts (encryption keys) are preserved.

## When to use these scripts?

You should run the migration script **only once**, after downloading the new version but **before** starting the new NaviServer for the first time or running the new installer.

## How to use

### Windows
1. Open a terminal or double-click `migrate.bat`.
2. Follow the instructions on the screen.
3. Once finished, you can proceed to install the new `NaviServer-windows.exe`.

### Linux & macOS
1. Open a terminal in this directory.
2. Make the script executable: `chmod +x migrate.sh`
3. Run the script: `./migrate.sh`
4. Follow any sudo prompts if required.
5. Once finished, proceed with the new installation (`install.sh` or `.pkg`/`.deb`).

## What do these scripts do?

1. **Stop Services:** They stop any running instances or background services of the old Naviger.
2. **Move Data:** They move your entire configuration, database, server files, and backups to the new `naviserver` directory.
3. **Migrate Encryption:** They rename your secret key file so your user passwords remain valid.
4. **Cleanup:** They remove old symlinks and legacy installation paths.
