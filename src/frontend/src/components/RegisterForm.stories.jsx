import React from 'react';
import { RegisterForm } from './RegisterForm';

export default {
  title: 'Components/RegisterForm',
  component: RegisterForm,
  parameters: {
    docs: {
      description: {
        component: '注册表单组件'
      }
    }
  }
};

const Template = (args) => <RegisterForm {...args} />;

export const Default = Template.bind({});
Default.args = {
  // 默认参数
};

export const WithData = Template.bind({});
WithData.args = {
  // 带数据的示例
};
