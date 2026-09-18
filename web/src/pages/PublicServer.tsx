import axios from 'axios';
import {
  AlertCircle,
  Check,
  Loader2,
  Power,
  Square,
  Wifi,
  WifiOff,
} from 'lucide-react';
import { useParams } from 'react-router-dom';

import React, { useCallback, useEffect, useState } from 'react';

import { api } from '../services/api';

interface PublicServerInfo {
  name: string;
  version: string;
  loader: string;
  status: string;
  id: string;
  onlinePlayers: number;
  maxPlayers: number;
}

const PublicServer: React.FC = () => {
  const { token } = useParams<{ token: string }>();
  const [info, setInfo] = useState<PublicServerInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const [refreshKey, setRefreshKey] = useState(0);

  const fetchInfo = useCallback(async () => {
    if (!token) return;
    try {
      const res = await api.getPublicServerInfo(token);
      setInfo(res.data as unknown as PublicServerInfo);
      setError('');
    } catch (err) {
      if (axios.isAxiosError(err)) {
        setError(err.response?.data || 'Failed to load server info');
      } else {
        setError('An unexpected error occurred');
      }
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    const initialFetch = window.setTimeout(() => {
      void fetchInfo();
    }, 0);
    const interval = setInterval(fetchInfo, 5000);
    return () => {
      window.clearTimeout(initialFetch);
      clearInterval(interval);
    };
  }, [fetchInfo, refreshKey]);

  const handleAction = async (action: 'start' | 'stop') => {
    if (!token) return;
    setActionLoading(true);
    setMessage('');
    try {
      await api.accessPublicLink(token, action);
      setMessage(`Server ${action} command sent!`);
      setTimeout(() => setRefreshKey((prev) => prev + 1), 1000);
    } catch (err) {
      if (axios.isAxiosError(err)) {
        setError(err.response?.data || `Failed to ${action} server`);
      } else {
        setError(`Failed to ${action} server`);
      }
    } finally {
      setActionLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="tw:relative tw:z-[9999] tw:box-border tw:flex tw:min-h-screen tw:w-full tw:items-center tw:justify-center tw:bg-bg-dark tw:p-5">
        <div className="tw:flex tw:items-center tw:justify-center tw:gap-2 tw:text-white">
          <Loader2 className="tw:animate-spin" size={32} /> Loading...
        </div>
      </div>
    );
  }

  if (error && !info) {
    return (
      <div className="tw:relative tw:z-[9999] tw:box-border tw:flex tw:min-h-screen tw:w-full tw:items-center tw:justify-center tw:bg-bg-dark tw:p-5">
        <div className="tw:w-[90%] tw:max-w-[400px] tw:rounded-3xl tw:border tw:border-white/8 tw:bg-[rgba(30,30,35,0.6)] tw:p-8 tw:text-center tw:text-red-500 tw:shadow-[0_25px_50px_-12px_rgba(0,0,0,0.5)] tw:backdrop-blur-2xl tw:max-[480px]:rounded-xl tw:max-[480px]:p-6">
          <AlertCircle size={48} className="tw:mx-auto tw:mb-4" />
          <p>{error}</p>
        </div>
      </div>
    );
  }

  if (!info) return null;
  const isStopping = info.status === 'STOPPING';

  return (
    <div className="tw:relative tw:z-[9999] tw:box-border tw:flex tw:min-h-screen tw:w-full tw:items-center tw:justify-center tw:bg-bg-dark tw:p-5">
      <div className="tw:w-[90%] tw:max-w-[400px] tw:rounded-3xl tw:border tw:border-white/8 tw:bg-[rgba(30,30,35,0.6)] tw:p-8 tw:text-center tw:shadow-[0_25px_50px_-12px_rgba(0,0,0,0.5)] tw:backdrop-blur-2xl tw:max-[480px]:rounded-xl tw:max-[480px]:p-6">
        <div className="tw:mb-6 tw:flex tw:flex-col tw:items-center">
          <div className="tw:mb-4 tw:h-24 tw:w-24 tw:overflow-hidden tw:rounded-2xl tw:border tw:border-white/10 tw:bg-white/5">
            <img
              src={`${api.getServerIconUrl(info.id)}`}
              alt="Server Icon"
              className="tw:h-full tw:w-full tw:object-contain [image-rendering:pixelated]"
              onError={(e) => {
                e.currentTarget.style.display = 'none';
                e.currentTarget.nextElementSibling?.setAttribute(
                  'style',
                  'display: flex; width: 100%; height: 100%; align-items: center; justify-content: center; font-size: 48px; color: #555;',
                );
              }}
            />
            <div className="tw:hidden">{info.name.charAt(0).toUpperCase()}</div>
          </div>

          <h2 className="tw:mt-0 tw:mb-1 tw:text-2xl tw:font-bold">
            {info.name}
          </h2>

          <div className="tw:mb-4 tw:flex tw:items-center tw:gap-2 tw:text-sm tw:text-gray-400">
            <span className="tw:font-semibold tw:text-white">
              {info.loader}
            </span>
            <span>•</span>
            <span>{info.version}</span>
          </div>

          <div
            className={`tw:mb-4 tw:inline-flex tw:items-center tw:gap-2 tw:rounded-xl tw:px-4 tw:py-1.5 tw:text-[0.9rem] tw:font-bold tw:text-white tw:uppercase ${info.status === 'RUNNING' ? 'tw:bg-green-600' : info.status === 'STOPPED' ? 'tw:bg-red-500' : info.status === 'CREATING' ? 'tw:bg-blue-500' : info.status === 'STARTING' || info.status === 'STOPPING' ? 'tw:bg-orange-500' : 'tw:bg-white/10'}`}
          >
            {info.status === 'RUNNING' ? (
              <Wifi size={16} />
            ) : (
              <WifiOff size={16} />
            )}
            {info.status}
          </div>

          {info.status === 'RUNNING' && (
            <div className="tw:mb-6 tw:flex tw:items-center tw:gap-2 tw:rounded-full tw:border tw:border-white/10 tw:bg-white/5 tw:px-3 tw:py-1 tw:text-sm tw:text-gray-400">
              <div className="tw:h-2 tw:w-2 tw:rounded-full tw:bg-green-500"></div>
              <span className="tw:font-medium tw:text-white">
                {info.onlinePlayers || 0} / {info.maxPlayers || 0}
              </span>
              <span>Players Online</span>
            </div>
          )}
        </div>

        {message && (
          <div className="tw:my-4 tw:flex tw:items-center tw:justify-center tw:gap-2 tw:rounded-lg tw:bg-green-500/20 tw:p-3 tw:text-green-400">
            <Check size={18} /> {message}
          </div>
        )}

        {error && (
          <div className="tw:my-4 tw:flex tw:items-center tw:justify-center tw:gap-2 tw:rounded-lg tw:bg-red-500/20 tw:p-3 tw:text-red-400">
            <AlertCircle size={18} /> {error}
          </div>
        )}

        <div className="tw:mt-6 tw:flex tw:justify-center tw:gap-3">
          {info.status === 'OFFLINE' || info.status === 'STOPPED' ? (
            <button
              type="button"
              className="tw:flex tw:flex-1 tw:cursor-pointer tw:items-center tw:justify-center tw:gap-2 tw:rounded-lg tw:border tw:border-transparent tw:bg-primary tw:px-4 tw:py-3 tw:font-semibold tw:text-white tw:transition-all tw:duration-200 tw:hover:bg-primary-hover tw:disabled:cursor-not-allowed tw:disabled:opacity-50"
              onClick={() => handleAction('start')}
              disabled={actionLoading}
            >
              {actionLoading ? (
                <Loader2 className="tw:animate-spin" />
              ) : (
                <Power size={20} />
              )}
              Start Server
            </button>
          ) : (
            <button
              type="button"
              className={`tw:flex tw:flex-1 tw:items-center tw:justify-center tw:gap-2 tw:rounded-lg tw:border tw:border-red-500/30 tw:bg-red-500 tw:px-4 tw:py-3 tw:font-semibold tw:text-white tw:transition-all tw:duration-200 tw:hover:bg-red-600 tw:disabled:cursor-not-allowed tw:disabled:opacity-50 ${isStopping || info.status === 'STARTING' ? 'tw:opacity-50' : 'tw:cursor-pointer'}`}
              onClick={() => handleAction('stop')}
              disabled={
                actionLoading || isStopping || info.status === 'STARTING'
              }
            >
              {actionLoading ? (
                <Loader2 className="tw:animate-spin" />
              ) : (
                <Square size={20} />
              )}
              Stop Server
            </button>
          )}
        </div>
      </div>

      <div className="tw:fixed tw:bottom-4 tw:left-0 tw:w-full tw:text-center tw:text-sm tw:text-gray-500">
        Powered by NaviServer
      </div>
    </div>
  );
};

export default PublicServer;
