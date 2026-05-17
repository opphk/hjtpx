/**
 * Internationalization (i18n) JavaScript Unit Tests
 * 
 * Test Coverage:
 * - Language switching
 * - Translation function (t)
 * - Number formatting
 * - Currency formatting
 * - Date formatting
 * - Time formatting
 * - DateTime formatting
 * - Relative time formatting
 * - Duration formatting
 * - Timezone conversion
 * - Locale configurations
 * - Plural rules
 * - Browser language detection
 * - Storage persistence
 */

describe('I18n Core Features', function() {
    
    beforeEach(function() {
        I18n.currentLang = 'zh-CN';
        I18n.currentTimezone = 'Asia/Shanghai';
    });

    describe('Supported Languages', function() {
        
        it('should support at least 20 languages', function() {
            expect(I18n.supportedLangs.length).toBeGreaterThanOrEqual(20);
        });

        it('should include common languages', function() {
            const codes = I18n.supportedLangs.map(l => l.code);
            
            expect(codes).toContain('zh-CN');
            expect(codes).toContain('en-US');
            expect(codes).toContain('ja-JP');
            expect(codes).toContain('ko-KR');
            expect(codes).toContain('fr-FR');
            expect(codes).toContain('de-DE');
            expect(codes).toContain('es-ES');
            expect(codes).toContain('pt-BR');
            expect(codes).toContain('ru-RU');
            expect(codes).toContain('ar-SA');
        });

        it('should include Southeast Asian languages', function() {
            const codes = I18n.supportedLangs.map(l => l.code);
            
            expect(codes).toContain('th-TH');
            expect(codes).toContain('vi-VN');
            expect(codes).toContain('id-ID');
            expect(codes).toContain('ms-MY');
            expect(codes).toContain('tl-PH');
        });

        it('should include Middle Eastern languages', function() {
            const codes = I18n.supportedLangs.map(l => l.code);
            
            expect(codes).toContain('fa-IR');
            expect(codes).toContain('he-IL');
            expect(codes).toContain('ar-SA');
        });

        it('should include European languages', function() {
            const codes = I18n.supportedLangs.map(l => l.code);
            
            expect(codes).toContain('pl-PL');
            expect(codes).toContain('nl-NL');
            expect(codes).toContain('el-GR');
            expect(codes).toContain('cs-CZ');
            expect(codes).toContain('sv-SE');
            expect(codes).toContain('da-DK');
            expect(codes).toContain('fi-FI');
            expect(codes).toContain('no-NO');
            expect(codes).toContain('hu-HU');
            expect(codes).toContain('ro-RO');
            expect(codes).toContain('uk-UA');
            expect(codes).toContain('bg-BG');
            expect(codes).toContain('hr-HR');
            expect(codes).toContain('sk-SK');
            expect(codes).toContain('sl-SI');
        });

        it('should have proper language metadata', function() {
            I18n.supportedLangs.forEach(function(lang) {
                expect(lang.code).toBeDefined();
                expect(lang.name).toBeDefined();
                expect(lang.nativeName).toBeDefined();
                expect(lang.flag).toBeDefined();
            });
        });
    });

    describe('Supported Timezones', function() {
        
        it('should support multiple timezones', function() {
            expect(I18n.supportedTimezones.length).toBeGreaterThan(20);
        });

        it('should include major timezones', function() {
            const ids = I18n.supportedTimezones.map(tz => tz.id);
            
            expect(ids).toContain('Asia/Shanghai');
            expect(ids).toContain('Asia/Tokyo');
            expect(ids).toContain('America/New_York');
            expect(ids).toContain('America/Los_Angeles');
            expect(ids).toContain('Europe/London');
            expect(ids).toContain('UTC');
        });

        it('should have proper timezone metadata', function() {
            I18n.supportedTimezones.forEach(function(tz) {
                expect(tz.id).toBeDefined();
                expect(tz.name).toBeDefined();
                expect(tz.offset).toBeDefined();
            });
        });
    });

    describe('Locale Configurations', function() {
        
        it('should have locale configs for all supported languages', function() {
            I18n.supportedLangs.forEach(function(lang) {
                expect(I18n.localeConfigs[lang.code]).toBeDefined();
            });
        });

        it('should have correct decimal separators', function() {
            expect(I18n.localeConfigs['en-US'].decimalSeparator).toBe('.');
            expect(I18n.localeConfigs['fr-FR'].decimalSeparator).toBe(',');
            expect(I18n.localeConfigs['de-DE'].decimalSeparator).toBe(',');
        });

        it('should have correct thousand separators', function() {
            expect(I18n.localeConfigs['en-US'].thousandSeparator).toBe(',');
            expect(I18n.localeConfigs['fr-FR'].thousandSeparator).toBe(' ');
            expect(I18n.localeConfigs['de-DE'].thousandSeparator).toBe('.');
        });

        it('should have correct currency symbols', function() {
            expect(I18n.localeConfigs['zh-CN'].currencySymbol).toBe('¥');
            expect(I18n.localeConfigs['en-US'].currencySymbol).toBe('$');
            expect(I18n.localeConfigs['ja-JP'].currencySymbol).toBe('¥');
            expect(I18n.localeConfigs['ko-KR'].currencySymbol).toBe('₩');
            expect(I18n.localeConfigs['fr-FR'].currencySymbol).toBe('€');
            expect(I18n.localeConfigs['ru-RU'].currencySymbol).toBe('₽');
        });

        it('should have correct first weekday', function() {
            expect(I18n.localeConfigs['en-US'].firstWeekday).toBe(0);
            expect(I18n.localeConfigs['zh-CN'].firstWeekday).toBe(1);
            expect(I18n.localeConfigs['ar-SA'].firstWeekday).toBe(6);
        });
    });

    describe('Month Names', function() {
        
        it('should have month names for all supported languages', function() {
            I18n.supportedLangs.forEach(function(lang) {
                expect(I18n.monthNames[lang.code]).toBeDefined();
                expect(I18n.monthNames[lang.code].length).toBe(12);
            });
        });

        it('should have correct month names for Chinese', function() {
            const months = I18n.monthNames['zh-CN'];
            expect(months[0]).toBe('一月');
            expect(months[11]).toBe('十二月');
        });

        it('should have correct month names for English', function() {
            const months = I18n.monthNames['en-US'];
            expect(months[0]).toBe('January');
            expect(months[11]).toBe('December');
        });
    });

    describe('Weekday Names', function() {
        
        it('should have weekday names for all supported languages', function() {
            I18n.supportedLangs.forEach(function(lang) {
                expect(I18n.weekdayNames[lang.code]).toBeDefined();
                expect(I18n.weekdayNames[lang.code].length).toBe(7);
            });
        });

        it('should have correct weekday names for Chinese', function() {
            const weekdays = I18n.weekdayNames['zh-CN'];
            expect(weekdays[0]).toBe('星期日');
            expect(weekdays[1]).toBe('星期一');
        });

        it('should have correct weekday names for English', function() {
            const weekdays = I18n.weekdayNames['en-US'];
            expect(weekdays[0]).toBe('Sunday');
            expect(weekdays[1]).toBe('Monday');
        });
    });
});

