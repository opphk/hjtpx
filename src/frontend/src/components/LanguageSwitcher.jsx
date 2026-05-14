import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';

const LanguageSwitcher = ({ onLanguageChange }) => {
  const { i18n, t } = useTranslation();
  const [currentLang, setCurrentLang] = useState(i18n.language || 'en');

  useEffect(() => {
    setCurrentLang(i18n.language);
  }, [i18n.language]);

  const languages = [
    { code: 'en', name: 'English', flag: '🇬🇧' },
    { code: 'zh', name: '中文', flag: '🇨🇳' },
    { code: 'ja', name: '日本語', flag: '🇯🇵' },
    { code: 'es', name: 'Español', flag: '🇪🇸' }
  ];

  const handleChange = (e) => {
    const newLang = e.target.value;
    setCurrentLang(newLang);
    i18n.changeLanguage(newLang);
    localStorage.setItem('language', newLang);
    if (onLanguageChange) {
      onLanguageChange(newLang);
    }
  };

  return (
    <div className="language-switcher">
      <select value={currentLang} onChange={handleChange} className="language-select">
        {languages.map((lang) => (
          <option key={lang.code} value={lang.code}>
            {lang.flag} {lang.name}
          </option>
        ))}
      </select>
    </div>
  );
};

export default LanguageSwitcher;
