import { useEffect, useRef, useCallback } from 'react';

export const useFocusTrap = (isActive, options = {}) => {
  const containerRef = useRef(null);
  const previousActiveElement = useRef(null);

  const {
    returnFocusOnDeactivate = true,
    initialFocus = null,
  } = options;

  const getFocusableElements = useCallback(() => {
    if (!containerRef.current) return [];
    
    const focusableSelectors = [
      'button:not([disabled])',
      '[href]',
      'input:not([disabled])',
      'select:not([disabled])',
      'textarea:not([disabled])',
      '[tabindex]:not([tabindex="-1"])',
      '[contenteditable="true"]',
    ].join(', ');

    return Array.from(
      containerRef.current.querySelectorAll(focusableSelectors)
    ).filter(el => {
      return el.offsetParent !== null;
    });
  }, []);

  const handleKeyDown = useCallback((e) => {
    if (!isActive || e.key !== 'Tab') return;
    
    const focusableElements = getFocusableElements();
    if (focusableElements.length === 0) return;

    const firstElement = focusableElements[0];
    const lastElement = focusableElements[focusableElements.length - 1];
    const activeElement = document.activeElement;

    if (e.shiftKey) {
      if (activeElement === firstElement || !containerRef.current.contains(activeElement)) {
        e.preventDefault();
        lastElement.focus();
      }
    } else {
      if (activeElement === lastElement || !containerRef.current.contains(activeElement)) {
        e.preventDefault();
        firstElement.focus();
      }
    }
  }, [isActive, getFocusableElements]);

  useEffect(() => {
    if (!isActive) return;

    previousActiveElement.current = document.activeElement;

    const focusInitialElement = () => {
      if (initialFocus) {
        const element = typeof initialFocus === 'string' 
          ? document.querySelector(initialFocus)
          : initialFocus;
        if (element) {
          element.focus();
          return;
        }
      }

      const focusableElements = getFocusableElements();
      if (focusableElements.length > 0) {
        focusableElements[0].focus();
      }
    };

    requestAnimationFrame(focusInitialElement);

    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      
      if (returnFocusOnDeactivate && previousActiveElement.current) {
        previousActiveElement.current.focus();
      }
    };
  }, [isActive, handleKeyDown, returnFocusOnDeactivate, initialFocus, getFocusableElements]);

  return containerRef;
};

export default useFocusTrap;
