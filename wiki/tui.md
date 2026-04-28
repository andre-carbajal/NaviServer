NaviServer includes multiple Bubble Tea based TUI screens.

# Main menu (`naviserver-cli tui`)

Primary keys:

- `Enter`: open selected section
- `s`: open Servers
- `b`: open Backups
- `g`: open Settings
- `u`: open Users
- `?`: toggle contextual help
- `q` or `Esc`: quit to terminal
- `Ctrl+C`: force exit

# Server dashboard (`naviserver-cli tui`)

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

# Settings dashboard (`naviserver-cli tui` -> Settings)

List mode keys:

- `Enter`: open/edit selected setting
- `r`: refresh settings from API
- `?`: toggle contextual help
- `q` or `Esc`: back to main menu

Sub-modes:

- Network Configuration:
  - `Enter`: edit Start/End port
  - `q` or `Esc`: back to Settings list
- Numeric Edit (ports / log buffer):
  - `Enter`: save
  - `q` or `Esc`: cancel
- Public Address selection:
  - `Enter`: save selected address
  - `q` or `Esc`: cancel without saving

Behavior highlights:

- Public Address shows `localhost`, detected interface IPs, and current unavailable value if present.
- Log buffer shows an estimated memory usage (`lines * ~200 bytes`).

# Users dashboard (`naviserver-cli tui` -> Users)

List mode keys:

- `/`: filter users
- `c`: create user
- `x`: edit permissions for selected user
- `p`: change password for selected user
- `d`: delete selected user (with confirmation)
- `r`: refresh users
- `?`: toggle contextual help
- `q` or `Esc`: back to main menu

Create mode:

- `Tab` / `Shift+Tab`: switch between username and password
- `Enter`: next field / create
- `Esc`: cancel

Edit permissions mode:

- `Up/Down`: select server row
- `Tab` / `Left` / `Right`: switch active permission column
- `Space`: toggle current permission
- `Enter` or `s`: save permissions
- `r`: reload servers + permissions
- `q` or `Esc`: cancel and return

Permission rule:

- `Console & Files` implies `Power Control`.
- Disabling `Power Control` disables `Console & Files`.

Delete confirmation:

- `y` or `Enter`: confirm delete
- `q`, `n`, or `Esc`: cancel
