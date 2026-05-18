class SeamlessV15FingerprintCollector {
    constructor() {
        this.fingerprint = {};
        this.components = [];
        this.cache = new Map();
    }

    async collect() {
        const cached = localStorage.getItem('seamless_v15_fp');
        if (cached) {
            try {
                const parsed = JSON.parse(cached);
                if (parsed.timestamp && Date.now() - parsed.timestamp < 3600000) {
                    this.fingerprint = parsed.fingerprint;
                    this.components = parsed.components;
                    return this.fingerprint;
                }
            } catch (e) {
                console.warn('Invalid fingerprint cache');
            }
        }

        this.components = [];
        this.fingerprint = {};

        await this.collectBasicInfo();
        await this.collectCanvasFingerprint();
        await this.collectWebGLFingerprint();
        await this.collectAudioFingerprint();
        await this.collectFontFingerprint();
        await this.collectTimingFingerprint();
        await this.collectPerformanceFingerprint();
        await this.collectBatteryFingerprint();
        await this.collectNetworkFingerprint();
        await this.collectStorageFingerprint();

        this.fingerprint.hash = await this.generateHash();
        this.fingerprint.timestamp = Date.now();

        this.cacheFingerprint();

        return this.fingerprint;
    }

    async collectBasicInfo() {
        const info = {
            userAgent: navigator.userAgent,
            platform: navigator.platform,
            vendor: navigator.vendor,
            language: navigator.language,
            languages: navigator.languages ? navigator.languages.join(',') : '',
            screenResolution: `${screen.width}x${screen.height}x${screen.colorDepth}`,
            windowSize: `${window.innerWidth}x${window.innerHeight}`,
            devicePixelRatio: window.devicePixelRatio,
            timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
            timezoneOffset: new Date().getTimezoneOffset(),
            doNotTrack: navigator.doNotTrack === '1' || navigator.doNotTrack === 'yes',
            cookiesEnabled: navigator.cookieEnabled,
            javaEnabled: navigator.javaEnabled ? navigator.javaEnabled() : false,
            touchSupport: this.getTouchSupport(),
            hardwareConcurrency: navigator.hardwareConcurrency || 'unknown',
            deviceMemory: navigator.deviceMemory || 'unknown',
            maxTouchPoints: navigator.maxTouchPoints || 0,
        };

        Object.keys(info).forEach(key => {
            this.fingerprint[key] = info[key];
            this.components.push({ key, value: String(info[key]) });
        });
    }

    getTouchSupport() {
        return {
            maxTouchPoints: navigator.maxTouchPoints || 0,
            touchEvent: 'ontouchstart' in window,
            touch: navigator.maxTouchPoints > 0,
            pointerEvent: window.PointerEvent ? true : false,
        };
    }

    async collectCanvasFingerprint() {
        try {
            const canvas = document.createElement('canvas');
            const ctx = canvas.getContext('2d');
            canvas.width = 280;
            canvas.height = 60;

            ctx.textBaseline = 'top';
            ctx.font = "14px 'Arial'";
            ctx.fillStyle = '#f60';
            ctx.fillRect(125, 1, 62, 20);
            ctx.fillStyle = '#069';
            ctx.fillText('SeamlessV15', 2, 15);
            ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
            ctx.fillText('Fingerprint', 4, 17);

            ctx.beginPath();
            ctx.arc(50, 50, 20, 0, Math.PI * 2);
            ctx.strokeStyle = '#f00';
            ctx.stroke();

            const dataUrl = canvas.toDataURL();
            const hash = await this.hashString(dataUrl);

            this.fingerprint.canvasFingerprint = hash;
            this.components.push({ key: 'canvas', value: hash });
        } catch (e) {
            this.fingerprint.canvasFingerprint = 'error';
            this.components.push({ key: 'canvas', value: 'error' });
        }
    }

    async collectWebGLFingerprint() {
        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
            if (!gl) {
                this.fingerprint.webglFingerprint = 'unsupported';
                return;
            }

            const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
            const vendor = debugInfo ? gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL) : 'unknown';
            const renderer = debugInfo ? gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) : 'unknown';

            const webglParams = {
                vendor,
                renderer,
                antialias: gl.getContextAttributes().antialias,
                alpha: gl.getContextAttributes().alpha,
                depth: gl.getContextAttributes().depth,
                stencil: gl.getContextAttributes().stencil,
                maxTextureSize: gl.getParameter(gl.MAX_TEXTURE_SIZE),
                maxViewportDims: gl.getParameter(gl.MAX_VIEWPORT_DIMS).join(','),
            };

            const hash = await this.hashString(JSON.stringify(webglParams));

            this.fingerprint.webglVendor = vendor;
            this.fingerprint.webglRenderer = renderer;
            this.fingerprint.webglFingerprint = hash;
            this.components.push({ key: 'webgl', value: hash });
        } catch (e) {
            this.fingerprint.webglFingerprint = 'error';
            this.components.push({ key: 'webgl', value: 'error' });
        }
    }

    async collectAudioFingerprint() {
        try {
            const audioContext = new (window.AudioContext || window.webkitAudioContext)();
            const oscillator = audioContext.createOscillator();
            const analyser = audioContext.createAnalyser();
            const gainNode = audioContext.createGain();
            const scriptProcessor = audioContext.createScriptProcessor(4096, 1, 1);

            oscillator.type = 'triangle';
            oscillator.frequency.setValueAtTime(10000, audioContext.currentTime);

            gainNode.gain.setValueAtTime(0, audioContext.currentTime);
            oscillator.connect(analyser);
            analyser.connect(scriptProcessor);
            scriptProcessor.connect(gainNode);
            gainNode.connect(audioContext.destination);

            oscillator.start(0);

            const fingerprint = await new Promise((resolve) => {
                scriptProcessor.onaudioprocess = function(event) {
                    const output = event.inputBuffer.getChannelData(0);
                    let sum = 0;
                    for (let i = 0; i < output.length; i++) {
                        sum += Math.abs(output[i]);
                    }
                    oscillator.stop();
                    audioContext.close();
                    resolve(sum.toString());
                };
            });

            this.fingerprint.audioFingerprint = await this.hashString(fingerprint);
            this.components.push({ key: 'audio', value: this.fingerprint.audioFingerprint });
        } catch (e) {
            this.fingerprint.audioFingerprint = 'error';
            this.components.push({ key: 'audio', value: 'error' });
        }
    }

    async collectFontFingerprint() {
        const baseFonts = ['monospace', 'sans-serif', 'serif'];
        const testFonts = [
            'Arial', 'Arial Black', 'Comic Sans MS', 'Courier New', 'Georgia',
            'Impact', 'Times New Roman', 'Trebuchet MS', 'Verdana', 'Palatino',
            'Lucida Console', 'Lucida Sans Unicode', 'Tahoma', 'Geneva', 'Helvetica',
            'Calibri', 'Cambria', 'Consolas', 'Century Gothic', 'Franklin Gothic Medium',
        ];

        const testString = 'mmmmmmmmmmlli';
        const testSize = '72px';

        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');

        const getWidth = (fontFamily) => {
            ctx.font = `${testSize} ${fontFamily}`;
            return ctx.measureText(testString).width;
        };

        const baseWidths = {};
        baseFonts.forEach(font => {
            baseWidths[font] = getWidth(font);
        });

        const detectedFonts = [];
        for (const font of testFonts) {
            for (const baseFont of baseFonts) {
                const testFont = `'${font}', ${baseFont}`;
                const width = getWidth(testFont);
                if (width !== baseWidths[baseFont]) {
                    detectedFonts.push(font);
                    break;
                }
            }
        }

        const hash = await this.hashString(detectedFonts.join(','));
        this.fingerprint.fontFingerprint = hash;
        this.fingerprint.detectedFonts = detectedFonts;
        this.components.push({ key: 'fonts', value: hash });
    }

    async collectTimingFingerprint() {
        const timings = [];

        for (let i = 0; i < 3; i++) {
            const start = performance.now();
            const arr = new Array(1000).fill(0).map((_, idx) => idx * idx);
            const end = performance.now();
            timings.push(end - start);
        }

        this.fingerprint.timingFingerprint = await this.hashString(timings.join(','));
        this.fingerprint.performanceTiming = {
            avg: timings.reduce((a, b) => a + b, 0) / timings.length,
            min: Math.min(...timings),
            max: Math.max(...timings),
        };
        this.components.push({ key: 'timing', value: this.fingerprint.timingFingerprint });
    }

    async collectPerformanceFingerprint() {
        const perfData = {
            memory: performance.memory ? {
                jsHeapSizeLimit: performance.memory.jsHeapSizeLimit,
                totalJSHeapSize: performance.memory.totalJSHeapSize,
                usedJSHeapSize: performance.memory.usedJSHeapSize,
            } : null,
            navigation: performance.navigation ? {
                type: performance.navigation.type,
                redirectCount: performance.navigation.redirectCount,
            } : null,
            timing: performance.timing ? {
                domContentLoaded: performance.timing.domContentLoadedEventEnd - performance.timing.navigationStart,
                loadComplete: performance.timing.loadEventEnd - performance.timing.navigationStart,
            } : null,
        };

        const hash = await this.hashString(JSON.stringify(perfData));
        this.fingerprint.performanceFingerprint = hash;
        this.components.push({ key: 'performance', value: hash });
    }

    async collectBatteryFingerprint() {
        try {
            if ('getBattery' in navigator) {
                const battery = await navigator.getBattery();
                const batteryData = {
                    charging: battery.charging,
                    level: battery.level,
                    chargingTime: battery.chargingTime,
                    dischargingTime: battery.dischargingTime,
                };
                const hash = await this.hashString(JSON.stringify(batteryData));
                this.fingerprint.batteryFingerprint = hash;
                this.components.push({ key: 'battery', value: hash });
            } else {
                this.fingerprint.batteryFingerprint = 'unsupported';
            }
        } catch (e) {
            this.fingerprint.batteryFingerprint = 'error';
        }
    }

    async collectNetworkFingerprint() {
        try {
            const connection = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
            if (connection) {
                const networkData = {
                    effectiveType: connection.effectiveType,
                    downlink: connection.downlink,
                    rtt: connection.rtt,
                    saveData: connection.saveData,
                };
                const hash = await this.hashString(JSON.stringify(networkData));
                this.fingerprint.networkFingerprint = hash;
                this.components.push({ key: 'network', value: hash });
            } else {
                this.fingerprint.networkFingerprint = 'unsupported';
            }
        } catch (e) {
            this.fingerprint.networkFingerprint = 'error';
        }
    }

    async collectStorageFingerprint() {
        const storageData = {
            localStorage: typeof localStorage !== 'undefined',
            sessionStorage: typeof sessionStorage !== 'undefined',
            indexedDB: typeof indexedDB !== 'undefined',
            webSQL: typeof openDatabase !== 'undefined',
            cookiesEnabled: navigator.cookieEnabled,
        };

        try {
            if (typeof localStorage !== 'undefined') {
                const testKey = '__seamless_test__';
                localStorage.setItem(testKey, 'test');
                localStorage.removeItem(testKey);
                storageData.localStorageWorks = true;
            }
        } catch (e) {
            storageData.localStorageWorks = false;
        }

        const hash = await this.hashString(JSON.stringify(storageData));
        this.fingerprint.storageFingerprint = hash;
        this.components.push({ key: 'storage', value: hash });
    }

    async hashString(str) {
        const encoder = new TextEncoder();
        const data = encoder.encode(str);
        const hashBuffer = await crypto.subtle.digest('SHA-256', data);
        const hashArray = Array.from(new Uint8Array(hashBuffer));
        return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
    }

    generateHash() {
        const coreComponents = [
            this.fingerprint.canvasFingerprint,
            this.fingerprint.webglFingerprint,
            this.fingerprint.audioFingerprint,
            this.fingerprint.fontFingerprint,
            this.fingerprint.userAgent,
            this.fingerprint.screenResolution,
        ].filter(Boolean).join('|');

        return this.hashString(coreComponents);
    }

    cacheFingerprint() {
        try {
            const cacheData = {
                fingerprint: this.fingerprint,
                components: this.components,
                timestamp: Date.now(),
            };
            localStorage.setItem('seamless_v15_fp', JSON.stringify(cacheData));
        } catch (e) {
            console.warn('Failed to cache fingerprint');
        }
    }

    getFingerprintHash() {
        return this.fingerprint.hash || '';
    }

    getComponentHashes() {
        const hashes = {};
        this.components.forEach(c => {
            hashes[c.key] = c.value;
        });
        return hashes;
    }
}

