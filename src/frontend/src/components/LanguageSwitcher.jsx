import React, { useState, useEffect, useCallback, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { languages, changeLanguage as i18nChangeLanguage, isRTL } from '../i18n';

export const LanguageSwitcher = ({ onLanguageChange, compact = false }) => {
  const { i18n } = useTranslation();
  const [currentLang, setCurrentLang] = useState(i18n.language || 'en');
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef(null);

  useEffect(() => {
    setCurrentLang(i18n.language);
  }, [i18n.language]);

  useEffect(() => {
    const handleClickOutside = (event) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleLanguageSelect = useCallback(async (langCode) => {
    setCurrentLang(langCode);
    setIsOpen(false);
    try {
      await i18nChangeLanguage(langCode);
      if (onLanguageChange) {
        onLanguageChange(langCode);
      }
    } catch (error) {
      console.error('Failed to change language:', error);
    }
  }, [onLanguageChange]);

  const currentLanguage = languages.find(l => l.code === currentLang) || languages[0];

  const groupedLanguages = languages.reduce((acc, lang) => {
    if (!acc[lang.region]) {
      acc[lang.region] = [];
    }
    acc[lang.region].push(lang);
    return acc;
  }, {});

  if (compact) {
    return (
      <div className="language-switcher-compact" ref={dropdownRef}>
        <button 
          className="language-switcher-trigger-compact"
          onClick={() => setIsOpen(!isOpen)}
          aria-expanded={isOpen}
          aria-haspopup="listbox"
        >
          <span className="language-flag">{currentLanguage.flag}</span>
          <span className="language-code">{currentLanguage.code.toUpperCase()}</span>
        </button>
        
        {isOpen && (
          <div className="language-switcher-menu-compact" role="listbox">
            {languages.map((lang) => (
              <button
                key={lang.code}
                onClick={() => handleLanguageSelect(lang.code)}
                className={`language-switcher-option-compact ${lang.code === currentLang ? 'active' : ''}`}
                role="option"
                aria-selected={lang.code === currentLang}
              >
                <span className="language-flag">{lang.flag}</span>
                <span className="language-native">{lang.nativeName}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    );
  }

  return (
    <div className="language-switcher-dropdown" ref={dropdownRef}>
      <button 
        className="language-switcher-trigger"
        onClick={() => setIsOpen(!isOpen)}
        aria-expanded={isOpen}
        aria-haspopup="listbox"
      >
        <span className="language-flag">{currentLanguage.flag}</span>
        <span className="language-native">{currentLanguage.nativeName}</span>
        <span className="dropdown-arrow">{isOpen ? '▲' : '▼'}</span>
      </button>
      
      {isOpen && (
        <div className="language-switcher-menu" role="listbox">
          {Object.entries(groupedLanguages).map(([region, langs]) => (
            <div key={region} className="language-region-group">
              <div className="language-region-label">{region}</div>
              {langs.map((lang) => (
                <button
                  key={lang.code}
                  onClick={() => handleLanguageSelect(lang.code)}
                  className={`language-switcher-option ${lang.code === currentLang ? 'active' : ''}`}
                  role="option"
                  aria-selected={lang.code === currentLang}
                >
                  <span className="language-flag">{lang.flag}</span>
                  <span className="language-native">{lang.nativeName}</span>
                  {lang.dir === 'rtl' && <span className="rtl-badge">RTL</span>}
                </button>
              ))}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default LanguageSwitcher;
