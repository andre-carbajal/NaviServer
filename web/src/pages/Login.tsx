import axios from 'axios';
import { useNavigate } from 'react-router-dom';

import React, { useCallback, useEffect, useRef, useState } from 'react';

import { useAuth } from '../context/AuthContext';
import { api } from '../services/api';

const Login: React.FC = () => {
  const [isSetup, setIsSetup] = useState(false);
  const [setupChecked, setSetupChecked] = useState(false);
  const [canToggle, setCanToggle] = useState(true);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const { login } = useAuth();
  const navigate = useNavigate();
  const isCheckingSetupRef = useRef(false);

  const usernameError = username.includes(' ')
    ? 'Username cannot contain spaces'
    : '';

  const checkSetupStatus = useCallback(async () => {
    if (isCheckingSetupRef.current) return;
    isCheckingSetupRef.current = true;
    try {
      const response = await api.checkSetup();
      if (response.data.setup_needed) {
        setIsSetup(true);
        setCanToggle(false);
      } else {
        setIsSetup(false);
        setCanToggle(false);
      }
      setError('');
      setSetupChecked(true);
    } catch (err) {
      if (axios.isAxiosError(err) && err.code === 'ERR_NETWORK') {
        setError('Backend unavailable. Start NaviServer daemon and try again.');
      } else {
        setError('Failed to check setup status.');
      }
      setSetupChecked(true);
    } finally {
      isCheckingSetupRef.current = false;
    }
  }, []);

  useEffect(() => {
    const initialCheck = window.setTimeout(() => {
      void checkSetupStatus();
    }, 0);
    const retryOnRecovery = () => {
      void checkSetupStatus();
    };
    window.addEventListener('network-recovered', retryOnRecovery);
    return () => {
      window.clearTimeout(initialCheck);
      window.removeEventListener('network-recovered', retryOnRecovery);
    };
  }, [checkSetupStatus]);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError('');

    if (username.includes(' ')) {
      setError('Username cannot contain spaces');
      return;
    }

    try {
      let response;
      if (isSetup) {
        response = await api.setup(username, password);
      } else {
        response = await api.login(username, password);
      }

      const data = response.data;
      login('', data.user);
      navigate('/');
    } catch (err) {
      let msg = 'Authentication failed';
      if (axios.isAxiosError(err)) {
        if (err.code === 'ERR_NETWORK') {
          msg = 'Backend unavailable. Start NaviServer daemon and try again.';
        } else {
          msg = err.response?.data?.trim() || err.message || msg;
        }
      } else if (err instanceof Error) {
        msg = err.message;
      }
      setError(msg);
    }
  };

  if (!setupChecked) {
    return null;
  }

  return (
    <div className="tw:relative tw:z-[9999] tw:box-border tw:flex tw:min-h-screen tw:w-full tw:items-center tw:justify-center tw:bg-bg-dark tw:p-5">
      <div className="tw:w-full tw:max-w-[440px] tw:rounded-3xl tw:border tw:border-white/8 tw:bg-[rgba(30,30,35,0.6)] tw:p-8 tw:shadow-[0_25px_50px_-12px_rgba(0,0,0,0.5)] tw:backdrop-blur-2xl tw:animate-[fadeIn_0.6s_ease-out] tw:max-[480px]:max-w-[calc(100%-20px)] tw:max-[480px]:rounded-xl tw:max-[480px]:p-6">
        <h2 className="tw:mb-8 tw:mt-0 tw:bg-gradient-to-r tw:from-white tw:to-indigo-300 tw:bg-clip-text tw:text-center tw:text-[2rem] tw:font-bold tw:tracking-[-0.025em] tw:text-transparent tw:max-[480px]:mb-5 tw:max-[480px]:text-xl">
          {isSetup ? 'First Time Setup' : 'Login'}
        </h2>
        {error && (
          <div className="tw:mb-6 tw:flex tw:items-center tw:justify-center tw:gap-2 tw:rounded-xl tw:border tw:border-red-500/20 tw:bg-red-500/10 tw:p-4 tw:text-center tw:text-[0.9rem] tw:text-red-400">
            {error}
          </div>
        )}
        <form onSubmit={handleSubmit}>
          <div className="tw:mb-6 tw:max-[480px]:mb-4">
            <label
              htmlFor="login-username"
              className="tw:mb-2 tw:block tw:text-[0.9rem] tw:font-medium tw:text-slate-400 tw:max-[480px]:text-xs"
            >
              Username
            </label>
            <input
              id="login-username"
              type="text"
              className="tw:box-border tw:w-full tw:rounded-xl tw:border tw:border-white/10 tw:bg-black/30 tw:px-4 tw:py-3 tw:text-base tw:text-white tw:outline-none tw:transition-all tw:duration-200 tw:focus:border-primary tw:focus:bg-black/50 tw:focus:ring-4 tw:focus:ring-primary/15 tw:max-[480px]:px-3.5 tw:max-[480px]:py-2.5"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
            />
            {usernameError && (
              <div className="tw:mt-1 tw:flex tw:items-center tw:justify-center tw:gap-2 tw:rounded-xl tw:border tw:border-red-500/20 tw:bg-red-500/10 tw:p-4 tw:text-center tw:text-[0.9rem] tw:text-red-400">
                {usernameError}
              </div>
            )}
          </div>
          <div className="tw:mb-6 tw:max-[480px]:mb-4">
            <label
              htmlFor="login-password"
              className="tw:mb-2 tw:block tw:text-[0.9rem] tw:font-medium tw:text-slate-400 tw:max-[480px]:text-xs"
            >
              Password
            </label>
            <input
              id="login-password"
              type="password"
              className="tw:box-border tw:w-full tw:rounded-xl tw:border tw:border-white/10 tw:bg-black/30 tw:px-4 tw:py-3 tw:text-base tw:text-white tw:outline-none tw:transition-all tw:duration-200 tw:focus:border-primary tw:focus:bg-black/50 tw:focus:ring-4 tw:focus:ring-primary/15 tw:max-[480px]:px-3.5 tw:max-[480px]:py-2.5"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </div>
          <button
            type="submit"
            className="tw:mt-2 tw:w-full tw:cursor-pointer tw:rounded-xl tw:border-0 tw:bg-gradient-to-r tw:from-primary tw:to-indigo-600 tw:px-4 tw:py-3.5 tw:text-base tw:font-semibold tw:text-white tw:transition-all tw:duration-300 tw:hover:-translate-y-0.5 tw:hover:shadow-[0_8px_20px_rgba(79,70,229,0.4)] tw:disabled:cursor-not-allowed tw:disabled:opacity-50"
            disabled={!!usernameError}
          >
            {isSetup ? 'Create Admin Account' : 'Login'}
          </button>
        </form>
        {canToggle && (
          <div className="tw:mt-6 tw:text-center">
            <button
              type="button"
              className="tw:cursor-pointer tw:border-0 tw:bg-transparent tw:p-2 tw:text-[0.9rem] tw:text-slate-500 tw:transition-colors tw:hover:text-primary tw:max-[480px]:text-xs"
              onClick={() => setIsSetup(!isSetup)}
            >
              {isSetup
                ? 'Already have an account? Login'
                : 'Need to setup? (First run only)'}
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

export default Login;