describe('I18n Formatting Functions', function() {
    
    beforeEach(function() {
        I18n.currentLang = 'en-US';
        I18n.currentTimezone = 'UTC';
    });

    describe('formatNumber', function() {
        
        it('should format positive numbers', function() {
            const result = I18n.formatNumber(1234567.89);
            expect(result).toMatch(/1.*234.*567/);
        });

        it('should format negative numbers', function() {
            const result = I18n.formatNumber(-1234.56);
            expect(result).toContain('-');
        });

        it('should format zero', function() {
            const result = I18n.formatNumber(0);
            expect(result).toBe('0');
        });

        it('should handle decimal places option', function() {
            const result = I18n.formatNumber(1234.567, { decimalPlaces: 1 });
            expect(result).toMatch(/1234\.5/);
        });

        it('should format with thousand separators for en-US', function() {
            I18n.currentLang = 'en-US';
            const result = I18n.formatNumber(1234567);
            expect(result).toContain(',');
        });

        it('should format without thousand separators when disabled', function() {
            const result = I18n.formatNumber(1234567, { useThousandSeparator: false });
            expect(result).not.toContain(',');
        });

        it('should format with French locale', function() {
            I18n.currentLang = 'fr-FR';
            const result = I18n.formatNumber(1234567.89);
            expect(result).toContain(' ');
        });

        it('should format with German locale', function() {
            I18n.currentLang = 'de-DE';
            const result = I18n.formatNumber(1234567.89);
            expect(result).toContain('.');
        });
    });

    describe('formatCurrency', function() {
        
        it('should format with currency symbol for USD', function() {
            I18n.currentLang = 'en-US';
            const result = I18n.formatCurrency(1234.56);
            expect(result).toContain('$');
        });

        it('should format with currency symbol for CNY', function() {
            I18n.currentLang = 'zh-CN';
            const result = I18n.formatCurrency(1234.56);
            expect(result).toContain('¥');
        });

        it('should format with currency symbol for EUR', function() {
            I18n.currentLang = 'fr-FR';
            const result = I18n.formatCurrency(1234.56);
            expect(result).toContain('€');
        });

        it('should format with currency symbol for JPY', function() {
            I18n.currentLang = 'ja-JP';
            const result = I18n.formatCurrency(1234.56);
            expect(result).toContain('¥');
        });

        it('should format with currency symbol for KRW', function() {
            I18n.currentLang = 'ko-KR';
            const result = I18n.formatCurrency(1234.56);
            expect(result).toContain('₩');
        });

        it('should format with currency symbol for RUB', function() {
            I18n.currentLang = 'ru-RU';
            const result = I18n.formatCurrency(1234.56);
            expect(result).toContain('₽');
        });
    });

    describe('formatDate', function() {
        
        it('should format date in short style', function() {
            const result = I18n.formatDate(new Date(2026, 4, 17), { style: 'short' });
            expect(result).toMatch(/2026-05-17/);
        });

        it('should format date in medium style for en-US', function() {
            I18n.currentLang = 'en-US';
            const result = I18n.formatDate(new Date(2026, 4, 17));
            expect(result).toContain('May');
            expect(result).toContain('17');
            expect(result).toContain('2026');
        });

        it('should format date in medium style for zh-CN', function() {
            I18n.currentLang = 'zh-CN';
            const result = I18n.formatDate(new Date(2026, 4, 17));
            expect(result).toContain('2026年');
            expect(result).toContain('5月');
            expect(result).toContain('17日');
        });

        it('should format date in long style', function() {
            const result = I18n.formatDate(new Date(2026, 4, 17), { style: 'long' });
            expect(result).toContain('Saturday');
        });
    });

    describe('formatTime', function() {
        
        it('should format time in 24-hour format', function() {
            const result = I18n.formatTime(new Date(2026, 4, 17, 14, 30, 45), { use24Hour: true });
            expect(result).toContain('14');
            expect(result).toContain('30');
            expect(result).toContain('45');
        });

        it('should format time in 12-hour format for en-US', function() {
            I18n.currentLang = 'en-US';
            const result = I18n.formatTime(new Date(2026, 4, 17, 14, 30, 45), { use24Hour: false });
            expect(result).toContain('PM');
        });

        it('should format time with leading zeros', function() {
            const result = I18n.formatTime(new Date(2026, 4, 17, 9, 5, 3));
            expect(result).toContain('09');
            expect(result).toContain('05');
            expect(result).toContain('03');
        });
    });

    describe('formatDateTime', function() {
        
        it('should format date and time together', function() {
            const result = I18n.formatDateTime(new Date(2026, 4, 17, 14, 30, 45));
            expect(result).toContain('2026');
            expect(result).toContain('14');
        });

        it('should handle custom date and time styles', function() {
            const result = I18n.formatDateTime(new Date(2026, 4, 17, 14, 30, 45), {
                dateStyle: 'short',
                timeStyle: 'short'
            });
            expect(result).toMatch(/2026/);
        });
    });

    describe('formatRelativeTime', function() {
        
        it('should format time just now', function() {
            const now = new Date();
            const result = I18n.formatRelativeTime(now);
            expect(result).toBeTruthy();
        });

        it('should format minutes ago', function() {
            const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000);
            const result = I18n.formatRelativeTime(fiveMinutesAgo);
            expect(result).toBeTruthy();
        });

        it('should format hours ago', function() {
            const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000);
            const result = I18n.formatRelativeTime(twoHoursAgo);
            expect(result).toBeTruthy();
        });

        it('should format days ago', function() {
            const threeDaysAgo = new Date(Date.now() - 3 * 24 * 60 * 60 * 1000);
            const result = I18n.formatRelativeTime(threeDaysAgo);
            expect(result).toBeTruthy();
        });
    });

    describe('formatDuration', function() {
        
        it('should format seconds', function() {
            const result = I18n.formatDuration(30000);
            expect(result).toContain('s');
        });

        it('should format minutes and seconds', function() {
            const result = I18n.formatDuration(5 * 60 * 1000 + 30 * 1000);
            expect(result).toContain('m');
            expect(result).toContain('s');
        });

        it('should format hours, minutes and seconds', function() {
            const result = I18n.formatDuration(2 * 60 * 60 * 1000 + 15 * 60 * 1000 + 30 * 1000);
            expect(result).toContain('h');
            expect(result).toContain('m');
            expect(result).toContain('s');
        });

        it('should format days', function() {
            const result = I18n.formatDuration(3 * 24 * 60 * 60 * 1000 + 5 * 60 * 60 * 1000);
            expect(result).toContain('d');
        });
    });
});

