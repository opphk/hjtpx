import React, { useEffect, useRef, useCallback } from 'react';

function Modal({
  isOpen,
  onClose,
  title,
  children,
  footer,
  size = 'medium',
  closeOnOverlayClick = true,
  className = '',
  'aria-label': ariaLabel,
  'aria-describedby': ariaDescribedBy
}) {
  const modalRef = useRef(null);
  const lastFocusedElementRef = useRef(null);
  const titleId = useRef(`modal-title-${Date.now()}`).current;
  const bodyId = useRef(`modal-body-${Date.now()}`).current;

  useEffect(() => {
    if (isOpen) {
      lastFocusedElementRef.current = document.activeElement;
      document.body.style.overflow = 'hidden';
      
      // Focus first interactive element or the modal itself
      setTimeout(() => {
        const focusableElements = modalRef.current?.querySelectorAll(
          'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );
        if (focusableElements && focusableElements.length > 0) {
          focusableElements[0].focus();
        } else if (modalRef.current) {
          modalRef.current.focus();
        }
      }, 10);
    } else {
      document.body.style.overflow = 'unset';
      if (lastFocusedElementRef.current) {
        lastFocusedElementRef.current.focus();
      }
    }

    return () => {
      document.body.style.overflow = 'unset';
    };
  }, [isOpen]);

  const focusTrap = useCallback((e) => {
    if (!modalRef.current || !isOpen || e.key !== 'Tab') return;
    
    const focusableElements = modalRef.current.querySelectorAll(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );
    
    if (focusableElements.length === 0) return;
    
    const firstElement = focusableElements[0];
    const lastElement = focusableElements[focusableElements.length - 1];
    
    if (e.shiftKey && document.activeElement === firstElement) {
      e.preventDefault();
      lastElement.focus();
    } else if (!e.shiftKey && document.activeElement === lastElement) {
      e.preventDefault();
      firstElement.focus();
    }
  }, [isOpen]);

  useEffect(() => {
    const handleEscape = (e) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
      document.addEventListener('keydown', focusTrap);
    }

    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.removeEventListener('keydown', focusTrap);
    };
  }, [isOpen, onClose, focusTrap]);

  if (!isOpen) {
    return null;
  }

  const handleOverlayClick = (e) => {
    if (closeOnOverlayClick && e.target === e.currentTarget) {
      onClose();
    }
  };

  const modalClass = `modal modal-${size} ${className}`.trim();

  return (
    <div 
      className="modal-overlay" 
      onClick={handleOverlayClick}
      role="presentation"
    >
      <div 
        className={modalClass} 
        ref={modalRef}
        role="dialog" 
        aria-modal="true"
        aria-labelledby={ariaLabel ? undefined : titleId}
        aria-describedby={ariaDescribedBy || (ariaLabel ? undefined : bodyId)}
        aria-label={ariaLabel}
        tabIndex={-1}
      >
        <div className="modal-header">
          <h2 className="modal-title" id={titleId}>{title}</h2>
          <button
            type="button"
            className="modal-close"
            onClick={onClose}
            aria-label="关闭对话框"
          >
            &times;
          </button>
        </div>
        <div className="modal-body" id={bodyId}>
          {children}
        </div>
        {footer && (
          <div className="modal-footer">
            {footer}
          </div>
        )}
      </div>
    </div>
  );
}

export default React.memo(Modal);
