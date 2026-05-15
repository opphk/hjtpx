import { describe, test, expect, beforeAll } from 'vitest';
import i18n from '../../src/i18n';
import { languages } from '../../src/i18n';
import en from '../../src/i18n/locales/en';
import zh from '../../src/i18n/locales/zh';
import fr from '../../src/i18n/locales/fr';
import de from '../../src/i18n/locales/de';
import es from '../../src/i18n/locales/es';
import ru from '../../src/i18n/locales/ru';
import ja from '../../src/i18n/locales/ja';
import ko from '../../src/i18n/locales/ko';
import ar from '../../src/i18n/locales/ar';
import pt from '../../src/i18n/locales/pt';
import it from '../../src/i18n/locales/it';
import nl from '../../src/i18n/locales/nl';

const allTranslations = { en, zh, fr, de, es, ru, ja, ko, ar, pt, it, nl };

describe('Internationalization Tests', () => {
  describe('Language Support', () => {
    test('should support 12 languages', () => {
      expect(languages.length).toBe(12);
    });

    test('should include all required language codes', () => {
      const requiredCodes = ['en', 'zh', 'fr', 'de', 'es', 'ru', 'ja', 'ko', 'ar', 'pt', 'it', 'nl'];
      const languageCodes = languages.map(l => l.code);
      requiredCodes.forEach(code => {
        expect(languageCodes).toContain(code);
      });
    });

    test('should have native names for all languages', () => {
      languages.forEach(lang => {
        expect(lang.nativeName).toBeDefined();
        expect(lang.nativeName.length).toBeGreaterThan(0);
      });
    });

    test('should have direction (dir) for all languages', () => {
      languages.forEach(lang => {
        expect(lang.dir).toBeDefined();
        expect(['ltr', 'rtl']).toContain(lang.dir);
      });
    });

    test('Arabic should be RTL', () => {
      const arabic = languages.find(l => l.code === 'ar');
      expect(arabic.dir).toBe('rtl');
    });

    test('English should be LTR', () => {
      const english = languages.find(l => l.code === 'en');
      expect(english.dir).toBe('ltr');
    });
  });

  describe('Translation Structure', () => {
    const requiredModules = ['common', 'nav', 'auth', 'dashboard', 'users', 'settings', 'logs', 'audit', 'dateTime', 'validation', 'pagination'];

    test('English translations should have all required modules', () => {
      requiredModules.forEach(module => {
        expect(en[module]).toBeDefined();
        expect(typeof en[module]).toBe('object');
      });
    });

    test('Chinese translations should have all required modules', () => {
      requiredModules.forEach(module => {
        expect(zh[module]).toBeDefined();
        expect(typeof zh[module]).toBe('object');
      });
    });

    test('French translations should have all required modules', () => {
      requiredModules.forEach(module => {
        expect(fr[module]).toBeDefined();
        expect(typeof fr[module]).toBe('object');
      });
    });

    test('All language translations should have consistent structure', () => {
      Object.entries(allTranslations).forEach(([langCode, translations]) => {
        requiredModules.forEach(module => {
          expect(translations[module]).toBeDefined();
        });
      });
    });
  });

  describe('Common Module Translations', () => {
    const requiredCommonKeys = [
      'appName', 'loading', 'save', 'cancel', 'delete', 'edit', 'add',
      'search', 'filter', 'export', 'import', 'refresh', 'back', 'next',
      'previous', 'submit', 'confirm', 'close', 'yes', 'no', 'success',
      'error', 'warning', 'info', 'apply', 'reset', 'clear'
    ];

    test('English common module should have all required keys', () => {
      requiredCommonKeys.forEach(key => {
        expect(en.common[key]).toBeDefined();
        expect(typeof en.common[key]).toBe('string');
      });
    });

    test('All languages should have common module keys', () => {
      Object.entries(allTranslations).forEach(([langCode, translations]) => {
        requiredCommonKeys.forEach(key => {
          expect(translations.common[key]).toBeDefined();
          expect(typeof translations.common[key]).toBe('string');
        });
      });
    });
  });

  describe('Auth Module Translations', () => {
    const requiredAuthKeys = [
      'login', 'logout', 'register', 'forgotPassword', 'email', 'password',
      'confirmPassword', 'username', 'rememberMe', 'noAccount', 'hasAccount',
      'signUpNow', 'signInNow', 'welcomeBack', 'pleaseLogin', 'createAccount',
      'joinUs', 'loginSuccess', 'registerSuccess', 'loginFailed', 'registerFailed'
    ];

    test('All languages should have auth module keys', () => {
      Object.entries(allTranslations).forEach(([langCode, translations]) => {
        requiredAuthKeys.forEach(key => {
          expect(translations.auth[key]).toBeDefined();
          expect(typeof translations.auth[key]).toBe('string');
        });
      });
    });
  });

  describe('Language Switching', () => {
    test('should change language correctly', async () => {
      await i18n.changeLanguage('zh');
      expect(i18n.language).toBe('zh');
      
      await i18n.changeLanguage('fr');
      expect(i18n.language).toBe('fr');
      
      await i18n.changeLanguage('ar');
      expect(i18n.language).toBe('ar');
    });

    test('should update document direction for RTL languages', async () => {
      await i18n.changeLanguage('ar');
      expect(document.documentElement.dir).toBe('rtl');
      
      await i18n.changeLanguage('en');
      expect(document.documentElement.dir).toBe('ltr');
    });

    test('should update document lang attribute', async () => {
      await i18n.changeLanguage('ja');
      expect(document.documentElement.lang).toBe('ja');
      
      await i18n.changeLanguage('ko');
      expect(document.documentElement.lang).toBe('ko');
    });
  });

  describe('RTL Support', () => {
    test('should identify RTL languages correctly', () => {
      const rtlLanguages = ['ar', 'he', 'fa', 'ur'];
      const ltrLanguages = ['en', 'zh', 'fr', 'de', 'es', 'ru', 'ja', 'ko', 'pt', 'it', 'nl'];
      
      rtlLanguages.forEach(lang => {
        const langConfig = languages.find(l => l.code === lang);
        if (langConfig) {
          expect(langConfig.dir).toBe('rtl');
        }
      });
      
      ltrLanguages.forEach(lang => {
        const langConfig = languages.find(l => l.code === lang);
        if (langConfig) {
          expect(langConfig.dir).toBe('ltr');
        }
      });
    });
  });

  describe('Translation Interpolation', () => {
    test('should support interpolation in translations', async () => {
      await i18n.changeLanguage('en');
      const translated = i18n.t('dateTime.daysAgo', { count: 5 });
      expect(translated).toContain('5');
    });

    test('should support interpolation in French', async () => {
      await i18n.changeLanguage('fr');
      const translated = i18n.t('dateTime.daysAgo', { count: 3 });
      expect(translated).toContain('3');
    });

    test('should support interpolation in German', async () => {
      await i18n.changeLanguage('de');
      const translated = i18n.t('validation.passwordMin', { min: 8 });
      expect(translated).toContain('8');
    });
  });

  describe('Fallback Language', () => {
    test('should fallback to English for unsupported language', async () => {
      const originalLanguage = i18n.language;
      await i18n.changeLanguage('unsupported-lang');
      expect(i18n.language).toBe('en');
      await i18n.changeLanguage(originalLanguage);
    });
  });
});