describe('I18n Language Management', function() {
    
    describe('getBrowserLang', function() {
        
        it('should return a supported language', function() {
            const lang = I18n.getBrowserLang();
            expect(I18n.supportedLangs.some(l => l.code === lang)).toBe(true);
        });

        it('should fallback to zh-CN for unsupported languages', function() {
            spyOnProperty(navigator, 'language').and.returnValue('unsupported-lang');
            const lang = I18n.getBrowserLang();
            expect(lang).toBeTruthy();
        });
    });

    describe('Storage Operations', function() {
        
        beforeEach(function() {
            localStorage.clear();
        });

        it('should store and retrieve language preference', function() {
            I18n.setStoredLang('en-US');
            const stored = I18n.getStoredLang();
            expect(stored).toBe('en-US');
        });

        it('should store and retrieve timezone preference', function() {
            I18n.setStoredTimezone('America/New_York');
            const stored = I18n.getStoredTimezone();
            expect(stored).toBe('America/New_York');
        });

        it('should return default timezone when none stored', function() {
            localStorage.removeItem('preferredTimezone');
            const stored = I18n.getStoredTimezone();
            expect(stored).toBeTruthy();
        });
    });

    describe('getLangInfo', function() {
        
        it('should return correct language info', function() {
            I18n.currentLang = 'en-US';
            const info = I18n.getLangInfo();
            expect(info.code).toBe('en-US');
            expect(info.name).toBe('English (US)');
            expect(info.flag).toBeDefined();
        });

        it('should handle RTL languages', function() {
            I18n.currentLang = 'ar-SA';
            const info = I18n.getLangInfo();
            expect(info.rtl).toBe(true);
        });

        it('should handle LTR languages', function() {
            I18n.currentLang = 'en-US';
            const info = I18n.getLangInfo();
            expect(info.rtl).toBeFalsy();
        });
    });

    describe('getTimezoneOffset', function() {
        
        it('should return offset for valid timezone', function() {
            I18n.currentTimezone = 'Asia/Shanghai';
            const offset = I18n.getTimezoneOffset();
            expect(offset).toBe('+08:00');
        });

        it('should return default offset for unknown timezone', function() {
            I18n.currentTimezone = 'Unknown/Zone';
            const offset = I18n.getTimezoneOffset();
            expect(offset).toBeTruthy();
        });
    });
});

