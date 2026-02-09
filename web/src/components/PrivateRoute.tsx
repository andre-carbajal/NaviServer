import { Navigate, Outlet } from 'react-router-dom';

import React from 'react';

import { useAuth } from '../context/AuthContext';

const PrivateRoute: React.FC = () => {
  const { token, loading } = useAuth();

  if (loading) {
    return <div>Loading...</div>;
  }

  return token ? <Outlet /> : <Navigate to="/login" />;
};

export default PrivateRoute;
