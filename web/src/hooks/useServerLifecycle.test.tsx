import { act, renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { api } from '../services/api';
import type { Server } from '../types';
import { useServerLifecycle } from './useServerLifecycle';

vi.mock('../services/api', () => ({
  api: {
    startServer: vi.fn(),
    stopServer: vi.fn(),
    restartServer: vi.fn(),
    killServer: vi.fn(),
  },
}));

describe('useServerLifecycle', () => {
  it('updates the optimistic status after starting', async () => {
    const server = { id: 'one', status: 'STOPPED' } as Server;
    const setServer = vi.fn();
    const { result } = renderHook(() =>
      useServerLifecycle({
        server,
        refresh: vi.fn().mockResolvedValue(true),
        setServer,
        showAlert: vi.fn(),
        closeMenu: vi.fn(),
      }),
    );

    await act(() => result.current.start());
    expect(api.startServer).toHaveBeenCalledWith('one');
    const updater = setServer.mock.calls[0][0] as (value: Server) => Server;
    expect(updater(server).status).toBe('STARTING');
  });
});
