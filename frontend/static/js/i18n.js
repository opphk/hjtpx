// 国际化支持 - 前端语言切换
const I18n = {
    currentLang: 'zh-CN',
    currentTimezone: 'Asia/Shanghai',
    translations: {},
    
    supportedLangs: [
        { code: 'zh-CN', name: '简体中文', flag: '🇨🇳', rtl: false, dateFormat: 'YYYY-MM-DD', currency: 'CNY' },
        { code: 'en-US', name: 'English', flag: '🇺🇸', rtl: false, dateFormat: 'MM/DD/YYYY', currency: 'USD' },
        { code: 'ja-JP', name: '日本語', flag: '🇯🇵', rtl: false, dateFormat: 'YYYY/MM/DD', currency: 'JPY' },
        { code: 'ko-KR', name: '한국어', flag: '🇰🇷', rtl: false, dateFormat: 'YYYY. MM. DD.', currency: 'KRW' },
        { code: 'fr-FR', name: 'Français', flag: '🇫🇷', rtl: false, dateFormat: 'DD/MM/YYYY', currency: 'EUR' },
        { code: 'de-DE', name: 'Deutsch', flag: '🇩🇪', rtl: false, dateFormat: 'DD.MM.YYYY', currency: 'EUR' },
        { code: 'es-ES', name: 'Español', flag: '🇪🇸', rtl: false, dateFormat: 'DD/MM/YYYY', currency: 'EUR' },
        { code: 'pt-BR', name: 'Português', flag: '🇧🇷', rtl: false, dateFormat: 'DD/MM/YYYY', currency: 'BRL' },
        { code: 'it-IT', name: 'Italiano', flag: '🇮🇹', rtl: false, dateFormat: 'DD/MM/YYYY', currency: 'EUR' },
        { code: 'ru-RU', name: 'Русский', flag: '🇷🇺', rtl: false, dateFormat: 'DD.MM.YYYY', currency: 'RUB' },
        { code: 'ar-SA', name: 'العربية', flag: '🇸🇦', rtl: true, dateFormat: 'DD/MM/YYYY', currency: 'SAR' },
        { code: 'fa-IR', name: 'فارسی', flag: '🇮🇷', rtl: true, dateFormat: 'DD/MM/YYYY', currency: 'IRR' },
        { code: 'he-IL', name: 'עברית', flag: '🇮🇱', rtl: true, dateFormat: 'DD/MM/YYYY', currency: 'ILS' },
        { code: 'ur-PK', name: 'اردو', flag: '🇵🇰', rtl: true, dateFormat: 'DD/MM/YYYY', currency: 'PKR' },
        { code: 'hi-IN', name: 'हिन्दी', flag: '🇮🇳', rtl: false, dateFormat: 'DD/MM/YYYY', currency: 'INR' },
        { code: 'vi-VN', name: 'Tiếng Việt', flag: '🇻🇳', rtl: false, dateFormat: 'DD/MM/YYYY', currency: 'VND' },
        { code: 'th-TH', name: 'ไทย', flag: '🇹🇭', rtl: false, dateFormat: 'DD/MM/YYYY', currency: 'THB' },
        { code: 'id-ID', name: 'Bahasa Indonesia', flag: '🇮🇩', rtl: false, dateFormat: 'DD/MM/YYYY', currency: 'IDR' },
        { code: 'tr-TR', name: 'Türkçe', flag: '🇹🇷', rtl: false, dateFormat: 'DD.MM.YYYY', currency: 'TRY' }
    ],
    
    supportedTimezones: [
        'Asia/Shanghai', 'America/New_York', 'America/Los_Angeles', 'Europe/London',
        'Europe/Paris', 'Europe/Berlin', 'Asia/Tokyo', 'Asia/Seoul', 'Australia/Sydney',
        'Asia/Dubai', 'Asia/Kolkata', 'Asia/Singapore', 'Asia/Hong_Kong', 'Asia/Bangkok',
        'Asia/Jakarta', 'Asia/Manila', 'Asia/Ho_Chi_Minh', 'Asia/Taipei', 'Europe/Madrid',
        'Europe/Rome', 'Europe/Moscow', 'Europe/Istanbul', 'America/Toronto', 'America/Vancouver',
        'America/Chicago', 'America/Denver', 'America/Mexico_City', 'America/Sao_Paulo',
        'Africa/Cairo', 'Africa/Johannesburg', 'Africa/Lagos', 'Africa/Nairobi',
        'Asia/Riyadh', 'Asia/Tehran', 'Asia/Karachi', 'Asia/Dhaka', 'Pacific/Auckland'
    ],
    
    numberFormats: {
        'zh-CN': { decimalSep: '.', thousandSep: ',', decimalDigits: 2 },
        'en-US': { decimalSep: '.', thousandSep: ',', decimalDigits: 2 },
        'ja-JP': { decimalSep: '.', thousandSep: ',', decimalDigits: 0 },
        'ko-KR': { decimalSep: '.', thousandSep: ',', decimalDigits: 0 },
        'fr-FR': { decimalSep: ',', thousandSep: '\u00A0', decimalDigits: 2 },
        'de-DE': { decimalSep: ',', thousandSep: '.', decimalDigits: 2 },
        'es-ES': { decimalSep: ',', thousandSep: '.', decimalDigits: 2 },
        'pt-BR': { decimalSep: ',', thousandSep: '.', decimalDigits: 2 },
        'it-IT': { decimalSep: ',', thousandSep: '.', decimalDigits: 2 },
        'ru-RU': { decimalSep: ',', thousandSep: ' ', decimalDigits: 2 },
        'ar-SA': { decimalSep: '٫', thousandSep: '٬', decimalDigits: 3 },
        'fa-IR': { decimalSep: '٫', thousandSep: '٬', decimalDigits: 0 },
        'he-IL': { decimalSep: '.', thousandSep: ',', decimalDigits: 2 },
        'ur-PK': { decimalSep: '.', thousandSep: ',', decimalDigits: 0 },
        'hi-IN': { decimalSep: '.', thousandSep: ',', decimalDigits: 2 },
        'vi-VN': { decimalSep: ',', thousandSep: '.', decimalDigits: 2 },
        'th-TH': { decimalSep: '.', thousandSep: ',', decimalDigits: 2 },
        'id-ID': { decimalSep: ',', thousandSep: '.', decimalDigits: 2 },
        'tr-TR': { decimalSep: ',', thousandSep: '.', decimalDigits: 2 }
    },
    
    currencyFormats: {
        'zh-CN': { symbol: '¥', position: 'before' },
        'en-US': { symbol: '$', position: 'before' },
        'ja-JP': { symbol: '¥', position: 'before' },
        'ko-KR': { symbol: '₩', position: 'before' },
        'fr-FR': { symbol: '€', position: 'after' },
        'de-DE': { symbol: '€', position: 'after' },
        'es-ES': { symbol: '€', position: 'after' },
        'pt-BR': { symbol: 'R$', position: 'before' },
        'it-IT': { symbol: '€', position: 'after' },
        'ru-RU': { symbol: '₽', position: 'after' },
        'ar-SA': { symbol: 'ر.س', position: 'after' },
        'fa-IR': { symbol: 'ریال', position: 'after' },
        'he-IL': { symbol: '₪', position: 'before' },
        'ur-PK': { symbol: '₨', position: 'before' },
        'hi-IN': { symbol: '₹', position: 'before' },
        'vi-VN': { symbol: '₫', position: 'after' },
        'th-TH': { symbol: '฿', position: 'before' },
        'id-ID': { symbol: 'Rp', position: 'before' },
        'tr-TR': { symbol: '₺', position: 'before' }
    },
    
    getBrowserLang: function() {
        const navLang = navigator.language || navigator.userLanguage;
        for (let lang of this.supportedLangs) {
            if (navLang.startsWith(lang.code.split('-')[0])) {
                return lang.code;
            }
        }
        return 'zh-CN';
    },
    
    getStoredLang: function() {
        return localStorage.getItem('preferredLang');
    },
    
    getStoredTimezone: function() {
        return localStorage.getItem('preferredTimezone');
    },
    
    getBrowserTimezone: function() {
        try {
            return Intl.DateTimeFormat().resolvedOptions().timeZone;
        } catch (e) {
            return 'Asia/Shanghai';
        }
    },
    
    setStoredLang: function(lang) {
        localStorage.setItem('preferredLang', lang);
    },
    
    setStoredTimezone: function(tz) {
        localStorage.setItem('preferredTimezone', tz);
    },
    
    init: async function() {
        this.currentLang = this.getStoredLang() || this.getBrowserLang();
        this.currentTimezone = this.getStoredTimezone() || this.getBrowserTimezone();
        await this.loadTranslations();
        this.renderLangSelector();
        this.renderTimezoneSelector();
        this.applyTranslations();
        this.applyRTL();
        document.documentElement.lang = this.currentLang;
    },
    
    loadTranslations: async function() {
        try {
            const zhCN = await fetch('/translations/zh-CN.json');
            this.translations['zh-CN'] = await zhCN.json();
            
            if (this.currentLang !== 'zh-CN') {
                try {
                    const target = await fetch(`/translations/${this.currentLang}.json`);
                    this.translations[this.currentLang] = await target.json();
                } catch (e) {
                    console.warn('Failed to load target language, using default');
                }
            }
        } catch (e) {
            console.error('Failed to load translations:', e);
        }
    },
    
    t: function(key, params = {}) {
        let text = this.translations[this.currentLang]?.[key] || 
                   this.translations['zh-CN']?.[key] || key;
        
        Object.keys(params).forEach(paramKey => {
            text = text.replace(`{${paramKey}}`, params[paramKey]);
        });
        
        return text;
    },
    
    applyTranslations: function() {
        document.querySelectorAll('[data-i18n]').forEach(el => {
            const key = el.getAttribute('data-i18n');
            el.textContent = this.t(key);
        });
        
        document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
            const key = el.getAttribute('data-i18n-placeholder');
            el.placeholder = this.t(key);
        });
        
        document.querySelectorAll('[data-i18n-title]').forEach(el => {
            const key = el.getAttribute('data-i18n-title');
            el.title = this.t(key);
        });
        
        const titleKey = document.documentElement.getAttribute('data-i18n-title');
        if (titleKey) {
            document.title = this.t(titleKey);
        }
    },
    
    renderLangSelector: function() {
        let container = document.getElementById('lang-selector-container');
        if (!container) {
            container = document.createElement('div');
            container.id = 'lang-selector-container';
            container.style.cssText = 'position: fixed; top: 20px; right: 20px; z-index: 9999;';
            document.body.appendChild(container);
        }
        
        container.innerHTML = '';
        
        const select = document.createElement('select');
        select.id = 'lang-selector';
        select.style.cssText = 'padding: 8px 12px; border: 1px solid #ddd; border-radius: 6px; background: white; font-size: 14px; cursor: pointer;';
        
        this.supportedLangs.forEach(lang => {
            const option = document.createElement('option');
            option.value = lang.code;
            option.textContent = `${lang.flag} ${lang.name}`;
            if (lang.code === this.currentLang) {
                option.selected = true;
            }
            select.appendChild(option);
        });
        
        select.addEventListener('change', (e) => {
            this.setLang(e.target.value);
        });
        
        container.appendChild(select);
    },
    
    setLang: async function(lang) {
        if (lang === this.currentLang) return;
        
        this.currentLang = lang;
        this.setStoredLang(lang);
        
        if (!this.translations[lang]) {
            try {
                const response = await fetch(`/translations/${lang}.json`);
                this.translations[lang] = await response.json();
            } catch (e) {
                console.error('Failed to load new language:', e);
            }
        }
        
        this.applyTranslations();
        document.documentElement.lang = lang;
        document.dispatchEvent(new CustomEvent('languageChange', { detail: { lang } }));
    },
    
    renderTimezoneSelector: function() {
        let container = document.getElementById('timezone-selector-container');
        if (!container) {
            container = document.createElement('div');
            container.id = 'timezone-selector-container';
            container.style.cssText = 'position: fixed; top: 20px; right: 20px; z-index: 9999;';
            document.body.appendChild(container);
        }
        
        container.innerHTML = '';
        
        const select = document.createElement('select');
        select.id = 'timezone-selector';
        select.style.cssText = 'padding: 8px 12px; border: 1px solid #ddd; border-radius: 6px; background: white; font-size: 14px; cursor: pointer;';
        
        this.supportedTimezones.forEach(tz => {
            const option = document.createElement('option');
            option.value = tz;
            option.textContent = tz;
            if (tz === this.currentTimezone) {
                option.selected = true;
            }
            select.appendChild(option);
        });
        
        select.addEventListener('change', (e) => {
            this.setTimezone(e.target.value);
        });
        
        container.appendChild(select);
    },
    
    setTimezone: function(tz) {
        if (tz === this.currentTimezone) return;
        
        this.currentTimezone = tz;
        this.setStoredTimezone(tz);
        
        document.dispatchEvent(new CustomEvent('timezoneChange', { detail: { timezone: tz } }));
    },
    
    applyRTL: function() {
        const langInfo = this.supportedLangs.find(l => l.code === this.currentLang);
        const isRTL = langInfo && langInfo.rtl;
        
        document.documentElement.dir = isRTL ? 'rtl' : 'ltr';
        document.documentElement.lang = this.currentLang;
        
        if (isRTL) {
            document.body.classList.add('rtl');
            document.body.classList.remove('ltr');
            this.applyRTLStyles();
        } else {
            document.body.classList.add('ltr');
            document.body.classList.remove('rtl');
            this.removeRTLStyles();
        }
    },
    
    applyRTLStyles: function() {
        let styleEl = document.getElementById('rtl-styles');
        if (!styleEl) {
            styleEl = document.createElement('style');
            styleEl.id = 'rtl-styles';
            document.head.appendChild(styleEl);
        }
        
        styleEl.textContent = `
            [dir="rtl"] .text-left { text-align: right !important; }
            [dir="rtl"] .text-right { text-align: left !important; }
            [dir="rtl"] .ml-auto { margin-left: 0 !important; margin-right: auto !important; }
            [dir="rtl"] .mr-auto { margin-right: 0 !important; margin-left: auto !important; }
            [dir="rtl"] .pl-3 { padding-left: 0 !important; padding-right: 1rem !important; }
            [dir="rtl"] .pr-3 { padding-right: 0 !important; padding-left: 1rem !important; }
            [dir="rtl"] .ml-3 { margin-left: 0 !important; margin-right: 1rem !important; }
            [dir="rtl"] .mr-3 { margin-right: 0 !important; margin-left: 1rem !important; }
            [dir="rtl"] .float-left { float: right !important; }
            [dir="rtl"] .float-right { float: left !important; }
            [dir="rtl"] .border-left { border-left: 0 !important; border-right: 1px solid #dee2e6; }
            [dir="rtl"] .border-right { border-right: 0 !important; border-left: 1px solid #dee2e6; }
            [dir="rtl"] .rounded-left { border-top-left-radius: 0 !important; border-bottom-left-radius: 0 !important; border-top-right-radius: 0.25rem !important; border-bottom-right-radius: 0.25rem !important; }
            [dir="rtl"] .rounded-right { border-top-right-radius: 0 !important; border-bottom-right-radius: 0 !important; border-top-left-radius: 0.25rem !important; border-bottom-left-radius: 0.25rem !important; }
            [dir="rtl"] input[type="text"] { text-align: right; }
            [dir="rtl"] input[type="email"] { text-align: right; }
            [dir="rtl"] input[type="number"] { text-align: right; }
            [dir="rtl"] input[type="password"] { text-align: right; }
            [dir="rtl"] textarea { text-align: right; }
        `;
    },
    
    removeRTLStyles: function() {
        const styleEl = document.getElementById('rtl-styles');
        if (styleEl) {
            styleEl.remove();
        }
    },
    
    formatNumber: function(value, decimalDigits = null) {
        const format = this.numberFormats[this.currentLang] || this.numberFormats['en-US'];
        const digits = decimalDigits !== null ? decimalDigits : format.decimalDigits;
        
        const parts = value.toFixed(digits).split('.');
        parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, format.thousandSep);
        
        if (digits === 0) {
            return parts[0];
        }
        return parts.join(format.decimalSep);
    },
    
    formatCurrency: function(value) {
        const currencyFormat = this.currencyFormats[this.currentLang] || this.currencyFormats['en-US'];
        const numberFormat = this.numberFormats[this.currentLang] || this.numberFormats['en-US'];
        
        const parts = value.toFixed(numberFormat.decimalDigits).split('.');
        parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, numberFormat.thousandSep);
        const formattedNumber = parts.join(numberFormat.decimalSep);
        
        if (currencyFormat.position === 'before') {
            return currencyFormat.symbol + formattedNumber;
        } else {
            return formattedNumber + ' ' + currencyFormat.symbol;
        }
    },
    
    formatPercent: function(value) {
        const percent = (value * 100).toFixed(2);
        return this.formatNumber(parseFloat(percent), 2) + '%';
    },
    
    formatDate: function(date, format = 'medium') {
        const langInfo = this.supportedLangs.find(l => l.code === this.currentLang);
        const dateFormat = langInfo ? langInfo.dateFormat : 'MM/DD/YYYY';
        
        const d = new Date(date);
        const day = String(d.getDate()).padStart(2, '0');
        const month = String(d.getMonth() + 1).padStart(2, '0');
        const year = d.getFullYear();
        
        return dateFormat
            .replace('DD', day)
            .replace('MM', month)
            .replace('YYYY', year);
    },
    
    formatTime: function(date, includeSeconds = false) {
        const d = new Date(date);
        const hours = String(d.getHours()).padStart(2, '0');
        const minutes = String(d.getMinutes()).padStart(2, '0');
        const seconds = String(d.getSeconds()).padStart(2, '0');
        
        if (includeSeconds) {
            return `${hours}:${minutes}:${seconds}`;
        }
        return `${hours}:${minutes}`;
    },
    
    formatDateTime: function(date, includeSeconds = false) {
        return this.formatDate(date) + ' ' + this.formatTime(date, includeSeconds);
    },
    
    formatRelativeTime: function(date) {
        const now = new Date();
        const target = new Date(date);
        const diffMs = now - target;
        const diffSecs = Math.floor(diffMs / 1000);
        const diffMins = Math.floor(diffSecs / 60);
        const diffHours = Math.floor(diffMins / 60);
        const diffDays = Math.floor(diffHours / 24);
        const diffWeeks = Math.floor(diffDays / 7);
        const diffMonths = Math.floor(diffDays / 30);
        const diffYears = Math.floor(diffDays / 365);
        
        const translations = this.translations[this.currentLang] || this.translations['zh-CN'];
        
        if (diffSecs < 60) {
            return translations.just_now || 'just now';
        } else if (diffMins < 60) {
            return translations.minutes_ago?.replace('{0}', diffMins) || `${diffMins}m ago`;
        } else if (diffHours < 24) {
            return translations.hours_ago?.replace('{0}', diffHours) || `${diffHours}h ago`;
        } else if (diffDays < 7) {
            return translations.days_ago?.replace('{0}', diffDays) || `${diffDays}d ago`;
        } else if (diffWeeks < 4) {
            return translations.weeks_ago?.replace('{0}', diffWeeks) || `${diffWeeks}w ago`;
        } else if (diffMonths < 12) {
            return translations.months_ago?.replace('{0}', diffMonths) || `${diffMonths}mo ago`;
        } else {
            return translations.years_ago?.replace('{0}', diffYears) || `${diffYears}y ago`;
        }
    },
    
    getLangInfo: function(lang) {
        return this.supportedLangs.find(l => l.code === lang) || this.supportedLangs[0];
    },
    
    isRTL: function(lang) {
        const langInfo = this.getLangInfo(lang);
        return langInfo.rtl || false;
    }
};

document.addEventListener('DOMContentLoaded', function() {
    I18n.init();
});
