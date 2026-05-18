class FingerprintCollector {
    constructor() {
        this.fingerprint = {};
        this.components = [];
        this.confidenceScore = 0;
    }

    async collect() {
        this.components = [];
        this.confidenceScore = 0;

        this.fingerprint.userAgent = this.getUserAgent();
        this.components.push({ key: 'userAgent', value: this.fingerprint.userAgent, entropy: 5 });

        this.fingerprint.screenResolution = this.getScreenResolution();
        this.components.push({ key: 'screenResolution', value: this.fingerprint.screenResolution, entropy: 10 });

        this.fingerprint.colorDepth = this.getColorDepth();
        this.components.push({ key: 'colorDepth', value: this.fingerprint.colorDepth, entropy: 3 });

        this.fingerprint.timezone = this.getTimezone();
        this.components.push({ key: 'timezone', value: this.fingerprint.timezone, entropy: 5 });

        this.fingerprint.timezoneOffset = this.getTimezoneOffset();
        this.components.push({ key: 'timezoneOffset', value: this.fingerprint.timezoneOffset, entropy: 4 });

        this.fingerprint.language = this.getLanguage();
        this.components.push({ key: 'language', value: this.fingerprint.language, entropy: 5 });

        this.fingerprint.languages = this.getLanguages();
        this.components.push({ key: 'languages', value: this.fingerprint.languages, entropy: 6 });

        this.fingerprint.platform = this.getPlatform();
        this.components.push({ key: 'platform', value: this.fingerprint.platform, entropy: 4 });

        this.fingerprint.cookieEnabled = this.getCookieEnabled();
        this.components.push({ key: 'cookieEnabled', value: this.fingerprint.cookieEnabled, entropy: 2 });

        this.fingerprint.canvasFingerprint = await this.getCanvasFingerprint();
        this.components.push({ key: 'canvasFingerprint', value: this.fingerprint.canvasFingerprint, entropy: 20 });

        this.fingerprint.webglFingerprint = this.getWebGLFingerprint();
        this.components.push({ key: 'webglFingerprint', value: this.fingerprint.webglFingerprint, entropy: 15 });

        this.fingerprint.webglParams = this.getWebGLParams();
        this.components.push({ key: 'webglParams', value: this.fingerprint.webglParams, entropy: 12 });

        this.fingerprint.audioFingerprint = await this.getAudioFingerprint();
        this.components.push({ key: 'audioFingerprint', value: this.fingerprint.audioFingerprint, entropy: 10 });

        this.fingerprint.fontFingerprint = await this.getFontFingerprint();
        this.components.push({ key: 'fontFingerprint', value: this.fingerprint.fontFingerprint, entropy: 15 });

        this.fingerprint.plugins = this.getPlugins();
        this.components.push({ key: 'plugins', value: this.fingerprint.plugins, entropy: 8 });

        this.fingerprint.doNotTrack = this.getDoNotTrack();
        this.components.push({ key: 'doNotTrack', value: this.fingerprint.doNotTrack, entropy: 2 });

        this.fingerprint.touchSupport = this.getTouchSupport();
        this.components.push({ key: 'touchSupport', value: JSON.stringify(this.fingerprint.touchSupport), entropy: 5 });

        this.fingerprint.deviceMemory = this.getDeviceMemory();
        this.components.push({ key: 'deviceMemory', value: this.fingerprint.deviceMemory, entropy: 4 });

        this.fingerprint.hardwareConcurrency = this.getHardwareConcurrency();
        this.components.push({ key: 'hardwareConcurrency', value: this.fingerprint.hardwareConcurrency, entropy: 5 });

        this.fingerprint.connectionType = await this.getConnectionType();
        this.components.push({ key: 'connectionType', value: this.fingerprint.connectionType, entropy: 3 });

        this.fingerprint.batteryStatus = await this.getBatteryStatus();
        this.components.push({ key: 'batteryStatus', value: this.fingerprint.batteryStatus, entropy: 4 });

        this.fingerprint.mediaDevices = await this.getMediaDevices();
        this.components.push({ key: 'mediaDevices', value: this.fingerprint.mediaDevices, entropy: 6 });

        this.fingerprint.performanceMetrics = this.getPerformanceMetrics();
        this.components.push({ key: 'performanceMetrics', value: this.fingerprint.performanceMetrics, entropy: 8 });

        this.fingerprint.videoCard = this.getVideoCard();
        this.components.push({ key: 'videoCard', value: this.fingerprint.videoCard, entropy: 10 });

        this.fingerprint.cpuClass = this.getCpuClass();
        this.components.push({ key: 'cpuClass', value: this.fingerprint.cpuClass, entropy: 4 });

        this.fingerprint.browserPlugins = this.getBrowserPlugins();
        this.components.push({ key: 'browserPlugins', value: this.fingerprint.browserPlugins, entropy: 5 });

        this.fingerprint.sessionStorage = this.checkSessionStorage();
        this.components.push({ key: 'sessionStorage', value: this.fingerprint.sessionStorage, entropy: 2 });

        this.fingerprint.localStorage = this.checkLocalStorage();
        this.components.push({ key: 'localStorage', value: this.fingerprint.localStorage, entropy: 2 });

        this.fingerprint.indexedDB = this.checkIndexedDB();
        this.components.push({ key: 'indexedDB', value: this.fingerprint.indexedDB, entropy: 2 });

        this.fingerprint.timestamp = Date.now();
        this.components.push({ key: 'timestamp', value: this.fingerprint.timestamp, entropy: 1 });

        this.fingerprint.hash = await this.generateHash();
        this.confidenceScore = this.calculateConfidenceScore();

        this.fingerprint.confidenceScore = this.confidenceScore;

        return this.fingerprint;
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

    getTimezoneOffset() {
        return new Date().getTimezoneOffset();
    }

    getLanguage() {
        return navigator.language || navigator.userLanguage;
    }

    getLanguages() {
        if (navigator.languages && navigator.languages.length > 0) {
            return navigator.languages.join(',');
        }
        return this.getLanguage();
    }

    getPlatform() {
        return navigator.platform;
    }

    getCookieEnabled() {
        return navigator.cookieEnabled ? '1' : '0';
    }

    async getCanvasFingerprint() {
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
            ctx.fillText('C2#IOJS@√M', 2, 15);
            ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
            ctx.fillText('Seamless Captcha FP', 4, 17);

            ctx.beginPath();
            ctx.arc(50, 50, 30, 0, Math.PI * 2, true);
            ctx.closePath();
            ctx.strokeStyle = '#ff0000';
            ctx.lineWidth = 2;
            ctx.stroke();

            ctx.save();
            ctx.rotate(Math.PI / 6);
            ctx.fillStyle = '#0000ff';
            ctx.fillRect(150, 20, 50, 25);
            ctx.restore();

            const gradient = ctx.createLinearGradient(0, 0, 200, 0);
            gradient.addColorStop(0, 'rgba(255, 0, 0, 0.5)');
            gradient.addColorStop(1, 'rgba(0, 255, 0, 0.5)');
            ctx.fillStyle = gradient;
            ctx.fillRect(200, 35, 70, 20);

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

            return `${vendor}~${renderer}`.substring(0, 100);
        } catch (e) {
            return 'webgl-error';
        }
    }

    getWebGLParams() {
        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
            if (!gl) return 'unsupported';

            const params = {
                antialias: gl.getContextAttributes()?.antialias ? '1' : '0',
                alpha: gl.getContextAttributes()?.alpha ? '1' : '0',
                depth: gl.getContextAttributes()?.depth ? '1' : '0',
                stencil: gl.getContextAttributes()?.stencil ? '1' : '0',
                maxTextureSize: gl.getParameter(gl.MAX_TEXTURE_SIZE),
                maxViewportDims: gl.getParameter(gl.MAX_VIEWPORT_DIMS).join('x'),
                vendor: gl.getParameter(gl.VENDOR),
                renderer: gl.getParameter(gl.RENDERER),
                glVersion: gl.getParameter(gl.VERSION),
                shadingLanguageVersion: gl.getParameter(gl.SHADING_LANGUAGE_VERSION)
            };

            return JSON.stringify(params).substring(0, 200);
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
                    let sumSquares = 0;
                    for (let i = 0; i < output.length; i++) {
                        sum += Math.abs(output[i]);
                        sumSquares += output[i] * output[i];
                    }
                    const rms = Math.sqrt(sumSquares / output.length);
                    const fingerprint = `sum:${sum.toFixed(6)}_rms:${rms.toFixed(8)}`;
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
            'Century Gothic', 'Consolas', 'Copperplate', 'Franklin Gothic Medium', 'Gill Sans'
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
                const plugin = navigator.plugins[i];
                plugins.push(`${plugin.name}~${plugin.filename}~${plugin.description}`.substring(0, 50));
            }
        }
        return plugins.join('|');
    }

    getDoNotTrack() {
        return navigator.doNotTrack === '1' ||
               navigator.doNotTrack === 'yes' ||
               window.doNotTrack === '1' ? '1' : '0';
    }

    getTouchSupport() {
        return {
            maxTouchPoints: navigator.maxTouchPoints || 0,
            touchEvent: 'ontouchstart' in window,
            touch: navigator.maxTouchPoints > 0,
            pointerEvent: window.PointerEvent ? '1' : '0'
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
                return `${connection.effectiveType || 'unknown'}|${connection.downlink || 'unknown'}|${connection.rtt || 'unknown'}`;
            }
            return 'unknown';
        } catch (e) {
            return 'unknown';
        }
    }

    async getBatteryStatus() {
        try {
            if (navigator.getBattery) {
                const battery = await navigator.getBattery();
                return `${battery.charging ? '1' : '0'}|${battery.level}|${battery.chargingTime}|${battery.dischargingTime}`;
            }
            return 'unsupported';
        } catch (e) {
            return 'error';
        }
    }

    async getMediaDevices() {
        try {
            if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
                const devices = await navigator.mediaDevices.enumerateDevices();
                const deviceInfo = devices.map(d => `${d.kind}:${d.label || 'unknown'}`.substring(0, 30)).join('|');
                return deviceInfo;
            }
            return 'unsupported';
        } catch (e) {
            return 'error';
        }
    }

    getPerformanceMetrics() {
        try {
            const perfData = performance.timing || {};
            const metrics = {
                loadTime: perfData.loadEventEnd - perfData.navigationStart,
                domReady: perfData.domContentLoadedEventEnd - perfData.navigationStart,
                firstPaint: perfData.responseStart - perfData.navigationStart
            };
            return JSON.stringify(metrics);
        } catch (e) {
            return 'error';
        }
    }

    getVideoCard() {
        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
            if (!gl) return 'unknown';

            const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
            if (debugInfo) {
                const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                const vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
                return `${vendor}|${renderer}`.substring(0, 80);
            }
            return 'unknown';
        } catch (e) {
            return 'error';
        }
    }

    getCpuClass() {
        return navigator.cpuClass || 'unknown';
    }

    getBrowserPlugins() {
        const features = [];
        features.push('pdfViewer:' + (navigator.pdfViewerEnabled !== undefined ? '1' : '0'));
        features.push('webdriver:' + (navigator.webdriver ? '1' : '0'));
        features.push('hdpi:' + (window.devicePixelRatio > 1 ? '1' : '0'));
        return features.join('|');
    }

    checkSessionStorage() {
        try {
            return sessionStorage ? '1' : '0';
        } catch (e) {
            return '0';
        }
    }

    checkLocalStorage() {
        try {
            return localStorage ? '1' : '0';
        } catch (e) {
            return '0';
        }
    }

    checkIndexedDB() {
        try {
            return window.indexedDB ? '1' : '0';
        } catch (e) {
            return '0';
        }
    }

    async hashString(str) {
        const encoder = new TextEncoder();
        const data = encoder.encode(str);
        const hashBuffer = await crypto.subtle.digest('SHA-256', data);
        const hashArray = Array.from(new Uint8Array(hashBuffer));
        return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
    }

    async generateHash() {
        const criticalComponents = [
            this.fingerprint.canvasFingerprint,
            this.fingerprint.webglFingerprint,
            this.fingerprint.audioFingerprint,
            this.fingerprint.screenResolution,
            this.fingerprint.timezone,
            this.fingerprint.fontFingerprint
        ].filter(c => c && c !== 'canvas-error' && c !== 'webgl-error' && c !== 'audio-error');

        const combined = criticalComponents.join('|');
        return await this.hashString(combined);
    }

    calculateConfidenceScore() {
        let score = 0;
        const totalEntropy = this.components.reduce((sum, c) => sum + (c.entropy || 0), 0);
        const collectedEntropy = this.components
            .filter(c => c.value && !c.value.includes('error') && !c.value.includes('unsupported') && c.value !== 'unknown')
            .reduce((sum, c) => sum + (c.entropy || 0), 0);

        if (totalEntropy > 0) {
            score = Math.round((collectedEntropy / totalEntropy) * 100);
        }

        return Math.min(100, Math.max(0, score));
    }

    getFingerprintHash() {
        return this.fingerprint.hash || '';
    }

    getConfidenceScore() {
        return this.confidenceScore;
    }
}

