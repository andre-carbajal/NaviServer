import React, { useState } from 'react';

import type { Server } from '../types';
import { Button } from './ui/Button';
import { Modal } from './ui/Modal';
import { Select } from './ui/Select';

interface EditBackupModalProps {
  isOpen: boolean;
  onClose: () => void;
  onUpdate: (serverId: string) => void;
  servers: Server[];
  currentServerId?: string;
  backupName: string;
}

const EditBackupModal: React.FC<EditBackupModalProps> = ({
  isOpen,
  onClose,
  onUpdate,
  servers,
  currentServerId,
  backupName,
}) => {
  const [selectedServerId, setSelectedServerId] = useState<string>(
    currentServerId || '',
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onUpdate(selectedServerId);
    onClose();
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Edit Backup Association">
      <form onSubmit={handleSubmit} className="modal-form">
        <p style={{ marginBottom: '15px', color: 'var(--text-muted)' }}>
          Changing the associated server for: <strong>{backupName}</strong>
        </p>
        <Select
          label="Associate with Server"
          value={selectedServerId}
          onChange={(e) => setSelectedServerId(e.target.value)}
        >
          <option value="">None (Orphaned)</option>
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
          <Button type="submit">Save Changes</Button>
        </div>
      </form>
    </Modal>
  );
};

export default EditBackupModal;
