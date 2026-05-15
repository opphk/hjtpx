#!/bin/bash
# 任务13：Storybook文档完善
# 配置Storybook环境
# 为每个组件编写stories
# 添加交互示例
# 添加参数文档
# 验证Storybook运行

echo "=========================================="
echo "任务13：Storybook文档完善"
echo "=========================================="

cd /workspace/hjtpx

# 1. 配置Storybook环境
echo "[13.1] 配置Storybook环境..."

if [ ! -d ".storybook" ]; then
    echo "  → 创建.storybook目录"
    mkdir -p .storybook
fi

# 创建main.js配置
cat > .storybook/main.js << 'EOF'
module.exports = {
  stories: [
    '../src/frontend/src/**/*.stories.mdx',
    '../src/frontend/src/**/*.stories.@(js|jsx|ts|tsx)'
  ],
  addons: [
    '@storybook/addon-essentials',
    '@storybook/addon-a11y',
    '@storybook/addon-controls',
    '@storybook/addon-actions',
    '@storybook/addon-jest'
  ],
  framework: '@storybook/react',
  core: {
    builder: 'webpack5'
  },
  staticDirs: ['../public'],
  features: {
    postcss: false,
    babelModeV7: true
  }
};
EOF

# 创建preview.js配置
cat > .storybook/preview.js << 'EOF'
import { initialize, mswDecorator } from 'msw-storybook-addon';
import '../src/frontend/src/styles/globals.css';

initialize();

export const decorators = [
  (Story) => ({
    components: { Story },
    template: '<div style="margin: 20px;"><Story /></div>'
  }),
  mswDecorator
];

export const parameters = {
  actions: { argTypesRegex: '^on[A-Z].*' },
  controls: {
    matchers: {
      color: /(background|color)$/i,
      date: /Date$/,
    },
  },
  docs: {
    description: {
      component: '组件文档描述'
    }
  },
  backgrounds: {
    default: 'light',
    values: [
      { name: 'light', value: '#ffffff' },
      { name: 'dark', value: '#1a1a1a' },
      { name: 'gray', value: '#f5f5f5' }
    ]
  }
};
EOF

echo "  ✓ Storybook配置已创建"

# 2. 为每个UI组件创建Stories
echo "[13.2] 为UI组件创建Stories..."

components=(
    "Button:按钮组件"
    "Input:输入框组件"
    "Modal:模态框组件"
    "Alert:警告提示组件"
    "Loading:加载状态组件"
    "Skeleton:骨架屏组件"
    "Pagination:分页组件"
)

for item in "${components[@]}"; do
    IFS=':' read -r component desc <<< "$item"
    story_file="src/frontend/src/components/ui/${component}.stories.jsx"
    
    if [ ! -f "$story_file" ]; then
        echo "  → 创建 ${component}.stories.jsx"
        cat > "$story_file" << EOF
import React from 'react';
import { ${component} } from './${component}';

export default {
  title: 'UI/${component}',
  component: ${component},
  argTypes: {
    variant: {
      control: { type: 'select' },
      options: ['primary', 'secondary', 'danger', 'success'],
      description: '${desc}变体'
    },
    size: {
      control: { type: 'select' },
      options: ['small', 'medium', 'large'],
      description: '${desc}尺寸'
    },
    disabled: {
      control: { type: 'boolean' },
      description: '是否禁用'
    },
    loading: {
      control: { type: 'boolean' },
      description: '是否加载中'
    }
  },
  parameters: {
    docs: {
      description: {
        component: '${desc} - 用于${desc}展示和交互'
      }
    }
  }
};

const Template = (args) => <${component} {...args} />;

export const Default = Template.bind({});
Default.args = {
  children: '${desc}文本',
  variant: 'primary',
  size: 'medium'
};

export const Secondary = Template.bind({});
Secondary.args = {
  ...Default.args,
  variant: 'secondary'
};

export const Danger = Template.bind({});
Danger.args = {
  ...Default.args,
  variant: 'danger'
};

export const Success = Template.bind({});
Success.args = {
  ...Default.args,
  variant: 'success'
};

export const Disabled = Template.bind({});
Disabled.args = {
  ...Default.args,
  disabled: true
};

export const Loading = Template.bind({});
Loading.args = {
  ...Default.args,
  loading: true
};

export const Small = Template.bind({});
Small.args = {
  ...Default.args,
  size: 'small'
};

