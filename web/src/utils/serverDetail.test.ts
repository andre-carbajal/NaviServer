import { describe, expect, it } from 'vitest';

import type { ServerSettings } from '../types';

import {
  clampRamAllocation,
  compareMinecraftVersions,
  getAvatarUrl,
  isFutureMinecraftVersion,
  normalizeServerSettings,
  readIntPropertyFromContent,
  upsertPropertyLine,
} from './serverDetail';

describe('server detail utilities', () => {
  it('parses and compares Minecraft versions safely', () => {
    expect(compareMinecraftVersions('1.21.1', '1.21.2')).toBe(-1);
    expect(compareMinecraftVersions('v1.21.2', '1.21.2')).toBe(0);
    expect(compareMinecraftVersions('1.21.10', '1.21.2')).toBe(1);
    expect(compareMinecraftVersions('latest', '1.21.2')).toBeNull();
  });

  it('detects future versions relative to the current one', () => {
    expect(isFutureMinecraftVersion('1.21.3', '1.21.2')).toBe(true);
    expect(isFutureMinecraftVersion('1.21.2', '1.21.2')).toBe(false);
    expect(isFutureMinecraftVersion('latest', '1.21.2')).toBe(false);
  });

  it('clamps RAM allocations to the supported bounds', () => {
    expect(clampRamAllocation(128, 4096)).toBe(512);
    expect(clampRamAllocation(8192, 4096)).toBe(4096);
    expect(clampRamAllocation(2048)).toBe(2048);
  });

  it('normalizes server settings defaults', () => {
    const settings = {
      name: 'Test',
      ram: 128,
      customArgs: '',
      loader: 'vanilla',
      version: '1.21.1',
      gamemode: 'survival',
      difficulty: 'normal',
      motd: 'Hello',
      onlineMode: undefined,
      spawnProtection: -1,
      pvp: true,
      allowFlight: false,
      enableCommandBlock: false,
      hardcore: false,
      maxPlayers: 20,
      viewDistance: 10,
      simulationDistance: 10,
    } as unknown as ServerSettings;

    expect(normalizeServerSettings(settings, 4096)).toMatchObject({
      ram: 512,
      onlineMode: true,
      spawnProtection: 16,
    });
  });

  it('updates and inserts property lines without touching comments', () => {
    const content = '# Minecraft server\nmax-players=20\nmotd=Old';

    expect(upsertPropertyLine(content, 'motd', 'New')).toContain('motd=New');
    expect(upsertPropertyLine(content, 'online-mode', 'false')).toContain(
      '\nonline-mode=false',
    );
  });

  it('reads integer properties from server.properties content', () => {
    const content = 'max-players=20\r\nview-distance=10\r\nmotd=Hello';

    expect(readIntPropertyFromContent(content, 'view-distance')).toBe(10);
    expect(readIntPropertyFromContent(content, 'motd')).toBeNull();
    expect(readIntPropertyFromContent(content, 'missing')).toBeNull();
  });

  it('builds avatar urls with a fallback skin', () => {
    expect(getAvatarUrl('abc-123')).toContain('/abc-123?');
    expect(getAvatarUrl()).toContain('8667ba71-b85a-4004-af54-457a9734eed7');
  });
});
