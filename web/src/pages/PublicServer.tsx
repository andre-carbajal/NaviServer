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

import '../App.css';
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
    fetchInfo();
    const interval = setInterval(fetchInfo, 5000);
    return () => clearInterval(interval);
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
      <div className="login-container">
        <div className="flex justify-center items-center text-white gap-2">
          <Loader2 className="animate-spin" size={32} /> Loading...
        </div>
      </div>
    );
  }

  if (error && !info) {
    return (
      <div className="login-container">
        <div className="login-card text-center text-red-500">
          <AlertCircle size={48} className="mx-auto mb-4" />
          <p>{error}</p>
        </div>
      </div>
    );
  }

  if (!info) return null;
  const isStopping = info.status === 'STOPPING';

  return (
    <div className="login-container">
      <div
        className="login-card"
        style={{ textAlign: 'center', maxWidth: '400px', width: '90%' }}
      >
        <div className="mb-6 flex flex-col items-center">
          <div
            style={{
              width: '96px',
              height: '96px',
              borderRadius: '16px',
              overflow: 'hidden',
              marginBottom: '16px',
              background: 'rgba(255,255,255,0.05)',
              border: '1px solid rgba(255,255,255,0.1)',
            }}
          >
            <img
              src={`${api.getServerIconUrl(info.id)}`}
              alt="Server Icon"
              style={{
                width: '100%',
                height: '100%',
                objectFit: 'contain',
                imageRendering: 'pixelated',
              }}
              onError={(e) => {
                e.currentTarget.style.display = 'none';
                e.currentTarget.nextElementSibling?.setAttribute(
                  'style',
                  'display: flex; width: 100%; height: 100%; align-items: center; justify-content: center; font-size: 48px; color: #555;',
                );
              }}
            />
            <div style={{ display: 'none' }}>
              {info.name.charAt(0).toUpperCase()}
            </div>
          </div>

          <h2 className="text-2xl font-bold mb-1">{info.name}</h2>

          <div className="flex items-center gap-2 text-sm text-gray-400 mb-4">
            <span className="font-semibold text-white">{info.loader}</span>
            <span>•</span>
            <span>{info.version}</span>
          </div>

          <div
            className={`status-badge status-${info.status.toLowerCase()} mb-4`}
            style={{ fontSize: '0.9rem', padding: '6px 16px' }}
          >
            {info.status === 'RUNNING' ? (
              <Wifi size={16} />
            ) : (
              <WifiOff size={16} />
            )}
            {info.status}
          </div>

          {info.status === 'RUNNING' && (
            <div
              className="text-sm text-gray-400 mb-6 flex items-center gap-2"
              style={{
                background: 'rgba(255,255,255,0.05)',
                padding: '4px 12px',
                borderRadius: '999px',
                border: '1px solid rgba(255,255,255,0.1)',
              }}
            >
              <div
                style={{
                  width: '8px',
                  height: '8px',
                  borderRadius: '50%',
                  backgroundColor: '#22c55e',
                }}
              ></div>
              <span className="text-white font-medium">
                {info.onlinePlayers || 0} / {info.maxPlayers || 0}
              </span>
              <span>Players Online</span>
            </div>
          )}
        </div>

        {message && (
          <div
            className="bg-green-500/20 text-green-400 p-3 rounded-lg"
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '8px',
              marginTop: '16px',
              marginBottom: '16px',
            }}
          >
            <Check size={18} /> {message}
          </div>
        )}

        {error && (
          <div
            className="bg-red-500/20 text-red-400 p-3 rounded-lg"
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '8px',
              marginTop: '16px',
              marginBottom: '16px',
            }}
          >
            <AlertCircle size={18} /> {error}
          </div>
        )}

        <div
          className="flex gap-3 justify-center"
          style={{ marginTop: '24px' }}
        >
          {info.status === 'OFFLINE' || info.status === 'STOPPED' ? (
            <button
              className="btn btn-primary"
              onClick={() => handleAction('start')}
              disabled={actionLoading}
              style={{ flex: 1, padding: '12px' }}
            >
              {actionLoading ? (
                <Loader2 className="animate-spin" />
              ) : (
                <Power size={20} />
              )}
              Start Server
            </button>
          ) : (
            <button
              className="btn btn-danger"
              onClick={() => handleAction('stop')}
              disabled={
                actionLoading || isStopping || info.status === 'STARTING'
              }
              style={{
                flex: 1,
                padding: '12px',
                backgroundColor: '#ef4444',
                color: 'white',
                opacity: isStopping || info.status === 'STARTING' ? 0.5 : 1,
              }}
            >
              {actionLoading ? (
                <Loader2 className="animate-spin" />
              ) : (
                <Square size={20} />
              )}
              Stop Server
            </button>
          )}
        </div>
      </div>

      <div
        style={{
          position: 'fixed',
          bottom: '1rem',
          left: 0,
          width: '100%',
          textAlign: 'center',
          color: '#6b7280',
          fontSize: '0.875rem',
        }}
      >
        Powered by Naviger
      </div>
    </div>
  );
};

export default PublicServer;
