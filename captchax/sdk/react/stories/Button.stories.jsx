import React from 'react';
import { CaptchaButton } from '../src/components/CaptchaButton';
import { CaptchaProvider } from '../src/CaptchaProvider';

export default {
  title: 'Components/CaptchaButton',
  component: CaptchaButton,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: '验证码按钮组件，点击触发验证流程'
      }
    }
  },
  decorators: [
    (Story) => (
      <CaptchaProvider config={{ apiServer: 'http://localhost:8080' }}>
        <Story />
      </CaptchaProvider>
    )
  ],
  tags: ['autodocs']
};

export const Default = {
  args: {
    children: '点击验证',
    scene: 'default',
    size: 'medium',
    theme: 'light'
  }
};

export const SmallSize = {
  args: {
    ...Default.args,
    size: 'small',
    children: '小按钮'
  }
};

export const LargeSize = {
  args: {
    ...Default.args,
    size: 'large',
    children: '大按钮'
  }
};

export const DarkTheme = {
  args: {
    ...Default.args,
    theme: 'dark',
    children: '深色主题'
  }
};

export const LoginScene = {
  args: {
    ...Default.args,
    scene: 'login',
    children: '登录验证'
  }
};

export const RegisterScene = {
  args: {
    ...Default.args,
    scene: 'register',
    children: '注册验证'
  }
};

export const WithSuccessCallback = {
  args: {
    ...Default.args,
    children: '验证成功回调'
  },
  render: (args) => {
    const handleSuccess = (token) => {
      console.log('Verification successful, token:', token);
      alert('验证成功！Token: ' + token);
    };

    return (
      <CaptchaButton
        {...args}
        onSuccess={handleSuccess}
      />
    );
  }
};

export const WithErrorCallback = {
  args: {
    ...Default.args,
    children: '验证失败回调'
  },
  render: (args) => {
    const handleError = (error) => {
      console.error('Verification failed:', error);
      alert('验证失败：' + error.message);
    };

    return (
      <CaptchaButton
        {...args}
        onError={handleError}
      />
    );
  }
};

export const Disabled = {
  args: {
    ...Default.args,
    children: '禁用状态',
    disabled: true
  }
};
