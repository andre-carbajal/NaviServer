import { Lock, X } from 'lucide-react';

import React, { useState } from 'react';

import { api } from '../services/api';
import type { User } from '../types';

interface Props {
  user: User;
  onClose: () => void;
}

const ChangePasswordModal: React.FC<Props> = ({ user, onClose }) => {
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    if (!password) {
      setError('Password cannot be empty');
      return;
    }
    if (password !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    setSaving(true);
    setError('');

    try {
      await api.updatePassword(user.id, password);
      onClose();
    } catch (err: any) {
      setError(err.response?.data || 'Failed to update password');
      setSaving(false);
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content" style={{ maxWidth: '400px' }}>
        <div className="modal-header">
          <h2 className="modal-title flex items-center gap-4">
            <Lock size={24} className="text-blue-500" />
            Change Password
          </h2>
          <button className="icon-action" onClick={onClose}>
            <X size={20} />
          </button>
        </div>

        <div className="modal-body">
          <p className="text-sm text-gray-400 mb-4">
            Changing password for <strong>{user.username}</strong>
          </p>

          {error && <div className="error-message">{error}</div>}

          <div className="form-group">
            <label>New Password</label>
            <input
              type="password"
              className="input-field"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Enter new password"
            />
          </div>

          <div className="form-group">
            <label>Confirm Password</label>
            <input
              type="password"
              className="input-field"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              placeholder="Confirm new password"
            />
          </div>
        </div>

        <div className="modal-actions">
          <button
            type="button"
            className="btn btn-secondary"
            onClick={onClose}
            disabled={saving}
          >
            Cancel
          </button>
          <button
            type="button"
            className="btn btn-primary"
            onClick={handleSave}
            disabled={saving}
          >
            {saving ? 'Updating...' : 'Update Password'}
          </button>
        </div>
      </div>
    </div>
  );
};

export default ChangePasswordModal;
