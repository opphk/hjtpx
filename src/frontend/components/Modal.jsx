import React, { useEffect, useRef } from 'react';

function Modal({
  isOpen,
  onClose,
  title,
  children,
  footer,
  size = 'medium',
  closeOnOverlayClick = true,
  className = ''
}) {
  const modalRef = useRef(null);
  const previousActiveElement = useRef(null);
  const modalTitleId = `modal-title-${Math.random().toString(36).substr(2, 9)}`;

  useEffect(() => {
    if (isOpen) {
      previousActiveElement.current = document.activeElement;
      document.body.style.overflow = 'hidden';
      
      setTimeout(() => {
        const closeButton = modalRef.current?.querySelector('button:not([disabled])');
        if (closeButton) {
          closeButton.focus();
        } else {
          modalRef.current?.focus();
        }
      }, 100);
    } else {
      document.body.style.overflow = 'unset';
      if (previousActiveElement.current) {
        previousActiveElement.current.focus();
      }
    }

    return () => {
      document.body.style.overflow = 'unset';
    };
  }, [isOpen]);

  useEffect(() => {
    const handleEscape = (e) => {
      if (e.key === 'Escape' && isOpen) {
        e.stopPropagation();
        onClose();
      }
    };

    const handleTab = (e) => {
      if (e.key !== 'Tab' || !isOpen) return;

      const focusableElements = modalRef.current?.querySelectorAll(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      );
      
      if (!focusableElements || focusableElements.length === 0) return;

      const firstElement = focusableElements[0];
      const lastElement = focusableElements[focusableElements.length - 1];

      if (e.shiftKey && document.activeElement === firstElement) {
        e.preventDefault();
        lastElement.focus();
      } else if (!e.shiftKey && document.activeElement === lastElement) {
        e.preventDefault();
        firstElement.focus();
      }
    };

    document.addEventListener('keydown', handleEscape);
    document.addEventListener('keydown', handleTab);
    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.removeEventListener('keydown', handleTab);
    };
  }, [isOpen, onClose]);

  if (!isOpen) {
    return null;
  }

  const handleOverlayClick = (e) => {
    if (closeOnOverlayClick && e.target === e.currentTarget) {
      onClose();
    }
  };

  const handleOverlayKeyDown = (e) => {
    if ((e.key === 'Enter' || e.key === ' ') && closeOnOverlayClick) {
      e.preventDefault();
      onClose();
    }
  };

  const modalClass = `modal modal-${size} ${className}`.trim();

  return (
    <div 
      className="modal-overlay" 
      onClick={handleOverlayClick}
      onKeyDown={handleOverlayKeyDown}
      aria-hidden="true"
    >
      <div 
        className={modalClass} 
        role="dialog" 
        aria-modal="true"
        aria-labelledby={modalTitleId}
        ref={modalRef}
        tabIndex={-1}
      >
        <div className="modal-header">
          <h2 id={modalTitleId} className="modal-title">{title}</h2>
          <button
            type="button"
            className="modal-close"
            onClick={onClose}
            aria-label="Close dialog"
            aria-describedby={modalTitleId}
          >
            &times;
          </button>
        </div>
        <div className="modal-body" role="document">
          {children}
        </div>
        {footer && (
          <div className="modal-footer" role="group" aria-label="Dialog actions">
            {footer}
          </div>
        )}
      </div>
    </div>
  );
}

export default React.memo(Modal);
