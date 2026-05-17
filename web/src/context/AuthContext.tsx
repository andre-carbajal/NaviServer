import axios from 'axios';

import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from 'react';

import { api } from '../services/api';

interface User {
  id: string;
  username: string;
  role: string;
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  login: (token: string, user: User) => void;
  logout: () => void;
  loading: boolean;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const isCheckingAuthRef = useRef(false);

  const checkAuth = useCallback(async () => {
    if (isCheckingAuthRef.current) return;
    isCheckingAuthRef.current = true;
    try {
      const response = await api.getMe();
      setUser(response.data);
      setToken('authenticated');
    } catch (error) {
      if (axios.isAxiosError(error)) {
        const isUnauthorized = error.response?.status === 401;
        const isNetworkError = error.code === 'ERR_NETWORK';
        if (!isUnauthorized && !isNetworkError) {
          console.error('Auth check failed', error);
        }
      } else {
        console.error('Auth check failed', error);
      }
      setUser(null);
      setToken(null);
    } finally {
      setLoading(false);
      isCheckingAuthRef.current = false;
    }
  }, []);

  useEffect(() => {
    void checkAuth();

    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        void checkAuth();
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () =>
      document.removeEventListener('visibilitychange', handleVisibilityChange);
  }, [checkAuth]);

  const login = (_newToken: string, newUser: User) => {
    setToken('authenticated');
    setUser(newUser);
  };

  const logout = async () => {
    try {
      await api.logout();
    } catch (error) {
      console.error('Logout failed', error);
    }
    setToken(null);
    setUser(null);
  };

  const isAuthenticated = token !== null && user !== null;

  return (
    <AuthContext.Provider
      value={{ user, token, login, logout, loading, isAuthenticated }}
    >
      {children}
    </AuthContext.Provider>
  );
};

// eslint-disable-next-line react-refresh/only-export-components
export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
