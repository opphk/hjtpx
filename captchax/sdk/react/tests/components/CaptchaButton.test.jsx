import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CaptchaButton } from '../../../src/components/CaptchaButton';
import { CaptchaProvider } from '../../../src/CaptchaProvider';

const mockVerify = jest.fn();

jest.mock('../../../src/CaptchaProvider', () => {
  const actual = jest.requireActual('../../../src/CaptchaProvider');
  return {
    ...actual,
    useCaptcha: () => ({
      verify: mockVerify,
      loading: false,
      error: null,
      clearError: jest.fn()
    })
  };
});

describe('CaptchaButton', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('renders with default props', () => {
    render(
      <CaptchaProvider>
        <CaptchaButton>点击验证</CaptchaButton>
      </CaptchaProvider>
    );

    expect(screen.getByRole('button', { name: /点击验证/i })).toBeInTheDocument();
  });

  test('renders with different sizes', () => {
    const { rerender } = render(
      <CaptchaProvider>
        <CaptchaButton size="small">小按钮</CaptchaButton>
      </CaptchaProvider>
    );

    expect(screen.getByRole('button')).toHaveClass('captcha-button--small');

    rerender(
      <CaptchaProvider>
        <CaptchaButton size="large">大按钮</CaptchaButton>
      </CaptchaProvider>
    );

    expect(screen.getByRole('button')).toHaveClass('captcha-button--large');
  });

  test('calls verify on click', async () => {
    mockVerify.mockResolvedValue('test-token-123');

    render(
      <CaptchaProvider>
        <CaptchaButton>验证</CaptchaButton>
      </CaptchaProvider>
    );

    const button = screen.getByRole('button');
    fireEvent.click(button);

    await waitFor(() => {
      expect(mockVerify).toHaveBeenCalledWith('default');
    });
  });

  test('calls onSuccess callback with token', async () => {
    const onSuccess = jest.fn();
    mockVerify.mockResolvedValue('success-token');

    render(
      <CaptchaProvider>
        <CaptchaButton onSuccess={onSuccess}>验证</CaptchaButton>
      </CaptchaProvider>
    );

    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledWith('success-token');
    });
  });

  test('calls onError callback on verification failure', async () => {
    const onError = jest.fn();
    const error = new Error('Verification failed');
    mockVerify.mockRejectedValue(error);

    render(
      <CaptchaProvider>
        <CaptchaButton onError={onError}>验证</CaptchaButton>
      </CaptchaProvider>
    );

    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(onError).toHaveBeenCalledWith(error);
    });
  });

  test('displays error message when verification fails', async () => {
    mockVerify.mockRejectedValue(new Error('Network error'));

    render(
      <CaptchaProvider>
        <CaptchaButton>验证</CaptchaButton>
      </CaptchaProvider>
    );

    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(screen.getByText(/网络错误/i)).toBeInTheDocument();
    });
  });

  test('is disabled when disabled prop is true', () => {
    render(
      <CaptchaProvider>
        <CaptchaButton disabled>验证</CaptchaButton>
      </CaptchaProvider>
    );

    expect(screen.getByRole('button')).toBeDisabled();
  });
});
