import React, { useEffect, useState } from 'react';

import type { Server } from '../../types';
import { Button } from '../ui/Button';
import { Modal } from '../ui/Modal';
import ServerIconSelect from './ServerIconSelect';

interface CreateBackupModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreate: (serverId: string, name: string) => Promise<void>;
  servers: Server[];
  defaultServerId?: string;
}

const CreateBackupModal: React.FC<CreateBackupModalProps> = ({
  isOpen,
  onClose,
  onCreate,
  servers,
  defaultServerId,
}) => {
  const [name, setName] = useState('');
  const [selectedServer, setSelectedServer] = useState(defaultServerId || '');
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (!isOpen) return;

    const resetTimer = window.setTimeout(() => {
      setName('');
      setSelectedServer(
        defaultServerId || (servers.length > 0 ? servers[0].id : ''),
      );
      setIsSubmitting(false);
    }, 0);

    return () => window.clearTimeout(resetTimer);
  }, [isOpen, defaultServerId, servers]);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!selectedServer) return;

    setIsSubmitting(true);
    try {
      await onCreate(selectedServer, name);
      onClose();
    } catch (error) {
      console.error('Failed to create backup:', error);
    } finally {
      setIsSubmitting(false);
    }
  };

  const showServerSelect = !defaultServerId;

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Create Backup">
      <form onSubmit={handleSubmit}>
        {showServerSelect && (
          <ServerIconSelect
            label="Server"
            value={selectedServer}
            onChange={setSelectedServer}
            servers={servers}
          />
        )}

        <div className="tw:mb-[15px] tw:max-[769px]:mb-3">
          <label
            htmlFor="create-backup-name"
            className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted tw:max-[769px]:mb-1.5 tw:max-[769px]:text-[0.85rem]"
          >
            Backup Name{' '}
            <span className="tw:text-[0.8em] tw:text-text-muted">
              (Optional)
            </span>
          </label>
          <input
            id="create-backup-name"
            type="text"
            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Defaults to timestamp"
          />
        </div>

        <div className="tw:mt-[25px] tw:flex tw:justify-end tw:gap-2.5 tw:max-[769px]:flex-col tw:max-[769px]:gap-2 tw:max-[769px]:[&_button]:w-full">
          <Button
            type="button"
            variant="secondary"
            onClick={onClose}
            disabled={isSubmitting}
          >
            Cancel
          </Button>
          <Button type="submit" disabled={isSubmitting || !selectedServer}>
            {isSubmitting ? 'Creating...' : 'Create'}
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default CreateBackupModal;
