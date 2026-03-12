import {
  Download,
  Loader2,
  Plus,
  RotateCcw,
  Trash2,
  Upload,
  X,
} from 'lucide-react';
import { useParams } from 'react-router-dom';
import { v4 as uuidv4 } from 'uuid';

import React, { useCallback, useEffect, useRef, useState } from 'react';

import ConfirmationModal from '../components/ConfirmationModal';
import CreateBackupModal from '../components/CreateBackupModal';
import RestoreBackupModal from '../components/RestoreBackupModal';
import type { RestoreData } from '../components/RestoreBackupModal';
import { Button } from '../components/ui/Button';
import { useAuth } from '../context/AuthContext';
import { useServers } from '../hooks/useServers';
import { WS_HOST, api } from '../services/api';
import type { Backup } from '../types';

interface CreatingBackup extends Backup {
  serverId: string;
}

interface UploadingBackup {
  id: string;
  name: string;
  progress: number;
}

const Backups: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { token } = useAuth();
  const [backups, setBackups] = useState<Backup[]>([]);
  const [creatingBackups, setCreatingBackups] = useState<CreatingBackup[]>([]);
  const [uploadingBackups, setUploadingBackups] = useState<UploadingBackup[]>(
    [],
  );
  const [isDragging, setIsDragging] = useState(false);
  const { servers, refresh: refreshServers } = useServers();
  const [isCreateModalOpen, setCreateModalOpen] = useState(false);
  const [restoreModalOpen, setRestoreModalOpen] = useState(false);
  const [selectedBackup, setSelectedBackup] = useState<string | null>(null);
  const [backupToDelete, setBackupToDelete] = useState<string | null>(null);
  const activeSockets = useRef<Set<string>>(new Set());
  const wsMap = useRef<Map<string, WebSocket>>(new Map());
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleDragOver = (event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.stopPropagation();
    setIsDragging(true);
  };

  const handleDragLeave = (event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.stopPropagation();
    setIsDragging(false);
  };

  const handleDrop = async (event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.stopPropagation();
    setIsDragging(false);
    const files = event.dataTransfer.files;
    if (files && files.length > 0) {
      await uploadFiles(files);
    }
  };

  const fetchBackups = useCallback(() => {
    const promise =
      id && id !== 'all' ? api.listBackups(id) : api.listAllBackups();
    promise
      .then((response) => {
        setBackups(response.data || []);
      })
      .catch((error) => {
        console.error('Failed to fetch backups:', error);
        setBackups([]);
      });
  }, [id]);

  useEffect(() => {
    fetchBackups();
  }, [id, fetchBackups]);

  const removeCreatingBackup = useCallback((requestId: string) => {
    setCreatingBackups((prev) => prev.filter((b) => b.requestId !== requestId));
    const stored = localStorage.getItem('creating_backups');
    if (stored) {
      try {
        const list: CreatingBackup[] = JSON.parse(stored);
        const newList = list.filter((b) => b.requestId !== requestId);
        localStorage.setItem('creating_backups', JSON.stringify(newList));
      } catch {
        // Ignore JSON parse errors
      }
    }

    const ws = wsMap.current.get(requestId);
    if (ws) {
      ws.close();
      wsMap.current.delete(requestId);
    }
    activeSockets.current.delete(requestId);
  }, []);

  const trackProgress = useCallback(
    (requestId: string) => {
      if (activeSockets.current.has(requestId) || !token) return;

      activeSockets.current.add(requestId);
      const ws = new WebSocket(
        `ws://${WS_HOST}/ws/progress/${requestId}?token=${token}`,
      );
      wsMap.current.set(requestId, ws);

      ws.onmessage = (event) => {
        try {
          const msgData = JSON.parse(event.data);

          if (msgData.progress >= 100 || msgData.progress === -1) {
            ws.close();
            removeCreatingBackup(requestId);
            fetchBackups();
          } else {
            setCreatingBackups((prev) =>
              prev.map((b) => {
                if (b.requestId === requestId) {
                  return {
                    ...b,
                    progress: msgData.progress,
                    progressMessage: msgData.message,
                  };
                }
                return b;
              }),
            );
          }
        } catch (e) {
          console.error('Error parsing progress message', e);
        }
      };

      ws.onclose = () => {
        activeSockets.current.delete(requestId);
        wsMap.current.delete(requestId);
      };
    },
    [fetchBackups, removeCreatingBackup, token],
  );

  useEffect(() => {
    const stored = localStorage.getItem('creating_backups');
    if (stored) {
      try {
        const list: CreatingBackup[] = JSON.parse(stored);
        setCreatingBackups(list);
        list.forEach((b) => {
          if (b.requestId) trackProgress(b.requestId);
        });
      } catch (e) {
        console.error(e);
      }
    }
  }, [trackProgress, token]);

  const handleCreateBackup = async (serverId: string, name: string) => {
    const requestId = uuidv4();
    const selectedServer = servers.find((s) => s.id === serverId);
    const serverName = selectedServer ? selectedServer.name : 'Unknown';

    const tempBackup: CreatingBackup = {
      name: name || `Backup for ${serverName}`,
      size: 0,
      status: 'CREATING',
      progress: 0,
      requestId: requestId,
      serverId: serverId,
      progressMessage: 'Initializing...',
    };

    setCreatingBackups((prev) => [...prev, tempBackup]);

    const stored = localStorage.getItem('creating_backups');
    const list: CreatingBackup[] = stored ? JSON.parse(stored) : [];
    list.push(tempBackup);
    localStorage.setItem('creating_backups', JSON.stringify(list));

    trackProgress(requestId);

    try {
      await api.createBackup(serverId, name, requestId);
    } catch (error) {
      console.error('Failed to initiate backup creation:', error);
      removeCreatingBackup(requestId);
      alert('Failed to start backup creation.');
    }
  };

  const handleCancelBackup = (requestId: string) => {
    api
      .cancelBackupCreation(requestId)
      .catch((e) => console.error('Error cancelling backup in backend:', e));
    removeCreatingBackup(requestId);
  };

  const handleDelete = (backupName: string) => {
    setBackupToDelete(backupName);
  };

  const confirmDelete = async () => {
    if (backupToDelete) {
      try {
        await api.deleteBackup(backupToDelete);
        fetchBackups();
      } catch (error) {
        console.error('Failed to delete backup:', error);
      }
      setBackupToDelete(null);
    }
  };

  const handleRestoreClick = (backupName: string) => {
    setSelectedBackup(backupName);
    setRestoreModalOpen(true);
  };

  const handleRestore = async (backupName: string, data: RestoreData) => {
    await api.restoreBackup(backupName, data);
    alert('Backup restored successfully!');
    refreshServers();
  };

  const handleUploadClick = () => {
    fileInputRef.current?.click();
  };

  const uploadFiles = async (files: FileList | File[]) => {
    if (!files || files.length === 0) return;

    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      const ext = file.name.split('.').pop()?.toLowerCase();

      if (ext !== 'zip' && ext !== 'rar') {
        alert(
          `File ${file.name} is not a valid backup file (.zip or .rar only).`,
        );
        continue;
      }

      const uploadId = uuidv4();
      const newUploadingBackup: UploadingBackup = {
        id: uploadId,
        name: file.name,
        progress: 0,
      };
      setUploadingBackups((prev) => [...prev, newUploadingBackup]);

      try {
        await api.uploadBackup(file, (progressEvent) => {
          const progress = Math.round(
            (progressEvent.loaded * 100) / (progressEvent.total ?? 1),
          );
          setUploadingBackups((prev) =>
            prev.map((b) =>
              b.id === uploadId ? { ...b, progress: progress } : b,
            ),
          );
        });
      } catch (error) {
        console.error(`Failed to upload backup ${file.name}:`, error);
        alert(`Failed to upload backup ${file.name}.`);
        try {
          await api.deleteBackup(file.name);
        } catch (e) {
          console.warn('Failed to cleanup failed backup upload:', e);
        }
      } finally {
        setUploadingBackups((prev) => prev.filter((b) => b.id !== uploadId));
      }
    }
    fetchBackups();
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleFileChange = async (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const files = event.target.files;
    if (files && files.length > 0) {
      await uploadFiles(files);
    }
  };

  const isGlobalView = !id || id === 'all';

  const visibleCreatingBackups = creatingBackups.filter(
    (b) => isGlobalView || b.serverId === id,
  );

  return (
    <div
      className={`backups-page ${isDragging ? 'dragging' : ''}`}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
      style={{
        position: 'relative',
        borderColor: isDragging ? '#646cff' : 'transparent',
        boxShadow: isDragging ? '0 0 0 2px rgba(100, 108, 255, 0.2)' : 'none',
      }}
    >
      <div className="modal-header">
        <h1>Backups</h1>
        <div className="backup-actions-header">
          <Button onClick={handleUploadClick} variant="secondary">
            <Upload size={20} /> <span className="btn-text">Upload Backup</span>
          </Button>
          <Button onClick={() => setCreateModalOpen(true)}>
            <Plus size={20} /> <span className="btn-text">Create Backup</span>
          </Button>
        </div>
      </div>
      {isDragging && (
        <div
          style={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: 'rgba(100, 108, 255, 0.1)',
            backdropFilter: 'blur(2px)',
            zIndex: 50,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            pointerEvents: 'none',
            borderRadius: '12px',
          }}
        >
          <div
            style={{
              color: 'white',
              fontWeight: 'bold',
              fontSize: '1.2rem',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: '10px',
            }}
          >
            <Upload size={48} />
            <span>Drop backups to upload (.zip, .rar)</span>
          </div>
        </div>
      )}
      <div className="card">
        <input
          type="file"
          ref={fileInputRef}
          onChange={handleFileChange}
          style={{ display: 'none' }}
          accept=".zip,.rar"
          multiple
        />
        <table className="data-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Size</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {uploadingBackups.map((upload) => (
              <tr key={upload.id}>
                <td>
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '8px',
                    }}
                  >
                    <Loader2 className="spin" size={16} />
                    <div>
                      <div>{upload.name}</div>
                      <div
                        style={{
                          fontSize: '0.8em',
                          color: 'var(--text-muted)',
                        }}
                      >
                        Uploading...
                      </div>
                    </div>
                  </div>
                  <div
                    className="progress-bar-container"
                    style={{ marginTop: '4px', height: '4px' }}
                  >
                    <div
                      className="progress-bar-fill"
                      style={{ width: `${upload.progress}%` }}
                    />
                  </div>
                </td>
                <td>-</td>
                <td>-</td>
              </tr>
            ))}
            {visibleCreatingBackups.map((backup) => (
              <tr key={backup.requestId}>
                <td>
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '8px',
                    }}
                  >
                    <Loader2 className="spin" size={16} />
                    <div>
                      <div>{backup.name}</div>
                      <div
                        style={{
                          fontSize: '0.8em',
                          color: 'var(--text-muted)',
                        }}
                      >
                        {backup.progressMessage}
                      </div>
                    </div>
                  </div>
                  {backup.progress !== undefined && (
                    <div
                      className="progress-bar-container"
                      style={{ marginTop: '4px', height: '4px' }}
                    >
                      <div
                        className="progress-bar-fill"
                        style={{ width: `${backup.progress}%` }}
                      />
                    </div>
                  )}
                </td>
                <td>-</td>
                <td>
                  <div style={{ display: 'flex', gap: '5px' }}>
                    <Button
                      variant="secondary"
                      onClick={() => handleCancelBackup(backup.requestId!)}
                      title="Dismiss / Cancel"
                    >
                      <X size={16} /> Cancel
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
            {backups.map((backup) => (
              <tr key={backup.name}>
                <td>{backup.name}</td>
                <td>{(backup.size / 1024 / 1024).toFixed(2)} MB</td>
                <td>
                  <div
                    className="actions-group"
                    style={{ border: 'none', padding: 0, margin: 0 }}
                  >
                    <a
                      className="icon-action"
                      title="Download"
                      href={api.getBackupDownloadUrl(backup.name)}
                      target="_blank"
                      rel="noreferrer"
                    >
                      <Download size={18} />
                    </a>
                    <button
                      className="icon-action"
                      title="Restore"
                      onClick={() => handleRestoreClick(backup.name)}
                    >
                      <RotateCcw size={18} />
                    </button>
                    <button
                      className="icon-action danger"
                      title="Delete"
                      onClick={() => handleDelete(backup.name)}
                    >
                      <Trash2 size={18} />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {backups.length === 0 &&
              visibleCreatingBackups.length === 0 &&
              uploadingBackups.length === 0 && (
                <tr>
                  <td
                    colSpan={3}
                    style={{
                      textAlign: 'center',
                      padding: '20px',
                      color: 'var(--text-muted)',
                    }}
                  >
                    No backups found.
                  </td>
                </tr>
              )}
          </tbody>
        </table>
      </div>

      <CreateBackupModal
        isOpen={isCreateModalOpen}
        onClose={() => setCreateModalOpen(false)}
        onCreate={handleCreateBackup}
        servers={servers}
        defaultServerId={!isGlobalView ? id : undefined}
      />

      {selectedBackup && (
        <RestoreBackupModal
          isOpen={restoreModalOpen}
          onClose={() => {
            setRestoreModalOpen(false);
            setSelectedBackup(null);
          }}
          onRestore={handleRestore}
          backupName={selectedBackup}
          servers={servers}
        />
      )}

      {backupToDelete && (
        <ConfirmationModal
          isOpen={!!backupToDelete}
          onClose={() => setBackupToDelete(null)}
          onConfirm={confirmDelete}
          title="Delete Backup"
          message={`Are you sure you want to delete the backup "${backupToDelete}"? This action cannot be undone.`}
          confirmText="Delete"
          isDangerous={true}
        />
      )}
    </div>
  );
};

export default Backups;
