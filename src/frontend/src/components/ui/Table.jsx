import React from 'react';

const Table = ({ 
  columns, 
  data, 
  onRowClick,
  emptyText = '暂无数据',
  loading = false,
  className = '',
  'aria-label': ariaLabel,
  'aria-describedby': ariaDescribedBy,
  caption,
  ...props
}) => {
  const tableId = React.useId();
  const captionId = caption ? `${tableId}-caption` : undefined;

  if (loading) {
    return (
      <div 
        className="table-loading" 
        role="status"
        aria-live="polite"
      >
        <p>加载中...</p>
      </div>
    );
  }

  if (!data || data.length === 0) {
    return (
      <div 
        className="table-empty"
        role="status"
        aria-live="polite"
      >
        <p>{emptyText}</p>
      </div>
    );
  }

  const handleRowKeyDown = (e, row) => {
    if (onRowClick && (e.key === 'Enter' || e.key === ' ')) {
      e.preventDefault();
      onRowClick(row);
    }
  };

  return (
    <div className={`table-wrapper ${className}`}>
      <table 
        className="data-table"
        role="grid"
        aria-label={ariaLabel}
        aria-describedby={ariaDescribedBy || captionId}
        {...props}
      >
        {caption && (
          <caption id={captionId} className="sr-only">
            {caption}
          </caption>
        )}
        <thead role="rowgroup">
          <tr role="row">
            {columns.map((col, index) => (
              <th 
                key={index} 
                role="columnheader"
                scope="col"
                style={{ width: col.width }}
                className={col.className || ''}
              >
                {col.title}
              </th>
            ))}
          </tr>
        </thead>
        <tbody role="rowgroup">
          {data.map((row, rowIndex) => (
            <tr 
              key={rowIndex}
              role="row"
              tabIndex={onRowClick ? 0 : -1}
              onClick={() => onRowClick && onRowClick(row)}
              onKeyDown={(e) => handleRowKeyDown(e, row)}
              className={onRowClick ? 'clickable' : ''}
              aria-selected={false}
            >
              {columns.map((col, colIndex) => (
                <td 
                  key={colIndex} 
                  role="gridcell"
                  className={col.className || ''}
                >
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
