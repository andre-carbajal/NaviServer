export type ApiUrlEnv = {
  DEV: boolean;
  VITE_API_BASE_URL?: string;
  VITE_API_PORT?: number | string;
  VITE_WS_BASE_URL?: string;
};

export type BrowserLocationLike = {
  host: string;
  hostname: string;
  protocol: string;
};

const DEV_API_PORT = 23009;

export const normalizeBaseUrl = (value: string) =>
  value.endsWith('/') ? value.slice(0, -1) : value;

export const resolveApiBaseUrl = (
  env: ApiUrlEnv,
  location: BrowserLocationLike,
) => {
  const envBaseUrl = env.VITE_API_BASE_URL;
  if (typeof envBaseUrl === 'string' && envBaseUrl.trim() !== '') {
    return normalizeBaseUrl(envBaseUrl.trim());
  }

  const envApiPort = env.VITE_API_PORT;
  let resolvedEnvApiPort = '';
  if (typeof envApiPort === 'number') {
    resolvedEnvApiPort = String(envApiPort);
  } else if (typeof envApiPort === 'string') {
    resolvedEnvApiPort = envApiPort;
  }

  if (resolvedEnvApiPort.trim() !== '') {
    return `${location.protocol}//${location.hostname}:${resolvedEnvApiPort.trim()}`;
  }

  if (env.DEV) {
    return `${location.protocol}//${location.hostname}:${DEV_API_PORT}`;
  }

  return normalizeBaseUrl(`${location.protocol}//${location.host}`);
};

export const resolveWsBaseUrl = (apiBaseUrl: string, env?: ApiUrlEnv) => {
  const envWsBaseUrl = env?.VITE_WS_BASE_URL;
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
