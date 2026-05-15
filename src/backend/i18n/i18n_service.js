class I18nService {
  constructor() {
    this.currentLocale = 'en';
    this.translations = {};
    this.supportedLocales = [
      'en',
      'zh',
      'fr',
      'de',
      'es',
      'ru',
      'ja',
      'ko',
      'ar',
      'pt',
      'it',
      'nl'
    ];
    this.loadedLocales = new Set();
  }

  async loadTranslations(locale) {
    if (this.loadedLocales.has(locale)) {
      return this.translations[locale];
    }

    try {
      const translations = await import(`./locales/${locale}.json`);
      this.translations[locale] = translations.default || translations;
      this.loadedLocales.add(locale);
      return this.translations[locale];
    } catch (error) {
      console.error(`Failed to load translations for locale: ${locale}`, error);
      if (locale !== 'en') {
        return this.translations.en || {};
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
    const keys = key.split('.');
    let value = this.translations[this.currentLocale];

    for (const k of keys) {
      if (value && typeof value === 'object' && k in value) {
        value = value[k];
      } else {
        value = this.translations.en;
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

    return this.interpolate(value, params);
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
    const defaultOptions = {
      locale: this.currentLocale,
      ...options
    };
    return new Intl.DateTimeFormat(defaultOptions.locale, defaultOptions).format(new Date(date));
  }

  formatNumber(number, options = {}) {
    const defaultOptions = {
      locale: this.currentLocale
    };
    return new Intl.NumberFormat(defaultOptions.locale, options).format(number);
  }

  formatCurrency(amount, currency = 'USD', options = {}) {
    const defaultOptions = {
      locale: this.currentLocale,
      style: 'currency',
      currency
    };
    return new Intl.NumberFormat(defaultOptions.locale, { ...defaultOptions, ...options }).format(
      amount
    );
  }

  getSupportedLocales() {
    return this.supportedLocales;
  }

  isRTL() {
    return ['ar', 'he', 'fa', 'ur'].includes(this.currentLocale);
  }
}

const i18nService = new I18nService();

export default i18nService;
export { I18nService };
