import type { ServerSettings } from '../types';

const MINEATAR_BASE_URL = 'https://api.mineatar.io/head';
const STEVE_UUID = '8667ba71-b85a-4004-af54-457a9734eed7';
export const RAM_MIN_MB = 512;
export const FALLBACK_RAM_MAX_MB = 262144;

export const getAvatarUrl = (uuid?: string) =>
  `${MINEATAR_BASE_URL}/${encodeURIComponent(uuid || STEVE_UUID)}?scale=8&overlay=true`;

const parseMinecraftVersionParts = (
  value: string,
): number[] | null => {
  const normalized = value.trim().toLowerCase().replace(/^v/, '');
  if (normalized === '') return null;

  const parts = normalized.split('.');
  const parsed: number[] = [];

  for (const part of parts) {
    const digits = part.match(/^\d+/)?.[0];
    if (!digits) return null;
    parsed.push(Number(digits));
  }

  return parsed;
};

export const compareMinecraftVersions = (
  a: string,
  b: string,
): number | null => {
  const left = parseMinecraftVersionParts(a);
  const right = parseMinecraftVersionParts(b);
  if (!left || !right) return null;

  const maxLen = Math.max(left.length, right.length);
  for (let i = 0; i < maxLen; i += 1) {
    const lv = left[i] ?? 0;
    const rv = right[i] ?? 0;
    if (lv > rv) return 1;
    if (lv < rv) return -1;
  }
  return 0;
};

export const isFutureMinecraftVersion = (
  candidate: string,
  current: string,
) => {
  const comparison = compareMinecraftVersions(candidate, current);
  return comparison !== null && comparison > 0;
};

export const clampRamAllocation = (value: number, maxRamMb?: number) => {
  const upperBound = Number.isFinite(maxRamMb)
    ? Math.max(RAM_MIN_MB, Number(maxRamMb))
    : FALLBACK_RAM_MAX_MB;

  return Math.min(upperBound, Math.max(RAM_MIN_MB, value));
};

export const normalizeServerSettings = (
  settings: ServerSettings,
  maxRamMb?: number,
): ServerSettings => ({
  ...settings,
  ram: clampRamAllocation(settings.ram, maxRamMb),
  onlineMode: settings.onlineMode ?? true,
  spawnProtection:
    Number.isFinite(settings.spawnProtection) && settings.spawnProtection >= 0
      ? settings.spawnProtection
      : 16,
});

export const upsertPropertyLine = (
  content: string,
  key: string,
  value: string,
) => {
  const normalized = content.replace(/\r\n/g, '\n');
  const lines = normalized.split('\n');
  let updated = false;
  const nextLines = lines.map((line) => {
    const trimmed = line.trim();
    if (trimmed === '' || trimmed.startsWith('#') || !line.includes('=')) {
      return line;
    }

    const [rawKey] = line.split('=', 1);
    if (rawKey.trim() !== key) {
      return line;
    }

    updated = true;
    return `${key}=${value}`;
  });

  if (!updated) {
    if (nextLines.length > 0 && nextLines[nextLines.length - 1] !== '') {
      nextLines.push('');
    }
    nextLines.push(`${key}=${value}`);
  }

  return nextLines.join('\n');
};

export const readIntPropertyFromContent = (
  content: string,
  key: string,
): number | null => {
  const normalized = content.replace(/\r\n/g, '\n');
  const lines = normalized.split('\n');
  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed === '' || trimmed.startsWith('#') || !line.includes('=')) {
      continue;
    }
    const [rawKey, rawValue = ''] = line.split('=', 2);
    if (rawKey.trim() !== key) {
      continue;
    }
    const parsed = Number.parseInt(rawValue.trim(), 10);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
    return null;
  }
  return null;
};
