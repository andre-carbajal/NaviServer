import type { AxiosProgressEvent } from 'axios';
import axios from 'axios';

import type {
  AddonListResponse,
  AddonSearchResponse,
  AddonVersionsResponse,
  Backup,
  FileEntry,
  Permission,
  Server,
  ServerSettings,
  ServerStats,
  ServerVersionUpdateResult,
} from '../types';

const API_HOST = window.location.hostname;
const API_PROTOCOL = window.location.protocol;
const API_ORIGIN = `${window.location.protocol}//${window.location.host}`;
const DEV_API_PORT = 23009;

const normalizeBaseUrl = (value: string) =>
  value.endsWith('/') ? value.slice(0, -1) : value;

const resolveApiBaseUrl = () => {
  const envBaseUrl = import.meta.env.VITE_API_BASE_URL;
  if (typeof envBaseUrl === 'string' && envBaseUrl.trim() !== '') {
    return normalizeBaseUrl(envBaseUrl.trim());
  }

  const envApiPort = import.meta.env.VITE_API_PORT;
  const resolvedEnvApiPort =
    typeof envApiPort === 'number'
      ? String(envApiPort)
      : typeof envApiPort === 'string'
        ? envApiPort
        : '';
  if (resolvedEnvApiPort.trim() !== '') {
    return `${API_PROTOCOL}//${API_HOST}:${resolvedEnvApiPort.trim()}`;
  }

  if (import.meta.env.DEV) {
    return `${API_PROTOCOL}//${API_HOST}:${DEV_API_PORT}`;
  }

  return normalizeBaseUrl(API_ORIGIN);
};

const resolveWsBaseUrl = (apiBaseUrl: string) => {
  const envWsBaseUrl = import.meta.env.VITE_WS_BASE_URL;
  if (typeof envWsBaseUrl === 'string' && envWsBaseUrl.trim() !== '') {
    return normalizeBaseUrl(envWsBaseUrl.trim());
  }

  if (apiBaseUrl.startsWith('https://')) {
    return `wss://${apiBaseUrl.slice('https://'.length)}`;
  }
  if (apiBaseUrl.startsWith('http://')) {
    return `ws://${apiBaseUrl.slice('http://'.length)}`;
  }

  return apiBaseUrl;
};

const API_BASE_URL = resolveApiBaseUrl();
export const WS_BASE_URL = resolveWsBaseUrl(API_BASE_URL);
const NETWORK_ERROR_COOLDOWN_MS = 8000;
const ADDON_SYNC_TIMEOUT_MS = 30000;
const ADDON_DOWNLOAD_TIMEOUT_MS = 300000;

let lastNetworkErrorAt = 0;
let isBackendOffline = false;

const apiInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 5000,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
});

apiInstance.interceptors.response.use(
  (response) => {
    if (isBackendOffline) {
      isBackendOffline = false;
      window.dispatchEvent(new CustomEvent('network-recovered'));
    }
    return response;
  },
  (error) => {
    if (error.code === 'ERR_NETWORK') {
      const now = Date.now();
      if (now - lastNetworkErrorAt >= NETWORK_ERROR_COOLDOWN_MS) {
        lastNetworkErrorAt = now;
        const event = new CustomEvent('network-error', {
          detail: {
            message: 'Backend unavailable. Make sure NaviServer is running.',
          },
        });
        window.dispatchEvent(event);
      }
      isBackendOffline = true;
    }
    return Promise.reject(error);
  },
);

