import { Upload, X } from 'lucide-react';

import React, { useRef, useState } from 'react';

import type { Server } from '../../types';
import { Button } from '../ui/Button';
import { Modal } from '../ui/Modal';
import ServerIconSelect from './ServerIconSelect';

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
    const file = e.target.files?.[0];
    if (file) {
      setSelectedFile(file);
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
      <form onSubmit={handleSubmit} className="tw:block">
        <div className="tw:mb-[15px] tw:max-[769px]:mb-3">
          <label
            htmlFor="backup-upload-file"
            className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted tw:max-[769px]:mb-1.5 tw:max-[769px]:text-[0.85rem]"
          >
            Backup File (.zip, .rar)
          </label>
          <label
            htmlFor="backup-upload-file"
            className="tw:mb-2.5 tw:cursor-pointer tw:rounded-lg tw:border-2 tw:border-dashed tw:border-border tw:p-5 tw:text-center"
          >
            {selectedFile ? (
              <div className="tw:flex tw:items-center tw:justify-center tw:gap-2.5">
                <span>{selectedFile.name}</span>
                <Button
                  variant="secondary"
                  aria-label="Remove selected backup file"
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    setSelectedFile(null);
                  }}
                >
                  <X size={14} />
                </Button>
              </div>
            ) : (
              <div className="tw:flex tw:flex-col tw:items-center tw:gap-2 tw:text-text-muted">
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
              className="tw:hidden"
            />
          </label>
        </div>

        <ServerIconSelect
          label="Associate with Server (Optional)"
          value={selectedServerId}
          onChange={setSelectedServerId}
          servers={servers}
          allowNone={true}
          noneLabel="None (Orphaned - Admin only)"
        />

        <div className="tw:mt-[25px] tw:flex tw:justify-end tw:gap-2.5 tw:max-[769px]:flex-col tw:max-[769px]:gap-2 tw:max-[769px]:[&_button]:w-full">
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
