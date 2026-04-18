NaviServer stores configuration in the user config directory under:
- `naviserver` (normal mode)
- `naviserver-dev` (when `NAVISERVER_DEV=true` or `1`)

Key files:
- `config.json`
- `.naviserver_secret` (auto-generated unless overridden via environment variable)

# `config.json` defaults

```json
{
  "servers_path": "<configDir>/servers",
  "backups_path": "<configDir>/backups",
  "runtimes_path": "<configDir>/runtimes",
  "database_path": "<configDir>/manager.db",
  "api": {
    "host": "0.0.0.0",
    "port": 23008,
    "allowed_origins": []
  }
}
```

If `NAVISERVER_DEV=true` or `1`, default API port becomes `23009`.

# Environment variables

- `NAVISERVER_DEV`: enables dev mode (`true`/`1`).
- `NAVISERVER_SECRET_KEY`: overrides generated secret.
- `NAVISERVER_HOST`: overrides `api.host`.
- `NAVISERVER_PORT`: overrides `api.port`.
- `NAVISERVER_ALLOWED_ORIGINS`: comma-separated CORS origins.
- `NAVISERVER_CORS_ALLOWED_ORIGINS`: fallback CORS origins variable.

