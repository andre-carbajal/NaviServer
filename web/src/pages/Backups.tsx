import {
  Download,
  Edit,
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
import EditBackupModal from '../components/EditBackupModal';
import type { RestoreData } from '../components/RestoreBackupModal';
import RestoreBackupModal from '../components/RestoreBackupModal';
import UploadBackupModal from '../components/UploadBackupModal';
import { Button } from '../components/ui/Button';
import { useAuth } from '../context/AuthContext';
import { useModalDialog } from '../hooks/useModalDialog';
import { useServers } from '../hooks/useServers';
import { WS_BASE_URL, api } from '../services/api';
import type { Backup } from '../types';

interface CreatingBackup extends Backup {
  serverId: string;
}

interface UploadingBackup {
  id: string;
  name: string;
  progress: number;
}

const CREATING_BACKUPS_STORAGE_KEY = 'creating_backups:v1';
const LEGACY_CREATING_BACKUPS_STORAGE_KEY = 'creating_backups';

const readCreatingBackups = (): CreatingBackup[] => {
  const stored =
    localStorage.getItem(CREATING_BACKUPS_STORAGE_KEY) ??
    localStorage.getItem(LEGACY_CREATING_BACKUPS_STORAGE_KEY);

  if (!stored) return [];

  try {
    return JSON.parse(stored) as CreatingBackup[];
  } catch (error) {
    console.error(error);
    return [];
  }
};

const writeCreatingBackups = (backups: CreatingBackup[]) => {
  localStorage.setItem(CREATING_BACKUPS_STORAGE_KEY, JSON.stringify(backups));
  localStorage.removeItem(LEGACY_CREATING_BACKUPS_STORAGE_KEY);
};

type AutoBackupUnit = 'minute' | 'hour' | 'day';

interface AutoBackupDraft {
  enabled: boolean;
  intervalValue: number;
  intervalUnit: AutoBackupUnit;
  maxBackups: number;
  saving: boolean;
  dirty: boolean;
  saved: boolean;
}

const formatBackupDateTime = (value?: string) => {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return date.toLocaleString();
};

const Backups: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { token, user } = useAuth();
  const [backups, setBackups] = useState<Backup[]>([]);
  const [creatingBackups, setCreatingBackups] = useState<CreatingBackup[]>([]);
  const [uploadingBackups, setUploadingBackups] = useState<UploadingBackup[]>(
    [],
  );
  const [isDragging, setIsDragging] = useState(false);
  const { servers, refresh: refreshServers } = useServers();
  const { showAlert, modalDialog } = useModalDialog();
  const [isCreateModalOpen, setCreateModalOpen] = useState(false);
  const [isUploadModalOpen, setUploadModalOpen] = useState(false);
  const [isEditModalOpen, setEditModalOpen] = useState(false);
  const [restoreModalOpen, setRestoreModalOpen] = useState(false);
  const [selectedBackup, setSelectedBackup] = useState<string | null>(null);
  const [backupToEdit, setBackupToEdit] = useState<Backup | null>(null);
  const [backupToDelete, setBackupToDelete] = useState<string | null>(null);
  const [autoBackupDrafts, setAutoBackupDrafts] = useState<
    Record<string, AutoBackupDraft>
  >({});
  const activeSockets = useRef<Set<string>>(null!);
  const wsMap = useRef<Map<string, WebSocket>>(null!);
  const fileInputRef = useRef<HTMLInputElement>(null);

  if (activeSockets.current === null) {
    activeSockets.current = new Set();
  }

  if (wsMap.current === null) {
    wsMap.current = new Map();
  }

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
      await uploadFiles(files, id && id !== 'all' ? id : undefined);
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
    const newList = readCreatingBackups().filter(
      (b) => b.requestId !== requestId,
    );
    writeCreatingBackups(newList);

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
        `${WS_BASE_URL}/ws/progress/${requestId}?token=${token}`,
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
    const restoreTimer = window.setTimeout(() => {
      const list = readCreatingBackups();
      if (list.length === 0) return;

      writeCreatingBackups(list);
      setCreatingBackups(list);
      list.forEach((b) => {
        if (b.requestId) trackProgress(b.requestId);
      });
    }, 0);

    return () => window.clearTimeout(restoreTimer);
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

    const list = readCreatingBackups();
    list.push(tempBackup);
    writeCreatingBackups(list);

    trackProgress(requestId);

    try {
      await api.createBackup(serverId, name, requestId);
    } catch (error) {
      console.error('Failed to initiate backup creation:', error);
      removeCreatingBackup(requestId);
      await showAlert({
        title: 'Backup Failed',
        message: 'Failed to start backup creation.',
        variant: 'danger',
      });
    }
  };

  const handleCancelBackup = (requestId: string) => {
    api
      .cancelBackupCreation(requestId)
      .catch((e) => console.error('Error cancelling backup in backend:', e));
    removeCreatingBackup(requestId);
  };

  const handleEditClick = (backup: Backup) => {
    setBackupToEdit(backup);
    setEditModalOpen(true);
  };

  const handleUpdateBackup = async (serverId: string) => {
    if (backupToEdit) {
      try {
        await api.updateBackup(backupToEdit.name, serverId);
        fetchBackups();
      } catch (error) {
        console.error('Failed to update backup association:', error);
      }
    }
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
    await showAlert({
      title: 'Backup Restored',
      message: 'Backup restored successfully.',
      variant: 'success',
    });
    await refreshServers();
  };

  const handleUploadClick = () => {
    setUploadModalOpen(true);
  };

  const uploadFiles = async (files: FileList | File[], serverId?: string) => {
    if (!files || files.length === 0) return;

    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      const ext = file.name.split('.').pop()?.toLowerCase();

      if (ext !== 'zip' && ext !== 'rar') {
        await showAlert({
          title: 'Invalid Backup File',
          message: `File ${file.name} is not a valid backup file (.zip or .rar only).`,
          variant: 'danger',
        });
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
        await api.uploadBackup(
          file,
          (progressEvent) => {
            const progress = Math.round(
              (progressEvent.loaded * 100) / (progressEvent.total ?? 1),
            );
            setUploadingBackups((prev) =>
              prev.map((b) =>
                b.id === uploadId ? { ...b, progress: progress } : b,
              ),
            );
          },
          serverId,
        );
      } catch (error) {
        console.error(`Failed to upload backup ${file.name}:`, error);
        await showAlert({
          title: 'Upload Failed',
          message: `Failed to upload backup ${file.name}.`,
          variant: 'danger',
        });
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
      await uploadFiles(files, id && id !== 'all' ? id : undefined);
    }
  };

  const isGlobalView = !id || id === 'all';

  const visibleCreatingBackups = creatingBackups.filter(
    (b) => isGlobalView || b.serverId === id,
  );

  useEffect(() => {
    const draftTimer = window.setTimeout(() => {
      setAutoBackupDrafts((prev) => {
        const next: Record<string, AutoBackupDraft> = {};

        servers.forEach((server) => {
          const serverConfig: Omit<
            AutoBackupDraft,
            'saving' | 'dirty' | 'saved'
          > = {
            enabled: server.autoBackupEnabled ?? false,
            intervalValue: server.autoBackupIntervalValue ?? 24,
            intervalUnit: server.autoBackupIntervalUnit ?? 'hour',
            maxBackups: server.autoBackupMaxBackups ?? 10,
          };

          const existing = prev[server.id];
          if (!existing) {
            next[server.id] = {
              ...serverConfig,
              saving: false,
              dirty: false,
              saved: false,
            };
            return;
          }

          if (existing.dirty || existing.saving) {
            next[server.id] = existing;
            return;
          }

          next[server.id] = {
            ...serverConfig,
            saving: false,
            dirty: false,
            saved: false,
          };
        });

        return next;
      });
    }, 0);

    return () => window.clearTimeout(draftTimer);
  }, [servers]);

  const updateAutoBackupDraft = (
    serverId: string,
    patch: Partial<AutoBackupDraft>,
    markDirty = false,
  ) => {
    setAutoBackupDrafts((prev) => ({
      ...prev,
      [serverId]: {
        ...(prev[serverId] ?? {
          enabled: false,
          intervalValue: 24,
          intervalUnit: 'hour' as AutoBackupUnit,
          maxBackups: 10,
          saving: false,
          dirty: false,
          saved: false,
        }),
        ...patch,
        dirty: markDirty
          ? true
          : (patch.dirty ?? prev[serverId]?.dirty ?? false),
      },
    }));
  };

  const handleSaveAutoBackup = async (serverId: string) => {
    const draft = autoBackupDrafts[serverId];
    if (!draft) return;

    const value = Number(draft.intervalValue);
    const limit = Number(draft.maxBackups);

    const minutes =
      draft.intervalUnit === 'minute'
        ? value
        : draft.intervalUnit === 'hour'
          ? value * 60
          : value * 24 * 60;

    if (minutes < 5) {
      await showAlert({
        title: 'Invalid Interval',
        message: 'Automatic backup interval must be at least 5 minutes.',
        variant: 'danger',
      });
      return;
    }
    if (minutes > 30 * 24 * 60) {
      await showAlert({
        title: 'Invalid Interval',
        message: 'Automatic backup interval cannot exceed 30 days.',
        variant: 'danger',
      });
      return;
    }
    if (!Number.isFinite(limit) || limit <= 0) {
      await showAlert({
        title: 'Invalid Backup Limit',
        message: 'Max backups must be greater than 0.',
        variant: 'danger',
      });
      return;
    }

    updateAutoBackupDraft(serverId, { saving: true, saved: false });
    try {
      await api.updateServerAutoBackup(serverId, {
        enabled: draft.enabled,
        intervalValue: value,
        intervalUnit: draft.intervalUnit,
        maxBackups: limit,
      });
      updateAutoBackupDraft(serverId, {
        dirty: false,
        saved: true,
      });
      setTimeout(() => {
        updateAutoBackupDraft(serverId, { saved: false });
      }, 2500);
    } catch (error) {
      console.error('Failed to save auto backup config:', error);
      await showAlert({
        title: 'Save Failed',
        message: 'Failed to save automatic backup configuration.',
        variant: 'danger',
      });
    } finally {
      updateAutoBackupDraft(serverId, { saving: false });
    }
  };

  const serverForBackup = (serverId?: string) =>
    serverId ? servers.find((server) => server.id === serverId) : undefined;

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
      {modalDialog}
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
        <h2 className="backup-section-title">Automatic Backups</h2>
        {user?.role !== 'admin' ? (
          <p className="text-muted">
            Only administrators can configure automatic backups.
          </p>
        ) : (
          <div className="auto-backup-grid">
            <div className="auto-backup-header">
              <span>Server</span>
              <span>Enabled</span>
              <span>Every</span>
              <span>Max backups</span>
              <span>Action</span>
            </div>
            {servers.map((server) => {
              const draft = autoBackupDrafts[server.id];
              if (!draft) return null;

              return (
                <div key={server.id} className="auto-backup-row">
                  <div className="auto-backup-row-main">
                    <img
                      src={api.getServerIconUrl(server.id)}
                      alt="Server icon"
                      className="auto-backup-server-icon"
                      onError={(event) => {
                        const target = event.currentTarget;
                        target.style.display = 'none';
                        const fallback = target.nextElementSibling;
                        if (fallback instanceof HTMLElement) {
                          fallback.style.display = 'flex';
                        }
                      }}
                    />
                    <div
                      className="auto-backup-server-fallback"
                      style={{ display: 'none' }}
                    >
                      {server.name.charAt(0).toUpperCase()}
                    </div>
                    <div>
                      <strong>{server.name}</strong>
                      <div className="text-muted">{server.id}</div>
                    </div>
                  </div>
                  <label className="auto-backup-toggle">
                    <input
                      type="checkbox"
                      checked={draft.enabled}
                      onChange={(event) =>
                        updateAutoBackupDraft(
                          server.id,
                          {
                            enabled: event.target.checked,
                          },
                          true,
                        )
                      }
                    />
                    <span>Enabled</span>
                  </label>
                  <div className="auto-backup-interval">
                    <input
                      type="number"
                      aria-label={`${server.name} auto backup interval value`}
                      min={1}
                      className="form-input"
                      value={draft.intervalValue}
                      onChange={(event) =>
                        updateAutoBackupDraft(
                          server.id,
                          {
                            intervalValue: Number(event.target.value),
                          },
                          true,
                        )
                      }
                    />
                    <select
                      aria-label={`${server.name} auto backup interval unit`}
                      className="form-select"
                      value={draft.intervalUnit}
                      onChange={(event) =>
                        updateAutoBackupDraft(
                          server.id,
                          {
                            intervalUnit: event.target.value as AutoBackupUnit,
                          },
                          true,
                        )
                      }
                    >
                      <option value="minute">Minutes</option>
                      <option value="hour">Hours</option>
                      <option value="day">Days</option>
                    </select>
                  </div>
                  <div className="auto-backup-limit">
                    <input
                      type="number"
                      aria-label={`${server.name} maximum automatic backups`}
                      min={1}
                      className="form-input"
                      value={draft.maxBackups}
                      onChange={(event) =>
                        updateAutoBackupDraft(
                          server.id,
                          {
                            maxBackups: Number(event.target.value),
                          },
                          true,
                        )
                      }
                    />
                  </div>
                  <Button
                    onClick={() => handleSaveAutoBackup(server.id)}
                    disabled={draft.saving}
                  >
                    {draft.saving ? 'Saving...' : 'Save'}
                  </Button>
                  {draft.saved && (
                    <span
                      style={{
                        color: '#22c55e',
                        fontSize: '0.85rem',
                        fontWeight: 600,
                      }}
                    >
                      Saved successfully
                    </span>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      <div className="card">
        <h2 className="backup-section-title">Backups</h2>
        <input
          type="file"
          aria-label="Upload backup files"
          ref={fileInputRef}
          onChange={handleFileChange}
          style={{ display: 'none' }}
          accept=".zip,.rar"
          multiple
        />
        <table className="data-table backups-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Server</th>
              <th>Date & Time</th>
              <th>Size</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {uploadingBackups.map((upload) => (
              <tr key={upload.id}>
                <td data-label="Name">
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
                <td data-label="Server">-</td>
                <td data-label="Date & Time">-</td>
                <td data-label="Size">-</td>
                <td data-label="Actions">-</td>
              </tr>
            ))}
            {visibleCreatingBackups.map((backup) => (
              <tr key={backup.requestId}>
                <td data-label="Name">
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
                <td data-label="Server">{backup.serverName || '-'}</td>
                <td data-label="Date & Time">-</td>
                <td data-label="Size">-</td>
                <td data-label="Actions">
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
                <td data-label="Name">{backup.name}</td>
                <td data-label="Server">
                  {backup.serverName ? (
                    <div className="backup-server-cell">
                      {serverForBackup(backup.serverId) ? (
                        <img
                          src={api.getServerIconUrl(backup.serverId!)}
                          alt="Server icon"
                          className="backup-server-icon"
                          onError={(event) => {
                            event.currentTarget.style.display = 'none';
                          }}
                        />
                      ) : null}
                      <span>{backup.serverName}</span>
                    </div>
                  ) : (
                    <span className="text-muted">None</span>
                  )}
                </td>
                <td data-label="Date & Time">
                  {formatBackupDateTime(backup.createdAt)}
                </td>
                <td data-label="Size">
                  {(backup.size / 1024 / 1024).toFixed(2)} MB
                </td>
                <td data-label="Actions">
                  <div
                    className="actions-group"
                    style={{ border: 'none', padding: 0, margin: 0 }}
                  >
                    {user?.role === 'admin' && (
                      <button
                        type="button"
                        className="icon-action"
                        title="Edit Association"
                        onClick={() => handleEditClick(backup)}
                      >
                        <Edit size={18} />
                      </button>
                    )}
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
                      type="button"
                      className="icon-action"
                      title="Restore"
                      onClick={() => handleRestoreClick(backup.name)}
                    >
                      <RotateCcw size={18} />
                    </button>
                    <button
                      type="button"
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
                    colSpan={5}
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

      <UploadBackupModal
        isOpen={isUploadModalOpen}
        onClose={() => setUploadModalOpen(false)}
        onUpload={(file, serverId) => uploadFiles([file], serverId)}
        servers={servers}
        defaultServerId={!isGlobalView ? id : undefined}
      />

      {backupToEdit && (
        <EditBackupModal
          isOpen={isEditModalOpen}
          onClose={() => {
            setEditModalOpen(false);
            setBackupToEdit(null);
          }}
          onUpdate={handleUpdateBackup}
          servers={servers}
          currentServerId={backupToEdit.serverId}
          backupName={backupToEdit.name}
        />
      )}

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
