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
  <div className="tw:rounded-xl tw:border tw:border-border tw:bg-bg-card tw:p-5">
    <h2 className="tw:mt-0 tw:mr-0 tw:mb-3 tw:ml-0">Automatic Backups</h2>
    {!canConfigure ? (
      <p className="tw:text-text-muted">
        Only administrators can configure automatic backups.
      </p>
    ) : (
      <div className="tw:grid tw:gap-2.5">
        <div className="tw:grid tw:grid-cols-[minmax(180px,1.4fr)_auto_minmax(170px,1fr)_110px_auto] tw:gap-2.5 tw:px-2.5 tw:text-[0.8rem] tw:font-semibold tw:tracking-[0.04em] tw:text-text-muted tw:uppercase tw:max-[1201px]:hidden">
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
            <div
              key={server.id}
              className="tw:grid tw:grid-cols-[minmax(180px,1.4fr)_auto_minmax(170px,1fr)_110px_auto] tw:items-center tw:gap-2.5 tw:rounded-[10px] tw:border tw:border-border tw:p-2.5 tw:max-[1201px]:grid-cols-1"
            >
              <div className="tw:flex tw:min-w-0 tw:items-center tw:gap-2.5">
                <img
                  src={api.getServerIconUrl(server.id)}
                  alt="Server icon"
                  className="tw:h-7 tw:w-7 tw:rounded-md tw:object-contain [image-rendering:pixelated]"
                  onError={(event) => {
                    const target = event.currentTarget;
                    target.style.display = 'none';
                    const fallback = target.nextElementSibling;
                    if (fallback instanceof HTMLElement) {
                      fallback.style.display = 'flex';
                    }
                  }}
                />
                <div className="tw:hidden tw:h-7 tw:w-7 tw:items-center tw:justify-center tw:rounded-md tw:bg-white/10 tw:text-[0.85rem] tw:font-semibold tw:text-text-muted [image-rendering:pixelated]">
                  {server.name.charAt(0).toUpperCase()}
                </div>
                <div>
                  <strong>{server.name}</strong>
                  <div className="tw:text-text-muted">{server.id}</div>
                </div>
              </div>
              <label className="tw:flex tw:items-center tw:gap-2">
                <input
                  type="checkbox"
                  className="tw:grid tw:h-5 tw:w-5 tw:shrink-0 tw:cursor-pointer tw:appearance-none tw:place-content-center tw:rounded tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:p-0 tw:before:block tw:before:h-[0.65em] tw:before:w-[0.65em] tw:before:scale-0 tw:before:origin-center tw:before:content-['✓'] tw:before:text-xs tw:before:font-bold tw:before:text-white tw:checked:border-blue-500 tw:checked:bg-blue-500 tw:checked:before:scale-100 tw:disabled:cursor-not-allowed tw:disabled:opacity-60"
                  checked={draft.enabled}
                  onChange={(event) =>
                    onUpdate(server.id, { enabled: event.target.checked }, true)
                  }
                />
                <span>Enabled</span>
              </label>
              <div className="tw:grid tw:grid-cols-2 tw:gap-2 tw:max-[1201px]:grid-cols-1">
                <input
                  type="number"
                  aria-label={`${server.name} auto backup interval value`}
                  min={1}
                  className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
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
                  className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
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
              <div className="tw:min-w-0">
                <input
                  type="number"
                  aria-label={`${server.name} maximum automatic backups`}
                  min={1}
                  className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
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
              <Button
                onClick={() => onSave(server.id)}
                disabled={draft.saving}
                className="tw:max-[1201px]:w-full tw:max-[1201px]:justify-center"
              >
                {draft.saving ? 'Saving...' : 'Save'}
              </Button>
              {draft.saved && (
                <span className="tw:text-[0.85rem] tw:font-semibold tw:text-green-500">
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
