import { v4 as uuidv4 } from 'uuid';

import React, { useEffect, useState } from 'react';

import { api } from '../services/api.ts';
import { Button } from './ui/Button';
import { Modal } from './ui/Modal';

interface CreateModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreate: (data: {
    name: string;
    loader: string;
    version: string;
    ram: number;
    requestId?: string;
  }) => void;
}

const CreateModal: React.FC<CreateModalProps> = ({
  isOpen,
  onClose,
  onCreate,
}) => {
  const [name, setName] = useState('');
  const [loader, setLoader] = useState('vanilla');
  const [version, setVersion] = useState('1.20.1');
  const [ram, setRam] = useState(2048);
  const [loaders, setLoaders] = useState<string[]>([]);
  const [versions, setVersions] = useState<string[]>([]);

  useEffect(() => {
    if (isOpen) {
      api
        .getLoaders()
        .then((response) => {
          setLoaders(response.data);
          if (response.data.length > 0) {
            setLoader(response.data[0]);
          }
        })
        .catch((error) => {
          console.error('Failed to fetch loaders', error);
        });
    }
  }, [isOpen]);

  useEffect(() => {
    if (loader) {
      api
        .getLoaderVersions(loader)
        .then((response) => {
          setVersions(response.data);
          if (response.data.length > 0) {
            setVersion(response.data[0]);
          }
        })
        .catch((error) => {
          console.error(`Failed to fetch versions for ${loader}`, error);
        });
    }
  }, [loader]);

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const newRequestId = uuidv4();

    onCreate({ name, loader, version, ram, requestId: newRequestId });

    onClose();
    setName('');
    setLoader(loaders.length > 0 ? loaders[0] : 'vanilla');
    setVersion('1.20.1');
    setRam(2048);
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Create New Server">
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label>Server Name</label>
          <input
            type="text"
            className="form-input"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="My Survival World"
            required
          />
        </div>

        <div
          style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            gap: '15px',
          }}
        >
          <div className="form-group">
            <label>Loader</label>
            <select
              className="form-select"
              value={loader}
              onChange={(e) => setLoader(e.target.value)}
            >
              {loaders.map((l) => (
                <option key={l} value={l}>
                  {l.charAt(0).toUpperCase() + l.slice(1)}
                </option>
              ))}
            </select>
          </div>
          <div className="form-group">
            <label>Version</label>
            <select
              className="form-select"
              value={version}
              onChange={(e) => setVersion(e.target.value)}
            >
              {versions.map((v) => (
                <option key={v} value={v}>
                  {v}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className="form-group">
          <label>RAM (MB)</label>
          <input
            type="number"
            className="form-input"
            value={ram}
            onChange={(e) => setRam(Number(e.target.value))}
            min="1024"
            step="512"
          />
        </div>

        <div className="modal-actions">
          <Button type="button" variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit">Create Server</Button>
        </div>
      </form>
    </Modal>
  );
};

export default CreateModal;
