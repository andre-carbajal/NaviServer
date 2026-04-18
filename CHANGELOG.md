# Changelog

## Unreleased

- **Config migration and backward compatibility**:
  - Added automatic non-destructive migration for `config.json` on startup.
  - Missing fields are backfilled (`servers_path`, `backups_path`, `runtimes_path`, `database_path`, `api.host`, `api.port`, `api.allowed_origins`).
  - Existing user values are preserved and never overwritten during migration.
  - Unknown/custom keys in `config.json` (including nested `api` keys) are preserved.

- **Frontend API base URL fix (dev/proxy scenarios)**:
  - Improved API base URL resolution to avoid unintended fallback to `:23009` when `VITE_API_PORT` is injected as a non-string value.
  - Better compatibility for local development and reverse proxy setups.

- **Tests**:
  - Added migration tests to validate:
	- Backfilling missing config fields.
	- Preserving existing and custom config values.
	- Keeping env override precedence without rewriting persisted file values.
