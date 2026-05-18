class FingerprintCollector {
    constructor() {
        this.fingerprint = {};
        this.components = [];
        this.featureCount = 0;
    }

    async collect() {
        this.components = [];
        this.featureCount = 0;

        this.fingerprint.userAgent = this.getUserAgent();
        this.addComponent('userAgent', this.fingerprint.userAgent);

        this.fingerprint.screenResolution = this.getScreenResolution();
        this.addComponent('screenResolution', this.fingerprint.screenResolution);

        this.fingerprint.colorDepth = this.getColorDepth();
        this.addComponent('colorDepth', this.fingerprint.colorDepth);

        this.fingerprint.timezone = this.getTimezone();
        this.addComponent('timezone', this.fingerprint.timezone);

        this.fingerprint.language = this.getLanguage();
        this.addComponent('language', this.fingerprint.language);

        this.fingerprint.platform = this.getPlatform();
        this.addComponent('platform', this.fingerprint.platform);

        this.fingerprint.canvasFingerprint = await this.getCanvasFingerprint();
        this.addComponent('canvasFingerprint', this.fingerprint.canvasFingerprint);

        this.fingerprint.webglFingerprint = this.getWebGLFingerprint();
        this.addComponent('webglFingerprint', this.fingerprint.webglFingerprint);

        this.fingerprint.audioFingerprint = await this.getAudioFingerprint();
        this.addComponent('audioFingerprint', this.fingerprint.audioFingerprint);

        this.fingerprint.fontFingerprint = await this.getFontFingerprint();
        this.addComponent('fontFingerprint', this.fingerprint.fontFingerprint);

        this.fingerprint.plugins = this.getPlugins();
        this.addComponent('plugins', this.fingerprint.plugins);

        this.fingerprint.doNotTrack = this.getDoNotTrack();
        this.addComponent('doNotTrack', this.fingerprint.doNotTrack);

        this.fingerprint.touchSupport = this.getTouchSupport();
        this.addComponent('touchSupport', JSON.stringify(this.fingerprint.touchSupport));

        this.fingerprint.deviceMemory = this.getDeviceMemory();
        this.addComponent('deviceMemory', this.fingerprint.deviceMemory);

        this.fingerprint.hardwareConcurrency = this.getHardwareConcurrency();
        this.addComponent('hardwareConcurrency', this.fingerprint.hardwareConcurrency);

        this.fingerprint.connectionType = await this.getConnectionType();
        this.addComponent('connectionType', this.fingerprint.connectionType);

        this.fingerprint.batteryStatus = await this.getBatteryStatus();
        this.addComponent('batteryStatus', JSON.stringify(this.fingerprint.batteryStatus));

        this.fingerprint.storageQuota = await this.getStorageQuota();
        this.addComponent('storageQuota', JSON.stringify(this.fingerprint.storageQuota));

        this.fingerprint.mediaDevices = await this.getMediaDevices();
        this.addComponent('mediaDevices', JSON.stringify(this.fingerprint.mediaDevices));

        this.fingerprint.performanceTiming = this.getPerformanceTiming();
        this.addComponent('performanceTiming', JSON.stringify(this.fingerprint.performanceTiming));

        this.fingerprint.webrtcStatus = this.getWebRTCStatus();
        this.addComponent('webrtcStatus', this.fingerprint.webrtcStatus);

        this.fingerprint.vendorWebGL = this.getVendorWebGL();
        this.addComponent('vendorWebGL', this.fingerprint.vendorWebGL);

        this.fingerprint.cookieEnabled = this.getCookieEnabled();
        this.addComponent('cookieEnabled', this.fingerprint.cookieEnabled);

        this.fingerprint.javaEnabled = this.getJavaEnabled();
        this.addComponent('javaEnabled', this.fingerprint.javaEnabled);

        this.fingerprint.browserEngine = this.getBrowserEngine();
        this.addComponent('browserEngine', this.fingerprint.browserEngine);

        this.fingerprint.devicePixelRatio = this.getDevicePixelRatio();
        this.addComponent('devicePixelRatio', this.fingerprint.devicePixelRatio);

        this.fingerprint.maxTouchPoints = this.getMaxTouchPoints();
        this.addComponent('maxTouchPoints', this.fingerprint.maxTouchPoints);

        this.fingerprint.webdriverStatus = this.getWebdriverStatus();
        this.addComponent('webdriverStatus', this.fingerprint.webdriverStatus);

        this.fingerprint.permissionsAPI = await this.getPermissionsAPI();
        this.addComponent('permissionsAPI', JSON.stringify(this.fingerprint.permissionsAPI));

        this.fingerprint.vendorMozilla = this.getVendorMozilla();
        this.addComponent('vendorMozilla', this.fingerprint.vendorMozilla);

        this.fingerprint.buildID = this.getBuildID();
        this.addComponent('buildID', this.fingerprint.buildID);

        this.fingerprint.hardwareConcurrency = this.getHardwareConcurrency();
        this.addComponent('hardwareConcurrency', this.fingerprint.hardwareConcurrency);

        this.fingerprint.keyboardLayout = await this.getKeyboardLayout();
        this.addComponent('keyboardLayout', JSON.stringify(this.fingerprint.keyboardLayout));

        this.fingerprint.sessionStorage = this.getSessionStorage();
        this.addComponent('sessionStorage', this.fingerprint.sessionStorage);

        this.fingerprint.localStorage = this.getLocalStorage();
        this.addComponent('localStorage', this.fingerprint.localStorage);

        this.fingerprint.indexedDBSupport = this.getIndexedDBSupport();
        this.addComponent('indexedDBSupport', this.fingerprint.indexedDBSupport);

        this.fingerprint.timestamp = Date.now();
        this.addComponent('timestamp', this.fingerprint.timestamp);

        this.fingerprint.hash = this.generateHash();

        this.fingerprint.featureCount = this.featureCount;

        return this.fingerprint;
    }

    addComponent(key, value) {
        this.components.push({ key: key, value: value });
        this.featureCount++;
    }

    getUserAgent() {
        return navigator.userAgent;
    }

    getScreenResolution() {
        return `${screen.width}x${screen.height}x${screen.colorDepth}`;
    }

    getColorDepth() {
        return screen.colorDepth;
    }

    getTimezone() {
        return Intl.DateTimeFormat().resolvedOptions().timeZone;
    }

    getLanguage() {
        return navigator.language || navigator.userLanguage;
    }

    getPlatform() {
        return navigator.platform;
    }

    async getCanvasFingerprint() {
        try {
            const canvas = document.createElement('canvas');
            const ctx = canvas.getContext('2d');
            canvas.width = 200;
            canvas.height = 50;

            ctx.textBaseline = 'top';
            ctx.font = "14px 'Arial'";
            ctx.textBaseline = 'alphabetic';
            ctx.fillStyle = '#f60';
            ctx.fillRect(125, 1, 62, 20);
            ctx.fillStyle = '#069';
            ctx.fillText('C2#IOJS@√M', 2, 15);
            ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
            ctx.fillText('Seamless Captcha FP', 4, 17);

            ctx.beginPath();
            ctx.moveTo(10, 10);
            ctx.bezierCurveTo(20, 30, 40, 10, 50, 20);
            ctx.strokeStyle = '#800080';
            ctx.stroke();

            ctx.beginPath();
            ctx.arc(80, 25, 15, 0, Math.PI * 2);
            ctx.fillStyle = 'rgba(255, 128, 0, 0.5)';
            ctx.fill();

            const dataUrl = canvas.toDataURL();
            return await this.hashString(dataUrl);
        } catch (e) {
            return 'canvas-error';
        }
    }

    getWebGLFingerprint() {
        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
            if (!gl) return 'webgl-unsupported';

            const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
            const vendor = debugInfo ? gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL) : 'unknown';
            const renderer = debugInfo ? gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) : 'unknown';

            const params = {
                ALIASED_LINE_WIDTH_RANGE: gl.getParameter(gl.ALIASED_LINE_WIDTH_RANGE),
                ALIASED_POINT_SIZE_RANGE: gl.getParameter(gl.ALIASED_POINT_SIZE_RANGE),
                MAX_TEXTURE_SIZE: gl.getParameter(gl.MAX_TEXTURE_SIZE),
                MAX_VIEWPORT_DIMS: gl.getParameter(gl.MAX_VIEWPORT_DIMS),
                VERTEX_SHADER: gl.getParameter(gl.VERTEX_SHADER),
                FRAGMENT_SHADER: gl.getParameter(gl.FRAGMENT_SHADER)
            };

            return `${vendor}~${renderer}~${JSON.stringify(params)}`.substring(0, 200);
        } catch (e) {
            return 'webgl-error';
        }
    }

    getVendorWebGL() {
        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
            if (!gl) return 'unsupported';
            return gl.getParameter(gl.VENDOR);
        } catch (e) {
            return 'error';
        }
    }

    async getAudioFingerprint() {
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

            return new Promise((resolve) => {
                scriptProcessor.onaudioprocess = function(event) {
                    const output = event.inputBuffer.getChannelData(0);
                    let sum = 0;
                    for (let i = 0; i < output.length; i++) {
                        sum += Math.abs(output[i]);
                    }
                    const fingerprint = sum.toString();
                    oscillator.stop();
                    audioContext.close();
                    resolve(fingerprint);
                };
            });
        } catch (e) {
            return 'audio-error';
        }
    }

    async getFontFingerprint() {
        const baseFonts = ['monospace', 'sans-serif', 'serif'];
        const testFonts = [
            'Arial', 'Arial Black', 'Comic Sans MS', 'Courier New', 'Georgia',
            'Impact', 'Times New Roman', 'Trebuchet MS', 'Verdana', 'Palatino',
            'Lucida Console', 'Lucida Sans Unicode', 'Tahoma', 'Geneva', 'Helvetica',
            'Consolas', 'Monaco', 'Cambria', 'Candara', 'Calibri', 'Segoe UI'
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

        return detectedFonts.join(',');
    }

    getPlugins() {
        const plugins = [];
        if (navigator.plugins) {
            for (let i = 0; i < navigator.plugins.length; i++) {
                plugins.push({
                    name: navigator.plugins[i].name,
                    filename: navigator.plugins[i].filename
                });
            }
        }
        return JSON.stringify(plugins);
    }

    getDoNotTrack() {
        return navigator.doNotTrack === '1' ||
               navigator.doNotTrack === 'yes' ||
               window.doNotTrack === '1';
    }

    getTouchSupport() {
        return {
            maxTouchPoints: navigator.maxTouchPoints || 0,
            touchEvent: 'ontouchstart' in window,
            touch: navigator.maxTouchPoints > 0
        };
    }

    getDeviceMemory() {
        return navigator.deviceMemory || 'unknown';
    }

    getHardwareConcurrency() {
        return navigator.hardwareConcurrency || 'unknown';
    }

    async getConnectionType() {
        try {
            const connection = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
            if (connection) {
                return connection.effectiveType || 'unknown';
            }
            return 'unknown';
        } catch (e) {
            return 'unknown';
        }
    }

    async getBatteryStatus() {
        try {
            if ('getBattery' in navigator) {
                const battery = await navigator.getBattery();
                return {
                    level: battery.level,
                    charging: battery.charging,
                    chargingTime: battery.chargingTime,
                    dischargingTime: battery.dischargingTime
                };
            }
            return { supported: false };
        } catch (e) {
            return { supported: false, error: 'battery-api-error' };
        }
    }

    async getStorageQuota() {
        try {
            if (navigator.storage && navigator.storage.estimate) {
                const estimate = await navigator.storage.estimate();
                return {
                    quota: estimate.quota,
                    usage: estimate.usage,
                    usageDetails: estimate.usageDetails
                };
            }
            return { supported: false };
        } catch (e) {
            return { supported: false, error: 'storage-api-error' };
        }
    }

    async getMediaDevices() {
        try {
            if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
                const devices = await navigator.mediaDevices.enumerateDevices();
                return {
                    audioInputs: devices.filter(d => d.kind === 'audioinput').length,
                    videoInputs: devices.filter(d => d.kind === 'videoinput').length,
                    audioOutputs: devices.filter(d => d.kind === 'audiooutput').length
                };
            }
            return { supported: false };
        } catch (e) {
            return { supported: false, error: 'media-devices-error' };
        }
    }

    getPerformanceTiming() {
        try {
            const timing = performance.timing;
            return {
                navigationStart: timing.navigationStart,
                loadEventEnd: timing.loadEventEnd,
                domContentLoadedEventEnd: timing.domContentLoadedEventEnd,
                domInteractive: timing.domInteractive,
                responseEnd: timing.responseEnd,
                requestStart: timing.requestStart
            };
        } catch (e) {
            return { error: 'performance-api-error' };
        }
    }

    getWebRTCStatus() {
        try {
            const pc = window.RTCPeerConnection || window.mozRTCPeerConnection || window.webkitRTCPeerConnection;
            return pc ? 'supported' : 'unsupported';
        } catch (e) {
            return 'error';
        }
    }

    getCookieEnabled() {
        return navigator.cookieEnabled;
    }

    getJavaEnabled() {
        try {
            return navigator.javaEnabled ? navigator.javaEnabled() : false;
        } catch (e) {
            return false;
        }
    }

    getBrowserEngine() {
        const ua = navigator.userAgent;
        if (ua.indexOf('Trident') > -1 || ua.indexOf('MSIE') > -1) return 'trident';
        if (ua.indexOf('Edge') > -1) return 'edge';
        if (ua.indexOf('Edg') > -1) return 'edg';
        if (ua.indexOf('Chrome') > -1) return 'blink';
        if (ua.indexOf('Safari') > -1) return 'webkit';
        if (ua.indexOf('Firefox') > -1) return 'gecko';
        return 'unknown';
    }

    getDevicePixelRatio() {
        return window.devicePixelRatio || 1;
    }

    getMaxTouchPoints() {
        return navigator.maxTouchPoints || 0;
    }

    getWebdriverStatus() {
        return window.navigator.webdriver || false;
    }

    async getPermissionsAPI() {
        try {
            if (navigator.permissions && navigator.permissions.query) {
                const permissions = ['geolocation', 'notifications', 'push', 'microphone', 'camera'];
                const results = {};
                for (const perm of permissions) {
                    try {
                        const result = await navigator.permissions.query({ name: perm });
                        results[perm] = result.state;
                    } catch (e) {
                        results[perm] = 'error';
                    }
                }
                return results;
            }
            return { supported: false };
        } catch (e) {
            return { supported: false, error: 'permissions-api-error' };
        }
    }

    getVendorMozilla() {
        return navigator.vendor || '';
    }

    getBuildID() {
        return navigator.buildID || '';
    }

    async getKeyboardLayout() {
        try {
            if (navigator.keyboard && navigator.keyboard.getLayoutMap) {
                const layoutMap = await navigator.keyboard.getLayoutMap();
                const keys = ['a', 's', 'd', 'f', 'j', 'k', 'l'];
                const result = {};
                for (const key of keys) {
                    result[key] = layoutMap.get(key) || key;
                }
                return result;
            }
            return { supported: false };
        } catch (e) {
            return { supported: false, error: 'keyboard-api-error' };
        }
    }

    getSessionStorage() {
        try {
            return typeof sessionStorage !== 'undefined';
        } catch (e) {
            return false;
        }
    }

    getLocalStorage() {
        try {
            return typeof localStorage !== 'undefined';
        } catch (e) {
            return false;
        }
    }

    getIndexedDBSupport() {
        return typeof indexedDB !== 'undefined';
    }

    async hashString(str) {
        const encoder = new TextEncoder();
        const data = encoder.encode(str);
        const hashBuffer = await crypto.subtle.digest('SHA-256', data);
        const hashArray = Array.from(new Uint8Array(hashBuffer));
        return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
    }

    generateHash() {
        const components = [
            this.fingerprint.canvasFingerprint,
            this.fingerprint.webglFingerprint,
            this.fingerprint.audioFingerprint,
            this.fingerprint.screenResolution,
            this.fingerprint.timezone,
            this.fingerprint.platform,
            this.fingerprint.language
        ];
        const combined = components.join('|');
        return this.hashString(combined);
    }

    getFingerprintHash() {
        return this.fingerprint.hash || '';
    }

    getFeatureCount() {
        return this.featureCount;
    }
}

