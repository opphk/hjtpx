import { test, expect, describe } from 'vitest';
import { render, screen } from '@testing-library/react';
import Button from '../components/ui/Button';
import Input from '../components/ui/Input';
import Modal from '../components/ui/Modal';

describe('Button Component', () => {
  test('renders with default props', () => {
    render(<Button>Click me</Button>);
    const button = screen.getByRole('button', { name: /click me/i });
    expect(button).toBeInTheDocument();
  });

  test('applies correct variant classes', () => {
    const { rerender } = render(<Button variant="primary">Primary</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-primary');

    rerender(<Button variant="secondary">Secondary</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-secondary');
  });

  test('handles disabled state', () => {
    render(<Button disabled>Disabled</Button>);
    const button = screen.getByRole('button');
    expect(button).toBeDisabled();
  });

  test('handles loading state', () => {
    render(<Button loading>Loading</Button>);
    const button = screen.getByRole('button');
    expect(button).toBeDisabled();
    expect(button).toHaveAttribute('aria-busy', 'true');
  });
});

describe('Input Component', () => {
  test('renders with label', () => {
    render(<Input label="Email" name="email" />);
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
  });

  test('displays error message', () => {
    render(<Input name="email" error="Invalid email" />);
    expect(screen.getByText(/invalid email/i)).toBeInTheDocument();
  });
});

describe('Modal Component', () => {
  const defaultProps = {
    isOpen: true,
    onClose: () => {},
    title: 'Test Modal'
  };

  test('renders when isOpen is true', () => {
    render(<Modal {...defaultProps}>Content</Modal>);
    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });

  test('does not render when isOpen is false', () => {
    render(<Modal {...defaultProps} isOpen={false}>Content</Modal>);
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});
