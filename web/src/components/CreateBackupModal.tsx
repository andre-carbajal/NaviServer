import React, { useEffect, useState } from 'react';

import type { Server } from '../types';
import { Button } from './ui/Button';
import { Modal } from './ui/Modal';

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
    if (isOpen) {
      setName('');
      setSelectedServer(
        defaultServerId || (servers.length > 0 ? servers[0].id : ''),
      );
      setIsSubmitting(false);
    }
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
          <div className="form-group">
            <label>Server</label>
            <select
              className="form-select"
              value={selectedServer}
              onChange={(e) => setSelectedServer(e.target.value)}
              required
            >
              <option value="" disabled>
                Select a server
              </option>
              {servers.map((server) => (
                <option key={server.id} value={server.id}>
                  {server.name} ({server.id})
                </option>
              ))}
            </select>
          </div>
        )}

        <div className="form-group">
          <label>
            Backup Name{' '}
            <span style={{ color: 'var(--text-muted)', fontSize: '0.8em' }}>
              (Optional)
            </span>
          </label>
          <input
            type="text"
            className="form-input"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Defaults to timestamp"
          />
        </div>

        <div className="modal-actions">
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
