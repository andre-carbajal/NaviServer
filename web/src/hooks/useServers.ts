import { useContext } from 'react';

import { ServerContext } from '../context/ServerContext.tsx';

export const useServers = () => {
  const context = useContext(ServerContext);
  if (context === undefined) {
    throw new Error('useServers must be used within a ServerProvider');
  }
  return context;
};
