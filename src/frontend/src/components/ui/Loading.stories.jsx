import React from 'react';
import Loading from './Loading';

export default {
  title: 'UI/Loading',
  component: Loading,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: '加载状态组件，用于显示异步操作或数据加载的状态。支持多种尺寸和全屏模式。',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    size: {
      control: 'select',
      options: ['small', 'medium', 'large'],
      description: '加载指示器尺寸',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: 'medium' },
      },
    },
    text: {
      control: 'text',
      description: '加载提示文本',
      table: {
        type: { summary: 'string' },
        defaultValue: { summary: '加载中...' },
      },
    },
    fullScreen: {
      control: 'boolean',
      description: '是否全屏显示',
      table: {
        type: { summary: 'boolean' },
        defaultValue: { summary: 'false' },
      },
    },
  },
};

export const Default = {
  args: {},
  parameters: {
    docs: {
      description: {
        story: '默认加载指示器，显示中等尺寸的加载动画和默认文本。',
      },
    },
  },
};

export const Small = {
  args: {
    size: 'small',
  },
  parameters: {
    docs: {
      description: {
        story: '小尺寸的加载指示器，适合紧凑的布局。',
      },
    },
  },
};

export const Medium = {
  args: {
    size: 'medium',
  },
  parameters: {
    docs: {
      description: {
        story: '中等尺寸的加载指示器，默认大小。',
      },
    },
  },
};

export const Large = {
  args: {
    size: 'large',
  },
  parameters: {
    docs: {
      description: {
        story: '大尺寸的加载指示器，适合需要突出显示的场景。',
      },
    },
  },
};

export const WithCustomText = {
  args: {
    text: '正在处理数据...',
  },
  parameters: {
    docs: {
      description: {
        story: '带有自定义文本的加载指示器。',
      },
    },
  },
};

export const WithoutText = {
  args: {
    text: '',
  },
  parameters: {
    docs: {
      description: {
        story: '不带文本的加载指示器，只显示动画。',
      },
    },
  },
};

export const AllSizes = {
  render: () => (
    <div style={{ display: 'flex', gap: '48px', alignItems: 'center' }}>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px' }}>
        <Loading size="small" text="" />
        <span>小</span>
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px' }}>
        <Loading size="medium" text="" />
        <span>中</span>
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px' }}>
        <Loading size="large" text="" />
        <span>大</span>
      </div>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: '所有尺寸加载指示器的展示。',
      },
    },
  },
};

export const CustomMessages = {
  render: () => (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      <Loading text="加载中..." />
      <Loading text="正在提交表单..." />
      <Loading text="正在获取数据..." />
      <Loading text="正在处理您的请求..." />
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: '不同场景下的加载提示文本。',
      },
    },
  },
};

export const ButtonLoading = {
  render: () => {
    const [loading, setLoading] = React.useState(false);

    const handleClick = () => {
      setLoading(true);
      setTimeout(() => setLoading(false), 2000);
    };

    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: '16px', alignItems: 'center' }}>
        <button 
          onClick={handleClick}
          disabled={loading}
          style={{
            padding: '10px 20px',
            background: loading ? '#ccc' : '#007bff',
            color: 'white',
            border: 'none',
            borderRadius: '4px',
            cursor: loading ? 'not-allowed' : 'pointer'
          }}
        >
          {loading ? '提交中...' : '提交'}
        </button>
        <Loading text={loading ? "正在提交..." : ""} />
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: '模拟按钮提交时的加载状态。',
      },
      code: `
const [loading, setLoading] = useState(false);

const handleSubmit = async () => {
  setLoading(true);
  try {
    await submitForm();
  } finally {
    setLoading(false);
  }
};

return (
  <>
    <button disabled={loading} onClick={handleSubmit}>
      {loading ? '提交中...' : '提交'}
    </button>
    {loading && <Loading text="正在提交..." />}
  </>
);
      `,
    },
  },
};

export const CardLoading = {
  render: () => (
    <div style={{ width: '300px', padding: '20px', border: '1px solid #f0f0f0', borderRadius: '8px' }}>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '16px' }}>
        <Loading size="large" text="加载内容中..." />
      </div>
    </div>
  ),
  parameters: {
    docs: {
    description: {
        story: '卡片容器中的加载状态。',
      },
    },
  },
};

export const DataLoading = {
  render: () => {
    const [loading, setLoading] = React.useState(true);
    const [data, setData] = React.useState(null);

    React.useEffect(() => {
      const timer = setTimeout(() => {
        setData({ name: '示例数据', count: 100 });
        setLoading(false);
      }, 2000);
      return () => clearTimeout(timer);
    }, []);

    return (
      <div style={{ width: '300px' }}>
        {loading ? (
          <Loading text="正在加载数据..." />
        ) : (
          <div style={{ padding: '16px', border: '1px solid #f0f0f0', borderRadius: '8px' }}>
            <h4>{data.name}</h4>
            <p>共 {data.count} 条记录</p>
          </div>
        )}
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: '模拟数据加载场景，先显示加载状态，加载完成后显示数据。',
      },
      code: `
const [loading, setLoading] = useState(true);
const [data, setData] = useState(null);

useEffect(() => {
  fetchData().then(result => {
    setData(result);
    setLoading(false);
  });
}, []);

return (
  <>
    {loading ? (
      <Loading text="正在加载数据..." />
    ) : (
      <DataDisplay data={data} />
    )}
  </>
);
      `,
    },
  },
};
