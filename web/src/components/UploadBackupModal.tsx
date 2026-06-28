import { Upload, X } from 'lucide-react';

import React, { useRef, useState } from 'react';

import type { Server } from '../types';
import ServerIconSelect from './ServerIconSelect';
import { Button } from './ui/Button';
import { Modal } from './ui/Modal';

interface UploadBackupModalProps {
  isOpen: boolean;
  onClose: () => void;
  onUpload: (file: File, serverId?: string) => void;
  servers: Server[];
  defaultServerId?: string;
}

const UploadBackupModal: React.FC<UploadBackupModalProps> = ({
  isOpen,
  onClose,
  onUpload,
  servers,
  defaultServerId,
}) => {
  const [selectedServerId, setSelectedServerId] = useState<string>(
    defaultServerId || '',
  );
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setSelectedFile(e.target.files[0]);
    }
  };

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (selectedFile) {
      onUpload(selectedFile, selectedServerId || undefined);
      setSelectedFile(null);
      onClose();
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Upload Backup">
      <form onSubmit={handleSubmit} className="modal-form">
        <div className="form-group">
          <label htmlFor="backup-upload-file">Backup File (.zip, .rar)</label>
          <div
            role="button"
            tabIndex={0}
            aria-label="Select backup file"
            onClick={() => fileInputRef.current?.click()}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault();
                fileInputRef.current?.click();
              }
            }}
            className="file-upload-zone"
            style={{
              border: '2px dashed var(--border-color)',
              borderRadius: '8px',
              padding: '20px',
              textAlign: 'center',
              cursor: 'pointer',
              marginBottom: '10px',
            }}
          >
            {selectedFile ? (
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: '10px',
                }}
              >
                <span>{selectedFile.name}</span>
                <Button
                  variant="secondary"
                  aria-label="Remove selected backup file"
                  onClick={(e) => {
                    e.stopPropagation();
                    setSelectedFile(null);
                  }}
                >
                  <X size={14} />
                </Button>
              </div>
            ) : (
              <div
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  gap: '8px',
                  color: 'var(--text-muted)',
                }}
              >
                <Upload size={24} />
                <span>Click to select or drag and drop</span>
              </div>
            )}
            <input
              id="backup-upload-file"
              aria-label="Backup file"
              type="file"
              ref={fileInputRef}
              onChange={handleFileChange}
              accept=".zip,.rar"
              style={{ display: 'none' }}
            />
          </div>
        </div>

        <ServerIconSelect
          label="Associate with Server (Optional)"
          value={selectedServerId}
          onChange={setSelectedServerId}
          servers={servers}
          allowNone={true}
          noneLabel="None (Orphaned - Admin only)"
        />

        <div className="modal-actions">
          <Button variant="secondary" type="button" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={!selectedFile}>
            Upload
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default UploadBackupModal;