class BehaviorAnalyzer {
    constructor() {
        this.mouseTrajectory = [];
        this.keyboardPattern = [];
        this.scrollBehavior = [];
        this.clickPattern = [];
        this.timingData = [];
        this.touchGestures = [];
        this.startTime = Date.now();
        this.lastActivityTime = Date.now();
        this.activityCount = 0;
        this.idleTime = 0;
    }

    startTracking() {
        this.startTime = Date.now();
        this.lastActivityTime = Date.now();
        this.mouseTrajectory = [];
        this.keyboardPattern = [];
        this.scrollBehavior = [];
        this.clickPattern = [];
        this.timingData = [];
        this.touchGestures = [];
        this.activityCount = 0;
        this.idleTime = 0;

        this.trackMouseMovement();
        this.trackKeyboardInput();
        this.trackScrollBehavior();
        this.trackClickPattern();
        this.trackTouchGestures();
        this.trackIdleTime();
    }

    trackMouseMovement() {
        let lastMoveTime = Date.now();
        let lastX = 0;
        let lastY = 0;
        let moveCount = 0;
        let velocitySamples = [];

        const handleMouseMove = (e) => {
            const currentTime = Date.now();
            const timeDelta = currentTime - lastMoveTime;

            if (lastX !== 0 || lastY !== 0) {
                const distance = Math.sqrt(Math.pow(e.clientX - lastX, 2) + Math.pow(e.clientY - lastY, 2));
                const speed = timeDelta > 0 ? distance / timeDelta : 0;
                const acceleration = moveCount > 0 && velocitySamples.length > 0
                    ? (speed - velocitySamples[velocitySamples.length - 1]) / timeDelta
                    : 0;

                this.mouseTrajectory.push({
                    x: e.clientX,
                    y: e.clientY,
                    time: currentTime,
                    timeDelta: timeDelta,
                    distance: distance,
                    speed: speed,
                    acceleration: acceleration,
                    angle: Math.atan2(e.clientY - lastY, e.clientX - lastX)
                });

                velocitySamples.push(speed);
                if (velocitySamples.length > 10) velocitySamples.shift();

                this.activityCount++;
                this.lastActivityTime = currentTime;
            }

            lastX = e.clientX;
            lastY = e.clientY;
            lastMoveTime = currentTime;
            moveCount++;
        };

        document.addEventListener('mousemove', handleMouseMove, { passive: true });
    }