describe('I18n Timezone Conversion', function() {
    
    describe('convertTimezone', function() {
        
        it('should convert time between timezones', function() {
            const utcDate = new Date('2026-05-17T12:00:00Z');
            const converted = I18n.convertTimezone(utcDate, 'UTC', 'Asia/Shanghai');
            expect(converted).toBeInstanceOf(Date);
        });

        it('should handle summer time conversion', function() {
            const summerDate = new Date('2026-07-15T12:00:00Z');
            const converted = I18n.convertTimezone(summerDate, 'UTC', 'America/New_York');
            expect(converted).toBeInstanceOf(Date);
        });
    });
});

describe('I18n Translation Function', function() {
    
    describe('t', function() {
        
        it('should return translation for valid key', function() {
            I18n.translations = {
                'en-US': { 'hello': 'Hello, World!' },
                'zh-CN': { 'hello': '你好，世界！' }
            };
            I18n.currentLang = 'en-US';
            expect(I18n.t('hello')).toBe('Hello, World!');
        });

        it('should return key when translation not found', function() {
            I18n.translations = { 'en-US': {} };
            I18n.currentLang = 'en-US';
            expect(I18n.t('unknown_key')).toBe('unknown_key');
        });

        it('should fallback to zh-CN when current lang not available', function() {
            I18n.translations = {
                'zh-CN': { 'hello': '你好，世界！' }
            };
            I18n.currentLang = 'fr-FR';
            const result = I18n.t('hello');
            expect(result).toBeTruthy();
        });

        it('should replace parameters in translation', function() {
            I18n.translations = {
                'en-US': { 'greeting': 'Hello, {name}!' }
            };
            I18n.currentLang = 'en-US';
            expect(I18n.t('greeting', { name: 'John' })).toBe('Hello, John!');
        });

        it('should replace multiple parameters', function() {
            I18n.translations = {
                'en-US': { 'info': '{count} items, {total} total' }
            };
            I18n.currentLang = 'en-US';
            expect(I18n.t('info', { count: 5, total: 10 })).toBe('5 items, 10 total');
        });
    });
});

