import React from 'react';
import { NotificationComponents } from './NotificationComponents';

export default {
  title: 'Components/NotificationComponents',
  component: NotificationComponents,
  parameters: {
    docs: {
      description: {
        component: '通知组件组件'
      }
    }
  }
};

const Template = (args) => <NotificationComponents {...args} />;

export const Default = Template.bind({});
Default.args = {
  // 默认参数
};

export const WithData = Template.bind({});
WithData.args = {
  // 带数据的示例
};
