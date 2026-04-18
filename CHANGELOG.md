# Changelog

- **API/WebSocket configuration improvements** (`73055d0`, `e310559`):
  - Refactored frontend API/WS URL resolution to improve environment-based configuration.
  - Extended backend config handling for API host/port and allowed CORS origins.
  - Improved environment variable support and precedence for runtime/network settings.
  - Updated related frontend flows and backend wiring (`cmd/server`, `internal/config`, `internal/api`, `web/src/services/api.ts`).

- **Documentation expansion** (`072175b`):
  - Added comprehensive documentation pages under `wiki/` for CLI usage, installation, configuration, migration from 1.x, TUI usage, and wiki home navigation.

- **Frontend dependency maintenance (security/stability)**:
  - Bumped `axios` from `1.14.0` to `1.15.0` (`8e58f40`, merged via `5ba995b`).
  - Bumped `follow-redirects` from `1.15.11` to `1.16.0` (`52804e1`, merged via `2f89296`).
  - Bumped `vite` from `8.0.3` to `8.0.8` (`ef1bad2`, merged via `ef5f0ec`).
