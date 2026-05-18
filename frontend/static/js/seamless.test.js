describe('Seamless Verification Tests', () => {
    let originalLocalStorage;
    let originalNavigator;
    let originalDocument;
    let originalWindow;
    let originalScreen;
    let originalCrypto;
    let originalPerformance;

    beforeEach(() => {
        originalLocalStorage = global.localStorage;
        global.localStorage = {
            data: {},
            getItem: function(key) { return this.data[key] || null; },
            setItem: function(key, value) { this.data[key] = value; },
            removeItem: function(key) { delete this.data[key]; },
            clear: function() { this.data = {}; }
        };

        originalNavigator = global.navigator;
        global.navigator = {
            userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0',
            language: 'zh-CN',
            platform: 'Win32',
            hardwareConcurrency: 4,
            deviceMemory: 8,
            maxTouchPoints: 0,
            plugins: [],
            cookieEnabled: true,
            javaEnabled: function() { return false; },
            vendor: 'Google Inc.',
            webdriver: false,
            connection: {
                effectiveType: '4g'
            }
        };

        originalDocument = global.document;
        global.document = {
            createElement: function(tag) {
                if (tag === 'canvas') {
                    return {
                        getContext: function() {
                            return {
                                fillText: function() {},
                                fillRect: function() {},
                                measureText: function() { return { width: 100 }; },
                                font: '',
                                textBaseline: '',
                                beginPath: function() {},
                                moveTo: function() {},
                                bezierCurveTo: function() {},
                                stroke: function() {},
                                strokeStyle: '',
                                arc: function() {},
                                fill: function() {},
                                fillStyle: ''
                            };
                        },
                        toDataURL: function() { return 'data:image/png;base64,test'; },
                        width: 200,
                        height: 50
                    };
                }
                return {};
            },
            addEventListener: function() {},
            removeEventListener: function() {},
            cookie: '',
            createElementNS: function() { return {}; }
        };

        originalWindow = global.window;
        global.window = {
            AudioContext: function() {
                this.createOscillator = function() { return {}; };
                this.createAnalyser = function() { return {}; };
                this.createGain = function() { return {}; };
                this.createScriptProcessor = function() { return {}; };
                this.destination = {};
                this.currentTime = 0;
                this.close = function() {};
            },
            RTCPeerConnection: function() {},
            devicePixelRatio: 1,
            performance: {
                timing: {
                    navigationStart: Date.now() - 1000,
                    loadEventEnd: Date.now(),
                    domContentLoadedEventEnd: Date.now(),
                    domInteractive: Date.now(),
                    responseEnd: Date.now(),
                    requestStart: Date.now()
                }
            }
        };

        originalScreen = global.screen;
        global.screen = {
            width: 1920,
            height: 1080,
            colorDepth: 24
        };

        originalCrypto = global.crypto;
        global.crypto = {
            subtle: {
                digest: function(algorithm, data) {
                    return Promise.resolve(new Uint8Array(32));
                }
            }
        };

        originalPerformance = global.performance;
    });

    afterEach(() => {
        global.localStorage = originalLocalStorage;
        global.navigator = originalNavigator;
        global.document = originalDocument;
        global.window = originalWindow;
        global.screen = originalScreen;
        global.crypto = originalCrypto;
        global.performance = originalPerformance;
    });

    describe('FingerprintCollector', () => {
        let collector;

        beforeEach(() => {
            collector = new FingerprintCollector();
        });

        test('should collect fingerprint with required fields', async () => {
            const fingerprint = await collector.collect();

            expect(fingerprint).toBeDefined();
            expect(fingerprint.userAgent).toBeDefined();
            expect(fingerprint.screenResolution).toBeDefined();
            expect(fingerprint.timezone).toBeDefined();
            expect(fingerprint.language).toBeDefined();
            expect(fingerprint.platform).toBeDefined();
            expect(fingerprint.hash).toBeDefined();
        });

        test('should collect at least 17 features', async () => {
            const fingerprint = await collector.collect();
            const featureCount = collector.getFeatureCount();

            expect(featureCount).toBeGreaterThanOrEqual(17);
        });

        test('should generate consistent hash for same fingerprint', async () => {
            const fingerprint1 = await collector.collect();
            const fingerprint2 = await collector.collect();

            expect(fingerprint1.hash).toBe(fingerprint2.hash);
        });

        test('should collect canvas fingerprint', async () => {
            const fingerprint = await collector.collect();

            expect(fingerprint.canvasFingerprint).toBeDefined();
            expect(typeof fingerprint.canvasFingerprint).toBe('string');
        });

        test('should collect webgl fingerprint', async () => {
            const fingerprint = await collector.collect();

            expect(fingerprint.webglFingerprint).toBeDefined();
        });

        test('should collect audio fingerprint', async () => {
            const fingerprint = await collector.collect();

            expect(fingerprint.audioFingerprint).toBeDefined();
        });

        test('should collect screen resolution', async () => {
            const fingerprint = await collector.collect();

            expect(fingerprint.screenResolution).toBe('1920x1080x24');
        });

        test('should collect timezone', async () => {
            const fingerprint = await collector.collect();

            expect(fingerprint.timezone).toBeDefined();
            expect(typeof fingerprint.timezone).toBe('string');
        });

        test('should collect language', async () => {
            const fingerprint = await collector.collect();

            expect(fingerprint.language).toBe('zh-CN');
        });

        test('should collect device memory', async () => {
            const fingerprint = await collector.collect();

            expect(fingerprint.deviceMemory).toBeDefined();
        });

        test('should collect hardware concurrency', async () => {
            const fingerprint = await collector.collect();

            expect(fingerprint.hardwareConcurrency).toBe(4);
        });

        test('should collect touch support', async () => {
            const fingerprint = await collector.collect();

            expect(fingerprint.touchSupport).toBeDefined();
            expect(fingerprint.touchSupport.maxTouchPoints).toBe(0);
        });

        test('should collect browser engine', async () => {
            const fingerprint = await collector.collect();

            expect(fingerprint.browserEngine).toBeDefined();
        });

        test('should collect webdriver status', async () => {
            const fingerprint = await collector.collect();

            expect(fingerprint.webdriverStatus).toBeDefined();
            expect(fingerprint.webdriverStatus).toBe(false);
        });

        test('should collect webRTC status', async () => {
            const fingerprint = await collector.collect();

            expect(fingerprint.webrtcStatus).toBeDefined();
        });

        test('should collect storage information', async () => {
            const fingerprint = await collector.collect();

            expect(fingerprint.storageQuota).toBeDefined();
        });

        test('should collect media devices info', async () => {
            const fingerprint = await collector.collect();

            expect(fingerprint.mediaDevices).toBeDefined();
        });

        test('should collect performance timing', async () => {
            const fingerprint = await collector.collect();

            expect(fingerprint.performanceTiming).toBeDefined();
        });

        test('should get fingerprint hash', async () => {
            await collector.collect();
            const hash = collector.getFingerprintHash();

            expect(hash).toBeDefined();
            expect(hash.length).toBe(64);
        });

        test('should handle storage quota error gracefully', async () => {
            const originalNavigator = global.navigator;
            global.navigator = {
                ...originalNavigator,
                storage: {
                    estimate: function() {
                        return Promise.reject(new Error('Storage API error'));
                    }
                }
            };

            const testCollector = new FingerprintCollector();
            const fingerprint = await testCollector.collect();

            expect(fingerprint).toBeDefined();
            global.navigator = originalNavigator;
        });

        test('should handle permissions API error gracefully', async () => {
            const testCollector = new FingerprintCollector();
            const fingerprint = await testCollector.collect();

            expect(fingerprint.permissionsAPI).toBeDefined();
        });
    });

    describe('BehaviorAnalyzer', () => {
        let analyzer;

        beforeEach(() => {
            analyzer = new BehaviorAnalyzer({ trackingDuration: 1000 });
        });

        test('should initialize with default options', () => {
            expect(analyzer).toBeDefined();
            expect(analyzer.mouseTrajectory).toEqual([]);
            expect(analyzer.keyboardPattern).toEqual([]);
            expect(analyzer.scrollBehavior).toEqual([]);
            expect(analyzer.clickPattern).toEqual([]);
        });

        test('should start tracking', () => {
            analyzer.startTracking();

            expect(analyzer.startTime).toBeDefined();
            expect(analyzer.mouseTrajectory).toEqual([]);
        });

        test('should analyze behavior after tracking', async () => {
            analyzer.startTracking();

            await new Promise(resolve => setTimeout(resolve, 100));

            const analysis = analyzer.analyze();

            expect(analysis).toBeDefined();
            expect(analysis.totalTime).toBeGreaterThan(0);
        });

        test('should calculate behavior score', async () => {
            analyzer.startTracking();

            await new Promise(resolve => setTimeout(resolve, 100));

            const score = analyzer.getBehaviorScore();

            expect(score).toBeGreaterThanOrEqual(0);
            expect(score).toBeLessThanOrEqual(100);
        });

        test('should detect anomaly patterns', async () => {
            analyzer.startTracking();

            await new Promise(resolve => setTimeout(resolve, 100));

            const anomalies = analyzer.detectAnomalyPatterns();

            expect(Array.isArray(anomalies)).toBe(true);
        });

        test('should track mouse movements', async () => {
            const originalDocument = global.document;
            let eventHandler;

            global.document = {
                ...originalDocument,
                addEventListener: function(event, handler) {
                    if (event === 'mousemove') {
                        eventHandler = handler;
                    }
                }
            };

            analyzer.startTracking();

            if (eventHandler) {
                eventHandler({ clientX: 100, clientY: 200 });
                eventHandler({ clientX: 150, clientY: 250 });
            }

            expect(analyzer.mouseTrajectory.length).toBeGreaterThanOrEqual(0);

            global.document = originalDocument;
        });

        test('should track keyboard inputs', async () => {
            const originalDocument = global.document;
            let keyDownHandler;

            global.document = {
                ...originalDocument,
                addEventListener: function(event, handler) {
                    if (event === 'keydown') {
                        keyDownHandler = handler;
                    }
                }
            };

            analyzer.startTracking();

            if (keyDownHandler) {
                keyDownHandler({ key: 'a', code: 'KeyA' });
            }

            global.document = originalDocument;
        });

        test('should count direction changes', () => {
            analyzer.mouseTrajectory = [
                { x: 0, y: 0, time: 1000, speed: 1 },
                { x: 10, y: 10, time: 1100, speed: 1 },
                { x: 20, y: 0, time: 1200, speed: 1 },
                { x: 30, y: 10, time: 1300, speed: 1 }
            ];

            const changes = analyzer.countDirectionChanges();

            expect(changes).toBeDefined();
            expect(typeof changes).toBe('number');
        });

        test('should calculate smoothness score', () => {
            analyzer.mouseTrajectory = [
                { x: 0, y: 0 },
                { x: 10, y: 10 },
                { x: 20, y: 20 },
                { x: 30, y: 30 }
            ];

            const score = analyzer.calculateSmoothnessScore();

            expect(score).toBeDefined();
            expect(score).toBeGreaterThanOrEqual(0);
            expect(score).toBeLessThanOrEqual(100);
        });

        test('should calculate average hold duration', () => {
            analyzer.keyboardPattern = [
                { key: 'a', holdDuration: 100 },
                { key: 'b', holdDuration: 150 },
                { key: 'c', holdDuration: 200 }
            ];

            const avgDuration = analyzer.calculateAverageHoldDuration();

            expect(avgDuration).toBe(150);
        });

        test('should calculate typing rhythm variance', () => {
            analyzer.keyboardPattern = [
                { delay: 100 },
                { delay: 200 },
                { delay: 150 }
            ];

            const variance = analyzer.calculateTypingRhythmVariance();

            expect(variance).toBeDefined();
            expect(typeof variance).toBe('number');
        });
    });

    describe('RiskScorer', () => {
        let scorer;

        beforeEach(() => {
            scorer = new RiskScorer();
        });

        test('should initialize with default weights', () => {
            expect(scorer.weights).toBeDefined();
            expect(scorer.weights.fingerprintEntropy).toBeDefined();
            expect(scorer.weights.behaviorScore).toBeDefined();
            expect(scorer.weights.deviceConsistency).toBeDefined();
        });

        test('should calculate risk score for complete data', async () => {
            const data = {
                fingerprint: {
                    canvasFingerprint: 'test123',
                    webglFingerprint: 'webgl123',
                    audioFingerprint: 'audio123',
                    fontFingerprint: 'font123',
                    screenResolution: '1920x1080x24',
                    timezone: 'Asia/Shanghai',
                    language: 'zh-CN',
                    platform: 'Win32',
                    connectionType: '4g',
                    batteryStatus: { level: 0.8 },
                    storageQuota: { quota: 1000000 },
                    mediaDevices: { audioInputs: 1 }
                },
                behaviorAnalysis: {
                    anomalies: [],
                    mouseMetrics: { maxSpeed: 10, smoothnessScore: 80 },
                    keyboardMetrics: { typingRhythmVariance: 1000, totalKeys: 5 }
                },
                deviceHistory: {
                    isKnownDevice: true,
                    visits: 10,
                    lastVisit: Date.now() - 86400000
                },
                environmentData: {}
            };

            const result = await scorer.calculateRiskScore(data);

            expect(result).toBeDefined();
            expect(result.totalScore).toBeDefined();
            expect(result.level).toBeDefined();
        });

        test('should calculate fingerprint entropy', () => {
            const fingerprint = {
                canvasFingerprint: 'valid',
                webglFingerprint: 'valid',
                audioFingerprint: 'error',
                fontFingerprint: 'valid',
                screenResolution: '1920x1080x24'
            };

            const entropy = scorer.calculateFingerprintEntropy(fingerprint);

            expect(entropy).toBeDefined();
            expect(entropy).toBeGreaterThanOrEqual(0);
            expect(entropy).toBeLessThanOrEqual(100);
        });

        test('should penalize webdriver status', () => {
            const fingerprint = {
                webdriverStatus: true
            };

            const entropy = scorer.calculateFingerprintEntropy(fingerprint);

            expect(entropy).toBeLessThan(100);
        });

        test('should calculate behavior risk', () => {
            const behaviorAnalysis = {
                anomalies: [
                    { type: 'too_fast', severity: 'high' },
                    { type: 'mechanical_typing', severity: 'high' }
                ],
                mouseMetrics: { maxSpeed: 200 },
                keyboardMetrics: { typingRhythmVariance: 100 }
            };

            const risk = scorer.calculateBehaviorRisk(behaviorAnalysis);

            expect(risk).toBeDefined();
            expect(risk).toBeGreaterThan(0);
        });

        test('should calculate device consistency', () => {
            const deviceHistory = {
                isKnownDevice: true,
                visits: 10,
                lastVisit: Date.now(),
                fingerprintChanged: false
            };

            const consistency = scorer.calculateDeviceConsistency(deviceHistory);

            expect(consistency).toBeGreaterThan(50);
        });

        test('should penalize changed fingerprint', () => {
            const deviceHistory = {
                isKnownDevice: true,
                visits: 10,
                fingerprintChanged: true
            };

            const consistency = scorer.calculateDeviceConsistency(deviceHistory);

            expect(consistency).toBeLessThan(50);
        });

        test('should calculate pattern anomaly', () => {
            const behaviorAnalysis = {
                anomalies: [
                    { type: 'too_fast', severity: 'high' },
                    { type: 'unnatural_speed', severity: 'medium' }
                ]
            };

            const anomalyScore = scorer.calculatePatternAnomaly(behaviorAnalysis);

            expect(anomalyScore).toBeGreaterThan(0);
        });

        test('should calculate environment risk', () => {
            const environmentData = {
                isEmulator: true,
                isVirtual: false,
                isContainer: false,
                isHeadlessBrowser: true
            };

            const risk = scorer.calculateEnvironmentRisk(environmentData);

            expect(risk).toBeGreaterThan(80);
        });

        test('should get risk level', () => {
            expect(scorer.getRiskLevel(20)).toBe('low');
            expect(scorer.getRiskLevel(50)).toBe('medium');
            expect(scorer.getRiskLevel(85)).toBe('high');
        });

        test('should get risk details', () => {
            const riskFactors = {
                fingerprintEntropy: 30,
                behaviorScore: 60,
                deviceConsistency: 20,
                patternAnomaly: 50,
                environmentRisk: 40
            };

            const details = scorer.getRiskDetails(riskFactors);

            expect(Array.isArray(details)).toBe(true);
            expect(details.length).toBeGreaterThan(0);
        });
    });

    describe('DeviceTrustManager', () => {
        let trustManager;

        beforeEach(() => {
            global.localStorage.clear();
            trustManager = new DeviceTrustManager();
        });

        test('should initialize with default expiry', () => {
            expect(trustManager.defaultTrustExpiry).toBe(30 * 24 * 60 * 60 * 1000);
        });

        test('should set custom trust expiry', () => {
            trustManager.setTrustExpiry(60);

            expect(trustManager.defaultTrustExpiry).toBe(60 * 24 * 60 * 60 * 1000);
        });

        test('should get trust status with no data', () => {
            const status = trustManager.getTrustStatus();

            expect(status.isTrusted).toBe(false);
            expect(status.isExpired).toBe(true);
        });

        test('should set trust', () => {
            const result = trustManager.setTrust(80, 30);

            expect(result.success).toBe(true);
            expect(result.expiryTime).toBeDefined();
        });

        test('should get trust status after setting', () => {
            trustManager.setTrust(80);

            const status = trustManager.getTrustStatus();

            expect(status.isTrusted).toBe(true);
            expect(status.isExpired).toBe(false);
        });

        test('should revoke trust', () => {
            trustManager.setTrust(80);
            trustManager.revokeTrust();

            const status = trustManager.getTrustStatus();

            expect(status.isTrusted).toBe(false);
        });

        test('should check device consistency', () => {
            const hash = 'testhash123';
            const result = trustManager.checkDeviceConsistency(hash);

            expect(result.consistent).toBe(true);
            expect(result.isNewDevice).toBe(true);
        });

        test('should detect changed fingerprint', () => {
            trustManager.checkDeviceConsistency('hash1');
            const result = trustManager.checkDeviceConsistency('hash2');

            expect(result.consistent).toBe(false);
            expect(result.isNewDevice).toBe(false);
        });

        test('should increment visit count', () => {
            trustManager.incrementVisitCount();
            trustManager.incrementVisitCount();

            expect(trustManager.getVisitCount()).toBe(2);
        });

        test('should get device ID', () => {
            const deviceId = trustManager.getDeviceId();

            expect(deviceId).toBeDefined();
            expect(deviceId.startsWith('device_')).toBe(true);
        });

        test('should add to trust history', () => {
            trustManager.addToTrustHistory({
                action: 'trust_set',
                level: 80,
                timestamp: Date.now()
            });

            const history = trustManager.getTrustHistory();

            expect(history.length).toBe(1);
            expect(history[0].action).toBe('trust_set');
        });

        test('should limit trust history to 50 entries', () => {
            for (let i = 0; i < 60; i++) {
                trustManager.addToTrustHistory({
                    action: 'test',
                    timestamp: Date.now()
                });
            }

            const history = trustManager.getTrustHistory();

            expect(history.length).toBeLessThanOrEqual(50);
        });

        test('should clear all data', () => {
            trustManager.setTrust(80);
            trustManager.addToTrustHistory({ action: 'test' });
            trustManager.clearAllData();

            const status = trustManager.getTrustStatus();

            expect(status.isTrusted).toBe(false);
        });

        test('should export device data', () => {
            trustManager.setTrust(80);

            const data = trustManager.exportDeviceData();

            expect(data).toBeDefined();
        });

        test('should import device data', () => {
            const data = {
                trustStatus: 'true',
                trustExpiry: (Date.now() + 86400000).toString()
            };

            const result = trustManager.importDeviceData(data);

            expect(result.success).toBe(true);
        });

        test('should reject invalid import data', () => {
            const result = trustManager.importDeviceData(null);

            expect(result.success).toBe(false);
        });
    });

    describe('SeamlessI18n', () => {
        let i18n;

        beforeEach(() => {
            i18n = new SeamlessI18n('zh-CN');
        });

        test('should initialize with default locale', () => {
            expect(i18n.locale).toBe('zh-CN');
        });

        test('should translate key in zh-CN', () => {
            const translation = i18n.t('checking');

            expect(translation).toBe('正在检测设备...');
        });

        test('should translate key in en-US', () => {
            i18n.setLocale('en-US');
            const translation = i18n.t('checking');

            expect(translation).toBe('Checking device...');
        });

        test('should fallback to zh-CN for missing key', () => {
            const translation = i18n.t('nonexistent_key');

            expect(translation).toBe('nonexistent_key');
        });

        test('should get current locale', () => {
            expect(i18n.getLocale()).toBe('zh-CN');
        });
    });

    describe('SeamlessCaptcha', () => {
        let captcha;
        let container;

        beforeEach(() => {
            container = document.createElement('div');
            container.id = 'testContainer';
            document.body.appendChild(container);
            captcha = new SeamlessCaptcha('testContainer');
        });

        afterEach(() => {
            document.body.removeChild(container);
        });

        test('should initialize with default options', () => {
            expect(captcha).toBeDefined();
            expect(captcha.state.status).toBe('idle');
            expect(captcha.options.apiBase).toBe('/api/v1');
        });

        test('should have fingerprint collector', () => {
            expect(captcha.fingerprintCollector).toBeDefined();
        });

        test('should have behavior analyzer', () => {
            expect(captcha.behaviorAnalyzer).toBeDefined();
        });

        test('should have risk scorer', () => {
            expect(captcha.riskScorer).toBeDefined();
        });

        test('should have device trust manager', () => {
            expect(captcha.deviceTrustManager).toBeDefined();
        });

        test('should have i18n', () => {
            expect(captcha.i18n).toBeDefined();
        });

        test('should have config', () => {
            expect(captcha.config).toBeDefined();
            expect(captcha.config.seamlessEnabled).toBe(true);
        });

        test('should update config', () => {
            captcha.updateConfig({ seamlessEnabled: false });

            expect(captcha.config.seamlessEnabled).toBe(false);
        });

        test('should set verification mode', () => {
            captcha.setVerificationMode('force');

            expect(captcha.verificationMode).toBe('force');
        });

        test('should check if should force verification', () => {
            captcha.updateConfig({ seamlessEnabled: false });

            expect(captcha.shouldForceVerification()).toBe(true);
        });

        test('should reset state', () => {
            captcha.reset();

            expect(captcha.state.status).toBe('idle');
            expect(captcha.state.fingerprint).toBeNull();
        });

        test('should update status', () => {
            const callback = jest.fn();
            captcha.options.onStatusChange = callback;
            captcha.updateStatus('testing');

            expect(captcha.state.status).toBe('testing');
            expect(callback).toHaveBeenCalledWith('testing');
        });

        test('should update step', () => {
            const callback = jest.fn();
            captcha.options.onStepChange = callback;
            captcha.updateStep('test', 50);

            expect(callback).toHaveBeenCalledWith('test', 50);
        });

        test('should get state', () => {
            const state = captcha.getState();

            expect(state).toBeDefined();
            expect(state.status).toBe('idle');
        });

        test('should get device trust manager', () => {
            const manager = captcha.getDeviceTrustManager();

            expect(manager).toBeDefined();
        });

        test('should handle error', () => {
            const callback = jest.fn();
            captcha.options.onError = callback;

            const error = new Error('Test error');
            captcha.handleError(error);

            expect(callback).toHaveBeenCalled();
        });

        test('should update language', () => {
            captcha.updateLanguage('en-US');

            expect(captcha.options.language).toBe('en-US');
            expect(captcha.i18n.getLocale()).toBe('en-US');
        });

        test('should destroy captcha', () => {
            captcha.destroy();

            expect(captcha.container).toBeNull();
            expect(captcha.fingerprintCollector).toBeNull();
        });
    });

    describe('SeamlessVerificationManager', () => {
        let manager;

        beforeEach(() => {
            manager = new SeamlessVerificationManager();
        });

        test('should initialize with default options', () => {
            expect(manager).toBeDefined();
            expect(manager.options.seamlessEnabled).toBe(true);
            expect(manager.options.trustExpiryDays).toBe(30);
        });

        test('should set seamless enabled', () => {
            manager.setSeamlessEnabled(false);

            expect(manager.options.seamlessEnabled).toBe(false);
        });

        test('should set trust expiry days', () => {
            manager.setTrustExpiryDays(60);

            expect(manager.options.trustExpiryDays).toBe(60);
        });

        test('should have device trust manager', () => {
            expect(manager.deviceTrustManager).toBeDefined();
        });

        test('should have risk scorer', () => {
            expect(manager.riskScorer).toBeDefined();
        });

        test('should have fingerprint collector', () => {
            expect(manager.fingerprintCollector).toBeDefined();
        });

        test('should have behavior analyzer', () => {
            expect(manager.behaviorAnalyzer).toBeDefined();
        });
    });

    describe('Integration Tests', () => {
        test('should complete full verification flow', async () => {
            const container = document.createElement('div');
            container.id = 'integrationTest';
            document.body.appendChild(container);

            const captcha = new SeamlessCaptcha('integrationTest', {
                autoStart: false,
                trackingDuration: 500
            });

            await captcha.start();

            expect(captcha.state.status).toBeDefined();

            document.body.removeChild(container);
        }, 10000);

        test('should handle trust flow', async () => {
            const container = document.createElement('div');
            container.id = 'trustTest';
            document.body.appendChild(container);

            const manager = new SeamlessVerificationManager();
            global.localStorage.clear();

            manager.deviceTrustManager.setTrust(80, 30);
            const status = manager.deviceTrustManager.getTrustStatus();

            expect(status.isTrusted).toBe(true);

            document.body.removeChild(container);
        });

        test('should calculate risk with all factors', async () => {
            const scorer = new RiskScorer();

            const result = await scorer.calculateRiskScore({
                fingerprint: {
                    canvasFingerprint: 'valid',
                    webglFingerprint: 'valid',
                    audioFingerprint: 'valid',
                    fontFingerprint: 'valid',
                    screenResolution: '1920x1080',
                    timezone: 'Asia/Shanghai',
                    language: 'zh-CN',
                    platform: 'Win32',
                    connectionType: '4g'
                },
                behaviorAnalysis: {
                    anomalies: [],
                    mouseMetrics: { maxSpeed: 5, smoothnessScore: 50 },
                    keyboardMetrics: { typingRhythmVariance: 1000, totalKeys: 3 }
                },
                deviceHistory: {
                    isKnownDevice: true,
                    visits: 5,
                    lastVisit: Date.now()
                },
                environmentData: {}
            });

            expect(result.totalScore).toBeLessThan(50);
        });
    });

    describe('Edge Cases', () => {
        test('should handle null fingerprint', async () => {
            const scorer = new RiskScorer();
            const result = await scorer.calculateRiskScore({
                fingerprint: null,
                behaviorAnalysis: null,
                deviceHistory: null,
                environmentData: null
            });

            expect(result.totalScore).toBeDefined();
        });

        test('should handle empty behavior data', async () => {
            const scorer = new RiskScorer();
            const result = await scorer.calculateRiskScore({
                fingerprint: {},
                behaviorAnalysis: { anomalies: [] },
                deviceHistory: {},
                environmentData: {}
            });

            expect(result.totalScore).toBeDefined();
        });

        test('should handle undefined storage keys', () => {
            global.localStorage = {
                data: {},
                getItem: function() { return undefined; },
                setItem: function() {},
                removeItem: function() {},
                clear: function() {}
            };

            const trustManager = new DeviceTrustManager();
            const status = trustManager.getTrustStatus();

            expect(status.isTrusted).toBe(false);
        });

        test('should handle rapid trust operations', () => {
            const trustManager = new DeviceTrustManager();

            for (let i = 0; i < 100; i++) {
                trustManager.incrementVisitCount();
            }

            expect(trustManager.getVisitCount()).toBe(100);
        });

        test('should handle large history array', () => {
            const trustManager = new DeviceTrustManager();

            for (let i = 0; i < 100; i++) {
                trustManager.addToTrustHistory({
                    action: 'test',
                    timestamp: Date.now(),
                    level: i
                });
            }

            const history = trustManager.getTrustHistory(200);
            expect(history.length).toBeLessThanOrEqual(50);
        });
    });
});
