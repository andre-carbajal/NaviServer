import {
  Cpu,
  HardDrive,
  MemoryStick,
  Plus,
  Server as ServerIcon,
} from 'lucide-react';

import React, { useEffect, useMemo, useState } from 'react';

import ConfirmationModal from '../components/ConfirmationModal';
import CreateModal from '../components/CreateModal';
import ServerListItem from '../components/ServerListItem';
import { Button } from '../components/ui/Button';
import { useServers } from '../hooks/useServers';
import { api } from '../services/api';
import type { ServerStats } from '../types';
import { formatBytes } from '../utils/format';

const Dashboard: React.FC = () => {
  const {
    servers,
    loading,
    createServer,
    startServer,
    stopServer,
    deleteServer,
  } = useServers();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [serverToDelete, setServerToDelete] = useState<string | null>(null);
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

  const handleDelete = (id: string) => {
    setServerToDelete(id);
  };

  const confirmDelete = async () => {
    if (serverToDelete) {
      await deleteServer(serverToDelete);
      setServerToDelete(null);
    }
  };

  if (loading && servers.length === 0) {
    return <div>Loading servers...</div>;
  }

  return (
    <div className="dashboard">
      <div className="modal-header">
        <h1>Dashboard</h1>
        <Button onClick={() => setIsCreateModalOpen(true)}>
          <Plus size={20} /> Create Server
        </Button>
      </div>

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
          gap: '15px',
          marginBottom: '30px',
        }}
      >
        <div
          className="card"
          style={{
            padding: '15px',
            display: 'flex',
            alignItems: 'center',
            gap: '15px',
          }}
        >
          <div
            style={{
              padding: '10px',
              borderRadius: '8px',
              background: 'rgba(59, 130, 246, 0.1)',
              color: '#3b82f6',
            }}
          >
            <Cpu size={24} />
          </div>
          <div>
            <div style={{ fontSize: '0.85rem', color: 'var(--text-muted)' }}>
              Total CPU Usage
            </div>
            <div style={{ fontSize: '1.2rem', fontWeight: 600 }}>
              {systemStats.cpu.toFixed(1)}%
            </div>
          </div>
        </div>
        <div
          className="card"
          style={{
            padding: '15px',
            display: 'flex',
            alignItems: 'center',
            gap: '15px',
          }}
        >
          <div
            style={{
              padding: '10px',
              borderRadius: '8px',
              background: 'rgba(168, 85, 247, 0.1)',
              color: '#a855f7',
            }}
          >
            <MemoryStick size={24} />
          </div>
          <div>
            <div style={{ fontSize: '0.85rem', color: 'var(--text-muted)' }}>
              Total RAM Usage
            </div>
            <div style={{ fontSize: '1.2rem', fontWeight: 600 }}>
              {formatBytes(systemStats.ram)}
            </div>
          </div>
        </div>
        <div
          className="card"
          style={{
            padding: '15px',
            display: 'flex',
            alignItems: 'center',
            gap: '15px',
          }}
        >
          <div
            style={{
              padding: '10px',
              borderRadius: '8px',
              background: 'rgba(234, 179, 8, 0.1)',
              color: '#eab308',
            }}
          >
            <HardDrive size={24} />
          </div>
          <div>
            <div style={{ fontSize: '0.85rem', color: 'var(--text-muted)' }}>
              Total Disk Usage
            </div>
            <div style={{ fontSize: '1.2rem', fontWeight: 600 }}>
              {formatBytes(systemStats.disk)}
            </div>
          </div>
        </div>
      </div>

      <h2 style={{ marginBottom: '20px', fontSize: '1.5rem', fontWeight: 600 }}>
        Lista de servidores
      </h2>

      {servers.length === 0 && !loading ? (
        <div className="card">
          <ServerIcon size={48} />
          <p>No servers found. Create your first server to get started!</p>
        </div>
      ) : (
        <div className="server-list">
          {servers.map((server) => (
            <ServerListItem
              key={server.id}
              server={server}
              stats={allStats[server.id]}
              onStart={startServer}
              onStop={stopServer}
              onDelete={handleDelete}
            />
          ))}
        </div>
      )}

      <CreateModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onCreate={createServer}
      />

      <ConfirmationModal
        isOpen={!!serverToDelete}
        onClose={() => setServerToDelete(null)}
        onConfirm={confirmDelete}
        title="Delete Server"
        message="Are you sure you want to delete this server? This action cannot be undone and all server files will be permanently lost."
        confirmText="Delete Server"
        isDangerous={true}
      />
    </div>
  );
};

export default Dashboard;
