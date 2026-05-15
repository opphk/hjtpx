import React from 'react';

const Chart = () => {
  const data = [
    { name: '周一', value: 120 },
    { name: '周二', value: 200 },
    { name: '周三', value: 150 },
    { name: '周四', value: 80 },
    { name: '周五', value: 70 },
    { name: '周六', value: 110 },
    { name: '周日', value: 90 }
  ];

  const maxValue = Math.max(...data.map(d => d.value));

  return (
    <div style={{
      padding: '20px',
      background: '#fff',
      borderRadius: '8px',
      marginTop: '20px'
    }}>
      <h3 style={{ marginBottom: '20px' }}>用户活跃度图表 (Lazy Loaded)</h3>
      
      <div style={{
        display: 'flex',
        alignItems: 'flex-end',
        height: '200px',
        gap: '12px',
        padding: '20px 0'
      }}>
        {data.map((item, index) => (
          <div
            key={index}
            style={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: '8px'
            }}
          >
            <div
              style={{
                width: '100%',
                height: `${(item.value / maxValue) * 160}px`,
                backgroundColor: `hsl(${200 + index * 10}, 70%, 50%)`,
                borderRadius: '4px 4px 0 0',
                transition: 'height 0.3s ease'
              }}
              title={`${item.name}: ${item.value}`}
            />
            <span style={{
              fontSize: '12px',
              color: '#666'
            }}>
              {item.name}
            </span>
          </div>
        ))}
      </div>

      <div style={{
        marginTop: '20px',
        padding: '16px',
        background: '#f5f5f5',
        borderRadius: '4px',
        fontSize: '14px'
      }}>
        <p><strong>图表加载方式:</strong> 懒加载 (Lazy Loading)</p>
        <p><strong>渲染时间:</strong> {Date.now() - performance.now()} ms</p>
        <p><strong>数据点:</strong> {data.length} 个</p>
      </div>
    </div>
  );
};

export default Chart;
