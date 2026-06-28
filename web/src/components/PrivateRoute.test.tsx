import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';

import PrivateRoute from './PrivateRoute';

const mockUseAuth = vi.fn();

vi.mock('../context/AuthContext', () => ({
  useAuth: () => mockUseAuth(),
}));

const renderPrivateRoute = () =>
  render(
    <MemoryRouter initialEntries={['/private']}>
      <Routes>
        <Route element={<PrivateRoute />}>
          <Route path="/private" element={<div>Secret page</div>} />
        </Route>
        <Route path="/login" element={<div>Login page</div>} />
      </Routes>
    </MemoryRouter>,
  );

describe('PrivateRoute', () => {
  it('shows a loading state while auth is being checked', () => {
    mockUseAuth.mockReturnValue({ token: null, loading: true });

    renderPrivateRoute();

    expect(screen.getByText('Loading...')).toBeTruthy();
  });

  it('renders the protected route when authenticated', () => {
    mockUseAuth.mockReturnValue({ token: 'authenticated', loading: false });

    renderPrivateRoute();

    expect(screen.getByText('Secret page')).toBeTruthy();
  });

  it('redirects unauthenticated users to login', () => {
    mockUseAuth.mockReturnValue({ token: null, loading: false });

    renderPrivateRoute();

    expect(screen.getByText('Login page')).toBeTruthy();
  });
});