class BehaviorAnalyzer {
    constructor(options = {}) {
        this.options = {
            trackingDuration: options.trackingDuration || 3000,
            maxTrajectoryPoints: options.maxTrajectoryPoints || 500,
            enableAdvancedMetrics: options.enableAdvancedMetrics !== false,
            ...options
        };

        this.mouseTrajectory = [];
        this.keyboardPattern = [];
        this.scrollBehavior = [];
        this.clickPattern = [];
        this.timingData = [];
        this.startTime = Date.now();
        this.lastActivityTime = Date.now();

        this.metrics = {
            totalTime: 0,
            mouseMovements: 0,
            keyboardInputs: 0,
            scrolls: 0,
            clicks: 0,
            idleTime: 0,
            activityPeriods: 0
        };

        this.sequencePatterns = {
            mouseSpeedChanges: [],
            keyHoldDurations: [],
            scrollVelocities: [],
            clickIntervals: []
        };
    }

    startTracking() {
        this.startTime = Date.now();
        this.lastActivityTime = Date.now();
        this.mouseTrajectory = [];
        this.keyboardPattern = [];
        this.scrollBehavior = [];
        this.clickPattern = [];
        this.timingData = [];
        this.metrics = {
            totalTime: 0,
            mouseMovements: 0,
            keyboardInputs: 0,
            scrolls: 0,
            clicks: 0,
            idleTime: 0,
            activityPeriods: 0
        };
        this.sequencePatterns = {
            mouseSpeedChanges: [],
            keyHoldDurations: [],
            scrollVelocities: [],
            clickIntervals: []
        };

        this.trackMouseMovement();
        this.trackKeyboardInput();
        this.trackScrollBehavior();
        this.trackClickPattern();
        this.trackIdleTime();
    }

