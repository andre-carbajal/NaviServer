# Quality and operations

## Canonical quality requirements

The non-functional requirement register is maintained in [functional requirements](../01-functional/index.md#non-functional-requirements-and-quality-targets). Current security and persistence behavior is observed in source; numeric availability, capacity, and recovery targets remain unspecified. The owner confirmed trusted LAN/VPN access; explicit EULA consent is an accepted target awaiting implementation. Do not interpret an existing test as proof of a broader quality goal than it covers.

## Verification strategy

| Level | Scope | Requirements | Command or evidence |
|---|---|---|---|
| Go tests | Backend packages: API, auth/security, managers, process supervision, storage/config, uploads. | RF-001–RF-008, RNF-001–RNF-003 (partial). | `go test -count=1 ./...` from repository root. Defined in [`release.yml`](../../.github/workflows/release.yml); not run in this documentation pass. |
| Go static analysis | Go source vet checks. | Maintainability/build hygiene; partial evidence only. | `go vet ./...`; defined in release workflow, not run here. |
| Web tests | Vitest unit/component coverage for API helpers, contexts, hooks, upload UI, formatting, and utilities. | RF-003–RF-010 (partial). | `cd web && npm ci --ignore-scripts && npm run test`; not run here. |
| Web build | TypeScript project build and Vite bundle. | RF-010, RNF-003 (build evidence only). | `cd web && npm ci --ignore-scripts && npm run build`; not run here. |
| Release builds | Native packages and Docker images. | RNF-003. | `.github/workflows/release.yml`; triggered by pushed `v*` tags. The workflow tests and vets before packaging. No live workflow result was checked. |
| Manual end-to-end | Real daemon, provider connectivity, UI/WebSocket, data persistence, restore, and public access policy. | RF-002–RF-010, RNF-001–RNF-005. | Cases TC-002, TC-003, TC-005, TC-007–TC-013 below; none run in this pass. |

The inspected repository has one workflow file, `.github/workflows/release.yml`; it runs validation for tagged releases. The inventory did not show a separate pull-request test workflow. Automated suites are not a substitute for live deployment, browser, network, or recovery acceptance.

## Test cases and verification

These cases map requirement IDs to evidence and clearly separate existing automation from checks still requiring a real environment.

| ID | Requirements | Preparation and action | Expected result | Method / current evidence |
|---|---|---|---|---|
| TC-001 — Bootstrap and authentication | RF-001, RF-007, RNF-001 | Start with an empty test database; attempt setup, sign in, then attempt setup again and invalid login. | First setup creates admin; subsequent setup and invalid login are rejected. | Automated: `internal/api/handlers/auth_test.go`; suite not run here. |
| TC-002 — Server provisioning | RF-002, RF-008 | Start an isolated installation with network access; create a Vanilla server, then exercise supported loaders as feasible; inspect progress, instance files, assigned port, and persisted record. | Success is reflected in UI/API and local state; errors are visible and failed partial output is cleaned where implemented. | Manual integration; no full end-to-end creation test was found in the inspected test inventory. Not run. |
| TC-003 — Lifecycle and shutdown | RF-003, RN-004 | Exercise start/stop/restart/kill on a test instance; attempt deletion while running and after stopping; terminate daemon with an active server. | Lifecycle state/output is coherent; active deletion conflicts; daemon asks servers to stop and force-exits only after its grace path. | Automated: `internal/runner/supervisor_test.go`, `supervisor_shutdown_test.go`, `internal/api/handlers/server_delete_test.go`; plus manual runtime smoke, not run here. |
| TC-004 — File operations and uploads | RF-004, RF-007 | Use authorized and unauthorized users; edit/download a file; upload a file/folder and cancel/retry a chunked transfer. | File access stays within server boundary; permission failure is rejected; transfer progress and terminal status are correct. | Automated: `internal/upload/manager_test.go`, `internal/api/handlers/uploads_test.go`, frontend upload context/component tests; not run here. |
| TC-005 — Backup and restore | RF-005, RNF-002, RNF-004 | With automatic backups disabled, verify no scheduled job runs; enable/configure them and verify scheduling/retention. Create/list/download a backup and restore into both an existing and a new instance; separately test full application-data recovery. | Minecraft backup restores the expected server state; app-level recovery restores DB/config/secrets/data consistently. | Automated scheduled-backup tests in `internal/backup/manager_auto_test.go`; restore and full-data recovery remain manual and were not run. |
| TC-006 — Add-on management | RF-006 | With provider connectivity and any required CurseForge key, search, preview required dependencies, install, update/disable/remove an add-on. | Selected source and dependency metadata are retained; errors from missing credentials/provider are shown. | Automated: `internal/addons/manager_test.go`; provider-backed smoke not run here. |
| TC-007 — Authorization and permission scope | RF-003, RF-004, RF-007, RNF-001 | Test admin, scoped user, no token, invalid token, and CLI token against representative protected/admin/server-specific endpoints. | Unauthenticated/unauthorized requests are rejected; only scoped server capabilities are reachable; CLI token has its current admin mapping. | Automated: `internal/api/security_test.go`, `internal/api/router_test.go`, handler tests; public-link policy separately tested manually in TC-009. Not run here. |
| TC-008 — Settings and managed Java | RF-008 | Read/update global and per-server settings; try invalid ports, RAM/Java values, and supported version update paths. | Valid choices persist; invalid values return errors; selected Java behavior matches the displayed requirement/warning. | Automated: `internal/api/handlers/server_settings_test.go`, `server_version_update_test.go`, `internal/server/settings_version_test.go`, `internal/jvm/version_test.go`; not run here. |
| TC-009 — Public-link access | RF-009, RN-003, RNF-001 | Check disabled-by-default state, explicitly enable a link, inspect from a signed-out browser on LAN/VPN, exercise status/player refresh and start/stop; verify admin and scoped console users can activate/revoke, other users cannot; revoke and repeat. | Initially no usable public link exists; explicit activation permits signed-out status/start/stop access; revocation/invalid tokens deny access. These are owner-approved criteria, not runtime-verified results. | Manual security-sensitive acceptance; no public-link handler test file was found in the inspected test inventory. Not run. |
| TC-010 — Web, CLI, and TUI parity | RF-010 | Against one test daemon, perform representative list/create/lifecycle/backup/settings tasks via browser, CLI, and TUI. | Each client reaches the same daemon-owned state using HTTP/WebSocket APIs. | Manual; CLI command/TUI references are in `wiki/cli.md` and `wiki/tui.md`. Not run. |
| TC-011 — Persistence and recovery | RNF-002, RNF-004 | Back up the complete configured data root; restore into an isolated host/container; inspect users, servers, secrets, backup metadata, and files. | State is recovered without a data/secret mismatch; measured recovery is compared to owner-approved RPO/RTO. | Manual operational test; no quantified recovery target and no result available. |
| TC-012 — Distribution compatibility | RNF-003 | Build/test native targets and Docker `linux/amd64`/`linux/arm64`; launch the produced artifacts on supported environments. | Each published target installs and starts using documented paths. | Release pipeline matrix in `.github/workflows/release.yml`; current release artifacts were not inspected or executed. |
| TC-013 — Remote transport and proxy trust | RNF-001, RNF-005 | Deploy on the intended LAN/VPN, verify only trusted network clients can reach the daemon and authentication still applies. If a TLS proxy is used, validate cookie/origin/proxy-trust settings. | The installation is not directly Internet-exposed; trusted network access does not bypass authentication. If HTTPS/proxy is configured, cookie behavior follows it and untrusted headers do not affect security. | Manual deployment/security review against owner-confirmed LAN/VPN scope; not run. |
| TC-014 — Explicit EULA gate (target) | RF-011, RN-005 | Create a new server with EULA false; request start, cancel, retry and accept; repeat with true. Exercise every backend start caller including public links and CLI/TUI. | False blocks launch pending explicit consent; cancellation does not accept; acceptance records true before launch; true does not prompt again; public-link/CLI report acceptance required and do not start; TUI offers consent to admin/console users while supported; other callers cannot bypass validation. | Planned target verification only; code/UI changes and tests have NOT been implemented in this documentation task. |

## Running checks from a checkout

Commands below are taken from the project manifests and release workflow; they were not executed during this documentation pass.

```bash
# Backend tests and vet (repository root)
go test -count=1 ./...
go vet ./...

# Frontend tests and build
cd web
npm ci --ignore-scripts
npm run test
npm run build
```

The Go server entry point supports headless mode (`go run ./cmd/server --headless`); native builds require the platform dependencies configured by the build scripts/workflow. `web/package.json` provides `npm run dev` for the frontend, but the [development guide](development.md) now documents configuration/API wiring from source; live two-process startup remains unverified. Docker Compose usage and production/user installation remain documented in [`wiki/installation.md`](../../wiki/installation.md).

## Configuration and deployment facts

- Default API host/port: `0.0.0.0:23008`; development mode uses port `23009`. See [`config.go`](../../internal/config/config.go) and the [configuration guide](../../wiki/configuration.md).
- The application stores config/database/server/backup/runtime paths under the OS config directory by default. In Docker, the image sets `HOME` and `XDG_CONFIG_HOME` to `/data`, and Compose mounts a named persistent volume there.
- Docker exposes the web/API/WebSocket port separately from Minecraft server ports. The default server port range is `25565-25600`; operators must publish the ports they need.
- Relevant secret/config variables include `NAVISERVER_SECRET_KEY`, `NAVISERVER_CLI_TOKEN`, `NAVISERVER_ALLOWED_ORIGINS`, `NAVISERVER_TRUST_PROXY`, and `CURSEFORGE_API_KEY`. Keep secrets out of source control and treat the CLI token as an admin credential.
- Owner-approved deployment scope is a trusted LAN or VPN, not direct Internet exposure. The daemon itself uses HTTP. If the operator uses a TLS/reverse proxy, `NAVISERVER_TRUST_PROXY` should only be enabled when a trusted proxy overwrites/removes client-provided `X-Forwarded-Proto`.
- Native install locations and platform details are maintained in the [installation guide](../../wiki/installation.md). No live host, production URL, or running deployment was inspected.

## Backup and recovery boundary

NaviServer's in-product backups protect Minecraft server instances. They are not a substitute for backing up the complete NaviServer data root, which contains the SQLite database, configuration, generated secrets/token files, server instances, backups, and managed runtimes. The existing [Docker installation guide](../../wiki/installation.md) shows a data-volume archive procedure. No automated off-host backup, restore verification schedule, RPO, or RTO target was found. Protect and test the whole data root before upgrades or migrations.

## Current evidence status

- **Inspected:** source structure, key Go managers/handlers, React routes, package manifests, Docker/Compose, README/wiki, and the release workflow at `d65df2d`.
- **Tests/builds:** no full suite/build; targeted config and frontend API-URL tests passed in the follow-up (see below).
- **Unverified:** live browser behavior, provider availability, container/native startup, deployed TLS/proxy behavior, restore from an actual backup, release artifact compatibility, and performance/capacity.
- **Owner-confirmed:** opt-in public start/stop links, console access including start/stop, optional automatic backups, restore to existing/new instances, and trusted LAN/VPN deployment.
- **Accepted target, not implemented:** first-start EULA modal/gate for a newly created server with `eula=false` (RF-011).
- **Owner-confirmed follow-up:** admin or scoped console users manage links and accept EULA; public-link/CLI refuse start pending consent; TUI offers acceptance while it exists. TUI discontinuation is intended, not implemented.
- **Still unspecified:** numeric recovery/capacity/availability targets, missing/malformed EULA handling, TUI retirement timing.


## Documentation-pass checks

Follow-up on 2026-10-04, separate from the earlier static baseline:

- `NAVISERVER_DEV=true go test ./internal/config` — passed. This exercises config tests, not a live daemon or all backend features.
- `NAVISERVER_DEV=true npm run test -- src/utils/apiUrls.test.ts` in `web` — passed, 7 tests. This exercises URL selection, not browser login/WebSocket behavior.
- Documentation link and format checks performed after edits.
- No dependency installation, full build, live server creation/restore, or EULA implementation was performed.

[Development guide](development.md) and [technical contracts](../02-architecture/contracts.md) provide executable instructions and current wire reference. The extraction records two source-level verification gaps: WebSocket per-server/job permission enforcement is absent in the handler; frames can batch newline-separated payloads while browser progress consumers parse a single JSON object. Neither was fixed or runtime-demonstrated in this documentation task.
