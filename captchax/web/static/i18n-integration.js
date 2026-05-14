(function() {
    'use strict';

    var CaptchaXI18n = {
        initialized: false,

        init: function(options) {
            options = options || {};
            options.defaultLocale = options.defaultLocale || this.detectAcceptLanguage();
            options.storageKey = options.storageKey || 'captchax_locale';

            return I18n.init(options).then(function() {
                CaptchaXI18n.initialized = true;
                CaptchaXI18n.applyTranslations();
                return CaptchaXI18n;
            });
        },

        detectAcceptLanguage: function() {
            var locales = {
                'zh': 'zh-CN',
                'zh-CN': 'zh-CN',
                'zh-TW': 'zh-TW',
                'zh-HK': 'zh-TW',
                'ja': 'ja',
                'ko': 'ko',
                'ar': 'ar',
                'en': 'en'
            };

            var nav = navigator || {};
            var languages = nav.languages || [nav.language || nav.userLanguage || ''];
            var acceptLanguage = nav.acceptLanguage || '';

            var allLocales = languages.slice();
            if (acceptLanguage) {
                acceptLanguage.split(',').forEach(function(lang) {
                    var parts = lang.split(';')[0].trim();
                    if (parts) allLocales.push(parts);
                });
            }

            for (var i = 0; i < allLocales.length; i++) {
                var locale = allLocales[i].toLowerCase().replace('_', '-');
                if (locales[locale]) {
                    return locales[locale];
                }
                var prefix = locale.split('-')[0];
                if (locales[prefix]) {
                    return locales[prefix];
                }
            }

            return 'en';
        },

        t: function(key, params) {
            return I18n.t(key, params);
        },

        setLocale: function(locale) {
            return I18n.setLocale(locale);
        },

        getLocale: function() {
            return I18n.getCurrentLocale();
        },

        applyTranslations: function() {
            var elements = document.querySelectorAll('[data-i18n]');
            elements.forEach(function(el) {
                var key = el.getAttribute('data-i18n');
                var text = I18n.t(key);
                if (text !== key) {
                    el.textContent = text;
                }
            });

            var attrElements = document.querySelectorAll('[data-i18n-attr]');
            attrElements.forEach(function(el) {
                var attrStr = el.getAttribute('data-i18n-attr');
                try {
                    var attrs = JSON.parse(attrStr);
                    for (var attr in attrs) {
                        if (attrs.hasOwnProperty(attr)) {
                            var text = I18n.t(attrs[attr]);
                            if (text !== attrs[attr]) {
                                el.setAttribute(attr, text);
                            }
                        }
                    }
                } catch (e) {}
            });

            this.updateCaptchaTemplates();
        },

        updateCaptchaTemplates: function() {
            var templates = document.querySelectorAll('template[id^="captchax-"]');
            templates.forEach(function(template) {
                var clone = template.content.cloneNode(true);
                var elements = clone.querySelectorAll('[data-i18n]');
                elements.forEach(function(el) {
                    var key = el.getAttribute('data-i18n');
                    var text = I18n.t(key);
                    if (text !== key) {
                        el.textContent = text;
                    }
                });
            });
        },

        createLanguageButton: function(containerId, options) {
            options = options || {};
            LanguageSwitcher.init({
                position: options.position || 'top-right',
                showFlag: options.showFlag !== false,
                compact: options.compact !== false,
                onChange: options.onChange || null
            });
        },

        onLocaleChange: function(callback) {
            I18n.onLocaleChange(callback);
        }
    };

    window.CaptchaXI18n = CaptchaXI18n;

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = CaptchaXI18n;
    }

})();
