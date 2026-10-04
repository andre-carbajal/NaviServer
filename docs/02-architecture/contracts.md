# Technical contracts — current implementation

Extracted from router, handlers, domain/storage models, client calls and WebSocket code on 2026-10-04. This is a source-based reference, not an OpenAPI specification or proof of runtime compliance. The EULA section is explicitly a target contract; no endpoint is invented for it.

## Transport and authentication

- Routes have no `/api` prefix or version prefix. Default production HTTP port is 23008; development default is 23009. See the [development guide](../04-quality-operations/development.md).
- Browser authentication: cookie `token`, HttpOnly, SameSite=Lax, path `/`, with a 7-day login expiry. Secure follows TLS/trusted forwarded HTTPS; direct HTTP is supported for the intended trusted LAN/VPN scope. Login returns `{ "token": "", "user": ... }`; it does not return the cookie JWT as a readable token.
- Middleware checks CLI headers first (`X-NaviServer-Client: CLI` plus `X-NaviServer-CLI-Token`), then cookie, `Authorization: Bearer ...`, then query `token`. Valid CLI credentials map to admin. Never put bearer tokens in examples/logs or assume query credentials are safe from URL logging.
- In the route table, **Authenticated** means middleware protection, not guaranteed server-level authorization. Admin bypasses `checkPermission`; non-admin needs the relevant server permission. Console includes power on permission save and is approved for public-link management.
- JSON is common, not universal: file bodies are plain text; downloads/icons are binary; old uploads are multipart and chunk uploads are raw bytes. Many failures use `http.Error` plain text. Some success handlers do not explicitly set JSON Content-Type. Do not assume a uniform envelope or always parse JSON.
- Common statuses: 400 malformed input, 401 invalid/missing login, 403 insufficient permission, 404 missing resource, 409 lifecycle/upload conflicts, 500 manager/provider/storage failures. These are not uniformly mapped; inspect the referenced handler for exact branches.

## Complete registered method/path inventory

Generated from [`router.go`](../../internal/api/router.go). SPA fallback `/` is not a JSON API. `GET /servers/{id}` serves the SPA for `Accept: text/html`; API clients should request JSON. `/health` only confirms the HTTP handler is responsive, not Minecraft/provider/recovery health.

