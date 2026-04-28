Binary:

- `naviserver-cli`

Global flag:

- `--url` (default: `http://localhost:<configured_port>`)

If you run `naviserver-cli` without subcommands, it shows CLI help.
Use `naviserver-cli tui` to open the interactive dashboard.

# Server commands

```bash
naviserver-cli server
naviserver-cli server list
naviserver-cli server create --name <name> [--loader <loader>] [--version <version>] [--ram <mb>]
naviserver-cli server create --name <name> --async
naviserver-cli server start <id>
naviserver-cli server stop <id>
naviserver-cli server delete <id>
```

`server create` runs in synchronous mode by default: it waits for completion using the progress WebSocket.
Use `--async` to return immediately after the create request is accepted.

# Backup commands

```bash
naviserver-cli backup
naviserver-cli backup create <serverId> [name]
naviserver-cli backup list [serverId]
naviserver-cli backup delete <name>
```

Restore backup to an existing server:

```bash
naviserver-cli backup restore <name> --target <serverId>
```

Restore backup to a new server:

```bash
naviserver-cli backup restore <name> --new --name <newName> --loader <loader> --version <version> --ram <mb>
```

Restore defaults:

- `--version 1.20.1`
- `--loader vanilla`
- `--ram 2048`

## Misc commands

```bash
naviserver-cli loaders
naviserver-cli update
naviserver-cli restart
```

## Settings commands

```bash
naviserver-cli settings
naviserver-cli settings port-range get
naviserver-cli settings port-range set --start <port> --end <port>
naviserver-cli settings public-ip get
naviserver-cli settings public-ip set --value <ip-or-host>
naviserver-cli settings interfaces list
naviserver-cli settings log-buffer get
naviserver-cli settings log-buffer set --lines <n>
```

## User commands

```bash
naviserver-cli user
naviserver-cli user list
naviserver-cli user create --username <name> --password <pass>
naviserver-cli user delete <user-id>
naviserver-cli user password set <user-id> --password <pass>
naviserver-cli user permissions get <user-id>
naviserver-cli user permissions set <user-id> --server <server-id> --power <true|false> --console <true|false>
```