    trackMouseMovement() {
        let lastMoveTime = Date.now();
        let lastX = 0;
        let lastY = 0;
        let lastSpeed = 0;
        let activityStart = Date.now();

        const handleMouseMove = (e) => {
            if (this.mouseTrajectory.length >= this.options.maxTrajectoryPoints) {
                this.mouseTrajectory.shift();
            }

            const currentTime = Date.now();
            const timeDelta = currentTime - lastMoveTime;

            if (lastX !== 0 || lastY !== 0) {
                const distance = Math.sqrt(Math.pow(e.clientX - lastX, 2) + Math.pow(e.clientY - lastY, 2));
                const speed = timeDelta > 0 ? distance / timeDelta : 0;

                if (speed !== lastSpeed) {
                    this.sequencePatterns.mouseSpeedChanges.push({
                        from: lastSpeed,
                        to: speed,
                        time: currentTime
                    });
                    lastSpeed = speed;
                }

                this.mouseTrajectory.push({
                    x: e.clientX,
                    y: e.clientY,
                    time: currentTime,
                    timeDelta: timeDelta,
                    distance: distance,
                    speed: speed,
                    acceleration: this.calculateAcceleration(speed, timeDelta)
                });

                this.updateIdleStatus(currentTime);
            }

            lastX = e.clientX;
            lastY = e.clientY;
            lastMoveTime = currentTime;
        };

        document.addEventListener('mousemove', handleMouseMove, { passive: true });
    }

    trackKeyboardInput() {
        let lastKeyTime = Date.now();
        let keyDownTime = null;
        let lastKey = null;

        const handleKeyDown = (e) => {
            keyDownTime = Date.now();
            lastKey = e.key;

            const currentTime = Date.now();
            const delay = currentTime - lastKeyTime;

            this.keyboardPattern.push({
                key: e.key,
                code: e.code,
                time: currentTime,
                delay: delay,
                type: 'down'
            });

            this.updateIdleStatus(currentTime);
            lastKeyTime = currentTime;
        };

        const handleKeyUp = (e) => {
            if (keyDownTime !== null && e.key === lastKey) {
                const holdDuration = Date.now() - keyDownTime;
                this.sequencePatterns.keyHoldDurations.push({
                    key: e.key,
                    duration: holdDuration,
                    time: Date.now()
                });

                const lastPattern = this.keyboardPattern[this.keyboardPattern.length - 1];
                if (lastPattern && lastPattern.type === 'down' && lastPattern.key === e.key) {
                    lastPattern.holdDuration = holdDuration;
                    lastPattern.type = 'complete';
                }

                keyDownTime = null;
            }
        };

        document.addEventListener('keydown', handleKeyDown, { passive: true });
        document.addEventListener('keyup', handleKeyUp, { passive: true });
    }

    trackScrollBehavior() {
        let lastScrollTime = Date.now();
        let lastScrollY = window.scrollY;

        const handleScroll = () => {
            const currentTime = Date.now();
            const currentScrollY = window.scrollY;
            const scrollDelta = Math.abs(currentScrollY - lastScrollY);
            const timeDelta = currentTime - lastScrollTime;
            const velocity = timeDelta > 0 ? scrollDelta / timeDelta : 0;

            if (velocity > 0) {
                this.sequencePatterns.scrollVelocities.push({
                    velocity: velocity,
                    delta: scrollDelta,
                    time: currentTime
                });
            }

            this.scrollBehavior.push({
                scrollY: currentScrollY,
                delta: scrollDelta,
                time: currentTime,
                timeDelta: timeDelta,
                velocity: velocity
            });

            lastScrollY = currentScrollY;
            lastScrollTime = currentTime;

            this.updateIdleStatus(currentTime);
        };

        window.addEventListener('scroll', handleScroll, { passive: true });
    }

    trackClickPattern() {
        let lastClickTime = Date.now();

        const handleClick = (e) => {
            const currentTime = Date.now();
            const interval = currentTime - lastClickTime;

            this.sequencePatterns.clickIntervals.push({
                interval: interval,
                target: e.target.tagName,
                x: e.clientX,
                y: e.clientY,
                time: currentTime
            });

            this.clickPattern.push({
                x: e.clientX,
                y: e.clientY,
                target: e.target.tagName,
                targetClasses: e.target.className,
                time: currentTime,
                clickInterval: interval
            });

            lastClickTime = currentTime;

            this.updateIdleStatus(currentTime);
        };

        document.addEventListener('click', handleClick, { passive: true });
    }

    trackIdleTime() {
        const idleCheck = setInterval(() => {
            const currentTime = Date.now();
            const idleDuration = currentTime - this.lastActivityTime;

            if (idleDuration > 5000) {
                this.metrics.idleTime += idleDuration;
            }
        }, 1000);
    }

    updateIdleStatus(currentTime) {
        if (currentTime - this.lastActivityTime > 3000) {
            this.metrics.activityPeriods++;
        }
        this.lastActivityTime = currentTime;
    }

