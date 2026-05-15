import React from 'react';
import { useTranslation } from 'react-i18next';

const Pagination = ({ 
  current = 1, 
  total = 0, 
  pageSize = 10,
  onChange,
  showTotal = true,
  className = '',
  ...props
}) => {
  const { t } = useTranslation();
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
          aria-label={t('pagination.gotoPage', { page: i }) + (isCurrent ? ` (${t('pagination.currentPage')})` : '')}
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
      aria-label={t('pagination.ariaLabel')}
      {...props}
    >
      {showTotal && (
        <span className="pagination-total" aria-live="polite">
          {t('pagination.total', { total, pages: totalPages, current })}
        </span>
      )}
      <div 
        className="pagination-controls"
        role="list"
        aria-label={t('pagination.pageList')}
      >
        <button 
          className="page-item"
          onClick={handlePrev}
          onKeyDown={handlePrevKeyDown}
          disabled={current === 1}
          aria-disabled={current === 1}
          aria-label={t('pagination.previous')}
          type="button"
        >
          {t('pagination.previous')}
        </button>
        {renderPageNumbers()}
        <button 
          className="page-item"
          onClick={handleNext}
          onKeyDown={handleNextKeyDown}
          disabled={current === totalPages}
          aria-disabled={current === totalPages}
          aria-label={t('pagination.next')}
          type="button"
        >
          {t('pagination.next')}
        </button>
      </div>
    </nav>
  );
};

export default Pagination;
