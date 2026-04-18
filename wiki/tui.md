NaviServer includes multiple Bubble Tea based TUI screens.

# Server dashboard (`naviserver-cli` or `naviserver-cli server`)

Primary keys:
- `/`: filter servers
- `c`: create server (wizard)
- `s`: start selected server
- `x`: stop selected server
- `d`: delete selected server (with confirmation)
- `Enter`: open logs for selected server
- `q` or `Esc`: quit

The dashboard refreshes automatically and shows per-server stats (CPU, RAM, disk).

# Logs screen

Opened from the dashboard using `Enter`.

Primary keys:
- `Esc`: return to dashboard
- `Ctrl+C`: quit
- `Enter`: send command from the console input

# Backup dashboard (`naviserver-cli backup`)

Keys in list mode:
- `/`: filter backups
- `c`: open create wizard
- `r`: open restore wizard
- `d`: delete selected backup (with confirmation)
- `q` or `Esc`: back/quit

Create flow:
1. Select server.
2. Optional backup name.
3. Confirm creation.

Restore flow:
1. Select target server or create a new one.
2. For new server: name -> loader -> version -> RAM.
3. Confirm restore.