    trackKeyboardInput() {
        let lastKeyTime = Date.now();
        let totalDelay = 0;
        let keyCount = 0;
        let keyDurations = [];
        let lastKey = null;

        const handleKeyDown = (e) => {
            const currentTime = Date.now();
            const delay = currentTime - lastKeyTime;

            if (lastKeyTime !== currentTime) {
                this.keyboardPattern.push({
                    key: e.key,
                    code: e.code,
                    time: currentTime,
                    delay: delay,
                    isModifier: e.ctrlKey || e.altKey || e.metaKey,
                    shiftHeld: e.shiftKey
                });

                if (lastKey && delay < 500) {
                    keyDurations.push(delay);
                }

                totalDelay += delay;
                keyCount++;
                this.activityCount++;
                this.lastActivityTime = currentTime;
            }

            lastKey = e.key;
            lastKeyTime = currentTime;
        };

        const handleKeyUp = (e) => {
            this.keyboardPattern.push({
                key: e.key,
                code: e.code,
                time: Date.now(),
                type: 'keyup'
            });
        };

        document.addEventListener('keydown', handleKeyDown, { passive: true });
        document.addEventListener('keyup', handleKeyUp, { passive: true });
    }

    trackScrollBehavior() {
        let lastScrollTime = Date.now();
        let lastScrollY = window.scrollY;
        let scrollCount = 0;
        let scrollDirections = [];

        const handleScroll = () => {
            const currentTime = Date.now();
            const currentScrollY = window.scrollY;
            const scrollDelta = currentScrollY - lastScrollY;
            const scrollDirection = scrollDelta > 0 ? 'down' : 'up';

            this.scrollBehavior.push({
                scrollY: currentScrollY,
                delta: scrollDelta,
                direction: scrollDirection,
                time: currentTime,
                timeDelta: currentTime - lastScrollTime
            });

            scrollDirections.push(scrollDirection);
            if (scrollDirections.length > 20) scrollDirections.shift();

            lastScrollY = currentScrollY;
            lastScrollTime = currentTime;
            scrollCount++;
            this.activityCount++;
            this.lastActivityTime = currentTime;
        };

        window.addEventListener('scroll', handleScroll, { passive: true });
    }

