# Changelog

- Web UI polish:
    - Replaced native browser `alert` and `confirm` dialogs with in-app modals
      across files, backups, settings, users, and server player actions.
    - Fixed server address copy controls on the dashboard and server detail page
      so they match the dark UI and truncate long addresses cleanly.
    - Reset native `<dialog>` styling for app modals to remove the unwanted
      browser overlay/border effect.
    - Fixed console inputs on mobile