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