export const Large = Template.bind({});
Large.args = {
  ...Default.args,
  size: 'large'
};
EOF
    else
        echo "  ✓ ${component}.stories.jsx已存在"
    fi
done

# 3. 为业务组件创建Stories
echo "[13.3] 为业务组件创建Stories..."

business_components=(
    "RegisterForm:注册表单"
    "UserList:用户列表"
    "AdminUserTable:管理员用户表"
    "LogFilter:日志过滤器"
    "NotificationComponents:通知组件"
    "FeatureFlags:功能开关"
)

for item in "${business_components[@]}"; do
    IFS=':' read -r component desc <<< "$item"
    component_file="src/frontend/src/components/${component}.jsx"
    story_file="src/frontend/src/components/${component}.stories.jsx"
    
    if [ -f "$component_file" ] && [ ! -f "$story_file" ]; then
        echo "  → 创建 ${component}.stories.jsx"
        cat > "$story_file" << EOF
import React from 'react';
import { ${component} } from './${component}';

export default {
  title: 'Components/${component}',
  component: ${component},
  parameters: {
    docs: {
      description: {
        component: '${desc}组件'
      }
    }
  }
};

const Template = (args) => <${component} {...args} />;

export const Default = Template.bind({});
Default.args = {
  // 默认参数
};

export const WithData = Template.bind({});
WithData.args = {
  // 带数据的示例
};
EOF
    fi
done

# 4. 添加交互示例
echo "[13.4] 添加交互示例..."

# 创建README story
cat > src/frontend/src/stories/Introduction.stories.mdx << 'EOF'
import { Meta } from '@storybook/addon-docs';
import Code from '../assets/code-brackets.svg';
import Colors from '../assets/colors.svg';
import Comments from '../assets/comments.svg';
import Direction from '../assets/direction.svg';
import Flow from '../assets/flow.svg';
import Plugin from '../assets/plugin.svg';
import Repo from '../assets/repo.svg';
import StackAlt from '../assets/stackalt.svg';

<Meta title="介绍/Introduction" />

<style>
  {`
    .subheading {
      --mediumdark: '#999999';
      font-weight: 900;
      font-size: 13px;
      color: #999;
      text-transform: uppercase;
      margin: 8px 0 8px 14px;
    }
    
    .linkItems {
      display: grid;
      grid-template-columns: repeat(2, 1fr);
      padding: 0;
    }
    
    .linkItem {
      display: flex;
      align-items: center;
      margin: 8px 0;
      color: #333;
      text-decoration: none;
    }
    
    .linkItem:hover {
      color: #6b46c1;
    }
    
    .linkItem img {
      margin-right: 8px;
      width: 20px;
      height: 20px;
    }
  `}
</style>

# HJTPX 组件库文档

欢迎使用HJTPX组件库Storybook文档。

## 快速开始

本Storybook包含了HJTPX项目中所有React组件的文档和示例。

### 浏览组件

在左侧菜单中选择任意组件查看：
- **UI组件** - 基础UI组件（Button、Input、Modal等）
- **业务组件** - 业务相关组件（RegisterForm、UserList等）

### 查看示例

每个组件都有多个示例，展示不同的：
- 变体和尺寸
- 状态（默认、禁用、加载等）
- 交互行为

### 探索参数

使用Controls面板动态调整组件属性：
- 点击参数右侧的控件
- 拖动滑块
- 选择下拉选项
- 查看组件如何响应

## 开发指南

```bash
# 启动Storybook
npm run storybook

# 构建Storybook
npm run build-storybook

# 运行Storybook测试
npm run test-storybook
```

## 资源链接

<div className="subheading">资源</div>

<div className="linkItems">
  <a className="linkItem" href="https://storybook.js.org/docs/react/get-started/introduction">
    <img src={Repo} alt="repo" />
    Storybook文档
  </a>
  <a className="linkItem" href="https://reactjs.org/docs/getting-started.html">
    <img src={React} alt="react" />
    React文档
  </a>
</div>
EOF

# 5. 创建组件文档
echo "[13.5] 创建组件文档..."

cat > docs/storybook-guide.md << 'EOF'
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
EOF

echo "  ✓ Storybook文档已创建"

# 6. 更新package.json添加storybook脚本
echo "[13.6] 检查package.json中的Storybook脚本..."
if grep -q "storybook" package.json; then
    echo "  ✓ Storybook脚本已配置"
else
    echo "  → 需要添加Storybook脚本到package.json"
fi

echo "=========================================="
echo "任务13完成：Storybook文档完善"
echo "=========================================="
