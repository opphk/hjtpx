import React, { useState } from 'react';
import Alert from './Alert';

export default {
  title: 'UI/Alert',
  component: Alert,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: '警告提示组件，用于显示重要信息。支持多种类型、可关闭和自定义内容。',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    type: {
      control: 'select',
      options: ['info', 'success', 'warning', 'error'],
      description: '警告提示的类型',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: 'info' },
      },
    },
    message: {
      control: 'text',
      description: '警告提示的主要消息',
      table: {
        type: { summary: 'string' },
      },
    },
    description: {
      control: 'text',
      description: '警告提示的详细描述',
      table: {
        type: { summary: 'string' },
      },
    },
    closable: {
      control: 'boolean',
      description: '是否显示关闭按钮',
      table: {
        type: { summary: 'boolean' },
        defaultValue: { summary: 'false' },
      },
    },
    onClose: {
      action: 'closed',
      description: '关闭按钮点击时的回调函数',
      table: {
        type: { summary: 'function' },
      },
    },
  },
};

export const Info = {
  args: {
    type: 'info',
    message: '这是一条信息提示',
  },
  parameters: {
    docs: {
      description: {
        story: '信息提示，用于普通信息的展示。',
      },
    },
  },
};

export const Success = {
  args: {
    type: 'success',
    message: '操作成功！',
  },
  parameters: {
    docs: {
      description: {
        story: '成功提示，用于操作成功后的反馈。',
      },
    },
  },
};

export const Warning = {
  args: {
    type: 'warning',
    message: '请注意此操作',
  },
  parameters: {
    docs: {
      description: {
        story: '警告提示，用于需要用户注意的情况。',
      },
    },
  },
};

export const Error = {
  args: {
    type: 'error',
    message: '操作失败！',
  },
  parameters: {
    docs: {
      description: {
        story: '错误提示，用于操作失败或错误情况的展示。',
      },
    },
  },
};

export const WithDescription = {
  args: {
    type: 'info',
    message: '提示标题',
    description: '这是详细的描述信息，用于更完整地说明问题。',
  },
  parameters: {
    docs: {
      description: {
        story: '带有详细描述的警告提示。',
      },
    },
  },
};

export const Closable = {
  render: () => {
    const [visible, setVisible] = useState(true);
    
    if (!visible) return (
      <button onClick={() => setVisible(true)}>
        重新显示 Alert
      </button>
    );
    
    return (
      <Alert
        type="info"
        message="可关闭的提示"
        description="点击右上角的关闭按钮可以隐藏此提示。"
        closable
        onClose={() => setVisible(false)}
      />
    );
  },
  parameters: {
    docs: {
      description: {
        story: '可关闭的警告提示，点击关闭按钮后隐藏。',
      },
      code: `
const [visible, setVisible] = useState(true);

<Alert
  type="info"
  message="可关闭的提示"
  closable
  onClose={() => setVisible(false)}
/>
      `,
    },
  },
};

export const SuccessWithDescription = {
  args: {
    type: 'success',
    message: '注册成功',
    description: '您的账号已成功创建，现在可以登录使用了。',
  },
  parameters: {
    docs: {
      description: {
        story: '成功提示，带有详细描述。',
      },
    },
  },
};

export const WarningWithDescription = {
  args: {
    type: 'warning',
    message: '存储空间不足',
    description: '您的账户已使用 95% 的存储空间。建议清理不需要的文件或升级到更高容量计划。',
  },
  parameters: {
    docs: {
      description: {
        story: '警告提示，带有详细描述和操作建议。',
      },
    },
  },
};

export const ErrorWithDescription = {
  args: {
    type: 'error',
    message: '网络连接失败',
    description: '无法连接到服务器。请检查您的网络连接后重试。如果问题持续存在，请联系技术支持。',
  },
  parameters: {
    docs: {
      description: {
        story: '错误提示，带有详细的问题描述和解决建议。',
      },
    },
  },
};

export const AllTypes = {
  render: () => (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px', width: '500px' }}>
      <Alert type="info" message="信息提示" description="这是一条普通的信息提示" />
      <Alert type="success" message="成功提示" description="操作成功完成！" />
      <Alert type="warning" message="警告提示" description="请注意，此操作可能有风险！" />
      <Alert type="error" message="错误提示" description="操作失败，请重试！" />
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: '所有类型警告提示的展示。',
      },
    },
  },
};

export const FormValidation = {
  render: () => {
    const [errors, setErrors] = useState({
      email: '',
      password: '',
    });

    const validate = () => {
      const newErrors = {};
      if (!errors.email) {
        newErrors.email = '请输入邮箱地址';
      } else if (!/\S+@\S+\.\S+/.test(errors.email)) {
        newErrors.email = '请输入有效的邮箱格式';
      }
      if (!errors.password) {
        newErrors.password = '请输入密码';
      } else if (errors.password.length < 6) {
        newErrors.password = '密码长度至少为 6 位';
      }
      setErrors(newErrors);
      return Object.keys(newErrors).length === 0;
    };

    const handleSubmit = (e) => {
      e.preventDefault();
      if (!validate()) {
        return;
      }
      alert('表单验证通过！');
    };

    return (
      <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px', width: '400px' }}>
        {errors.email && (
          <Alert type="error" message="邮箱错误" description={errors.email} closable />
        )}
        {errors.password && (
          <Alert type="error" message="密码错误" description={errors.password} closable />
        )}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          <label>邮箱</label>
          <input
            type="email"
            value={errors.email}
            onChange={(e) => setErrors({ ...errors, email: e.target.value })}
            placeholder="请输入邮箱"
          />
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          <label>密码</label>
          <input
            type="password"
            value={errors.password}
            onChange={(e) => setErrors({ ...errors, password: e.target.value })}
            placeholder="请输入密码"
          />
        </div>
        <button type="submit">提交</button>
      </form>
    );
  },
  parameters: {
    docs: {
      description: {
        story: '表单验证中使用警告提示的示例。',
      },
    },
  },
};
