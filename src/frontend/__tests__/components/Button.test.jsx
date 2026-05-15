import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import Button from '../../src/components/ui/Button';
import { describe, test, expect, vi, beforeEach } from 'vitest';

describe('Button Component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test('renders button with text', () => {
    render(<Button>Click Me</Button>);
    const buttonElement = screen.getByRole('button', { name: /click me/i });
    expect(buttonElement).toBeInTheDocument();
  });

  test('calls onClick handler when clicked', () => {
    const handleClick = vi.fn();
    render(<Button onClick={handleClick}>Click Me</Button>);
    const buttonElement = screen.getByRole('button', { name: /click me/i });
    fireEvent.click(buttonElement);
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  test('does not call onClick when disabled', () => {
    const handleClick = vi.fn();
    render(<Button onClick={handleClick} disabled>Click Me</Button>);
    const buttonElement = screen.getByRole('button', { name: /click me/i });
    expect(buttonElement).toBeDisabled();
    fireEvent.click(buttonElement);
    expect(handleClick).not.toHaveBeenCalled();
  });

  test('does not call onClick when loading', () => {
    const handleClick = vi.fn();
    render(<Button onClick={handleClick} loading>Click Me</Button>);
    const buttonElement = screen.getByRole('button');
    expect(buttonElement).toBeDisabled();
    fireEvent.click(buttonElement);
    expect(handleClick).not.toHaveBeenCalled();
  });

  test('applies variant classes correctly', () => {
    const { rerender } = render(<Button variant="primary">Primary</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-primary');

    rerender(<Button variant="secondary">Secondary</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-secondary');

    rerender(<Button variant="success">Success</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-success');

    rerender(<Button variant="danger">Danger</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-danger');
  });

  test('applies size classes correctly', () => {
    const { rerender } = render(<Button size="small">Small</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-small');

    rerender(<Button size="medium">Medium</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-medium');

    rerender(<Button size="large">Large</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-large');
  });

  test('renders with correct type attribute', () => {
    const { rerender } = render(<Button type="button">Button</Button>);
    expect(screen.getByRole('button')).toHaveAttribute('type', 'button');

    rerender(<Button type="submit">Submit</Button>);
    expect(screen.getByRole('button')).toHaveAttribute('type', 'submit');

    rerender(<Button type="reset">Reset</Button>);
    expect(screen.getByRole('button')).toHaveAttribute('type', 'reset');
  });

  test('renders loading indicator when loading prop is true', () => {
    render(<Button loading>Loading</Button>);
    const buttonElement = screen.getByRole('button');
    expect(buttonElement).toHaveClass('btn-loading');
    const spinner = buttonElement.querySelector('.spinner');
    expect(spinner).toBeInTheDocument();
  });

  test('applies custom className', () => {
    render(<Button className="custom-class">Custom</Button>);
    expect(screen.getByRole('button')).toHaveClass('custom-class');
  });

  test('renders with aria-label when provided', () => {
    render(<Button aria-label="Close dialog">×</Button>);
    expect(screen.getByRole('button')).toHaveAttribute('aria-label', 'Close dialog');
  });

  test('applies aria-disabled when disabled', () => {
    render(<Button disabled>Disabled</Button>);
    expect(screen.getByRole('button')).toHaveAttribute('aria-disabled', 'true');
  });

  test('applies aria-disabled when loading', () => {
    render(<Button loading>Loading</Button>);
    expect(screen.getByRole('button')).toHaveAttribute('aria-disabled', 'true');
  });

  test('applies aria-busy when loading', () => {
    render(<Button loading>Loading</Button>);
    expect(screen.getByRole('button')).toHaveAttribute('aria-busy', 'true');
  });

  test('renders warning variant', () => {
    render(<Button variant="warning">Warning</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-warning');
  });

  test('handles keyboard events', () => {
    const handleKeyDown = vi.fn();
    render(<Button onKeyDown={handleKeyDown}>Button</Button>);
    const buttonElement = screen.getByRole('button');
    fireEvent.keyDown(buttonElement, { key: 'Enter' });
    expect(handleKeyDown).toHaveBeenCalledTimes(1);
  });
});
