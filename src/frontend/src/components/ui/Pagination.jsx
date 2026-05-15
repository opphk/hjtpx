import React from 'react';

const Pagination = ({ 
  current = 1, 
  total = 0, 
  pageSize = 10,
  onChange,
  showTotal = true,
  className = '',
  'aria-label': ariaLabel = '分页导航',
  ...props
}) => {
  const totalPages = Math.ceil(total / pageSize);
  
  if (totalPages <= 1) return null;

  const handlePrev = () => {
    if (current > 1) {
      onChange(current - 1);
    }
  };

  const handleNext = () => {
    if (current < totalPages) {
      onChange(current + 1);
    }
  };

  const handlePageClick = (page) => {
    if (page >= 1 && page <= totalPages && page !== current) {
      onChange(page);
    }
  };

  const handlePageKeyDown = (e, page) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handlePageClick(page);
    }
  };

  const handlePrevKeyDown = (e) => {
    if ((e.key === 'Enter' || e.key === ' ') && current > 1) {
      e.preventDefault();
      handlePrev();
    }
  };

  const handleNextKeyDown = (e) => {
    if ((e.key === 'Enter' || e.key === ' ') && current < totalPages) {
      e.preventDefault();
      handleNext();
    }
  };

  const renderPageNumbers = () => {
    const pages = [];
    const maxVisible = 5;
    let start = Math.max(1, current - Math.floor(maxVisible / 2));
    let end = Math.min(totalPages, start + maxVisible - 1);
    
    if (end - start < maxVisible - 1) {
      start = Math.max(1, end - maxVisible + 1);
    }

    for (let i = start; i <= end; i++) {
      const isCurrent = i === current;
      pages.push(
        <button
          key={i}
          className={`page-item ${isCurrent ? 'active' : ''}`}
          onClick={() => handlePageClick(i)}
          onKeyDown={(e) => handlePageKeyDown(e, i)}
          disabled={isCurrent}
          aria-current={isCurrent ? 'page' : 'false'}
          aria-label={`第 ${i} 页${isCurrent ? '（当前页）' : ''}`}
          type="button"
        >
          {i}
        </button>
      );
    }
    
    return pages;
  };

  return (
    <nav 
      className={`pagination ${className}`}
      role="navigation"
      aria-label={ariaLabel}
      {...props}
    >
      {showTotal && (
        <span className="pagination-total" aria-live="polite">
          共 {total} 条记录，共 {totalPages} 页，当前第 {current} 页
        </span>
      )}
      <div 
        className="pagination-controls"
        role="list"
        aria-label="页码列表"
      >
        <button 
          className="page-item"
          onClick={handlePrev}
          onKeyDown={handlePrevKeyDown}
          disabled={current === 1}
          aria-disabled={current === 1}
          aria-label="上一页"
          type="button"
        >
          上一页
        </button>
        {renderPageNumbers()}
        <button 
          className="page-item"
          onClick={handleNext}
          onKeyDown={handleNextKeyDown}
          disabled={current === totalPages}
          aria-disabled={current === totalPages}
          aria-label="下一页"
          type="button"
        >
          下一页
        </button>
      </div>
    </nav>
  );
};

export default Pagination;
