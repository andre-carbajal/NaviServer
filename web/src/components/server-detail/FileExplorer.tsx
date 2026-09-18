import type { AxiosError } from 'axios';
import {
  ArrowUp,
  ChevronRight,
  Download,
  Edit2,
  File as FileIcon,
  Folder,
  FolderUp,
  Home,
  Loader2,
  Plus,
  RefreshCw,
  Trash2,
  Upload,
} from 'lucide-react';

import React, { useCallback, useEffect, useState } from 'react';

import { useModalDialog } from '../../hooks/useModalDialog';
import { api } from '../../services/api';
import type { FileEntry } from '../../types';
import {
  formatFileSize,
  getParentServerPath,
  isEditableFile,
  joinServerPath,
} from '../../utils/fileExplorer';
import { Button } from '../ui/Button';
import { Modal } from '../ui/Modal';
import FileEditor from './FileEditor';

interface FileExplorerProps {
  serverId: string;
}

interface ExtendedFile extends File {
  webkitRelativePath: string;
}

interface FileSystemEntry {
  isFile: boolean;
  isDirectory: boolean;
  name: string;
  fullPath: string;
  file: (callback: (file: File) => void) => void;
  createReader: () => {
    readEntries: (callback: (entries: FileSystemEntry[]) => void) => void;
  };
}

