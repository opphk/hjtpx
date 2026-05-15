import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import Input from '../../src/components/ui/Input';
import { describe, test, expect, vi, beforeEach } from 'vitest';

describe('Input Component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test('renders input with label', () => {
    render(<Input label="Email" name="email" />);
    const labelElement = screen.getByText(/email/i);
    const inputElement = screen.getByRole('textbox', { name: /email/i });
    expect(labelElement).toBeInTheDocument();
    expect(inputElement).toBeInTheDocument();
  });

  test('renders input without label', () => {
    render(<Input name="username" />);
    const inputElement = screen.getByRole('textbox');
    expect(inputElement).toBeInTheDocument();
  });

  test('renders with placeholder text', () => {
    render(<Input name="email" placeholder="Enter your email" />);
    const inputElement = screen.getByPlaceholderText(/enter your email/i);
    expect(inputElement).toBeInTheDocument();
  });

  test('calls onChange handler when value changes', () => {
    const handleChange = vi.fn();
    render(<Input name="email" onChange={handleChange} />);
    const inputElement = screen.getByRole('textbox');
    fireEvent.change(inputElement, { target: { value: 'test@example.com' } });
    expect(handleChange).toHaveBeenCalledTimes(1);
  });

  test('displays error message when error prop is provided', () => {
    render(<Input name="email" error="Invalid email format" />);
    const errorElement = screen.getByText(/invalid email format/i);
    expect(errorElement).toBeInTheDocument();
    expect(errorElement).toHaveClass('error-text');
  });

  test('applies error class when error prop is provided', () => {
    render(<Input name="email" error="Error" />);
    const inputElement = screen.getByRole('textbox');
    expect(inputElement).toHaveClass('input-error');
  });

  test('renders required asterisk when required prop is true', () => {
    render(<Input label="Email" name="email" required />);
    const requiredElement = screen.getByText(/\*/);
    expect(requiredElement).toBeInTheDocument();
  });

  test('renders disabled input', () => {
    render(<Input name="email" disabled />);
    const inputElement = screen.getByRole('textbox');
    expect(inputElement).toBeDisabled();
  });

  test('renders with correct type attribute', () => {
    render(<Input type="password" name="password" />);
    const inputElement = document.querySelector('input[type="password"]');
    expect(inputElement).toBeInTheDocument();
    expect(inputElement).toHaveAttribute('type', 'password');
  });

  test('renders with email type', () => {
    render(<Input type="email" name="email" />);
    const inputElement = screen.getByRole('textbox');
    expect(inputElement).toHaveAttribute('type', 'email');
  });

  test('applies custom className', () => {
    render(<Input name="email" className="custom-input" />);
    const wrapper = screen.getByRole('textbox').closest('.form-group');
    expect(wrapper).toHaveClass('custom-input');
  });

  test('renders with correct name attribute', () => {
    render(<Input name="username" />);
    const inputElement = screen.getByRole('textbox');
    expect(inputElement).toHaveAttribute('name', 'username');
  });

  test('renders with correct id attribute', () => {
    render(<Input name="email" />);
    const inputElement = screen.getByRole('textbox');
    const id = inputElement.getAttribute('id');
    expect(id).toBeTruthy();
  });

  test('handles keyboard events', () => {
    const handleKeyDown = vi.fn();
    render(<Input name="email" onKeyDown={handleKeyDown} />);
    const inputElement = screen.getByRole('textbox');
    fireEvent.keyDown(inputElement, { key: 'Enter' });
    expect(handleKeyDown).toHaveBeenCalledTimes(1);
  });

  test('applies aria-invalid when error prop is provided', () => {
    render(<Input name="email" error="Error" />);
    const inputElement = screen.getByRole('textbox');
    expect(inputElement).toHaveAttribute('aria-invalid', 'true');
  });

  test('applies aria-required when required prop is true', () => {
    render(<Input name="email" required />);
    const inputElement = screen.getByRole('textbox');
    expect(inputElement).toHaveAttribute('aria-required', 'true');
  });

  test('applies aria-disabled when disabled prop is true', () => {
    render(<Input name="email" disabled />);
    const inputElement = screen.getByRole('textbox');
    expect(inputElement).toHaveAttribute('aria-disabled', 'true');
  });

  test('has correct for attribute on label', () => {
    render(<Input label="Email" name="email" />);
    const labelElement = screen.getByText(/email/i).closest('label');
    expect(labelElement).toHaveAttribute('for', 'email');
  });

  test('renders number input type', () => {
    render(<Input type="number" name="age" />);
    const inputElement = screen.getByRole('spinbutton');
    expect(inputElement).toBeInTheDocument();
  });

  test('renders with controlled value', () => {
    render(<Input name="email" value="test@example.com" onChange={() => {}} />);
    const inputElement = screen.getByRole('textbox');
    expect(inputElement).toHaveValue('test@example.com');
  });

  test('connects error message via aria-describedby', () => {
    render(<Input name="email" error="Invalid email" />);
    const inputElement = screen.getByRole('textbox');
    const describedBy = inputElement.getAttribute('aria-describedby');
    expect(describedBy).toContain('email-error');
  });
});
