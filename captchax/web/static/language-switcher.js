(function(global) {
    'use strict';

    var LanguageSwitcher = {
        container: null,
        isOpen: false,

        init: function(options) {
            options = options || {};
            this.position = options.position || 'top-right';
            this.showFlag = options.showFlag !== false;
            this.compact = options.compact !== false;
            this.onChange = options.onChange || null;

            this.createSwitcher();
            this.attachEvents();

            return this;
        },

        createSwitcher: function() {
            var existing = document.getElementById('captchax-lang-switcher');
            if (existing) {
                existing.parentNode.removeChild(existing);
            }

            var wrapper = document.createElement('div');
            wrapper.id = 'captchax-lang-switcher';
            wrapper.className = 'captchax-lang-switcher';

            var btn = document.createElement('button');
            btn.type = 'button';
            btn.id = 'captchax-lang-btn';
            btn.className = 'captchax-lang-btn';
            btn.setAttribute('aria-haspopup', 'listbox');
            btn.setAttribute('aria-expanded', 'false');

            var currentLocale = typeof I18n !== 'undefined' ? I18n.getCurrentLocale() : 'en';
            var localeInfo = typeof I18n !== 'undefined' ? I18n.getLocaleInfo(currentLocale) : { name: 'English', code: 'en' };

            btn.innerHTML = this.createButtonContent(localeInfo);

            var dropdown = document.createElement('div');
            dropdown.id = 'captchax-lang-dropdown';
            dropdown.className = 'captchax-lang-dropdown';
            dropdown.setAttribute('role', 'listbox');
            dropdown.setAttribute('aria-label', 'Select language');

            var locales = typeof I18n !== 'undefined' ? I18n.getSupportedLocales() : ['en', 'zh-CN', 'zh-TW', 'ja', 'ko', 'ar'];

            locales.forEach(function(locale) {
                var localeData = typeof I18n !== 'undefined' ? I18n.getLocaleInfo(locale) : { name: locale, code: locale, dir: 'ltr' };
                var option = document.createElement('div');
                option.className = 'captchax-lang-option' + (locale === currentLocale ? ' active' : '');
                option.setAttribute('role', 'option');
                option.setAttribute('data-locale', locale);
                option.setAttribute('aria-selected', locale === currentLocale ? 'true' : 'false');
                option.setAttribute('dir', localeData.dir || 'ltr');
                option.innerHTML = this.createOptionContent(localeData);
                dropdown.appendChild(option);
            }.bind(this));

            wrapper.appendChild(btn);
            wrapper.appendChild(dropdown);

            if (this.position === 'top-right') {
                wrapper.style.top = '20px';
                wrapper.style.right = '20px';
                wrapper.style.left = 'auto';
            } else if (this.position === 'top-left') {
                wrapper.style.top = '20px';
                wrapper.style.left = '20px';
            } else if (this.position === 'bottom-right') {
                wrapper.style.bottom = '20px';
                wrapper.style.right = '20px';
                wrapper.style.top = 'auto';
            } else if (this.position === 'bottom-left') {
                wrapper.style.bottom = '20px';
                wrapper.style.left = '20px';
                wrapper.style.top = 'auto';
            }

            document.body.appendChild(wrapper);
            this.container = wrapper;

            this.injectStyles();
        },

        createButtonContent: function(localeInfo) {
            var flag = this.getFlagEmoji(localeInfo.code);
            var name = localeInfo.name;

            if (this.compact) {
                return '<span class="captchax-lang-flag">' + flag + '</span><span class="captchax-lang-code">' + localeInfo.code.toUpperCase() + '</span><svg class="captchax-lang-arrow" width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><path d="M2.5 4.5L6 8L9.5 4.5"/></svg>';
            }
            return '<span class="captchax-lang-flag">' + flag + '</span><span class="captchax-lang-name">' + name + '</span><svg class="captchax-lang-arrow" width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><path d="M2.5 4.5L6 8L9.5 4.5"/></svg>';
        },

        createOptionContent: function(localeInfo) {
            var flag = this.getFlagEmoji(localeInfo.code);
            return '<span class="captchax-lang-option-flag">' + flag + '</span><span class="captchax-lang-option-name">' + localeInfo.name + '</span><span class="captchax-lang-option-code">' + localeInfo.code + '</span>';
        },

        getFlagEmoji: function(locale) {
            var flags = {
                'zh-CN': '🇨🇳',
                'zh-TW': '🇹🇼',
                'en': '🇺🇸',
                'ja': '🇯🇵',
                'ko': '🇰🇷',
                'ar': '🇸🇦'
            };
            return flags[locale] || '🌐';
        },

        attachEvents: function() {
            var self = this;
            var btn = document.getElementById('captchax-lang-btn');
            var dropdown = document.getElementById('captchax-lang-dropdown');

            if (btn) {
                btn.addEventListener('click', function(e) {
                    e.stopPropagation();
                    self.toggle();
                });
            }

            if (dropdown) {
                dropdown.addEventListener('click', function(e) {
                    var option = e.target.closest('.captchax-lang-option');
                    if (option) {
                        var locale = option.getAttribute('data-locale');
                        self.selectLocale(locale);
                    }
                });
            }

            document.addEventListener('click', function(e) {
                if (self.isOpen && !self.container.contains(e.target)) {
                    self.close();
                }
            });

            document.addEventListener('keydown', function(e) {
                if (e.key === 'Escape' && self.isOpen) {
                    self.close();
                }
            });

            if (typeof I18n !== 'undefined') {
                I18n.onLocaleChange(function(newLocale) {
                    self.updateButton(newLocale);
                    self.updateOptions(newLocale);
                });
            }
        },

        toggle: function() {
            if (this.isOpen) {
                this.close();
            } else {
                this.open();
            }
        },

        open: function() {
            this.isOpen = true;
            var dropdown = document.getElementById('captchax-lang-dropdown');
            var btn = document.getElementById('captchax-lang-btn');
            if (dropdown) {
                dropdown.classList.add('open');
            }
            if (btn) {
                btn.setAttribute('aria-expanded', 'true');
            }
        },

        close: function() {
            this.isOpen = false;
            var dropdown = document.getElementById('captchax-lang-dropdown');
            var btn = document.getElementById('captchax-lang-btn');
            if (dropdown) {
                dropdown.classList.remove('open');
            }
            if (btn) {
                btn.setAttribute('aria-expanded', 'false');
            }
        },

        async selectLocale: function(locale) {
            this.close();

            if (typeof I18n !== 'undefined') {
                await I18n.setLocale(locale);
            }

            if (this.onChange && typeof this.onChange === 'function') {
                this.onChange(locale);
            }
        },

        updateButton: function(locale) {
            var btn = document.getElementById('captchax-lang-btn');
            if (!btn) return;

            var localeInfo = typeof I18n !== 'undefined' ? I18n.getLocaleInfo(locale) : { name: locale, code: locale };
            btn.innerHTML = this.createButtonContent(localeInfo);
        },

        updateOptions: function(currentLocale) {
            var options = document.querySelectorAll('.captchax-lang-option');
            options.forEach(function(option) {
                var locale = option.getAttribute('data-locale');
                if (locale === currentLocale) {
                    option.classList.add('active');
                    option.setAttribute('aria-selected', 'true');
                } else {
                    option.classList.remove('active');
                    option.setAttribute('aria-selected', 'false');
                }
            });
        },

        destroy: function() {
            if (this.container) {
                this.container.parentNode.removeChild(this.container);
                this.container = null;
            }
        },

        injectStyles: function() {
            var styleId = 'captchax-lang-switcher-styles';
            if (document.getElementById(styleId)) return;

            var styles = document.createElement('style');
            styles.id = styleId;
            styles.textContent = [
                '.captchax-lang-switcher {',
                '    position: fixed;',
                '    z-index: 9999;',
                '    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;',
                '}',
                '.captchax-lang-btn {',
                '    display: flex;',
                '    align-items: center;',
                '    gap: 8px;',
                '    padding: 8px 12px;',
                '    background: rgba(255, 255, 255, 0.95);',
                '    border: 1px solid #e5e7eb;',
                '    border-radius: 8px;',
                '    cursor: pointer;',
                '    font-size: 14px;',
                '    color: #374151;',
                '    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);',
                '    transition: all 0.2s ease;',
                '}',
                '.captchax-lang-btn:hover {',
                '    background: #fff;',
                '    border-color: #d1d5db;',
                '    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);',
                '}',
                '.captchax-lang-flag {',
                '    font-size: 18px;',
                '    line-height: 1;',
                '}',
                '.captchax-lang-name, .captchax-lang-code {',
                '    font-weight: 500;',
                '}',
                '.captchax-lang-arrow {',
                '    transition: transform 0.2s ease;',
                '    opacity: 0.6;',
                '}',
                '.captchax-lang-btn:hover .captchax-lang-arrow {',
                '    opacity: 1;',
                '}',
                '.captchax-lang-dropdown {',
                '    position: absolute;',
                '    top: calc(100% + 8px);',
                '    right: 0;',
                '    min-width: 180px;',
                '    background: #fff;',
                '    border: 1px solid #e5e7eb;',
                '    border-radius: 12px;',
                '    box-shadow: 0 10px 40px rgba(0, 0, 0, 0.15);',
                '    padding: 8px;',
                '    opacity: 0;',
                '    visibility: hidden;',
                '    transform: translateY(-10px);',
                '    transition: all 0.2s ease;',
                '}',
                '.captchax-lang-dropdown.open {',
                '    opacity: 1;',
                '    visibility: visible;',
                '    transform: translateY(0);',
                '}',
                '.captchax-lang-option {',
                '    display: flex;',
                '    align-items: center;',
                '    gap: 10px;',
                '    padding: 10px 12px;',
                '    border-radius: 8px;',
                '    cursor: pointer;',
                '    transition: background 0.15s ease;',
                '}',
                '.captchax-lang-option:hover {',
                '    background: #f3f4f6;',
                '}',
                '.captchax-lang-option.active {',
                '    background: #eff6ff;',
                '}',
                '.captchax-lang-option-flag {',
                '    font-size: 20px;',
                '    line-height: 1;',
                '}',
                '.captchax-lang-option-name {',
                '    flex: 1;',
                '    font-size: 14px;',
                '    color: #374151;',
                '}',
                '.captchax-lang-option-code {',
                '    font-size: 11px;',
                '    color: #9ca3af;',
                '    text-transform: uppercase;',
                '}',
                '.captchax-lang-option.active .captchax-lang-option-name {',
                '    color: #2563eb;',
                '    font-weight: 500;',
                '}',
                '[dir="rtl"] .captchax-lang-dropdown {',
                '    right: auto;',
                '    left: 0;',
                '}',
                '[dir="rtl"] .captchax-lang-arrow {',
                '    transform: rotate(180deg);',
                '}'
            ].join('');

            document.head.appendChild(styles);
        }
    };

    global.LanguageSwitcher = LanguageSwitcher;

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = LanguageSwitcher;
    }

})(typeof window !== 'undefined' ? window : this);