    trackClickPattern() {
        let lastClickTime = Date.now();
        let clickIntervals = [];

        const handleClick = (e) => {
            const currentTime = Date.now();
            const interval = currentTime - lastClickTime;

            if (lastClickTime !== currentTime) {
                clickIntervals.push(interval);
                if (clickIntervals.length > 10) clickIntervals.shift();

                this.clickPattern.push({
                    x: e.clientX,
                    y: e.clientY,
                    target: e.target.tagName,
                    targetClass: e.target.className,
                    targetId: e.target.id,
                    time: currentTime,
                    interval: interval
                });

                this.activityCount++;
                this.lastActivityTime = currentTime;
            }

            lastClickTime = currentTime;
        };

        document.addEventListener('click', handleClick, { passive: true });
    }

    trackTouchGestures() {
        let touchStartTime = 0;
        let touchStartX = 0;
        let touchStartY = 0;
        let touchPoints = [];

        const handleTouchStart = (e) => {
            if (e.touches.length === 1) {
                touchStartTime = Date.now();
                touchStartX = e.touches[0].clientX;
                touchStartY = e.touches[0].clientY;
                touchPoints = [{ x: touchStartX, y: touchStartY, time: touchStartTime }];
            }
        };

        const handleTouchMove = (e) => {
            if (e.touches.length === 1) {
                touchPoints.push({
                    x: e.touches[0].clientX,
                    y: e.touches[0].clientY,
                    time: Date.now()
                });
            }
        };

        const handleTouchEnd = (e) => {
            const endTime = Date.now();
            if (touchPoints.length > 0) {
                const startPoint = touchPoints[0];
                const endPoint = touchPoints[touchPoints.length - 1];
                const duration = endTime - touchStartTime;
                const distance = Math.sqrt(
                    Math.pow(endPoint.x - startPoint.x, 2) +
                    Math.pow(endPoint.y - startPoint.y, 2)
                );
                const velocity = duration > 0 ? distance / duration : 0;

                this.touchGestures.push({
                    startX: touchStartX,
                    startY: touchStartY,
                    endX: endPoint.x,
                    endY: endPoint.y,
                    distance: distance,
                    duration: duration,
                    velocity: velocity,
                    points: touchPoints.slice(0, 10),
                    time: touchStartTime
                });

                this.activityCount++;
                this.lastActivityTime = endTime;
            }
        };

        document.addEventListener('touchstart', handleTouchStart, { passive: true });
        document.addEventListener('touchmove', handleTouchMove, { passive: true });
        document.addEventListener('touchend', handleTouchEnd, { passive: true });
    }

    trackIdleTime() {
        const checkIdle = () => {
            const now = Date.now();
            const idle = now - this.lastActivityTime;
            if (idle > 1000) {
                this.idleTime += idle;
            }
        };

        setInterval(checkIdle, 1000);
    }