export const api = {
  getLoaders: () => apiInstance.get<string[]>('/loaders'),
  getLoaderVersions: (loader: string) =>
    apiInstance.get<string[]>(`/loaders/${loader}/versions`),
  getLoaderMetadata: (
    loader: string,
    options?: {
      mcVersion?: string;
      includeSnapshots?: boolean;
      includeUnstable?: boolean;
      buildVersion?: string;
      loaderVersion?: string;
      installerVersion?: string;
    },
  ) =>
    apiInstance.get<{
      latestVersion?: string;
      minecraftVersions?: string[];
      buildVersions?: string[];
      loaderVersions?: string[];
      installerVersions?: string[];
    }>(`/loaders/${loader}/metadata`, { params: options }),
  getServers: () => apiInstance.get<Server[]>('/servers'),
  getServer: (id: string) => apiInstance.get<Server>(`/servers/${id}`),
  getServerStats: (id: string) =>
    apiInstance.get<ServerStats>(`/servers/${id}/stats`),
  getAllServerStats: () =>
    apiInstance.get<Record<string, ServerStats>>('/servers-stats'),
  getServerIconUrl: (id: string) => `${API_BASE_URL}/servers/${id}/icon`,
  uploadServerIcon: (id: string, file: File) => {
    const formData = new FormData();
    formData.append('icon', file);
    return apiInstance.post(`/servers/${id}/icon`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  },
  createServer: (data: {
    name: string;
    loader: string;
    version?: string;
    ram: number;
    requestId?: string;
    loaderOptions?: {
      mcVersion?: string;
      includeSnapshots?: boolean;
      includeUnstable?: boolean;
      buildVersion?: string;
      loaderVersion?: string;
      installerVersion?: string;
    };
  }) => apiInstance.post<Server>('/servers', data),
  updateServer: (
    id: string,
    data: {
      name?: string;
      ram?: number;
      customArgs?: string;
    },
  ) => apiInstance.put<Server>(`/servers/${id}`, data),
  getServerSettings: (id: string) =>
    apiInstance.get<ServerSettings>(`/servers/${id}/settings`),
  updateServerSettings: (id: string, data: ServerSettings) =>
    apiInstance.put(`/servers/${id}/settings`, data),
  updateServerAutoBackup: (
    id: string,
    data: {
      enabled: boolean;
      intervalValue: number;
      intervalUnit: 'minute' | 'hour' | 'day';
      maxBackups: number;
    },
  ) => apiInstance.put<Server>(`/servers/${id}/auto-backup`, data),
  getServerVersionOptions: (id: string) =>
    apiInstance.get<{ versions: string[] }>(`/servers/${id}/version-options`),
  updateServerVersion: (
    id: string,
    data: { version: string; includeDependencies?: boolean },
  ) =>
    apiInstance.post<ServerVersionUpdateResult>(
      `/servers/${id}/version-update`,
      data,
      { timeout: 300000 },
    ),
  deleteServer: (id: string) => apiInstance.delete(`/servers/${id}`),
  startServer: (id: string) => apiInstance.post(`/servers/${id}/start`),
  stopServer: (id: string) => apiInstance.post(`/servers/${id}/stop`),
  restartServer: (id: string) => apiInstance.post(`/servers/${id}/restart`),
  killServer: (id: string) => apiInstance.post(`/servers/${id}/kill`),
  getPortRange: () => apiInstance.get('/settings/port-range'),
  updatePortRange: (data: { start: number; end: number }) =>
    apiInstance.put('/settings/port-range', data),
  getLogBufferSize: () => apiInstance.get('/settings/log-buffer-size'),
  updateLogBufferSize: (data: { log_buffer_size: number }) =>
    apiInstance.put('/settings/log-buffer-size', data),
  getPublicIP: () =>
    apiInstance.get<{ public_ip: string }>('/settings/public-ip'),
  updatePublicIP: (data: { public_ip: string }) =>
    apiInstance.put('/settings/public-ip', data),
  getCurseForgeKeyStatus: () =>
    apiInstance.get<{
      hasCustomKey: boolean;
      hasEmbeddedKey: boolean;
      effectiveSource: 'custom' | 'embedded' | 'none';
    }>('/settings/curseforge-key'),
  setCurseForgeKey: (apiKey: string) =>
    apiInstance.put('/settings/curseforge-key', { apiKey }),
  clearCurseForgeKey: () => apiInstance.delete('/settings/curseforge-key'),
  getNetworkInterfaces: () =>
    apiInstance.get<{ interfaces: string[] }>('/system/interfaces'),
  getSystemResources: () =>
    apiInstance.get<{ totalMemoryMb: number }>('/system/resources'),
  listBackups: (serverId: string) =>
    apiInstance.get<Backup[]>(`/servers/${serverId}/backups`),
  listAllBackups: () => apiInstance.get<Backup[]>('/backups'),
  uploadBackup: (
    file: File,
    onUploadProgress: (progressEvent: AxiosProgressEvent) => void,
    serverId?: string,
  ) => {
    const formData = new FormData();
    formData.append('backup', file);
    if (serverId) {
      formData.append('serverId', serverId);
    }
    return apiInstance.post('/backups/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      onUploadProgress,
    });
  },
  createBackup: (serverId: string, name?: string, requestId?: string) =>
    apiInstance.post<{
      status: string;
      id: string;
    }>(`/servers/${serverId}/backup`, { name, requestId }),
  deleteBackup: (backupName: string) =>
    apiInstance.delete(`/backups/${backupName}`),
  updateBackup: (backupName: string, serverId: string) =>
    apiInstance.put(`/backups/${backupName}`, { serverId }),
  getBackupDownloadUrl: (backupName: string) =>
    `${API_BASE_URL}/backups/${backupName}/download`,
  cancelBackupCreation: (requestId: string) =>
    apiInstance.delete(`/backups/progress/${requestId}`),
  restoreBackup: (
    backupName: string,
    data: {
      targetServerId?: string;
      newServerName?: string;
      newServerRam?: number;
      newServerLoader?: string;
      newServerVersion?: string;
    },
  ) => apiInstance.post(`/backups/${backupName}/restore`, data),
  checkUpdates: () =>
    apiInstance.get<{
      current_version: string;
      latest_version: string;
      update_available: boolean;
      release_url: string;
    }>('/updates'),
  getVersion: () => apiInstance.get<{ version: string }>('/version'),
  restartDaemon: () => apiInstance.post('/system/restart'),
  listFiles: (serverId: string, path: string) =>
    apiInstance.get<FileEntry[]>(`/servers/${serverId}/files`, {
      params: { path },
    }),
  getFileContent: (serverId: string, path: string) =>
    apiInstance.get(`/servers/${serverId}/files/content`, {
      params: { path },
      responseType: 'text',
    }),
  saveFileContent: (serverId: string, path: string, content: string) =>
    apiInstance.put(`/servers/${serverId}/files/content`, content, {
      params: { path },
      headers: { 'Content-Type': 'text/plain' },
    }),
  createDirectory: (serverId: string, path: string) =>
    apiInstance.post(`/servers/${serverId}/files/directory`, { path }),
  deleteFile: (serverId: string, path: string) =>
    apiInstance.delete(`/servers/${serverId}/files`, { params: { path } }),
  downloadFile: (serverId: string, path: string) =>
    apiInstance.get(`/servers/${serverId}/files/download`, {
      params: { path },
      responseType: 'blob',
    }),
  uploadFile: (
    serverId: string,
    path: string,
    file: File,
    relativePath?: string,
  ) => {
    const formData = new FormData();
    formData.append('file', file);
    return apiInstance.post(`/servers/${serverId}/files/upload`, formData, {
      params: { path, relative_path: relativePath },
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  },
  listAddons: (serverId: string) =>
    apiInstance.get<AddonListResponse>(`/servers/${serverId}/addons`, {
      timeout: ADDON_SYNC_TIMEOUT_MS,
    }),
  syncAddons: (serverId: string) =>
    apiInstance.post<AddonListResponse>(
      `/servers/${serverId}/addons/sync`,
      undefined,
      {
        timeout: ADDON_SYNC_TIMEOUT_MS,
      },
    ),
  searchAddons: (
    serverId: string,
    data: {
      query: string;
      source?: 'modrinth' | 'curseforge';
      offset?: number;
      limit?: number;
    },
  ) =>
    apiInstance.post<AddonSearchResponse>(
      `/servers/${serverId}/addons/search`,
      data,
      {
        timeout: 30000,
      },
    ),
  getAddonVersions: (
    serverId: string,
    data: {
      source: 'modrinth' | 'curseforge';
      projectId: string;
    },
  ) =>
    apiInstance.post<AddonVersionsResponse>(
      `/servers/${serverId}/addons/versions`,
      data,
      {
        timeout: 30000,
      },
    ),
  installAddon: (
    serverId: string,
    data: {
      source: 'modrinth' | 'curseforge';
      projectId: string;
      versionId?: string;
      fileId?: number;
      includeDependencies?: boolean;
    },
  ) =>
    apiInstance.post(`/servers/${serverId}/addons/install`, data, {
      timeout: ADDON_DOWNLOAD_TIMEOUT_MS,
    }),
  updateAddon: (
    serverId: string,
    addonId: string,
    data?: {
      includeDependencies?: boolean;
    },
  ) =>
    apiInstance.post(
      `/servers/${serverId}/addons/${encodeURIComponent(addonId)}/update`,
      data,
      {
        timeout: ADDON_DOWNLOAD_TIMEOUT_MS,
      },
    ),
  setAddonDisabled: (
    serverId: string,
    addonId: string,
    data: {
      disabled: boolean;
    },
  ) =>
    apiInstance.post(
      `/servers/${serverId}/addons/${encodeURIComponent(addonId)}/disabled`,
      data,
    ),
  updateAllAddons: (
    serverId: string,
    data?: {
      includeDependencies?: boolean;
    },
  ) =>
    apiInstance.post(`/servers/${serverId}/addons/update-all`, data, {
      timeout: ADDON_DOWNLOAD_TIMEOUT_MS,
    }),
  deleteAddon: (serverId: string, addonId: string) =>
    apiInstance.delete(
      `/servers/${serverId}/addons/${encodeURIComponent(addonId)}`,
    ),
  login: (username: string, password: string) =>
    apiInstance.post('/auth/login', { username, password }),
  logout: () => apiInstance.post('/auth/logout'),
  setup: (username: string, password: string) =>
    apiInstance.post('/auth/setup', { username, password }),
  checkSetup: () => apiInstance.get<{ setup_needed: boolean }>('/auth/setup'),
  getMe: () => apiInstance.get('/auth/me'),
  listUsers: () => apiInstance.get('/users'),
  createUser: (data: { username: string; password?: string }) =>
    apiInstance.post('/users', data),
  deleteUser: (id: string) => apiInstance.delete(`/users/${id}`),
  updatePassword: (id: string, password: string) =>
    apiInstance.put(`/users/${id}/password`, { password }),
  updatePermissions: (perms: Permission[]) =>
    apiInstance.put('/users/permissions', perms),
  getPermissions: (userId: string) =>
    apiInstance.get(`/users/${userId}/permissions`),
  createPublicLink: (serverId: string) =>
    apiInstance.post<{
      token: string;
      serverId: string;
      action: string;
    }>('/public-links', { serverId }),
  getPublicLink: (serverId: string) =>
    apiInstance.get<{
      token: string;
      serverId: string;
      action: string;
    }>(`/servers/${serverId}/public-link`),
  deletePublicLink: (token: string) =>
    apiInstance.delete(`/public-links/${token}`),
  getPublicServerInfo: (token: string) =>
    apiInstance.get<{
      name: string;
      version: string;
      loader: string;
      status: string;
      id: string;
    }>(`/public-links/${token}`),
  accessPublicLink: (token: string, action: 'start' | 'stop') =>
    apiInstance.post(`/public-links/${token}/access`, { action }),
};
