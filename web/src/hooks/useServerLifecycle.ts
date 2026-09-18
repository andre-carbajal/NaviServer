import { useState } from 'react';
import type { Dispatch, SetStateAction } from 'react';

import { api } from '../services/api';
import type { Server } from '../types';
import { getApiErrorMessage } from '../utils/apiError';
import type { AlertOptions } from './useModalDialog';

type PowerAction = null | 'start' | 'stop' | 'restart' | 'kill';

interface UseServerLifecycleOptions {
  server: Server | null;
  refresh: () => Promise<boolean>;
  setServer: Dispatch<SetStateAction<Server | null>>;
  showAlert: (options: AlertOptions) => Promise<void>;
  closeMenu: () => void;
}

export const useServerLifecycle = ({
  server,
  refresh,
  setServer,
  showAlert,
  closeMenu,
}: UseServerLifecycleOptions) => {
  const [powerAction, setPowerAction] = useState<PowerAction>(null);

  const run = async (
    action: Exclude<PowerAction, null>,
    request: (id: string) => Promise<unknown>,
    status: Server['status'],
  ) => {
    if (!server) return;
    setPowerAction(action);
    try {
      await request(server.id);
      setServer((current) => (current ? { ...current, status } : null));
    } catch (error) {
      console.error(error);
      if (action === 'start') {
        await refresh();
        await showAlert({
          title: 'Start Failed',
          message: getApiErrorMessage(error, 'Failed to start server.'),
          variant: 'danger',
        });
      }
    } finally {
      setPowerAction(null);
      if (action !== 'start') closeMenu();
    }
  };

  return {
    powerAction,
    start: () => run('start', api.startServer, 'STARTING'),
    stop: () => run('stop', api.stopServer, 'STOPPING'),
    restart: () => run('restart', api.restartServer, 'STARTING'),
    kill: () => run('kill', api.killServer, 'STOPPED'),
  };
};
