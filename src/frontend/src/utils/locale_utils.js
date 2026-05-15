export const formatDate = (date, locale = 'en-US') => {
  return new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: 'long',
    day: 'numeric'
  }).format(new Date(date));
};

export const formatDateTime = (date, locale = 'en-US') => {
  return new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  }).format(new Date(date));
};

export const formatNumber = (number, locale = 'en-US') => {
  return new Intl.NumberFormat(locale).format(number);
};

export const formatCurrency = (amount, currency = 'USD', locale = 'en-US') => {
  return new Intl.NumberFormat(locale, {
    style: 'currency',
    currency
  }).format(amount);
};

export const formatRelativeTime = (date, locale = 'en-US') => {
  const now = new Date();
  const targetDate = new Date(date);
  const diffMs = targetDate - now;
  const diffSec = Math.round(diffMs / 1000);
  const diffMin = Math.round(diffMs / (1000 * 60));
  const diffHour = Math.round(diffMs / (1000 * 60 * 60));
  const diffDay = Math.round(diffMs / (1000 * 60 * 60 * 24));
  const diffWeek = Math.round(diffMs / (1000 * 60 * 60 * 24 * 7));
  const diffMonth = Math.round(diffMs / (1000 * 60 * 60 * 24 * 30));
  const diffYear = Math.round(diffMs / (1000 * 60 * 60 * 24 * 365));

  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' });

  if (Math.abs(diffSec) < 60) {
    return rtf.format(diffSec, 'second');
  } else if (Math.abs(diffMin) < 60) {
    return rtf.format(diffMin, 'minute');
  } else if (Math.abs(diffHour) < 24) {
    return rtf.format(diffHour, 'hour');
  } else if (Math.abs(diffDay) < 7) {
    return rtf.format(diffDay, 'day');
  } else if (Math.abs(diffWeek) < 4) {
    return rtf.format(diffWeek, 'week');
  } else if (Math.abs(diffMonth) < 12) {
    return rtf.format(diffMonth, 'month');
  } else {
    return rtf.format(diffYear, 'year');
  }
};

export const formatPercentage = (value, locale = 'en-US', decimals = 2) => {
  return new Intl.NumberFormat(locale, {
    style: 'percent',
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals
  }).format(value);
};

export const formatList = (items, locale = 'en-US', options = {}) => {
  return new Intl.ListFormat(locale, options).format(items);
};

export const getLocaleFromLanguage = (lang) => {
  const localeMap = {
    'en': 'en-US',
    'zh': 'zh-CN',
    'fr': 'fr-FR',
    'de': 'de-DE',
    'es': 'es-ES',
    'ru': 'ru-RU',
    'ja': 'ja-JP',
    'ko': 'ko-KR',
    'ar': 'ar-SA',
    'pt': 'pt-BR',
    'it': 'it-IT',
    'nl': 'nl-NL'
  };
  return localeMap[lang] || 'en-US';
};
