import React from 'react';
import Skeleton, { 
  SkeletonText, 
  SkeletonTitle, 
  SkeletonAvatar, 
  SkeletonCard, 
  SkeletonTable, 
  SkeletonList,
  SkeletonForm 
} from './Skeleton';

export default {
  title: 'UI/Skeleton',
  component: Skeleton,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: '骨架屏组件，用于内容加载时显示占位效果。支持多种预设组件和自定义样式。',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    width: {
      control: 'text',
      description: '骨架屏宽度',
      table: {
        type: { summary: 'string | number' },
      },
    },
    height: {
      control: 'text',
      description: '骨架屏高度',
      table: {
        type: { summary: 'string | number' },
      },
    },
    borderRadius: {
      control: 'text',
      description: '圆角大小',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: '4px' },
      },
    },
  },
};

export const Default = {
  args: {},
  parameters: {
    docs: {
      description: {
        story: '默认骨架屏组件，使用基本样式。',
      },
    },
  },
};

export const CustomSize = {
  args: {
    width: '300px',
    height: '40px',
  },
  parameters: {
    docs: {
      description: {
        story: '自定义尺寸的骨架屏。',
      },
    },
  },
};

export const Text = {
  render: () => <SkeletonText lines={4} lastLineWidth="50%" />,
  parameters: {
    docs: {
      description: {
        story: '多行文本骨架屏，最后一行可以单独设置宽度。',
      },
    },
  },
};

export const Title = {
  render: () => <SkeletonTitle width="50%" />,
  parameters: {
    docs: {
      description: {
        story: '标题骨架屏，用于页面标题加载状态。',
      },
    },
  },
};

export const Avatar = {
  render: () => (
    <div style={{ display: 'flex', gap: '16px' }}>
      <SkeletonAvatar size={32} />
      <SkeletonAvatar size={48} />
      <SkeletonAvatar size={64} />
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: '不同尺寸的头像骨架屏。',
      },
    },
  },
};

export const Card = {
  render: () => (
    <div style={{ width: '400px' }}>
      <SkeletonCard showAvatar showImage />
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: '卡片骨架屏，可选显示头像和图片。',
      },
    },
  },
};

export const Table = {
  render: () => (
    <div style={{ width: '800px' }}>
      <SkeletonTable rows={5} columns={4} />
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: '表格骨架屏，用于表格数据加载状态。',
      },
    },
  },
};

export const List = {
  render: () => (
    <div style={{ width: '500px' }}>
      <SkeletonList items={3} />
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: '列表骨架屏，用于列表数据加载状态。',
      },
    },
  },
};

export const Form = {
  render: () => (
    <div style={{ width: '400px' }}>
      <SkeletonForm fields={4} />
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: '表单骨架屏，用于表单加载状态。',
      },
    },
  },
};

export const AllComponents = {
  render: () => (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '32px', width: '600px' }}>
      <div>
        <h4>基础 Skeleton</h4>
        <Skeleton width="200px" height="20px" />
      </div>
      <div>
        <h4>SkeletonText</h4>
        <SkeletonText lines={3} />
      </div>
      <div>
        <h4>SkeletonCard</h4>
        <SkeletonCard showAvatar />
      </div>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: '展示所有骨架屏组件类型。',
      },
    },
  },
};

export const ArticlePage = {
  render: () => (
    <div style={{ width: '600px', display: 'flex', flexDirection: 'column', gap: '24px' }}>
      <SkeletonTitle width="60%" />
      <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
        <SkeletonAvatar size={40} />
        <div style={{ flex: 1 }}>
          <SkeletonText lines={1} />
        </div>
      </div>
      <Skeleton height="300px" />
      <SkeletonText lines={4} />
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: '模拟文章页面加载的骨架屏组合。',
      },
      code: `
<div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
  <SkeletonTitle width="60%" />
  <div style={{ display: 'flex', gap: '12px' }}>
    <SkeletonAvatar size={40} />
    <SkeletonText lines={1} />
  </div>
  <Skeleton height="300px" />
  <SkeletonText lines={4} />
</div>
      `,
    },
  },
};

export const UserProfile = {
  render: () => (
    <div style={{ width: '400px', padding: '24px', border: '1px solid #f0f0f0', borderRadius: '8px' }}>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '16px' }}>
        <SkeletonAvatar size={80} />
        <SkeletonTitle width="50%" />
        <SkeletonText lines={2} lastLineWidth="70%" />
        <div style={{ display: 'flex', gap: '16px', marginTop: '8px' }}>
          <Skeleton width="60px" height="24px" />
          <Skeleton width="60px" height="24px" />
          <Skeleton width="60px" height="24px" />
        </div>
      </div>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: '用户资料页面骨架屏。',
      },
    },
  },
};
