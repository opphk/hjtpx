// 国际化支持 - 管理后台语言切换
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
        { code: 'tr-TR', name: 'Türkçe', flag: '🇹🇷', rtl: false, dateFormat: 'DD.MM.YYYY', currency: 'TRY' },
        { code: 'pl-PL', name: 'Polski', flag: '🇵🇱', rtl: false, dateFormat: 'DD.MM.YYYY', currency: 'PLN' },
        { code: 'nl-NL', name: 'Nederlands', flag: '🇳🇱', rtl: false, dateFormat: 'DD-MM-YYYY', currency: 'EUR' },
        { code: 'sv-SE', name: 'Svenska', flag: '🇸🇪', rtl: false, dateFormat: 'YYYY-MM-DD', currency: 'SEK' },
        { code: 'da-DK', name: 'Dansk', flag: '🇩🇰', rtl: false, dateFormat: 'DD-MM-YYYY', currency: 'DKK' },
        { code: 'nb-NO', name: 'Norsk', flag: '🇳🇴', rtl: false, dateFormat: 'DD.MM.YYYY', currency: 'NOK' },
        { code: 'fi-FI', name: 'Suomi', flag: '🇫🇮', rtl: false, dateFormat: 'DD.MM.YYYY', currency: 'EUR' },
        { code: 'cs-CZ', name: 'Čeština', flag: '🇨🇿', rtl: false, dateFormat: 'DD.MM.YYYY', currency: 'CZK' },
        { code: 'hu-HU', name: 'Magyar', flag: '🇭🇺', rtl: false, dateFormat: 'YYYY. MM. DD.', currency: 'HUF' },
        { code: 'ro-RO', name: 'Română', flag: '🇷🇴', rtl: false, dateFormat: 'DD.MM.YYYY', currency: 'RON' },
        { code: 'bg-BG', name: 'Български', flag: '🇧🇬', rtl: false, dateFormat: 'DD.MM.YYYY', currency: 'BGN' }
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
        'tr-TR': { decimalSep: ',', thousandSep: '.', decimalDigits: 2 },
        'pl-PL': { decimalSep: ',', thousandSep: ' ', decimalDigits: 2 },
        'nl-NL': { decimalSep: ',', thousandSep: '.', decimalDigits: 2 },
        'sv-SE': { decimalSep: ',', thousandSep: ' ', decimalDigits: 2 },
        'da-DK': { decimalSep: ',', thousandSep: '.', decimalDigits: 2 },
        'nb-NO': { decimalSep: ',', thousandSep: ' ', decimalDigits: 2 },
        'fi-FI': { decimalSep: ',', thousandSep: ' ', decimalDigits: 2 },
        'cs-CZ': { decimalSep: ',', thousandSep: ' ', decimalDigits: 2 },
        'hu-HU': { decimalSep: ',', thousandSep: ' ', decimalDigits: 2 },
        'ro-RO': { decimalSep: ',', thousandSep: '.', decimalDigits: 2 },
        'bg-BG': { decimalSep: ',', thousandSep: ' ', decimalDigits: 2 }
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
        'tr-TR': { symbol: '₺', position: 'before' },
        'pl-PL': { symbol: 'zł', position: 'after' },
        'nl-NL': { symbol: '€', position: 'before' },
        'sv-SE': { symbol: 'kr', position: 'after' },
        'da-DK': { symbol: 'kr', position: 'after' },
        'nb-NO': { symbol: 'kr', position: 'after' },
        'fi-FI': { symbol: '€', position: 'after' },
        'cs-CZ': { symbol: 'Kč', position: 'after' },
        'hu-HU': { symbol: 'Ft', position: 'after' },
        'ro-RO': { symbol: 'lei', position: 'after' },
        'bg-BG': { symbol: 'лв', position: 'after' }
    },
    
    // 从浏览器获取语言
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
        return localStorage.getItem('adminPreferredLang');
    },
    
    getStoredTimezone: function() {
        return localStorage.getItem('adminPreferredTimezone');
    },
    
    getBrowserTimezone: function() {
        try {
            return Intl.DateTimeFormat().resolvedOptions().timeZone;
        } catch (e) {
            return 'Asia/Shanghai';
        }
    },
    
    setStoredLang: function(lang) {
        localStorage.setItem('adminPreferredLang', lang);
    },
    
    setStoredTimezone: function(tz) {
        localStorage.setItem('adminPreferredTimezone', tz);
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
    
    // 加载翻译文件
    loadTranslations: async function() {
        try {
            // 先加载中文作为默认
            const zhCN = await fetch('/admin/translations/zh-CN.json');
            this.translations['zh-CN'] = await zhCN.json();
            
            // 加载其他需要的语言
            if (this.currentLang !== 'zh-CN') {
                try {
                    const target = await fetch(`/admin/translations/${this.currentLang}.json`);
                    this.translations[this.currentLang] = await target.json();
                } catch (e) {
                    console.warn('Failed to load target language, using default');
                }
            }
        } catch (e) {
            console.error('Failed to load translations:', e);
        }
    },
    
    // 翻译函数
    t: function(key, params = {}) {
        let text = this.translations[this.currentLang]?.[key] || 
                   this.translations['zh-CN']?.[key] || key;
        
        // 替换参数
        Object.keys(params).forEach(paramKey => {
            text = text.replace(`{${paramKey}}`, params[paramKey]);
        });
        
        return text;
    },
    
    // 应用翻译到DOM
    applyTranslations: function() {
        // 查找所有带有data-i18n属性的元素
        document.querySelectorAll('[data-i18n]').forEach(el => {
            const key = el.getAttribute('data-i18n');
            el.textContent = this.t(key);
        });
        
        // 查找所有带有data-i18n-placeholder属性的元素
        document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
            const key = el.getAttribute('data-i18n-placeholder');
            el.placeholder = this.t(key);
        });
        
        // 查找所有带有data-i18n-title属性的元素
        document.querySelectorAll('[data-i18n-title]').forEach(el => {
            const key = el.getAttribute('data-i18n-title');
            el.title = this.t(key);
        });
        
        // 更新页面标题
        const titleKey = document.documentElement.getAttribute('data-i18n-title');
        if (titleKey) {
            document.title = this.t(titleKey);
        }
    },
    
    // 渲染语言选择器
    renderLangSelector: function() {
        // 查找或创建语言选择器容器
        let container = document.getElementById('admin-lang-selector-container');
        if (!container) {
            container = document.createElement('div');
            container.id = 'admin-lang-selector-container';
            
            // 添加到页面顶部导航区域
            const headerRight = document.querySelector('.top-header-right');
            if (headerRight) {
                headerRight.insertBefore(container, headerRight.firstChild);
            } else {
                // 如果找不到，添加到body
                container.style.cssText = 'position: fixed; top: 20px; right: 20px; z-index: 9999;';
                document.body.appendChild(container);
            }
        }
        
        // 清空容器
        container.innerHTML = '';
        
        // 创建选择器
        const select = document.createElement('select');
        select.id = 'admin-lang-selector';
        select.style.cssText = 'padding: 4px 8px; border: 1px solid #ddd; border-radius: 4px; background: white; font-size: 13px; cursor: pointer; margin-right: 10px;';
        
        // 添加选项
        this.supportedLangs.forEach(lang => {
            const option = document.createElement('option');
            option.value = lang.code;
            option.textContent = `${lang.flag} ${lang.name}`;
            if (lang.code === this.currentLang) {
                option.selected = true;
            }
            select.appendChild(option);
        });
        
        // 添加事件监听
        select.addEventListener('change', (e) => {
            this.setLang(e.target.value);
        });
        
        container.appendChild(select);
    },
    
    // 设置语言
    setLang: async function(lang) {
        if (lang === this.currentLang) return;
        
        this.currentLang = lang;
        this.setStoredLang(lang);
        
        // 加载新语言（如果尚未加载）
        if (!this.translations[lang]) {
            try {
                const response = await fetch(`/admin/translations/${lang}.json`);
                this.translations[lang] = await response.json();
            } catch (e) {
                console.error('Failed to load new language:', e);
            }
        }
        
        // 应用翻译
        this.applyTranslations();
        
        // 更新HTML lang属性
        document.documentElement.lang = lang;
        
        // 触发自定义事件
        document.dispatchEvent(new CustomEvent('adminLanguageChange', { detail: { lang } }));
    },
    
    // 渲染时区选择器
    renderTimezoneSelector: function() {
        let container = document.getElementById('admin-timezone-selector-container');
        if (!container) {
            container = document.createElement('div');
            container.id = 'admin-timezone-selector-container';
            
            const headerRight = document.querySelector('.top-header-right');
            if (headerRight) {
                headerRight.insertBefore(container, headerRight.firstChild);
            } else {
                container.style.cssText = 'position: fixed; top: 20px; right: 120px; z-index: 9999;';
                document.body.appendChild(container);
            }
        }
        
        container.innerHTML = '';
        
        const select = document.createElement('select');
        select.id = 'admin-timezone-selector';
        select.style.cssText = 'padding: 4px 8px; border: 1px solid #ddd; border-radius: 4px; background: white; font-size: 13px; cursor: pointer; margin-right: 10px;';
        
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
    
    // 设置时区
    setTimezone: function(tz) {
        if (tz === this.currentTimezone) return;
        
        this.currentTimezone = tz;
        this.setStoredTimezone(tz);
        
        document.dispatchEvent(new CustomEvent('adminTimezoneChange', { detail: { timezone: tz } }));
    },
    
    // 应用RTL
    applyRTL: function() {
        const langInfo = this.supportedLangs.find(l => l.code === this.currentLang);
        const isRTL = langInfo && langInfo.rtl;
        
        document.documentElement.dir = isRTL ? 'rtl' : 'ltr';
        
        if (isRTL) {
            document.body.classList.add('rtl');
            document.body.classList.remove('ltr');
        } else {
            document.body.classList.add('ltr');
            document.body.classList.remove('rtl');
        }
    },
    
    // 格式化数字
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
    
    // 格式化货币
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
    
    // 格式化百分比
    formatPercent: function(value) {
        const percent = (value * 100).toFixed(2);
        return this.formatNumber(parseFloat(percent), 2) + '%';
    },
    
    // 格式化日期
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
    
    // 格式化时间
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
    
    // 格式化日期时间
    formatDateTime: function(date, includeSeconds = false) {
        return this.formatDate(date) + ' ' + this.formatTime(date, includeSeconds);
    },
    
    // 相对时间（刚刚、几分钟前等）
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
            return translations.just_now || '刚刚';
        } else if (diffMins < 60) {
            return translations.minutes_ago?.replace('{0}', diffMins) || `${diffMins}分钟前`;
        } else if (diffHours < 24) {
            return translations.hours_ago?.replace('{0}', diffHours) || `${diffHours}小时前`;
        } else if (diffDays < 7) {
            return translations.days_ago?.replace('{0}', diffDays) || `${diffDays}天前`;
        } else if (diffWeeks < 4) {
            return translations.weeks_ago?.replace('{0}', diffWeeks) || `${diffWeeks}周前`;
        } else if (diffMonths < 12) {
            return translations.months_ago?.replace('{0}', diffMonths) || `${diffMonths}月前`;
        } else {
            return translations.years_ago?.replace('{0}', diffYears) || `${diffYears}年前`;
        }
    },
    
    // 获取语言信息
    getLangInfo: function(lang) {
        return this.supportedLangs.find(l => l.code === lang) || this.supportedLangs[0];
    },
    
    // 检查是否为RTL语言
    isRTL: function(lang) {
        const langInfo = this.getLangInfo(lang);
        return langInfo.rtl || false;
    }
};

// 自动初始化
document.addEventListener('DOMContentLoaded', function() {
    I18n.init();
});
