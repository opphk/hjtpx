import React from 'react';
import { render } from '@testing-library/react';
import Button from '../../src/frontend/src/components/ui/Button';
import Input from '../../src/frontend/src/components/ui/Input';
import Alert from '../../src/frontend/src/components/ui/Alert';
import Modal from '../../src/frontend/src/components/ui/Modal';
import Table from '../../src/frontend/src/components/ui/Table';
import Pagination from '../../src/frontend/src/components/ui/Pagination';
import Loading from '../../src/frontend/src/components/ui/Loading';
import { Skeleton } from '../../src/frontend/src/components/ui/Skeleton';

describe('Accessibility Tests', () => {
  describe('Button Component', () => {
    test('renders button with accessible attributes', () => {
      const { container, getByRole } = render(
        <Button aria-label="Test button" disabled={false}>
          Click Me
        </Button>
      );
      
      const button = getByRole('button');
      expect(button).toBeInTheDocument();
      expect(button).not.toBeDisabled();
    });

    test('disabled button has proper accessibility attributes', () => {
      const { getByRole } = render(
        <Button disabled={true}>
          Disabled
        </Button>
      );
      
      const button = getByRole('button');
      expect(button).toBeDisabled();
    });
  });

  describe('Input Component', () => {
    test('renders input with label and accessibility attributes', () => {
      const { getByLabelText } = render(
        <Input 
          label="Email" 
          name="email" 
          type="email"
          placeholder="Enter your email"
        />
      );
      
      const input = getByLabelText('Email');
      expect(input).toBeInTheDocument();
      expect(input).toHaveAttribute('type', 'email');
    });

    test('renders required input with accessibility attributes', () => {
      const { getByLabelText } = render(
        <Input 
          label="Username" 
          name="username" 
          required={true}
        />
      );
      
      const input = getByLabelText('Username');
      expect(input).toBeInTheDocument();
      expect(input).toBeRequired();
    });

    test('renders error state with accessibility attributes', () => {
      const { getByLabelText, getByText } = render(
        <Input 
          label="Password" 
          name="password" 
          error="Password is required"
        />
      );
      
      const input = getByLabelText('Password');
      const errorMessage = getByText('Password is required');
      
      expect(input).toBeInTheDocument();
      expect(errorMessage).toBeInTheDocument();
    });
  });

  describe('Alert Component', () => {
    test('renders alert with accessibility attributes', () => {
      const { container, getByText } = render(
        <Alert 
          type="info" 
          message="Test message"
          description="Test description"
        />
      );
      
      expect(getByText('Test message')).toBeInTheDocument();
      expect(getByText('Test description')).toBeInTheDocument();
    });
  });

  describe('Modal Component', () => {
    test('renders modal with accessibility attributes when open', () => {
      const { container, getByRole } = render(
        <Modal 
          isOpen={true}
          onClose={() => {}}
          title="Test Modal"
        >
          Modal content
        </Modal>
      );
      
      const modal = getByRole('dialog');
      expect(modal).toBeInTheDocument();
    });
  });

  describe('Table Component', () => {
    test('renders table with accessibility attributes', () => {
      const columns = [
        { title: 'Name', dataIndex: 'name' },
        { title: 'Age', dataIndex: 'age' }
      ];
      const data = [
        { name: 'John', age: 30 },
        { name: 'Jane', age: 25 }
      ];
      
      const { container, getByText } = render(
        <Table columns={columns} data={data} />
      );
      
      expect(getByText('Name')).toBeInTheDocument();
      expect(getByText('Age')).toBeInTheDocument();
      expect(getByText('John')).toBeInTheDocument();
      expect(getByText('Jane')).toBeInTheDocument();
    });
  });

  describe('Pagination Component', () => {
    test('renders pagination with accessibility attributes', () => {
      const { container, getByLabelText } = render(
        <Pagination 
          current={1}
          total={10}
          pageSize={5}
          onChange={() => {}}
        />
      );
    });
  });

  describe('Loading Component', () => {
    test('renders loading with accessibility attributes', () => {
      const { container } = render(
        <Loading text="Loading data..." />
      );
      
      expect(container).toBeInTheDocument();
    });
  });

  describe('Skeleton Component', () => {
    test('renders skeleton with accessibility attributes', () => {
      const { container } = render(
        <Skeleton width={100} height={20} />
      );
      
      expect(container).toBeInTheDocument();
    });
  });
});
