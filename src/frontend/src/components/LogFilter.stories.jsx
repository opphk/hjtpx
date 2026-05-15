import React from 'react';
import { LogFilter } from './LogFilter';

export default {
  title: 'Components/LogFilter',
  component: LogFilter,
  parameters: {
    docs: {
      description: {
        component: '日志过滤器组件'
      }
    }
  }
};

const Template = (args) => <LogFilter {...args} />;

export const Default = Template.bind({});
Default.args = {
  // 默认参数
};

export const WithData = Template.bind({});
WithData.args = {
  // 带数据的示例
};
