import type { ReactNode } from 'react';
import React, {
  createContext,
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react';

import { WS_HOST, api } from '../services/api';
import type { Server } from '../types';
import { useAuth } from './AuthContext';

interface ServerContextType {
  servers: Server[];
  loading: boolean;
  createServer: (data: {
    name: string;
    loader: string;
    version: string;
    ram: number;
    requestId?: string;
  }) => Promise<boolean>;
  startServer: (id: string) => Promise<void>;
  stopServer: (id: string) => Promise<void>;
  deleteServer: (id: string) => Promise<void>;
  refresh: () => Promise<void>;
}

export const ServerContext = createContext<ServerContextType | undefined>(
  undefined,
);

export const ServerProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const { token } = useAuth();
  const [servers, setServers] = useState<Server[]>([]);
  const [loading, setLoading] = useState(true);
  const activeSockets = useRef<Set<string>>(new Set());
  const wsMap = useRef<Map<string, WebSocket>>(new Map());

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
    const stored = localStorage.getItem('creating_servers');
    if (stored) {
      try {
        const list: Server[] = JSON.parse(stored);
        const newList = list.filter((s) => s.id !== id);
        localStorage.setItem('creating_servers', JSON.stringify(newList));
      } catch (e) {
        console.error(e);
      }
    }

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
        `ws://${WS_HOST}/ws/progress/${requestId}?token=${token}`,
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

                  if (
                    newSteps.length === 0 ||
                    newSteps[newSteps.length - 1].label !== msg
                  ) {
                    if (
                      newSteps.length > 0 &&
                      newSteps[newSteps.length - 1].state === 'running'
                    ) {
                      newSteps[newSteps.length - 1].state = 'done';
                      newSteps[newSteps.length - 1].progress = undefined;
                    }
                    newSteps.push({
                      label: msg,
                      state: 'running',
                      progress: progress > 0 ? progress : undefined,
                    });
                  } else {
                    if (newSteps.length > 0) {
                      newSteps[newSteps.length - 1].progress =
                        progress > 0 ? progress : undefined;
                    }
                  }

                  if (progress === -1) {
                    if (newSteps.length > 0) {
                      newSteps[newSteps.length - 1].state = 'failed';
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
              if (newSteps.length > 0) {
                newSteps[newSteps.length - 1].state = 'failed';
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
    const stored = localStorage.getItem('creating_servers');
    if (stored) {
      try {
        const creatingServers: Server[] = JSON.parse(stored);
        setServers((prev) => {
          const existingIds = new Set(prev.map((s) => s.id));
          const toAdd = creatingServers.filter((s) => !existingIds.has(s.id));
          return [...prev, ...toAdd];
        });
        creatingServers.forEach((s) => trackProgress(s.id));
      } catch (e) {
        console.error(e);
      }
    }
  }, [trackProgress]);

  const createServer = async (data: {
    name: string;
    loader: string;
    version: string;
    ram: number;
    requestId?: string;
  }) => {
    const tempId = data.requestId || `temp-${Date.now()}`;

    const tempServer: Server = {
      id: tempId,
      name: data.name,
      loader: data.loader,
      version: data.version,
      ram: data.ram,
      port: 0,
      status: 'CREATING',
      progress: 0,
      progressMessage: 'Initializing...',
    };

    setServers((prev) => [...prev, tempServer]);

    const stored = localStorage.getItem('creating_servers');
    const list: Server[] = stored ? JSON.parse(stored) : [];
    list.push(tempServer);
    localStorage.setItem('creating_servers', JSON.stringify(list));

    trackProgress(tempId);

    try {
      await api.createServer(data);
      return true;
    } catch (err) {
      console.error(err);
      removeCreatingServer(tempId);
      throw err;
    }
  };

  const startServer = async (id: string) => {
    try {
      setServers((prev) =>
        prev.map((s) => (s.id === id ? { ...s, status: 'STARTING' } : s)),
      );
      await api.startServer(id);
    } catch (err) {
      console.error(err);
      await fetchServers();
    }
  };

  const stopServer = async (id: string) => {
    try {
      await api.stopServer(id);
      await fetchServers();
    } catch (err) {
      console.error(err);
    }
  };

  const deleteServer = async (id: string) => {
    const isCreating = servers.find((s) => s.id === id)?.status === 'CREATING';

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
  };

  useEffect(() => {
    if (!token) {
      setServers([]);
      return;
    }

    fetchServers();
    const interval = setInterval(fetchServers, 5000);
    return () => clearInterval(interval);
  }, [fetchServers, token]);

  return (
    <ServerContext.Provider
      value={{
        servers,
        loading,
        createServer,
        startServer,
        stopServer,
        deleteServer,
        refresh: fetchServers,
      }}
    >
      {children}
    </ServerContext.Provider>
  );
};
