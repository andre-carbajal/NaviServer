import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { api } from '../services/api';
import { useGlobalSettings } from './useGlobalSettings';

vi.mock('../services/api', () => ({
  api: {
    getPortRange: vi.fn(),
    getLogBufferSize: vi.fn(),
    getPublicIP: vi.fn(),
    getNetworkInterfaces: vi.fn(),
    getCurseForgeKeyStatus: vi.fn(),
  },
}));

describe('useGlobalSettings', () => {
  beforeEach(() => {
    vi.mocked(api.getPortRange).mockResolvedValue({
      data: { start: 25565, end: 25575 },
    } as never);
    vi.mocked(api.getLogBufferSize).mockResolvedValue({
      data: { log_buffer_size: 1000 },
    } as never);
    vi.mocked(api.getPublicIP).mockResolvedValue({
      data: { public_ip: 'localhost' },
    } as never);
    vi.mocked(api.getNetworkInterfaces).mockResolvedValue({
      data: { interfaces: [] },
    } as never);
    vi.mocked(api.getCurseForgeKeyStatus).mockResolvedValue({
      data: {
        hasCustomKey: false,
        hasEmbeddedKey: false,
        effectiveSource: 'none',
      },
    } as never);
  });

  it('validates negative console buffer values', async () => {
    const { result } = renderHook(() => useGlobalSettings(vi.fn(), vi.fn()));
    await waitFor(() => expect(result.current.loading).toBe(false));
    act(() => result.current.changeLogBuffer('-1'));
    expect(result.current.logBufferError).toBe('The value cannot be negative');
  });
});
