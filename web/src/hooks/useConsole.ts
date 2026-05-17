import { useEffect, useRef, useState } from 'react';

import { useAuth } from '../context/AuthContext';
import { WS_BASE_URL } from '../services/api';

export const useConsole = (serverId: string) => {
  const { token } = useAuth();
  const ws = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<number | null>(null);
  const reconnectAttempts = useRef(0);
  const shouldReconnect = useRef(false);
  const hasLoggedDisconnect = useRef(false);
  const [logs, setLogs] = useState<string[]>([]);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    if (!serverId || !token) return;

    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLogs([]);
    reconnectAttempts.current = 0;
    shouldReconnect.current = true;
    hasLoggedDisconnect.current = false;

    const url = `${WS_BASE_URL}/ws/servers/${serverId}/console?token=${token}`;

    const clearReconnectTimer = () => {
      if (reconnectTimer.current !== null) {
        window.clearTimeout(reconnectTimer.current);
        reconnectTimer.current = null;
      }
    };

    const scheduleReconnect = () => {
      if (!shouldReconnect.current) return;
      clearReconnectTimer();

      const delay = Math.min(30000, 1000 * 2 ** reconnectAttempts.current);
      reconnectAttempts.current += 1;
      reconnectTimer.current = window.setTimeout(() => {
        connect();
      }, delay);
    };

    const connect = () => {
      if (!shouldReconnect.current) return;
      try {
        ws.current = new WebSocket(url);
      } catch {
        scheduleReconnect();
        return;
      }

      ws.current.onopen = () => {
        reconnectAttempts.current = 0;
        hasLoggedDisconnect.current = false;
        setIsConnected(true);
      };

      ws.current.onmessage = (event) => {
        const data = event.data;
        if (typeof data === 'string') {
          const lines = data.split(/\r?\n/).filter((line) => line.length > 0);
          setLogs((prev) => [...prev, ...lines]);
        }
      };

      ws.current.onclose = () => {
        setIsConnected(false);
        if (!hasLoggedDisconnect.current) {
          hasLoggedDisconnect.current = true;
          console.info('Console socket disconnected. Waiting to reconnect...');
        }
        scheduleReconnect();
      };

      ws.current.onerror = () => {
        setIsConnected(false);
      };
    };

    connect();

    return () => {
      shouldReconnect.current = false;
      clearReconnectTimer();
      if (ws.current) {
        if (ws.current.readyState === WebSocket.CONNECTING) {
          const currentWs = ws.current;
          currentWs.onopen = () => currentWs.close(1000, 'Cleanup');
        } else {
          ws.current.close(1000, 'Component unmounted');
        }
        ws.current = null;
      }
    };
  }, [serverId, token]);

  const sendCommand = (cmd: string) => {
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
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
