import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import Form from '../../src/components/ui/Form';
import { describe, test, expect, vi, beforeEach } from 'vitest';

describe('Form Component', () => {
  const mockOnSubmit = vi.fn();

  const defaultFields = [
    { name: 'email', label: '邮箱', type: 'email', required: true },
    { name: 'password', label: '密码', type: 'password', required: true },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
  });

  test('renders form with all fields', () => {
    render(<Form fields={defaultFields} onSubmit={mockOnSubmit} />);
    expect(screen.getByLabelText(/邮箱/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/密码/i)).toBeInTheDocument();
  });

  test('renders submit button with default text', () => {
    render(<Form fields={defaultFields} onSubmit={mockOnSubmit} />);
    expect(screen.getByRole('button', { name: /提交/i })).toBeInTheDocument();
  });

  test('renders submit button with custom text', () => {
    render(<Form fields={defaultFields} onSubmit={mockOnSubmit} submitText="登录" />);
    expect(screen.getByRole('button', { name: /登录/i })).toBeInTheDocument();
  });

  test('handles input changes', () => {
    render(<Form fields={defaultFields} onSubmit={mockOnSubmit} />);
    const emailInput = screen.getByLabelText(/邮箱/i);
    fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
    expect(emailInput).toHaveValue('test@example.com');
  });

  test('calls onSubmit with form values when valid', async () => {
    render(<Form fields={defaultFields} onSubmit={mockOnSubmit} />);
    const emailInput = screen.getByLabelText(/邮箱/i);
    const passwordInput = screen.getByLabelText(/密码/i);

    fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
    fireEvent.change(passwordInput, { target: { value: 'password123' } });
    fireEvent.click(screen.getByRole('button', { name: /提交/i }));

    await waitFor(() => {
      expect(mockOnSubmit).toHaveBeenCalledWith({
        email: 'test@example.com',
        password: 'password123',
      });
    });
  });

  test('shows validation errors for required fields', async () => {
    render(<Form fields={defaultFields} onSubmit={mockOnSubmit} />);
    fireEvent.click(screen.getByRole('button', { name: /提交/i }));

    await waitFor(() => {
      expect(screen.getByText(/邮箱不能为空/i)).toBeInTheDocument();
      expect(screen.getByText(/密码不能为空/i)).toBeInTheDocument();
    });

    expect(mockOnSubmit).not.toHaveBeenCalled();
  });

  test('displays errors for touched fields only after blur', async () => {
    render(<Form fields={defaultFields} onSubmit={mockOnSubmit} />);
    const emailInput = screen.getByLabelText(/邮箱/i);

    fireEvent.change(emailInput, { target: { value: '' } });
    expect(screen.queryByText(/邮箱不能为空/i)).not.toBeInTheDocument();

    fireEvent.blur(emailInput);
    await waitFor(() => {
      expect(screen.getByText(/邮箱不能为空/i)).toBeInTheDocument();
    });
  });

  test('clears error when user starts typing', async () => {
    render(<Form fields={defaultFields} onSubmit={mockOnSubmit} />);
    const emailInput = screen.getByLabelText(/邮箱/i);

    fireEvent.blur(emailInput);
    await waitFor(() => {
      expect(screen.getByText(/邮箱不能为空/i)).toBeInTheDocument();
    });

    fireEvent.change(emailInput, { target: { value: 't' } });
    await waitFor(() => {
      expect(screen.queryByText(/邮箱不能为空/i)).not.toBeInTheDocument();
    });
  });

  test('renders with initial values', () => {
    render(
      <Form
        fields={defaultFields}
        initialValues={{ email: 'initial@example.com' }}
        onSubmit={mockOnSubmit}
      />
    );
    expect(screen.getByLabelText(/邮箱/i)).toHaveValue('initial@example.com');
  });

  test('disables submit button when loading', () => {
    render(<Form fields={defaultFields} onSubmit={mockOnSubmit} loading={true} />);
    expect(screen.getByRole('button', { name: /提交/i })).toBeDisabled();
  });

  test('renders custom children', () => {
    render(
      <Form fields={defaultFields} onSubmit={mockOnSubmit}>
        <div data-testid="custom-content">自定义内容</div>
      </Form>
    );
    expect(screen.getByTestId('custom-content')).toBeInTheDocument();
  });

  test('applies custom className', () => {
    const { container } = render(
      <Form fields={defaultFields} onSubmit={mockOnSubmit} className="custom-form" />
    );
    expect(container.querySelector('.custom-form')).toBeInTheDocument();
  });

  test('has aria-label on form when provided', () => {
    render(<Form fields={defaultFields} onSubmit={mockOnSubmit} aria-label="登录表单" />);
    expect(screen.getByRole('form', { name: /登录表单/i })).toBeInTheDocument();
  });

  test('validates with custom validation schema', async () => {
    const customFields = [
      {
        name: 'username',
        label: '用户名',
        validation: {
          required: true,
          minLength: 2,
          message: '用户名至少2个字符',
        },
      },
    ];

    render(<Form fields={customFields} onSubmit={mockOnSubmit} />);
    const usernameInput = screen.getByLabelText(/用户名/i);

    fireEvent.change(usernameInput, { target: { value: 'a' } });
    fireEvent.blur(usernameInput);
    fireEvent.click(screen.getByRole('button', { name: /提交/i }));

    await waitFor(() => {
      expect(screen.getByText(/用户名至少2个字符/i)).toBeInTheDocument();
    });
  });

  test('validates with function validator', async () => {
    const customFields = [
      {
        name: 'age',
        label: '年龄',
        validation: (value) => {
          if (value < 18) return '必须年满18岁';
          return '';
        },
      },
    ];

    render(<Form fields={customFields} onSubmit={mockOnSubmit} />);
    const ageInput = screen.getByLabelText(/年龄/i);

    fireEvent.change(ageInput, { target: { value: '16' } });
    fireEvent.blur(ageInput);
    fireEvent.click(screen.getByRole('button', { name: /提交/i }));

    await waitFor(() => {
      expect(screen.getByText(/必须年满18岁/i)).toBeInTheDocument();
    });
  });

  test('handles checkbox fields', () => {
    const checkboxFields = [
      { name: 'agree', label: '同意条款', type: 'checkbox' },
    ];

    render(<Form fields={checkboxFields} onSubmit={mockOnSubmit} />);
    const checkbox = screen.getByRole('checkbox', { name: /同意条款/i });

    expect(checkbox).not.toBeChecked();
    fireEvent.click(checkbox);
    expect(checkbox).toBeChecked();
  });

  test('prevents submission when there are validation errors', async () => {
    render(<Form fields={defaultFields} onSubmit={mockOnSubmit} />);
    fireEvent.click(screen.getByRole('button', { name: /提交/i }));

    await waitFor(() => {
      expect(screen.getByText(/邮箱不能为空/i)).toBeInTheDocument();
    });

    expect(mockOnSubmit).not.toHaveBeenCalled();
  });
});
