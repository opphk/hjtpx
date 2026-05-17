class FingerprintCollector {
    constructor() {
        this.fingerprint = {};
        this.components = [];
    }

    async collect() {
        this.components = [];

        this.fingerprint.userAgent = this.getUserAgent();
        this.components.push({ key: 'userAgent', value: this.fingerprint.userAgent });

        this.fingerprint.screenResolution = this.getScreenResolution();
        this.components.push({ key: 'screenResolution', value: this.fingerprint.screenResolution });

        this.fingerprint.colorDepth = this.getColorDepth();
        this.components.push({ key: 'colorDepth', value: this.fingerprint.colorDepth });

        this.fingerprint.timezone = this.getTimezone();
        this.components.push({ key: 'timezone', value: this.fingerprint.timezone });

        this.fingerprint.language = this.getLanguage();
        this.components.push({ key: 'language', value: this.fingerprint.language });

        this.fingerprint.platform = this.getPlatform();
        this.components.push({ key: 'platform', value: this.fingerprint.platform });

        this.fingerprint.canvasFingerprint = await this.getCanvasFingerprint();
        this.components.push({ key: 'canvasFingerprint', value: this.fingerprint.canvasFingerprint });

        this.fingerprint.webglFingerprint = this.getWebGLFingerprint();
        this.components.push({ key: 'webglFingerprint', value: this.fingerprint.webglFingerprint });

        this.fingerprint.audioFingerprint = await this.getAudioFingerprint();
        this.components.push({ key: 'audioFingerprint', value: this.fingerprint.audioFingerprint });

        this.fingerprint.fontFingerprint = await this.getFontFingerprint();
        this.components.push({ key: 'fontFingerprint', value: this.fingerprint.fontFingerprint });

        this.fingerprint.plugins = this.getPlugins();
        this.components.push({ key: 'plugins', value: this.fingerprint.plugins });

        this.fingerprint.doNotTrack = this.getDoNotTrack();
        this.components.push({ key: 'doNotTrack', value: this.fingerprint.doNotTrack });

        this.fingerprint.touchSupport = this.getTouchSupport();
        this.components.push({ key: 'touchSupport', value: this.fingerprint.touchSupport });

        this.fingerprint.deviceMemory = this.getDeviceMemory();
        this.components.push({ key: 'deviceMemory', value: this.fingerprint.deviceMemory });

        this.fingerprint.hardwareConcurrency = this.getHardwareConcurrency();
        this.components.push({ key: 'hardwareConcurrency', value: this.fingerprint.hardwareConcurrency });

        this.fingerprint.connectionType = await this.getConnectionType();
        this.components.push({ key: 'connectionType', value: this.fingerprint.connectionType });

        this.fingerprint.timestamp = Date.now();
        this.components.push({ key: 'timestamp', value: this.fingerprint.timestamp });

        this.fingerprint.hash = this.generateHash();

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
            'Lucida Console', 'Lucida Sans Unicode', 'Tahoma', ' Geneva', 'Helvetica'
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
                plugins.push(navigator.plugins[i].name);
            }
        }
        return plugins.join(',');
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
            this.fingerprint.timezone
        ];
        const combined = components.join('|');
        return this.hashString(combined);
    }

    getFingerprintHash() {
        return this.fingerprint.hash || '';
    }
}

class BehaviorAnalyzer {
    constructor() {
        this.mouseTrajectory = [];
        this.keyboardPattern = [];
        this.scrollBehavior = [];
        this.clickPattern = [];
        this.timingData = [];
        this.startTime = Date.now();
    }

    startTracking() {
        this.startTime = Date.now();
        this.mouseTrajectory = [];
        this.keyboardPattern = [];
        this.scrollBehavior = [];
        this.clickPattern = [];
        this.timingData = [];

        this.trackMouseMovement();
        this.trackKeyboardInput();
        this.trackScrollBehavior();
        this.trackClickPattern();
    }

