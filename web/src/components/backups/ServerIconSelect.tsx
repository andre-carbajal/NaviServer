import React, { useEffect, useMemo, useRef, useState } from 'react';

import { api } from '../../services/api';
import type { Server } from '../../types';

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
        className="tw:h-6 tw:w-6 tw:rounded-md tw:object-contain [image-rendering:pixelated]"
        onError={() => setError(true)}
      />
    );
  }

  return (
    <div className="tw:inline-flex tw:h-6 tw:w-6 tw:items-center tw:justify-center tw:rounded-md tw:bg-white/10 tw:text-xs tw:font-semibold tw:text-text-muted [image-rendering:pixelated]">
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
    <div className="tw:mb-[15px] tw:max-[769px]:mb-3">
      <label className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted tw:max-[769px]:mb-1.5 tw:max-[769px]:text-[0.85rem]">
        {label}
      </label>
      <div ref={rootRef} className="tw:relative">
        <button
          type="button"
          className="tw:flex tw:w-full tw:cursor-pointer tw:items-center tw:gap-2 tw:rounded-lg tw:border tw:border-border tw:bg-bg-card tw:p-2.5 tw:text-text-main tw:disabled:cursor-not-allowed tw:disabled:opacity-60"
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
              <span className="tw:overflow-hidden tw:text-ellipsis tw:whitespace-nowrap">
                {selectedServer.name}
              </span>
            </>
          ) : (
            <span className="tw:text-text-muted">
              {allowNone && value === '' ? noneLabel : placeholder}
            </span>
          )}
          <span className="tw:ml-auto tw:text-xs tw:text-text-muted">
            {open ? '▲' : '▼'}
          </span>
        </button>
        {open && (
          <div className="tw:absolute tw:z-60 tw:mt-1.5 tw:max-h-[220px] tw:w-full tw:overflow-auto tw:rounded-lg tw:border tw:border-border tw:bg-bg-card tw:shadow-[0_8px_24px_rgba(0,0,0,0.25)]">
            {allowNone && (
              <button
                type="button"
                className="tw:flex tw:w-full tw:cursor-pointer tw:items-center tw:gap-2.5 tw:border-0 tw:border-b tw:border-border tw:bg-transparent tw:px-2.5 tw:py-[9px] tw:text-left tw:text-text-main tw:last:border-b-0 tw:hover:bg-white/6"
                onClick={() => {
                  onChange('');
                  setOpen(false);
                }}
              >
                <div className="tw:inline-flex tw:h-6 tw:w-6 tw:items-center tw:justify-center tw:rounded-md tw:bg-white/10 tw:text-xs tw:font-semibold tw:text-text-muted [image-rendering:pixelated]">
                  -
                </div>
                <span>{noneLabel}</span>
              </button>
            )}
            {servers.map((server) => (
              <button
                type="button"
                key={server.id}
                className={`tw:flex tw:w-full tw:cursor-pointer tw:items-center tw:gap-2.5 tw:border-0 tw:border-b tw:border-border tw:bg-transparent tw:px-2.5 tw:py-[9px] tw:text-left tw:text-text-main tw:last:border-b-0 tw:hover:bg-white/6 ${value === server.id ? 'tw:bg-white/6' : ''}`}
                onClick={() => {
                  onChange(server.id);
                  setOpen(false);
                }}
              >
                <ServerIconImage server={server} />
                <div className="tw:flex tw:min-w-0 tw:flex-col tw:[&_span]:overflow-hidden tw:[&_span]:text-ellipsis tw:[&_span]:whitespace-nowrap tw:[&_small]:text-xs tw:[&_small]:text-text-muted">
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
