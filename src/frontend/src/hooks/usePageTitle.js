import { useEffect } from 'react';

export const usePageTitle = (title, translate = false) => {
  useEffect(() => {
    const baseTitle = 'HJTPX 系统';
    const pageTitle = translate ? `${title} - ${baseTitle}` : `${title} | ${baseTitle}`;
    document.title = pageTitle;
    
    return () => {
      document.title = baseTitle;
    };
  }, [title]);
};

export const getPageTitle = (key, baseTitle = 'HJTPX 系统') => {
  return `${key} | ${baseTitle}`;
};
