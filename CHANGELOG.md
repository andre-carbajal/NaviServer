# Changelog

- Automated first-time setup process:
  - Added `GET /auth/setup` endpoint to check if the system requires an initial administrator account.
  - The login page now automatically detects if a setup is needed and redirects to the setup form.
  - Hidden the "Need to setup?" manual link when an administrator already exists.