    calculateAcceleration(speed, timeDelta) {
        if (this.mouseTrajectory.length < 2) return 0;
        const prev = this.mouseTrajectory[this.mouseTrajectory.length - 2];
        if (!prev) return 0;
        const prevSpeed = prev.speed;
        const deltaTime = timeDelta / 1000;
        return deltaTime > 0 ? (speed - prevSpeed) / deltaTime : 0;
    }

    analyze() {
        const analysis = {
            totalTime: Date.now() - this.startTime,
            mouseMovements: this.mouseTrajectory.length,
            keyboardInputs: this.keyboardPattern.length,
            scrolls: this.scrollBehavior.length,
            clicks: this.clickPattern.length,
            idleTime: this.metrics.idleTime,
            activityPeriods: this.metrics.activityPeriods,
            effectiveActivityTime: 0,
            mouseMetrics: {},
            keyboardMetrics: {},
            scrollMetrics: {},
            clickMetrics: {}
        };

        analysis.effectiveActivityTime = analysis.totalTime - analysis.idleTime;

        if (this.mouseTrajectory.length > 0) {
            const speeds = this.mouseTrajectory.map(m => m.speed);
            const totalSpeed = speeds.reduce((sum, s) => sum + s, 0);
            analysis.mouseMetrics = {
                averageSpeed: totalSpeed / speeds.length,
                maxSpeed: Math.max(...speeds),
                minSpeed: Math.min(...speeds),
                totalDistance: this.mouseTrajectory.reduce((sum, m) => sum + m.distance, 0),
                trajectoryLength: this.mouseTrajectory.length,
                averageAcceleration: this.mouseTrajectory.reduce((sum, m) => sum + Math.abs(m.acceleration || 0), 0) / this.mouseTrajectory.length,
                directionChanges: this.countDirectionChanges(),
                smoothnessScore: this.calculateSmoothnessScore()
            };
        }

        if (this.keyboardPattern.length > 1) {
            const delays = this.keyboardPattern.slice(1).map(k => k.delay);
            analysis.keyboardMetrics = {
                averageKeyDelay: delays.reduce((sum, d) => sum + d, 0) / delays.length,
                maxKeyDelay: Math.max(...delays),
                minKeyDelay: Math.min(...delays),
                totalKeys: this.keyboardPattern.length,
                averageHoldDuration: this.calculateAverageHoldDuration(),
                typingRhythmVariance: this.calculateTypingRhythmVariance()
            };
        }

        if (this.scrollBehavior.length > 0) {
            const velocities = this.scrollBehavior.map(s => s.velocity);
            analysis.scrollMetrics = {
                totalScrolls: this.scrollBehavior.length,
                averageVelocity: velocities.reduce((sum, v) => sum + v, 0) / velocities.length,
                maxVelocity: Math.max(...velocities),
                totalScrollDistance: this.scrollBehavior.reduce((sum, s) => sum + s.delta, 0)
            };
        }

        if (this.clickPattern.length > 1) {
            const intervals = this.clickPattern.slice(1).map(c => c.clickInterval);
            analysis.clickMetrics = {
                totalClicks: this.clickPattern.length,
                averageInterval: intervals.reduce((sum, i) => sum + i, 0) / intervals.length,
                maxInterval: Math.max(...intervals),
                minInterval: Math.min(...intervals)
            };
        }

        analysis.patternSequences = {
            speedChangeCount: this.sequencePatterns.mouseSpeedChanges.length,
            keyHoldDurations: this.sequencePatterns.keyHoldDurations,
            scrollVelocities: this.sequencePatterns.scrollVelocities,
            clickIntervals: this.sequencePatterns.clickIntervals
        };

        return analysis;
    }

    countDirectionChanges() {
        if (this.mouseTrajectory.length < 3) return 0;

        let changes = 0;
        for (let i = 2; i < this.mouseTrajectory.length; i++) {
            const prev = this.mouseTrajectory[i - 2];
            const curr = this.mouseTrajectory[i - 1];
            const next = this.mouseTrajectory[i];

            const angle1 = Math.atan2(curr.y - prev.y, curr.x - prev.x);
            const angle2 = Math.atan2(next.y - curr.y, next.x - curr.x);

            const angleDiff = Math.abs(angle2 - angle1);
            if (angleDiff > Math.PI / 4) {
                changes++;
            }
        }

        return changes;
    }

    calculateSmoothnessScore() {
        if (this.mouseTrajectory.length < 3) return 100;

        let totalCurvature = 0;
        for (let i = 2; i < this.mouseTrajectory.length; i++) {
            const prev = this.mouseTrajectory[i - 2];
            const curr = this.mouseTrajectory[i - 1];
            const next = this.mouseTrajectory[i];

            const angle1 = Math.atan2(curr.y - prev.y, curr.x - prev.x);
            const angle2 = Math.atan2(next.y - curr.y, next.x - curr.x);

            let angleDiff = Math.abs(angle2 - angle1);
            if (angleDiff > Math.PI) {
                angleDiff = 2 * Math.PI - angleDiff;
            }

            totalCurvature += angleDiff;
        }

        const avgCurvature = totalCurvature / (this.mouseTrajectory.length - 2);
        const smoothness = Math.max(0, 100 - avgCurvature * 100);

        return Math.round(smoothness);
    }

    calculateAverageHoldDuration() {
        const completedKeys = this.keyboardPattern.filter(k => k.holdDuration !== undefined);
        if (completedKeys.length === 0) return 0;

        const totalDuration = completedKeys.reduce((sum, k) => sum + k.holdDuration, 0);
        return totalDuration / completedKeys.length;
    }

    calculateTypingRhythmVariance() {
        const delays = this.keyboardPattern.slice(1).map(k => k.delay);
        if (delays.length < 2) return 0;

        const mean = delays.reduce((sum, d) => sum + d, 0) / delays.length;
        const variance = delays.reduce((sum, d) => sum + Math.pow(d - mean, 2), 0) / delays.length;

        return Math.round(variance);
    }

    getBehaviorScore() {
        const analysis = this.analyze();
        let score = 100;

        if (!analysis.mouseMovements && !analysis.keyboardInputs && !analysis.clicks) {
            score -= 40;
        } else if (analysis.mouseMovements < 5) {
            score -= 15;
        }

        if (analysis.mouseMetrics.averageSpeed > 100) {
            score -= 25;
        } else if (analysis.mouseMetrics.averageSpeed > 50) {
            score -= 10;
        }

        if (analysis.totalTime < 1000) {
            score -= 30;
        } else if (analysis.totalTime < 2000) {
            score -= 15;
        }

        if (!analysis.keyboardInputs && !analysis.clicks) {
            score -= 20;
        }

        if (analysis.idleTime > analysis.totalTime * 0.5) {
            score -= 15;
        }

        if (analysis.mouseMetrics.smoothnessScore < 50) {
            score += 10;
        }

        if (analysis.mouseMetrics.directionChanges > analysis.mouseMovements * 0.3) {
            score -= 10;
        }

        if (analysis.keyboardMetrics.typingRhythmVariance > 10000) {
            score -= 15;
        }

        if (analysis.effectiveActivityTime > 10000) {
            score += 5;
        }

        return Math.max(0, Math.min(100, score));
    }

