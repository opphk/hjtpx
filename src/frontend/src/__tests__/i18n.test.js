import { describe, it, expect } from 'vitest';
import i18n, { changeLanguage, getCurrentLanguage, isRTL, getSupportedLanguages, isLanguageLoaded } from '../i18n/index';
import { formatDate, formatDateTime, formatTime, formatShortDate, formatInTimezone, getSupportedLocales } from '../i18n/dateFormat';

describe('i18n', () => {
  describe('basic functionality', () => {
    it('should initialize with English as default', async () => {
      await new Promise(resolve => setTimeout(resolve, 100));
      expect(getSupportedLanguages()).toContain('en');
    });

    it('should support all required languages', () => {
      const languages = getSupportedLanguages();
      expect(languages).toContain('en');
      expect(languages).toContain('zh');
      expect(languages).toContain('fr');
      expect(languages).toContain('de');
      expect(languages).toContain('es');
      expect(languages).toContain('ru');
      expect(languages).toContain('ja');
      expect(languages).toContain('ko');
      expect(languages).toContain('ar');
      expect(languages).toContain('pt');
      expect(languages).toContain('it');
      expect(languages).toContain('nl');
    });

    it('should have 12 supported languages', () => {
      expect(getSupportedLanguages().length).toBe(12);
    });
  });

  describe('language switching', () => {
    it('should change language correctly', async () => {
      await changeLanguage('zh');
      await new Promise(resolve => setTimeout(resolve, 100));
      expect(getCurrentLanguage()).toBe('zh');
    });

    it('should change to French', async () => {
      await changeLanguage('fr');
      await new Promise(resolve => setTimeout(resolve, 100));
      expect(getCurrentLanguage()).toBe('fr');
    });

    it('should change to German', async () => {
      await changeLanguage('de');
      await new Promise(resolve => setTimeout(resolve, 100));
      expect(getCurrentLanguage()).toBe('de');
    });

    it('should change to Spanish', async () => {
      await changeLanguage('es');
      await new Promise(resolve => setTimeout(resolve, 100));
      expect(getCurrentLanguage()).toBe('es');
    });

    it('should change to Portuguese', async () => {
      await changeLanguage('pt');
      await new Promise(resolve => setTimeout(resolve, 100));
      expect(getCurrentLanguage()).toBe('pt');
    });

    it('should change to Italian', async () => {
      await changeLanguage('it');
      await new Promise(resolve => setTimeout(resolve, 100));
      expect(getCurrentLanguage()).toBe('it');
    });

    it('should change to Dutch', async () => {
      await changeLanguage('nl');
      await new Promise(resolve => setTimeout(resolve, 100));
      expect(getCurrentLanguage()).toBe('nl');
    });

    it('should change to Japanese', async () => {
      await changeLanguage('ja');
      await new Promise(resolve => setTimeout(resolve, 100));
      expect(getCurrentLanguage()).toBe('ja');
    });

    it('should change to Korean', async () => {
      await changeLanguage('ko');
      await new Promise(resolve => setTimeout(resolve, 100));
      expect(getCurrentLanguage()).toBe('ko');
    });

    it('should change to Russian', async () => {
      await changeLanguage('ru');
      await new Promise(resolve => setTimeout(resolve, 100));
      expect(getCurrentLanguage()).toBe('ru');
    });

    it('should change to Arabic (RTL)', async () => {
      await changeLanguage('ar');
      await new Promise(resolve => setTimeout(resolve, 100));
      expect(getCurrentLanguage()).toBe('ar');
      expect(isRTL('ar')).toBe(true);
    });
  });

  describe('RTL language support', () => {
    it('should identify RTL languages correctly', () => {
      expect(isRTL('ar')).toBe(true);
      expect(isRTL('en')).toBe(false);
      expect(isRTL('zh')).toBe(false);
      expect(isRTL('fr')).toBe(false);
      expect(isRTL('de')).toBe(false);
      expect(isRTL('es')).toBe(false);
    });
  });

  describe('lazy loading', () => {
    it('should preload language on demand', async () => {
      await changeLanguage('zh');
      await new Promise(resolve => setTimeout(resolve, 100));
      expect(isLanguageLoaded('zh')).toBe(true);
    });

    it('should cache loaded languages', async () => {
      await changeLanguage('fr');
      await new Promise(resolve => setTimeout(resolve, 100));
      expect(isLanguageLoaded('fr')).toBe(true);

      await changeLanguage('de');
      await new Promise(resolve => setTimeout(resolve, 100));
      expect(isLanguageLoaded('de')).toBe(true);
    });
  });
});

