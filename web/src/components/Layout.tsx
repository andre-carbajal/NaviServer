import {
  ArrowUpCircle,
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
  }, []);

  return (
    <div className="layout">
      <header className="mobile-header">
        <div className="brand">
          <img
            src="/apple-touch-icon.png"
            alt="Naviger"
            style={{ width: '24px', height: '24px' }}
          />
          <span>Naviger</span>
        </div>
        <div className="user-info">
          <span className="user-name">{user?.username}</span>
          <span className="user-role-badge">{user?.role}</span>
          <button onClick={logout} className="logout-btn" title="Logout">
            <LogOut size={18} />
          </button>
        </div>
      </header>
      <aside className="sidebar">
        <div className="brand">
          <img
            src="/apple-touch-icon.png"
            alt="Naviger"
            style={{ width: '24px', height: '24px' }}
          />
          <span>Naviger</span>
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
            <span className="user-name">{user?.username}</span>
            <span className="user-role-badge">{user?.role}</span>
            <button onClick={logout} className="logout-btn" title="Logout">
              <LogOut size={18} />
            </button>
          </div>
        </div>
        {updateAvailable && (
          <div className="update-notification">
            <a
              href={releaseUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="update-link"
            >
              <ArrowUpCircle size={20} />
              <span>Update</span>
            </a>
          </div>
        )}
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
