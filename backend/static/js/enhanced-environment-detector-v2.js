class EnhancedEnvironmentDetector {
    constructor(options = {}) {
        this.options = Object.assign({
            apiBase: '/api/v1',
            sampleRate: 1.0,
            chainCount: 30,
            enableAll: true,
            sessionId: null,
            timeout: 15000,
            retries: 3,
            enableAdvancedDetection: true,
            enableMLScoring: true
        }, options);
        this.results = {};
        this.riskScore = 0;
        this.detectionChain = [];
        this.detectionId = 'det_v2_' + Date.now() + '_' + Math.random().toString(36).substr(2, 12);
        this.timingData = {};
        this.fingerprintComponents = {};
        this.mlFeatures = {};
        this.mlModel = null;
        this.mlThreshold = 0.75;
        this.weights = {
            canvas: 14,
            canvasStable: 12,
            canvasEntropy: 10,
            webgl: 16,
            webgl2: 12,
            webglVendor: 10,
            webglRenderer: 10,
            audio: 13,
            fonts: 12,
            fontEnumeration: 10,
            fontMetrics: 8,
            plugins: 8,
            pluginFingerprint: 7,
            webrtc: 17,
            webrtcLeak: 14,
            webdriver: 22,
            selenium: 20,
            puppeteer: 20,
            playwright: 20,
            chromeRuntime: 12,
            headless: 17,
            permissions: 8,
            languages: 6,
            timezone: 7,
            screen: 5,
            hardware: 7,
            memory: 6,
            storage: 7,
            navigator: 7,
            windowProps: 6,
            iframe: 8,
            notification: 4,
            battery: 5,
            mediaDevices: 7,
            connection: 10,
            adblock: 6,
            math: 5,
            gpu: 9,
            speech: 4,
            proxyVPN: 20,
            torExitNode: 17,
            vpnIndicators: 16,
            virtualization: 14,
            sandbox: 12,
            automationFrameworks: 18,
            vmFeatures: 17,
            sandboxEscape: 16,
            debuggerDetection: 14,
            advancedHeadless: 18,
            stealthMode: 15,
            browserProfile: 12,
            timingAnomaly: 10,
            networkFingerprint: 11,
            behaviorPattern: 13
        };
    }

    getDetectionMethods() {
        return [
            { name: 'detectHeadless', category: 'automation' },
            { name: 'detectWebDriver', category: 'automation' },
            { name: 'detectPuppeteer', category: 'automation' },
            { name: 'detectPlaywright', category: 'automation' },
            { name: 'detectSelenium', category: 'automation' },
            { name: 'detectChromeRuntime', category: 'automation' },
            { name: 'detectAutomationFrameworks', category: 'automation' },
            { name: 'detectAdvancedHeadless', category: 'automation' },
            { name: 'detectStealthMode', category: 'automation' },
            { name: 'detectVirtualization', category: 'environment' },
            { name: 'detectSandbox', category: 'environment' },
            { name: 'detectCanvasEnhanced', category: 'fingerprint' },
            { name: 'detectCanvasStable', category: 'fingerprint' },
            { name: 'detectCanvasEntropy', category: 'fingerprint' },
            { name: 'detectWebGLEnhanced', category: 'fingerprint' },
            { name: 'detectWebGL2Enhanced', category: 'fingerprint' },
            { name: 'detectWebGLExtended', category: 'fingerprint' },
            { name: 'detectAudioEnhanced', category: 'fingerprint' },
            { name: 'detectFontsEnhanced', category: 'fingerprint' },
            { name: 'detectFontEnumeration', category: 'fingerprint' },
            { name: 'detectFontMetrics', category: 'fingerprint' },
            { name: 'detectPluginsEnhanced', category: 'fingerprint' },
            { name: 'detectPluginFingerprint', category: 'fingerprint' },
            { name: 'detectBrowserProfile', category: 'fingerprint' },
            { name: 'detectWebRTCEnhanced', category: 'network' },
            { name: 'detectWebRTCLeak', category: 'network' },
            { name: 'detectProxyVPN', category: 'network' },
            { name: 'detectVPNIndicators', category: 'network' },
            { name: 'detectTorExitNode', category: 'network' },
            { name: 'detectNetworkFingerprint', category: 'network' },
            { name: 'detectPermissions', category: 'system' },
            { name: 'detectLanguages', category: 'system' },
            { name: 'detectTimezone', category: 'system' },
            { name: 'detectScreen', category: 'system' },
            { name: 'detectHardwareConcurrency', category: 'system' },
            { name: 'detectDeviceMemory', category: 'system' },
            { name: 'detectStorage', category: 'system' },
            { name: 'detectNavigatorProps', category: 'system' },
            { name: 'detectWindowProps', category: 'system' },
            { name: 'detectIframe', category: 'system' },
            { name: 'detectNotification', category: 'system' },
            { name: 'detectBattery', category: 'system' },
            { name: 'detectMediaDevices', category: 'system' },
            { name: 'detectConnection', category: 'network' },
            { name: 'detectAdBlock', category: 'system' },
            { name: 'detectMathFingerprint', category: 'fingerprint' },
            { name: 'detectGPUFingerprint', category: 'fingerprint' },
            { name: 'detectSpeech', category: 'system' },
            { name: 'detectVMFeatures', category: 'vm' },
            { name: 'detectSandboxEscape', category: 'sandbox' },
            { name: 'detectDebuggerEnhanced', category: 'debugger' },
            { name: 'detectTimingAnomaly', category: 'behavior' },
            { name: 'detectBehaviorPattern', category: 'behavior' }
        ];
    }

    generateDetectionChain(count) {
        const allMethods = this.getDetectionMethods();
        const shuffled = [...allMethods].sort(() => Math.random() - 0.5);
        const selected = shuffled.slice(0, Math.min(count, allMethods.length));
        const methodAliases = {};
        selected.forEach((method, i) => {
            methodAliases[method.name] = 'chk_' + i.toString(36) + '_' + Math.random().toString(36).substr(2, 5);
        });
        return { selected, methodAliases };
    }

    async runChain() {
        const { selected, methodAliases } = this.generateDetectionChain(this.options.chainCount);
        this.detectionChain = selected;
        const chainResults = {};
        const startTime = performance.now();

        for (const method of selected) {
            const methodStart = performance.now();
            try {
                const alias = methodAliases[method.name];
                const result = await this[method.name]();
                const methodDuration = performance.now() - methodStart;
                result.duration_ms = Math.round(methodDuration);
                chainResults[alias] = result;
                this.results[method.name] = result;
                this.timingData[method.name] = methodDuration;
            } catch (e) {
                const alias = methodAliases[method.name];
                chainResults[alias] = { detected: false, score: 0, error: e.message, duration_ms: 0 };
            }
        }

        const duration = performance.now() - startTime;
        this.riskScore = this.calculateRiskScore();
        this.extractMLFeatures();

        return {
            detection_id: this.detectionId,
            chain: chainResults,
            chain_order: Object.values(methodAliases),
            chain_categories: selected.map(m => m.category),
            risk_score: this.riskScore,
            duration_ms: Math.round(duration),
            timing_data: this.timingData,
            timestamp: Date.now(),
            fingerprint: this.generateFingerprint(),
            ml_features: this.mlFeatures,
            ml_risk_score: this.options.enableMLScoring ? this.calculateMLRiskScore() : null
        };
    }

    extractMLFeatures() {
        this.mlFeatures = {
            total_checks: Object.keys(this.results).length,
            detected_checks: 0,
            avg_score: 0,
            max_score: 0,
            automation_score: 0,
            fingerprint_score: 0,
            network_score: 0,
            system_score: 0,
            vm_score: 0,
            suspicious_patterns: [],
            timing_variance: this.calculateTimingVariance(),
            entropy_score: this.calculateEntropyScore(),
            consistency_score: this.calculateConsistencyScore()
        };

        let totalScore = 0;
        let count = 0;

        for (const key in this.results) {
            const result = this.results[key];
            if (result && typeof result.score === 'number') {
                totalScore += result.score;
                count++;
                if (result.score > this.mlFeatures.max_score) {
                    this.mlFeatures.max_score = result.score;
                }
                if (result.detected) {
                    this.mlFeatures.detected_checks++;
                    this.mlFeatures.suspicious_patterns.push(...(result.detections || []));
                }
            }

            const category = this.getMethodCategory(key);
            if (category && result && result.score > 0) {
                switch (category) {
                    case 'automation':
                        this.mlFeatures.automation_score += result.score;
                        break;
                    case 'fingerprint':
                        this.mlFeatures.fingerprint_score += result.score;
                        break;
                    case 'network':
                        this.mlFeatures.network_score += result.score;
                        break;
                    case 'system':
                        this.mlFeatures.system_score += result.score;
                        break;
                    case 'vm':
                        this.mlFeatures.vm_score += result.score;
                        break;
                }
            }
        }

        if (count > 0) {
            this.mlFeatures.avg_score = totalScore / count;
        }
    }

    calculateMLRiskScore() {
        let mlScore = 0;

        mlScore += this.mlFeatures.detected_checks * 5;

        mlScore += Math.min(this.mlFeatures.automation_score * 0.3, 30);
        mlScore += Math.min(this.mlFeatures.fingerprint_score * 0.2, 20);
        mlScore += Math.min(this.mlFeatures.network_score * 0.25, 25);
        mlScore += Math.min(this.mlFeatures.vm_score * 0.25, 20);

        if (this.mlFeatures.timing_variance > 0.8) {
            mlScore += 10;
        }

        if (this.mlFeatures.entropy_score < 0.2) {
            mlScore += 8;
        }

        if (this.mlFeatures.consistency_score < 0.7) {
            mlScore += 7;
        }

        const highRiskPatterns = ['headless', 'webdriver', 'puppeteer', 'playwright', 'selenium', 'tor', 'vpn', 'proxy', 'vm_', 'sandbox'];
        for (const pattern of highRiskPatterns) {
            if (this.mlFeatures.suspicious_patterns.some(p => p.toLowerCase().includes(pattern))) {
                mlScore += 3;
            }
        }

        return Math.round(Math.min(Math.max(mlScore, 0), 100));
    }

    calculateTimingVariance() {
        const timings = Object.values(this.timingData);
        if (timings.length < 2) return 0;

        const mean = timings.reduce((a, b) => a + b, 0) / timings.length;
        const variance = timings.reduce((sum, t) => sum + Math.pow(t - mean, 2), 0) / timings.length;
        const stdDev = Math.sqrt(variance);

        return stdDev / mean;
    }

    calculateEntropyScore() {
        const scores = [];
        for (const key in this.results) {
            if (this.results[key] && this.results[key].score > 0) {
                scores.push(this.results[key].score);
            }
        }

        if (scores.length === 0) return 1;

        const maxPossibleScore = scores.length * 100;
        const actualScore = scores.reduce((a, b) => a + b, 0);

        return actualScore / maxPossibleScore;
    }

    calculateConsistencyScore() {
        let consistentCount = 0;
        let totalChecks = 0;

        const canvasChecks = ['detectCanvasEnhanced', 'detectCanvasStable', 'detectCanvasEntropy'];
        if (canvasChecks.every(m => this.results[m])) {
            const scores = canvasChecks.map(m => this.results[m].score);
            const variance = this.calculateVariance(scores);
            if (variance < 20) consistentCount++;
            totalChecks++;
        }

        const webglChecks = ['detectWebGLEnhanced', 'detectWebGL2Enhanced', 'detectWebGLExtended'];
        if (webglChecks.every(m => this.results[m])) {
            const scores = webglChecks.map(m => this.results[m].score);
            const variance = this.calculateVariance(scores);
            if (variance < 15) consistentCount++;
            totalChecks++;
        }

        const headlessChecks = ['detectHeadless', 'detectAdvancedHeadless', 'detectStealthMode'];
        if (headlessChecks.every(m => this.results[m])) {
            const detected = headlessChecks.filter(m => this.results[m].detected).length;
            if (detected === 0 || detected === headlessChecks.length) consistentCount++;
            totalChecks++;
        }

        return totalChecks > 0 ? consistentCount / totalChecks : 1;
    }

    calculateVariance(arr) {
        if (arr.length < 2) return 0;
        const mean = arr.reduce((a, b) => a + b, 0) / arr.length;
        return arr.reduce((sum, val) => sum + Math.pow(val - mean, 2), 0) / arr.length;
    }

    calculateRiskScore() {
        let weightedScore = 0;
        let totalWeight = 0;
        const categoryScores = { automation: 0, fingerprint: 0, network: 0, system: 0, environment: 0, vm: 0, sandbox: 0, debugger: 0, behavior: 0 };
        const categoryWeights = { automation: 0, fingerprint: 0, network: 0, system: 0, environment: 0, vm: 0, sandbox: 0, debugger: 0, behavior: 0 };

        for (const key in this.results) {
            const result = this.results[key];
            if (result && typeof result.score === 'number') {
                const weight = this.weights[key] || 5;
                const category = this.getMethodCategory(key);
                if (category) {
                    categoryScores[category] += result.score * weight;
                    categoryWeights[category] += weight;
                }
                weightedScore += result.score * weight;
                totalWeight += weight;
            }
        }

        if (totalWeight === 0) return 0;

        let baseScore = weightedScore / totalWeight;

        const automationMethods = ['detectWebDriver', 'detectPuppeteer', 'detectPlaywright', 'detectSelenium', 'detectAutomationFrameworks', 'detectAdvancedHeadless', 'detectStealthMode'];
        const automationDetected = automationMethods.filter(m => {
            const r = this.results[m];
            return r && r.detected === true;
        }).length;

        if (automationDetected >= 4) {
            baseScore = Math.min(baseScore * 2.0 + 30, 100);
        } else if (automationDetected >= 3) {
            baseScore = Math.min(baseScore * 1.8 + 25, 100);
        } else if (automationDetected >= 2) {
            baseScore = Math.min(baseScore * 1.5 + 15, 100);
        } else if (automationDetected >= 1) {
            baseScore = Math.min(baseScore * 1.3 + 10, 100);
        }

        const proxyMethods = ['detectProxyVPN', 'detectVPNIndicators', 'detectTorExitNode', 'detectWebRTCLeak'];
        const proxyDetected = proxyMethods.filter(m => {
            const r = this.results[m];
            return r && r.detected === true;
        }).length;

        if (proxyDetected >= 3) {
            baseScore = Math.min(baseScore * 1.6 + 25, 100);
        } else if (proxyDetected >= 2) {
            baseScore = Math.min(baseScore * 1.4 + 20, 100);
        } else if (proxyDetected >= 1) {
            baseScore = Math.min(baseScore * 1.2 + 10, 100);
        }

        const virtualizationMethods = ['detectVirtualization', 'detectSandbox', 'detectVMFeatures'];
        const virtualizationDetected = virtualizationMethods.filter(m => {
            const r = this.results[m];
            return r && r.detected === true;
        }).length;

        if (virtualizationDetected >= 3) {
            baseScore = Math.min(baseScore * 1.4 + 20, 100);
        } else if (virtualizationDetected >= 2) {
            baseScore = Math.min(baseScore * 1.3 + 15, 100);
        }

        const behaviorMethods = ['detectTimingAnomaly', 'detectBehaviorPattern'];
        const behaviorDetected = behaviorMethods.filter(m => {
            const r = this.results[m];
            return r && r.detected === true;
        }).length;

        if (behaviorDetected >= 1) {
            baseScore = Math.min(baseScore * 1.15 + 8, 100);
        }

        return Math.round(Math.min(Math.max(baseScore, 0), 100));
    }

    getMethodCategory(methodName) {
        const methods = this.getDetectionMethods();
        const method = methods.find(m => m.name === methodName);
        return method ? method.category : null;
    }

    async detectAdvancedHeadless() {
        let score = 0;
        const detections = [];

        try {
            if (navigator.webdriver === true) {
                score += 35;
                detections.push('navigator_webdriver_true');
            }

            const testChrome = window.chrome;
            if (testChrome) {
                const hasRuntime = testChrome.runtime !== undefined;
                const hasApp = testChrome.app !== undefined;
                const hasLoadTimes = testChrome.loadTimes !== undefined;
                const hasCsi = testChrome.csi !== undefined;

                const chromeFeatures = [hasRuntime, hasApp, hasLoadTimes, hasCsi].filter(Boolean).length;

                if (chromeFeatures < 2) {
                    score += 25;
                    detections.push('chrome_features_missing');
                }

                if (!hasRuntime) {
                    score += 15;
                    detections.push('chrome_runtime_missing');
                }
            }

            const testCanvas = document.createElement('canvas');
            const testCtx = testCanvas.getContext('2d');
            if (testCtx) {
                testCtx.fillStyle = 'rgba(255, 128, 0, 0.5)';
                testCtx.fillRect(0, 0, 5, 5);
                const imgData = testCtx.getImageData(0, 0, 5, 5);
                const hasContent = Array.from(imgData.data).some(v => v > 0);
                if (!hasContent) {
                    score += 20;
                    detections.push('canvas_no_content');
                }
            }

            const screenProps = [window.screen.width, window.screen.height, window.screen.colorDepth];
            const unusualScreenSizes = ['800x600', '1024x768'];
            const screenStr = `${screenProps[0]}x${screenProps[1]}`;
            if (unusualScreenSizes.includes(screenStr)) {
                score += 15;
                detections.push('unusual_screen_size:' + screenStr);
            }

            const testPlugins = navigator.plugins;
            if (!testPlugins || testPlugins.length === 0) {
                score += 20;
                detections.push('no_plugins');
            }

            const testLangs = navigator.languages;
            if (!testLangs || testLangs.length === 0) {
                score += 20;
                detections.push('no_languages');
            }

            const testMimeTypes = navigator.mimeTypes;
            if (!testMimeTypes || testMimeTypes.length === 0) {
                score += 20;
                detections.push('no_mimetypes');
            }

            if (window.outerWidth === 0 || window.outerHeight === 0) {
                score += 30;
                detections.push('zero_outer_dimensions');
            }

            if (window.innerWidth > window.outerWidth || window.innerHeight > window.outerHeight) {
                score += 25;
                detections.push('inner_larger_than_outer');
            }

            try {
                const testEl = document.createElement('div');
                testEl.style.cssText = 'position:absolute;left:-9999px;top:-9999px';
                const testStyle = window.getComputedStyle(testEl);
                if (testStyle.position !== 'absolute') {
                    score += 15;
                    detections.push('css_mismatch');
                }
            } catch (e) {
                score += 10;
                detections.push('css_check_error');
            }

            try {
                const perfData = performance.getEntriesByType('navigation');
                if (perfData.length > 0) {
                    const nav = perfData[0];
                    if (nav.domainLookupStart === 0 && nav.domainLookupEnd === 0) {
                        score += 15;
                        detections.push('no_dns_timing');
                    }
                }
            } catch (e) {}

        } catch (e) {
            score += 30;
            detections.push('advanced_headless_error: ' + e.message);
        }

        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectStealthMode() {
        let score = 0;
        const detections = [];

        try {
            const originalUserAgent = navigator.userAgent;
            const uaLower = originalUserAgent.toLowerCase();

            if (/headless/i.test(uaLower)) {
                score += 35;
                detections.push('headless_in_ua');
            }

            if (/phantom/i.test(uaLower)) {
                score += 40;
                detections.push('phantom_in_ua');
            }

            if (/puppet/i.test(uaLower)) {
                score += 40;
                detections.push('puppet_in_ua');
            }

            const testPermissions = navigator.permissions;
            if (testPermissions) {
                try {
                    const result = await testPermissions.query({ name: 'notifications' });
                    if (result.state === 'prompt') {
                        score += 5;
                        detections.push('notifications_prompt');
                    }
                } catch (e) {}
            }

            const testWebGL = document.createElement('canvas').getContext('webgl');
            if (testWebGL) {
                const debugInfo = testWebGL.getExtension('WEBGL_debug_renderer_info');
                if (debugInfo) {
                    const renderer = testWebGL.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                    const vendor = testWebGL.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);

                    if (renderer && /swiftshader|llvmpipe|software/i.test(renderer)) {
                        score += 30;
                        detections.push('software_renderer');
                    }

                    if (originalUserAgent.includes('Chrome') && !renderer.includes('Chrome')) {
                        score += 25;
                        detections.push('ua_renderer_mismatch');
                    }
                }
            }

            const testCanvas = document.createElement('canvas');
            const testCtx = testCanvas.getContext('2d');
            if (testCtx) {
                testCtx.fillStyle = '#f60ca';
                testCtx.fillRect(1, 1, 62, 20);
                const dataURL = testCtx.canvas.toDataURL();
                if (dataURL.includes('data:image/png')) {
                    const hash = this.hashString(dataURL);
                    this.fingerprintComponents.canvas = hash;
                }
            }

            if (navigator.platform) {
                const platformLower = navigator.platform.toLowerCase();
                if (/win32|macintel|linux/i.test(platformLower)) {
                    score += 5;
                    detections.push('common_platform');
                }
            }

            const testHardware = navigator.hardwareConcurrency;
            if (testHardware) {
                if (testHardware === 1 || testHardware > 64) {
                    score += 15;
                    detections.push('unusual_hardware_concurrency:' + testHardware);
                }
            }

            const testMemory = navigator.deviceMemory;
            if (testMemory) {
                if (testMemory < 1 || testMemory > 64) {
                    score += 10;
                    detections.push('unusual_device_memory:' + testMemory);
                }
            }

            const testConnection = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
            if (testConnection) {
                if (testConnection.saveData === true) {
                    score += 10;
                    detections.push('data_saver_enabled');
                }
            }

        } catch (e) {
            score += 25;
            detections.push('stealth_mode_error: ' + e.message);
        }

        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectWebGLExtended() {
        let score = 0;
        const detections = [];

        try {
            const canvas = document.createElement('canvas');
            canvas.width = 500;
            canvas.height = 150;
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');

            if (!gl) {
                score += 55;
                detections.push('no_webgl');
                return { detected: true, score: Math.min(score, 100), detections };
            }

            const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
            if (debugInfo) {
                const vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
                const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);

                this.fingerprintComponents.webglVendor = vendor;
                this.fingerprintComponents.webglRenderer = renderer;

                detections.push('webgl_vendor:' + (vendor || 'null').substring(0, 40));
                detections.push('webgl_renderer:' + (renderer || 'null').substring(0, 60));

                const extendedSoftwarePatterns = [
                    /swiftshader/i, /llvmpipe/i, /mesa/i, /virtual/i,
                    /google\s*inc/i, /software/i, /microsoft/i, /apple/i,
                    /vmware/i, /parallels/i, /virtualbox/i, /qxl/i,
                    /render/i, /swiftshader/i, /angle/i, /skia/i,
                    /generic/i, /unknown/i, /default/i
                ];

                let softwareScore = 0;
                for (const pattern of extendedSoftwarePatterns) {
                    if (pattern.test(renderer)) {
                        softwareScore += 20;
                    }
                }

                if (softwareScore > 0) {
                    score += Math.min(softwareScore, 45);
                    detections.push('extended_software_renderer');
                }
            } else {
                score += 30;
                detections.push('no_webgl_debug_info');
            }

            const maxTexSize = gl.getParameter(gl.MAX_TEXTURE_SIZE);
            detections.push('max_texture_size:' + maxTexSize);
            if (maxTexSize <= 2048) {
                score += 20;
                detections.push('limited_texture_size');
            }

            const maxVertAttribs = gl.getParameter(gl.MAX_VERTEX_ATTRIBS);
            detections.push('max_vertex_attribs:' + maxVertAttribs);
            if (maxVertAttribs <= 8) {
                score += 15;
                detections.push('limited_vertex_attribs');
            }

            const aliasedRange = gl.getParameter(gl.ALIASED_LINE_WIDTH_RANGE);
            if (aliasedRange && aliasedRange[1] <= 1) {
                score += 15;
                detections.push('aliased_rendering_only');
            }

            const shaderPrecision = gl.getShaderPrecisionFormat(gl.FRAGMENT_SHADER, gl.HIGH_FLOAT);
            if (shaderPrecision) {
                detections.push('shader_precision:' + shaderPrecision.precision);
                if (shaderPrecision.precision < 16) {
                    score += 25;
                    detections.push('low_shader_precision');
                }
            }

            const ext = gl.getExtension('EXT_texture_filter_anisotropic');
            if (!ext) {
                score += 10;
                detections.push('no_anisotropic_filtering');
            }

            const supportedExts = gl.getSupportedExtensions();
            if (supportedExts) {
                detections.push('extension_count:' + supportedExts.length);
                if (supportedExts.length < 15) {
                    score += 20;
                    detections.push('few_webgl_extensions');
                }

                const criticalExts = [
                    'OES_texture_float', 'WEBGL_debug_renderer_info',
                    'EXT_texture_filter_anisotropic', 'OES_standard_derivatives',
                    'WEBGL_lose_context', 'OES_vertex_array_object'
                ];

                let missingCritical = 0;
                for (const extName of criticalExts) {
                    if (!supportedExts.includes(extName) && extName !== 'WEBGL_debug_renderer_info') {
                        missingCritical++;
                    }
                }

                if (missingCritical >= 3) {
                    score += 15;
                    detections.push('many_missing_critical_exts');
                }
            }

            const vertexShader = gl.createShader(gl.VERTEX_SHADER);
            const fragmentShader = gl.createShader(gl.FRAGMENT_SHADER);
            gl.shaderSource(vertexShader, 'attribute vec2 position; void main() { gl_Position = vec4(position, 0.0, 1.0); }');
            gl.compileShader(vertexShader);
            const vertCompiled = gl.getShaderParameter(vertexShader, gl.COMPILE_STATUS);
            if (!vertCompiled) {
                score += 15;
                detections.push('vertex_shader_compile_failed');
            }

            const testProgram = gl.createProgram();
            gl.attachShader(testProgram, vertexShader);
            gl.linkProgram(testProgram);
            const linked = gl.getProgramParameter(testProgram, gl.LINK_STATUS);
            if (!linked) {
                score += 10;
                detections.push('program_link_failed');
            }

            gl.deleteProgram(testProgram);
            gl.deleteShader(vertexShader);
            gl.deleteShader(fragmentShader);

        } catch (e) {
            score += 45;
            detections.push('webgl_extended_error: ' + e.message);
        }

        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectBrowserProfile() {
        let score = 0;
        const detections = [];

        try {
            const ua = navigator.userAgent || '';
            const platform = navigator.platform || '';
            const vendor = navigator.vendor || '';
            const product = navigator.product || '';

            const browserPatterns = {
                chrome: /Chrome|Chromium/i,
                firefox: /Firefox/i,
                safari: /Safari/i,
                edge: /Edge|Edg/i,
                opera: /Opera|OPR/i,
                ie: /MSIE|Trident/i
            };

            let detectedBrowsers = [];
            for (const [browser, pattern] of Object.entries(browserPatterns)) {
                if (pattern.test(ua)) {
                    detectedBrowsers.push(browser);
                }
            }

            if (detectedBrowsers.length > 1) {
                score += 20;
                detections.push('multiple_browsers:' + detectedBrowsers.join(','));
            } else if (detectedBrowsers.length === 0) {
                score += 15;
                detections.push('unknown_browser');
            }

            if (product === 'Gecko' && !browserPatterns.firefox.test(ua)) {
                score += 25;
                detections.push('gecko_mismatch');
            }

            if (vendor === 'Google, Inc.' && !browserPatterns.chrome.test(ua) && !browserPatterns.edge.test(ua)) {
                score += 20;
                detections.push('vendor_mismatch');
            }

            const uaVersionMatch = ua.match(/(Chrome|Firefox|Safari|Edge|Opera)\/([\d.]+)/i);
            if (uaVersionMatch) {
                const browser = uaVersionMatch[1];
                const version = parseFloat(uaVersionMatch[2]);
                this.fingerprintComponents.browserVersion = version;
                detections.push('browser_version:' + version);

                if (browser === 'Chrome' && version < 70) {
                    score += 10;
                    detections.push('old_chrome_version');
                }
                if (browser === 'Firefox' && version < 65) {
                    score += 10;
                    detections.push('old_firefox_version');
                }
            }

            if (navigator.appVersion) {
                const appVersion = navigator.appVersion;
                if (appVersion.includes('Win') && !ua.includes('Windows')) {
                    score += 15;
                    detections.push('appversion_platform_mismatch');
                }
                if (appVersion.includes('Mac') && !ua.includes('Mac')) {
                    score += 15;
                    detections.push('appversion_mac_mismatch');
                }
            }

            if (navigator.platform) {
                if (platform.includes('Linux') && !ua.includes('Linux')) {
                    score += 15;
                    detections.push('platform_linux_mismatch');
                }
                if (platform.includes('Win') && !ua.includes('Windows')) {
                    score += 15;
                    detections.push('platform_windows_mismatch');
                }
                if (platform.includes('Mac') && !ua.includes('Mac')) {
                    score += 15;
                    detections.push('platform_mac_mismatch');
                }
            }

            const buildID = navigator.buildID;
            if (buildID) {
                detections.push('has_build_id');
            } else {
                score += 10;
                detections.push('no_build_id');
            }

        } catch (e) {
            score += 20;
            detections.push('browser_profile_error: ' + e.message);
        }

        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectNetworkFingerprint() {
        let score = 0;
        const detections = [];

        try {
            const connection = navigator.connection || navigator.mozConnection || navigator.webkitConnection;

            if (connection) {
                const networkInfo = {
                    type: connection.type,
                    effectiveType: connection.effectiveType,
                    downlink: connection.downlink,
                    rtt: connection.rtt,
                    saveData: connection.saveData
                };

                this.fingerprintComponents.networkInfo = networkInfo;

                if (networkInfo.rtt > 300) {
                    score += 15;
                    detections.push('high_round_trip_time:' + networkInfo.rtt);
                }

                if (networkInfo.downlink < 1) {
                    score += 10;
                    detections.push('slow_downlink:' + networkInfo.downlink);
                }

                if (networkInfo.effectiveType === 'slow-2g' || networkInfo.effectiveType === '2g') {
                    score += 20;
                    detections.push('slow_network_type:' + networkInfo.effectiveType);
                }

                if (networkInfo.type === 'cellular') {
                    score += 5;
                    detections.push('cellular_connection');
                }
            } else {
                score += 10;
                detections.push('no_connection_api');
            }

            if (navigator.onLine !== undefined) {
                detections.push('online_status:' + navigator.onLine);
                if (!navigator.onLine) {
                    score += 15;
                    detections.push('browser_offline');
                }
            }

            try {
                const pingStart = performance.now();
                await fetch('/favicon.ico', { method: 'HEAD', cache: 'no-cache' });
                const pingDuration = performance.now() - pingStart;

                this.fingerprintComponents.networkLatency = pingDuration;
                detections.push('network_latency:' + Math.round(pingDuration));

                if (pingDuration > 500) {
                    score += 15;
                    detections.push('slow_network_latency');
                } else if (pingDuration < 10) {
                    score += 10;
                    detections.push('unusually_fast_latency');
                }
            } catch (e) {}

            if (navigator.doNotTrack !== undefined) {
                const dnt = navigator.doNotTrack;
                if (dnt === '1' || dnt === 'yes') {
                    score += 5;
                    detections.push('do_not_track_enabled');
                }
            }

            if (navigator.cookieEnabled !== undefined) {
                if (!navigator.cookieEnabled) {
                    score += 10;
                    detections.push('cookies_disabled');
                }
            }

        } catch (e) {
            score += 15;
            detections.push('network_fingerprint_error: ' + e.message);
        }

        return { detected: score > 20, score: Math.min(score, 100), detections };
    }

    async detectTimingAnomaly() {
        let score = 0;
        const detections = [];

        try {
            const timings = [];
            const iterations = 20;

            for (let i = 0; i < iterations; i++) {
                const start = performance.now();
                let dummy = 0;
                for (let j = 0; j < 10000; j++) {
                    dummy += Math.sqrt(j);
                }
                timings.push(performance.now() - start);
            }

            const avgTime = timings.reduce((a, b) => a + b, 0) / timings.length;
            const variance = timings.reduce((sum, t) => sum + Math.pow(t - avgTime, 2), 0) / timings.length;
            const stdDev = Math.sqrt(variance);

            this.mlFeatures.mathTimings = { avg: avgTime, stdDev };

            detections.push('math_avg:' + avgTime.toFixed(2));
            detections.push('math_stddev:' + stdDev.toFixed(2));

            if (avgTime < 1) {
                score += 25;
                detections.push('math_too_fast');
            } else if (avgTime > 50) {
                score += 15;
                detections.push('math_too_slow');
            }

            if (stdDev / avgTime > 0.5) {
                score += 20;
                detections.push('high_timing_variance');
            }

            const sortTimings = [];
            for (let i = 0; i < iterations / 2; i++) {
                const start = performance.now();
                const arr = new Array(1000);
                for (let j = 0; j < arr.length; j++) {
                    arr[j] = Math.random();
                }
                arr.sort();
                sortTimings.push(performance.now() - start);
            }

            const avgSortTime = sortTimings.reduce((a, b) => a + b, 0) / sortTimings.length;
            this.mlFeatures.sortTimings = avgSortTime;
            detections.push('sort_avg:' + avgSortTime.toFixed(2));

            if (avgSortTime < 0.5) {
                score += 20;
                detections.push('sort_too_fast');
            }

            const loopTimings = [];
            for (let i = 0; i < iterations / 2; i++) {
                const start = performance.now();
                let sum = 0;
                for (let j = 0; j < 50000; j++) {
                    sum += j;
                }
                loopTimings.push(performance.now() - start);
            }

            const avgLoopTime = loopTimings.reduce((a, b) => a + b, 0) / loopTimings.length;
            this.mlFeatures.loopTimings = avgLoopTime;
            detections.push('loop_avg:' + avgLoopTime.toFixed(2));

            if (avgLoopTime < 1) {
                score += 20;
                detections.push('loop_too_fast');
            }

            if (this.mlFeatures.timing_variance > 0.8) {
                score += 15;
                detections.push('overall_timing_anomaly');
            }

        } catch (e) {
            score += 20;
            detections.push('timing_anomaly_error: ' + e.message);
        }

        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectBehaviorPattern() {
        let score = 0;
        const detections = [];

        try {
            if (this.mlFeatures && this.mlFeatures.timing_variance !== undefined) {
                if (this.mlFeatures.timing_variance > 0.5) {
                    score += 20;
                    detections.push('irregular_timing_patterns');
                }
            }

            if (this.mlFeatures && this.mlFeatures.entropy_score !== undefined) {
                if (this.mlFeatures.entropy_score < 0.15) {
                    score += 25;
                    detections.push('low_behavioral_entropy');
                }
            }

            if (this.mlFeatures && this.mlFeatures.consistency_score !== undefined) {
                if (this.mlFeatures.consistency_score < 0.6) {
                    score += 20;
                    detections.push('inconsistent_detection_patterns');
                }
            }

            const canvasChecks = ['detectCanvasEnhanced', 'detectCanvasStable'];
            const canvasResults = canvasChecks.map(m => this.results[m]).filter(Boolean);
            if (canvasResults.length > 0) {
                const avgScore = canvasResults.reduce((sum, r) => sum + r.score, 0) / canvasResults.length;
                if (avgScore > 30) {
                    score += 15;
                    detections.push('suspicious_canvas_pattern');
                }
            }

            const webglChecks = ['detectWebGLEnhanced', 'detectWebGLExtended'];
            const webglResults = webglChecks.map(m => this.results[m]).filter(Boolean);
            if (webglResults.length > 0) {
                const avgScore = webglResults.reduce((sum, r) => sum + r.score, 0) / webglResults.length;
                if (avgScore > 25) {
                    score += 15;
                    detections.push('suspicious_webgl_pattern');
                }
            }

            const automationChecks = ['detectHeadless', 'detectAdvancedHeadless', 'detectStealthMode'];
            const automationResults = automationChecks.map(m => this.results[m]).filter(Boolean);
            if (automationResults.length >= 2) {
                const detected = automationResults.filter(r => r.detected).length;
                if (detected >= 2) {
                    score += 25;
                    detections.push('multiple_automation_indicators');
                }
            }

            if (this.mlFeatures && this.mlFeatures.suspicious_patterns) {
                const highRiskPatterns = ['headless', 'webdriver', 'puppeteer', 'playwright', 'selenium'];
                const found = highRiskPatterns.filter(p =>
                    this.mlFeatures.suspicious_patterns.some(d => d.toLowerCase().includes(p))
                );
                if (found.length >= 3) {
                    score += 30;
                    detections.push('multiple_known_automation_patterns');
                }
            }

            this.fingerprintComponents.behaviorScore = score;

        } catch (e) {
            score += 15;
            detections.push('behavior_pattern_error: ' + e.message);
        }

        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectCanvasEnhanced() {
        let score = 0;
        const detections = [];
        try {
            const canvas = document.createElement('canvas');
            canvas.width = 450;
            canvas.height = 140;
            const ctx = canvas.getContext('2d');
            if (!ctx) {
                score += 55;
                detections.push('no_canvas_context');
                return { detected: true, score: Math.min(score, 100), detections };
            }

            ctx.textBaseline = 'alphabetic';
            ctx.fillStyle = '#f60';
            ctx.fillRect(125, 1, 62, 20);
            ctx.fillStyle = '#069';
            ctx.font = '11pt Arial';
            ctx.fillText('Cwm fjordbank glyphs vext quiz, 😀', 2, 15);
            ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
            ctx.font = '18pt Arial';
            ctx.fillText('Cwm fjordbank glyphs vext quiz, 😀', 4, 45);

            ctx.globalCompositeOperation = 'multiply';
            ctx.fillStyle = 'rgb(255,0,255)';
            ctx.beginPath();
            ctx.arc(50, 50, 50, 0, Math.PI * 2, true);
            ctx.closePath();
            ctx.fill();
            ctx.fillStyle = 'rgb(0,255,255)';
            ctx.beginPath();
            ctx.arc(100, 50, 50, 0, Math.PI * 2 / 3, true);
            ctx.closePath();
            ctx.fill();
            ctx.fillStyle = 'rgb(255,255,0)';
            ctx.beginPath();
            ctx.arc(75, 50, 50, 0, Math.PI * 2 / 3, false);
            ctx.closePath();
            ctx.fill();

            ctx.fillStyle = '#fff';
            ctx.font = 'bold 16pt Arial';
            ctx.fillText('abcdefghijklmnopqrstuvwxyz 0123456789', 4, 70);

            ctx.fillStyle = 'rgba(100, 100, 100, 0.5)';
            ctx.beginPath();
            ctx.moveTo(10, 80);
            ctx.lineTo(100, 20);
            ctx.lineTo(200, 80);
            ctx.lineTo(300, 20);
            ctx.lineTo(390, 80);
            ctx.stroke();

            const gradients = ctx.createLinearGradient(0, 0, 200, 0);
            gradients.addColorStop(0, 'red');
            gradients.addColorStop(1, 'blue');
            ctx.fillStyle = gradients;
            ctx.fillRect(10, 90, 380, 25);

            const radialGrad = ctx.createRadialGradient(350, 60, 5, 350, 60, 30);
            radialGrad.addColorStop(0, 'rgba(255, 255, 0, 0.8)');
            radialGrad.addColorStop(1, 'rgba(255, 0, 0, 0)');
            ctx.fillStyle = radialGrad;
            ctx.fillRect(320, 30, 60, 60);

            const dataURL = canvas.toDataURL();
            const dataURL2 = canvas.toDataURL();
            if (dataURL !== dataURL2) {
                score += 35;
                detections.push('canvas_unstable');
            }

            const imageData = ctx.getImageData(0, 0, 60, 60);
            const pixelSum = Array.from(imageData.data.slice(0, 240)).reduce((a, b) => a + b, 0);
            if (pixelSum === 0) {
                score += 30;
                detections.push('canvas_empty_readback');
            }

            const uniquePixels = new Set();
            for (let i = 0; i < imageData.data.length; i += 4) {
                const hex = [imageData.data[i], imageData.data[i+1], imageData.data[i+2]].join(',');
                uniquePixels.add(hex);
            }
            const uniqueRatio = uniquePixels.size / (imageData.width * imageData.height);
            if (uniqueRatio < 0.3) {
                score += 15;
                detections.push('low_pixel_diversity');
            }

            const hash = this.hashString(dataURL);
            this.fingerprintComponents.canvas = hash;
            detections.push('canvas_hash:' + hash.substring(0, 20));

        } catch (e) {
            score += 45;
            detections.push('canvas_error: ' + e.message);
        }
        return { detected: score > 40, score: Math.min(score, 100), detections };
    }

    async detectCanvasStable() {
        let score = 0;
        const detections = [];
        const results = [];
        try {
            for (let i = 0; i < 5; i++) {
                const canvas = document.createElement('canvas');
                canvas.width = 250;
                canvas.height = 60;
                const ctx = canvas.getContext('2d');
                if (ctx) {
                    ctx.fillStyle = '#f60';
                    ctx.fillRect(10, 10, 60, 40);
                    ctx.fillStyle = '#069';
                    ctx.font = '16px Arial';
                    ctx.fillText('Fingerprint Test ' + i, 25, 35);
                    results.push(canvas.toDataURL());
                }
            }

            if (results.length >= 2) {
                const matches = [];
                for (let i = 1; i < results.length; i++) {
                    if (results[0] === results[i]) {
                        matches.push(i);
                    }
                }
                if (matches.length === results.length - 1) {
                    score += 30;
                    detections.push('canvas_identical_across_renders');
                } else if (matches.length > 0) {
                    score += 15;
                    detections.push('canvas_some_identical');
                }
            }

            const emptyCanvas = document.createElement('canvas');
            emptyCanvas.width = 120;
            emptyCanvas.height = 120;
            const emptyCtx = emptyCanvas.getContext('2d');
            if (emptyCtx) {
                const emptyData = emptyCtx.getImageData(0, 0, 12, 12);
                const allZero = Array.from(emptyData.data).every(v => v === 0);
                if (allZero) {
                    score += 25;
                    detections.push('empty_canvas_reads_zero');
                }
            }

            const hiddenCanvas = document.createElement('canvas');
            hiddenCanvas.style.display = 'none';
            hiddenCanvas.width = 250;
            hiddenCanvas.height = 60;
            document.body.appendChild(hiddenCanvas);
            const hiddenCtx = hiddenCanvas.getContext('2d');
            if (hiddenCtx) {
                hiddenCtx.fillStyle = '#000';
                hiddenCtx.fillRect(0, 0, 250, 60);
                const hiddenData = hiddenCtx.getImageData(0, 0, 12, 12);
                const allBlack = Array.from(hiddenData.data.slice(0, 48)).every(v => v === 0);
                if (allBlack) {
                    score += 15;
                    detections.push('hidden_canvas_unreadable');
                }
            }
            document.body.removeChild(hiddenCanvas);

        } catch (e) {
            score += 30;
            detections.push('canvas_stable_error: ' + e.message);
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectCanvasEntropy() {
        let score = 0;
        const detections = [];
        try {
            const canvas = document.createElement('canvas');
            canvas.width = 350;
            canvas.height = 120;
            const ctx = canvas.getContext('2d');
            if (!ctx) {
                return { detected: false, score: 0, detections: ['no_context'] };
            }

            const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()';
            const text = Array.from({ length: 60 }, () => chars[Math.floor(Math.random() * chars.length)]).join('');
            ctx.font = '14px Arial';
            ctx.fillText(text, 8, 25);

            ctx.font = 'bold 18px Arial';
            ctx.fillText('Testing Canvas Entropy', 8, 55);

            ctx.font = 'italic 16px Georgia';
            ctx.fillText('FingerPrinting Detection', 8, 85);

            const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
            const data = imageData.data;

            let uniqueValues = new Set();
            for (let i = 0; i < data.length; i += 4) {
                const hex = data[i].toString(16) + data[i+1].toString(16) + data[i+2].toString(16);
                uniqueValues.add(hex);
            }

            const entropyRatio = uniqueValues.size / (data.length / 4);
            if (entropyRatio < 0.08) {
                score += 35;
                detections.push('very_low_entropy:' + entropyRatio.toFixed(4));
            } else if (entropyRatio < 0.18) {
                score += 18;
                detections.push('low_entropy:' + entropyRatio.toFixed(4));
            }

            let zeroCount = 0;
            for (let i = 0; i < data.length; i++) {
                if (data[i] === 0) zeroCount++;
            }
            const zeroRatio = zeroCount / data.length;
            if (zeroRatio > 0.85) {
                score += 30;
                detections.push('high_zero_ratio:' + zeroRatio.toFixed(4));
            }

            this.fingerprintComponents.canvasEntropy = entropyRatio;
            detections.push('entropy:' + entropyRatio.toFixed(4));

        } catch (e) {
            score += 25;
            detections.push('entropy_error: ' + e.message);
        }
        return { detected: score > 28, score: Math.min(score, 100), detections };
    }

    async detectWebGLEnhanced() {
        let score = 0;
        const detections = [];
        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
            if (!gl) {
                score += 55;
                detections.push('no_webgl');
                return { detected: true, score: Math.min(score, 100), detections };
            }

            const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
            if (debugInfo) {
                const vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
                const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);

                if (!vendor || !renderer) {
                    score += 25;
                    detections.push('webgl_no_vendor_renderer');
                } else {
                    this.fingerprintComponents.webglVendor = vendor;
                    this.fingerprintComponents.webglRenderer = renderer;
                    detections.push('webgl_vendor:' + vendor.substring(0, 35));
                    detections.push('webgl_renderer:' + renderer.substring(0, 55));
                }

                const softwarePatterns = [
                    /swiftshader/i, /llvmpipe/i, /mesa/i, /virtual/i,
                    /google\s*inc/i, /software/i, /microsoft/i,
                    /vmware/i, /parallels/i, /virtualbox/i, /qxl/i,
                    /angle/i, /skia/i, /render/i
                ];

                for (const pattern of softwarePatterns) {
                    if (pattern.test(renderer)) {
                        score += 40;
                        detections.push('software_renderer_detected');
                        break;
                    }
                }

                const unknownPatterns = [/unknown/i, /generic/i, /default/i, /basic/i];
                for (const pattern of unknownPatterns) {
                    if (pattern.test(renderer)) {
                        score += 25;
                        detections.push('generic_renderer');
                        break;
                    }
                }
            } else {
                score += 30;
                detections.push('no_webgl_debug_info');
            }

            const maxTexSize = gl.getParameter(gl.MAX_TEXTURE_SIZE);
            detections.push('max_texture_size:' + maxTexSize);
            if (maxTexSize <= 1024) {
                score += 25;
                detections.push('small_max_texture');
            }

            const maxVertAttribs = gl.getParameter(gl.MAX_VERTEX_ATTRIBS);
            detections.push('max_vertex_attribs:' + maxVertAttribs);
            if (maxVertAttribs <= 8) {
                score += 18;
                detections.push('few_vertex_attribs');
            }

            const aliasedRange = gl.getParameter(gl.ALIASED_LINE_WIDTH_RANGE);
            if (aliasedRange && aliasedRange[1] <= 1) {
                score += 18;
                detections.push('aliased_only');
            }

            const shaderPrecision = gl.getShaderPrecisionFormat(gl.FRAGMENT_SHADER, gl.HIGH_FLOAT);
            if (shaderPrecision) {
                detections.push('shader_precision:' + shaderPrecision.precision);
                if (shaderPrecision.precision < 16) {
                    score += 25;
                    detections.push('low_shader_precision');
                }
            }

            const ext = gl.getExtension('EXT_texture_filter_anisotropic');
            if (!ext) {
                score += 12;
                detections.push('no_anisotropic_filtering');
            }

            const supportedExts = gl.getSupportedExtensions();
            if (supportedExts) {
                detections.push('extension_count:' + supportedExts.length);
                if (supportedExts.length < 18) {
                    score += 18;
                    detections.push('few_webgl_extensions');
                }
            }

        } catch (e) {
            score += 45;
            detections.push('webgl_error: ' + e.message);
        }
        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectWebGL2Enhanced() {
        let score = 0;
        const detections = [];
        try {
            const canvas = document.createElement('canvas');
            const gl2 = canvas.getContext('webgl2');
            if (!gl2) {
                return { detected: false, score: 0, detections: ['no_webgl2'] };
            }

            const debugInfo = gl2.getExtension('WEBGL_debug_renderer_info');
            if (debugInfo) {
                const renderer = gl2.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                const softwarePatterns = [/swiftshader/i, /llvmpipe/i, /mesa/i, /virtual/i];

                for (const pattern of softwarePatterns) {
                    if (pattern.test(renderer)) {
                        score += 35;
                        detections.push('webgl2_software_renderer');
                        break;
                    }
                }
            }

            const maxTexSize = gl2.getParameter(gl2.MAX_TEXTURE_SIZE);
            detections.push('webgl2_max_texture:' + maxTexSize);
            if (maxTexSize <= 1024) {
                score += 18;
                detections.push('webgl2_small_texture');
            }

            const supportedExts = gl2.getSupportedExtensions();
            if (supportedExts) {
                detections.push('webgl2_ext_count:' + supportedExts.length);
                if (supportedExts.length < 12) {
                    score += 12;
                    detections.push('few_webgl2_extensions');
                }
            }

            const transformFeedback = gl2.getParameter(gl2.MAX_TRANSFORM_FEEDBACK_SEPARATE_ATTRIBS);
            detections.push('transform_feedback_attrs:' + transformFeedback);

        } catch (e) {
            score += 28;
            detections.push('webgl2_error: ' + e.message);
        }
        return { detected: score > 28, score: Math.min(score, 100), detections };
    }

    async detectAudioEnhanced() {
        let score = 0;
        const detections = [];
        try {
            const AudioContext = window.OfflineAudioContext || window.webkitOfflineAudioContext;
            if (!AudioContext) {
                score += 40;
                detections.push('no_audio_context');
                return { detected: true, score: Math.min(score, 100), detections };
            }

            const ctx = new AudioContext(1, 44100, 44100);
            const osc = ctx.createOscillator();
            osc.type = 'triangle';
            osc.frequency.setValueAtTime(10000, ctx.currentTime);

            const compressor = ctx.createDynamicsCompressor();
            compressor.threshold.setValueAtTime(-50, ctx.currentTime);
            compressor.knee.setValueAtTime(40, ctx.currentTime);
            compressor.ratio.setValueAtTime(12, ctx.currentTime);
            compressor.attack.setValueAtTime(0, ctx.currentTime);
            compressor.release.setValueAtTime(0.25, ctx.currentTime);

            const gain = ctx.createGain();
            gain.gain.setValueAtTime(0.5, ctx.currentTime);

            osc.connect(compressor);
            compressor.connect(gain);
            gain.connect(ctx.destination);
            osc.start(0);

            const startTime = performance.now();
            const buffer = await ctx.startRendering();
            const renderTime = performance.now() - startTime;

            detections.push('audio_render_time:' + renderTime.toFixed(2));
            if (renderTime < 5) {
                score += 28;
                detections.push('audio_render_too_fast');
            }

            const channelData = buffer.getChannelData(0);
            let sumAbs = 0;
            let nonZeroCount = 0;
            for (let i = 4500; i < 5000; i++) {
                const abs = Math.abs(channelData[i]);
                sumAbs += abs;
                if (abs > 0) nonZeroCount++;
            }

            detections.push('audio_non_zero_ratio:' + (nonZeroCount / 500).toFixed(4));
            if (nonZeroCount < 100) {
                score += 35;
                detections.push('audio_mostly_silent');
            }

            const multipleBuffers = [];
            for (let i = 0; i < 4; i++) {
                const tempCtx = new AudioContext(1, 44100, 44100);
                const tempOsc = tempCtx.createOscillator();
                tempOsc.frequency.setValueAtTime(1000, tempCtx.currentTime);
                tempOsc.connect(tempCtx.destination);
                tempOsc.start(0);
                const tempBuffer = await tempCtx.startRendering();
                multipleBuffers.push(tempBuffer.getChannelData(0).slice(4500, 4550).join(','));
            }

            if (multipleBuffers[0] === multipleBuffers[1] && multipleBuffers[1] === multipleBuffers[2]) {
                score += 30;
                detections.push('audio_identical_across_renders');
            }

            const hash = this.hashString(multipleBuffers[0]);
            this.fingerprintComponents.audio = hash;
            detections.push('audio_hash:' + hash.substring(0, 16));

        } catch (e) {
            score += 40;
            detections.push('audio_error: ' + e.message);
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectFontsEnhanced() {
        let score = 0;
        const detections = [];
        const baseFonts = ['monospace', 'sans-serif', 'serif'];
        const testFonts = [
            'Arial', 'Helvetica', 'Times New Roman', 'Courier New',
            'Verdana', 'Georgia', 'Palatino', 'Garamond',
            'Impact', 'Comic Sans MS', 'Trebuchet MS', 'Lucida Console',
            'Tahoma', 'Segoe UI', 'Roboto', 'Open Sans',
            'Lato', 'Montserrat', 'Source Sans Pro', 'Raleway',
            'Ubuntu', 'Noto Sans', 'Droid Sans', 'Fira Sans',
            'Merriweather', 'Playfair Display', 'PT Sans', 'Nunito',
            'Quicksand', 'Work Sans', 'Oswald', 'Roboto Condensed',
            'Noto Serif', 'Lora', 'IBM Plex Sans', 'JetBrains Mono',
            'SF Pro Display', 'SF Pro Text', 'Calibri', 'Candara',
            'Corbel', 'Cambria', 'Bookman', 'Futura', 'Optima'
        ];

        try {
            const el = document.createElement('div');
            el.style.cssText = 'position:absolute;left:-9999px;font-size:72px;visibility:hidden;white-space:nowrap';
            el.textContent = 'mmmmmmmmmmlli';
            document.body.appendChild(el);

            const baseWidths = {};
            for (const base of baseFonts) {
                el.style.fontFamily = base;
                baseWidths[base] = el.offsetWidth;
            }

            const detectedFonts = [];
            for (const font of testFonts) {
                for (const base of baseFonts) {
                    el.style.fontFamily = `"${font}", ${base}`;
                    if (el.offsetWidth !== baseWidths[base]) {
                        detectedFonts.push(font);
                        break;
                    }
                }
            }

            document.body.removeChild(el);

            detections.push('detected_font_count:' + detectedFonts.length);
            if (detectedFonts.length < 3) {
                score += 40;
                detections.push('very_few_fonts');
            } else if (detectedFonts.length < 8) {
                score += 25;
                detections.push('few_fonts');
            }

            this.fingerprintComponents.fonts = detectedFonts;

        } catch (e) {
            score += 35;
            detections.push('font_detection_error: ' + e.message);
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectHeadless() {
        let score = 0;
        const detections = [];
        if (navigator.webdriver === true || navigator.webdriver === false) {
        } else {
            score += 35;
            detections.push('webdriver_undefined');
        }
        if (navigator.plugins && navigator.plugins.length === 0) {
            score += 25;
            detections.push('no_plugins');
        }
        if (navigator.languages && navigator.languages.length === 0) {
            score += 25;
            detections.push('no_languages');
        }
        if (window.chrome && window.chrome.runtime === undefined) {
            score += 30;
            detections.push('chrome_no_runtime');
        }
        const mimeTypes = navigator.mimeTypes;
        if (mimeTypes && mimeTypes.length === 0) {
            score += 30;
            detections.push('no_mimetypes');
        }
        try {
            const ua = navigator.userAgent || '';
            if (/headless|phantom/i.test(ua)) {
                score += 45;
                detections.push('headless_ua');
            }
        } catch (e) {}
        try {
            if (window.outerHeight === 0 && window.outerWidth === 0) {
                score += 35;
                detections.push('zero_window_size');
            }
        } catch (e) {}
        return { detected: score > 40, score: Math.min(score, 100), detections };
    }

    async detectWebDriver() {
        let score = 0;
        const detections = [];
        const wdProps = [
            'webdriver', '__webdriver_evaluate', '__selenium_evaluate',
            '__webdriver_script_fn', '__driver_evaluate', '__fxdriver_evaluate',
            '__webdriver_unwrapped', '__lastWatirAlert', '__$webdriverAsyncExecutor',
            'callSelenium', '__selenium', 'Selenium'
        ];
        for (const prop of wdProps) {
            if (window[prop] !== undefined) {
                score += 25;
                detections.push(prop);
            }
        }
        try {
            if (navigator.webdriver === true) {
                score += 40;
                detections.push('navigator_webdriver_true');
            }
        } catch (e) {}
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectPuppeteer() {
        let score = 0;
        const detections = [];
        try {
            if (navigator.webdriver === true) {
                score += 35;
                detections.push('webdriver_true');
            }
        } catch (e) {}
        try {
            if (document.$cdc_asdjflasutopfhvcZLmcfl_) {
                score += 50;
                detections.push('puppeteer_cdc_marker');
            }
        } catch (e) {}
        try {
            if (document.$chrome_asyncScriptInfo) {
                score += 40;
                detections.push('chrome_async_script');
            }
        } catch (e) {}
        try {
            const userAgent = navigator.userAgent || '';
            if (/headless/i.test(userAgent)) {
                score += 40;
                detections.push('headless_ua');
            }
            if (/puppet/i.test(userAgent)) {
                score += 55;
                detections.push('puppeteer_ua');
            }
        } catch (e) {}
        return { detected: score > 40, score: Math.min(score, 100), detections };
    }

    async detectPlaywright() {
        let score = 0;
        const detections = [];
        try {
            if (window.__playwright__ !== undefined ||
                window.__pw_tags !== undefined ||
                window.__pw_resume__ !== undefined) {
                score += 60;
                detections.push('playwright_global');
            }
        } catch (e) {}
        try {
            const ua = navigator.userAgent || '';
            if (/playwright/i.test(ua)) {
                score += 60;
                detections.push('playwright_ua');
            }
        } catch (e) {}
        return { detected: score > 45, score: Math.min(score, 100), detections };
    }

    async detectSelenium() {
        let score = 0;
        const detections = [];
        const selProps = [
            'selenium', '_selenium', 'callSelenium', '__selenium',
            'document__selenium', 'Selenium', '__webdriver_script_fn',
            'Selenium.prototype'
        ];
        for (const prop of selProps) {
            if (window[prop] !== undefined || document[prop] !== undefined) {
                score += 30;
                detections.push(prop);
            }
        }
        try {
            if (document.documentElement.getAttribute('webdriver') !== null) {
                score += 40;
                detections.push('webdriver_attr');
            }
        } catch (e) {}
        try {
            const ua = navigator.userAgent || '';
            if (/selenium/i.test(ua)) {
                score += 55;
                detections.push('selenium_ua');
            }
        } catch (e) {}
        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectChromeRuntime() {
        let score = 0;
        const detections = [];
        try {
            if (window.chrome) {
                if (window.chrome.runtime === undefined) {
                    score += 30;
                    detections.push('chrome_runtime_missing');
                }
                if (window.chrome.loadTimes === undefined) {
                    score += 18;
                    detections.push('chrome_loadtimes_missing');
                }
                if (window.chrome.csi === undefined) {
                    score += 18;
                    detections.push('chrome_csi_missing');
                }
                if (window.chrome.app === undefined) {
                    score += 18;
                    detections.push('chrome_app_missing');
                }
            } else {
                if (!/Edge|Edg|Firefox|Safari/i.test(navigator.userAgent || '')) {
                    score += 40;
                    detections.push('no_chrome_no_alt');
                }
            }
        } catch (e) {
            score += 35;
            detections.push('chrome_check_error');
        }
        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectAutomationFrameworks() {
        let score = 0;
        const detections = [];
        const knownMarkers = {
            puppeteer: ['$cdc_asdjflasutopfhvcZLmcfl_', '$chrome_asyncScriptInfo'],
            playwright: ['__playwright__', '__pw_tags', '__pw_resume__'],
            selenium: ['__selenium_evaluate', '__webdriver_evaluate'],
            automation: ['navigator.webdriver === true']
        };

        for (const [framework, markers] of Object.entries(knownMarkers)) {
            for (const marker of markers) {
                if (marker.includes('navigator')) {
                    if (eval(marker)) {
                        score += 55;
                        detections.push(`${framework}_detected:${marker}`);
                    }
                } else if (window[marker] !== undefined || document[marker] !== undefined) {
                    score += 50;
                    detections.push(`${framework}_detected:${marker}`);
                }
            }
        }

        return { detected: score > 40, score: Math.min(score, 100), detections };
    }

    async detectVirtualization() {
        let score = 0;
        const detections = [];
        try {
            const vmIndicators = [
                { name: 'vmware', check: () => /vmware/i.test(navigator.userAgent || '') },
                { name: 'virtualbox', check: () => /virtualbox/i.test(navigator.userAgent || '') },
                { name: 'parallels', check: () => /parallels/i.test(navigator.userAgent || '') },
                { name: 'hyperv', check: () => /hyperv/i.test(navigator.userAgent || '') },
                { name: 'qemu', check: () => /qemu/i.test(navigator.userAgent || '') }
            ];

            for (const indicator of vmIndicators) {
                if (indicator.check()) {
                    score += 45;
                    detections.push(`${indicator.name}_detected`);
                }
            }

            const cpu = navigator.hardwareConcurrency;
            if (cpu && cpu < 2) {
                score += 25;
                detections.push('low_core_count');
            }

            const mem = navigator.deviceMemory;
            if (mem && mem < 1) {
                score += 30;
                detections.push('low_device_memory');
            }

            try {
                const canvas = document.createElement('canvas');
                const gl = canvas.getContext('webgl');
                if (gl) {
                    const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                    if (debugInfo) {
                        const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                        if (/vmware|virtualbox|parallels|qemu|kvm|hyperv/i.test(renderer)) {
                            score += 50;
                            detections.push('vm_webgl_renderer');
                        }
                    }
                }
            } catch (e) {}

        } catch (e) {
            score += 25;
            detections.push('virtualization_error: ' + e.message);
        }
        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectSandbox() {
        let score = 0;
        const detections = [];
        try {
            try {
                new Worker('data:text/javascript;base64,');
                detections.push('worker_available');
            } catch (e) {
                score += 25;
                detections.push('worker_blocked');
            }

            try {
                if (typeof SharedArrayBuffer !== 'undefined') {
                    detections.push('shared_array_buffer_available');
                } else {
                    score += 12;
                    detections.push('shared_array_buffer_unavailable');
                }
            } catch (e) {
                score += 12;
                detections.push('shared_array_buffer_error');
            }

        } catch (e) {
            score += 25;
            detections.push('sandbox_error: ' + e.message);
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectWebRTCEnhanced() {
        let score = 0;
        const detections = [];
        try {
            const RTCPeerConnection = window.RTCPeerConnection ||
                window.webkitRTCPeerConnection ||
                window.mozRTCPeerConnection;

            if (!RTCPeerConnection) {
                score += 25;
                detections.push('no_webrtc');
                return { detected: true, score: Math.min(score, 100), detections };
            }

            const ips = new Set();
            const pc = new RTCPeerConnection({
                iceServers: [
                    { urls: 'stun:stun.l.google.com:19302' },
                    { urls: 'stun:stun1.l.google.com:19302' },
                    { urls: 'stun:stun2.l.google.com:19302' }
                ]
            });

            pc.createDataChannel('');
            const offer = await pc.createOffer();
            await pc.setLocalDescription(offer);

            await new Promise(resolve => setTimeout(resolve, 1200));

            const sdp = pc.localDescription.sdp;
            const lines = sdp.split('\n');
            for (const line of lines) {
                if (line.indexOf('candidate') > -1) {
                    const parts = line.split(' ');
                    if (parts[4] && parts[4] !== '0.0.0.0') {
                        ips.add(parts[4]);
                        if (parts[7] !== 'host') {
                            detections.push('relay_candidate:' + parts[4]);
                        }
                    }
                }
            }

            pc.close();

            detections.push('ip_count:' + ips.size);

            if (ips.size === 0) {
                score += 18;
                detections.push('no_webrtc_ips');
            }

            const privateIPs = Array.from(ips).filter(ip =>
                ip.startsWith('10.') ||
                ip.startsWith('172.16.') || ip.startsWith('172.17.') ||
                ip.startsWith('172.18.') || ip.startsWith('172.19.') ||
                ip.startsWith('172.20.') || ip.startsWith('172.21.') ||
                ip.startsWith('172.22.') || ip.startsWith('172.23.') ||
                ip.startsWith('172.24.') || ip.startsWith('172.25.') ||
                ip.startsWith('172.26.') || ip.startsWith('172.27.') ||
                ip.startsWith('172.28.') || ip.startsWith('172.29.') ||
                ip.startsWith('172.30.') || ip.startsWith('172.31.') ||
                ip.startsWith('192.168.') ||
                ip.startsWith('127.') ||
                ip.startsWith('169.254.')
            );

            const publicIPs = Array.from(ips).filter(ip => !privateIPs.includes(ip));

            if (publicIPs.length > 0) {
                this.fingerprintComponents.publicIP = publicIPs[0];
                detections.push('public_ip_found');
            }

            if (privateIPs.length > 0 && publicIPs.length > 0) {
                score += 30;
                detections.push('vpn_ip_mismatch');
            }

            this.fingerprintComponents.webrtcIPs = Array.from(ips);

        } catch (e) {
            score += 25;
            detections.push('webrtc_error: ' + e.message);
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectWebRTCLeak() {
        let score = 0;
        const detections = [];
        try {
            const RTCPeerConnection = window.RTCPeerConnection ||
                window.webkitRTCPeerConnection ||
                window.mozRTCPeerConnection;

            if (!RTCPeerConnection) {
                return { detected: false, score: 0, detections: ['no_webrtc'] };
            }

            const ipsBefore = this.fingerprintComponents.webrtcIPs || [];

            const pc = new RTCPeerConnection({
                iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
            });

            pc.createDataChannel('leaktest');
            const offer = await pc.createOffer();
            await pc.setLocalDescription(offer);

            await new Promise(resolve => setTimeout(resolve, 1800));

            const sdp = pc.localDescription.sdp;
            const lines = sdp.split('\n');
            const newIPs = [];
            for (const line of lines) {
                if (line.indexOf('candidate') > -1) {
                    const parts = line.split(' ');
                    if (parts[4] && parts[4] !== '0.0.0.0' && !ipsBefore.includes(parts[4])) {
                        newIPs.push(parts[4]);
                    }
                }
            }

            pc.close();

            if (newIPs.length > 0) {
                score += 35;
                detections.push('ip_leak_detected');
                detections.push('leaked_ips:' + newIPs.join(','));
            }

        } catch (e) {
            score += 18;
            detections.push('webrtc_leak_error: ' + e.message);
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectProxyVPN() {
        let score = 0;
        const detections = [];
        try {
            const xff = this.getHeader('X-Forwarded-For');
            const xri = this.getHeader('X-Real-IP');
            const via = this.getHeader('Via');
            const proxyChain = this.getHeader('X-ProxyChain');

            if (xff) {
                const xffIPs = xff.split(',').map(ip => ip.trim());
                detections.push('xff_count:' + xffIPs.length);
                if (xffIPs.length > 2) {
                    score += 30;
                    detections.push('multi_hop_proxy');
                }
            }

            if (via) {
                score += 18;
                detections.push('via_header_present');
                const proxyKeywords = ['proxy', 'vpn', 'squid', 'nginx', 'apache', 'varnish'];
                for (const keyword of proxyKeywords) {
                    if (via.toLowerCase().includes(keyword)) {
                        score += 18;
                        detections.push('known_proxy_via:' + keyword);
                        break;
                    }
                }
            }

            if (xff && xri && xff !== xri) {
                score += 25;
                detections.push('ip_mismatch');
            }

            const publicIP = this.fingerprintComponents.publicIP;
            if (publicIP) {
                const isDatacenter = this.checkDatacenterIP(publicIP);
                if (isDatacenter) {
                    score += 35;
                    detections.push('datacenter_ip');
                }
            }

        } catch (e) {
            score += 18;
            detections.push('proxy_vpn_error: ' + e.message);
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectVPNIndicators() {
        let score = 0;
        const detections = [];
        try {
            const conn = navigator.connection || navigator.mozConnection || navigator.webkitConnection;

            if (conn) {
                if (conn.type === 'vpn') {
                    score += 45;
                    detections.push('vpn_type_detected');
                }

                if (conn.type === 'ethernet' || conn.type === 'wifi') {
                    if (conn.effectiveType === 'slow-2g' || conn.effectiveType === '2g') {
                        score += 12;
                        detections.push('slow_connection');
                    }
                }

                if (conn.saveData === true) {
                    score += 12;
                    detections.push('data_saver_enabled');
                }

                if (conn.rtt !== null && conn.rtt > 500) {
                    score += 18;
                    detections.push('high_latency:' + conn.rtt);
                }
            }

            if (navigator.onLine === false) {
                score += 12;
                detections.push('offline');
            }

        } catch (e) {
            score += 18;
            detections.push('vpn_indicators_error: ' + e.message);
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectTorExitNode() {
        let score = 0;
        const detections = [];
        try {
            const publicIP = this.fingerprintComponents.publicIP;

            if (publicIP) {
                const torExitNodes = [
                    '128.31.0.34', '128.93.34.5', '131.188.40.189',
                    '154.35.22.1', '171.25.193.77', '176.10.99.200'
                ];

                const isKnownTorIP = torExitNodes.some(node =>
                    publicIP.startsWith(node.substring(0, 8))
                );

                if (isKnownTorIP) {
                    score += 55;
                    detections.push('known_tor_exit_node');
                }
            }

            const userAgent = navigator.userAgent || '';
            if (/tor|onion/i.test(userAgent)) {
                score += 45;
                detections.push('tor_in_user_agent');
            }

            if (this.fingerprintComponents.webrtcIPs) {
                const hasTorIndicator = this.fingerprintComponents.webrtcIPs.some(ip =>
                    ip.endsWith('.onion') || ip.match(/\.exit$/i)
                );
                if (hasTorIndicator) {
                    score += 40;
                    detections.push('tor_onion_detected');
                }
            }

        } catch (e) {
            score += 18;
            detections.push('tor_check_error: ' + e.message);
        }
        return { detected: score > 40, score: Math.min(score, 100), detections };
    }

    async detectPermissions() {
        let score = 0;
        const detections = [];
        try {
            if (navigator.permissions && navigator.permissions.query) {
                const permNames = ['notifications', 'geolocation', 'camera', 'microphone', 'midi'];
                const permChecks = await Promise.all(
                    permNames.map(name =>
                        navigator.permissions.query({ name }).catch(() => ({ state: 'error' }))
                    )
                );
                const allDenied = permChecks.every(p => p.state === 'denied' || p.state === 'error');
                if (allDenied) {
                    score += 30;
                    detections.push('all_permissions_denied');
                }
            } else {
                score += 30;
                detections.push('permissions_api_missing');
            }
        } catch (e) {
            score += 35;
            detections.push('permissions_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectLanguages() {
        let score = 0;
        const detections = [];
        try {
            const langs = navigator.languages;
            if (!langs || langs.length === 0) {
                score += 35;
                detections.push('no_languages');
            }
            const lang = navigator.language;
            if (!lang) {
                score += 30;
                detections.push('no_language');
            }
        } catch (e) {
            score += 40;
            detections.push('languages_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectTimezone() {
        let score = 0;
        const detections = [];
        try {
            const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
            if (!tz) {
                score += 40;
                detections.push('no_timezone');
            }
            const offset = new Date().getTimezoneOffset();
            if (offset === 0 && !tz) {
                score += 30;
                detections.push('utc_offset_no_tz');
            }
        } catch (e) {
            score += 45;
            detections.push('timezone_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectScreen() {
        let score = 0;
        const detections = [];
        try {
            const { width, height, colorDepth, pixelDepth } = screen;
            if (!width || !height) {
                score += 40;
                detections.push('no_screen_size');
            }
            if (colorDepth === 0 || pixelDepth === 0) {
                score += 35;
                detections.push('zero_depth');
            }
            if (width <= 800 || height <= 600) {
                score += 18;
                detections.push('small_screen');
            }
        } catch (e) {
            score += 40;
            detections.push('screen_error');
        }
        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectHardwareConcurrency() {
        let score = 0;
        const detections = [];
        try {
            const c = navigator.hardwareConcurrency;
            if (c === undefined || c === null) {
                score += 40;
                detections.push('no_concurrency');
            } else if (c <= 1) {
                score += 35;
                detections.push('single_core');
            } else if (c > 64) {
                score += 30;
                detections.push('unrealistic_cores');
            }
        } catch (e) {
            score += 40;
            detections.push('concurrency_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectDeviceMemory() {
        let score = 0;
        const detections = [];
        try {
            const mem = navigator.deviceMemory;
            if (mem === undefined || mem === null) {
                score += 30;
                detections.push('no_device_memory');
            } else if (mem <= 0.25) {
                score += 35;
                detections.push('low_memory');
            } else if (mem > 64) {
                score += 25;
                detections.push('unrealistic_memory');
            }
        } catch (e) {
            score += 30;
            detections.push('memory_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectStorage() {
        let score = 0;
        const detections = [];
        try {
            localStorage.setItem('_md_test', '1');
            localStorage.removeItem('_md_test');
        } catch (e) {
            score += 30;
            detections.push('localStorage_denied');
        }
        try {
            sessionStorage.setItem('_md_test', '1');
            sessionStorage.removeItem('_md_test');
        } catch (e) {
            score += 30;
            detections.push('sessionStorage_denied');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectNavigatorProps() {
        let score = 0;
        const detections = [];
        try {
            if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
                const devices = await navigator.mediaDevices.enumerateDevices().catch(() => []);
                if (devices.length === 0) {
                    score += 25;
                    detections.push('no_media_devices');
                }
            }
        } catch (e) {
            score += 25;
            detections.push('media_devices_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectWindowProps() {
        let score = 0;
        const detections = [];
        try {
            const outerW = window.outerWidth;
            const outerH = window.outerHeight;
            if (outerW === 0 || outerH === 0) {
                score += 35;
                detections.push('zero_outer_size');
            }
            if (window.innerWidth > outerW || window.innerHeight > outerH) {
                score += 25;
                detections.push('inner_larger_than_outer');
            }
        } catch (e) {
            score += 30;
            detections.push('window_size_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectIframe() {
        let score = 0;
        const detections = [];
        try {
            if (window.self !== window.top) {
                score += 25;
                detections.push('in_iframe');
            }
        } catch (e) {
            score += 45;
            detections.push('cross_origin_frame');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectNotification() {
        let score = 0;
        const detections = [];
        try {
            if ('Notification' in window) {
                if (Notification.permission === 'denied') {
                    score += 12;
                    detections.push('notification_denied');
                }
            } else {
                score += 25;
                detections.push('no_notification');
            }
        } catch (e) {
            score += 25;
            detections.push('notification_error');
        }
        return { detected: score > 20, score: Math.min(score, 100), detections };
    }

    async detectBattery() {
        let score = 0;
        const detections = [];
        try {
            if (navigator.getBattery) {
                const battery = await navigator.getBattery().catch(() => null);
                if (battery) {
                    if (battery.level === undefined || battery.charging === undefined) {
                        score += 25;
                        detections.push('battery_props_missing');
                    }
                }
            } else {
                score += 18;
                detections.push('no_battery_api');
            }
        } catch (e) {
            score += 25;
            detections.push('battery_error');
        }
        return { detected: score > 20, score: Math.min(score, 100), detections };
    }

    async detectMediaDevices() {
        let score = 0;
        const detections = [];
        try {
            if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
                const devices = await navigator.mediaDevices.enumerateDevices().catch(() => []);
                const videoInputs = devices.filter(d => d.kind === 'videoinput');
                const audioInputs = devices.filter(d => d.kind === 'audioinput');
                if (videoInputs.length === 0 && audioInputs.length === 0) {
                    score += 30;
                    detections.push('no_media_inputs');
                }
            } else {
                score += 25;
                detections.push('no_media_api');
            }
        } catch (e) {
            score += 30;
            detections.push('media_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectConnection() {
        let score = 0;
        const detections = [];
        try {
            const conn = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
            if (!conn) {
                score += 18;
                detections.push('no_connection_api');
            } else {
                if (conn.type === 'vpn') {
                    score += 50;
                    detections.push('vpn_detected');
                }
                if (conn.type === 'proxy') {
                    score += 50;
                    detections.push('proxy_detected');
                }
                if (conn.saveData === true) {
                    score += 18;
                    detections.push('save_data_enabled');
                }
            }
        } catch (e) {
            score += 18;
            detections.push('connection_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectAdBlock() {
        let score = 0;
        const detections = [];
        try {
            const el = document.createElement('div');
            el.innerHTML = '&nbsp;';
            el.className = 'adsbox';
            el.style.cssText = 'position:absolute;left:-9999px;top:-9999px;width:1px;height:1px';
            document.body.appendChild(el);
            if (el.offsetHeight === 0) {
                score += 25;
                detections.push('adblock_detected');
            }
            document.body.removeChild(el);
        } catch (e) {
            score += 18;
            detections.push('adblock_check_error');
        }
        return { detected: score > 18, score: Math.min(score, 100), detections };
    }

    async detectMathFingerprint() {
        let score = 0;
        const detections = [];
        try {
            const mathResults = {
                sin: Math.sin(Math.PI / 3),
                tan: Math.tan(1e7),
                log10: Math.log10(100),
                asin: Math.asin(0.5),
                atan2: Math.atan2(1, 2),
                cos: Math.cos(Math.PI / 4),
                exp: Math.exp(1),
                sqrt: Math.sqrt(2)
            };
            for (const key in mathResults) {
                if (!isFinite(mathResults[key]) || isNaN(mathResults[key])) {
                    score += 25;
                    detections.push('math_' + key + '_invalid');
                }
            }
        } catch (e) {
            score += 30;
            detections.push('math_error');
        }
        return { detected: score > 18, score: Math.min(score, 100), detections };
    }

    async detectGPUFingerprint() {
        let score = 0;
        const detections = [];
        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
            if (gl) {
                const maxRenderSize = gl.getParameter(gl.MAX_RENDERBUFFER_SIZE);
                const maxCombinedTexUnits = gl.getParameter(gl.MAX_COMBINED_TEXTURE_IMAGE_UNITS);
                if (maxRenderSize <= 1024) {
                    score += 25;
                    detections.push('small_renderbuffer');
                }
                if (maxCombinedTexUnits <= 8) {
                    score += 18;
                    detections.push('few_texture_units');
                }
            }
        } catch (e) {
            score += 18;
            detections.push('gpu_error');
        }
        return { detected: score > 18, score: Math.min(score, 100), detections };
    }

    async detectSpeech() {
        let score = 0;
        const detections = [];
        try {
            if ('speechSynthesis' in window) {
                const voices = window.speechSynthesis.getVoices();
                if (voices.length === 0) {
                    score += 18;
                    detections.push('no_speech_voices');
                }
            } else {
                score += 25;
                detections.push('no_speech_api');
            }
        } catch (e) {
            score += 18;
            detections.push('speech_error');
        }
        return { detected: score > 12, score: Math.min(score, 100), detections };
    }

    async detectVMFeatures() {
        let score = 0;
        const detections = [];
        try {
            const vmPatterns = {
                vmware: [/vmware/i, /virtualbox/i, /parallels/i, /hyper[- ]?v/i, /qemu/i, /kvm/i, /xen/i],
                cpuFeatures: [/[0-9]+cpu/i, /core\s*\d+/i, /processor/i],
            };

            const ua = navigator.userAgent || '';
            for (const [vmType, patterns] of Object.entries(vmPatterns)) {
                for (const pattern of patterns) {
                    if (pattern.test(ua)) {
                        score += 40;
                        detections.push(`vm_pattern:${vmType}`);
                    }
                }
            }

            try {
                const canvas = document.createElement('canvas');
                const gl = canvas.getContext('webgl');
                if (gl) {
                    const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                    if (debugInfo) {
                        const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                        const vmRendererPatterns = [
                            /vmware/i, /virtualbox/i, /parallels/i, /qemu/i,
                            /kvm/i, /xen/i, /hyper[ -]?v/i, /virtual/i,
                            /swiftshader/i, /llvmpipe/i
                        ];

                        for (const pattern of vmRendererPatterns) {
                            if (pattern.test(renderer)) {
                                score += 45;
                                detections.push(`vm_renderer:${pattern}`);
                            }
                        }
                    }
                }
            } catch (e) {}

            try {
                const cpuCores = navigator.hardwareConcurrency;
                if (cpuCores === 1) {
                    score += 25;
                    detections.push('vm_single_core');
                } else if (cpuCores === 2) {
                    score += 12;
                    detections.push('vm_dual_core');
                }
            } catch (e) {}

            try {
                const memory = navigator.deviceMemory;
                if (memory && memory < 2) {
                    score += 30;
                    detections.push('vm_low_memory');
                }
            } catch (e) {}

        } catch (e) {
            score += 30;
            detections.push('vm_features_error: ' + e.message);
        }
        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectSandboxEscape() {
        let score = 0;
        const detections = [];
        try {
            try {
                if (typeof process !== 'undefined') {
                    detections.push('node_env_detected');
                    score += 35;
                }
            } catch (e) {}

            try {
                if (typeof require !== 'undefined') {
                    detections.push('require_defined');
                    score += 45;
                }
            } catch (e) {}

            try {
                if (typeof __dirname !== 'undefined' || typeof __filename !== 'undefined') {
                    detections.push('node_path_vars');
                    score += 40;
                }
            } catch (e) {}

        } catch (e) {
            score += 25;
            detections.push('sandbox_escape_error: ' + e.message);
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectDebuggerEnhanced() {
        let score = 0;
        const detections = [];
        try {
            const devToolsChecks = {
                windowSize: () => {
                    return window.outerWidth === 0 || window.outerHeight === 0;
                },
                consoleDebug: () => {
                    const start = performance.now();
                    console.debug('debugger check');
                    const end = performance.now();
                    return end - start > 100;
                }
            };

            for (const [checkName, checkFn] of Object.entries(devToolsChecks)) {
                try {
                    if (checkFn()) {
                        score += 30;
                        detections.push(`devtools_${checkName}`);
                    }
                } catch (e) {}
            }

            const props = ['__webdriver_evaluate', '__selenium_evaluate', '__webdriver_script_fn'];
            for (const prop of props) {
                if (window[prop] !== undefined) {
                    score += 35;
                    detections.push(`debugger_prop:${prop}`);
                }
            }

        } catch (e) {
            score += 25;
            detections.push('debugger_enhanced_error: ' + e.message);
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectFontEnumeration() {
        let score = 0;
        const detections = [];
        try {
            if (document.fonts && document.fonts.forEach) {
                const fontList = [];
                document.fonts.forEach(font => {
                    fontList.push(font.family);
                });

                detections.push('enumerated_fonts:' + fontList.length);
                if (fontList.length === 0) {
                    score += 25;
                    detections.push('no_enumerated_fonts');
                } else if (fontList.length < 5) {
                    score += 12;
                    detections.push('few_enumerated_fonts');
                }

                this.fingerprintComponents.enumeratedFonts = fontList;
            } else {
                score += 18;
                detections.push('font_api_unavailable');
            }

        } catch (e) {
            score += 25;
            detections.push('font_enumeration_error: ' + e.message);
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectFontMetrics() {
        let score = 0;
        const detections = [];
        try {
            const testString = 'mmmmmmmmmmlli';
            const canvas = document.createElement('canvas');
            const ctx = canvas.getContext('2d');

            const metrics = [];
            const testFonts = ['Arial', 'Times New Roman', 'Courier New', 'Georgia', 'Verdana'];

            for (const font of testFonts) {
                ctx.font = `72px "${font}", sans-serif`;
                const metrics2 = ctx.measureText(testString);
                metrics.push({
                    font: font,
                    width: metrics2.width,
                    actualBoundingBoxLeft: metrics2.actualBoundingBoxLeft,
                    actualBoundingBoxRight: metrics2.actualBoundingBoxRight
                });
            }

            let uniqueWidths = new Set(metrics.map(m => Math.round(m.width)));
            detections.push('unique_widths:' + uniqueWidths.size);

            if (uniqueWidths.size < 3) {
                score += 30;
                detections.push('fonts_have_same_width');
            }

        } catch (e) {
            score += 25;
            detections.push('font_metrics_error: ' + e.message);
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectPluginsEnhanced() {
        let score = 0;
        const detections = [];
        try {
            const plugins = navigator.plugins;
            if (!plugins || plugins.length === 0) {
                score += 35;
                detections.push('no_plugins');
            } else {
                const pluginNames = Array.from(plugins).map(p => p.name);
                detections.push('plugin_count:' + plugins.length);

                if (plugins.length < 2) {
                    score += 30;
                    detections.push('very_few_plugins');
                }

                const commonPlugins = ['PDF Viewer', 'Chrome PDF Viewer', 'Chromium PDF Viewer'];
                const hasPDF = pluginNames.some(p =>
                    commonPlugins.some(cp => p.includes(cp))
                );
                if (!hasPDF) {
                    score += 18;
                    detections.push('no_pdf_plugin');
                }
            }

            if (navigator.mimeTypes) {
                const mimeTypes = Array.from(navigator.mimeTypes).map(m => m.type);
                detections.push('mime_type_count:' + mimeTypes.length);
                if (mimeTypes.length < 2) {
                    score += 18;
                    detections.push('few_mime_types');
                }
            }

        } catch (e) {
            score += 40;
            detections.push('plugins_error: ' + e.message);
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectPluginFingerprint() {
        let score = 0;
        const detections = [];
        try {
            const plugins = navigator.plugins;
            const mimeTypes = navigator.mimeTypes || [];

            const pluginData = {
                plugins: [],
                mimeTypes: [],
                totalLength: 0
            };

            if (plugins && plugins.length > 0) {
                for (let i = 0; i < plugins.length; i++) {
                    const p = plugins[i];
                    const pluginInfo = {
                        name: p.name,
                        filename: p.filename,
                        description: p.description
                    };
                    pluginData.plugins.push(pluginInfo);
                    pluginData.totalLength += (p.name + p.description + p.filename).length;
                }
            }

            const hash = this.hashString(JSON.stringify(pluginData.plugins));
            this.fingerprintComponents.pluginFingerprint = hash;
            detections.push('plugin_hash:' + hash.substring(0, 16));
            detections.push('plugin_data_length:' + pluginData.totalLength);

            if (pluginData.totalLength < 50) {
                score += 25;
                detections.push('minimal_plugin_data');
            }

        } catch (e) {
            score += 30;
            detections.push('plugin_fingerprint_error: ' + e.message);
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    getHeader(name) {
        return null;
    }

    hashString(str) {
        let hash = 0;
        for (let i = 0; i < str.length; i++) {
            const char = str.charCodeAt(i);
            hash = ((hash << 5) - hash) + char;
            hash = hash & hash;
        }
        return Math.abs(hash).toString(16);
    }

    checkDatacenterIP(ip) {
        const datacenterPrefixes = [
            '3.', '4.', '8.', '13.', '15.', '16.', '17.', '18.', '20.',
            '23.', '34.', '35.', '40.', '44.', '45.', '47.', '48.', '49.',
            '50.', '52.', '54.', '63.', '64.', '65.', '66.', '67.', '68.',
            '69.', '70.', '71.', '72.', '73.', '74.', '75.', '76.', '77.',
            '78.', '79.', '80.', '81.', '82.', '83.', '84.', '85.', '86.',
            '87.', '88.', '89.', '90.', '91.', '92.', '93.', '94.', '95.',
            '96.', '97.', '98.', '99.', '104.', '108.', '130.', '131.',
            '136.', '142.', '143.', '144.', '146.', '147.', '148.', '149.',
            '150.', '151.', '157.', '158.', '159.', '160.', '161.', '162.',
            '163.', '164.', '165.', '166.', '167.', '168.', '169.', '170.'
        ];

        for (const prefix of datacenterPrefixes) {
            if (ip.startsWith(prefix)) {
                return true;
            }
        }
        return false;
    }

    generateFingerprint() {
        const components = [];
        try {
            components.push('scrn:' + screen.width + 'x' + screen.height + 'x' + screen.colorDepth);
        } catch (e) {}
        try {
            components.push('lang:' + (navigator.language || ''));
        } catch (e) {}
        try {
            components.push('tz:' + (Intl.DateTimeFormat().resolvedOptions().timeZone || ''));
        } catch (e) {}
        try {
            components.push('cpu:' + (navigator.hardwareConcurrency || ''));
        } catch (e) {}
        try {
            components.push('mem:' + (navigator.deviceMemory || ''));
        } catch (e) {}
        try {
            components.push('plat:' + (navigator.platform || ''));
        } catch (e) {}
        try {
            const canvas = document.createElement('canvas');
            canvas.width = 120;
            canvas.height = 60;
            const ctx = canvas.getContext('2d');
            if (ctx) {
                ctx.textBaseline = 'top';
                ctx.font = '16px Arial';
                ctx.fillStyle = '#f60';
                ctx.fillRect(0, 0, 60, 60);
                ctx.fillStyle = '#069';
                ctx.fillText('fp', 12, 25);
                const dataUrl = canvas.toDataURL();
                const hash = dataUrl.split(',')[1] || dataUrl;
                components.push('cnv:' + hash.substring(0, 32));
            }
        } catch (e) {}
        try {
            const gl = document.createElement('canvas').getContext('webgl');
            if (gl) {
                const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                if (debugInfo) {
                    const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                    components.push('wgl:' + (renderer || '').substring(0, 48));
                }
            }
        } catch (e) {}
        try {
            const offset = new Date().getTimezoneOffset();
            components.push('tzoff:' + offset);
        } catch (e) {}
        try {
            components.push('cookie:' + (navigator.cookieEnabled ? '1' : '0'));
        } catch (e) {}
        if (this.fingerprintComponents.canvas) {
            components.push('canvash:' + this.fingerprintComponents.canvas.substring(0, 16));
        }
        if (this.fingerprintComponents.audio) {
            components.push('audioh:' + this.fingerprintComponents.audio.substring(0, 16));
        }
        if (this.fingerprintComponents.fonts && this.fingerprintComponents.fonts.length > 0) {
            components.push('fonts:' + this.fingerprintComponents.fonts.slice(0, 5).join(','));
        }
        return components.join('|');
    }

    async runAll() {
        const chainResult = await this.runChain();
        const fingerprint = this.generateFingerprint();
        return Object.assign(chainResult, { fingerprint });
    }

    toJSON() {
        return {
            risk_score: this.riskScore,
            chain_count: this.detectionChain.length,
            results: this.results,
            fingerprint_components: this.fingerprintComponents,
            timing_data: this.timingData,
            ml_features: this.mlFeatures,
            ml_risk_score: this.options.enableMLScoring ? this.calculateMLRiskScore() : null
        };
    }
}

if (typeof window !== 'undefined') {
    window.EnhancedEnvironmentDetector = EnhancedEnvironmentDetector;
}
