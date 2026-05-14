/**
 * CaptchaX Client SDK
 * @version 1.0.0
 * @license MIT
 */
(function(global) {
    'use strict';

    const CaptchaX = {
        version: '1.0.0',
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
            '<button type="button" class="captchax-close">&times;</button>' +
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

        if (instance.config.theme === 'dark') {
            addClass(container, 'captchax-dark');
        } else if (instance.config.theme === 'light') {
            addClass(container, 'captchax-light');
        }

        showLoading(instance);

        fetchCaptcha(instance).then(data => {
            instance.state.captchaId = data.id;
            instance.state.loading = false;
            renderCaptcha(instance, data);
        }).catch(err => {
            instance.state.loading = false;
            renderError(instance, err.message || CaptchaX.config.errorText);
        });
    }

    function showLoading(instance) {
        const { body } = instance.elements;
        body.innerHTML = '<div class="captchax-loading">' +
            '<div class="captchax-spinner"></div>' +
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
        const background = createElement('img', 'captchax-slider-bg', {
            src: `data:image/png;base64,${data.background_b64}`,
            alt: '验证码背景',
            draggable: 'false'
        });
        const slider = createElement('img', 'captchax-slider-img', {
            src: `data:image/png;base64,${data.slider_b64}`,
            alt: '滑块',
            draggable: 'false'
        });
        const sliderTrack = createElement('div', 'captchax-slider-track');
        const sliderBar = createElement('div', 'captchax-slider-bar');
        const sliderThumb = createElement('div', 'captchax-slider-thumb');
        const sliderTip = createElement('span', 'captchax-slider-tip');
        const message = createElement('div', 'captchax-message');

        sliderTip.textContent = '拖动滑块完成验证';
        sliderThumb.appendChild(sliderTip);
        sliderBar.appendChild(sliderThumb);
        sliderTrack.appendChild(sliderBar);

        sliderContainer.appendChild(background);
        sliderContainer.appendChild(slider);
        sliderContainer.appendChild(sliderTrack);
        container.appendChild(sliderContainer);
        container.appendChild(message);

        instance.elements.sliderContainer = sliderContainer;
        instance.elements.sliderBg = background;
        instance.elements.sliderImg = slider;
        instance.elements.sliderTrack = sliderTrack;
        instance.elements.sliderBar = sliderBar;
        instance.elements.sliderThumb = sliderThumb;
        instance.elements.message = message;

        initSliderInteraction(instance, data);
    }

    function initSliderInteraction(instance, data) {
        const { sliderImg, sliderThumb, sliderTrack, message } = instance.elements;

        let isDragging = false;
        let startX = 0;
        let currentX = 0;
        let track = [];
        let startTime = 0;

        const sliderSize = instance.config.sliderSize || 50;
        const maxX = sliderTrack.offsetWidth - sliderSize;

        function onMouseDown(e) {
            if (instance.state.verified) return;

            isDragging = true;
            startX = e.type.includes('touch') ? e.touches[0].clientX : e.clientX;
            startTime = Date.now();
            track = [];
            instance.track = [];

            addClass(sliderThumb, 'captchax-dragging');
            document.body.style.userSelect = 'none';
        }

        function onMouseMove(e) {
            if (!isDragging) return;

            const clientX = e.type.includes('touch') ? e.touches[0].clientX : e.clientX;
            currentX = clientX - startX;
            currentX = Math.max(0, Math.min(currentX, maxX));

            const percent = currentX / maxX;

            sliderImg.style.left = currentX + 'px';
            sliderThumb.style.left = currentX + 'px';

            const now = Date.now();
            const timeDiff = now - (track.length > 0 ? track[track.length - 1].t : startTime);
            track.push({
                x: currentX,
                y: 0,
                t: now,
                dt: timeDiff
            });

            instance.track = track;
        }

        function onMouseUp(e) {
            if (!isDragging) return;

            isDragging = false;
            removeClass(sliderThumb, 'captchax-dragging');
            document.body.style.userSelect = '';

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

            verifySlider(instance, data, currentX);
        }

        sliderThumb.addEventListener('mousedown', onMouseDown);
        sliderThumb.addEventListener('touchstart', onMouseDown, { passive: true });

        document.addEventListener('mousemove', onMouseMove);
        document.addEventListener('touchmove', onMouseMove, { passive: true });

        document.addEventListener('mouseup', onMouseUp);
        document.addEventListener('touchend', onMouseUp);

        sliderThumb.addEventListener('touchcancel', onMouseUp);
    }

    function verifySlider(instance, data, position) {
        const { message } = instance.elements;

        message.innerHTML = '<span class="captchax-loading-inline">验证中...</span>';
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
        const image = createElement('img', 'captchax-click-img', {
            src: `data:image/png;base64,${data.image}`,
            alt: '点选验证码',
            draggable: 'false'
        });
        const instruction = createElement('div', 'captchax-click-instruction');
        const message = createElement('div', 'captchax-message');
        const clickIndicators = createElement('div', 'captchax-click-indicators');

        instruction.innerHTML = `<span>请依次点击：</span><strong class="captchax-target-chars">${(data.target_chars || []).join(' ')}</strong>`;
        instruction.innerHTML += '<button type="button" class="captchax-refresh-btn" title="刷新">&#8635;</button>';

        clickContainer.appendChild(image);
        clickContainer.appendChild(instruction);
        container.appendChild(clickContainer);
        container.appendChild(message);
        container.appendChild(clickIndicators);

        instance.elements.clickContainer = clickContainer;
        instance.elements.clickImage = image;
        instance.elements.clickInstruction = instruction;
        instance.elements.clickIndicators = clickIndicators;
        instance.elements.message = message;

        initClickInteraction(instance, data);
    }

    function initClickInteraction(instance, data) {
        const { clickImage, clickIndicators, clickInstruction, message } = instance.elements;

        let clicks = [];
        let clickElements = [];
        const targetChars = data.target_chars || [];

        if (targetChars.length === 0) {
            message.textContent = '配置错误：未找到目标字符';
            show(message);
            addClass(message, 'captchax-message-error');
            return;
        }

        function onImageClick(e) {
            if (instance.state.verified) return;
            if (clicks.length >= targetChars.length) return;

            const rect = clickImage.getBoundingClientRect();
            const scaleX = clickImage.naturalWidth / rect.width;
            const scaleY = clickImage.naturalHeight / rect.height;

            const x = Math.round((e.clientX - rect.left) * scaleX);
            const y = Math.round((e.clientY - rect.top) * scaleY);

            const time = Date.now() - instance.clickStartTime;

            clicks.push({ x, y, t: time });
            instance.track = clicks;

            const indicator = createElement('span', 'captchax-click-indicator');
            indicator.textContent = clicks.length;
            indicator.style.left = (e.clientX - rect.left) + 'px';
            indicator.style.top = (e.clientY - rect.top) + 'px';
            clickIndicators.appendChild(indicator);
            clickElements.push(indicator);

            addClass(indicator, 'captchax-click-indicator-animate');

            if (clicks.length === targetChars.length) {
                verifyClick(instance, data, clicks);
            }
        }

        instance.clickStartTime = Date.now();
        clickImage.addEventListener('click', onImageClick);

        const refreshBtn = clickInstruction.querySelector('.captchax-refresh-btn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                instance.state.captchaId = null;
                showLoading(instance);
                fetchCaptcha(instance).then(captchaData => {
                    instance.state.captchaId = captchaData.id;
                    clicks = [];
                    clickElements.forEach(el => el.remove());
                    clickElements = [];
                    instance.track = [];
                    renderCaptcha(instance, captchaData);
                }).catch(err => {
                    handleError(instance, err.message);
                });
            });
        }
    }

    function verifyClick(instance, data, clicks) {
        const { message } = instance.elements;

        message.innerHTML = '<span class="captchax-loading-inline">验证中...</span>';
        show(message);
        removeClass(message, 'captchax-message-success captchax-message-error');

        const verifyData = {
            captcha_id: instance.state.captchaId,
            clicks: clicks
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
        const imageWrapper = createElement('div', 'captchax-rotate-image-wrapper');
        const image = createElement('img', 'captchax-rotate-img', {
            src: `data:image/png;base64,${data.image}`,
            alt: '旋转验证码',
            draggable: 'false'
        });
        const handle = createElement('div', 'captchax-rotate-handle');
        const control = createElement('div', 'captchax-rotate-control');
        const slider = createElement('input', 'captchax-rotate-slider', {
            type: 'range',
            min: '0',
            max: '360',
            value: '0'
        });
        const valueDisplay = createElement('span', 'captchax-rotate-value');
        const verifyBtn = createElement('button', 'captchax-rotate-verify-btn', {
            type: 'button'
        });
        const message = createElement('div', 'captchax-message');

        valueDisplay.textContent = '0°';
        verifyBtn.textContent = '确认';

        control.appendChild(slider);
        control.appendChild(valueDisplay);
        control.appendChild(verifyBtn);

        imageWrapper.appendChild(image);
        imageWrapper.appendChild(handle);

        rotateContainer.appendChild(imageWrapper);
        rotateContainer.appendChild(control);
        container.appendChild(rotateContainer);
        container.appendChild(message);

        instance.elements.rotateContainer = rotateContainer;
        instance.elements.rotateImage = image;
        instance.elements.rotateHandle = handle;
        instance.elements.rotateSlider = slider;
        instance.elements.rotateValue = valueDisplay;
        instance.elements.rotateVerifyBtn = verifyBtn;
        instance.elements.message = message;

        initRotateInteraction(instance, data);
    }

    function initRotateInteraction(instance, data) {
        const { rotateImage, rotateSlider, rotateValue, rotateVerifyBtn, message } = instance.elements;

        let currentAngle = 0;
        let isDragging = false;
        let startAngle = 0;

        function updateRotation(angle) {
            currentAngle = angle % 360;
            if (currentAngle < 0) currentAngle += 360;
            rotateImage.style.transform = `rotate(${currentAngle}deg)`;
            rotateSlider.value = currentAngle;
            rotateValue.textContent = `${Math.round(currentAngle)}°`;
        }

        function onSliderInput(e) {
            updateRotation(parseInt(e.target.value, 10));
        }

        function onVerifyClick() {
            if (instance.state.verified) return;

            message.innerHTML = '<span class="captchax-loading-inline">验证中...</span>';
            show(message);

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
        rotateVerifyBtn.addEventListener('click', onVerifyClick);
    }

    function handleSuccess(instance, response) {
        const { message, elements } = instance;

        instance.state.verified = true;

        if (CaptchaX.defaultInstance) {
            CaptchaX.defaultInstance.state.verified = true;
        }

        message.textContent = CaptchaX.config.successText;
        show(message);
        removeClass(message, 'captchax-message-error');
        addClass(message, 'captchax-message-success');

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
                    destroyInstance(instance);
                }
            }, 1500);
        }
    }

    function handleError(instance, errorMessage) {
        const { message, elements } = instance;

        message.textContent = errorMessage || CaptchaX.config.failText;
        show(message);
        removeClass(message, 'captchax-message-success');
        addClass(message, 'captchax-message-error');

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

        const errorType = errorMessage || '';
        if (errorType.includes('过期') || errorType.includes('expired')) {
            setTimeout(() => {
                if (!instance.state.destroyed && !instance.state.verified) {
                    reload(instance);
                }
            }, 2000);
        } else if (errorType.includes('验证失败') || errorType.includes('incorrect')) {
            setTimeout(() => {
                if (!instance.state.destroyed && !instance.state.verified) {
                    reload(instance);
                }
            }, 1500);
        }
    }

    function renderError(instance, message) {
        const { body, elements } = instance;

        body.innerHTML = '<div class="captchax-error">' +
            '<span class="captchax-error-icon">&#9888;</span>' +
            `<span class="captchax-error-text">${message}</span>` +
            '<button type="button" class="captchax-retry-btn">重试</button>' +
            '</div>';

        const retryBtn = body.querySelector('.captchax-retry-btn');
        if (retryBtn) {
            retryBtn.addEventListener('click', () => {
                reload(instance);
            });
        }
    }

    function reload(instance) {
        instance.state.captchaId = null;
        instance.state.verified = false;
        instance.track = [];
        showLoading(instance);

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
            removeClass(instance.container, 'captchax-container captchax-dark captchax-light');
        }
    }

    function bindEvents(instance) {
        const { closeBtn } = instance.elements;

        if (closeBtn) {
            closeBtn.addEventListener('click', () => {
                destroyInstance(instance);
            });
        }

        instance.container.addEventListener('click', (e) => {
            if (e.target === instance.container && instance.config.closeOnBackdrop !== false) {
                destroyInstance(instance);
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
