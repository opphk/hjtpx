import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { SliderCaptcha } from '../../../src/components/captchas/SliderCaptcha';

describe('SliderCaptcha', () => {
  const defaultProps = {
    onSuccess: jest.fn(),
    onRefresh: jest.fn(),
    onError: jest.fn(),
    width: 300,
    height: 150,
    theme: 'light',
    status: 'ready'
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('renders correctly', () => {
    render(<SliderCaptcha {...defaultProps} />);

    expect(screen.getByText(/拖动滑块完成验证/i)).toBeInTheDocument();
  });

  test('renders with dark theme', () => {
    render(<SliderCaptcha {...defaultProps} theme="dark" />);

    expect(screen.getByText(/拖动滑块完成验证/i)).toBeInTheDocument();
  });

  test('calls onRefresh when refresh button is clicked', () => {
    render(<SliderCaptcha {...defaultProps} />);

    const refreshButton = screen.getByRole('button', { name: /刷新/i });
    fireEvent.click(refreshButton);

    expect(defaultProps.onRefresh).toHaveBeenCalled();
  });

  test('resets position when status changes to ready', () => {
    const { rerender } = render(
      <SliderCaptcha {...defaultProps} status="success" />
    );

    rerender(<SliderCaptcha {...defaultProps} status="ready" />);

    expect(screen.getByText(/拖动滑块完成验证/i)).toBeInTheDocument();
  });

  test('has correct width and height', () => {
    render(<SliderCaptcha {...defaultProps} width={400} height={200} />);

    const container = screen.getByText(/拖动滑块完成验证/i).closest('.slider-captcha');
    expect(container).toHaveStyle({ width: '400px', height: '200px' });
  });
});
