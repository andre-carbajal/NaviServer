import { useEffect, useState } from 'react';

import { api } from '../services/api';
import type { ServerStats } from '../types';

export const useServerStats = (serverId: string, isRunning: boolean) => {
  const [stats, setStats] = useState<ServerStats>({ cpu: 0, ram: 0, disk: 0 });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!serverId) return;

    const fetchStats = async () => {
      try {
        const res = await api.getServerStats(serverId);
        setStats(res.data);
      } catch (error) {
        console.error('Failed to fetch server stats:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchStats();

    const interval = setInterval(fetchStats, 2000);

    return () => clearInterval(interval);
  }, [serverId, isRunning]);

  return { stats, loading };
};
