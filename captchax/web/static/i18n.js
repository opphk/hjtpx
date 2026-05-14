(function(global) {
    'use strict';

    var I18n = {
        currentLocale: 'en',
        defaultLocale: 'en',
        supportedLocales: ['zh-CN', 'zh-TW', 'en', 'ja', 'ko', 'ar'],
        translations: {},
        listeners: [],

        localeMap: {
            'zh': 'zh-CN',
            'zh-CN': 'zh-CN',
            'zh-SG': 'zh-CN',
            'zh-TW': 'zh-TW',
            'zh-HK': 'zh-TW',
            'ja': 'ja',
            'ko': 'ko',
            'ko-KR': 'ko',
            'ar': 'ar',
            'ar-SA': 'ar',
            'ar-AE': 'ar',
            'en': 'en',
            'en-US': 'en',
            'en-GB': 'en'
        },

        async init(options) {
            options = options || {};
            this.defaultLocale = options.defaultLocale || 'en';
            this.localeParam = options.localeParam || 'lang';
            this.storageKey = options.storageKey || 'captchax_locale';

            await this.loadAllTranslations();
            var detectedLocale = this.detectLocale();
            await this.setLocale(detectedLocale, options.skipPersist);

            return this;
        },

        async loadAllTranslations() {
            var self = this;
            var promises = this.supportedLocales.map(function(locale) {
                return self.loadTranslation(locale);
            });
            await Promise.all(promises);
        },

        async loadTranslation(locale) {
            try {
                var response = await fetch('/i18n/' + locale + '.json');
                if (!response.ok) {
                    throw new Error('Failed to load translation: ' + locale);
                }
                this.translations[locale] = await response.json();
            } catch (error) {
                console.warn('I18n: Could not load translation for', locale, error);
                this.translations[locale] = null;
            }
        },

        detectLocale() {
            var stored = this.getStoredLocale();
            if (stored && this.isSupported(stored)) {
                return stored;
            }

            var browserLocale = this.getBrowserLocale();
            if (browserLocale && this.isSupported(browserLocale)) {
                return browserLocale;
            }

            return this.defaultLocale;
        },

        getBrowserLocale() {
            var nav = typeof navigator !== 'undefined' ? navigator : {};
            var languages = nav.languages || [nav.language || nav.userLanguage || ''];
            var acceptLanguage = nav.acceptLanguage || '';

            var allLocales = languages.concat(acceptLanguage.split(',').map(function(l) {
                return l.split(';')[0].trim();
            }));

            for (var i = 0; i < allLocales.length; i++) {
                var locale = allLocales[i].toLowerCase().replace('_', '-');
                var mapped = this.localeMap[locale];
                if (mapped) {
                    return mapped;
                }

                var prefix = locale.split('-')[0];
                for (var j = 0; j < this.supportedLocales.length; j++) {
                    var supported = this.supportedLocales[j].toLowerCase();
                    if (supported === prefix || supported.startsWith(prefix + '-')) {
                        return this.supportedLocales[j];
                    }
                }
            }

            return null;
        },

        getStoredLocale() {
            try {
                var stored = localStorage.getItem(this.storageKey);
                if (stored && this.isSupported(stored)) {
                    return stored;
                }
                var metaLocale = document.querySelector('meta[name="language"]');
                if (metaLocale && this.isSupported(metaLocale.content)) {
                    return metaLocale.content;
                }
                var htmlLang = document.documentElement.getAttribute('lang');
                if (htmlLang && this.isSupported(htmlLang)) {
                    return htmlLang;
                }
            } catch (e) {}
            return null;
        },

        isSupported(locale) {
            return this.supportedLocales.indexOf(locale) !== -1;
        },

        async setLocale(locale, skipPersist) {
            if (!this.isSupported(locale)) {
                console.warn('I18n: Unsupported locale:', locale);
                return false;
            }

            if (!this.translations[locale]) {
                await this.loadTranslation(locale);
                if (!this.translations[locale]) {
                    console.warn('I18n: Could not load translations for:', locale);
                    return false;
                }
            }

            var oldLocale = this.currentLocale;
            this.currentLocale = locale;

            if (!skipPersist) {
                try {
                    localStorage.setItem(this.storageKey, locale);
                } catch (e) {}
            }

            this.applyDirection(locale);
            this.updateDOM();
            this.notifyListeners(oldLocale, locale);

            return true;
        },

        applyDirection(locale) {
            var translation = this.translations[locale];
            var dir = translation && translation.dir ? translation.dir : 'ltr';

            if (document.documentElement) {
                document.documentElement.setAttribute('dir', dir);
                document.documentElement.setAttribute('lang', locale);
            }
        },

        t(key, params) {
            var translation = this.translations[this.currentLocale];
            if (!translation) {
                console.warn('I18n: No translation loaded for', this.currentLocale);
                return key;
            }

            var keys = key.split('.');
            var value = translation;

            for (var i = 0; i < keys.length; i++) {
                if (value === undefined || value === null) {
                    return key;
                }
                value = value[keys[i]];
            }

            if (typeof value !== 'string') {
                var fallback = this.getFallbackTranslation(key);
                return fallback !== null ? fallback : key;
            }

            if (params) {
                value = this.interpolate(value, params);
            }

            return value;
        },

        getFallbackTranslation(key) {
            if (this.currentLocale === this.defaultLocale) {
                return null;
            }

            var fallback = this.translations[this.defaultLocale];
            if (!fallback) {
                return null;
            }

            var keys = key.split('.');
            var value = fallback;

            for (var i = 0; i < keys.length; i++) {
                if (value === undefined || value === null) {
                    return null;
                }
                value = value[keys[i]];
            }

            return typeof value === 'string' ? value : null;
        },

        interpolate(str, params) {
            return str.replace(/\{(\w+)\}/g, function(match, key) {
                return params[key] !== undefined ? params[key] : match;
            });
        },

        updateDOM() {
            var self = this;
            var nodes = document.querySelectorAll('[data-i18n]');
            nodes.forEach(function(node) {
                var key = node.getAttribute('data-i18n');
                var translated = self.t(key);
                if (translated !== key) {
                    node.textContent = translated;
                }
            });

            var attrNodes = document.querySelectorAll('[data-i18n-attr]');
            attrNodes.forEach(function(node) {
                var attrStr = node.getAttribute('data-i18n-attr');
                try {
                    var attrs = JSON.parse(attrStr);
                    for (var attr in attrs) {
                        if (attrs.hasOwnProperty(attr)) {
                            var translated = self.t(attrs[attr]);
                            if (translated !== attrs[attr]) {
                                node.setAttribute(attr, translated);
                            }
                        }
                    }
                } catch (e) {}
            });
        },

        onLocaleChange(callback) {
            if (typeof callback === 'function') {
                this.listeners.push(callback);
            }
        },

        offLocaleChange(callback) {
            var index = this.listeners.indexOf(callback);
            if (index > -1) {
                this.listeners.splice(index, 1);
            }
        },

        notifyListeners(oldLocale, newLocale) {
            this.listeners.forEach(function(callback) {
                try {
                    callback(newLocale, oldLocale);
                } catch (e) {
                    console.error('I18n: Listener error', e);
                }
            });
        },

        getCurrentLocale() {
            return this.currentLocale;
        },

        getTranslation() {
            return this.translations[this.currentLocale];
        },

        getSupportedLocales() {
            return this.supportedLocales.slice();
        },

        getLocaleInfo(locale) {
            locale = locale || this.currentLocale;
            var translation = this.translations[locale];
            if (translation) {
                return {
                    code: translation.code || locale,
                    name: translation.name || locale,
                    dir: translation.dir || 'ltr'
                };
            }
            return null;
        },

        formatNumber(num, options) {
            var locale = this.currentLocale;
            if (typeof Intl !== 'undefined' && Intl.NumberFormat) {
                return new Intl.NumberFormat(locale, options).format(num);
            }
            return num.toString();
        },

        formatDate(date, options) {
            var locale = this.currentLocale;
            if (typeof Intl !== 'undefined' && Intl.DateTimeFormat) {
                return new Intl.DateTimeFormat(locale, options).format(date);
            }
            return date.toLocaleDateString();
        },

        formatRelativeTime(seconds) {
            var translation = this.translations[this.currentLocale];
            var time = translation && translation.time ? translation.time : {};

            if (seconds < 60) {
                return time.justNow || 'Just now';
            } else if (seconds < 3600) {
                var minutes = Math.floor(seconds / 60);
                var template = time.minutesAgo || '{n} minutes ago';
                return template.replace('{n}', minutes);
            } else if (seconds < 86400) {
                var hours = Math.floor(seconds / 3600);
                var template = time.hoursAgo || '{n} hours ago';
                return template.replace('{n}', hours);
            } else if (seconds < 604800) {
                var days = Math.floor(seconds / 86400);
                var template = time.daysAgo || '{n} days ago';
                return template.replace('{n}', days);
            } else {
                var weeks = Math.floor(seconds / 604800);
                var template = time.weeksAgo || '{n} weeks ago';
                return template.replace('{n}', weeks);
            }
        }
    };

    global.I18n = I18n;

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = I18n;
    }

})(typeof window !== 'undefined' ? window : this);
