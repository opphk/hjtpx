import React from 'react';
import Loading from '../components/ui/Loading';

export default {
  title: 'UI/Loading',
  component: Loading,
  tags: ['autodocs'],
  argTypes: {
    size: {
      control: 'radio',
      options: ['small', 'medium', 'large'],
      description: '加载指示器尺寸',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: 'medium' }
      }
    },
    text: {
      control: 'text',
      description: '加载提示文本',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: '加载中...' }
      }
    },
    fullScreen: {
      control: 'boolean',
      description: '是否全屏显示',
      table: {
        type: { summary: 'boolean' },
        defaultValue: { summary: 'false' }
      }
    }
  },
  parameters: {
    docs: {
      description: {
        component: '加载状态指示器组件，用于显示异步操作或数据加载状态。'
      }
    }
  }
};

const Template = (args) => <Loading {...args} />;

export const Small = Template.bind({});
Small.args = {
  size: 'small',
  text: '加载中'
};

export const Medium = Template.bind({});
Medium.args = {
  size: 'medium',
  text: '加载中...'
};

export const Large = Template.bind({});
Large.args = {
  size: 'large',
  text: '正在加载数据'
};

export const WithoutText = Template.bind({});
WithoutText.args = {
  size: 'medium'
};

export const AllSizes = () => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: '20px', alignItems: 'flex-start' }}>
    <div>
      <h4>Small</h4>
      <Loading size="small" />
    </div>
    <div>
      <h4>Medium</h4>
      <Loading size="medium" />
    </div>
    <div>
      <h4>Large</h4>
      <Loading size="large" />
    </div>
  </div>
);

AllSizes.parameters = {
  docs: {
    description: {
      story: '不同尺寸的加载指示器'
    }
  }
};

export const CustomText = Template.bind({});
CustomText.args = {
  size: 'medium',
  text: '正在处理您的请求...'
};
