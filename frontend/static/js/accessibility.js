const AccessibilityManager = {
    config: {
        enableFocusManagement: true,
        enableKeyboardNavigation: true,
        enableScreenReader: true,
        enableHighContrast: false,
        enableReducedMotion: true,
        enableSkipLinks: true,
        focusVisibleStyle: true,
        announcements: true
    },
    focusStack: [],
    lastFocusedElement: null,
    announcer: null,
    rovingTabindex: false,
    liveRegions: new Map(),

    async init(options = {}) {
        this.config = { ...this.config, ...options };
        this.setupAccessibilityFeatures();
        this.setupFocusManagement();
        this.setupKeyboardNavigation();
        this.setupScreenReaderSupport();
        this.setupSkipLinks();
        this.setupHighContrastMode();
        this.setupReducedMotion();
        this.setupARIA();
        this.setupLiveRegions();
        this.setupFocusVisible();
        return this;
    },

    setupAccessibilityFeatures() {
        document.body.classList.add('accessibility-enabled');

        this.detectPreferences();

        if (window.matchMedia) {
            const contrastQuery = window.matchMedia('(prefers-contrast: more)');
            contrastQuery.addEventListener('change', (e) => {
                this.handleContrastChange(e.matches);
            });

            const motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)');
            motionQuery.addEventListener('change', (e) => {
                this.handleReducedMotionChange(e.matches);
            });
        }
    },

    detectPreferences() {
        if (window.matchMedia) {
            if (window.matchMedia('(prefers-contrast: more)').matches) {
                this.config.enableHighContrast = true;
                document.body.classList.add('high-contrast');
            }

            if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
                this.config.enableReducedMotion = true;
                document.body.classList.add('reduced-motion');
            }
        }
    },

    setupFocusManagement() {
        if (!this.config.enableFocusManagement) return;

        document.addEventListener('focusout', (e) => {
            this.lastFocusedElement = e.relatedTarget;
        });

        document.addEventListener('focusin', (e) => {
            if (e.target !== document.body && e.target !== document.documentElement) {
                this.focusStack.push(e.target);
                if (this.focusStack.length > 10) {
                    this.focusStack.shift();
                }
            }
        });

        this.setupFocusTraps();
        this.setupFocusRestoration();
    },

    setupFocusTraps() {
        const trapContainers = document.querySelectorAll('[data-focus-trap]');

        trapContainers.forEach(container => {
            this.activateFocusTrap(container);
        });
    },

    activateFocusTrap(container) {
        const focusableElements = this.getFocusableElements(container);
        if (focusableElements.length === 0) return;

        const firstElement = focusableElements[0];
        const lastElement = focusableElements[focusableElements.length - 1];

        const handleTab = (e) => {
            if (e.key !== 'Tab') return;

            if (e.shiftKey) {
                if (document.activeElement === firstElement) {
                    e.preventDefault();
                    lastElement.focus();
                }
            } else {
                if (document.activeElement === lastElement) {
                    e.preventDefault();
                    firstElement.focus();
                }
            }
        };

        container.addEventListener('keydown', handleTab);
        container.dataset.focusTrapActive = 'true';
    },

    deactivateFocusTrap(container) {
        container.dataset.focusTrapActive = 'false';
    },

    getFocusableElements(container) {
        const selector = [
            'button:not([disabled])',
            'input:not([disabled])',
            'select:not([disabled])',
            'textarea:not([disabled])',
            'a[href]',
            '[tabindex]:not([tabindex="-1"])',
            '[contenteditable="true"]'
        ].join(',');

        return Array.from(container.querySelectorAll(selector))
            .filter(el => {
                const style = window.getComputedStyle(el);
                return style.display !== 'none' &&
                       style.visibility !== 'hidden' &&
                       el.offsetParent !== null;
            });
    },

    setupFocusRestoration() {
        document.addEventListener('click', (e) => {
            if (e.target.tagName === 'A' || e.target.closest('button')) {
                this.lastFocusedElement = document.activeElement;
            }
        });

        window.addEventListener('popstate', () => {
            if (this.lastFocusedElement && typeof this.lastFocusedElement.focus === 'function') {
                setTimeout(() => {
                    this.lastFocusedElement.focus();
                }, 100);
            }
        });
    },

    restoreFocus() {
        if (this.focusStack.length > 0) {
            const previousElement = this.focusStack[this.focusStack.length - 2];
            if (previousElement && typeof previousElement.focus === 'function') {
                previousElement.focus();
                return true;
            }
        }
        return false;
    },

    focusFirst(container = document) {
        const focusable = this.getFocusableElements(container);
        if (focusable.length > 0) {
            focusable[0].focus();
            return focusable[0];
        }
        return null;
    },

    focusLast(container = document) {
        const focusable = this.getFocusableElements(container);
        if (focusable.length > 0) {
            focusable[focusable.length - 1].focus();
            return focusable[focusable.length - 1];
        }
        return null;
    },

    setupKeyboardNavigation() {
        if (!this.config.enableKeyboardNavigation) return;

        document.addEventListener('keydown', (e) => {
            this.handleKeyboardEvent(e);
        });

        this.setupRovingTabindex();
        this.setupArrowNavigation();
    },

    handleKeyboardEvent(e) {
        if (e.ctrlKey || e.altKey || e.metaKey) return;

        switch (e.key) {
            case 'Tab':
                this.handleTabKey(e);
                break;
            case 'Escape':
                this.handleEscapeKey(e);
                break;
            case 'Enter':
            case ' ':
                this.handleActivationKey(e);
                break;
            case 'ArrowUp':
            case 'ArrowDown':
            case 'ArrowLeft':
            case 'ArrowRight':
                this.handleArrowKey(e);
                break;
            case 'Home':
            case 'End':
                this.handleHomeEndKey(e);
                break;
        }
    },

    handleTabKey(e) {
        const activeElement = document.activeElement;
        if (!activeElement) return;

        if (e.shiftKey) {
            this.dispatchEvent('keyboard:navigate-backward');
        } else {
            this.dispatchEvent('keyboard:navigate-forward');
        }
    },

    handleEscapeKey(e) {
        const activeElement = document.activeElement;

        if (activeElement && activeElement.blur) {
            activeElement.blur();
        }

        const modals = document.querySelectorAll('.modal.show, [role="dialog"][aria-hidden="false"]');
        modals.forEach(modal => {
            const closeButton = modal.querySelector('[data-dismiss], [aria-label="关闭"], .btn-close');
            if (closeButton) {
                closeButton.click();
            }
        });

        this.dispatchEvent('keyboard:escape');
    },

    handleActivationKey(e) {
        const target = e.target;
        if (target.tagName === 'BUTTON' || target.tagName === 'INPUT' || target.tagName === 'SELECT') {
            return;
        }

        if (target.getAttribute('role') === 'button' || target.classList.contains('btn')) {
            e.preventDefault();
            target.click();
        }
    },

    handleArrowKey(e) {
        const activeElement = document.activeElement;
        if (!activeElement) return;

        const roving = activeElement.closest('[data-roving-tabindex]');
        if (roving) {
            e.preventDefault();
            this.navigateRovingTabindex(roving, e.key);
        }
    },

    handleHomeEndKey(e) {
        const activeElement = document.activeElement;
        if (!activeElement) return;

        const container = activeElement.closest('[data-roving-tabindex]');
        if (container) {
            e.preventDefault();
            const items = container.querySelectorAll('[data-roving-item]');
            if (e.key === 'Home' && items.length > 0) {
                items[0].focus();
            } else if (e.key === 'End' && items.length > 0) {
                items[items.length - 1].focus();
            }
        }
    },

    setupRovingTabindex() {
        const rovingContainers = document.querySelectorAll('[data-roving-tabindex]');

        rovingContainers.forEach(container => {
            const items = container.querySelectorAll('[data-roving-item]');

            items.forEach((item, index) => {
                item.setAttribute('tabindex', index === 0 ? '0' : '-1');
            });
        });
    },

    navigateRovingTabindex(container, key) {
        const items = Array.from(container.querySelectorAll('[data-roving-item]'));
        const currentIndex = items.indexOf(document.activeElement);

        if (currentIndex === -1) return;

        let nextIndex;
        const isHorizontal = key === 'ArrowLeft' || key === 'ArrowRight';
        const isVertical = key === 'ArrowUp' || key === 'ArrowDown';

        if (key === 'ArrowRight' || key === 'ArrowDown') {
            nextIndex = (currentIndex + 1) % items.length;
        } else {
            nextIndex = (currentIndex - 1 + items.length) % items.length;
        }

        items.forEach((item, index) => {
            item.setAttribute('tabindex', index === nextIndex ? '0' : '-1');
        });

        items[nextIndex].focus();
        items[nextIndex].scrollIntoView({ block: 'nearest', inline: 'nearest' });
    },

    setupArrowNavigation() {
        const navigableLists = document.querySelectorAll('[data-keyboard-nav]');

        navigableLists.forEach(list => {
            list.addEventListener('keydown', (e) => {
                if (!['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'].includes(e.key)) return;

                const items = Array.from(list.querySelectorAll('[data-nav-item]'));
                const currentIndex = items.indexOf(document.activeElement);

                if (currentIndex === -1) return;

                e.preventDefault();

                let nextIndex;
                if (e.key === 'ArrowDown' || e.key === 'ArrowRight') {
                    nextIndex = Math.min(currentIndex + 1, items.length - 1);
                } else {
                    nextIndex = Math.max(currentIndex - 1, 0);
                }

                items[nextIndex].focus();
            });
        });
    },

    setupScreenReaderSupport() {
        if (!this.config.enableScreenReader) return;

        this.createScreenReaderAnnouncer();
        this.setupAriaDescriptions();
        this.setupAriaLabels();
    },

    createScreenReaderAnnouncer() {
        this.announcer = document.createElement('div');
        this.announcer.id = 'screen-reader-announcer';
        this.announcer.setAttribute('role', 'status');
        this.announcer.setAttribute('aria-live', 'polite');
        this.announcer.setAttribute('aria-atomic', 'true');
        this.announcer.className = 'sr-only';
        document.body.appendChild(this.announcer);
    },

    announce(message, priority = 'polite') {
        if (!this.config.announcements) return;

        this.announcer.setAttribute('aria-live', priority);
        this.announcer.textContent = '';

        setTimeout(() => {
            this.announcer.textContent = message;
        }, 100);

        setTimeout(() => {
            this.announcer.textContent = '';
        }, 1000);
    },

    announceAssertive(message) {
        this.announce(message, 'assertive');
    },

    setupAriaDescriptions() {
        const elementsWithTitles = document.querySelectorAll('[title]');

        elementsWithTitles.forEach(el => {
            if (!el.getAttribute('aria-describedby')) {
                const id = `desc-${Math.random().toString(36).substr(2, 9)}`;
                const descElement = document.createElement('span');
                descElement.id = id;
                descElement.className = 'sr-only';
                descElement.textContent = el.getAttribute('title');
                el.setAttribute('aria-describedby', id);
                el.removeAttribute('title');
                document.body.appendChild(descElement);
            }
        });
    },

    setupAriaLabels() {
        const unlabeledButtons = document.querySelectorAll('button:not([aria-label]):not([aria-labelledby])');
        unlabeledButtons.forEach(btn => {
            if (!btn.textContent.trim()) {
                const icon = btn.querySelector('i, span[class*="icon"]');
                if (icon) {
                    btn.setAttribute('aria-label', icon.textContent.trim() || '按钮');
                }
            }
        });
    },

    setupSkipLinks() {
        if (!this.config.enableSkipLinks) return;

        const skipLink = document.createElement('a');
        skipLink.href = '#main-content';
        skipLink.className = 'skip-link sr-only sr-only-focusable';
        skipLink.textContent = '跳转到主要内容';
        skipLink.addEventListener('click', (e) => {
            e.preventDefault();
            const target = document.querySelector('#main-content');
            if (target) {
                target.tabIndex = -1;
                target.focus();
                target.scrollIntoView();
            }
        });

        document.body.insertBefore(skipLink, document.body.firstChild);

        const mainContent = document.querySelector('main, [role="main"], #main-content');
        if (mainContent && !mainContent.id) {
            mainContent.id = 'main-content';
        }
    },

    setupHighContrastMode() {
        const toggle = document.getElementById('contrastToggle');
        if (toggle) {
            toggle.addEventListener('click', () => {
                this.toggleHighContrast();
            });

            if (this.config.enableHighContrast) {
                this.enableHighContrast();
            }
        }
    },

    toggleHighContrast() {
        this.config.enableHighContrast = !this.config.enableHighContrast;

        if (this.config.enableHighContrast) {
            this.enableHighContrast();
        } else {
            this.disableHighContrast();
        }
    },

    enableHighContrast() {
        document.body.classList.add('high-contrast');

        const style = document.createElement('style');
        style.id = 'high-contrast-styles';
        style.textContent = `
            .high-contrast * {
                border-color: #000 !important;
            }
            .high-contrast a {
                text-decoration: underline !important;
            }
            .high-contrast img {
                filter: contrast(1.2);
            }
            .high-contrast .btn {
                border-width: 2px !important;
            }
        `;
        document.head.appendChild(style);
    },

    disableHighContrast() {
        document.body.classList.remove('high-contrast');
        const style = document.getElementById('high-contrast-styles');
        if (style) {
            style.remove();
        }
    },

    handleContrastChange(matches) {
        if (matches) {
            this.enableHighContrast();
        } else {
            this.disableHighContrast();
        }
    },

    setupReducedMotion() {
        const toggle = document.getElementById('motionToggle');
        if (toggle) {
            toggle.addEventListener('click', () => {
                this.toggleReducedMotion();
            });
        }

        if (this.config.enableReducedMotion) {
            this.applyReducedMotion();
        }
    },

    toggleReducedMotion() {
        this.config.enableReducedMotion = !this.config.enableReducedMotion;

        if (this.config.enableReducedMotion) {
            this.applyReducedMotion();
        } else {
            this.removeReducedMotion();
        }
    },

    applyReducedMotion() {
        document.body.classList.add('reduced-motion');

        const style = document.createElement('style');
        style.id = 'reduced-motion-styles';
        style.textContent = `
            .reduced-motion *,
            .reduced-motion *::before,
            .reduced-motion *::after {
                animation-duration: 0.001ms !important;
                animation-iteration-count: 1 !important;
                transition-duration: 0.001ms !important;
            }
        `;
        document.head.appendChild(style);
    },

    removeReducedMotion() {
        document.body.classList.remove('reduced-motion');
        const style = document.getElementById('reduced-motion-styles');
        if (style) {
            style.remove();
        }
    },

    handleReducedMotionChange(matches) {
        if (matches) {
            this.config.enableReducedMotion = true;
            this.applyReducedMotion();
        } else {
            this.config.enableReducedMotion = false;
            this.removeReducedMotion();
        }
    },

    setupARIA() {
        this.updateLandmarkRoles();
        this.setupARIAStates();
        this.setupARIAProperties();
    },

    updateLandmarkRoles() {
        const main = document.querySelector('main:not([role])');
        if (main) {
            main.setAttribute('role', 'main');
        }

        const headers = document.querySelectorAll('header:not([role])');
        headers.forEach(header => {
            const nav = header.querySelector('nav');
            if (!nav) {
                header.setAttribute('role', 'banner');
            }
        });

        const footers = document.querySelectorAll('footer:not([role])');
        footers.forEach(footer => {
            footer.setAttribute('role', 'contentinfo');
        });

        const navs = document.querySelectorAll('nav:not([aria-label])');
        navs.forEach((nav, index) => {
            const navName = nav.getAttribute('aria-labelledby') ||
                           nav.closest('section, article')?.querySelector('h1, h2, h3')?.textContent ||
                           `导航 ${index + 1}`;
            nav.setAttribute('aria-label', navName);
        });
    },

    setupARIAStates() {
        const expandableElements = document.querySelectorAll('[aria-expanded]');

        expandableElements.forEach(el => {
            const expanded = el.getAttribute('aria-expanded') === 'true';
            this.updateExpandedState(el, expanded);
        });

        const collapsibleSections = document.querySelectorAll('[data-collapse]');
        collapsibleSections.forEach(section => {
            const toggle = section.querySelector('[aria-expanded]');
            if (toggle) {
                toggle.addEventListener('click', () => {
                    const isExpanded = toggle.getAttribute('aria-expanded') === 'true';
                    toggle.setAttribute('aria-expanded', !isExpanded);
                    section.classList.toggle('show', !isExpanded);
                });
            }
        });
    },

    updateExpandedState(element, expanded) {
        element.setAttribute('aria-expanded', expanded.toString());

        const controls = element.getAttribute('aria-controls');
        if (controls) {
            const target = document.getElementById(controls);
            if (target) {
                target.hidden = expanded;
            }
        }
    },

    setupARIAProperties() {
        const items = document.querySelectorAll('[data-item-count]');
        items.forEach(item => {
            const container = item.closest('ul, ol');
            if (container) {
                const items = container.querySelectorAll('li').length;
                const currentIndex = Array.from(container.querySelectorAll('li')).indexOf(item) + 1;
                item.setAttribute('aria-setsize', items.toString());
                item.setAttribute('aria-posinset', currentIndex.toString());
            }
        });

        const progressElements = document.querySelectorAll('[data-progress]');
        progressElements.forEach(el => {
            const value = el.getAttribute('data-progress');
            el.setAttribute('aria-valuenow', value);
            el.setAttribute('aria-valuemin', '0');
            el.setAttribute('aria-valuemax', '100');
            el.setAttribute('role', 'progressbar');
        });
    },

    setupLiveRegions() {
        const liveRegionConfigs = [
            { id: 'status', type: 'status', politeness: 'polite' },
            { id: 'alert', type: 'alert', politeness: 'assertive' },
            { id: 'log', type: 'log', politeness: 'polite' }
        ];

        liveRegionConfigs.forEach(config => {
            const existing = document.getElementById(config.id);
            if (existing) {
                this.liveRegions.set(config.id, existing);
                return;
            }

            const region = document.createElement('div');
            region.id = config.id;
            region.setAttribute('role', config.type);
            region.setAttribute('aria-live', config.politeness);
            region.setAttribute('aria-atomic', 'true');
            region.className = 'sr-only';
            document.body.appendChild(region);

            this.liveRegions.set(config.id, region);
        });
    },

    updateLiveRegion(regionId, message) {
        const region = this.liveRegions.get(regionId);
        if (region) {
            region.textContent = '';
            setTimeout(() => {
                region.textContent = message;
            }, 50);
        }
    },

    setupFocusVisible() {
        if (!this.config.focusVisibleStyle) return;

        const style = document.createElement('style');
        style.id = 'focus-visible-styles';
        style.textContent = `
            :focus-visible {
                outline: 2px solid var(--focus-color, #0d6efd) !important;
                outline-offset: 2px !important;
            }

            :focus:not(:focus-visible) {
                outline: none !important;
            }

            .high-contrast :focus-visible {
                outline-width: 3px !important;
            }
        `;
        document.head.appendChild(style);
    },

    getAccessibilityInfo() {
        return {
            highContrast: this.config.enableHighContrast,
            reducedMotion: this.config.enableReducedMotion,
            keyboardNavigation: this.config.enableKeyboardNavigation,
            screenReaderSupport: this.config.enableScreenReader,
            skipLinksEnabled: this.config.enableSkipLinks
        };
    },

    dispatchEvent(eventName, detail = {}) {
        const event = new CustomEvent(`accessibility:${eventName}`, { detail });
        document.dispatchEvent(event);
    },

    on(eventName, handler) {
        document.addEventListener(`accessibility:${eventName}`, (e) => handler(e.detail));
    },

    off(eventName, handler) {
        document.removeEventListener(`accessibility:${eventName}`, (e) => handler(e.detail));
    }
};

if (typeof window !== 'undefined') {
    window.AccessibilityManager = AccessibilityManager;
}
