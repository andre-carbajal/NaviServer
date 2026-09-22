import axios from 'axios';
import type { AxiosProgressEvent } from 'axios';
import { v4 as uuidv4 } from 'uuid';

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import type { ReactNode } from 'react';

import { WS_BASE_URL, api } from '../services/api';
import type { UploadStatusResponse, UploadTarget } from '../types';
import { useAuth } from './AuthContext';

export interface UploadTask {
  id: string;
  name: string;
  size: number;
  type: string;
  target: UploadTarget;
  status: UploadStatusResponse['status'];
  progress: number;
  uploadedBytes: number;
  totalBytes: number;
  serverUploadId?: string;
  message?: string;
  error?: string;
  createdAt: number;
}

interface UploadRecord extends UploadTask {
  file?: Blob;
}

interface EnqueueUploadInput {
  file: File | Blob;
  name: string;
  target: UploadTarget;
  contentType?: string;
}

interface UploadCallbacks {
  onComplete?: () => void;
  onError?: (message: string) => void;
}

interface EnqueueUploadItem {
  input: EnqueueUploadInput;
  callbacks?: UploadCallbacks;
}

interface UploadContextValue {
  uploads: UploadTask[];
  aggregateProgress: number;
  activeCount: number;
  enqueueUpload: (
    input: EnqueueUploadInput,
    callbacks?: UploadCallbacks,
  ) => Promise<string>;
  enqueueUploads: (items: EnqueueUploadItem[]) => Promise<string[]>;
  cancelAllUploads: () => Promise<void>;
  dismissUpload: (id: string) => Promise<void>;
}

const UploadContext = createContext<UploadContextValue | undefined>(undefined);
const CHUNK_SIZE = 5 * 1024 * 1024;
const POLL_INTERVAL_MS = 1500;
const PROGRESS_PUBLISH_INTERVAL_MS = 150;
const NETWORK_RETRY_DELAYS_MS = [1000, 2000, 4000, 8000, 10000];

const terminalStatuses = new Set<UploadTask['status']>([
  'completed',
  'error',
  'cancelled',
]);

const isActiveStatus = (status: UploadTask['status']) =>
  status === 'pending' || status === 'uploading' || status === 'processing';

const toTask = (record: UploadRecord): UploadTask => {
  const task = { ...record };
  delete task.file;
  return task;
};

const getErrorMessage = (error: unknown) => {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data;
    if (typeof data === 'string' && data) return data;
    if (data && typeof data === 'object' && 'message' in data) {
      return String(data.message);
    }
    return error.message;
  }
  return error instanceof Error ? error.message : 'Upload failed';
};

const isAbortError = (error: unknown) =>
  axios.isAxiosError(error) && error.code === 'ERR_CANCELED';

const isRetryableUploadError = (error: unknown) =>
  axios.isAxiosError(error) &&
  ['ERR_NETWORK', 'ECONNABORTED', 'ETIMEDOUT'].includes(error.code ?? '');

const wait = (duration: number, signal: AbortSignal) =>
  new Promise<void>((resolve, reject) => {
    let timer = 0;
    const cleanup = () => signal.removeEventListener('abort', abort);
    const abort = () => {
      window.clearTimeout(timer);
      cleanup();
      reject(new DOMException('Upload cancelled', 'AbortError'));
    };
    if (signal.aborted) {
      abort();
      return;
    }
    signal.addEventListener('abort', abort, { once: true });
    timer = window.setTimeout(() => {
      cleanup();
      resolve();
    }, duration);
  });

