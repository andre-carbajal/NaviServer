import React, { useEffect, useState } from 'react';

import { api } from '../services/api.ts';
import type { Server } from '../types';
import { Button } from './ui/Button';

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
      setMode('existing');
      setSelectedServer('');
      setNewServerName('');
      setNewServerRam(2048);
      setNewServerLoader('vanilla');
      setNewServerVersion('1.20.1');
      setIsSubmitting(false);

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

  const handleSubmit = async (e: React.FormEvent) => {
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

  if (!isOpen) return null;

  const stoppedServers = servers.filter((s) => s.status === 'STOPPED');

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <div className="modal-header">
          <h2 className="modal-title">Restore Backup: {backupName}</h2>
        </div>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Restore To</label>
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
              <label>Select Server (Must be STOPPED)</label>
              <select
                className="form-select"
                value={selectedServer}
                onChange={(e) => setSelectedServer(e.target.value)}
                required
              >
                <option value="" disabled>
                  Select a server
                </option>
                {stoppedServers.map((server) => (
                  <option key={server.id} value={server.id}>
                    {server.name} ({server.id})
                  </option>
                ))}
              </select>
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
                <label>New Server Name</label>
                <input
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
                  <label>Loader</label>
                  <select
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
                  <label>Version</label>
                  <select
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
              <div className="form-group">
                <label>RAM (MB)</label>
                <input
                  type="number"
                  className="form-input"
                  value={newServerRam}
                  onChange={(e) => setNewServerRam(Number(e.target.value))}
                  min="1024"
                  step="512"
                />
              </div>
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
      </div>
    </div>
  );
};

export default RestoreBackupModal;
