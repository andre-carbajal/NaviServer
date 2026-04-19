# Changelog

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
- Docs: updated sprint plan to unified `naviserver` model with `naviserver tui` for interactive navigation and
  `naviserver <subcommand>` for parameter-based CLI usage.
