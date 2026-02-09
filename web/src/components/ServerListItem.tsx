import { Play, Square, Terminal, Trash2 } from 'lucide-react';
import { Link } from 'react-router-dom';

import React, { useState } from 'react';

import { useAuth } from '../context/AuthContext';
import { useCopy } from '../hooks/useCopy';
import { api } from '../services/api';
import type { Server, ServerStats } from '../types';
import { formatBytes } from '../utils/format';
import { Button } from './ui/Button';
import { CopyButton } from './ui/CopyButton';

interface ServerListItemProps {
  server: Server;
  stats?: ServerStats;
  onStart: (id: string) => void;
  onStop: (id: string) => void;
  onDelete: (id: string) => void;
}

const ServerListItem: React.FC<ServerListItemProps> = ({
  server,
  stats,
  onStart,
  onStop,
  onDelete,
}) => {
  const { user } = useAuth();
  const [iconError, setIconError] = useState(false);
  const { copy } = useCopy(1200);

  const hostIp =
    typeof window !== 'undefined' ? window.location.hostname : 'localhost';
  const address = `${hostIp}:${server.port}`;

  const handleCopyAddress = () => {
    copy(address);
  };

  if (server.status === 'CREATING') {
    return (
      <div className="server-list-item creating">
        <div className="server-info-section">
          <div className="status-dot status-creating"></div>

          <div className="server-icon-placeholder-list">
            {server.name.charAt(0).toUpperCase()}
          </div>

          <div className="server-details">
            <div className="server-name-row">
              <span className="server-name">{server.name}</span>
            </div>
            <div className="server-meta">Creating...</div>
          </div>
        </div>
        <div className="server-stats">
          <div className="creating-progress">
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
    <div className="server-list-item">
      <div className="server-info-section">
        <div
          className={`status-dot status-${server.status.toLowerCase()}`}
        ></div>

        {!iconError ? (
          <img
            src={api.getServerIconUrl(server.id)}
            alt="Server Icon"
            onError={() => setIconError(true)}
            className="server-icon-list"
          />
        ) : (
          <div className="server-icon-placeholder-list">
            {server.name.charAt(0).toUpperCase()}
          </div>
        )}

        <div className="server-details">
          <div className="server-name-row">
            <span className="server-name">{server.name}</span>
          </div>
          <div className="server-meta">
            {server.loader} {server.version}
          </div>

          <div className="server-meta">
            <span
              role="button"
              tabIndex={0}
              onClick={handleCopyAddress}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  handleCopyAddress();
                }
              }}
              aria-label="Copiar dirección"
              className="address-text"
            >
              {address}
            </span>

            <CopyButton
              text={address}
              aria-label="Copiar dirección"
              className="address-copy-btn"
            />
          </div>
        </div>
      </div>

      <div className="server-stats-actions">
        <div className="stat-group">
          <div className="stat-label">CPU</div>
          <div className="stat-value">
            {isRunning && stats ? `${stats.cpu.toFixed(1)}%` : '0.0%'}
          </div>
        </div>

        <div className="stat-group">
          <div className="stat-label">Memory</div>
          <div className="stat-value">
            {isRunning && stats ? formatBytes(stats.ram) : '0 B'}
          </div>
        </div>

        <div className="stat-group">
          <div className="stat-label">Disk</div>
          <div className="stat-value">
            {stats ? formatBytes(stats.disk) : '0 B'}
          </div>
        </div>

        <div className="actions-group">
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
              className="icon-action console-btn"
              title="Console"
            >
              <Terminal size={18} />
            </Link>
          )}

          {user?.role === 'admin' && (
            <button
              className="icon-action danger"
              onClick={() => onDelete(server.id)}
              title="Delete"
            >
              <Trash2 size={18} />
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

export default ServerListItem;
