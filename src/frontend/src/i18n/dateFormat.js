import { format, formatDistanceToNow, isToday, isYesterday, parseISO } from 'date-fns';
import { enUS, zhCN, ja, ko, de, fr, es, pt, ru, it, nl, ar } from 'date-fns/locale';
import i18n from './index';

const localeMap = {
  en: enUS,
  zh: zhCN,
  ja: ja,
  ko: ko,
  de: de,
  fr: fr,
  es: es,
  pt: pt,
  ru: ru,
  it: it,
  nl: nl,
  ar: ar
};

const getDateFnsLocale = (lng) => {
  return localeMap[lng] || enUS;
};

const localeDateFormats = {
  en: 'MM/dd/yyyy',
  zh: 'yyyy年MM月dd日',
  ja: 'yyyy年MM月dd日',
  ko: 'yyyy년 MM월 dd일',
  de: 'dd.MM.yyyy',
  fr: 'dd/MM/yyyy',
  es: 'dd/MM/yyyy',
  pt: 'dd/MM/yyyy',
  ru: 'dd.MM.yyyy',
  it: 'dd/MM/yyyy',
  nl: 'dd/MM/yyyy',
  ar: 'dd/MM/yyyy'
};

const localeDateTimeFormats = {
  en: 'MM/dd/yyyy HH:mm:ss',
  zh: 'yyyy年MM月dd日 HH:mm:ss',
  ja: 'yyyy年MM月dd日 HH:mm:ss',
  ko: 'yyyy년 MM월 dd일 HH:mm:ss',
  de: 'dd.MM.yyyy HH:mm:ss',
  fr: 'dd/MM/yyyy HH:mm:ss',
  es: 'dd/MM/yyyy HH:mm:ss',
  pt: 'dd/MM/yyyy HH:mm:ss',
  ru: 'dd.MM.yyyy HH:mm:ss',
  it: 'dd/MM/yyyy HH:mm:ss',
  nl: 'dd/MM/yyyy HH:mm:ss',
  ar: 'dd/MM/yyyy HH:mm:ss'
};

const localeTimeFormats = {
  en: 'HH:mm:ss',
  zh: 'HH:mm:ss',
  ja: 'HH:mm:ss',
  ko: 'HH:mm:ss',
  de: 'HH:mm:ss',
  fr: 'HH:mm:ss',
  es: 'HH:mm:ss',
  pt: 'HH:mm:ss',
  ru: 'HH:mm:ss',
  it: 'HH:mm:ss',
  nl: 'HH:mm:ss',
  ar: 'HH:mm:ss'
};

const localeShortDateFormats = {
  en: 'MM/dd',
  zh: 'MM/dd',
  ja: 'MM/dd',
  ko: 'MM/dd',
  de: 'dd.MM',
  fr: 'dd/MM',
  es: 'dd/MM',
  pt: 'dd/MM',
  ru: 'dd.MM',
  it: 'dd/MM',
  nl: 'dd/MM',
  ar: 'dd/MM'
};

export const formatDate = (date, lng = i18n.language) => {
  if (!date) return '';
  const d = typeof date === 'string' ? parseISO(date) : date;
  const formatStr = localeDateFormats[lng] || localeDateFormats.en;
  const locale = getDateFnsLocale(lng);
  return format(d, formatStr, { locale });
};

export const formatDateTime = (date, lng = i18n.language) => {
  if (!date) return '';
  const d = typeof date === 'string' ? parseISO(date) : date;
  const formatStr = localeDateTimeFormats[lng] || localeDateTimeFormats.en;
  const locale = getDateFnsLocale(lng);
  return format(d, formatStr, { locale });
};

export const formatTime = (date, lng = i18n.language) => {
  if (!date) return '';
  const d = typeof date === 'string' ? parseISO(date) : date;
  const formatStr = localeTimeFormats[lng] || localeTimeFormats.en;
  const locale = getDateFnsLocale(lng);
  return format(d, formatStr, { locale });
};

export const formatTimeAgo = (date, lng = i18n.language) => {
  if (!date) return '';
  const d = typeof date === 'string' ? parseISO(date) : date;
  const now = new Date();
  const diffMs = now - d;
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) {
    return i18n.t('dateTime.justNow', { lng });
  } else if (diffMins < 60) {
    return i18n.t('dateTime.minutesAgo', { count: diffMins, lng });
  } else if (diffHours < 24) {
    return i18n.t('dateTime.hoursAgo', { count: diffHours, lng });
  } else {
    return i18n.t('dateTime.daysAgo', { count: diffDays, lng });
  }
};

export const formatRelativeTime = (date, lng = i18n.language) => {
  if (!date) return '';
  const d = typeof date === 'string' ? parseISO(date) : date;
  
  if (isToday(d)) {
    return i18n.t('dateTime.today');
  }
  
  if (isYesterday(d)) {
    return i18n.t('dateTime.yesterday');
  }
  
  const locale = getDateFnsLocale(lng);
  const distance = formatDistanceToNow(d, { addSuffix: true, locale });
  return distance;
};

export const formatDateRange = (startDate, endDate, lng = i18n.language) => {
  if (!startDate || !endDate) return '';
  const start = formatDate(startDate, lng);
  const end = formatDate(endDate, lng);
  return `${start} - ${end}`;
};

export const formatShortDate = (date, lng = i18n.language) => {
  if (!date) return '';
  const d = typeof date === 'string' ? parseISO(date) : date;
  const formatStr = localeShortDateFormats[lng] || localeShortDateFormats.en;
  const locale = getDateFnsLocale(lng);
  return format(d, formatStr, { locale });
};

export const getLocalizedDateFormat = (lng) => {
  return localeDateFormats[lng] || localeDateFormats.en;
};

export const getLocalizedDateTimeFormat = (lng) => {
  return localeDateTimeFormats[lng] || localeDateTimeFormats.en;
};

export const getLocalizedTimeFormat = (lng) => {
  return localeTimeFormats[lng] || localeTimeFormats.en;
};

export const getSupportedLocales = () => {
  return Object.keys(localeMap);
};