| Method/path | Route authentication | Handler/source |
|---|---|---|
| `GET /health` | Public | Inline health response |
| `POST /auth/login` | Public | [`HandleLogin`](../../internal/api/handlers/auth.go) |
| `POST /auth/logout` | Public | [`HandleLogout`](../../internal/api/handlers/auth.go) |
| `POST /auth/setup` | Public | [`HandleSetup`](../../internal/api/handlers/auth.go) |
| `GET /auth/setup` | Public | [`HandleCheckSetup`](../../internal/api/handlers/auth.go) |
| `POST /public-links/{token}/access` | Public | [`HandleAccessPublicLink`](../../internal/api/handlers/links.go) |
| `GET /public-links/{token}` | Public | [`HandleGetPublicServerInfo`](../../internal/api/handlers/links.go) |
| `DELETE /public-links/{token}` | Authenticated | [`HandleDeletePublicLink`](../../internal/api/handlers/links.go) |
| `GET /auth/me` | Authenticated | [`HandleMe`](../../internal/api/handlers/auth.go) |
| `POST /uploads` | Authenticated | [`HandleCreate`](../../internal/api/handlers/uploads.go) |
| `GET /uploads/{id}` | Authenticated | [`HandleGet`](../../internal/api/handlers/uploads.go) |
| `PUT /uploads/{id}/chunk` | Authenticated | [`HandleChunk`](../../internal/api/handlers/uploads.go) |
| `POST /uploads/{id}/complete` | Authenticated | [`HandleComplete`](../../internal/api/handlers/uploads.go) |
| `DELETE /uploads/{id}` | Authenticated | [`HandleDelete`](../../internal/api/handlers/uploads.go) |
| `GET /loaders` | Authenticated | [`HandleGetLoaders`](../../internal/api/handlers/loaders.go) |
| `GET /loaders/{name}/versions` | Authenticated | [`HandleGetLoaderVersions`](../../internal/api/handlers/loaders.go) |
| `GET /loaders/{name}/metadata` | Authenticated | [`HandleGetLoaderMetadata`](../../internal/api/handlers/loaders.go) |
| `GET /servers` | Authenticated | [`HandleListServers`](../../internal/api/handlers/server.go) |
| `GET /servers-stats` | Authenticated | [`HandleGetAllServerStats`](../../internal/api/handlers/server.go) |
| `POST /servers` | Admin | [`HandleCreateServer`](../../internal/api/handlers/server.go) |
| `GET /servers/{id}` | Authenticated for JSON; public SPA shell | Router wrapper → authenticated HandleGetServer for JSON |
| `GET /servers/{id}/stats` | Authenticated | [`HandleGetServerStats`](../../internal/api/handlers/server.go) |
| `GET /servers/{id}/settings` | Admin | [`HandleGetServerSettings`](../../internal/api/handlers/server.go) |
| `PUT /servers/{id}/settings` | Admin | [`HandleUpdateServerSettings`](../../internal/api/handlers/server.go) |
| `PUT /servers/{id}/auto-backup` | Admin | [`HandleUpdateServerAutoBackup`](../../internal/api/handlers/server.go) |
| `GET /servers/{id}/version-options` | Admin | [`HandleGetVersionOptions`](../../internal/api/handlers/server.go) |
| `POST /servers/{id}/version-update` | Admin | [`HandleUpdateServerVersion`](../../internal/api/handlers/server.go) |
| `GET /servers/{id}/icon` | Public | [`HandleGetServerIcon`](../../internal/api/handlers/server.go) |
| `POST /servers/{id}/icon` | Admin | [`HandleUploadServerIcon`](../../internal/api/handlers/server.go) |
| `PUT /servers/{id}` | Admin | [`HandleUpdateServer`](../../internal/api/handlers/server.go) |
| `DELETE /servers/{id}` | Admin | [`HandleDeleteServer`](../../internal/api/handlers/server.go) |
| `GET /servers/{id}/files` | Authenticated | [`HandleListFiles`](../../internal/api/handlers/files.go) |
| `GET /servers/{id}/files/content` | Authenticated | [`HandleGetFileContent`](../../internal/api/handlers/files.go) |
| `PUT /servers/{id}/files/content` | Authenticated | [`HandleSaveFileContent`](../../internal/api/handlers/files.go) |
| `POST /servers/{id}/files/directory` | Authenticated | [`HandleCreateDirectory`](../../internal/api/handlers/files.go) |
| `DELETE /servers/{id}/files` | Authenticated | [`HandleDeleteFile`](../../internal/api/handlers/files.go) |
| `GET /servers/{id}/files/download` | Authenticated | [`HandleDownloadFile`](../../internal/api/handlers/files.go) |
| `POST /servers/{id}/files/upload` | Authenticated | [`HandleUploadFile`](../../internal/api/handlers/files.go) |
| `GET /servers/{id}/addons` | Authenticated | [`HandleListAddons`](../../internal/api/handlers/addons.go) |
| `POST /servers/{id}/addons/sync` | Authenticated | [`HandleSyncAddons`](../../internal/api/handlers/addons.go) |
| `POST /servers/{id}/addons/search` | Authenticated | [`HandleSearchAddons`](../../internal/api/handlers/addons.go) |
| `POST /servers/{id}/addons/versions` | Authenticated | [`HandleAddonVersions`](../../internal/api/handlers/addons.go) |
| `POST /servers/{id}/addons/install-preview` | Authenticated | [`HandleInstallPreview`](../../internal/api/handlers/addons.go) |
| `POST /servers/{id}/addons/install` | Authenticated | [`HandleInstallAddon`](../../internal/api/handlers/addons.go) |
| `POST /servers/{id}/addons/update-all` | Authenticated | [`HandleUpdateAllAddons`](../../internal/api/handlers/addons.go) |
| `POST /servers/{id}/addons/{addonId}/update` | Authenticated | [`HandleUpdateAddon`](../../internal/api/handlers/addons.go) |
| `POST /servers/{id}/addons/{addonId}/disabled` | Authenticated | [`HandleSetAddonDisabled`](../../internal/api/handlers/addons.go) |
| `DELETE /servers/{id}/addons/{addonId}` | Authenticated | [`HandleDeleteAddon`](../../internal/api/handlers/addons.go) |
| `POST /servers/{id}/start` | Authenticated | [`HandleStartServer`](../../internal/api/handlers/server.go) |
| `POST /servers/{id}/stop` | Authenticated | [`HandleStopServer`](../../internal/api/handlers/server.go) |
| `POST /servers/{id}/restart` | Authenticated | [`HandleRestartServer`](../../internal/api/handlers/server.go) |
| `POST /servers/{id}/kill` | Authenticated | [`HandleKillServer`](../../internal/api/handlers/server.go) |
| `POST /servers/{id}/backup` | Authenticated | [`HandleBackupServer`](../../internal/api/handlers/server.go) |
| `GET /servers/{id}/backups` | Authenticated | [`HandleListBackupsByServer`](../../internal/api/handlers/backup.go) |
| `GET /backups` | Authenticated | [`HandleListAllBackups`](../../internal/api/handlers/backup.go) |
| `POST /backups/upload` | Authenticated | [`HandleUploadBackup`](../../internal/api/handlers/backup.go) |
| `PUT /backups/{name}` | Admin | [`HandleUpdateBackup`](../../internal/api/handlers/backup.go) |
| `DELETE /backups/{name}` | Authenticated | [`HandleDeleteBackup`](../../internal/api/handlers/backup.go) |
| `GET /backups/{name}/download` | Authenticated | [`HandleDownloadBackup`](../../internal/api/handlers/backup.go) |
| `DELETE /backups/progress/{id}` | Authenticated | [`HandleCancelBackup`](../../internal/api/handlers/backup.go) |
| `POST /backups/{name}/restore` | Authenticated | [`HandleRestoreBackup`](../../internal/api/handlers/backup.go) |
| `GET /settings/port-range` | Admin | [`HandleGetPortRange`](../../internal/api/handlers/settings.go) |
| `PUT /settings/port-range` | Admin | [`HandleSetPortRange`](../../internal/api/handlers/settings.go) |
| `GET /settings/log-buffer-size` | Admin | [`HandleGetLogBufferSize`](../../internal/api/handlers/settings.go) |
| `PUT /settings/log-buffer-size` | Admin | [`HandleSetLogBufferSize`](../../internal/api/handlers/settings.go) |
| `GET /settings/public-ip` | Authenticated | [`HandleGetPublicIP`](../../internal/api/handlers/settings.go) |
| `PUT /settings/public-ip` | Admin | [`HandleSetPublicIP`](../../internal/api/handlers/settings.go) |
| `GET /settings/curseforge-key` | Admin | [`HandleGetCurseForgeKeyStatus`](../../internal/api/handlers/settings.go) |
| `PUT /settings/curseforge-key` | Admin | [`HandleSetCurseForgeKey`](../../internal/api/handlers/settings.go) |
| `DELETE /settings/curseforge-key` | Admin | [`HandleDeleteCurseForgeKey`](../../internal/api/handlers/settings.go) |
| `GET /system/interfaces` | Admin | [`HandleGetNetworkInterfaces`](../../internal/api/handlers/system.go) |
| `GET /system/resources` | Admin | [`HandleGetSystemResources`](../../internal/api/handlers/system.go) |
| `POST /system/restart` | Admin | [`HandleRestartDaemon`](../../internal/api/handlers/system.go) |
| `GET /updates` | Admin | [`HandleCheckUpdates`](../../internal/api/handlers/system.go) |
| `GET /version` | Authenticated | [`HandleGetVersion`](../../internal/api/handlers/system.go) |
| `GET /ws/servers/{id}/console` | Authenticated | [`HandleWebSocket`](../../internal/api/handlers/ws.go) |
| `GET /ws/progress/{id}` | Authenticated | [`HandleWebSocket`](../../internal/api/handlers/ws.go) |
| `GET /users` | Admin | [`HandleListUsers`](../../internal/api/handlers/users.go) |
| `POST /users` | Admin | [`HandleCreateUser`](../../internal/api/handlers/users.go) |
| `PUT /users/permissions` | Admin | [`HandleUpdatePermissions`](../../internal/api/handlers/users.go) |
| `GET /users/{id}/permissions` | Admin | [`HandleGetPermissions`](../../internal/api/handlers/users.go) |
| `DELETE /users/{id}` | Admin | [`HandleDeleteUser`](../../internal/api/handlers/users.go) |
| `PUT /users/{id}/password` | Authenticated | [`HandleUpdatePassword`](../../internal/api/handlers/users.go) |
| `POST /public-links` | Authenticated | [`HandleCreatePublicLink`](../../internal/api/handlers/links.go) |
| `GET /servers/{id}/public-link` | Authenticated | [`HandleGetPublicLink`](../../internal/api/handlers/links.go) |

