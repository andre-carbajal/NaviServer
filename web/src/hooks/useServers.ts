import { use } from 'react';

import { ServerContext } from '../context/ServerContext.tsx';

export const useServers = () => {
  const context = use(ServerContext);
  if (context === undefined) {
    throw new Error('useServers must be used within a ServerProvider');
  }
  return context;
};
