import axios from 'axios';
import {
  ArrowLeft,
  Ban,
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
  Play,
  PowerOff,
  RotateCcw,
  Settings2,
  Share2,
  Shield,
  Skull,
  Square,
  Trash2,
  Upload,
  UserX,
} from 'lucide-react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import type { ChangeEvent, FormEvent, KeyboardEvent } from 'react';
import React, {
  Suspense,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';

import AddonsPanel from '../components/server-detail/AddonsPanel';
import { ConsolePanel } from '../components/server-detail/ConsolePanel';
import {
  type PlayerFilter,
  PlayersPanel,
} from '../components/server-detail/PlayersPanel';
import {
  type DetailTab,
  ServerDetailNav,
} from '../components/server-detail/ServerDetailNav';
import ShareModal from '../components/server-detail/ShareModal';
import { Button } from '../components/ui/Button';
import { CopyButton } from '../components/ui/CopyButton';
import { Modal } from '../components/ui/Modal';
import { useAuth } from '../context/AuthContext';
import { useUploads } from '../context/UploadContext';
import { useConsole } from '../hooks/useConsole';
import { useCopy } from '../hooks/useCopy';
import { useModalDialog } from '../hooks/useModalDialog';
import {
  type BannedIPEntry,
  type BannedListItem,
  type BannedPlayerEntry,
  type OperatorEntry,
  usePlayerLists,
} from '../hooks/usePlayerLists';
import { useServerLifecycle } from '../hooks/useServerLifecycle';
import { useServerStats } from '../hooks/useServerStats';
import { api } from '../services/api';
import type {
  PlayerInfo,
  Server,
  ServerSettings,
  ServerVersionUpdateResult,
} from '../types';
import { getApiErrorMessage } from '../utils/apiError';
import {
  FALLBACK_RAM_MAX_MB,
  MANAGED_JAVA_VERSIONS,
  RAM_MIN_MB,
  clampRamAllocation,
  getPowerControlState,
  isFutureMinecraftVersion,
  normalizeServerSettings,
  readIntPropertyFromContent,
  upsertPropertyLine,
} from '../utils/serverDetail';

const FileExplorer = React.lazy(
  () => import('../components/server-detail/FileExplorer'),
);

type ChartRange = '1m' | '5m' | '30m' | '1h' | '4h';

interface StatSnapshot {
  ts: number;
  cpu: number;
  ramMb: number;
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
const SERVER_POLL_MS = 2000;
const SERVER_POLL_MAX_BACKOFF_MS = 30000;

const formatBytes = (bytes: number) => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return (
    Number.parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  );
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

const ServerDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();
  const { enqueueUpload } = useUploads();
  const { showAlert, showConfirm, modalDialog } = useModalDialog();

  const [server, setServer] = useState<Server | null>(null);
  const [loading, setLoading] = useState(true);
  const [isShareModalOpen, setIsShareModalOpen] = useState(false);
  const [commandInput, setCommandInput] = useState('');
  const [commandHistory, setCommandHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [iconError, setIconError] = useState(false);
  const [serverIconVersion, setServerIconVersion] = useState(() => Date.now());
  const [activeTab, setActiveTab] = useState<DetailTab>('performance');
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
  const [systemMemoryMb, setSystemMemoryMb] = useState<number | null>(null);
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

    void fetchPublicIP();
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

  const {
    powerAction,
    start: handleStart,
    stop: handleStop,
    restart: handleRestart,
    kill: handleKill,
  } = useServerLifecycle({
    server,
    refresh: fetchServer,
    setServer,
    showAlert,
    closeMenu: () => setIsPowerMenuOpen(false),
  });

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
  }, [fetchServer, id]);

  useEffect(() => {
    if (server?.status !== 'STOPPED') return;

    const resetCommandState = window.setTimeout(() => {
      setCommandHistory([]);
      setHistoryIndex(-1);
    }, 0);

    return () => window.clearTimeout(resetCommandState);
  }, [server?.status]);

  useEffect(() => {
    const resetSettingsIconState = window.setTimeout(() => {
      setSelectedSettingsIcon(null);
      setSettingsIconPreview(null);
      setSettingsIconError(false);
    }, 0);

    return () => window.clearTimeout(resetSettingsIconState);
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

    const refreshTimeout = window.setTimeout(() => {
      void refreshPlayerLists();
    }, 0);
    const interval = window.setInterval(() => {
      void refreshPlayerLists();
    }, 8000);

    return () => {
      window.clearTimeout(refreshTimeout);
      window.clearInterval(interval);
    };
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

    const snapshotTimeout = window.setTimeout(() => {
      setStatsHistory((prev) => {
        const cutoff = nextSnapshot.ts - MAX_HISTORY_WINDOW_MS;
        const pruned = prev.filter((item) => item.ts >= cutoff);
        return [...pruned, nextSnapshot];
      });
    }, 0);

    return () => window.clearTimeout(snapshotTimeout);
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

  const fetchSettingsData = useCallback(async () => {
    if (!id || user?.role !== 'admin') return;
    try {
      setIsLoadingSettings(true);
      const [settingsRes, versionsRes, resourcesRes] = await Promise.all([
        api.getServerSettings(id),
        api.getServerVersionOptions(id),
        api.getSystemResources().catch(() => null),
      ]);
      const maxRamMb = resourcesRes?.data.totalMemoryMb ?? settingsRes.data.ram;
      if (resourcesRes) {
        setSystemMemoryMb(resourcesRes.data.totalMemoryMb);
      }
      let normalizedSettings = normalizeServerSettings(
        settingsRes.data,
        maxRamMb,
      );
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
    if (activeTab !== 'settings' || user?.role !== 'admin') return;

    const fetchSettingsTimeout = window.setTimeout(() => {
      void fetchSettingsData();
    }, 0);

    return () => window.clearTimeout(fetchSettingsTimeout);
  }, [activeTab, fetchSettingsData, user?.role]);

  const handleSaveSettings = async () => {
    if (!server || !settingsDraft) return;
    try {
      setIsSavingSettings(true);
      const expectedSpawnProtection = settingsDraft.spawnProtection;
      await api.updateServerSettings(server.id, settingsDraft);

      try {
        const refreshed = await api.getServerSettings(server.id);
        const normalized = normalizeServerSettings(
          refreshed.data,
          systemMemoryMb ?? undefined,
        );
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

  const handleSettingsIconSelected = (e: ChangeEvent<HTMLInputElement>) => {
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
    const file = selectedSettingsIcon;
    try {
      setIsUploadingSettingsIcon(true);
      await enqueueUpload(
        {
          file,
          name: file.name,
          target: { kind: 'server-icon', serverId: server.id },
          contentType: file.type,
        },
        {
          onComplete: () => {
            setIsUploadingSettingsIcon(false);
            setServerIconVersion(Date.now());
            setIconError(false);
            setSettingsIconError(false);
            setIconUploadModalTitle('Icon Updated');
            setIconUploadModalMessage('Server icon uploaded successfully.');
            setIsIconUploadModalOpen(true);
          },
          onError: (message) => {
            setIsUploadingSettingsIcon(false);
            setIconUploadModalTitle('Icon Upload Failed');
            setIconUploadModalMessage(message);
            setIsIconUploadModalOpen(true);
          },
        },
      );
      setSelectedSettingsIcon(null);
      setSettingsIconPreview(null);
      if (settingsIconInputRef.current) {
        settingsIconInputRef.current.value = '';
      }
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
      setIsDeleteModalOpen(false);
      setIsDeletingServer(false);
      await showAlert({
        title: 'Delete Failed',
        message: getApiErrorMessage(err, 'Failed to delete server.'),
        variant: 'danger',
      });
    } finally {
      setIsDeletingServer(false);
    }
  };

  const handleCommandSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!commandInput.trim()) return;

    setCommandHistory((prev) => [commandInput, ...prev]);
    setHistoryIndex(-1);
    sendCommand(commandInput);
    setCommandInput('');
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
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

  const selectedRangeMs = RANGE_TO_MS[chartRange];
  const rangeEnd = statsHistory[statsHistory.length - 1]?.ts ?? 0;
  const rangeStart = rangeEnd - selectedRangeMs;

  const visibleHistory = useMemo(() => {
    const inRange = statsHistory.filter(
      (point) => point.ts >= rangeStart && point.ts <= rangeEnd,
    );

    if (inRange.length > 0) {
      return inRange;
    }

    if (statsHistory.length > 0) {
      return [statsHistory[statsHistory.length - 1]];
    }

    return [];
  }, [rangeEnd, rangeStart, statsHistory]);

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
    <div className="tw:flex tw:flex-col tw:items-center tw:gap-4 tw:text-center">
      <div className="tw:grid tw:h-14 tw:w-14 tw:place-items-center tw:rounded-[18px] tw:border tw:border-white/12 tw:bg-blue-600/18 tw:text-blue-300">
        <LoaderCircle size={28} className="tw:animate-spin" />
      </div>
      <div>
        <strong className="tw:block tw:text-base tw:text-slate-50">
          Updating to {selectedVersion}
        </strong>
        <p className="tw:mb-0 tw:mt-1.5 tw:text-slate-300">
          Creating a backup, updating the server, and checking addons.
        </p>
      </div>
      <div className="tw:grid tw:w-full tw:gap-2">
        <span className="tw:relative tw:rounded-xl tw:border tw:border-slate-400/18 tw:bg-slate-900/42 tw:px-3 tw:py-2.5 tw:pl-[34px] tw:text-left tw:text-blue-100 tw:before:absolute tw:before:left-3 tw:before:top-1/2 tw:before:h-2 tw:before:w-2 tw:before:-translate-y-1/2 tw:before:rounded-full tw:before:bg-blue-400 tw:before:shadow-[0_0_0_4px_rgba(96,165,250,0.16)]">
          Backup server files
        </span>
        <span className="tw:relative tw:rounded-xl tw:border tw:border-slate-400/18 tw:bg-slate-900/42 tw:px-3 tw:py-2.5 tw:pl-[34px] tw:text-left tw:text-blue-100 tw:before:absolute tw:before:left-3 tw:before:top-1/2 tw:before:h-2 tw:before:w-2 tw:before:-translate-y-1/2 tw:before:rounded-full tw:before:bg-blue-400 tw:before:shadow-[0_0_0_4px_rgba(96,165,250,0.16)]">
          Install new server version
        </span>
        <span className="tw:relative tw:rounded-xl tw:border tw:border-slate-400/18 tw:bg-slate-900/42 tw:px-3 tw:py-2.5 tw:pl-[34px] tw:text-left tw:text-blue-100 tw:before:absolute tw:before:left-3 tw:before:top-1/2 tw:before:h-2 tw:before:w-2 tw:before:-translate-y-1/2 tw:before:rounded-full tw:before:bg-blue-400 tw:before:shadow-[0_0_0_4px_rgba(96,165,250,0.16)]">
          Update or disable incompatible addons
        </span>
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
      <div className="tw:flex tw:flex-col tw:gap-4">
        <div className="tw:flex tw:items-start tw:gap-3 tw:rounded-2xl tw:border tw:border-green-500/25 tw:bg-green-500/12 tw:p-3.5 tw:text-green-300">
          <Shield size={22} />
          <div>
            <strong className="tw:block tw:text-slate-50">
              Server updated to {versionUpdateResult.version}
            </strong>
            <span className="tw:mt-[3px] tw:block tw:text-slate-300">
              Backup created before applying changes.
            </span>
          </div>
        </div>
        <div className="tw:grid tw:gap-1.5 tw:rounded-[14px] tw:border tw:border-slate-400/16 tw:bg-slate-900/38 tw:p-3">
          <span className="tw:text-[0.76rem] tw:font-bold tw:tracking-[0.08em] tw:text-slate-400 tw:uppercase">
            Backup
          </span>
          <code className="tw:break-words tw:text-sky-100">
            {versionUpdateResult.backupName}
          </code>
        </div>
        <div className="tw:grid tw:grid-cols-3 tw:gap-2.5 tw:max-[480px]:grid-cols-1">
          <div className="tw:rounded-[14px] tw:border tw:border-slate-400/16 tw:bg-slate-900/38 tw:p-3">
            <strong className="tw:block tw:text-xl tw:text-slate-50">
              {updatedCount}
            </strong>
            <span className="tw:mt-[3px] tw:block tw:text-[0.78rem] tw:text-slate-400">
              Addons updated
            </span>
          </div>
          <div className="tw:rounded-[14px] tw:border tw:border-slate-400/16 tw:bg-slate-900/38 tw:p-3">
            <strong className="tw:block tw:text-xl tw:text-slate-50">
              {disabledCount}
            </strong>
            <span className="tw:mt-[3px] tw:block tw:text-[0.78rem] tw:text-slate-400">
              Addons disabled
            </span>
          </div>
          <div className="tw:rounded-[14px] tw:border tw:border-slate-400/16 tw:bg-slate-900/38 tw:p-3">
            <strong className="tw:block tw:text-xl tw:text-slate-50">
              {failedCount}
            </strong>
            <span className="tw:mt-[3px] tw:block tw:text-[0.78rem] tw:text-slate-400">
              Addon failures
            </span>
          </div>
        </div>
        {failedCount > 0 && addons && (
          <div className="tw:grid tw:gap-2">
            {addons.failed.slice(0, 3).map((failure) => (
              <p
                key={failure.id}
                className="tw:m-0 tw:rounded-xl tw:bg-red-400/12 tw:p-2.5 tw:text-red-200"
              >
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
  const performanceEmptyState = (() => {
    switch (server?.status) {
      case 'STOPPED':
        return {
          title: 'Server is offline',
          description: 'Start the server to view CPU and RAM performance.',
        };
      case 'STARTING':
        return {
          title: 'Server is starting',
          description: 'Performance data will appear when startup finishes.',
        };
      case 'STOPPING':
        return {
          title: 'Server is stopping',
          description: 'Performance data is unavailable while it shuts down.',
        };
      case 'CREATING':
        return {
          title: 'Server is being created',
          description: 'Performance data will appear when setup finishes.',
        };
      case 'RUNNING':
        return isStatsOffline
          ? {
              title: 'Performance data unavailable',
              description: 'Waiting for the statistics connection to recover.',
            }
          : null;
      default:
        return null;
    }
  })();
  let chartOverlayMessage: string | null = null;
  if (isServerOnlineForChart) {
    if (chartSize.width <= 0 || chartSize.height <= 0) {
      chartOverlayMessage = 'Preparing chart layout...';
    } else if (visibleHistory.length === 0) {
      chartOverlayMessage = 'Waiting for performance data...';
    }
  }

  const ramAllocationMaxMb = Math.max(
    RAM_MIN_MB,
    systemMemoryMb ?? server?.ram ?? settingsDraft?.ram ?? FALLBACK_RAM_MAX_MB,
  );

  const updateSettingsField = <K extends keyof ServerSettings>(
    field: K,
    value: ServerSettings[K],
  ) => {
    setSettingsDraft((prev) => {
      if (!prev) return prev;

      return {
        ...prev,
        [field]:
          field === 'ram'
            ? clampRamAllocation(Number(value), ramAllocationMaxMb)
            : value,
      };
    });
  };

  const players = useMemo(() => stats.players ?? [], [stats.players]);
  const canModeratePlayers = Boolean(server?.permissions?.canViewConsole);
  const {
    onlineItems,
    operatorItems,
    bannedItems,
    operatorNameSet,
    operatorUuidSet,
    filteredOnlineItems,
    filteredOperatorItems,
    filteredBannedItems,
  } = usePlayerLists(
    players,
    operators,
    bannedPlayers,
    bannedIps,
    playersSearch,
  );

  const selectedPlayerCanDeleteData = Boolean(selectedPlayer?.id?.trim());

  const queuePlayerDataRefresh = () => {
    setTimeout(() => {
      void refreshPlayerLists();
    }, 1200);
  };

  const runConsoleAction = async (command: string) => {
    if (!canModeratePlayers) {
      void showAlert({
        title: 'Permission Required',
        message: 'You do not have permission to perform this action.',
        variant: 'danger',
      });
      return;
    }
    if (!isConnected) {
      void showAlert({
        title: 'Console Disconnected',
        message: 'Console is disconnected. Please wait and try again.',
        variant: 'danger',
      });
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
    if (!id || !selectedPlayer?.id || !canModeratePlayers) {
      return;
    }

    const confirmed = await showConfirm({
      title: 'Delete Player Data',
      message: `Delete ${selectedPlayer.name} playerdata file from this server?`,
      confirmText: 'Delete',
      variant: 'danger',
    });
    if (!confirmed) return;

    setIsPlayerActionLoading(true);
    try {
      await api.deleteFile(id, `/world/playerdata/${selectedPlayer.id}.dat`);
      await showAlert({
        title: 'Player Data Deleted',
        message: 'Player data deleted successfully.',
        variant: 'success',
      });
      setIsPlayerActionsOpen(false);
    } catch (err) {
      console.error('Failed to delete player data:', err);
      await showAlert({
        title: 'Delete Failed',
        message: 'Failed to delete player data.',
        variant: 'danger',
      });
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
  const powerControlState = getPowerControlState(server.status, powerAction);
  const isStoppedLike = server.status === 'STOPPED';

  return (
    <div className="tw:flex tw:h-full tw:min-w-0 tw:flex-col tw:gap-4">
      {modalDialog}
      <header className="tw:flex tw:min-w-0 tw:items-center tw:gap-3.5 tw:overflow-hidden tw:rounded-2xl tw:border tw:border-border tw:bg-bg-card tw:p-3.5 tw:max-[1024px]:!flex-nowrap tw:max-[1024px]:!gap-2.5 tw:max-[1024px]:!p-3 tw:max-[640px]:!grid tw:max-[640px]:!grid-cols-[40px_minmax(0,1fr)_auto] tw:max-[640px]:!items-center tw:max-[640px]:!gap-2 tw:max-[480px]:!grid-cols-[40px_minmax(0,1fr)] tw:max-[480px]:!gap-2.5">
        <button
          type="button"
          className="tw:flex tw:h-[42px] tw:w-[42px] tw:shrink-0 tw:cursor-pointer tw:items-center tw:justify-center tw:rounded-xl tw:border tw:border-border tw:bg-white/[0.02] tw:p-0 tw:text-text-muted tw:hover:bg-white/[0.06] tw:hover:text-text-main tw:max-[640px]:h-10 tw:max-[640px]:w-10"
          onClick={() => navigate('/')}
          title="Back to dashboard"
        >
          <ArrowLeft size={18} />
        </button>

        <div className="tw:flex tw:min-w-0 tw:flex-1 tw:items-center tw:gap-3 tw:max-[1024px]:!grid tw:max-[1024px]:!grid-cols-[auto_minmax(0,1fr)] tw:max-[1024px]:!items-start tw:max-[1024px]:!gap-x-3 tw:max-[1024px]:!gap-y-2">
          <div className="tw:h-9 tw:w-9 tw:shrink-0 tw:overflow-hidden tw:rounded">
            {!iconError ? (
              <img
                src={`${api.getServerIconUrl(server.id)}?v=${serverIconVersion}`}
                alt="Server Icon"
                onError={() => setIconError(true)}
                className="tw:h-full tw:w-full tw:rounded tw:bg-black/20 tw:object-contain [image-rendering:pixelated]"
              />
            ) : (
              <div className="tw:flex tw:h-full tw:w-full tw:items-center tw:justify-center tw:rounded tw:bg-white/10 tw:text-base tw:font-semibold tw:text-text-muted">
                {server.name.charAt(0).toUpperCase()}
              </div>
            )}
          </div>

          <div className="tw:flex tw:min-w-0 tw:flex-col tw:gap-1.5">
            <div className="tw:flex tw:min-w-0 tw:flex-wrap tw:items-center tw:gap-2.5">
              <h1 className="tw:m-0 tw:min-w-0 tw:overflow-hidden tw:text-ellipsis tw:whitespace-nowrap tw:text-[1.8rem] tw:leading-[1.1] tw:max-[1024px]:text-2xl tw:max-[640px]:text-[1.3rem]">
                {server.name}
              </h1>
              <span
                className={`tw:rounded-full tw:px-3 tw:py-1 tw:text-[0.75rem] tw:font-bold tw:tracking-[0.04em] tw:text-white ${server.status === 'RUNNING' ? 'tw:bg-emerald-500' : server.status === 'STOPPED' ? 'tw:bg-red-500' : server.status === 'CREATING' ? 'tw:bg-blue-500' : 'tw:bg-orange-500'}`}
              >
                {server.status}
              </span>
            </div>
            <div className="tw:flex tw:min-w-0 tw:flex-wrap tw:items-center tw:gap-2 tw:text-[0.9rem] tw:text-text-muted tw:max-[640px]:text-[0.82rem]">
              <span className="tw:font-semibold tw:text-text-main">
                {server.loader}
              </span>
              <span>•</span>
              <span>{server.version}</span>
              <span className="tw:max-[1024px]:!hidden">•</span>
              <button
                type="button"
                className="tw:inline-flex tw:h-6 tw:max-w-40 tw:min-w-0 tw:cursor-pointer tw:items-center tw:overflow-hidden tw:rounded-[7px] tw:border tw:border-white/10 tw:bg-white/4 tw:px-2 tw:font-mono tw:text-[0.78rem] tw:leading-none tw:text-ellipsis tw:whitespace-nowrap tw:text-text-muted tw:transition-all tw:duration-150 tw:hover:border-indigo-500/55 tw:hover:bg-indigo-500/12 tw:hover:text-text-main tw:focus-visible:outline-2 tw:focus-visible:outline-indigo-500/70 tw:focus-visible:outline-offset-2 tw:max-[1024px]:!hidden"
                onClick={() => copy(address)}
                title="Click to copy"
              >
                {address}
              </button>
              <CopyButton
                text={address}
                variant="secondary"
                title="Copy address"
                className="tw:h-6 tw:w-6 tw:min-h-6 tw:min-w-6 tw:shrink-0 tw:p-0 tw:max-[1024px]:!hidden"
              />
            </div>
          </div>

          <div className="tw:col-span-full tw:hidden tw:w-full tw:min-w-0 tw:items-center tw:gap-1.5 tw:max-[1024px]:!flex">
            <button
              type="button"
              className="tw:inline-flex tw:h-6 tw:w-full tw:min-w-0 tw:!max-w-[390px] tw:flex-1 tw:cursor-pointer tw:items-center tw:overflow-hidden tw:rounded-[7px] tw:border tw:border-white/10 tw:bg-white/4 tw:px-2 tw:font-mono tw:text-[0.78rem] tw:leading-none tw:text-ellipsis tw:whitespace-nowrap tw:text-text-muted tw:transition-all tw:duration-150 tw:hover:border-indigo-500/55 tw:hover:bg-indigo-500/12 tw:hover:text-text-main tw:focus-visible:outline-2 tw:focus-visible:outline-indigo-500/70 tw:focus-visible:outline-offset-2"
              onClick={() => copy(address)}
              title="Click to copy"
            >
              {address}
            </button>
            <CopyButton
              text={address}
              variant="secondary"
              title="Copy address"
              iconSize={16}
              className="tw:h-6 tw:w-6 tw:min-h-6 tw:min-w-6 tw:shrink-0 tw:p-0 tw:text-text-main"
            />
          </div>
        </div>

        <div className="tw:flex tw:items-center tw:gap-2 tw:max-[640px]:!col-auto tw:max-[640px]:!w-auto tw:max-[640px]:!justify-end tw:max-[480px]:!col-span-full tw:max-[480px]:!w-full tw:max-[480px]:!justify-between">
          {(server.permissions?.canControlPower ||
            server.permissions?.canViewConsole) &&
            (isStoppedLike ? (
              <Button
                onClick={handleStart}
                disabled={powerAction === 'start'}
                className="tw:max-[640px]:!min-w-24 tw:max-[640px]:!justify-center tw:max-[480px]:!flex-1"
              >
                {powerAction === 'start' ? (
                  <LoaderCircle size={16} className="tw:animate-spin" />
                ) : (
                  <Play size={16} />
                )}
                Start
              </Button>
            ) : (
              <div
                className="tw:relative tw:flex tw:items-center tw:gap-2"
                ref={powerMenuRef}
              >
                <Button
                  variant="danger"
                  onClick={handleStop}
                  disabled={powerControlState.stopDisabled}
                >
                  {powerAction === 'stop' ? (
                    <LoaderCircle size={16} className="tw:animate-spin" />
                  ) : (
                    <Square size={16} />
                  )}
                  Stop
                </Button>
                <button
                  type="button"
                  className="tw:flex tw:h-[38px] tw:w-[38px] tw:cursor-pointer tw:items-center tw:justify-center tw:rounded-lg tw:border tw:border-border tw:bg-white/[0.03] tw:p-0 tw:text-text-muted tw:hover:bg-white/[0.06] tw:hover:text-text-main tw:disabled:cursor-not-allowed tw:disabled:opacity-55"
                  onClick={() => setIsPowerMenuOpen((prev) => !prev)}
                  disabled={powerControlState.moreDisabled}
                >
                  <MoreVertical size={16} />
                </button>
                {isPowerMenuOpen && (
                  <div className="tw:absolute tw:right-0 tw:top-11 tw:z-10 tw:min-w-[140px] tw:overflow-hidden tw:rounded-[10px] tw:border tw:border-border tw:bg-bg-sidebar tw:shadow-[0_10px_30px_rgba(0,0,0,0.4)]">
                    <button
                      type="button"
                      className="tw:flex tw:w-full tw:cursor-pointer tw:items-center tw:gap-2 tw:border-0 tw:bg-transparent tw:px-3 tw:py-2.5 tw:text-left tw:text-text-main tw:hover:bg-white/[0.06] tw:disabled:cursor-not-allowed tw:disabled:text-text-muted tw:disabled:opacity-55"
                      onClick={handleRestart}
                      disabled={powerControlState.restartDisabled}
                    >
                      <RotateCcw size={15} /> Restart
                    </button>
                    <button
                      type="button"
                      className="tw:flex tw:w-full tw:cursor-pointer tw:items-center tw:gap-2 tw:border-0 tw:bg-transparent tw:px-3 tw:py-2.5 tw:text-left tw:text-text-main tw:hover:bg-white/[0.06] tw:disabled:cursor-not-allowed tw:disabled:text-text-muted tw:disabled:opacity-55"
                      onClick={handleKill}
                      disabled={powerControlState.killDisabled}
                    >
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
              className="tw:max-[640px]:!h-10 tw:max-[640px]:!w-10 tw:max-[640px]:!min-w-10 tw:max-[640px]:!shrink-0 tw:max-[640px]:!p-0"
            >
              <Share2 size={16} />
            </Button>
          )}
        </div>
      </header>

      <div className="tw:grid tw:min-h-0 tw:flex-1 tw:grid-cols-[minmax(0,1fr)_240px] tw:gap-4 tw:max-[1024px]:flex tw:max-[1024px]:flex-col tw:max-[1024px]:gap-2.5">
        <section className="server-v2-content tw:min-h-0 tw:overflow-x-hidden tw:overflow-y-auto tw:pr-3 tw:max-[1024px]:flex tw:max-[1024px]:min-h-0 tw:max-[1024px]:flex-1 tw:max-[1024px]:flex-col tw:max-[1024px]:overflow-visible tw:max-[1024px]:pr-0">
          {activeTab === 'performance' && (
            <div className="tw:grid tw:grid-cols-4 tw:content-start tw:gap-3 tw:max-[1024px]:grid-cols-2 tw:max-[640px]:grid-cols-1">
              <div className="tw:box-border tw:flex tw:min-h-0 tw:min-w-0 tw:flex-col tw:gap-1 tw:rounded-[14px] tw:border tw:border-border tw:bg-bg-card tw:p-3">
                <div className="tw:inline-flex tw:items-center tw:gap-1.5 tw:text-[0.95rem] tw:font-semibold tw:leading-[1.2] tw:text-text-muted">
                  <Cpu size={16} /> CPU Usage
                </div>
                <div className="tw:text-[1.05rem] tw:font-bold tw:leading-[1.15] tw:tracking-[-0.01em]">
                  {server.status === 'RUNNING'
                    ? `${stats.cpu.toFixed(1)}%`
                    : 'Offline'}
                </div>
              </div>
              <div className="tw:box-border tw:flex tw:min-h-0 tw:min-w-0 tw:flex-col tw:gap-1 tw:rounded-[14px] tw:border tw:border-border tw:bg-bg-card tw:p-3">
                <div className="tw:inline-flex tw:items-center tw:gap-1.5 tw:text-[0.95rem] tw:font-semibold tw:leading-[1.2] tw:text-text-muted">
                  <MemoryStick size={16} /> RAM Usage
                </div>
                <div className="tw:text-[1.05rem] tw:font-bold tw:leading-[1.15] tw:tracking-[-0.01em]">
                  {server.status === 'RUNNING'
                    ? `${(stats.ram / 1024 / 1024).toFixed(0)} MB / ${server.ram} MB`
                    : 'Offline'}
                </div>
              </div>
              <div className="tw:box-border tw:flex tw:min-h-0 tw:min-w-0 tw:flex-col tw:gap-1 tw:rounded-[14px] tw:border tw:border-border tw:bg-bg-card tw:p-3">
                <div className="tw:inline-flex tw:items-center tw:gap-1.5 tw:text-[0.95rem] tw:font-semibold tw:leading-[1.2] tw:text-text-muted">
                  <Clock3 size={16} /> Uptime
                </div>
                <div className="tw:text-[1.05rem] tw:font-bold tw:leading-[1.15] tw:tracking-[-0.01em]">
                  {server.status === 'RUNNING'
                    ? formatDuration(stats.uptimeSeconds)
                    : 'Offline'}
                </div>
              </div>
              <div className="tw:box-border tw:flex tw:min-h-0 tw:min-w-0 tw:flex-col tw:gap-1 tw:rounded-[14px] tw:border tw:border-border tw:bg-bg-card tw:p-3">
                <div className="tw:inline-flex tw:items-center tw:gap-1.5 tw:text-[0.95rem] tw:font-semibold tw:leading-[1.2] tw:text-text-muted">
                  <HardDrive size={16} /> Disk
                </div>
                <div className="tw:text-[1.05rem] tw:font-bold tw:leading-[1.15] tw:tracking-[-0.01em]">
                  {formatBytes(stats.disk)}
                </div>
              </div>

              <div className="tw:col-span-full tw:flex tw:min-w-0 tw:flex-col tw:gap-3 tw:rounded-[14px] tw:border tw:border-border tw:bg-bg-card tw:p-3.5">
                <div className="tw:flex tw:items-center tw:justify-between tw:gap-2.5">
                  <h2 className="tw:m-0 tw:text-[1.2rem] tw:leading-[1.2]">
                    Performance Graph
                  </h2>
                  {!performanceEmptyState && (
                    <div className="tw:inline-flex tw:gap-1 tw:rounded-full tw:border tw:border-border tw:bg-white/3 tw:p-1">
                      {(Object.keys(RANGE_TO_MS) as ChartRange[]).map(
                        (range) => (
                          <button
                            key={range}
                            type="button"
                            className={`tw:cursor-pointer tw:rounded-full tw:border-0 tw:bg-transparent tw:px-2.5 tw:py-1 tw:text-text-muted ${chartRange === range ? 'tw:bg-primary/20 tw:text-white' : ''}`}
                            onClick={() => setChartRange(range)}
                          >
                            {range}
                          </button>
                        ),
                      )}
                    </div>
                  )}
                </div>

                <div
                  className="tw:grid tw:min-w-0 tw:grid-cols-1 tw:gap-3"
                  ref={chartShellRef}
                >
                  {performanceEmptyState ? (
                    <div
                      className="tw:col-span-full tw:flex tw:min-h-[280px] tw:flex-col tw:items-center tw:justify-center tw:gap-2 tw:rounded-xl tw:border tw:border-border tw:bg-bg-dark tw:p-6 tw:text-center tw:text-text-muted"
                      role="status"
                    >
                      <PowerOff size={32} aria-hidden="true" />
                      <strong className="tw:text-[1.05rem] tw:text-text-main">
                        {performanceEmptyState.title}
                      </strong>
                      <span className="tw:max-w-[360px]">
                        {performanceEmptyState.description}
                      </span>
                    </div>
                  ) : (
                    <>
                      <section className="tw:flex tw:min-w-0 tw:flex-col tw:gap-2">
                        <h3 className="tw:m-0 tw:text-[0.95rem] tw:text-text-muted">
                          CPU Usage
                        </h3>
                        <div className="server-v2-chart-shell tw:relative tw:h-[280px] tw:min-h-[250px] tw:min-w-0 tw:overflow-hidden tw:rounded-xl tw:border tw:border-border tw:bg-bg-dark">
                          {chartSize.width > 0 && chartSize.height > 0 ? (
                            <ResponsiveContainer
                              width="100%"
                              height="100%"
                              minWidth={0}
                              minHeight={0}
                              initialDimension={{ width: 1, height: 1 }}
                            >
                              <LineChart
                                data={visibleHistory}
                                margin={{
                                  top: 12,
                                  right: 20,
                                  left: 12,
                                  bottom: 12,
                                }}
                              >
                                <CartesianGrid
                                  strokeDasharray="3 3"
                                  stroke="rgba(255,255,255,0.08)"
                                />
                                <XAxis
                                  dataKey="ts"
                                  type="number"
                                  scale="time"
                                  domain={[rangeStart, rangeEnd]}
                                  stroke="var(--text-muted)"
                                  tickFormatter={formatTimeTick}
                                  minTickGap={24}
                                  padding={{ left: 8, right: 8 }}
                                />
                                <YAxis
                                  domain={[0, 100]}
                                  tickFormatter={(value) => `${value}%`}
                                  stroke="var(--text-muted)"
                                  width={60}
                                />
                                <Tooltip
                                  labelFormatter={(value) =>
                                    formatTimeTick(Number(value))
                                  }
                                  formatter={(value) => [
                                    `${Number(value ?? 0).toFixed(1)}%`,
                                    'CPU',
                                  ]}
                                  contentStyle={{
                                    background: '#1e1e1e',
                                    border: '1px solid var(--border-color)',
                                    borderRadius: '8px',
                                  }}
                                  labelStyle={{ color: 'var(--text-main)' }}
                                />
                                <Line
                                  type="monotone"
                                  dataKey="cpu"
                                  name="CPU"
                                  stroke="#60a5fa"
                                  strokeWidth={2}
                                  dot={false}
                                  isAnimationActive={false}
                                />
                              </LineChart>
                            </ResponsiveContainer>
                          ) : null}

                          {chartOverlayMessage && (
                            <div className="tw:absolute tw:inset-0 tw:flex tw:h-full tw:items-center tw:justify-center tw:bg-[rgba(26,26,26,0.4)] tw:text-text-muted tw:pointer-events-none">
                              {chartOverlayMessage}
                            </div>
                          )}
                        </div>
                      </section>

                      <section className="tw:flex tw:min-w-0 tw:flex-col tw:gap-2">
                        <h3 className="tw:m-0 tw:text-[0.95rem] tw:text-text-muted">
                          RAM Usage
                        </h3>
                        <div className="server-v2-chart-shell tw:relative tw:h-[280px] tw:min-h-[250px] tw:min-w-0 tw:overflow-hidden tw:rounded-xl tw:border tw:border-border tw:bg-bg-dark">
                          {chartSize.width > 0 && chartSize.height > 0 ? (
                            <ResponsiveContainer
                              width="100%"
                              height="100%"
                              minWidth={0}
                              minHeight={0}
                              initialDimension={{ width: 1, height: 1 }}
                            >
                              <LineChart
                                data={visibleHistory}
                                margin={{
                                  top: 12,
                                  right: 20,
                                  left: 12,
                                  bottom: 12,
                                }}
                              >
                                <CartesianGrid
                                  strokeDasharray="3 3"
                                  stroke="rgba(255,255,255,0.08)"
                                />
                                <XAxis
                                  dataKey="ts"
                                  type="number"
                                  scale="time"
                                  domain={[rangeStart, rangeEnd]}
                                  stroke="var(--text-muted)"
                                  tickFormatter={formatTimeTick}
                                  minTickGap={24}
                                  padding={{ left: 8, right: 8 }}
                                />
                                <YAxis
                                  domain={[0, ramDomainMax]}
                                  tickFormatter={(value) =>
                                    `${Math.round(Number(value))}MB`
                                  }
                                  stroke="var(--text-muted)"
                                  width={68}
                                />
                                <Tooltip
                                  labelFormatter={(value) =>
                                    formatTimeTick(Number(value))
                                  }
                                  formatter={(value) => [
                                    `${Math.round(Number(value ?? 0))} MB`,
                                    'RAM',
                                  ]}
                                  contentStyle={{
                                    background: '#1e1e1e',
                                    border: '1px solid var(--border-color)',
                                    borderRadius: '8px',
                                  }}
                                  labelStyle={{ color: 'var(--text-main)' }}
                                />
                                <Line
                                  type="monotone"
                                  dataKey="ramMb"
                                  name="RAM"
                                  stroke="#a855f7"
                                  strokeWidth={2}
                                  dot={false}
                                  isAnimationActive={false}
                                />
                              </LineChart>
                            </ResponsiveContainer>
                          ) : null}

                          {chartOverlayMessage && (
                            <div className="tw:absolute tw:inset-0 tw:flex tw:h-full tw:items-center tw:justify-center tw:bg-[rgba(26,26,26,0.4)] tw:text-text-muted tw:pointer-events-none">
                              {chartOverlayMessage}
                            </div>
                          )}
                        </div>
                      </section>
                    </>
                  )}
                </div>
              </div>
            </div>
          )}

          {activeTab === 'console' && (
            <ConsolePanel
              command={commandInput}
              connected={isConnected}
              logs={logs}
              onCommandChange={setCommandInput}
              onKeyDown={handleKeyDown}
              onSubmit={handleCommandSubmit}
            />
          )}

          {activeTab === 'players' && (
            <PlayersPanel
              banned={filteredBannedItems}
              bannedCount={bannedItems.length}
              canModerate={canModeratePlayers}
              filter={playerFilter}
              isActionLoading={isPlayerActionLoading}
              maxPlayers={stats.maxPlayers}
              online={filteredOnlineItems}
              onlineCount={onlineItems.length}
              onlinePlayers={stats.onlinePlayers}
              operators={filteredOperatorItems}
              operatorCount={operatorItems.length}
              search={playersSearch}
              onFilterChange={setPlayerFilter}
              onPardon={handlePardon}
              onSearchChange={setPlayersSearch}
              onSelectPlayer={(player, isOperator) => {
                if (!canModeratePlayers) return;
                const normalizedName = player.name.toLowerCase();
                const normalizedUuid = player.uuid?.toLowerCase();
                setSelectedPlayer({
                  name: player.name,
                  id: player.uuid || '',
                  isOnline: player.isOnline,
                  isOperator:
                    isOperator ||
                    operatorNameSet.has(normalizedName) ||
                    Boolean(
                      normalizedUuid && operatorUuidSet.has(normalizedUuid),
                    ),
                });
                setIsPlayerActionsOpen(true);
              }}
            />
          )}

          {activeTab === 'files' && (
            <Suspense
              fallback={
                <div className="tw:box-border tw:min-w-0 tw:rounded-[14px] tw:border tw:border-border tw:bg-bg-card tw:p-3.5 tw:[&_h2]:mt-0 tw:[&_h2]:mb-2">
                  <p>Loading files...</p>
                </div>
              }
            >
              <FileExplorer serverId={server.id} />
            </Suspense>
          )}

          {activeTab === 'addons' && supportsAddons && (
            <AddonsPanel
              server={server}
              canManage={canModeratePlayers}
              confirmAction={(title, message, confirmText) =>
                showConfirm({
                  title,
                  message,
                  confirmText,
                  variant: 'danger',
                })
              }
            />
          )}

          {activeTab === 'settings' && (
            <div className="tw:flex tw:flex-col tw:gap-4">
              {!canEditSettings && (
                <div className="tw:box-border tw:min-w-0 tw:rounded-[14px] tw:border tw:border-border tw:bg-bg-card tw:p-3.5 tw:[&_h2]:mt-0 tw:[&_h2]:mb-2">
                  <h2>Server Settings</h2>
                  <p>Only admins can edit server settings.</p>
                </div>
              )}
              {canEditSettings && (isLoadingSettings || !settingsDraft) && (
                <div className="tw:box-border tw:min-w-0 tw:rounded-[14px] tw:border tw:border-border tw:bg-bg-card tw:p-3.5 tw:[&_h2]:mt-0 tw:[&_h2]:mb-2">
                  <p>Loading settings...</p>
                </div>
              )}
              {canEditSettings && !isLoadingSettings && settingsDraft && (
                <>
                  <div className="tw:grid tw:grid-cols-2 tw:gap-4 tw:max-[1024px]:grid-cols-1">
                    <div className="tw:rounded-[14px] tw:border tw:border-border tw:bg-white/[0.02] tw:p-4 tw:max-[640px]:p-3">
                      <div className="tw:mb-4 tw:flex tw:items-center tw:gap-3 tw:max-[640px]:items-center">
                        <div className="tw:flex tw:h-10 tw:w-10 tw:shrink-0 tw:items-center tw:justify-center tw:rounded-xl tw:border tw:border-border tw:bg-white/4">
                          <Gamepad2 size={18} />
                        </div>
                        <div>
                          <h3>Gameplay</h3>
                          <p>General gameplay settings</p>
                        </div>
                      </div>

                      <div className="tw:grid tw:grid-cols-2 tw:gap-3 tw:max-[1024px]:grid-cols-1 tw:[&_label]:flex tw:[&_label]:flex-col tw:[&_label]:gap-1.5 tw:[&_label>span]:inline-flex tw:[&_label>span]:items-center tw:[&_label>span]:gap-1.5 tw:[&_label>span]:text-[0.9rem] tw:[&_label>span]:text-text-muted tw:[&_input[type=range]]:accent-purple-500">
                        <label className="tw:col-span-full">
                          <span>
                            Server Name{' '}
                            <span title="Display name used across NaviServer.">
                              <CircleHelp size={14} />
                            </span>
                          </span>
                          <input
                            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:disabled:cursor-not-allowed tw:disabled:opacity-60 tw:max-[1024px]:text-base"
                            value={settingsDraft.name}
                            onChange={(e) =>
                              updateSettingsField('name', e.target.value)
                            }
                            disabled={!canApplySettings}
                            maxLength={64}
                          />
                        </label>

                        <label>
                          <span>
                            Game Mode{' '}
                            <span title="Default game mode for new players.">
                              <CircleHelp size={14} />
                            </span>
                          </span>
                          <select
                            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:disabled:cursor-not-allowed tw:disabled:opacity-60 tw:max-[1024px]:text-base"
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
                            Difficulty{' '}
                            <span title="World difficulty level.">
                              <CircleHelp size={14} />
                            </span>
                          </span>
                          <select
                            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:disabled:cursor-not-allowed tw:disabled:opacity-60 tw:max-[1024px]:text-base"
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

                        <label className="tw:col-span-full">
                          <span>
                            Server Message (MOTD){' '}
                            <span title="Server list message players see.">
                              <CircleHelp size={14} />
                            </span>
                          </span>
                          <input
                            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:disabled:cursor-not-allowed tw:disabled:opacity-60 tw:max-[1024px]:text-base"
                            value={settingsDraft.motd}
                            onChange={(e) =>
                              updateSettingsField('motd', e.target.value)
                            }
                            disabled={!canApplySettings}
                          />
                        </label>

                        <label>
                          <span>
                            Spawn Protection{' '}
                            <span title="Spawn protection radius in blocks. Set 0 to disable.">
                              <CircleHelp size={14} />
                            </span>
                          </span>
                          <input
                            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:disabled:cursor-not-allowed tw:disabled:opacity-60 tw:max-[1024px]:text-base"
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

                        <label className="tw:col-span-1 tw:flex tw:items-center tw:gap-2 tw:rounded-[10px] tw:border tw:border-border tw:bg-white/[0.02] tw:p-2.5">
                          <input
                            type="checkbox"
                            className="tw:relative tw:grid tw:h-5 tw:w-5 tw:shrink-0 tw:cursor-pointer tw:appearance-none tw:place-content-center tw:rounded tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:text-current tw:transition-all tw:duration-200 tw:before:block tw:before:h-[0.65em] tw:before:w-[0.65em] tw:before:scale-0 tw:before:origin-center tw:before:content-['✓'] tw:before:text-xs tw:before:font-bold tw:before:text-white tw:checked:border-blue-500 tw:checked:bg-blue-500 tw:checked:before:scale-100 tw:disabled:cursor-not-allowed tw:disabled:opacity-60"
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

                        <label className="tw:col-span-1 tw:flex tw:items-center tw:gap-2 tw:rounded-[10px] tw:border tw:border-border tw:bg-white/[0.02] tw:p-2.5">
                          <input
                            type="checkbox"
                            className="tw:relative tw:grid tw:h-5 tw:w-5 tw:shrink-0 tw:cursor-pointer tw:appearance-none tw:place-content-center tw:rounded tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:text-current tw:transition-all tw:duration-200 tw:before:block tw:before:h-[0.65em] tw:before:w-[0.65em] tw:before:scale-0 tw:before:origin-center tw:before:content-['✓'] tw:before:text-xs tw:before:font-bold tw:before:text-white tw:checked:border-blue-500 tw:checked:bg-blue-500 tw:checked:before:scale-100 tw:disabled:cursor-not-allowed tw:disabled:opacity-60"
                            checked={settingsDraft.pvp}
                            onChange={(e) =>
                              updateSettingsField('pvp', e.target.checked)
                            }
                            disabled={!canApplySettings}
                          />
                          <span>Enable PvP</span>
                        </label>

                        <label className="tw:col-span-1 tw:flex tw:items-center tw:gap-2 tw:rounded-[10px] tw:border tw:border-border tw:bg-white/[0.02] tw:p-2.5">
                          <input
                            type="checkbox"
                            className="tw:relative tw:grid tw:h-5 tw:w-5 tw:shrink-0 tw:cursor-pointer tw:appearance-none tw:place-content-center tw:rounded tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:text-current tw:transition-all tw:duration-200 tw:before:block tw:before:h-[0.65em] tw:before:w-[0.65em] tw:before:scale-0 tw:before:origin-center tw:before:content-['✓'] tw:before:text-xs tw:before:font-bold tw:before:text-white tw:checked:border-blue-500 tw:checked:bg-blue-500 tw:checked:before:scale-100 tw:disabled:cursor-not-allowed tw:disabled:opacity-60"
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

                        <label className="tw:col-span-1 tw:flex tw:items-center tw:gap-2 tw:rounded-[10px] tw:border tw:border-border tw:bg-white/[0.02] tw:p-2.5">
                          <input
                            type="checkbox"
                            className="tw:relative tw:grid tw:h-5 tw:w-5 tw:shrink-0 tw:cursor-pointer tw:appearance-none tw:place-content-center tw:rounded tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:text-current tw:transition-all tw:duration-200 tw:before:block tw:before:h-[0.65em] tw:before:w-[0.65em] tw:before:scale-0 tw:before:origin-center tw:before:content-['✓'] tw:before:text-xs tw:before:font-bold tw:before:text-white tw:checked:border-blue-500 tw:checked:bg-blue-500 tw:checked:before:scale-100 tw:disabled:cursor-not-allowed tw:disabled:opacity-60"
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

                        <label className="tw:col-span-1 tw:flex tw:items-center tw:gap-2 tw:rounded-[10px] tw:border tw:border-border tw:bg-white/[0.02] tw:p-2.5">
                          <input
                            type="checkbox"
                            className="tw:relative tw:grid tw:h-5 tw:w-5 tw:shrink-0 tw:cursor-pointer tw:appearance-none tw:place-content-center tw:rounded tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:text-current tw:transition-all tw:duration-200 tw:before:block tw:before:h-[0.65em] tw:before:w-[0.65em] tw:before:scale-0 tw:before:origin-center tw:before:content-['✓'] tw:before:text-xs tw:before:font-bold tw:before:text-white tw:checked:border-blue-500 tw:checked:bg-blue-500 tw:checked:before:scale-100 tw:disabled:cursor-not-allowed tw:disabled:opacity-60"
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

                    <div className="tw:rounded-[14px] tw:border tw:border-border tw:bg-white/[0.02] tw:p-4 tw:max-[640px]:p-3">
                      <div className="tw:mb-4 tw:flex tw:items-center tw:gap-3 tw:max-[640px]:items-center">
                        <div className="tw:flex tw:h-10 tw:w-10 tw:shrink-0 tw:items-center tw:justify-center tw:rounded-xl tw:border tw:border-border tw:bg-white/4">
                          <Gauge size={18} />
                        </div>
                        <div>
                          <h3>Performance</h3>
                          <p>Optimize server performance settings</p>
                        </div>
                      </div>

                      <div className="tw:grid tw:grid-cols-2 tw:gap-3 tw:max-[1024px]:grid-cols-1 tw:[&_label]:flex tw:[&_label]:flex-col tw:[&_label]:gap-1.5 tw:[&_label>span]:inline-flex tw:[&_label>span]:items-center tw:[&_label>span]:gap-1.5 tw:[&_label>span]:text-[0.9rem] tw:[&_label>span]:text-text-muted tw:[&_input[type=range]]:accent-purple-500">
                        <label>
                          <span>
                            Max Players{' '}
                            <span title="Maximum connected players.">
                              <CircleHelp size={14} />
                            </span>
                          </span>
                          <input
                            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:disabled:cursor-not-allowed tw:disabled:opacity-60 tw:max-[1024px]:text-base"
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
                            View Distance{' '}
                            <span title="Chunks sent to players.">
                              <CircleHelp size={14} />
                            </span>
                          </span>
                          <input
                            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:disabled:cursor-not-allowed tw:disabled:opacity-60 tw:max-[1024px]:text-base"
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
                            Simulation Distance{' '}
                            <span title="Chunks actively simulated.">
                              <CircleHelp size={14} />
                            </span>
                          </span>
                          <input
                            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:disabled:cursor-not-allowed tw:disabled:opacity-60 tw:max-[1024px]:text-base"
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
                            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:disabled:cursor-not-allowed tw:disabled:opacity-60 tw:max-[1024px]:text-base"
                            type="number"
                            min={RAM_MIN_MB}
                            max={ramAllocationMaxMb}
                            step={256}
                            value={settingsDraft.ram}
                            onChange={(e) =>
                              updateSettingsField('ram', Number(e.target.value))
                            }
                            disabled={!canApplySettings}
                          />
                        </label>

                        <label>
                          <span>
                            Java Version{' '}
                            <span title="Java runtime used by this server and its managed loader operations.">
                              <CircleHelp size={14} />
                            </span>
                          </span>
                          <select
                            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:disabled:cursor-not-allowed tw:disabled:opacity-60 tw:max-[1024px]:text-base"
                            value={settingsDraft.javaVersion}
                            onChange={(e) =>
                              updateSettingsField(
                                'javaVersion',
                                Number(e.target.value),
                              )
                            }
                            disabled={!canApplySettings}
                          >
                            <option value={0}>
                              Automatic
                              {settingsDraft.requiredJavaVersion > 0
                                ? ` (Java ${settingsDraft.requiredJavaVersion})`
                                : ''}
                            </option>
                            {MANAGED_JAVA_VERSIONS.map((version) => (
                              <option key={version} value={version}>
                                Java {version}
                              </option>
                            ))}
                          </select>
                          {settingsDraft.javaVersion > 0 &&
                            settingsDraft.requiredJavaVersion > 0 &&
                            settingsDraft.javaVersion <
                              settingsDraft.requiredJavaVersion && (
                              <span className="tw:text-[0.8rem] tw:text-amber-300">
                                Java {settingsDraft.javaVersion} is below the
                                required Java{' '}
                                {settingsDraft.requiredJavaVersion} version and
                                may fail to start this server.
                              </span>
                            )}
                        </label>
                      </div>
                    </div>
                  </div>

                  <div className="tw:flex tw:flex-wrap tw:items-center tw:gap-3">
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
                      <p className="tw:m-0 tw:text-[0.85rem] tw:text-amber-300">
                        Stop the server first to modify gameplay or performance
                        settings.
                      </p>
                    )}
                  </div>

                  <div className="tw:box-border tw:min-w-0 tw:rounded-[14px] tw:border tw:border-border tw:bg-bg-card tw:p-3.5 tw:[&_h2]:mt-0 tw:[&_h2]:mb-2">
                    <div className="tw:mb-4 tw:flex tw:items-center tw:gap-3 tw:max-[640px]:items-center">
                      <div className="tw:flex tw:h-10 tw:w-10 tw:shrink-0 tw:items-center tw:justify-center tw:rounded-xl tw:border tw:border-border tw:bg-white/4">
                        <Upload size={18} />
                      </div>
                      <div>
                        <h3>Server Icon</h3>
                        <p>Upload a new icon for this server</p>
                      </div>
                    </div>
                    <div className="tw:flex tw:flex-wrap tw:items-start tw:gap-3.5 tw:max-[640px]:flex-nowrap tw:max-[640px]:gap-2.5">
                      <div className="tw:flex tw:h-16 tw:w-16 tw:shrink-0 tw:items-center tw:justify-center tw:overflow-hidden tw:rounded-xl tw:border tw:border-border tw:bg-white/4 [&_img]:h-full [&_img]:w-full [&_img]:object-fill [&_img]:[image-rendering:pixelated] [&_span]:text-[1.35rem] [&_span]:font-bold [&_span]:text-text-muted">
                        {settingsIconPreview && (
                          <img
                            src={settingsIconPreview}
                            alt="Selected server icon"
                          />
                        )}
                        {!settingsIconPreview && !settingsIconError && (
                          <img
                            src={`${api.getServerIconUrl(server.id)}?v=${serverIconVersion}`}
                            alt="Current server icon"
                            onError={() => setSettingsIconError(true)}
                          />
                        )}
                        {!settingsIconPreview && settingsIconError && (
                          <span>{server.name.charAt(0).toUpperCase()}</span>
                        )}
                      </div>
                      <div className="tw:flex tw:min-w-0 tw:flex-1 tw:flex-col tw:gap-2 tw:max-[640px]:w-auto">
                        <input
                          ref={settingsIconInputRef}
                          type="file"
                          aria-label="Server icon image"
                          accept="image/png,image/jpeg"
                          onChange={handleSettingsIconSelected}
                          hidden
                        />
                        <div className="tw:flex tw:flex-wrap tw:justify-start tw:gap-2.5 tw:max-[640px]:!flex-col tw:max-[640px]:!items-stretch tw:max-[640px]:!gap-2">
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
                        <p className="tw:col-span-full tw:m-0 tw:text-[0.85rem] tw:text-text-muted">
                          Recommended size: 64x64. Supported formats: PNG, JPG.
                        </p>
                      </div>
                    </div>
                  </div>

                  <div className="tw:box-border tw:min-w-0 tw:rounded-[14px] tw:border tw:border-border tw:bg-bg-card tw:p-3.5 tw:[&_h2]:mt-0 tw:[&_h2]:mb-2">
                    <div className="tw:mb-4 tw:flex tw:items-center tw:gap-3 tw:max-[640px]:items-center">
                      <div className="tw:flex tw:h-10 tw:w-10 tw:shrink-0 tw:items-center tw:justify-center tw:rounded-xl tw:border tw:border-border tw:bg-white/4">
                        <Download size={18} />
                      </div>
                      <div>
                        <h3>Server Version</h3>
                        <p>Update the Minecraft version for this server</p>
                      </div>
                    </div>
                    <div className="tw:grid tw:grid-cols-2 tw:gap-3 tw:max-[1024px]:grid-cols-1 tw:[&_label]:flex tw:[&_label]:flex-col tw:[&_label]:gap-1.5 tw:[&_label>span]:inline-flex tw:[&_label>span]:items-center tw:[&_label>span]:gap-1.5 tw:[&_label>span]:text-[0.9rem] tw:[&_label>span]:text-text-muted tw:[&_input[type=range]]:accent-purple-500">
                      <label>
                        <span>
                          Select New Version{' '}
                          <span title="Only versions available for the current loader are shown.">
                            <CircleHelp size={14} />
                          </span>
                        </span>
                        <select
                          className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:disabled:cursor-not-allowed tw:disabled:opacity-60 tw:max-[1024px]:text-base"
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
                      <div className="tw:flex tw:flex-wrap tw:justify-start tw:gap-2.5">
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
                        <p className="tw:col-span-full tw:m-0 tw:text-[0.85rem] tw:text-text-muted">
                          Current version ({server.version}) is already the
                          latest available.
                        </p>
                      )}
                    </div>
                  </div>

                  <div className="tw:box-border tw:min-w-0 tw:rounded-[14px] tw:border tw:border-rose-500/35 tw:bg-bg-card tw:p-3.5 tw:[&_h2]:mt-0 tw:[&_h2]:mb-2">
                    <div className="tw:mb-4 tw:flex tw:items-center tw:gap-3 tw:max-[640px]:items-center">
                      <div className="tw:flex tw:h-10 tw:w-10 tw:shrink-0 tw:items-center tw:justify-center tw:rounded-xl tw:border tw:border-rose-500/35 tw:bg-rose-500/12 tw:text-rose-300">
                        <Trash2 size={18} />
                      </div>
                      <div>
                        <h3>Delete Server</h3>
                        <p>Permanently delete this server and all its files</p>
                      </div>
                    </div>
                    <p className="tw:my-2 tw:mb-3.5 tw:rounded-[10px] tw:border tw:border-rose-500/35 tw:bg-rose-500/15 tw:p-3 tw:text-rose-200">
                      Warning: this action cannot be undone. All worlds,
                      configurations, and related files will be deleted.
                    </p>
                    {!isServerStopped && (
                      <p>Stop the server first before deleting it.</p>
                    )}
                    <Button
                      type="button"
                      variant="danger"
                      onClick={() => {
                        setDeleteConfirmName('');
                        setIsDeleteModalOpen(true);
                      }}
                      disabled={isDeletingServer || !isServerStopped}
                    >
                      Delete Server
                    </Button>
                  </div>
                </>
              )}
            </div>
          )}
        </section>

        <ServerDetailNav
          activeTab={activeTab}
          addonsLabel={addonsLabel}
          supportsAddons={supportsAddons}
          onSelect={setActiveTab}
        />
      </div>

      <Modal
        isOpen={isPlayerActionsOpen}
        onClose={() => setIsPlayerActionsOpen(false)}
        title="Player Actions"
      >
        <div className="tw:px-5 tw:pb-5">
          <p className="tw:mt-0 tw:text-text-muted">
            Select an action for{' '}
            <strong>{selectedPlayer?.name || 'player'}</strong>.
          </p>

          <div className="tw:flex tw:flex-col tw:gap-2.5">
            <button
              type="button"
              className="tw:flex tw:w-full tw:cursor-pointer tw:items-center tw:gap-2 tw:rounded-[10px] tw:border tw:border-border tw:bg-white/4 tw:px-3 tw:py-2.5 tw:text-text-main tw:disabled:cursor-not-allowed tw:disabled:opacity-50"
              disabled={isPlayerActionLoading || !selectedPlayer?.isOnline}
              title={
                selectedPlayer?.isOnline === false
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
              className="tw:flex tw:w-full tw:cursor-pointer tw:items-center tw:gap-2 tw:rounded-[10px] tw:border tw:border-border tw:bg-white/4 tw:px-3 tw:py-2.5 tw:text-text-main tw:disabled:cursor-not-allowed tw:disabled:opacity-50"
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
              className="tw:flex tw:w-full tw:cursor-pointer tw:items-center tw:gap-2 tw:rounded-[10px] tw:border tw:border-pink-400/35 tw:bg-pink-400/10 tw:px-3 tw:py-2.5 tw:text-pink-300 tw:disabled:cursor-not-allowed tw:disabled:opacity-50"
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
              className="tw:flex tw:w-full tw:cursor-pointer tw:items-center tw:gap-2 tw:rounded-[10px] tw:border tw:border-pink-400/35 tw:bg-pink-400/10 tw:px-3 tw:py-2.5 tw:text-pink-300 tw:disabled:cursor-not-allowed tw:disabled:opacity-50"
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
              className="tw:flex tw:w-full tw:cursor-pointer tw:items-center tw:gap-2 tw:rounded-[10px] tw:border tw:border-red-300/35 tw:bg-red-300/10 tw:px-3 tw:py-2.5 tw:text-red-200 tw:disabled:cursor-not-allowed tw:disabled:opacity-50"
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
        <div className="tw:flex tw:flex-col tw:gap-3">
          <p>{settingsModalMessage}</p>
          <div className="tw:mt-[25px] tw:flex tw:justify-end tw:gap-2.5 tw:max-[769px]:flex-col tw:max-[769px]:gap-2">
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
        <div className="tw:flex tw:flex-col tw:gap-3">
          {isUpdatingVersion && renderVersionUpdateProgress()}
          {!isUpdatingVersion &&
            versionUpdateResult &&
            renderVersionUpdateResult()}
          {!isUpdatingVersion && versionUpdateError && (
            <div className="tw:flex tw:flex-col tw:gap-4">
              <div className="tw:flex tw:items-start tw:gap-3 tw:rounded-2xl tw:border tw:border-red-400/25 tw:bg-red-400/12 tw:p-3.5 tw:text-red-300">
                <Ban size={22} />
                <div>
                  <strong className="tw:block tw:text-slate-50">
                    Update failed
                  </strong>
                  <span className="tw:mt-[3px] tw:block tw:text-slate-300">
                    {versionUpdateError}
                  </span>
                </div>
              </div>
            </div>
          )}
          {!isUpdatingVersion && (
            <div className="tw:mt-[25px] tw:flex tw:justify-end tw:gap-2.5 tw:max-[769px]:flex-col tw:max-[769px]:gap-2">
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
        <div className="tw:flex tw:flex-col tw:gap-3">
          <p>{iconUploadModalMessage}</p>
          <div className="tw:mt-[25px] tw:flex tw:justify-end tw:gap-2.5 tw:max-[769px]:flex-col tw:max-[769px]:gap-2">
            <Button
              type="button"
              variant="primary"
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
        <div className="tw:flex tw:flex-col tw:gap-3">
          <p>
            Do you want to delete server <strong>{server?.name}</strong>?
          </p>
          <p>Type the server name to confirm deletion.</p>
          <input
            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:disabled:cursor-not-allowed tw:disabled:opacity-60 tw:max-[1024px]:text-base"
            aria-label="Confirm server name"
            value={deleteConfirmName}
            onChange={(e) => setDeleteConfirmName(e.target.value)}
            placeholder="Type the server name"
          />
          <div className="tw:mt-[25px] tw:flex tw:justify-end tw:gap-2.5 tw:max-[769px]:flex-col tw:max-[769px]:gap-2">
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
