import { describe, test, expect } from 'vitest';
import { formatDate, formatNumber, formatCurrency, formatRelativeTime, getLocaleFromLanguage } from '../../src/utils/locale_utils';

describe('Locale Utilities Tests', () => {
  describe('formatDate', () => {
    test('should format date in English locale', () => {
      const date = new Date('2024-01-15');
      const result = formatDate(date, 'en-US');
      expect(result).toContain('2024');
      expect(result).toContain('January');
      expect(result).toContain('15');
    });

    test('should format date in Chinese locale', () => {
      const date = new Date('2024-01-15');
      const result = formatDate(date, 'zh-CN');
      expect(result).toContain('2024');
    });

    test('should format date in French locale', () => {
      const date = new Date('2024-01-15');
      const result = formatDate(date, 'fr-FR');
      expect(result).toContain('2024');
      expect(result).toContain('janvier');
    });
  });

  describe('formatNumber', () => {
    test('should format number in English locale', () => {
      const result = formatNumber(1234567.89, 'en-US');
      expect(result).toContain('1');
      expect(result).toContain('2');
      expect(result).toContain('3');
    });

    test('should format number in German locale', () => {
      const result = formatNumber(1234567.89, 'de-DE');
      expect(result).toContain('1');
    });

    test('should format number in French locale', () => {
      const result = formatNumber(1234567.89, 'fr-FR');
      expect(result).toContain('1');
    });
  });

  describe('formatCurrency', () => {
    test('should format currency in USD', () => {
      const result = formatCurrency(1234.56, 'USD', 'en-US');
      expect(result).toContain('$');
      expect(result).toContain('1');
    });

    test('should format currency in EUR', () => {
      const result = formatCurrency(1234.56, 'EUR', 'de-DE');
      expect(result).toContain('1');
    });

    test('should format currency in CNY', () => {
      const result = formatCurrency(1234.56, 'CNY', 'zh-CN');
      expect(result).toContain('¥');
    });
  });

  describe('getLocaleFromLanguage', () => {
    test('should map en to en-US', () => {
      expect(getLocaleFromLanguage('en')).toBe('en-US');
    });

    test('should map zh to zh-CN', () => {
      expect(getLocaleFromLanguage('zh')).toBe('zh-CN');
    });

    test('should map fr to fr-FR', () => {
      expect(getLocaleFromLanguage('fr')).toBe('fr-FR');
    });

    test('should map de to de-DE', () => {
      expect(getLocaleFromLanguage('de')).toBe('de-DE');
    });

    test('should map ja to ja-JP', () => {
      expect(getLocaleFromLanguage('ja')).toBe('ja-JP');
    });

    test('should map ko to ko-KR', () => {
      expect(getLocaleFromLanguage('ko')).toBe('ko-KR');
    });

    test('should map ar to ar-SA', () => {
      expect(getLocaleFromLanguage('ar')).toBe('ar-SA');
    });

    test('should return en-US for unknown language', () => {
      expect(getLocaleFromLanguage('unknown')).toBe('en-US');
    });
  });
});
