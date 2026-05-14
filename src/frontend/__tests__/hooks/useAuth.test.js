import { renderHook, act } from '@testing-library/react';
import { useAuth } from '../../hooks/useAuth';
import { AuthProvider } from '../../context/AuthContext';

jest.mock('../../api/client', () => ({
  apiClient: {
    post: jest.fn(),
    get: jest.fn(),
  },
  setAuthToken: jest.fn(),
  getAuthToken: jest.fn(),
}));

const wrapper = ({ children }) => <AuthProvider>{children}</AuthProvider>;

describe('useAuth Hook', () => {
  beforeEach(() => {
    localStorage.clear();
    jest.clearAllMocks();
  });

  test('throws error when used outside AuthProvider', () => {
    const consoleError = jest.spyOn(console, 'error').mockImplementation(() => {});
    expect(() => {
      renderHook(() => useAuth());
    }).toThrow('useAuth must be used within an AuthProvider');
    consoleError.mockRestore();
  });

  test('returns initial state when no user is logged in', () => {
    const { result } = renderHook(() => useAuth(), { wrapper });
    expect(result.current.user).toBeNull();
    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
  });

  test('login updates user state', async () => {
    const mockUser = { id: 1, username: 'testuser', email: 'test@example.com' };
    const { apiClient } = require('../../api/client');
    apiClient.post.mockResolvedValueOnce({ data: { token: 'fake-token', user: mockUser } });

    const { result } = renderHook(() => useAuth(), { wrapper });

    await act(async () => {
      await result.current.login('test@example.com', 'password123');
    });

    expect(result.current.user).toEqual(mockUser);
    expect(result.current.isAuthenticated).toBe(true);
  });

  test('login throws error on failure', async () => {
    const { apiClient } = require('../../api/client');
    apiClient.post.mockRejectedValueOnce(new Error('Invalid credentials'));

    const { result } = renderHook(() => useAuth(), { wrapper });

    await act(async () => {
      try {
        await result.current.login('invalid@example.com', 'wrongpassword');
      } catch (error) {
        expect(error.message).toBe('Invalid credentials');
      }
    });
  });

  test('register updates user state', async () => {
    const mockUser = { id: 1, username: 'newuser', email: 'new@example.com' };
    const { apiClient } = require('../../api/client');
    apiClient.post.mockResolvedValueOnce({ data: { token: 'fake-token', user: mockUser } });

    const { result } = renderHook(() => useAuth(), { wrapper });

    await act(async () => {
      await result.current.register('newuser', 'new@example.com', 'password123');
    });

    expect(result.current.user).toEqual(mockUser);
    expect(result.current.isAuthenticated).toBe(true);
  });

  test('logout clears user state', async () => {
    const mockUser = { id: 1, username: 'testuser', email: 'test@example.com' };
    const { apiClient } = require('../../api/client');
    apiClient.post.mockResolvedValueOnce({ data: { token: 'fake-token', user: mockUser } });

    const { result } = renderHook(() => useAuth(), { wrapper });

    await act(async () => {
      await result.current.login('test@example.com', 'password123');
    });

    expect(result.current.isAuthenticated).toBe(true);

    await act(async () => {
      result.current.logout();
    });
  });

  test('updateUser updates user data', async () => {
    const mockUser = { id: 1, username: 'testuser', email: 'test@example.com' };
    const { apiClient } = require('../../api/client');
    apiClient.post.mockResolvedValueOnce({ data: { token: 'fake-token', user: mockUser } });

    const { result } = renderHook(() => useAuth(), { wrapper });

    await act(async () => {
      await result.current.login('test@example.com', 'password123');
    });

    const updatedUserData = { username: 'updateduser' };

    await act(async () => {
      result.current.updateUser(updatedUserData);
    });

    expect(result.current.user.username).toBe('updateduser');
    expect(result.current.user.email).toBe('test@example.com');
  });

  test('initializes from localStorage if token exists', () => {
    const { getAuthToken } = require('../../api/client');
    const storedUser = { id: 1, username: 'storeduser', email: 'stored@example.com' };
    getAuthToken.mockReturnValueOnce('existing-token');
    localStorage.setItem('user', JSON.stringify(storedUser));

    const { result } = renderHook(() => useAuth(), { wrapper });

    expect(result.current.user).toEqual(storedUser);
    expect(result.current.isAuthenticated).toBe(true);
  });
});