    detectAnomalyPatterns() {
        const analysis = this.analyze();
        const anomalies = [];

        if (analysis.totalTime < 500 && (analysis.mouseMovements > 0 || analysis.keyboardInputs > 0)) {
            anomalies.push({
                type: 'too_fast',
                severity: 'high',
                description: '行为发生时间过短，疑似自动化'
            });
        }

        if (analysis.mouseMetrics.maxSpeed > 200) {
            anomalies.push({
                type: 'unnatural_speed',
                severity: 'medium',
                description: '检测到异常移动速度'
            });
        }

        if (analysis.keyboardMetrics.typingRhythmVariance < 100 && analysis.keyboardInputs > 5) {
            anomalies.push({
                type: 'mechanical_typing',
                severity: 'high',
                description: '按键节奏过于规律，疑似机器'
            });
        }

        if (analysis.clickMetrics.maxInterval < 50 && analysis.clickMetrics.totalClicks > 3) {
            anomalies.push({
                type: 'rapid_clicks',
                severity: 'medium',
                description: '点击过于频繁'
            });
        }

        if (analysis.mouseMetrics.smoothnessScore > 95) {
            anomalies.push({
                type: 'too_smooth',
                severity: 'medium',
                description: '移动轨迹过于平滑'
            });
        }

        if (analysis.scrollMetrics.totalScrolls === 0 && analysis.totalTime > 5000) {
            anomalies.push({
                type: 'no_scrolling',
                severity: 'low',
                description: '长时间无滚动行为'
            });
        }

        return anomalies;
    }
}

class RiskScorer {
    constructor() {
        this.weights = {
            fingerprintEntropy: 0.25,
            behaviorScore: 0.30,
            deviceConsistency: 0.20,
            patternAnomaly: 0.15,
            environmentRisk: 0.10
        };

        this.riskThresholds = {
            low: 30,
            medium: 60,
            high: 80
        };
    }

    async calculateRiskScore(data) {
        const {
            fingerprint,
            behaviorAnalysis,
            deviceHistory,
            environmentData
        } = data;

        const riskFactors = {
            fingerprintEntropy: await this.calculateFingerprintEntropy(fingerprint),
            behaviorScore: this.calculateBehaviorRisk(behaviorAnalysis),
            deviceConsistency: this.calculateDeviceConsistency(deviceHistory),
            patternAnomaly: this.calculatePatternAnomaly(behaviorAnalysis),
            environmentRisk: this.calculateEnvironmentRisk(environmentData)
        };

        const weightedScore =
            riskFactors.fingerprintEntropy * this.weights.fingerprintEntropy +
            riskFactors.behaviorScore * this.weights.behaviorScore +
            riskFactors.deviceConsistency * this.weights.deviceConsistency +
            riskFactors.patternAnomaly * this.weights.patternAnomaly +
            riskFactors.environmentRisk * this.weights.environmentRisk;

        return {
            totalScore: Math.round(weightedScore),
            riskFactors: riskFactors,
            level: this.getRiskLevel(weightedScore),
            details: this.getRiskDetails(riskFactors)
        };
    }

    calculateFingerprintEntropy(fingerprint) {
        if (!fingerprint) return 0;

        let entropy = 0;
        const features = [
            fingerprint.canvasFingerprint,
            fingerprint.webglFingerprint,
            fingerprint.audioFingerprint,
            fingerprint.fontFingerprint,
            fingerprint.screenResolution,
            fingerprint.timezone,
            fingerprint.language,
            fingerprint.platform,
            fingerprint.connectionType,
            fingerprint.batteryStatus,
            fingerprint.storageQuota,
            fingerprint.mediaDevices
        ];

        const validFeatures = features.filter(f => {
            if (typeof f === 'string') return f && f !== 'error' && f !== 'unsupported';
            if (typeof f === 'object') return f && f.supported !== false;
            return f !== null && f !== undefined;
        });

        entropy = (validFeatures.length / features.length) * 100;

        if (fingerprint.webdriverStatus) {
            entropy -= 30;
        }

        return Math.max(0, Math.min(100, entropy));
    }

    calculateBehaviorRisk(behaviorAnalysis) {
        if (!behaviorAnalysis) return 100;

        let risk = 0;

        if (behaviorAnalysis.anomalies && behaviorAnalysis.anomalies.length > 0) {
            for (const anomaly of behaviorAnalysis.anomalies) {
                switch (anomaly.severity) {
                    case 'high':
                        risk += 30;
                        break;
                    case 'medium':
                        risk += 15;
                        break;
                    case 'low':
                        risk += 5;
                        break;
                }
            }
        }

        const metrics = behaviorAnalysis.mouseMetrics || {};
        if (metrics.maxSpeed > 100) risk += 15;
        if (metrics.smoothnessScore > 95) risk += 10;

        const keyboardMetrics = behaviorAnalysis.keyboardMetrics || {};
        if (keyboardMetrics.typingRhythmVariance < 500 && keyboardMetrics.totalKeys > 5) {
            risk += 20;
        }

        return Math.max(0, Math.min(100, risk));
    }

    calculateDeviceConsistency(deviceHistory) {
        if (!deviceHistory) return 50;

        let consistencyScore = 50;

        if (deviceHistory.isKnownDevice) {
            consistencyScore += 30;
        }

        if (deviceHistory.visits > 5) {
            consistencyScore += 10;
        }

        if (deviceHistory.lastVisit && Date.now() - deviceHistory.lastVisit < 7 * 24 * 60 * 60 * 1000) {
            consistencyScore += 10;
        }

        if (deviceHistory.fingerprintChanged) {
            consistencyScore -= 40;
        }

        return Math.max(0, Math.min(100, consistencyScore));
    }

    calculatePatternAnomaly(behaviorAnalysis) {
        if (!behaviorAnalysis) return 0;

        let anomalyScore = 0;

        const anomalies = behaviorAnalysis.anomalies || [];
        for (const anomaly of anomalies) {
            switch (anomaly.type) {
                case 'too_fast':
                    anomalyScore += 40;
                    break;
                case 'mechanical_typing':
                    anomalyScore += 35;
                    break;
                case 'unnatural_speed':
                    anomalyScore += 20;
                    break;
                case 'rapid_clicks':
                    anomalyScore += 15;
                    break;
                case 'too_smooth':
                    anomalyScore += 10;
                    break;
                case 'no_scrolling':
                    anomalyScore += 5;
                    break;
            }
        }

        return Math.min(100, anomalyScore);
    }

    calculateEnvironmentRisk(environmentData) {
        if (!environmentData) return 30;

        let risk = 0;

        if (environmentData.isEmulator) risk += 50;
        if (environmentData.isVirtual) risk += 40;
        if (environmentData.isContainer) risk += 30;
        if (environmentData.isMultiBox) risk += 35;

        if (environmentData.isHeadlessBrowser) risk += 45;

        return Math.min(100, risk);
    }

    getRiskLevel(score) {
        if (score >= this.riskThresholds.high) return 'high';
        if (score >= this.riskThresholds.medium) return 'medium';
        return 'low';
    }

    getRiskDetails(riskFactors) {
        const details = [];

        if (riskFactors.fingerprintEntropy < 50) {
            details.push('设备指纹信息不完整');
        }

        if (riskFactors.behaviorScore > 50) {
            details.push('检测到可疑行为模式');
        }

        if (riskFactors.deviceConsistency < 30) {
            details.push('设备历史信息不一致');
        }

        if (riskFactors.patternAnomaly > 40) {
            details.push('存在异常行为特征');
        }

        if (riskFactors.environmentRisk > 30) {
            details.push('检测到可疑运行环境');
        }

        return details;
    }
}

class DeviceTrustManager {
    constructor() {
        this.storageKeys = {
            fingerprintHash: 'hjtpx_fp_hash',
            trustStatus: 'hjtpx_trust_status',
            trustExpiry: 'hjtpx_trust_expiry',
            deviceId: 'hjtpx_device_id',
            lastVerification: 'hjtpx_last_verify',
            visitCount: 'hjtpx_visit_count',
            trustHistory: 'hjtpx_trust_history'
        };

        this.defaultTrustExpiry = 30 * 24 * 60 * 60 * 1000;
        this.minTrustLevel = 60;
    }

    setTrustExpiry(days) {
        if (days && days > 0) {
            this.defaultTrustExpiry = days * 24 * 60 * 60 * 1000;
        }
    }

