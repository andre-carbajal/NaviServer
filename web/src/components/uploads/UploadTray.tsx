import {
  AlertCircle,
  CheckCircle2,
  ChevronDown,
  Loader2,
  UploadCloud,
  X,
} from 'lucide-react';

import { memo, useCallback, useEffect, useRef, useState } from 'react';
import type { UIEvent } from 'react';

import { type UploadTask, useUploads } from '../../context/UploadContext';

const ROW_HEIGHT_PX = 61;
const OVERSCAN_ROWS = 6;
const DEFAULT_VIEWPORT_HEIGHT_PX = 420;
const COMPLETED_DISMISS_DELAY_MS = 5000;

const statusLabel = (task: UploadTask) => {
  switch (task.status) {
    case 'pending':
      return task.message || 'Pending';
    case 'processing':
      return 'Processing...';
    case 'completed':
      return 'Completed';
    case 'error':
      return task.error || 'Upload failed';
    case 'cancelled':
      return 'Cancelled';
    default:
      return 'Uploading...';
  }
};

interface UploadRowProps {
  task: UploadTask;
  index: number;
  total: number;
  onDismiss: (id: string) => void;
}

const UploadRow = memo(({ task, index, total, onDismiss }: UploadRowProps) => {
  const isActive = task.status === 'uploading' || task.status === 'processing';
  const isTerminal =
    task.status === 'completed' ||
    task.status === 'error' ||
    task.status === 'cancelled';

  return (
    <li
      className="tw:box-border tw:h-[61px] tw:border-t tw:border-white/10 tw:px-3 tw:py-2"
      aria-posinset={index + 1}
      aria-setsize={total}
    >
      <div className="tw:flex tw:items-start tw:gap-2">
        {task.status === 'completed' ? (
          <CheckCircle2
            className="tw:mt-0.5 tw:shrink-0 tw:text-emerald-400"
            size={16}
          />
        ) : task.status === 'error' || task.status === 'cancelled' ? (
          <AlertCircle
            className="tw:mt-0.5 tw:shrink-0 tw:text-red-400"
            size={16}
          />
        ) : (
          <Loader2
            className="tw:mt-0.5 tw:shrink-0 tw:animate-spin tw:text-primary"
            size={16}
          />
        )}
        <div className="tw:min-w-0 tw:flex-1">
          <div className="tw:truncate tw:text-sm tw:text-text-main">
            {task.name}
          </div>
          <div className="tw:text-xs tw:text-text-muted">
            {statusLabel(task)}
          </div>
          {isActive || task.status === 'pending' ? (
            <div className="tw:mt-1.5 tw:flex tw:items-center tw:gap-2">
              <div
                className="tw:h-1 tw:min-w-0 tw:flex-1 tw:overflow-hidden tw:rounded tw:bg-white/10"
                role="progressbar"
                aria-label={`${task.name} upload progress`}
                aria-valuemin={0}
                aria-valuemax={100}
                aria-valuenow={task.progress}
              >
                <div
                  className="tw:h-full tw:rounded tw:bg-primary tw:transition-[width] tw:duration-200"
                  style={{ width: `${task.progress}%` }}
                />
              </div>
              <span className="tw:shrink-0 tw:text-xs tw:text-text-muted">
                {task.progress}%
              </span>
            </div>
          ) : null}
        </div>
        {isTerminal ? (
          <button
            type="button"
            className="tw:flex tw:h-7 tw:w-7 tw:shrink-0 tw:cursor-pointer tw:items-center tw:justify-center tw:rounded tw:border-0 tw:bg-transparent tw:text-text-muted tw:hover:bg-white/10 tw:hover:text-white"
            aria-label={`Dismiss ${task.name}`}
            onClick={() => onDismiss(task.id)}
          >
            <X size={15} />
          </button>
        ) : null}
      </div>
    </li>
  );
});

UploadRow.displayName = 'UploadRow';

interface VirtualUploadListProps {
  uploads: UploadTask[];
  onDismiss: (id: string) => void;
}

const VirtualUploadList = ({ uploads, onDismiss }: VirtualUploadListProps) => {
  const listRef = useRef<HTMLUListElement>(null);
  const [scrollTop, setScrollTop] = useState(0);
  const [viewportHeight, setViewportHeight] = useState(
    DEFAULT_VIEWPORT_HEIGHT_PX,
  );

  useEffect(() => {
    const list = listRef.current;
    if (!list) return;

    const updateViewportHeight = () => {
      const nextHeight = list.clientHeight || DEFAULT_VIEWPORT_HEIGHT_PX;
      setViewportHeight((current) =>
        current === nextHeight ? current : nextHeight,
      );
    };
    updateViewportHeight();
    if (typeof ResizeObserver === 'undefined') return;

    const observer = new ResizeObserver(updateViewportHeight);
    observer.observe(list);
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    const list = listRef.current;
    if (!list) return;
    const maxScrollTop = Math.max(0, list.scrollHeight - list.clientHeight);
    if (scrollTop > maxScrollTop) {
      list.scrollTop = maxScrollTop;
      setScrollTop(maxScrollTop);
    }
  }, [scrollTop, uploads.length]);

  const handleScroll = useCallback((event: UIEvent<HTMLUListElement>) => {
    setScrollTop(event.currentTarget.scrollTop);
  }, []);

  const startIndex = Math.max(
    0,
    Math.floor(scrollTop / ROW_HEIGHT_PX) - OVERSCAN_ROWS,
  );
  const endIndex = Math.min(
    uploads.length,
    Math.ceil((scrollTop + viewportHeight) / ROW_HEIGHT_PX) + OVERSCAN_ROWS,
  );
  const visibleUploads = uploads.slice(startIndex, endIndex);

  return (
    <ul
      ref={listRef}
      className="tw:m-0 tw:max-h-[min(420px,60vh)] tw:list-none tw:overflow-y-auto tw:p-0"
      onScroll={handleScroll}
    >
      <li
        aria-hidden="true"
        role="presentation"
        style={{ height: startIndex * ROW_HEIGHT_PX }}
      />
      {visibleUploads.map((task, offset) => (
        <UploadRow
          key={task.id}
          task={task}
          index={startIndex + offset}
          total={uploads.length}
          onDismiss={onDismiss}
        />
      ))}
      <li
        aria-hidden="true"
        role="presentation"
        style={{ height: (uploads.length - endIndex) * ROW_HEIGHT_PX }}
      />
    </ul>
  );
};

