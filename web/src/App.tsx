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

    window.addEventListener('network-error', handleNetworkError);

    return () => {
      window.removeEventListener('network-error', handleNetworkError);
    };
  }, []);

  useEffect(() => {
    if (notification.visible) {
      const timer = setTimeout(() => {
        setNotification((prev) => ({ ...prev, visible: false }));
      }, 5000);
      return () => clearTimeout(timer);
    }
  }, [notification.visible]);

  return (
    <AuthProvider>
      <ServerProvider>
        <BrowserRouter>
          {notification.visible && (
            <div className={`notification ${notification.type}`}>
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

const styleSheet = document.createElement('style');
styleSheet.type = 'text/css';
styleSheet.innerText = `
  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(-20px); }
    to { opacity: 1; transform: translateY(0); }
  }
  @keyframes fadeOut {
    from { opacity: 1; transform: translateY(0); }
    to { opacity: 0; transform: translateY(-20px); }
  }
`;
document.head.appendChild(styleSheet);

export default App;
