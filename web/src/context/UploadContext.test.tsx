import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import React from 'react';

import { UploadProvider, useUploads } from './UploadContext';

const { mockApi } = vi.hoisted(() => ({
  mockApi: {
    createUpload: vi.fn(),
    getUpload: vi.fn(),
    uploadChunk: vi.fn(),
    completeUpload: vi.fn(),
    cancelUpload: vi.fn(),
    cancelUploadKeepAlive: vi.fn(),
  },
}));

vi.mock('../services/api', () => ({
  WS_BASE_URL: 'ws://localhost',
  api: mockApi,
}));

vi.mock('./AuthContext', () => ({
  useAuth: () => ({ isAuthenticated: true }),
}));

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <UploadProvider>{children}</UploadProvider>
);

const createServerResponse = (
  id: string,
  filename: string,
  status = 'pending',
) => ({
  data: {
    id,
    clientId: `client-${id}`,
    kind: 'server-file',
    filename,
    status,
    receivedBytes: 0,
    totalBytes: 3,
    progress: 0,
  },
});

describe('UploadContext', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.createUpload.mockImplementation(
      ({ filename }: { filename: string }) =>
        Promise.resolve(createServerResponse(`server-${filename}`, filename)),
    );
    mockApi.uploadChunk.mockImplementation(
      (_id: string, _start: number, _end: number, totalBytes: number) =>
        Promise.resolve({
          data: {
            ...createServerResponse('chunk', 'file').data,
            status: 'ready',
            receivedBytes: totalBytes,
            totalBytes,
            progress: 100,
          },
        }),
    );
    mockApi.completeUpload.mockResolvedValue({
      data: {
        ...createServerResponse('complete', 'file').data,
        status: 'processing',
        receivedBytes: 3,
        progress: 100,
      },
    });
    mockApi.cancelUpload.mockResolvedValue({});
    mockApi.getUpload.mockResolvedValue({
      data: {
        ...createServerResponse('complete', 'file').data,
        status: 'completed',
        receivedBytes: 3,
        progress: 100,
      },
    });
  });

  it('processes queued uploads sequentially', async () => {
    const order: string[] = [];
    mockApi.createUpload.mockImplementation(
      ({ filename }: { filename: string }) => {
        order.push(`create:${filename}`);
        return Promise.resolve(
          createServerResponse(`server-${filename}`, filename),
        );
      },
    );
    mockApi.uploadChunk.mockImplementation(
      (id: string, _start: number, _end: number, totalBytes: number) => {
        order.push(`chunk:${id}`);
        return Promise.resolve({
          data: {
            ...createServerResponse(id, id).data,
            status: 'ready',
            receivedBytes: totalBytes,
            totalBytes,
            progress: 100,
          },
        });
      },
    );

    const { result } = renderHook(() => useUploads(), { wrapper });
    await act(async () => {
      await result.current.enqueueUpload({
        file: new File(['one'], 'one.txt'),
        name: 'one.txt',
        target: { kind: 'server-file', serverId: 'server-1' },
      });
      await result.current.enqueueUpload({
        file: new File(['two'], 'two.txt'),
        name: 'two.txt',
        target: { kind: 'server-file', serverId: 'server-1' },
      });
    });

    await waitFor(() =>
      expect(
        result.current.uploads.every((upload) => upload.status === 'completed'),
      ).toBe(true),
    );
    expect(result.current.aggregateProgress).toBe(100);
    expect(result.current.activeCount).toBe(0);
    expect(order).toEqual([
      'create:one.txt',
      'chunk:server-one.txt',
      'create:two.txt',
      'chunk:server-two.txt',
    ]);
  });

  it('cancels active and pending uploads on pagehide', async () => {
    let rejectChunk: (error: Error) => void = () => undefined;
    mockApi.uploadChunk.mockImplementation(
      () =>
        new Promise((_resolve, reject) => {
          rejectChunk = reject;
        }),
    );

    const { result } = renderHook(() => useUploads(), { wrapper });
    await act(async () => {
      await result.current.enqueueUpload({
        file: new File(['one'], 'one.txt'),
        name: 'one.txt',
        target: { kind: 'server-file', serverId: 'server-1' },
      });
    });
    await waitFor(() => expect(mockApi.uploadChunk).toHaveBeenCalled());

    await act(async () => {
      await result.current.enqueueUpload({
        file: new File(['two'], 'two.txt'),
        name: 'two.txt',
        target: { kind: 'server-file', serverId: 'server-1' },
      });
    });
    expect(result.current.uploads).toHaveLength(2);

    act(() => window.dispatchEvent(new Event('pagehide')));
    expect(result.current.uploads).toEqual([]);
    expect(mockApi.cancelUploadKeepAlive).toHaveBeenCalledWith(
      'server-one.txt',
    );
    rejectChunk(new Error('cancelled'));
  });

  it('cancels the active queue while preserving terminal results', async () => {
    const { result } = renderHook(() => useUploads(), { wrapper });
    await act(async () => {
      await result.current.enqueueUpload({
        file: new File(['one'], 'one.txt'),
        name: 'one.txt',
        target: { kind: 'server-file', serverId: 'server-1' },
      });
    });
    await waitFor(() =>
      expect(result.current.uploads[0]?.status).toBe('completed'),
    );

    mockApi.uploadChunk.mockImplementation(() => new Promise(() => undefined));
    await act(async () => {
      await result.current.enqueueUpload({
        file: new File(['two'], 'two.txt'),
        name: 'two.txt',
        target: { kind: 'server-file', serverId: 'server-1' },
      });
    });
    await waitFor(() => expect(mockApi.uploadChunk).toHaveBeenCalled());

    await act(async () => {
      await result.current.cancelAllUploads();
    });

    expect(result.current.uploads).toHaveLength(1);
    expect(result.current.uploads[0].status).toBe('completed');
    expect(mockApi.cancelUpload).toHaveBeenCalledWith('server-two.txt');
  });

  it('enqueues a large batch with one visible state append', async () => {
    let rejectCreate: (error: Error) => void = () => undefined;
    mockApi.createUpload.mockImplementation(
      () =>
        new Promise((_resolve, reject) => {
          rejectCreate = reject;
        }),
    );
    const items = Array.from({ length: 1501 }, (_, index) => ({
      input: {
        file: new File(['x'], `file-${index}.txt`),
        name: `file-${index}.txt`,
        target: { kind: 'server-file' as const, serverId: 'server-1' },
      },
    }));

    const { result } = renderHook(() => useUploads(), { wrapper });
    let ids: string[] = [];
    await act(async () => {
      ids = await result.current.enqueueUploads(items);
    });

    expect(ids).toHaveLength(1501);
    expect(result.current.uploads).toHaveLength(1501);
    expect(result.current.aggregateProgress).toBe(0);
    expect(result.current.activeCount).toBe(1501);
    await act(async () => {
      await result.current.cancelAllUploads();
      rejectCreate(new Error('cancelled'));
    });
    expect(result.current.uploads).toEqual([]);
    expect(result.current.aggregateProgress).toBe(0);
    expect(result.current.activeCount).toBe(0);
  });

  it('throttles progress updates and flushes the final chunk immediately', async () => {
    vi.useFakeTimers();
    try {
      let resolveChunk: (response: unknown) => void = () => undefined;
      mockApi.uploadChunk.mockImplementation(
        (
          _id: string,
          _start: number,
          _end: number,
          _totalBytes: number,
          _chunk: Blob,
          onUploadProgress?: (event: { loaded: number }) => void,
        ) => {
          onUploadProgress?.({ loaded: 1 });
          onUploadProgress?.({ loaded: 2 });
          return new Promise((resolve) => {
            resolveChunk = resolve;
          });
        },
      );

      const { result } = renderHook(() => useUploads(), { wrapper });
      await act(async () => {
        await result.current.enqueueUpload({
          file: new File(['one'], 'one.txt'),
          name: 'one.txt',
          target: { kind: 'server-file', serverId: 'server-1' },
        });
        await Promise.resolve();
        await Promise.resolve();
      });

      expect(mockApi.uploadChunk).toHaveBeenCalledOnce();
      expect(result.current.uploads[0]?.progress).toBe(0);

      await act(async () => {
        await vi.advanceTimersByTimeAsync(149);
      });
      expect(result.current.uploads[0]?.progress).toBe(0);

      await act(async () => {
        await vi.advanceTimersByTimeAsync(1);
      });
      expect(result.current.uploads[0]?.progress).toBe(66);

      await act(async () => {
        resolveChunk({
          data: {
            ...createServerResponse('server-one', 'one.txt', 'ready').data,
            receivedBytes: 3,
            totalBytes: 3,
            progress: 100,
          },
        });
        await Promise.resolve();
        await Promise.resolve();
      });

      expect(result.current.uploads[0]).toMatchObject({
        status: 'completed',
        progress: 100,
      });
    } finally {
      vi.useRealTimers();
    }
  });

  it('pauses on a network error and resumes after recovery', async () => {
    mockApi.createUpload.mockRejectedValueOnce({
      isAxiosError: true,
      code: 'ERR_NETWORK',
      message: 'Network Error',
    });

    const { result } = renderHook(() => useUploads(), { wrapper });
    await act(async () => {
      await result.current.enqueueUpload({
        file: new File(['one'], 'one.txt'),
        name: 'one.txt',
        target: { kind: 'server-file', serverId: 'server-1' },
      });
      await result.current.enqueueUpload({
        file: new File(['two'], 'two.txt'),
        name: 'two.txt',
        target: { kind: 'server-file', serverId: 'server-1' },
      });
    });

    await waitFor(() =>
      expect(result.current.uploads[0]).toMatchObject({
        status: 'pending',
        message: 'Waiting for connection...',
      }),
    );
    expect(mockApi.createUpload).toHaveBeenCalledTimes(1);

    act(() => window.dispatchEvent(new Event('network-recovered')));
    await waitFor(() =>
      expect(
        result.current.uploads.every((upload) => upload.status === 'completed'),
      ).toBe(true),
    );
    expect(mockApi.createUpload).toHaveBeenCalledTimes(3);
  });

  it('automatically retries a network error after the backoff delay', async () => {
    vi.useFakeTimers();
    try {
      mockApi.createUpload.mockRejectedValueOnce({
        isAxiosError: true,
        code: 'ERR_NETWORK',
        message: 'Network Error',
      });

      const { result } = renderHook(() => useUploads(), { wrapper });
      await act(async () => {
        await result.current.enqueueUpload({
          file: new File(['one'], 'one.txt'),
          name: 'one.txt',
          target: { kind: 'server-file', serverId: 'server-1' },
        });
        await Promise.resolve();
        await Promise.resolve();
      });
      expect(mockApi.createUpload).toHaveBeenCalledTimes(1);

      await act(async () => {
        await vi.advanceTimersByTimeAsync(1000);
      });
      expect(mockApi.createUpload).toHaveBeenCalledTimes(2);
    } finally {
      vi.useRealTimers();
    }
  });

  it.each(['ECONNABORTED', 'ETIMEDOUT'])(
    'retries a %s chunk from the confirmed server offset',
    async (code) => {
      vi.useFakeTimers();
      try {
        mockApi.createUpload.mockResolvedValue({
          data: {
            ...createServerResponse('server-large', 'large.bin').data,
            totalBytes: 6,
          },
        });
        mockApi.uploadChunk.mockReset();
        mockApi.uploadChunk
          .mockRejectedValueOnce({
            isAxiosError: true,
            code,
            message: 'timeout of 60000ms exceeded',
          })
          .mockResolvedValueOnce({
            data: {
              ...createServerResponse('server-large', 'large.bin', 'ready')
                .data,
              receivedBytes: 6,
              totalBytes: 6,
              progress: 100,
            },
          });
        mockApi.getUpload.mockReset();
        mockApi.getUpload
          .mockResolvedValueOnce({
            data: {
              ...createServerResponse('server-large', 'large.bin', 'uploading')
                .data,
              receivedBytes: 3,
              totalBytes: 6,
              progress: 50,
            },
          })
          .mockResolvedValueOnce({
            data: {
              ...createServerResponse('server-large', 'large.bin', 'completed')
                .data,
              receivedBytes: 6,
              totalBytes: 6,
              progress: 100,
            },
          });

        const { result } = renderHook(() => useUploads(), { wrapper });
        await act(async () => {
          await result.current.enqueueUpload({
            file: new File(['123456'], 'large.bin'),
            name: 'large.bin',
            target: { kind: 'server-file', serverId: 'server-1' },
          });
          await Promise.resolve();
          await Promise.resolve();
        });

        expect(result.current.uploads[0]).toMatchObject({
          status: 'pending',
          message: 'Waiting for connection...',
        });

        await act(async () => {
          await vi.advanceTimersByTimeAsync(1000);
        });

        expect(mockApi.uploadChunk).toHaveBeenCalledTimes(2);
        expect(mockApi.uploadChunk.mock.calls[1].slice(0, 4)).toEqual([
          'server-large',
          3,
          6,
          6,
        ]);
        expect(result.current.uploads[0]?.status).toBe('completed');
      } finally {
        vi.useRealTimers();
      }
    },
  );
});
