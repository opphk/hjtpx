import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import Modal from '../../src/components/ui/Modal';
import { describe, test, expect, vi, beforeEach } from 'vitest';

describe('Modal Component', () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    title: 'Test Modal',
    children: <p>Modal content</p>,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  test('renders modal when isOpen is true', () => {
    render(<Modal {...defaultProps} />);
    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });

  test('does not render modal when isOpen is false', () => {
    render(<Modal {...defaultProps} isOpen={false} />);
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  test('displays title correctly', () => {
    render(<Modal {...defaultProps} title="Custom Title" />);
    expect(screen.getByText('Custom Title')).toBeInTheDocument();
  });

  test('displays children content', () => {
    render(<Modal {...defaultProps} children={<span data-testid="content">Custom Content</span>} />);
    expect(screen.getByTestId('content')).toBeInTheDocument();
  });

  test('displays footer when provided', () => {
    render(<Modal {...defaultProps} footer={<button>Footer Action</button>} />);
    expect(screen.getByText('Footer Action')).toBeInTheDocument();
  });

  test('calls onClose when close button is clicked', () => {
    render(<Modal {...defaultProps} />);
    const closeButton = screen.getByRole('button', { name: /关闭对话框/i });
    fireEvent.click(closeButton);
    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });

  test('has aria-modal attribute set to true', () => {
    render(<Modal {...defaultProps} />);
    expect(screen.getByRole('dialog')).toHaveAttribute('aria-modal', 'true');
  });

  test('has tabIndex on modal container', () => {
    render(<Modal {...defaultProps} />);
    expect(screen.getByRole('dialog')).toHaveAttribute('tabIndex', '-1');
  });

  test('closes on Escape key press', () => {
    render(<Modal {...defaultProps} />);
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });

  test('uses custom aria-label when provided', () => {
    render(<Modal {...defaultProps} aria-label="自定义对话框" />);
    expect(screen.getByRole('dialog')).toHaveAttribute('aria-label', '自定义对话框');
  });

  test('renders without footer', () => {
    const { container } = render(<Modal {...defaultProps} />);
    expect(container.querySelector('.modal-footer')).not.toBeInTheDocument();
  });

  test('handles rapid open/close transitions', () => {
    const { rerender } = render(<Modal {...defaultProps} isOpen={true} />);
    expect(screen.queryByRole('dialog')).toBeInTheDocument();

    rerender(<Modal {...defaultProps} isOpen={false} />);
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});
