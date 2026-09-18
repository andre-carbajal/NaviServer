import React from 'react';

import { api } from '../../services/api';
import type { Server } from '../../types';
import { Button } from '../ui/Button';

export type AutoBackupUnit = 'minute' | 'hour' | 'day';

export interface AutoBackupDraft {
  enabled: boolean;
  intervalValue: number;
  intervalUnit: AutoBackupUnit;
  maxBackups: number;
  saving: boolean;
  dirty: boolean;
  saved: boolean;
}

interface AutomaticBackupSettingsProps {
  canConfigure: boolean;
  servers: Server[];
  drafts: Record<string, AutoBackupDraft>;
  onUpdate: (
    serverId: string,
    patch: Partial<AutoBackupDraft>,
    markDirty?: boolean,
  ) => void;
  onSave: (serverId: string) => void;
}

const AutomaticBackupSettings: React.FC<AutomaticBackupSettingsProps> = ({
  canConfigure,
  servers,
  drafts,
  onUpdate,
  onSave,
}) => (
  <div className="card">
    <h2 className="backup-section-title">Automatic Backups</h2>
    {!canConfigure ? (
      <p className="text-muted">
        Only administrators can configure automatic backups.
      </p>
    ) : (
      <div className="auto-backup-grid">
        <div className="auto-backup-header">
          <span>Server</span>
          <span>Enabled</span>
          <span>Every</span>
          <span>Max backups</span>
          <span>Action</span>
        </div>
        {servers.map((server) => {
          const draft = drafts[server.id];
          if (!draft) return null;

          return (
            <div key={server.id} className="auto-backup-row">
              <div className="auto-backup-row-main">
                <img
                  src={api.getServerIconUrl(server.id)}
                  alt="Server icon"
                  className="auto-backup-server-icon"
                  onError={(event) => {
                    const target = event.currentTarget;
                    target.style.display = 'none';
                    const fallback = target.nextElementSibling;
                    if (fallback instanceof HTMLElement) {
                      fallback.style.display = 'flex';
                    }
                  }}
                />
                <div
                  className="auto-backup-server-fallback"
                  style={{ display: 'none' }}
                >
                  {server.name.charAt(0).toUpperCase()}
                </div>
                <div>
                  <strong>{server.name}</strong>
                  <div className="text-muted">{server.id}</div>
                </div>
              </div>
              <label className="auto-backup-toggle">
                <input
                  type="checkbox"
                  checked={draft.enabled}
                  onChange={(event) =>
                    onUpdate(server.id, { enabled: event.target.checked }, true)
                  }
                />
                <span>Enabled</span>
              </label>
              <div className="auto-backup-interval">
                <input
                  type="number"
                  aria-label={`${server.name} auto backup interval value`}
                  min={1}
                  className="form-input"
                  value={draft.intervalValue}
                  onChange={(event) =>
                    onUpdate(
                      server.id,
                      { intervalValue: Number(event.target.value) },
                      true,
                    )
                  }
                />
                <select
                  aria-label={`${server.name} auto backup interval unit`}
                  className="form-select"
                  value={draft.intervalUnit}
                  onChange={(event) =>
                    onUpdate(
                      server.id,
                      { intervalUnit: event.target.value as AutoBackupUnit },
                      true,
                    )
                  }
                >
                  <option value="minute">Minutes</option>
                  <option value="hour">Hours</option>
                  <option value="day">Days</option>
                </select>
              </div>
              <div className="auto-backup-limit">
                <input
                  type="number"
                  aria-label={`${server.name} maximum automatic backups`}
                  min={1}
                  className="form-input"
                  value={draft.maxBackups}
                  onChange={(event) =>
                    onUpdate(
                      server.id,
                      { maxBackups: Number(event.target.value) },
                      true,
                    )
                  }
                />
              </div>
              <Button onClick={() => onSave(server.id)} disabled={draft.saving}>
                {draft.saving ? 'Saving...' : 'Save'}
              </Button>
              {draft.saved && (
                <span
                  style={{
                    color: '#22c55e',
                    fontSize: '0.85rem',
                    fontWeight: 600,
                  }}
                >
                  Saved successfully
                </span>
              )}
            </div>
          );
        })}
      </div>
    )}
  </div>
);

export default AutomaticBackupSettings;
