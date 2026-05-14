import React, { useState } from 'react';
import Button from './Button';
import Modal from '../components/Modal';

export default {
  title: 'Components/Modal',
  component: Modal,
  parameters: {
    docs: {
      description: {
        component: '模态框组件，支持多种尺寸、标题、内容和页脚。',
      },
    },
  },
  argTypes: {
    isOpen: {
      control: 'boolean',
      description: '是否打开',
    },
    title: {
      control: 'text',
      description: '模态框标题',
    },
    size: {
      control: { type: 'select' },
      options: ['small', 'medium', 'large'],
      description: '模态框尺寸',
    },
    closeOnOverlayClick: {
      control: 'boolean',
      description: '点击遮罩层是否关闭',
    },
  },
  tags: ['autodocs'],
};

const ModalTemplate = ({ isOpen: isOpenProp, ...args }) => {
  const [isOpen, setIsOpen] = useState(isOpenProp || false);

  return (
    <>
      <Button onClick={() => setIsOpen(true)}>Open Modal</Button>
      <Modal {...args} isOpen={isOpen} onClose={() => setIsOpen(false)} />
    </>
  );
};

export const Default = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);
    return (
      <>
        <Button onClick={() => setIsOpen(true)}>Open Modal</Button>
        <Modal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          title="Modal Title"
        >
          <p>This is the modal content. You can put any content here.</p>
        </Modal>
      </>
    );
  },
};

export const WithFooter = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);
    return (
      <>
        <Button onClick={() => setIsOpen(true)}>Open Modal with Footer</Button>
        <Modal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          title="Confirmation"
          footer={
            <>
              <Button variant="secondary" onClick={() => setIsOpen(false)}>
                Cancel
              </Button>
              <Button variant="primary" onClick={() => setIsOpen(false)}>
                Confirm
              </Button>
            </>
          }
        >
          <p>Are you sure you want to proceed?</p>
        </Modal>
      </>
    );
  },
};

export const Small = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);
    return (
      <>
        <Button onClick={() => setIsOpen(true)}>Open Small Modal</Button>
        <Modal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          title="Small Modal"
          size="small"
        >
          <p>This is a small modal.</p>
        </Modal>
      </>
    );
  },
};

export const Large = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);
    return (
      <>
        <Button onClick={() => setIsOpen(true)}>Open Large Modal</Button>
        <Modal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          title="Large Modal"
          size="large"
        >
          <p>This is a large modal with more content space.</p>
        </Modal>
      </>
    );
  },
};

export const NoCloseOnOverlay = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);
    return (
      <>
        <Button onClick={() => setIsOpen(true)}>Open Modal</Button>
        <Modal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          title="Cannot Close on Overlay"
          closeOnOverlayClick={false}
        >
          <p>Click outside to see - this modal won't close!</p>
        </Modal>
      </>
    );
  },
};

export const AllSizes = {
  render: () => {
    const [activeModal, setActiveModal] = useState(null);
    return (
      <>
        <div style={{ display: 'flex', gap: '10px' }}>
          <Button size="small" onClick={() => setActiveModal('small')}>Small</Button>
          <Button size="small" onClick={() => setActiveModal('medium')}>Medium</Button>
          <Button size="small" onClick={() => setActiveModal('large')}>Large</Button>
        </div>
        <Modal
          isOpen={activeModal === 'small'}
          onClose={() => setActiveModal(null)}
          title="Small Modal"
          size="small"
        >
          <p>This is a small modal.</p>
        </Modal>
        <Modal
          isOpen={activeModal === 'medium'}
          onClose={() => setActiveModal(null)}
          title="Medium Modal"
          size="medium"
        >
          <p>This is a medium modal.</p>
        </Modal>
        <Modal
          isOpen={activeModal === 'large'}
          onClose={() => setActiveModal(null)}
          title="Large Modal"
          size="large"
        >
          <p>This is a large modal.</p>
        </Modal>
      </>
    );
  },
};
