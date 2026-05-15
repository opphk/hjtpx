import { test, expect, describe, vi, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import React from 'react';

vi.mock('../services/auth', () => ({
  authService: {
    getToken: vi.fn(() => 'mock-token'),
    getUser: vi.fn(() => ({ id: '1', email: 'test@example.com', name: 'Test User' })),
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn()
  }
}));

const mockAuthContextValue = {
  user: null,
  isAuthenticated: false,
  isLoading: false,
  loading: false,
  login: vi.fn(),
  logout: vi.fn(),
  register: vi.fn(),
  checkAuth: vi.fn()
};

vi.mock('../context/AuthContext', () => ({
  AuthContext: React.createContext(mockAuthContextValue),
  AuthProvider: ({ children }) => children
}));

import { useAuth } from '../hooks/useAuth';

describe('useAuth Hook', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test('returns auth context with all required properties', () => {
    const { result } = renderHook(() => useAuth());

    expect(result.current).toHaveProperty('user');
    expect(result.current).toHaveProperty('isAuthenticated');
    expect(result.current).toHaveProperty('loading');
    expect(result.current).toHaveProperty('login');
    expect(result.current).toHaveProperty('logout');
    expect(result.current).toHaveProperty('register');
    expect(result.current).toHaveProperty('checkAuth');
  });

  test('has correct initial state', () => {
    const { result } = renderHook(() => useAuth());

    expect(result.current.user).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.loading).toBe(false);
  });

  test('login function is callable', async () => {
    const { result } = renderHook(() => useAuth());

    await result.current.login({ email: 'test@example.com', password: 'password' });
    expect(result.current.login).toHaveBeenCalled();
  });

  test('logout function is callable', async () => {
    const { result } = renderHook(() => useAuth());

    await result.current.logout();
    expect(result.current.logout).toHaveBeenCalled();
  });

  test('register function is callable', async () => {
    const { result } = renderHook(() => useAuth());

    await result.current.register({ email: 'test@example.com', password: 'password' });
    expect(result.current.register).toHaveBeenCalled();
  });

  test('checkAuth function is callable', () => {
    const { result } = renderHook(() => useAuth());

    result.current.checkAuth();
    expect(result.current.checkAuth).toHaveBeenCalled();
  });
});
