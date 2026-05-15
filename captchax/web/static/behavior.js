(function(global) {
    'use strict';

    const BehaviorCaptcha = {
        version: '1.0.0',
        config: {
            serverUrl: '',
            appId: '',
            challengeType: 'click_order',
            timeout: 30000,
            maxRetries: 3,
            enableMouseTrack: true,
            enableClickTrack: true,
            enableKeyTrack: false,
            enableScrollTrack: false,
            enableFingerprint: true,
            sampleRate: 50,
            trackInterval: 50
        },
        instances: new Map(),
        defaultInstance: null,
        state: {
            ready: false,
            collecting: false
        },
        behaviorCollector: null
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

    function getAbsoluteUrl(relativePath) {
        if (!relativePath) return '';
        if (relativePath.startsWith('http://') || relativePath.startsWith('https://')) {
            return relativePath;
        }
        const base = BehaviorCaptcha.config.serverUrl.replace(/\/$/, '');
        const path = relativePath.replace(/^\//, '');
        return `${base}/${path}`;
    }

    function request(url, options) {
        const config = extend({
            method: 'GET',
            headers: {
                'Content-Type': 'application/json'
            },
            body: null,
            timeout: BehaviorCaptcha.config.timeout
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

    class BehaviorCollector {
        constructor(options = {}) {
            this.enabled = options.enableMouseTrack !== false;
            this.sampleRate = options.sampleRate || 50;
            this.trackInterval = options.trackInterval || 50;

            this.mouseTracks = [];
            this.clickEvents = [];
            this.keyPressIntervals = [];
            this.scrollPatterns = [];

            this.lastMouseTime = 0;
            this.lastClickTime = 0;
            this.lastKeyTime = 0;
            this.trackCounter = 0;

            this.boundMouseMoveHandler = this.handleMouseMove.bind(this);
            this.boundClickHandler = this.handleClick.bind(this);
            this.boundKeyHandler = this.handleKey.bind(this);
            this.boundScrollHandler = this.handleScroll.bind(this);
        }

        start() {
            if (!this.enabled) return;

            this.mouseTracks = [];
            this.clickEvents = [];
            this.keyPressIntervals = [];
            this.scrollPatterns = [];
            this.lastMouseTime = Date.now();
            this.lastClickTime = Date.now();
            this.lastKeyTime = Date.now();
            this.trackCounter = 0;

            document.addEventListener('mousemove', this.boundMouseMoveHandler, { passive: true });
            document.addEventListener('click', this.boundClickHandler, { passive: true });
            document.addEventListener('keydown', this.boundKeyHandler, { passive: true });
            document.addEventListener('scroll', this.boundScrollHandler, { passive: true });
        }

        stop() {
            document.removeEventListener('mousemove', this.boundMouseMoveHandler);
            document.removeEventListener('click', this.boundClickHandler);
            document.removeEventListener('keydown', this.boundKeyHandler);
            document.removeEventListener('scroll', this.boundScrollHandler);
        }

        handleMouseMove(e) {
            this.trackCounter++;
            if (this.trackCounter % this.sampleRate !== 0) return;

            const now = Date.now();
            const timeDiff = now - this.lastMouseTime;
            this.lastMouseTime = now;

            if (timeDiff <= 0) return;

            const dx = e.movementX || 0;
            const dy = e.movementY || 0;
            const distance = Math.sqrt(dx * dx + dy * dy);
            const velocity = distance / timeDiff;

            this.mouseTracks.push({
                x: e.clientX,
                y: e.clientY,
                timestamp: now,
                velocity: velocity
            });

            if (this.mouseTracks.length > 500) {
                this.mouseTracks = this.mouseTracks.slice(-500);
            }
        }

        handleClick(e) {
            const now = Date.now();
            const timeDiff = now - this.lastClickTime;
            this.lastClickTime = now;

            this.clickEvents.push({
                x: e.clientX,
                y: e.clientY,
                timestamp: now,
                duration: timeDiff,
                pressure: e.pressure || 0.5
            });

            if (this.clickEvents.length > 50) {
                this.clickEvents = this.clickEvents.slice(-50);
            }
        }

        handleKey(e) {
            const now = Date.now();
            const timeDiff = now - this.lastKeyTime;
            this.lastKeyTime = now;

            if (timeDiff > 0 && timeDiff < 10000) {
                this.keyPressIntervals.push(timeDiff);

                if (this.keyPressIntervals.length > 100) {
                    this.keyPressIntervals = this.keyPressIntervals.slice(-100);
                }
            }
        }

        handleScroll(e) {
            const now = Date.now();

            this.scrollPatterns.push({
                x: window.scrollX || window.pageXOffset,
                y: window.scrollY || window.pageYOffset,
                timestamp: now,
                deltaY: e.deltaY || 0
            });

            if (this.scrollPatterns.length > 100) {
                this.scrollPatterns = this.scrollPatterns.slice(-100);
            }
        }

        getData() {
            return {
                mouse_tracks: this.mouseTracks,
                click_events: this.clickEvents,
                key_press_intervals: this.keyPressIntervals,
                scroll_patterns: this.scrollPatterns,
                fingerprint: this.generateFingerprint()
            };
        }

        generateFingerprint() {
            const components = [];

            components.push(navigator.userAgent);
            components.push(navigator.language);
            components.push(screen.width);
            components.push(screen.height);
            components.push(screen.colorDepth);
            components.push(new Date().getTimezoneOffset());

            if (navigator.plugins) {
                components.push(navigator.plugins.length);
            }

            if (window.devicePixelRatio) {
                components.push(window.devicePixelRatio);
            }

            const canvas = document.createElement('canvas');
            const ctx = canvas.getContext('2d');
            if (ctx) {
                ctx.textBaseline = 'top';
                ctx.font = '14px Arial';
                ctx.fillStyle = '#f60';
                ctx.fillRect(125, 1, 62, 20);
                ctx.fillStyle = '#069';
                ctx.fillText('CaptchaX Fingerprint', 2, 15);
                ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
                ctx.fillText('CaptchaX Fingerprint', 4, 17);
                components.push(canvas.toDataURL());
            }

            let hash = 0;
            const str = components.join('###');
            for (let i = 0; i < str.length; i++) {
                const char = str.charCodeAt(i);
                hash = ((hash << 5) - hash) + char;
                hash = hash & hash;
            }

            return Math.abs(hash).toString(36);
        }

        clear() {
            this.mouseTracks = [];
            this.clickEvents = [];
            this.keyPressIntervals = [];
            this.scrollPatterns = [];
        }
    }

    BehaviorCaptcha.BehaviorCollector = BehaviorCollector;

    BehaviorCaptcha.init = function(options) {
        return new Promise((resolve, reject) => {
            const config = deepMerge(BehaviorCaptcha.config, options || {});

            if (!config.serverUrl) {
                const scripts = document.getElementsByTagName('script');
                for (let i = scripts.length - 1; i >= 0; i--) {
                    const src = scripts[i].getAttribute('src');
                    if (src && src.includes('behavior')) {
                        config.serverUrl = src.replace(/\/[^/]+\.js$/, '');
                        break;
                    }
                }
                if (!config.serverUrl) {
                    config.serverUrl = window.location.origin;
                }
            }

            BehaviorCaptcha.config = config;
            BehaviorCaptcha.state.ready = true;

            BehaviorCaptcha.behaviorCollector = new BehaviorCollector(config);

            resolve(BehaviorCaptcha);
        });
    };

    BehaviorCaptcha.create = function(options) {
        if (!BehaviorCaptcha.state.ready) {
            console.warn('[BehaviorCaptcha] SDK not initialized, calling init() automatically');
            return BehaviorCaptcha.init().then(() => BehaviorCaptcha.create(options));
        }

        const instanceId = generateUUID();
        const container = typeof options.container === 'string' 
            ? document.querySelector(options.container) 
            : options.container;

        if (!container) {
            throw new Error('[BehaviorCaptcha] Container element not found');
        }

        const instance = {
            id: instanceId,
            container: container,
            config: extend({}, BehaviorCaptcha.config, options || {}),
            state: {
                token: null,
                challengeType: options.challengeType || BehaviorCaptcha.config.challengeType,
                verified: false,
                loading: false,
                destroyed: false
            },
            data: {
                clicks: [],
                dragPath: [],
                hoverSequence: []
            },
            collector: null,
            elements: {}
        };

        container.innerHTML = '';
        container.className = 'behavior-captcha-container';

        BehaviorCaptcha.instances.set(instanceId, instance);
        BehaviorCaptcha.defaultInstance = instance;

        instance.collector = new BehaviorCollector(instance.config);
        instance.collector.start();

        renderWidget(instance).then(() => {
            bindEvents(instance);
        }).catch(err => {
            console.error('[BehaviorCaptcha] Failed to render widget:', err);
            renderError(instance, err.message);
        });

        return instance;
    };

    function renderWidget(instance) {
        return new Promise((resolve, reject) => {
            instance.state.loading = true;
            showLoading(instance);

            fetchCaptcha(instance).then(data => {
                instance.state.token = data.token;
                instance.state.challengeType = data.challenge_type;
                instance.state.loading = false;
                renderCaptcha(instance, data);
                resolve();
            }).catch(err => {
                instance.state.loading = false;
                renderError(instance, err.message || 'Failed to load captcha');
                reject(err);
            });
        });
    }

    function showLoading(instance) {
        const { body } = instance.elements;
        if (!body) return;

        body.innerHTML = `
            <div class="behavior-captcha-loading" role="status" aria-label="加载中">
                <div class="behavior-captcha-spinner"></div>
                <span class="behavior-captcha-loading-text">加载中...</span>
            </div>
        `;
    }

    function fetchCaptcha(instance) {
        const challengeType = instance.state.challengeType;
        const url = getAbsoluteUrl(`/api/v2/captcha/behavior`);

        return request(url, {
            method: 'POST',
            body: {
                challenge_type: challengeType
            }
        });
    }

    function renderCaptcha(instance, data) {
        const { body } = instance.elements;
        if (!body) return;

        const guideLabels = data.guide_points || [];
        const labelsHtml = guideLabels.length > 0
            ? `<div class="behavior-captcha-labels">
                 ${guideLabels.map(gp => `<span class="behavior-captcha-label" style="left:${gp.x}px;top:${gp.y}px">${gp.label}</span>`).join('')}
               </div>`
            : '';

        body.innerHTML = `
            <div class="behavior-captcha-widget">
                <div class="behavior-captcha-header">
                    <span class="behavior-captcha-title">行为验证</span>
                    <button type="button" class="behavior-captcha-refresh" aria-label="刷新">↻</button>
                </div>
                <div class="behavior-captcha-body">
                    <div class="behavior-captcha-image-wrapper behavior-captcha-captcha-area">
                        <img class="behavior-captcha-image" 
                             src="data:image/png;base64,${data.image_b64}" 
                             alt="验证码图片"
                             draggable="false" />
                        <div class="behavior-captcha-overlay"></div>
                        ${labelsHtml}
                    </div>
                    <div class="behavior-captcha-instruction">
                        <span class="behavior-captcha-instruction-text">${getInstructionText(data.challenge_type)}</span>
                        <span class="behavior-captcha-progress">0/${data.target_count}</span>
                    </div>
                </div>
                <div class="behavior-captcha-message" role="alert" aria-live="assertive"></div>
            </div>
        `;

        instance.elements.widget = body.querySelector('.behavior-captcha-widget');
        instance.elements.image = body.querySelector('.behavior-captcha-image');
        instance.elements.imageWrapper = body.querySelector('.behavior-captcha-image-wrapper');
        instance.elements.overlay = body.querySelector('.behavior-captcha-overlay');
        instance.elements.labelsContainer = body.querySelector('.behavior-captcha-labels');
        instance.elements.instruction = body.querySelector('.behavior-captcha-instruction');
        instance.elements.progress = body.querySelector('.behavior-captcha-progress');
        instance.elements.message = body.querySelector('.behavior-captcha-message');
        instance.elements.refreshBtn = body.querySelector('.behavior-captcha-refresh');

        initInteraction(instance, data);
    }

    function getInstructionText(challengeType) {
        switch (challengeType) {
            case 'click_order':
                return '请按顺序点击圆圈';
            case 'drag_path':
                return '请按顺序拖动经过圆圈';
            case 'hover_sequence':
                return '请按顺序悬停圆圈';
            default:
                return '请完成验证';
        }
    }

    function initInteraction(instance, data) {
        const { imageWrapper, image, overlay } = instance.elements;
        if (!imageWrapper) return;

        instance.data.clicks = [];
        instance.data.dragPath = [];
        instance.data.hoverSequence = [];

        const targetCount = data.target_count || 4;
        const guidePoints = data.guide_points || [];
        let clickIndex = 0;
        const startTime = Date.now();

        function updateProgress() {
            const count = instance.data.clicks.length;
            if (instance.elements.progress) {
                instance.elements.progress.textContent = `${count}/${targetCount}`;
            }

            guidePoints.forEach((gp, idx) => {
                if (idx < count) {
                    const label = instance.elements.labelsContainer?.querySelector(
                        `.behavior-captcha-label:nth-child(${idx + 1})`
                    );
                    if (label) {
                        label.classList.add('behavior-captcha-label-completed');
                    }
                }
            });
        }

        function onImageClick(e) {
            if (instance.state.verified || instance.state.loading) return;
            if (clickIndex >= targetCount) return;

            const rect = image.getBoundingClientRect();
            const scaleX = image.naturalWidth / rect.width;
            const scaleY = image.naturalHeight / rect.height;

            const x = Math.round((e.clientX - rect.left) * scaleX);
            const y = Math.round((e.clientY - rect.top) * scaleY);
            const time = Date.now() - startTime;

            instance.data.clicks.push({
                x: x,
                y: y,
                index: clickIndex,
                time: time
            });

            createClickIndicator(instance, e.clientX - rect.left, e.clientY - rect.top, clickIndex + 1);

            clickIndex++;
            updateProgress();

            if (clickIndex >= targetCount) {
                instance.collector.stop();
                verifyCaptcha(instance, data);
            }
        }

        function onMouseDown(e) {
            if (instance.state.verified || instance.state.loading) return;
            if (instance.state.challengeType !== 'drag_path') return;

            instance.data.dragPath = [{
                x: e.clientX,
                y: e.clientY,
                time: Date.now() - startTime
            }];

            instance.state.dragging = true;
        }

        function onMouseMove(e) {
            if (!instance.state.dragging) return;

            instance.data.dragPath.push({
                x: e.clientX,
                y: e.clientY,
                time: Date.now() - startTime
            });

            createDragTrail(instance, e.clientX, e.clientY);
        }

        function onMouseUp(e) {
            if (!instance.state.dragging) return;

            instance.state.dragging = false;
            instance.collector.stop();

            if (instance.data.dragPath.length >= 5) {
                verifyCaptcha(instance, data);
            } else {
                showError(instance, '请拖动更长的路径');
            }
        }

        if (instance.state.challengeType === 'drag_path') {
            imageWrapper.addEventListener('mousedown', onMouseDown);
            document.addEventListener('mousemove', onMouseMove);
            document.addEventListener('mouseup', onMouseUp);
            imageWrapper.addEventListener('touchstart', onMouseDown, { passive: false });
            document.addEventListener('touchmove', onMouseMove, { passive: false });
            document.addEventListener('touchend', onMouseUp);
        } else {
            image.addEventListener('click', onImageClick);
        }

        if (instance.elements.refreshBtn) {
            instance.elements.refreshBtn.addEventListener('click', () => {
                reload(instance);
            });
        }
    }

    function createClickIndicator(instance, x, y, number) {
        const indicator = document.createElement('div');
        indicator.className = 'behavior-captcha-click-indicator';
        indicator.textContent = number;
        indicator.style.left = x + 'px';
        indicator.style.top = y + 'px';
        instance.elements.overlay.appendChild(indicator);

        setTimeout(() => {
            indicator.remove();
        }, 2000);
    }

    function createDragTrail(instance, x, y) {
        const trail = document.createElement('div');
        trail.className = 'behavior-captcha-drag-trail';
        trail.style.left = x + 'px';
        trail.style.top = y + 'px';
        instance.elements.overlay.appendChild(trail);

        setTimeout(() => {
            trail.remove();
        }, 500);
    }

    function verifyCaptcha(instance, data) {
        showVerifying(instance);

        const behaviorData = instance.collector.getData();

        const verifyData = {
            token: instance.state.token,
            challenge_type: instance.state.challengeType,
            click_sequence: instance.data.clicks,
            drag_path: instance.data.dragPath.map(p => ({
                x: Math.round(p.x),
                y: Math.round(p.y),
                time: p.time
            })),
            behavior_data: behaviorData
        };

        const url = getAbsoluteUrl('/api/v2/captcha/behavior/verify');

        request(url, {
            method: 'POST',
            body: verifyData
        }).then(response => {
            if (response.success) {
                handleSuccess(instance, response);
            } else {
                handleError(instance, response.message || '验证失败');
            }
        }).catch(err => {
            handleError(instance, '网络错误，请重试');
        });
    }

    function showVerifying(instance) {
        if (instance.elements.message) {
            instance.elements.message.textContent = '验证中...';
            instance.elements.message.className = 'behavior-captcha-message behavior-captcha-message-show';
        }
    }

    function handleSuccess(instance, response) {
        instance.state.verified = true;

        if (instance.elements.message) {
            instance.elements.message.textContent = '验证成功';
            instance.elements.message.className = 'behavior-captcha-message behavior-captcha-message-show behavior-captcha-message-success';
        }

        if (instance.elements.imageWrapper) {
            instance.elements.imageWrapper.classList.add('behavior-captcha-success');
        }

        BehaviorCaptcha.callbacks?.onSuccess?.forEach(cb => {
            try {
                cb({
                    token: instance.state.token,
                    score: response.score,
                    response: response
                });
            } catch (e) {
                console.error('[BehaviorCaptcha] onSuccess callback error:', e);
            }
        });

        if (instance.config.autoClose !== false) {
            setTimeout(() => {
                if (!instance.state.destroyed) {
                    closeInstance(instance);
                }
            }, 1500);
        }
    }

    function handleError(instance, errorMessage) {
        if (instance.elements.message) {
            instance.elements.message.textContent = errorMessage;
            instance.elements.message.className = 'behavior-captcha-message behavior-captcha-message-show behavior-captcha-message-error';
        }

        if (instance.elements.imageWrapper) {
            instance.elements.imageWrapper.classList.add('behavior-captcha-error');
            setTimeout(() => {
                instance.elements.imageWrapper?.classList.remove('behavior-captcha-error');
            }, 500);
        }

        BehaviorCaptcha.callbacks?.onError?.forEach(cb => {
            try {
                cb({
                    token: instance.state.token,
                    error: errorMessage
                });
            } catch (e) {
                console.error('[BehaviorCaptcha] onError callback error:', e);
            }
        });

        setTimeout(() => {
            if (!instance.state.destroyed && !instance.state.verified) {
                reload(instance);
            }
        }, 2000);
    }

    function showError(instance, message) {
        if (instance.elements.message) {
            instance.elements.message.textContent = message;
            instance.elements.message.className = 'behavior-captcha-message behavior-captcha-message-show behavior-captcha-message-error';
        }
    }

    function renderError(instance, errorMessage) {
        const { body } = instance.elements;
        if (!body) return;

        body.innerHTML = `
            <div class="behavior-captcha-error" role="alert">
                <div class="behavior-captcha-error-icon">⚠</div>
                <span class="behavior-captcha-error-text">${errorMessage}</span>
                <button type="button" class="behavior-captcha-retry-btn">重新加载</button>
            </div>
        `;

        const retryBtn = body.querySelector('.behavior-captcha-retry-btn');
        if (retryBtn) {
            retryBtn.addEventListener('click', () => {
                reload(instance);
            });
        }
    }

    function reload(instance) {
        instance.state.token = null;
        instance.state.verified = false;
        instance.data = {
            clicks: [],
            dragPath: [],
            hoverSequence: []
        };

        instance.collector.clear();
        instance.collector.start();

        renderWidget(instance).catch(err => {
            console.error('[BehaviorCaptcha] Reload failed:', err);
        });
    }

    function closeInstance(instance) {
        instance.state.destroyed = true;
        instance.collector.stop();
        BehaviorCaptcha.instances.delete(instance.id);

        if (BehaviorCaptcha.defaultInstance === instance) {
            const remaining = Array.from(BehaviorCaptcha.instances.values());
            BehaviorCaptcha.defaultInstance = remaining.length > 0 ? remaining[0] : null;
        }

        if (instance.container) {
            instance.container.innerHTML = '';
            instance.container.className = '';
        }
    }

    function bindEvents(instance) {
        instance.container.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && instance.config.closeOnBackdrop !== false) {
                closeInstance(instance);
            }
        });
    }

    BehaviorCaptcha.verify = function(options) {
        if (!BehaviorCaptcha.state.ready) {
            return BehaviorCaptcha.init().then(() => BehaviorCaptcha.verify(options));
        }

        let container;

        if (options && options.container) {
            container = typeof options.container === 'string'
                ? document.querySelector(options.container)
                : options.container;
        }

        if (!container) {
            container = document.createElement('div');
            document.body.appendChild(container);
        }

        const instanceOptions = extend({}, options, { container: container });
        const instance = BehaviorCaptcha.create(instanceOptions);

        return {
            then: (resolve, reject) => {
                const originalOnSuccess = instance.config.onSuccess;
                instance.config.onSuccess = (result) => {
                    if (originalOnSuccess) {
                        try {
                            originalOnSuccess(result);
                        } catch (e) {
                            console.error('[BehaviorCaptcha] onSuccess error:', e);
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
                            console.error('[BehaviorCaptcha] onError error:', e);
                        }
                    }
                    if (reject) {
                        reject(error);
                    }
                };

                return instance;
            },
            destroy: () => closeInstance(instance)
        };
    };

    BehaviorCaptcha.onSuccess = function(callback) {
        if (typeof callback === 'function') {
            if (!BehaviorCaptcha.callbacks) {
                BehaviorCaptcha.callbacks = { onSuccess: [], onError: [] };
            }
            BehaviorCaptcha.callbacks.onSuccess.push(callback);
        }
        return BehaviorCaptcha;
    };

    BehaviorCaptcha.onError = function(callback) {
        if (typeof callback === 'function') {
            if (!BehaviorCaptcha.callbacks) {
                BehaviorCaptcha.callbacks = { onSuccess: [], onError: [] };
            }
            BehaviorCaptcha.callbacks.onError.push(callback);
        }
        return BehaviorCaptcha;
    };

    BehaviorCaptcha.destroy = function(instanceId) {
        if (instanceId) {
            const instance = BehaviorCaptcha.instances.get(instanceId);
            if (instance) {
                closeInstance(instance);
            }
        } else {
            BehaviorCaptcha.instances.forEach((instance) => {
                closeInstance(instance);
            });
            BehaviorCaptcha.defaultInstance = null;
        }
    };

    BehaviorCaptcha.getInstance = function(instanceId) {
        if (instanceId) {
            return BehaviorCaptcha.instances.get(instanceId) || null;
        }
        return BehaviorCaptcha.defaultInstance;
    };

    BehaviorCaptcha.refresh = function(instanceId) {
        const instance = instanceId ? BehaviorCaptcha.instances.get(instanceId) : BehaviorCaptcha.defaultInstance;
        if (instance && !instance.state.destroyed) {
            reload(instance);
        }
    };

    BehaviorCaptcha.startCollection = function() {
        if (BehaviorCaptcha.behaviorCollector) {
            BehaviorCaptcha.behaviorCollector.start();
            BehaviorCaptcha.state.collecting = true;
        }
    };

    BehaviorCaptcha.stopCollection = function() {
        if (BehaviorCaptcha.behaviorCollector) {
            BehaviorCaptcha.behaviorCollector.stop();
            BehaviorCaptcha.state.collecting = false;
        }
    };

    BehaviorCaptcha.getBehaviorData = function() {
        if (BehaviorCaptcha.behaviorCollector) {
            return BehaviorCaptcha.behaviorCollector.getData();
        }
        return null;
    };

    const originalBehaviorCaptcha = global.BehaviorCaptcha;

    BehaviorCaptcha.noConflict = function() {
        global.BehaviorCaptcha = originalBehaviorCaptcha;
        return BehaviorCaptcha;
    };

    global.BehaviorCaptcha = BehaviorCaptcha;

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = BehaviorCaptcha;
    }

})(typeof window !== 'undefined' ? window : this);
