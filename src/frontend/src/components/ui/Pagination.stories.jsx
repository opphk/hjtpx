import React, { useState } from 'react';
import Pagination from './Pagination';

export default {
  title: 'UI/Pagination',
  component: Pagination,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: '分页组件，用于大量数据的分页展示。支持当前页、总页数、页大小等配置。',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    current: {
      control: 'number',
      description: '当前页码',
      table: {
        type: { summary: 'number' },
        defaultValue: { summary: '1' },
      },
    },
    total: {
      control: 'number',
      description: '总记录数',
      table: {
        type: { summary: 'number' },
      },
    },
    pageSize: {
      control: 'number',
      description: '每页显示的记录数',
      table: {
        type: { summary: 'number' },
        defaultValue: { summary: '10' },
      },
    },
    showTotal: {
      control: 'boolean',
      description: '是否显示总记录数信息',
      table: {
        type: { summary: 'boolean' },
        defaultValue: { summary: 'true' },
      },
    },
    onChange: {
      action: 'page changed',
      description: '页码变化时的回调函数',
      table: {
        type: { summary: 'function' },
      },
    },
  },
};

const Template = (args) => {
  const [current, setCurrent] = useState(args.current || 1);
  return (
    <Pagination
      {...args}
      current={current}
      onChange={(page) => {
        setCurrent(page);
        if (args.onChange) args.onChange(page);
      }}
    />
  );
};

export const Default = Template.bind({});
Default.args = {
  current: 1,
  total: 50,
  pageSize: 10,
};
Default.parameters = {
  docs: {
    description: {
      story: '默认分页组件，显示当前页码和总记录数。',
    },
  },
};

export const CurrentPage3 = Template.bind({});
CurrentPage3.args = {
  current: 3,
  total: 50,
  pageSize: 10,
};
CurrentPage3.parameters = {
  docs: {
    description: {
      story: '当前页为第3页的分页组件。',
    },
  },
};

export const ManyPages = Template.bind({});
ManyPages.args = {
  current: 10,
  total: 200,
  pageSize: 10,
};
ManyPages.parameters = {
  docs: {
    description: {
      story: '大量数据时的分页组件，共20页。',
    },
  },
};

export const WithoutTotal = Template.bind({});
WithoutTotal.args = {
  current: 2,
  total: 50,
  pageSize: 10,
  showTotal: false,
};
WithoutTotal.parameters = {
  docs: {
    description: {
      story: '不显示总记录数的分页组件。',
    },
  },
};

export const SmallDataSet = Template.bind({});
SmallDataSet.args = {
  current: 1,
  total: 15,
  pageSize: 10,
};
SmallDataSet.parameters = {
  docs: {
    description: {
      story: '数据集较小时的分页组件，当总页数小于等于1时不显示。',
    },
  },
};

export const WithCallback = {
  render: () => {
    const [current, setCurrent] = useState(1);
    const [logs, setLogs] = useState(['分页已初始化']);

    const handlePageChange = (page) => {
      setCurrent(page);
      setLogs((prev) => [...prev, `切换到第 ${page} 页`]);
    };

    return (
      <div>
        <Pagination
          current={current}
          total={100}
          pageSize={10}
          onChange={handlePageChange}
        />
        <div style={{ marginTop: '16px', padding: '12px', background: '#f5f5f5', borderRadius: '4px' }}>
          <strong>切换日志:</strong>
          <ul style={{ margin: '8px 0 0 0', paddingLeft: '20px' }}>
            {logs.map((log, index) => (
              <li key={index}>{log}</li>
            ))}
          </ul>
        </div>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: '带页码切换回调的分页组件，可记录切换日志。',
      },
      code: `
const [current, setCurrent] = useState(1);

const handlePageChange = (page) => {
  console.log('切换到第', page, '页');
  setCurrent(page);
};

<Pagination
  current={current}
  total={100}
  pageSize={10}
  onChange={handlePageChange}
/>
      `,
    },
  },
};

export const DifferentPageSizes = {
  render: () => {
    const [current, setCurrent] = useState(1);
    const [pageSize, setPageSize] = useState(10);

    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <label>每页显示:</label>
          <select
            value={pageSize}
            onChange={(e) => {
              setPageSize(Number(e.target.value));
              setCurrent(1);
            }}
            style={{ padding: '4px 8px' }}
          >
            <option value={5}>5条/页</option>
            <option value={10}>10条/页</option>
            <option value={20}>20条/页</option>
            <option value={50}>50条/页</option>
          </select>
        </div>
        <Pagination
          current={current}
          total={100}
          pageSize={pageSize}
          onChange={setCurrent}
        />
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: '可切换每页显示条数的分页组件。',
      },
    },
  },
};