    analyze() {
        const analysis = {
            totalTime: Date.now() - this.startTime,
            mouseMovements: this.mouseTrajectory.length,
            keyboardInputs: this.keyboardPattern.length,
            scrolls: this.scrollBehavior.length,
            clicks: this.clickPattern.length,
            touchGestures: this.touchGestures.length,
            activityCount: this.activityCount,
            idleTime: this.idleTime,
            activeTime: (Date.now() - this.startTime) - this.idleTime
        };

        if (this.mouseTrajectory.length > 0) {
            const speeds = this.mouseTrajectory.map(m => m.speed);
            const accelerations = this.mouseTrajectory.map(m => Math.abs(m.acceleration));

            const totalSpeed = speeds.reduce((sum, s) => sum + s, 0);
            const totalAccel = accelerations.reduce((sum, a) => sum + a, 0);
            const totalDistance = this.mouseTrajectory.reduce((sum, m) => sum + m.distance, 0);

            analysis.averageSpeed = totalSpeed / speeds.length;
            analysis.maxSpeed = Math.max(...speeds);
            analysis.minSpeed = Math.min(...speeds);
            analysis.speedVariance = this.calculateVariance(speeds);

            analysis.averageAcceleration = totalAccel / (accelerations.length || 1);
            analysis.maxAcceleration = Math.max(...accelerations);

            analysis.totalDistance = totalDistance;

            const angles = this.mouseTrajectory.map(m => m.angle);
            analysis.directionChanges = this.countDirectionChanges(angles);

            analysis.curvatureScore = this.calculateCurvature(angles);
        } else {
            analysis.averageSpeed = 0;
            analysis.maxSpeed = 0;
            analysis.minSpeed = 0;
            analysis.speedVariance = 0;
            analysis.averageAcceleration = 0;
            analysis.maxAcceleration = 0;
            analysis.totalDistance = 0;
            analysis.directionChanges = 0;
            analysis.curvatureScore = 0;
        }

        if (this.keyboardPattern.length > 1) {
            const delays = this.keyboardPattern.slice(1).map(k => k.delay).filter(d => d > 0 && d < 10000);
            if (delays.length > 0) {
                analysis.averageKeyDelay = delays.reduce((sum, d) => sum + d, 0) / delays.length;
                analysis.maxKeyDelay = Math.max(...delays);
                analysis.minKeyDelay = Math.min(...delays);
                analysis.keyDelayVariance = this.calculateVariance(delays);
            } else {
                analysis.averageKeyDelay = 0;
                analysis.maxKeyDelay = 0;
                analysis.minKeyDelay = 0;
                analysis.keyDelayVariance = 0;
            }

            analysis.modifierUsage = this.keyboardPattern.filter(k => k.isModifier).length;
        } else {
            analysis.averageKeyDelay = 0;
            analysis.maxKeyDelay = 0;
            analysis.minKeyDelay = 0;
            analysis.keyDelayVariance = 0;
            analysis.modifierUsage = 0;
        }

        if (this.clickPattern.length > 1) {
            const intervals = this.clickPattern.slice(1).map(c => c.interval).filter(i => i > 0 && i < 10000);
            if (intervals.length > 0) {
                analysis.averageClickInterval = intervals.reduce((sum, i) => sum + i, 0) / intervals.length;
                analysis.clickIntervalVariance = this.calculateVariance(intervals);
            } else {
                analysis.averageClickInterval = 0;
                analysis.clickIntervalVariance = 0;
            }
        } else {
            analysis.averageClickInterval = 0;
            analysis.clickIntervalVariance = 0;
        }

        if (this.touchGestures.length > 0) {
            const velocities = this.touchGestures.map(t => t.velocity);
            analysis.averageTouchVelocity = velocities.reduce((sum, v) => sum + v, 0) / velocities.length;
            analysis.maxTouchVelocity = Math.max(...velocities);
        } else {
            analysis.averageTouchVelocity = 0;
            analysis.maxTouchVelocity = 0;
        }

        analysis.scrollActivity = this.scrollBehavior.length > 0;
        analysis.clickActivity = this.clickPattern.length > 0;
        analysis.mouseActivity = this.mouseTrajectory.length > 10;
        analysis.touchActivity = this.touchGestures.length > 0;
        analysis.keyboardActivity = this.keyboardPattern.length > 3;

        return analysis;
    }

    calculateVariance(values) {
        if (values.length < 2) return 0;
        const mean = values.reduce((sum, v) => sum + v, 0) / values.length;
        const squaredDiffs = values.map(v => Math.pow(v - mean, 2));
        return squaredDiffs.reduce((sum, d) => sum + d, 0) / values.length;
    }

    countDirectionChanges(angles) {
        if (angles.length < 2) return 0;
        let changes = 0;
        let lastDirection = null;

        for (let i = 1; i < angles.length; i++) {
            const diff = angles[i] - angles[i - 1];
            let direction;
            if (Math.abs(diff) < 0.1) {
                direction = 'straight';
            } else if (diff > 0) {
                direction = 'right';
            } else {
                direction = 'left';
            }

            if (lastDirection && direction !== lastDirection) {
                changes++;
            }
            lastDirection = direction;
        }

        return changes;
    }

    calculateCurvature(angles) {
        if (angles.length < 3) return 0;
        let totalCurvature = 0;

        for (let i = 1; i < angles.length - 1; i++) {
            const angleDiff = Math.abs(angles[i + 1] - angles[i]);
            totalCurvature += angleDiff;
        }

        return totalCurvature / (angles.length - 2);
    }

