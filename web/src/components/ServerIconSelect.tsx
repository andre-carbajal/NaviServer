import React, { useEffect, useMemo, useRef, useState } from 'react';

import { api } from '../services/api';
import type { Server } from '../types';

interface ServerIconSelectProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  servers: Server[];
  placeholder?: string;
  allowNone?: boolean;
  noneLabel?: string;
  disabled?: boolean;
}

const ServerIconImage: React.FC<{ server: Server }> = ({ server }) => {
  const [error, setError] = useState(false);

  if (!error) {
    return (
      <img
        src={api.getServerIconUrl(server.id)}
        alt={`${server.name} icon`}
        className="server-icon-select-image"
        onError={() => setError(true)}
      />
    );
  }

  return (
    <div className="server-icon-select-fallback">
      {server.name.charAt(0).toUpperCase()}
    </div>
  );
};

const ServerIconSelect: React.FC<ServerIconSelectProps> = ({
  label,
  value,
  onChange,
  servers,
  placeholder = 'Select a server',
  allowNone = false,
  noneLabel = 'None',
  disabled = false,
}) => {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);

  const selectedServer = useMemo(
    () => servers.find((server) => server.id === value),
    [servers, value],
  );

  useEffect(() => {
    const handleOutside = (event: MouseEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handleOutside);
    return () => document.removeEventListener('mousedown', handleOutside);
  }, []);

  const canOpen = !disabled;

  return (
    <div className="form-group">
      <label>{label}</label>
      <div
        ref={rootRef}
        className={`server-icon-select ${disabled ? 'is-disabled' : ''}`}
      >
        <button
          type="button"
          className="server-icon-select-trigger"
          onClick={() => {
            if (canOpen) {
              setOpen((prev) => !prev);
            }
          }}
          disabled={disabled}
        >
          {selectedServer ? (
            <>
              <ServerIconImage server={selectedServer} />
              <span className="server-icon-select-text">
                {selectedServer.name}
              </span>
            </>
          ) : (
            <span className="server-icon-select-placeholder">
              {allowNone && value === '' ? noneLabel : placeholder}
            </span>
          )}
          <span className="server-icon-select-caret">{open ? '▲' : '▼'}</span>
        </button>
        {open && (
          <div className="server-icon-select-menu">
            {allowNone && (
              <button
                type="button"
                className="server-icon-select-option"
                onClick={() => {
                  onChange('');
                  setOpen(false);
                }}
              >
                <div className="server-icon-select-fallback">-</div>
                <span>{noneLabel}</span>
              </button>
            )}
            {servers.map((server) => (
              <button
                type="button"
                key={server.id}
                className={`server-icon-select-option ${value === server.id ? 'is-selected' : ''}`}
                onClick={() => {
                  onChange(server.id);
                  setOpen(false);
                }}
              >
                <ServerIconImage server={server} />
                <div className="server-icon-select-option-text">
                  <span>{server.name}</span>
                  <small>{server.id}</small>
                </div>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default ServerIconSelect;
