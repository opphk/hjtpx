import React, { useState } from 'react';
import Button from './Button';

function Table({
  data = [],
  columns = [],
  currentPage = 1,
  totalPages = 1,
  onPageChange,
  sortable = false,
  className = '',
  'aria-label': ariaLabel,
  caption
}) {
  const [sortColumn, setSortColumn] = useState(null);
  const [sortDirection, setSortDirection] = useState('asc');

  const handleSort = (columnKey) => {
    if (!sortable) return;

    if (sortColumn === columnKey) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortColumn(columnKey);
      setSortDirection('asc');
    }
  };

  const handleSortKeyDown = (e, columnKey) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handleSort(columnKey);
    }
  };

  const sortedData = React.useMemo(() => {
    if (!sortColumn) return data;

    return [...data].sort((a, b) => {
      const aVal = a[sortColumn];
      const bVal = b[sortColumn];

      if (aVal < bVal) return sortDirection === 'asc' ? -1 : 1;
      if (aVal > bVal) return sortDirection === 'asc' ? 1 : -1;
      return 0;
    });
  }, [data, sortColumn, sortDirection]);

  const tableClass = `table-container ${className}`.trim();

  return (
    <div className={tableClass}>
      <table 
        className="table"
        role="grid"
        aria-label={ariaLabel}
      >
        {caption && (
          <caption className="visually-hidden">{caption}</caption>
        )}
        <thead role="rowgroup">
          <tr role="row">
            {columns.map((column) => (
              <th
                key={column.key}
                role="columnheader"
                scope="col"
                onClick={() => handleSort(column.key)}
                onKeyDown={(e) => handleSortKeyDown(e, column.key)}
                className={sortable ? 'sortable' : ''}
                tabIndex={sortable ? 0 : -1}
                aria-sort={sortColumn === column.key 
                  ? (sortDirection === 'asc' ? 'ascending' : 'descending') 
                  : 'none'}
              >
                {column.label}
                {sortable && sortColumn === column.key && (
                  <span className="sort-indicator" aria-hidden="true">
                    {sortDirection === 'asc' ? ' ▲' : ' ▼'}
                  </span>
                )}
              </th>
            ))}
          </tr>
        </thead>
        <tbody role="rowgroup">
          {sortedData.length === 0 ? (
            <tr role="row">
              <td 
                colSpan={columns.length} 
                className="no-data"
                role="gridcell"
              >
                No data available
              </td>
            </tr>
          ) : (
            sortedData.map((row, index) => (
              <tr key={row.id || index} role="row">
                {columns.map((column) => (
                  <td key={column.key} role="gridcell">
                    {column.render ? column.render(row[column.key], row) : row[column.key]}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>

      {totalPages > 1 && onPageChange && (
        <div 
          className="table-pagination"
          role="navigation"
          aria-label="Pagination"
        >
          <Button
            onClick={() => onPageChange(currentPage - 1)}
            disabled={currentPage === 1}
            size="small"
            aria-label="Previous page"
          >
            Previous
          </Button>
          <span 
            className="page-info"
            aria-live="polite"
          >
            Page {currentPage} of {totalPages}
          </span>
          <Button
            onClick={() => onPageChange(currentPage + 1)}
            disabled={currentPage === totalPages}
            size="small"
            aria-label="Next page"
          >
            Next
          </Button>
        </div>
      )}
    </div>
  );
}

export default React.memo(Table);
