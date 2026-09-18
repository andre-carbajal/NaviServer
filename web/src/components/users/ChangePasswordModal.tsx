import axios from 'axios';
import { Lock } from 'lucide-react';

import React, { useState } from 'react';

import { api } from '../../services/api';
import type { User } from '../../types';
import { Button } from '../ui/Button';
import { Modal } from '../ui/Modal';

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
    } catch (err) {
      if (axios.isAxiosError(err)) {
        setError(err.response?.data || 'Failed to update password');
      } else {
        setError('An unexpected error occurred');
      }
      setSaving(false);
    }
  };

  return (
    <Modal
      isOpen
      onClose={onClose}
      contentClassName="tw:max-w-[400px]!"
      title={
        <span className="tw:flex tw:items-center tw:gap-4">
          <Lock size={24} className="tw:text-blue-500" />
          Change Password
        </span>
      }
    >
      <div>
        <p className="tw:mb-4 tw:text-sm tw:text-gray-400">
          Changing password for <strong>{user.username}</strong>
        </p>

        {error && (
          <div className="tw:mb-6 tw:flex tw:items-center tw:justify-center tw:gap-2 tw:rounded-xl tw:border tw:border-red-600/20 tw:bg-red-600/10 tw:p-4 tw:text-center tw:text-[0.9rem] tw:text-red-400">
            {error}
          </div>
        )}

        <div className="tw:mb-[15px] tw:max-[769px]:mb-3">
          <label
            className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted"
            htmlFor="change-password-new"
          >
            New Password
          </label>
          <input
            id="change-password-new"
            type="password"
            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Enter new password"
          />
        </div>

        <div className="tw:mb-[15px] tw:max-[769px]:mb-3">
          <label
            className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted"
            htmlFor="change-password-confirm"
          >
            Confirm Password
          </label>
          <input
            id="change-password-confirm"
            type="password"
            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            placeholder="Confirm new password"
          />
        </div>
        <div className="tw:mt-[25px] tw:flex tw:justify-end tw:gap-2.5 tw:max-[769px]:flex-col tw:max-[769px]:gap-2 tw:max-[769px]:[&_button]:w-full">
          <Button variant="secondary" onClick={onClose} disabled={saving}>
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={saving}>
            {saving ? 'Updating...' : 'Update Password'}
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ChangePasswordModal;
