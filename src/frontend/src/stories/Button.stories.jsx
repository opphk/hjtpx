import React from 'react';
import Button from '../components/ui/Button';
import { action } from '@storybook/addon-actions';

export default {
  title: 'UI/Button',
  component: Button,
  tags: ['autodocs'],
  argTypes: {
    variant: {
      control: 'select',
      options: ['primary', 'secondary', 'success', 'warning', 'danger', 'outline'],
      description: '按钮的视觉风格',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: 'primary' }
      }
    },
    size: {
      control: 'radio',
      options: ['small', 'medium', 'large'],
      description: '按钮尺寸',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: 'medium' }
      }
    },
    disabled: {
      control: 'boolean',
      description: '禁用状态',
      table: {
        type: { summary: 'boolean' },
        defaultValue: { summary: 'false' }
      }
    },
    loading: {
      control: 'boolean',
      description: '加载状态',
      table: {
        type: { summary: 'boolean' },
        defaultValue: { summary: 'false' }
      }
    },
    type: {
      control: 'select',
      options: ['button', 'submit', 'reset'],
      description: '按钮类型',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: 'button' }
      }
    },
    onClick: {
      action: 'clicked',
      description: '点击事件处理函数'
    },
    children: {
      control: 'text',
      description: '按钮内容'
    }
  },
  parameters: {
    docs: {
      description: {
        component: '通用按钮组件，支持多种变体和尺寸。'
      }
    }
  }
};

const Template = (args) => <Button {...args} />;

export const Primary = Template.bind({});
Primary.args = {
  variant: 'primary',
  children: '主要按钮',
  onClick: action('Button clicked')
};

export const Secondary = Template.bind({});
Secondary.args = {
  variant: 'secondary',
  children: '次要按钮'
};

export const Success = Template.bind({});
Success.args = {
  variant: 'success',
  children: '成功按钮'
};

export const Warning = Template.bind({});
Warning.args = {
  variant: 'warning',
  children: '警告按钮'
};

export const Danger = Template.bind({});
Danger.args = {
  variant: 'danger',
  children: '危险按钮'
};

export const Outline = Template.bind({});
Outline.args = {
  variant: 'outline',
  children: '轮廓按钮'
};

export const Small = Template.bind({});
Small.args = {
  size: 'small',
  children: '小按钮'
};

export const Medium = Template.bind({});
Medium.args = {
  size: 'medium',
  children: '中按钮'
};

export const Large = Template.bind({});
Large.args = {
  size: 'large',
  children: '大按钮'
};

export const Disabled = Template.bind({});
Disabled.args = {
  disabled: true,
  children: '禁用按钮'
};

export const Loading = Template.bind({});
Loading.args = {
  loading: true,
  children: '加载中...'
};

export const AllVariants = () => (
  <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap' }}>
    <Button variant="primary">主要</Button>
    <Button variant="secondary">次要</Button>
    <Button variant="success">成功</Button>
    <Button variant="warning">警告</Button>
    <Button variant="danger">危险</Button>
    <Button variant="outline">轮廓</Button>
  </div>
);

AllVariants.parameters = {
  docs: {
    description: {
      story: '所有按钮变体示例'
    }
  }
};