    getBehaviorScore() {
        const analysis = this.analyze();
        let score = 100;

        if (!analysis.mouseActivity && !analysis.touchActivity && !analysis.keyboardActivity) {
            score -= 30;
        }

        if (analysis.maxSpeed > 150) {
            score -= 20;
        } else if (analysis.maxSpeed > 100) {
            score -= 10;
        }

        if (analysis.totalTime < 1000) {
            score -= 25;
        } else if (analysis.totalTime < 2000) {
            score -= 15;
        }

        if (analysis.activeTime < analysis.totalTime * 0.3) {
            score -= 15;
        }

        if (analysis.speedVariance > 500) {
            score -= 10;
        }

        if (analysis.directionChanges < 3 && analysis.mouseMovements > 20) {
            score -= 15;
        }

        if (analysis.curvatureScore < 0.1 && analysis.mouseMovements > 30) {
            score -= 10;
        }

        if (analysis.averageKeyDelay < 30 && analysis.keyboardInputs > 5) {
            score -= 15;
        }

        if (analysis.averageClickInterval < 100 && analysis.clicks > 3) {
            score -= 10;
        }

        if (analysis.activityCount < 5) {
            score -= 10;
        }

        if (analysis.totalTime > 60000) {
            score += 5;
        }

        if (analysis.modifierUsage > 0) {
            score += 3;
        }

        return Math.max(0, Math.min(100, score));
    }

    getDetailedScore() {
        const analysis = this.analyze();
        return {
            overall: this.getBehaviorScore(),
            mouseScore: this.getMouseScore(analysis),
            keyboardScore: this.getKeyboardScore(analysis),
            clickScore: this.getClickScore(analysis),
            patternScore: this.getPatternScore(analysis),
            timingScore: this.getTimingScore(analysis)
        };
    }

    getMouseScore(analysis) {
        let score = 100;

        if (analysis.mouseMovements < 5) {
            score -= 30;
        } else if (analysis.mouseMovements < 15) {
            score -= 15;
        }

        if (analysis.maxSpeed > 200) {
            score -= 25;
        } else if (analysis.maxSpeed > 100) {
            score -= 10;
        }

        if (analysis.speedVariance > 1000) {
            score -= 15;
        }

        if (analysis.directionChanges < 2 && analysis.mouseMovements > 15) {
            score -= 20;
        }

        if (analysis.curvatureScore < 0.05 && analysis.mouseMovements > 25) {
            score -= 15;
        }

        return Math.max(0, Math.min(100, score));
    }

    getKeyboardScore(analysis) {
        let score = 100;

        if (analysis.keyboardInputs === 0) {
            score -= 20;
        } else if (analysis.keyboardInputs < 5) {
            score -= 10;
        }

        if (analysis.averageKeyDelay < 20 && analysis.keyboardInputs > 3) {
            score -= 25;
        }

        if (analysis.keyDelayVariance < 500 && analysis.keyboardInputs > 5) {
            score -= 15;
        }

        if (analysis.modifierUsage > 0) {
            score += 5;
        }

        return Math.max(0, Math.min(100, score));
    }

    getClickScore(analysis) {
        let score = 100;

        if (analysis.clicks === 0) {
            score -= 10;
        }

        if (analysis.averageClickInterval < 80 && analysis.clicks > 2) {
            score -= 20;
        }

        if (analysis.clickIntervalVariance < 1000 && analysis.clicks > 3) {
            score -= 15;
        }

        return Math.max(0, Math.min(100, score));
    }

    getPatternScore(analysis) {
        let score = 100;

        if (analysis.totalDistance > 0) {
            const efficiency = analysis.directionChanges / (analysis.totalDistance / 100);
            if (efficiency < 0.5) {
                score -= 15;
            }
        }

        const behaviorTypes = [
            analysis.mouseActivity,
            analysis.keyboardActivity,
            analysis.clickActivity,
            analysis.touchActivity,
            analysis.scrollActivity
        ].filter(Boolean).length;

        if (behaviorTypes >= 3) {
            score += 10;
        } else if (behaviorTypes === 1) {
            score -= 15;
        }

        return Math.max(0, Math.min(100, score));
    }

    getTimingScore(analysis) {
        let score = 100;

        if (analysis.totalTime < 500) {
            score -= 30;
        } else if (analysis.totalTime < 1000) {
            score -= 15;
        }

        if (analysis.activeTime / analysis.totalTime < 0.5) {
            score -= 15;
        }

        if (analysis.totalTime > 30000 && analysis.activityCount < 10) {
            score -= 20;
        }

        return Math.max(0, Math.min(100, score));
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
                retry: '重试'
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
                retry: 'Retry'
            },
            'ja-JP': {
                checking: 'デバイスを確認中...',
                fingerprinting: 'フィンガープリント収集中...',
                analyzing: '行動を分析中...',
                scoring: 'リスクスコア計算中...',
                trusted: 'デバイス信頼済み',
                untrusted: 'デバイス未信頼',
                pending: '確認待ち',
                error: '確認エラー',
                retry: '再試行'
            },
            'ko-KR': {
                checking: '장치 확인 중...',
                fingerprinting: '지문 수집 중...',
                analyzing: '행동 분석 중...',
                scoring: '위험 점수 계산 중...',
                trusted: '장치 신뢰됨',
                untrusted: '장치 신뢰되지 않음',
                pending: '확인 대기 중',
                error: '확인 오류',
                retry: '다시 시도'
            },
            'fr-FR': {
                checking: 'Vérification de l\'appareil...',
                fingerprinting: 'Collecte des données...',
                analyzing: 'Analyse du comportement...',
                scoring: 'Calcul du score de risque...',
                trusted: 'Appareil de confiance',
                untrusted: 'Appareil non fiable',
                pending: 'En attente de vérification',
                error: 'Erreur de vérification',
                retry: 'Réessayer'
            },
            'de-DE': {
                checking: 'Gerät wird überprüft...',
                fingerprinting: 'Fingerabdruck wird gesammelt...',
                analyzing: 'Verhalten wird analysiert...',
                scoring: 'Risikobewertung wird berechnet...',
                trusted: 'Gerät vertrauenswürdig',
                untrusted: 'Gerät nicht vertrauenswürdig',
                pending: 'Warten auf Überprüfung',
                error: 'Überprüfungsfehler',
                retry: 'Erneut versuchen'
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

    getSupportedLocales() {
        return Object.keys(this.translations);
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
            trackingDuration: 2000,
            minConfidenceScore: 50,
            onStatusChange: null,
            onStepChange: null,
            onTrustChange: null,
            onComplete: null,
            onError: null,
            ...options
        };

        this.fingerprintCollector = new FingerprintCollector();
        this.behaviorAnalyzer = new BehaviorAnalyzer();
        this.i18n = new SeamlessI18n(this.options.language);

        this.state = {
            status: 'idle',
            fingerprint: null,
            behaviorAnalysis: null,
            trustStatus: 'pending',
            trustLevel: 0,
            sessionId: null,
            isVerifying: false,
            verificationAttempts: 0,
            lastVerificationTime: null
        };

        this.init();
    }