The route table covers 85 registered method/path pairs. The SPA fallback is separate. CORS middleware responds to OPTIONS with 200, advertises GET/POST/PUT/OPTIONS/DELETE, credentials and Content-Type/Content-Range/Authorization/X-Requested-With headers; it does not itself grant authorization.

## Inputs declared by handlers

This extraction lists JSON-tagged request fields, query keys and multipart form keys. Go pointer fields distinguish omitted values where the handler uses them; plain fields do not imply validation/requiredness. Named domain/add-on/settings payloads and nested validation remain defined in linked source. No global pagination, idempotency key or error schema is asserted.

| Handler | Declared body/query/form input | Source |
|---|---|---|
| `AddonsHandler.HandleAddonVersions` | body: `addons.VersionsRequest` | [addons.go](../../internal/api/handlers/addons.go) |
| `AddonsHandler.HandleInstallAddon` | body: `addons.InstallRequest` | [addons.go](../../internal/api/handlers/addons.go) |
| `AddonsHandler.HandleInstallPreview` | body: `addons.InstallPreviewRequest` | [addons.go](../../internal/api/handlers/addons.go) |
| `AddonsHandler.HandleSearchAddons` | body: `addons.SearchRequest` | [addons.go](../../internal/api/handlers/addons.go) |
| `AuthHandler.HandleLogin` | body: `LoginRequest`; `username`: string; `password`: string | [auth.go](../../internal/api/handlers/auth.go) |
| `AuthHandler.HandleSetup` | body: `RegisterRequest`; `username`: string; `password`: string | [auth.go](../../internal/api/handlers/auth.go) |
| `BackupHandler.HandleRestoreBackup` | `targetServerId`: string; `newServerName`: string; `newServerRam`: int; `newServerLoader`: string; `newServerVersion`: string | [backup.go](../../internal/api/handlers/backup.go) |
| `BackupHandler.HandleUpdateBackup` | `serverId`: string | [backup.go](../../internal/api/handlers/backup.go) |
| `BackupHandler.HandleUploadBackup` | form `backup`; form `serverId` | [backup.go](../../internal/api/handlers/backup.go) |
| `FilesHandler.HandleCreateDirectory` | `path`: string | [files.go](../../internal/api/handlers/files.go) |
| `FilesHandler.HandleDeleteFile` | query `path` | [files.go](../../internal/api/handlers/files.go) |
| `FilesHandler.HandleDownloadFile` | query `path` | [files.go](../../internal/api/handlers/files.go) |
| `FilesHandler.HandleGetFileContent` | query `path` | [files.go](../../internal/api/handlers/files.go) |
| `FilesHandler.HandleListFiles` | query `path` | [files.go](../../internal/api/handlers/files.go) |
| `FilesHandler.HandleSaveFileContent` | query `path` | [files.go](../../internal/api/handlers/files.go) |
| `FilesHandler.HandleUploadFile` | query `path`; query `relative_path`; form `file` | [files.go](../../internal/api/handlers/files.go) |
| `LinksHandler.HandleAccessPublicLink` | `action`: string | [links.go](../../internal/api/handlers/links.go) |
| `LinksHandler.HandleCreatePublicLink` | `serverId`: string | [links.go](../../internal/api/handlers/links.go) |
| `LoadersHandler.HandleGetLoaderMetadata` | query `includeSnapshots`; query `includeUnstable`; query `mcVersion`; query `buildVersion`; query `loaderVersion`; query `installerVersion` | [loaders.go](../../internal/api/handlers/loaders.go) |
| `LoadersHandler.HandleGetLoaderVersions` | query `includeSnapshots`; query `includeUnstable`; query `mcVersion` | [loaders.go](../../internal/api/handlers/loaders.go) |
| `ServerHandler.HandleBackupServer` | `name` (omitempty): string; `requestId`: string | [server.go](../../internal/api/handlers/server.go) |
| `ServerHandler.HandleCreateServer` | `name`: string; `version`: string; `loader`: string; `loaderOptions`: loader.LoaderOptions; `ram`: int; `requestId`: string | [server.go](../../internal/api/handlers/server.go) |
| `ServerHandler.HandleUpdateServer` | `name`: *string; `ram`: *int; `customArgs`: *string | [server.go](../../internal/api/handlers/server.go) |
| `ServerHandler.HandleUpdateServerAutoBackup` | `enabled`: bool; `intervalValue`: int; `intervalUnit`: string; `maxBackups`: int | [server.go](../../internal/api/handlers/server.go) |
| `ServerHandler.HandleUpdateServerSettings` | body: `server.ServerSettings` | [server.go](../../internal/api/handlers/server.go) |
| `ServerHandler.HandleUpdateServerVersion` | `version`: string; `includeDependencies` (omitempty): *bool | [server.go](../../internal/api/handlers/server.go) |
| `ServerHandler.HandleUploadServerIcon` | form `icon` | [server.go](../../internal/api/handlers/server.go) |
| `SettingsHandler.HandleSetCurseForgeKey` | `apiKey`: string | [settings.go](../../internal/api/handlers/settings.go) |
| `SettingsHandler.HandleSetLogBufferSize` | `log_buffer_size`: int | [settings.go](../../internal/api/handlers/settings.go) |
| `SettingsHandler.HandleSetPortRange` | `start`: int; `end`: int | [settings.go](../../internal/api/handlers/settings.go) |
| `SettingsHandler.HandleSetPublicIP` | `public_ip`: string | [settings.go](../../internal/api/handlers/settings.go) |
| `UploadHandler.HandleCreate` | body: `createUploadRequest`; `clientId`: string; `kind`: upload.Kind; `serverId`: string; `directoryPath`: string; `relativePath`: string; `filename`: string; `contentType`: string; `totalBytes`: int64 | [uploads.go](../../internal/api/handlers/uploads.go) |
| `UsersHandler.HandleCreateUser` | body: `RegisterRequest`; `username`: string; `password`: string | [users.go](../../internal/api/handlers/users.go) |
| `UsersHandler.HandleUpdatePassword` | `password`: string | [users.go](../../internal/api/handlers/users.go) |

