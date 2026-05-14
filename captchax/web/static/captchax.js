/**
 * CaptchaX Client SDK
 * @version 1.1.0
 * @license MIT
 */
(function(global) {
    'use strict';

    const CaptchaX = {
        version: '1.1.0',
        config: {
            serverUrl: '',
            appId: '',
            timeout: 30000,
            retryAttempts: 3,
            retryDelay: 1000,
            loadingText: '加载中...',
            errorText: '网络错误，请重试',
            successText: '验证成功',
            failText: '验证失败',
            expireText: '验证码已过期'
        },
        instances: new Map(),
        defaultInstance: null,
        callbacks: {
            onSuccess: [],
            onError: [],
            onReady: [],
            onVerify: []
        },
        state: {
            ready: false,
            loading: false
        }
    };

    function generateUUID() {
        return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
            const r = Math.random() * 16 | 0;
            const v = c === 'x' ? r : (r & 0x3 | 0x8);
            return v.toString(16);
        });
    }

    function extend(target, source) {
        for (const key in source) {
            if (Object.prototype.hasOwnProperty.call(source, key)) {
                target[key] = source[key];
            }
        }
        return target;
    }

    function deepMerge(target, source) {
        const result = extend({}, target);
        for (const key in source) {
            if (Object.prototype.hasOwnProperty.call(source, key)) {
                if (typeof source[key] === 'object' && source[key] !== null && !Array.isArray(source[key])) {
                    result[key] = deepMerge(result[key] || {}, source[key]);
                } else {
                    result[key] = source[key];
                }
            }
        }
        return result;
    }

    function getElement(element) {
        if (typeof element === 'string') {
            return document.querySelector(element);
        }
        return element || null;
    }

    function createElement(tag, className, attributes) {
        const element = document.createElement(tag);
        if (className) {
            element.className = className;
        }
        if (attributes) {
            for (const key in attributes) {
                if (key === 'style' && typeof attributes[key] === 'object') {
                    for (const styleKey in attributes[key]) {
                        element.style[styleKey] = attributes[key][styleKey];
                    }
                } else if (key === 'dataset') {
                    for (const dataKey in attributes[key]) {
                        element.dataset[dataKey] = attributes[key][dataKey];
                    }
                } else {
                    element.setAttribute(key, attributes[key]);
                }
            }
        }
        return element;
    }

    function addClass(element, className) {
        if (element && className) {
            const classes = className.split(' ').filter(c => c);
            element.classList.add(...classes);
        }
    }

    function removeClass(element, className) {
        if (element && className) {
            const classes = className.split(' ').filter(c => c);
            element.classList.remove(...classes);
        }
    }

    function hasClass(element, className) {
        return element && className && element.classList.contains(className);
    }

    function show(element) {
        if (element) {
            element.style.display = '';
            removeClass(element, 'captchax-hidden');
        }
    }

    function hide(element) {
        if (element) {
            element.style.display = 'none';
            addClass(element, 'captchax-hidden');
        }
    }

    function announceToScreenReader(message, politeness = 'polite') {
        const liveRegion = document.createElement('div');
        liveRegion.setAttribute('role', 'status');
        liveRegion.setAttribute('aria-live', politeness);
        liveRegion.setAttribute('aria-atomic', 'true');
        liveRegion.className = 'captchax-sr-announce';
        liveRegion.textContent = message;
        document.body.appendChild(liveRegion);
        setTimeout(() => {
            liveRegion.remove();
        }, 1000);
    }

    function request(url, options) {
        const config = extend({
            method: 'GET',
            headers: {
                'Content-Type': 'application/json'
            },
            body: null,
            timeout: CaptchaX.config.timeout
        }, options);

        return new Promise((resolve, reject) => {
            const xhr = new XMLHttpRequest();

            xhr.open(config.method, url, true);
            xhr.setRequestHeader('Content-Type', 'application/json');

            for (const header in config.headers) {
                xhr.setRequestHeader(header, config.headers[header]);
            }

            xhr.timeout = config.timeout;

            xhr.onload = function() {
                if (xhr.status >= 200 && xhr.status < 300) {
                    try {
                        const response = JSON.parse(xhr.responseText);
                        resolve(response);
                    } catch (e) {
                        resolve(xhr.responseText);
                    }
                } else {
                    reject(new Error(`HTTP ${xhr.status}: ${xhr.statusText}`));
                }
            };

            xhr.onerror = function() {
                reject(new Error('Network error'));
            };

            xhr.ontimeout = function() {
                reject(new Error('Request timeout'));
            };

            if (config.body) {
                xhr.send(typeof config.body === 'string' ? config.body : JSON.stringify(config.body));
            } else {
                xhr.send();
            }
        });
    }

    function getAbsoluteUrl(relativePath) {
        if (!relativePath) return '';
        if (relativePath.startsWith('http://') || relativePath.startsWith('https://')) {
            return relativePath;
        }
        const base = CaptchaX.config.serverUrl.replace(/\/$/, '');
        const path = relativePath.replace(/^\//, '');
        return `${base}/${path}`;
    }

    CaptchaX.init = function(options) {
        return new Promise((resolve, reject) => {
            const config = deepMerge(CaptchaX.config, options || {});

            if (!config.serverUrl) {
                const scripts = document.getElementsByTagName('script');
                for (let i = scripts.length - 1; i >= 0; i--) {
                    const src = scripts[i].getAttribute('src');
                    if (src && src.includes('captchax')) {
                        config.serverUrl = src.replace(/\/[^/]+\.js$/, '');
                        break;
                    }
                }
                if (!config.serverUrl) {
                    config.serverUrl = window.location.origin;
                }
            }

            CaptchaX.config = config;
            CaptchaX.state.ready = true;

            CaptchaX.callbacks.onReady.forEach(cb => {
                try {
                    cb(CaptchaX);
                } catch (e) {
                    console.error('[CaptchaX] onReady callback error:', e);
                }
            });

            resolve(CaptchaX);
        });
    };

    CaptchaX.create = function(options) {
        if (!CaptchaX.state.ready) {
            console.warn('[CaptchaX] SDK not initialized, calling init() automatically');
            return CaptchaX.init().then(() => CaptchaX.create(options));
        }

        const instanceId = generateUUID();
        const container = getElement(options.container);

        if (!container) {
            throw new Error('[CaptchaX] Container element not found');
        }

        const instance = {
            id: instanceId,
            container: container,
            config: extend({}, CaptchaX.config, options || {}),
            state: {
                captchaId: null,
                captchaType: options.type || 'slider',
                verified: false,
                loading: false,
                destroyed: false
            },
            elements: {},
            track: []
        };

        container.innerHTML = '';
        addClass(container, 'captchax-container');
        container.dataset.captchaxId = instanceId;

        if (instance.config.theme === 'auto') {
            addClass(container, 'captchax-auto');
        } else if (instance.config.theme === 'dark') {
            addClass(container, 'captchax-dark');
        } else if (instance.config.theme === 'light') {
            addClass(container, 'captchax-light');
        }

        CaptchaX.instances.set(instanceId, instance);
        CaptchaX.defaultInstance = instance;

        loadResources(instance).then(() => {
            renderWidget(instance);
            bindEvents(instance);
        }).catch(err => {
            console.error('[CaptchaX] Failed to load resources:', err);
            renderError(instance, err.message);
        });

        return instance;
    };

    function loadResources(instance) {
        return new Promise((resolve, reject) => {
            let cssLoaded = false;
            let templateLoaded = false;

            const baseUrl = instance.config.serverUrl || CaptchaX.config.serverUrl;

            const loadCSS = () => {
                const linkId = 'captchax-styles';
                let link = document.getElementById(linkId);

                if (!link) {
                    link = createElement('link', '', {
                        id: linkId,
                        rel: 'stylesheet',
                        href: getAbsoluteUrl('/static/styles.css')
                    });
                    document.head.appendChild(link);
                }

                link.addEventListener('load', () => {
                    cssLoaded = true;
                    if (templateLoaded) resolve();
                });

                link.addEventListener('error', () => {
                    cssLoaded = true;
                    console.warn('[CaptchaX] CSS load failed, using inline styles');
                    if (templateLoaded) resolve();
                });

                setTimeout(() => {
                    if (!cssLoaded) {
                        cssLoaded = true;
                        if (templateLoaded) resolve();
                    }
                }, 5000);
            };

            const loadTemplate = () => {
                const xhr = new XMLHttpRequest();
                xhr.open('GET', getAbsoluteUrl('/templates/widget.html'), true);
                xhr.onload = function() {
                    if (xhr.status === 200) {
                        instance.template = xhr.responseText;
                    } else {
                        instance.template = getDefaultTemplate();
                    }
                    templateLoaded = true;
                    if (cssLoaded) resolve();
                };
                xhr.onerror = function() {
                    instance.template = getDefaultTemplate();
                    templateLoaded = true;
                    if (cssLoaded) resolve();
                };
                xhr.send();
            };

            loadCSS();
            loadTemplate();
        });
    }

    function getDefaultTemplate() {
        return '<div class="captchax-widget">' +
            '<div class="captchax-header">' +
            '<span class="captchax-title">安全验证</span>' +
            '<button type="button" class="captchax-close" aria-label="关闭">×</button>' +
            '</div>' +
            '<div class="captchax-body"></div>' +
            '<div class="captchax-footer"></div>' +
            '</div>';
    }

    function renderWidget(instance) {
        const { container, template } = instance;

        container.innerHTML = template;

        instance.elements.widget = container.querySelector('.captchax-widget');
        instance.elements.header = container.querySelector('.captchax-header');
        instance.elements.title = container.querySelector('.captchax-title');
        instance.elements.body = container.querySelector('.captchax-body');
        instance.elements.footer = container.querySelector('.captchax-footer');
        instance.elements.closeBtn = container.querySelector('.captchax-close');

        const liveRegion = document.createElement('div');
        liveRegion.setAttribute('role', 'status');
        liveRegion.setAttribute('aria-live', 'polite');
        liveRegion.className = 'captchax-live-region';
        container.appendChild(liveRegion);
        instance.elements.liveRegion = liveRegion;

        showLoading(instance);

        fetchCaptcha(instance).then(data => {
            instance.state.captchaId = data.id;
            instance.state.loading = false;
            renderCaptcha(instance, data);
            updateLiveRegion(instance, '验证码已加载完成');
        }).catch(err => {
            instance.state.loading = false;
            renderError(instance, err.message || CaptchaX.config.errorText);
            updateLiveRegion(instance, '验证码加载失败');
        });
    }

    function updateLiveRegion(instance, message) {
        if (instance.elements.liveRegion) {
            instance.elements.liveRegion.textContent = message;
        }
    }

    function showLoading(instance) {
        const { body } = instance.elements;
        body.innerHTML = '<div class="captchax-loading" role="status" aria-label="加载中">' +
            '<div class="captchax-loading-spinner" aria-hidden="true">' +
            '<div class="captchax-spinner-ring"></div>' +
            '<div class="captchax-spinner-ring"></div>' +
            '<div class="captchax-spinner-ring"></div>' +
            '</div>' +
            '<span class="captchax-loading-text">' + CaptchaX.config.loadingText + '</span>' +
            '</div>';
        show(body);
    }

    function fetchCaptcha(instance) {
        const type = instance.state.captchaType;
        let url;

        switch (type) {
            case 'slider':
                url = getAbsoluteUrl(`/api/captcha/slider?app_id=${instance.config.appId || ''}`);
                break;
            case 'click':
                url = getAbsoluteUrl(`/api/captcha/click?app_id=${instance.config.appId || ''}&char_count=4`);
                break;
            case 'rotate':
                url = getAbsoluteUrl(`/api/captcha/rotate?app_id=${instance.config.appId || ''}`);
                break;
            default:
                url = getAbsoluteUrl(`/api/captcha/slider?app_id=${instance.config.appId || ''}`);
        }

        return request(url, { method: 'GET' });
    }

    function renderCaptcha(instance, data) {
        const { body } = instance.elements;

        switch (instance.state.captchaType) {
            case 'slider':
                renderSlider(instance, data, body);
                break;
            case 'click':
                renderClick(instance, data, body);
                break;
            case 'rotate':
                renderRotate(instance, data, body);
                break;
            default:
                renderSlider(instance, data, body);
        }
    }

    function renderSlider(instance, data, container) {
        container.innerHTML = '';

        const sliderContainer = createElement('div', 'captchax-slider-container');
        const bgWrapper = createElement('div', 'captchax-slider-bg-wrapper');
        const background = createElement('img', 'captchax-slider-bg', {
            src: `data:image/png;base64,${data.background_b64}`,
            alt: '验证码背景图片',
            draggable: 'false',
            role: 'img'
        });
        const puzzleWrapper = createElement('div', 'captchax-slider-puzzle-wrapper');
        const slider = createElement('img', 'captchax-slider-puzzle', {
            src: `data:image/png;base64,${data.slider_b64}`,
            alt: '拖动滑块',
            draggable: 'false'
        });
        const sliderTrack = createElement('div', 'captchax-slider-track');
        const sliderBar = createElement('div', 'captchax-slider-bar');
        const sliderProgress = createElement('div', 'captchax-slider-progress');
        const trackPattern = createElement('div', 'captchax-track-pattern');
        const sliderThumb = createElement('div', 'captchax-slider-thumb', {
            role: 'slider',
            'aria-label': '拖动滑块完成验证',
            'aria-valuemin': 0,
            'aria-valuemax': 100,
            'aria-valuenow': 0,
            tabindex: 0
        });
        const sliderArrow = createElement('div', 'captchax-slider-arrow');
        sliderArrow.innerHTML = '<svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><path d="M6 0L12 6L6 12L0 6L6 0Z"/></svg>';
        const sliderTip = createElement('span', 'captchax-slider-tip');
        const message = createElement('div', 'captchax-message', {
            role: 'alert',
            'aria-live': 'assertive'
        });

        sliderTip.innerHTML = '<span class="captchax-tip-icon">🔓</span><span class="captchax-tip-text">拖动滑块完成验证</span>';
        sliderThumb.appendChild(sliderArrow);
        sliderBar.appendChild(sliderProgress);
        sliderBar.appendChild(trackPattern);
        sliderBar.appendChild(sliderThumb);
        sliderTrack.appendChild(sliderBar);

        bgWrapper.appendChild(background);
        puzzleWrapper.appendChild(slider);
        sliderContainer.appendChild(bgWrapper);
        sliderContainer.appendChild(puzzleWrapper);
        sliderContainer.appendChild(sliderTrack);
        sliderContainer.appendChild(sliderTip);
        container.appendChild(sliderContainer);
        container.appendChild(message);

        instance.elements.sliderContainer = sliderContainer;
        instance.elements.sliderBg = background;
        instance.elements.sliderImg = slider;
        instance.elements.sliderTrack = sliderTrack;
        instance.elements.sliderBar = sliderBar;
        instance.elements.sliderProgress = sliderProgress;
        instance.elements.sliderThumb = sliderThumb;
        instance.elements.message = message;

        initSliderInteraction(instance, data);
    }

    function initSliderInteraction(instance, data) {
        const { sliderImg, sliderThumb, sliderTrack, sliderProgress, message } = instance.elements;

        if (!sliderThumb) return;

        let isDragging = false;
        let startX = 0;
        let currentX = 0;
        let track = [];
        let startTime = 0;

        const sliderSize = instance.config.sliderSize || 44;
        const maxX = sliderTrack.offsetWidth - sliderSize;

        function getClientX(e) {
            if (e.touches && e.touches.length > 0) {
                return e.touches[0].clientX;
            }
            return e.clientX;
        }

        function onMouseDown(e) {
            if (instance.state.verified) return;

            e.preventDefault();
            isDragging = true;
            startX = getClientX(e);
            startTime = Date.now();
            track = [];
            instance.track = [];

            addClass(sliderThumb, 'captchax-dragging');
            addClass(sliderTrack, 'captchax-captcha-area');
            addClass(sliderTrack, 'captchax-area-active');
            document.body.style.userSelect = 'none';
            document.body.style.cursor = 'grabbing';

            sliderThumb.setAttribute('aria-valuenow', 0);
            sliderThumb.focus();
            updateLiveRegion(instance, '开始拖动滑块');
        }

        function onMouseMove(e) {
            if (!isDragging) return;

            const clientX = getClientX(e);
            currentX = clientX - startX;
            currentX = Math.max(0, Math.min(currentX, maxX));

            const percent = Math.round((currentX / maxX) * 100);

            sliderImg.style.left = currentX + 'px';
            sliderThumb.style.left = currentX + 'px';
            sliderProgress.style.width = percent + '%';

            const now = Date.now();
            const timeDiff = now - (track.length > 0 ? track[track.length - 1].t : startTime);
            track.push({
                x: currentX,
                y: 0,
                t: now,
                dt: timeDiff
            });

            instance.track = track;
            sliderThumb.setAttribute('aria-valuenow', percent);
            sliderThumb.setAttribute('aria-valuetext', `进度 ${percent}%`);
        }

        function onMouseUp(e) {
            if (!isDragging) return;

            isDragging = false;
            removeClass(sliderThumb, 'captchax-dragging');
            removeClass(sliderTrack, 'captchax-area-active');
            document.body.style.userSelect = '';
            document.body.style.cursor = '';

            if (track.length > 0) {
                const duration = Date.now() - startTime;
                instance.track = track.map(p => ({
                    x: p.x,
                    y: p.y,
                    t: p.t - startTime,
                    dt: p.dt
                }));
                instance.track.duration = duration;
            }

            updateLiveRegion(instance, '验证中...');
            verifySlider(instance, data, currentX);
        }

        function onKeyDown(e) {
            if (instance.state.verified) return;
            if (document.activeElement !== sliderThumb) return;

            const step = maxX / 20;
            let newX = currentX;

            switch (e.key) {
                case 'ArrowRight':
                case 'ArrowUp':
                    e.preventDefault();
                    newX = Math.min(currentX + step, maxX);
                    break;
                case 'ArrowLeft':
                case 'ArrowDown':
                    e.preventDefault();
                    newX = Math.max(currentX - step, 0);
                    break;
                case 'Home':
                    e.preventDefault();
                    newX = 0;
                    break;
                case 'End':
                    e.preventDefault();
                    newX = maxX;
                    break;
                default:
                    return;
            }

            if (newX !== currentX) {
                currentX = newX;
                const percent = Math.round((currentX / maxX) * 100);

                sliderImg.style.left = currentX + 'px';
                sliderThumb.style.left = currentX + 'px';
                sliderProgress.style.width = percent + '%';

                track.push({
                    x: currentX,
                    y: 0,
                    t: Date.now() - startTime,
                    dt: 50
                });
                instance.track = track;

                sliderThumb.setAttribute('aria-valuenow', percent);
                sliderThumb.setAttribute('aria-valuetext', `进度 ${percent}%`);
            }
        }

        function onKeyUp(e) {
            if (instance.state.verified) return;
            if (document.activeElement !== sliderThumb) return;

            if (e.key === ' ' || e.key === 'Enter') {
                e.preventDefault();
                if (track.length > 0) {
                    const duration = Date.now() - startTime;
                    instance.track = track.map(p => ({
                        x: p.x,
                        y: p.y,
                        t: p.t,
                        dt: p.dt
                    }));
                    instance.track.duration = duration;
                }
                updateLiveRegion(instance, '验证中...');
                verifySlider(instance, data, currentX);
            }
        }

        sliderThumb.addEventListener('mousedown', onMouseDown);
        sliderThumb.addEventListener('touchstart', onMouseDown, { passive: false });
        sliderThumb.addEventListener('keydown', onKeyDown);
        sliderThumb.addEventListener('keyup', onKeyUp);

        document.addEventListener('mousemove', onMouseMove);
        document.addEventListener('touchmove', onMouseMove, { passive: false });

        document.addEventListener('mouseup', onMouseUp);
        document.addEventListener('touchend', onMouseUp);
        document.addEventListener('touchcancel', onMouseUp);
    }

    function verifySlider(instance, data, position) {
        const { message, sliderThumb, sliderTrack, sliderProgress } = instance.elements;

        message.innerHTML = '<span class="captchax-loading-inline">验证中...</span>';
        message.className = 'captchax-message captchax-message-show';
        show(message);
        removeClass(message, 'captchax-message-success captchax-message-error');

        const verifyData = {
            captcha_id: instance.state.captchaId,
            target_x: Math.round(position),
            target_y: data.target_y || 0,
            track: instance.track
        };

        request(getAbsoluteUrl('/api/captcha/slider/verify'), {
            method: 'POST',
            body: verifyData
        }).then(response => {
            if (response.success) {
                handleSuccess(instance, response);
            } else {
                handleError(instance, response.message || CaptchaX.config.failText);
            }
        }).catch(err => {
            handleError(instance, CaptchaX.config.errorText);
        });
    }

    function renderClick(instance, data, container) {
        container.innerHTML = '';

        const clickContainer = createElement('div', 'captchax-click-container');
        const imageWrapper = createElement('div', 'captchax-click-image-wrapper captchax-captcha-area');
        const image = createElement('img', 'captchax-click-img', {
            src: `data:image/png;base64,${data.image}`,
            alt: '点选验证码图片',
            draggable: 'false',
            role: 'img'
        });
        const clickMarkers = createElement('div', 'captchax-click-markers');
        const instruction = createElement('div', 'captchax-click-instruction');
        const message = createElement('div', 'captchax-message', {
            role: 'alert',
            'aria-live': 'assertive'
        });
        const clickIndicators = createElement('div', 'captchax-click-indicators');

        const targetChars = data.target_chars || [];
        const instructionText = instruction.innerHTML = `
            <span class="captchax-instruction-text">请依次点击：</span>
            <span class="captchax-target-chars" aria-label="目标字符">${targetChars.join(' ')}</span>
            <button type="button" class="captchax-refresh-btn" title="刷新" aria-label="刷新验证码">
                <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
                    <path d="M14 8c0-3.314-2.686-6-6-6v2l-4-4 4-4v2c4.418 0 8 3.582 8 8s-3.582 8-8 8c-1.5 0-2.898-.42-4.08-1.14l1.44-1.44A5.98 5.98 0 0 0 8 14c3.314 0 6-2.686 6-6z"/>
                    <path d="M2 8c0 1.5.42 2.898 1.14 4.08l-1.44 1.44A5.98 5.98 0 0 0 2 8z"/>
                </svg>
            </button>
        `;

        imageWrapper.appendChild(image);
        imageWrapper.appendChild(clickMarkers);
        clickContainer.appendChild(imageWrapper);
        clickContainer.appendChild(instruction);
        container.appendChild(clickContainer);
        container.appendChild(message);
        container.appendChild(clickIndicators);

        instance.elements.clickContainer = clickContainer;
        instance.elements.clickImage = image;
        instance.elements.clickImageWrapper = imageWrapper;
        instance.elements.clickInstruction = instruction;
        instance.elements.clickIndicators = clickIndicators;
        instance.elements.message = message;

        updateLiveRegion(instance, `点选验证：请依次点击 ${targetChars.join('、')}`);
        initClickInteraction(instance, data);
    }

    function initClickInteraction(instance, data) {
        const { clickImage, clickImageWrapper, clickIndicators, clickInstruction, message } = instance.elements;

        if (!clickImage) return;

        let clicks = [];
        let clickElements = [];
        let rippleElements = [];
        const targetChars = data.target_chars || [];
        const totalClicks = targetChars.length;
        const startTime = Date.now();

        instance.track = [];

        function updateInstruction() {
            const clickedChars = targetChars.slice(0, clicks.length);
            const remainingChars = targetChars.slice(clicks.length);
            const charsContainer = clickInstruction.querySelector('.captchax-target-chars');

            if (charsContainer) {
                charsContainer.innerHTML = clickedChars.map(c => `<span class="captchax-chars-clicked">${c}</span>`).join(' ') + ' ' + remainingChars.join(' ');
            }

            const progressFill = instance.container.querySelector('.captchax-progress-fill');
            const clickedCount = instance.container.querySelector('.captchax-clicked-count');

            if (progressFill) {
                const progress = totalClicks > 0 ? (clicks.length / totalClicks) * 100 : 0;
                progressFill.style.width = progress + '%';
            }

            if (clickedCount) {
                clickedCount.textContent = clicks.length;
            }
        }

        function createRipple(x, y) {
            const ripple = document.createElement('div');
            ripple.className = 'captchax-ripple';
            ripple.style.left = x + 'px';
            ripple.style.top = y + 'px';
            clickIndicators.appendChild(ripple);
            rippleElements.push(ripple);
            setTimeout(() => {
                ripple.remove();
                rippleElements = rippleElements.filter(r => r !== ripple);
            }, 600);
        }

        function onImageClick(e) {
            if (instance.state.verified) return;
            if (clicks.length >= totalClicks) return;

            const rect = clickImage.getBoundingClientRect();
            const scaleX = clickImage.naturalWidth / rect.width;
            const scaleY = clickImage.naturalHeight / rect.height;

            const x = Math.round((e.clientX - rect.left) * scaleX);
            const y = Math.round((e.clientY - rect.top) * scaleY);
            const time = Date.now() - startTime;

            clicks.push({ x, y, t: time });
            instance.track = clicks;

            const indicator = document.createElement('span');
            indicator.className = 'captchax-click-indicator';
            indicator.textContent = clicks.length;
            indicator.style.left = (e.clientX - rect.left) + 'px';
            indicator.style.top = (e.clientY - rect.top) + 'px';
            indicator.setAttribute('role', 'img');
            indicator.setAttribute('aria-label', `第${clicks.length}个点击位置`);
            clickIndicators.appendChild(indicator);
            clickElements.push(indicator);

            createRipple(e.clientX - rect.left, e.clientY - rect.top);

            requestAnimationFrame(() => {
                addClass(indicator, 'captchax-click-indicator-animate');
            });

            updateInstruction();
            updateLiveRegion(instance, `已点击 ${clicks.length} 个，还需点击 ${totalClicks - clicks.length} 个`);

            if (clicks.length === totalClicks) {
                addClass(clickImageWrapper, 'captchax-area-active');
                verifyClick(instance, data, clicks);
            }
        }

        clickImage.addEventListener('click', onImageClick);

        const refreshBtn = clickInstruction.querySelector('.captchax-refresh-btn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                instance.state.captchaId = null;
                clicks = [];
                clickElements.forEach(el => el.remove());
                clickElements = [];
                rippleElements.forEach(el => el.remove());
                rippleElements = [];
                instance.track = [];
                showLoading(instance);
                fetchCaptcha(instance).then(captchaData => {
                    instance.state.captchaId = captchaData.id;
                    renderCaptcha(instance, captchaData);
                }).catch(err => {
                    handleError(instance, err.message);
                });
            });
        }

        instance.clickStartTime = startTime;
    }

    function verifyClick(instance, data, clicks) {
        const { message, clickImageWrapper } = instance.elements;

        message.innerHTML = '<span class="captchax-loading-inline">验证中...</span>';
        message.className = 'captchax-message captchax-message-show';
        show(message);
        removeClass(message, 'captchax-message-success captchax-message-error');
        removeClass(clickImageWrapper, 'captchax-area-active');

        const verifyData = {
            captcha_id: instance.state.captchaId,
            clicks: clicks,
            click_count: clicks.length,
            target_chars: data.target_chars || [],
            duration: Date.now() - (instance.clickStartTime || Date.now())
        };

        request(getAbsoluteUrl('/api/captcha/click/verify'), {
            method: 'POST',
            body: verifyData
        }).then(response => {
            if (response.success) {
                handleSuccess(instance, response);
            } else {
                handleError(instance, response.message || CaptchaX.config.failText);
            }
        }).catch(err => {
            handleError(instance, CaptchaX.config.errorText);
        });
    }

    function renderRotate(instance, data, container) {
        container.innerHTML = '';

        const rotateContainer = createElement('div', 'captchax-rotate-container');
        const imageWrapper = createElement('div', 'captchax-rotate-image-wrapper captchax-captcha-area');
        const image = createElement('img', 'captchax-rotate-img', {
            src: `data:image/png;base64,${data.image}`,
            alt: '旋转验证码图片',
            draggable: 'false',
            role: 'img'
        });
        const handle = createElement('div', 'captchax-rotate-handle', {
            role: 'slider',
            'aria-label': '旋转角度',
            'aria-valuemin': 0,
            'aria-valuemax': 360,
            'aria-valuenow': 0,
            tabindex: 0
        });
        const control = createElement('div', 'captchax-rotate-controls');
        const sliderWrapper = createElement('div', 'captchax-rotate-slider-wrapper');
        const slider = createElement('input', 'captchax-rotate-slider', {
            type: 'range',
            min: '0',
            max: '360',
            value: '0',
            'aria-label': '旋转角度滑块'
        });
        const valueDisplay = createElement('div', 'captchax-rotate-value-display');
        const valueSpan = createElement('span', 'captchax-rotate-value');
        valueSpan.textContent = '0';
        const unitSpan = createElement('span', 'captchax-rotate-unit');
        unitSpan.textContent = '°';
        const verifyBtn = createElement('button', 'captchax-rotate-verify-btn');
        verifyBtn.textContent = '确认旋转';
        verifyBtn.setAttribute('aria-label', '确认旋转角度');
        const message = createElement('div', 'captchax-message', {
            role: 'alert',
            'aria-live': 'assertive'
        });

        valueDisplay.appendChild(valueSpan);
        valueDisplay.appendChild(unitSpan);
        sliderWrapper.appendChild(slider);
        sliderWrapper.appendChild(valueDisplay);
        control.appendChild(sliderWrapper);
        control.appendChild(verifyBtn);

        imageWrapper.appendChild(image);
        imageWrapper.appendChild(handle);

        rotateContainer.appendChild(imageWrapper);
        rotateContainer.appendChild(control);
        container.appendChild(rotateContainer);
        container.appendChild(message);

        instance.elements.rotateContainer = rotateContainer;
        instance.elements.rotateImage = image;
        instance.elements.rotateImageWrapper = imageWrapper;
        instance.elements.rotateHandle = handle;
        instance.elements.rotateSlider = slider;
        instance.elements.rotateValue = valueSpan;
        instance.elements.rotateVerifyBtn = verifyBtn;
        instance.elements.message = message;

        updateLiveRegion(instance, '旋转验证：将图片旋转到正确角度');
        initRotateInteraction(instance, data);
    }

    function initRotateInteraction(instance, data) {
        const { rotateImage, rotateHandle, rotateSlider, rotateValue, rotateVerifyBtn, message } = instance.elements;

        if (!rotateSlider) return;

        let currentAngle = 0;
        let isDragging = false;
        let startAngle = 0;
        let startX = 0;

        function updateRotation(angle) {
            currentAngle = angle % 360;
            if (currentAngle < 0) currentAngle += 360;
            rotateImage.style.transform = `rotate(${currentAngle}deg)`;
            rotateSlider.value = currentAngle;
            rotateValue.textContent = Math.round(currentAngle);
            rotateHandle.setAttribute('aria-valuenow', Math.round(currentAngle));
            rotateHandle.setAttribute('aria-valuetext', `${Math.round(currentAngle)}度`);
        }

        function onSliderInput(e) {
            updateRotation(parseInt(e.target.value, 10));
        }

        function onHandleMouseDown(e) {
            e.preventDefault();
            isDragging = true;
            startX = e.type.includes('touch') ? e.touches[0].clientX : e.clientX;
            startAngle = currentAngle;
            addClass(rotateHandle, 'captchax-dragging');
            document.body.style.cursor = 'grabbing';
        }

        function onHandleMouseMove(e) {
            if (!isDragging) return;

            const clientX = e.type.includes('touch') ? e.touches[0].clientX : e.clientX;
            const deltaX = clientX - startX;
            const newAngle = startAngle + deltaX * 0.5;
            updateRotation(newAngle);
        }

        function onHandleMouseUp() {
            isDragging = false;
            removeClass(rotateHandle, 'captchax-dragging');
            document.body.style.cursor = '';
        }

        function onKeyDown(e) {
            if (document.activeElement !== rotateHandle && document.activeElement !== rotateSlider) return;

            if (e.key === 'ArrowRight' || e.key === 'ArrowUp') {
                e.preventDefault();
                updateRotation(currentAngle + 5);
            } else if (e.key === 'ArrowLeft' || e.key === 'ArrowDown') {
                e.preventDefault();
                updateRotation(currentAngle - 5);
            } else if (e.key === 'Home') {
                e.preventDefault();
                updateRotation(0);
            } else if (e.key === 'End') {
                e.preventDefault();
                updateRotation(360);
            }
        }

        function onVerifyClick() {
            if (instance.state.verified) return;

            message.innerHTML = '<span class="captchax-loading-inline">验证中...</span>';
            message.className = 'captchax-message captchax-message-show';
            show(message);
            removeClass(message, 'captchax-message-success captchax-message-error');

            updateLiveRegion(instance, '验证中...');

            const verifyData = {
                captcha_id: instance.state.captchaId,
                angle: Math.round(currentAngle)
            };

            request(getAbsoluteUrl('/api/captcha/rotate/verify'), {
                method: 'POST',
                body: verifyData
            }).then(response => {
                if (response.success) {
                    handleSuccess(instance, response);
                } else {
                    handleError(instance, response.message || CaptchaX.config.failText);
                }
            }).catch(() => {
                handleError(instance, CaptchaX.config.errorText);
            });
        }

        rotateSlider.addEventListener('input', onSliderInput);
        rotateHandle.addEventListener('mousedown', onHandleMouseDown);
        rotateHandle.addEventListener('touchstart', onHandleMouseDown, { passive: false });
        document.addEventListener('mousemove', onHandleMouseMove);
        document.addEventListener('touchmove', onHandleMouseMove, { passive: false });
        document.addEventListener('mouseup', onHandleMouseUp);
        document.addEventListener('touchend', onHandleMouseUp);
        rotateHandle.addEventListener('keydown', onKeyDown);
        rotateSlider.addEventListener('keydown', onKeyDown);
        rotateVerifyBtn.addEventListener('click', onVerifyClick);

        rotateVerifyBtn.addEventListener('click', () => {
            updateLiveRegion(instance, '已确认旋转角度');
        });
    }

    function handleSuccess(instance, response) {
        const { message, sliderThumb, sliderTrack, sliderProgress, clickImageWrapper, clickElements, rotateVerifyBtn } = instance.elements;

        instance.state.verified = true;

        if (CaptchaX.defaultInstance) {
            CaptchaX.defaultInstance.state.verified = true;
        }

        message.textContent = CaptchaX.config.successText;
        message.className = 'captchax-message captchax-message-show captchax-message-success';
        show(message);

        if (sliderThumb) {
            addClass(sliderThumb, 'captchax-slider-success');
            sliderThumb.style.cursor = 'default';
            sliderThumb.setAttribute('aria-valuetext', '验证成功');
        }

        if (sliderTrack) {
            addClass(sliderTrack, 'captchax-slider-track-success captchax-captcha-area captchax-area-success');
        }

        if (sliderProgress) {
            sliderProgress.style.background = 'linear-gradient(90deg, rgba(82, 196, 26, 0.3) 0%, rgba(82, 196, 26, 0.2) 100%)';
        }

        if (clickImageWrapper) {
            addClass(clickImageWrapper, 'captchax-area-success');
            addClass(clickImageWrapper, 'captchax-click-image-success');
            clickElements.forEach(el => {
                removeClass(el, 'captchax-click-indicator-animate');
                addClass(el, 'captchax-click-indicator-success');
            });
        }

        if (rotateVerifyBtn) {
            addClass(rotateVerifyBtn, 'captchax-rotate-verify-success');
        }

        showSuccessOverlay(instance);
        updateLiveRegion(instance, '验证成功');

        CaptchaX.callbacks.onSuccess.forEach(cb => {
            try {
                cb({
                    captchaId: instance.state.captchaId,
                    token: response.token || instance.state.captchaId,
                    response: response
                });
            } catch (e) {
                console.error('[CaptchaX] onSuccess callback error:', e);
            }
        });

        CaptchaX.callbacks.onVerify.forEach(cb => {
            try {
                cb({
                    success: true,
                    captchaId: instance.state.captchaId,
                    token: response.token || instance.state.captchaId
                });
            } catch (e) {
                console.error('[CaptchaX] onVerify callback error:', e);
            }
        });

        if (instance.config.autoClose !== false) {
            setTimeout(() => {
                if (!instance.state.destroyed) {
                    closeWithAnimation(instance);
                }
            }, 1500);
        }
    }

    function showSuccessOverlay(instance) {
        const overlay = document.createElement('div');
        overlay.className = 'captchax-success-overlay';
        overlay.setAttribute('aria-hidden', 'true');

        const area = instance.elements.sliderContainer || instance.elements.clickContainer || instance.elements.rotateContainer;
        if (area) {
            area.appendChild(overlay);
            setTimeout(() => overlay.remove(), 600);
        }
    }

    function showFailOverlay(instance) {
        const overlay = document.createElement('div');
        overlay.className = 'captchax-fail-overlay';
        overlay.setAttribute('aria-hidden', 'true');

        const area = instance.elements.sliderContainer || instance.elements.clickContainer || instance.elements.rotateContainer;
        if (area) {
            area.appendChild(overlay);
            setTimeout(() => overlay.remove(), 400);
        }
    }

    function handleError(instance, errorMessage) {
        const { message, sliderThumb, sliderTrack, sliderProgress, clickImageWrapper, clickElements, rotateVerifyBtn } = instance.elements;

        message.textContent = errorMessage || CaptchaX.config.failText;
        message.className = 'captchax-message captchax-message-show captchax-message-error';
        show(message);

        if (sliderThumb) {
            addClass(sliderThumb, 'captchax-slider-error');
            setTimeout(() => {
                removeClass(sliderThumb, 'captchax-slider-error');
            }, 500);
        }

        if (sliderTrack) {
            addClass(sliderTrack, 'captchax-captcha-area captchax-area-error');
        }

        if (sliderProgress) {
            sliderProgress.style.background = 'linear-gradient(90deg, rgba(255, 77, 79, 0.3) 0%, rgba(255, 77, 79, 0.2) 100%)';
        }

        if (clickImageWrapper) {
            addClass(clickImageWrapper, 'captchax-area-error');
            addClass(clickImageWrapper, 'captchax-click-image-error');
        }

        if (rotateVerifyBtn) {
            addClass(rotateVerifyBtn, 'captchax-rotate-verify-error');
        }

        showFailOverlay(instance);
        updateLiveRegion(instance, `验证失败: ${errorMessage}`);

        CaptchaX.callbacks.onError.forEach(cb => {
            try {
                cb({
                    captchaId: instance.state.captchaId,
                    error: errorMessage
                });
            } catch (e) {
                console.error('[CaptchaX] onError callback error:', e);
            }
        });

        const isExpired = errorMessage.includes('过期') || errorMessage.includes('expired');
        const isFailed = errorMessage.includes('验证失败') || errorMessage.includes('incorrect');

        if (isExpired || isFailed) {
            setTimeout(() => {
                if (!instance.state.destroyed && !instance.state.verified) {
                    reload(instance);
                }
            }, 1500);
        }
    }

    function closeWithAnimation(instance) {
        const container = instance.container;
        addClass(container, 'closing');

        setTimeout(() => {
            destroyInstance(instance);
        }, 250);
    }

    function renderError(instance, errorMessage) {
        const { body, liveRegion } = instance.elements;

        body.innerHTML = `
            <div class="captchax-error" role="alert">
                <div class="captchax-error-icon" aria-hidden="true">
                    <svg width="48" height="48" viewBox="0 0 24 24" fill="#ff4d4f">
                        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/>
                    </svg>
                </div>
                <span class="captchax-error-text">${errorMessage}</span>
                <button type="button" class="captchax-retry-btn" aria-label="重新加载验证码">重新加载</button>
            </div>
        `;

        const retryBtn = body.querySelector('.captchax-retry-btn');
        if (retryBtn) {
            retryBtn.addEventListener('click', () => {
                reload(instance);
            });
        }

        if (liveRegion) {
            liveRegion.textContent = `错误: ${errorMessage}`;
        }
    }

    function reload(instance) {
        instance.state.captchaId = null;
        instance.state.verified = false;
        instance.track = [];
        showLoading(instance);

        if (instance.elements.sliderThumb) {
            instance.elements.sliderThumb.style.left = '0';
            instance.elements.sliderThumb.className = 'captchax-slider-thumb';
        }

        if (instance.elements.sliderProgress) {
            instance.elements.sliderProgress.style.width = '0';
        }

        fetchCaptcha(instance).then(data => {
            instance.state.captchaId = data.id;
            renderCaptcha(instance, data);
        }).catch(err => {
            renderError(instance, err.message || CaptchaX.config.errorText);
        });
    }

    function destroyInstance(instance) {
        instance.state.destroyed = true;
        CaptchaX.instances.delete(instance.id);

        if (CaptchaX.defaultInstance === instance) {
            const remaining = Array.from(CaptchaX.instances.values());
            CaptchaX.defaultInstance = remaining.length > 0 ? remaining[0] : null;
        }

        if (instance.container) {
            instance.container.innerHTML = '';
            removeClass(instance.container, 'captchax-container captchax-dark captchax-light captchax-auto');
        }
    }

    function bindEvents(instance) {
        const { closeBtn, container } = instance.elements;

        if (closeBtn) {
            closeBtn.addEventListener('click', () => {
                closeWithAnimation(instance);
            });

            closeBtn.addEventListener('keydown', (e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    closeWithAnimation(instance);
                }
            });
        }

        container.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && instance.config.closeOnBackdrop !== false) {
                closeWithAnimation(instance);
            }
        });
    }

    CaptchaX.verify = function(options) {
        if (!CaptchaX.state.ready) {
            return CaptchaX.init().then(() => CaptchaX.verify(options));
        }

        let container;

        if (options && options.container) {
            container = getElement(options.container);
        }

        if (!container) {
            container = createElement('div', 'captchax-overlay');
            document.body.appendChild(container);
        }

        const instanceOptions = extend({}, options, { container: container });
        const instance = CaptchaX.create(instanceOptions);

        return {
            then: (resolve, reject) => {
                const originalOnSuccess = instance.config.onSuccess;
                instance.config.onSuccess = (result) => {
                    if (originalOnSuccess) {
                        try {
                            originalOnSuccess(result);
                        } catch (e) {
                            console.error('[CaptchaX] onSuccess error:', e);
                        }
                    }
                    resolve(result);
                };

                const originalOnError = instance.config.onError;
                instance.config.onError = (error) => {
                    if (originalOnError) {
                        try {
                            originalOnError(error);
                        } catch (e) {
                            console.error('[CaptchaX] onError error:', e);
                        }
                    }
                    if (reject) {
                        reject(error);
                    }
                };

                return instance;
            },
            destroy: () => destroyInstance(instance)
        };
    };

    CaptchaX.onSuccess = function(callback) {
        if (typeof callback === 'function') {
            CaptchaX.callbacks.onSuccess.push(callback);
        }
        return CaptchaX;
    };

    CaptchaX.onError = function(callback) {
        if (typeof callback === 'function') {
            CaptchaX.callbacks.onError.push(callback);
        }
        return CaptchaX;
    };

    CaptchaX.onReady = function(callback) {
        if (typeof callback === 'function') {
            if (CaptchaX.state.ready) {
                try {
                    callback(CaptchaX);
                } catch (e) {
                    console.error('[CaptchaX] onReady callback error:', e);
                }
            } else {
                CaptchaX.callbacks.onReady.push(callback);
            }
        }
        return CaptchaX;
    };

    CaptchaX.onVerify = function(callback) {
        if (typeof callback === 'function') {
            CaptchaX.callbacks.onVerify.push(callback);
        }
        return CaptchaX;
    };

    CaptchaX.offSuccess = function(callback) {
        if (callback) {
            const index = CaptchaX.callbacks.onSuccess.indexOf(callback);
            if (index > -1) {
                CaptchaX.callbacks.onSuccess.splice(index, 1);
            }
        } else {
            CaptchaX.callbacks.onSuccess = [];
        }
        return CaptchaX;
    };

    CaptchaX.offError = function(callback) {
        if (callback) {
            const index = CaptchaX.callbacks.onError.indexOf(callback);
            if (index > -1) {
                CaptchaX.callbacks.onError.splice(index, 1);
            }
        } else {
            CaptchaX.callbacks.onError = [];
        }
        return CaptchaX;
    };

    CaptchaX.offReady = function(callback) {
        if (callback) {
            const index = CaptchaX.callbacks.onReady.indexOf(callback);
            if (index > -1) {
                CaptchaX.callbacks.onReady.splice(index, 1);
            }
        } else {
            CaptchaX.callbacks.onReady = [];
        }
        return CaptchaX;
    };

    CaptchaX.offVerify = function(callback) {
        if (callback) {
            const index = CaptchaX.callbacks.onVerify.indexOf(callback);
            if (index > -1) {
                CaptchaX.callbacks.onVerify.splice(index, 1);
            }
        } else {
            CaptchaX.callbacks.onVerify = [];
        }
        return CaptchaX;
    };

    CaptchaX.destroy = function(instanceId) {
        if (instanceId) {
            const instance = CaptchaX.instances.get(instanceId);
            if (instance) {
                destroyInstance(instance);
            }
        } else {
            CaptchaX.instances.forEach((instance) => {
                destroyInstance(instance);
            });
            CaptchaX.defaultInstance = null;
        }
    };

    CaptchaX.getInstance = function(instanceId) {
        if (instanceId) {
            return CaptchaX.instances.get(instanceId) || null;
        }
        return CaptchaX.defaultInstance;
    };

    CaptchaX.refresh = function(instanceId) {
        const instance = instanceId ? CaptchaX.instances.get(instanceId) : CaptchaX.defaultInstance;
        if (instance && !instance.state.destroyed) {
            reload(instance);
        }
    };

    CaptchaX.reset = function(instanceId) {
        const instance = instanceId ? CaptchaX.instances.get(instanceId) : CaptchaX.defaultInstance;
        if (instance && !instance.state.destroyed) {
            instance.state.verified = false;
            instance.track = [];
            reload(instance);
        }
    };

    const originalCaptchaX = global.CaptchaX;

    CaptchaX.noConflict = function() {
        global.CaptchaX = originalCaptchaX;
        return CaptchaX;
    };

    global.CaptchaX = CaptchaX;

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = CaptchaX;
    }

})(typeof window !== 'undefined' ? window : this);