    trackMouseMovement() {
        let lastMoveTime = Date.now();
        let lastX = 0;
        let lastY = 0;
        let moveCount = 0;

        const handleMouseMove = (e) => {
            const currentTime = Date.now();
            const timeDelta = currentTime - lastMoveTime;
            
            if (lastX !== 0 || lastY !== 0) {
                const distance = Math.sqrt(Math.pow(e.clientX - lastX, 2) + Math.pow(e.clientY - lastY, 2));
                const speed = timeDelta > 0 ? distance / timeDelta : 0;

                this.mouseTrajectory.push({
                    x: e.clientX,
                    y: e.clientY,
                    time: currentTime,
                    timeDelta: timeDelta,
                    distance: distance,
                    speed: speed
                });
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

        const handleKeyDown = (e) => {
            const currentTime = Date.now();
            const delay = currentTime - lastKeyTime;

            if (lastKeyTime !== currentTime) {
                this.keyboardPattern.push({
                    key: e.key,
                    time: currentTime,
                    delay: delay
                });
                totalDelay += delay;
                keyCount++;
            }

            lastKeyTime = currentTime;
        };

        document.addEventListener('keydown', handleKeyDown, { passive: true });
    }

    trackScrollBehavior() {
        let lastScrollTime = Date.now();
        let lastScrollY = window.scrollY;
        let scrollCount = 0;

        const handleScroll = () => {
            const currentTime = Date.now();
            const currentScrollY = window.scrollY;
            const scrollDelta = Math.abs(currentScrollY - lastScrollY);

            this.scrollBehavior.push({
                scrollY: currentScrollY,
                delta: scrollDelta,
                time: currentTime,
                timeDelta: currentTime - lastScrollTime
            });

            lastScrollY = currentScrollY;
            lastScrollTime = currentTime;
            scrollCount++;
        };

        window.addEventListener('scroll', handleScroll, { passive: true });
    }

    trackClickPattern() {
        const handleClick = (e) => {
            this.clickPattern.push({
                x: e.clientX,
                y: e.clientY,
                target: e.target.tagName,
                time: Date.now()
            });
        };

        document.addEventListener('click', handleClick, { passive: true });
    }

    analyze() {
        const analysis = {
            totalTime: Date.now() - this.startTime,
            mouseMovements: this.mouseTrajectory.length,
            keyboardInputs: this.keyboardPattern.length,
            scrolls: this.scrollBehavior.length,
            clicks: this.clickPattern.length
        };

        if (this.mouseTrajectory.length > 0) {
            const totalSpeed = this.mouseTrajectory.reduce((sum, m) => sum + m.speed, 0);
            analysis.averageSpeed = totalSpeed / this.mouseTrajectory.length;
            
            const speeds = this.mouseTrajectory.map(m => m.speed);
            analysis.maxSpeed = Math.max(...speeds);
            analysis.minSpeed = Math.min(...speeds);
            
            const totalDistance = this.mouseTrajectory.reduce((sum, m) => sum + m.distance, 0);
            analysis.totalDistance = totalDistance;
        } else {
            analysis.averageSpeed = 0;
            analysis.maxSpeed = 0;
            analysis.minSpeed = 0;
            analysis.totalDistance = 0;
        }

        if (this.keyboardPattern.length > 1) {
            const delays = this.keyboardPattern.slice(1).map(k => k.delay);
            analysis.averageKeyDelay = delays.reduce((sum, d) => sum + d, 0) / delays.length;
            analysis.maxKeyDelay = Math.max(...delays);
            analysis.minKeyDelay = Math.min(...delays);
        } else {
            analysis.averageKeyDelay = 0;
            analysis.maxKeyDelay = 0;
            analysis.minKeyDelay = 0;
        }

        analysis.scrollActivity = this.scrollBehavior.length > 0;
        analysis.clickActivity = this.clickPattern.length > 0;
        analysis.mouseActivity = this.mouseTrajectory.length > 10;

        return analysis;
    }

    getBehaviorScore() {
        const analysis = this.analyze();
        let score = 100;

        if (!analysis.mouseActivity) {
            score -= 20;
        }

        if (analysis.averageSpeed > 100) {
            score -= 30;
        }

        if (!analysis.keyboardInputs && !analysis.clicks) {
            score -= 15;
        }

        if (analysis.totalTime < 1000) {
            score -= 25;
        }

        if (analysis.totalTime > 60000) {
            score += 5;
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
            trackingDuration: 2000,
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
            verificationAttempts: 0
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

            const riskFactors = {
                fingerprintEntropy: this.calculateFingerprintEntropy(),
                behaviorScore: behaviorScore,
                deviceConsistency: await this.checkDeviceConsistency(),
                patternAnomaly: this.detectPatternAnomaly()
            };

            this.state.riskScore = this.calculateOverallRiskScore(riskFactors);

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

        return entropy;
    }

    async checkDeviceConsistency() {
        const storedHash = localStorage.getItem('device_fingerprint_hash');
        const currentHash = this.fingerprintCollector.getFingerprintHash();

        if (storedHash && storedHash !== currentHash) {
            return 50;
        }

        return 100;
    }

    detectPatternAnomaly() {
        if (!this.state.behaviorAnalysis) return 0;

        const analysis = this.state.behaviorAnalysis;
        let anomalyScore = 0;

        if (analysis.mouseMovements === 0 && analysis.keyboardInputs === 0) {
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

        return Math.max(0, 100 - anomalyScore);
    }

    calculateOverallRiskScore(riskFactors) {
        const weights = {
            fingerprintEntropy: 0.3,
            behaviorScore: 0.4,
            deviceConsistency: 0.2,
            patternAnomaly: 0.1
        };

        const score = 
            riskFactors.fingerprintEntropy * weights.fingerprintEntropy +
            riskFactors.behaviorScore * weights.behaviorScore +
            riskFactors.deviceConsistency * weights.deviceConsistency +
            riskFactors.patternAnomaly * weights.patternAnomaly;

        return Math.round(score);
    }

    async verifyWithServer() {
        this.updateStep('result', 90);

        try {
            const response = await this.apiCall('/seamless/check-status', 'POST', {
                fingerprint: this.state.fingerprint,
                behavior_analysis: this.state.behaviorAnalysis,
                risk_score: this.state.riskScore
            });

            if (response.success) {
                this.state.trustStatus = response.trusted ? 'trusted' : 'untrusted';
                this.state.trustLevel = response.trust_level || this.state.riskScore;
                this.state.sessionId = response.session_id;

                localStorage.setItem('device_fingerprint_hash', 
                    this.fingerprintCollector.getFingerprintHash());

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
                        behaviorAnalysis: this.state.behaviorAnalysis
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
            verificationAttempts: 0
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
            deviceEl && (deviceEl.textContent = browser);
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
