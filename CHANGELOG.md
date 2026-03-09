# Changelog

### Added
- Automated first-time setup process:
  - Added `GET /auth/setup` endpoint to check if the system requires an initial administrator account.
  - The login page now automatically detects if a setup is needed and redirects to the setup form.
  - Hidden the "Need to setup?" manual link when an administrator already exists.
- Username validation and character support:
  - Prevented spaces in usernames during user creation (both for setup and new users).
  - Added backend and frontend validation to ensure no spaces are allowed in usernames (real-time).
  - Confirmed and ensured that server names correctly allow spaces and special characters.
  - Database supports UTF-8 characters (like tildes) for all names.
- File Management:
  - Added support for uploading entire folders in the server file manager.
  - The folder structure is preserved upon upload using `webkitRelativePath` or recursive traversal for drag & drop.
  - Added a "Folder Up" icon for the new folder upload button in the toolbar.
  - Enabled drag & drop for folders, preserving the entire directory structure.
  - Added the ability to download entire folders as ZIP archives directly from the file explorer.
- Console and Command Management:
  - Added command history in the server console: users can now navigate through previously entered commands using the Up and Down arrow keys (note: this history is local to the current session and is lost upon reloading or leaving the page).
  - Implemented automatic log buffer clearing in the backend when a server stops, ensuring a clean state for the next session.
  - The command history is automatically cleared when the server is stopped for enhanced security and session isolation.
- Dashboard Improvements:
  - Enhanced RAM usage display in the server list: now shows both current usage and maximum allocated memory (e.g., "1.2 GB / 4.0 GB") even when the server is stopped.

### Fixed
- Server icon upload: Added automatic high-quality resizing to 64x64 on the backend, allowing users to upload images of any resolution as server icons.
- Console stability: Fixed race conditions between WebSocket connections and Xterm.js initialization, preventing crashes and missing logs.