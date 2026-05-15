import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  formatDate,
  formatDateTime,
  formatTime,
  formatTimeAgo,
  formatRelativeTime,
  formatShortDate,
  formatDateRange
} from '../i18n/dateFormat';

export const useDateFormat = () => {
  const { i18n } = useTranslation();
  const currentLanguage = i18n.language;

  const formatDateLocalized = useCallback((date) => {
    return formatDate(date, currentLanguage);
  }, [currentLanguage]);

  const formatDateTimeLocalized = useCallback((date) => {
    return formatDateTime(date, currentLanguage);
  }, [currentLanguage]);

  const formatTimeLocalized = useCallback((date) => {
    return formatTime(date, currentLanguage);
  }, [currentLanguage]);

  const formatTimeAgoLocalized = useCallback((date) => {
    return formatTimeAgo(date, currentLanguage);
  }, [currentLanguage]);

  const formatRelativeTimeLocalized = useCallback((date) => {
    return formatRelativeTime(date, currentLanguage);
  }, [currentLanguage]);

  const formatShortDateLocalized = useCallback((date) => {
    return formatShortDate(date, currentLanguage);
  }, [currentLanguage]);

  const formatDateRangeLocalized = useCallback((startDate, endDate) => {
    return formatDateRange(startDate, endDate, currentLanguage);
  }, [currentLanguage]);

  return {
    formatDate: formatDateLocalized,
    formatDateTime: formatDateTimeLocalized,
    formatTime: formatTimeLocalized,
    formatTimeAgo: formatTimeAgoLocalized,
    formatRelativeTime: formatRelativeTimeLocalized,
    formatShortDate: formatShortDateLocalized,
    formatDateRange: formatDateRangeLocalized,
    currentLanguage
  };
};

export default useDateFormat;
