import { Globe, Loader2 } from 'lucide-react';

import React, { useCallback, useEffect, useState } from 'react';

import { api } from '../../services/api';
import { Button } from '../ui/Button';
import { CopyButton } from '../ui/CopyButton';
import { Modal } from '../ui/Modal';

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
    const syncTimer = window.setTimeout(() => {
      if (isOpen && serverId) {
        void checkLinkStatus();
      } else {
        setToken(null);
        setError('');
      }
    }, 0);

    return () => window.clearTimeout(syncTimer);
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
      <div className="tw:max-w-[480px]">
        <div>
          <div className="tw:mt-2.5 tw:mb-4 tw:flex tw:items-start tw:justify-between tw:gap-4">
            <div>
              <div className="tw:font-semibold tw:text-base tw:mb-0.5">
                Public Access
              </div>
              <div className="tw:text-xs tw:leading-[1.4] tw:text-gray-500">
                Allow anyone with the link to view status
                <br />
                and start/stop this server.
              </div>
            </div>

            <div className="tw:flex tw:items-center tw:gap-2 tw:shrink-0">
              {loading && (
                <Loader2
                  className="tw:animate-spin tw:text-blue-500"
                  size={16}
                />
              )}
              {token ? (
                <Button
                  variant="danger"
                  onClick={handleDeactivate}
                  disabled={loading}
                  className="tw:h-8 tw:w-20 tw:justify-center tw:text-[0.85rem]"
                >
                  Disable
                </Button>
              ) : (
                <Button
                  variant="secondary"
                  onClick={handleActivate}
                  disabled={loading}
                  className="tw:h-8 tw:w-20 tw:justify-center tw:border-0 tw:bg-blue-500 tw:text-[0.85rem] tw:text-white"
                >
                  Enable
                </Button>
              )}
            </div>
          </div>

          {error && (
            <div className="tw:bg-red-500/10 tw:text-red-500 tw:p-2 tw:rounded tw:mb-3 tw:text-xs tw:flex tw:items-center tw:gap-2">
              {error}
            </div>
          )}

          <div className="tw:min-h-[130px]">
            {token ? (
              <div className="tw:mt-2 tw:rounded-lg tw:border tw:border-border tw:bg-black/20 tw:px-[14px] tw:py-3">
                <div className="tw:text-[10px] tw:text-gray-500 tw:uppercase tw:font-bold tw:mb-1.5 tw:tracking-wider">
                  Public Link
                </div>
                <div className="tw:flex tw:gap-2">
                  <input
                    type="text"
                    aria-label="Public share link"
                    readOnly
                    value={`${publicBaseUrl}/public/${token}`}
                    className="tw:box-border tw:h-8 tw:flex-1 tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-2.5 tw:py-1.5 tw:font-mono tw:text-[0.85rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400"
                    onClick={(e) => e.currentTarget.select()}
                  />
                  <CopyButton
                    text={`${publicBaseUrl}/public/${token}`}
                    variant="secondary"
                    title="Copy to clipboard"
                    className="tw:inline-flex tw:h-[26px] tw:w-[26px] tw:items-center tw:justify-center tw:overflow-visible tw:rounded-md tw:border tw:border-white/4 tw:bg-white/2 tw:p-1 tw:text-text-muted tw:transition-none tw:hover:border-white/8 tw:hover:bg-white/6 tw:hover:text-white"
                  />
                </div>
                <div className="tw:mt-2.5 tw:flex tw:gap-2 tw:text-[11px] tw:text-blue-400 tw:items-start tw:leading-tight">
                  <span className="tw:text-base tw:leading-none">ℹ</span>
                  <div className="tw:mt-px tw:opacity-80">
                    This reusable link persists until you click Disable.
                  </div>
                </div>
              </div>
            ) : (
              <div className="tw:mt-2 tw:flex tw:h-[115px] tw:flex-col tw:items-center tw:justify-center tw:rounded-lg tw:border tw:border-dashed tw:border-border tw:p-5 tw:text-center tw:opacity-40">
                <Globe size={32} className="tw:mx-auto tw:mb-2 tw:opacity-50" />
                <p className="tw:m-0 tw:text-[0.9rem]">
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
