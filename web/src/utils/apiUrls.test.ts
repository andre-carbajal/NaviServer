import { describe, expect, it } from 'vitest';

import {
  normalizeBaseUrl,
  resolveApiBaseUrl,
  resolveWsBaseUrl,
  type ApiUrlEnv,
  type BrowserLocationLike,
} from './apiUrls';

const baseLocation: BrowserLocationLike = {
  host: 'panel.example.com',
  hostname: 'panel.example.com',
  protocol: 'https:',
};

const makeEnv = (overrides: Partial<ApiUrlEnv> = {}): ApiUrlEnv => ({
  DEV: false,
  ...overrides,
});

describe('api url helpers', () => {
  it('normalizes trailing slashes', () => {
    expect(normalizeBaseUrl('https://api.example.com/')).toBe(
      'https://api.example.com',
    );
  });

  it('prefers explicit API base URL from env', () => {
    expect(
      resolveApiBaseUrl(
        makeEnv({ VITE_API_BASE_URL: 'https://api.example.com/' }),
        baseLocation,
      ),
    ).toBe('https://api.example.com');
  });

  it('uses a configured API port when provided', () => {
    expect(
      resolveApiBaseUrl(makeEnv({ VITE_API_PORT: '24000' }), baseLocation),
    ).toBe('https://panel.example.com:24000');
  });

  it('falls back to the dev API port in development', () => {
    expect(resolveApiBaseUrl(makeEnv({ DEV: true }), baseLocation)).toBe(
      'https://panel.example.com:23009',
    );
  });

  it('uses the browser origin in production when no env overrides exist', () => {
    expect(resolveApiBaseUrl(makeEnv(), baseLocation)).toBe(
      'https://panel.example.com',
    );
  });

  it('derives websocket URLs from the resolved API URL', () => {
    expect(resolveWsBaseUrl('https://api.example.com')).toBe(
      'wss://api.example.com',
    );
    expect(resolveWsBaseUrl('http://localhost:23009')).toBe(
      'ws://localhost:23009',
    );
  });

  it('prefers an explicit websocket base URL when provided', () => {
    expect(
      resolveWsBaseUrl('https://api.example.com', {
        ...makeEnv(),
        VITE_WS_BASE_URL: 'wss://socket.example.com/',
      }),
    ).toBe('wss://socket.example.com');
  });
});