const FileExplorer: React.FC<FileExplorerProps> = ({ serverId }) => {
  const [currentPath, setCurrentPath] = useState('/');
  const [files, setFiles] = useState<FileEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editingFile, setEditingFile] = useState<string | null>(null);
  const [creatingDir, setCreatingDir] = useState(false);
  const [newDirName, setNewDirName] = useState('');
  const [uploading, setUploading] = useState(false);
  const [filePendingDelete, setFilePendingDelete] = useState<FileEntry | null>(
    null,
  );
  const [deletingFile, setDeletingFile] = useState(false);
  const { showAlert, showConfirm, modalDialog } = useModalDialog();
  const fileInputRef = React.useRef<HTMLInputElement>(null);
  const folderInputRef = React.useRef<HTMLInputElement>(null);

  const loadFiles = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await api.listFiles(serverId, currentPath);
      setFiles(response.data || []);
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message || 'Failed to load files');
      } else if (err && typeof err === 'object' && 'response' in err) {
        const axiosError = err as AxiosError<string>;
        setError(axiosError.response?.data || 'Failed to load files');
      } else {
        setError('Failed to load files');
      }
    } finally {
      setLoading(false);
    }
  }, [serverId, currentPath]);

  useEffect(() => {
    const loadTimer = window.setTimeout(() => {
      void loadFiles();
    }, 0);
    return () => window.clearTimeout(loadTimer);
  }, [loadFiles]);

  const handleUp = () => {
    if (currentPath === '/') return;
    const parentPath = getParentServerPath(currentPath);
    setCurrentPath(parentPath);
  };

  const handleFileClick = (file: FileEntry) => {
    if (file.isDirectory) {
      const newPath = joinServerPath(currentPath, file.name);
      setCurrentPath(newPath);
    } else {
      if (!isEditableFile(file.name)) {
        void showAlert({
          title: 'Cannot Edit File',
          message: 'This file type cannot be edited.',
          variant: 'danger',
        });
        return;
      }
      const filePath = joinServerPath(currentPath, file.name);
      setEditingFile(filePath);
    }
  };

  const handleDelete = (file: FileEntry) => {
    setFilePendingDelete(file);
  };

  const handleConfirmDelete = async () => {
    if (!filePendingDelete || deletingFile) return;

    const filePath = joinServerPath(currentPath, filePendingDelete.name);

    setDeletingFile(true);
    try {
      await api.deleteFile(serverId, filePath);
      setFilePendingDelete(null);
      void loadFiles();
    } catch (err: unknown) {
      let errorMessage = 'Failed to delete file';
      if (err instanceof Error) {
        errorMessage = err.message;
      } else if (err && typeof err === 'object' && 'response' in err) {
        const axiosError = err as AxiosError<string>;
        errorMessage = axiosError.response?.data || errorMessage;
      }
      setError(errorMessage);
    } finally {
      setDeletingFile(false);
    }
  };

  const handleCreateDir = async () => {
    if (!newDirName) return;
    const newPath = joinServerPath(currentPath, newDirName);

    try {
      await api.createDirectory(serverId, newPath);
      setCreatingDir(false);
      setNewDirName('');
      await loadFiles();
    } catch (err: unknown) {
      let errorMessage = 'Failed to create directory';
      if (err instanceof Error) {
        errorMessage = err.message;
      } else if (err && typeof err === 'object' && 'response' in err) {
        const axiosError = err as AxiosError<string>;
        errorMessage = axiosError.response?.data || errorMessage;
      }
      await showAlert({
        title: 'Action Failed',
        message: errorMessage,
        variant: 'danger',
      });
    }
  };

  const [isDragging, setIsDragging] = useState(false);

  const uploadFiles = async (filesToUpload: FileList | File[]) => {
    if (!filesToUpload.length) return;

    // Check if we are uploading a folder (using button)
    const foldersToUpload = new Set<string>();
    for (const file of filesToUpload) {
      const relativePath = (file as ExtendedFile).webkitRelativePath;
      if (relativePath) {
        const rootFolder = relativePath.split('/')[0];
        if (rootFolder) foldersToUpload.add(rootFolder);
      }
    }

    for (const folderName of foldersToUpload) {
      // Use the 'files' state (existing files) to check for duplicates
      if (files.some((f) => f.name === folderName && f.isDirectory)) {
        await showAlert({
          title: 'Folder Already Exists',
          message: `A folder named "${folderName}" already exists. Please delete it or rename it before uploading.`,
          variant: 'danger',
        });
        return;
      }
    }

    const confirmMessage =
      filesToUpload.length === 1
        ? `Are you sure you want to upload ${filesToUpload[0].name}?`
        : `Are you sure you want to upload ${filesToUpload.length} items?`;

    const shouldUpload = await showConfirm({
      title: 'Upload Files',
      message: confirmMessage,
      confirmText: 'Upload',
    });
    if (!shouldUpload) return;

    setUploading(true);
    try {
      for (const file of filesToUpload) {
        const relativePath = (file as ExtendedFile).webkitRelativePath;
        await api.uploadFile(serverId, currentPath, file, relativePath);
      }
      await loadFiles();
    } catch (err: unknown) {
      let errorMessage = 'Failed to upload file';
      if (err instanceof Error) {
        errorMessage = err.message;
      } else if (err && typeof err === 'object' && 'response' in err) {
        const axiosError = err as AxiosError<string>;
        errorMessage = axiosError.response?.data || errorMessage;
      }
      await showAlert({
        title: 'Upload Failed',
        message: errorMessage,
        variant: 'danger',
      });
    } finally {
      setUploading(false);
    }
  };

  const handleUploadClick = () => {
    fileInputRef.current?.click();
  };

  const handleFolderClick = () => {
    folderInputRef.current?.click();
  };

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files?.length) {
      await uploadFiles(e.target.files);
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  };

  const handleFolderChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files?.length) {
      await uploadFiles(e.target.files);
      if (folderInputRef.current) folderInputRef.current.value = '';
    }
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(true);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);
  };

  const handleDrop = async (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);

    const items = e.dataTransfer.items;
    if (!items) return;

    // Check for existing folders before processing
    for (const item of items) {
      const entry = item.webkitGetAsEntry() as FileSystemEntry | null;
      if (entry?.isDirectory) {
        if (files.some((f) => f.name === entry.name && f.isDirectory)) {
          await showAlert({
            title: 'Folder Already Exists',
            message: `A folder named "${entry.name}" already exists. Please delete it or rename it before uploading.`,
            variant: 'danger',
          });
          return;
        }
      }
    }

    const confirmMessage =
      items.length === 1
        ? `Are you sure you want to upload the dropped item?`
        : `Are you sure you want to upload ${items.length} dropped items?`;

    const shouldUploadDroppedItems = await showConfirm({
      title: 'Upload Dropped Items',
      message: confirmMessage,
      confirmText: 'Upload',
    });
    if (!shouldUploadDroppedItems) return;

    const filesToUpload: { file: File; relativePath?: string }[] = [];

    const traverseFileTree = async (
      entry: FileSystemEntry,
      path: string = '',
    ) => {
      if (entry.isFile) {
        const file = await new Promise<File>((resolve) => entry.file(resolve));
        filesToUpload.push({
          file,
          relativePath: path ? `${path}/${file.name}` : undefined,
        });
      } else if (entry.isDirectory) {
        const dirReader = entry.createReader();
        const entries = await new Promise<FileSystemEntry[]>((resolve) => {
          dirReader.readEntries(resolve);
        });
        for (const childEntry of entries) {
          await traverseFileTree(
            childEntry,
            path ? `${path}/${entry.name}` : entry.name,
          );
        }
      }
    };

    const promises = [];
    for (const item of items) {
      if (item.kind === 'file') {
        const entry = item.webkitGetAsEntry() as FileSystemEntry | null;
        if (entry) {
          promises.push(traverseFileTree(entry));
        }
      }
    }

    setUploading(true);
    await Promise.all(promises);

    try {
      for (const item of filesToUpload) {
        await api.uploadFile(
          serverId,
          currentPath,
          item.file,
          item.relativePath,
        );
      }
      loadFiles();
    } catch (err: unknown) {
      let errorMessage = 'Failed to upload files';
      if (err instanceof Error) {
        errorMessage = err.message;
      } else if (err && typeof err === 'object' && 'response' in err) {
        const axiosError = err as AxiosError<string>;
        errorMessage = axiosError.response?.data || errorMessage;
      }
      await showAlert({
        title: 'Upload Failed',
        message: errorMessage,
        variant: 'danger',
      });
    } finally {
      setUploading(false);
    }
  };

  const handleDownload = async (file: FileEntry) => {
    try {
      const filePath = joinServerPath(currentPath, file.name);
      const response = await api.downloadFile(serverId, filePath);
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      const downloadName = file.isDirectory ? `${file.name}.zip` : file.name;
      link.setAttribute('download', downloadName);
      document.body.appendChild(link);
      link.click();
      link.remove();
    } catch {
      await showAlert({
        title: 'Download Failed',
        message: 'Failed to download.',
        variant: 'danger',
      });
    }
  };

  if (editingFile) {
    return (
      <FileEditor
        serverId={serverId}
        filePath={editingFile}
        onClose={() => {
          setEditingFile(null);
          loadFiles();
        }}
      />
    );
  }

  const pathParts = currentPath.split('/').filter(Boolean);

  return (
    <div
      className="tw:relative tw:flex tw:h-[600px] tw:flex-col tw:rounded-lg tw:border tw:border-border tw:bg-[#1e1e1e] tw:shadow-sm tw:max-[1024px]:h-auto tw:max-[1024px]:min-h-0 tw:max-[1024px]:flex-1"
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
      style={{
        borderColor: isDragging ? '#646cff' : 'var(--border-color)',
        boxShadow: isDragging ? '0 0 0 2px rgba(100, 108, 255, 0.2)' : 'none',
      }}
    >
      {modalDialog}
      <Modal
        isOpen={filePendingDelete !== null}
        onClose={() => {
          if (!deletingFile) setFilePendingDelete(null);
        }}
        title={`Delete ${filePendingDelete?.isDirectory ? 'Folder' : 'File'}`}
      >
        <div className="tw:max-w-[420px] tw:pt-1 tw:[&_p]:m-0 tw:[&_p]:leading-[1.5] tw:[&_p]:text-text-muted tw:[&_strong]:text-text-main">
          <p>
            Do you want to delete{' '}
            <strong>{filePendingDelete?.name || 'this item'}</strong>?
          </p>
          <p className="tw:mt-3! tw:rounded-lg tw:border tw:border-red-500/30 tw:bg-red-500/10 tw:p-3 tw:text-red-200!">
            This action cannot be undone.
          </p>
          <div className="tw:mt-[25px] tw:flex tw:justify-end tw:gap-2.5 tw:max-[769px]:flex-col tw:max-[769px]:gap-2 tw:max-[769px]:[&_button]:w-full">
            <Button
              variant="secondary"
              onClick={() => setFilePendingDelete(null)}
              disabled={deletingFile}
            >
              Cancel
            </Button>
            <Button
              variant="danger"
              onClick={handleConfirmDelete}
              disabled={deletingFile}
            >
              {deletingFile ? 'Deleting...' : 'Delete'}
            </Button>
          </div>
        </div>
      </Modal>
      {isDragging && (
        <div className="tw:pointer-events-none tw:absolute tw:inset-0 tw:z-50 tw:flex tw:items-center tw:justify-center tw:rounded-lg tw:bg-primary/10 tw:backdrop-blur-[2px]">
          <div className="tw:flex tw:flex-col tw:items-center tw:gap-2.5 tw:text-[1.2rem] tw:font-bold tw:text-white">
            <Upload size={48} />
            <span>Drop files to upload</span>
          </div>
        </div>
      )}
      <div className="tw:flex tw:items-center tw:justify-between tw:border-b tw:border-border tw:p-4">
        <div className="tw:flex tw:items-center tw:gap-2 tw:overflow-hidden tw:text-sm tw:text-gray-400 tw:max-[769px]:gap-1 tw:max-[769px]:text-xs">
          <button
            type="button"
            onClick={() => setCurrentPath('/')}
            className="tw:flex tw:cursor-pointer tw:items-center tw:border-0 tw:bg-transparent tw:p-0 tw:text-inherit tw:hover:text-white"
          >
            <Home className="tw:w-4 tw:h-4" size={16} />
          </button>
          {pathParts.map((part, index) => {
            const path = '/' + pathParts.slice(0, index + 1).join('/');
            return (
              <React.Fragment key={path}>
                <ChevronRight
                  className="tw:w-4 tw:h-4 tw:text-gray-600"
                  size={16}
                />
                <button
                  type="button"
                  onClick={() => setCurrentPath(path)}
                  className="tw:flex tw:max-w-[150px] tw:cursor-pointer tw:items-center tw:overflow-hidden tw:border-0 tw:bg-transparent tw:p-0 tw:text-ellipsis tw:whitespace-nowrap tw:text-inherit tw:hover:text-white tw:max-[769px]:px-1"
                >
                  {part}
                </button>
              </React.Fragment>
            );
          })}
        </div>
        <div className="tw:flex tw:items-center tw:gap-2">
          <button
            type="button"
            onClick={loadFiles}
            className="tw:flex tw:cursor-pointer tw:items-center tw:justify-center tw:rounded-md tw:border-0 tw:bg-transparent tw:p-2 tw:text-gray-400 tw:transition-all tw:duration-200 tw:hover:bg-[#2d2d2d] tw:hover:text-white tw:disabled:cursor-not-allowed tw:disabled:bg-transparent tw:disabled:opacity-50 tw:disabled:text-gray-400"
            title="Refresh"
          >
            <RefreshCw
              size={16}
              className={loading ? 'tw:animate-spin' : undefined}
            />
          </button>
          <button
            type="button"
            onClick={handleUp}
            disabled={currentPath === '/'}
            className="tw:flex tw:cursor-pointer tw:items-center tw:justify-center tw:rounded-md tw:border-0 tw:bg-transparent tw:p-2 tw:text-gray-400 tw:transition-all tw:duration-200 tw:hover:bg-[#2d2d2d] tw:hover:text-white tw:disabled:cursor-not-allowed tw:disabled:bg-transparent tw:disabled:opacity-50 tw:disabled:text-gray-400"
            title="Go Up"
          >
            <ArrowUp size={16} />
          </button>
          <button
            type="button"
            onClick={() => setCreatingDir(true)}
            className="tw:flex tw:cursor-pointer tw:items-center tw:justify-center tw:rounded-md tw:border-0 tw:bg-transparent tw:p-2 tw:text-gray-400 tw:transition-all tw:duration-200 tw:hover:bg-[#2d2d2d] tw:hover:text-white tw:disabled:cursor-not-allowed tw:disabled:bg-transparent tw:disabled:opacity-50 tw:disabled:text-gray-400"
            title="New Folder"
          >
            <Plus size={16} />
          </button>
          <button
            type="button"
            onClick={handleUploadClick}
            className="tw:flex tw:cursor-pointer tw:items-center tw:justify-center tw:rounded-md tw:border-0 tw:bg-transparent tw:p-2 tw:text-gray-400 tw:transition-all tw:duration-200 tw:hover:bg-[#2d2d2d] tw:hover:text-white tw:disabled:cursor-not-allowed tw:disabled:bg-transparent tw:disabled:opacity-50 tw:disabled:text-gray-400"
            title="Upload File"
            disabled={uploading}
          >
            {uploading ? (
              <Loader2 size={16} className="tw:animate-spin" />
            ) : (
              <Upload size={16} />
            )}
          </button>
          <button
            type="button"
            onClick={handleFolderClick}
            className="tw:flex tw:cursor-pointer tw:items-center tw:justify-center tw:rounded-md tw:border-0 tw:bg-transparent tw:p-2 tw:text-gray-400 tw:transition-all tw:duration-200 tw:hover:bg-[#2d2d2d] tw:hover:text-white tw:disabled:cursor-not-allowed tw:disabled:bg-transparent tw:disabled:opacity-50 tw:disabled:text-gray-400"
            title="Upload Folder"
            disabled={uploading}
          >
            {uploading ? (
              <Loader2 size={16} className="tw:animate-spin" />
            ) : (
              <FolderUp size={16} />
            )}
          </button>
          <input
            type="file"
            aria-label="Upload files"
            ref={fileInputRef}
            onChange={handleFileChange}
            className="tw:hidden"
          />
          <input
            type="file"
            aria-label="Upload folder"
            ref={folderInputRef}
            onChange={handleFolderChange}
            className="tw:hidden"
            {...({
              webkitdirectory: '',
              directory: '',
            } as React.InputHTMLAttributes<HTMLInputElement> & {
              webkitdirectory?: string;
            })}
          />
        </div>
      </div>

      {error && (
        <div className="tw:border-b tw:border-red-900/50 tw:bg-red-900/20 tw:p-4 tw:text-red-400">
          {error}
        </div>
      )}

      {creatingDir && (
        <div className="tw:flex tw:items-center tw:gap-2 tw:border-b tw:border-border tw:bg-[#252525] tw:p-2">
          <Folder size={16} className="tw:ml-2 tw:text-indigo-400" />
          <input
            type="text"
            aria-label="New folder name"
            value={newDirName}
            onChange={(e) => setNewDirName(e.target.value)}
            placeholder="New folder name..."
            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-2 tw:py-1 tw:text-sm tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400"
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleCreateDir();
              if (e.key === 'Escape') setCreatingDir(false);
            }}
          />
          <button
            type="button"
            onClick={handleCreateDir}
            className="tw:whitespace-nowrap tw:inline-flex tw:items-center tw:justify-center tw:gap-2 tw:rounded-lg tw:border tw:border-transparent tw:bg-primary tw:px-3 tw:py-1 tw:text-xs tw:leading-[1.5] tw:font-semibold tw:text-white tw:transition-all tw:duration-200"
          >
            Create
          </button>
          <button
            type="button"
            onClick={() => setCreatingDir(false)}
            className="tw:whitespace-nowrap tw:inline-flex tw:items-center tw:justify-center tw:gap-2 tw:rounded-lg tw:border tw:border-border tw:bg-transparent tw:px-3 tw:py-1 tw:text-xs tw:leading-[1.5] tw:font-semibold tw:text-text-main tw:transition-all tw:duration-200"
          >
            Cancel
          </button>
        </div>
      )}

      <div className="file-list-container tw:flex-1 tw:overflow-y-auto">
        {loading && (!files || files.length === 0) ? (
          <div className="tw:flex tw:h-full tw:items-center tw:justify-center tw:text-gray-500">
            Loading...
          </div>
        ) : (
          <table className="tw:w-full tw:border-collapse tw:text-left tw:text-sm tw:[&_thead]:sticky tw:[&_thead]:top-0 tw:[&_thead]:z-10 tw:[&_thead]:bg-[#252525] tw:[&_thead]:text-xs tw:[&_thead]:text-gray-500 tw:[&_thead]:uppercase tw:[&_th]:px-4 tw:[&_th]:py-2 tw:[&_th]:font-medium tw:[&_td]:border-b tw:[&_td]:border-[#2d2d2d] tw:[&_td]:px-4 tw:[&_td]:py-3 tw:max-[769px]:[&_th]:px-2 tw:max-[769px]:[&_th]:py-1.5 tw:max-[769px]:[&_th]:text-[0.7rem] tw:max-[769px]:[&_td]:p-2 tw:max-[769px]:[&_td]:text-[0.8rem]">
            <thead>
              <tr>
                <th className="tw:w-8" aria-label="File type"></th>
                <th>Name</th>
                <th className="tw:w-32">Size</th>
                <th className="tw:w-48">Last Modified</th>
                <th className="tw:w-24">Actions</th>
              </tr>
            </thead>
            <tbody>
              {currentPath !== '/' && (
                <tr
                  className="tw:group tw:cursor-pointer tw:transition-colors tw:duration-200 tw:hover:bg-[#252525]"
                  onClick={handleUp}
                >
                  <td className="tw:text-center">
                    <Folder size={16} className="tw:text-indigo-400" />
                  </td>
                  <td className="tw:font-medium tw:text-indigo-300">..</td>
                  <td className="tw:text-gray-500">-</td>
                  <td className="tw:text-gray-500">-</td>
                  <td aria-label="No actions available"></td>
                </tr>
              )}
              {(files || []).map((file) => (
                <tr
                  key={file.name}
                  className="tw:group tw:cursor-pointer tw:transition-colors tw:duration-200 tw:hover:bg-[#252525]"
                  onClick={() => handleFileClick(file)}
                >
                  <td className="tw:text-center">
                    {file.isDirectory ? (
                      <Folder size={16} className="tw:text-indigo-400" />
                    ) : (
                      <FileIcon size={16} className="tw:text-gray-400" />
                    )}
                  </td>
                  <td
                    className={
                      file.isDirectory
                        ? 'tw:font-medium tw:text-indigo-300'
                        : 'tw:text-gray-300'
                    }
                  >
                    {file.name}
                  </td>
                  <td className="tw:text-gray-500">
                    {file.isDirectory ? '-' : formatFileSize(file.size)}
                  </td>
                  <td className="tw:text-gray-500">
                    {new Date(file.lastModified).toLocaleString()}
                  </td>
                  <td>
                    <div className="tw:flex tw:items-center tw:gap-2 tw:opacity-0 tw:transition-opacity tw:duration-200 tw:group-hover:opacity-100">
                      {!file.isDirectory && isEditableFile(file.name) && (
                        <button
                          type="button"
                          onClick={(event) => {
                            event.stopPropagation();
                            handleFileClick(file);
                          }}
                          className="tw:cursor-pointer tw:rounded-sm tw:border-0 tw:bg-transparent tw:p-1 tw:text-gray-400 tw:transition-all tw:duration-200 tw:hover:bg-[#333] tw:hover:text-white"
                          title="Edit"
                        >
                          <Edit2 size={14} />
                        </button>
                      )}
                      <button
                        type="button"
                        onClick={(event) => {
                          event.stopPropagation();
                          handleDownload(file);
                        }}
                        className="tw:cursor-pointer tw:rounded-sm tw:border-0 tw:bg-transparent tw:p-1 tw:text-gray-400 tw:transition-all tw:duration-200 tw:hover:bg-[#333] tw:hover:text-white"
                        title="Download"
                      >
                        <Download size={14} />
                      </button>
                      <button
                        type="button"
                        onClick={(event) => {
                          event.stopPropagation();
                          handleDelete(file);
                        }}
                        className="tw:cursor-pointer tw:rounded-sm tw:border-0 tw:bg-transparent tw:p-1 tw:text-gray-400 tw:transition-all tw:duration-200 tw:hover:bg-[#333] tw:hover:text-white tw:hover:text-red-400"
                        title="Delete"
                      >
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
              {(!files || files.length === 0) && !loading && (
                <tr>
                  <td
                    colSpan={5}
                    className="tw:p-8 tw:text-center tw:text-gray-500"
                  >
                    Folder is empty
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
};

export default FileExplorer;
