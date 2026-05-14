import React from 'react';
import Input from '../components/ui/Input';
import { action } from '@storybook/addon-actions';

export default {
  title: 'UI/Input',
  component: Input,
  tags: ['autodocs'],
  argTypes: {
    label: {
      control: 'text',
      description: '输入框标签',
      table: {
        type: { summary: 'string' }
      }
    },
    type: {
      control: 'select',
      options: ['text', 'password', 'email', 'number', 'tel', 'url'],
      description: '输入框类型',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: 'text' }
      }
    },
    name: {
      control: 'text',
      description: '输入框名称（用于表单）',
      table: {
        type: { summary: 'string' }
      }
    },
    value: {
      control: 'text',
      description: '输入框值',
      table: {
        type: { summary: 'string' }
      }
    },
    placeholder: {
      control: 'text',
      description: '占位符文本',
      table: {
        type: { summary: 'string' }
      }
    },
    error: {
      control: 'text',
      description: '错误信息',
      table: {
        type: { summary: 'string' }
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
    required: {
      control: 'boolean',
      description: '必填标记',
      table: {
        type: { summary: 'boolean' },
        defaultValue: { summary: 'false' }
      }
    },
    onChange: {
      action: 'changed',
      description: '值变更事件'
    }
  },
  parameters: {
    docs: {
      description: {
        component: '通用输入框组件，支持标签、验证和错误提示。'
      }
    }
  }
};

const Template = (args) => <Input {...args} />;

export const Basic = Template.bind({});
Basic.args = {
  label: '用户名',
  name: 'username',
  placeholder: '请输入用户名'
};

export const WithError = Template.bind({});
WithError.args = {
  label: '邮箱',
  name: 'email',
  type: 'email',
  placeholder: '请输入邮箱',
  error: '请输入有效的邮箱地址'
};

export const Required = Template.bind({});
Required.args = {
  label: '密码',
  name: 'password',
  type: 'password',
  required: true,
  placeholder: '请输入密码'
};

export const Disabled = Template.bind({});
Disabled.args = {
  label: '禁用输入框',
  name: 'disabled',
  disabled: true,
  value: '不可编辑的内容'
};

export const Password = Template.bind({});
Password.args = {
  label: '密码',
  name: 'password',
  type: 'password',
  placeholder: '请输入密码'
};

export const AllExamples = () => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
    <Input label="基本输入" name="basic" placeholder="基本输入框" />
    <Input label="邮箱输入" name="email" type="email" placeholder="请输入邮箱" />
    <Input label="必填字段" name="required" required placeholder="这是必填字段" />
    <Input label="错误状态" name="error" error="这是一个错误信息" placeholder="输入内容" />
    <Input label="禁用状态" name="disabled" disabled value="不可编辑" />
  </div>
);

AllExamples.parameters = {
  docs: {
    description: {
      story: '输入框的各种状态示例'
    }
  }
};
