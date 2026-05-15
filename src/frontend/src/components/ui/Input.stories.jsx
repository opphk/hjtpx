import React, { useState } from 'react';
import Input from './Input';

export default {
  title: 'UI/Input',
  component: Input,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: '输入框组件，用于用户输入文本。支持多种类型、验证和错误提示。',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    type: {
      control: 'select',
      options: ['text', 'email', 'password', 'number', 'tel', 'url'],
      description: '输入框类型',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: 'text' },
      },
    },
    label: {
      control: 'text',
      description: '输入框标签',
      table: {
        type: { summary: 'string' },
      },
    },
    name: {
      control: 'text',
      description: '输入框名称',
      table: {
        type: { summary: 'string' },
      },
    },
    value: {
      control: 'text',
      description: '输入框值（受控组件）',
      table: {
        type: { summary: 'string' },
      },
    },
    placeholder: {
      control: 'text',
      description: '占位符文本',
      table: {
        type: { summary: 'string' },
      },
    },
    disabled: {
      control: 'boolean',
      description: '是否禁用',
      table: {
        type: { summary: 'boolean' },
        defaultValue: { summary: 'false' },
      },
    },
    required: {
      control: 'boolean',
      description: '是否为必填字段',
      table: {
        type: { summary: 'boolean' },
        defaultValue: { summary: 'false' },
      },
    },
    error: {
      control: 'text',
      description: '错误提示信息',
      table: {
        type: { summary: 'string' },
      },
    },
    onChange: {
      action: 'changed',
      description: '值变化时的回调函数',
      table: {
        type: { summary: 'function' },
      },
    },
  },
};

const Template = (args) => {
  const [value, setValue] = useState(args.value || '');
  return (
    <Input
      {...args}
      value={value}
      onChange={(e) => {
        setValue(e.target.value);
        if (args.onChange) args.onChange(e);
      }}
    />
  );
};

export const Default = Template.bind({});
Default.args = {
  label: '用户名',
  placeholder: '请输入用户名',
  name: 'username',
};
Default.parameters = {
  docs: {
    description: {
      story: '默认输入框，显示标签和占位符。',
    },
  },
};

export const WithValue = Template.bind({});
WithValue.args = {
  label: '邮箱',
  placeholder: '请输入邮箱',
  type: 'email',
  value: 'user@example.com',
  name: 'email',
};
WithValue.parameters = {
  docs: {
    description: {
      story: '带有默认值的输入框。',
    },
  },
};

export const WithError = Template.bind({});
WithError.args = {
  label: '密码',
  type: 'password',
  placeholder: '请输入密码',
  error: '密码长度至少为 6 位',
  name: 'password',
};
WithError.parameters = {
  docs: {
    description: {
      story: '带有错误提示的输入框，通常用于表单验证失败的情况。',
    },
  },
};

export const Disabled = Template.bind({});
Disabled.args = {
  label: '禁用输入框',
  placeholder: '无法输入',
  disabled: true,
  name: 'disabled',
  value: '已被禁用',
};
Disabled.parameters = {
  docs: {
    description: {
      story: '禁用状态下的输入框，无法编辑。',
    },
  },
};

export const Required = Template.bind({});
Required.args = {
  label: '必填字段',
  placeholder: '这是必填字段',
  required: true,
  name: 'required',
};
Required.parameters = {
  docs: {
    description: {
      story: '必填字段的输入框，显示必填标记。',
    },
  },
};

export const Password = Template.bind({});
Password.args = {
  label: '密码',
  type: 'password',
  placeholder: '请输入密码',
  name: 'password',
};
Password.parameters = {
  docs: {
    description: {
      story: '密码输入框，内容以掩码形式显示。',
    },
  },
};

export const Number = Template.bind({});
Number.args = {
  label: '数量',
  type: 'number',
  placeholder: '请输入数量',
  name: 'quantity',
};
Number.parameters = {
  docs: {
    description: {
      story: '数字输入框，只能输入数字。',
    },
  },
};

export const Telephone = Template.bind({});
Telephone.args = {
  label: '电话',
  type: 'tel',
  placeholder: '请输入电话号码',
  name: 'phone',
};
Telephone.parameters = {
  docs: {
    description: {
      story: '电话输入框，用于输入电话号码。',
    },
  },
};

export const URL = Template.bind({});
URL.args = {
  label: '网站地址',
  type: 'url',
  placeholder: 'https://example.com',
  name: 'website',
};
URL.parameters = {
  docs: {
    description: {
      story: 'URL输入框，用于输入网址。',
    },
  },
};

export const AllTypes = {
  render: () => {
    const [values, setValues] = useState({
      text: '',
      email: '',
      password: '',
      number: '',
    });

    const handleChange = (name) => (e) => {
      setValues({ ...values, [name]: e.target.value });
    };

    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: '24px', maxWidth: '400px' }}>
        <Input
          label="文本输入"
          type="text"
          placeholder="请输入文本"
          name="text"
          value={values.text}
          onChange={handleChange('text')}
        />
        <Input
          label="邮箱"
          type="email"
          placeholder="请输入邮箱"
          name="email"
          value={values.email}
          onChange={handleChange('email')}
        />
        <Input
          label="密码"
          type="password"
          placeholder="请输入密码"
          name="password"
          value={values.password}
          onChange={handleChange('password')}
        />
        <Input
          label="数字"
          type="number"
          placeholder="请输入数字"
          name="number"
          value={values.number}
          onChange={handleChange('number')}
        />
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: '展示所有类型的输入框。',
      },
    },
  },
};

export const LoginForm = {
  render: () => {
    const [values, setValues] = useState({
      email: '',
      password: '',
    });
    const [errors, setErrors] = useState({});

    const handleChange = (name) => (e) => {
      setValues({ ...values, [name]: e.target.value });
      if (errors[name]) {
        setErrors({ ...errors, [name]: '' });
      }
    };

    const handleSubmit = (e) => {
      e.preventDefault();
      const newErrors = {};
      if (!values.email) {
        newErrors.email = '请输入邮箱';
      }
      if (!values.password) {
        newErrors.password = '请输入密码';
      } else if (values.password.length < 6) {
        newErrors.password = '密码长度至少为 6 位';
      }
      setErrors(newErrors);
      if (Object.keys(newErrors).length === 0) {
        alert('表单验证通过！');
      }
    };

    return (
      <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px', maxWidth: '300px' }}>
        <Input
          label="邮箱"
          type="email"
          name="email"
          placeholder="请输入邮箱"
          value={values.email}
          onChange={handleChange('email')}
          error={errors.email}
        />
        <Input
          label="密码"
          type="password"
          name="password"
          placeholder="请输入密码"
          value={values.password}
          onChange={handleChange('password')}
          error={errors.password}
        />
        <button type="submit">登录</button>
      </form>
    );
  },
  parameters: {
    docs: {
      description: {
        story: '使用Input组件构建的登录表单示例，包含验证逻辑。',
      },
      code: `
const [values, setValues] = useState({ email: '', password: '' });
const [errors, setErrors] = useState({});

const handleSubmit = (e) => {
  e.preventDefault();
  // 验证逻辑...
};

<form onSubmit={handleSubmit}>
  <Input
    label="邮箱"
    type="email"
    name="email"
    error={errors.email}
  />
  <Input
    label="密码"
    type="password"
    name="password"
    error={errors.password}
  />
</form>
      `,
    },
  },
};
