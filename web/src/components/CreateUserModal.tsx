import axios from 'axios';

import React, { useState } from 'react';

import { api } from '../services/api';
import type { User } from '../types';
import { Button } from './ui/Button';
import { Modal } from './ui/Modal';

interface Props {
  onClose: () => void;
  onCreated: (user: User) => void;
}

const CreateUserModal: React.FC<Props> = ({ onClose, onCreated }) => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [usernameError, setUsernameError] = useState('');
  const [loading, setLoading] = useState(false);

  React.useEffect(() => {
    if (username.includes(' ')) {
      setUsernameError('Username cannot contain spaces');
    } else {
      setUsernameError('');
    }
  }, [username]);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError('');

    if (username.includes(' ')) {
      setError('Username cannot contain spaces');
      return;
    }

    setLoading(true);

    try {
      const response = await api.createUser({ username, password });
      onCreated(response.data);
    } catch (err) {
      if (axios.isAxiosError(err)) {
        setError(err.response?.data || 'Failed to create user');
      } else {
        setError('An unexpected error occurred');
      }
      setLoading(false);
    }
  };

  return (
    <Modal isOpen={true} onClose={onClose} title="Create New User">
      <div style={{ padding: '5px 0' }}>
        {error && <div className="error-message">{error}</div>}

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Username</label>
            <input
              type="text"
              className="form-input"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
            />
            {usernameError && (
              <div className="error-message" style={{ marginTop: '5px' }}>
                {usernameError}
              </div>
            )}
          </div>
          <div className="form-group">
            <label>Password</label>
            <input
              type="password"
              className="form-input"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </div>

          <div className="modal-actions">
            <Button
              type="button"
              variant="secondary"
              onClick={onClose}
              disabled={loading}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={loading || !!usernameError}>
              {loading ? 'Creating...' : 'Create User'}
            </Button>
          </div>
        </form>
      </div>
    </Modal>
  );
};

export default CreateUserModal;
