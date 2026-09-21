import {
  AlertTriangle,
  DatabaseBackup,
  LayoutDashboard,
  LogOut,
  Menu,
  Settings,
  Users,
  X,
} from 'lucide-react';
import { NavLink, Outlet } from 'react-router-dom';

import React, { useEffect, useState } from 'react';

import { useAuth } from '../context/AuthContext';
import { api } from '../services/api';

const Layout: React.FC = () => {
  const [updateAvailable, setUpdateAvailable] = useState(false);
  const [releaseUrl, setReleaseUrl] = useState('');
  const [version, setVersion] = useState('');
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const { user, logout } = useAuth();

  const closeMobileMenu = () => setIsMobileMenuOpen(false);

  useEffect(() => {
    if (!isMobileMenuOpen) return;

    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        closeMobileMenu();
      }
    };

    document.addEventListener('keydown', closeOnEscape);
    return () => document.removeEventListener('keydown', closeOnEscape);
  }, [isMobileMenuOpen]);

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
    <div className="tw:box-border tw:flex tw:h-full tw:w-full tw:min-w-0 tw:flex-col tw:pt-[50px] tw:min-[769px]:flex-row tw:min-[769px]:p-0">
      <header className="tw:fixed tw:inset-x-0 tw:top-0 tw:z-[100] tw:flex tw:h-[50px] tw:min-w-0 tw:items-center tw:gap-2 tw:border-b tw:border-border tw:bg-bg-sidebar tw:px-3 tw:min-[769px]:hidden">
        <button
          type="button"
          className="tw:flex tw:h-9 tw:w-9 tw:shrink-0 tw:cursor-pointer tw:items-center tw:justify-center tw:rounded-md tw:border-0 tw:bg-transparent tw:p-1.5 tw:text-text-muted tw:transition-all tw:duration-200 tw:hover:bg-white/10 tw:hover:text-text-main"
          onClick={() => setIsMobileMenuOpen((open) => !open)}
          aria-controls="mobile-navigation"
          aria-expanded={isMobileMenuOpen}
          aria-label={isMobileMenuOpen ? 'Close navigation' : 'Open navigation'}
        >
          {isMobileMenuOpen ? <X size={20} /> : <Menu size={20} />}
        </button>
        <div className="tw:flex tw:min-w-0 tw:flex-1 tw:items-center tw:gap-2 tw:border-0 tw:p-0 tw:text-[1.1rem] tw:font-bold tw:text-primary tw:max-[481px]:text-base tw:max-[320px]:[&>span]:hidden">
          <img
            src="/apple-touch-icon.png"
            alt="NaviServer"
            className="tw:h-6 tw:w-6 tw:shrink-0 tw:max-[481px]:h-5 tw:max-[481px]:w-5"
          />
          <span className="tw:truncate">NaviServer</span>
        </div>
        <div className="tw:flex tw:min-w-0 tw:items-center tw:gap-2 tw:text-text-muted tw:max-[320px]:max-w-[72px]">
          <span className="tw:min-w-0 tw:overflow-hidden tw:text-[0.9rem] tw:font-medium tw:text-ellipsis tw:whitespace-nowrap tw:text-text-main">
            {user?.username}
          </span>
          <button
            type="button"
            className="tw:ml-auto tw:flex tw:cursor-pointer tw:items-center tw:justify-center tw:rounded-md tw:border-0 tw:bg-transparent tw:p-1.5 tw:text-text-muted tw:transition-all tw:duration-200 tw:hover:bg-white/10 tw:hover:text-danger"
            title="Logout"
            onClick={logout}
          >
            <LogOut size={18} />
          </button>
        </div>
      </header>
      {isMobileMenuOpen && (
        <button
          type="button"
          className="tw:fixed tw:inset-0 tw:z-[105] tw:cursor-default tw:border-0 tw:bg-black/50 tw:min-[769px]:hidden"
          aria-label="Close navigation"
          onClick={closeMobileMenu}
        />
      )}
      <aside
        id="mobile-navigation"
        className={`tw:fixed tw:inset-y-0 tw:left-0 tw:z-[110] tw:box-border tw:flex tw:h-full tw:w-[240px] tw:max-w-[calc(100vw-24px)] tw:-translate-x-full tw:flex-col tw:overflow-x-hidden tw:overflow-y-auto tw:border-r tw:border-border tw:bg-bg-sidebar tw:shadow-[10px_0_30px_rgba(0,0,0,0.3)] tw:transition-transform tw:duration-200 tw:min-[769px]:static tw:min-[769px]:z-auto tw:min-[769px]:h-auto tw:min-[769px]:w-[250px] tw:min-[769px]:max-w-none tw:min-[769px]:translate-x-0 tw:min-[769px]:overflow-visible tw:min-[769px]:border-r ${isMobileMenuOpen ? 'tw:translate-x-0' : ''}`}
      >
        <div className="tw:flex tw:h-[60px] tw:shrink-0 tw:items-center tw:gap-3 tw:border-b tw:border-border tw:px-4 tw:text-[1.2rem] tw:font-bold tw:text-primary tw:min-[769px]:px-5">
          <img
            src="/apple-touch-icon.png"
            alt="NaviServer"
            className="tw:h-6 tw:w-6"
          />
          <span>NaviServer</span>
          <button
            type="button"
            className="tw:ml-auto tw:flex tw:cursor-pointer tw:items-center tw:justify-center tw:rounded-md tw:border-0 tw:bg-transparent tw:p-1.5 tw:text-text-muted tw:hover:bg-white/10 tw:hover:text-text-main tw:min-[769px]:hidden"
            aria-label="Close navigation"
            onClick={closeMobileMenu}
          >
            <X size={18} />
          </button>
        </div>
        <nav
          className="tw:box-border tw:flex tw:w-full tw:flex-col tw:gap-1 tw:p-2.5 tw:min-[769px]:h-auto tw:min-[769px]:gap-[5px] tw:min-[769px]:px-2.5 tw:min-[769px]:py-5"
          aria-label="Primary navigation"
        >
          <NavLink
            to="/"
            onClick={closeMobileMenu}
            className={({ isActive }) =>
              `tw:flex tw:items-center tw:justify-start tw:gap-3 tw:rounded-lg tw:px-4 tw:py-3 tw:text-text-muted tw:no-underline tw:transition-all tw:duration-200 tw:hover:bg-primary/10 tw:hover:text-white ${isActive ? 'tw:bg-primary tw:text-white' : ''}`
            }
          >
            <LayoutDashboard size={20} />
            <span>Dashboard</span>
          </NavLink>
          <NavLink
            to="/servers/backups/all"
            onClick={closeMobileMenu}
            className={({ isActive }) =>
              `tw:flex tw:items-center tw:justify-start tw:gap-3 tw:rounded-lg tw:px-4 tw:py-3 tw:text-text-muted tw:no-underline tw:transition-all tw:duration-200 tw:hover:bg-primary/10 tw:hover:text-white ${isActive ? 'tw:bg-primary tw:text-white' : ''}`
            }
          >
            <DatabaseBackup size={20} />
            <span>Backups</span>
          </NavLink>
          {user?.role === 'admin' && (
            <NavLink
              to="/users"
              onClick={closeMobileMenu}
              className={({ isActive }) =>
                `tw:flex tw:items-center tw:justify-start tw:gap-3 tw:rounded-lg tw:px-4 tw:py-3 tw:text-text-muted tw:no-underline tw:transition-all tw:duration-200 tw:hover:bg-primary/10 tw:hover:text-white ${isActive ? 'tw:bg-primary tw:text-white' : ''}`
              }
            >
              <Users size={20} />
              <span>Users</span>
            </NavLink>
          )}
          {user?.role === 'admin' && (
            <NavLink
              to="/settings"
              onClick={closeMobileMenu}
              className={({ isActive }) =>
                `tw:flex tw:items-center tw:justify-start tw:gap-3 tw:rounded-lg tw:px-4 tw:py-3 tw:text-text-muted tw:no-underline tw:transition-all tw:duration-200 tw:hover:bg-primary/10 tw:hover:text-white ${isActive ? 'tw:bg-primary tw:text-white' : ''}`
              }
            >
              <Settings size={20} />
              <span>Settings</span>
            </NavLink>
          )}
        </nav>
        <div className="tw:mt-auto tw:shrink-0 tw:border-t tw:border-border tw:bg-black/20 tw:p-4">
          <div className="tw:flex tw:min-w-0 tw:items-center tw:gap-3 tw:text-text-muted">
            <div className="tw:flex tw:min-w-0 tw:flex-1 tw:flex-col tw:gap-1">
              <div className="tw:flex tw:items-center tw:gap-2">
                <span className="tw:min-w-0 tw:overflow-hidden tw:text-[0.9rem] tw:font-medium tw:text-ellipsis tw:whitespace-nowrap tw:text-text-main">
                  {user?.username}
                </span>
                <span className="tw:rounded-sm tw:bg-primary/10 tw:px-1.5 tw:py-0.5 tw:text-[0.7rem] tw:font-semibold tw:text-primary tw:uppercase">
                  {user?.role}
                </span>
              </div>
              <div className="tw:flex tw:items-center tw:gap-1.5 tw:text-[0.8rem] tw:text-text-muted">
                {version}
                {updateAvailable && (
                  <a
                    href={releaseUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    title="Update Available"
                    className="tw:ml-1 tw:flex tw:cursor-pointer tw:items-center tw:gap-1 tw:rounded-sm tw:bg-amber-400/10 tw:px-1.5 tw:py-0.5 tw:text-xs tw:font-semibold tw:text-amber-300 tw:no-underline"
                  >
                    <AlertTriangle size={12} />
                    Update
                  </a>
                )}
              </div>
            </div>
            <button
              type="button"
              onClick={logout}
              className="tw:ml-auto tw:flex tw:cursor-pointer tw:items-center tw:justify-center tw:rounded-md tw:border-0 tw:bg-transparent tw:p-1.5 tw:text-text-muted tw:transition-all tw:duration-200 tw:hover:bg-white/10 tw:hover:text-danger"
              title="Logout"
            >
              <LogOut size={18} />
            </button>
          </div>
        </div>
      </aside>
      <main className="tw:flex tw:min-w-0 tw:flex-1 tw:flex-col tw:overflow-hidden">
        <div className="tw:min-w-0 tw:flex-1 tw:overflow-y-auto tw:p-4 tw:min-[769px]:p-[30px] tw:max-[481px]:p-3">
          <Outlet />
        </div>
      </main>
    </div>
  );
};

export default Layout;