    getTrustStatus() {
        const status = localStorage.getItem(this.storageKeys.trustStatus);
        const expiry = localStorage.getItem(this.storageKeys.trustExpiry);
        const fpHash = localStorage.getItem(this.storageKeys.fingerprintHash);

        if (!status || !expiry || !fpHash) {
            return {
                isTrusted: false,
                isExpired: true,
                reason: 'No trust record'
            };
        }

        const now = Date.now();
        const expiryTime = parseInt(expiry, 10);

        if (now > expiryTime) {
            return {
                isTrusted: false,
                isExpired: true,
                reason: 'Trust expired'
            };
        }

        return {
            isTrusted: status === 'true',
            isExpired: false,
            expiryTime: expiryTime,
            remainingTime: expiryTime - now,
            fingerprintHash: fpHash
        };
    }

    setTrust(trustLevel, expiryDays = null) {
        const expiry = expiryDays
            ? Date.now() + (expiryDays * 24 * 60 * 60 * 1000)
            : Date.now() + this.defaultTrustExpiry;

        localStorage.setItem(this.storageKeys.trustStatus, 'true');
        localStorage.setItem(this.storageKeys.trustExpiry, expiry.toString());
        localStorage.setItem(this.storageKeys.lastVerification, Date.now().toString());

        this.incrementVisitCount();

        this.addToTrustHistory({
            action: 'trust_set',
            level: trustLevel,
            timestamp: Date.now(),
            expiry: expiry
        });

        return {
            success: true,
            expiryTime: expiry
        };
    }

    revokeTrust() {
        localStorage.setItem(this.storageKeys.trustStatus, 'false');
        localStorage.removeItem(this.storageKeys.trustExpiry);

        this.addToTrustHistory({
            action: 'trust_revoked',
            timestamp: Date.now()
        });

        return { success: true };
    }

    updateTrustLevel(level) {
        const currentExpiry = localStorage.getItem(this.storageKeys.trustExpiry);
        if (currentExpiry) {
            localStorage.setItem(this.storageKeys.lastVerification, Date.now().toString());

            this.addToTrustHistory({
                action: 'trust_updated',
                level: level,
                timestamp: Date.now()
            });
        }

        return { success: true };
    }

    checkDeviceConsistency(currentFingerprintHash) {
        const storedHash = localStorage.getItem(this.storageKeys.fingerprintHash);

        if (!storedHash) {
            localStorage.setItem(this.storageKeys.fingerprintHash, currentFingerprintHash);
            return {
                consistent: true,
                isNewDevice: true
            };
        }

        if (storedHash !== currentFingerprintHash) {
            return {
                consistent: false,
                isNewDevice: false,
                storedHash: storedHash,
                currentHash: currentFingerprintHash
            };
        }

        return {
            consistent: true,
            isNewDevice: false
        };
    }

    incrementVisitCount() {
        const currentCount = parseInt(localStorage.getItem(this.storageKeys.visitCount) || '0', 10);
        localStorage.setItem(this.storageKeys.visitCount, (currentCount + 1).toString());
    }

    getVisitCount() {
        return parseInt(localStorage.getItem(this.storageKeys.visitCount) || '0', 10);
    }

    getLastVerificationTime() {
        const time = localStorage.getItem(this.storageKeys.lastVerification);
        return time ? parseInt(time, 10) : null;
    }

    getDeviceId() {
        let deviceId = localStorage.getItem(this.storageKeys.deviceId);
        if (!deviceId) {
            deviceId = 'device_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
            localStorage.setItem(this.storageKeys.deviceId, deviceId);
        }
        return deviceId;
    }

    addToTrustHistory(entry) {
        try {
            const historyStr = localStorage.getItem(this.storageKeys.trustHistory);
            let history = historyStr ? JSON.parse(historyStr) : [];

            history.unshift(entry);

            if (history.length > 50) {
                history = history.slice(0, 50);
            }

            localStorage.setItem(this.storageKeys.trustHistory, JSON.stringify(history));
        } catch (e) {
            console.error('Failed to add trust history:', e);
        }
    }

    getTrustHistory(limit = 10) {
        try {
            const historyStr = localStorage.getItem(this.storageKeys.trustHistory);
            const history = historyStr ? JSON.parse(historyStr) : [];
            return history.slice(0, limit);
        } catch (e) {
            return [];
        }
    }

    clearAllData() {
        Object.values(this.storageKeys).forEach(key => {
            localStorage.removeItem(key);
        });
    }

    exportDeviceData() {
        const data = {};
        Object.keys(this.storageKeys).forEach(key => {
            const value = localStorage.getItem(this.storageKeys[key]);
            if (value) {
                data[key] = value;
            }
        });
        return data;
    }

    importDeviceData(data) {
        if (!data || typeof data !== 'object') {
            return { success: false, error: 'Invalid data format' };
        }

        Object.keys(data).forEach(key => {
            if (this.storageKeys[key]) {
                localStorage.setItem(this.storageKeys[key], data[key]);
            }
        });

        return { success: true };
    }
}

class SeamlessI18n {
    constructor(locale = 'zh-CN') {
        this.locale = locale;
        this.translations = {
            'zh-CN': {
                checking: '正在检测设备...',
                fingerprinting: '采集设备指纹...',
                analyzing: '分析行为模式...',
                scoring: '计算风险评分...',
                trusted: '设备可信',
                untrusted: '设备不可信',
                pending: '等待验证',
                error: '验证出错',
                retry: '重试',
                featureCount: '已采集特征',
                deviceTrusted: '设备已信任',
                deviceUntrusted: '设备不可信',
                trustExpiring: '信任即将过期',
                verificationRequired: '需要验证',
                lowRisk: '低风险',
                mediumRisk: '中风险',
                highRisk: '高风险'
            },
            'en-US': {
                checking: 'Checking device...',
                fingerprinting: 'Collecting fingerprint...',
                analyzing: 'Analyzing behavior...',
                scoring: 'Calculating risk score...',
                trusted: 'Device Trusted',
                untrusted: 'Device Untrusted',
                pending: 'Pending Verification',
                error: 'Verification Error',
                retry: 'Retry',
                featureCount: 'Features collected',
                deviceTrusted: 'Device is trusted',
                deviceUntrusted: 'Device is not trusted',
                trustExpiring: 'Trust expiring soon',
                verificationRequired: 'Verification required',
                lowRisk: 'Low Risk',
                mediumRisk: 'Medium Risk',
                highRisk: 'High Risk'
            }
        };
    }

    t(key) {
        return this.translations[this.locale]?.[key] || this.translations['zh-CN'][key] || key;
    }

    setLocale(locale) {
        this.locale = locale;
    }

    getLocale() {
        return this.locale;
    }
}

class SeamlessCaptcha {
    constructor(containerId, options = {}) {
        this.container = document.getElementById(containerId);
        if (!this.container) {
            console.error('SeamlessCaptcha container not found');
            return;
        }

        this.options = {
            apiBase: '/api/v1',
            language: 'zh-CN',
            autoStart: false,
            showProgress: true,
            showFingerprint: true,
            trackingDuration: 3000,
            enableAdvancedMetrics: true,
            trustExpiryDays: 30,
            riskThresholdLow: 30,
            riskThresholdMedium: 60,
            riskThresholdHigh: 80,
            onStatusChange: null,
            onStepChange: null,
            onTrustChange: null,
            onComplete: null,
            onError: null,
            onRiskCalculated: null,
            ...options
        };

        this.fingerprintCollector = new FingerprintCollector();
        this.behaviorAnalyzer = new BehaviorAnalyzer({
            trackingDuration: this.options.trackingDuration,
            enableAdvancedMetrics: this.options.enableAdvancedMetrics
        });
        this.riskScorer = new RiskScorer();
        this.deviceTrustManager = new DeviceTrustManager();
        this.deviceTrustManager.setTrustExpiry(this.options.trustExpiryDays);
        this.i18n = new SeamlessI18n(this.options.language);

        this.state = {
            status: 'idle',
            fingerprint: null,
            behaviorAnalysis: null,
            riskResult: null,
            trustStatus: 'pending',
            trustLevel: 0,
            sessionId: null,
            isVerifying: false,
            verificationAttempts: 0,
            featureCount: 0
        };

        this.verificationMode = 'seamless';
        this.config = {
            seamlessEnabled: true,
            forceVerificationThreshold: 80,
            requireBehaviorAnalysis: true,
            enableAutoTrust: true
        };

        this.init();
    }

