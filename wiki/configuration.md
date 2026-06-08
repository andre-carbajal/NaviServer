NaviServer stores configuration in the user config directory under:

- `naviserver` (normal mode)
- `naviserver-dev` (when `NAVISERVER_DEV=true` or `1`)

Key files:

- `config.json`
- `.naviserver_secret` (auto-generated unless overridden via environment variable)
- `.naviserver_cli_token` (auto-generated local CLI authentication token unless overridden via environment variable)

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
- `NAVISERVER_SECRET_KEY`: overrides generated JWT secret.
- `NAVISERVER_CLI_TOKEN`: overrides generated CLI authentication token. Remote CLI clients must set this when they
  cannot read the local `.naviserver_cli_token` file.
- `NAVISERVER_HOST`: overrides `api.host`.
- `NAVISERVER_PORT`: overrides `api.port`.
- `NAVISERVER_ALLOWED_ORIGINS`: comma-separated CORS origins.
- `NAVISERVER_CORS_ALLOWED_ORIGINS`: fallback CORS origins variable.

# CORS origins

NaviServer only allows browser credentials from origins that match one of these rules:

- exact loopback hostnames: `localhost`, `127.0.0.1`, or `::1`;
- exact strings listed in `api.allowed_origins`;
- exact same hostname as the incoming request host.

Lookalike hostnames such as `localhost.example.com` or `127.0.0.1.example.com` are rejected.

# CLI authentication

CLI requests send `X-NaviServer-Client: CLI` and must also include `X-NaviServer-CLI-Token`. The token is generated with
the local configuration in `.naviserver_cli_token` using file mode `0600`. Set `NAVISERVER_CLI_TOKEN` on both the server
and CLI when using the CLI from a separate machine or account that cannot read the generated local token file.
