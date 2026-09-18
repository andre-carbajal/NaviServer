import { BarChart3, Play, Square } from 'lucide-react';
import { Link } from 'react-router-dom';

import React, { useEffect, useState } from 'react';

import { useCopy } from '../../hooks/useCopy';
import { api } from '../../services/api';
import type { Server, ServerStats } from '../../types';
import { formatBytes } from '../../utils/format';
import { Button } from '../ui/Button';
import { CopyButton } from '../ui/CopyButton';

interface ServerListItemProps {
  server: Server;
  stats?: ServerStats;
  onStart: (id: string) => void;
  onStop: (id: string) => void;
}

const ServerListItem: React.FC<ServerListItemProps> = ({
  server,
  stats,
  onStart,
  onStop,
}) => {
  const [iconError, setIconError] = useState(false);
  const [publicIP, setPublicIP] = useState<string>(
    typeof window !== 'undefined' ? window.location.hostname : 'localhost',
  );
  const { copy } = useCopy(1200);

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

  const address = `${publicIP}:${server.port}`;

  const handleCopyAddress = () => {
    copy(address);
  };

  if (server.status === 'CREATING') {
    return (
      <div className="tw:flex tw:min-h-0 tw:flex-col tw:items-start tw:justify-between tw:gap-3 tw:rounded-lg tw:border tw:border-border tw:bg-bg-card tw:px-4 tw:py-3 tw:transition-colors tw:duration-200 tw:hover:bg-white/5 tw:min-[1025px]:min-h-[72px] tw:min-[1025px]:flex-row tw:min-[1025px]:items-center tw:min-[1025px]:px-5 tw:min-[1025px]:py-4">
        <div className="tw:flex tw:min-w-0 tw:items-center tw:gap-4 tw:max-[1025px]:w-full">
          <div className="tw:h-2.5 tw:w-2.5 tw:shrink-0 tw:animate-pulse tw:rounded-full tw:bg-blue-500"></div>

          <div className="tw:box-border tw:flex tw:aspect-square tw:h-9 tw:w-9 tw:shrink-0 tw:items-center tw:justify-center tw:rounded-sm tw:bg-white/10 tw:text-base tw:font-semibold tw:text-text-muted">
            {server.name.charAt(0).toUpperCase()}
          </div>

          <div className="tw:flex tw:min-w-0 tw:overflow-hidden tw:flex-col tw:gap-1">
            <div className="tw:flex tw:items-center tw:gap-2">
              <span className="tw:overflow-hidden tw:text-[1.1rem] tw:font-semibold tw:text-ellipsis tw:whitespace-nowrap tw:text-text-main">
                {server.name}
              </span>
            </div>
            <div className="tw:flex tw:items-center tw:gap-1.5 tw:text-[0.85rem] tw:text-text-muted">
              Creating...
            </div>
          </div>
        </div>
        <div>
          <div className="tw:rounded-sm tw:bg-blue-500/10 tw:px-3 tw:py-1 tw:font-mono tw:text-[0.9rem] tw:text-blue-500">
            {server.steps && server.steps.length > 0
              ? server.steps[server.steps.length - 1].label
              : 'Initializing...'}
          </div>
        </div>
      </div>
    );
  }

  const isRunning = server.status === 'RUNNING';

  return (
    <div className="tw:flex tw:min-h-0 tw:flex-col tw:items-start tw:justify-between tw:gap-3 tw:rounded-lg tw:border tw:border-border tw:bg-bg-card tw:px-4 tw:py-3 tw:transition-colors tw:duration-200 tw:hover:bg-white/5 tw:min-[1025px]:min-h-[72px] tw:min-[1025px]:flex-row tw:min-[1025px]:items-center tw:min-[1025px]:px-5 tw:min-[1025px]:py-4">
      <div className="tw:flex tw:min-w-0 tw:items-center tw:gap-4 tw:max-[1025px]:w-full">
        <div
          className={`tw:h-2.5 tw:w-2.5 tw:shrink-0 tw:rounded-full ${isRunning ? 'tw:bg-emerald-500 tw:shadow-[0_0_10px_rgba(16,185,129,0.4)]' : 'tw:bg-red-500'}`}
        ></div>

        {!iconError ? (
          <img
            src={api.getServerIconUrl(server.id)}
            alt="Server Icon"
            onError={() => setIconError(true)}
            className="tw:aspect-square tw:h-9 tw:w-9 tw:shrink-0 tw:rounded-sm tw:bg-black/20 tw:object-contain [image-rendering:pixelated]"
          />
        ) : (
          <div className="tw:box-border tw:flex tw:aspect-square tw:h-9 tw:w-9 tw:shrink-0 tw:items-center tw:justify-center tw:rounded-sm tw:bg-white/10 tw:text-base tw:font-semibold tw:text-text-muted">
            {server.name.charAt(0).toUpperCase()}
          </div>
        )}

        <div className="tw:flex tw:min-w-0 tw:overflow-hidden tw:flex-col tw:gap-1">
          <div className="tw:flex tw:items-center tw:gap-2">
            <span className="tw:overflow-hidden tw:text-[1.1rem] tw:font-semibold tw:text-ellipsis tw:whitespace-nowrap tw:text-text-main">
              {server.name}
            </span>
          </div>
          <div className="tw:flex tw:min-w-0 tw:items-center tw:gap-[15px] tw:text-[0.85rem] tw:text-text-muted">
            <div className="tw:flex tw:items-center tw:gap-1.5 tw:whitespace-nowrap">
              <span className="tw:font-semibold tw:text-text-main">
                {server.loader}
              </span>
              <span>{server.version}</span>
            </div>
            <div className="tw:h-1 tw:w-1 tw:rounded-full tw:bg-text-muted"></div>
            <div className="tw:flex tw:min-w-0 tw:items-center tw:gap-1.5 tw:whitespace-nowrap">
              <button
                type="button"
                onClick={handleCopyAddress}
                aria-label="Copiar dirección"
                className="tw:inline-flex tw:h-6 tw:min-w-0 tw:max-w-[140px] tw:shrink-0 tw:cursor-pointer tw:appearance-none tw:items-center tw:overflow-hidden tw:rounded-[7px] tw:border tw:border-white/10 tw:bg-white/4 tw:px-2 tw:font-mono tw:text-[0.78rem] tw:leading-none tw:text-ellipsis tw:whitespace-nowrap tw:text-text-muted tw:transition-colors tw:duration-150 tw:hover:border-indigo-500/55 tw:hover:bg-indigo-500/12 tw:hover:text-text-main tw:focus-visible:outline-2 tw:focus-visible:outline-offset-2 tw:focus-visible:outline-indigo-500/70"
                title="Click to copy"
              >
                {address}
              </button>

              <CopyButton
                text={address}
                aria-label="Copiar dirección"
                variant="secondary"
                title="Copy to clipboard"
                className="tw:inline-flex tw:!h-6 tw:!min-h-6 tw:!w-6 tw:!min-w-0 tw:!p-0 tw:shrink-0 tw:items-center tw:justify-center tw:overflow-visible tw:rounded-md tw:border tw:border-white/4 tw:bg-white/2 tw:text-text-muted tw:transition-none tw:hover:border-white/8 tw:hover:bg-white/6 tw:hover:text-white tw:focus:outline-2 tw:focus:outline-offset-2 tw:focus:outline-primary/20"
              />
            </div>
          </div>
        </div>
      </div>

      <div className="tw:grid tw:w-full tw:shrink-0 tw:grid-cols-4 tw:items-center tw:gap-3 tw:border-t tw:border-border tw:pt-3 tw:min-[1025px]:ml-auto tw:min-[1025px]:flex tw:min-[1025px]:w-auto tw:min-[1025px]:gap-8 tw:min-[1025px]:border-0 tw:min-[1025px]:pt-0">
        <div className="tw:flex tw:min-w-0 tw:flex-col tw:items-center tw:gap-0.5 tw:min-[1025px]:min-w-[70px] tw:min-[1025px]:items-end">
          <div className="tw:text-[0.65rem] tw:font-medium tw:tracking-[0.02em] tw:text-text-muted tw:uppercase tw:min-[1025px]:text-xs">
            CPU
          </div>
          <div className="tw:font-mono tw:text-[0.85rem] tw:font-semibold tw:text-text-main tw:min-[1025px]:text-[0.95rem]">
            {isRunning && stats ? `${stats.cpu.toFixed(1)}%` : '0.0%'}
          </div>
        </div>

        <div className="tw:flex tw:min-w-0 tw:flex-col tw:items-center tw:gap-0.5 tw:min-[1025px]:min-w-[70px] tw:min-[1025px]:items-end">
          <div className="tw:text-[0.65rem] tw:font-medium tw:tracking-[0.02em] tw:text-text-muted tw:uppercase tw:min-[1025px]:text-xs">
            Memory
          </div>
          <div className="tw:font-mono tw:text-[0.85rem] tw:font-semibold tw:text-text-main tw:min-[1025px]:text-[0.95rem]">
            {isRunning && stats
              ? `${formatBytes(stats.ram)} / ${formatBytes(server.ram * 1024 * 1024)}`
              : `0 B / ${formatBytes(server.ram * 1024 * 1024)}`}
          </div>
        </div>

        <div className="tw:flex tw:min-w-0 tw:flex-col tw:items-center tw:gap-0.5 tw:min-[1025px]:min-w-[70px] tw:min-[1025px]:items-end">
          <div className="tw:text-[0.65rem] tw:font-medium tw:tracking-[0.02em] tw:text-text-muted tw:uppercase tw:min-[1025px]:text-xs">
            Disk
          </div>
          <div className="tw:font-mono tw:text-[0.85rem] tw:font-semibold tw:text-text-main tw:min-[1025px]:text-[0.95rem]">
            {stats ? formatBytes(stats.disk) : '0 B'}
          </div>
        </div>

        <div className="tw:flex tw:min-w-0 tw:flex-col tw:items-center tw:gap-0.5 tw:min-[1025px]:min-w-[70px] tw:min-[1025px]:items-end">
          <div className="tw:text-[0.65rem] tw:font-medium tw:tracking-[0.02em] tw:text-text-muted tw:uppercase tw:min-[1025px]:text-xs">
            Players
          </div>
          <div className="tw:font-mono tw:text-[0.85rem] tw:font-semibold tw:text-text-main tw:min-[1025px]:text-[0.95rem]">
            {isRunning && stats
              ? `${stats.onlinePlayers} / ${stats.maxPlayers}`
              : '0 / 0'}
          </div>
        </div>

        <div className="tw:col-span-full tw:flex tw:w-full tw:min-w-0 tw:flex-wrap tw:items-center tw:justify-between tw:gap-3 tw:border-t tw:border-border tw:pt-3 tw:min-[1025px]:ml-3 tw:min-[1025px]:w-auto tw:min-[1025px]:min-w-[180px] tw:min-[1025px]:flex-nowrap tw:min-[1025px]:justify-start tw:min-[1025px]:border-t-0 tw:min-[1025px]:border-l tw:min-[1025px]:pt-0 tw:min-[1025px]:pl-3">
          {(server.permissions?.canControlPower ||
            server.permissions?.canViewConsole) &&
            (isRunning ? (
              <Button variant="danger" onClick={() => onStop(server.id)}>
                <Square size={16} fill="currentColor" /> Stop
              </Button>
            ) : (
              <Button
                onClick={() => onStart(server.id)}
                disabled={server.status !== 'STOPPED'}
              >
                <Play size={16} /> Start
              </Button>
            ))}

          {server.permissions?.canViewConsole && (
            <Link
              to={`/servers/${server.id}`}
              className="tw:flex tw:h-9 tw:w-9 tw:cursor-pointer tw:items-center tw:justify-center tw:rounded-sm tw:border-0 tw:bg-white/10 tw:p-0 tw:text-white tw:transition-all tw:duration-200 tw:hover:bg-white/20"
              title="Open server dashboard"
            >
              <BarChart3 size={18} />
            </Link>
          )}
        </div>
      </div>
    </div>
  );
};

export default ServerListItem;