    init() {
        if (this.options.autoStart) {
            this.start();
        }
    }

    async start(mode = 'seamless') {
        if (this.state.isVerifying) {
            return;
        }

        this.verificationMode = mode;
        this.state.isVerifying = true;
        this.state.verificationAttempts++;
        this.updateStatus('checking');

        try {
            await this.checkPreVerification();

            await this.collectFingerprint();
            await this.analyzeBehavior();
            await this.calculateRiskScore();
            await this.verifyWithServer();

            this.state.isVerifying = false;
            this.updateStatus(this.state.trustStatus);

        } catch (error) {
            console.error('SeamlessCaptcha verification error:', error);
            this.state.isVerifying = false;
            this.handleError(error);
        }
    }

    async checkPreVerification() {
        const trustStatus = this.deviceTrustManager.getTrustStatus();
        const consistencyCheck = this.deviceTrustManager.checkDeviceConsistency(
            this.fingerprintCollector.getFingerprintHash()
        );

        if (trustStatus.isTrusted && consistencyCheck.consistent) {
            this.state.trustStatus = 'trusted';
            this.state.trustLevel = 100;
            throw new Error('DEVICE_TRUSTED');
        }

        if (!consistencyCheck.consistent) {
            this.deviceTrustManager.revokeTrust();
            this.state.trustStatus = 'untrusted';
        }
    }

    async collectFingerprint() {
        this.updateStatus('fingerprinting');
        this.updateStep('fingerprint', 25);

        try {
            this.state.fingerprint = await this.fingerprintCollector.collect();
            this.state.featureCount = this.fingerprintCollector.getFeatureCount();

            if (this.options.showFingerprint) {
                this.displayFingerprintInfo();
            }

            return this.state.fingerprint;
        } catch (error) {
            console.error('Fingerprint collection error:', error);
            throw error;
        }
    }

    async analyzeBehavior() {
        this.updateStatus('analyzing');
        this.updateStep('behavior', 50);

        try {
            this.behaviorAnalyzer.startTracking();

            await new Promise(resolve => setTimeout(resolve, this.options.trackingDuration));

            this.state.behaviorAnalysis = this.behaviorAnalyzer.analyze();
            this.state.behaviorAnalysis.anomalies = this.behaviorAnalyzer.detectAnomalyPatterns();

            return this.state.behaviorAnalysis;
        } catch (error) {
            console.error('Behavior analysis error:', error);
            throw error;
        }
    }

    async calculateRiskScore() {
        this.updateStatus('scoring');
        this.updateStep('risk', 75);

        try {
            const riskResult = await this.riskScorer.calculateRiskScore({
                fingerprint: this.state.fingerprint,
                behaviorAnalysis: this.state.behaviorAnalysis,
                deviceHistory: this.getDeviceHistory(),
                environmentData: this.getEnvironmentData()
            });

            this.state.riskResult = riskResult;
            this.state.trustLevel = 100 - riskResult.totalScore;

            if (this.options.onRiskCalculated) {
                this.options.onRiskCalculated(riskResult);
            }

            return riskResult;
        } catch (error) {
            console.error('Risk calculation error:', error);
            throw error;
        }
    }

    getDeviceHistory() {
        return {
            isKnownDevice: this.deviceTrustManager.getVisitCount() > 0,
            visits: this.deviceTrustManager.getVisitCount(),
            lastVisit: this.deviceTrustManager.getLastVerificationTime(),
            fingerprintChanged: false
        };
    }

    getEnvironmentData() {
        return {
            isEmulator: this.checkEmulator(),
            isVirtual: this.checkVirtual(),
            isContainer: this.checkContainer(),
            isHeadlessBrowser: this.checkHeadlessBrowser()
        };
    }

    checkEmulator() {
        const ua = navigator.userAgent.toLowerCase();
        return /android.*emulator|iphone.*simulator|ipad.*simulator/.test(ua);
    }

    checkVirtual() {
        const ua = navigator.userAgent;
        return /vmware|virtualbox|qemu|kvm/.test(ua.toLowerCase());
    }

    checkContainer() {
        try {
            return document.cookie === '' &&
                   navigator.userAgent.includes('node') === false &&
                   navigator.plugins.length === 0;
        } catch (e) {
            return false;
        }
    }

    checkHeadlessBrowser() {
        return window.navigator.webdriver === true ||
               navigator.userAgent.includes('HeadlessChrome') ||
               navigator.userAgent.includes('PhantomJS');
    }

    async verifyWithServer() {
        this.updateStep('result', 90);

        try {
            const response = await this.apiCall('/seamless/check-status', 'POST', {
                fingerprint: this.state.fingerprint,
                behavior_analysis: this.state.behaviorAnalysis,
                risk_score: this.state.riskResult,
                session_id: this.state.sessionId,
                verification_mode: this.verificationMode
            });

            if (response.success) {
                this.handleVerificationSuccess(response);
            } else {
                throw new Error(response.message || 'Verification failed');
            }
        } catch (error) {
            console.error('Server verification error:', error);

            if (error.message === 'DEVICE_TRUSTED') {
                this.handleTrustedDevice();
            } else {
                this.handleVerificationFailure(error);
            }
        }
    }

    handleVerificationSuccess(response) {
        this.state.trustStatus = response.trusted ? 'trusted' : 'untrusted';
        this.state.trustLevel = response.trust_level || (100 - this.state.riskResult.totalScore);
        this.state.sessionId = response.session_id;

        localStorage.setItem('hjtpx_fp_hash', this.fingerprintCollector.getFingerprintHash());

        if (response.trusted && this.options.enableAutoTrust) {
            this.deviceTrustManager.setTrust(this.state.trustLevel, this.options.trustExpiryDays);
        }

        this.updateStatus(this.state.trustStatus);
        this.updateStep('result', 100);

        if (this.options.onTrustChange) {
            this.options.onTrustChange(
                this.state.trustStatus === 'trusted',
                this.state.trustLevel
            );
        }

        if (this.options.onComplete) {
            this.options.onComplete({
                success: true,
                trusted: this.state.trustStatus === 'trusted',
                trustLevel: this.state.trustLevel,
                sessionId: this.state.sessionId,
                riskResult: this.state.riskResult,
                fingerprint: this.state.fingerprint,
                behaviorAnalysis: this.state.behaviorAnalysis,
                featureCount: this.state.featureCount
            });
        }
    }

    handleTrustedDevice() {
        this.state.trustStatus = 'trusted';
        this.state.trustLevel = 100;
        this.updateStatus('trusted');
        this.updateStep('result', 100);

        if (this.options.onTrustChange) {
            this.options.onTrustChange(true, 100);
        }

        if (this.options.onComplete) {
            this.options.onComplete({
                success: true,
                trusted: true,
                trustLevel: 100,
                cached: true,
                message: 'Device already trusted'
            });
        }
    }

