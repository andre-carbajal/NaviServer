import { Upload, X } from 'lucide-react';
import React, { useRef, useState } from 'react';
import type { Server } from '../types';
import { Button } from './ui/Button';
import { Modal } from './ui/Modal';
import { Select } from './ui/Select';

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
          <label>Backup File (.zip, .rar)</label>
          <div
            onClick={() => fileInputRef.current?.click()}
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
              type="file"
              ref={fileInputRef}
              onChange={handleFileChange}
              accept=".zip,.rar"
              style={{ display: 'none' }}
            />
          </div>
        </div>

        <Select
          label="Associate with Server (Optional)"
          value={selectedServerId}
          onChange={(e) => setSelectedServerId(e.target.value)}
        >
          <option value="">None (Orphaned - Admin only)</option>
          {servers.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </Select>

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
