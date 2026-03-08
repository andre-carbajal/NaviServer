import { useNavigate } from 'react-router-dom';

import React, { useEffect, useState } from 'react';

import '../App.css';
import { useAuth } from '../context/AuthContext';
import { api } from '../services/api';

const Login: React.FC = () => {
  const [isSetup, setIsSetup] = useState(false);
  const [setupChecked, setSetupChecked] = useState(false);
  const [canToggle, setCanToggle] = useState(true);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [usernameError, setUsernameError] = useState('');
  const { login } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (username.includes(' ')) {
      setUsernameError('Username cannot contain spaces');
    } else {
      setUsernameError('');
    }
  }, [username]);

  useEffect(() => {
    const checkSetupStatus = async () => {
      try {
        const response = await api.checkSetup();
        if (response.data.setup_needed) {
          setIsSetup(true);
          setCanToggle(false);
        } else {
          setIsSetup(false);
          setCanToggle(false);
        }
        setSetupChecked(true);
      } catch (err) {
        console.error('Failed to check setup status:', err);
        setSetupChecked(true);
      }
    };
    checkSetupStatus();
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
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
    } catch (err: any) {
      console.error(err);
      const msg =
        err.response?.data?.trim() || err.message || 'Authentication failed';
      setError(msg);
    }
  };

  if (!setupChecked) {
    return null;
  }

  return (
    <div className="login-container">
      <div className="login-card">
        <h2>{isSetup ? 'First Time Setup' : 'Login'}</h2>
        {error && <div className="error-message">{error}</div>}
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Username</label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
            />
            {usernameError && (
              <div className="error-message" style={{ marginTop: '5px' }}>
                {usernameError}
              </div>
            )}
          </div>
          <div className="form-group">
            <label>Password</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </div>
          <button
            type="submit"
            className="btn-primary"
            disabled={!!usernameError}
          >
            {isSetup ? 'Create Admin Account' : 'Login'}
          </button>
        </form>
        {canToggle && (
          <div className="login-footer">
            <button className="btn-link" onClick={() => setIsSetup(!isSetup)}>
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
