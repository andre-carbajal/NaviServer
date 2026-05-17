export interface Server {
  id: string;
  name: string;
  version: string;
  loader: string;
  port: number;
  ram: number;
  status: 'STOPPED' | 'RUNNING' | 'STARTING' | 'STOPPING' | 'CREATING';
  customArgs?: string;
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
  pvp: boolean;
  allowFlight: boolean;
  enableCommandBlock: boolean;
  hardcore: boolean;
  maxPlayers: number;
  viewDistance: number;
  simulationDistance: number;
}
