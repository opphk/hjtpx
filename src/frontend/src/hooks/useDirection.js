import { useState, useEffect } from 'react';
import i18n from '../i18n';

export const useDirection = () => {
  const [direction, setDirection] = useState('ltr');
  
  useEffect(() => {
    const updateDirection = () => {
      const lang = i18n.language;
      const rtlLanguages = ['ar', 'he', 'fa', 'ur'];
      setDirection(rtlLanguages.includes(lang) ? 'rtl' : 'ltr');
    };
    
    updateDirection();
    i18n.on('languageChanged', updateDirection);
    
    return () => i18n.off('languageChanged', updateDirection);
  }, []);
  
  return direction;
};

export default useDirection;
