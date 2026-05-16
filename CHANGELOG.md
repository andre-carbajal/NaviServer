# Changelog

- Loader catalog expansion: added Quilt as a supported loader in backend factory and loader listings.
- CLI server create: added advanced flags `--mc-version`, `--include-snapshots`, `--include-unstable`,
  `--build-version`, `--loader-version`, `--installer-version` with latest-stable defaults.
- TUI create wizard compatibility: create requests now send `loaderOptions.mcVersion` to align with the new server
  creation contract.