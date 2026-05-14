import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  changeLanguage as i18nChangeLanguage,
  getCurrentLanguage,
  getLanguageInfo,
  getAvailableLanguages,
  isLanguageSupported,
  supportedLanguages
} from '../i18n/i18n';

export function useLanguage() {
  const { i18n, t } = useTranslation();
  const [currentLanguage, setCurrentLanguage] = useState(getCurrentLanguage());
  const [isChanging, setIsChanging] = useState(false);
  const [error, setError] = useState(null);

  useEffect(() => {
    const handleLanguageChanged = (lng) => {
      setCurrentLanguage(lng);
    };

    i18n.on('languageChanged', handleLanguageChanged);

    return () => {
      i18n.off('languageChanged', handleLanguageChanged);
    };
  }, [i18n]);

  const changeLanguage = useCallback(async (lng) => {
    if (!isLanguageSupported(lng)) {
      setError(`Language '${lng}' is not supported`);
      return false;
    }

    if (lng === currentLanguage) {
      return true;
    }

    setIsChanging(true);
    setError(null);

    try {
      await i18nChangeLanguage(lng);
      setCurrentLanguage(lng);
      setIsChanging(false);
      return true;
    } catch (err) {
      setError(err.message || 'Failed to change language');
      setIsChanging(false);
      return false;
    }
  }, [currentLanguage]);

  const languageInfo = getLanguageInfo(currentLanguage);
  const availableLanguages = getAvailableLanguages();

  const isRTL = languageInfo?.dir === 'rtl';
  const isLoading = isChanging;

  return {
    currentLanguage,
    changeLanguage,
    isChanging: isLoading,
    error,
    languageInfo,
    availableLanguages,
    isRTL,
    supportedLanguages,
    t
  };
}

export function useLanguageDetector() {
  const [detectedLanguage, setDetectedLanguage] = useState(null);
  const [isDetecting, setIsDetecting] = useState(true);
  const [confidence, setConfidence] = useState(0);

  useEffect(() => {
    const detectLanguage = async () => {
      setIsDetecting(true);

      try {
        const browserLang = navigator.language || navigator.userLanguage;
        const systemLang = Intl.DateTimeFormat().resolvedOptions().locale;

        const matchedLang = supportedLanguages.find(
          lang =>
            lang.code === browserLang ||
            lang.code === systemLang ||
            browserLang.startsWith(lang.code) ||
            systemLang.startsWith(lang.code)
        );

        if (matchedLang) {
          setDetectedLanguage(matchedLang.code);
          setConfidence(0.8);
        } else {
          setDetectedLanguage('zh');
          setConfidence(0.5);
        }
      } catch (error) {
        console.error('Language detection error:', error);
        setDetectedLanguage('zh');
        setConfidence(0.3);
      } finally {
        setIsDetecting(false);
      }
    };

    detectLanguage();
  }, []);

  return {
    detectedLanguage,
    isDetecting,
    confidence
  };
}

export function useLanguageSync() {
  const { i18n } = useTranslation();

  useEffect(() => {
    const syncInterval = setInterval(() => {
      const storedLang = localStorage.getItem('hjtpx-language');
      if (storedLang && storedLang !== i18n.language) {
        i18n.changeLanguage(storedLang);
      }
    }, 5000);

    return () => clearInterval(syncInterval);
  }, [i18n]);

  useEffect(() => {
    const handleStorageChange = (e) => {
      if (e.key === 'hjtpx-language' && e.newValue) {
        i18n.changeLanguage(e.newValue);
      }
    };

    window.addEventListener('storage', handleStorageChange);

    return () => {
      window.removeEventListener('storage', handleStorageChange);
    };
  }, [i18n]);
}

export function useTranslations() {
  const { t, i18n } = useTranslation();

  const translate = useCallback((key, options = {}) => {
    return t(key, options);
  }, [t]);

  const translateBatch = useCallback((keys, options = {}) => {
    return keys.map(key => t(key, options));
  }, [t]);

  const getTranslation = useCallback((key, defaultValue = '', options = {}) => {
    const translation = t(key, options);
    return translation === key ? defaultValue : translation;
  }, [t]);

  return {
    t: translate,
    translateBatch,
    getTranslation,
    currentLanguage: i18n.language,
    languages: supportedLanguages
  };
}
