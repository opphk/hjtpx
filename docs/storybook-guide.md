# Storybook文档指南

## 概述

本项目使用Storybook进行组件文档和开发。

## 安装和启动

```bash
# 安装依赖
npm install

# 启动Storybook开发服务器
npm run storybook

# 构建静态Storybook
npm run build-storybook
```

## 组件组织

```
src/frontend/src/
├── components/
│   ├── ui/                    # UI基础组件
│   │   ├── Button.jsx
│   │   ├── Button.stories.jsx
│   │   ├── Input.jsx
│   │   └── Input.stories.jsx
│   ├── RegisterForm.jsx       # 业务组件
│   └── RegisterForm.stories.jsx
└── stories/                   # Storybook配置和介绍
    └── Introduction.stories.mdx
```

## 编写Storybook

### 基本结构

```jsx
import React from 'react';
import { Component } from './Component';

export default {
  title: 'Category/ComponentName',  // 组件在菜单中的位置
  component: Component,
  argTypes: {
    // 参数配置
  },
  parameters: {
    docs: {
      description: {
        component: '组件描述'
      }
    }
  }
};

const Template = (args) => <Component {...args} />;

export const Default = Template.bind({});
Default.args = {
  // 默认参数
};
```

### 参数类型

| 参数类型 | 控制类型 | 示例 |
|---------|---------|------|
| boolean | checkbox | `control: { type: 'boolean' }` |
| string | text | `control: { type: 'text' }` |
| number | number | `control: { type: 'number' }` |
| select | dropdown | `control: { type: 'select', options: [...] }` |
| color | color picker | `control: { type: 'color' }` |
| date | date picker | `control: { type: 'date' }` |

### 使用装饰器

```jsx
export default {
  decorators: [
    (Story) => (
      <div style={{ margin: '20px' }}>
        <Story />
      </div>
    )
  ]
};
```

### 添加注释

```jsx
export const Example = Template.bind({});
Example.args = {
  label: 'Click me'
};
Example.parameters = {
  docs: {
    description: {
      story: '这是一个示例说明'
    }
  }
};
```

## 最佳实践

1. **命名约定**
   - Stories文件: `ComponentName.stories.jsx`
   - Story名称: `PascalCase`
   - Category: `PascalCase/ComponentName`

2. **组织结构**
   - 按组件类型分组
   - 使用清晰的描述
   - 提供足够的示例

3. **参数配置**
   - 为每个prop提供argType
   - 添加描述
   - 设置合理的默认值

4. **交互测试**
   - 使用play函数进行交互测试
   - 测试用户行为
   - 验证状态变化

## 常用命令
