class HJTPXEnvironmentDetector {
    constructor(options = {}) {
        this.options = Object.assign({
            apiBase: '/api/v1',
            sampleRate: 1.0,
            chainCount: 35,
            enableAll: true,
            sessionId: null,
            timeout: 20000,
            fingerprintPrecision: 'high',
            enableMLScoring: true,
            mlThreshold: 0.75
        }, options);
        
        this.results = {};
        this.riskScore = 0;
        this.detectionChain = [];
        this.detectionId = 'hjtpx_det_' + Date.now() + '_' + Math.random().toString(36).substr(2, 16);
        this.timingData = {};
        this.fingerprintComponents = {};
        this.mlFeatures = {};
        
        this.weights = {
            canvas256: 18,
            canvasStable: 15,
            canvasEntropy: 12,
            webgl: 16,
            webgl2: 14,
            webglVendor: 12,
            webglRenderer: 12,
            audio: 14,
            fonts: 12,
            plugins: 8,
            webrtc: 18,
            webrtcLeak: 16,
            webdriver: 22,
            puppeteer: 22,
            playwright: 22,
            selenium: 20,
            headless: 20,
            advancedHeadless: 18,
            stealthMode: 16,
            proxyVPN: 22,
            vpnIndicators: 18,
            torExitNode: 20,
            emulator: 18,
            virtualization: 16,
            container: 15,
            deviceFingerprint: 20
        };
    }

    getDetectionMethods() {
        return [
            { name: 'detectCanvas256', category: 'fingerprint' },
            { name: 'detectCanvasStable', category: 'fingerprint' },
            { name: 'detectCanvasEntropy', category: 'fingerprint' },
            { name: 'detectWebGLEnhanced', category: 'fingerprint' },
            { name: 'detectWebGL2Enhanced', category: 'fingerprint' },
            { name: 'detectWebGLVendor', category: 'fingerprint' },
            { name: 'detectAudioFingerprint', category: 'fingerprint' },
            { name: 'detectFontsEnhanced', category: 'fingerprint' },
            { name: 'detectPluginsFingerprint', category: 'fingerprint' },
            { name: 'detectWebRTCEnhanced', category: 'network' },
            { name: 'detectWebRTCLeak', category: 'network' },
            { name: 'detectProxyVPN', category: 'network' },
            { name: 'detectVPNIndicators', category: 'network' },
            { name: 'detectTorExitNode', category: 'network' },
            { name: 'detectNetworkAnomalies', category: 'network' },
            { name: 'detectHeadless', category: 'automation' },
            { name: 'detectAdvancedHeadless', category: 'automation' },
            { name: 'detectStealthMode', category: 'automation' },
            { name: 'detectWebDriver', category: 'automation' },
            { name: 'detectPuppeteer', category: 'automation' },
            { name: 'detectPlaywright', category: 'automation' },
            { name: 'detectSelenium', category: 'automation' },
            { name: 'detectAutomationFramework', category: 'automation' },
            { name: 'detectEmulator', category: 'environment' },
            { name: 'detectVirtualization', category: 'environment' },
            { name: 'detectContainer', category: 'environment' },
            { name: 'detectDeviceFingerprint', category: 'fingerprint' },
            { name: 'detectSystemMetrics', category: 'system' },
            { name: 'detectNavigatorProps', category: 'system' },
            { name: 'detectTimingAnomaly', category: 'behavior' }
        ];
    }