    handleVerificationFailure(error) {
        this.state.trustStatus = 'pending';
        this.state.trustLevel = 0;
        this.updateStatus('pending');

        if (this.options.onComplete) {
            this.options.onComplete({
                success: false,
                trusted: false,
                error: error
            });
        }
    }

    async trustDevice(expiryDays = null) {
        if (this.state.trustStatus !== 'trusted') {
            console.warn('Cannot trust unverified device');
            return false;
        }

        try {
            const response = await this.apiCall('/seamless/trust-device', 'POST', {
                fingerprint: this.state.fingerprint,
                session_id: this.state.sessionId,
                trust_level: this.state.trustLevel
            });

            if (response.success) {
                this.deviceTrustManager.setTrust(this.state.trustLevel, expiryDays || this.options.trustExpiryDays);
                return true;
            }

            return false;
        } catch (error) {
            console.error('Trust device error:', error);
            return false;
        }
    }

    async checkStatus() {
        const trustStatus = this.deviceTrustManager.getTrustStatus();
        const currentHash = this.fingerprintCollector.getFingerprintHash();
        const consistency = this.deviceTrustManager.checkDeviceConsistency(currentHash);

        if (trustStatus.isTrusted && consistency.consistent) {
            return {
                trusted: true,
                trustLevel: 100,
                cached: true,
                remainingTime: trustStatus.remainingTime
            };
        }

        return {
            trusted: false,
            trustLevel: 0,
            cached: false,
            reason: consistency.consistent ? 'Trust expired' : 'Device fingerprint changed'
        };
    }

    updateConfig(newConfig) {
        this.config = { ...this.config, ...newConfig };
    }

    setVerificationMode(mode) {
        this.verificationMode = mode;
    }

    shouldForceVerification() {
        if (!this.config.seamlessEnabled) {
            return true;
        }

        if (this.state.riskResult && this.state.riskResult.totalScore >= this.config.forceVerificationThreshold) {
            return true;
        }

        return false;
    }

    reset() {
        this.state = {
            status: 'idle',
            fingerprint: null,
            behaviorAnalysis: null,
            riskResult: null,
            trustStatus: 'pending',
            trustLevel: 0,
            sessionId: null,
            isVerifying: false,
            verificationAttempts: 0,
            featureCount: 0
        };

        this.updateStatus('checking');
        this.updateStep('fingerprint', 0);
    }

    updateStatus(status) {
        this.state.status = status;

        if (this.options.onStatusChange) {
            this.options.onStatusChange(status);
        }
    }

    updateStep(step, progress) {
        if (this.options.onStepChange) {
            this.options.onStepChange(step, progress);
        }
    }

    handleError(error) {
        if (this.options.onError) {
            this.options.onError({
                title: this.i18n.t('error'),
                message: error.message || 'Unknown error occurred',
                code: error.code || 'UNKNOWN_ERROR'
            });
        }

        if (this.options.onComplete) {
            this.options.onComplete({
                success: false,
                trusted: false,
                error: error
            });
        }
    }

    displayFingerprintInfo() {
        const fp = this.state.fingerprint;
        if (!fp) return;

        const deviceEl = document.getElementById('fpDevice');
        const browserEl = document.getElementById('fpBrowser');
        const locationEl = document.getElementById('fpLocation');
        const timezoneEl = document.getElementById('fpTimezone');

        if (deviceEl) {
            deviceEl.textContent = `${fp.screenResolution} (${fp.platform})`;
        }

        if (browserEl) {
            const ua = fp.userAgent;
            let browser = 'Unknown Browser';
            if (ua.indexOf('Firefox') > -1) browser = 'Firefox';
            else if (ua.indexOf('Chrome') > -1) browser = 'Chrome';
            else if (ua.indexOf('Safari') > -1) browser = 'Safari';
            else if (ua.indexOf('Edge') > -1) browser = 'Edge';
            browserEl.textContent = browser;
        }

        if (timezoneEl) {
            timezoneEl.textContent = fp.timezone;
        }

        if (locationEl) {
            locationEl.textContent = fp.language;
        }
    }

    async apiCall(endpoint, method = 'GET', data = null) {
        const url = this.options.apiBase + endpoint;

        const options = {
            method: method,
            headers: {
                'Content-Type': 'application/json'
            },
            credentials: 'same-origin'
        };

        if (data && method !== 'GET') {
            options.body = JSON.stringify(data);
        }

        try {
            const response = await fetch(url, options);

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            return result;
        } catch (error) {
            console.error('API call error:', error);

            return {
                success: true,
                trusted: this.state.trustLevel >= this.options.riskThresholdLow,
                trust_level: this.state.trustLevel,
                session_id: 'demo_' + Date.now(),
                message: 'Demo mode - using client-side risk score'
            };
        }
    }

    updateLanguage(locale) {
        this.i18n.setLocale(locale);
        this.options.language = locale;
    }

    getState() {
        return { ...this.state };
    }

    getDeviceTrustManager() {
        return this.deviceTrustManager;
    }

    destroy() {
        this.reset();
        this.container = null;
        this.options = {};
        this.fingerprintCollector = null;
        this.behaviorAnalyzer = null;
        this.riskScorer = null;
        this.deviceTrustManager = null;
    }
}

class SeamlessVerificationManager {
    constructor(options = {}) {
        this.options = {
            apiBase: '/api/v1',
            seamlessEnabled: true,
            forceVerificationThreshold: 80,
            requireBehaviorAnalysis: true,
            enableAutoTrust: true,
            trustExpiryDays: 30,
            onVerificationRequired: null,
            onVerificationComplete: null,
            ...options
        };

        this.deviceTrustManager = new DeviceTrustManager();
        this.riskScorer = new RiskScorer();
        this.fingerprintCollector = new FingerprintCollector();
        this.behaviorAnalyzer = new BehaviorAnalyzer();
    }

    async checkVerificationRequired(context = {}) {
        const trustStatus = this.deviceTrustManager.getTrustStatus();

        if (trustStatus.isTrusted && !trustStatus.isExpired) {
            return {
                required: false,
                reason: 'device_trusted',
                remainingTime: trustStatus.remainingTime
            };
        }

        return {
            required: true,
            reason: trustStatus.isExpired ? 'trust_expired' : 'device_not_trusted'
        };
    }

    async performSeamlessVerification(context = {}) {
        const fingerprint = await this.fingerprintCollector.collect();
        const behaviorData = this.behaviorAnalyzer.analyze();

        const deviceHistory = {
            isKnownDevice: this.deviceTrustManager.getVisitCount() > 0,
            visits: this.deviceTrustManager.getVisitCount(),
            lastVisit: this.deviceTrustManager.getLastVerificationTime(),
            fingerprintChanged: false
        };

        const riskResult = await this.riskScorer.calculateRiskScore({
            fingerprint: fingerprint,
            behaviorAnalysis: behaviorData,
            deviceHistory: deviceHistory,
            environmentData: context.environmentData || {}
        });

        if (riskResult.totalScore < this.options.forceVerificationThreshold) {
            this.deviceTrustManager.setTrust(100 - riskResult.totalScore, this.options.trustExpiryDays);

            return {
                verified: true,
                trusted: true,
                riskResult: riskResult,
                mode: 'seamless'
            };
        }

        return {
            verified: false,
            trusted: false,
            riskResult: riskResult,
            mode: 'seamless',
            requiresAdditionalVerification: true
        };
    }

    setSeamlessEnabled(enabled) {
        this.options.seamlessEnabled = enabled;
    }

    setTrustExpiryDays(days) {
        this.options.trustExpiryDays = days;
        this.deviceTrustManager.setTrustExpiry(days);
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        SeamlessCaptcha,
        FingerprintCollector,
        BehaviorAnalyzer,
        RiskScorer,
        DeviceTrustManager,
        SeamlessI18n,
        SeamlessVerificationManager
    };
}
