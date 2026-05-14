import React from 'react';
import Loading from '../components/Loading';

export default {
  title: 'Components/Loading',
  component: Loading,
  parameters: {
    docs: {
      description: {
        component: '加载指示器组件，支持多种尺寸和全屏模式。',
      },
    },
  },
  argTypes: {
    size: {
      control: { type: 'select' },
      options: ['small', 'medium', 'large'],
      description: '加载器尺寸',
    },
    fullScreen: {
      control: 'boolean',
      description: '是否全屏显示',
    },
    text: {
      control: 'text',
      description: '加载文本',
    },
  },
  tags: ['autodocs'],
};

export const Small = {
  args: {
    size: 'small',
  },
};

export const Medium = {
  args: {
    size: 'medium',
  },
};

export const Large = {
  args: {
    size: 'large',
  },
};

export const WithText = {
  args: {
    text: 'Loading data...',
  },
};

export const FullScreen = {
  args: {
    fullScreen: true,
    text: 'Loading...',
  },
};

export const AllSizes = {
  render: () => (
    <div style={{ display: 'flex', gap: '40px', alignItems: 'center' }}>
      <div style={{ textAlign: 'center' }}>
        <Loading size="small" />
        <p>Small</p>
      </div>
      <div style={{ textAlign: 'center' }}>
        <Loading size="medium" />
        <p>Medium</p>
      </div>
      <div style={{ textAlign: 'center' }}>
        <Loading size="large" />
        <p>Large</p>
      </div>
    </div>
  ),
};