export const UploadProvider = ({ children }: { children: ReactNode }) => {
  const { isAuthenticated } = useAuth();
  const [uploads, setUploads] = useState<UploadTask[]>([]);
  const [aggregateProgress, setAggregateProgress] = useState(0);
  const [activeCount, setActiveCount] = useState(0);
  const recordsRef = useRef<Map<string, UploadRecord>>(new Map());
  const callbacksRef = useRef<Map<string, UploadCallbacks>>(new Map());
  const socketsRef = useRef<Map<string, WebSocket>>(new Map());
  const controllersRef = useRef<Map<string, AbortController>>(new Map());
  const retryTimerRef = useRef<number | null>(null);
  const retryResolverRef = useRef<(() => void) | null>(null);
  const retryAttemptRef = useRef(0);
  const runningRef = useRef(false);
  const authenticatedRef = useRef(isAuthenticated);
  const unloadingRef = useRef(false);
  const progressTotalRef = useRef(0);
  const activeCountRef = useRef(0);
  const progressTimersRef = useRef<Map<string, number>>(new Map());
  const pendingProgressRef = useRef<
    Map<string, Pick<UploadRecord, 'uploadedBytes' | 'progress'>>
  >(new Map());

  const wakeNetworkRetry = useCallback(() => {
    if (retryTimerRef.current !== null) {
      window.clearTimeout(retryTimerRef.current);
      retryTimerRef.current = null;
    }
    const resolveRetry = retryResolverRef.current;
    retryResolverRef.current = null;
    resolveRetry?.();
  }, []);

  const waitForNetworkRetry = useCallback((duration: number) => {
    return new Promise<void>((resolve) => {
      retryResolverRef.current = () => {
        retryResolverRef.current = null;
        retryTimerRef.current = null;
        resolve();
      };
      retryTimerRef.current = window.setTimeout(() => {
        retryResolverRef.current?.();
      }, duration);
    });
  }, []);

  const updateDerivedStats = useCallback(
    (previous: UploadRecord | undefined, next: UploadRecord | undefined) => {
      progressTotalRef.current +=
        (next?.progress ?? 0) - (previous?.progress ?? 0);
      activeCountRef.current +=
        (next && isActiveStatus(next.status) ? 1 : 0) -
        (previous && isActiveStatus(previous.status) ? 1 : 0);

      const count = recordsRef.current.size;
      const nextAggregateProgress = count
        ? Math.round(progressTotalRef.current / count)
        : 0;
      setAggregateProgress((current) =>
        current === nextAggregateProgress ? current : nextAggregateProgress,
      );
      setActiveCount((current) =>
        current === activeCountRef.current ? current : activeCountRef.current,
      );
    },
    [],
  );

  const setRecord = useCallback(
    (id: string, patch: Partial<UploadRecord>) => {
      const current = recordsRef.current.get(id);
      if (!current) return;
      const hasChanges = Object.entries(patch).some(
        ([key, value]) => current[key as keyof UploadRecord] !== value,
      );
      if (!hasChanges) return;

      const next = { ...current, ...patch };
      recordsRef.current.set(id, next);
      updateDerivedStats(current, next);
      const hasVisibleChanges =
        current.status !== next.status ||
        current.progress !== next.progress ||
        current.message !== next.message ||
        current.error !== next.error;
      if (!hasVisibleChanges) return;

      setUploads((previous) => {
        const index = previous.findIndex((task) => task.id === id);
        if (index === -1) return previous;
        const nextUploads = [...previous];
        nextUploads[index] = toTask(next);
        return nextUploads;
      });
    },
    [updateDerivedStats],
  );

  const clearProgressForTask = useCallback((id: string) => {
    const timer = progressTimersRef.current.get(id);
    if (timer !== undefined) {
      window.clearTimeout(timer);
      progressTimersRef.current.delete(id);
    }
    pendingProgressRef.current.delete(id);
  }, []);

  const clearProgressTimers = useCallback(() => {
    progressTimersRef.current.forEach((timer) => window.clearTimeout(timer));
    progressTimersRef.current.clear();
    pendingProgressRef.current.clear();
  }, []);

  const flushProgress = useCallback(
    (id: string) => {
      const pending = pendingProgressRef.current.get(id);
      if (!pending) return;
      clearProgressForTask(id);
      setRecord(id, pending);
    },
    [clearProgressForTask, setRecord],
  );

  const publishProgress = useCallback(
    (
      id: string,
      progress: Pick<UploadRecord, 'uploadedBytes' | 'progress'>,
    ) => {
      pendingProgressRef.current.set(id, progress);
      if (progressTimersRef.current.has(id)) return;
      progressTimersRef.current.set(
        id,
        window.setTimeout(() => {
          progressTimersRef.current.delete(id);
          flushProgress(id);
        }, PROGRESS_PUBLISH_INTERVAL_MS),
      );
    },
    [flushProgress],
  );

  const finishTask = useCallback(
    (id: string, status: 'completed' | 'error', message?: string) => {
      const current = recordsRef.current.get(id);
      if (!current) return;
      clearProgressForTask(id);
      setRecord(id, {
        status,
        progress: status === 'completed' ? 100 : current.progress,
        message: status === 'completed' ? 'Upload completed' : 'Upload failed',
        error: status === 'error' ? message || 'Upload failed' : undefined,
        file: undefined,
      });
      const callbacks = callbacksRef.current.get(id);
      callbacksRef.current.delete(id);
      if (status === 'completed') callbacks?.onComplete?.();
      if (status === 'error') callbacks?.onError?.(message || 'Upload failed');
    },
    [clearProgressForTask, setRecord],
  );

  const updateFromServer = useCallback(
    (id: string, response: UploadStatusResponse) => {
      setRecord(id, {
        status: response.status === 'ready' ? 'uploading' : response.status,
        uploadedBytes: response.receivedBytes,
        progress: response.progress,
        message: response.message,
        error: response.error,
      });
    },
    [setRecord],
  );

  const waitForCompletion = useCallback(
    async (taskId: string, serverUploadId: string, signal: AbortSignal) => {
      let socket: WebSocket | null = null;
      try {
        socket = new WebSocket(`${WS_BASE_URL}/ws/progress/${serverUploadId}`);
        socketsRef.current.set(taskId, socket);
        socket.onmessage = (event) => {
          try {
            const response = JSON.parse(event.data) as UploadStatusResponse;
            if (response.id === serverUploadId) {
              updateFromServer(taskId, response);
            }
          } catch {
            // Polling remains the source of truth.
          }
        };
      } catch {
        // Polling still works when the WebSocket cannot be opened.
      }

      try {
        while (
          authenticatedRef.current &&
          !signal.aborted &&
          !unloadingRef.current
        ) {
          const response = (await api.getUpload(serverUploadId, signal)).data;
          updateFromServer(taskId, response);
          if (response.status === 'completed') {
            finishTask(taskId, 'completed');
            return;
          }
          if (response.status === 'error' || response.status === 'cancelled') {
            finishTask(taskId, 'error', response.error || response.message);
            return;
          }
          await wait(POLL_INTERVAL_MS, signal);
        }
      } finally {
        socket?.close();
        socketsRef.current.delete(taskId);
      }
    },
    [finishTask, updateFromServer],
  );

  const runTask = useCallback(
    async (id: string) => {
      const record = recordsRef.current.get(id);
      if (!record?.file) {
        finishTask(id, 'error', 'Upload file is no longer available');
        return;
      }

      const controller = new AbortController();
      controllersRef.current.set(id, controller);
      const { signal } = controller;
      let serverStatus: UploadStatusResponse | undefined;

      try {
        if (record.serverUploadId) {
          try {
            serverStatus = (await api.getUpload(record.serverUploadId, signal))
              .data;
          } catch (error) {
            if (!axios.isAxiosError(error) || error.response?.status !== 404) {
              throw error;
            }
            setRecord(id, { serverUploadId: undefined, uploadedBytes: 0 });
          }
        }

        if (!serverStatus) {
          serverStatus = (
            await api.createUpload(
              {
                clientId: record.id,
                filename: record.name,
                totalBytes: record.totalBytes,
                contentType: record.type,
                target: record.target,
              },
              signal,
            )
          ).data;
          setRecord(id, {
            serverUploadId: serverStatus.id,
            uploadedBytes: serverStatus.receivedBytes,
            progress: serverStatus.progress,
            status:
              serverStatus.status === 'pending'
                ? 'uploading'
                : serverStatus.status,
          });
        }

        if (serverStatus.status === 'completed') {
          finishTask(id, 'completed');
          return;
        }
        if (
          serverStatus.status === 'error' ||
          serverStatus.status === 'cancelled'
        ) {
          finishTask(id, 'error', serverStatus.error || serverStatus.message);
          return;
        }
        if (serverStatus.status === 'processing') {
          setRecord(id, { status: 'processing' });
          await waitForCompletion(id, serverStatus.id, signal);
          return;
        }

        const serverUploadId = serverStatus.id;
        let offset = serverStatus.receivedBytes;
        if (offset > record.totalBytes) {
          throw new Error('Server upload offset is invalid');
        }
        setRecord(id, {
          status: 'uploading',
          uploadedBytes: offset,
          progress: serverStatus.progress,
        });

        while (
          offset < record.totalBytes &&
          authenticatedRef.current &&
          !unloadingRef.current
        ) {
          const start = offset;
          const end = Math.min(start + CHUNK_SIZE, record.totalBytes);
          const response = await api.uploadChunk(
            serverUploadId,
            start,
            end,
            record.totalBytes,
            record.file.slice(start, end),
            (progressEvent: AxiosProgressEvent) => {
              const loaded = Math.min(progressEvent.loaded, end - start);
              const uploadedBytes = start + loaded;
              publishProgress(id, {
                uploadedBytes,
                progress: Math.floor(
                  (uploadedBytes * 100) / Math.max(record.totalBytes, 1),
                ),
              });
            },
            signal,
          );
          flushProgress(id);
          offset = response.data.receivedBytes;
          setRecord(id, {
            uploadedBytes: offset,
            progress: response.data.progress,
            status:
              response.data.status === 'ready'
                ? 'uploading'
                : response.data.status,
          });
        }

        if (
          signal.aborted ||
          unloadingRef.current ||
          !authenticatedRef.current
        ) {
          return;
        }
        setRecord(id, { status: 'processing' });
        const completeResponse = (
          await api.completeUpload(serverUploadId, signal)
        ).data;
        updateFromServer(id, completeResponse);
        await waitForCompletion(id, serverUploadId, signal);
      } finally {
        controllersRef.current.delete(id);
      }
    },
    [
      finishTask,
      flushProgress,
      publishProgress,
      setRecord,
      updateFromServer,
      waitForCompletion,
    ],
  );

  const drainQueue = useCallback(async () => {
    if (
      runningRef.current ||
      unloadingRef.current ||
      !authenticatedRef.current
    ) {
      return;
    }
    runningRef.current = true;
    try {
      while (authenticatedRef.current && !unloadingRef.current) {
        const next = [...recordsRef.current.values()]
          .filter((record) => !terminalStatuses.has(record.status))
          .sort((a, b) => a.createdAt - b.createdAt)[0];
        if (!next) return;
        try {
          await runTask(next.id);
          retryAttemptRef.current = 0;
        } catch (error) {
          if (
            unloadingRef.current ||
            !authenticatedRef.current ||
            isAbortError(error)
          ) {
            return;
          }
          if (isRetryableUploadError(error)) {
            clearProgressForTask(next.id);
            setRecord(next.id, {
              status: 'pending',
              message: 'Waiting for connection...',
            });
            const retryIndex = Math.min(
              retryAttemptRef.current,
              NETWORK_RETRY_DELAYS_MS.length - 1,
            );
            retryAttemptRef.current = Math.min(
              retryAttemptRef.current + 1,
              NETWORK_RETRY_DELAYS_MS.length - 1,
            );
            await waitForNetworkRetry(NETWORK_RETRY_DELAYS_MS[retryIndex]);
            continue;
          }
          finishTask(next.id, 'error', getErrorMessage(error));
        }
      }
    } finally {
      runningRef.current = false;
    }
  }, [
    clearProgressForTask,
    finishTask,
    runTask,
    setRecord,
    waitForNetworkRetry,
  ]);

  useEffect(() => {
    const resumeOnNetworkRecovery = () => {
      if (!unloadingRef.current && authenticatedRef.current) {
        wakeNetworkRetry();
        void drainQueue();
      }
    };
    window.addEventListener('network-recovered', resumeOnNetworkRecovery);
    return () =>
      window.removeEventListener('network-recovered', resumeOnNetworkRecovery);
  }, [drainQueue, wakeNetworkRetry]);

  const cancelAll = useCallback(
    (keepAlive: boolean, preserveTerminal: boolean) => {
      const records = [...recordsRef.current.values()];
      const activeRecords = records.filter(
        (record) => !terminalStatuses.has(record.status),
      );
      const terminalRecords = preserveTerminal
        ? records.filter((record) => terminalStatuses.has(record.status))
        : [];
      wakeNetworkRetry();
      clearProgressTimers();
      retryAttemptRef.current = 0;
      recordsRef.current.clear();
      terminalRecords.forEach((record) =>
        recordsRef.current.set(record.id, record),
      );
      progressTotalRef.current = terminalRecords.reduce(
        (total, record) => total + record.progress,
        0,
      );
      activeCountRef.current = 0;
      callbacksRef.current.clear();
      controllersRef.current.forEach((controller) => controller.abort());
      controllersRef.current.clear();
      socketsRef.current.forEach((socket) => socket.close());
      socketsRef.current.clear();
      setAggregateProgress(
        terminalRecords.length
          ? Math.round(progressTotalRef.current / terminalRecords.length)
          : 0,
      );
      setActiveCount(0);
      setUploads(terminalRecords.map(toTask));

      for (const record of activeRecords) {
        if (record.serverUploadId) {
          if (keepAlive) {
            api.cancelUploadKeepAlive(record.serverUploadId);
          } else {
            void api.cancelUpload(record.serverUploadId).catch(() => undefined);
          }
        }
      }
    },
    [clearProgressTimers, wakeNetworkRetry],
  );

  useEffect(() => {
    authenticatedRef.current = isAuthenticated;
    if (!isAuthenticated) {
      const cancelTimer = window.setTimeout(() => cancelAll(false, false), 0);
      return () => window.clearTimeout(cancelTimer);
    }
  }, [cancelAll, isAuthenticated]);

  useEffect(() => {
    const cancelOnPageHide = () => {
      if (unloadingRef.current) return;
      unloadingRef.current = true;
      cancelAll(true, false);
    };
    window.addEventListener('pagehide', cancelOnPageHide);
    window.addEventListener('beforeunload', cancelOnPageHide);
    return () => {
      window.removeEventListener('pagehide', cancelOnPageHide);
      window.removeEventListener('beforeunload', cancelOnPageHide);
      clearProgressTimers();
    };
  }, [cancelAll, clearProgressTimers]);

  const enqueueUploads = useCallback(
    async (items: EnqueueUploadItem[]) => {
      if (!authenticatedRef.current || unloadingRef.current) {
        throw new Error('Uploads are unavailable');
      }
      const records = items.map(({ input, callbacks }) => {
        const id = uuidv4();
        const record: UploadRecord = {
          id,
          name: input.name,
          size: input.file.size,
          type:
            input.contentType || input.file.type || 'application/octet-stream',
          target: input.target,
          status: 'pending',
          progress: 0,
          uploadedBytes: 0,
          totalBytes: input.file.size,
          createdAt: Date.now(),
          file: input.file,
        };
        recordsRef.current.set(id, record);
        callbacksRef.current.set(id, callbacks || {});
        return record;
      });
      progressTotalRef.current += records.reduce(
        (total, record) => total + record.progress,
        0,
      );
      activeCountRef.current += records.length;
      setAggregateProgress(
        recordsRef.current.size
          ? Math.round(progressTotalRef.current / recordsRef.current.size)
          : 0,
      );
      setActiveCount(activeCountRef.current);
      setUploads((previous) => [
        ...previous,
        ...records.map((record) => toTask(record)),
      ]);
      void drainQueue();
      return records.map((record) => record.id);
    },
    [drainQueue],
  );

  const enqueueUpload = useCallback(
    async (input: EnqueueUploadInput, callbacks?: UploadCallbacks) => {
      const [id] = await enqueueUploads([{ input, callbacks }]);
      return id;
    },
    [enqueueUploads],
  );

  const cancelAllUploads = useCallback(async () => {
    cancelAll(false, true);
  }, [cancelAll]);

  const dismissUpload = useCallback(
    async (id: string) => {
      const record = recordsRef.current.get(id);
      if (!record || !terminalStatuses.has(record.status)) return;
      recordsRef.current.delete(id);
      callbacksRef.current.delete(id);
      clearProgressForTask(id);
      updateDerivedStats(record, undefined);
      setUploads((previous) => previous.filter((task) => task.id !== id));
    },
    [clearProgressForTask, updateDerivedStats],
  );

  const contextValue = useMemo(
    () => ({
      uploads,
      aggregateProgress,
      activeCount,
      enqueueUpload,
      enqueueUploads,
      cancelAllUploads,
      dismissUpload,
    }),
    [
      activeCount,
      aggregateProgress,
      uploads,
      enqueueUpload,
      enqueueUploads,
      cancelAllUploads,
      dismissUpload,
    ],
  );

  return (
    <UploadContext.Provider value={contextValue}>
      {children}
    </UploadContext.Provider>
  );
};

// eslint-disable-next-line react-refresh/only-export-components
export const useUploads = () => {
  const context = useContext(UploadContext);
  if (!context) {
    throw new Error('useUploads must be used within an UploadProvider');
  }
  return context;
};