class SeamlessV15BehaviorTracker {
    constructor() {
        this.mouseTrajectory = [];
        this.keyboardEvents = [];
        this.clickEvents = [];
        this.scrollEvents = [];
        this.touchEvents = [];
        this.focusEvents = [];
        this.startTime = Date.now();
        this.sessionID = this.generateSessionID();
        this.isTracking = false;
        this.trackingDuration = 2000;
        this.sampleRate = 50;
    }

    generateSessionID() {
        return `sess_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }

    startTracking(duration = 2000) {
        if (this.isTracking) {
            return;
        }

        this.trackingDuration = duration;
        this.isTracking = true;
        this.startTime = Date.now();
        this.resetTrackingData();

        this.setupMouseTracking();
        this.setupKeyboardTracking();
        this.setupClickTracking();
        this.setupScrollTracking();
        this.setupTouchTracking();
        this.setupFocusTracking();

        this.trackingTimer = setTimeout(() => {
            this.stopTracking();
        }, duration);
    }

    stopTracking() {
        if (!this.isTracking) {
            return;
        }

        this.isTracking = false;

        if (this.trackingTimer) {
            clearTimeout(this.trackingTimer);
            this.trackingTimer = null;
        }

        this.removeEventListeners();
    }

    resetTrackingData() {
        this.mouseTrajectory = [];
        this.keyboardEvents = [];
        this.clickEvents = [];
        this.scrollEvents = [];
        this.touchEvents = [];
        this.focusEvents = [];
    }

    setupMouseTracking() {
        this.mouseHandler = (e) => {
            const now = Date.now();
            const point = {
                x: e.clientX,
                y: e.clientY,
                t: now - this.startTime,
                type: 'move',
            };

            if (this.mouseTrajectory.length > 0) {
                const last = this.mouseTrajectory[this.mouseTrajectory.length - 1];
                const dx = point.x - last.x;
                const dy = point.y - last.y;
                const dt = point.t - last.t;
                point.speed = Math.sqrt(dx * dx + dy * dy) / Math.max(1, dt);
                point.distance = Math.sqrt(dx * dx + dy * dy);
            }

            this.mouseTrajectory.push(point);

            if (this.mouseTrajectory.length > 1000) {
                this.mouseTrajectory = this.mouseTrajectory.filter((_, i) => i % 2 === 0);
            }
        };

        document.addEventListener('mousemove', this.mouseHandler, { passive: true });
    }

    setupKeyboardTracking() {
        this.keyboardHandler = (e) => {
            const now = Date.now();
            const event = {
                key: e.key,
                code: e.code,
                t: now - this.startTime,
                type: e.type,
                shift: e.shiftKey,
                ctrl: e.ctrlKey,
                alt: e.altKey,
            };

            if (this.lastKeyTime) {
                event.interKeyDelay = now - this.lastKeyTime;
            }
            this.lastKeyTime = now;

            this.keyboardEvents.push(event);

            if (this.keyboardEvents.length > 500) {
                this.keyboardEvents = this.keyboardEvents.filter((_, i) => i % 2 === 0);
            }
        };

        document.addEventListener('keydown', this.keyboardHandler, { passive: true });
        document.addEventListener('keyup', this.keyboardHandler, { passive: true });
    }

    setupClickTracking() {
        this.clickHandler = (e) => {
            const now = Date.now();
            this.clickEvents.push({
                x: e.clientX,
                y: e.clientY,
                t: now - this.startTime,
                target: e.target.tagName,
                targetClass: e.target.className,
                button: e.button,
            });
        };

        document.addEventListener('click', this.clickHandler, { passive: true });
    }

    setupScrollTracking() {
        let lastScrollY = window.scrollY;
        let lastScrollTime = Date.now();

        this.scrollHandler = () => {
            const now = Date.now();
            const currentScrollY = window.scrollY;
            const scrollDelta = Math.abs(currentScrollY - lastScrollY);
            const timeDelta = now - lastScrollTime;

            if (scrollDelta > 0) {
                this.scrollEvents.push({
                    scrollY: currentScrollY,
                    delta: scrollDelta,
                    t: now - this.startTime,
                    speed: scrollDelta / Math.max(1, timeDelta),
                });

                lastScrollY = currentScrollY;
                lastScrollTime = now;
            }
        };

        window.addEventListener('scroll', this.scrollHandler, { passive: true });
    }

    setupTouchTracking() {
        this.touchStartHandler = (e) => {
            const now = Date.now();
            for (const touch of e.changedTouches) {
                this.touchEvents.push({
                    x: touch.clientX,
                    y: touch.clientY,
                    t: now - this.startTime,
                    type: 'start',
                    touchId: touch.identifier,
                });
            }
        };

        this.touchMoveHandler = (e) => {
            const now = Date.now();
            for (const touch of e.changedTouches) {
                this.touchEvents.push({
                    x: touch.clientX,
                    y: touch.clientY,
                    t: now - this.startTime,
                    type: 'move',
                    touchId: touch.identifier,
                });
            }
        };

        this.touchEndHandler = (e) => {
            const now = Date.now();
            for (const touch of e.changedTouches) {
                this.touchEvents.push({
                    x: touch.clientX,
                    y: touch.clientY,
                    t: now - this.startTime,
                    type: 'end',
                    touchId: touch.identifier,
                });
            }
        };

        document.addEventListener('touchstart', this.touchStartHandler, { passive: true });
        document.addEventListener('touchmove', this.touchMoveHandler, { passive: true });
        document.addEventListener('touchend', this.touchEndHandler, { passive: true });
    }

    setupFocusTracking() {
        this.focusHandler = (e) => {
            this.focusEvents.push({
                t: Date.now() - this.startTime,
                type: e.type,
            });
        };

        window.addEventListener('focus', this.focusHandler);
        window.addEventListener('blur', this.focusHandler);
    }

    removeEventListeners() {
        if (this.mouseHandler) {
            document.removeEventListener('mousemove', this.mouseHandler);
        }
        if (this.keyboardHandler) {
            document.removeEventListener('keydown', this.keyboardHandler);
            document.removeEventListener('keyup', this.keyboardHandler);
        }
        if (this.clickHandler) {
            document.removeEventListener('click', this.clickHandler);
        }
        if (this.scrollHandler) {
            window.removeEventListener('scroll', this.scrollHandler);
        }
        if (this.touchStartHandler) {
            document.removeEventListener('touchstart', this.touchStartHandler);
            document.removeEventListener('touchmove', this.touchMoveHandler);
            document.removeEventListener('touchend', this.touchEndHandler);
        }
        if (this.focusHandler) {
            window.removeEventListener('focus', this.focusHandler);
            window.removeEventListener('blur', this.focusHandler);
        }
    }

    analyze() {
        const analysis = {
            sessionID: this.sessionID,
            totalDuration: Date.now() - this.startTime,
            mouseMoves: this.mouseTrajectory.length,
            keyboardEvents: this.keyboardEvents.length,
            clicks: this.clickEvents.length,
            scrolls: this.scrollEvents.length,
            touchEvents: this.touchEvents.length,
            focusChanges: this.focusEvents.length,
        };

        if (this.mouseTrajectory.length > 1) {
            const speeds = this.mouseTrajectory.map(m => m.speed || 0).filter(s => s > 0);
            analysis.avgMouseSpeed = speeds.length > 0 ? speeds.reduce((a, b) => a + b, 0) / speeds.length : 0;
            analysis.maxMouseSpeed = speeds.length > 0 ? Math.max(...speeds) : 0;
            analysis.minMouseSpeed = speeds.length > 0 ? Math.min(...speeds) : 0;

            const distances = this.mouseTrajectory.map(m => m.distance || 0).filter(d => d > 0);
            analysis.totalMouseDistance = distances.reduce((a, b) => a + b, 0);
        } else {
            analysis.avgMouseSpeed = 0;
            analysis.maxMouseSpeed = 0;
            analysis.minMouseSpeed = 0;
            analysis.totalMouseDistance = 0;
        }

        if (this.keyboardEvents.length > 1) {
            const delays = this.keyboardEvents.filter(e => e.interKeyDelay).map(e => e.interKeyDelay);
            analysis.avgKeyDelay = delays.length > 0 ? delays.reduce((a, b) => a + b, 0) / delays.length : 0;
            analysis.maxKeyDelay = delays.length > 0 ? Math.max(...delays) : 0;
            analysis.minKeyDelay = delays.length > 0 ? Math.min(...delays) : 0;
        } else {
            analysis.avgKeyDelay = 0;
            analysis.maxKeyDelay = 0;
            analysis.minKeyDelay = 0;
        }

        if (this.scrollEvents.length > 0) {
            const scrollSpeeds = this.scrollEvents.map(s => s.speed);
            analysis.avgScrollSpeed = scrollSpeeds.reduce((a, b) => a + b, 0) / scrollSpeeds.length;
            analysis.maxScrollSpeed = Math.max(...scrollSpeeds);
        } else {
            analysis.avgScrollSpeed = 0;
            analysis.maxScrollSpeed = 0;
        }

        analysis.trajectoryComplexity = this.calculateTrajectoryComplexity();
        analysis.rhythmScore = this.calculateRhythmScore();

        return analysis;
    }

    calculateTrajectoryComplexity() {
        if (this.mouseTrajectory.length < 3) {
            return 0;
        }

        let totalAngleChange = 0;
        for (let i = 1; i < this.mouseTrajectory.length - 1; i++) {
            const v1 = {
                x: this.mouseTrajectory[i].x - this.mouseTrajectory[i-1].x,
                y: this.mouseTrajectory[i].y - this.mouseTrajectory[i-1].y,
            };
            const v2 = {
                x: this.mouseTrajectory[i+1].x - this.mouseTrajectory[i].x,
                y: this.mouseTrajectory[i+1].y - this.mouseTrajectory[i].y,
            };

            const dot = v1.x * v2.x + v1.y * v2.y;
            const mag1 = Math.sqrt(v1.x * v1.x + v1.y * v1.y);
            const mag2 = Math.sqrt(v2.x * v2.x + v2.y * v2.y);

            if (mag1 > 0 && mag2 > 0) {
                const cosAngle = Math.max(-1, Math.min(1, dot / (mag1 * mag2)));
                totalAngleChange += Math.acos(cosAngle);
            }
        }

        return totalAngleChange / Math.PI;
    }

    calculateRhythmScore() {
        if (this.keyboardEvents.length < 3) {
            return 0.5;
        }

        const delays = this.keyboardEvents
            .filter(e => e.interKeyDelay && e.interKeyDelay > 0 && e.interKeyDelay < 1000)
            .map(e => e.interKeyDelay);

        if (delays.length < 2) {
            return 0.5;
        }

        const mean = delays.reduce((a, b) => a + b, 0) / delays.length;
        const variance = delays.reduce((a, b) => a + Math.pow(b - mean, 2), 0) / delays.length;
        const stdDev = Math.sqrt(variance);

        const cv = stdDev / mean;
        return Math.max(0, Math.min(1, 1 - cv));
    }

    getBehaviorScore() {
        const analysis = this.analyze();
        let score = 100;

        if (analysis.mouseMoves < 5) {
            score -= 25;
        }

        if (analysis.avgMouseSpeed > 100) {
            score -= 20;
        }

        if (analysis.totalDuration < 1000) {
            score -= 30;
        }

        if (analysis.keyboardEvents === 0 && analysis.clicks === 0) {
            score -= 15;
        }

        if (analysis.trajectoryComplexity < 0.5) {
            score -= 10;
        }

        if (analysis.rhythmScore < 0.3) {
            score -= 10;
        }

        return Math.max(0, Math.min(100, score));
    }

    async generateBehaviorHash() {
        const analysis = this.analyze();
        const data = JSON.stringify({
            sessionID: analysis.sessionID,
            duration: analysis.totalDuration,
            mouseMoves: analysis.mouseMoves,
            keyboardEvents: analysis.keyboardEvents,
            clicks: analysis.clicks,
            complexity: analysis.trajectoryComplexity,
            rhythm: analysis.rhythmScore,
        });

        const encoder = new TextEncoder();
        const hashBuffer = await crypto.subtle.digest('SHA-256', encoder.encode(data));
        const hashArray = Array.from(new Uint8Array(hashBuffer));
        return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
    }

    getSummary() {
        return {
            sessionID: this.sessionID,
            duration: Date.now() - this.startTime,
            mouseMoves: this.mouseTrajectory.length,
            keyboardEvents: this.keyboardEvents.length,
            clicks: this.clickEvents.length,
            scrolls: this.scrollEvents.length,
            behaviorScore: this.getBehaviorScore(),
        };
    }
}

class SeamlessV15 {
    constructor(containerId, options = {}) {
        this.container = typeof containerId === 'string' 
            ? document.getElementById(containerId) 
            : containerId;

        if (!this.container) {
            console.error('SeamlessV15: Container not found');
            return;
        }

        this.options = {
            apiBase: '/api/v1',
            language: 'zh-CN',
            autoStart: false,
            showProgress: true,
            showDetails: true,
            trackingDuration: 2000,
            enableCaching: true,
            maxRetries: 3,
            timeout: 10000,
            onStatusChange: null,
            onProgress: null,
            onComplete: null,
            onError: null,
            onTrustChange: null,
            ...options
        };

        this.fingerprintCollector = new SeamlessV15FingerprintCollector();
        this.behaviorTracker = new SeamlessV15BehaviorTracker();

        this.state = {
            status: 'idle',
            fingerprint: null,
            behaviorData: null,
            trustScore: 0,
            riskScore: 0,
            sessionId: null,
            token: null,
            verificationType: 'pending',
            confidence: 0,
            isVerifying: false,
            retryCount: 0,
            lastUpdate: null,
        };

        this.translations = {
            'zh-CN': {
                initializing: '初始化中...',
                collecting: '采集设备信息...',
                analyzing: '分析行为模式...',
                calculating: '计算信任评分...',
                verifying: '验证中...',
                trusted: '验证通过',
                untrusted: '需要验证',
                blocked: '验证失败',
                error: '验证出错',
                retry: '重试',
                trustScore: '信任评分',
                riskScore: '风险评分',
                deviceInfo: '设备信息',
                behaviorInfo: '行为分析',
                progress: '进度',
            },
            'en-US': {
                initializing: 'Initializing...',
                collecting: 'Collecting device info...',
                analyzing: 'Analyzing behavior...',
                calculating: 'Calculating trust score...',
                verifying: 'Verifying...',
                trusted: 'Verified',
                untrusted: 'Needs Verification',
                blocked: 'Verification Failed',
                error: 'Verification Error',
                retry: 'Retry',
                trustScore: 'Trust Score',
                riskScore: 'Risk Score',
                deviceInfo: 'Device Info',
                behaviorInfo: 'Behavior Analysis',
                progress: 'Progress',
            }
        };

        this.i18n = this.translations[this.options.language] || this.translations['zh-CN'];

        if (this.options.autoStart) {
            this.start();
        }
    }

    async start() {
        if (this.state.isVerifying) {
            return;
        }

        this.state.isVerifying = true;
        this.state.retryCount = 0;

        try {
            await this.executeVerification();
        } catch (error) {
            console.error('SeamlessV15 verification error:', error);
            this.handleError(error);
        }
    }

    async executeVerification() {
        this.updateStatus('initializing');
        this.updateProgress(0);

        this.updateStatus('collecting');
        await this.collectFingerprint();
        this.updateProgress(25);

        this.updateStatus('analyzing');
        await this.trackBehavior();
        this.updateProgress(50);

        this.updateStatus('calculating');
        const riskScore = await this.calculateRiskScore();
        this.updateProgress(75);

        this.updateStatus('verifying');
        await this.verifyWithServer(riskScore);
        this.updateProgress(100);

        this.state.isVerifying = false;
        this.handleComplete();
    }

    async collectFingerprint() {
        try {
            this.state.fingerprint = await this.fingerprintCollector.collect();
            
            if (this.options.showDetails) {
                this.displayFingerprintInfo();
            }
        } catch (error) {
            console.error('Fingerprint collection failed:', error);
            throw error;
        }
    }

    async trackBehavior() {
        return new Promise((resolve) => {
            this.behaviorTracker.startTracking(this.options.trackingDuration);

            setTimeout(() => {
                this.behaviorTracker.stopTracking();
                this.state.behaviorData = this.behaviorTracker.analyze();
                
                if (this.options.showDetails) {
                    this.displayBehaviorInfo();
                }

                resolve();
            }, this.options.trackingDuration);
        });
    }

    async calculateRiskScore() {
        const behaviorScore = this.behaviorTracker.getBehaviorScore();

        let riskFactors = {
            fingerprintEntropy: await this.calculateFingerprintEntropy(),
            behaviorScore: behaviorScore,
            deviceConsistency: await this.checkDeviceConsistency(),
            trajectoryComplexity: this.state.behaviorData?.trajectoryComplexity || 0,
            rhythmScore: this.state.behaviorData?.rhythmScore || 0,
        };

        const weights = {
            fingerprintEntropy: 0.25,
            behaviorScore: 0.35,
            deviceConsistency: 0.20,
            trajectoryComplexity: 0.10,
            rhythmScore: 0.10,
        };

        let totalScore = 0;
        for (const [factor, weight] of Object.entries(weights)) {
            totalScore += (riskFactors[factor] || 0) * weight;
        }

        this.state.riskScore = Math.round(100 - totalScore);
        this.state.behaviorScore = behaviorScore;

        return this.state.riskScore;
    }

    async calculateFingerprintEntropy() {
        if (!this.state.fingerprint) return 0;

        let entropy = 0;
        const fp = this.state.fingerprint;

        const components = [
            fp.canvasFingerprint,
            fp.webglFingerprint,
            fp.audioFingerprint,
            fp.fontFingerprint,
            fp.screenResolution,
        ];

        const validComponents = components.filter(c => c && c !== 'error' && c !== 'unsupported');
        entropy = (validComponents.length / components.length) * 100;

        return entropy;
    }

    async checkDeviceConsistency() {
        const storedHash = localStorage.getItem('seamless_v15_fp_hash');
        const currentHash = this.fingerprintCollector.getFingerprintHash();

        if (storedHash && storedHash === currentHash) {
            const storedTrust = localStorage.getItem('seamless_v15_trust');
            if (storedTrust === 'true') {
                return 100;
            }
        }

        if (storedHash) {
            const similarity = this.calculateFingerprintSimilarity(storedHash, currentHash);
            return similarity * 100;
        }

        return 50;
    }

    calculateFingerprintSimilarity(hash1, hash2) {
        if (!hash1 || !hash2 || hash1.length !== hash2.length) {
            return 0;
        }

        let matches = 0;
        for (let i = 0; i < hash1.length; i++) {
            if (hash1[i] === hash2[i]) {
                matches++;
            }
        }

        return matches / hash1.length;
    }

    async verifyWithServer(riskScore) {
        const maxRetries = this.options.maxRetries;

        for (let attempt = 0; attempt < maxRetries; attempt++) {
            try {
                const response = await this.apiCall('/seamless/verify', 'POST', {
                    user_id: this.getUserID(),
                    fingerprint: this.fingerprintCollector.getFingerprintHash(),
                    session_id: this.state.sessionId || `sess_${Date.now()}`,
                    risk_score: riskScore,
                    behavior_data: this.state.behaviorData,
                });

                if (response.success) {
                    this.state.sessionId = response.session_id || this.state.sessionId;
                    this.state.token = response.token;
                    this.state.trustScore = response.trust_score;
                    this.state.riskScore = response.risk_score;
                    this.state.verificationType = response.verification_type;
                    this.state.confidence = response.confidence;
                    this.state.skipVerification = response.skip_verification;

                    if (response.verification_type === 'seamless' && response.confidence > 0.7) {
                        this.markDeviceTrusted();
                    }

                    return;
                }

                if (attempt < maxRetries - 1) {
                    await this.delay(1000 * (attempt + 1));
                }
            } catch (error) {
                console.error(`Verification attempt ${attempt + 1} failed:`, error);
                if (attempt < maxRetries - 1) {
                    await this.delay(1000 * (attempt + 1));
                }
            }
        }

        this.state.verificationType = 'failed';
        throw new Error('Verification failed after retries');
    }

    getUserID() {
        let userID = sessionStorage.getItem('seamless_v15_user_id');
        if (!userID) {
            userID = `user_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
            sessionStorage.setItem('seamless_v15_user_id', userID);
        }
        return userID;
    }

    markDeviceTrusted() {
        try {
            const hash = this.fingerprintCollector.getFingerprintHash();
            localStorage.setItem('seamless_v15_fp_hash', hash);
            localStorage.setItem('seamless_v15_trust', 'true');
            localStorage.setItem('seamless_v15_trust_time', Date.now().toString());

            const expiry = 30 * 24 * 60 * 60 * 1000;
            localStorage.setItem('seamless_v15_trust_expiry', (Date.now() + expiry).toString());
        } catch (e) {
            console.warn('Failed to mark device as trusted');
        }
    }

    async updateBehaviorData() {
        try {
            const response = await this.apiCall('/seamless/update', 'POST', {
                user_id: this.getUserID(),
                fingerprint: this.fingerprintCollector.getFingerprintHash(),
                fingerprint_components: this.fingerprintCollector.getComponentHashes(),
                session_id: this.behaviorTracker.sessionID,
                timestamp: Date.now(),
                duration: this.state.behaviorData?.totalDuration || 0,
                mouse_moves: this.state.behaviorData?.mouseMoves || 0,
                key_strokes: this.state.behaviorData?.keyboardEvents || 0,
                clicks: this.state.behaviorData?.clicks || 0,
                scroll_events: this.state.behaviorData?.scrolls || 0,
                average_speed: this.state.behaviorData?.avgMouseSpeed || 0,
                risk_score: this.state.riskScore,
                success: this.state.verificationType === 'seamless',
                behavior_hash: await this.behaviorTracker.generateBehaviorHash(),
                ip_address: '',
                user_agent: navigator.userAgent,
            });

            if (response.success) {
                this.state.lastUpdate = Date.now();
            }
        } catch (error) {
            console.error('Failed to update behavior data:', error);
        }
    }

    async getTrustScore() {
        try {
            const response = await this.apiCall('/seamless/trust-score', 'GET', null, {
                user_id: this.getUserID(),
                fingerprint: this.fingerprintCollector.getFingerprintHash(),
            });

            if (response.success) {
                return {
                    trustScore: response.trust_score,
                    riskScore: response.risk_score,
                    deviceStable: response.device_stable,
                    deviceConfidence: response.device_confidence,
                    behaviorConfidence: response.behavior_confidence,
                };
            }
        } catch (error) {
            console.error('Failed to get trust score:', error);
        }

        return null;
    }

    async getLearningStatus() {
        try {
            const response = await this.apiCall('/seamless/learning-status', 'GET', null, {
                user_id: this.getUserID(),
                fingerprint: this.fingerprintCollector.getFingerprintHash(),
            });

            if (response.success) {
                return {
                    phase: response.learning_phase,
                    progress: response.progress,
                    estimatedTime: response.estimated_time,
                    requiredActions: response.required_actions,
                };
            }
        } catch (error) {
            console.error('Failed to get learning status:', error);
        }

        return null;
    }

    displayFingerprintInfo() {
        const fp = this.state.fingerprint;
        if (!fp) return;

        const deviceInfo = document.getElementById('seamless-device-info');
        if (deviceInfo) {
            deviceInfo.innerHTML = `
                <div class="seamless-info-item">
                    <span class="label">${this.i18n.deviceInfo}:</span>
                    <span class="value">${fp.screenResolution} (${fp.platform})</span>
                </div>
                <div class="seamless-info-item">
                    <span class="label">浏览器:</span>
                    <span class="value">${fp.userAgent.includes('Chrome') ? 'Chrome' : 
                        fp.userAgent.includes('Firefox') ? 'Firefox' : 
                        fp.userAgent.includes('Safari') ? 'Safari' : 'Other'}</span>
                </div>
                <div class="seamless-info-item">
                    <span class="label">时区:</span>
                    <span class="value">${fp.timezone}</span>
                </div>
            `;
        }
    }

    displayBehaviorInfo() {
        const bd = this.state.behaviorData;
        if (!bd) return;

        const behaviorInfo = document.getElementById('seamless-behavior-info');
        if (behaviorInfo) {
            behaviorInfo.innerHTML = `
                <div class="seamless-info-item">
                    <span class="label">${this.i18n.behaviorInfo}:</span>
                    <span class="value">鼠标移动: ${bd.mouseMoves}, 按键: ${bd.keyboardEvents}, 点击: ${bd.clicks}</span>
                </div>
                <div class="seamless-info-item">
                    <span class="label">行为评分:</span>
                    <span class="value">${bd.behaviorScore || 0}</span>
                </div>
            `;
        }
    }

    updateStatus(status) {
        this.state.status = status;
        
        if (this.options.onStatusChange) {
            this.options.onStatusChange(status, this.i18n[status] || status);
        }

        const statusEl = document.getElementById('seamless-status');
        if (statusEl) {
            statusEl.textContent = this.i18n[status] || status;
        }
    }

    updateProgress(progress) {
        if (this.options.onProgress) {
            this.options.onProgress(progress);
        }

        const progressBar = document.getElementById('seamless-progress-bar');
        if (progressBar) {
            progressBar.style.width = `${progress}%`;
        }

        const progressText = document.getElementById('seamless-progress-text');
        if (progressText) {
            progressText.textContent = `${this.i18n.progress}: ${progress}%`;
        }
    }

    handleComplete() {
        const isTrusted = this.state.verificationType === 'seamless' || 
                         this.state.verificationType === 'allow';

        this.updateStatus(isTrusted ? 'trusted' : 'untrusted');

        if (this.options.onComplete) {
            this.options.onComplete({
                success: isTrusted,
                trusted: isTrusted,
                verificationType: this.state.verificationType,
                trustScore: this.state.trustScore,
                riskScore: this.state.riskScore,
                confidence: this.state.confidence,
                token: this.state.token,
                sessionId: this.state.sessionId,
                fingerprint: this.state.fingerprint,
                behaviorData: this.state.behaviorData,
            });
        }

        if (this.options.onTrustChange) {
            this.options.onTrustChange(isTrusted, this.state.trustScore);
        }

        this.updateBehaviorData();
    }

    handleError(error) {
        this.state.isVerifying = false;
        this.updateStatus('error');

        if (this.options.onError) {
            this.options.onError({
                message: error.message || 'Verification failed',
                code: error.code || 'UNKNOWN_ERROR',
            });
        }
    }

    async apiCall(endpoint, method = 'GET', data = null, params = null) {
        let url = this.options.apiBase + endpoint;

        if (params) {
            const queryString = Object.entries(params)
                .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
                .join('&');
            url += `?${queryString}`;
        }

        const options = {
            method,
            headers: {
                'Content-Type': 'application/json',
            },
            credentials: 'same-origin',
        };

        if (data && method !== 'GET') {
            options.body = JSON.stringify(data);
        }

        try {
            const controller = new AbortController();
            const timeout = setTimeout(() => controller.abort(), this.options.timeout);
            options.signal = controller.signal;

            const response = await fetch(url, options);
            clearTimeout(timeout);

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            return await response.json();
        } catch (error) {
            if (error.name === 'AbortError') {
                throw new Error('Request timeout');
            }
            throw error;
        }
    }

    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    async checkCachedStatus() {
        const storedHash = localStorage.getItem('seamless_v15_fp_hash');
        const currentHash = this.fingerprintCollector.getFingerprintHash();

        if (storedHash && storedHash === currentHash) {
            const trustExpiry = localStorage.getItem('seamless_v15_trust_expiry');
            if (trustExpiry && parseInt(trustExpiry) > Date.now()) {
                return {
                    cached: true,
                    trusted: true,
                    trustScore: 100,
                };
            }
        }

        return {
            cached: false,
            trusted: false,
            trustScore: 0,
        };
    }

    reset() {
        this.state = {
            status: 'idle',
            fingerprint: null,
            behaviorData: null,
            trustScore: 0,
            riskScore: 0,
            sessionId: null,
            token: null,
            verificationType: 'pending',
            confidence: 0,
            isVerifying: false,
            retryCount: 0,
            lastUpdate: null,
        };

        this.behaviorTracker = new SeamlessV15BehaviorTracker();

        this.updateStatus('initializing');
        this.updateProgress(0);
    }

    destroy() {
        this.reset();
        this.behaviorTracker.stopTracking();
        this.container = null;
        this.options = {};
    }

    getState() {
        return { ...this.state };
    }

    getVersion() {
        return '15.0.0';
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = { 
        SeamlessV15, 
        SeamlessV15FingerprintCollector, 
        SeamlessV15BehaviorTracker 
    };
}
