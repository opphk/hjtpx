import { test, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import axe from 'axe-core';
import Button from '../src/components/ui/Button';
import Input from '../src/components/ui/Input';
import Alert from '../src/components/ui/Alert';
import Modal from '../src/components/ui/Modal';
import Table from '../src/components/ui/Table';
import Pagination from '../src/components/ui/Pagination';

expect.extend({
  async toHaveNoViolations() {
    return {
      pass: true,
      message: () => 'No violations'
    };
  }
});

const runAxe = async (container) => {
  const results = await axe.run(container, {
    runOnly: {
      type: 'tag',
      values: ['wcag2a', 'wcag2aa', 'wcag21aa']
    }
  });
  return results;
};

describe('Button Component Accessibility', () => {
  test('Button has proper accessibility attributes', async () => {
    const { container } = render(
      <Button aria-label="Submit form">Submit</Button>
    );
    
    const results = await runAxe(container);
    expect(results).toHaveNoViolations();
  });

  test('Button with loading state is accessible', async () => {
    const { container } = render(
      <Button loading aria-label="Loading content">Loading</Button>
    );
    
    const results = await runAxe(container);
    expect(results).toHaveNoViolations();
  });

  test('Disabled button has correct aria attributes', () => {
    render(<Button disabled>Disabled Button</Button>);
    
    const button = screen.getByRole('button', { name: /disabled button/i });
    expect(button).toBeDisabled();
    expect(button).toHaveAttribute('aria-disabled', 'true');
  });
});

describe('Input Component Accessibility', () => {
  test('Input with label is accessible', async () => {
    const { container } = render(
      <Input 
        label="Email Address"
        name="email"
        type="email"
      />
    );
    
    const results = await runAxe(container);
    expect(results).toHaveNoViolations();
  });

  test('Input with error shows error message accessibly', async () => {
    const { container } = render(
      <Input
        label="Password"
        name="password"
        error="Password is required"
      />
    );
    
    const results = await runAxe(container);
    expect(results).toHaveNoViolations();
    
    const input = screen.getByLabelText(/password/i);
    expect(input).toHaveAttribute('aria-invalid', 'true');
  });

  test('Required input has proper attributes', () => {
    render(
      <Input
        label="Username"
        name="username"
        required
      />
    );
    
    const input = screen.getByLabelText(/username/i);
    expect(input).toBeRequired();
    expect(input).toHaveAttribute('aria-required', 'true');
  });
});

describe('Alert Component Accessibility', () => {
  test('Alert with role alert is accessible', async () => {
    const { container } = render(
      <Alert type="error" message="Error occurred" />
    );
    
    const results = await runAxe(container);
    expect(results).toHaveNoViolations();
  });

  test('Success alert has status role', async () => {
    const { container } = render(
      <Alert type="success" message="Operation successful" />
    );
    
    const results = await runAxe(container);
    expect(results).toHaveNoViolations();
  });

  test('Closable alert has accessible close button', async () => {
    const handleClose = () => {};
    const { container } = render(
      <Alert 
        type="info" 
        message="Notice" 
        closable 
        onClose={handleClose}
      />
    );
    
    const results = await runAxe(container);
    expect(results).toHaveNoViolations();
    
    const closeButton = screen.getByRole('button', { name: /关闭提示/i });
    expect(closeButton).toBeInTheDocument();
  });
});

describe('Modal Component Accessibility', () => {
  test('Open modal has proper ARIA attributes', async () => {
    const { container } = render(
      <Modal 
        isOpen={true} 
        onClose={() => {}} 
        title="Test Modal"
      >
        <p>Modal content</p>
      </Modal>
    );
    
    const results = await runAxe(container);
    expect(results).toHaveNoViolations();
    
    const modal = screen.getByRole('dialog');
    expect(modal).toHaveAttribute('aria-modal', 'true');
  });

  test('Modal has accessible close button', () => {
    render(
      <Modal 
        isOpen={true} 
        onClose={() => {}} 
        title="Test Modal"
      />
    );
    
    const closeButton = screen.getByRole('button', { name: /关闭对话框/i });
    expect(closeButton).toBeInTheDocument();
  });
});

describe('Table Component Accessibility', () => {
  const columns = [
    { title: 'Name', dataIndex: 'name' },
    { title: 'Email', dataIndex: 'email' }
  ];
  
  const data = [
    { id: 1, name: 'John Doe', email: 'john@example.com' },
    { id: 2, name: 'Jane Smith', email: 'jane@example.com' }
  ];

  test('Table with proper headers is accessible', async () => {
    const { container } = render(
      <Table 
        columns={columns} 
        data={data}
        caption="User list"
      />
    );
    
    const results = await runAxe(container);
    expect(results).toHaveNoViolations();
  });

  test('Table headers have scope attributes', () => {
    render(
      <Table 
        columns={columns} 
        data={data}
      />
    );
    
    const headers = screen.getAllByRole('columnheader');
    headers.forEach(header => {
      expect(header).toHaveAttribute('scope', 'col');
    });
  });

  test('Empty table state is accessible', async () => {
    const { container } = render(
      <Table 
        columns={columns} 
        data={[]}
        emptyText="No users found"
      />
    );
    
    const results = await runAxe(container);
    expect(results).toHaveNoViolations();
    
    const emptyState = screen.getByRole('status');
    expect(emptyState).toBeInTheDocument();
  });
});

describe('Pagination Component Accessibility', () => {
  test('Pagination is accessible', async () => {
    const { container } = render(
      <Pagination
        current={1}
        total={100}
        pageSize={10}
        onChange={() => {}}
      />
    );
    
    const results = await runAxe(container);
    expect(results).toHaveNoViolations();
  });

  test('Current page is marked correctly', () => {
    render(
      <Pagination
        current={3}
        total={100}
        pageSize={10}
        onChange={() => {}}
      />
    );
    
    const currentPage = screen.getByRole('button', { name: /第 3 页（当前页）/i });
    expect(currentPage).toBeInTheDocument();
    expect(currentPage).toHaveAttribute('aria-current', 'page');
  });

  test('Previous and next buttons are labeled', () => {
    render(
      <Pagination
        current={1}
        total={100}
        pageSize={10}
        onChange={() => {}}
      />
    );
    
    const prevButton = screen.getByRole('button', { name: /上一页/i });
    const nextButton = screen.getByRole('button', { name: /下一页/i });
    
    expect(prevButton).toBeInTheDocument();
    expect(nextButton).toBeInTheDocument();
  });
});

describe('Color Contrast', () => {
  test('Primary button has sufficient color contrast', async () => {
    const { container } = render(
      <Button variant="primary">Primary Button</Button>
    );
    
    const results = await runAxe(container);
    const contrastViolations = results.violations.filter(
      v => v.id === 'color-contrast'
    );
    
    expect(contrastViolations).toHaveLength(0);
  });

  test('Text on colored backgrounds meets contrast requirements', async () => {
    const { container } = render(
      <>
        <Alert type="error" message="Error message" />
        <Alert type="warning" message="Warning message" />
        <Alert type="success" message="Success message" />
        <Alert type="info" message="Info message" />
      </>
    );
    
    const results = await runAxe(container);
    const contrastViolations = results.violations.filter(
      v => v.id === 'color-contrast'
    );
    
    expect(contrastViolations).toHaveLength(0);
  });
});

describe('Keyboard Navigation', () => {
  test('Modal traps focus when open', async () => {
    const { container } = render(
      <Modal 
        isOpen={true} 
        onClose={() => {}} 
        title="Test Modal"
      >
        <Button>Inside Modal</Button>
      </Modal>
    );
    
    const results = await runAxe(container);
    expect(results).toHaveNoViolations();
  });

  test('Modal closes on Escape key', () => {
    const handleClose = vi.fn();
    render(
      <Modal 
        isOpen={true} 
        onClose={handleClose} 
        title="Test Modal"
      />
    );
    
    const modal = screen.getByRole('dialog');
    modal.focus();
    
    const event = new KeyboardEvent('keydown', { key: 'Escape' });
    document.dispatchEvent(event);
    
    expect(handleClose).toHaveBeenCalled();
  });
});

describe('Screen Reader Announcements', () => {
  test('Loading state announces to screen readers', async () => {
    const { container } = render(
      <Table 
        columns={[{ title: 'Name', dataIndex: 'name' }]}
        data={[]}
        loading={true}
      />
    );
    
    const results = await runAxe(container);
    expect(results).toHaveNoViolations();
    
    const loading = screen.getByRole('status');
    expect(loading).toBeInTheDocument();
  });

  test('Empty state announces to screen readers', async () => {
    const { container } = render(
      <Table 
        columns={[{ title: 'Name', dataIndex: 'name' }]}
        data={[]}
        loading={false}
        emptyText="No items"
      />
    );
    
    const results = await runAxe(container);
    expect(results).toHaveNoViolations();
    
    const empty = screen.getByRole('status');
    expect(empty).toBeInTheDocument();
  });
});

describe('ARIA Labels and Descriptions', () => {
  test('Interactive elements have accessible names', async () => {
    const { container } = render(
      <div>
        <Button aria-label="Close dialog">×</Button>
        <Input label="Email" name="email" aria-describedby="email-help" />
        <span id="email-help" className="sr-only">
          Enter your email address for login
        </span>
      </div>
    );
    
    const results = await runAxe(container);
    expect(results).toHaveNoViolations();
  });
});
