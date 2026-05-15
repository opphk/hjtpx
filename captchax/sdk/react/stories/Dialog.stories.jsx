import React, { useState } from 'react';
import { CaptchaDialog } from '../src/components/CaptchaDialog';
import { CaptchaProvider, useCaptcha } from '../src';

const DialogDemo = ({ type, theme }) => {
  const [visible, setVisible] = useState(false);

  return (
    <>
      <button onClick={() => setVisible(true)}>
        打开 {type} 验证
      </button>
      <CaptchaDialog
        visible={visible}
        type={type}
        theme={theme}
        onSuccess={(token) => {
          console.log('Verified:', token);
          setVisible(false);
        }}
        onClose={() => setVisible(false)}
      />
    </>
  );
};

export default {
  title: 'Components/CaptchaDialog',
  component: CaptchaDialog,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: '验证码弹窗组件，提供模态框形式的验证交互'
      }
    }
  },
  decorators: [
    (Story) => (
      <CaptchaProvider config={{ apiServer: 'http://localhost:8080' }}>
        <div style={{ padding: '20px' }}>
          <Story />
        </div>
      </CaptchaProvider>
    )
  ],
  tags: ['autodocs']
};

export const Default = {
  render: () => <DialogDemo type="slider" theme="light" />
};

export const ClickType = {
  render: () => <DialogDemo type="click" theme="light" />
};

export const RotateType = {
  render: () => <DialogDemo type="rotate" theme="light" />
};

export const PuzzleType = {
  render: () => <DialogDemo type="puzzle" theme="light" />
};

export const TextType = {
  render: () => <DialogDemo type="text" theme="light" />
};

export const IconType = {
  render: () => <DialogDemo type="icon" theme="light" />
};

export const DarkTheme = {
  render: () => <DialogDemo type="slider" theme="dark" />
};

export const NonMaskClosable = {
  render: () => {
    const [visible, setVisible] = useState(false);

    return (
      <>
        <button onClick={() => setVisible(true)}>
          不可点击遮罩关闭
        </button>
        <CaptchaDialog
          visible={visible}
          type="slider"
          maskClosable={false}
          onSuccess={(token) => {
            console.log('Verified:', token);
            setVisible(false);
          }}
          onClose={() => setVisible(false)}
        />
      </>
    );
  }
};

export const CustomWidth = {
  render: () => {
    const [visible, setVisible] = useState(false);

    return (
      <>
        <button onClick={() => setVisible(true)}>
          自定义宽度
        </button>
        <CaptchaDialog
          visible={visible}
          type="slider"
          width={400}
          onSuccess={(token) => {
            console.log('Verified:', token);
            setVisible(false);
          }}
          onClose={() => setVisible(false)}
        />
      </>
    );
  }
};
