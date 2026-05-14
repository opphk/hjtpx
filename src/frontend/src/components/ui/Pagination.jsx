import React from 'react';

const Pagination = ({ 
  current = 1, 
  total = 0, 
  pageSize = 10,
  onChange,
  showTotal = true,
  className = ''
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

  const renderPageNumbers = () => {
    const pages = [];
    const maxVisible = 5;
    let start = Math.max(1, current - Math.floor(maxVisible / 2));
    let end = Math.min(totalPages, start + maxVisible - 1);
    
    if (end - start < maxVisible - 1) {
      start = Math.max(1, end - maxVisible + 1);
    }

    for (let i = start; i <= end; i++) {
      pages.push(
        <button
          key={i}
          className={`page-item ${i === current ? 'active' : ''}`}
          onClick={() => handlePageClick(i)}
        >
          {i}
        </button>
      );
    }
    
    return pages;
  };

  return (
    <div className={`pagination ${className}`}>
      {showTotal && (
        <span className="pagination-total">
          共 {total} 条记录
        </span>
      )}
      <div className="pagination-controls">
        <button 
          className="page-item"
          onClick={handlePrev}
          disabled={current === 1}
        >
          上一页
        </button>
        {renderPageNumbers()}
        <button 
          className="page-item"
          onClick={handleNext}
          disabled={current === totalPages}
        >
          下一页
        </button>
      </div>
    </div>
  );
};

export default Pagination;
