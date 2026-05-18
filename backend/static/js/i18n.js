// 国际化支持 - 前端语言切换
const I18n = {
    currentLang: 'zh-CN',
    translations: {},
    
    // 支持的语言列表
    supportedLangs: [
        { code: 'zh-CN', name: '简体中文', flag: '🇨🇳' },
        { code: 'en-US', name: 'English', flag: '🇺🇸' },
        { code: 'ja-JP', name: '日本語', flag: '🇯🇵' },
        { code: 'ko-KR', name: '한국어', flag: '🇰🇷' },
        { code: 'fr-FR', name: 'Français', flag: '🇫🇷' },
        { code: 'de-DE', name: 'Deutsch', flag: '🇩🇪' },
        { code: 'es-ES', name: 'Español', flag: '🇪🇸' },
        { code: 'pt-BR', name: 'Português', flag: '🇧🇷' },
        { code: 'it-IT', name: 'Italiano', flag: '🇮🇹' },
        { code: 'ru-RU', name: 'Русский', flag: '🇷🇺' },
        { code: 'ar-SA', name: 'العربية', flag: '🇸🇦' },
        { code: 'fa-IR', name: 'فارسی', flag: '🇮🇷' },
        { code: 'he-IL', name: 'עברית', flag: '🇮🇱' },
        { code: 'ur-PK', name: 'اردو', flag: '🇵🇰' }
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
        
        // 触发自定义事件
        document.dispatchEvent(new CustomEvent('languageChange', { detail: { lang } }));
    }
};

// 自动初始化
document.addEventListener('DOMContentLoaded', function() {
    I18n.init();
});
