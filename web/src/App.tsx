import { BrowserRouter, Route, Routes } from 'react-router-dom';

import { useEffect, useState } from 'react';

import Layout from './components/Layout';
import PrivateRoute from './components/PrivateRoute';
import { AuthProvider } from './context/AuthContext';
import { ServerProvider } from './context/ServerContext';
import Backups from './pages/Backups';
import Dashboard from './pages/Dashboard';
import Login from './pages/Login';
import PublicServer from './pages/PublicServer';
import ServerDetail from './pages/ServerDetail';
import Settings from './pages/Settings';
import UsersPage from './pages/Users';

const App = () => {
  const [notification, setNotification] = useState<{
    message: string;
    visible: boolean;
    type: 'info' | 'error';
  }>({
    message: '',
    visible: false,
    type: 'info',
  });

  useEffect(() => {
    const handleNetworkError = (event: Event) => {
      const customEvent = event as CustomEvent;
      setNotification({
        message: customEvent.detail.message,
        visible: true,
        type: 'error',
      });
    };
    const handleNetworkRecovered = () => {
      setNotification({
        message: 'Connection restored.',
        visible: true,
        type: 'info',
      });
    };

    window.addEventListener('network-error', handleNetworkError);
    window.addEventListener('network-recovered', handleNetworkRecovered);

    return () => {
      window.removeEventListener('network-error', handleNetworkError);
      window.removeEventListener('network-recovered', handleNetworkRecovered);
    };
  }, []);

  useEffect(() => {
    if (notification.visible && notification.type === 'info') {
      const timer = setTimeout(() => {
        setNotification((prev) => ({ ...prev, visible: false }));
      }, 3000);
      return () => clearTimeout(timer);
    }
  }, [notification.type, notification.visible]);

  return (
    <AuthProvider>
      <ServerProvider>
        <BrowserRouter>
          {notification.visible && (
            <div
              className={`tw:fixed tw:right-5 tw:top-5 tw:z-[1000] tw:rounded-lg tw:px-5 tw:py-[15px] tw:text-white tw:shadow-[0_4px_8px_rgba(0,0,0,0.2)] tw:animate-[fadeIn_0.5s_0s_both,fadeOut_0.5s_4.5s_both] tw:max-[768px]:top-2.5 tw:max-[768px]:right-2.5 tw:max-[768px]:left-2.5 tw:max-[768px]:max-w-none ${notification.type === 'error' ? 'tw:bg-danger' : 'tw:bg-primary'}`}
            >
              {notification.message}
            </div>
          )}
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/public/:token" element={<PublicServer />} />
            <Route element={<PrivateRoute />}>
              <Route path="/" element={<Layout />}>
                <Route index element={<Dashboard />} />
                <Route path="servers/backups/all" element={<Backups />} />
                <Route path="servers/:id" element={<ServerDetail />} />
                <Route path="servers/:id/backups" element={<Backups />} />
                <Route path="users" element={<UsersPage />} />
                <Route path="settings" element={<Settings />} />
              </Route>
            </Route>
          </Routes>
        </BrowserRouter>
      </ServerProvider>
    </AuthProvider>
  );
};

export default App;
