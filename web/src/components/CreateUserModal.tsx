import { UserPlus, X } from 'lucide-react';

import React, { useState } from 'react';

import { api } from '../services/api';
import type { User } from '../types';

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

  const handleSubmit = async (e: React.FormEvent) => {
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
    } catch (err: any) {
      setError(err.response?.data || 'Failed to create user');
      setLoading(false);
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <div className="modal-header">
          <h2 className="modal-title flex items-center gap-4">
            <UserPlus size={24} className="text-blue-500" />
            Create New User
          </h2>
          <button className="icon-action" onClick={onClose}>
            <X size={20} />
          </button>
        </div>

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
            <button
              type="button"
              className="btn btn-secondary"
              onClick={onClose}
              disabled={loading}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="btn btn-primary"
              disabled={loading || !!usernameError}
            >
              {loading ? 'Creating...' : 'Create User'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default CreateUserModal;
