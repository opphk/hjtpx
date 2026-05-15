import React from 'react';
import { AdminUserTable } from './AdminUserTable';

export default {
  title: 'Components/AdminUserTable',
  component: AdminUserTable,
  parameters: {
    docs: {
      description: {
        component: '管理员用户表组件'
      }
    }
  }
};

const Template = (args) => <AdminUserTable {...args} />;

export const Default = Template.bind({});
Default.args = {
  // 默认参数
};

export const WithData = Template.bind({});
WithData.args = {
  // 带数据的示例
};
