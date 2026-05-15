import { renderHook } from '@testing-library/react';
import { describe, test, expect, jest, beforeEach } from '@jest/globals';
import { AuthProvider } from '../context/AuthContext';
import { useAuth } from '../hooks/useAuth';

jest.mock('../context/AuthContext', () => ({
  AuthContext: {
    Consumer: ({ children }) => children({
      user: null,
      isAuthenticated: false,
      isLoading: false,
      login: jest.fn(),
      logout: jest.fn(),
      register: jest.fn(),
      refreshToken: jest.fn()
    })
  },
  AuthProvider: ({ children }) => children({
    user: null,
    isAuthenticated: false,
    isLoading: false,
    login: jest.fn(),
    logout: jest.fn(),
    register: jest.fn(),
    refreshToken: jest.fn()
  })
}));

describe('useAuth Hook', () => {
  test('throws error when used outside AuthProvider', () => {
    const consoleError = jest.spyOn(console, 'error').mockImplementation(() => {});

    expect(() => {
      renderHook(() => useAuth());
    }).toThrow('useAuth must be used within an AuthProvider');

    consoleError.mockRestore();
  });

  test('returns auth context with all required properties', () => {
    const { result } = renderHook(() => useAuth(), {
      wrapper: ({ children }) => (
        <AuthProvider>{children}</AuthProvider>
      )
    });

    expect(result.current).toHaveProperty('user');
    expect(result.current).toHaveProperty('isAuthenticated');
    expect(result.current).toHaveProperty('isLoading');
    expect(result.current).toHaveProperty('login');
    expect(result.current).toHaveProperty('logout');
  });

  test('has correct initial state', () => {
    const { result } = renderHook(() => useAuth(), {
      wrapper: ({ children }) => (
        <AuthProvider>{children}</AuthProvider>
      )
    });

    expect(result.current.user).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.isLoading).toBe(false);
  });

  test('login function is callable', async () => {
    const { result } = renderHook(() => useAuth(), {
      wrapper: ({ children }) => (
        <AuthProvider>{children}</AuthProvider>
      )
    });

    await result.current.login({ email: 'test@example.com', password: 'password' });
    expect(result.current.login).toHaveBeenCalledWith({
      email: 'test@example.com',
      password: 'password'
    });
  });

  test('logout function is callable', async () => {
    const { result } = renderHook(() => useAuth(), {
      wrapper: ({ children }) => (
        <AuthProvider>{children}</AuthProvider>
      )
    });

    await result.current.logout();
    expect(result.current.logout).toHaveBeenCalled();
  });
});
