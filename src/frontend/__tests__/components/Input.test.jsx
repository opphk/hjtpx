import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, test, expect, vi } from 'vitest';
import Input from '../../components/Input';

describe('Input Component', () => {
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
    expect(handleChange).toHaveBeenCalledWith(expect.objectContaining({
      target: expect.objectContaining({ value: 'test@example.com' })
    }));
  });

  test('displays error message when error prop is provided', () => {
    render(<Input name="email" error="Invalid email format" />);
    const errorElement = screen.getByText(/invalid email format/i);
    expect(errorElement).toBeInTheDocument();
    expect(errorElement).toHaveClass('input-error-message');
  });

  test('applies error class when error prop is provided', () => {
    render(<Input name="email" error="Error" />);
    const wrapper = screen.getByTestId('input-wrapper') || screen.getByRole('textbox').parentElement;
    expect(screen.getByRole('textbox').closest('.input-wrapper')).toHaveClass('input-error');
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
    const { rerender } = render(<Input type="text" name="text" />);
    expect(screen.getByRole('textbox')).toHaveAttribute('type', 'text');

    rerender(<Input type="email" name="email" />);
    expect(screen.getByRole('textbox')).toHaveAttribute('type', 'email');

    rerender(<Input type="password" name="password" />);
    const passwordInput = document.querySelector('input[type="password"]') || screen.getByRole('textbox');
    expect(passwordInput).toHaveAttribute('type', 'password');
  });

  test('applies custom className', () => {
    render(<Input name="email" className="custom-input" />);
    const inputWrapper = screen.getByRole('textbox').closest('.input-wrapper');
    expect(inputWrapper).toHaveClass('custom-input');
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
});
