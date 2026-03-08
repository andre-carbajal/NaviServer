# Changelog

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