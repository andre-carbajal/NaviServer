import React, { useEffect, useState } from 'react';

import type { Server } from '../../types';
import { Button } from '../ui/Button';
import { Modal } from '../ui/Modal';
import ServerIconSelect from './ServerIconSelect';

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
    if (!isOpen) return;

    const resetTimer = window.setTimeout(() => {
      setSelectedServerId(currentServerId || '');
    }, 0);

    return () => window.clearTimeout(resetTimer);
  }, [currentServerId, isOpen]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onUpdate(selectedServerId);
    onClose();
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Edit Backup Association">
      <form onSubmit={handleSubmit} className="tw:block">
        <p className="tw:mb-[15px] tw:text-text-muted">
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

        <div className="tw:mt-[25px] tw:flex tw:justify-end tw:gap-2.5 tw:max-[769px]:flex-col tw:max-[769px]:gap-2 tw:max-[769px]:[&_button]:w-full">
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
