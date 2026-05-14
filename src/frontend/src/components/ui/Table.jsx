import React from 'react';

const Table = ({ 
  columns, 
  data, 
  onRowClick,
  emptyText = '暂无数据',
  loading = false,
  className = ''
}) => {
  if (loading) {
    return (
      <div className="table-loading">
        <p>加载中...</p>
      </div>
    );
  }

  if (!data || data.length === 0) {
    return (
      <div className="table-empty">
        <p>{emptyText}</p>
      </div>
    );
  }

  return (
    <div className={`table-wrapper ${className}`}>
      <table className="data-table">
        <thead>
          <tr>
            {columns.map((col, index) => (
              <th 
                key={index} 
                style={{ width: col.width }}
                className={col.className || ''}
              >
                {col.title}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.map((row, rowIndex) => (
            <tr 
              key={rowIndex}
              onClick={() => onRowClick && onRowClick(row)}
              className={onRowClick ? 'clickable' : ''}
            >
              {columns.map((col, colIndex) => (
                <td key={colIndex} className={col.className || ''}>
                  {col.render ? col.render(row[col.dataIndex], row) : row[col.dataIndex]}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default Table;
