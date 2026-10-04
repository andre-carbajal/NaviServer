# Development guide

## Scope and prerequisites

Source-derived instructions for the current checkout; Linux/Bash commands below. See [verification evidence](index.md#documentation-pass-checks) for what was actually exercised. This guide does not claim a clean-machine install or a live browser smoke test.

- Go **1.26.8**, declared in [`go.mod`](../../go.mod) and the release workflow.
- Node.js **24** matches the release workflow; npm and the checked-in `web/package-lock.json` provide frontend dependencies. The locally inspected runtime is Node 26.10.0, not a test of Node 24 compatibility.
- Native compiler and platform GUI libraries for the server's systray dependency. Linux CI installs `gcc`, `libgtk-3-dev`, and `libayatana-appindicator3-dev` on Ubuntu. These are CI package names, not Fedora/macOS instructions. `--headless` skips the GUI at runtime but does not remove the compiled systray dependency.
- Internet access for initial dependency retrieval and provider-backed creation/downloads. Java is managed through `internal/jvm`; do not assume one system Java version fits every Minecraft version.

## Run backend and frontend separately

Run from the repository root, using two terminals. Set `NAVISERVER_DEV=true` for **both** processes as the owner-requested development convention.

```bash
# Terminal 1: backend
NAVISERVER_DEV=true NAVISERVER_HOST=127.0.0.1 go run ./cmd/server --headless

# Terminal 2: frontend
cd web
npm ci --ignore-scripts
NAVISERVER_DEV=true npm run dev
```

Open the frontend URL printed by Vite (configured port `5173`; use its actual printed URL). Check the backend independently:

```bash
curl --fail http://127.0.0.1:23009/health
# Expected: {"status":"ok"}
```

Use `localhost` consistently for frontend/backend browser access rather than switching between `localhost` and `127.0.0.1` during login. Initialize the first admin through the web setup screen. Development data is separate from production by default: on Linux, `$XDG_CONFIG_HOME/naviserver-dev` or `~/.config/naviserver-dev`. Config/database/server/backup/runtime files and generated secrets are created there. Never point development config paths at production data.

### What the environment variable actually does

Backend [`config.IsDev`](../../internal/config/config.go) accepts `true` or `1`, selects the `naviserver-dev` config directory and default API port **23009**, rather than production **23008**. A previously saved API port or `NAVISERVER_PORT` override takes precedence over the default; inspect the startup address if the browser cannot connect.

Frontend code does **not** read `NAVISERVER_DEV`. [`apiUrls.ts`](../../web/src/utils/apiUrls.ts) uses Vite's `import.meta.env.DEV` to choose port 23009. Passing the variable to both processes is the project convention, not evidence of a frontend feature flag. Vite only exposes `VITE_`-prefixed custom variables by default; see [Vite environment documentation](https://vite.dev/guide/env-and-mode).

API resolution order: nonblank `VITE_API_BASE_URL`, then `VITE_API_PORT` with the browser protocol/hostname, then dev port 23009, otherwise the page origin. WebSocket base uses `VITE_WS_BASE_URL` if supplied; otherwise it converts HTTP/HTTPS to WS/WSS. No Vite API proxy is configured. If overriding ports/hosts, configure matching frontend values:

```bash
# Example frontend override; use only when backend really listens there
NAVISERVER_DEV=true VITE_API_BASE_URL=http://localhost:24000 npm run dev
```

`VITE_*` values are public client configuration: never put secret keys, CLI tokens or provider credentials there. Restart Vite after changing its environment. Backend CORS allows localhost origins and configured origins; CORS is not authentication or a network firewall.

## Tests, builds and debugging

```bash
# Root
NAVISERVER_DEV=true go test -count=1 ./...
go vet ./...
go build -o /tmp/naviserver-cli ./cmd/cli

# web/
NAVISERVER_DEV=true npm run test
npm run lint
npm run check-format
npm run build
```

Tests still create/use their own fixtures as implemented; `NAVISERVER_DEV` is not a universal test sandbox. Use isolated configuration for manual tests. The Go daemon logs startup, background operations and process errors to its terminal; inspect browser Console/Network for HTTP status, cookies and WebSocket failures. The frontend reports an ongoing backend network outage after an 8-second grace period; its ordinary HTTP timeout is 5 seconds, with larger operation-specific timeouts in [`api.ts`](../../web/src/services/api.ts).

For packaged local output, [`build.sh`](../../build.sh) builds web/server/CLI and places the frontend in `dist/web_dist`. **It deletes `dist` first** and may create native packages; do not use it to diagnose one frontend change. A standalone `npm run build` writes `web/dist`, whereas the daemon locates `web_dist` next to the executable, in macOS Resources, or relative to the working directory. `go run` does not automatically serve `web/dist`; use Vite during development.

## Where to change behavior

| Concern | Starting points | Verification boundary |
|---|---|---|
| Backend composition/config | `cmd/server/main.go`, `internal/config` | Config tests and isolated startup |
| HTTP/auth/permissions | `internal/api/router.go`, middleware, `internal/api/handlers` | Router/security/handler tests; check every caller |
| Server creation/settings | `internal/server`, `internal/loader`, `internal/jvm` | Manager/loader tests and provider smoke |
| Process lifecycle/console | `internal/runner`, `internal/ws` | Supervisor/shutdown tests and actual child process |
| Persistence | `internal/storage`, `internal/domain` | Schema compatibility and full-data restore |
| Backups/uploads/add-ons | Matching capability package and handler | Unit tests plus actual archive/transfer/provider |
| Browser UI | `web/src/pages`, components, contexts, hooks, services | Component tests, build and browser flow |
| CLI and legacy TUI | `cmd/cli`, `internal/cli`, `pkg/sdk` | SDK/API behavior; TUI retirement is intended, not implemented |

To add a loader, implement [`ServerLoader`](../../internal/loader/contract.go), update the factories and available-loader list in [`factory.go`](../../internal/loader/factory.go), and review runner selection in [`strategy/factory.go`](../../internal/runner/strategy/factory.go), Java resolution and UI loader handling. Preserve metadata/progress contracts and test invalid/provider-failure paths. For a new add-on provider, inspect existing Modrinth/CurseForge adapters and the dispatch in `internal/addons`; update API/UI source choices and preview/install/dependency behavior together. Neither extension is a plugin deployment system.

## Troubleshooting checklist

- **Build fails before startup:** verify Go/Node versions, native compiler and systray libraries; a headless flag cannot solve missing compile-time libraries.
- **Backend unreachable:** read actual startup port, check saved development config and environment overrides, then `/health`; inspect frontend API base separately.
- **Login missing after success:** check hostname consistency, cookie transport, credentials and origin settings; never bypass auth to fix development wiring.
- **UI 404 from daemon:** confirm `web_dist` packaging; use the Vite URL for the two-terminal workflow.
- **Creation/download fails:** distinguish provider connectivity/credentials, managed Java, disk/path permissions and job progress errors.
- **Progress reconnect:** REST state is authoritative; WebSocket history is bounded in memory, not a durable job log.

## Maintenance convention

When changing API fields, roles, progress framing or schema, update [technical contracts](../02-architecture/contracts.md), impacted RF/RN/TC records, and relevant tests. Preserve current behavior versus accepted targets. The web UI and CLI remain documented; the owner intends to discontinue the TUI, with no removal version/date yet assigned.
