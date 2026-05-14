import React from 'react';
import Input from '../components/Input';

export default {
  title: 'Components/Input',
  component: Input,
  parameters: {
    docs: {
      description: {
        component: '输入框组件，支持多种类型、标签和错误状态。',
      },
    },
  },
  argTypes: {
    type: {
      control: { type: 'select' },
      options: ['text', 'email', 'password', 'number', 'tel', 'url'],
      description: '输入框类型',
    },
    label: {
      control: 'text',
      description: '输入框标签',
    },
    placeholder: {
      control: 'text',
      description: '占位符文本',
    },
    error: {
      control: 'text',
      description: '错误信息',
    },
    required: {
      control: 'boolean',
      description: '是否为必填项',
    },
    disabled: {
      control: 'boolean',
      description: '禁用状态',
    },
  },
  tags: ['autodocs'],
};

export const Default = {
  args: {
    placeholder: 'Enter text...',
  },
};

export const WithLabel = {
  args: {
    label: 'Email Address',
    type: 'email',
    placeholder: 'example@email.com',
  },
};

export const Required = {
  args: {
    label: 'Required Field',
    required: true,
    placeholder: 'This field is required',
  },
};

export const Disabled = {
  args: {
    label: 'Disabled Input',
    disabled: true,
    value: 'Disabled value',
  },
};

export const WithError = {
  args: {
    label: 'Email Address',
    type: 'email',
    value: 'invalid-email',
    error: 'Please enter a valid email address',
  },
};

export const Password = {
  args: {
    label: 'Password',
    type: 'password',
    placeholder: 'Enter password',
  },
};

export const FormExample = {
  render: () => (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '20px', maxWidth: '400px' }}>
      <Input
        label="Username"
        name="username"
        placeholder="Enter username"
        required
      />
      <Input
        label="Email"
        name="email"
        type="email"
        placeholder="Enter email"
        required
      />
      <Input
        label="Password"
        name="password"
        type="password"
        placeholder="Enter password"
        required
      />
    </div>
  ),
};

export const ErrorStates = {
  render: () => (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '20px', maxWidth: '400px' }}>
      <Input
        label="Email"
        type="email"
        value="invalid"
        error="Invalid email format"
      />
      <Input
        label="Required Field"
        error="This field is required"
      />
      <Input
        label="Phone Number"
        type="tel"
        value="abc"
        error="Please enter a valid phone number"
      />
    </div>
  ),
};
