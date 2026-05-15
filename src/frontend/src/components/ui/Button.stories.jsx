import React from 'react';
import Button from './Button';

export default {
  title: 'UI/Button',
  component: Button,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: '按钮组件，用于触发操作。支持多种变体、大小和状态。',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    variant: {
      control: 'select',
      options: ['primary', 'secondary', 'danger', 'success', 'warning'],
      description: '按钮的视觉风格',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: 'primary' },
      },
    },
    size: {
      control: 'select',
      options: ['small', 'medium', 'large'],
      description: '按钮尺寸',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: 'medium' },
      },
    },
    disabled: {
      control: 'boolean',
      description: '是否禁用按钮',
      table: {
        type: { summary: 'boolean' },
        defaultValue: { summary: 'false' },
      },
    },
    loading: {
      control: 'boolean',
      description: '是否显示加载状态',
      table: {
        type: { summary: 'boolean' },
        defaultValue: { summary: 'false' },
      },
    },
    type: {
      control: 'select',
      options: ['button', 'submit', 'reset'],
      description: '按钮类型',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: 'button' },
      },
    },
    onClick: {
      action: 'clicked',
      description: '点击事件处理函数',
      table: {
        type: { summary: 'function' },
      },
    },
    children: {
      control: 'text',
      description: '按钮内容',
      table: {
        type: { summary: 'ReactNode' },
      },
    },
  },
};

export const Primary = {
  args: {
    variant: 'primary',
    children: '主按钮',
  },
  parameters: {
    docs: {
      description: {
        story: '主要按钮，用于主要操作。',
      },
    },
  },
};

export const Secondary = {
  args: {
    variant: 'secondary',
    children: '次按钮',
  },
  parameters: {
    docs: {
      description: {
        story: '次要按钮，用于次要操作。',
      },
    },
  },
};

export const Danger = {
  args: {
    variant: 'danger',
    children: '危险按钮',
  },
  parameters: {
    docs: {
      description: {
        story: '危险操作按钮，用于删除等危险操作。',
      },
    },
  },
};

export const Success = {
  args: {
    variant: 'success',
    children: '成功按钮',
  },
  parameters: {
    docs: {
      description: {
        story: '成功操作按钮，用于成功状态的操作。',
      },
    },
  },
};

export const Warning = {
  args: {
    variant: 'warning',
    children: '警告按钮',
  },
  parameters: {
    docs: {
      description: {
        story: '警告按钮，用于需要特别注意的操作。',
      },
    },
  },
};

export const Small = {
  args: {
    size: 'small',
    children: '小按钮',
  },
  parameters: {
    docs: {
      description: {
        story: '小尺寸按钮，适合紧凑的布局。',
      },
    },
  },
};

export const Medium = {
  args: {
    size: 'medium',
    children: '中按钮',
  },
  parameters: {
    docs: {
      description: {
        story: '中等尺寸按钮，默认大小。',
      },
    },
  },
};

export const Large = {
  args: {
    size: 'large',
    children: '大按钮',
  },
  parameters: {
    docs: {
      description: {
        story: '大尺寸按钮，适合突出的操作。',
      },
    },
  },
};

export const Disabled = {
  args: {
    disabled: true,
    children: '禁用按钮',
  },
  parameters: {
    docs: {
      description: {
        story: '禁用状态下的按钮，无法点击。',
      },
    },
  },
};

export const Loading = {
  args: {
    loading: true,
    children: '加载中',
  },
  parameters: {
    docs: {
      description: {
        story: '加载状态下的按钮，显示加载指示器。',
      },
    },
  },
};

export const WithClickHandler = {
  render: () => {
    const handleClick = () => {
      alert('按钮被点击了！');
    };
    return (
      <Button variant="primary" onClick={handleClick}>
        点击我
      </Button>
    );
  },
  parameters: {
    docs: {
      description: {
        story: '带有点击事件处理的按钮。',
      },
      code: `
const handleClick = () => {
  alert('按钮被点击了！');
};

<Button variant="primary" onClick={handleClick}>
  点击我
</Button>
      `,
    },
  },
};

export const AllVariants = {
  render: () => (
    <div style={{ display: 'flex', gap: '12px', flexWrap: 'wrap' }}>
      <Button variant="primary">主按钮</Button>
      <Button variant="secondary">次按钮</Button>
      <Button variant="success">成功按钮</Button>
      <Button variant="warning">警告按钮</Button>
      <Button variant="danger">危险按钮</Button>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: '所有按钮变体的展示。',
      },
    },
  },
};

export const AllSizes = {
  render: () => (
    <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
      <Button size="small">小按钮</Button>
      <Button size="medium">中按钮</Button>
      <Button size="large">大按钮</Button>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: '所有按钮尺寸的展示。',
      },
    },
  },
};

export const ButtonGroup = {
  render: () => (
    <div style={{ display: 'flex', gap: '12px' }}>
      <Button variant="secondary">取消</Button>
      <Button variant="primary">确认</Button>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: '按钮组示例，用于表单操作。',
      },
      code: `
<div style={{ display: 'flex', gap: '12px' }}>
  <Button variant="secondary">取消</Button>
  <Button variant="primary">确认</Button>
</div>
      `,
    },
  },
};
