import { useEffect, useRef, useState } from 'react';

import { api } from '../services/api';
import type { ServerStats } from '../types';

const RUNNING_POLL_MS = 2000;
const IDLE_POLL_MS = 4000;
const MAX_BACKOFF_MS = 30000;

export const useServerStats = (serverId: string, isRunning: boolean) => {
  const [stats, setStats] = useState<ServerStats>({
    cpu: 0,
    ram: 0,
    disk: 0,
    onlinePlayers: 0,
    maxPlayers: 0,
    uptimeSeconds: 0,
    players: [],
  });
  const [loading, setLoading] = useState(true);
  const [isOffline, setIsOffline] = useState(false);
  const retryDelayRef = useRef(RUNNING_POLL_MS);
  const offlineLoggedRef = useRef(false);

  useEffect(() => {
    if (!serverId) return;

    let cancelled = false;
    let timer: number | null = null;

    retryDelayRef.current = isRunning ? RUNNING_POLL_MS : IDLE_POLL_MS;
    offlineLoggedRef.current = false;
    const resetTimer = window.setTimeout(() => {
      setLoading(true);
      setIsOffline(false);
    }, 0);

    const scheduleNext = (delay: number) => {
      if (cancelled) return;
      timer = window.setTimeout(() => {
        void fetchStats();
      }, delay);
    };

    const fetchStats = async () => {
      try {
        const res = await api.getServerStats(serverId);
        if (cancelled) return;
        setStats(res.data);
        setIsOffline(false);
        retryDelayRef.current = isRunning ? RUNNING_POLL_MS : IDLE_POLL_MS;
        offlineLoggedRef.current = false;
      } catch {
        if (!cancelled && !offlineLoggedRef.current) {
          offlineLoggedRef.current = true;
          console.warn('Server stats unavailable, retrying with backoff.');
        }
        setIsOffline(true);
        retryDelayRef.current = Math.min(
          MAX_BACKOFF_MS,
          retryDelayRef.current * 2,
        );
      } finally {
        if (!cancelled) {
          setLoading(false);
          scheduleNext(retryDelayRef.current);
        }
      }
    };

    void fetchStats();

    return () => {
      cancelled = true;
      window.clearTimeout(resetTimer);
      if (timer !== null) {
        window.clearTimeout(timer);
      }
    };
  }, [serverId, isRunning]);

  return { stats, loading, isOffline };
};
