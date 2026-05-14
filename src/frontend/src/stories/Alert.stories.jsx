import React from 'react';
import Alert from '../components/ui/Alert';

export default {
  title: 'UI/Alert',
  component: Alert,
  tags: ['autodocs'],
  argTypes: {
    type: {
      control: 'select',
      options: ['info', 'success', 'warning', 'error'],
      description: '警告框类型',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: 'info' }
      }
    },
    message: {
      control: 'text',
      description: '主要消息内容',
      table: {
        type: { summary: 'string' }
      }
    },
    description: {
      control: 'text',
      description: '详细描述信息',
      table: {
        type: { summary: 'string' }
      }
    },
    closable: {
      control: 'boolean',
      description: '是否可关闭',
      table: {
        type: { summary: 'boolean' },
        defaultValue: { summary: 'false' }
      }
    },
    onClose: {
      action: 'closed',
      description: '关闭事件处理函数'
    }
  },
  parameters: {
    docs: {
      description: {
        component: '信息提示警告框组件，支持多种类型和可关闭功能。'
      }
    }
  }
};

const Template = (args) => <Alert {...args} />;

export const Info = Template.bind({});
Info.args = {
  type: 'info',
  message: '提示信息',
  description: '这是一条普通的信息提示'
};

export const Success = Template.bind({});
Success.args = {
  type: 'success',
  message: '操作成功',
  description: '您的请求已成功完成'
};

export const Warning = Template.bind({});
Warning.args = {
  type: 'warning',
  message: '警告提示',
  description: '请注意此操作的潜在风险'
};

export const Error = Template.bind({});
Error.args = {
  type: 'error',
  message: '错误提示',
  description: '操作失败，请重试'
};

export const Closable = Template.bind({});
Closable.args = {
  type: 'info',
  message: '可关闭的提示',
  description: '点击右侧关闭按钮可以移除此提示',
  closable: true
};

export const AllTypes = () => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
    <Alert type="info" message="信息提示" description="普通信息提示内容" />
    <Alert type="success" message="成功提示" description="操作已成功完成" />
    <Alert type="warning" message="警告提示" description="请注意此操作的潜在风险" />
    <Alert type="error" message="错误提示" description="操作失败，请检查输入" />
  </div>
);

AllTypes.parameters = {
  docs: {
    description: {
      story: '所有警告框类型示例'
    }
  }
};

export const WithDescription = Template.bind({});
WithDescription.args = {
  type: 'info',
  message: '重要通知',
  description: '这是一条带有详细描述的提示信息，用于提供更多上下文和说明。'
};
