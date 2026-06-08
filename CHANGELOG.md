# Changelog

- CLI authentication and token management:
    - Added secure CLI token generation, loading, and validation.
    - Hardened API authentication middleware and SDK client handling.
    - Documented the token-based configuration flow and covered it with security
      and secret-management tests.
- Server version updates:
    - Added pre-update backups before changing server versions.
    - Added automatic addon compatibility resolution during version updates.
    - Improved frontend progress tracking for long-running update operations.
    - Updated Paper loader support for Paper API v3.
- Addons management:
    - Added CurseForge API key management across configuration, API endpoints, and
      the Settings UI.
    - Added CurseForge and Modrinth addon discovery/management support, including
      fingerprinting and addon metadata handling.
    - Added enable/disable actions for addons through the backend API and the
      Addons panel UI.
- Server dashboard:
    - Updated the server detail dashboard with CPU and RAM usage graphs for
      real-time performance monitoring.
    - Added in-dashboard player administration for search, filtering, and
      moderation actions.
    - Added server settings management directly from the dashboard, including
      gameplay and performance configuration.
- Server settings and customization:
    - Added server settings retrieval and update endpoints backed by
      `server.properties` parsing/writing.
    - Added server version update endpoints and UI integration.
    - Added spawn protection configuration.
    - Added server icon selection and upload support.
- Backups:
    - Added automatic backup configuration management in storage, API handlers,
      backup manager logic, and the Backups UI.
    - Improved backup modals and backup list interactions.
- Player and process management:
    - Added server restart and force-kill actions.
    - Enhanced server stats with uptime and player information.
    - Improved player management with search, filters, and a moderation actions
      modal.
- Server creation and loader metadata:
    - Added loader options and metadata retrieval for Vanilla, Fabric, Forge,
      NeoForge, Paper, and Quilt, including loader icon assets and API support.
    - Improved create flows in the web UI, SDK, CLI, and TUI with loader-aware
      options and metadata.
    - TUI create wizard parity: replaced fixed flow with loader-aware dynamic
      steps, including snapshots/unstable toggles and conditional build, loader,
      and installer selectors.
    - CLI server create: added advanced flags `--mc-version`,
      `--include-snapshots`, `--include-unstable`, `--build-version`,
      `--loader-version`, and `--installer-version` with latest-stable defaults.
    - TUI create wizard compatibility: create requests now send
      `loaderOptions.mcVersion` to align with the server creation contract.
