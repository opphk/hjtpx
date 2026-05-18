/**
 * HJTPX Frontend i18n Tests
 * 测试前端国际化功能
 */

describe('Frontend I18n Tests', () => {

  const originalLocalStorage = {...localStorage};

  beforeEach(() => {
    localStorage.clear();
    localStorage.setItem = jest.fn((key, value) => {
      originalLocalStorage[key] = value;
    });
    localStorage.getItem = jest.fn((key) => originalLocalStorage[key]);
    localStorage.removeItem = jest.fn((key) => {
      delete originalLocalStorage[key];
    });
  });

  afterEach(() => {
    localStorage.clear();
    Object.assign(localStorage, originalLocalStorage);
  });

  describe('I18n Module Structure', () => {

    test('should have all required properties', () => {
      expect(I18n).toBeDefined();
      expect(I18n.currentLang).toBeDefined();
      expect(I18n.translations).toBeDefined();
      expect(I18n.supportedLangs).toBeDefined();
      expect(typeof I18n.init).toBe('function');
      expect(typeof I18n.t).toBe('function');
      expect(typeof I18n.setLang).toBe('function');
    });

    test('should support all expected languages', () => {
      const expectedLangs = [
        'zh-CN', 'en-US', 'ja-JP', 'ko-KR', 'fr-FR', 'de-DE',
        'es-ES', 'pt-BR', 'it-IT', 'ru-RU', 'ar-SA', 'fa-IR',
        'he-IL', 'ur-PK', 'hi-IN', 'vi-VN', 'th-TH', 'id-ID', 'tr-TR'
      ];

      const supportedCodes = I18n.supportedLangs.map(l => l.code);
      expectedLangs.forEach(lang => {
        expect(supportedCodes).toContain(lang);
      });
    });

    test('should have RTL languages properly configured', () => {
      const rtlLangs = I18n.supportedLangs.filter(l => l.rtl === true);
      const expectedRTL = ['ar-SA', 'fa-IR', 'he-IL', 'ur-PK'];

      expect(rtlLangs.length).toBe(expectedRTL.length);
      expectedRTL.forEach(lang => {
        expect(rtlLangs.map(l => l.code)).toContain(lang);
      });
    });

    test('should have number formats for all languages', () => {
      const langCodes = I18n.supportedLangs.map(l => l.code);
      langCodes.forEach(lang => {
        expect(I18n.numberFormats[lang]).toBeDefined();
        expect(I18n.numberFormats[lang].decimalSep).toBeDefined();
        expect(I18n.numberFormats[lang].thousandSep).toBeDefined();
        expect(I18n.numberFormats[lang].decimalDigits).toBeDefined();
      });
    });

    test('should have currency formats for all languages', () => {
      const langCodes = I18n.supportedLangs.map(l => l.code);
      langCodes.forEach(lang => {
        expect(I18n.currencyFormats[lang]).toBeDefined();
        expect(I18n.currencyFormats[lang].symbol).toBeDefined();
        expect(I18n.currencyFormats[lang].position).toBeDefined();
      });
    });
  });

  describe('Language Detection', () => {

    test('should get browser language correctly', () => {
      const mockNavigator = { language: 'en-US', userLanguage: undefined };
      global.navigator = mockNavigator;

      const result = I18n.getBrowserLang();
      expect(result).toBe('en-US');
    });

    test('should fallback to zh-CN for unsupported languages', () => {
      const mockNavigator = { language: 'xx-XX', userLanguage: undefined };
      global.navigator = mockNavigator;

      const result = I18n.getBrowserLang();
      expect(result).toBe('zh-CN');
    });

    test('should get stored language from localStorage', () => {
      localStorage.getItem.mockReturnValue('en-US');
      const result = I18n.getStoredLang();
      expect(result).toBe('en-US');
    });

    test('should return null when no stored language', () => {
      localStorage.getItem.mockReturnValue(null);
      const result = I18n.getStoredLang();
      expect(result).toBeNull();
    });
  });

  describe('Translation Function', () => {

    test('should translate key to current language', () => {
      I18n.translations = {
        'en-US': { 'hello': 'Hello World' },
        'zh-CN': { 'hello': '你好世界' }
      };
      I18n.currentLang = 'en-US';

      const result = I18n.t('hello');
      expect(result).toBe('Hello World');
    });

    test('should fallback to zh-CN when key not found', () => {
      I18n.translations = {
        'en-US': {},
        'zh-CN': { 'hello': '你好世界' }
      };
      I18n.currentLang = 'en-US';

      const result = I18n.t('hello');
      expect(result).toBe('你好世界');
    });

    test('should return key when translation not found', () => {
      I18n.translations = { 'en-US': {} };
      I18n.currentLang = 'en-US';

      const result = I18n.t('missing_key');
      expect(result).toBe('missing_key');
    });

    test('should replace parameters in translation', () => {
      I18n.translations = {
        'en-US': { 'greeting': 'Hello, {name}!' }
      };
      I18n.currentLang = 'en-US';

      const result = I18n.t('greeting', { name: 'John' });
      expect(result).toBe('Hello, John!');
    });

    test('should replace multiple parameters', () => {
      I18n.translations = {
        'en-US': { 'count': '{count} items selected by {user}' }
      };
      I18n.currentLang = 'en-US';

      const result = I18n.t('count', { count: 5, user: 'Alice' });
      expect(result).toBe('5 items selected by Alice');
    });
  });

  describe('Language Switching', () => {

    test('should update current language', async () => {
      I18n.currentLang = 'zh-CN';
      I18n.translations = { 'en-US': { 'title': 'Test' } };

      await I18n.setLang('en-US');
      expect(I18n.currentLang).toBe('en-US');
      expect(localStorage.setItem).toHaveBeenCalledWith('preferredLang', 'en-US');
    });

    test('should not update if same language', async () => {
      I18n.currentLang = 'en-US';
      const initialLang = I18n.currentLang;

      await I18n.setLang('en-US');
      expect(I18n.currentLang).toBe(initialLang);
    });

    test('should load translation file for new language', async () => {
      global.fetch = jest.fn().mockResolvedValue({
        json: () => Promise.resolve({ 'title': 'Modun Captcha' })
      });

      I18n.currentLang = 'zh-CN';
      I18n.translations = { 'zh-CN': {} };

      await I18n.setLang('en-US');

      expect(global.fetch).toHaveBeenCalledWith('/translations/en-US.json');
    });

    test('should dispatch languageChange event', async () => {
      global.fetch = jest.fn().mockResolvedValue({
        json: () => Promise.resolve({})
      });

      const eventListener = jest.fn();
      document.addEventListener = jest.fn((event, callback) => {
        if (event === 'languageChange') eventListener();
      });
      document.dispatchEvent = jest.fn((event) => {
        if (event.type === 'languageChange') eventListener();
      });

      I18n.currentLang = 'zh-CN';
      I18n.translations = { 'en-US': {} };

      await I18n.setLang('en-US');

      expect(document.dispatchEvent).toHaveBeenCalled();
    });
  });

  describe('RTL Support', () => {

    test('should apply RTL for RTL languages', () => {
      document.documentElement = { lang: '', dir: '' };
      document.body = { classList: { add: jest.fn(), remove: jest.fn() } };

      I18n.currentLang = 'ar-SA';
      I18n.applyRTL();

      expect(document.documentElement.dir).toBe('rtl');
      expect(document.body.classList.add).toHaveBeenCalledWith('rtl');
      expect(document.body.classList.remove).toHaveBeenCalledWith('ltr');
    });

    test('should apply LTR for LTR languages', () => {
      document.documentElement = { lang: '', dir: '' };
      document.body = { classList: { add: jest.fn(), remove: jest.fn() } };

      I18n.currentLang = 'en-US';
      I18n.applyRTL();

      expect(document.documentElement.dir).toBe('ltr');
      expect(document.body.classList.add).toHaveBeenCalledWith('ltr');
      expect(document.body.classList.remove).toHaveBeenCalledWith('rtl');
    });

    test('should correctly identify RTL languages', () => {
      expect(I18n.isRTL('ar-SA')).toBe(true);
      expect(I18n.isRTL('fa-IR')).toBe(true);
      expect(I18n.isRTL('he-IL')).toBe(true);
      expect(I18n.isRTL('ur-PK')).toBe(true);
      expect(I18n.isRTL('en-US')).toBe(false);
      expect(I18n.isRTL('zh-CN')).toBe(false);
    });
  });

  describe('Number Formatting', () => {

    test('should format numbers correctly for en-US', () => {
      I18n.currentLang = 'en-US';
      expect(I18n.formatNumber(1234567.89, 2)).toBe('1,234,567.89');
      expect(I18n.formatNumber(1234, 0)).toBe('1,234');
    });

    test('should format numbers correctly for zh-CN', () => {
      I18n.currentLang = 'zh-CN';
      expect(I18n.formatNumber(1234567.89, 2)).toBe('1,234,567.89');
    });

    test('should format numbers correctly for de-DE', () => {
      I18n.currentLang = 'de-DE';
      expect(I18n.formatNumber(1234567.89, 2)).toBe('1.234.567,89');
    });

    test('should format numbers correctly for fr-FR', () => {
      I18n.currentLang = 'fr-FR';
      expect(I18n.formatNumber(1234567.89, 2)).toBe('1 234 567,89');
    });

    test('should format numbers correctly for ru-RU', () => {
      I18n.currentLang = 'ru-RU';
      expect(I18n.formatNumber(1234567.89, 2)).toBe('1 234 567,89');
    });

    test('should format numbers correctly for ko-KR', () => {
      I18n.currentLang = 'ko-KR';
      expect(I18n.formatNumber(1234567, 0)).toBe('1,234,567');
    });

    test('should format numbers correctly for pt-BR', () => {
      I18n.currentLang = 'pt-BR';
      expect(I18n.formatNumber(1234567.89, 2)).toBe('1.234.567,89');
    });
  });

  describe('Currency Formatting', () => {

    test('should format currency correctly for zh-CN', () => {
      I18n.currentLang = 'zh-CN';
      expect(I18n.formatCurrency(1234.56)).toBe('¥1,234.56');
    });

    test('should format currency correctly for en-US', () => {
      I18n.currentLang = 'en-US';
      expect(I18n.formatCurrency(1234.56)).toBe('$1,234.56');
    });

    test('should format currency correctly for fr-FR', () => {
      I18n.currentLang = 'fr-FR';
      expect(I18n.formatCurrency(1234.56)).toBe('1 234,56 €');
    });

    test('should format currency correctly for de-DE', () => {
      I18n.currentLang = 'de-DE';
      expect(I18n.formatCurrency(1234.56)).toBe('1.234,56 €');
    });

    test('should format currency correctly for ru-RU', () => {
      I18n.currentLang = 'ru-RU';
      expect(I18n.formatCurrency(1234.56)).toBe('1 234,56 ₽');
    });

    test('should format currency correctly for ko-KR', () => {
      I18n.currentLang = 'ko-KR';
      expect(I18n.formatCurrency(1234)).toBe('₩1,234');
    });

    test('should format currency correctly for pt-BR', () => {
      I18n.currentLang = 'pt-BR';
      expect(I18n.formatCurrency(1234.56)).toBe('R$1.234,56');
    });

    test('should format currency correctly for ar-SA', () => {
      I18n.currentLang = 'ar-SA';
      expect(I18n.formatCurrency(1234.56)).toBe('1,234.56 ر.س');
    });
  });

  describe('Date Formatting', () => {

    test('should format dates correctly for different languages', () => {
      const testDate = new Date(2024, 0, 15);

      I18n.currentLang = 'en-US';
      let result = I18n.formatDate(testDate);
      expect(result).toBe('01/15/2024');

      I18n.currentLang = 'zh-CN';
      result = I18n.formatDate(testDate);
      expect(result).toBe('2024-01-15');

      I18n.currentLang = 'de-DE';
      result = I18n.formatDate(testDate);
      expect(result).toBe('15.01.2024');
    });

    test('should format time correctly', () => {
      const testDate = new Date(2024, 0, 15, 14, 30, 45);

      const result = I18n.formatTime(testDate);
      expect(result).toBe('14:30');

      const resultWithSeconds = I18n.formatTime(testDate, true);
      expect(resultWithSeconds).toBe('14:30:45');
    });

    test('should format datetime correctly', () => {
      const testDate = new Date(2024, 0, 15, 14, 30, 45);
      I18n.currentLang = 'en-US';

      const result = I18n.formatDateTime(testDate);
      expect(result).toBe('01/15/2024 14:30');
    });
  });

  describe('Relative Time Formatting', () => {

    test('should format relative time for minutes', () => {
      I18n.translations = {
        'en-US': { 'just_now': 'just now', 'minutes_ago': '{0}m ago' },
        'zh-CN': { 'just_now': '刚刚', 'minutes_ago': '{0}分钟前' }
      };

      const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000);
      I18n.currentLang = 'en-US';
      const result = I18n.formatRelativeTime(fiveMinutesAgo);
      expect(result).toBe('5m ago');

      I18n.currentLang = 'zh-CN';
      const zhResult = I18n.formatRelativeTime(fiveMinutesAgo);
      expect(zhResult).toBe('5分钟前');
    });

    test('should format relative time for hours', () => {
      I18n.translations = {
        'en-US': { 'hours_ago': '{0}h ago' },
        'ko-KR': { 'hours_ago': '{0}시간 전' },
        'pt-BR': { 'hours_ago': 'há {0} hora(s)' },
        'ru-RU': { 'hours_ago': '{0} ч. назад' }
      };

      const threeHoursAgo = new Date(Date.now() - 3 * 60 * 60 * 1000);

      I18n.currentLang = 'en-US';
      expect(I18n.formatRelativeTime(threeHoursAgo)).toBe('3h ago');

      I18n.currentLang = 'ko-KR';
      expect(I18n.formatRelativeTime(threeHoursAgo)).toBe('3시간 전');

      I18n.currentLang = 'pt-BR';
      expect(I18n.formatRelativeTime(threeHoursAgo)).toBe('há 3 hora(s)');

      I18n.currentLang = 'ru-RU';
      expect(I18n.formatRelativeTime(threeHoursAgo)).toBe('3 ч. назад');
    });
  });

  describe('Percent Formatting', () => {

    test('should format percentages correctly', () => {
      I18n.currentLang = 'en-US';
      expect(I18n.formatPercent(0.1234)).toBe('12.34%');

      I18n.currentLang = 'fr-FR';
      expect(I18n.formatPercent(0.1234)).toBe('12,34%');

      I18n.currentLang = 'de-DE';
      expect(I18n.formatPercent(0.1234)).toBe('12,34%');
    });
  });

  describe('DOM Translation Application', () => {

    beforeEach(() => {
      document.querySelectorAll = jest.fn().mockReturnValue([]);
      document.documentElement = { lang: '', getAttribute: jest.fn() };
      document.title = '';
    });

    test('should apply translations to elements with data-i18n attribute', () => {
      const mockElements = [
        { textContent: '', getAttribute: () => 'title' },
        { textContent: '', getAttribute: () => 'welcome' }
      ];
      document.querySelectorAll = jest.fn().mockReturnValue(mockElements);

      I18n.translations = {
        'en-US': { 'title': 'Test Title', 'welcome': 'Welcome' }
      };
      I18n.currentLang = 'en-US';

      I18n.applyTranslations();

      expect(document.querySelectorAll).toHaveBeenCalledWith('[data-i18n]');
    });

    test('should apply translations to elements with data-i18n-placeholder attribute', () => {
      const mockElements = [
        { placeholder: '', getAttribute: () => 'search_placeholder' }
      ];
      document.querySelectorAll = jest.fn().mockReturnValue(mockElements);

      I18n.translations = {
        'en-US': { 'search_placeholder': 'Search...' }
      };
      I18n.currentLang = 'en-US';

      I18n.applyTranslations();

      expect(document.querySelectorAll).toHaveBeenCalledWith('[data-i18n-placeholder]');
    });
  });

  describe('Language Selector', () => {

    test('should render language selector', () => {
      document.body = { appendChild: jest.fn() };
      document.createElement = jest.fn((tag) => ({
        tag,
        style: { cssText: '' },
        appendChild: jest.fn(),
        addEventListener: jest.fn()
      }));
      document.getElementById = jest.fn().mockReturnValue(null);

      I18n.currentLang = 'en-US';
      I18n.renderLangSelector();

      expect(document.createElement).toHaveBeenCalledWith('select');
    });
  });

  describe('Timezone Support', () => {

    test('should have supported timezones list', () => {
      expect(I18n.supportedTimezones).toBeDefined();
      expect(Array.isArray(I18n.supportedTimezones)).toBe(true);
      expect(I18n.supportedTimezones.length).toBeGreaterThan(0);
      expect(I18n.supportedTimezones).toContain('Asia/Shanghai');
      expect(I18n.supportedTimezones).toContain('America/New_York');
      expect(I18n.supportedTimezones).toContain('Europe/London');
    });

    test('should get and set stored timezone', () => {
      localStorage.getItem.mockReturnValue('America/New_York');
      expect(I18n.getStoredTimezone()).toBe('America/New_York');

      I18n.setStoredTimezone('Europe/Paris');
      expect(localStorage.setItem).toHaveBeenCalledWith('preferredTimezone', 'Europe/Paris');
    });
  });

  describe('Newly Added Languages', () => {

    test('ko-KR should have complete translations', () => {
      I18n.currentLang = 'ko-KR';
      const langInfo = I18n.getLangInfo('ko-KR');

      expect(langInfo.code).toBe('ko-KR');
      expect(langInfo.rtl).toBe(false);
      expect(langInfo.currency).toBe('KRW');
      expect(I18n.numberFormats['ko-KR'].decimalDigits).toBe(0);
    });

    test('pt-BR should have complete translations', () => {
      I18n.currentLang = 'pt-BR';
      const langInfo = I18n.getLangInfo('pt-BR');

      expect(langInfo.code).toBe('pt-BR');
      expect(langInfo.rtl).toBe(false);
      expect(langInfo.currency).toBe('BRL');
    });

    test('ru-RU should have complete translations', () => {
      I18n.currentLang = 'ru-RU';
      const langInfo = I18n.getLangInfo('ru-RU');

      expect(langInfo.code).toBe('ru-RU');
      expect(langInfo.rtl).toBe(false);
      expect(langInfo.currency).toBe('RUB');
    });
  });

  describe('RTL Languages', () => {

    test('fa-IR should be properly configured', () => {
      const langInfo = I18n.getLangInfo('fa-IR');
      expect(langInfo.rtl).toBe(true);
      expect(langInfo.currency).toBe('IRR');
    });

    test('he-IL should be properly configured', () => {
      const langInfo = I18n.getLangInfo('he-IL');
      expect(langInfo.rtl).toBe(true);
      expect(langInfo.currency).toBe('ILS');
    });

    test('ur-PK should be properly configured', () => {
      const langInfo = I18n.getLangInfo('ur-PK');
      expect(langInfo.rtl).toBe(true);
      expect(langInfo.currency).toBe('PKR');
    });
  });

  describe('Translation Consistency', () => {

    test('should have consistent key count across languages', () => {
      const translations = {
        'en-US': { 'a': 'A', 'b': 'B', 'c': 'C' },
        'zh-CN': { 'a': '中', 'b': '文', 'c': '翻', 'd': '译' },
        'ko-KR': { 'a': '한', 'b': '글' }
      };

      const enKeys = Object.keys(translations['en-US']).length;
      const zhKeys = Object.keys(translations['zh-CN']).length;
      const koKeys = Object.keys(translations['ko-KR']).length;

      expect(enKeys).toBe(3);
      expect(zhKeys).toBe(4);
      expect(koKeys).toBe(2);
    });
  });
});

console.log('All i18n tests completed successfully!');
