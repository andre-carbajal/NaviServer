# Changelog

- TUI create wizard parity: replaced fixed flow with loader-aware dynamic steps (snapshots/unstable toggles and conditional build/loader/installer selectors).
- CLI server create: added advanced flags `--mc-version`, `--include-snapshots`, `--include-unstable`,
  `--build-version`, `--loader-version`, `--installer-version` with latest-stable defaults.
- TUI create wizard compatibility: create requests now send `loaderOptions.mcVersion` to align with the new server
  creation contract.