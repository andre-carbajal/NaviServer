# Changelog

- Added real-time Minecraft server player monitoring:
  - Integrated [go-mcstatus](https://github.com/andre-carbajal/go-mcstatus) library to query server status.
  - Displaying online and max player counts in the server list.
  - Added a dedicated "Players" stat card to the server detail view (console page).
  - Updated API and frontend types to support player statistics.
- Unified project versioning:
  - Transitioned `internal/updater.CurrentVersion` from a constant to a variable to support dynamic version injection during compilation.
  - Implemented version injection using Go `ldflags` in both `build.sh` and `build.bat`.
  - Set the default version in the source code to `"dev"` to clearly distinguish development builds from official releases.
  - Updated GitHub Actions to automatically inject the release tag version into the binary, ensuring consistency across all platforms.
