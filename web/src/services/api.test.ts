import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const { mockAxiosInstance } = vi.hoisted(() => ({
  mockAxiosInstance: {
    interceptors: {
      response: {
        use: vi.fn(),
      },
    },
    post: vi.fn(),
    put: vi.fn(),
  },
}));

vi.mock('axios', () => ({
  default: {
    create: vi.fn(() => mockAxiosInstance),
  },
}));

const getNetworkHandlers = async () => {
  vi.resetModules();
  mockAxiosInstance.interceptors.response.use.mockReset();
  mockAxiosInstance.post.mockReset();
  mockAxiosInstance.put.mockReset();
  await import('./api');

  const [onFulfilled, onRejected] =
    mockAxiosInstance.interceptors.response.use.mock.calls[0];
  const apiModule = await import('./api');
  return { onFulfilled, onRejected, api: apiModule.api };
};

const captureNetworkEvents = () => {
  const events: string[] = [];
  const onError = () => events.push('error');
  const onRecovered = () => events.push('recovered');

  window.addEventListener('network-error', onError);
  window.addEventListener('network-recovered', onRecovered);

  return {
    events,
    cleanup: () => {
      window.removeEventListener('network-error', onError);
      window.removeEventListener('network-recovered', onRecovered);
    },
  };
};

const rejectNetworkRequest = (
  onRejected: (error: unknown) => Promise<unknown>,
) => onRejected({ code: 'ERR_NETWORK' }).catch(() => undefined);

describe('API network recovery notifications', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(0);
  });

  afterEach(() => {
    vi.clearAllTimers();
    vi.useRealTimers();
  });

  it('ignores an intermittent failure recovered before the grace period', async () => {
    const { onFulfilled, onRejected } = await getNetworkHandlers();
    const networkEvents = captureNetworkEvents();

    await rejectNetworkRequest(onRejected);
    vi.advanceTimersByTime(7999);
    onFulfilled({ data: {} });
    vi.advanceTimersByTime(1);

    expect(networkEvents.events).toEqual([]);
    networkEvents.cleanup();
  });

  it('announces a sustained outage and its first recovery only once', async () => {
    const { onFulfilled, onRejected } = await getNetworkHandlers();
    const networkEvents = captureNetworkEvents();

    await rejectNetworkRequest(onRejected);
    vi.advanceTimersByTime(8000);
    expect(networkEvents.events).toEqual(['error']);

    onFulfilled({ data: {} });
    onFulfilled({ data: {} });
    expect(networkEvents.events).toEqual(['error', 'recovered']);

    networkEvents.cleanup();
  });

  it('bounds upload requests so a stalled chunk cannot block the queue forever', async () => {
    const { api } = await getNetworkHandlers();
    mockAxiosInstance.put.mockResolvedValue({ data: {} });
    mockAxiosInstance.post.mockResolvedValue({ data: {} });

    await api.uploadChunk('upload-1', 0, 3, 3, new Blob(['abc']));
    await api.completeUpload('upload-1');

    expect(mockAxiosInstance.put).toHaveBeenCalledWith(
      '/uploads/upload-1/chunk',
      expect.any(Blob),
      expect.objectContaining({
        timeout: 60000,
        headers: expect.objectContaining({
          'Content-Type': 'application/octet-stream',
          'Content-Range': 'bytes 0-2/3',
        }),
      }),
    );
    expect(mockAxiosInstance.post).toHaveBeenCalledWith(
      '/uploads/upload-1/complete',
      null,
      expect.objectContaining({ timeout: 30000 }),
    );
  });
});
