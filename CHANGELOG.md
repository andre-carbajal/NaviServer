# Changelog

- Added logs autoscroll toggle and simple in-buffer search (`/`, `n/N`) to inspect long console streams without losing context.
- Improved logs WebSocket observability with explicit connection state (connecting/connected/reconnecting/disconnected) and automatic reconnection behavior.
- Improved long-flow feedback in server create and backup restore with explicit `Status: Running/Done/Failed` states to avoid ambiguous outcomes.
- Added contextual help toggle (`?`) across main menu, servers dashboard, backups dashboard, logs,
  and create wizards.
- TUI keybinding unification: aligned global behavior and footer hints for `q/esc`, `ctrl+c`, and `enter` across
  primary views.
- TUI messaging standardization: introduced shared status/error rendering (`Status:` / `Error:`) and a common
  confirmation convention (`Confirm: y/Enter | Cancel: n/Esc`) in delete/restore/create flows.
- Fixed mojibake/corrupt characters in dashboards and wizards.
- Logs: fixed title typo from `SEVER CONSOLE LOGS` to `SERVER CONSOLE LOGS`.
- UI consistency: standardized footer/help separators and aligned status iconography across server, backup, and logs
  views.
- UI consistency: standardized server creation wizard footer/help styling to match the shared TUI pattern (`keyStyle` +
  `descStyle` + ` • ` separators).
- CLI behavior: fixed command surface so TUI opens only with `naviserver-cli tui`; running `naviserver-cli`,
  `naviserver-cli backup`, or `naviserver-cli server` now shows contextual help instead of opening interactive UI.
- TUI navigation: added a main hub section for easier entry to `Servers` and `Backups`; removed top-level `Logs` from
  hub since logs are accessed per server.