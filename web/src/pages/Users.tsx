import { Key, Lock, Trash2, UserPlus } from 'lucide-react';

import React, { useEffect, useState } from 'react';

import '../App.css';
import ChangePasswordModal from '../components/ChangePasswordModal';
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

  const handleDelete = async (id: string) => {
    if (confirm('Are you sure you want to delete this user?')) {
      try {
        await api.deleteUser(id);
        setUsers(users.filter((u) => u.id !== id));
      } catch (_) {
        alert('Failed to delete user');
      }
    }
  };

  const handleUserCreated = (newUser: User) => {
    setUsers([...users, newUser]);
    setShowCreateModal(false);
  };

  return (
    <div className="users-page">
      <div className="card-header">
        <h1 className="card-title">User Management</h1>
        <button
          className="btn btn-primary"
          onClick={() => setShowCreateModal(true)}
        >
          <UserPlus size={18} />
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
                        onClick={() => handleDelete(user.id)}
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
    </div>
  );
};

export default UsersPage;
