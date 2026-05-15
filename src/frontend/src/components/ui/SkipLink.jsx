import React from 'react';
import { useTranslation } from 'react-i18next';

const SkipLink = ({ targetId = 'main-content', children }) => {
  const { t } = useTranslation();
  
  const handleClick = (e) => {
    e.preventDefault();
    const target = document.getElementById(targetId);
    if (target) {
      target.setAttribute('tabindex', '-1');
      target.focus();
      target.removeAttribute('tabindex');
    }
  };

  return (
    <a
      href={`#${targetId}`}
      className="skip-link"
      onClick={handleClick}
    >
      {children || t('a11y.skipToMain')}
    </a>
  );
};

export default SkipLink;
