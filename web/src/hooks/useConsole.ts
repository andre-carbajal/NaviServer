import { useEffect, useRef, useState } from 'react';

import { useAuth } from '../context/AuthContext';
import { WS_BASE_URL } from '../services/api';

export const useConsole = (serverId: string) => {
  const { token } = useAuth();
  const ws = useRef<WebSocket | null>(null);
  const [logs, setLogs] = useState<string[]>([]);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    if (!serverId || !token) return;

    let isActive = true;
    let socket: WebSocket | null = null;
    let reconnectTimer: number | null = null;
    let reconnectAttempts = 0;
    let hasLoggedDisconnect = false;

    const resetLogsTimer = window.setTimeout(() => {
      if (isActive) {
        setLogs([]);
      }
    }, 0);

    const url = `${WS_BASE_URL}/ws/servers/${serverId}/console?token=${token}`;

    const clearReconnectTimer = () => {
      if (reconnectTimer !== null) {
        window.clearTimeout(reconnectTimer);
        reconnectTimer = null;
      }
    };

    const scheduleReconnect = () => {
      if (!isActive || reconnectTimer !== null) return;

      const delay = Math.min(30000, 1000 * 2 ** reconnectAttempts);
      reconnectAttempts += 1;
      reconnectTimer = window.setTimeout(() => {
        reconnectTimer = null;
        connect();
      }, delay);
    };

    const connect = () => {
      if (
        !isActive ||
        socket?.readyState === WebSocket.CONNECTING ||
        socket?.readyState === WebSocket.OPEN
      ) {
        return;
      }

      let nextSocket: WebSocket;
      try {
        nextSocket = new WebSocket(url);
      } catch {
        scheduleReconnect();
        return;
      }

      socket = nextSocket;
      ws.current = nextSocket;

      const isCurrentSocket = () => isActive && socket === nextSocket;

      nextSocket.onopen = () => {
        if (!isCurrentSocket()) {
          nextSocket.close(1000, 'Stale console socket');
          return;
        }

        reconnectAttempts = 0;
        hasLoggedDisconnect = false;
        setIsConnected(true);
      };

      nextSocket.onmessage = (event) => {
        if (!isCurrentSocket()) return;

        const data = event.data;
        if (typeof data === 'string') {
          const lines = data.split(/\r?\n/).filter((line) => line.length > 0);
          setLogs((prev) => [...prev, ...lines]);
        }
      };

      nextSocket.onclose = () => {
        if (!isCurrentSocket()) return;

        socket = null;
        if (ws.current === nextSocket) {
          ws.current = null;
        }
        setIsConnected(false);
        if (!hasLoggedDisconnect) {
          hasLoggedDisconnect = true;
          console.info('Console socket disconnected. Waiting to reconnect...');
        }
        scheduleReconnect();
      };

      nextSocket.onerror = () => {
        if (isCurrentSocket()) {
          setIsConnected(false);
        }
      };
    };

    connect();

    return () => {
      isActive = false;
      window.clearTimeout(resetLogsTimer);
      clearReconnectTimer();

      const currentSocket = socket;
      socket = null;
      if (ws.current === currentSocket) {
        ws.current = null;
      }

      if (currentSocket) {
        if (currentSocket.readyState === WebSocket.CONNECTING) {
          currentSocket.onopen = () => currentSocket.close(1000, 'Cleanup');
        } else if (currentSocket.readyState === WebSocket.OPEN) {
          currentSocket.close(1000, 'Component unmounted');
        }
      }
    };
  }, [serverId, token]);

  const sendCommand = (cmd: string) => {
    if (ws.current?.readyState === WebSocket.OPEN) {
      ws.current.send(cmd + '\n');
    } else {
      console.warn('WebSocket not connected, cannot send command');
    }
  };

  const clearLogs = () => {
    setLogs([]);
  };

  return { logs, sendCommand, isConnected, clearLogs };
};
