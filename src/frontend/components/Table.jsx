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
  caption,
  rowCount = data.length
}) {
  const [sortColumn, setSortColumn] = useState(null);
  const [sortDirection, setSortDirection] = useState('asc');
  const [focusedRow, setFocusedRow] = useState(null);

  const tableId = `table-${Math.random().toString(36).substr(2, 9)}`;

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

  const handleRowKeyDown = (e, rowIndex) => {
    let newRowIndex = null;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        newRowIndex = Math.min(rowIndex + 1, sortedData.length - 1);
        break;
      case 'ArrowUp':
        e.preventDefault();
        newRowIndex = Math.max(rowIndex - 1, 0);
        break;
      case 'Home':
        e.preventDefault();
        newRowIndex = 0;
        break;
      case 'End':
        e.preventDefault();
        newRowIndex = sortedData.length - 1;
        break;
      default:
        return;
    }

    if (newRowIndex !== null) {
      setFocusedRow(newRowIndex);
      const rowElement = document.querySelector(`#${tableId} tbody tr:nth-child(${newRowIndex + 1})`);
      rowElement?.focus();
    }
  };

  const tableClass = `table-container ${className}`.trim();

  return (
    <div className={tableClass}>
      <table 
        className="table" 
        id={tableId}
        aria-label={caption}
      >
        {caption && <caption className="visually-hidden">{caption}</caption>}
        <thead>
          <tr>
            {columns.map((column) => (
              <th
                key={column.key}
                onClick={() => handleSort(column.key)}
                onKeyDown={(e) => handleSortKeyDown(e, column.key)}
                className={sortable ? 'sortable' : ''}
                scope="col"
                aria-sort={
                  sortable && sortColumn === column.key
                    ? sortDirection === 'asc'
                      ? 'ascending'
                      : 'descending'
                    : undefined
                }
                tabIndex={sortable ? 0 : undefined}
                role={sortable ? 'button' : undefined}
              >
                {column.label}
                {sortable && (
                  <span className="visually-hidden">
                    {sortColumn === column.key
                      ? sortDirection === 'asc'
                        ? '(sorted ascending)'
                        : '(sorted descending)'
                      : '(unsorted)'}
                  </span>
                )}
                {sortable && sortColumn === column.key && (
                  <span className="sort-indicator" aria-hidden="true">
                    {sortDirection === 'asc' ? ' ▲' : ' ▼'}
                  </span>
                )}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {sortedData.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="no-data" role="status">
                No data available
              </td>
            </tr>
          ) : (
            sortedData.map((row, index) => (
              <tr
                key={row.id || index}
                tabIndex={0}
                onFocus={() => setFocusedRow(index)}
                onKeyDown={(e) => handleRowKeyDown(e, index)}
                aria-selected={focusedRow === index}
              >
                {columns.map((column) => (
                  <td key={column.key}>
                    {column.render ? column.render(row[column.key], row) : row[column.key]}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>

      {totalPages > 1 && onPageChange && (
        <nav className="table-pagination" aria-label="Table pagination">
          <Button
            onClick={() => onPageChange(currentPage - 1)}
            disabled={currentPage === 1}
            size="small"
            aria-label="Previous page"
          >
            Previous
          </Button>
          <span className="page-info" aria-live="polite">
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
        </nav>
      )}
      <div className="visually-hidden" role="status" aria-live="polite">
        Showing {rowCount} rows
      </div>
    </div>
  );
}

export default React.memo(Table);
