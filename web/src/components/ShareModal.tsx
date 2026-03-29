import { Globe, Loader2 } from 'lucide-react';

import React, { useCallback, useEffect, useState } from 'react';

import { api } from '../services/api';
import { Button } from './ui/Button';
import { CopyButton } from './ui/CopyButton';
import { Modal } from './ui/Modal';

interface ShareModalProps {
  isOpen: boolean;
  onClose: () => void;
  serverId: string;
}

const ShareModal: React.FC<ShareModalProps> = ({
  isOpen,
  onClose,
  serverId,
}) => {
  const [loading, setLoading] = useState(false);
  const [token, setToken] = useState<string | null>(null);
  const [error, setError] = useState('');
  const [publicBaseUrl, setPublicBaseUrl] = useState(window.location.origin);

  const checkLinkStatus = useCallback(async () => {
    setLoading(true);
    try {
      const [linkRes, publicIPRes] = await Promise.all([
        api.getPublicLink(serverId).catch(() => ({ data: null })),
        api.getPublicIP(),
      ]);

      if (linkRes.data) {
        setToken(linkRes.data.token);
      } else {
        setToken(null);
      }

      const publicIP = publicIPRes.data?.public_ip ?? 'localhost';
      const port = import.meta.env.VITE_API_PORT || 23008;
      const protocol = window.location.protocol;
      setPublicBaseUrl(`${protocol}//${publicIP}:${port}`);
    } catch (err) {
      console.error(err);
      setError('Failed to check sharing status');
    } finally {
      setLoading(false);
    }
  }, [serverId]);

  useEffect(() => {
    if (isOpen && serverId) {
      checkLinkStatus();
    } else {
      setToken(null);
      setError('');
    }
  }, [isOpen, serverId, checkLinkStatus]);

  const handleDeactivate = async () => {
    if (!token) return;
    setLoading(true);
    try {
      await api.deletePublicLink(token);
      setToken(null);
    } catch (err) {
      console.error(err);
      setError('Failed to deactivate link');
    } finally {
      setLoading(false);
    }
  };

  const handleActivate = async () => {
    setLoading(true);
    try {
      const res = await api.createPublicLink(serverId);
      setToken(res.data.token);
      setError('');
    } catch {
      setError('Failed to activate link');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Share Server">
      <div style={{ maxWidth: '480px' }}>
        <div>
          <div
            className="flex justify-between items-start mb-4"
            style={{ gap: '16px', marginTop: '10px' }}
          >
            <div>
              <div className="font-semibold text-base mb-0.5">
                Public Access
              </div>
              <div
                className="text-xs text-gray-500"
                style={{ lineHeight: '1.4' }}
              >
                Allow anyone with the link to view status
                <br />
                and start/stop this server.
              </div>
            </div>

            <div className="flex items-center gap-2 shrink-0">
              {loading && (
                <Loader2 className="animate-spin text-blue-500" size={16} />
              )}
              {token ? (
                <Button
                  variant="danger"
                  onClick={handleDeactivate}
                  disabled={loading}
                  style={{
                    width: '80px',
                    height: '32px',
                    fontSize: '0.85rem',
                    justifyContent: 'center',
                  }}
                >
                  Disable
                </Button>
              ) : (
                <Button
                  variant="secondary"
                  onClick={handleActivate}
                  disabled={loading}
                  style={{
                    background: '#3b82f6',
                    color: 'white',
                    border: 'none',
                    width: '80px',
                    height: '32px',
                    fontSize: '0.85rem',
                    justifyContent: 'center',
                  }}
                >
                  Enable
                </Button>
              )}
            </div>
          </div>

          {error && (
            <div
              className="bg-red-500/10 text-red-500 p-2 rounded mb-3 text-xs flex items-center gap-2"
              style={{ background: 'rgba(239, 68, 68, 0.1)', color: '#ef4444' }}
            >
              {error}
            </div>
          )}

          <div style={{ minHeight: '130px' }}>
            {token ? (
              <div
                style={{
                  backgroundColor: 'rgba(0,0,0,0.2)',
                  padding: '12px 14px',
                  borderRadius: '8px',
                  border: '1px solid var(--border-color)',
                  marginTop: '8px',
                }}
              >
                <div className="text-[10px] text-gray-500 uppercase font-bold mb-1.5 tracking-wider">
                  Public Link
                </div>
                <div className="flex gap-2">
                  <input
                    type="text"
                    readOnly
                    value={`${publicBaseUrl}/public/${token}`}
                    className="form-input"
                    style={{
                      flex: 1,
                      fontFamily: 'monospace',
                      fontSize: '0.85rem',
                      padding: '6px 10px',
                      height: '32px',
                    }}
                    onClick={(e) => e.currentTarget.select()}
                  />
                  <CopyButton
                    text={`${publicBaseUrl}/public/${token}`}
                    variant="secondary"
                    title="Copy to clipboard"
                    className="address-copy-btn"
                  />
                </div>
                <div className="mt-2.5 flex gap-2 text-[11px] text-blue-400 items-start leading-tight">
                  <span style={{ fontSize: '1rem', lineHeight: 1 }}>ℹ</span>
                  <div style={{ opacity: 0.8, marginTop: '1px' }}>
                    This reusable link persists until you click Disable.
                  </div>
                </div>
              </div>
            ) : (
              <div
                style={{
                  padding: '20px',
                  borderRadius: '8px',
                  border: '1px dashed var(--border-color)',
                  textAlign: 'center',
                  opacity: 0.4,
                  marginTop: '8px',
                  height: '115px',
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <Globe size={32} className="mx-auto mb-2 opacity-50" />
                <p style={{ margin: 0, fontSize: '0.9rem' }}>
                  Public access is disabled.
                </p>
              </div>
            )}
          </div>
        </div>
      </div>
    </Modal>
  );
};

export default ShareModal;