describe('Date formatting', () => {
  const testDate = new Date('2025-05-15T10:30:00');

  describe('formatDate', () => {
    it('should format date in English format', () => {
      const result = formatDate(testDate, 'en');
      expect(result).toMatch(/05\/15\/2025|5\/15\/2025/);
    });

    it('should format date in Chinese format', () => {
      const result = formatDate(testDate, 'zh');
      expect(result).toContain('2025');
      expect(result).toContain('05');
      expect(result).toContain('15');
    });

    it('should format date in Japanese format', () => {
      const result = formatDate(testDate, 'ja');
      expect(result).toContain('2025');
    });

    it('should format date in German format', () => {
      const result = formatDate(testDate, 'de');
      expect(result).toContain('.');
    });

    it('should format date in French format', () => {
      const result = formatDate(testDate, 'fr');
      expect(result).toContain('/');
    });

    it('should handle empty date', () => {
      expect(formatDate(null)).toBe('');
      expect(formatDate('')).toBe('');
      expect(formatDate(undefined)).toBe('');
    });
  });

  describe('formatDateTime', () => {
    it('should format datetime in English format', () => {
      const result = formatDateTime(testDate, 'en');
      expect(result).toMatch(/05\/15\/2025|5\/15\/2025/);
      expect(result).toMatch(/\d{2}:\d{2}:\d{2}/);
    });

    it('should format datetime in Chinese format', () => {
      const result = formatDateTime(testDate, 'zh');
      expect(result).toContain('2025');
      expect(result).toContain('10:30:00');
    });

    it('should handle empty datetime', () => {
      expect(formatDateTime(null)).toBe('');
      expect(formatDateTime('')).toBe('');
    });
  });

  describe('formatTime', () => {
    it('should format time correctly', () => {
      const result = formatTime(testDate, 'en');
      expect(result).toMatch(/\d{2}:\d{2}:\d{2}/);
    });

    it('should handle empty time', () => {
      expect(formatTime(null)).toBe('');
      expect(formatTime('')).toBe('');
    });
  });

  describe('formatShortDate', () => {
    it('should format short date for English', () => {
      const result = formatShortDate(testDate, 'en');
      expect(result).toMatch(/05\/15|5\/15/);
    });

    it('should format short date for German', () => {
      const result = formatShortDate(testDate, 'de');
      expect(result).toContain('.');
    });

    it('should handle empty date', () => {
      expect(formatShortDate(null)).toBe('');
    });
  });

  describe('timezone support', () => {
    it('should format date in specific timezone', () => {
      const result = formatInTimezone(testDate, 'America/New_York', 'yyyy-MM-dd HH:mm:ss');
      expect(result).toBeTruthy();
      expect(typeof result).toBe('string');
    });

    it('should handle empty date in timezone formatting', () => {
      expect(formatInTimezone(null, 'UTC')).toBe('');
    });
  });

  describe('getSupportedLocales', () => {
    it('should return all supported locales', () => {
      const locales = getSupportedLocales();
      expect(locales).toContain('en');
      expect(locales).toContain('zh');
      expect(locales).toContain('fr');
      expect(locales).toContain('de');
      expect(locales).toContain('es');
      expect(locales).toContain('ja');
      expect(locales).toContain('ko');
      expect(locales).toContain('ar');
      expect(locales).toContain('pt');
      expect(locales).toContain('it');
      expect(locales).toContain('nl');
      expect(locales).toContain('ru');
    });
  });
});

describe('Translation coverage', () => {
  const requiredKeys = [
    'common.appName',
    'common.loading',
    'common.save',
    'common.cancel',
    'common.delete',
    'common.edit',
    'common.add',
    'common.search',
    'common.filter',
    'common.export',
    'common.import',
    'search.placeholder',
    'nav.dashboard',
    'nav.users',
    'nav.settings',
    'auth.login',
    'auth.register',
    'auth.logout',
    'dashboard.welcome',
    'users.title',
    'settings.title',
    'logs.title',
    'audit.title',
    'dateTime.today',
    'dateTime.yesterday',
    'validation.required',
    'pagination.showing',
    'pagination.next',
    'pagination.previous'
  ];

  const languages = ['en', 'zh', 'fr', 'de', 'es', 'ru', 'ja', 'ko', 'ar', 'pt', 'it', 'nl'];

  languages.forEach(lang => {
    describe(`${lang} translations`, () => {
      requiredKeys.forEach(key => {
        it(`should have translation for ${key}`, async () => {
          await changeLanguage(lang);
          await new Promise(resolve => setTimeout(resolve, 50));
          const translation = i18n.t(key);
          expect(translation).toBeTruthy();
          expect(translation).not.toBe(key);
        });
      });
    });
  });
});
