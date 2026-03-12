import {
  Cpu,
  Folder,
  HardDrive,
  MemoryStick,
  Play,
  Settings,
  Share2,
  Square,
  Terminal,
  Users,
} from 'lucide-react';
import { useParams } from 'react-router-dom';

import React, { useCallback, useEffect, useState } from 'react';

import ConsoleView from '../components/ConsoleView';
import EditServerModal from '../components/EditServerModal';
import FileExplorer from '../components/FileExplorer';
import ShareModal from '../components/ShareModal';
import { Button } from '../components/ui/Button';
import { CopyButton } from '../components/ui/CopyButton';
import { useAuth } from '../context/AuthContext';
import { useConsole } from '../hooks/useConsole';
import { useCopy } from '../hooks/useCopy';
import { useServerStats } from '../hooks/useServerStats';
import { api } from '../services/api';
import type { Server } from '../types';

const ServerDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { user } = useAuth();
  const [server, setServer] = useState<Server | null>(null);
  const [loading, setLoading] = useState(true);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [isShareModalOpen, setIsShareModalOpen] = useState(false);
  const [commandInput, setCommandInput] = useState('');
  const [commandHistory, setCommandHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [iconError, setIconError] = useState(false);
  const [iconRefreshKey, setIconRefreshKey] = useState(0);
  const [activeTab, setActiveTab] = useState<'console' | 'files'>('console');
  const [publicIP, setPublicIP] = useState<string>(
    typeof window !== 'undefined' ? window.location.hostname : 'localhost',
  );

  const { logs, sendCommand, isConnected } = useConsole(id || '');
  const { stats } = useServerStats(id || '', server?.status === 'RUNNING');
  const { copy } = useCopy(1500);

  useEffect(() => {
    const fetchPublicIP = async () => {
      try {
        const response = await api.getPublicIP();
        if (response.data?.public_ip) {
          setPublicIP(response.data.public_ip);
        }
      } catch (err) {
        console.error('Failed to fetch public IP:', err);
      }
    };

    fetchPublicIP();
  }, []);

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const fetchServer = useCallback(async () => {
    if (!id) return;
    try {
      const res = await api.getServer(id);
      setServer(res.data);
    } catch (err) {
      console.error('Failed to fetch server:', err);
      if ((err as any).response?.status === 404) {
        setServer(null);
      }
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    fetchServer();
    const interval = setInterval(fetchServer, 2000);
    return () => clearInterval(interval);
  }, [fetchServer]);

  useEffect(() => {
    if (server?.status === 'STOPPED') {
      setCommandHistory([]);
      setHistoryIndex(-1);
    }
  }, [server?.status]);

  const handleStart = async () => {
    if (!server) return;
    try {
      await api.startServer(server.id);
      setServer((prev) => (prev ? { ...prev, status: 'STARTING' } : null));
    } catch (e) {
      console.error(e);
    }
  };

  const handleStop = async () => {
    if (!server) return;
    try {
      await api.stopServer(server.id);
      setServer((prev) => (prev ? { ...prev, status: 'STOPPING' } : null));
    } catch (e) {
      console.error(e);
    }
  };

  const handleShare = () => {
    setIsShareModalOpen(true);
  };

  const handleSaveSettings = async (data: {
    name: string;
    ram: number;
    customArgs?: string;
    icon?: File;
  }) => {
    if (!server) return;
    try {
      await api.updateServer(server.id, {
        name: data.name,
        ram: data.ram,
        customArgs: data.customArgs,
      });

      if (data.icon) {
        await api.uploadServerIcon(server.id, data.icon);
        setIconRefreshKey((prev) => prev + 1);
        setIconError(false);
      }

      await fetchServer();
    } catch (err) {
      console.error('Failed to save settings:', err);
    }
  };

  const handleCommandSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!commandInput.trim()) return;

    // Add to history
    setCommandHistory((prev) => [commandInput, ...prev]);
    setHistoryIndex(-1);

    sendCommand(commandInput);
    setCommandInput('');
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (historyIndex < commandHistory.length - 1) {
        const nextIndex = historyIndex + 1;
        setHistoryIndex(nextIndex);
        setCommandInput(commandHistory[nextIndex]);
      }
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (historyIndex > 0) {
        const nextIndex = historyIndex - 1;
        setHistoryIndex(nextIndex);
        setCommandInput(commandHistory[nextIndex]);
      } else if (historyIndex === 0) {
        setHistoryIndex(-1);
        setCommandInput('');
      }
    }
  };

  if (loading) return <div>Loading...</div>;
  if (!server) return <div>Server not found</div>;

  const address = `${publicIP}:${server.port}`;

  return (
    <div className="server-detail h-full flex flex-col">
      <div className="server-detail-header shrink-0">
        <div className="header-content-wrapper">
          <div className="header-icon">
            {!iconError ? (
              <img
                src={`${api.getServerIconUrl(server.id)}?t=${iconRefreshKey}`}
                alt="Server Icon"
                onError={() => setIconError(true)}
                className="server-icon-img"
              />
            ) : (
              <div className="server-icon-placeholder">
                {server.name.charAt(0).toUpperCase()}
              </div>
            )}
          </div>
          <div className="header-info">
            <div className="header-title-row">
              <h1>{server.name}</h1>
              <span
                className={`status-badge status-${server.status.toLowerCase()}`}
              >
                {server.status}
              </span>
            </div>

            <div className="header-meta">
              <div className="meta-item">
                <span className="meta-label">{server.loader}</span>
                <span>{server.version}</span>
              </div>
              <div className="meta-dot"></div>
              <div className="meta-item">
                <span
                  className="meta-value"
                  onClick={() => copy(address)}
                  style={{
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '6px',
                  }}
                  title="Click to copy"
                >
                  {address}
                </span>
                <CopyButton
                  text={address}
                  variant="secondary"
                  title="Copy to clipboard"
                  className="address-copy-btn"
                />
              </div>
            </div>
          </div>
        </div>

        <div className="header-actions">
          {(server.permissions?.canControlPower ||
            server.permissions?.canViewConsole) &&
            (server.status === 'STOPPED' ? (
              <Button onClick={handleStart}>
                <Play size={18} /> Start
              </Button>
            ) : (
              <Button
                variant="danger"
                onClick={handleStop}
                disabled={
                  server.status === 'STARTING' || server.status === 'STOPPING'
                }
              >
                <Square size={18} /> Stop
              </Button>
            ))}
          {server.permissions?.canViewConsole && (
            <Button
              variant="secondary"
              onClick={handleShare}
              title="Create Public Link"
            >
              <Share2 size={18} />
            </Button>
          )}
          {user?.role === 'admin' && (
            <Button
              variant="secondary"
              onClick={() => setIsEditModalOpen(true)}
            >
              <Settings size={18} />
            </Button>
          )}
        </div>
      </div>

      <div className="server-tabs">
        <button
          onClick={() => setActiveTab('console')}
          className={`tab-btn ${activeTab === 'console' ? 'active' : ''}`}
        >
          <Terminal size={16} />
          Console
          {activeTab === 'console' && <div className="tab-indicator"></div>}
        </button>
        <button
          onClick={() => setActiveTab('files')}
          className={`tab-btn ${activeTab === 'files' ? 'active' : ''}`}
        >
          <Folder size={16} />
          Files
          {activeTab === 'files' && <div className="tab-indicator"></div>}
        </button>
      </div>

      <div
        style={{
          display: activeTab === 'console' ? 'flex' : 'none',
          flex: 1,
          flexDirection: 'column',
          gap: '15px',
        }}
      >
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
            gap: '15px',
            marginBottom: '10px',
          }}
        >
          <div
            className="card"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '15px',
              padding: '15px',
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
                CPU Usage
              </div>
              <div style={{ fontSize: '1.2rem', fontWeight: 600 }}>
                {server.status === 'RUNNING'
                  ? `${stats.cpu.toFixed(1)}%`
                  : 'Offline'}
              </div>
            </div>
          </div>

          <div
            className="card"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '15px',
              padding: '15px',
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
                RAM Usage
              </div>
              <div style={{ fontSize: '1.2rem', fontWeight: 600 }}>
                {server.status === 'RUNNING'
                  ? `${formatBytes(stats.ram)} / ${server.ram}MB`
                  : 'Offline'}
              </div>
            </div>
          </div>

          <div
            className="card"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '15px',
              padding: '15px',
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
                Disk Usage
              </div>
              <div style={{ fontSize: '1.2rem', fontWeight: 600 }}>
                {formatBytes(stats.disk)}
              </div>
            </div>
          </div>

          <div
            className="card"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '15px',
              padding: '15px',
            }}
          >
            <div
              style={{
                padding: '10px',
                borderRadius: '8px',
                background: 'rgba(16, 185, 129, 0.1)',
                color: '#10b981',
              }}
            >
              <Users size={24} />
            </div>
            <div>
              <div style={{ fontSize: '0.85rem', color: 'var(--text-muted)' }}>
                Players
              </div>
              <div style={{ fontSize: '1.2rem', fontWeight: 600 }}>
                {server.status === 'RUNNING'
                  ? `${stats.onlinePlayers} / ${stats.maxPlayers}`
                  : 'Offline'}
              </div>
            </div>
          </div>
        </div>

        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: '10px',
          }}
        >
          <h2 style={{ margin: 0, fontSize: '1.2rem' }}>Console</h2>
          <span
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              fontSize: '0.9rem',
              color: isConnected ? '#4ade80' : 'var(--text-muted)',
            }}
          >
            {isConnected ? '● Connected' : '○ Disconnected'}
          </span>
        </div>

        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: '10px',
            flex: 1,
          }}
        >
          <ConsoleView logs={logs} />

          <form
            onSubmit={handleCommandSubmit}
            style={{ display: 'flex', gap: '10px' }}
            className="console-input-form"
          >
            <input
              type="text"
              value={commandInput}
              onChange={(e) => setCommandInput(e.target.value)}
              onKeyDown={handleKeyDown}
              className="form-input"
              placeholder="Type a command..."
              disabled={!isConnected}
              style={{ flex: 1 }}
            />
            <Button
              type="submit"
              disabled={!isConnected || !commandInput.trim()}
            >
              Send
            </Button>
          </form>
        </div>
      </div>

      <div
        style={{ display: activeTab === 'files' ? 'block' : 'none', flex: 1 }}
      >
        <FileExplorer serverId={server.id} />
      </div>

      <EditServerModal
        isOpen={isEditModalOpen}
        onClose={() => setIsEditModalOpen(false)}
        onSave={handleSaveSettings}
        server={server}
      />
      <ShareModal
        isOpen={isShareModalOpen}
        onClose={() => setIsShareModalOpen(false)}
        serverId={server.id}
      />
    </div>
  );
};

export default ServerDetail;
