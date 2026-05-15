import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { languages, changeLanguage as i18nChangeLanguage } from '../i18n';

const LanguageSwitcher = ({ onLanguageChange }) => {
  const { i18n } = useTranslation();
  const [currentLang, setCurrentLang] = useState(i18n.language || 'en');
  const [isOpen, setIsOpen] = useState(false);

  useEffect(() => {
    setCurrentLang(i18n.language);
  }, [i18n.language]);

  const handleChange = useCallback(async (e) => {
    const newLang = e.target.value;
    setCurrentLang(newLang);
    try {
      await i18nChangeLanguage(newLang);
      if (onLanguageChange) {
        onLanguageChange(newLang);
      }
    } catch (error) {
      console.error('Failed to change language:', error);
    }
  }, [onLanguageChange]);

  const handleDropdownChange = useCallback(async (langCode) => {
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

  return (
    <div className="language-switcher">
      <select 
        value={currentLang} 
        onChange={handleChange} 
        className="language-select"
        aria-label="Select language"
      >
        {languages.map((lang) => (
          <option key={lang.code} value={lang.code}>
            {lang.flag} {lang.name}
          </option>
        ))}
      </select>
    </div>
  );
};

export const LanguageSwitcherDropdown = ({ onLanguageChange }) => {
  const { i18n } = useTranslation();
  const [currentLang, setCurrentLang] = useState(i18n.language || 'en');
  const [isOpen, setIsOpen] = useState(false);

  useEffect(() => {
    setCurrentLang(i18n.language);
  }, [i18n.language]);

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

  return (
    <div className="language-switcher-dropdown">
      <button 
        className="language-switcher-trigger"
        onClick={() => setIsOpen(!isOpen)}
        aria-expanded={isOpen}
      >
        <span className="language-flag">{currentLanguage.flag}</span>
        <span>{currentLanguage.name}</span>
        <span className="dropdown-arrow">{isOpen ? '▲' : '▼'}</span>
      </button>
      
      {isOpen && (
        <div className="language-switcher-menu">
          {languages.map((lang) => (
            <button
              key={lang.code}
              onClick={() => handleLanguageSelect(lang.code)}
              className={`language-switcher-option ${lang.code === currentLang ? 'active' : ''}`}
            >
              <span className="language-flag">{lang.flag}</span>
              <span className="language-name">{lang.nativeName}</span>
              {lang.dir === 'rtl' && <span className="rtl-badge">RTL</span>}
            </button>
          ))}
        </div>
      )}
    </div>
  );
};

export default LanguageSwitcher;
