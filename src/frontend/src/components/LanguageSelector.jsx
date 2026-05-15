import { useTranslation } from 'react-i18next';
import { languages } from '../i18n';
import './LanguageSelector.css';

const LanguageSelector = ({ className = '', showFlag = true, showNativeName = false }) => {
  const { i18n } = useTranslation();

  const handleChange = (e) => {
    const newLang = e.target.value;
    i18n.changeLanguage(newLang);
    document.documentElement.dir = ['ar', 'he', 'fa', 'ur'].includes(newLang) ? 'rtl' : 'ltr';
    document.documentElement.lang = newLang;
  };

  return (
    <select 
      value={i18n.language} 
      onChange={handleChange}
      className={`language-selector ${className}`}
      aria-label="Select language"
    >
      {languages.map(lang => (
        <option key={lang.code} value={lang.code}>
          {showFlag ? `${lang.flag} ` : ''}{showNativeName ? lang.nativeName : lang.name}
        </option>
      ))}
    </select>
  );
};

export default LanguageSelector;
