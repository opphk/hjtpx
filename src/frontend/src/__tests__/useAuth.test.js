import { renderHook, act } from '@testing-library/react';
import { describe, test, expect, jest } from '@jest/globals';
import { AuthProvider } from '../context/AuthContext';
import { useAuth } from '../hooks/useAuth';

const mockAuthContextValue = {
  user: null,
  isAuthenticated: false,
  isLoading: false,
  login: jest.fn(),
  logout: jest.fn(),
  register: jest.fn(),
  refreshToken: jest.fn()
};

jest.mock('../context/AuthContext', () => ({
  AuthContext: {
    Consumer: ({ children }) => children(mockAuthContextValue)
  },
  AuthProvider: ({ children }) => children(mockAuthContextValue)
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
    expect(result.current).toHaveProperty('register');
    expect(result.current).toHaveProperty('refreshToken');
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

    mockAuthContextValue.login.mockResolvedValue({ success: true });

    await act(async () => {
      await result.current.login({ email: 'test@example.com', password: 'password' });
    });

    expect(mockAuthContextValue.login).toHaveBeenCalledWith({
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

    mockAuthContextValue.logout.mockResolvedValue();

    await act(async () => {
      await result.current.logout();
    });

    expect(mockAuthContextValue.logout).toHaveBeenCalled();
  });

  test('register function is callable', async () => {
    const { result } = renderHook(() => useAuth(), {
      wrapper: ({ children }) => (
        <AuthProvider>{children}</AuthProvider>
      )
    });

    const userData = {
      email: 'test@example.com',
      password: 'password123',
      name: 'Test User'
    };

    mockAuthContextValue.register.mockResolvedValue({ success: true });

    await act(async () => {
      await result.current.register(userData);
    });

    expect(mockAuthContextValue.register).toHaveBeenCalledWith(userData);
  });

  test('refreshToken function is callable', async () => {
    const { result } = renderHook(() => useAuth(), {
      wrapper: ({ children }) => (
        <AuthProvider>{children}</AuthProvider>
      )
    });

    mockAuthContextValue.refreshToken.mockResolvedValue({ token: 'new-token' });

    await act(async () => {
      await result.current.refreshToken();
    });

    expect(mockAuthContextValue.refreshToken).toHaveBeenCalled();
  });

  test('updates when user state changes', () => {
    const { result } = renderHook(() => useAuth(), {
      wrapper: ({ children }) => (
        <AuthProvider>{children}</AuthProvider>
      )
    });

    const mockUser = { id: '1', email: 'test@example.com', name: 'Test User' };

    act(() => {
      mockAuthContextValue.user = mockUser;
      mockAuthContextValue.isAuthenticated = true;
    });

    expect(result.current.user).toEqual(mockUser);
    expect(result.current.isAuthenticated).toBe(true);
  });
});
