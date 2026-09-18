import React, { useEffect, useState } from 'react';

import { api } from '../../services/api';
import type { Server } from '../../types';
import { Button } from '../ui/Button';
import { Modal } from '../ui/Modal';
import ServerIconSelect from './ServerIconSelect';

export interface RestoreData {
  targetServerId?: string;
  newServerName?: string;
  newServerRam?: number;
  newServerLoader?: string;
  newServerVersion?: string;
}

interface RestoreBackupModalProps {
  isOpen: boolean;
  onClose: () => void;
  onRestore: (backupName: string, data: RestoreData) => Promise<void>;
  backupName: string;
  servers: Server[];
}

const RestoreBackupModal: React.FC<RestoreBackupModalProps> = ({
  isOpen,
  onClose,
  onRestore,
  backupName,
  servers,
}) => {
  const [mode, setMode] = useState<'existing' | 'new'>('existing');
  const [selectedServer, setSelectedServer] = useState('');
  const [newServerName, setNewServerName] = useState('');
  const [newServerRam, setNewServerRam] = useState(2048);
  const [newServerLoader, setNewServerLoader] = useState('vanilla');
  const [newServerVersion, setNewServerVersion] = useState('1.20.1');
  const [loaders, setLoaders] = useState<string[]>([]);
  const [versions, setVersions] = useState<string[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (isOpen) {
      const resetTimer = window.setTimeout(() => {
        setMode('existing');
        setSelectedServer('');
        setNewServerName('');
        setNewServerRam(2048);
        setNewServerLoader('vanilla');
        setNewServerVersion('1.20.1');
        setIsSubmitting(false);
      }, 0);

      api
        .getLoaders()
        .then((response) => {
          setLoaders(response.data);
          if (response.data.length > 0) {
            setNewServerLoader(response.data[0]);
          }
        })
        .catch((error) => {
          console.error('Failed to fetch loaders', error);
        });

      return () => window.clearTimeout(resetTimer);
    }
  }, [isOpen]);

  useEffect(() => {
    if (newServerLoader) {
      api
        .getLoaderVersions(newServerLoader)
        .then((response) => {
          setVersions(response.data);
          if (response.data.length > 0) {
            setNewServerVersion(response.data[0]);
          }
        })
        .catch((error) => {
          console.error(
            `Failed to fetch versions for ${newServerLoader}`,
            error,
          );
        });
    }
  }, [newServerLoader]);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsSubmitting(true);

    const data: RestoreData = {};
    if (mode === 'existing') {
      if (!selectedServer) return;
      data.targetServerId = selectedServer;
    } else {
      if (!newServerName) return;
      data.newServerName = newServerName;
      data.newServerRam = newServerRam;
      data.newServerLoader = newServerLoader;
      data.newServerVersion = newServerVersion;
    }

    try {
      await onRestore(backupName, data);
      onClose();
    } catch (error) {
      console.error('Failed to restore backup:', error);
    } finally {
      setIsSubmitting(false);
    }
  };

  const stoppedServers = servers.filter((s) => s.status === 'STOPPED');

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`Restore Backup: ${backupName}`}
    >
      <form onSubmit={handleSubmit}>
        <div className="tw:mb-[15px] tw:max-[769px]:mb-3">
          <span className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted">
            Restore To
          </span>
          <div className="tw:mb-2.5 tw:flex tw:gap-2.5">
            <label>
              <input
                type="radio"
                name="mode"
                value="existing"
                checked={mode === 'existing'}
                onChange={() => setMode('existing')}
              />{' '}
              Existing Server
            </label>
            <label>
              <input
                type="radio"
                name="mode"
                value="new"
                checked={mode === 'new'}
                onChange={() => setMode('new')}
              />{' '}
              New Server
            </label>
          </div>
        </div>

        {mode === 'existing' ? (
          <div className="tw:mb-[15px] tw:max-[769px]:mb-3">
            <ServerIconSelect
              label="Select Server (Must be STOPPED)"
              value={selectedServer}
              onChange={setSelectedServer}
              servers={stoppedServers}
            />
            {stoppedServers.length === 0 && (
              <p className="tw:mt-[5px] tw:text-[0.8em] tw:text-danger">
                No stopped servers available.
              </p>
            )}
          </div>
        ) : (
          <>
            <div className="tw:mb-[15px] tw:max-[769px]:mb-3">
              <label
                htmlFor="restore-new-server-name"
                className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted tw:max-[769px]:mb-1.5 tw:max-[769px]:text-[0.85rem]"
              >
                New Server Name
              </label>
              <input
                id="restore-new-server-name"
                type="text"
                className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
                value={newServerName}
                onChange={(e) => setNewServerName(e.target.value)}
                required
              />
            </div>
            <div className="tw:grid tw:grid-cols-2 tw:gap-[15px]">
              <div className="tw:mb-[15px] tw:max-[769px]:mb-3">
                <label
                  htmlFor="restore-new-server-loader"
                  className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted tw:max-[769px]:mb-1.5 tw:max-[769px]:text-[0.85rem]"
                >
                  Loader
                </label>
                <select
                  id="restore-new-server-loader"
                  className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
                  value={newServerLoader}
                  onChange={(e) => setNewServerLoader(e.target.value)}
                >
                  {loaders.map((l) => (
                    <option key={l} value={l}>
                      {l.charAt(0).toUpperCase() + l.slice(1)}
                    </option>
                  ))}
                </select>
              </div>
              <div className="tw:mb-[15px] tw:max-[769px]:mb-3">
                <label
                  htmlFor="restore-new-server-version"
                  className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted tw:max-[769px]:mb-1.5 tw:max-[769px]:text-[0.85rem]"
                >
                  Version
                </label>
                <select
                  id="restore-new-server-version"
                  className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
                  value={newServerVersion}
                  onChange={(e) => setNewServerVersion(e.target.value)}
                >
                  {versions.map((v) => (
                    <option key={v} value={v}>
                      {v}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            <div className="tw:mb-[15px] tw:max-[769px]:mb-3">
              <label
                htmlFor="restore-new-server-ram"
                className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted tw:max-[769px]:mb-1.5 tw:max-[769px]:text-[0.85rem]"
              >
                RAM (MB)
              </label>
              <input
                id="restore-new-server-ram"
                type="number"
                className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
                value={newServerRam}
                onChange={(e) => setNewServerRam(Number(e.target.value))}
                min="1024"
                step="512"
              />
            </div>
          </>
        )}

        <div className="tw:mt-[25px] tw:flex tw:justify-end tw:gap-2.5 tw:max-[769px]:flex-col tw:max-[769px]:gap-2 tw:max-[769px]:[&_button]:w-full">
          <Button
            type="button"
            variant="secondary"
            onClick={onClose}
            disabled={isSubmitting}
          >
            Cancel
          </Button>
          <Button
            type="submit"
            disabled={
              isSubmitting ||
              (mode === 'existing' && !selectedServer) ||
              (mode === 'new' && !newServerName)
            }
          >
            {isSubmitting ? 'Restoring...' : 'Restore'}
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default RestoreBackupModal;
