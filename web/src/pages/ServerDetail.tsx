import axios from 'axios';
import {
  ArrowLeft,
  Ban,
  BarChart3,
  CircleHelp,
  Clock3,
  Cpu,
  Download,
  Gamepad2,
  Gauge,
  Globe,
  HardDrive,
  LoaderCircle,
  MemoryStick,
  MoreVertical,
  Package,
  Play,
  RotateCcw,
  Search,
  Settings2,
  Share2,
  Shield,
  Skull,
  Square,
  Terminal,
  Trash2,
  Upload,
  UserX,
  Users,
} from 'lucide-react';
import { useNavigate, useParams } from 'react-router-dom';
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

import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';

import AddonsPanel from '../components/AddonsPanel';
import ConsoleView from '../components/ConsoleView';
import FileExplorer from '../components/FileExplorer';
import ShareModal from '../components/ShareModal';
import { Button } from '../components/ui/Button';
import { CopyButton } from '../components/ui/CopyButton';
import { Modal } from '../components/ui/Modal';
import { useAuth } from '../context/AuthContext';
import { useConsole } from '../hooks/useConsole';
import { useCopy } from '../hooks/useCopy';
import { useServerStats } from '../hooks/useServerStats';
import { api } from '../services/api';
import type {
  PlayerInfo,
  Server,
  ServerSettings,
  ServerVersionUpdateResult,
} from '../types';

type DetailTab =
  | 'performance'
  | 'console'
  | 'players'
  | 'files'
  | 'addons'
  | 'settings';
type ChartRange = '1m' | '5m' | '30m' | '1h' | '4h';
type PlayerFilter = 'all' | 'admins' | 'banned';

interface StatSnapshot {
  ts: number;
  cpu: number;
  ramMb: number;
}

interface OperatorEntry {
  uuid?: string;
  name?: string;
  level?: number;
  bypassesPlayerLimit?: boolean;
}

interface BannedPlayerEntry {
  uuid?: string;
  name?: string;
  created?: string;
  source?: string;
  expires?: string;
  reason?: string;
}

interface BannedIPEntry {
  ip?: string;
  created?: string;
  source?: string;
  expires?: string;
  reason?: string;
}

interface PlayerListItem {
  key: string;
  name: string;
  uuid?: string;
  isOnline: boolean;
  source: 'online' | 'operator';
}

interface BannedListItem {
  key: string;
  type: 'player' | 'ip';
  label: string;
  uuid?: string;
  detail?: string;
}

