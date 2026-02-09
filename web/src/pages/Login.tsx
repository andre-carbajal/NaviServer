import { useNavigate } from 'react-router-dom';

import React, { useState } from 'react';

import '../App.css';
import { useAuth } from '../context/AuthContext';
import { api } from '../services/api';

const Login: React.FC = () => {
  const [isSetup, setIsSetup] = useState(false);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

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
          <button type="submit" className="btn-primary">
            {isSetup ? 'Create Admin Account' : 'Login'}
          </button>
        </form>
        <div className="login-footer">
          <button className="btn-link" onClick={() => setIsSetup(!isSetup)}>
            {isSetup
              ? 'Already have an account? Login'
              : 'Need to setup? (First run only)'}
          </button>
        </div>
      </div>
    </div>
  );
};

export default Login;
