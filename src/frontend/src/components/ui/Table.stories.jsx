import React from 'react';
import Table from './Table';
import Button from './Button';

export default {
  title: 'UI/Table',
  component: Table,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: '表格组件，用于展示结构化数据。支持列配置、行点击、空状态和加载状态。',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    columns: {
      control: false,
      description: '表格列配置',
      table: {
        type: { summary: 'Array<Column>' },
      },
    },
    data: {
      control: false,
      description: '表格数据',
      table: {
        type: { summary: 'Array<object>' },
      },
    },
    loading: {
      control: 'boolean',
      description: '是否显示加载状态',
      table: {
        type: { summary: 'boolean' },
        defaultValue: { summary: 'false' },
      },
    },
    emptyText: {
      control: 'text',
      description: '空数据时显示的文本',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: '暂无数据' },
      },
    },
    onRowClick: {
      action: 'row clicked',
      description: '行点击事件处理函数',
      table: {
        type: { summary: 'function' },
      },
    },
    caption: {
      control: 'text',
      description: '表格标题（屏幕阅读器可见）',
      table: {
        type: { summary: 'string' },
      },
    },
  },
};

const sampleColumns = [
  { title: '姓名', dataIndex: 'name', width: '150px' },
  { title: '年龄', dataIndex: 'age', width: '100px' },
  { title: '邮箱', dataIndex: 'email' },
  { title: '状态', dataIndex: 'status', width: '100px' },
];

const sampleData = [
  { name: '张三', age: 28, email: 'zhangsan@example.com', status: '活跃' },
  { name: '李四', age: 32, email: 'lisi@example.com', status: '离线' },
  { name: '王五', age: 25, email: 'wangwu@example.com', status: '活跃' },
  { name: '赵六', age: 30, email: 'zhaoliu@example.com', status: '离开' },
  { name: '钱七', age: 27, email: 'qianqi@example.com', status: '活跃' },
];

export const Default = {
  args: {
    columns: sampleColumns,
    data: sampleData,
  },
  parameters: {
    docs: {
      description: {
        story: '默认表格，显示数据列表。',
      },
    },
  },
};

export const WithRender = {
  args: {
    columns: [
      { title: '姓名', dataIndex: 'name', width: '150px' },
      { title: '年龄', dataIndex: 'age', width: '100px' },
      { title: '邮箱', dataIndex: 'email' },
      { 
        title: '状态', 
        dataIndex: 'status', 
        width: '100px',
        render: (text) => {
          const color = text === '活跃' ? 'green' : text === '离线' ? 'gray' : 'orange';
          return <span style={{ color }}>{text}</span>;
        }
      },
    ],
    data: sampleData,
  },
  parameters: {
    docs: {
      description: {
        story: '带自定义渲染的表格，可以格式化单元格内容。',
      },
    },
  },
};

export const Empty = {
  args: {
    columns: sampleColumns,
    data: [],
  },
  parameters: {
    docs: {
      description: {
        story: '空数据状态下的表格。',
      },
    },
  },
};

export const Loading = {
  args: {
    columns: sampleColumns,
    data: sampleData,
    loading: true,
  },
  parameters: {
    docs: {
      description: {
        story: '加载状态下的表格。',
      },
    },
  },
};

export const CustomEmptyText = {
  args: {
    columns: sampleColumns,
    data: [],
    emptyText: '没有找到相关数据',
  },
  parameters: {
    docs: {
      description: {
        story: '自定义空数据提示文本。',
      },
    },
  },
};

export const ClickableRow = {
  render: () => {
    const handleRowClick = (row) => {
      alert(`点击了: ${row.name}`);
    };

    return (
      <Table
        columns={sampleColumns}
        data={sampleData}
        onRowClick={handleRowClick}
      />
    );
  },
  parameters: {
    docs: {
      description: {
        story: '可点击的表格行，支持键盘导航（Enter/Space）。',
      },
      code: `
const handleRowClick = (row) => {
  console.log('Row clicked:', row);
};

<Table
  columns={columns}
  data={data}
  onRowClick={handleRowClick}
/>
      `,
    },
  },
};

export const WithActions = {
  render: () => {
    const handleEdit = (row) => {
      alert(`编辑: ${row.name}`);
    };

    const handleDelete = (row) => {
      alert(`删除: ${row.name}`);
    };

    const columnsWithActions = [
      ...sampleColumns,
      {
        title: '操作',
        dataIndex: 'actions',
        width: '200px',
        render: (_, row) => (
          <div style={{ display: 'flex', gap: '8px' }}>
            <Button size="small" onClick={() => handleEdit(row)}>编辑</Button>
            <Button size="small" variant="danger" onClick={() => handleDelete(row)}>删除</Button>
          </div>
        ),
      },
    ];

    return (
      <Table
        columns={columnsWithActions}
        data={sampleData}
      />
    );
  },
  parameters: {
    docs: {
      description: {
        story: '带操作按钮的表格。',
      },
      code: `
const columns = [
  ...sampleColumns,
  {
    title: '操作',
    dataIndex: 'actions',
    render: (_, row) => (
      <div>
        <Button onClick={() => handleEdit(row)}>编辑</Button>
        <Button onClick={() => handleDelete(row)}>删除</Button>
      </div>
    ),
  },
];

<Table columns={columns} data={data} />
      `,
    },
  },
};

export const WithCaption = {
  args: {
    columns: sampleColumns,
    data: sampleData,
    caption: '用户信息表',
  },
  parameters: {
    docs: {
      description: {
        story: '带屏幕阅读器可见标题的表格。',
      },
    },
  },
};
