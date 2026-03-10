# Changelog

- Unified project versioning:
  - Transitioned `internal/updater.CurrentVersion` from a constant to a variable to support dynamic version injection during compilation.
  - Implemented version injection using Go `ldflags` in both `build.sh` and `build.bat`.
  - Set the default version in the source code to `"dev"` to clearly distinguish development builds from official releases.
  - Updated GitHub Actions to automatically inject the release tag version into the binary, ensuring consistency across all platforms.
