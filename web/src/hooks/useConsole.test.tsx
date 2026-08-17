import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { useConsole } from './useConsole';

const { mockUseAuth } = vi.hoisted(() => ({
  mockUseAuth: vi.fn(),
}));

vi.mock('../context/AuthContext', () => ({
  useAuth: mockUseAuth,
}));

vi.mock('../services/api', () => ({
  WS_BASE_URL: 'ws://localhost:23008',
}));

type CloseCall = {
  code?: number;
  reason?: string;
};

class FakeWebSocket {
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSING = 2;
  static readonly CLOSED = 3;
  static instances: FakeWebSocket[] = [];

  readonly url: string;
  readonly closeCalls: CloseCall[] = [];
  readyState = FakeWebSocket.CONNECTING;
  onopen: (() => void) | null = null;
  onmessage: ((event: MessageEvent<string>) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;

  constructor(url: string) {
    this.url = url;
    FakeWebSocket.instances.push(this);
  }

  send(_data: string) {
    void _data;
  }

  close(code?: number, reason?: string) {
    this.readyState = FakeWebSocket.CLOSED;
    this.closeCalls.push({ code, reason });
  }

  open() {
    this.readyState = FakeWebSocket.OPEN;
    this.onopen?.();
  }

  emitMessage(data: string) {
    this.onmessage?.({ data } as MessageEvent<string>);
  }

  emitClose() {
    this.readyState = FakeWebSocket.CLOSED;
    this.onclose?.();
  }
}

describe('useConsole', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.stubGlobal('WebSocket', FakeWebSocket);
    FakeWebSocket.instances = [];
    mockUseAuth.mockReturnValue({ token: 'test-token' });
  });

  afterEach(() => {
    vi.clearAllTimers();
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it('does not reconnect a stale StrictMode socket', () => {
    const { unmount } = renderHook(() => useConsole('server-1'), {
      reactStrictMode: true,
    });

    expect(FakeWebSocket.instances).toHaveLength(2);
    const staleSocket = FakeWebSocket.instances[0];
    const activeSocket = FakeWebSocket.instances[1];

    expect(staleSocket).toBeDefined();
    expect(activeSocket).toBeDefined();

    act(() => {
      staleSocket?.open();
      staleSocket?.emitClose();
      vi.advanceTimersByTime(1000);
    });

    expect(FakeWebSocket.instances).toHaveLength(2);

    act(() => {
      activeSocket?.open();
    });
    unmount();
  });

  it('ignores messages from stale sockets and preserves repeated active lines', () => {
    const { result } = renderHook(() => useConsole('server-1'), {
      reactStrictMode: true,
    });

    const staleSocket = FakeWebSocket.instances[0];
    const activeSocket = FakeWebSocket.instances[1];

    act(() => {
      vi.runOnlyPendingTimers();
      staleSocket?.open();
      staleSocket?.emitMessage('stale line');
      activeSocket?.open();
      activeSocket?.emitMessage('same line\nsame line\n');
      activeSocket?.emitMessage('same line');
    });

    expect(result.current.logs).toEqual([
      'same line',
      'same line',
      'same line',
    ]);
    expect(result.current.isConnected).toBe(true);

    act(() => {
      staleSocket?.emitClose();
    });

    expect(result.current.isConnected).toBe(true);
  });

  it('reconnects once after the active socket closes', () => {
    const { unmount } = renderHook(() => useConsole('server-1'));

    const activeSocket = FakeWebSocket.instances[0];

    act(() => {
      activeSocket?.open();
      activeSocket?.emitClose();
      vi.advanceTimersByTime(1000);
    });

    expect(FakeWebSocket.instances).toHaveLength(2);

    const reconnectedSocket = FakeWebSocket.instances[1];
    act(() => {
      reconnectedSocket?.open();
      vi.advanceTimersByTime(30000);
    });

    expect(FakeWebSocket.instances).toHaveLength(2);
    unmount();
  });

  it('cancels a pending reconnection on unmount', () => {
    const { unmount } = renderHook(() => useConsole('server-1'));

    const activeSocket = FakeWebSocket.instances[0];

    act(() => {
      activeSocket?.open();
      activeSocket?.emitClose();
    });

    unmount();

    act(() => {
      vi.advanceTimersByTime(30000);
    });

    expect(FakeWebSocket.instances).toHaveLength(1);
  });

  it('closes the active socket on unmount', () => {
    const { unmount } = renderHook(() => useConsole('server-1'));

    const activeSocket = FakeWebSocket.instances[0];

    act(() => {
      activeSocket?.open();
    });

    unmount();

    expect(activeSocket?.closeCalls).toEqual([
      { code: 1000, reason: 'Component unmounted' },
    ]);
  });

  it('closes the previous socket and ignores its messages when the server changes', () => {
    const { result, rerender, unmount } = renderHook(
      ({ serverId }: { serverId: string }) => useConsole(serverId),
      {
        initialProps: { serverId: 'server-1' },
      },
    );

    const previousSocket = FakeWebSocket.instances[0];
    act(() => {
      vi.runOnlyPendingTimers();
      previousSocket?.open();
    });

    rerender({ serverId: 'server-2' });

    const nextSocket = FakeWebSocket.instances[1];
    act(() => {
      previousSocket?.emitMessage('old line');
      nextSocket?.open();
      nextSocket?.emitMessage('new line');
    });

    expect(previousSocket?.closeCalls).toEqual([
      { code: 1000, reason: 'Component unmounted' },
    ]);
    expect(result.current.logs).toEqual(['new line']);

    unmount();
  });
});
