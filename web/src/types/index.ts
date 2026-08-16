export interface Server {
  id: string;
  name: string;
  version: string;
  loader: string;
  port: number;
  ram: number;
  status: 'STOPPED' | 'RUNNING' | 'STARTING' | 'STOPPING' | 'CREATING';
  customArgs?: string;
  autoBackupEnabled?: boolean;
  autoBackupIntervalValue?: number;
  autoBackupIntervalUnit?: 'minute' | 'hour' | 'day';
  autoBackupMaxBackups?: number;
  autoBackupLastRunAt?: string;
  progress?: number;
  progressMessage?: string;
  steps?: ProgressStep[];
  permissions?: Permission;
}

export interface ProgressStep {
  label: string;
  state: 'pending' | 'running' | 'done' | 'failed';
  progress?: number;
}

export interface Backup {
  name: string;
  size: number;
  createdAt?: string;
  serverId?: string;
  serverName?: string;
  status?: 'CREATING' | 'READY' | 'ERROR';
  progress?: number;
  requestId?: string;
  progressMessage?: string;
}

export interface ServerStats {
  cpu: number;
  ram: number;
  disk: number;
  onlinePlayers: number;
  maxPlayers: number;
  uptimeSeconds: number;
  players: PlayerInfo[];
}

export interface PlayerInfo {
  name: string;
  id: string;
}

export interface User {
  id: string;
  username: string;
  role: string;
}

export interface Permission {
  userId: string;
  serverId: string;
  canViewConsole: boolean;
  canControlPower: boolean;
}

export interface FileEntry {
  name: string;
  path: string;
  isDirectory: boolean;
  size: number;
  lastModified: string;
}

export interface ServerSettings {
  name: string;
  ram: number;
  customArgs: string;
  loader: string;
  version: string;
  gamemode: 'survival' | 'creative' | 'adventure' | 'spectator';
  difficulty: 'peaceful' | 'easy' | 'normal' | 'hard';
  motd: string;
  onlineMode: boolean;
  spawnProtection: number;
  pvp: boolean;
  allowFlight: boolean;
  enableCommandBlock: boolean;
  hardcore: boolean;
  maxPlayers: number;
  viewDistance: number;
  simulationDistance: number;
}

export type AddonSource = 'modrinth' | 'curseforge' | 'manual';

export type AddonStatus = 'installed' | 'update_available' | 'unknown_source';

export type AddonType = 'mod' | 'plugin';

export type AddonReleaseType = 'release' | 'beta' | 'alpha';

export interface AddonVersion {
  versionId: string;
  versionName: string;
  versionLabel: string;
  releaseType: AddonReleaseType;
  publishedAt?: string;
  downloadUrl: string;
  filename: string;
  source: AddonSource;
  fileId?: number;
}

export interface AddonDependency {
  projectId: string;
  fileId?: number;
  name?: string;
  required: boolean;
  source: string;
  description?: string;
}

export interface Addon {
  id: string;
  name: string;
  fileName: string;
  path: string;
  iconUrl?: string;
  source: AddonSource;
  type: AddonType;
  status: AddonStatus;
  projectId?: string;
  projectSlug?: string;
  projectName?: string;
  projectUrl?: string;
  versionId?: string;
  versionName?: string;
  versionLabel?: string;
  releaseType?: AddonReleaseType;
  hashSha1?: string;
  hashSha512?: string;
  curseFingerprint?: number;
  size: number;
  modifiedAt: string;
  latest?: AddonVersion;
  missingDependencies?: AddonDependency[];
  disabled: boolean;
}

export interface AddonListResponse {
  addonType: AddonType;
  items: Addon[];
}

export interface AddonSearchResult {
  source: AddonSource;
  projectId: string;
  projectSlug?: string;
  projectName: string;
  authorName?: string;
  description?: string;
  projectUrl?: string;
  iconUrl?: string;
  downloads?: number;
  latest?: AddonVersion;
  versions?: AddonVersion[];
}

export interface AddonSearchResponse {
  items: AddonSearchResult[];
  hasMore: boolean;
  nextOffset: number;
}

export interface AddonVersionsResponse {
  versions: AddonVersion[];
}

export interface AddonInstallDependency {
  name: string;
  source: Exclude<AddonSource, 'manual'>;
  projectId: string;
  iconUrl?: string;
  versionId?: string;
  fileId?: number;
  versionLabel?: string;
  filename?: string;
}

export interface AddonInstallPreviewResponse {
  dependencies: AddonInstallDependency[];
}

export interface ServerVersionUpdateResult {
  backupName: string;
  restored: boolean;
  serverUpdated: boolean;
  version: string;
  addons?: {
    updated: Addon[];
    disabled: Addon[];
    failed: Array<{
      id: string;
      name?: string;
      reason: string;
    }>;
  } | null;
}
