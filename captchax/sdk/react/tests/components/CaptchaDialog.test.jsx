import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { CaptchaDialog } from '../../../src/components/CaptchaDialog';
import { CaptchaProvider } from '../../../src/CaptchaProvider';

const mockRefresh = jest.fn().mockResolvedValue({ success: true });
const mockVerify = jest.fn().mockResolvedValue('test-token');

jest.mock('../../../src/CaptchaProvider', () => {
  const actual = jest.requireActual('../../../src/CaptchaProvider');
  return {
    ...actual,
    useCaptcha: () => ({
      verify: mockVerify,
      refresh: mockRefresh,
      loading: false,
      error: null
    })
  };
});

describe('CaptchaDialog', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockRefresh.mockResolvedValue({ success: true });
    mockVerify.mockResolvedValue('test-token');
  });

  test('renders when visible', () => {
    render(
      <CaptchaProvider>
        <CaptchaDialog
          visible={true}
          onClose={() => {}}
        />
      </CaptchaProvider>
    );

    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText(/安全验证/i)).toBeInTheDocument();
  });

  test('does not render when not visible', () => {
    render(
      <CaptchaProvider>
        <CaptchaDialog
          visible={false}
          onClose={() => {}}
        />
      </CaptchaProvider>
    );

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  test('calls onClose when close button is clicked', () => {
    const onClose = jest.fn();

    render(
      <CaptchaProvider>
        <CaptchaDialog
          visible={true}
          onClose={onClose}
        />
      </CaptchaProvider>
    );

    fireEvent.click(screen.getByRole('button', { name: /关闭/i }));

    expect(onClose).toHaveBeenCalled();
  });

  test('closes on Escape key press', () => {
    const onClose = jest.fn();

    render(
      <CaptchaProvider>
        <CaptchaDialog
          visible={true}
          onClose={onClose}
        />
      </CaptchaProvider>
    );

    fireEvent.keyDown(document, { key: 'Escape' });

    expect(onClose).toHaveBeenCalled();
  });

  test('calls refresh when dialog becomes visible', async () => {
    render(
      <CaptchaProvider>
        <CaptchaDialog
          visible={true}
          onClose={() => {}}
        />
      </CaptchaProvider>
    );

    await waitFor(() => {
      expect(mockRefresh).toHaveBeenCalled();
    });
  });

  test('displays loading state', () => {
    render(
      <CaptchaProvider>
        <CaptchaDialog
          visible={true}
          onClose={() => {}}
        />
      </CaptchaProvider>
    );

    expect(screen.getByText(/加载中/i)).toBeInTheDocument();
  });

  test('has correct title text', () => {
    render(
      <CaptchaProvider>
        <CaptchaDialog
          visible={true}
          onClose={() => {}}
        />
      </CaptchaProvider>
    );

    expect(screen.getByText('安全验证')).toBeInTheDocument();
  });
});
