import { renderHook, act } from '@testing-library/react';
import { useCaptchaState } from '../../../src/hooks/useCaptchaState';
import { CaptchaProvider } from '../../../src/CaptchaProvider';

const mockVerify = jest.fn();

jest.mock('../../../src/CaptchaProvider', () => {
  const actual = jest.requireActual('../../../src/CaptchaProvider');
  return {
    ...actual,
    useCaptcha: () => ({
      verify: mockVerify,
      config: {},
      loading: false,
      error: null,
      clearError: jest.fn()
    })
  };
});

describe('useCaptchaState', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  const wrapper = ({ children }) => (
    <CaptchaProvider>{children}</CaptchaProvider>
  );

  test('returns initial state', () => {
    const { result } = renderHook(() => useCaptchaState(), { wrapper });

    expect(result.current.state.isVerified).toBe(false);
    expect(result.current.state.token).toBeNull();
    expect(result.current.state.attempts).toBe(0);
    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  test('updates state on successful verification', async () => {
    mockVerify.mockResolvedValue('test-token-123');

    const { result } = renderHook(() => useCaptchaState(), { wrapper });

    await act(async () => {
      await result.current.verify('test-scene');
    });

    expect(result.current.state.isVerified).toBe(true);
    expect(result.current.state.token).toBe('test-token-123');
    expect(result.current.loading).toBe(false);
  });

  test('updates state on failed verification', async () => {
    mockVerify.mockRejectedValue(new Error('Verification failed'));

    const { result } = renderHook(() => useCaptchaState(), { wrapper });

    await act(async () => {
      try {
        await result.current.verify('test-scene');
      } catch (e) {}
    });

    expect(result.current.state.isVerified).toBe(false);
    expect(result.current.error).toBe('Verification failed');
    expect(result.current.state.attempts).toBe(1);
  });

  test('resets state correctly', async () => {
    mockVerify.mockResolvedValue('test-token');

    const { result } = renderHook(() => useCaptchaState(), { wrapper });

    await act(async () => {
      await result.current.verify('test-scene');
    });

    expect(result.current.state.isVerified).toBe(true);

    await act(async () => {
      result.current.reset();
    });

    expect(result.current.state.isVerified).toBe(false);
    expect(result.current.state.token).toBeNull();
    expect(result.current.state.attempts).toBe(0);
  });

  test('canRetry returns true when attempts < maxAttempts', () => {
    const { result } = renderHook(() => useCaptchaState({ maxAttempts: 3 }), { wrapper });

    expect(result.current.canRetry).toBe(true);
  });

  test('canRetry returns false when attempts >= maxAttempts', async () => {
    mockVerify.mockRejectedValue(new Error('Error'));

    const { result } = renderHook(() => useCaptchaState({ maxAttempts: 1 }), { wrapper });

    for (let i = 0; i < 2; i++) {
      await act(async () => {
        try {
          await result.current.verify('test');
        } catch (e) {}
      });
    }

    expect(result.current.canRetry).toBe(false);
  });

  test('clearError clears error state', async () => {
    mockVerify.mockRejectedValue(new Error('Test error'));

    const { result } = renderHook(() => useCaptchaState(), { wrapper });

    await act(async () => {
      try {
        await result.current.verify('test');
      } catch (e) {}
    });

    expect(result.current.error).toBe('Test error');

    act(() => {
      result.current.clearError();
    });

    expect(result.current.error).toBeNull();
  });
});
