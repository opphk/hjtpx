import React, { createContext, useContext, useState, useCallback } from 'react';

const AccessibilityContext = createContext();

export const useAccessibility = () => {
  const context = useContext(AccessibilityContext);
  if (!context) {
    throw new Error('useAccessibility must be used within AccessibilityProvider');
  }
  return context;
};

export const AccessibilityProvider = ({ children }) => {
  const [announcement, setAnnouncement] = useState('');
  const [politeAnnouncement, setPoliteAnnouncement] = useState('');

  const announce = useCallback((message, priority = 'polite') => {
    if (priority === 'assertive') {
      setAnnouncement(message);
      setTimeout(() => setAnnouncement(''), 1000);
    } else {
      setPoliteAnnouncement(message);
      setTimeout(() => setPoliteAnnouncement(''), 1000);
    }
  }, []);

  const value = {
    announce,
    announcement,
    politeAnnouncement
  };

  return (
    <AccessibilityContext.Provider value={value}>
      {children}
      <div 
        role="status" 
        aria-live="assertive" 
        aria-atomic="true"
        className="sr-only"
      >
        {announcement}
      </div>
      <div 
        role="status" 
        aria-live="polite" 
        aria-atomic="true"
        className="sr-only"
      >
        {politeAnnouncement}
      </div>
    </AccessibilityContext.Provider>
  );
};

export default AccessibilityProvider;
