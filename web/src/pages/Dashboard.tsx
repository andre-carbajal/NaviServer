import {
  Cpu,
  HardDrive,
  MemoryStick,
  Plus,
  Server as ServerIcon,
} from 'lucide-react';

import React, { useEffect, useMemo, useState } from 'react';

import CreateModal from '../components/dashboard/CreateModal';
import ServerListItem from '../components/dashboard/ServerListItem';
import { Button } from '../components/ui/Button';
import { useServers } from '../hooks/useServers';
import { api } from '../services/api';
import type { ServerStats } from '../types';
import { formatBytes } from '../utils/format';

const Dashboard: React.FC = () => {
  const { servers, loading, createServer, startServer, stopServer } =
    useServers();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [allStats, setAllStats] = useState<Record<string, ServerStats>>({});

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const res = await api.getAllServerStats();
        setAllStats(res.data);
      } catch (error) {
        console.error('Failed to fetch server stats', error);
      }
    };

    fetchStats();
    const interval = setInterval(fetchStats, 2000);
    return () => clearInterval(interval);
  }, []);

  const systemStats = useMemo(() => {
    let totalCpu = 0;
    let totalRam = 0;
    let totalDisk = 0;

    servers.forEach((server) => {
      const stats = allStats[server.id];
      if (stats) {
        totalCpu += stats.cpu;
        totalRam += stats.ram;
        totalDisk += stats.disk;
      }
    });

    return { cpu: totalCpu, ram: totalRam, disk: totalDisk };
  }, [servers, allStats]);

  if (loading && servers.length === 0) {
    return <div>Loading servers...</div>;
  }

  return (
    <div className="tw:flex tw:flex-col">
      <div className="tw:mb-5 tw:flex tw:items-center tw:justify-between tw:gap-3">
        <h1 className="tw:m-0">Dashboard</h1>
        <Button
          className="tw:shrink-0"
          onClick={() => setIsCreateModalOpen(true)}
        >
          <Plus size={20} /> Create Server
        </Button>
      </div>

      <div className="tw:mb-[30px] tw:grid tw:grid-cols-[repeat(auto-fit,minmax(200px,1fr))] tw:gap-[15px]">
        <div className="tw:flex tw:items-center tw:gap-[15px] tw:rounded-xl tw:border tw:border-border tw:bg-bg-card tw:p-[15px]">
          <div className="tw:rounded-lg tw:bg-blue-500/10 tw:p-2.5 tw:text-blue-500">
            <Cpu size={24} />
          </div>
          <div>
            <div className="tw:text-[0.85rem] tw:text-text-muted">
              Total CPU Usage
            </div>
            <div className="tw:text-[1.2rem] tw:font-semibold">
              {systemStats.cpu.toFixed(1)}%
            </div>
          </div>
        </div>
        <div className="tw:flex tw:items-center tw:gap-[15px] tw:rounded-xl tw:border tw:border-border tw:bg-bg-card tw:p-[15px]">
          <div className="tw:rounded-lg tw:bg-purple-500/10 tw:p-2.5 tw:text-purple-500">
            <MemoryStick size={24} />
          </div>
          <div>
            <div className="tw:text-[0.85rem] tw:text-text-muted">
              Total RAM Usage
            </div>
            <div className="tw:text-[1.2rem] tw:font-semibold">
              {formatBytes(systemStats.ram)}
            </div>
          </div>
        </div>
        <div className="tw:flex tw:items-center tw:gap-[15px] tw:rounded-xl tw:border tw:border-border tw:bg-bg-card tw:p-[15px]">
          <div className="tw:rounded-lg tw:bg-yellow-500/10 tw:p-2.5 tw:text-yellow-500">
            <HardDrive size={24} />
          </div>
          <div>
            <div className="tw:text-[0.85rem] tw:text-text-muted">
              Total Disk Usage
            </div>
            <div className="tw:text-[1.2rem] tw:font-semibold">
              {formatBytes(systemStats.disk)}
            </div>
          </div>
        </div>
      </div>

      <h2 className="tw:mb-5 tw:text-[1.5rem] tw:font-semibold">
        Lista de servidores
      </h2>

      {servers.length === 0 && !loading ? (
        <div className="tw:rounded-xl tw:border tw:border-border tw:bg-bg-card tw:p-5">
          <ServerIcon size={48} />
          <p>No servers found. Create your first server to get started!</p>
        </div>
      ) : (
        <div className="tw:flex tw:flex-col tw:gap-3">
          {servers.map((server) => (
            <ServerListItem
              key={server.id}
              server={server}
              stats={allStats[server.id]}
              onStart={startServer}
              onStop={stopServer}
            />
          ))}
        </div>
      )}

      <CreateModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onCreate={createServer}
      />
    </div>
  );
};

export default Dashboard;
