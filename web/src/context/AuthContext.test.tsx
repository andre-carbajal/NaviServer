import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import React from 'react';

import { AuthProvider, useAuth } from './AuthContext';

const { mockApi } = vi.hoisted(() => ({
  mockApi: {
    getMe: vi.fn(),
    logout: vi.fn(),
  },
}));

vi.mock('../services/api', () => ({
  api: mockApi,
}));

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <AuthProvider>{children}</AuthProvider>
);

describe('AuthContext', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    Object.defineProperty(document, 'visibilityState', {
      configurable: true,
      value: 'visible',
    });
  });

  it('hydrates an authenticated session from /auth/me', async () => {
    mockApi.getMe.mockResolvedValue({
      data: { id: 'u1', username: 'andre', role: 'admin' },
    });

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.user?.username).toBe('andre');
    expect(result.current.token).toBe('authenticated');
    expect(result.current.isAuthenticated).toBe(true);
  });

  it('clears auth state when the auth check fails', async () => {
    mockApi.getMe.mockRejectedValue({
      isAxiosError: true,
      response: { status: 401 },
    });

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.user).toBeNull();
    expect(result.current.token).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
  });

  it('re-checks auth when the tab becomes visible again', async () => {
    mockApi.getMe
      .mockResolvedValueOnce({
        data: { id: 'u1', username: 'andre', role: 'admin' },
      })
      .mockResolvedValueOnce({
        data: { id: 'u1', username: 'andre', role: 'admin' },
      });

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.loading).toBe(false));
    const initialCalls = mockApi.getMe.mock.calls.length;

    act(() => {
      document.dispatchEvent(new Event('visibilitychange'));
    });

    await waitFor(() =>
      expect(mockApi.getMe.mock.calls.length).toBeGreaterThan(initialCalls),
    );
  });

  it('lets login and logout update auth state', async () => {
    mockApi.getMe.mockRejectedValue({
      isAxiosError: true,
      response: { status: 401 },
    });
    mockApi.logout.mockResolvedValue(undefined);

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.loading).toBe(false));

    act(() => {
      result.current.login('ignored-token', {
        id: 'u2',
        username: 'new-user',
        role: 'viewer',
      });
    });

    expect(result.current.isAuthenticated).toBe(true);
    expect(result.current.user?.username).toBe('new-user');

    await act(async () => {
      result.current.logout();
    });

    expect(result.current.user).toBeNull();
    expect(result.current.token).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
  });
});
