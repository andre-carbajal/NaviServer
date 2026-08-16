import type { ReactNode } from 'react';
import React, {
  createContext,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';

import { WS_BASE_URL, api } from '../services/api';
import type { Server } from '../types';
import { useAuth } from './AuthContext';

const CREATING_SERVERS_STORAGE_KEY = 'creating_servers:v1';
const LEGACY_CREATING_SERVERS_STORAGE_KEY = 'creating_servers';

const readCreatingServers = (): Server[] => {
  const stored =
    localStorage.getItem(CREATING_SERVERS_STORAGE_KEY) ??
    localStorage.getItem(LEGACY_CREATING_SERVERS_STORAGE_KEY);

  if (!stored) return [];

  try {
    return JSON.parse(stored) as Server[];
  } catch (error) {
    console.error(error);
    return [];
  }
};

const writeCreatingServers = (servers: Server[]) => {
  localStorage.setItem(CREATING_SERVERS_STORAGE_KEY, JSON.stringify(servers));
  localStorage.removeItem(LEGACY_CREATING_SERVERS_STORAGE_KEY);
};

interface ServerContextType {
  servers: Server[];
  loading: boolean;
  createServer: (data: {
    name: string;
    loader: string;
    version?: string;
    ram: number;
    requestId?: string;
    loaderOptions?: {
      mcVersion?: string;
      includeSnapshots?: boolean;
      includeUnstable?: boolean;
      buildVersion?: string;
      loaderVersion?: string;
      installerVersion?: string;
    };
  }) => Promise<boolean>;
  startServer: (id: string) => Promise<void>;
  stopServer: (id: string) => Promise<void>;
  deleteServer: (id: string) => Promise<void>;
  refresh: () => Promise<void>;
}

// eslint-disable-next-line react-refresh/only-export-components
export const ServerContext = createContext<ServerContextType | undefined>(
  undefined,
);

export const ServerProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const { token } = useAuth();
  const [servers, setServers] = useState<Server[]>([]);
  const [loading, setLoading] = useState(true);
  const activeSockets = useRef<Set<string>>(null!);
  const wsMap = useRef<Map<string, WebSocket>>(null!);

  if (activeSockets.current === null) {
    activeSockets.current = new Set();
  }

  if (wsMap.current === null) {
    wsMap.current = new Map();
  }

  const fetchServers = useCallback(async () => {
    try {
      const response = await api.getServers();
      setServers((prevServers) => {
        const creatingServers = prevServers.filter(
          (s) => s.status === 'CREATING',
        );
        const newServers = Array.isArray(response.data) ? response.data : [];

        const newServerIds = new Set(newServers.map((s) => s.id));
        const uniqueCreating = creatingServers.filter(
          (s) => !newServerIds.has(s.id),
        );

        return [...newServers, ...uniqueCreating];
      });
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, []);

  const removeCreatingServer = useCallback((id: string) => {
    setServers((prev) => prev.filter((s) => s.id !== id));
    const newList = readCreatingServers().filter((s) => s.id !== id);
    writeCreatingServers(newList);

    const ws = wsMap.current.get(id);
    if (ws) {
      ws.close();
      wsMap.current.delete(id);
    }
    activeSockets.current.delete(id);
  }, []);

  const trackProgress = useCallback(
    (requestId: string) => {
      if (activeSockets.current.has(requestId) || !token) return;

      activeSockets.current.add(requestId);
      const ws = new WebSocket(
        `${WS_BASE_URL}/ws/progress/${requestId}?token=${token}`,
      );
      wsMap.current.set(requestId, ws);

      ws.onmessage = (event) => {
        try {
          const msgData = JSON.parse(event.data);

          if (msgData.message === 'Server created successfully') {
            ws.close();
            removeCreatingServer(requestId);
            fetchServers();
          } else {
            setServers((prev) =>
              prev.map((s) => {
                if (s.id === requestId) {
                  const currentSteps = s.steps || [];
                  const newSteps = [...currentSteps];
                  const msg = msgData.message;
                  const progress = msgData.progress;
                  const lastStep = newSteps.at(-1);

                  if (!lastStep || lastStep.label !== msg) {
                    if (lastStep?.state === 'running') {
                      lastStep.state = 'done';
                      lastStep.progress = undefined;
                    }
                    newSteps.push({
                      label: msg,
                      state: 'running',
                      progress: progress > 0 ? progress : undefined,
                    });
                  } else if (lastStep) {
                    lastStep.progress = progress > 0 ? progress : undefined;
                  }

                  if (progress === -1) {
                    const latestStep = newSteps.at(-1);
                    if (latestStep) {
                      latestStep.state = 'failed';
                    }
                  }

                  return {
                    ...s,
                    progress: msgData.progress,
                    progressMessage: msgData.message,
                    steps: newSteps,
                  };
                }
                return s;
              }),
            );
          }
        } catch (e) {
          console.error('Error parsing progress message', e);
        }
      };

      ws.onerror = (e) => {
        console.error('WebSocket error', e);
        setServers((prev) =>
          prev.map((s) => {
            if (s.id === requestId) {
              const currentSteps = s.steps || [];
              const newSteps = [...currentSteps];
              const lastStep = newSteps.at(-1);
              if (lastStep) {
                lastStep.state = 'failed';
              } else {
                newSteps.push({ label: 'Connection Error', state: 'failed' });
              }

              return {
                ...s,
                progressMessage: 'Error connecting to progress stream',
                steps: newSteps,
              };
            }
            return s;
          }),
        );
      };

      ws.onclose = () => {
        activeSockets.current.delete(requestId);
        wsMap.current.delete(requestId);
      };
    },
    [fetchServers, removeCreatingServer, token],
  );

  useEffect(() => {
    const restoreTimer = window.setTimeout(() => {
      const creatingServers = readCreatingServers();
      if (creatingServers.length === 0) return;

      writeCreatingServers(creatingServers);
      setServers((prev) => {
        const existingIds = new Set(prev.map((s) => s.id));
        const toAdd = creatingServers.filter((s) => !existingIds.has(s.id));
        return [...prev, ...toAdd];
      });
      creatingServers.forEach((s) => trackProgress(s.id));
    }, 0);

    return () => window.clearTimeout(restoreTimer);
  }, [trackProgress]);

  const createServer = useCallback(
    async (data: {
      name: string;
      loader: string;
      version?: string;
      ram: number;
      requestId?: string;
      loaderOptions?: {
        mcVersion?: string;
        includeSnapshots?: boolean;
        includeUnstable?: boolean;
        buildVersion?: string;
        loaderVersion?: string;
        installerVersion?: string;
      };
    }) => {
      const tempId = data.requestId || `temp-${Date.now()}`;

      const tempServer: Server = {
        id: tempId,
        name: data.name,
        loader: data.loader,
        version: data.loaderOptions?.mcVersion || data.version || 'latest',
        ram: data.ram,
        port: 0,
        status: 'CREATING',
        progress: 0,
        progressMessage: 'Initializing...',
      };

      setServers((prev) => [...prev, tempServer]);

      const list = readCreatingServers();
      list.push(tempServer);
      writeCreatingServers(list);

      trackProgress(tempId);

      try {
        await api.createServer(data);
        return true;
      } catch (err) {
        console.error(err);
        removeCreatingServer(tempId);
        throw err;
      }
    },
    [removeCreatingServer, trackProgress],
  );

  const startServer = useCallback(
    async (id: string) => {
      try {
        setServers((prev) =>
          prev.map((s) => (s.id === id ? { ...s, status: 'STARTING' } : s)),
        );
        await api.startServer(id);
      } catch (err) {
        console.error(err);
        await fetchServers();
      }
    },
    [fetchServers],
  );

  const stopServer = useCallback(
    async (id: string) => {
      try {
        await api.stopServer(id);
        await fetchServers();
      } catch (err) {
        console.error(err);
      }
    },
    [fetchServers],
  );

  const deleteServer = useCallback(
    async (id: string) => {
      const isCreating =
        servers.find((s) => s.id === id)?.status === 'CREATING';

      if (isCreating) {
        removeCreatingServer(id);
        return;
      }

      try {
        setServers((prev) => prev.filter((s) => s.id !== id));
        await api.deleteServer(id);
      } catch (err) {
        console.error(err);
        await fetchServers();
      }
    },
    [fetchServers, removeCreatingServer, servers],
  );

  useEffect(() => {
    if (!token) {
      const clearServersTimer = window.setTimeout(() => {
        setServers([]);
      }, 0);
      return () => window.clearTimeout(clearServersTimer);
    }

    const initialFetch = window.setTimeout(() => {
      void fetchServers();
    }, 0);
    const interval = setInterval(fetchServers, 5000);
    return () => {
      window.clearTimeout(initialFetch);
      clearInterval(interval);
    };
  }, [fetchServers, token]);

  const contextValue = useMemo(
    () => ({
      servers,
      loading,
      createServer,
      startServer,
      stopServer,
      deleteServer,
      refresh: fetchServers,
    }),
    [
      servers,
      loading,
      createServer,
      startServer,
      stopServer,
      deleteServer,
      fetchServers,
    ],
  );

  return (
    <ServerContext.Provider value={contextValue}>
      {children}
    </ServerContext.Provider>
  );
};
