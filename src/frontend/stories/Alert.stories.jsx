import React from 'react';
import Alert from '../components/Alert';

export default {
  title: 'Components/Alert',
  component: Alert,
  parameters: {
    docs: {
      description: {
        component: '警告提示组件，支持多种类型、自动关闭和关闭按钮。',
      },
    },
  },
  argTypes: {
    type: {
      control: { type: 'select' },
      options: ['info', 'success', 'warning', 'danger', 'error'],
      description: '警告类型',
    },
    message: {
      control: 'text',
      description: '警告消息内容',
    },
    autoClose: {
      control: 'boolean',
      description: '是否自动关闭',
    },
    autoCloseTime: {
      control: 'number',
      description: '自动关闭时间（毫秒）',
    },
  },
  tags: ['autodocs'],
};

export const Info = {
  args: {
    type: 'info',
    message: 'This is an informational message.',
  },
};

export const Success = {
  args: {
    type: 'success',
    message: 'Operation completed successfully!',
  },
};

export const Warning = {
  args: {
    type: 'warning',
    message: 'Please review your input before proceeding.',
  },
};

export const Danger = {
  args: {
    type: 'danger',
    message: 'An error occurred. Please try again.',
  },
};

export const Error = {
  args: {
    type: 'error',
    message: 'Something went wrong!',
  },
};

export const WithCloseButton = {
  args: {
    type: 'info',
    message: 'You can close this alert manually.',
    onClose: () => console.log('Alert closed'),
    autoClose: false,
  },
};

export const AutoClosing = {
  args: {
    type: 'success',
    message: 'This alert will close in 3 seconds.',
    autoClose: true,
    autoCloseTime: 3000,
  },
};

export const AllTypes = {
  render: () => (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '15px' }}>
      <Alert type="info" message="Information alert message." autoClose={false} />
      <Alert type="success" message="Success alert message." autoClose={false} />
      <Alert type="warning" message="Warning alert message." autoClose={false} />
      <Alert type="danger" message="Danger alert message." autoClose={false} />
      <Alert type="error" message="Error alert message." autoClose={false} />
    </div>
  ),
};