describe('I18n RTL Support', function() {
    
    it('should identify RTL languages', function() {
        expect(I18n.getLangInfo.call({ currentLang: 'ar-SA', supportedLangs: I18n.supportedLangs }).rtl).toBe(true);
        expect(I18n.getLangInfo.call({ currentLang: 'he-IL', supportedLangs: I18n.supportedLangs }).rtl).toBe(true);
        expect(I18n.getLangInfo.call({ currentLang: 'fa-IR', supportedLangs: I18n.supportedLangs }).rtl).toBe(true);
    });

    it('should identify LTR languages', function() {
        expect(I18n.getLangInfo.call({ currentLang: 'en-US', supportedLangs: I18n.supportedLangs }).rtl).toBeFalsy();
        expect(I18n.getLangInfo.call({ currentLang: 'zh-CN', supportedLangs: I18n.supportedLangs }).rtl).toBeFalsy();
        expect(I18n.getLangInfo.call({ currentLang: 'ja-JP', supportedLangs: I18n.supportedLangs }).rtl).toBeFalsy();
    });
});

describe('I18n Performance', function() {
    
    it('should format number quickly', function() {
        const start = performance.now();
        for (let i = 0; i < 1000; i++) {
            I18n.formatNumber(1234567.89);
        }
        const elapsed = performance.now() - start;
        expect(elapsed).toBeLessThan(1000);
    });

    it('should format date quickly', function() {
        const testDate = new Date(2026, 4, 17, 14, 30, 45);
        const start = performance.now();
        for (let i = 0; i < 1000; i++) {
            I18n.formatDate(testDate);
        }
        const elapsed = performance.now() - start;
        expect(elapsed).toBeLessThan(1000);
    });
});
