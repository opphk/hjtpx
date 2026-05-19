// 国际化支持 - 前端语言切换
const I18n = {
    currentLang: 'zh-CN',
    translations: {},
    
    // 支持的语言列表
    supportedLangs: [
        { code: 'zh-CN', name: '简体中文', nativeName: '简体中文', flag: '🇨🇳', isRTL: false, dateFormat: 'YYYY-MM-DD', numberFormat: '1,234.56', currency: '¥' },
        { code: 'en-US', name: 'English', nativeName: 'English', flag: '🇺🇸', isRTL: false, dateFormat: 'MM/DD/YYYY', numberFormat: '1,234.56', currency: '$' },
        { code: 'ja-JP', name: '日本語', nativeName: '日本語', flag: '🇯🇵', isRTL: false, dateFormat: 'YYYY/MM/DD', numberFormat: '1,234.56', currency: '¥' },
        { code: 'ko-KR', name: '한국어', nativeName: '한국어', flag: '🇰🇷', isRTL: false, dateFormat: 'YYYY. MM. DD', numberFormat: '1,234.56', currency: '₩' },
        { code: 'fr-FR', name: 'Français', nativeName: 'Français', flag: '🇫🇷', isRTL: false, dateFormat: 'DD/MM/YYYY', numberFormat: '1 234,56', currency: '€' },
        { code: 'de-DE', name: 'Deutsch', nativeName: 'Deutsch', flag: '🇩🇪', isRTL: false, dateFormat: 'DD.MM.YYYY', numberFormat: '1.234,56', currency: '€' },
        { code: 'es-ES', name: 'Español', nativeName: 'Español', flag: '🇪🇸', isRTL: false, dateFormat: 'DD/MM/YYYY', numberFormat: '1.234,56', currency: '€' },
        { code: 'pt-BR', name: 'Português', nativeName: 'Português', flag: '🇧🇷', isRTL: false, dateFormat: 'DD/MM/YYYY', numberFormat: '1.234,56', currency: 'R$' },
        { code: 'it-IT', name: 'Italiano', nativeName: 'Italiano', flag: '🇮🇹', isRTL: false, dateFormat: 'DD/MM/YYYY', numberFormat: '1.234,56', currency: '€' },
        { code: 'ru-RU', name: 'Русский', nativeName: 'Русский', flag: '🇷🇺', isRTL: false, dateFormat: 'DD.MM.YYYY', numberFormat: '1 234,56', currency: '₽' },
        { code: 'ar-SA', name: 'العربية', nativeName: 'العربية', flag: '🇸🇦', isRTL: true, dateFormat: 'DD/MM/YYYY', numberFormat: '١٬٢٣٤٫٥٦', currency: 'ر.س' },
        { code: 'fa-IR', name: 'فارسی', nativeName: 'فارسی', flag: '🇮🇷', isRTL: true, dateFormat: 'DD/MM/YYYY', numberFormat: '۱٬۲۳۴٫۵۶', currency: 'ریال' },
        { code: 'he-IL', name: 'עברית', nativeName: 'עברית', flag: '🇮🇱', isRTL: true, dateFormat: 'DD/MM/YYYY', numberFormat: '1,234.56', currency: '₪' },
        { code: 'ur-PK', name: 'اردو', nativeName: 'اردو', flag: '🇵🇰', isRTL: true, dateFormat: 'DD/MM/YYYY', numberFormat: '1,234.56', currency: '₨' },
        { code: 'hi-IN', name: 'हिन्दी', nativeName: 'हिन्दी', flag: '🇮🇳', isRTL: false, dateFormat: 'DD/MM/YYYY', numberFormat: '1,23,456.78', currency: '₹' },
        { code: 'vi-VN', name: 'Tiếng Việt', nativeName: 'Tiếng Việt', flag: '🇻🇳', isRTL: false, dateFormat: 'DD/MM/YYYY', numberFormat: '1.234,56', currency: '₫' },
        { code: 'th-TH', name: 'ไทย', nativeName: 'ไทย', flag: '🇹🇭', isRTL: false, dateFormat: 'DD/MM/YYYY', numberFormat: '1,234.56', currency: '฿' },
        { code: 'id-ID', name: 'Bahasa Indonesia', nativeName: 'Bahasa Indonesia', flag: '🇮🇩', isRTL: false, dateFormat: 'DD/MM/YYYY', numberFormat: '1.234,56', currency: 'Rp' },
        { code: 'tr-TR', name: 'Türkçe', nativeName: 'Türkçe', flag: '🇹🇷', isRTL: false, dateFormat: 'DD.MM.YYYY', numberFormat: '1.234,56', currency: '₺' }
    ],
    
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
    
    // 从本地存储获取语言
    getStoredLang: function() {
        return localStorage.getItem('preferredLang');
    },
    
    // 设置语言到本地存储
    setStoredLang: function(lang) {
        localStorage.setItem('preferredLang', lang);
    },
    
    // 初始化
    init: async function() {
        // 获取语言
        this.currentLang = this.getStoredLang() || this.getBrowserLang();
        
        // 加载翻译
        await this.loadTranslations();
        
        // 渲染语言选择器
        this.renderLangSelector();
        
        // 应用翻译
        this.applyTranslations();
        
        // 设置HTML lang属性
        document.documentElement.lang = this.currentLang;
    },
    
    // 加载翻译文件
    loadTranslations: async function() {
        try {
            // 先加载中文作为默认
            const zhCN = await fetch('/translations/zh-CN.json');
            this.translations['zh-CN'] = await zhCN.json();
            
            // 加载其他需要的语言
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
        let container = document.getElementById('lang-selector-container');
        if (!container) {
            container = document.createElement('div');
            container.id = 'lang-selector-container';
            container.style.cssText = 'position: fixed; top: 20px; right: 20px; z-index: 9999;';
            
            // 添加到页面
            document.body.appendChild(container);
        }
        
        // 清空容器
        container.innerHTML = '';
        
        // 创建选择器
        const select = document.createElement('select');
        select.id = 'lang-selector';
        select.style.cssText = 'padding: 8px 12px; border: 1px solid #ddd; border-radius: 6px; background: white; font-size: 14px; cursor: pointer;';
        
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
                const response = await fetch(`/translations/${lang}.json`);
                this.translations[lang] = await response.json();
            } catch (e) {
                console.error('Failed to load new language:', e);
            }
        }
        
        // 应用翻译
        this.applyTranslations();
        
        // 更新HTML lang属性
        document.documentElement.lang = lang;
        
        // 应用RTL布局
        this.applyRTLSupport(lang);
        
        // 触发自定义事件
        document.dispatchEvent(new CustomEvent('languageChange', { detail: { lang } }));
    },
    
    // 应用RTL支持
    applyRTLSupport: function(lang) {
        const langInfo = this.supportedLangs.find(l => l.code === lang);
        const isRTL = langInfo && langInfo.isRTL;
        
        if (isRTL) {
            document.documentElement.setAttribute('dir', 'rtl');
            document.documentElement.classList.add('rtl');
            document.documentElement.classList.remove('ltr');
            this.applyRTLStyles();
        } else {
            document.documentElement.setAttribute('dir', 'ltr');
            document.documentElement.classList.add('ltr');
            document.documentElement.classList.remove('rtl');
        }
    },
    
    // 应用RTL样式
    applyRTLStyles: function() {
        // 添加RTL特定样式
        let rtlStyle = document.getElementById('rtl-styles');
        if (!rtlStyle) {
            rtlStyle = document.createElement('style');
            rtlStyle.id = 'rtl-styles';
            document.head.appendChild(rtlStyle);
        }
        
        rtlStyle.textContent = `
            [dir="rtl"] .text-left { text-align: right; }
            [dir="rtl"] .text-right { text-align: left; }
            [dir="rtl"] .ml-auto { margin-left: 0; margin-right: auto; }
            [dir="rtl"] .mr-auto { margin-right: 0; margin-left: auto; }
            [dir="rtl"] .pl-3 { padding-left: 0; padding-right: 1rem; }
            [dir="rtl"] .pr-3 { padding-right: 0; padding-left: 1rem; }
            [dir="rtl"] .ml-3 { margin-left: 0; margin-right: 1rem; }
            [dir="rtl"] .mr-3 { margin-right: 0; margin-left: 1rem; }
            [dir="rtl"] .float-left { float: right; }
            [dir="rtl"] .float-right { float: left; }
            [dir="rtl"] .border-left { border-left: none; border-right: 1px solid; }
            [dir="rtl"] .border-right { border-right: none; border-left: 1px solid; }
            [dir="rtl"] .captcha-container { direction: rtl; }
            [dir="rtl"] .captcha-slider-container { direction: rtl; }
            [dir="rtl"] input { text-align: right; }
            [dir="rtl"] input[type="text"], [dir="rtl"] input[type="email"], [dir="rtl"] input[type="password"] {
                text-align: ${this.getComputedTextAlign()};
            }
        `;
    },
    
    // 获取计算后的文本对齐方式
    getComputedTextAlign: function() {
        const isRTL = document.documentElement.classList.contains('rtl');
        return isRTL ? 'right' : 'left';
    },
    
    // 格式化日期
    formatDate: function(date, format) {
        const langInfo = this.supportedLangs.find(l => l.code === this.currentLang);
        const dateFormat = format || (langInfo && langInfo.dateFormat) || 'YYYY-MM-DD';
        
        const d = new Date(date);
        const year = d.getFullYear();
        const month = String(d.getMonth() + 1).padStart(2, '0');
        const day = String(d.getDate()).padStart(2, '0');
        
        return dateFormat
            .replace('YYYY', year)
            .replace('MM', month)
            .replace('DD', day);
    },
    
    // 格式化数字
    formatNumber: function(num) {
        const langInfo = this.supportedLangs.find(l => l.code === this.currentLang);
        const numberFormat = langInfo && langInfo.numberFormat;
        
        if (!numberFormat) {
            return num.toLocaleString();
        }
        
        // 简单的数字格式化
        const parts = num.toString().split('.');
        parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ',');
        return parts.join('.');
    },
    
    // 格式化货币
    formatCurrency: function(amount, currency) {
        const langInfo = this.supportedLangs.find(l => l.code === this.currentLang);
        const currencySymbol = currency || (langInfo && langInfo.currency) || '$';
        
        const formatted = this.formatNumber(amount);
        return `${currencySymbol}${formatted}`;
    },
    
    // 获取语言信息
    getLangInfo: function(lang) {
        return this.supportedLangs.find(l => l.code === lang) || null;
    },
    
    // 检查是否为RTL语言
    isRTL: function(lang) {
        const langInfo = this.getLangInfo(lang || this.currentLang);
        return langInfo && langInfo.isRTL;
    },
    
    // 获取支持的语言数量
    getSupportedLanguagesCount: function() {
        return this.supportedLangs.length;
    },
    
    // 获取所有语言代码
    getAllLanguageCodes: function() {
        return this.supportedLangs.map(l => l.code);
    },
    
    // 搜索语言
    searchLanguages: function(query) {
        const lowerQuery = query.toLowerCase();
        return this.supportedLangs.filter(lang => 
            lang.name.toLowerCase().includes(lowerQuery) ||
            lang.nativeName.toLowerCase().includes(lowerQuery) ||
            lang.code.toLowerCase().includes(lowerQuery)
        );
    }
};

// 自动初始化
document.addEventListener('DOMContentLoaded', function() {
    I18n.init();
});
