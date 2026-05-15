const fs = require('fs');
const path = require('path');

class I18nService {
  constructor() {
    this.currentLocale = 'en';
    this.translations = {};
    this.supportedLocales = ['en', 'zh', 'fr', 'de', 'es', 'ru', 'ja', 'ko', 'ar', 'pt', 'it', 'nl'];
    this.loadedLocales = new Set();
    this.cache = new Map();
    this.cacheTimeout = 3600000;
  }

  async loadTranslations(locale) {
    if (this.loadedLocales.has(locale)) {
      return this.translations[locale];
    }

    try {
      const filePath = path.join(__dirname, 'locales', `${locale}.json`);

      if (!fs.existsSync(filePath)) {
        console.warn(`Translation file not found for locale: ${locale}`);
        if (locale !== 'en') {
          return this.translations['en'] || {};
        }
        return {};
      }

      const content = fs.readFileSync(filePath, 'utf8');
      const translations = JSON.parse(content);

      this.translations[locale] = translations;
      this.loadedLocales.add(locale);

      return translations;
    } catch (error) {
      console.error(`Failed to load translations for locale: ${locale}`, error);
      if (locale !== 'en') {
        return this.translations['en'] || {};
      }
      return {};
    }
  }

  async setLocale(locale) {
    if (!this.supportedLocales.includes(locale)) {
      console.warn(`Unsupported locale: ${locale}, falling back to 'en'`);
      locale = 'en';
    }

    await this.loadTranslations(locale);
    this.currentLocale = locale;
    return this.currentLocale;
  }

  getLocale() {
    return this.currentLocale;
  }

  t(key, params = {}) {
    const cacheKey = `${this.currentLocale}:${key}:${JSON.stringify(params)}`;

    if (this.cache.has(cacheKey)) {
      const cached = this.cache.get(cacheKey);
      if (Date.now() - cached.timestamp < this.cacheTimeout) {
        return cached.value;
      }
    }

    const keys = key.split('.');
    let value = this.translations[this.currentLocale];

    for (const k of keys) {
      if (value && typeof value === 'object' && k in value) {
        value = value[k];
      } else {
        value = this.translations['en'];
        for (const fallbackKey of keys) {
          if (value && typeof value === 'object' && fallbackKey in value) {
            value = value[fallbackKey];
          } else {
            return key;
          }
        }
        break;
      }
    }

    if (typeof value !== 'string') {
      return key;
    }

    const result = this.interpolate(value, params);

    this.cache.set(cacheKey, {
      value: result,
      timestamp: Date.now()
    });

    return result;
  }

  interpolate(text, params) {
    return text.replace(/\{(\w+)\}/g, (match, key) => {
      return params.hasOwnProperty(key) ? params[key] : match;
    });
  }

  async initialize(defaultLocale = 'en') {
    await this.setLocale(defaultLocale);
  }

  formatDate(date, options = {}) {
    const localeMap = {
      'zh': 'zh-CN',
      'ja': 'ja-JP',
      'ko': 'ko-KR',
      'ar': 'ar-SA'
    };

    const locale = localeMap[this.currentLocale] || this.currentLocale;

    const defaultOptions = {
      locale,
      ...options
    };

    return new Intl.DateTimeFormat(locale, defaultOptions).format(new Date(date));
  }

  formatNumber(number, options = {}) {
    const localeMap = {
      'zh': 'zh-CN',
      'ja': 'ja-JP',
      'ko': 'ko-KR',
      'ar': 'ar-SA'
    };

    const locale = localeMap[this.currentLocale] || this.currentLocale;

    const defaultOptions = {
      locale
    };

    return new Intl.NumberFormat(locale, { ...defaultOptions, ...options }).format(number);
  }

  formatCurrency(amount, currency = 'USD', options = {}) {
    const localeMap = {
      'zh': 'zh-CN',
      'ja': 'ja-JP',
      'ko': 'ko-KR',
      'ar': 'ar-SA'
    };

    const locale = localeMap[this.currentLocale] || this.currentLocale;

    const defaultOptions = {
      locale,
      style: 'currency',
      currency
    };

    return new Intl.NumberFormat(locale, { ...defaultOptions, ...options }).format(amount);
  }

  getSupportedLocales() {
    return this.supportedLocales;
  }

  isRTL() {
    return ['ar', 'he', 'fa', 'ur'].includes(this.currentLocale);
  }

  getLocaleInfo(locale) {
    const localeInfo = {
      en: { name: 'English', nativeName: 'English', dir: 'ltr' },
      zh: { name: 'Chinese', nativeName: '中文', dir: 'ltr' },
      fr: { name: 'French', nativeName: 'Français', dir: 'ltr' },
      de: { name: 'German', nativeName: 'Deutsch', dir: 'ltr' },
      es: { name: 'Spanish', nativeName: 'Español', dir: 'ltr' },
      ru: { name: 'Russian', nativeName: 'Русский', dir: 'ltr' },
      ja: { name: 'Japanese', nativeName: '日本語', dir: 'ltr' },
      ko: { name: 'Korean', nativeName: '한국어', dir: 'ltr' },
      ar: { name: 'Arabic', nativeName: 'العربية', dir: 'rtl' },
      pt: { name: 'Portuguese', nativeName: 'Português', dir: 'ltr' },
      it: { name: 'Italian', nativeName: 'Italiano', dir: 'ltr' },
      nl: { name: 'Dutch', nativeName: 'Nederlands', dir: 'ltr' }
    };

    return localeInfo[locale] || { name: locale, nativeName: locale, dir: 'ltr' };
  }

  clearCache() {
    this.cache.clear();
  }

  reloadTranslations(locale) {
    if (this.loadedLocales.has(locale)) {
      this.loadedLocales.delete(locale);
      this.cache.clear();
    }
    return this.loadTranslations(locale);
  }
}

const i18nService = new I18nService();

module.exports = i18nService;
