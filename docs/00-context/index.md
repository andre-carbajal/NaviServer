# Context

## Purpose and audience

**DOCUMENTED / owner-confirmed:** NaviServer is a control panel and daemon for creating, configuring, operating, and monitoring Minecraft server instances. Its stated target is personal use and small self-hosted environments. The requested audience for this technical baseline is maintainers and developers.

The [README](../../README.md) describes the end-user value and distribution choices. This page records the product context relevant to technical changes; it does not establish an uptime, scale, or public-hosting commitment.

## Users and callers

| Actor | Need or responsibility | Evidence and status |
|---|---|---|
| Administrator | Initialize the first account, manage server instances and settings, and administer users. | First account becomes `admin`; routes and permissions observed in [`auth.go`](../../internal/api/handlers/auth.go), [`users.go`](../../internal/api/handlers/users.go), and [`router.go`](../../internal/api/router.go). |
| Scoped user/operator | Access only assigned server capabilities through the web UI or CLI/TUI. | Per-server permission records and checks observed in [`base.go`](../../internal/api/handlers/base.go) and [`server.go`](../../internal/api/handlers/server.go). Owner-confirmed: console permission also grants start/stop control. |
| Public-link visitor | View a server summary and, with the current `control` link, request start or stop without signing in. | Observed in [`links.go`](../../internal/api/handlers/links.go) and [`PublicServer.tsx`](../../web/src/pages/PublicServer.tsx); Owner-confirmed policy: links are disabled by default; after a server administrator enables a link, possession permits status lookup and start/stop control. |
| Host operator | Install/run the daemon, protect its data directory and secrets, expose the web endpoint, and maintain backups. | Operational responsibilities documented in the [installation](../../wiki/installation.md) and [configuration](../../wiki/configuration.md) guides. |
| External providers | Supply loader/JDK metadata or artifacts, add-on data, and update information. | Integration calls observed in source; summarized in [architecture](../02-architecture/index.md). |

## Product scope

**Current documented and observed capabilities:**

- Create and manage multiple Minecraft instances using Vanilla, Paper, Fabric, Forge, or NeoForge.
- Manage Java runtimes, server settings, lifecycle, status, resource/player metrics, and live console traffic.
- Browse, edit, upload, and download server files; manage add-ons and backups, including scheduled backups.
- Manage local users and per-server permissions; optionally enable public server links, disabled by default, for status and start/stop control.
- Use a browser interface served by the daemon, or administer it with `naviserver-cli` and its TUI.
- Install as a native application/service or run in Docker; migrate legacy NaviServer 1.x data.

**Boundary:** NaviServer orchestrates server processes and local data; the Minecraft game server itself and upstream loader/add-on/JDK providers are not owned by NaviServer. No accepted requirement for hosted SaaS, multi-tenant service operation, clustering, or high availability was found. These are **not recorded as out of scope**; they remain unaccepted options unless the owner decides otherwise.

## Constraints and assumptions

- **OBSERVED:** the daemon starts and supervises local Minecraft child processes; it stores metadata in SQLite and server/runtime/backup data in local directories.
- **OBSERVED:** the web build is served by the daemon; the CLI/TUI calls the same HTTP/WebSocket API.
- **DOCUMENTED / owner-confirmed:** the product is for personal and small self-hosted use.
- **INFERRED:** the current deployment shape is one daemon coordinating local state on a host. No multi-node coordination or shared-storage protocol was found.
- **TBD:** supported concurrency, maximum instance/player counts, performance targets, availability expectations, and recovery objectives.
- **Owner-confirmed:** access is currently designed for trusted networks, such as a LAN or VPN. Direct Internet exposure is not the supported baseline; the term public link means bearer-token access, not a requirement to publish the installation on the Internet.

## Owner-approved behavior and implementation gaps

- Public links are opt-in and disabled by default. Once enabled by the server administrator, anyone who can reach NaviServer and holds the link can see whether the server is on/off and start/stop it.
- Console permission intentionally includes starting and stopping the server. Admins and users with console permission on the server may activate/revoke its public link.
- Automatic backups are configurable and run when enabled. A backup can be restored into a previously created server or into a new server. This describes functionality, not a numeric RPO/RTO guarantee.
- **Accepted target, not implemented:** after creation, attempting to start a new server with `eula=false` must present an explicit EULA acceptance modal. The process must not start until acceptance; when EULA is already accepted, no modal is needed. Current creation still writes `eula=true` automatically.

## Remaining clarifications

1. No numeric availability, capacity, retention, RPO, or RTO objectives were supplied. Keep them unspecified rather than interpreting the backup feature description as a recovery guarantee.
2. EULA missing/malformed-file behavior is not yet specified.
3. No version/date or compatibility policy is assigned to TUI retirement; it remains available in the current code.

## Additional owner decisions (2026-10-04)

- EULA may be accepted by an admin or a user with console permission on that server. Browser start shows explicit consent when false; public-link and noninteractive CLI starts must explain acceptance is required and stop. While TUI remains available, it should offer explicit acceptance to an eligible user. This remains an unimplemented target.
- Development convention: set `NAVISERVER_DEV=true` for both backend and frontend; actual frontend dev-mode detection is explained in the [development guide](../04-quality-operations/development.md).
- The owner intends to discontinue TUI because the web UI covers its needs. This is a future product direction, not evidence that TUI has been removed, and does not discontinue CLI.
