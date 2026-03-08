# Changelog

- Automated first-time setup process:
  - Added `GET /auth/setup` endpoint to check if the system requires an initial administrator account.
  - The login page now automatically detects if a setup is needed and redirects to the setup form.
  - Hidden the "Need to setup?" manual link when an administrator already exists.
- Username validation and character support:
  - Prevented spaces in usernames during user creation (both for setup and new users).
  - Added backend and frontend validation to ensure no spaces are allowed in usernames.
  - Confirmed and ensured that server names correctly allow spaces and special characters.
  - Database supports UTF-8 characters (like tildes) for all names.