## Critical request/response flows

### Provisioning and lifecycle

`POST /servers` (admin) accepts `name`, `loader`, `version`, `ram`, `requestId`, and `loaderOptions`. It returns **202** `{ "status": "creating", "id": "<requestId>" }`, not a completed Server. Subscribe to `/ws/progress/{requestId}` before submitting; omitted IDs use the shared `progress` hub. `loaderOptions.mcVersion` falls back to `version`. Completion/error is asynchronous. The frontend `createServer` generic currently names `Server`, despite this different handler response; consumers should follow the handler wire contract.

`GET /servers` returns an array of domain Server records filtered for non-admin users; a nil Go slice can serialize as `null`, so clients must not assume an empty result is always `[]`. `/servers/{id}/stats` returns ServerStats; `/servers-stats` aggregates stats. Start/stop/restart/kill enforce `CanControlPower || CanViewConsole` in their HTTP handlers. Creation currently writes `eula=true`; explicit consent is not available yet.

### Public links

`POST /public-links` accepts `{ "serverId": "..." }` and returns PublicLink; an existing server link is returned rather than creating another. Admin or a user with that server's console permission may create/read/revoke it. No record means disabled; get returns 404. Deletion returns 204. New links use `action: "control"`.

`GET /public-links/{token}` returns `name`, `version`, `loader`, `status`, `id`, `port`, `onlinePlayers`, `maxPlayers`. `POST /public-links/{token}/access` accepts `{ "action": "start" }` or `stop`, and returns `{ "status": "executed" }`; unsupported action is 400, invalid token 404. The handler also supports stored `start`-only links although creation uses `control`. Its "expired link" text is not proof of a time-based expiry field.

