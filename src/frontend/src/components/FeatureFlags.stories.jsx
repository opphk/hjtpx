import React from 'react';
import { FeatureFlags } from './FeatureFlags';

export default {
  title: 'Components/FeatureFlags',
  component: FeatureFlags,
  parameters: {
    docs: {
      description: {
        component: '功能开关组件'
      }
    }
  }
};

const Template = (args) => <FeatureFlags {...args} />;

export const Default = Template.bind({});
Default.args = {
  // 默认参数
};

export const WithData = Template.bind({});
WithData.args = {
  // 带数据的示例
};
