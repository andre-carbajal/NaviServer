# Changelog

## Unreleased

- Security hardening:
    - Pinned the release workflow's third-party GitHub Actions to full commit
      SHAs and disabled npm lifecycle scripts during all supported builds.
    - Restricted installer downloads to HTTPS with TLS 1.2 or newer.
    - Added transport-aware authentication cookies and the
      `NAVISERVER_TRUST_PROXY` setting, including configuration migration and
      documentation.
    - Made Forge and NeoForge use an absolute path to a managed Java runtime
      for server creation and updates; managed Java binaries now use `0700`
      permissions.
    - Documented the Modrinth-required SHA-1 content lookup as non-cryptographic
      and unrelated to authentication or trust decisions.

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
    - Made addon selection signatures use locale-aware ordering for reliable
      alphabetical sorting.
    - Moved file-row click handling onto the action buttons so file operations
      remain keyboard-accessible.
    - Consolidated duplicate responsive CSS selectors and replaced the obsolete
      `word-break: break-word` declaration with the existing overflow wrapping.

- Reliability:
    - Updated Bash conditionals in the build, installation, migration, and
      uninstallation scripts to use Bash's safer `[[ ... ]]` syntax.
    - Made JSX spacing explicit around inline controls and standardized numeric
      parsing and newline normalization in the web frontend.
    - Simplified frontend iteration, DOM cleanup, optional access, and progress
      state handling without changing upload or console behavior.
    - Replaced nested frontend conditional expressions with explicit branches
      while preserving editor, addon, backup, chart, and server-settings states.
    - Released service context resources when daemon goroutines finish and
      preserved the full graceful-shutdown timeout during cancellation.

- Build and installer maintenance:
    - Sorted Docker package lists, quoted runtime UID/GID arguments, routed
      installer errors to stderr, and clarified blank-import registration.

- CLI and API maintenance:
    - Reused the WebSocket handler for console and progress routes, merged
      duplicate CLI key branches, and removed an empty dashboard branch.
    - Grouped consecutive same-typed Go parameters across repository,
      backup, server, JVM, storage, and supervisor APIs.
