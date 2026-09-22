import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { UploadTask } from '../../context/UploadContext';
import UploadTray from './UploadTray';

const { mockUseUploads } = vi.hoisted(() => ({
  mockUseUploads: vi.fn(),
}));

vi.mock('../../context/UploadContext', () => ({
  useUploads: mockUseUploads,
}));

const makeTask = (overrides: Partial<UploadTask> = {}): UploadTask => ({
  id: 'upload-1',
  name: 'file.txt',
  size: 100,
  type: 'text/plain',
  target: { kind: 'server-file', serverId: 'server-1' },
  status: 'pending',
  progress: 42,
  uploadedBytes: 42,
  totalBytes: 100,
  createdAt: 1,
  ...overrides,
});

const renderExpandedTray = (uploads: UploadTask[]) => {
  const cancelAllUploads = vi.fn();
  const dismissUpload = vi.fn();
  mockUseUploads.mockReturnValue({
    uploads,
    aggregateProgress: Math.round(
      uploads.reduce((total, upload) => total + upload.progress, 0) /
        Math.max(uploads.length, 1),
    ),
    activeCount: uploads.filter(
      (upload) =>
        upload.status === 'pending' ||
        upload.status === 'uploading' ||
        upload.status === 'processing',
    ).length,
    cancelAllUploads,
    dismissUpload,
  });
  render(<UploadTray />);
  const toggle = screen.getByRole('button', {
    name: /uploads? in progress|uploads finished/i,
  });
  if (toggle.getAttribute('aria-expanded') === 'false') {
    fireEvent.click(toggle);
  }
  return { cancelAllUploads, dismissUpload };
};

describe('UploadTray item controls', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('shows one global cancellation control for pending uploads', () => {
    const { cancelAllUploads } = renderExpandedTray([makeTask()]);

    expect(screen.getByText('42%')).toBeTruthy();
    expect(
      screen.queryByRole('button', { name: 'Cancel file.txt' }),
    ).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Cancel all uploads' }));

    expect(cancelAllUploads).toHaveBeenCalledOnce();
  });

  it('shows global cancellation for an active upload', () => {
    const { cancelAllUploads } = renderExpandedTray([
      makeTask({ id: 'active-1', name: 'active.txt', status: 'uploading' }),
    ]);

    expect(
      screen.queryByRole('button', { name: 'Cancel active.txt' }),
    ).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Cancel all uploads' }));
    expect(cancelAllUploads).toHaveBeenCalledOnce();
  });

  it('keeps dismiss available for terminal uploads', () => {
    const { dismissUpload } = renderExpandedTray([
      makeTask({
        id: 'done-1',
        name: 'done.txt',
        status: 'completed',
        progress: 100,
        uploadedBytes: 100,
      }),
    ]);

    fireEvent.click(screen.getByRole('button', { name: 'Dismiss done.txt' }));

    expect(dismissUpload).toHaveBeenCalledWith('done-1');
    expect(
      screen.queryByRole('button', { name: 'Cancel done.txt' }),
    ).toBeNull();
    expect(
      screen.queryByRole('button', { name: 'Cancel all uploads' }),
    ).toBeNull();
  });

  it('automatically dismisses completed uploads after a short delay', () => {
    vi.useFakeTimers();
    try {
      const { dismissUpload } = renderExpandedTray([
        makeTask({
          id: 'done-1',
          name: 'done.txt',
          status: 'completed',
          progress: 100,
          uploadedBytes: 100,
        }),
      ]);

      act(() => {
        vi.advanceTimersByTime(4999);
      });
      expect(dismissUpload).not.toHaveBeenCalled();

      act(() => {
        vi.advanceTimersByTime(1);
      });
      expect(dismissUpload).toHaveBeenCalledWith('done-1');
    } finally {
      vi.useRealTimers();
    }
  });

  it('opens automatically when uploads are added', () => {
    const cancelAllUploads = vi.fn();
    const dismissUpload = vi.fn();
    mockUseUploads.mockReturnValue({
      uploads: [],
      aggregateProgress: 0,
      activeCount: 0,
      cancelAllUploads,
      dismissUpload,
    });
    const { rerender } = render(<UploadTray />);

    mockUseUploads.mockReturnValue({
      uploads: [makeTask()],
      aggregateProgress: 42,
      activeCount: 1,
      cancelAllUploads,
      dismissUpload,
    });
    rerender(<UploadTray />);

    expect(
      screen
        .getByRole('button', { name: /uploads? in progress/i })
        .getAttribute('aria-expanded'),
    ).toBe('true');
  });

  it('virtualizes large upload lists and updates the visible window on scroll', () => {
    const uploads = Array.from({ length: 1516 }, (_, index) =>
      makeTask({
        id: `upload-${index}`,
        name: `file-${index}.txt`,
      }),
    );

    renderExpandedTray(uploads);

    expect(screen.getAllByRole('progressbar')).toHaveLength(13);
    expect(screen.getByText('file-0.txt')).toBeTruthy();
    expect(screen.queryByText('file-100.txt')).toBeNull();

    const list = screen.getByRole('list');
    Object.defineProperties(list, {
      clientHeight: { configurable: true, value: 420 },
      scrollHeight: { configurable: true, value: 1516 * 61 },
      scrollTop: { configurable: true, value: 100 * 61, writable: true },
    });
    fireEvent.scroll(list);

    expect(screen.getByText('file-100.txt')).toBeTruthy();
    expect(screen.queryByText('file-0.txt')).toBeNull();
    expect(screen.getAllByRole('progressbar')).toHaveLength(19);
  });
});
