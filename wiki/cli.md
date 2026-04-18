Binary:
- `naviserver-cli`

Global flag:
- `--url` (default: `http://localhost:<configured_port>`)

If you run `naviserver-cli` without subcommands, it opens the interactive server dashboard (TUI).

# Server commands

```bash
naviserver-cli server
naviserver-cli server start <id>
naviserver-cli server stop <id>
naviserver-cli server delete <id>
```

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
naviserver-cli ports get
naviserver-cli ports set --start <port> --end <port>
naviserver-cli loaders
naviserver-cli update
naviserver-cli restart
```

