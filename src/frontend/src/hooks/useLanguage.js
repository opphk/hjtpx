import { useState, useCallback, useEffect } from 'react';
import i18n from '../i18n';
import { useDirection } from './useDirection';

export const useLanguage = () => {
  const [currentLanguage, setCurrentLanguage] = useState(i18n.language || 'en');
  const { setDirection } = useDirection();

  const changeLanguage = useCallback(async (language) => {
    try {
      await i18n.changeLanguage(language);
      setCurrentLanguage(language);

      if (['ar', 'he', 'fa', 'ur'].includes(language)) {
        setDirection('rtl');
      } else {
        setDirection('ltr');
      }

      localStorage.setItem('i18nextLng', language);

      return true;
    } catch (error) {
      console.error('Failed to change language:', error);
      return false;
    }
  }, [setDirection]);

  const t = useCallback((key, options) => {
    return i18n.t(key, options);
  }, [currentLanguage]);

  useEffect(() => {
    const savedLanguage = localStorage.getItem('i18nextLng');
    if (savedLanguage && savedLanguage !== currentLanguage) {
      changeLanguage(savedLanguage);
    }
  }, []);

  return {
    currentLanguage,
    changeLanguage,
    t,
    supportedLanguages: i18n.options.supportedLngs || ['en', 'zh', 'fr', 'de', 'es', 'ru', 'ja', 'ko', 'ar', 'pt', 'it', 'nl'],
    isRTL: ['ar', 'he', 'fa', 'ur'].includes(currentLanguage)
  };
};

export default useLanguage;
