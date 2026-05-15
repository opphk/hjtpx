import React, { useEffect, useRef, useCallback } from 'react';

const Modal = ({ 
  isOpen, 
  onClose, 
  title, 
  children, 
  footer,
  size = 'medium',
  'aria-label': ariaLabel,
  'aria-describedby': ariaDescribedBy
}) => {
  const overlayRef = useRef(null);
  const modalRef = useRef(null);
  const titleId = useRef(`modal-title-${Date.now()}`).current;
  const bodyId = useRef(`modal-body-${Date.now()}`).current;
  const lastFocusedElementRef = useRef(null);

  const focusTrap = useCallback((e) => {
    if (!modalRef.current) return;
    
    const focusableElements = modalRef.current.querySelectorAll(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );
    
    if (focusableElements.length === 0) return;
    
    const firstElement = focusableElements[0];
    const lastElement = focusableElements[focusableElements.length - 1];
    
    if (e.key === 'Tab') {
      if (e.shiftKey && document.activeElement === firstElement) {
        e.preventDefault();
        lastElement.focus();
      } else if (!e.shiftKey && document.activeElement === lastElement) {
        e.preventDefault();
        firstElement.focus();
      }
    }
  }, []);

  useEffect(() => {
    if (isOpen) {
      lastFocusedElementRef.current = document.activeElement;
      document.body.style.overflow = 'hidden';
      
      const focusableElements = modalRef.current?.querySelectorAll(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      );
      
      if (focusableElements?.[0]) {
        focusableElements[0].focus();
      } else if (modalRef.current) {
        modalRef.current.focus();
      }
    } else {
      document.body.style.overflow = '';
      if (lastFocusedElementRef.current) {
        lastFocusedElementRef.current.focus();
      }
    }
    
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

  if (!isOpen) return null;

  const handleBackdropClick = (e) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  return (
    <div 
      className="modal-overlay" 
      ref={overlayRef}
      onClick={handleBackdropClick}
      role="presentation"
    >
      <div 
        className={`modal-container modal-${size}`}
        ref={modalRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={ariaLabel ? undefined : titleId}
        aria-describedby={ariaDescribedBy || (ariaLabel ? undefined : bodyId)}
        aria-label={ariaLabel}
        tabIndex={-1}
      >
        <div className="modal-header">
          <h3 className="modal-title" id={titleId}>{title}</h3>
          <button 
            className="modal-close" 
            onClick={onClose}
            aria-label="关闭对话框"
            type="button"
          >
            ×
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
};

export default Modal;