const UploadTray = () => {
  const {
    uploads,
    aggregateProgress,
    activeCount,
    cancelAllUploads,
    dismissUpload,
  } = useUploads();
  const [expanded, setExpanded] = useState(false);
  const previousUploadCount = useRef(uploads.length);
  const completedDismissTimers = useRef<Map<string, number>>(new Map());
  const handleDismiss = useCallback(
    (id: string) => {
      void dismissUpload(id);
    },
    [dismissUpload],
  );

  useEffect(() => {
    if (uploads.length > previousUploadCount.current) {
      setExpanded(true);
    }
    previousUploadCount.current = uploads.length;
  }, [uploads.length]);

  useEffect(() => {
    const completedIds = new Set(
      uploads
        .filter((task) => task.status === 'completed')
        .map((task) => task.id),
    );

    uploads.forEach((task) => {
      if (
        task.status !== 'completed' ||
        completedDismissTimers.current.has(task.id)
      ) {
        return;
      }
      completedDismissTimers.current.set(
        task.id,
        window.setTimeout(() => {
          completedDismissTimers.current.delete(task.id);
          void dismissUpload(task.id);
        }, COMPLETED_DISMISS_DELAY_MS),
      );
    });

    completedDismissTimers.current.forEach((timer, id) => {
      if (!completedIds.has(id)) {
        window.clearTimeout(timer);
        completedDismissTimers.current.delete(id);
      }
    });
  }, [dismissUpload, uploads]);

  useEffect(() => {
    const timers = completedDismissTimers.current;
    return () => {
      timers.forEach((timer) => window.clearTimeout(timer));
      timers.clear();
    };
  }, []);

  if (uploads.length === 0) return null;

  return (
    <section
      className="tw:fixed tw:right-5 tw:bottom-5 tw:z-[1200] tw:w-[min(380px,calc(100vw-24px))] tw:overflow-hidden tw:rounded-xl tw:border tw:border-border tw:bg-bg-card tw:shadow-[0_18px_45px_rgba(0,0,0,0.35)] tw:max-[640px]:right-3 tw:max-[640px]:bottom-3"
      aria-label="Upload status"
    >
      <div className="tw:flex tw:items-center tw:border-b tw:border-white/10">
        <button
          type="button"
          className="tw:flex tw:min-w-0 tw:flex-1 tw:cursor-pointer tw:items-center tw:gap-2.5 tw:border-0 tw:bg-transparent tw:px-3.5 tw:py-3 tw:text-left tw:text-text-main tw:hover:bg-white/[0.04]"
          aria-expanded={expanded}
          onClick={() => setExpanded((value) => !value)}
        >
          {activeCount > 0 ? (
            <Loader2
              className="tw:shrink-0 tw:animate-spin tw:text-primary"
              size={18}
            />
          ) : (
            <UploadCloud className="tw:shrink-0 tw:text-primary" size={18} />
          )}
          <span className="tw:min-w-0 tw:flex-1 tw:text-sm">
            {activeCount > 0
              ? `${activeCount} upload${activeCount === 1 ? '' : 's'} in progress`
              : 'Uploads finished'}
            <span className="tw:ml-1 tw:text-text-muted">
              ({aggregateProgress}%)
            </span>
          </span>
          <ChevronDown
            className={`tw:shrink-0 tw:transition-transform ${expanded ? 'tw:rotate-180' : ''}`}
            size={16}
          />
        </button>
        {activeCount > 0 && (
          <button
            type="button"
            className="tw:mr-2 tw:flex tw:h-8 tw:shrink-0 tw:cursor-pointer tw:items-center tw:justify-center tw:gap-1 tw:rounded-md tw:border tw:border-danger/40 tw:bg-danger/10 tw:px-2 tw:text-xs tw:font-medium tw:text-danger tw:hover:bg-danger/20 tw:hover:text-white"
            aria-label="Cancel all uploads"
            title="Cancel all uploads"
            onClick={() => void cancelAllUploads()}
          >
            <X size={16} />
            <span>Cancel</span>
          </button>
        )}
      </div>
      {!expanded && (
        <div className="tw:h-1 tw:bg-white/10">
          <div
            className="tw:h-full tw:bg-primary tw:transition-[width] tw:duration-200"
            style={{ width: `${aggregateProgress}%` }}
          />
        </div>
      )}
      {expanded && (
        <VirtualUploadList uploads={uploads} onDismiss={handleDismiss} />
      )}
    </section>
  );
};

export default UploadTray;
