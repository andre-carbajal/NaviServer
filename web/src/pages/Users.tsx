import { Key, Lock, Trash2, UserPlus } from 'lucide-react';

import React, { useCallback, useEffect, useState } from 'react';

import ConfirmationModal from '../components/ConfirmationModal';
import { Button } from '../components/ui/Button';
import ChangePasswordModal from '../components/users/ChangePasswordModal';
import CreateUserModal from '../components/users/CreateUserModal';
import PermissionsModal from '../components/users/PermissionsModal';
import { useAuth } from '../context/AuthContext';
import { useModalDialog } from '../hooks/useModalDialog';
import { api } from '../services/api';
import type { User } from '../types';

const UsersPage: React.FC = () => {
  const { user: currentUser } = useAuth();
  const { showAlert, modalDialog } = useModalDialog();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingPermissionsUser, setEditingPermissionsUser] =
    useState<User | null>(null);
  const [changingPasswordUser, setChangingPasswordUser] = useState<User | null>(
    null,
  );
  const [userToDelete, setUserToDelete] = useState<User | null>(null);

  const fetchUsers = useCallback(async () => {
    try {
      const response = await api.listUsers();
      setUsers(response.data);
      setLoading(false);
    } catch {
      setError('Failed to fetch users');
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    fetchUsers();
  }, [fetchUsers]);

  const handleDelete = (user: User) => {
    setUserToDelete(user);
  };

  const confirmDelete = async () => {
    if (userToDelete) {
      try {
        await api.deleteUser(userToDelete.id);
        setUsers(users.filter((u) => u.id !== userToDelete.id));
      } catch {
        await showAlert({
          title: 'Delete Failed',
          message: 'Failed to delete user.',
          variant: 'danger',
        });
      }
      setUserToDelete(null);
    }
  };

  const handleUserCreated = (newUser: User) => {
    setUsers([...users, newUser]);
    setShowCreateModal(false);
  };

  return (
    <div className="tw:flex tw:flex-col tw:gap-4">
      {modalDialog}
      <div className="tw:mb-5 tw:flex tw:items-center tw:justify-between tw:gap-3">
        <h1 className="tw:m-0">User Management</h1>
        <Button onClick={() => setShowCreateModal(true)}>
          <UserPlus size={20} />
          <span>Create User</span>
        </Button>
      </div>

      {error && (
        <div className="tw:mb-6 tw:flex tw:items-center tw:justify-center tw:gap-2 tw:rounded-xl tw:border tw:border-red-500/20 tw:bg-red-500/10 tw:p-4 tw:text-center tw:text-[0.9rem] tw:text-red-400">
          {error}
        </div>
      )}

      {loading ? (
        <div>Loading...</div>
      ) : (
        <div className="tw:rounded-xl tw:border tw:border-border tw:bg-bg-card tw:p-5">
          <table className="tw:box-border tw:mt-2.5 tw:w-full tw:border-collapse tw:text-[0.95rem] tw:[&_th]:border-b tw:[&_th]:border-border tw:[&_th]:p-4 tw:[&_th]:text-left tw:[&_th]:text-[0.8rem] tw:[&_th]:font-semibold tw:[&_th]:tracking-[0.05em] tw:[&_th]:text-text-muted tw:[&_th]:uppercase tw:[&_td]:border-b tw:[&_td]:border-border tw:[&_td]:p-4 tw:[&_td]:text-left tw:[&_tr:last-child_td]:border-b-0 tw:[&_tbody_tr]:transition-colors tw:[&_tbody_tr:hover]:bg-white/[0.03]">
            <thead>
              <tr>
                <th>Username</th>
                <th>Role</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {users.map((user) => (
                <tr key={user.id}>
                  <td>{user.username}</td>
                  <td>
                    <span className="tw:rounded-sm tw:bg-primary/10 tw:px-1.5 tw:py-0.5 tw:text-[0.7rem] tw:font-semibold tw:text-primary tw:uppercase">
                      {user.role}
                    </span>
                  </td>
                  <td>
                    <div className="tw:flex tw:flex-wrap tw:items-center tw:gap-3">
                      {user.role !== 'admin' && (
                        <>
                          <button
                            type="button"
                            className="tw:flex tw:h-9 tw:w-9 tw:items-center tw:justify-center tw:rounded tw:border-0 tw:bg-transparent tw:p-0 tw:text-text-muted tw:transition-all tw:duration-200 tw:hover:bg-white/10 tw:hover:text-white"
                            title="Permissions"
                            onClick={() => setEditingPermissionsUser(user)}
                          >
                            <Key size={18} />
                          </button>
                          <button
                            type="button"
                            className="tw:flex tw:h-9 tw:w-9 tw:items-center tw:justify-center tw:rounded tw:border-0 tw:bg-transparent tw:p-0 tw:text-text-muted tw:transition-all tw:duration-200 tw:hover:bg-white/10 tw:hover:text-white"
                            title="Change Password"
                            onClick={() => setChangingPasswordUser(user)}
                          >
                            <Lock size={18} />
                          </button>
                        </>
                      )}
                      <button
                        type="button"
                        className={`tw:flex tw:h-9 tw:w-9 tw:items-center tw:justify-center tw:rounded tw:border-0 tw:bg-transparent tw:p-0 tw:text-text-muted tw:transition-all tw:duration-200 tw:hover:bg-red-500/10 tw:hover:text-red-500 ${currentUser?.id === user.id ? 'tw:cursor-not-allowed tw:opacity-50' : 'tw:cursor-pointer'}`}
                        title="Delete"
                        onClick={() => handleDelete(user)}
                        disabled={currentUser?.id === user.id}
                      >
                        <Trash2 size={18} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showCreateModal && (
        <CreateUserModal
          onClose={() => setShowCreateModal(false)}
          onCreated={handleUserCreated}
        />
      )}

      {editingPermissionsUser && (
        <PermissionsModal
          user={editingPermissionsUser}
          onClose={() => setEditingPermissionsUser(null)}
        />
      )}

      {changingPasswordUser && (
        <ChangePasswordModal
          user={changingPasswordUser}
          onClose={() => setChangingPasswordUser(null)}
        />
      )}

      {userToDelete && (
        <ConfirmationModal
          isOpen={!!userToDelete}
          onClose={() => setUserToDelete(null)}
          onConfirm={confirmDelete}
          title="Delete User"
          message={`Are you sure you want to delete the user "${userToDelete.username}"? This action cannot be undone.`}
          confirmText="Delete User"
          isDangerous={true}
        />
      )}
    </div>
  );
};

export default UsersPage;
