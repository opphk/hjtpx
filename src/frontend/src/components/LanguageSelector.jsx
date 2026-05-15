import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { languages, changeLanguage as i18nChangeLanguage } from '../i18n';
import { isRTL } from '../i18n';
import './LanguageSelector.css';

const LanguageSelector = ({ 
  className = '', 
  showFlag = true, 
  showNativeName = false,
  showRegion = false,
  variant = 'dropdown',
  onLanguageChange 
}) => {
  const { i18n } = useTranslation();
  const [currentLang, setCurrentLang] = useState(i18n.language);
  const [isOpen, setIsOpen] = useState(false);

  useEffect(() => {
    setCurrentLang(i18n.language);
  }, [i18n.language]);

  const handleLanguageChange = useCallback(async (langCode) => {
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

  const handleToggle = useCallback(() => {
    setIsOpen(prev => !prev);
  }, []);

  const handleClose = useCallback(() => {
    setIsOpen(false);
  }, []);

  const handleClickOutside = useCallback((e) => {
    if (!e.target.closest('.language-selector')) {
      handleClose();
    }
  }, [handleClose]);

  useEffect(() => {
    if (isOpen) {
      document.addEventListener('click', handleClickOutside);
    }
    return () => {
      document.removeEventListener('click', handleClickOutside);
    };
  }, [isOpen, handleClickOutside]);

  const currentLanguage = useMemo(() => {
    return languages.find(lang => lang.code === currentLang) || languages[0];
  }, [currentLang]);

  const groupedLanguages = useMemo(() => {
    const groups = {};
    languages.forEach(lang => {
      const region = lang.region || 'Other';
      if (!groups[region]) {
        groups[region] = [];
      }
      groups[region].push(lang);
    });
    return groups;
  }, []);

  const renderDropdownContent = () => (
    <div className="language-dropdown-content">
      {Object.entries(groupedLanguages).map(([region, langs]) => (
        <div key={region} className="language-region-group">
          {showRegion && (
            <div className="language-region-label">{region}</div>
          )}
          {langs.map(lang => (
            <button
              key={lang.code}
              onClick={() => handleLanguageChange(lang.code)}
              className={`language-option ${lang.code === currentLang ? 'active' : ''}`}
              aria-selected={lang.code === currentLang}
            >
              {showFlag && <span className="language-flag">{lang.flag}</span>}
              <span className="language-names">
                <span className="language-name">{lang.name}</span>
                {showNativeName && (
                  <span className="language-native-name">{lang.nativeName}</span>
                )}
              </span>
              {lang.dir === 'rtl' && (
                <span className="language-rtl-badge">RTL</span>
              )}
            </button>
          ))}
        </div>
      ))}
    </div>
  );

  if (variant === 'buttons') {
    return (
      <div className={`language-selector language-selector-buttons ${className}`}>
        {languages.map(lang => (
          <button
            key={lang.code}
            onClick={() => handleLanguageChange(lang.code)}
            className={`language-button ${lang.code === currentLang ? 'active' : ''}`}
            title={lang.nativeName}
          >
            {showFlag && <span className="language-flag">{lang.flag}</span>}
            {showNativeName && <span>{lang.nativeName}</span>}
          </button>
        ))}
      </div>
    );
  }

  return (
    <div className={`language-selector language-selector-dropdown ${className}`}>
      <button
        onClick={handleToggle}
        className="language-selector-trigger"
        aria-expanded={isOpen}
        aria-haspopup="listbox"
        aria-label="Select language"
      >
        {showFlag && <span className="language-flag">{currentLanguage.flag}</span>}
        <span className="language-selected-name">
          {showNativeName ? currentLanguage.nativeName : currentLanguage.name}
        </span>
        <span className={`language-arrow ${isOpen ? 'open' : ''}`}>▼</span>
      </button>
      
      {isOpen && renderDropdownContent()}
    </div>
  );
};

export const LanguageDropdownMenu = ({ 
  trigger, 
  onLanguageChange,
  className = '' 
}) => {
  const { i18n } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);

  const handleLanguageSelect = async (langCode) => {
    setIsOpen(false);
    if (onLanguageChange) {
      onLanguageChange(langCode);
    }
  };

  return (
    <div className={`language-dropdown-menu ${className}`}>
      <div onClick={() => setIsOpen(!isOpen)}>
        {trigger}
      </div>
      {isOpen && (
        <>
          <div 
            className="dropdown-menu-overlay" 
            onClick={() => setIsOpen(false)} 
          />
          <div className="dropdown-menu-content">
            {languages.map(lang => (
              <button
                key={lang.code}
                onClick={() => handleLanguageSelect(lang.code)}
                className={`dropdown-menu-item ${lang.code === i18n.language ? 'active' : ''}`}
              >
                <span className="language-flag">{lang.flag}</span>
                <span>{lang.nativeName}</span>
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
};

export default LanguageSelector;
