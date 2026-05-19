describe('HJTPX Environment Detector', function() {
    let detector;

    beforeEach(function() {
        detector = new HJTPXEnvironmentDetector({
            chainCount: 10,
            sampleRate: 1.0
        });
    });

    describe('Canvas Fingerprint (256-bit)', function() {
        it('should generate a 64-character SHA-256 fingerprint', async function() {
            const result = await detector.detectCanvas256();
            expect(result.fingerprint).toBeDefined();
            expect(result.fingerprint.length).toBe(64);
        });

        it('should return valid detection result', async function() {
            const result = await detector.detectCanvas256();
            expect(result.detected).toBeDefined();
            expect(typeof result.score).toBe('number');
            expect(Array.isArray(result.detections)).toBe(true);
        });
    });

    describe('WebGL Detection', function() {
        it('should detect WebGL capabilities', async function() {
            const result = await detector.detectWebGLEnhanced();
            expect(result.detected).toBeDefined();
            expect(typeof result.score).toBe('number');
        });

        it('should detect WebGL2 capabilities', async function() {
            const result = await detector.detectWebGL2Enhanced();
            expect(result.detected).toBeDefined();
            expect(typeof result.score).toBe('number');
        });
    });

    describe('Proxy/VPN Detection', function() {
        it('should detect proxy/VPN indicators', async function() {
            const result = await detector.detectProxyVPN();
            expect(result.detected).toBeDefined();
            expect(typeof result.score).toBe('number');
        });

        it('should detect VPN indicators', async function() {
            const result = await detector.detectVPNIndicators();
            expect(result.detected).toBeDefined();
            expect(typeof result.score).toBe('number');
        });

        it('should detect WebRTC leaks', async function() {
            const result = await detector.detectWebRTCLeak();
            expect(result.detected).toBeDefined();
            expect(typeof result.score).toBe('number');
        });
    });

    describe('Headless Detection', function() {
        it('should detect headless browser indicators', async function() {
            const result = await detector.detectHeadless();
            expect(result.detected).toBeDefined();
            expect(typeof result.score).toBe('number');
        });

        it('should detect advanced headless indicators', async function() {
            const result = await detector.detectAdvancedHeadless();
            expect(result.detected).toBeDefined();
            expect(typeof result.score).toBe('number');
        });

        it('should detect stealth mode', async function() {
            const result = await detector.detectStealthMode();
            expect(result.detected).toBeDefined();
            expect(typeof result.score).toBe('number');
        });
    });

    describe('Automation Detection', function() {
        it('should detect WebDriver', async function() {
            const result = await detector.detectWebDriver();
            expect(result.detected).toBeDefined();
            expect(typeof result.score).toBe('number');
        });

        it('should detect Puppeteer', async function() {
            const result = await detector.detectPuppeteer();
            expect(result.detected).toBeDefined();
            expect(typeof result.score).toBe('number');
        });

        it('should detect Playwright', async function() {
            const result = await detector.detectPlaywright();
            expect(result.detected).toBeDefined();
            expect(typeof result.score).toBe('number');
        });

        it('should detect Selenium', async function() {
            const result = await detector.detectSelenium();
            expect(result.detected).toBeDefined();
            expect(typeof result.score).toBe('number');
        });

        it('should detect automation frameworks', async function() {
            const result = await detector.detectAutomationFramework();
            expect(result.detected).toBeDefined();
            expect(typeof result.score).toBe('number');
        });
    });

    describe('Emulator/Virtualization Detection', function() {
        it('should detect emulators', async function() {
            const result = await detector.detectEmulator();
            expect(result.detected).toBeDefined();
            expect(typeof result.score).toBe('number');
        });

        it('should detect virtualization', async function() {
            const result = await detector.detectVirtualization();
            expect(result.detected).toBeDefined();
            expect(typeof result.score).toBe('number');
        });

        it('should detect containers', async function() {
            const result = await detector.detectContainer();
            expect(result.detected).toBeDefined();
            expect(typeof result.score).toBe('number');
        });
    });

    describe('Device Fingerprint', function() {
        it('should generate a device fingerprint', function() {
            const fingerprint = detector.generateDeviceFingerprint();
            expect(fingerprint).toBeDefined();
            expect(fingerprint.length).toBe(64);
        });

        it('should detect device fingerprint', async function() {
            const result = await detector.detectDeviceFingerprint();
            expect(result.fingerprint).toBeDefined();
            expect(result.fingerprint.length).toBe(64);
        });
    });

    describe('Detection Chain', function() {
        it('should run detection chain', async function() {
            const result = await detector.runChain();
            expect(result.detection_id).toBeDefined();
            expect(result.risk_score).toBeDefined();
            expect(result.fingerprint).toBeDefined();
            expect(typeof result.risk_score).toBe('number');
            expect(result.risk_score).toBeGreaterThanOrEqual(0);
            expect(result.risk_score).toBeLessThanOrEqual(100);
        });

        it('should return valid chain results', async function() {
            const result = await detector.runChain();
            expect(result.chain).toBeDefined();
            expect(result.chain_order).toBeDefined();
            expect(Array.isArray(result.chain_order)).toBe(true);
        });
    });

    describe('Risk Score Calculation', function() {
        it('should calculate risk score between 0 and 100', function() {
            detector.results = {
                detectHeadless: { detected: false, score: 10 },
                detectWebDriver: { detected: false, score: 5 },
                detectCanvas256: { detected: false, score: 0 }
            };
            const score = detector.calculateRiskScore();
            expect(score).toBeGreaterThanOrEqual(0);
            expect(score).toBeLessThanOrEqual(100);
        });
    });

    describe('ML Features Extraction', function() {
        it('should extract ML features', async function() {
            await detector.runChain();
            expect(detector.mlFeatures).toBeDefined();
            expect(typeof detector.mlFeatures.total_checks).toBe('number');
            expect(typeof detector.mlFeatures.avg_score).toBe('number');
        });
    });

    describe('Helper Functions', function() {
        it('should hash strings with SHA-256', function() {
            const hash = detector.sha256Hash('test');
            expect(hash).toBeDefined();
            expect(hash.length).toBe(64);
        });

        it('should compare fingerprints', function() {
            const similarity = detector.compareFingerprints('abc123', 'abc123');
            expect(similarity).toBe(1.0);
        });

        it('should create PRNG with seed', function() {
            const prng = detector.createPRNG(12345);
            const value = prng();
            expect(typeof value).toBe('number');
            expect(value).toBeGreaterThanOrEqual(0);
            expect(value).toBeLessThan(1);
        });
    });

    describe('Detection Methods', function() {
        it('should return detection methods with categories', function() {
            const methods = detector.getDetectionMethods();
            expect(Array.isArray(methods)).toBe(true);
            expect(methods.length).toBeGreaterThan(0);
            methods.forEach(method => {
                expect(method.name).toBeDefined();
                expect(method.category).toBeDefined();
            });
        });

        it('should generate detection chain', function() {
            const chain = detector.generateDetectionChain(5);
            expect(chain.selected).toBeDefined();
            expect(chain.methodAliases).toBeDefined();
            expect(chain.selected.length).toBe(5);
        });
    });
});