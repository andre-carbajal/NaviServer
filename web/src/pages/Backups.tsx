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
import AutomaticBackupSettings, {
  type AutoBackupDraft,
  type AutoBackupUnit,
} from '../components/backups/AutomaticBackupSettings';
import CreateBackupModal from '../components/backups/CreateBackupModal';
import EditBackupModal from '../components/backups/EditBackupModal';
import type { RestoreData } from '../components/backups/RestoreBackupModal';
import RestoreBackupModal from '../components/backups/RestoreBackupModal';
import UploadBackupModal from '../components/backups/UploadBackupModal';
import { Button } from '../components/ui/Button';
import { useAuth } from '../context/AuthContext';
import { useModalDialog } from '../hooks/useModalDialog';
import { useServers } from '../hooks/useServers';
import { WS_BASE_URL, api } from '../services/api';
import type { Backup } from '../types';
import {
  type CreatingBackup,
  formatBackupDateTime,
  readCreatingBackups,
  writeCreatingBackups,
} from '../utils/backups';

interface UploadingBackup {
  id: string;
  name: string;
  progress: number;
}

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
    await Promise.all([
      refreshServers(),
      showAlert({
        title: 'Backup Restored',
        message: 'Backup restored successfully.',
        variant: 'success',
      }),
    ]);
  };

  const handleUploadClick = () => {
    setUploadModalOpen(true);
  };

  const uploadFiles = async (files: FileList | File[], serverId?: string) => {
    if (!files || files.length === 0) return;

    for (const file of files) {
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

    let minutes = value * 24 * 60;
    if (draft.intervalUnit === 'minute') {
      minutes = value;
    } else if (draft.intervalUnit === 'hour') {
      minutes = value * 60;
    }

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
      className={`tw:relative tw:flex tw:min-h-full tw:flex-col tw:gap-4 ${isDragging ? 'tw:rounded-xl tw:border-2 tw:border-dashed tw:border-primary tw:shadow-[0_0_0_2px_rgba(100,108,255,0.2)]' : ''}`}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      {modalDialog}
      <div className="tw:mb-5 tw:flex tw:items-center tw:justify-between tw:gap-3 tw:max-[640px]:items-stretch">
        <h1 className="tw:m-0">Backups</h1>
        <div className="tw:flex tw:flex-wrap tw:items-center tw:gap-2.5 tw:max-[640px]:w-full tw:max-[640px]:flex-nowrap">
          <Button
            className="tw:max-[640px]:min-w-0 tw:max-[640px]:flex-1 tw:max-[640px]:justify-center tw:max-[640px]:px-2"
            onClick={handleUploadClick}
            variant="secondary"
          >
            <Upload size={20} />{' '}
            <span className="tw:max-[640px]:hidden">Upload Backup</span>
          </Button>
          <Button
            className="tw:max-[640px]:min-w-0 tw:max-[640px]:flex-1 tw:max-[640px]:justify-center tw:max-[640px]:px-2"
            onClick={() => setCreateModalOpen(true)}
          >
            <Plus size={20} />{' '}
            <span className="tw:max-[640px]:hidden">Create Backup</span>
          </Button>
        </div>
      </div>
      {isDragging && (
        <div className="tw:absolute tw:inset-0 tw:z-50 tw:flex tw:items-center tw:justify-center tw:rounded-xl tw:bg-primary/10 tw:backdrop-blur-sm tw:pointer-events-none">
          <div className="tw:flex tw:flex-col tw:items-center tw:gap-2.5 tw:text-[1.2rem] tw:font-bold tw:text-white">
            <Upload size={48} />
            <span>Drop backups to upload (.zip, .rar)</span>
          </div>
        </div>
      )}
      <AutomaticBackupSettings
        canConfigure={user?.role === 'admin'}
        servers={servers}
        drafts={autoBackupDrafts}
        onUpdate={updateAutoBackupDraft}
        onSave={handleSaveAutoBackup}
      />

      <div className="tw:rounded-xl tw:border tw:border-border tw:bg-bg-card tw:p-5">
        <h2 className="tw:mb-3 tw:mt-0">Backups</h2>
        <input
          type="file"
          aria-label="Upload backup files"
          ref={fileInputRef}
          onChange={handleFileChange}
          className="tw:hidden"
          accept=".zip,.rar"
          multiple
        />
        <table className="tw:box-border tw:mt-2.5 tw:w-full tw:border-collapse tw:text-[0.95rem] tw:[&_th]:border-b tw:[&_th]:border-border tw:[&_th]:p-4 tw:[&_th]:text-left tw:[&_th]:text-[0.8rem] tw:[&_th]:font-semibold tw:[&_th]:tracking-[0.05em] tw:[&_th]:text-text-muted tw:[&_th]:uppercase tw:[&_td]:border-b tw:[&_td]:border-border tw:[&_td]:p-4 tw:[&_td]:text-left tw:[&_tr:last-child_td]:border-b-0 tw:[&_tbody_tr]:transition-colors tw:[&_tbody_tr:hover]:bg-white/[0.03] tw:max-[1100px]:[&_thead]:hidden tw:max-[1100px]:[&_tbody]:block tw:max-[1100px]:[&_tr]:mb-2.5 tw:max-[1100px]:[&_tr]:block tw:max-[1100px]:[&_tr]:rounded-[10px] tw:max-[1100px]:[&_tr]:border tw:max-[1100px]:[&_tr]:border-border tw:max-[1100px]:[&_tr]:p-2 tw:max-[1100px]:[&_td]:block tw:max-[1100px]:[&_td]:box-border tw:max-[1100px]:[&_td]:w-full tw:max-[1100px]:[&_td]:!border-0 tw:max-[1100px]:[&_td]:!p-[6px_4px] tw:max-[1100px]:[&_td]:before:mb-0.5 tw:max-[1100px]:[&_td]:before:block tw:max-[1100px]:[&_td]:before:text-[0.72rem] tw:max-[1100px]:[&_td]:before:text-text-muted tw:max-[1100px]:[&_td]:before:uppercase tw:max-[1100px]:[&_td]:before:content-[attr(data-label)]">
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
                  <div className="tw:flex tw:items-center tw:gap-2">
                    <Loader2 className="tw:animate-spin" size={16} />
                    <div>
                      <div>{upload.name}</div>
                      <div className="tw:text-[0.8em] tw:text-text-muted">
                        Uploading...
                      </div>
                    </div>
                  </div>
                  <div className="tw:h-1 tw:w-full tw:overflow-hidden tw:rounded tw:bg-white/10">
                    <div
                      className="tw:h-full tw:rounded tw:bg-primary tw:transition-[width] tw:duration-300"
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
                  <div className="tw:flex tw:items-center tw:gap-2">
                    <Loader2 className="tw:animate-spin" size={16} />
                    <div>
                      <div>{backup.name}</div>
                      <div className="tw:text-[0.8em] tw:text-text-muted">
                        {backup.progressMessage}
                      </div>
                    </div>
                  </div>
                  {backup.progress !== undefined && (
                    <div className="tw:h-1 tw:w-full tw:overflow-hidden tw:rounded tw:bg-white/10">
                      <div
                        className="tw:h-full tw:rounded tw:bg-primary tw:transition-[width] tw:duration-300"
                        style={{ width: `${backup.progress}%` }}
                      />
                    </div>
                  )}
                </td>
                <td data-label="Server">{backup.serverName || '-'}</td>
                <td data-label="Date & Time">-</td>
                <td data-label="Size">-</td>
                <td data-label="Actions">
                  <div className="tw:flex tw:gap-[5px]">
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
                    <div className="tw:flex tw:items-center tw:gap-2">
                      {serverForBackup(backup.serverId) ? (
                        <img
                          src={api.getServerIconUrl(backup.serverId!)}
                          alt="Server icon"
                          className="tw:h-7 tw:w-7 tw:shrink-0 tw:rounded-md tw:object-contain [image-rendering:pixelated]"
                          onError={(event) => {
                            event.currentTarget.style.display = 'none';
                          }}
                        />
                      ) : null}
                      <span>{backup.serverName}</span>
                    </div>
                  ) : (
                    <span className="tw:text-text-muted">None</span>
                  )}
                </td>
                <td data-label="Date & Time">
                  {formatBackupDateTime(backup.createdAt)}
                </td>
                <td data-label="Size">
                  {(backup.size / 1024 / 1024).toFixed(2)} MB
                </td>
                <td data-label="Actions">
                  <div className="tw:flex tw:min-w-[180px] tw:flex-wrap tw:items-center tw:gap-3">
                    {user?.role === 'admin' && (
                      <button
                        type="button"
                        className="tw:flex tw:h-9 tw:w-9 tw:cursor-pointer tw:items-center tw:justify-center tw:rounded tw:border-0 tw:bg-transparent tw:p-0 tw:text-text-muted tw:transition-all tw:duration-200 tw:hover:bg-white/10 tw:hover:text-white"
                        title="Edit Association"
                        onClick={() => handleEditClick(backup)}
                      >
                        <Edit size={18} />
                      </button>
                    )}
                    <a
                      className="tw:flex tw:h-9 tw:w-9 tw:cursor-pointer tw:items-center tw:justify-center tw:rounded tw:border-0 tw:bg-transparent tw:p-0 tw:text-text-muted tw:transition-all tw:duration-200 tw:hover:bg-white/10 tw:hover:text-white"
                      title="Download"
                      href={api.getBackupDownloadUrl(backup.name)}
                      target="_blank"
                      rel="noreferrer"
                    >
                      <Download size={18} />
                    </a>
                    <button
                      type="button"
                      className="tw:flex tw:h-9 tw:w-9 tw:cursor-pointer tw:items-center tw:justify-center tw:rounded tw:border-0 tw:bg-transparent tw:p-0 tw:text-text-muted tw:transition-all tw:duration-200 tw:hover:bg-white/10 tw:hover:text-white"
                      title="Restore"
                      onClick={() => handleRestoreClick(backup.name)}
                    >
                      <RotateCcw size={18} />
                    </button>
                    <button
                      type="button"
                      className="tw:flex tw:h-9 tw:w-9 tw:cursor-pointer tw:items-center tw:justify-center tw:rounded tw:border-0 tw:bg-transparent tw:p-0 tw:text-text-muted tw:transition-all tw:duration-200 tw:hover:bg-red-500/10 tw:hover:text-red-500"
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
                    className="tw:!p-5 tw:!text-center tw:!text-text-muted"
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