### Backup restore

`POST /backups/{name}/restore` accepts `targetServerId` for an existing instance, or `newServerName`, `newServerRam`, `newServerLoader`, `newServerVersion` for a new one. Success is 200 `{ "status": "restored" }`. Non-admin needs power permission on the backup's associated server and the target; creation of a new instance from backup requires admin. This policy is narrower than saying every console user may restore everywhere.

### Files and uploads

File operations use server-relative `path`; listing defaults to `/`, content read/write requires a path. Content GET returns plain text and PUT takes raw text, not a JSON `{content}` wrapper. Directory creation takes `{path}`; downloads return bytes. The server manager handles filesystem boundary checks.

`POST /uploads` accepts `clientId`, `kind`, `serverId`, `directoryPath`, `relativePath`, `filename`, `contentType`, `totalBytes`; returns **201** Upload View. Kinds are `server-file`, `backup`, `server-icon`. File targets require console permission, backup targets with a server require power, icons require admin. Subsequent get/chunk/complete/cancel use the creating user's identity.

`PUT /uploads/{id}/chunk` takes octet-stream bytes and `Content-Range: bytes START-END/TOTAL`, inclusive END; max chunk is **5 MiB**. Success returns 200 View; completion returns 202 View while processing; cancel returns 204. Use receivedBytes/status to reconcile retry rather than guessing progress. Status values: pending, uploading, ready, processing, completed, error, cancelled. Sessions are daemon-memory state, not durable resumable transfers across restart.

