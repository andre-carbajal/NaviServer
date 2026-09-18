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
    <div className="tw:flex tw:h-full tw:w-full tw:flex-col tw:pt-[50px] tw:pb-[50px] tw:min-[769px]:flex-row tw:min-[769px]:p-0">
      <header className="tw:fixed tw:inset-x-0 tw:top-0 tw:z-[100] tw:flex tw:h-[50px] tw:items-center tw:justify-between tw:border-b tw:border-border tw:bg-bg-sidebar tw:px-4 tw:min-[769px]:hidden">
        <div className="tw:flex tw:h-[60px] tw:items-center tw:gap-3 tw:border-0 tw:p-0 tw:text-[1.1rem] tw:font-bold tw:text-primary tw:max-[481px]:gap-1.5 tw:max-[481px]:text-base">
          <img
            src="/apple-touch-icon.png"
            alt="NaviServer"
            className="tw:h-6 tw:w-6 tw:max-[481px]:h-5 tw:max-[481px]:w-5"
          />
          <span>NaviServer</span>
        </div>
        <div className="tw:flex tw:items-center tw:gap-3 tw:text-text-muted tw:max-[481px]:gap-2 tw:max-[481px]:text-[0.8rem]">
          <div className="tw:mr-2 tw:flex tw:items-center tw:gap-2">
            <span className="tw:text-xs tw:text-text-muted">{version}</span>
            {updateAvailable && (
              <a
                href={releaseUrl}
                target="_blank"
                rel="noopener noreferrer"
                title="Update Available"
                className="tw:flex tw:cursor-pointer tw:items-center tw:gap-1 tw:rounded-sm tw:bg-amber-400/10 tw:px-1.5 tw:py-0.5 tw:text-xs tw:font-semibold tw:text-amber-300 tw:no-underline"
              >
                <AlertTriangle size={12} />
              </a>
            )}
          </div>
          <div className="tw:flex tw:items-center tw:gap-2">
            <span className="tw:overflow-hidden tw:text-[0.9rem] tw:font-medium tw:text-ellipsis tw:whitespace-nowrap tw:text-text-main">
              {user?.username}
            </span>
            <span className="tw:rounded-sm tw:bg-primary/10 tw:px-1.5 tw:py-0.5 tw:text-[0.7rem] tw:font-semibold tw:text-primary tw:uppercase">
              {user?.role}
            </span>
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
      </header>
      <aside className="tw:fixed tw:inset-x-0 tw:bottom-0 tw:z-[100] tw:flex tw:h-[50px] tw:w-full tw:flex-row tw:items-center tw:border-t tw:border-border tw:bg-bg-sidebar tw:min-[769px]:static tw:min-[769px]:h-auto tw:min-[769px]:w-[250px] tw:min-[769px]:flex-col tw:min-[769px]:items-stretch tw:min-[769px]:border-t-0 tw:min-[769px]:border-r">
        <div className="tw:hidden tw:h-[60px] tw:items-center tw:gap-3 tw:border-b tw:border-border tw:px-5 tw:text-[1.2rem] tw:font-bold tw:text-primary tw:min-[769px]:flex">
          <img
            src="/apple-touch-icon.png"
            alt="NaviServer"
            className="tw:h-6 tw:w-6"
          />
          <span>NaviServer</span>
        </div>
        <nav className="tw:flex tw:h-full tw:w-full tw:flex-row tw:justify-around tw:p-0 tw:min-[769px]:h-auto tw:min-[769px]:flex-col tw:min-[769px]:justify-start tw:min-[769px]:gap-[5px] tw:min-[769px]:px-2.5 tw:min-[769px]:py-5">
          <NavLink
            to="/"
            className={({ isActive }) =>
              `tw:flex tw:h-full tw:flex-1 tw:items-center tw:justify-center tw:gap-3 tw:rounded-none tw:p-0 tw:text-text-muted tw:no-underline tw:transition-all tw:duration-200 tw:hover:bg-primary/10 tw:hover:text-white tw:min-[769px]:h-auto tw:min-[769px]:flex-none tw:min-[769px]:justify-start tw:min-[769px]:rounded-lg tw:min-[769px]:px-4 tw:min-[769px]:py-3 ${isActive ? 'tw:bg-primary tw:text-white' : ''}`
            }
          >
            <LayoutDashboard size={20} />
            <span className="tw:hidden tw:min-[769px]:inline">Dashboard</span>
          </NavLink>
          <NavLink
            to="/servers/backups/all"
            className={({ isActive }) =>
              `tw:flex tw:h-full tw:flex-1 tw:items-center tw:justify-center tw:gap-3 tw:rounded-none tw:p-0 tw:text-text-muted tw:no-underline tw:transition-all tw:duration-200 tw:hover:bg-primary/10 tw:hover:text-white tw:min-[769px]:h-auto tw:min-[769px]:flex-none tw:min-[769px]:justify-start tw:min-[769px]:rounded-lg tw:min-[769px]:px-4 tw:min-[769px]:py-3 ${isActive ? 'tw:bg-primary tw:text-white' : ''}`
            }
          >
            <DatabaseBackup size={20} />
            <span className="tw:hidden tw:min-[769px]:inline">Backups</span>
          </NavLink>
          {user?.role === 'admin' && (
            <NavLink
              to="/users"
              className={({ isActive }) =>
                `tw:flex tw:h-full tw:flex-1 tw:items-center tw:justify-center tw:gap-3 tw:rounded-none tw:p-0 tw:text-text-muted tw:no-underline tw:transition-all tw:duration-200 tw:hover:bg-primary/10 tw:hover:text-white tw:min-[769px]:h-auto tw:min-[769px]:flex-none tw:min-[769px]:justify-start tw:min-[769px]:rounded-lg tw:min-[769px]:px-4 tw:min-[769px]:py-3 ${isActive ? 'tw:bg-primary tw:text-white' : ''}`
              }
            >
              <Users size={20} />
              <span className="tw:hidden tw:min-[769px]:inline">Users</span>
            </NavLink>
          )}
          {user?.role === 'admin' && (
            <NavLink
              to="/settings"
              className={({ isActive }) =>
                `tw:flex tw:h-full tw:flex-1 tw:items-center tw:justify-center tw:gap-3 tw:rounded-none tw:p-0 tw:text-text-muted tw:no-underline tw:transition-all tw:duration-200 tw:hover:bg-primary/10 tw:hover:text-white tw:min-[769px]:h-auto tw:min-[769px]:flex-none tw:min-[769px]:justify-start tw:min-[769px]:rounded-lg tw:min-[769px]:px-4 tw:min-[769px]:py-3 ${isActive ? 'tw:bg-primary tw:text-white' : ''}`
              }
            >
              <Settings size={20} />
              <span className="tw:hidden tw:min-[769px]:inline">Settings</span>
            </NavLink>
          )}
        </nav>
        <div className="tw:mt-auto tw:hidden tw:border-t tw:border-border tw:bg-black/20 tw:p-4 tw:min-[769px]:block">
          <div className="tw:flex tw:items-center tw:gap-3 tw:text-text-muted">
            <div className="tw:flex tw:flex-col tw:gap-1">
              <div className="tw:flex tw:items-center tw:gap-2">
                <span className="tw:overflow-hidden tw:text-[0.9rem] tw:font-medium tw:text-ellipsis tw:whitespace-nowrap tw:text-text-main">
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
      <main className="tw:flex tw:flex-1 tw:flex-col tw:overflow-hidden">
        <div className="tw:flex-1 tw:overflow-y-auto tw:p-4 tw:min-[769px]:p-[30px] tw:max-[481px]:p-3">
          <Outlet />
        </div>
      </main>
    </div>
  );
};

export default Layout;
