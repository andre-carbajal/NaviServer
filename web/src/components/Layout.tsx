import {
  AlertTriangle,
  DatabaseBackup,
  LayoutDashboard,
  LogOut,
  Settings,
  Users,
} from 'lucide-react';
import { NavLink, Outlet } from 'react-router-dom';

import React, { useEffect, useState } from 'react';

import '../App.css';
import { useAuth } from '../context/AuthContext';
import { api } from '../services/api';

const Layout: React.FC = () => {
  const [updateAvailable, setUpdateAvailable] = useState(false);
  const [releaseUrl, setReleaseUrl] = useState('');
  const [version, setVersion] = useState('');
  const { user, logout } = useAuth();

  useEffect(() => {
    api
      .checkUpdates()
      .then((response) => {
        if (response.data.update_available) {
          setUpdateAvailable(true);
          setReleaseUrl(response.data.release_url);
        }
      })
      .catch(console.error);

    api
      .getVersion()
      .then((response) => {
        setVersion(response.data.version);
      })
      .catch(console.error);
  }, []);

  return (
    <div className="layout">
      <header className="mobile-header">
        <div className="brand">
          <img
            src="/apple-touch-icon.png"
            alt="NaviServer"
            style={{ width: '24px', height: '24px' }}
          />
          <span>NaviServer</span>
        </div>
        <div className="user-info">
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              marginRight: '8px',
            }}
          >
            <span
              className="mobile-version"
              style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}
            >
              {version}
            </span>
            {updateAvailable && (
              <a
                href={releaseUrl}
                target="_blank"
                rel="noopener noreferrer"
                title="Update Available"
                style={{
                  fontSize: '0.75rem',
                  color: '#fbbf24',
                  textDecoration: 'none',
                  backgroundColor: 'rgba(251, 191, 36, 0.1)',
                  padding: '2px 6px',
                  borderRadius: '4px',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '4px',
                  fontWeight: 600,
                  cursor: 'pointer',
                }}
              >
                <AlertTriangle size={12} />
              </a>
            )}
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span className="user-name">{user?.username}</span>
            <span className="user-role-badge">{user?.role}</span>
          </div>
          <button onClick={logout} className="logout-btn" title="Logout">
            <LogOut size={18} />
          </button>
        </div>
      </header>
      <aside className="sidebar">
        <div className="brand">
           <img
             src="/apple-touch-icon.png"
             alt="NaviServer"
             style={{ width: '24px', height: '24px' }}
           />
           <span>NaviServer</span>
         </div>
         <nav>
          <NavLink
            to="/"
            className={({ isActive }) =>
              isActive ? 'nav-item active' : 'nav-item'
            }
          >
            <LayoutDashboard size={20} />
            <span>Dashboard</span>
          </NavLink>
          <NavLink
            to="/servers/backups/all"
            className={({ isActive }) =>
              isActive ? 'nav-item active' : 'nav-item'
            }
          >
            <DatabaseBackup size={20} />
            <span>Backups</span>
          </NavLink>
          {user?.role === 'admin' && (
            <NavLink
              to="/users"
              className={({ isActive }) =>
                isActive ? 'nav-item active' : 'nav-item'
              }
            >
              <Users size={20} />
              <span>Users</span>
            </NavLink>
          )}
          {user?.role === 'admin' && (
            <NavLink
              to="/settings"
              className={({ isActive }) =>
                isActive ? 'nav-item active' : 'nav-item'
              }
            >
              <Settings size={20} />
              <span>Settings</span>
            </NavLink>
          )}
        </nav>
        <div className="sidebar-footer">
          <div className="user-info">
            <div
              style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}
            >
              <div
                style={{ display: 'flex', alignItems: 'center', gap: '8px' }}
              >
                <span className="user-name">{user?.username}</span>
                <span className="user-role-badge">{user?.role}</span>
              </div>
              <div
                className="version-info"
                style={{
                  fontSize: '0.8rem',
                  color: 'var(--text-muted)',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '6px',
                }}
              >
                {version}
                {updateAvailable && (
                  <a
                    href={releaseUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    title="Update Available"
                    style={{
                      fontSize: '0.75rem',
                      color: '#fbbf24',
                      textDecoration: 'none',
                      backgroundColor: 'rgba(251, 191, 36, 0.1)',
                      padding: '2px 6px',
                      borderRadius: '4px',
                      display: 'flex',
                      alignItems: 'center',
                      gap: '4px',
                      marginLeft: '4px',
                      fontWeight: 600,
                      cursor: 'pointer',
                    }}
                  >
                    <AlertTriangle size={12} />
                    Update
                  </a>
                )}
              </div>
            </div>
            <button onClick={logout} className="logout-btn" title="Logout">
              <LogOut size={18} />
            </button>
          </div>
        </div>
      </aside>
      <main className="content">
        <div className="page-content">
          <Outlet />
        </div>
      </main>
    </div>
  );
};

export default Layout;
