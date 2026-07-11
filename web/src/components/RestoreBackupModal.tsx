import React, { useEffect, useState } from 'react';

import { api } from '../services/api.ts';
import type { Server } from '../types';
import ServerIconSelect from './ServerIconSelect';
import { Button } from './ui/Button';
import { Modal } from './ui/Modal';

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
  const [includePreviews, setIncludePreviews] = useState(false);
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
        setIncludePreviews(false);
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
        .getLoaderVersions(newServerLoader, {
          includeSnapshots: newServerLoader === 'bedrock' && includePreviews,
        })
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
  }, [includePreviews, newServerLoader]);

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
      data.newServerRam = newServerLoader === 'bedrock' ? 4096 : newServerRam;
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
        <div className="form-group">
          <span className="form-label">Restore To</span>
          <div style={{ display: 'flex', gap: '10px', marginBottom: '10px' }}>
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
          <div className="form-group">
            <ServerIconSelect
              label="Select Server (Must be STOPPED)"
              value={selectedServer}
              onChange={setSelectedServer}
              servers={stoppedServers}
            />
            {stoppedServers.length === 0 && (
              <p
                style={{
                  color: 'var(--danger)',
                  fontSize: '0.8em',
                  marginTop: '5px',
                }}
              >
                No stopped servers available.
              </p>
            )}
          </div>
        ) : (
          <>
            <div className="form-group">
              <label htmlFor="restore-new-server-name">New Server Name</label>
              <input
                id="restore-new-server-name"
                type="text"
                className="form-input"
                value={newServerName}
                onChange={(e) => setNewServerName(e.target.value)}
                required
              />
            </div>
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: '1fr 1fr',
                gap: '15px',
              }}
            >
              <div className="form-group">
                <label htmlFor="restore-new-server-loader">Loader</label>
                <select
                  id="restore-new-server-loader"
                  className="form-select"
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
              <div className="form-group">
                <label htmlFor="restore-new-server-version">Version</label>
                <select
                  id="restore-new-server-version"
                  className="form-select"
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
            {newServerLoader === 'bedrock' && (
              <label className="checkbox-row">
                <input
                  type="checkbox"
                  checked={includePreviews}
                  onChange={(e) => setIncludePreviews(e.target.checked)}
                />{' '}
                Show previews
              </label>
            )}
            {newServerLoader === 'bedrock' ? (
              <p className="form-hint">
                Bedrock Dedicated Server manages memory automatically.
              </p>
            ) : (
              <div className="form-group">
                <label htmlFor="restore-new-server-ram">RAM (MB)</label>
                <input
                  id="restore-new-server-ram"
                  type="number"
                  className="form-input"
                  value={newServerRam}
                  onChange={(e) => setNewServerRam(Number(e.target.value))}
                  min="1024"
                  step="512"
                />
              </div>
            )}
          </>
        )}

        <div className="modal-actions">
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