### Add-ons and settings

Add-on handlers accept provider source/project/version/file selections and optional dependency inclusion; response DTOs are in [`internal/addons/manager.go`](../../internal/addons/manager.go) and [`web/src/types/index.ts`](../../web/src/types/index.ts). Files, add-ons and link management use console permission where implemented; backup/lifecycle rules differ. Settings routes that require admin are marked in the inventory. Per-server settings contract is in [`internal/server/settings.go`](../../internal/server/settings.go); do not infer JSON schema from the Server list DTO.

## WebSocket contract

Both paths are authenticated HTTP upgrades: `/ws/servers/{id}/console` and `/ws/progress/{id}`. The handler selects an in-memory hub by `id`; the path category does not create a different protocol or namespace.

- Console outbound: text log lines; inbound: raw command bytes forwarded via hub Commands to the process supervisor. It is not a JSON command envelope.
- Provisioning outbound: JSON `ProgressEvent` with `serverId`, `message`, `progress`, `currentBytes`, `totalBytes`. Success uses progress 100 and message `Server created successfully`; failure uses serverId `error`, message prefixed `Error:`, progress 0. These are current producer conventions, not a typed universal event discriminator.
- Upload progress broadcasts Upload View on the upload ID hub. Other jobs can have their own progress conventions; consumers must use the corresponding producer contract.
- **Framing:** writePump may combine multiple messages into one text frame with newline separators. A frame is not guaranteed to contain exactly one JSON object. Existing browser progress handlers call JSON.parse(event.data); this is a source-level mismatch to verify, not a runtime-tested failure.
- Bounded in-memory history is replayed on connection. No cursor, persistent replay or automatic reconnection guarantee is defined. Slow clients can be disconnected; fetch REST state after interruption. Inbound limit 512 bytes; ping interval 54 seconds, pong deadline 60 seconds, write deadline 10 seconds.
- **Authorization limit observed:** WSHandler validates a nonempty ID but does not call a server-permission or job-owner check. Routes require authentication; upgrader CheckOrigin currently returns true. Do not claim HTTP per-server checks automatically cover these sockets. This is a review/verification gap, not an authorized code change.

Sources: [`ws.go`](../../internal/api/handlers/ws.go), [`hub.go`](../../internal/ws/hub.go), [`client.go`](../../internal/ws/client.go), [`hub_manager.go`](../../internal/ws/hub_manager.go).

## Additional named request and response DTOs

Settings and add-on structures below are extracted from the producer declarations. Requiredness and allowed combinations come from handler/manager validation, not merely Go field types.

### ServerSettings (server)

Source: [settings.go](../../internal/server/settings.go).

```go
type ServerSettings struct {
	Name                string `json:"name"`
	RAM                 int    `json:"ram"`
	CustomArgs          string `json:"customArgs"`
	Loader              string `json:"loader"`
	Version             string `json:"version"`
	JavaVersion         int    `json:"javaVersion"`
	RequiredJavaVersion int    `json:"requiredJavaVersion"`
	Gamemode            string `json:"gamemode"`
	Difficulty          string `json:"difficulty"`
	MOTD                string `json:"motd"`
	OnlineMode          bool   `json:"onlineMode"`
	SpawnProtection     int    `json:"spawnProtection"`
	PvP                 bool   `json:"pvp"`
	AllowFlight         bool   `json:"allowFlight"`
	EnableCommandBlock  bool   `json:"enableCommandBlock"`
	Hardcore            bool   `json:"hardcore"`
	MaxPlayers          int    `json:"maxPlayers"`
	ViewDistance        int    `json:"viewDistance"`
	SimulationDistance  int    `json:"simulationDistance"`
}
```

### SearchRequest (addons)

Source: [manager.go](../../internal/addons/manager.go).

```go
type SearchRequest struct {
	Query  string `json:"query"`
	Source string `json:"source"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}
