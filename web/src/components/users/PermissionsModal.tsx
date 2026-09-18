import { Key } from 'lucide-react';

import React, { useEffect, useState } from 'react';

import { api } from '../../services/api';
import type { Permission, Server, User } from '../../types';
import { Button } from '../ui/Button';
import { Modal } from '../ui/Modal';

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
    <Modal
      isOpen
      onClose={onClose}
      contentClassName="tw:max-w-[700px]!"
      title={
        <span className="tw:flex tw:items-center tw:gap-4">
          <Key size={24} className="tw:text-blue-500" />
          Permissions for {user.username}
        </span>
      }
    >
      {error && (
        <div className="tw:mb-6 tw:flex tw:items-center tw:justify-center tw:gap-2 tw:rounded-xl tw:border tw:border-red-600/20 tw:bg-red-600/10 tw:p-4 tw:text-center tw:text-[0.9rem] tw:text-red-400">
          {error}
        </div>
      )}

      {loading ? (
        <div>Loading...</div>
      ) : (
        <div className="tw:max-h-[60vh] tw:overflow-y-auto">
          <table className="tw:w-full tw:border-collapse tw:text-left tw:[&_th]:border-b tw:[&_th]:border-border tw:[&_th]:bg-black/10 tw:[&_th]:px-4 tw:[&_th]:py-3 tw:[&_th]:text-xs tw:[&_th]:font-semibold tw:[&_th]:text-text-muted tw:[&_th]:uppercase tw:[&_td]:border-b tw:[&_td]:border-border tw:[&_td]:px-4 tw:[&_td]:py-3 tw:[&_tbody_tr]:transition-colors tw:[&_tbody_tr:hover]:bg-white/3">
            <thead>
              <tr>
                <th>Server</th>
                <th className="tw:text-center">Power Control</th>
                <th className="tw:text-center">Console & Files</th>
              </tr>
            </thead>
            <tbody>
              {servers.map((server) => {
                const perm = permissions[server.id] || {};
                return (
                  <tr key={server.id}>
                    <td>{server.name}</td>
                    <td className="tw:text-center">
                      <input
                        type="checkbox"
                        className="tw:grid tw:h-5 tw:w-5 tw:shrink-0 tw:cursor-pointer tw:appearance-none tw:place-content-center tw:rounded tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:p-0 tw:before:block tw:before:h-[0.65em] tw:before:w-[0.65em] tw:before:scale-0 tw:before:origin-center tw:before:content-['✓'] tw:before:text-xs tw:before:font-bold tw:before:text-white tw:checked:border-blue-500 tw:checked:bg-blue-500 tw:checked:before:scale-100 tw:disabled:cursor-not-allowed tw:disabled:opacity-60"
                        aria-label={`${server.name} power control permission`}
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
                    <td className="tw:text-center">
                      <input
                        type="checkbox"
                        className="tw:grid tw:h-5 tw:w-5 tw:shrink-0 tw:cursor-pointer tw:appearance-none tw:place-content-center tw:rounded tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:p-0 tw:before:block tw:before:h-[0.65em] tw:before:w-[0.65em] tw:before:scale-0 tw:before:origin-center tw:before:content-['✓'] tw:before:text-xs tw:before:font-bold tw:before:text-white tw:checked:border-blue-500 tw:checked:bg-blue-500 tw:checked:before:scale-100 tw:disabled:cursor-not-allowed tw:disabled:opacity-60"
                        aria-label={`${server.name} console and files permission`}
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

      <div className="tw:mt-[25px] tw:flex tw:justify-end tw:gap-2.5 tw:max-[769px]:flex-col tw:max-[769px]:gap-2 tw:max-[769px]:[&_button]:w-full">
        <Button variant="secondary" onClick={onClose} disabled={saving}>
          Cancel
        </Button>
        <Button onClick={handleSave} disabled={saving}>
          {saving ? 'Saving...' : 'Save Permissions'}
        </Button>
      </div>
    </Modal>
  );
};

export default PermissionsModal;
