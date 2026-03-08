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

import { api } from '../services/api';
import type { FileEntry } from '../types';
import FileEditor from './FileEditor';

interface FileExplorerProps {
  serverId: string;
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
    loadFiles();
  }, [loadFiles]);

  const handleUp = () => {
    if (currentPath === '/') return;
    const parentPath = currentPath.split('/').slice(0, -1).join('/') || '/';
    setCurrentPath(parentPath);
  };

  const isEditable = (filename: string) => {
    const ignoredExtensions = [
      '.jar',
      '.zip',
      '.tar',
      '.gz',
      '.png',
      '.jpg',
      '.jpeg',
      '.gif',
      '.ico',
      '.exe',
      '.dll',
      '.so',
      '.dylib',
      '.DS_Store',
    ];
    if (filename === '.DS_Store') return false;
    const ext = filename.slice(filename.lastIndexOf('.')).toLowerCase();
    return !ignoredExtensions.includes(ext);
  };

  const handleFileClick = (file: FileEntry) => {
    if (file.isDirectory) {
      const newPath =
        currentPath === '/' ? `/${file.name}` : `${currentPath}/${file.name}`;
      setCurrentPath(newPath);
    } else {
      if (!isEditable(file.name)) {
        alert('This file type cannot be edited.');
        return;
      }
      const filePath =
        currentPath === '/' ? `/${file.name}` : `${currentPath}/${file.name}`;
      setEditingFile(filePath);
    }
  };

  const handleDelete = async (file: FileEntry) => {
    if (!confirm(`Are you sure you want to delete ${file.name}?`)) return;

    const filePath =
      currentPath === '/' ? `/${file.name}` : `${currentPath}/${file.name}`;

    try {
      await api.deleteFile(serverId, filePath);
      loadFiles();
    } catch (err: unknown) {
      let errorMessage = 'Failed to delete file';
      if (err instanceof Error) {
        errorMessage = err.message;
      } else if (err && typeof err === 'object' && 'response' in err) {
        const axiosError = err as AxiosError<string>;
        errorMessage = axiosError.response?.data || errorMessage;
      }
      alert(errorMessage);
    }
  };

  const handleCreateDir = async () => {
    if (!newDirName) return;
    const newPath =
      currentPath === '/' ? `/${newDirName}` : `${currentPath}/${newDirName}`;

    try {
      await api.createDirectory(serverId, newPath);
      setCreatingDir(false);
      setNewDirName('');
      loadFiles();
    } catch (err: unknown) {
      let errorMessage = 'Failed to create directory';
      if (err instanceof Error) {
        errorMessage = err.message;
      } else if (err && typeof err === 'object' && 'response' in err) {
        const axiosError = err as AxiosError<string>;
        errorMessage = axiosError.response?.data || errorMessage;
      }
      alert(errorMessage);
    }
  };

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const [isDragging, setIsDragging] = useState(false);

  const uploadFiles = async (filesToUpload: FileList | File[]) => {
    if (!filesToUpload.length) return;

    // Check if we are uploading a folder (using button)
    const foldersToUpload = new Set<string>();
    for (let i = 0; i < filesToUpload.length; i++) {
      const relativePath = (filesToUpload[i] as any).webkitRelativePath;
      if (relativePath) {
        const rootFolder = relativePath.split('/')[0];
        if (rootFolder) foldersToUpload.add(rootFolder);
      }
    }

    for (const folderName of foldersToUpload) {
      // Use the 'files' state (existing files) to check for duplicates
      if (files.some((f) => f.name === folderName && f.isDirectory)) {
        alert(
          `A folder named "${folderName}" already exists. Please delete it or rename it before uploading.`,
        );
        return;
      }
    }

    const confirmMessage =
      filesToUpload.length === 1
        ? `Are you sure you want to upload ${filesToUpload[0].name}?`
        : `Are you sure you want to upload ${filesToUpload.length} items?`;

    if (!confirm(confirmMessage)) return;

    setUploading(true);
    try {
      for (let i = 0; i < filesToUpload.length; i++) {
        const file = filesToUpload[i];
        const relativePath = (file as any).webkitRelativePath;
        await api.uploadFile(serverId, currentPath, file, relativePath);
      }
      loadFiles();
    } catch (err: unknown) {
      let errorMessage = 'Failed to upload file';
      if (err instanceof Error) {
        errorMessage = err.message;
      } else if (err && typeof err === 'object' && 'response' in err) {
        const axiosError = err as AxiosError<string>;
        errorMessage = axiosError.response?.data || errorMessage;
      }
      alert(errorMessage);
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
    for (let i = 0; i < items.length; i++) {
      const entry = items[i].webkitGetAsEntry();
      if (entry && entry.isDirectory) {
        if (files.some((f) => f.name === entry.name && f.isDirectory)) {
          alert(
            `A folder named "${entry.name}" already exists. Please delete it or rename it before uploading.`,
          );
          return;
        }
      }
    }

    const confirmMessage =
      items.length === 1
        ? `Are you sure you want to upload the dropped item?`
        : `Are you sure you want to upload ${items.length} dropped items?`;

    if (!confirm(confirmMessage)) return;

    const filesToUpload: { file: File; relativePath?: string }[] = [];

    const traverseFileTree = async (entry: any, path: string = '') => {
      if (entry.isFile) {
        const file = await new Promise<File>((resolve) => entry.file(resolve));
        filesToUpload.push({
          file,
          relativePath: path ? `${path}/${file.name}` : undefined,
        });
      } else if (entry.isDirectory) {
        const dirReader = entry.createReader();
        const entries = await new Promise<any[]>((resolve) => {
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
    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      if (item.kind === 'file') {
        const entry = item.webkitGetAsEntry();
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
      alert(errorMessage);
    } finally {
      setUploading(false);
    }
  };

  const handleDownload = async (file: FileEntry) => {
    try {
      const filePath =
        currentPath === '/' ? `/${file.name}` : `${currentPath}/${file.name}`;
      const response = await api.downloadFile(serverId, filePath);
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', file.name);
      document.body.appendChild(link);
      link.click();
      link.remove();
    } catch {
      alert('Failed to download file');
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

  const pathParts = currentPath.split('/').filter((p) => p);

  return (
    <div
      className="file-explorer-container"
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
      style={{
        position: 'relative',
        borderColor: isDragging ? '#646cff' : 'var(--border-color)',
        boxShadow: isDragging ? '0 0 0 2px rgba(100, 108, 255, 0.2)' : 'none',
      }}
    >
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
            borderRadius: '8px',
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
            <span>Drop files to upload</span>
          </div>
        </div>
      )}
      <div className="file-toolbar">
        <div className="breadcrumb-nav">
          <button
            onClick={() => setCurrentPath('/')}
            className="breadcrumb-btn"
          >
            <Home className="w-4 h-4" size={16} />
          </button>
          {pathParts.map((part, index) => {
            const path = '/' + pathParts.slice(0, index + 1).join('/');
            return (
              <React.Fragment key={path}>
                <ChevronRight className="w-4 h-4 text-gray-600" size={16} />
                <button
                  onClick={() => setCurrentPath(path)}
                  className="breadcrumb-btn"
                  style={{
                    maxWidth: '150px',
                    whiteSpace: 'nowrap',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                  }}
                >
                  {part}
                </button>
              </React.Fragment>
            );
          })}
        </div>
        <div className="toolbar-actions">
          <button onClick={loadFiles} className="toolbar-btn" title="Refresh">
            <RefreshCw size={16} className={`${loading ? 'spin' : ''}`} />
          </button>
          <button
            onClick={handleUp}
            disabled={currentPath === '/'}
            className="toolbar-btn"
            title="Go Up"
          >
            <ArrowUp size={16} />
          </button>
          <button
            onClick={() => setCreatingDir(true)}
            className="toolbar-btn"
            title="New Folder"
          >
            <Plus size={16} />
          </button>
          <button
            onClick={handleUploadClick}
            className="toolbar-btn"
            title="Upload File"
            disabled={uploading}
          >
            {uploading ? (
              <Loader2 size={16} className="spin" />
            ) : (
              <Upload size={16} />
            )}
          </button>
          <button
            onClick={handleFolderClick}
            className="toolbar-btn"
            title="Upload Folder"
            disabled={uploading}
          >
            {uploading ? (
              <Loader2 size={16} className="spin" />
            ) : (
              <FolderUp size={16} />
            )}
          </button>
          <input
            type="file"
            ref={fileInputRef}
            onChange={handleFileChange}
            style={{ display: 'none' }}
          />
          <input
            type="file"
            ref={folderInputRef}
            onChange={handleFolderChange}
            style={{ display: 'none' }}
            {...({ webkitdirectory: '', directory: '' } as any)}
          />
        </div>
      </div>

      {error && (
        <div
          className="p-4 bg-red-900/20 text-red-400 border-b border-red-900/50"
          style={{
            padding: '16px',
            backgroundColor: 'rgba(127, 29, 29, 0.2)',
            color: '#f87171',
            borderBottom: '1px solid rgba(127, 29, 29, 0.5)',
          }}
        >
          {error}
        </div>
      )}

      {creatingDir && (
        <div className="new-folder-input-container">
          <Folder size={16} style={{ color: '#818cf8', marginLeft: '8px' }} />
          <input
            type="text"
            value={newDirName}
            onChange={(e) => setNewDirName(e.target.value)}
            placeholder="New folder name..."
            className="form-input"
            style={{ padding: '4px 8px', fontSize: '0.875rem' }}
            autoFocus
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleCreateDir();
              if (e.key === 'Escape') setCreatingDir(false);
            }}
          />
          <button
            onClick={handleCreateDir}
            className="btn btn-primary"
            style={{ padding: '4px 12px', fontSize: '0.75rem' }}
          >
            Create
          </button>
          <button
            onClick={() => setCreatingDir(false)}
            className="btn btn-secondary"
            style={{ padding: '4px 12px', fontSize: '0.75rem' }}
          >
            Cancel
          </button>
        </div>
      )}

      <div className="file-list-container">
        {loading && (!files || files.length === 0) ? (
          <div
            style={{
              display: 'flex',
              justifyContent: 'center',
              alignItems: 'center',
              height: '100%',
              color: '#6b7280',
            }}
          >
            Loading...
          </div>
        ) : (
          <table className="file-table">
            <thead>
              <tr>
                <th style={{ width: '32px' }}></th>
                <th>Name</th>
                <th style={{ width: '128px' }}>Size</th>
                <th style={{ width: '192px' }}>Last Modified</th>
                <th style={{ width: '96px' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {currentPath !== '/' && (
                <tr className="file-row" onClick={handleUp}>
                  <td style={{ textAlign: 'center' }}>
                    <Folder size={16} style={{ color: '#818cf8' }} />
                  </td>
                  <td style={{ color: '#a5b4fc', fontWeight: 500 }}>..</td>
                  <td style={{ color: '#6b7280' }}>-</td>
                  <td style={{ color: '#6b7280' }}>-</td>
                  <td></td>
                </tr>
              )}
              {(files || []).map((file) => (
                <tr
                  key={file.name}
                  className="file-row"
                  onClick={() => handleFileClick(file)}
                >
                  <td style={{ textAlign: 'center' }}>
                    {file.isDirectory ? (
                      <Folder size={16} style={{ color: '#818cf8' }} />
                    ) : (
                      <FileIcon size={16} style={{ color: '#9ca3af' }} />
                    )}
                  </td>
                  <td
                    style={
                      file.isDirectory
                        ? { color: '#a5b4fc', fontWeight: 500 }
                        : { color: '#d1d5db' }
                    }
                  >
                    {file.name}
                  </td>
                  <td style={{ color: '#6b7280' }}>
                    {file.isDirectory ? '-' : formatSize(file.size)}
                  </td>
                  <td style={{ color: '#6b7280' }}>
                    {new Date(file.lastModified).toLocaleString()}
                  </td>
                  <td>
                    <div
                      className="row-actions"
                      onClick={(e) => e.stopPropagation()}
                    >
                      {!file.isDirectory && isEditable(file.name) && (
                        <button
                          onClick={() => handleFileClick(file)}
                          className="file-manage-btn"
                          title="Edit"
                        >
                          <Edit2 size={14} />
                        </button>
                      )}
                      {!file.isDirectory && (
                        <button
                          onClick={() => handleDownload(file)}
                          className="file-manage-btn"
                          title="Download"
                        >
                          <Download size={14} />
                        </button>
                      )}
                      <button
                        onClick={() => handleDelete(file)}
                        className="file-manage-btn delete"
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
                    style={{
                      padding: '32px',
                      textAlign: 'center',
                      color: '#6b7280',
                    }}
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