    init() {
        if (this.options.autoStart) {
            this.start();
        }
    }

    async start() {
        if (this.state.isVerifying) {
            return;
        }

        this.state.isVerifying = true;
        this.state.verificationAttempts++;
        this.updateStatus('checking');

        try {
            await this.collectFingerprint();
            await this.analyzeBehavior();
            await this.calculateRiskScore();
            await this.verifyWithServer();
            this.state.isVerifying = false;
            this.state.lastVerificationTime = Date.now();
        } catch (error) {
            console.error('SeamlessCaptcha verification error:', error);
            this.state.isVerifying = false;
            this.handleError(error);
        }
    }

    async collectFingerprint() {
        this.updateStatus('fingerprinting');
        this.updateStep('fingerprint', 25);

        try {
            this.state.fingerprint = await this.fingerprintCollector.collect();

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
            this.state.detailedScore = this.behaviorAnalyzer.getDetailedScore();

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
            const behaviorScore = this.behaviorAnalyzer.getBehaviorScore();
            const detailedScore = this.state.detailedScore;

            const riskFactors = {
                fingerprintEntropy: this.calculateFingerprintEntropy(),
                fingerprintConfidence: this.fingerprintCollector.getConfidenceScore(),
                behaviorScore: behaviorScore,
                behaviorDetails: detailedScore,
                deviceConsistency: await this.checkDeviceConsistency(),
                patternAnomaly: this.detectPatternAnomaly(),
                velocityAnomaly: this.detectVelocityAnomaly(),
                timingAnomaly: this.detectTimingAnomaly()
            };

            this.state.riskScore = this.calculateOverallRiskScore(riskFactors);
            this.state.riskFactors = riskFactors;

            return this.state.riskScore;
        } catch (error) {
            console.error('Risk calculation error:', error);
            throw error;
        }
    }

    calculateFingerprintEntropy() {
        if (!this.state.fingerprint) return 0;

        let entropy = 0;
        const fp = this.state.fingerprint;

        if (fp.canvasFingerprint && fp.canvasFingerprint !== 'canvas-error') entropy += 20;
        if (fp.webglFingerprint && fp.webglFingerprint !== 'webgl-error') entropy += 15;
        if (fp.audioFingerprint && fp.audioFingerprint !== 'audio-error') entropy += 10;
        if (fp.screenResolution && fp.screenResolution !== '0x0') entropy += 10;
        if (fp.timezone) entropy += 5;
        if (fp.language) entropy += 5;
        if (fp.fontFingerprint && fp.fontFingerprint.length > 50) entropy += 10;
        if (fp.plugins && fp.plugins !== 'unsupported') entropy += 5;

        return entropy;
    }

    async checkDeviceConsistency() {
        const storedHash = localStorage.getItem('device_fingerprint_hash');
        const currentHash = this.fingerprintCollector.getFingerprintHash();

        if (storedHash && storedHash !== currentHash) {
            const lastVisit = localStorage.getItem('device_last_visit');
            const visitCount = parseInt(localStorage.getItem('device_visit_count') || '0');

            if (visitCount > 3) {
                return 30;
            }
            return 50;
        }

        return 100;
    }

    detectPatternAnomaly() {
        if (!this.state.behaviorAnalysis) return 0;

        const analysis = this.state.behaviorAnalysis;
        let anomalyScore = 0;

        if (analysis.mouseMovements === 0 && analysis.keyboardInputs === 0 && analysis.clicks === 0) {
            anomalyScore += 30;
        }

        if (analysis.totalTime < 500) {
            anomalyScore += 25;
        }

        if (analysis.maxSpeed > 50) {
            anomalyScore += 20;
        }

        if (analysis.averageKeyDelay < 50 && analysis.keyboardInputs > 5) {
            anomalyScore += 15;
        }

        if (analysis.directionChanges < 3 && analysis.mouseMovements > 20) {
            anomalyScore += 15;
        }

        if (analysis.curvatureScore < 0.05 && analysis.mouseMovements > 25) {
            anomalyScore += 10;
        }

        return Math.max(0, 100 - anomalyScore);
    }

