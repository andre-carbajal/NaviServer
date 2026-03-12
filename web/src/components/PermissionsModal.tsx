import { Key, X } from 'lucide-react';

import React, { useEffect, useState } from 'react';

import { api } from '../services/api';
import type { Permission, Server, User } from '../types';

interface Props {
  user: User;
  onClose: () => void;
}

const PermissionsModal: React.FC<Props> = ({ user, onClose }) => {
  const [servers, setServers] = useState<Server[]>([]);
  const [permissions, setPermissions] = useState<Record<string, Permission>>(
    {},
  );
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [serversRes, permsRes] = await Promise.all([
          api.getServers(),
          api.getPermissions(user.id),
        ]);

        setServers(serversRes.data);

        const permsMap: Record<string, Permission> = {};
        (permsRes.data || []).forEach((p: Permission) => {
          permsMap[p.serverId] = p;
        });
        setPermissions(permsMap);

        setLoading(false);
      } catch {
        setError('Failed to fetch data');
        setLoading(false);
      }
    };
    fetchData();
  }, [user.id]);

  const handleCheck = (
    serverId: string,
    field: keyof Permission,
    checked: boolean,
  ) => {
    setPermissions((prev) => {
      const current = prev[serverId] || {
        userId: user.id,
        serverId,
        canViewConsole: false,
        canControlPower: false,
      };

      const updated = { ...current, [field]: checked };

      if (field === 'canViewConsole' && checked) {
        updated.canControlPower = true;
      }
      if (field === 'canControlPower' && !checked) {
        updated.canViewConsole = false;
      }

      return {
        ...prev,
        [serverId]: updated,
      };
    });
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      const permsArray = Object.values(permissions);
      await api.updatePermissions(permsArray);
      onClose();
    } catch {
      setError('Failed to save permissions');
      setSaving(false);
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content" style={{ maxWidth: '700px' }}>
        <div className="modal-header">
          <h2 className="modal-title flex items-center gap-4">
            <Key size={24} className="text-blue-500" />
            Permissions for {user.username}
          </h2>
          <button className="icon-action" onClick={onClose}>
            <X size={20} />
          </button>
        </div>

        {error && <div className="error-message">{error}</div>}

        {loading ? (
          <div>Loading...</div>
        ) : (
          <div style={{ maxHeight: '60vh', overflowY: 'auto' }}>
            <table className="data-table">
              <thead>
                <tr>
                  <th>Server</th>
                  <th className="text-center">Power Control</th>
                  <th className="text-center">Console & Files</th>
                </tr>
              </thead>
              <tbody>
                {servers.map((server) => {
                  const perm = permissions[server.id] || {};
                  return (
                    <tr key={server.id}>
                      <td>{server.name}</td>
                      <td className="text-center">
                        <input
                          type="checkbox"
                          checked={perm.canControlPower || false}
                          onChange={(e) =>
                            handleCheck(
                              server.id,
                              'canControlPower',
                              e.target.checked,
                            )
                          }
                        />
                      </td>
                      <td className="text-center">
                        <input
                          type="checkbox"
                          checked={perm.canViewConsole || false}
                          onChange={(e) =>
                            handleCheck(
                              server.id,
                              'canViewConsole',
                              e.target.checked,
                            )
                          }
                        />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

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
            {saving ? 'Saving...' : 'Save Permissions'}
          </button>
        </div>
      </div>
    </div>
  );
};

export default PermissionsModal;