    generateDetectionChain(count) {
        const allMethods = this.getDetectionMethods();
        const shuffled = [...allMethods].sort(() => Math.random() - 0.5);
        const selected = shuffled.slice(0, Math.min(count, allMethods.length));
        const methodAliases = {};
        selected.forEach((method, i) => {
            methodAliases[method.name] = 'chk_' + i.toString(36) + '_' + Math.random().toString(36).substr(2, 6);
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
            fingerprint: this.generateDeviceFingerprint(),
            ml_features: this.mlFeatures,
            ml_risk_score: this.options.enableMLScoring ? this.calculateMLRiskScore() : null,
            fingerprint_matches: this.fingerprintMatches
        };
    }

    calculateRiskScore() {
        let weightedScore = 0;
        let totalWeight = 0;

        for (const key in this.results) {
            const result = this.results[key];
            if (result && typeof result.score === 'number') {
                const weight = this.weights[key] || 5;
                weightedScore += result.score * weight;
                totalWeight += weight;
            }
        }

        if (totalWeight === 0) return 0;

        let baseScore = weightedScore / totalWeight;

        const automationMethods = ['detectWebDriver', 'detectPuppeteer', 'detectPlaywright', 'detectSelenium', 'detectHeadless', 'detectAdvancedHeadless', 'detectStealthMode'];
        const automationDetected = automationMethods.filter(m => this.results[m] && this.results[m].detected).length;

        if (automationDetected >= 4) {
            baseScore = Math.min(baseScore * 2.2 + 35, 100);
        } else if (automationDetected >= 3) {
            baseScore = Math.min(baseScore * 1.8 + 25, 100);
        } else if (automationDetected >= 2) {
            baseScore = Math.min(baseScore * 1.5 + 15, 100);
        } else if (automationDetected >= 1) {
            baseScore = Math.min(baseScore * 1.3 + 10, 100);
        }

        const proxyMethods = ['detectProxyVPN', 'detectVPNIndicators', 'detectTorExitNode', 'detectWebRTCLeak'];
        const proxyDetected = proxyMethods.filter(m => this.results[m] && this.results[m].detected).length;

        if (proxyDetected >= 3) {
            baseScore = Math.min(baseScore * 1.6 + 25, 100);
        } else if (proxyDetected >= 2) {
            baseScore = Math.min(baseScore * 1.4 + 20, 100);
        } else if (proxyDetected >= 1) {
            baseScore = Math.min(baseScore * 1.2 + 10, 100);
        }

        const virtualizationMethods = ['detectEmulator', 'detectVirtualization', 'detectContainer'];
        const virtualizationDetected = virtualizationMethods.filter(m => this.results[m] && this.results[m].detected).length;

        if (virtualizationDetected >= 2) {
            baseScore = Math.min(baseScore * 1.4 + 15, 100);
        } else if (virtualizationDetected >= 1) {
            baseScore = Math.min(baseScore * 1.2 + 8, 100);
        }

        return Math.round(Math.min(Math.max(baseScore, 0), 100));
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
            environment_score: 0,
            system_score: 0,
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
                    case 'environment':
                        this.mlFeatures.environment_score += result.score;
                        break;
                    case 'system':
                        this.mlFeatures.system_score += result.score;
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

        mlScore += this.mlFeatures.detected_checks * 6;

        mlScore += Math.min(this.mlFeatures.automation_score * 0.35, 35);
        mlScore += Math.min(this.mlFeatures.fingerprint_score * 0.25, 25);
        mlScore += Math.min(this.mlFeatures.network_score * 0.25, 25);
        mlScore += Math.min(this.mlFeatures.environment_score * 0.25, 20);

        if (this.mlFeatures.timing_variance > 0.85) {
            mlScore += 12;
        }

        if (this.mlFeatures.entropy_score < 0.15) {
            mlScore += 10;
        }

        if (this.mlFeatures.consistency_score < 0.65) {
            mlScore += 8;
        }

        const highRiskPatterns = ['headless', 'webdriver', 'puppeteer', 'playwright', 'selenium', 'tor', 'vpn', 'proxy', 'vm_', 'sandbox', 'emulator'];
        for (const pattern of highRiskPatterns) {
            if (this.mlFeatures.suspicious_patterns.some(p => p.toLowerCase().includes(pattern))) {
                mlScore += 4;
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

        const canvasChecks = ['detectCanvas256', 'detectCanvasStable', 'detectCanvasEntropy'];
        if (canvasChecks.every(m => this.results[m])) {
            const scores = canvasChecks.map(m => this.results[m].score);
            const variance = this.calculateVariance(scores);
            if (variance < 15) consistentCount++;
            totalChecks++;
        }

        const webglChecks = ['detectWebGLEnhanced', 'detectWebGL2Enhanced', 'detectWebGLVendor'];
        if (webglChecks.every(m => this.results[m])) {
            const scores = webglChecks.map(m => this.results[m].score);
            const variance = this.calculateVariance(scores);
            if (variance < 12) consistentCount++;
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

    getMethodCategory(methodName) {
        const methods = this.getDetectionMethods();
        const method = methods.find(m => m.name === methodName);
        return method ? method.category : null;
    }

    async detectCanvas256() {
        let score = 0;
        const detections = [];
        let fingerprint = '';

        try {
            const canvas = document.createElement('canvas');
            canvas.width = 512;
            canvas.height = 256;
            const ctx = canvas.getContext('2d');

            if (!ctx) {
                score += 60;
                detections.push('no_canvas_context');
                return { detected: true, score: Math.min(score, 100), detections, fingerprint: '' };
            }

            ctx.fillStyle = '#000';
            ctx.fillRect(0, 0, 512, 256);

            const drawPattern = (seed) => {
                const rng = this.createPRNG(seed);
                for (let i = 0; i < 1024; i++) {
                    const x = Math.floor(rng() * 512);
                    const y = Math.floor(rng() * 256);
                    const r = Math.floor(rng() * 256);
                    const g = Math.floor(rng() * 256);
                    const b = Math.floor(rng() * 256);
                    const a = rng();
                    ctx.fillStyle = `rgba(${r},${g},${b},${a})`;
                    ctx.fillRect(x, y, 2, 2);
                }
            };

            drawPattern(0xDEADBEEF);
            drawPattern(0xCAFEBABE);

            ctx.textBaseline = 'top';
            ctx.font = 'bold 12pt Arial';
            ctx.fillStyle = '#fff';
            ctx.fillText('HJTPX Fingerprint 256-bit', 10, 10);

            for (let i = 0; i < 26; i++) {
                const char = String.fromCharCode(65 + i);
                ctx.font = `${8 + (i % 6)}pt Times New Roman`;
                ctx.fillStyle = `rgba(${128 + i * 5}, ${255 - i * 5}, ${100 + i * 6}, 0.8)`;
                ctx.fillText(char.repeat(10), 10 + i * 18, 40 + (i % 4) * 20);
            }

            const gradient = ctx.createLinearGradient(0, 100, 512, 256);
            gradient.addColorStop(0, 'rgba(255, 0, 0, 0.3)');
            gradient.addColorStop(0.33, 'rgba(0, 255, 0, 0.3)');
            gradient.addColorStop(0.66, 'rgba(0, 0, 255, 0.3)');
            gradient.addColorStop(1, 'rgba(255, 255, 0, 0.3)');
            ctx.fillStyle = gradient;
            ctx.fillRect(0, 100, 512, 156);

            ctx.globalCompositeOperation = 'overlay';
            ctx.fillStyle = 'rgba(255, 255, 255, 0.1)';
            for (let i = 0; i < 50; i++) {
                ctx.beginPath();
                ctx.arc(Math.random() * 512, Math.random() * 256, Math.random() * 20 + 5, 0, Math.PI * 2);
                ctx.fill();
            }

            const dataURL = canvas.toDataURL('image/png');
            const base64Data = dataURL.split(',')[1];
            fingerprint = this.sha256Hash(base64Data).substring(0, 64);

            this.fingerprintComponents.canvas256 = fingerprint;
            detections.push('canvas_fingerprint_generated');

            const imageData = ctx.getImageData(0, 0, 512, 256);
            let zeroPixels = 0;
            for (let i = 0; i < imageData.data.length; i += 4) {
                if (imageData.data[i] === 0 && imageData.data[i + 1] === 0 && imageData.data[i + 2] === 0) {
                    zeroPixels++;
                }
            }

            if (zeroPixels > imageData.data.length / 4 * 0.9) {
                score += 50;
                detections.push('canvas_all_black');
            }

            const dataURL2 = canvas.toDataURL('image/png');
            if (dataURL !== dataURL2) {
                score += 35;
                detections.push('canvas_unstable');
            }

        } catch (e) {
            score += 50;
            detections.push('canvas_error: ' + e.message);
        }

        return { detected: score > 40, score: Math.min(score, 100), detections, fingerprint };
    }

    async detectCanvasStable() {
        let score = 0;
        const detections = [];
        const results = [];

        try {
            for (let i = 0; i < 8; i++) {
                const canvas = document.createElement('canvas');
                canvas.width = 256;
                canvas.height = 128;
                const ctx = canvas.getContext('2d');
                if (ctx) {
                    ctx.fillStyle = `rgb(${100 + i * 15}, ${150 + i * 10}, ${200 - i * 10})`;
                    ctx.fillRect(0, 0, 256, 128);
                    ctx.font = `${12 + (i % 4)}px Arial`;
                    ctx.fillStyle = '#fff';
                    ctx.fillText(`Test ${i + 1}: HJTPX`, 10, 30 + i * 15);
                    results.push(canvas.toDataURL());
                }
            }

            if (results.length >= 4) {
                const firstHash = this.sha256Hash(results[0]);
                let matches = 0;
                for (let i = 1; i < results.length; i++) {
                    if (this.sha256Hash(results[i]) === firstHash) {
                        matches++;
                    }
                }

                if (matches === results.length - 1) {
                    score += 25;
                    detections.push('canvas_fully_stable');
                } else if (matches >= results.length - 2) {
                    score += 15;
                    detections.push('canvas_mostly_stable');
                } else if (matches < 2) {
                    score += 30;
                    detections.push('canvas_highly_unstable');
                }
            }

        } catch (e) {
            score += 35;
            detections.push('canvas_stability_error: ' + e.message);
        }

        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectCanvasEntropy() {
        let score = 0;
        const detections = [];

        try {
            const canvas = document.createElement('canvas');
            canvas.width = 128;
            canvas.height = 128;
            const ctx = canvas.getContext('2d');

            if (!ctx) {
                score += 40;
                detections.push('no_canvas_context');
                return { detected: true, score: Math.min(score, 100), detections };
            }

            for (let y = 0; y < 128; y++) {
                for (let x = 0; x < 128; x++) {
                    const r = (x * y + x + y) % 256;
                    const g = ((x * 3) ^ (y * 5)) % 256;
                    const b = Math.sin((x + y) * 0.1) * 128 + 127;
                    ctx.fillStyle = `rgb(${r}, ${g}, ${Math.floor(b)})`;
                    ctx.fillRect(x, y, 1, 1);
                }
            }

            const imageData = ctx.getImageData(0, 0, 128, 128);
            const pixelValues = new Set();

            for (let i = 0; i < imageData.data.length; i += 4) {
                const rgb = `${imageData.data[i]},${imageData.data[i + 1]},${imageData.data[i + 2]}`;
                pixelValues.add(rgb);
            }

            const entropy = pixelValues.size / (128 * 128);
            this.fingerprintComponents.canvasEntropy = entropy;

            if (entropy < 0.1) {
                score += 45;
                detections.push('canvas_low_entropy');
            } else if (entropy < 0.3) {
                score += 25;
                detections.push('canvas_moderate_entropy');
            }

        } catch (e) {
            score += 30;
            detections.push('canvas_entropy_error: ' + e.message);
        }

        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectWebGLEnhanced() {
        let score = 0;
        const detections = [];

        try {
            const canvas = document.createElement('canvas');
            canvas.width = 512;
            canvas.height = 256;
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

                this.fingerprintComponents.webglVendor = vendor || '';
                this.fingerprintComponents.webglRenderer = renderer || '';

                detections.push('webgl_vendor:' + (vendor || 'unknown').substring(0, 40));
                detections.push('webgl_renderer:' + (renderer || 'unknown').substring(0, 60));

                const softwarePatterns = [
                    /swiftshader/i, /llvmpipe/i, /mesa/i, /virtual/i,
                    /google\s*inc/i, /software/i, /microsoft/i, /apple/i,
                    /vmware/i, /parallels/i, /virtualbox/i, /qxl/i,
                    /render/i, /angle/i, /skia/i, /generic/i, /unknown/i, /default/i
                ];

                let softwareScore = 0;
                for (const pattern of softwarePatterns) {
                    if (pattern.test(renderer || '')) {
                        softwareScore += 15;
                    }
                }

                if (softwareScore > 0) {
                    score += Math.min(softwareScore, 50);
                    detections.push('software_renderer_detected');
                }
            } else {
                score += 35;
                detections.push('no_webgl_debug_info');
            }

            const maxTexSize = gl.getParameter(gl.MAX_TEXTURE_SIZE);
            detections.push('max_texture_size:' + maxTexSize);
            if (maxTexSize <= 2048) {
                score += 25;
                detections.push('limited_texture_size');
            }

            const maxVertAttribs = gl.getParameter(gl.MAX_VERTEX_ATTRIBS);
            detections.push('max_vertex_attribs:' + maxVertAttribs);
            if (maxVertAttribs <= 8) {
                score += 20;
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
                    score += 30;
                    detections.push('low_shader_precision');
                }
            }

            const ext = gl.getExtension('EXT_texture_filter_anisotropic');
            if (!ext) {
                score += 15;
                detections.push('no_anisotropic_filtering');
            }

            const supportedExts = gl.getSupportedExtensions();
            if (supportedExts) {
                detections.push('extension_count:' + supportedExts.length);
                if (supportedExts.length < 15) {
                    score += 25;
                    detections.push('few_webgl_extensions');
                }

                const criticalExts = [
                    'OES_texture_float', 'WEBGL_debug_renderer_info',
                    'EXT_texture_filter_anisotropic', 'OES_standard_derivatives',
                    'WEBGL_lose_context', 'OES_vertex_array_object'
                ];

                let missingCritical = 0;
                for (const extName of criticalExts) {
                    if (!supportedExts.includes(extName)) {
                        missingCritical++;
                    }
                }

                if (missingCritical >= 3) {
                    score += 20;
                    detections.push('many_missing_critical_exts');
                }
            }

            const vertexShader = gl.createShader(gl.VERTEX_SHADER);
            const fragmentShader = gl.createShader(gl.FRAGMENT_SHADER);
            gl.shaderSource(vertexShader, 'attribute vec2 position; void main() { gl_Position = vec4(position, 0.0, 1.0); }');
            gl.compileShader(vertexShader);
            const vertCompiled = gl.getShaderParameter(vertexShader, gl.COMPILE_STATUS);
            if (!vertCompiled) {
                score += 20;
                detections.push('vertex_shader_compile_failed');
            }

            const testProgram = gl.createProgram();
            gl.attachShader(testProgram, vertexShader);
            gl.linkProgram(testProgram);
            const linked = gl.getProgramParameter(testProgram, gl.LINK_STATUS);
            if (!linked) {
                score += 15;
                detections.push('program_link_failed');
            }

            gl.deleteProgram(testProgram);
            gl.deleteShader(vertexShader);
            gl.deleteShader(fragmentShader);

        } catch (e) {
            score += 50;
            detections.push('webgl_error: ' + e.message);
        }

        return { detected: score > 40, score: Math.min(score, 100), detections };
    }

    async detectWebGL2Enhanced() {
        let score = 0;
        const detections = [];

        try {
            const canvas = document.createElement('canvas');
            canvas.width = 512;
            canvas.height = 256;
            const gl2 = canvas.getContext('webgl2');

            if (!gl2) {
                return { detected: false, score: 0, detections: ['no_webgl2'] };
            }

            const debugInfo = gl2.getExtension('WEBGL_debug_renderer_info');
            if (debugInfo) {
                const renderer = gl2.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                
                if (/swiftshader|llvmpipe|mesa|virtual|software/i.test(renderer || '')) {
                    score += 35;
                    detections.push('webgl2_software_renderer');
                }
                
                if (/vmware|virtualbox|parallels|qemu/i.test(renderer || '')) {
                    score += 55;
                    detections.push('webgl2_vm_renderer');
                }
            }

            const maxTexSize = gl2.getParameter(gl2.MAX_TEXTURE_SIZE);
            if (maxTexSize <= 2048) {
                score += 20;
                detections.push('webgl2_small_texture');
            }

            const supportedExts = gl2.getSupportedExtensions();
            if (supportedExts && supportedExts.length < 8) {
                score += 20;
                detections.push('few_webgl2_extensions:' + supportedExts.length);
            }

            const max3DTextureSize = gl2.getParameter(gl2.MAX_3D_TEXTURE_SIZE);
            if (max3DTextureSize < 512) {
                score += 25;
                detections.push('webgl2_limited_3d_texture');
            }

            const maxRenderbufferSize = gl2.getParameter(gl2.MAX_RENDERBUFFER_SIZE);
            if (maxRenderbufferSize < 4096) {
                score += 20;
                detections.push('webgl2_small_renderbuffer');
            }

            const maxComputeWorkGroupSize = gl2.getParameter(gl2.MAX_COMPUTE_WORK_GROUP_SIZE);
            if (maxComputeWorkGroupSize && maxComputeWorkGroupSize[0] < 256) {
                score += 15;
                detections.push('webgl2_limited_compute');
            }

        } catch (e) {
            score += 30;
            detections.push('webgl2_error: ' + e.message);
        }

        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectWebGLVendor() {
        let score = 0;
        const detections = [];

        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');

            if (!gl) {
                score += 30;
                detections.push('no_webgl_for_vendor_check');
                return { detected: score > 30, score: Math.min(score, 100), detections };
            }

            const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
            if (!debugInfo) {
                score += 25;
                detections.push('webgl_vendor_info_unavailable');
                return { detected: score > 30, score: Math.min(score, 100), detections };
            }

            const vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
            const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);

            const knownVendors = ['NVIDIA', 'AMD', 'Intel', 'Apple', 'ARM', 'Qualcomm', 'Mali'];
            const isKnownVendor = knownVendors.some(v => vendor && vendor.includes(v));

            if (!isKnownVendor && vendor) {
                score += 35;
                detections.push('unknown_gpu_vendor:' + vendor.substring(0, 30));
            }

            const suspiciousPatterns = [
                { pattern: /google\s*inc/i, score: 35, name: 'google_gpu' },
                { pattern: /chromium/i, score: 30, name: 'chromium_gpu' },
                { pattern: /virtual/i, score: 40, name: 'virtual_gpu' },
                { pattern: /software/i, score: 35, name: 'software_gpu' },
                { pattern: /unknown/i, score: 30, name: 'unknown_gpu' },
                { pattern: /swiftshader/i, score: 45, name: 'swiftshader' },
                { pattern: /llvmpipe/i, score: 40, name: 'llvmpipe' },
                { pattern: /mesa/i, score: 35, name: 'mesa_software' }
            ];

            for (const { pattern, score: patScore, name } of suspiciousPatterns) {
                if (pattern.test(vendor || '') || pattern.test(renderer || '')) {
                    score += patScore;
                    detections.push(name);
                }
            }

        } catch (e) {
            score += 25;
            detections.push('webgl_vendor_error: ' + e.message);
        }

        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectAudioFingerprint() {
        let score = 0;
        const detections = [];

        try {
            const AudioContext = window.OfflineAudioContext || window.webkitOfflineAudioContext;

            if (!AudioContext) {
                score += 40;
                detections.push('no_audiocontext');
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

            const gain = ctx.createGain();
            gain.gain.setValueAtTime(0.5, ctx.currentTime);

            const filter = ctx.createBiquadFilter();
            filter.type = 'lowpass';
            filter.frequency.setValueAtTime(5000, ctx.currentTime);

            osc.connect(filter);
            filter.connect(compressor);
            compressor.connect(gain);
            gain.connect(ctx.destination);
            osc.start(0);

            const startTime = performance.now();
            const buffer = await ctx.startRendering();
            const renderTime = performance.now() - startTime;

            if (renderTime < 3) {
                score += 30;
                detections.push('audio_render_too_fast');
            } else if (renderTime > 2000) {
                score += 25;
                detections.push('audio_render_too_slow');
            }

            const channelData = buffer.getChannelData(0);
            let sumAbs = 0;
            let sumSq = 0;

            for (let i = 4500; i < 5000; i++) {
                sumAbs += Math.abs(channelData[i]);
            }
            for (let i = 0; i < channelData.length; i++) {
                sumSq += channelData[i] * channelData[i];
            }

            if (sumAbs === 0 && sumSq === 0) {
                score += 40;
                detections.push('audio_silent');
            }

            let uniqueValues = new Set();
            for (let i = 0; i < Math.min(2000, channelData.length); i++) {
                uniqueValues.add(channelData[i].toFixed(8));
            }
            if (uniqueValues.size < 50) {
                score += 35;
                detections.push('audio_low_entropy');
            }

            const hashData = channelData.slice(4500, 5500).map(v => v.toFixed(6)).join(',');
            this.fingerprintComponents.audio = this.sha256Hash(hashData).substring(0, 32);

        } catch (e) {
            score += 40;
            detections.push('audio_error: ' + e.message);
        }

        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectFontsEnhanced() {
        let score = 0;
        const detections = [];

        const testFonts = [
            'Arial', 'Helvetica', 'Times New Roman', 'Courier New', 'Verdana', 'Georgia',
            'Palatino', 'Garamond', 'Impact', 'Comic Sans MS', 'Trebuchet MS', 'Lucida Console',
            'Tahoma', 'Segoe UI', 'Roboto', 'Open Sans', 'Lato', 'Montserrat',
            'Source Sans Pro', 'Raleway', 'Ubuntu', 'Noto Sans', 'Droid Sans', 'Fira Sans',
            'Merriweather', 'Playfair Display', 'PT Sans', 'Nunito', 'Quicksand',
            'Work Sans', 'Oswald', 'Roboto Condensed', 'Noto Serif', 'Lora',
            'IBM Plex Sans', 'JetBrains Mono', 'SF Pro Display', 'SF Pro Text',
            'Calibri', 'Candara', 'Corbel', 'Cambria', 'Bookman', 'Futura', 'Optima',
            'Century Gothic', 'Consolas', 'Monaco', 'Menlo'
        ];

        try {
            const el = document.createElement('div');
            el.style.cssText = 'position:absolute;left:-9999px;font-size:72px;visibility:hidden;white-space:nowrap';
            el.textContent = 'mmmmmmmmmmlli';
            document.body.appendChild(el);

            const baseFonts = ['monospace', 'sans-serif', 'serif'];
            const baseWidths = {};
            for (const base of baseFonts) {
                el.style.fontFamily = base;
                baseWidths[base] = el.offsetWidth;
            }

            let fontCount = 0;
            const detectedFonts = [];

            for (const font of testFonts) {
                for (const base of baseFonts) {
                    el.style.fontFamily = `"${font}", ${base}`;
                    if (el.offsetWidth !== baseWidths[base]) {
                        fontCount++;
                        detectedFonts.push(font);
                        break;
                    }
                }
            }

            document.body.removeChild(el);

            if (fontCount < 3) {
                score += 35;
                detections.push('too_few_fonts:' + fontCount);
            } else if (fontCount < 8) {
                score += 20;
                detections.push('limited_fonts:' + fontCount);
            }

            if (detectedFonts.length === 0) {
                score += 50;
                detections.push('no_detected_fonts');
            }

            const uncommonFonts = detectedFonts.filter(f => 
                !['Arial', 'Helvetica', 'Times New Roman', 'Courier New', 'Georgia', 'Verdana'].includes(f)
            );
            if (uncommonFonts.length === 0 && fontCount > 0) {
                score += 25;
                detections.push('only_basic_fonts');
            }

            this.fingerprintComponents.fonts = detectedFonts.length;

        } catch (e) {
            score += 35;
            detections.push('font_detection_error: ' + e.message);
        }

        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectPluginsFingerprint() {
        let score = 0;
        const detections = [];

        try {
            const plugins = navigator.plugins;
            
            if (!plugins || plugins.length === 0) {
                score += 35;
                detections.push('no_plugins');
            } else {
                const pluginNames = Array.from(plugins).map(p => p.name);
                const hasPDF = pluginNames.some(p => 
                    p.includes('PDF') || p.includes('pdf')
                );
                
                if (!hasPDF) {
                    score += 20;
                    detections.push('no_pdf_plugin');
                }

                if (plugins.length < 2) {
                    score += 15;
                    detections.push('very_few_plugins:' + plugins.length);
                }

                this.fingerprintComponents.plugins = plugins.length;
            }

        } catch (e) {
            score += 30;
            detections.push('plugins_error: ' + e.message);
        }

        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectWebRTCEnhanced() {
        let score = 0;
        const detections = [];
        const detectedIPs = new Set();

        try {
            const RTCPeerConnection = window.RTCPeerConnection || window.webkitRTCPeerConnection;
            
            if (!RTCPeerConnection) {
                score += 20;
                detections.push('no_webrtc');
                return { detected: score > 25, score: Math.min(score, 100), detections };
            }

            const pc = new RTCPeerConnection({
                iceServers: [
                    { urls: 'stun:stun.l.google.com:19302' },
                    { urls: 'stun:stun1.l.google.com:19302' },
                    { urls: 'stun:stun2.l.google.com:19302' },
                    { urls: 'stun:stun3.l.google.com:19302' },
                    { urls: 'stun:stun4.l.google.com:19302' }
                ]
            });

            pc.createDataChannel('');

            const offer = await pc.createOffer();
            await pc.setLocalDescription(offer);

            const sdp = pc.localDescription.sdp;
            const lines = sdp.split('\n');

            for (const line of lines) {
                if (line.indexOf('candidate') > -1) {
                    const parts = line.split(' ');
                    if (parts[4] && parts[4] !== '0.0.0.0') {
                        const ip = parts[4];
                        detectedIPs.add(ip);
                        
                        if (parts[7] === 'relay') {
                            score += 40;
                            detections.push('relay_ip_detected:' + ip);
                        } else if (parts[7] === 'srflx') {
                            score += 25;
                            detections.push('server_reflexive_ip:' + ip);
                        }
                    }
                }
            }

            pc.close();

            const ipArray = Array.from(detectedIPs);
            this.fingerprintComponents.webrtcIPs = ipArray.length;

            if (ipArray.length > 2) {
                score += 30;
                detections.push('multiple_ips_detected:' + ipArray.length);
            }

            const publicIPs = ipArray.filter(ip => 
                !ip.startsWith('10.') && 
                !ip.startsWith('172.16.') && 
                !ip.startsWith('172.17.') && 
                !ip.startsWith('172.18.') && 
                !ip.startsWith('172.19.') && 
                !ip.startsWith('172.20.') && 
                !ip.startsWith('172.21.') && 
                !ip.startsWith('172.22.') && 
                !ip.startsWith('172.23.') && 
                !ip.startsWith('172.24.') && 
                !ip.startsWith('172.25.') && 
                !ip.startsWith('172.26.') && 
                !ip.startsWith('172.27.') && 
                !ip.startsWith('172.28.') && 
                !ip.startsWith('172.29.') && 
                !ip.startsWith('172.30.') && 
                !ip.startsWith('172.31.') && 
                !ip.startsWith('192.168.') &&
                !ip.startsWith('127.')
            );

            if (publicIPs.length > 0) {
                detections.push('public_ip_count:' + publicIPs.length);
                if (publicIPs.length > 1) {
                    score += 35;
                    detections.push('multiple_public_ips');
                }
            }

        } catch (e) {
            score += 25;
            detections.push('webrtc_error: ' + e.message);
        }

        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectWebRTCLeak() {
        let score = 0;
        const detections = [];

        try {
            const RTCPeerConnection = window.RTCPeerConnection || window.webkitRTCPeerConnection;
            
            if (!RTCPeerConnection) {
                return { detected: false, score: 0, detections: ['no_webrtc'] };
            }

            const pc = new RTCPeerConnection({
                iceServers: [
                    { urls: 'stun:stun.l.google.com:19302' },
                    { urls: 'turn:turn.example.com:3478', credential: 'test', username: 'test' }
                ]
            });

            let hasLocalIP = false;
            let hasPublicIP = false;

            pc.onicecandidate = (event) => {
                if (event.candidate) {
                    const ip = event.candidate.candidate.split(' ')[4];
                    if (ip) {
                        if (ip.startsWith('10.') || ip.startsWith('192.168.') || ip.startsWith('172.')) {
                            hasLocalIP = true;
                        } else if (!ip.startsWith('127.')) {
                            hasPublicIP = true;
                        }
                    }
                }
            };

            pc.createDataChannel('');
            const offer = await pc.createOffer();
            await pc.setLocalDescription(offer);

            await new Promise(resolve => setTimeout(resolve, 1000));
            pc.close();

            if (hasPublicIP && hasLocalIP) {
                score += 45;
                detections.push('webrtc_ip_leak');
            } else if (hasPublicIP) {
                score += 30;
                detections.push('webrtc_public_ip_only');
            }

        } catch (e) {
            score += 20;
            detections.push('webrtc_leak_error: ' + e.message);
        }

        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectProxyVPN() {
        let score = 0;
        const detections = [];

        try {
            const connection = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
            
            if (connection) {
                if (connection.type === 'vpn') {
                    score += 55;
                    detections.push('vpn_detected_by_api');
                }
                if (connection.type === 'proxy') {
                    score += 50;
                    detections.push('proxy_detected_by_api');
                }
            }

            const webrtcResult = await this.detectWebRTCEnhanced();
            if (webrtcResult.detected) {
                score += webrtcResult.score * 0.5;
                detections.push(...webrtcResult.detections.filter(d => d.includes('relay') || d.includes('public')));
            }

            const ua = navigator.userAgent || '';
            const proxyPatterns = [
                { pattern: /proxy/i, score: 35, name: 'proxy_in_ua' },
                { pattern: /vpn/i, score: 40, name: 'vpn_in_ua' },
                { pattern: /tor/i, score: 50, name: 'tor_in_ua' },
                { pattern: /anonymous/i, score: 30, name: 'anonymous_in_ua' },
                { pattern: /hidemyass/i, score: 50, name: 'hidemyass' },
                { pattern: /nordvpn/i, score: 50, name: 'nordvpn' },
                { pattern: /expressvpn/i, score: 50, name: 'expressvpn' },
                { pattern: /surfshark/i, score: 50, name: 'surfshark' },
                { pattern: /cyberghost/i, score: 50, name: 'cyberghost' },
                { pattern: /privateinternetaccess/i, score: 50, name: 'pia' },
                { pattern: /windscribe/i, score: 50, name: 'windscribe' },
                { pattern: /protonvpn/i, score: 50, name: 'protonvpn' }
            ];

            for (const { pattern, score: patScore, name } of proxyPatterns) {
                if (pattern.test(ua)) {
                    score += patScore;
                    detections.push(name);
                }
            }

            try {
                const response = await fetch('https://api.ipify.org?format=json', { 
                    method: 'GET',
                    mode: 'no-cors'
                }).catch(() => null);
                
                if (!response) {
                    score += 15;
                    detections.push('external_ip_check_blocked');
                }
            } catch (e) {
                score += 10;
                detections.push('ip_check_error');
            }

            try {
                const startTime = performance.now();
                await fetch('/.well-known/acme-challenge/test', { method: 'HEAD' }).catch(() => null);
                const latency = performance.now() - startTime;
                
                if (latency > 3000) {
                    score += 25;
                    detections.push('high_latency_indicates_proxy');
                }
            } catch (e) {}

        } catch (e) {
            score += 30;
            detections.push('proxy_vpn_error: ' + e.message);
        }

        return { detected: score > 40, score: Math.min(score, 100), detections };
    }

    async detectVPNIndicators() {
        let score = 0;
        const detections = [];

        try {
            const timezones = [
                'America/New_York', 'America/Los_Angeles', 'Europe/London', 
                'Europe/Paris', 'Asia/Tokyo', 'Asia/Shanghai', 'Australia/Sydney'
            ];
            
            const userTZ = Intl.DateTimeFormat().resolvedOptions().timeZone;
            const commonTZ = timezones.includes(userTZ);

            if (!commonTZ) {
                score += 20;
                detections.push('uncommon_timezone:' + userTZ);
            }

            const language = navigator.language || '';
            const timezoneLanguageMap = {
                'America/New_York': ['en-US', 'es-US'],
                'America/Los_Angeles': ['en-US', 'es-US'],
                'Europe/London': ['en-GB'],
                'Europe/Paris': ['fr-FR'],
                'Asia/Tokyo': ['ja-JP'],
                'Asia/Shanghai': ['zh-CN']
            };

            if (timezoneLanguageMap[userTZ]) {
                const expectedLangs = timezoneLanguageMap[userTZ];
                if (!expectedLangs.some(l => language.startsWith(l.split('-')[0]))) {
                    score += 25;
                    detections.push('language_timezone_mismatch');
                }
            }

            const connection = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
            if (connection && connection.saveData === true) {
                score += 15;
                detections.push('data_saver_may_indicate_vpn');
            }

            const platform = navigator.platform || '';
            const userAgent = navigator.userAgent || '';

            if (platform.includes('Win') && userAgent.includes('Linux')) {
                score += 30;
                detections.push('platform_ua_mismatch');
            }

            if (navigator.hardwareConcurrency && navigator.hardwareConcurrency <= 2) {
                score += 20;
                detections.push('low_hardware_concurrency_vpn');
            }

        } catch (e) {
            score += 25;
            detections.push('vpn_indicators_error: ' + e.message);
        }

        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectTorExitNode() {
        let score = 0;
        const detections = [];

        try {
            const ua = navigator.userAgent || '';
            
            if (/tor|torrent/i.test(ua)) {
                score += 55;
                detections.push('tor_in_user_agent');
            }

            const knownTorExitPorts = [9001, 9030, 9040, 9050, 9051, 9150];
            
            try {
                const startTime = performance.now();
                await fetch(`http://localhost:${knownTorExitPorts[0]}`, { 
                    method: 'HEAD',
                    timeout: 500
                }).catch(() => null);
                const latency = performance.now() - startTime;
                
                if (latency < 100) {
                    score += 35;
                    detections.push('local_tor_port_open');
                }
            } catch (e) {}

            try {
                const webrtcResult = await this.detectWebRTCEnhanced();
                const hasRelayIP = webrtcResult.detections.some(d => d.includes('relay'));
                if (hasRelayIP) {
                    score += 25;
                    detections.push('tor_like_relay_pattern');
                }
            } catch (e) {}

            const hostname = window.location.hostname;
            if (hostname.includes('.onion')) {
                score += 60;
                detections.push('tor_onion_domain');
            }

        } catch (e) {
            score += 25;
            detections.push('tor_detection_error: ' + e.message);
        }

        return { detected: score > 40, score: Math.min(score, 100), detections };
    }

    async detectNetworkAnomalies() {
        let score = 0;
        const detections = [];

        try {
            const connection = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
            
            if (connection) {
                if (connection.rtt > 500) {
                    score += 20;
                    detections.push('high_rtt:' + connection.rtt);
                }
                
                if (connection.downlink && connection.downlink < 0.5) {
                    score += 15;
                    detections.push('very_slow_connection');
                }
            }

            const onlineStatus = navigator.onLine;
            if (!onlineStatus) {
                score += 25;
                detections.push('browser_offline');
            }

            const cookieEnabled = navigator.cookieEnabled;
            if (!cookieEnabled) {
                score += 15;
                detections.push('cookies_disabled');
            }

            const doNotTrack = navigator.doNotTrack;
            if (doNotTrack === '1' || doNotTrack === 'yes') {
                score += 10;
                detections.push('do_not_track_enabled');
            }

            try {
                const pingStart = performance.now();
                await fetch('/favicon.ico', { method: 'HEAD', cache: 'no-cache' });
                const pingDuration = performance.now() - pingStart;

                if (pingDuration < 5) {
                    score += 25;
                    detections.push('unrealistically_fast_latency');
                } else if (pingDuration > 5000) {
                    score += 20;
                    detections.push('extremely_high_latency');
                }
            } catch (e) {
                score += 15;
                detections.push('network_ping_error');
            }

        } catch (e) {
            score += 20;
            detections.push('network_anomalies_error: ' + e.message);
        }

        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectHeadless() {
        let score = 0;
        const detections = [];

        try {
            if (navigator.webdriver === true) {
                score += 40;
                detections.push('navigator_webdriver_true');
            } else if (navigator.webdriver === undefined) {
                score += 25;
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
                detections.push('chrome_runtime_missing');
            }

            const mimeTypes = navigator.mimeTypes;
            if (mimeTypes && mimeTypes.length === 0) {
                score += 25;
                detections.push('no_mimetypes');
            }

            const ua = navigator.userAgent || '';
            if (/headless|phantom/i.test(ua)) {
                score += 50;
                detections.push('headless_in_ua');
            }

            if (window.outerHeight === 0 && window.outerWidth === 0) {
                score += 35;
                detections.push('zero_window_size');
            }

            if (window.devicePixelRatio === 0) {
                score += 30;
                detections.push('zero_device_pixel_ratio');
            }

            if (typeof navigator.maxTouchPoints === 'number' && navigator.maxTouchPoints === 0) {
                if (/mobile|android|iphone/i.test(ua)) {
                    score += 40;
                    detections.push('touch_discrepancy_on_mobile');
                }
            }

            const gl = document.createElement('canvas').getContext('webgl');
            if (gl) {
                const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                if (!debugInfo) {
                    score += 25;
                    detections.push('webgl_debug_blocked');
                }
            }

        } catch (e) {
            score += 35;
            detections.push('headless_detection_error: ' + e.message);
        }

        return { detected: score > 40, score: Math.min(score, 100), detections };
    }

    async detectAdvancedHeadless() {
        let score = 0;
        const detections = [];

        try {
            const chromeFeatures = [
                window.chrome && window.chrome.runtime,
                window.chrome && window.chrome.app,
                window.chrome && window.chrome.loadTimes,
                window.chrome && window.chrome.csi
            ].filter(Boolean).length;

            if (chromeFeatures < 2) {
                score += 35;
                detections.push('chrome_features_missing:' + chromeFeatures);
            }

            const canvas = document.createElement('canvas');
            const ctx = canvas.getContext('2d');
            if (ctx) {
                ctx.fillStyle = 'rgba(255, 128, 0, 0.5)';
                ctx.fillRect(0, 0, 10, 10);
                const imgData = ctx.getImageData(0, 0, 10, 10);
                const hasContent = Array.from(imgData.data).some(v => v > 0);
                if (!hasContent) {
                    score += 30;
                    detections.push('canvas_no_content');
                }
            }

            if (window.outerWidth > window.innerWidth || window.outerHeight > window.innerHeight) {
                score += 25;
                detections.push('outer_larger_than_inner');
            }

            const testEl = document.createElement('div');
            testEl.style.cssText = 'position:absolute;left:-9999px';
            document.body.appendChild(testEl);
            const computedStyle = window.getComputedStyle(testEl);
            if (computedStyle.position !== 'absolute') {
                score += 20;
                detections.push('css_computestyle_mismatch');
            }
            document.body.removeChild(testEl);

            try {
                const perfData = performance.getEntriesByType('navigation');
                if (perfData.length > 0) {
                    const nav = perfData[0];
                    if (nav.domainLookupStart === 0 && nav.domainLookupEnd === 0 && nav.connectStart === 0) {
                        score += 25;
                        detections.push('no_network_timing_data');
                    }
                }
            } catch (e) {}

            const plugins = navigator.plugins;
            if (!plugins || plugins.length === 0) {
                score += 20;
                detections.push('no_plugins_advanced');
            }

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
            const ua = navigator.userAgent.toLowerCase();
            
            if (/headless|phantom|puppet/i.test(ua)) {
                score += 45;
                detections.push('stealth_ua_marker');
            }

            const gl = document.createElement('canvas').getContext('webgl');
            if (gl) {
                const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                if (debugInfo) {
                    const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                    if (/swiftshader|llvmpipe|software/i.test(renderer || '')) {
                        score += 35;
                        detections.push('stealth_software_renderer');
                    }

                    if (ua.includes('chrome') && (!renderer || !renderer.includes('NVIDIA') && !renderer.includes('AMD') && !renderer.includes('Intel'))) {
                        score += 30;
                        detections.push('stealth_renderer_mismatch');
                    }
                }
            }

            const hardwareConcurrency = navigator.hardwareConcurrency;
            if (hardwareConcurrency === 1 || hardwareConcurrency > 128) {
                score += 25;
                detections.push('stealth_unusual_hardware:' + hardwareConcurrency);
            }

            const deviceMemory = navigator.deviceMemory;
            if (deviceMemory && (deviceMemory < 0.5 || deviceMemory > 128)) {
                score += 20;
                detections.push('stealth_unusual_memory:' + deviceMemory);
            }

            const testCanvas = document.createElement('canvas');
            const ctx = testCanvas.getContext('2d');
            if (ctx) {
                ctx.fillStyle = '#f60ca';
                ctx.fillRect(1, 1, 62, 20);
                const dataURL = testCanvas.toDataURL();
                if (!dataURL.includes('data:image/png')) {
                    score += 25;
                    detections.push('stealth_canvas_encoding');
                }
            }

        } catch (e) {
            score += 25;
            detections.push('stealth_mode_error: ' + e.message);
        }

        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectWebDriver() {
        let score = 0;
        const detections = [];

        const wdProps = [
            'webdriver', '__webdriver_evaluate', '__selenium_evaluate',
            '__webdriver_script_fn', '__driver_evaluate', '__fxdriver_evaluate',
            '__webdriver_unwrapped', '__lastWatirAlert', '__$webdriverAsyncExecutor',
            'callSelenium', '__selenium', 'Selenium', '_selenium',
            'document.__selenium', ' Selenium', '__webdriver_script_func'
        ];

        for (const prop of wdProps) {
            if (window[prop] !== undefined) {
                score += 20;
                detections.push(prop);
            }
        }

        try {
            if (navigator.webdriver === true) {
                score += 40;
                detections.push('navigator.webdriver_true');
            }
        } catch (e) {}

        try {
            const el = document.createElement('div');
            el.setAttribute('onclick', 'return __webdriver_script_fn()');
            if (el.onclick !== null) {
                score += 15;
                detections.push('webdriver_script_fn_detected');
            }
        } catch (e) {}

        try {
            const keys = Object.keys(window);
            const seleniumKeys = keys.filter(k => /selenium|webdriver|__wd|__sel/i.test(k));
            if (seleniumKeys.length > 2) {
                score += 30;
                detections.push('multiple_webdriver_keys:' + seleniumKeys.length);
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
                detections.push('puppeteer_webdriver');
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
                detections.push('puppeteer_async_script');
            }
        } catch (e) {}

        try {
            if (document.__webdriver_evaluate) {
                score += 45;
                detections.push('puppeteer_webdriver_eval');
            }
        } catch (e) {}

        try {
            const userAgent = navigator.userAgent || '';
            if (/headless/i.test(userAgent)) {
                score += 40;
                detections.push('puppeteer_headless_ua');
            }
            if (/puppet/i.test(userAgent)) {
                score += 55;
                detections.push('puppeteer_ua');
            }
            if (/Chrome\/[\d.]+\s+Headless/i.test(userAgent)) {
                score += 60;
                detections.push('chrome_headless_ua');
            }
        } catch (e) {}

        try {
            if (window._puppeteer_globals !== undefined) {
                score += 45;
                detections.push('puppeteer_globals');
            }
        } catch (e) {}

        try {
            const canvas = document.createElement('canvas');
            canvas.width = 200;
            canvas.height = 100;
            const ctx = canvas.getContext('2d');
            if (ctx) {
                ctx.textBaseline = 'top';
                ctx.font = '14px Arial';
                ctx.fillStyle = '#f60';
                ctx.fillText('Puppeteer', 2, 2);
                const dataURL = canvas.toDataURL();
                if (dataURL.includes('puppeteer') || dataURL.includes('Headless')) {
                    score += 50;
                    detections.push('puppeteer_canvas_signature');
                }
            }
        } catch (e) {}

        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectPlaywright() {
        let score = 0;
        const detections = [];

        try {
            const playwrightGlobals = [
                window.__playwright__,
                window.__pw_tags,
                window.__pw_resume__,
                window.__pw_connect__,
                window.__playwright_unstripped__
            ].filter(Boolean);

            if (playwrightGlobals.length > 0) {
                score += 60;
                detections.push('playwright_globals_detected');
            }
        } catch (e) {}

        try {
            const el = document.createElement('div');
            el.setAttribute('onfocus', 'return __pw_resume__()');
            if (el.onfocus !== null) {
                score += 45;
                detections.push('playwright_onfocus');
            }
        } catch (e) {}

        try {
            const ua = navigator.userAgent || '';
            if (/playwright/i.test(ua)) {
                score += 65;
                detections.push('playwright_ua');
            }
        } catch (e) {}

        try {
            if (navigator.plugins.length === 0 && navigator.mimeTypes.length === 0) {
                score += 50;
                detections.push('playwright_no_plugins_mime');
            }
        } catch (e) {}

        try {
            const langs = navigator.languages;
            if (langs && langs.length === 1 && langs[0] === 'en-US') {
                score += 35;
                detections.push('playwright_default_language');
            }
        } catch (e) {}

        return { detected: score > 40, score: Math.min(score, 100), detections };
    }

    async detectSelenium() {
        let score = 0;
        const detections = [];

        const selProps = [
            'selenium', '_selenium', 'callSelenium', '__selenium',
            'document__selenium', 'Selenium', '__webdriver_script_fn',
            'Selenium.prototype', '_selenium_evaluate', 'selenium_evaluate',
            '__selenium_script_fn', '__selenium_script_func'
        ];

        for (const prop of selProps) {
            if (window[prop] !== undefined || document[prop] !== undefined) {
                score += 25;
                detections.push(prop);
            }
        }

        try {
            if (document.documentElement.getAttribute('webdriver') !== null) {
                score += 40;
                detections.push('selenium_webdriver_attr');
            }
        } catch (e) {}

        try {
            const el = document.createElement('div');
            el.setAttribute('onmouseover', 'return Selenium.prototype.whatever');
            if (el.onmouseover !== null) {
                score += 25;
                detections.push('selenium_prototype');
            }
        } catch (e) {}

        try {
            const ua = navigator.userAgent || '';
            if (/selenium/i.test(ua)) {
                score += 55;
                detections.push('selenium_ua');
            }
        } catch (e) {}

        try {
            const keys = Object.keys(window);
            const seleniumRelated = keys.filter(k => /selenium|webdriver|__sel/i.test(k));
            if (seleniumRelated.length > 3) {
                score += 50;
                detections.push('selenium_multiple_markers:' + seleniumRelated.length);
            }
        } catch (e) {}

        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectAutomationFramework() {
        let score = 0;
        const detections = [];

        try {
            const automationGlobals = [
                '__webdriver', '__driver', '__fxdriver', '__selenium',
                '__webdriver_func', '__selenium_func', '__driver_func',
                '__webdriver_script_func', '__selenium_script_func',
                '__webdriver_unwrapped', '__fxdriver_unwrapped',
                '__driver_unwrapped', '__selenium_unwrapped'
            ];

            for (const global of automationGlobals) {
                if (window[global] !== undefined) {
                    score += 40;
                    detections.push('automation_global:' + global);
                }
            }
        } catch (e) {}

        try {
            const automationPatterns = [
                { name: 'cypress', pattern: /cypress/i, window: 'Cypress' },
                { name: 'nightmare', pattern: /nightmare/i, window: 'Nightmare' },
                { name: 'testcafe', pattern: /testcafe/i, window: '__TESTCAFE' },
                { name: 'webdriverio', pattern: /webdriver/i, window: 'WebDriver' },
                { name: 'wdio', pattern: /wdio/i, window: '_WDIO' },
                { name: 'splinter', pattern: /splinter/i, window: 'Splinter' },
                { name: 'mechanize', pattern: /mechanize/i, window: 'Mechanize' },
                { name: 'testem', pattern: /testem/i, window: 'Testem' },
                { name: 'karma', pattern: /karma/i, window: '__karma__' }
            ];

            for (const ap of automationPatterns) {
                if (window[ap.window] !== undefined) {
                    score += 55;
                    detections.push('framework_window:' + ap.name);
                }
                if (ap.pattern.test(navigator.userAgent || '')) {
                    score += 50;
                    detections.push('framework_ua:' + ap.name);
                }
            }
        } catch (e) {}

        try {
            const keys = Object.keys(window);
            const automationKeys = keys.filter(k => 
                /^(?:__)?(?:selenium|webdriver|driver|wdio|cypress|pw_|playwright_|__pw)/i.test(k) ||
                /_(?:selenium|webdriver|driver|pw_|playwright_)(?:_)?/i.test(k)
            );

            if (automationKeys.length > 3) {
                score += 45;
                detections.push('automation_keys_multiple:' + automationKeys.length);
            }
        } catch (e) {}

        try {
            const ua = navigator.userAgent || '';
            if (/automation|bot|crawler|spider|scraper/i.test(ua)) {
                score += 65;
                detections.push('automation_ua_detected');
            }
        } catch (e) {}

        try {
            const startTime = performance.now();
            await new Promise(resolve => setTimeout(resolve, 0));
            const elapsed = performance.now() - startTime;
            if (elapsed < 0.3) {
                score += 35;
                detections.push('unrealistic_zero_delay');
            }
        } catch (e) {}

        return { detected: score > 40, score: Math.min(score, 100), detections };
    }

    async detectEmulator() {
        let score = 0;
        const detections = [];

        try {
            const ua = navigator.userAgent || '';
            const emulatorPatterns = [
                { pattern: /android.*emulator/i, name: 'android_emulator_ua' },
                { pattern: /iphone.*simulator|ipad.*simulator/i, name: 'ios_simulator_ua' },
                { pattern: /bluestacks/i, name: 'bluestacks' },
                { pattern: /nox|genymotion/i, name: 'genymotion' },
                { pattern: /memu/i, name: 'memu' },
                { pattern: /ldplayer/i, name: 'ldplayer' },
                { pattern: /koplayer/i, name: 'koplayer' },
                { pattern: /droid4x/i, name: 'droid4x' },
                { pattern: /mitsumeshi/i, name: 'mitsumeshi' },
                { pattern: /x86.*android/i, name: 'x86_android' },
                { pattern: /qemu/i, name: 'qemu_emulator' },
                { pattern: /virtualbox/i, name: 'virtualbox_emulator' },
                { pattern: /vmware/i, name: 'vmware_emulator' },
                { pattern: /parallels/i, name: 'parallels_emulator' }
            ];

            for (const emu of emulatorPatterns) {
                if (emu.pattern.test(ua)) {
                    score += 55;
                    detections.push(emu.name);
                }
            }

            if (/iPhone.*CPU.*OS/i.test(ua)) {
                const platform = navigator.platform || '';
                if (!/iPhone|iPad/i.test(platform)) {
                    score += 50;
                    detections.push('ios_platform_mismatch');
                }
            }

            if (navigator.maxTouchPoints === 0) {
                if (/mobile|android|iphone|ipad/i.test(ua)) {
                    score += 45;
                    detections.push('mobile_no_touch');
                }
            }

            if (navigator.platform) {
                const platform = navigator.platform.toLowerCase();
                if (platform.includes('linux') && /mobile|android/i.test(ua)) {
                    score += 50;
                    detections.push('linux_android_emulator');
                }
            }

            try {
                const gl = document.createElement('canvas').getContext('webgl');
                if (gl) {
                    const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                    if (debugInfo) {
                        const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) || '';
                        if (/android|adr|goldfish|llvmpipe|swiftshader/i.test(renderer)) {
                            score += 45;
                            detections.push('emulator_webgl_renderer:' + renderer.substring(0, 30));
                        }
                    }
                }
            } catch (e) {}

            try {
                const screenWidth = screen.width;
                const screenHeight = screen.height;
                const knownEmulatorResolutions = [
                    { w: 320, h: 480 },
                    { w: 360, h: 640 },
                    { w: 375, h: 667 },
                    { w: 414, h: 896 },
                    { w: 768, h: 1024 }
                ];

                const isKnownResolution = knownEmulatorResolutions.some(r => 
                    r.w === screenWidth && r.h === screenHeight
                );

                if (isKnownResolution) {
                    score += 35;
                    detections.push('known_emulator_resolution:' + screenWidth + 'x' + screenHeight);
                }
            } catch (e) {}

            if (screen.width === screen.availWidth && screen.height === screen.availHeight) {
                score += 30;
                detections.push('fullscreen_emulator_indicator');
            }

        } catch (e) {
            score += 30;
            detections.push('emulator_detection_error: ' + e.message);
        }

        return { detected: score > 40, score: Math.min(score, 100), detections };
    }

    async detectVirtualization() {
        let score = 0;
        const detections = [];

        try {
            const ua = navigator.userAgent || '';
            const vmPatterns = [
                { pattern: /vmware|virtualbox|parallels/i, name: 'vm_detected' },
                { pattern: /xen/i, name: 'xen_vm' },
                { pattern: /hyperv/i, name: 'hyperv_vm' },
                { pattern: /qemu|kvm/i, name: 'qemu_kvm' },
                { pattern: /virtual/i, name: 'virtual_generic' },
                { pattern: /openvz/i, name: 'openvz' },
                { pattern: /container/i, name: 'container_marker' },
                { pattern: /docker/i, name: 'docker_virtualization' },
                { pattern: /kubernetes|k8s/i, name: 'kubernetes' },
                { pattern: /lxc/i, name: 'lxc_container' },
                { pattern: /proxmox/i, name: 'proxmox' },
                { pattern: /esxi/i, name: 'esxi' }
            ];

            for (const vm of vmPatterns) {
                if (vm.pattern.test(ua)) {
                    score += 50;
                    detections.push(vm.name);
                }
            }

            try {
                const canvas = document.createElement('canvas');
                const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
                if (gl) {
                    const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                    if (debugInfo) {
                        const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) || '';
                        const vmRendererPatterns = [
                            { pattern: /vmware|virtualbox|parallels/i, name: 'vm_renderer' },
                            { pattern: /swiftshader|llvmpipe|mesa|software/i, name: 'software_renderer_vm' },
                            { pattern: /virtual|gpu|graphics/i, name: 'virtual_gpu' },
                            { pattern: /qxl|cirrus|bochs/i, name: 'virtual_display' }
                        ];

                        for (const r of vmRendererPatterns) {
                            if (r.pattern.test(renderer)) {
                                score += 55;
                                detections.push(r.name);
                            }
                        }
                    }
                }
            } catch (e) {}

            if (navigator.hardwareConcurrency) {
                if (navigator.hardwareConcurrency === 1) {
                    score += 35;
                    detections.push('single_core_vm');
                } else if (navigator.hardwareConcurrency === 2) {
                    score += 25;
                    detections.push('dual_core_vm');
                } else if (navigator.hardwareConcurrency > 64) {
                    score += 20;
                    detections.push('high_core_count_vm');
                }
            }

            if (navigator.deviceMemory) {
                if (navigator.deviceMemory <= 0.5) {
                    score += 30;
                    detections.push('minimal_memory_vm');
                } else if (navigator.deviceMemory > 128) {
                    score += 20;
                    detections.push('excessive_memory_vm');
                }
            }

        } catch (e) {
            score += 35;
            detections.push('virtualization_detection_error: ' + e.message);
        }

        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectContainer() {
        let score = 0;
        const detections = [];

        try {
            const ua = navigator.userAgent || '';
            const containerPatterns = [
                { pattern: /docker/i, name: 'docker' },
                { pattern: /kubernetes|k8s/i, name: 'kubernetes' },
                { pattern: /lxc/i, name: 'lxc_container' },
                { pattern: /containerd/i, name: 'containerd' },
                { pattern: /runc/i, name: 'runc' },
                { pattern: /podman/i, name: 'podman' }
            ];

            for (const container of containerPatterns) {
                if (container.pattern.test(ua)) {
                    score += 60;
                    detections.push(container.name);
                }
            }

            try {
                const response = await fetch('/.dockerenv', { method: 'HEAD' }).catch(() => null);
                if (response && response.ok) {
                    score += 70;
                    detections.push('dockerenv_file_exists');
                }
            } catch (e) {}

            try {
                if (navigator.storage && navigator.storage.estimate) {
                    const estimate = await navigator.storage.estimate();
                    if (estimate.quota === 0) {
                        score += 45;
                        detections.push('zero_quota_container');
                    }
                    if (estimate.quota && estimate.quota < 50000000) {
                        score += 35;
                        detections.push('low_quota_container');
                    }
                }
            } catch (e) {}

            try {
                const canvas = document.createElement('canvas');
                canvas.width = 100;
                canvas.height = 100;
                const ctx = canvas.getContext('2d');
                if (ctx) {
                    ctx.fillStyle = '#ff0000';
                    ctx.fillRect(0, 0, 50, 50);
                    ctx.fillStyle = '#0000ff';
                    ctx.fillRect(50, 0, 50, 50);

                    const data = ctx.getImageData(25, 25, 50, 50).data;
                    let blackCount = 0;
                    for (let i = 0; i < data.length; i += 4) {
                        if (data[i] === 0 && data[i + 1] === 0 && data[i + 2] === 0) {
                            blackCount++;
                        }
                    }
                    if (blackCount > data.length / 4 * 0.7) {
                        score += 40;
                        detections.push('canvas_container_artifact');
                    }
                }
            } catch (e) {}

        } catch (e) {
            score += 35;
            detections.push('container_detection_error: ' + e.message);
        }

        return { detected: score > 45, score: Math.min(score, 100), detections };
    }

    async detectDeviceFingerprint() {
        const fingerprint = this.generateDeviceFingerprint();
        const matches = await this.checkFingerprintMatch(fingerprint);

        return { 
            detected: matches > 0, 
            score: matches > 2 ? 30 : (matches > 0 ? 15 : 0), 
            detections: matches > 0 ? ['fingerprint_matches:' + matches] : [],
            fingerprint
        };
    }

    async detectSystemMetrics() {
        let score = 0;
        const detections = [];

        try {
            if (!navigator.hardwareConcurrency) {
                score += 20;
                detections.push('no_hardware_concurrency');
            }

            if (!navigator.deviceMemory) {
                score += 15;
                detections.push('no_device_memory');
            }

            if (!navigator.cookieEnabled) {
                score += 15;
                detections.push('cookies_disabled');
            }

            if (!('serviceWorker' in navigator)) {
                score += 10;
                detections.push('no_service_worker');
            }

            const permissions = ['notifications', 'geolocation', 'camera', 'microphone'];
            try {
                const results = await Promise.all(
                    permissions.map(p => 
                        navigator.permissions?.query({ name: p }).catch(() => ({ state: 'error' }))
                    )
                );
                const deniedCount = results.filter(r => r.state === 'denied').length;
                if (deniedCount === permissions.length) {
                    score += 25;
                    detections.push('all_permissions_denied');
                }
            } catch (e) {}

        } catch (e) {
            score += 20;
            detections.push('system_metrics_error: ' + e.message);
        }

        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectNavigatorProps() {
        let score = 0;
        const detections = [];

        try {
            const vendor = navigator.vendor || '';
            const product = navigator.product || '';
            const appVersion = navigator.appVersion || '';
            const platform = navigator.platform || '';

            if (product === 'Gecko' && !/Firefox/i.test(navigator.userAgent || '')) {
                score += 30;
                detections.push('gecko_mismatch');
            }

            if (vendor === 'Google, Inc.' && !/Chrome|Edge/i.test(navigator.userAgent || '')) {
                score += 25;
                detections.push('vendor_mismatch');
            }

            if (platform.includes('Win') && !navigator.userAgent.includes('Windows')) {
                score += 20;
                detections.push('platform_windows_mismatch');
            }

            if (platform.includes('Mac') && !navigator.userAgent.includes('Mac')) {
                score += 20;
                detections.push('platform_mac_mismatch');
            }

            const buildID = navigator.buildID;
            if (!buildID) {
                score += 15;
                detections.push('no_build_id');
            }

        } catch (e) {
            score += 20;
            detections.push('navigator_props_error: ' + e.message);
        }

        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectTimingAnomaly() {
        let score = 0;
        const detections = [];

        try {
            const iterations = 30;
            const timings = [];

            for (let i = 0; i < iterations; i++) {
                const start = performance.now();
                let dummy = 0;
                for (let j = 0; j < 15000; j++) {
                    dummy += Math.sqrt(j) * Math.sin(j);
                }
                timings.push(performance.now() - start);
            }

            const avgTime = timings.reduce((a, b) => a + b, 0) / timings.length;
            const variance = timings.reduce((sum, t) => sum + Math.pow(t - avgTime, 2), 0) / timings.length;
            const stdDev = Math.sqrt(variance);

            detections.push('math_avg:' + avgTime.toFixed(2));
            detections.push('math_stddev:' + stdDev.toFixed(2));

            if (avgTime < 0.5) {
                score += 35;
                detections.push('math_operation_too_fast');
            } else if (avgTime > 100) {
                score += 25;
                detections.push('math_operation_too_slow');
            }

            if (stdDev / avgTime > 0.6) {
                score += 30;
                detections.push('high_timing_variance');
            }

            const sortTimings = [];
            for (let i = 0; i < iterations / 3; i++) {
                const start = performance.now();
                const arr = new Array(2000);
                for (let j = 0; j < arr.length; j++) {
                    arr[j] = Math.random();
                }
                arr.sort();
                sortTimings.push(performance.now() - start);
            }

            const avgSortTime = sortTimings.reduce((a, b) => a + b, 0) / sortTimings.length;
            if (avgSortTime < 0.3) {
                score += 30;
                detections.push('sort_operation_too_fast');
            }

        } catch (e) {
            score += 25;
            detections.push('timing_anomaly_error: ' + e.message);
        }

        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    generateDeviceFingerprint() {
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
            const canvas = document.createElement('canvas');
            canvas.width = 256;
            canvas.height = 128;
            const ctx = canvas.getContext('2d');
            if (ctx) {
                ctx.fillStyle = '#f60';
                ctx.fillRect(0, 0, 256, 128);
                ctx.font = '16px Arial';
                ctx.fillStyle = '#fff';
                ctx.fillText('HJTPX Device FP', 10, 30);
                
                for (let i = 0; i < 26; i++) {
                    ctx.fillStyle = `rgb(${i * 10}, ${255 - i * 10}, ${128 + i * 5})`;
                    ctx.fillText(String.fromCharCode(65 + i), 10 + i * 9, 60);
                }
                
                const dataUrl = canvas.toDataURL();
                const hash = this.sha256Hash(dataUrl).substring(0, 48);
                components.push('cnv:' + hash);
            }
        } catch (e) {}

        try {
            const gl = document.createElement('canvas').getContext('webgl');
            if (gl) {
                const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                if (debugInfo) {
                    const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                    const vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
                    components.push('wgl:' + this.sha256Hash(renderer + vendor).substring(0, 32));
                }
            }
        } catch (e) {}

        try {
            const audioCtx = new (window.OfflineAudioContext || window.webkitOfflineAudioContext)(1, 44100, 44100);
            const osc = audioCtx.createOscillator();
            osc.type = 'sine';
            osc.frequency.setValueAtTime(440, audioCtx.currentTime);
            osc.connect(audioCtx.destination);
            osc.start();
            const bufferPromise = audioCtx.startRendering();
            bufferPromise.then(buf => {
                const data = buf.getChannelData(0);
                const hash = this.sha256Hash(data.slice(0, 1000).join(','));
                components.push('aud:' + hash.substring(0, 32));
            }).catch(() => {});
        } catch (e) {}

        try {
            components.push('tzoff:' + new Date().getTimezoneOffset());
        } catch (e) {}

        try {
            components.push('cookie:' + (navigator.cookieEnabled ? '1' : '0'));
        } catch (e) {}

        try {
            components.push('plugins:' + (navigator.plugins?.length || 0));
        } catch (e) {}

        try {
            components.push('useragent:' + this.sha256Hash(navigator.userAgent || '').substring(0, 32));
        } catch (e) {}

        const rawFingerprint = components.join('|');
        return this.sha256Hash(rawFingerprint);
    }

    async checkFingerprintMatch(fingerprint) {
        let matchCount = 0;

        try {
            const storedFingerprints = this.loadStoredFingerprints();
            for (const stored of storedFingerprints) {
                const similarity = this.compareFingerprints(fingerprint, stored);
                if (similarity > 0.8) {
                    matchCount++;
                }
            }
        } catch (e) {
            console.warn('Fingerprint match check failed:', e);
        }

        this.fingerprintMatches = matchCount;
        return matchCount;
    }

    loadStoredFingerprints() {
        try {
            const stored = localStorage.getItem('hjtpx_fingerprints');
            return stored ? JSON.parse(stored) : [];
        } catch (e) {
            return [];
        }
    }

    compareFingerprints(fp1, fp2) {
        if (fp1 === fp2) return 1.0;
        
        let matches = 0;
        const length = Math.min(fp1.length, fp2.length);
        
        for (let i = 0; i < length; i++) {
            if (fp1[i] === fp2[i]) matches++;
        }
        
        return matches / length;
    }

    sha256Hash(input) {
        const encoder = new TextEncoder();
        const data = encoder.encode(input);
        
        return Array.from(crypto.subtle.digest('SHA-256', data))
            .map(b => b.toString(16).padStart(2, '0'))
            .join('');
    }

    createPRNG(seed) {
        let s = seed;
        return function() {
            s = Math.sin(s) * 10000;
            return s - Math.floor(s);
        };
    }

    async runAll() {
        const chainResult = await this.runChain();
        return chainResult;
    }

    toJSON() {
        return {
            detection_id: this.detectionId,
            risk_score: this.riskScore,
            chain_count: this.detectionChain.length,
            results: this.results,
            fingerprint: this.generateDeviceFingerprint()
        };
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = HJTPXEnvironmentDetector;
} else if (typeof window !== 'undefined') {
    window.HJTPXEnvironmentDetector = HJTPXEnvironmentDetector;
}