```

### VersionsRequest (addons)

Source: [manager.go](../../internal/addons/manager.go).

```go
type VersionsRequest struct {
	Source    AddonSource `json:"source"`
	ProjectID string      `json:"projectId"`
}
```

### VersionsResponse (addons)

Source: [manager.go](../../internal/addons/manager.go).

```go
type VersionsResponse struct {
	Versions []AddonVersion `json:"versions"`
}
```

### SearchResponse (addons)

Source: [manager.go](../../internal/addons/manager.go).

```go
type SearchResponse struct {
	Items      []SearchResult `json:"items"`
	HasMore    bool           `json:"hasMore"`
	NextOffset int            `json:"nextOffset"`
}
```

### InstallRequest (addons)

Source: [manager.go](../../internal/addons/manager.go).

```go
type InstallRequest struct {
	Source              AddonSource `json:"source"`
	ProjectID           string      `json:"projectId"`
	VersionID           string      `json:"versionId,omitempty"`
	FileID              int64       `json:"fileId,omitempty"`
	IncludeDependencies bool        `json:"includeDependencies"`
}
```

### InstallPreviewRequest (addons)

Source: [install_preview.go](../../internal/addons/install_preview.go).

```go
type InstallPreviewRequest struct {
	Source    AddonSource `json:"source"`
	ProjectID string      `json:"projectId"`
	VersionID string      `json:"versionId,omitempty"`
	FileID    int64       `json:"fileId,omitempty"`
}
```

### InstallPreviewResponse (addons)

Source: [install_preview.go](../../internal/addons/install_preview.go).

```go
type InstallPreviewResponse struct {
	Dependencies []InstallPreviewDependency `json:"dependencies"`
}
```

### View (upload)

Source: [manager.go](../../internal/upload/manager.go).

```go
type View struct {
	ID            string `json:"id"`
	ClientID      string `json:"clientId"`
	Kind          Kind   `json:"kind"`
	ServerID      string `json:"serverId,omitempty"`
	Filename      string `json:"filename"`
	Status        Status `json:"status"`
	ReceivedBytes int64  `json:"receivedBytes"`
	TotalBytes    int64  `json:"totalBytes"`
	Progress      int    `json:"progress"`
	Message       string `json:"message,omitempty"`
	Error         string `json:"error,omitempty"`
}
```

## Serialized domain DTOs

Exact field names/types copied from the current Go declarations; `json:"-"` fields are not serialized. JSON tags mix camelCase and snake_case intentionally as currently implemented. Go zero values are not evidence of accepted quality targets.

### Server

Source: [server.go](../../internal/domain/server.go).

```go
type Server struct {
	ID                      string      `json:"id"`
	Name                    string      `json:"name"`
	FolderName              string      `json:"folderName"`
	Version                 string      `json:"version"`
	Loader                  string      `json:"loader"`
	Port                    int         `json:"port"`
	RAM                     int         `json:"ram"`
	Status                  string      `json:"status"`
	CustomArgs              string      `json:"customArgs"`
	JavaVersion             int         `json:"-"`
	CreatedAt               time.Time   `json:"created_at"`
	AutoBackupEnabled       bool        `json:"autoBackupEnabled"`
	AutoBackupIntervalValue int         `json:"autoBackupIntervalValue"`
	AutoBackupIntervalUnit  string      `json:"autoBackupIntervalUnit"`
	AutoBackupMaxBackups    int         `json:"autoBackupMaxBackups"`
	AutoBackupLastRunAt     *time.Time  `json:"autoBackupLastRunAt,omitempty"`
	Permissions             *Permission `json:"permissions,omitempty"`
}
```

### BackupInfo

Source: [server.go](../../internal/domain/server.go).

```go
type BackupInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}
```

### ProgressEvent

Source: [server.go](../../internal/domain/server.go).

```go
type ProgressEvent struct {
	ServerID     string  `json:"serverId"`
	Message      string  `json:"message"`
	Progress     float64 `json:"progress"`
	CurrentBytes int64   `json:"currentBytes"`
	TotalBytes   int64   `json:"totalBytes"`
}
```

### ServerStats

Source: [server.go](../../internal/domain/server.go).

```go
type ServerStats struct {
	CPU           float64  `json:"cpu"`
	RAM           uint64   `json:"ram"`
	Disk          int64    `json:"disk"`
	OnlinePlayers int      `json:"onlinePlayers"`
	MaxPlayers    int      `json:"maxPlayers"`
	UptimeSeconds int64    `json:"uptimeSeconds"`
	Players       []Player `json:"players"`
}
```

### Player

Source: [server.go](../../internal/domain/server.go).

```go
type Player struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}
```

### User

Source: [user.go](../../internal/domain/user.go).

```go
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
	Role     string `json:"role"`
}
```

### Permission

Source: [user.go](../../internal/domain/user.go).

```go
type Permission struct {
	UserID          string `json:"userId"`
	ServerID        string `json:"serverId"`
	CanViewConsole  bool   `json:"canViewConsole"`
	CanControlPower bool   `json:"canControlPower"`
}
```

### PublicLink

Source: [user.go](../../internal/domain/user.go).

```go
type PublicLink struct {
	Token    string `json:"token"`
	ServerID string `json:"serverId"`
	Action   string `json:"action"`
}
```

### Backup

Source: [backup.go](../../internal/domain/backup.go).

```go
type Backup struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	FileName   string    `json:"fileName"`
	ServerID   string    `json:"serverId"`
	ServerName string    `json:"serverName"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"createdAt"`
	CreatedBy  string    `json:"createdBy"`
}
```

### LoaderOptions

Source: [contract.go](../../internal/loader/contract.go).

```go
type LoaderOptions struct {
	MCVersion        string `json:"mcVersion,omitempty"`
	IncludeSnapshots bool   `json:"includeSnapshots,omitempty"`
	IncludeUnstable  bool   `json:"includeUnstable,omitempty"`
	BuildVersion     string `json:"buildVersion,omitempty"`
	LoaderVersion    string `json:"loaderVersion,omitempty"`
	InstallerVersion string `json:"installerVersion,omitempty"`
	JavaPath         string `json:"-"`
}
```

### LoaderMetadata

Source: [contract.go](../../internal/loader/contract.go).

```go
type LoaderMetadata struct {
	LatestVersion     string   `json:"latestVersion,omitempty"`
	MinecraftVersions []string `json:"minecraftVersions,omitempty"`
	BuildVersions     []string `json:"buildVersions,omitempty"`
	LoaderVersions    []string `json:"loaderVersions,omitempty"`
	InstallerVersions []string `json:"installerVersions,omitempty"`
}
```

## Persistence and migration contract

SQLite models are separate from transport DTOs in [`gorm.go`](../../internal/storage/gorm.go). Server and User use ID primary keys; usernames are unique. Permission has composite `(UserID, ServerID)` key. PublicLink uses Token primary key and stores ServerID/action. Backup has ID key, unique Name, indexed ServerID, archive FileName and metadata; ServerName is a read-only projection. Setting uses Key as primary key and string Value. These are logical ID references; the declarations do not establish enforced database foreign-key/cascade relationships.

Startup calls AutoMigrate for Server, Setting, User, Permission, PublicLink and Backup, then initializes missing default settings. There is no versioned migration/rollback contract in this flow. Test schema upgrades against a copy and retain a consistent DB/config/files backup; do not promise automatic downgrade. Backup scheduling normalizes invalid/missing interval to 24 hours and maximum count to 10, with units minute/hour/day; enabled defaults false. These implementation defaults are not RPO/RTO guarantees. EULA state is a server file, not a persisted consent record in these models.

## Accepted EULA target — not implemented

Admin and server-specific console users may explicitly accept. For a newly created server with EULA false, browser start presents the modal; decline leaves it stopped/unaccepted, accept persists consent before launch, true skips prompting. Public-link and noninteractive CLI starts must report that EULA acceptance is required and stop without launching or accepting. While TUI exists, it should offer explicit acceptance to an eligible admin/console user; the CLI admin credential maps to admin under current auth. TUI retirement is planned, not completed.

No target endpoint, status code or response DTO has been selected. Do not invent an implemented `/eula` route. Creation defaults, common start validation, UI/TUI acceptance and user-visible CLI/public refusal remain implementation work (RF-011 / TC-014). Missing or malformed EULA-file handling remains unspecified.

## Contract maintenance and verification

Keep router inventory, handler DTOs, SDK and frontend types synchronized when behavior changes. Type annotations alone do not prove the emitted response matches. Current checks are listed in [quality/operations](../04-quality-operations/index.md#documentation-pass-checks); no complete endpoint runtime or WebSocket compatibility suite was run for this extraction.