interface SelectedPlayerAction extends PlayerInfo {
  isOnline: boolean;
  isOperator: boolean;
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
const SERVER_POLL_MS = 2000;
const SERVER_POLL_MAX_BACKOFF_MS = 30000;

const getAvatarUrl = (uuid?: string) =>
  `${MINEATAR_BASE_URL}/${encodeURIComponent(uuid || STEVE_UUID)}?scale=8&overlay=true`;

const parseMinecraftVersionParts = (value: string): number[] | null => {
  const normalized = value.trim().toLowerCase().replace(/^v/, '');
  if (normalized === '') return null;

  const parts = normalized.split('.');
  const parsed: number[] = [];

  for (const part of parts) {
    const digits = part.match(/^\d+/)?.[0];
    if (!digits) return null;
    parsed.push(Number(digits));
  }

  return parsed;
};

const compareMinecraftVersions = (a: string, b: string): number | null => {
  const left = parseMinecraftVersionParts(a);
  const right = parseMinecraftVersionParts(b);
  if (!left || !right) return null;

  const maxLen = Math.max(left.length, right.length);
  for (let i = 0; i < maxLen; i += 1) {
    const lv = left[i] ?? 0;
    const rv = right[i] ?? 0;
    if (lv > rv) return 1;
    if (lv < rv) return -1;
  }
  return 0;
};

const isFutureMinecraftVersion = (candidate: string, current: string) => {
  const comparison = compareMinecraftVersions(candidate, current);
  return comparison !== null && comparison > 0;
};

const normalizeServerSettings = (settings: ServerSettings): ServerSettings => ({
  ...settings,
  onlineMode: settings.onlineMode ?? true,
  spawnProtection:
    Number.isFinite(settings.spawnProtection) && settings.spawnProtection >= 0
      ? settings.spawnProtection
      : 16,
});

const upsertPropertyLine = (content: string, key: string, value: string) => {
  const normalized = content.replace(/\r\n/g, '\n');
  const lines = normalized.split('\n');
  let updated = false;
  const nextLines = lines.map((line) => {
    const trimmed = line.trim();
    if (trimmed === '' || trimmed.startsWith('#') || !line.includes('=')) {
      return line;
    }

    const [rawKey] = line.split('=', 1);
    if (rawKey.trim() !== key) {
      return line;
    }

    updated = true;
    return `${key}=${value}`;
  });

  if (!updated) {
    if (nextLines.length > 0 && nextLines[nextLines.length - 1] !== '') {
      nextLines.push('');
    }
    nextLines.push(`${key}=${value}`);
  }

  return nextLines.join('\n');
};

const readIntPropertyFromContent = (
  content: string,
  key: string,
): number | null => {
  const normalized = content.replace(/\r\n/g, '\n');
  const lines = normalized.split('\n');
  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed === '' || trimmed.startsWith('#') || !line.includes('=')) {
      continue;
    }
    const [rawKey, rawValue = ''] = line.split('=', 2);
    if (rawKey.trim() !== key) {
      continue;
    }
    const parsed = Number.parseInt(rawValue.trim(), 10);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
    return null;
  }
  return null;
};

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
  const [serverIconVersion, setServerIconVersion] = useState(() => Date.now());
  const [activeTab, setActiveTab] = useState<DetailTab>('performance');
  const [powerAction, setPowerAction] = useState<
    null | 'start' | 'stop' | 'restart' | 'kill'
  >(null);
  const [isPowerMenuOpen, setIsPowerMenuOpen] = useState(false);
  const [chartRange, setChartRange] = useState<ChartRange>('1m');
  const [statsHistory, setStatsHistory] = useState<StatSnapshot[]>([]);
  const [settingsDraft, setSettingsDraft] = useState<ServerSettings | null>(
    null,
  );
  const [settingsSnapshot, setSettingsSnapshot] =
    useState<ServerSettings | null>(null);
  const [isLoadingSettings, setIsLoadingSettings] = useState(false);
  const [isSavingSettings, setIsSavingSettings] = useState(false);
  const [isSettingsModalOpen, setIsSettingsModalOpen] = useState(false);
  const [settingsModalTitle, setSettingsModalTitle] = useState('');
  const [settingsModalMessage, setSettingsModalMessage] = useState('');
  const [selectedSettingsIcon, setSelectedSettingsIcon] = useState<File | null>(
    null,
  );
  const [settingsIconPreview, setSettingsIconPreview] = useState<string | null>(
    null,
  );
  const [settingsIconError, setSettingsIconError] = useState(false);
  const [isUploadingSettingsIcon, setIsUploadingSettingsIcon] = useState(false);
  const [isIconUploadModalOpen, setIsIconUploadModalOpen] = useState(false);
  const [iconUploadModalTitle, setIconUploadModalTitle] = useState('');
  const [iconUploadModalMessage, setIconUploadModalMessage] = useState('');
  const [versionOptions, setVersionOptions] = useState<string[]>([]);
  const [selectedVersion, setSelectedVersion] = useState('');
  const [isUpdatingVersion, setIsUpdatingVersion] = useState(false);
  const [isVersionUpdateModalOpen, setIsVersionUpdateModalOpen] =
    useState(false);
  const [versionUpdateModalTitle, setVersionUpdateModalTitle] = useState('');
  const [versionUpdateResult, setVersionUpdateResult] =
    useState<ServerVersionUpdateResult | null>(null);
  const [versionUpdateError, setVersionUpdateError] = useState('');
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [deleteConfirmName, setDeleteConfirmName] = useState('');
  const [isDeletingServer, setIsDeletingServer] = useState(false);
  const [playerFilter, setPlayerFilter] = useState<PlayerFilter>('all');
  const [playersSearch, setPlayersSearch] = useState('');
  const [operators, setOperators] = useState<OperatorEntry[]>([]);
  const [bannedPlayers, setBannedPlayers] = useState<BannedPlayerEntry[]>([]);
  const [bannedIps, setBannedIps] = useState<BannedIPEntry[]>([]);
  const [selectedPlayer, setSelectedPlayer] =
    useState<SelectedPlayerAction | null>(null);
  const [isPlayerActionsOpen, setIsPlayerActionsOpen] = useState(false);
  const [isPlayerActionLoading, setIsPlayerActionLoading] = useState(false);
  const [publicIP, setPublicIP] = useState<string>(
    typeof window !== 'undefined' ? window.location.hostname : 'localhost',
  );
  const powerMenuRef = useRef<HTMLDivElement>(null);
  const settingsIconInputRef = useRef<HTMLInputElement>(null);
  const chartShellRef = useRef<HTMLDivElement>(null);
  const serverPollDelayRef = useRef(SERVER_POLL_MS);
  const hasLoggedServerOfflineRef = useRef(false);
  const [chartSize, setChartSize] = useState({ width: 0, height: 0 });

  const { logs, sendCommand, isConnected } = useConsole(id || '');
  const { stats, isOffline: isStatsOffline } = useServerStats(
    id || '',
    server?.status === 'RUNNING',
  );
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

  const fetchServer = useCallback(async (): Promise<boolean> => {
    if (!id) return false;
    try {
      const res = await api.getServer(id);
      setServer(res.data);
      hasLoggedServerOfflineRef.current = false;
      serverPollDelayRef.current = SERVER_POLL_MS;
      return true;
    } catch (err) {
      if (!hasLoggedServerOfflineRef.current) {
        hasLoggedServerOfflineRef.current = true;
        console.warn('Server unavailable, retrying with backoff.');
      }
      serverPollDelayRef.current = Math.min(
        SERVER_POLL_MAX_BACKOFF_MS,
        serverPollDelayRef.current * 2,
      );
      if (axios.isAxiosError(err) && err.response?.status === 404) {
        setServer(null);
      }
      return false;
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    if (!id) return;

    let cancelled = false;
    let timer: number | null = null;

    const scheduleNext = (delay: number) => {
      if (cancelled) return;
      timer = window.setTimeout(async () => {
        const ok = await fetchServer();
        if (!cancelled) {
          scheduleNext(ok ? SERVER_POLL_MS : serverPollDelayRef.current);
        }
      }, delay);
    };

    scheduleNext(0);

    return () => {
      cancelled = true;
      if (timer !== null) {
        window.clearTimeout(timer);
      }
    };
  }, [fetchServer]);

  useEffect(() => {
    if (server?.status === 'STOPPED') {
      setCommandHistory([]);
      setHistoryIndex(-1);
    }
  }, [server?.status]);

  useEffect(() => {
    setSelectedSettingsIcon(null);
    setSettingsIconPreview(null);
    setSettingsIconError(false);
  }, [server?.id]);

  const readJsonList = useCallback(
    async <T,>(path: string): Promise<T[]> => {
      if (!id) return [];
      try {
        const res = await api.getFileContent(id, path);
        const parsed = JSON.parse(String(res.data));
        return Array.isArray(parsed) ? (parsed as T[]) : [];
      } catch (err) {
        console.debug(`Unable to read ${path}:`, err);
        return [];
      }
    },
    [id],
  );

  const refreshPlayerLists = useCallback(async () => {
    const [nextOps, nextBannedPlayers, nextBannedIps] = await Promise.all([
      readJsonList<OperatorEntry>('/ops.json'),
      readJsonList<BannedPlayerEntry>('/banned-players.json'),
      readJsonList<BannedIPEntry>('/banned-ips.json'),
    ]);
    setOperators(nextOps);
    setBannedPlayers(nextBannedPlayers);
    setBannedIps(nextBannedIps);
  }, [readJsonList]);

  useEffect(() => {
    if (!id || activeTab !== 'players') return;

    refreshPlayerLists();
    const interval = setInterval(refreshPlayerLists, 8000);
    return () => clearInterval(interval);
  }, [activeTab, id, refreshPlayerLists]);

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

  useEffect(() => {
    const node = chartShellRef.current;
    if (!node) return;

    const updateSize = () => {
      const width = node.clientWidth;
      const height = node.clientHeight;
      setChartSize((prev) =>
        prev.width === width && prev.height === height
          ? prev
          : { width, height },
      );
    };

    updateSize();

    const observer = new ResizeObserver(() => updateSize());
    observer.observe(node);
    return () => observer.disconnect();
  }, [activeTab, loading]);

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

  const fetchSettingsData = useCallback(async () => {
    if (!id || user?.role !== 'admin') return;
    try {
      setIsLoadingSettings(true);
      const [settingsRes, versionsRes] = await Promise.all([
        api.getServerSettings(id),
        api.getServerVersionOptions(id),
      ]);
      let normalizedSettings = normalizeServerSettings(settingsRes.data);
      if (
        !Number.isFinite(settingsRes.data.spawnProtection) ||
        (settingsRes.data.spawnProtection ?? -1) < 0
      ) {
        try {
          const fileRes = await api.getFileContent(id, '/server.properties');
          const rawContent = String(fileRes.data ?? '');
          const fromFile = readIntPropertyFromContent(
            rawContent,
            'spawn-protection',
          );
          if (fromFile !== null && fromFile >= 0) {
            normalizedSettings = {
              ...normalizedSettings,
              spawnProtection: fromFile,
            };
          }
        } catch (err) {
          console.debug(
            'Unable to read spawn-protection from server.properties fallback:',
            err,
          );
        }
      }
      const currentVersion = settingsRes.data.version || '';
      const futureVersions = (versionsRes.data.versions || []).filter(
        (version) => isFutureMinecraftVersion(version, currentVersion),
      );
      setSettingsSnapshot(normalizedSettings);
      setSettingsDraft(normalizedSettings);
      setVersionOptions(futureVersions);
      setSelectedVersion(futureVersions[0] || '');
    } catch (err) {
      console.error('Failed to load server settings:', err);
    } finally {
      setIsLoadingSettings(false);
    }
  }, [id, user?.role]);

  useEffect(() => {
    if (activeTab === 'settings' && user?.role === 'admin') {
      fetchSettingsData();
    }
  }, [activeTab, fetchSettingsData, user?.role]);

  const handleSaveSettings = async () => {
    if (!server || !settingsDraft) return;
    try {
      setIsSavingSettings(true);
      const expectedSpawnProtection = settingsDraft.spawnProtection;
      await api.updateServerSettings(server.id, settingsDraft);

      try {
        const refreshed = await api.getServerSettings(server.id);
        const normalized = normalizeServerSettings(refreshed.data);
        if (normalized.spawnProtection !== expectedSpawnProtection) {
          const fileRes = await api.getFileContent(
            server.id,
            '/server.properties',
          );
          const rawContent = String(fileRes.data ?? '');
          const patched = upsertPropertyLine(
            rawContent,
            'spawn-protection',
            String(expectedSpawnProtection),
          );
          if (patched !== rawContent) {
            await api.saveFileContent(server.id, '/server.properties', patched);
          }
        }
      } catch (err) {
        console.warn(
          'Unable to verify spawn-protection via settings API, skipping fallback patch:',
          err,
        );
      }

      await Promise.all([fetchServer(), fetchSettingsData()]);
      setSettingsModalTitle('Settings Saved');
      setSettingsModalMessage('Settings saved successfully.');
      setIsSettingsModalOpen(true);
    } catch (err) {
      console.error('Failed to save settings:', err);
      let errorMessage = 'Failed to save settings.';
      if (axios.isAxiosError(err)) {
        const responseMessage =
          typeof err.response?.data === 'string'
            ? err.response.data
            : (err.response?.data as { error?: string; message?: string })
                ?.error ||
              (err.response?.data as { error?: string; message?: string })
                ?.message;
        if (responseMessage) {
          errorMessage = responseMessage;
        }
      }
      setSettingsModalTitle('Save Failed');
      setSettingsModalMessage(errorMessage);
      setIsSettingsModalOpen(true);
    } finally {
      setIsSavingSettings(false);
    }
  };

  const handleSettingsIconSelected = (
    e: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setSelectedSettingsIcon(file);
    const reader = new FileReader();
    reader.onloadend = () => {
      setSettingsIconPreview(reader.result as string);
      setSettingsIconError(false);
    };
    reader.readAsDataURL(file);
  };

  const handleUploadSettingsIcon = async () => {
    if (!server || !selectedSettingsIcon) return;
    try {
      setIsUploadingSettingsIcon(true);
      await api.uploadServerIcon(server.id, selectedSettingsIcon);
      setServerIconVersion(Date.now());
      setIconError(false);
      setSettingsIconError(false);
      setSelectedSettingsIcon(null);
      setSettingsIconPreview(null);
      if (settingsIconInputRef.current) {
        settingsIconInputRef.current.value = '';
      }
      setIconUploadModalTitle('Icon Updated');
      setIconUploadModalMessage('Server icon uploaded successfully.');
      setIsIconUploadModalOpen(true);
    } catch (err) {
      console.error('Failed to upload server icon:', err);
      let errorMessage = 'Failed to upload server icon.';
      if (axios.isAxiosError(err)) {
        const responseMessage =
          typeof err.response?.data === 'string'
            ? err.response.data
            : (err.response?.data as { error?: string; message?: string })
                ?.error ||
              (err.response?.data as { error?: string; message?: string })
                ?.message;
        if (responseMessage) {
          errorMessage = responseMessage;
        }
      }
      setIconUploadModalTitle('Icon Upload Failed');
      setIconUploadModalMessage(errorMessage);
      setIsIconUploadModalOpen(true);
    } finally {
      setIsUploadingSettingsIcon(false);
    }
  };

  const handleVersionUpdate = async () => {
    if (!server || !selectedVersion) return;
    try {
      setVersionUpdateResult(null);
      setVersionUpdateError('');
      setVersionUpdateModalTitle('Updating Server Version');
      setIsVersionUpdateModalOpen(true);
      setIsUpdatingVersion(true);
      const result = await api.updateServerVersion(server.id, {
        version: selectedVersion,
        includeDependencies: true,
      });
      await Promise.all([fetchServer(), fetchSettingsData()]);
      setVersionUpdateModalTitle('Version Updated');
      setVersionUpdateResult(result.data);
    } catch (err) {
      console.error('Failed to update server version:', err);
      let errorMessage = 'Failed to update server version.';
      if (axios.isAxiosError(err)) {
        const responseMessage =
          typeof err.response?.data === 'string'
            ? err.response.data
            : (err.response?.data as { error?: string; message?: string })
                ?.error ||
              (err.response?.data as { error?: string; message?: string })
                ?.message;
        if (responseMessage) {
          errorMessage = responseMessage;
        }
      }
      setVersionUpdateModalTitle('Version Update Failed');
      setVersionUpdateError(errorMessage);
    } finally {
      setIsUpdatingVersion(false);
    }
  };

  const handleDeleteServer = async () => {
    if (!server) return;
    try {
      setIsDeletingServer(true);
      await api.deleteServer(server.id);
      setIsDeleteModalOpen(false);
      navigate('/');
    } catch (err) {
      console.error('Failed to delete server:', err);
      alert('Failed to delete server.');
    } finally {
      setIsDeletingServer(false);
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

  const canEditSettings = user?.role === 'admin';
  const isServerStopped = server?.status === 'STOPPED';
  const canApplySettings = canEditSettings && isServerStopped;
  const supportsAddons = ['paper', 'fabric', 'forge', 'neoforge'].includes(
    server?.loader || '',
  );
  const addonsLabel = server?.loader === 'paper' ? 'Plugins' : 'Mods';

  const renderVersionUpdateProgress = () => (
    <div className="server-v2-version-update-progress">
      <div className="server-v2-version-update-spinner">
        <LoaderCircle size={28} className="spin" />
      </div>
      <div>
        <strong>Updating to {selectedVersion}</strong>
        <p>Creating a backup, updating the server, and checking addons.</p>
      </div>
      <div className="server-v2-version-update-steps">
        <span>Backup server files</span>
        <span>Install new server version</span>
        <span>Update or disable incompatible addons</span>
      </div>
    </div>
  );

  const renderVersionUpdateResult = () => {
    if (!versionUpdateResult) return null;
    const addons = versionUpdateResult.addons;
    const updatedCount = addons?.updated.length ?? 0;
    const disabledCount = addons?.disabled.length ?? 0;
    const failedCount = addons?.failed.length ?? 0;

    return (
      <div className="server-v2-version-update-result">
        <div className="server-v2-version-update-hero success">
          <Shield size={22} />
          <div>
            <strong>Server updated to {versionUpdateResult.version}</strong>
            <span>Backup created before applying changes.</span>
          </div>
        </div>
        <div className="server-v2-version-update-backup">
          <span>Backup</span>
          <code>{versionUpdateResult.backupName}</code>
        </div>
        <div className="server-v2-version-update-grid">
          <div>
            <strong>{updatedCount}</strong>
            <span>Addons updated</span>
          </div>
          <div>
            <strong>{disabledCount}</strong>
            <span>Addons disabled</span>
          </div>
          <div>
            <strong>{failedCount}</strong>
            <span>Addon failures</span>
          </div>
        </div>
        {failedCount > 0 && addons && (
          <div className="server-v2-version-update-failures">
            {addons.failed.slice(0, 3).map((failure) => (
              <p key={failure.id}>
                <strong>{failure.name || failure.id}:</strong> {failure.reason}
              </p>
            ))}
          </div>
        )}
      </div>
    );
  };
  const isDeleteNameMatch =
    deleteConfirmName.trim() !== '' &&
    deleteConfirmName.trim() === (server?.name || '');
  const isServerOnlineForChart =
    server?.status === 'RUNNING' && !isStatsOffline;
  const chartOverlayMessage = !isServerOnlineForChart
    ? null
    : chartSize.width <= 0 || chartSize.height <= 0
      ? 'Preparing chart layout...'
      : visibleHistory.length === 0
        ? 'Waiting for performance data...'
        : null;

  const updateSettingsField = <K extends keyof ServerSettings>(
    field: K,
    value: ServerSettings[K],
  ) => {
    setSettingsDraft((prev) => (prev ? { ...prev, [field]: value } : prev));
  };

  const players = stats.players || [];
  const canModeratePlayers = Boolean(server?.permissions?.canViewConsole);
  const normalizedSearch = playersSearch.trim().toLowerCase();

  const onlineNameSet = useMemo(
    () => new Set(players.map((player) => player.name.toLowerCase())),
    [players],
  );

  const onlineIdSet = useMemo(
    () =>
      new Set(
        players
          .map((player) => player.id)
          .filter(Boolean)
          .map((playerId) => playerId.toLowerCase()),
      ),
    [players],
  );

  const onlineItems = useMemo<PlayerListItem[]>(
    () =>
      players.map((player, idx) => ({
        key: `${player.id || player.name}-${idx}`,
        name: player.name,
        uuid: player.id,
        isOnline: true,
        source: 'online',
      })),
    [players],
  );

  const operatorItems = useMemo<PlayerListItem[]>(
    () =>
      operators.map((operator, idx) => {
        const normalizedName = operator.name?.toLowerCase();
        const normalizedUuid = operator.uuid?.toLowerCase();
        const isOnline = Boolean(
          (normalizedName && onlineNameSet.has(normalizedName)) ||
          (normalizedUuid && onlineIdSet.has(normalizedUuid)),
        );
        const fallbackName =
          operator.name || operator.uuid || `Operator ${idx + 1}`;
        return {
          key: `op-${operator.uuid || operator.name || idx}`,
          name: fallbackName,
          uuid: operator.uuid,
          isOnline,
          source: 'operator',
        };
      }),
    [operators, onlineIdSet, onlineNameSet],
  );

  const operatorNameSet = useMemo(
    () =>
      new Set(
        operators
          .map((operator) => operator.name)
          .filter(Boolean)
          .map((name) => String(name).toLowerCase()),
      ),
    [operators],
  );

  const operatorUuidSet = useMemo(
    () =>
      new Set(
        operators
          .map((operator) => operator.uuid)
          .filter(Boolean)
          .map((uuid) => String(uuid).toLowerCase()),
      ),
    [operators],
  );

  const bannedItems = useMemo<BannedListItem[]>(
    () => [
      ...bannedPlayers.map((bannedPlayer, idx) => ({
        key: `bp-${bannedPlayer.uuid || bannedPlayer.name || idx}`,
        type: 'player' as const,
        label:
          bannedPlayer.name || bannedPlayer.uuid || `Banned player ${idx + 1}`,
        uuid: bannedPlayer.uuid,
        detail: bannedPlayer.reason || bannedPlayer.source,
      })),
      ...bannedIps.map((bannedIp, idx) => ({
        key: `bi-${bannedIp.ip || idx}`,
        type: 'ip' as const,
        label: bannedIp.ip || `Banned IP ${idx + 1}`,
        detail: bannedIp.reason || bannedIp.source,
      })),
    ],
    [bannedIps, bannedPlayers],
  );

  const filteredOnlineItems = useMemo(
    () =>
      onlineItems.filter((player) => {
        if (!normalizedSearch) return true;
        return (
          player.name.toLowerCase().includes(normalizedSearch) ||
          player.uuid?.toLowerCase().includes(normalizedSearch)
        );
      }),
    [normalizedSearch, onlineItems],
  );

  const filteredOperatorItems = useMemo(() => {
    const sortedOperators = [...operatorItems].sort((a, b) => {
      if (a.isOnline !== b.isOnline) {
        return a.isOnline ? -1 : 1;
      }
      return a.name.localeCompare(b.name);
    });

    return sortedOperators.filter((player) => {
      if (!normalizedSearch) return true;
      return (
        player.name.toLowerCase().includes(normalizedSearch) ||
        player.uuid?.toLowerCase().includes(normalizedSearch)
      );
    });
  }, [normalizedSearch, operatorItems]);

  const filteredBannedItems = useMemo(
    () =>
      bannedItems.filter((entry) => {
        if (!normalizedSearch) return true;
        return (
          entry.label.toLowerCase().includes(normalizedSearch) ||
          entry.detail?.toLowerCase().includes(normalizedSearch) ||
          entry.uuid?.toLowerCase().includes(normalizedSearch)
        );
      }),
    [bannedItems, normalizedSearch],
  );

  const selectedPlayerCanDeleteData = Boolean(selectedPlayer?.id?.trim());

  const queuePlayerDataRefresh = () => {
    setTimeout(() => {
      refreshPlayerLists();
    }, 1200);
  };

  const runConsoleAction = async (command: string) => {
    if (!canModeratePlayers) {
      alert('You do not have permission to perform this action.');
      return;
    }
    if (!isConnected) {
      alert('Console is disconnected. Please wait and try again.');
      return;
    }

    setIsPlayerActionLoading(true);
    try {
      sendCommand(command);
      queuePlayerDataRefresh();
    } finally {
      setIsPlayerActionLoading(false);
    }
  };

  const handleDeletePlayerData = async () => {
    if (!id || !selectedPlayer || !selectedPlayer.id || !canModeratePlayers) {
      return;
    }

    const confirmed = window.confirm(
      `Delete ${selectedPlayer.name} playerdata file from this server?`,
    );
    if (!confirmed) return;

    setIsPlayerActionLoading(true);
    try {
      await api.deleteFile(id, `/world/playerdata/${selectedPlayer.id}.dat`);
      alert('Player data deleted successfully.');
      setIsPlayerActionsOpen(false);
    } catch (err) {
      console.error('Failed to delete player data:', err);
      alert('Failed to delete player data.');
    } finally {
      setIsPlayerActionLoading(false);
    }
  };

  const handlePardon = async (item: BannedListItem) => {
    if (!canModeratePlayers) return;
    if (item.type === 'player') {
      await runConsoleAction(`pardon ${item.label}`);
    } else {
      await runConsoleAction(`pardon-ip ${item.label}`);
    }
  };

  if (loading) return <div>Loading...</div>;
  if (!server) return <div>Server not found</div>;

  const address = `${publicIP}:${server.port}`;
  const isPowerDisabled =
    server.status === 'STARTING' || server.status === 'STOPPING';
  const isStoppedLike = server.status === 'STOPPED';

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
                src={`${api.getServerIconUrl(server.id)}?v=${serverIconVersion}`}
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
                <div className="server-v2-card-value">
                  {formatBytes(stats.disk)}
                </div>
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

                <div className="server-v2-chart-shell" ref={chartShellRef}>
                  {chartSize.width > 0 && chartSize.height > 0 ? (
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
                          tickFormatter={(value) =>
                            `${Math.round(Number(value))}MB`
                          }
                          stroke="var(--text-muted)"
                          width={60}
                        />
                        <Tooltip
                          labelFormatter={(value) =>
                            formatTimeTick(Number(value))
                          }
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
                  ) : null}

                  {chartOverlayMessage && (
                    <div className="server-v2-empty-chart">
                      {chartOverlayMessage}
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
                <h2>Player Management</h2>
                <span>
                  Online {stats.onlinePlayers}/{stats.maxPlayers}
                </span>
              </div>

              <div className="server-v2-players-controls">
                <label className="server-v2-players-search">
                  <Search size={16} />
                  <input
                    type="text"
                    value={playersSearch}
                    onChange={(e) => setPlayersSearch(e.target.value)}
                    placeholder="Search players..."
                  />
                </label>

                <div className="server-v2-players-filters">
                  <button
                    type="button"
                    className={playerFilter === 'all' ? 'active' : ''}
                    onClick={() => setPlayerFilter('all')}
                  >
                    All ({onlineItems.length})
                  </button>
                  <button
                    type="button"
                    className={playerFilter === 'admins' ? 'active' : ''}
                    onClick={() => setPlayerFilter('admins')}
                  >
                    Admins ({operatorItems.length})
                  </button>
                  <button
                    type="button"
                    className={playerFilter === 'banned' ? 'active' : ''}
                    onClick={() => setPlayerFilter('banned')}
                  >
                    Banned ({bannedItems.length})
                  </button>
                </div>
              </div>

              {playerFilter === 'all' &&
                (filteredOnlineItems.length === 0 ? (
                  <div className="server-v2-empty-players">
                    No players found
                  </div>
                ) : (
                  <ul className="server-v2-player-list">
                    {filteredOnlineItems.map((player) => (
                      <li
                        key={player.key}
                        className={canModeratePlayers ? 'clickable' : ''}
                        onClick={() => {
                          if (!canModeratePlayers) return;
                          const normalizedName = player.name.toLowerCase();
                          const normalizedUuid = player.uuid?.toLowerCase();
                          const isOperator = Boolean(
                            operatorNameSet.has(normalizedName) ||
                            (normalizedUuid &&
                              operatorUuidSet.has(normalizedUuid)),
                          );
                          setSelectedPlayer({
                            name: player.name,
                            id: player.uuid || '',
                            isOnline: true,
                            isOperator,
                          });
                          setIsPlayerActionsOpen(true);
                        }}
                      >
                        <PlayerAvatar
                          player={{ name: player.name, id: player.uuid || '' }}
                        />
                        <div>
                          <strong>{player.name}</strong>
                          <small>{player.uuid || 'No UUID available'}</small>
                        </div>
                        <span className="server-v2-player-badge online">
                          Online
                        </span>
                      </li>
                    ))}
                  </ul>
                ))}

              {playerFilter === 'admins' &&
                (filteredOperatorItems.length === 0 ? (
                  <div className="server-v2-empty-players">
                    No operators found
                  </div>
                ) : (
                  <ul className="server-v2-player-list">
                    {filteredOperatorItems.map((operator) => (
                      <li
                        key={operator.key}
                        className={canModeratePlayers ? 'clickable' : ''}
                        onClick={() => {
                          if (!canModeratePlayers) return;
                          setSelectedPlayer({
                            name: operator.name,
                            id: operator.uuid || '',
                            isOnline: operator.isOnline,
                            isOperator: true,
                          });
                          setIsPlayerActionsOpen(true);
                        }}
                      >
                        <PlayerAvatar
                          player={{
                            name: operator.name,
                            id: operator.uuid || '',
                          }}
                        />
                        <div>
                          <strong>{operator.name}</strong>
                          <small>{operator.uuid || 'No UUID available'}</small>
                        </div>
                        <span
                          className={`server-v2-player-badge ${operator.isOnline ? 'online' : 'offline'}`}
                        >
                          {operator.isOnline ? 'Online' : 'Offline'}
                        </span>
                      </li>
                    ))}
                  </ul>
                ))}

              {playerFilter === 'banned' &&
                (filteredBannedItems.length === 0 ? (
                  <div className="server-v2-empty-players">
                    No banned entries
                  </div>
                ) : (
                  <ul className="server-v2-player-list">
                    {filteredBannedItems.map((item) => (
                      <li key={item.key}>
                        <div className="server-v2-player-icon">
                          {item.type === 'player' ? (
                            <Ban size={16} />
                          ) : (
                            <Globe size={16} />
                          )}
                        </div>
                        <div>
                          <strong>{item.label}</strong>
                          <small>
                            {item.type === 'player'
                              ? item.uuid || item.detail || 'Banned player'
                              : item.detail || 'Banned IP'}
                          </small>
                        </div>
                        <button
                          type="button"
                          className="server-v2-pardon-btn"
                          onClick={() => handlePardon(item)}
                          disabled={
                            !canModeratePlayers || isPlayerActionLoading
                          }
                        >
                          Pardon
                        </button>
                      </li>
                    ))}
                  </ul>
                ))}

              {!canModeratePlayers && (
                <p className="server-v2-players-note">
                  You can view players, but moderation actions require console
                  permission.
                </p>
              )}
            </div>
          )}

          {activeTab === 'files' && <FileExplorer serverId={server.id} />}

          {activeTab === 'addons' && supportsAddons && (
            <AddonsPanel server={server} canManage={canModeratePlayers} />
          )}

          {activeTab === 'settings' && (
            <div className="server-v2-settings-layout">
              {!canEditSettings ? (
                <div className="server-v2-settings-card">
                  <h2>Server Settings</h2>
                  <p>Only admins can edit server settings.</p>
                </div>
              ) : isLoadingSettings || !settingsDraft ? (
                <div className="server-v2-settings-card">
                  <p>Loading settings...</p>
                </div>
              ) : (
                <>
                  <div className="server-v2-settings-dual-grid">
                    <div className="server-v2-settings-panel">
                      <div className="server-v2-settings-panel-head">
                        <div className="server-v2-settings-panel-icon">
                          <Gamepad2 size={18} />
                        </div>
                        <div>
                          <h3>Gameplay</h3>
                          <p>General gameplay settings</p>
                        </div>
                      </div>

                      <div className="server-v2-settings-form-grid">
                        <label>
                          <span>
                            Game Mode
                            <span title="Default game mode for new players.">
                              <CircleHelp size={14} />
                            </span>
                          </span>
                          <select
                            className="form-input"
                            value={settingsDraft.gamemode}
                            onChange={(e) =>
                              updateSettingsField(
                                'gamemode',
                                e.target.value as ServerSettings['gamemode'],
                              )
                            }
                            disabled={!canApplySettings}
                          >
                            <option value="survival">Survival</option>
                            <option value="creative">Creative</option>
                            <option value="adventure">Adventure</option>
                            <option value="spectator">Spectator</option>
                          </select>
                        </label>

                        <label>
                          <span>
                            Difficulty
                            <span title="World difficulty level.">
                              <CircleHelp size={14} />
                            </span>
                          </span>
                          <select
                            className="form-input"
                            value={settingsDraft.difficulty}
                            onChange={(e) =>
                              updateSettingsField(
                                'difficulty',
                                e.target.value as ServerSettings['difficulty'],
                              )
                            }
                            disabled={!canApplySettings}
                          >
                            <option value="peaceful">Peaceful</option>
                            <option value="easy">Easy</option>
                            <option value="normal">Normal</option>
                            <option value="hard">Hard</option>
                          </select>
                        </label>

                        <label className="server-v2-settings-full">
                          <span>
                            Server Message (MOTD)
                            <span title="Server list message players see.">
                              <CircleHelp size={14} />
                            </span>
                          </span>
                          <input
                            className="form-input"
                            value={settingsDraft.motd}
                            onChange={(e) =>
                              updateSettingsField('motd', e.target.value)
                            }
                            disabled={!canApplySettings}
                          />
                        </label>

                        <label>
                          <span>
                            Spawn Protection
                            <span title="Spawn protection radius in blocks. Set 0 to disable.">
                              <CircleHelp size={14} />
                            </span>
                          </span>
                          <input
                            className="form-input"
                            type="number"
                            min={0}
                            value={settingsDraft.spawnProtection ?? 16}
                            onChange={(e) =>
                              updateSettingsField(
                                'spawnProtection',
                                Math.max(0, Number(e.target.value)),
                              )
                            }
                            disabled={!canApplySettings}
                          />
                        </label>

                        <label className="server-v2-settings-toggle">
                          <input
                            type="checkbox"
                            checked={settingsDraft.onlineMode}
                            onChange={(e) =>
                              updateSettingsField(
                                'onlineMode',
                                e.target.checked,
                              )
                            }
                            disabled={!canApplySettings}
                          />
                          <span title="Verify player accounts with Mojang/Microsoft authentication.">
                            Online Mode
                          </span>
                        </label>

                        <label className="server-v2-settings-toggle">
                          <input
                            type="checkbox"
                            checked={settingsDraft.pvp}
                            onChange={(e) =>
                              updateSettingsField('pvp', e.target.checked)
                            }
                            disabled={!canApplySettings}
                          />
                          <span>Enable PvP</span>
                        </label>

                        <label className="server-v2-settings-toggle">
                          <input
                            type="checkbox"
                            checked={settingsDraft.allowFlight}
                            onChange={(e) =>
                              updateSettingsField(
                                'allowFlight',
                                e.target.checked,
                              )
                            }
                            disabled={!canApplySettings}
                          />
                          <span>Allow Flying</span>
                        </label>

                        <label className="server-v2-settings-toggle">
                          <input
                            type="checkbox"
                            checked={settingsDraft.enableCommandBlock}
                            onChange={(e) =>
                              updateSettingsField(
                                'enableCommandBlock',
                                e.target.checked,
                              )
                            }
                            disabled={!canApplySettings}
                          />
                          <span>Enable Command Blocks</span>
                        </label>

                        <label className="server-v2-settings-toggle">
                          <input
                            type="checkbox"
                            checked={settingsDraft.hardcore}
                            onChange={(e) =>
                              updateSettingsField('hardcore', e.target.checked)
                            }
                            disabled={!canApplySettings}
                          />
                          <span>Hardcore Mode</span>
                        </label>
                      </div>
                    </div>

                    <div className="server-v2-settings-panel">
                      <div className="server-v2-settings-panel-head">
                        <div className="server-v2-settings-panel-icon">
                          <Gauge size={18} />
                        </div>
                        <div>
                          <h3>Performance</h3>
                          <p>Optimize server performance settings</p>
                        </div>
                      </div>

                      <div className="server-v2-settings-form-grid">
                        <label>
                          <span>
                            Max Players
                            <span title="Maximum connected players.">
                              <CircleHelp size={14} />
                            </span>
                          </span>
                          <input
                            className="form-input"
                            type="number"
                            min={1}
                            max={1000}
                            value={settingsDraft.maxPlayers}
                            onChange={(e) =>
                              updateSettingsField(
                                'maxPlayers',
                                Math.max(1, Number(e.target.value)),
                              )
                            }
                            disabled={!canApplySettings}
                          />
                        </label>

                        <label>
                          <span>
                            View Distance
                            <span title="Chunks sent to players.">
                              <CircleHelp size={14} />
                            </span>
                          </span>
                          <input
                            className="form-input"
                            type="number"
                            min={2}
                            max={32}
                            value={settingsDraft.viewDistance}
                            onChange={(e) =>
                              updateSettingsField(
                                'viewDistance',
                                Math.max(2, Number(e.target.value)),
                              )
                            }
                            disabled={!canApplySettings}
                          />
                        </label>

                        <label>
                          <span>
                            Simulation Distance
                            <span title="Chunks actively simulated.">
                              <CircleHelp size={14} />
                            </span>
                          </span>
                          <input
                            className="form-input"
                            type="number"
                            min={2}
                            max={32}
                            value={settingsDraft.simulationDistance}
                            onChange={(e) =>
                              updateSettingsField(
                                'simulationDistance',
                                Math.max(2, Number(e.target.value)),
                              )
                            }
                            disabled={!canApplySettings}
                          />
                        </label>

                        <label>
                          <span>RAM Allocation (MB)</span>
                          <input
                            className="form-input"
                            type="number"
                            min={512}
                            max={262144}
                            step={256}
                            value={settingsDraft.ram}
                            onChange={(e) =>
                              updateSettingsField(
                                'ram',
                                Math.max(512, Number(e.target.value)),
                              )
                            }
                            disabled={!canApplySettings}
                          />
                        </label>

                        <label>
                          <span>
                            Memory Slider{' '}
                            <strong>{settingsDraft.ram} MB</strong>
                          </span>
                          <input
                            type="range"
                            min={512}
                            max={Math.max(16384, settingsDraft.ram + 1024)}
                            step={256}
                            value={settingsDraft.ram}
                            onChange={(e) =>
                              updateSettingsField('ram', Number(e.target.value))
                            }
                            disabled={!canApplySettings}
                          />
                        </label>
                      </div>
                    </div>
                  </div>

                  <div className="server-v2-settings-toolbar">
                    <Button
                      type="button"
                      onClick={handleSaveSettings}
                      disabled={
                        isSavingSettings ||
                        !canApplySettings ||
                        !settingsSnapshot ||
                        JSON.stringify(settingsSnapshot) ===
                          JSON.stringify(settingsDraft)
                      }
                    >
                      <Settings2 size={16} />
                      {isSavingSettings ? 'Saving...' : 'Save Settings'}
                    </Button>
                    {!isServerStopped && (
                      <p>
                        Stop the server first to modify gameplay or performance
                        settings.
                      </p>
                    )}
                  </div>

                  <div className="server-v2-settings-card">
                    <div className="server-v2-settings-panel-head">
                      <div className="server-v2-settings-panel-icon">
                        <Upload size={18} />
                      </div>
                      <div>
                        <h3>Server Icon</h3>
                        <p>Upload a new icon for this server</p>
                      </div>
                    </div>
                    <div className="server-v2-icon-upload-row">
                      <div className="server-v2-icon-upload-preview">
                        {settingsIconPreview ? (
                          <img
                            src={settingsIconPreview}
                            alt="Selected server icon"
                          />
                        ) : !settingsIconError ? (
                          <img
                            src={`${api.getServerIconUrl(server.id)}?v=${serverIconVersion}`}
                            alt="Current server icon"
                            onError={() => setSettingsIconError(true)}
                          />
                        ) : (
                          <span>{server.name.charAt(0).toUpperCase()}</span>
                        )}
                      </div>
                      <div className="server-v2-icon-upload-actions">
                        <input
                          ref={settingsIconInputRef}
                          type="file"
                          accept="image/png,image/jpeg"
                          onChange={handleSettingsIconSelected}
                          hidden
                        />
                        <div className="server-v2-settings-actions-inline">
                          <Button
                            type="button"
                            variant="secondary"
                            onClick={() => {
                              if (settingsIconInputRef.current) {
                                settingsIconInputRef.current.value = '';
                                settingsIconInputRef.current.click();
                              }
                            }}
                            disabled={isUploadingSettingsIcon}
                          >
                            Choose Image
                          </Button>
                          <Button
                            type="button"
                            onClick={handleUploadSettingsIcon}
                            disabled={
                              isUploadingSettingsIcon || !selectedSettingsIcon
                            }
                          >
                            {isUploadingSettingsIcon
                              ? 'Uploading...'
                              : 'Upload Icon'}
                          </Button>
                        </div>
                        <p className="server-v2-settings-hint">
                          Recommended size: 64x64. Supported formats: PNG, JPG.
                        </p>
                      </div>
                    </div>
                  </div>

                  <div className="server-v2-settings-card">
                    <div className="server-v2-settings-panel-head">
                      <div className="server-v2-settings-panel-icon">
                        <Download size={18} />
                      </div>
                      <div>
                        <h3>Server Version</h3>
                        <p>Update the Minecraft version for this server</p>
                      </div>
                    </div>
                    <div className="server-v2-settings-form-grid">
                      <label>
                        <span>
                          Select New Version
                          <span title="Only versions available for the current loader are shown.">
                            <CircleHelp size={14} />
                          </span>
                        </span>
                        <select
                          className="form-input"
                          value={selectedVersion}
                          onChange={(e) => setSelectedVersion(e.target.value)}
                          disabled={!canApplySettings || isUpdatingVersion}
                        >
                          {versionOptions.length > 0 ? (
                            versionOptions.map((option) => (
                              <option key={option} value={option}>
                                {option}
                              </option>
                            ))
                          ) : (
                            <option value="">
                              No future versions available
                            </option>
                          )}
                        </select>
                      </label>
                      <div className="server-v2-settings-actions-inline">
                        <Button
                          type="button"
                          onClick={handleVersionUpdate}
                          disabled={
                            !canApplySettings ||
                            isUpdatingVersion ||
                            !selectedVersion ||
                            selectedVersion === server.version
                          }
                        >
                          {isUpdatingVersion ? 'Updating...' : 'Update Version'}
                        </Button>
                      </div>
                      {versionOptions.length === 0 && (
                        <p className="server-v2-settings-hint">
                          Current version ({server.version}) is already the
                          latest available.
                        </p>
                      )}
                    </div>
                  </div>

                  <div className="server-v2-settings-card danger-card">
                    <div className="server-v2-settings-panel-head">
                      <div className="server-v2-settings-panel-icon danger">
                        <Trash2 size={18} />
                      </div>
                      <div>
                        <h3>Delete Server</h3>
                        <p>Permanently delete this server and all its files</p>
                      </div>
                    </div>
                    <p className="server-v2-delete-warning">
                      Warning: this action cannot be undone. All worlds,
                      configurations, and related files will be deleted.
                    </p>
                    <Button
                      type="button"
                      variant="danger"
                      onClick={() => {
                        setDeleteConfirmName('');
                        setIsDeleteModalOpen(true);
                      }}
                      disabled={isDeletingServer}
                    >
                      Delete Server
                    </Button>
                  </div>
                </>
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
          {supportsAddons && (
            <button
              type="button"
              className={activeTab === 'addons' ? 'active' : ''}
              onClick={() => setActiveTab('addons')}
            >
              <Package size={16} />
              {addonsLabel}
            </button>
          )}
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

      <Modal
        isOpen={isPlayerActionsOpen}
        onClose={() => setIsPlayerActionsOpen(false)}
        title="Player Actions"
      >
        <div className="server-v2-player-actions-modal">
          <p>
            Select an action for{' '}
            <strong>{selectedPlayer?.name || 'player'}</strong>.
          </p>

          <div className="server-v2-player-actions-list">
            <button
              type="button"
              disabled={
                isPlayerActionLoading ||
                !selectedPlayer ||
                !selectedPlayer.isOnline
              }
              title={
                selectedPlayer && !selectedPlayer.isOnline
                  ? 'Player is offline, cannot be kicked.'
                  : undefined
              }
              onClick={async () => {
                if (!selectedPlayer) return;
                await runConsoleAction(`kick ${selectedPlayer.name}`);
                setIsPlayerActionsOpen(false);
              }}
            >
              <UserX size={16} />
              Kick Player
            </button>
            <button
              type="button"
              disabled={isPlayerActionLoading || !selectedPlayer}
              onClick={async () => {
                if (!selectedPlayer) return;
                await runConsoleAction(
                  `${selectedPlayer.isOperator ? 'deop' : 'op'} ${selectedPlayer.name}`,
                );
                setIsPlayerActionsOpen(false);
              }}
            >
              <Shield size={16} />
              {selectedPlayer?.isOperator ? 'Remove Operator' : 'Make Operator'}
            </button>
            <button
              type="button"
              className="danger"
              disabled={isPlayerActionLoading || !selectedPlayer}
              onClick={async () => {
                if (!selectedPlayer) return;
                await runConsoleAction(`ban ${selectedPlayer.name}`);
                setIsPlayerActionsOpen(false);
              }}
            >
              <Ban size={16} />
              Ban Player
            </button>
            <button
              type="button"
              className="danger"
              disabled={isPlayerActionLoading || !selectedPlayer}
              onClick={async () => {
                if (!selectedPlayer) return;
                await runConsoleAction(`ban-ip ${selectedPlayer.name}`);
                setIsPlayerActionsOpen(false);
              }}
            >
              <Globe size={16} />
              Ban IP Address
            </button>
            <button
              type="button"
              className="danger-subtle"
              disabled={
                isPlayerActionLoading ||
                !selectedPlayer ||
                !selectedPlayerCanDeleteData
              }
              onClick={handleDeletePlayerData}
              title={
                selectedPlayerCanDeleteData
                  ? undefined
                  : 'Player UUID is required to delete playerdata.'
              }
            >
              <Trash2 size={16} />
              Delete Player Data
            </button>
          </div>
        </div>
      </Modal>

      <Modal
        isOpen={isSettingsModalOpen}
        onClose={() => setIsSettingsModalOpen(false)}
        title={settingsModalTitle}
      >
        <div className="server-v2-delete-modal">
          <p>{settingsModalMessage}</p>
          <div className="modal-actions">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setIsSettingsModalOpen(false)}
            >
              OK
            </Button>
          </div>
        </div>
      </Modal>

      <Modal
        isOpen={isVersionUpdateModalOpen}
        onClose={() => {
          if (!isUpdatingVersion) setIsVersionUpdateModalOpen(false);
        }}
        title={versionUpdateModalTitle}
        hideCloseButton={isUpdatingVersion}
      >
        <div className="server-v2-delete-modal">
          {isUpdatingVersion && renderVersionUpdateProgress()}
          {!isUpdatingVersion &&
            versionUpdateResult &&
            renderVersionUpdateResult()}
          {!isUpdatingVersion && versionUpdateError && (
            <div className="server-v2-version-update-result">
              <div className="server-v2-version-update-hero danger">
                <Ban size={22} />
                <div>
                  <strong>Update failed</strong>
                  <span>{versionUpdateError}</span>
                </div>
              </div>
            </div>
          )}
          {!isUpdatingVersion && (
            <div className="modal-actions">
              <Button
                type="button"
                variant="secondary"
                onClick={() => setIsVersionUpdateModalOpen(false)}
              >
                OK
              </Button>
            </div>
          )}
        </div>
      </Modal>

      <Modal
        isOpen={isIconUploadModalOpen}
        onClose={() => setIsIconUploadModalOpen(false)}
        title={iconUploadModalTitle}
      >
        <div className="server-v2-delete-modal">
          <p>{iconUploadModalMessage}</p>
          <div className="modal-actions">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setIsIconUploadModalOpen(false)}
            >
              OK
            </Button>
          </div>
        </div>
      </Modal>

      <Modal
        isOpen={isDeleteModalOpen}
        onClose={() => setIsDeleteModalOpen(false)}
        title="Delete Server"
      >
        <div className="server-v2-delete-modal">
          <p>
            Do you want to delete server <strong>{server?.name}</strong>?
          </p>
          <p>Type the server name to confirm deletion.</p>
          <input
            className="form-input"
            value={deleteConfirmName}
            onChange={(e) => setDeleteConfirmName(e.target.value)}
            placeholder="Type the server name"
          />
          <div className="modal-actions">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setIsDeleteModalOpen(false)}
              disabled={isDeletingServer}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="danger"
              onClick={handleDeleteServer}
              disabled={!isDeleteNameMatch || isDeletingServer}
            >
              {isDeletingServer ? 'Deleting...' : 'Delete Server'}
            </Button>
          </div>
        </div>
      </Modal>

      <ShareModal
        isOpen={isShareModalOpen}
        onClose={() => setIsShareModalOpen(false)}
        serverId={server.id}
      />
    </div>
  );
};

export default ServerDetail;
