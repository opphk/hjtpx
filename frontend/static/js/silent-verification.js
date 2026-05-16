class SilentVerification {
    constructor(options = {}) {
        this.options = {
            apiBase: '/api/v1',
            sessionId: this.generateSessionId(),
            userId: null,
            appId: null,
            enableMouseTrack: true,
            enableKeyboardTrack: true,
            enableClickTrack: true,
            enableScrollTrack: true,
            enableTouchTrack: true,
            minDataPoints: 20,
            maxCollectTime: 5000,
            onVerifyStart: null,
            onVerifyComplete: null,
            onCaptchaRequired: null,
            onError: null,
            ...options
        };

        this.behaviorData = [];
        this.startTime = Date.now();
        this.lastMoveTime = Date.now();
        this.moveThrottle = 50;
        this.lastMoveX = 0;
        this.lastMoveY = 0;

        this.keyboardBuffer = [];
        this.lastKeyTime = 0;

        this.scrollData = [];
        this.lastScrollY = 0;

        this.touchData = [];
        this.isCollecting = false;

        this.fingerprint = '';
        this.token = '';
        this.verificationResult = null;

        this.init();
    }

    init() {
        this.generateFingerprint();
        this.startCollection();
        this.bindEvents();
        this.setupAutoVerify();
    }

    generateSessionId() {
        return 'silent_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
    }

    generateFingerprint() {
        const components = [
            navigator.userAgent,
            navigator.language,
            screen.width + 'x' + screen.height,
            screen.colorDepth,
            new Date().getTimezoneOffset(),
            navigator.hardwareConcurrency || 'unknown',
            navigator.platform,
        ];

        if (navigator.plugins) {
            components.push(navigator.plugins.length);
        }

        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');
        if (ctx) {
            ctx.textBaseline = 'top';
            ctx.font = '14px Arial';
            ctx.fillText('silent verification fingerprint', 2, 2);
            components.push(canvas.toDataURL());
        }

        try {
            const keys = [];
            for (let key in localStorage) {
                if (localStorage.hasOwnProperty(key)) {
                    keys.push(key);
                }
            }
            components.push(keys.length);
        } catch (e) {
            components.push('localStorage:not-available');
        }

        try {
            if (sessionStorage) {
                components.push(sessionStorage.length);
            }
        } catch (e) {
            components.push('sessionStorage:not-available');
        }

        this.fingerprint = this.sha256(components.join('|'));
    }

    sha256(str) {
        const utf8 = unescape(encodeURIComponent(str));
        const len = utf8.length;
        const words = [];

        for (let i = 0; i < len; i++) {
            const charCode = utf8.charCodeAt(i);
            if (charCode < 0x80) {
                words.push(charCode);
            } else if (charCode < 0x800) {
                words.push(0xC0 | (charCode >> 6));
                words.push(0x80 | (charCode & 0x3F));
            } else if (charCode < 0xD800 || charCode >= 0xE000) {
                words.push(0xE0 | (charCode >> 12));
                words.push(0x80 | ((charCode >> 6) & 0x3F));
                words.push(0x80 | (charCode & 0x3F));
            } else {
                const cp = ((charCode - 0xD800) * 0x400) + (utf8.charCodeAt(++i) - 0xDC00) + 0x10000;
                words.push(0xF0 | (cp >> 18));
                words.push(0x80 | ((cp >> 12) & 0x3F));
                words.push(0x80 | ((cp >> 6) & 0x3F));
                words.push(0x80 | (cp & 0x3F));
            }
        }

        while (words.length % 16 !== 0) {
            words.push(0);
        }

        const h = [0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
                   0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19];

        const k = [0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5,
                   0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
                   0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
                   0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174];

        for (let i = 0; i < words.length; i += 16) {
            let w = [];
            for (let j = 0; j < 16; j++) {
                w.push(words[i + j]);
            }
            for (let j = 16; j < 64; j++) {
                const s0 = this.rotr(w[j - 15], 7) ^ this.rotr(w[j - 15], 18) ^ (w[j - 15] >>> 3);
                const s1 = this.rotr(w[j - 2], 17) ^ this.rotr(w[j - 2], 19) ^ (w[j - 2] >>> 10);
                w.push((w[j - 16] + s0 + w[j - 7] + s1) >>> 0);
            }

            let a = h[0], b = h[1], c = h[2], d = h[3];
            let e = h[4], f = h[5], g = h[6], hh = h[7];

            for (let j = 0; j < 64; j++) {
                const S1 = this.rotr(e, 6) ^ this.rotr(e, 11) ^ this.rotr(e, 25);
                const ch = (e & f) ^ ((~e) & g);
                const temp1 = (hh + S1 + ch + k[j] + w[j]) >>> 0;
                const S0 = this.rotr(a, 2) ^ this.rotr(a, 13) ^ this.rotr(a, 22);
                const maj = (a & b) ^ (a & c) ^ (b & c);
                const temp2 = (S0 + maj) >>> 0;

                hh = g;
                g = f;
                f = e;
                e = (d + temp1) >>> 0;
                d = c;
                c = b;
                b = a;
                a = (temp1 + temp2) >>> 0;
            }

            h[0] = (h[0] + a) >>> 0;
            h[1] = (h[1] + b) >>> 0;
            h[2] = (h[2] + c) >>> 0;
            h[3] = (h[3] + d) >>> 0;
            h[4] = (h[4] + e) >>> 0;
            h[5] = (h[5] + f) >>> 0;
            h[6] = (h[6] + g) >>> 0;
            h[7] = (h[7] + hh) >>> 0;
        }

        return h.map(v => v.toString(16).padStart(8, '0')).join('');
    }

    rotr(x, n) {
        return (x >>> n) | (x << (32 - n));
    }

    startCollection() {
        this.isCollecting = true;
        this.startTime = Date.now();
        this.behaviorData = [];
        this.keyboardBuffer = [];
        this.scrollData = [];
        this.touchData = [];
    }

    stopCollection() {
        this.isCollecting = false;
    }

    bindEvents() {
        if (this.options.enableMouseTrack) {
            this.bindMouseEvents();
        }
        if (this.options.enableKeyboardTrack) {
            this.bindKeyboardEvents();
        }
        if (this.options.enableClickTrack) {
            this.bindClickEvents();
        }
        if (this.options.enableScrollTrack) {
            this.bindScrollEvents();
        }
        if (this.options.enableTouchTrack) {
            this.bindTouchEvents();
        }

        window.addEventListener('beforeunload', () => {
            this.stopCollection();
        });
    }

    bindMouseEvents() {
        let lastMoveTime = 0;

        document.addEventListener('mousemove', (e) => {
            if (!this.isCollecting) return;

            const now = Date.now();
            if (now - lastMoveTime < this.moveThrottle) return;
            lastMoveTime = now;

            const dx = e.clientX - this.lastMoveX;
            const dy = e.clientY - this.lastMoveY;
            const dt = now - this.lastMoveTime;

            if (dt > 0 && (Math.abs(dx) > 1 || Math.abs(dy) > 1)) {
                this.behaviorData.push({
                    x: e.clientX,
                    y: e.clientY,
                    timestamp: now,
                    event: 'mousemove',
                    dx: dx,
                    dy: dy,
                    dt: dt
                });

                this.lastMoveX = e.clientX;
                this.lastMoveY = e.clientY;
                this.lastMoveTime = now;
            }
        }, { passive: true });
    }

    bindKeyboardEvents() {
        document.addEventListener('keydown', (e) => {
            if (!this.isCollecting) return;

            const now = Date.now();
            const timeSinceLastKey = now - this.lastKeyTime;

            this.behaviorData.push({
                x: 0,
                y: 0,
                timestamp: now,
                event: 'keydown',
                keyCode: e.keyCode,
                key: e.key,
                timeSinceLastKey: timeSinceLastKey,
                ctrlKey: e.ctrlKey,
                shiftKey: e.shiftKey,
                altKey: e.altKey
            });

            this.keyboardBuffer.push({
                key: e.key,
                time: now,
                interval: timeSinceLastKey
            });

            this.lastKeyTime = now;

            if (this.keyboardBuffer.length > 50) {
                this.keyboardBuffer.shift();
            }
        }, { passive: true });

        document.addEventListener('keyup', (e) => {
            if (!this.isCollecting) return;

            this.behaviorData.push({
                x: 0,
                y: 0,
                timestamp: Date.now(),
                event: 'keyup',
                keyCode: e.keyCode,
                key: e.key
            });
        }, { passive: true });
    }

    bindClickEvents() {
        document.addEventListener('mousedown', (e) => {
            if (!this.isCollecting) return;

            this.behaviorData.push({
                x: e.clientX,
                y: e.clientY,
                timestamp: Date.now(),
                event: 'mousedown',
                button: e.button,
                target: e.target.tagName
            });
        }, { passive: true });

        document.addEventListener('mouseup', (e) => {
            if (!this.isCollecting) return;

            this.behaviorData.push({
                x: e.clientX,
                y: e.clientY,
                timestamp: Date.now(),
                event: 'mouseup',
                button: e.button,
                target: e.target.tagName
            });
        }, { passive: true });

        document.addEventListener('click', (e) => {
            if (!this.isCollecting) return;

            this.behaviorData.push({
                x: e.clientX,
                y: e.clientY,
                timestamp: Date.now(),
                event: 'click',
                target: e.target.tagName,
                targetId: e.target.id || '',
                targetClass: e.target.className || ''
            });
        }, { passive: true });
    }

    bindScrollEvents() {
        let lastScrollTime = 0;

        window.addEventListener('scroll', () => {
            if (!this.isCollecting) return;

            const now = Date.now();
            if (now - lastScrollTime < 100) return;
            lastScrollTime = now;

            const currentScrollY = window.scrollY || document.documentElement.scrollTop;
            const scrollDelta = currentScrollY - this.lastScrollY;

            if (Math.abs(scrollDelta) > 0) {
                this.behaviorData.push({
                    x: currentScrollY,
                    y: 0,
                    timestamp: now,
                    event: 'scroll',
                    scrollY: currentScrollY,
                    scrollDelta: scrollDelta,
                    scrollDirection: scrollDelta > 0 ? 'down' : 'up'
                });

                this.scrollData.push({
                    scrollY: currentScrollY,
                    delta: scrollDelta,
                    time: now
                });

                this.lastScrollY = currentScrollY;
            }
        }, { passive: true });

        document.addEventListener('wheel', (e) => {
            if (!this.isCollecting) return;

            this.behaviorData.push({
                x: e.deltaX,
                y: e.deltaY,
                timestamp: Date.now(),
                event: 'wheel',
                deltaX: e.deltaX,
                deltaY: e.deltaY,
                deltaMode: e.deltaMode
            });
        }, { passive: true });
    }

    bindTouchEvents() {
        document.addEventListener('touchstart', (e) => {
            if (!this.isCollecting) return;

            const touch = e.touches[0];
            if (touch) {
                this.behaviorData.push({
                    x: touch.clientX,
                    y: touch.clientY,
                    timestamp: Date.now(),
                    event: 'touchstart',
                    touchCount: e.touches.length
                });

                this.touchData.push({
                    x: touch.clientX,
                    y: touch.clientY,
                    time: Date.now(),
                    type: 'start'
                });
            }
        }, { passive: true });

        document.addEventListener('touchmove', (e) => {
            if (!this.isCollecting) return;

            const touch = e.touches[0];
            if (touch) {
                this.behaviorData.push({
                    x: touch.clientX,
                    y: touch.clientY,
                    timestamp: Date.now(),
                    event: 'touchmove',
                    touchCount: e.touches.length
                });

                this.touchData.push({
                    x: touch.clientX,
                    y: touch.clientY,
                    time: Date.now(),
                    type: 'move'
                });
            }
        }, { passive: true });

        document.addEventListener('touchend', (e) => {
            if (!this.isCollecting) return;

            this.behaviorData.push({
                x: 0,
                y: 0,
                timestamp: Date.now(),
                event: 'touchend',
                touchCount: e.touches.length
            });

            this.touchData.push({
                x: 0,
                y: 0,
                time: Date.now(),
                type: 'end'
            });
        }, { passive: true });
    }

    setupAutoVerify() {
        setTimeout(() => {
            if (this.isCollecting && this.behaviorData.length >= this.options.minDataPoints) {
                this.verify();
            }
        }, this.options.maxCollectTime);
    }

    collectBehaviorData() {
        const processedData = this.processBehaviorData();
        return processedData;
    }

    processBehaviorData() {
        const mouseMoves = this.behaviorData.filter(d => d.event === 'mousemove');
        const clicks = this.behaviorData.filter(d => d.event === 'click');
        const keydowns = this.behaviorData.filter(d => d.event === 'keydown');
        const scrolls = this.behaviorData.filter(d => d.event === 'scroll' || d.event === 'wheel');
        const touches = this.behaviorData.filter(d => d.event.startsWith('touch'));

        const mouseFeatures = this.extractMouseFeatures(mouseMoves);
        const clickFeatures = this.extractClickFeatures(clicks);
        const keyboardFeatures = this.extractKeyboardFeatures(keydowns);
        const scrollFeatures = this.extractScrollFeatures(scrolls);

        return {
            events: this.behaviorData.slice(0, 200),
            features: {
                mouse: mouseFeatures,
                click: clickFeatures,
                keyboard: keyboardFeatures,
                scroll: scrollFeatures,
                touch: {
                    touchCount: touches.length,
                    touchEvents: touches.length
                }
            },
            stats: {
                totalEvents: this.behaviorData.length,
                mouseMoveCount: mouseMoves.length,
                clickCount: clicks.length,
                keydownCount: keydowns.length,
                scrollCount: scrolls.length,
                touchCount: touches.length,
                collectDuration: Date.now() - this.startTime
            }
        };
    }

    extractMouseFeatures(moves) {
        if (moves.length < 2) {
            return { speed: 0, directionChanges: 0, avgSpeed: 0 };
        }

        let totalSpeed = 0;
        let directionChanges = 0;
        let lastAngle = null;

        for (let i = 1; i < moves.length; i++) {
            const dx = moves[i].x - moves[i - 1].x;
            const dy = moves[i].y - moves[i - 1].y;
            const dt = moves[i].timestamp - moves[i - 1].timestamp;

            if (dt > 0) {
                const distance = Math.sqrt(dx * dx + dy * dy);
                const speed = distance / dt * 1000;
                totalSpeed += speed;

                const angle = Math.atan2(dy, dx);
                if (lastAngle !== null) {
                    const angleDiff = Math.abs(angle - lastAngle);
                    if (angleDiff > 0.5 && angleDiff < Math.PI * 2 - 0.5) {
                        directionChanges++;
                    }
                }
                lastAngle = angle;
            }
        }

        return {
            moveCount: moves.length,
            avgSpeed: moves.length > 0 ? totalSpeed / (moves.length - 1) : 0,
            directionChanges: directionChanges,
            lastX: moves.length > 0 ? moves[moves.length - 1].x : 0,
            lastY: moves.length > 0 ? moves[moves.length - 1].y : 0
        };
    }

    extractClickFeatures(clicks) {
        if (clicks.length < 2) {
            return { intervals: [], avgInterval: 0, regularity: 0 };
        }

        const intervals = [];
        for (let i = 1; i < clicks.length; i++) {
            intervals.push(clicks[i].timestamp - clicks[i - 1].timestamp);
        }

        const avgInterval = intervals.reduce((a, b) => a + b, 0) / intervals.length;

        let variance = 0;
        for (const interval of intervals) {
            variance += Math.pow(interval - avgInterval, 2);
        }
        variance /= intervals.length;
        const stdDev = Math.sqrt(variance);

        const regularity = avgInterval > 0 ? 1 - (stdDev / avgInterval) : 0;

        return {
            clickCount: clicks.length,
            intervals: intervals.slice(0, 10),
            avgInterval: avgInterval,
            regularity: Math.max(0, regularity),
            targets: clicks.map(c => c.target).slice(0, 10)
        };
    }

    extractKeyboardFeatures(keydowns) {
        if (keydowns.length < 2) {
            return { intervals: [], avgInterval: 0 };
        }

        const intervals = [];
        for (let i = 1; i < keydowns.length; i++) {
            const interval = keydowns[i].timestamp - keydowns[i - 1].timestamp;
            intervals.push(interval);
        }

        const avgInterval = intervals.reduce((a, b) => a + b, 0) / intervals.length;

        return {
            keydownCount: keydowns.length,
            intervals: intervals.slice(0, 20),
            avgInterval: avgInterval,
            keys: keydowns.map(k => k.key).slice(-20)
        };
    }

    extractScrollFeatures(scrolls) {
        if (scrolls.length === 0) {
            return { scrollCount: 0, totalScroll: 0 };
        }

        let totalScroll = 0;
        for (const scroll of scrolls) {
            totalScroll += Math.abs(scroll.scrollDelta || 0);
        }

        return {
            scrollCount: scrolls.length,
            totalScroll: totalScroll,
            directions: scrolls.filter(s => s.scrollDirection).map(s => s.scrollDirection)
        };
    }

    generateBehaviorSignature() {
        const processed = this.processBehaviorData();

        const signatureData = {
            t: processed.stats.totalEvents,
            m: processed.features.mouse.moveCount || 0,
            c: processed.features.click.clickCount || 0,
            k: processed.features.keyboard.keydownCount || 0,
            s: processed.features.scroll.scrollCount || 0,
            ms: Math.round(processed.features.mouse.avgSpeed * 100) / 100,
            ki: Math.round(processed.features.keyboard.avgInterval),
            ci: Math.round(processed.features.click.avgInterval),
            r: Math.round(processed.features.click.regularity * 100) / 100
        };

        const str = JSON.stringify(signatureData);
        return this.sha256(str);
    }

    async verify() {
        if (this.options.onVerifyStart) {
            this.options.onVerifyStart();
        }

        this.stopCollection();

        const processedData = this.collectBehaviorData();
        const behaviorSignature = this.generateBehaviorSignature();

        const payload = {
            device_fingerprint: this.fingerprint,
            session_id: this.options.sessionId,
            behavior_data: processedData.events,
            timestamp: Date.now(),
            user_id: this.options.userId,
            application_id: this.options.appId,
            signature: behaviorSignature,
            features: processedData.features
        };

        try {
            const response = await fetch(`${this.options.apiBase}/verify/silent`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(payload)
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();

            this.token = result.token;
            this.verificationResult = result;

            if (result.success) {
                if (result.need_captcha) {
                    if (this.options.onCaptchaRequired) {
                        this.options.onCaptchaRequired({
                            type: result.captcha_type,
                            token: result.token,
                            strategy: result.strategy
                        });
                    }
                } else {
                    if (this.options.onVerifyComplete) {
                        this.options.onVerifyComplete({
                            pass: result.pass,
                            token: result.token,
                            riskLevel: result.risk_level
                        });
                    }
                }
            } else {
                throw new Error('Verification failed');
            }

        } catch (error) {
            console.error('Silent verification error:', error);

            if (this.options.onError) {
                this.options.onError(error);
            }

            return this.createFallbackResult();
        }

        return this.verificationResult;
    }

    createFallbackResult() {
        const fallbackResult = {
            pass: true,
            token: 'fallback_' + Date.now(),
            risk_level: 'low',
            need_captcha: false,
            captcha_type: 'none',
            message: '降级验证通过'
        };

        if (this.options.onVerifyComplete) {
            this.options.onVerifyComplete(fallbackResult);
        }

        return fallbackResult;
    }

    async checkStatus(token) {
        if (!token) {
            token = this.token;
        }

        try {
            const response = await fetch(
                `${this.options.apiBase}/verify/silent/status?token=${token}`,
                {
                    method: 'GET',
                    headers: {
                        'Content-Type': 'application/json'
                    }
                }
            );

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            return result;

        } catch (error) {
            console.error('Check status error:', error);
            throw error;
        }
    }

    reset() {
        this.stopCollection();
        this.behaviorData = [];
        this.keyboardBuffer = [];
        this.scrollData = [];
        this.touchData = [];
        this.startTime = Date.now();
        this.lastMoveTime = Date.now();
        this.lastScrollY = 0;
        this.lastKeyTime = 0;
        this.token = '';
        this.verificationResult = null;
        this.options.sessionId = this.generateSessionId();

        this.startCollection();
    }

    getToken() {
        return this.token;
    }

    getFingerprint() {
        return this.fingerprint;
    }

    getBehaviorDataCount() {
        return this.behaviorData.length;
    }

    isReady() {
        return this.behaviorData.length >= this.options.minDataPoints;
    }

    setUserId(userId) {
        this.options.userId = userId;
    }

    setAppId(appId) {
        this.options.appId = appId;
    }

    enable() {
        if (!this.isCollecting) {
            this.startCollection();
        }
    }

    disable() {
        this.stopCollection();
    }

    destroy() {
        this.disable();
        this.behaviorData = null;
        this.keyboardBuffer = null;
        this.scrollData = null;
        this.touchData = null;
    }
}

class SilentVerificationManager {
    constructor() {
        this.instances = new Map();
        this.defaultOptions = {
            apiBase: '/api/v1',
            enableMouseTrack: true,
            enableKeyboardTrack: true,
            enableClickTrack: true,
            enableScrollTrack: true,
            enableTouchTrack: true,
            minDataPoints: 20,
            maxCollectTime: 5000
        };
    }

    create(id, options = {}) {
        const instanceOptions = { ...this.defaultOptions, ...options };
        const instance = new SilentVerification(instanceOptions);
        this.instances.set(id, instance);
        return instance;
    }

    get(id) {
        return this.instances.get(id);
    }

    remove(id) {
        const instance = this.instances.get(id);
        if (instance) {
            instance.destroy();
            this.instances.delete(id);
        }
    }

    getAll() {
        return this.instances;
    }

    clear() {
        for (const [id, instance] of this.instances) {
            instance.destroy();
        }
        this.instances.clear();
    }
}

if (typeof window !== 'undefined') {
    window.SilentVerification = SilentVerification;
    window.SilentVerificationManager = SilentVerificationManager;

    window.silentVerification = {
        manager: new SilentVerificationManager(),

        create: function(options) {
            const id = 'sv_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
            return {
                id: id,
                instance: this.manager.create(id, options)
            };
        },

        quickVerify: async function(options = {}) {
            const { id, instance } = this.create(options);
            try {
                const result = await instance.verify();
                return result;
            } finally {
                setTimeout(() => this.manager.remove(id), 60000);
            }
        }
    };
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = { SilentVerification, SilentVerificationManager };
}
