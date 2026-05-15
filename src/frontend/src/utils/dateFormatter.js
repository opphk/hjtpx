import { format, formatDistanceToNow, parseISO } from 'date-fns';
import { enUS, zhCN, ja, ko, es, fr, de } from 'date-fns/locale';

const locales = {
  en: enUS,
  zh: zhCN,
  ja: ja,
  ko: ko,
  es: es,
  fr: fr,
  de: de
};

const getLocale = (language) => {
  return locales[language] || locales.en;
};

export const formatDate = (date, formatStr = 'PPP', language = 'en') => {
  if (!date) return '';
  
  const dateObj = typeof date === 'string' ? parseISO(date) : date;
  const locale = getLocale(language);
  
  return format(dateObj, formatStr, { locale });
};

export const formatDateTime = (date, language = 'en') => {
  return formatDate(date, 'PPp', language);
};

export const formatDateShort = (date, language = 'en') => {
  return formatDate(date, 'P', language);
};

export const formatTime = (date, language = 'en') => {
  return formatDate(date, 'p', language);
};

export const formatRelativeTime = (date, language = 'en') => {
  if (!date) return '';
  
  const dateObj = typeof date === 'string' ? parseISO(date) : date;
  const locale = getLocale(language);
  
  return formatDistanceToNow(dateObj, { addSuffix: true, locale });
};

export const formatDateRange = (startDate, endDate, language = 'en') => {
  if (!startDate || !endDate) return '';
  
  const start = formatDateShort(startDate, language);
  const end = formatDateShort(endDate, language);
  
  return `${start} - ${end}`;
};

export const getDateFormatPatterns = () => ({
  full: 'PPP',
  long: 'PPPP',
  medium: 'PP',
  short: 'P',
  time: 'p',
  datetime: 'PPp',
  datetimeshort: 'P p'
});

export const datePatterns = {
  en: {
    date: 'MM/dd/yyyy',
    datetime: 'MM/dd/yyyy hh:mm a',
    time: 'hh:mm a'
  },
  zh: {
    date: 'yyyy年MM月dd日',
    datetime: 'yyyy年MM月dd日 HH:mm',
    time: 'HH:mm'
  },
  ja: {
    date: 'yyyy年MM月dd日',
    datetime: 'yyyy年MM月dd日 HH:mm',
    time: 'HH:mm'
  },
  ko: {
    date: 'yyyy년 MM월 dd일',
    datetime: 'yyyy년 MM월 dd일 HH:mm',
    time: 'HH:mm'
  },
  es: {
    date: 'dd/MM/yyyy',
    datetime: 'dd/MM/yyyy HH:mm',
    time: 'HH:mm'
  },
  fr: {
    date: 'dd/MM/yyyy',
    datetime: 'dd/MM/yyyy HH:mm',
    time: 'HH:mm'
  },
  de: {
    date: 'dd.MM.yyyy',
    datetime: 'dd.MM.yyyy HH:mm',
    time: 'HH:mm'
  }
};

export const formatWithPattern = (date, type = 'date', language = 'en') => {
  const pattern = datePatterns[language]?.[type] || datePatterns.en[type];
  return formatDate(date, pattern, language);
};