    detectVelocityAnomaly() {
        if (!this.state.behaviorAnalysis) return 100;

        const analysis = this.state.behaviorAnalysis;
        let score = 100;

        if (analysis.maxSpeed > 200) {
            score -= 30;
        } else if (analysis.maxSpeed > 150) {
            score -= 15;
        }

        if (analysis.averageAcceleration > 5) {
            score -= 20;
        }

        if (analysis.speedVariance > 1000) {
            score -= 10;
        }

        return Math.max(0, score);
    }

    detectTimingAnomaly() {
        if (!this.state.behaviorAnalysis) return 100;

        const analysis = this.state.behaviorAnalysis;
        let score = 100;

        if (analysis.totalTime < 1000 && analysis.activityCount < 5) {
            score -= 25;
        }

        if (analysis.activeTime / analysis.totalTime < 0.4) {
            score -= 15;
        }

        if (analysis.averageKeyDelay < 25 && analysis.keyboardInputs > 5) {
            score -= 20;
        }

        return Math.max(0, score);
    }

    calculateOverallRiskScore(riskFactors) {
        const weights = {
            fingerprintEntropy: 0.2,
            fingerprintConfidence: 0.15,
            behaviorScore: 0.25,
            deviceConsistency: 0.15,
            patternAnomaly: 0.1,
            velocityAnomaly: 0.08,
            timingAnomaly: 0.07
        };

        const score =
            (riskFactors.fingerprintEntropy / 100) * weights.fingerprintEntropy * 100 +
            riskFactors.fingerprintConfidence * weights.fingerprintConfidence +
            riskFactors.behaviorScore * weights.behaviorScore +
            riskFactors.deviceConsistency * weights.deviceConsistency +
            riskFactors.patternAnomaly * weights.patternAnomaly +
            riskFactors.velocityAnomaly * weights.velocityAnomaly +
            riskFactors.timingAnomaly * weights.timingAnomaly;

        return Math.round(score);
    }

    async verifyWithServer() {
        this.updateStep('result', 90);

        try {
            const response = await this.apiCall('/seamless/check-status', 'POST', {
                fingerprint: this.state.fingerprint,
                behavior_analysis: this.state.behaviorAnalysis,
                detailed_score: this.state.detailedScore,
                risk_score: this.state.riskScore,
                risk_factors: this.state.riskFactors
            });

            if (response.success) {
                this.state.trustStatus = response.trusted ? 'trusted' : 'untrusted';
                this.state.trustLevel = response.trust_level || this.state.riskScore;
                this.state.sessionId = response.session_id;

                localStorage.setItem('device_fingerprint_hash',
                    this.fingerprintCollector.getFingerprintHash());
                localStorage.setItem('device_last_visit', Date.now().toString());
                const visitCount = parseInt(localStorage.getItem('device_visit_count') || '0') + 1;
                localStorage.setItem('device_visit_count', visitCount.toString());

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
                        riskScore: this.state.riskScore,
                        fingerprint: this.state.fingerprint,
                        behaviorAnalysis: this.state.behaviorAnalysis,
                        detailedScore: this.state.detailedScore,
                        riskFactors: this.state.riskFactors
                    });
                }
            } else {
                throw new Error(response.message || 'Verification failed');
            }
        } catch (error) {
            console.error('Server verification error:', error);

            this.state.trustStatus = 'pending';
            this.state.trustLevel = this.state.riskScore;

            this.updateStatus('pending');

            throw error;
        }
    }

    async trustDevice() {
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
                localStorage.setItem('device_trusted', 'true');
                localStorage.setItem('device_trust_expiry',
                    (Date.now() + 30 * 24 * 60 * 60 * 1000).toString());

                return true;
            }

            return false;
        } catch (error) {
            console.error('Trust device error:', error);
            return false;
        }
    }

    async checkStatus() {
        const storedHash = localStorage.getItem('device_fingerprint_hash');
        const currentHash = this.fingerprintCollector.getFingerprintHash();

        if (storedHash && storedHash === currentHash) {
            const trusted = localStorage.getItem('device_trusted');
            const expiry = localStorage.getItem('device_trust_expiry');

            if (trusted === 'true' && expiry && parseInt(expiry) > Date.now()) {
                return {
                    trusted: true,
                    trustLevel: 100,
                    cached: true
                };
            }
        }

        return {
            trusted: false,
            trustLevel: 0,
            cached: false
        };
    }

    reset() {
        this.state = {
            status: 'idle',
            fingerprint: null,
            behaviorAnalysis: null,
            trustStatus: 'pending',
            trustLevel: 0,
            sessionId: null,
            isVerifying: false,
            verificationAttempts: 0,
            lastVerificationTime: null
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
                trusted: this.state.riskScore >= 60,
                trust_level: this.state.riskScore,
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

    getRiskFactors() {
        return this.state.riskFactors || null;
    }

    getDetailedScore() {
        return this.state.detailedScore || null;
    }

    destroy() {
        this.reset();
        this.container = null;
        this.options = {};
        this.fingerprintCollector = null;
        this.behaviorAnalyzer = null;
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = { SeamlessCaptcha, FingerprintCollector, BehaviorAnalyzer, SeamlessI18n };
}
