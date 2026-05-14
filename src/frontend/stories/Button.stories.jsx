import React from 'react';
import Button from '../components/Button';

export default {
  title: 'Components/Button',
  component: Button,
  parameters: {
    docs: {
      description: {
        component: '按钮组件，支持多种变体、大小和状态。',
      },
    },
  },
  argTypes: {
    variant: {
      control: { type: 'select' },
      options: ['primary', 'secondary', 'success', 'danger', 'outline'],
      description: '按钮变体样式',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: 'primary' },
      },
    },
    size: {
      control: { type: 'select' },
      options: ['small', 'medium', 'large'],
      description: '按钮尺寸',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: 'medium' },
      },
    },
    loading: {
      control: { type: 'boolean' },
      description: '加载状态',
    },
    disabled: {
      control: { type: 'boolean' },
      description: '禁用状态',
    },
    type: {
      control: { type: 'select' },
      options: ['button', 'submit', 'reset'],
      description: '按钮类型',
    },
  },
  tags: ['autodocs'],
};

export const Primary = {
  args: {
    variant: 'primary',
    children: 'Primary Button',
  },
};

export const Secondary = {
  args: {
    variant: 'secondary',
    children: 'Secondary Button',
  },
};

export const Success = {
  args: {
    variant: 'success',
    children: 'Success Button',
  },
};

export const Danger = {
  args: {
    variant: 'danger',
    children: 'Danger Button',
  },
};

export const Outline = {
  args: {
    variant: 'outline',
    children: 'Outline Button',
  },
};

export const Small = {
  args: {
    size: 'small',
    children: 'Small Button',
  },
};

export const Medium = {
  args: {
    size: 'medium',
    children: 'Medium Button',
  },
};

export const Large = {
  args: {
    size: 'large',
    children: 'Large Button',
  },
};

export const Loading = {
  args: {
    loading: true,
    children: 'Loading Button',
  },
};

export const Disabled = {
  args: {
    disabled: true,
    children: 'Disabled Button',
  },
};

export const WithClickHandler = {
  args: {
    children: 'Click Me',
    onClick: () => alert('Button clicked!'),
  },
};

export const AllVariants = {
  render: () => (
    <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap' }}>
      <Button variant="primary">Primary</Button>
      <Button variant="secondary">Secondary</Button>
      <Button variant="success">Success</Button>
      <Button variant="danger">Danger</Button>
      <Button variant="outline">Outline</Button>
    </div>
  ),
};

export const AllSizes = {
  render: () => (
    <div style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
      <Button size="small">Small</Button>
      <Button size="medium">Medium</Button>
      <Button size="large">Large</Button>
    </div>
  ),
};
