(function(global) {
    'use strict';

    var I18n = {
        currentLocale: 'en',
        defaultLocale: 'en',
        supportedLocales: ['zh-CN', 'zh-TW', 'en', 'ja', 'ko', 'ar', 'de', 'es', 'fr', 'ru', 'it', 'nl'],
        translations: {},
        listeners: [],
        initialized: false,
        initPromise: null,

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
            'ar-EG': 'ar',
            'ar-MA': 'ar',
            'ar-DZ': 'ar',
            'ar-LB': 'ar',
            'ar-IQ': 'ar',
            'ar-QA': 'ar',
            'en': 'en',
            'en-US': 'en',
            'en-GB': 'en',
            'en-AU': 'en',
            'en-CA': 'en',
            'de': 'de',
            'de-DE': 'de',
            'de-AT': 'de',
            'de-CH': 'de',
            'es': 'es',
            'es-ES': 'es',
            'es-MX': 'es',
            'es-AR': 'es',
            'fr': 'fr',
            'fr-FR': 'fr',
            'fr-CA': 'fr',
            'fr-BE': 'fr',
            'ru': 'ru',
            'ru-RU': 'ru',
            'ru-UA': 'ru',
            'it': 'it',
            'it-IT': 'it',
            'nl': 'nl',
            'nl-NL': 'nl',
            'nl-BE': 'nl'
        },

        async init(options) {
            if (this.initPromise) {
                return this.initPromise;
            }

            var self = this;
            this.initPromise = (async function() {
                options = options || {};
                self.defaultLocale = options.defaultLocale || 'en';
                self.localeParam = options.localeParam || 'lang';
                self.storageKey = options.storageKey || 'captchax_locale';
                self.containerSelector = options.containerSelector || '.language-switcher';

                await self.loadAllTranslations();
                var detectedLocale = self.detectLocale();
                await self.setLocale(detectedLocale, options.skipPersist);

                if (!self.initialized) {
                    self.setupLanguageSwitcher();
                    self.initialized = true;
                }

                return self;
            })();

            return this.initPromise;
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
            var urlLocale = this.getUrlLocale();
            if (urlLocale && this.isSupported(urlLocale)) {
                return urlLocale;
            }

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

        getUrlLocale() {
            try {
                var params = new URLSearchParams(window.location.search);
                var locale = params.get(this.localeParam);
                if (locale) {
                    var normalized = locale.toLowerCase().replace('_', '-');
                    var mapped = this.localeMap[normalized];
                    if (mapped) {
                        return mapped;
                    }
                    var prefix = normalized.split('-')[0];
                    for (var i = 0; i < this.supportedLocales.length; i++) {
                        var supported = this.supportedLocales[i].toLowerCase();
                        if (supported === prefix || supported.startsWith(prefix + '-')) {
                            return this.supportedLocales[i];
                        }
                    }
                    if (this.isSupported(normalized)) {
                        return normalized;
                    }
                }
            } catch (e) {}
            return null;
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
            this.updateLanguageSwitcher();
            this.notifyListeners(oldLocale, locale);

            return true;
        },

        applyDirection(locale) {
            var translation = this.translations[locale];
            var dir = translation && translation.dir ? translation.dir : 'ltr';

            if (document.documentElement) {
                document.documentElement.setAttribute('dir', dir);
                document.documentElement.setAttribute('lang', locale);
                document.body.classList.remove('rtl', 'ltr');
                document.body.classList.add(dir);
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

        setupLanguageSwitcher() {
            var container = document.querySelector(this.containerSelector);
            if (!container) {
                return;
            }

            var currentInfo = this.getLocaleInfo();
            container.innerHTML = this.createSwitcherHTML(currentInfo);

            var self = this;
            container.addEventListener('click', function(e) {
                var item = e.target.closest('[data-locale]');
                if (item) {
                    var locale = item.getAttribute('data-locale');
                    self.setLocale(locale);
                }
            });

            container.addEventListener('keydown', function(e) {
                if (e.key === 'Enter' || e.key === ' ') {
                    var item = e.target.closest('[data-locale]');
                    if (item) {
                        e.preventDefault();
                        var locale = item.getAttribute('data-locale');
                        self.setLocale(locale);
                    }
                }
            });

            this.updateLanguageSwitcher();
        },

        createSwitcherHTML(currentInfo) {
            var html = '<div class="language-switcher-dropdown">';
            html += '<button class="language-switcher-trigger" aria-haspopup="listbox" aria-expanded="false">';
            html += '<span class="language-switcher-current">' + (currentInfo ? currentInfo.name : 'English') + '</span>';
            html += '<span class="language-switcher-arrow"></span>';
            html += '</button>';
            html += '<ul class="language-switcher-list" role="listbox">';

            this.supportedLocales.forEach(function(locale) {
                var info = I18n.getLocaleInfo(locale);
                var isActive = locale === currentInfo.code;
                html += '<li role="option" aria-selected="' + (isActive ? 'true' : 'false') + '"';
                html += ' class="' + (isActive ? 'active' : '') + '"';
                html += ' data-locale="' + locale + '">';
                html += '<span class="language-name">' + (info ? info.name : locale) + '</span>';
                if (info && info.dir === 'rtl') {
                    html += '<span class="language-rtl-indicator" title="RTL">RTL</span>';
                }
                html += '</li>';
            });

            html += '</ul></div>';
            return html;
        },

        updateLanguageSwitcher() {
            var container = document.querySelector(this.containerSelector);
            if (!container) {
                return;
            }

            var currentInfo = this.getLocaleInfo();
            var trigger = container.querySelector('.language-switcher-current');
            if (trigger) {
                trigger.textContent = currentInfo ? currentInfo.name : 'English';
            }

            var items = container.querySelectorAll('[data-locale]');
            var self = this;
            items.forEach(function(item) {
                var locale = item.getAttribute('data-locale');
                var isActive = locale === self.currentLocale;
                item.classList.toggle('active', isActive);
                item.setAttribute('aria-selected', isActive ? 'true' : 'false');
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
        },

        isRTL() {
            var translation = this.translations[this.currentLocale];
            return translation && translation.dir === 'rtl';
        },

        isLTR() {
            return !this.isRTL();
        },

        clearStoredLocale() {
            try {
                localStorage.removeItem(this.storageKey);
            } catch (e) {}
        }
    };

    global.I18n = I18n;

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = I18n;
    }

})(typeof window !== 'undefined' ? window : this);
