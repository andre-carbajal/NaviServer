import { Key, Lock, Trash2, UserPlus } from 'lucide-react';

import React, { useEffect, useState } from 'react';

import '../App.css';
import ChangePasswordModal from '../components/ChangePasswordModal';
import ConfirmationModal from '../components/ConfirmationModal';
import CreateUserModal from '../components/CreateUserModal';
import PermissionsModal from '../components/PermissionsModal';
import { useAuth } from '../context/AuthContext';
import { api } from '../services/api';
import type { User } from '../types';

const UsersPage: React.FC = () => {
  const { user: currentUser } = useAuth();
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

  const fetchUsers = async () => {
    try {
      const response = await api.listUsers();
      setUsers(response.data);
      setLoading(false);
    } catch (_) {
      setError('Failed to fetch users');
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, []);

  const handleDelete = (user: User) => {
    setUserToDelete(user);
  };

  const confirmDelete = async () => {
    if (userToDelete) {
      try {
        await api.deleteUser(userToDelete.id);
        setUsers(users.filter((u) => u.id !== userToDelete.id));
      } catch (_) {
        alert('Failed to delete user');
      }
      setUserToDelete(null);
    }
  };

  const handleUserCreated = (newUser: User) => {
    setUsers([...users, newUser]);
    setShowCreateModal(false);
  };

  return (
    <div className="users-page">
      <div className="modal-header">
        <h1>User Management</h1>
        <button
          className="btn btn-primary"
          onClick={() => setShowCreateModal(true)}
        >
          <UserPlus size={20} />
          <span>Create User</span>
        </button>
      </div>

      {error && <div className="error-message">{error}</div>}

      {loading ? (
        <div>Loading...</div>
      ) : (
        <div className="card">
          <table className="data-table">
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
                    <span
                      className="status-badge status-running"
                      style={{
                        backgroundColor:
                          user.role === 'admin' ? '#f59e0b' : '#3b82f6',
                      }}
                    >
                      {user.role}
                    </span>
                  </td>
                  <td>
                    <div
                      className="actions-group"
                      style={{ border: 'none', padding: 0, margin: 0 }}
                    >
                      {user.role !== 'admin' && (
                        <>
                          <button
                            className="icon-action"
                            title="Permissions"
                            onClick={() => setEditingPermissionsUser(user)}
                          >
                            <Key size={18} />
                          </button>
                          <button
                            className="icon-action"
                            title="Change Password"
                            onClick={() => setChangingPasswordUser(user)}
                          >
                            <Lock size={18} />
                          </button>
                        </>
                      )}
                      <button
                        className="icon-action danger"
                        title="Delete"
                        onClick={() => handleDelete(user)}
                        disabled={currentUser?.id === user.id}
                        style={{
                          opacity: currentUser?.id === user.id ? 0.5 : 1,
                          cursor:
                            currentUser?.id === user.id
                              ? 'not-allowed'
                              : 'pointer',
                        }}
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
