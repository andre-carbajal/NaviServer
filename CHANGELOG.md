# Changelog

## Unreleased

- Addon installation:
    - Fixed CurseForge loader and server-environment filtering, including
      server and client/server files.
    - Preserved the source of installed addons and required dependencies,
      including nested dependencies, without cross-source fallback.
    - Added an installation preview that lists missing required dependencies
      with their source, version, filename, and project icon.

- Web UI polish:
    - Replaced native browser `alert` and `confirm` dialogs with in-app modals
      across files, backups, settings, users, and server player actions.
    - Fixed server address copy controls on the dashboard and server detail page
      so they match the dark UI and truncate long addresses cleanly.
    - Reset native `<dialog>` styling for app modals to remove the unwanted
      browser overlay/border effect.
    - Fixed console inputs on mobile
