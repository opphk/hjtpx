import React, { useState } from 'react';
import Modal from './Modal';
import Button from './Button';
import Input from './Input';

export default {
  title: 'UI/Modal',
  component: Modal,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: '模态框组件，用于显示对话框内容。支持多种尺寸、可访问性特性和无障碍操作。',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    isOpen: {
      control: 'boolean',
      description: '是否显示模态框',
      table: {
        type: { summary: 'boolean' },
        defaultValue: { summary: 'false' },
      },
    },
    onClose: {
      action: 'closed',
      description: '关闭模态框的回调函数',
      table: {
        type: { summary: 'function' },
      },
    },
    title: {
      control: 'text',
      description: '模态框标题',
      table: {
        type: { summary: 'string' },
      },
    },
    size: {
      control: 'select',
      options: ['small', 'medium', 'large'],
      description: '模态框尺寸',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: 'medium' },
      },
    },
    children: {
      control: 'text',
      description: '模态框内容',
      table: {
        type: { summary: 'ReactNode' },
      },
    },
    footer: {
      control: 'text',
      description: '模态框底部内容',
      table: {
        type: { summary: 'ReactNode' },
      },
    },
  },
};

const Template = (args) => {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <Button onClick={() => setIsOpen(true)}>打开弹窗</Button>
      <Modal
        {...args}
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
      />
    </>
  );
};

export const Default = Template.bind({});
Default.args = {
  title: '默认弹窗',
  children: <p>这是弹窗的内容区域，可以放置任何内容。</p>,
};
Default.parameters = {
  docs: {
    description: {
      story: '默认模态框，显示标题和内容。',
    },
  },
};

export const WithFooter = Template.bind({});
WithFooter.args = {
  title: '带底部按钮的弹窗',
  children: <p>这是弹窗的内容区域，底部有操作按钮。</p>,
  footer: (
    <div style={{ display: 'flex', gap: '12px' }}>
      <Button variant="secondary" onClick={() => {}}>取消</Button>
      <Button variant="primary" onClick={() => {}}>确定</Button>
    </div>
  ),
};
WithFooter.parameters = {
  docs: {
    description: {
      story: '带有底部操作按钮的模态框。',
    },
  },
};

export const Small = Template.bind({});
Small.args = {
  size: 'small',
  title: '小弹窗',
  children: <p>这是小尺寸的弹窗。</p>,
};
Small.parameters = {
  docs: {
    description: {
      story: '小尺寸的模态框，适合简单的确认操作。',
    },
  },
};

export const Medium = Template.bind({});
Medium.args = {
  size: 'medium',
  title: '中等弹窗',
  children: <p>这是中等尺寸的弹窗，适合大多数场景。</p>,
};
Medium.parameters = {
  docs: {
    description: {
      story: '中等尺寸的模态框，默认大小。',
    },
  },
};

export const Large = Template.bind({});
Large.args = {
  size: 'large',
  title: '大弹窗',
  children: (
    <div>
      <p>这是大尺寸的弹窗，可以容纳更多内容。</p>
      <p style={{ marginTop: '16px' }}>支持放置多行文本、表单等复杂内容。</p>
    </div>
  ),
};
Large.parameters = {
  docs: {
    description: {
      story: '大尺寸的模态框，适合需要更多空间的场景。',
    },
  },
};

export const ConfirmationModal = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);
    const [result, setResult] = useState(null);

    const handleConfirm = () => {
      setResult('已确认');
      setIsOpen(false);
    };

    const handleCancel = () => {
      setResult('已取消');
      setIsOpen(false);
    };

    return (
      <div>
        <Button onClick={() => setIsOpen(true)}>显示确认对话框</Button>
        {result && <p style={{ marginTop: '16px' }}>结果: {result}</p>}
        <Modal
          isOpen={isOpen}
          onClose={handleCancel}
          title="确认操作"
          size="small"
          footer={
            <div style={{ display: 'flex', gap: '12px' }}>
              <Button variant="secondary" onClick={handleCancel}>取消</Button>
              <Button variant="danger" onClick={handleConfirm}>确认删除</Button>
            </div>
          }
        >
          <p>确定要删除此项目吗？此操作无法撤销。</p>
        </Modal>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: '确认对话框示例，用于需要用户确认的操作。',
      },
      code: `
const [isOpen, setIsOpen] = useState(false);

<Modal
  isOpen={isOpen}
  onClose={() => setIsOpen(false)}
  title="确认操作"
  footer={
    <div style={{ display: 'flex', gap: '12px' }}>
      <Button variant="secondary">取消</Button>
      <Button variant="danger">确认删除</Button>
    </div>
  }
>
  <p>确定要删除此项目吗？此操作无法撤销。</p>
</Modal>
      `,
    },
  },
};

export const FormModal = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);
    const [values, setValues] = useState({ name: '', email: '' });

    const handleChange = (name) => (e) => {
      setValues({ ...values, [name]: e.target.value });
    };

    const handleSubmit = (e) => {
      e.preventDefault();
      alert(`提交的数据: ${JSON.stringify(values)}`);
      setIsOpen(false);
    };

    return (
      <div>
        <Button onClick={() => setIsOpen(true)}>添加新用户</Button>
        <Modal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          title="添加用户"
          footer={
            <div style={{ display: 'flex', gap: '12px' }}>
              <Button variant="secondary" onClick={() => setIsOpen(false)}>取消</Button>
              <Button variant="primary" onClick={handleSubmit}>保存</Button>
            </div>
          }
        >
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <Input
              label="姓名"
              name="name"
              value={values.name}
              onChange={handleChange('name')}
              placeholder="请输入姓名"
            />
            <Input
              label="邮箱"
              type="email"
              name="email"
              value={values.email}
              onChange={handleChange('email')}
              placeholder="请输入邮箱"
            />
          </form>
        </Modal>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: '表单模态框示例，包含表单元素和提交逻辑。',
      },
      code: `
const [isOpen, setIsOpen] = useState(false);
const [values, setValues] = useState({ name: '', email: '' });

<Modal
  isOpen={isOpen}
  onClose={() => setIsOpen(false)}
  title="添加用户"
  footer={<Button>保存</Button>}
>
  <form>
    <Input label="姓名" />
    <Input label="邮箱" />
  </form>
</Modal>
      `,
    },
  },
};

export const AllSizes = {
  render: () => {
    const [activeModal, setActiveModal] = useState(null);

    const modals = [
      { size: 'small', title: '小弹窗' },
      { size: 'medium', title: '中等弹窗' },
      { size: 'large', title: '大弹窗' },
    ];

    return (
      <div style={{ display: 'flex', gap: '12px' }}>
        {modals.map((modal) => (
          <div key={modal.size}>
            <Button onClick={() => setActiveModal(modal.size)}>
              打开{modal.title}
            </Button>
            <Modal
              isOpen={activeModal === modal.size}
              onClose={() => setActiveModal(null)}
              title={modal.title}
              size={modal.size}
              footer={
                <Button variant="secondary" onClick={() => setActiveModal(null)}>
                  关闭
                </Button>
              }
            >
              <p>这是{modal.title}的内容区域。</p>
              <p>尺寸: {modal.size}</p>
            </Modal>
          </div>
        ))}
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: '所有尺寸模态框的展示。',
      },
    },
  },
};
