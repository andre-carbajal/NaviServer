import axios from 'axios';
import {
  ArrowLeft,
  BarChart3,
  Clock3,
  Cpu,
  HardDrive,
  LoaderCircle,
  MemoryStick,
  MoreVertical,
  Play,
  RotateCcw,
  Settings2,
  Share2,
  Skull,
  Square,
  Terminal,
  Users,
} from 'lucide-react';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { useNavigate, useParams } from 'react-router-dom';

import ConsoleView from '../components/ConsoleView';
import FileExplorer from '../components/FileExplorer';
import ShareModal from '../components/ShareModal';
import { Button } from '../components/ui/Button';
import { CopyButton } from '../components/ui/CopyButton';
import { useAuth } from '../context/AuthContext';
import { useConsole } from '../hooks/useConsole';
import { useCopy } from '../hooks/useCopy';
import { useServerStats } from '../hooks/useServerStats';
import { api } from '../services/api';
import type { PlayerInfo, Server } from '../types';

type DetailTab = 'performance' | 'console' | 'players' | 'files' | 'settings';
type ChartRange = '1m' | '5m' | '30m' | '1h' | '4h';

interface StatSnapshot {
  ts: number;
  cpu: number;
  ramMb: number;
}

const RANGE_TO_MS: Record<ChartRange, number> = {
  '1m': 60 * 1000,
  '5m': 5 * 60 * 1000,
  '30m': 30 * 60 * 1000,
  '1h': 60 * 60 * 1000,
  '4h': 4 * 60 * 60 * 1000,
};

const MAX_HISTORY_WINDOW_MS = RANGE_TO_MS['4h'];
const MINEATAR_BASE_URL = 'https://api.mineatar.io/head';
const STEVE_UUID = '8667ba71-b85a-4004-af54-457a9734eed7';

const getAvatarUrl = (uuid?: string) =>
  `${MINEATAR_BASE_URL}/${encodeURIComponent(uuid || STEVE_UUID)}?scale=8&overlay=true`;

const PlayerAvatar: React.FC<{ player: PlayerInfo }> = ({ player }) => {
  const [src, setSrc] = useState(getAvatarUrl(player.id));

  useEffect(() => {
    setSrc(getAvatarUrl(player.id));
  }, [player.id]);

  return (
    <img
      src={src}
      alt={`${player.name} avatar`}
      className="server-v2-player-avatar"
      onError={() => {
        if (src !== getAvatarUrl()) {
          setSrc(getAvatarUrl());
        }
      }}
    />
  );
};

const ServerDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();

  const [server, setServer] = useState<Server | null>(null);
  const [loading, setLoading] = useState(true);
  const [isShareModalOpen, setIsShareModalOpen] = useState(false);
  const [commandInput, setCommandInput] = useState('');
  const [commandHistory, setCommandHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [iconError, setIconError] = useState(false);
  const [iconRefreshKey, setIconRefreshKey] = useState(0);
  const [activeTab, setActiveTab] = useState<DetailTab>('performance');
  const [powerAction, setPowerAction] = useState<
    null | 'start' | 'stop' | 'restart' | 'kill'
  >(null);
  const [isPowerMenuOpen, setIsPowerMenuOpen] = useState(false);
  const [chartRange, setChartRange] = useState<ChartRange>('1m');
  const [statsHistory, setStatsHistory] = useState<StatSnapshot[]>([]);
  const [settingsName, setSettingsName] = useState('');
  const [settingsRam, setSettingsRam] = useState(2048);
  const [settingsCustomArgs, setSettingsCustomArgs] = useState('');
  const [settingsIcon, setSettingsIcon] = useState<File | undefined>(undefined);
  const [isSavingSettings, setIsSavingSettings] = useState(false);
  const [publicIP, setPublicIP] = useState<string>(
    typeof window !== 'undefined' ? window.location.hostname : 'localhost',
  );
  const powerMenuRef = useRef<HTMLDivElement>(null);

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

  const fetchServer = useCallback(async () => {
    if (!id) return;
    try {
      const res = await api.getServer(id);
      setServer(res.data);
    } catch (err) {
      console.error('Failed to fetch server:', err);
      if (axios.isAxiosError(err) && err.response?.status === 404) {
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

  useEffect(() => {
    if (!server) return;
    setSettingsName(server.name);
    setSettingsRam(server.ram);
    setSettingsCustomArgs(server.customArgs || '');
    setSettingsIcon(undefined);
  }, [server]);

  useEffect(() => {
    if (server?.status !== 'RUNNING') {
      return;
    }

    const nextSnapshot: StatSnapshot = {
      ts: Date.now(),
      cpu: stats.cpu,
      ramMb: stats.ram / 1024 / 1024,
    };

    setStatsHistory((prev) => {
      const cutoff = nextSnapshot.ts - MAX_HISTORY_WINDOW_MS;
      const pruned = prev.filter((item) => item.ts >= cutoff);
      return [...pruned, nextSnapshot];
    });
  }, [server?.status, stats.cpu, stats.ram]);

  useEffect(() => {
    const closePowerMenu = (event: MouseEvent) => {
      if (
        powerMenuRef.current &&
        !powerMenuRef.current.contains(event.target as Node)
      ) {
        setIsPowerMenuOpen(false);
      }
    };

    document.addEventListener('mousedown', closePowerMenu);
    return () => document.removeEventListener('mousedown', closePowerMenu);
  }, []);

  const handleStart = async () => {
    if (!server) return;
    try {
      setPowerAction('start');
      await api.startServer(server.id);
      setServer((prev) => (prev ? { ...prev, status: 'STARTING' } : null));
    } catch (err) {
      console.error(err);
    } finally {
      setPowerAction(null);
    }
  };

  const handleStop = async () => {
    if (!server) return;
    try {
      setPowerAction('stop');
      await api.stopServer(server.id);
      setServer((prev) => (prev ? { ...prev, status: 'STOPPING' } : null));
    } catch (err) {
      console.error(err);
    } finally {
      setPowerAction(null);
      setIsPowerMenuOpen(false);
    }
  };

  const handleRestart = async () => {
    if (!server) return;
    try {
      setPowerAction('restart');
      await api.restartServer(server.id);
      setServer((prev) => (prev ? { ...prev, status: 'STARTING' } : null));
    } catch (err) {
      console.error(err);
    } finally {
      setPowerAction(null);
      setIsPowerMenuOpen(false);
    }
  };

  const handleKill = async () => {
    if (!server) return;
    try {
      setPowerAction('kill');
      await api.killServer(server.id);
      setServer((prev) => (prev ? { ...prev, status: 'STOPPED' } : null));
    } catch (err) {
      console.error(err);
    } finally {
      setPowerAction(null);
      setIsPowerMenuOpen(false);
    }
  };

  const handleSaveSettings = async (data: {
    name: string;
    ram: number;
    customArgs?: string;
    icon?: File;
  }) => {
    if (!server) return;
    try {
      setIsSavingSettings(true);
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
    } finally {
      setIsSavingSettings(false);
    }
  };

  const handleCommandSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!commandInput.trim()) return;

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

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatDuration = (seconds: number) => {
    if (!seconds || seconds < 0) return '0s';
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = Math.floor(seconds % 60);

    if (hrs > 0) return `${hrs}h ${mins}m ${secs}s`;
    if (mins > 0) return `${mins}m ${secs}s`;
    return `${secs}s`;
  };

  const now = Date.now();
  const selectedRangeMs = RANGE_TO_MS[chartRange];
  const rangeStart = now - selectedRangeMs;

  const visibleHistory = useMemo(() => {
    const inRange = statsHistory.filter(
      (point) => point.ts >= rangeStart && point.ts <= now,
    );

    if (inRange.length > 0) {
      return inRange;
    }

    if (statsHistory.length > 0) {
      return [statsHistory[statsHistory.length - 1]];
    }

    return [];
  }, [now, rangeStart, statsHistory]);

  const ramDomainMax = useMemo(() => {
    const dataMax = visibleHistory.reduce(
      (acc, point) => Math.max(acc, point.ramMb),
      0,
    );
    return Math.max(server?.ram || 0, dataMax, 1);
  }, [server?.ram, visibleHistory]);

  const formatTimeTick = useCallback(
    (timestamp: number) => {
      const includeSeconds = selectedRangeMs <= RANGE_TO_MS['5m'];
      return new Date(timestamp).toLocaleTimeString([], {
        hour: '2-digit',
        minute: '2-digit',
        second: includeSeconds ? '2-digit' : undefined,
        hour12: false,
      });
    },
    [selectedRangeMs],
  );

  if (loading) return <div>Loading...</div>;
  if (!server) return <div>Server not found</div>;

  const address = `${publicIP}:${server.port}`;
  const isPowerDisabled =
    server.status === 'STARTING' || server.status === 'STOPPING';
  const isStoppedLike = server.status === 'STOPPED';
  const players = stats.players || [];

  return (
    <div className="server-v2">
      <header className="server-v2-header">
        <button
          className="server-v2-back"
          onClick={() => navigate('/')}
          title="Back to dashboard"
          type="button"
        >
          <ArrowLeft size={18} />
        </button>

        <div className="server-v2-identity">
          <div className="server-v2-icon-shell">
            {!iconError ? (
              <img
                src={`${api.getServerIconUrl(server.id)}?t=${iconRefreshKey}`}
                alt="Server Icon"
                onError={() => setIconError(true)}
                className="server-v2-icon"
              />
            ) : (
              <div className="server-v2-icon-placeholder">
                {server.name.charAt(0).toUpperCase()}
              </div>
            )}
          </div>

          <div className="server-v2-title-wrap">
            <div className="server-v2-title-row">
              <h1>{server.name}</h1>
              <span
                className={`server-v2-status status-${server.status.toLowerCase()}`}
              >
                {server.status}
              </span>
            </div>
            <div className="server-v2-meta-row">
              <span className="server-v2-loader">{server.loader}</span>
              <span>•</span>
              <span>{server.version}</span>
              <span>•</span>
              <span
                className="server-v2-address"
                onClick={() => copy(address)}
                title="Click to copy"
              >
                {address}
              </span>
              <CopyButton
                text={address}
                variant="secondary"
                title="Copy address"
                className="address-copy-btn"
              />
            </div>
          </div>
        </div>

        <div className="server-v2-actions">
          {(server.permissions?.canControlPower ||
            server.permissions?.canViewConsole) &&
            (isStoppedLike ? (
              <Button onClick={handleStart} disabled={powerAction === 'start'}>
                {powerAction === 'start' ? (
                  <LoaderCircle size={16} className="spin" />
                ) : (
                  <Play size={16} />
                )}
                Start
              </Button>
            ) : (
              <div className="server-v2-power-menu" ref={powerMenuRef}>
                <Button
                  variant="danger"
                  onClick={handleStop}
                  disabled={isPowerDisabled || powerAction !== null}
                >
                  {powerAction === 'stop' ? (
                    <LoaderCircle size={16} className="spin" />
                  ) : (
                    <Square size={16} />
                  )}
                  Stop
                </Button>
                <button
                  type="button"
                  className="server-v2-more-btn"
                  onClick={() => setIsPowerMenuOpen((prev) => !prev)}
                  disabled={isPowerDisabled || powerAction !== null}
                >
                  <MoreVertical size={16} />
                </button>
                {isPowerMenuOpen && (
                  <div className="server-v2-dropdown">
                    <button type="button" onClick={handleRestart}>
                      <RotateCcw size={15} /> Restart
                    </button>
                    <button type="button" onClick={handleKill}>
                      <Skull size={15} /> Kill
                    </button>
                  </div>
                )}
              </div>
            ))}

          {server.permissions?.canViewConsole && (
            <Button
              variant="secondary"
              onClick={() => setIsShareModalOpen(true)}
              title="Create Public Link"
            >
              <Share2 size={16} />
            </Button>
          )}
        </div>
      </header>

      <div className="server-v2-body">
        <section className="server-v2-content">
          {activeTab === 'performance' && (
            <div className="server-v2-grid">
              <div className="server-v2-card">
                <div className="server-v2-card-label">
                  <Cpu size={16} /> CPU Usage
                </div>
                <div className="server-v2-card-value">
                  {server.status === 'RUNNING'
                    ? `${stats.cpu.toFixed(1)}%`
                    : 'Offline'}
                </div>
              </div>
              <div className="server-v2-card">
                <div className="server-v2-card-label">
                  <MemoryStick size={16} /> RAM Usage
                </div>
                <div className="server-v2-card-value">
                  {server.status === 'RUNNING'
                    ? `${(stats.ram / 1024 / 1024).toFixed(0)} MB / ${server.ram} MB`
                    : 'Offline'}
                </div>
              </div>
              <div className="server-v2-card">
                <div className="server-v2-card-label">
                  <Clock3 size={16} /> Uptime
                </div>
                <div className="server-v2-card-value">
                  {server.status === 'RUNNING'
                    ? formatDuration(stats.uptimeSeconds)
                    : 'Offline'}
                </div>
              </div>
              <div className="server-v2-card">
                <div className="server-v2-card-label">
                  <HardDrive size={16} /> Disk
                </div>
                <div className="server-v2-card-value">{formatBytes(stats.disk)}</div>
              </div>

              <div className="server-v2-chart-card">
                <div className="server-v2-chart-header">
                  <h2>Performance Graph</h2>
                  <div className="server-v2-range-selector">
                    {(Object.keys(RANGE_TO_MS) as ChartRange[]).map((range) => (
                      <button
                        key={range}
                        type="button"
                        className={chartRange === range ? 'active' : ''}
                        onClick={() => setChartRange(range)}
                      >
                        {range}
                      </button>
                    ))}
                  </div>
                </div>

                <div className="server-v2-chart-shell">
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart
                      data={visibleHistory}
                      margin={{ top: 12, right: 24, left: 8, bottom: 4 }}
                    >
                      <CartesianGrid
                        strokeDasharray="3 3"
                        stroke="rgba(255,255,255,0.08)"
                      />
                      <XAxis
                        dataKey="ts"
                        type="number"
                        scale="time"
                        domain={[rangeStart, now]}
                        stroke="var(--text-muted)"
                        tickFormatter={formatTimeTick}
                        minTickGap={24}
                      />
                      <YAxis
                        yAxisId="cpu"
                        domain={[0, 100]}
                        tickFormatter={(value) => `${value}%`}
                        stroke="var(--text-muted)"
                        width={44}
                      />
                      <YAxis
                        yAxisId="ram"
                        orientation="right"
                        domain={[0, ramDomainMax]}
                        tickFormatter={(value) => `${Math.round(Number(value))}MB`}
                        stroke="var(--text-muted)"
                        width={60}
                      />
                      <Tooltip
                        labelFormatter={(value) => formatTimeTick(Number(value))}
                        formatter={(value, name) => {
                          const numericValue = Number(value ?? 0);
                          if (name === 'CPU %') {
                            return [`${numericValue.toFixed(1)}%`, name];
                          }
                          return [`${Math.round(numericValue)} MB`, name];
                        }}
                        contentStyle={{
                          background: '#1e1e1e',
                          border: '1px solid var(--border-color)',
                          borderRadius: '8px',
                        }}
                        labelStyle={{ color: 'var(--text-main)' }}
                      />
                      <Legend />
                      <Line
                        yAxisId="cpu"
                        type="monotone"
                        dataKey="cpu"
                        name="CPU %"
                        stroke="#60a5fa"
                        strokeWidth={2}
                        dot={false}
                        isAnimationActive={false}
                      />
                      <Line
                        yAxisId="ram"
                        type="monotone"
                        dataKey="ramMb"
                        name="RAM MB"
                        stroke="#a855f7"
                        strokeWidth={2}
                        dot={false}
                        isAnimationActive={false}
                      />
                    </LineChart>
                  </ResponsiveContainer>

                  {visibleHistory.length === 0 && (
                    <div className="server-v2-empty-chart">
                      Waiting for performance data...
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}

          {activeTab === 'console' && (
            <div className="server-v2-console-wrap">
              <div className="server-v2-console-header">
                <h2>Console</h2>
                <span className={isConnected ? 'ok' : 'bad'}>
                  {isConnected ? '● Connected' : '○ Disconnected'}
                </span>
              </div>

              <ConsoleView logs={logs} />

              <form
                onSubmit={handleCommandSubmit}
                className="server-v2-console-input"
              >
                <input
                  type="text"
                  value={commandInput}
                  onChange={(e) => setCommandInput(e.target.value)}
                  onKeyDown={handleKeyDown}
                  className="form-input"
                  placeholder="Type a command..."
                  disabled={!isConnected}
                />
                <Button
                  type="submit"
                  disabled={!isConnected || !commandInput.trim()}
                >
                  Send
                </Button>
              </form>
            </div>
          )}

          {activeTab === 'players' && (
            <div className="server-v2-players-card">
              <div className="server-v2-players-head">
                <h2>Online Players</h2>
                <span>
                  {stats.onlinePlayers}/{stats.maxPlayers}
                </span>
              </div>

              {players.length === 0 ? (
                <div className="server-v2-empty-players">No players found</div>
              ) : (
                <ul className="server-v2-player-list">
                  {players.map((player, idx) => (
                    <li key={`${player.id || player.name}-${idx}`}>
                      <PlayerAvatar player={player} />
                      <div>
                        <strong>{player.name}</strong>
                        <small>{player.id || 'No UUID available'}</small>
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}

          {activeTab === 'files' && <FileExplorer serverId={server.id} />}

          {activeTab === 'settings' && (
            <div className="server-v2-settings-card">
              <h2>Server Settings</h2>
              <p>Manage name, RAM, custom Java args, and server icon.</p>

              {user?.role === 'admin' ? (
                <form
                  className="server-v2-settings-form"
                  onSubmit={(e) => {
                    e.preventDefault();
                    if (!settingsName.trim()) return;
                    handleSaveSettings({
                      name: settingsName.trim(),
                      ram: settingsRam,
                      customArgs: settingsCustomArgs.trim() || undefined,
                      icon: settingsIcon,
                    });
                  }}
                >
                  <div className="server-v2-settings-grid">
                    <label>
                      <span>Name</span>
                      <input
                        className="form-input"
                        value={settingsName}
                        onChange={(e) => setSettingsName(e.target.value)}
                        required
                      />
                    </label>
                    <label>
                      <span>Loader</span>
                      <input
                        className="form-input"
                        value={server.loader}
                        disabled
                      />
                    </label>
                    <label>
                      <span>Version</span>
                      <input
                        className="form-input"
                        value={server.version}
                        disabled
                      />
                    </label>
                    <label>
                      <span>RAM (MB)</span>
                      <input
                        className="form-input"
                        type="number"
                        min={512}
                        step={128}
                        value={settingsRam}
                        onChange={(e) =>
                          setSettingsRam(Math.max(512, Number(e.target.value)))
                        }
                        required
                      />
                    </label>
                  </div>

                  <label className="server-v2-settings-field">
                    <span>Custom Java Args</span>
                    <input
                      className="form-input"
                      value={settingsCustomArgs}
                      onChange={(e) => setSettingsCustomArgs(e.target.value)}
                      placeholder="-XX:+UseG1GC"
                    />
                  </label>

                  <label className="server-v2-settings-field">
                    <span>Server Icon</span>
                    <div className="server-v2-file-upload">
                      <label className="server-v2-file-btn" htmlFor="server-icon">
                        Choose file
                      </label>
                      <input
                        id="server-icon"
                        type="file"
                        accept="image/png,image/jpeg"
                        onChange={(e) =>
                          setSettingsIcon(e.target.files?.[0] || undefined)
                        }
                      />
                      <span className="server-v2-file-name">
                        {settingsIcon ? settingsIcon.name : 'No file selected'}
                      </span>
                    </div>
                  </label>

                  <Button
                    type="submit"
                    disabled={isSavingSettings || !settingsName.trim()}
                  >
                    <Settings2 size={16} />
                    {isSavingSettings ? 'Saving...' : 'Save Settings'}
                  </Button>
                </form>
              ) : (
                <div className="server-v2-settings-grid">
                  <div>
                    <span>Name</span>
                    <strong>{server.name}</strong>
                  </div>
                  <div>
                    <span>Loader</span>
                    <strong>{server.loader}</strong>
                  </div>
                  <div>
                    <span>Version</span>
                    <strong>{server.version}</strong>
                  </div>
                  <div>
                    <span>RAM</span>
                    <strong>{server.ram} MB</strong>
                  </div>
                </div>
              )}
            </div>
          )}
        </section>

        <aside className="server-v2-sidebar">
          <button
            type="button"
            className={activeTab === 'performance' ? 'active' : ''}
            onClick={() => setActiveTab('performance')}
          >
            <BarChart3 size={16} />
            Performance
          </button>
          <button
            type="button"
            className={activeTab === 'console' ? 'active' : ''}
            onClick={() => setActiveTab('console')}
          >
            <Terminal size={16} />
            Console
          </button>
          <button
            type="button"
            className={activeTab === 'players' ? 'active' : ''}
            onClick={() => setActiveTab('players')}
          >
            <Users size={16} />
            Players
          </button>
          <button
            type="button"
            className={activeTab === 'files' ? 'active' : ''}
            onClick={() => setActiveTab('files')}
          >
            <HardDrive size={16} />
            Files
          </button>
          <button
            type="button"
            className={activeTab === 'settings' ? 'active' : ''}
            onClick={() => setActiveTab('settings')}
          >
            <Settings2 size={16} />
            Settings
          </button>
        </aside>
      </div>

      <ShareModal
        isOpen={isShareModalOpen}
        onClose={() => setIsShareModalOpen(false)}
        serverId={server.id}
      />
    </div>
  );
};

export default ServerDetail;
