import React from 'react';
import { UserList } from './UserList';

export default {
  title: 'Components/UserList',
  component: UserList,
  parameters: {
    docs: {
      description: {
        component: '用户列表组件'
      }
    }
  }
};

const Template = (args) => <UserList {...args} />;

export const Default = Template.bind({});
Default.args = {
  // 默认参数
};

export const WithData = Template.bind({});
WithData.args = {
  // 带数据的示例
};
