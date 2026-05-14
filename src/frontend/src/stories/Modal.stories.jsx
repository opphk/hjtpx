import React, { useState } from 'react';
import Modal from '../components/ui/Modal';
import Button from '../components/ui/Button';

export default {
  title: 'UI/Modal',
  component: Modal,
  tags: ['autodocs'],
  argTypes: {
    isOpen: {
      control: 'boolean',
      description: '是否打开模态框',
      table: {
        type: { summary: 'boolean' },
        defaultValue: { summary: 'false' }
      }
    },
    onClose: {
      action: 'closed',
      description: '关闭事件处理函数'
    },
    title: {
      control: 'text',
      description: '模态框标题',
      table: {
        type: { summary: 'string' }
      }
    },
    size: {
      control: 'select',
      options: ['small', 'medium', 'large'],
      description: '模态框尺寸',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: 'medium' }
      }
    },
    children: {
      control: 'text',
      description: '模态框内容'
    },
    footer: {
      control: 'text',
      description: '模态框底部内容'
    }
  },
  parameters: {
    docs: {
      description: {
        component: '模态框对话框组件，支持多种尺寸和自定义底部内容。'
      }
    }
  }
};

const Template = (args) => {
  const [isOpen, setIsOpen] = useState(false);
  return (
    <>
      <Button onClick={() => setIsOpen(true)}>打开模态框</Button>
      <Modal {...args} isOpen={isOpen} onClose={() => setIsOpen(false)} />
    </>
  );
};

export const Default = Template.bind({});
Default.args = {
  title: '默认模态框',
  children: '这是模态框的内容区域',
  size: 'medium'
};

export const Small = () => {
  const [isOpen, setIsOpen] = useState(false);
  return (
    <>
      <Button onClick={() => setIsOpen(true)}>打开小模态框</Button>
      <Modal
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        title="小尺寸"
        size="small"
      >
        <p>小尺寸模态框内容</p>
      </Modal>
    </>
  );
};

export const Large = () => {
  const [isOpen, setIsOpen] = useState(false);
  return (
    <>
      <Button onClick={() => setIsOpen(true)}>打开大模态框</Button>
      <Modal
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        title="大尺寸模态框"
        size="large"
      >
        <p>大尺寸模态框适合显示更多内容或复杂布局</p>
      </Modal>
    </>
  );
};

export const WithFooter = () => {
  const [isOpen, setIsOpen] = useState(false);
  return (
    <>
      <Button onClick={() => setIsOpen(true)}>打开带底部的模态框</Button>
      <Modal
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        title="确认操作"
        footer={
          <>
            <Button variant="secondary" onClick={() => setIsOpen(false)}>
              取消
            </Button>
            <Button variant="primary" onClick={() => setIsOpen(false)}>
              确认
            </Button>
          </>
        }
      >
        <p>请确认您要执行的操作</p>
      </Modal>
    </>
  );
};

WithFooter.parameters = {
  docs: {
    description: {
      story: '带有自定义底部按钮的模态框'
    }
  }
};
