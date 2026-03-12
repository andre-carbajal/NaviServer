import { useEffect, useRef, useState } from 'react';

import { useAuth } from '../context/AuthContext';
import { WS_HOST } from '../services/api';

export const useConsole = (serverId: string) => {
  const { token } = useAuth();
  const ws = useRef<WebSocket | null>(null);
  const [logs, setLogs] = useState<string[]>([]);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    if (!serverId || !token) return;

    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLogs([]);

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const url = `${protocol}//${WS_HOST}/ws/servers/${serverId}/console?token=${token}`;

    console.log(`Connecting to WS: ${url}`);
    ws.current = new WebSocket(url);

    ws.current.onopen = () => {
      console.log('WS Connected');
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
      console.log('WS Closed');
      setIsConnected(false);
    };

    ws.current.onerror = (error) => {
      console.error('WS Error:', error);
      setIsConnected(false);
    };

    return () => {
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
