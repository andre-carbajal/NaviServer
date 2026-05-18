import React, { useEffect, useState } from 'react';

import type { Server } from '../types';
import ServerIconSelect from './ServerIconSelect';
import { Button } from './ui/Button';
import { Modal } from './ui/Modal';

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

  useEffect(() => {
    if (isOpen) {
      setSelectedServerId(currentServerId || '');
    }
  }, [currentServerId, isOpen]);

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
        <ServerIconSelect
          label="Associate with Server"
          value={selectedServerId}
          onChange={setSelectedServerId}
          servers={servers}
          allowNone={true}
          noneLabel="None (Orphaned)"
        />

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
