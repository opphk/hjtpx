import React from 'react';
import Form from './Form';

export default {
  title: 'UI/Form',
  component: Form,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: '## 描述\n\nForm 组件是一个灵活的表单组件，支持多种配置选项，包括字段定义、验证规则和自定义样式。\n\n### 主要功能\n\n- 支持多种输入类型\n- 内置验证规则（必填、最小/最大长度、正则表达式）\n- 支持自定义验证函数\n- 实时验证和提交时验证\n- 表单重置功能\n- 无障碍支持（ARIA 属性）',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    fields: {
      control: 'array',
      description: '表单字段配置数组',
    },
    initialValues: {
      control: 'object',
      description: '表单初始值',
    },
    validationSchema: {
      control: 'object',
      description: '验证规则配置',
    },
    onSubmit: {
      action: 'submitted',
      description: '表单提交回调函数',
    },
    submitText: {
      control: 'text',
      description: '提交按钮文本',
    },
    loading: {
      control: 'boolean',
      description: '是否显示加载状态',
    },
    disabled: {
      control: 'boolean',
      description: '是否禁用表单',
    },
  },
};

const Template = (args) => {
  const handleSubmit = (data) => {
    console.log('表单数据:', data);
    args.onSubmit?.(data);
  };

  return <Form {...args} onSubmit={handleSubmit} />;
};

export const 基础表单 = Template.bind({});
基础表单.args = {
  fields: [
    {
      name: 'username',
      label: '用户名',
      placeholder: '请输入用户名',
      type: 'text',
      required: true,
    },
    {
      name: 'email',
      label: '邮箱',
      placeholder: '请输入邮箱',
      type: 'email',
      required: true,
    },
    {
      name: 'password',
      label: '密码',
      placeholder: '请输入密码',
      type: 'password',
      required: true,
    },
  ],
  submitText: '提交',
};

export const 必填字段 = Template.bind({});
必填字段.args = {
  fields: [
    {
      name: 'name',
      label: '姓名',
      placeholder: '请输入姓名',
      required: true,
    },
    {
      name: 'phone',
      label: '电话',
      placeholder: '请输入电话号码',
      required: true,
    },
  ],
  submitText: '验证',
};

export const 带初始值 = Template.bind({});
带初始值.args = {
  fields: [
    {
      name: 'email',
      label: '邮箱',
      type: 'email',
      required: true,
    },
    {
      name: 'bio',
      label: '简介',
      placeholder: '请输入个人简介',
    },
  ],
  initialValues: {
    email: 'user@example.com',
    bio: '这是默认的个人简介',
  },
  submitText: '更新',
};

export const 加载状态 = Template.bind({});
加载状态.args = {
  fields: [
    {
      name: 'username',
      label: '用户名',
      required: true,
    },
  ],
  loading: true,
  submitText: '提交中...',
};

export const 自定义验证规则 = Template.bind({});
自定义验证规则.args = {
  fields: [
    {
      name: 'username',
      label: '用户名',
      placeholder: '至少2个字符',
      validation: {
        required: true,
        minLength: 2,
        message: '用户名至少需要2个字符',
      },
    },
    {
      name: 'password',
      label: '密码',
      type: 'password',
      placeholder: '至少8个字符，包含数字和字母',
      validation: {
        required: true,
        minLength: 8,
        message: '密码至少需要8个字符',
      },
    },
  ],
  submitText: '注册',
};

export const 登录表单 = Template.bind({});
登录表单.args = {
  fields: [
    {
      name: 'email',
      label: '邮箱',
      type: 'email',
      placeholder: '请输入邮箱',
      required: true,
    },
    {
      name: 'password',
      label: '密码',
      type: 'password',
      placeholder: '请输入密码',
      required: true,
    },
  ],
  submitText: '登录',
  className: 'auth-form',
};

export const 注册表单 = Template.bind({});
注册表单.args = {
  fields: [
    {
      name: 'name',
      label: '姓名',
      placeholder: '请输入真实姓名',
      required: true,
    },
    {
      name: 'email',
      label: '邮箱',
      type: 'email',
      placeholder: '请输入邮箱地址',
      required: true,
    },
    {
      name: 'password',
      label: '密码',
      type: 'password',
      placeholder: '请输入密码（至少8位）',
      required: true,
    },
    {
      name: 'confirmPassword',
      label: '确认密码',
      type: 'password',
      placeholder: '请再次输入密码',
      required: true,
    },
  ],
  submitText: '注册',
};

export const 联系表单 = Template.bind({});
联系表单.args = {
  fields: [
    {
      name: 'name',
      label: '姓名',
      placeholder: '请输入您的姓名',
      required: true,
    },
    {
      name: 'email',
      label: '邮箱',
      type: 'email',
      placeholder: '请输入邮箱地址',
      required: true,
    },
    {
      name: 'subject',
      label: '主题',
      placeholder: '请输入邮件主题',
      required: true,
    },
    {
      name: 'message',
      label: '留言内容',
      placeholder: '请输入您的留言内容',
      required: true,
    },
  ],
  submitText: '发送消息',
};
