class EnvironmentDetectorEnhanced {
    constructor(options = {}) {
        this.options = Object.assign({
            apiBase: '/api/v1',
            sampleRate: 1.0,
            chainCount: 25,
            enableAll: true,
            sessionId: null,
            timeout: 10000,
            retries: 2
        }, options);
        this.results = {};
        this.riskScore = 0;
        this.detectionChain = [];
        this.detectionId = 'det_enhanced_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
        this.timingData = {};
        this.fingerprintComponents = {};
        this.weights = {
            canvas: 12,
            canvasStable: 10,
            canvasEntropy: 8,
            webgl: 14,
            webgl2: 10,
            webglVendor: 8,
            webglRenderer: 8,
            audio: 11,
            fonts: 10,
            fontEnumeration: 8,
            fontMetrics: 6,
            plugins: 6,
            pluginFingerprint: 5,
            webrtc: 15,
            webrtcLeak: 12,
            webdriver: 20,
            selenium: 18,
            puppeteer: 18,
            playwright: 18,
            chromeRuntime: 10,
            headless: 15,
            permissions: 6,
            languages: 4,
            timezone: 5,
            screen: 3,
            hardware: 5,
            memory: 4,
            storage: 5,
            navigator: 5,
            windowProps: 4,
            iframe: 6,
            notification: 3,
            battery: 4,
            mediaDevices: 5,
            connection: 8,
            adblock: 5,
            math: 4,
            gpu: 7,
            speech: 3,
            proxyVPN: 18,
            torExitNode: 15,
            vpnIndicators: 14,
            virtualization: 12,
            sandbox: 10,
            automationFrameworks: 16,
            vmFeatures: 15,
            sandboxEscape: 14,
            debuggerDetection: 12
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
            { name: 'detectVirtualization', category: 'environment' },
            { name: 'detectSandbox', category: 'environment' },
            { name: 'detectCanvasEnhanced', category: 'fingerprint' },
            { name: 'detectCanvasStable', category: 'fingerprint' },
            { name: 'detectCanvasEntropy', category: 'fingerprint' },
            { name: 'detectWebGLEnhanced', category: 'fingerprint' },
            { name: 'detectWebGL2Enhanced', category: 'fingerprint' },
            { name: 'detectAudioEnhanced', category: 'fingerprint' },
            { name: 'detectFontsEnhanced', category: 'fingerprint' },
            { name: 'detectFontEnumeration', category: 'fingerprint' },
            { name: 'detectFontMetrics', category: 'fingerprint' },
            { name: 'detectPluginsEnhanced', category: 'fingerprint' },
            { name: 'detectPluginFingerprint', category: 'fingerprint' },
            { name: 'detectWebRTCEnhanced', category: 'network' },
            { name: 'detectWebRTCLeak', category: 'network' },
            { name: 'detectProxyVPN', category: 'network' },
            { name: 'detectVPNIndicators', category: 'network' },
            { name: 'detectTorExitNode', category: 'network' },
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
            { name: 'detectDebuggerEnhanced', category: 'debugger' }
        ];
    }

    generateDetectionChain(count) {
        const allMethods = this.getDetectionMethods();
        const shuffled = [...allMethods].sort(() => Math.random() - 0.5);
        const selected = shuffled.slice(0, Math.min(count, allMethods.length));
        const methodAliases = {};
        selected.forEach((method, i) => {
            methodAliases[method.name] = 'chk_' + i.toString(36) + '_' + Math.random().toString(36).substr(2, 4);
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

        return {
            detection_id: this.detectionId,
            chain: chainResults,
            chain_order: Object.values(methodAliases),
            chain_categories: selected.map(m => m.category),
            risk_score: this.riskScore,
            duration_ms: Math.round(duration),
            timing_data: this.timingData,
            timestamp: Date.now(),
            fingerprint: this.generateFingerprint()
        };
    }

    calculateRiskScore() {
        let weightedScore = 0;
        let totalWeight = 0;
        const categoryScores = { automation: 0, fingerprint: 0, network: 0, system: 0, environment: 0 };
        const categoryWeights = { automation: 0, fingerprint: 0, network: 0, system: 0, environment: 0 };

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

        const automationMethods = ['detectWebDriver', 'detectPuppeteer', 'detectPlaywright', 'detectSelenium', 'detectAutomationFrameworks'];
        const automationDetected = automationMethods.filter(m => {
            const r = this.results[m];
            return r && r.detected === true;
        }).length;

        if (automationDetected >= 3) {
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

        if (proxyDetected >= 2) {
            baseScore = Math.min(baseScore * 1.4 + 20, 100);
        } else if (proxyDetected >= 1) {
            baseScore = Math.min(baseScore * 1.2 + 10, 100);
        }

        const virtualizationMethods = ['detectVirtualization', 'detectSandbox'];
        const virtualizationDetected = virtualizationMethods.filter(m => {
            const r = this.results[m];
            return r && r.detected === true;
        }).length;

        if (virtualizationDetected >= 2) {
            baseScore = Math.min(baseScore * 1.3 + 15, 100);
        }

        return Math.round(Math.min(Math.max(baseScore, 0), 100));
    }

    getMethodCategory(methodName) {
        const methods = this.getDetectionMethods();
        const method = methods.find(m => m.name === methodName);
        return method ? method.category : null;
    }

    async detectCanvasEnhanced() {
        let score = 0;
        const detections = [];
        try {
            const canvas = document.createElement('canvas');
            canvas.width = 400;
            canvas.height = 120;
            const ctx = canvas.getContext('2d');
            if (!ctx) {
                score += 50;
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

            const dataURL = canvas.toDataURL();
            const dataURL2 = canvas.toDataURL();
            if (dataURL !== dataURL2) {
                score += 30;
                detections.push('canvas_unstable');
            }

            const imageData = ctx.getImageData(0, 0, 50, 50);
            const pixelSum = Array.from(imageData.data.slice(0, 200)).reduce((a, b) => a + b, 0);
            if (pixelSum === 0) {
                score += 25;
                detections.push('canvas_empty_readback');
            }

            const hash = this.hashString(dataURL);
            this.fingerprintComponents.canvas = hash;
            detections.push('canvas_hash:' + hash.substring(0, 16));

        } catch (e) {
            score += 40;
            detections.push('canvas_error: ' + e.message);
        }
        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectCanvasStable() {
        let score = 0;
        const detections = [];
        const results = [];
        try {
            for (let i = 0; i < 3; i++) {
                const canvas = document.createElement('canvas');
                canvas.width = 200;
                canvas.height = 50;
                const ctx = canvas.getContext('2d');
                if (ctx) {
                    ctx.fillStyle = '#f60';
                    ctx.fillRect(10, 10, 50, 30);
                    ctx.fillStyle = '#069';
                    ctx.font = '14px Arial';
                    ctx.fillText('Fingerprint Test ' + i, 20, 30);
                    results.push(canvas.toDataURL());
                }
            }

            if (results.length >= 2 && results[0] !== results[1]) {
                score += 15;
                detections.push('canvas_unstable_across_renders');
            }

            const emptyCanvas = document.createElement('canvas');
            emptyCanvas.width = 100;
            emptyCanvas.height = 100;
            const emptyCtx = emptyCanvas.getContext('2d');
            if (emptyCtx) {
                const emptyData = emptyCtx.getImageData(0, 0, 10, 10);
                const allZero = Array.from(emptyData.data).every(v => v === 0);
                if (allZero) {
                    score += 20;
                    detections.push('empty_canvas_reads_zero');
                }
            }

            const hiddenCanvas = document.createElement('canvas');
            hiddenCanvas.style.display = 'none';
            hiddenCanvas.width = 200;
            hiddenCanvas.height = 50;
            document.body.appendChild(hiddenCanvas);
            const hiddenCtx = hiddenCanvas.getContext('2d');
            if (hiddenCtx) {
                hiddenCtx.fillStyle = '#000';
                hiddenCtx.fillRect(0, 0, 200, 50);
                const hiddenData = hiddenCtx.getImageData(0, 0, 10, 10);
                const allBlack = Array.from(hiddenData.data.slice(0, 40)).every(v => v === 0);
                if (allBlack) {
                    score += 10;
                    detections.push('hidden_canvas_unreadable');
                }
            }
            document.body.removeChild(hiddenCanvas);

        } catch (e) {
            score += 25;
            detections.push('canvas_stable_error: ' + e.message);
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectCanvasEntropy() {
        let score = 0;
        const detections = [];
        try {
            const canvas = document.createElement('canvas');
            canvas.width = 300;
            canvas.height = 100;
            const ctx = canvas.getContext('2d');
            if (!ctx) {
                return { detected: false, score: 0, detections: ['no_context'] };
            }

            const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()';
            const text = Array.from({ length: 50 }, () => chars[Math.floor(Math.random() * chars.length)]).join('');
            ctx.font = '12px Arial';
            ctx.fillText(text, 5, 20);

            const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
            const data = imageData.data;

            let uniqueValues = new Set();
            for (let i = 0; i < data.length; i += 4) {
                const hex = data[i].toString(16) + data[i+1].toString(16) + data[i+2].toString(16);
                uniqueValues.add(hex);
            }

            const entropyRatio = uniqueValues.size / (data.length / 4);
            if (entropyRatio < 0.1) {
                score += 30;
                detections.push('very_low_entropy:' + entropyRatio.toFixed(4));
            } else if (entropyRatio < 0.2) {
                score += 15;
                detections.push('low_entropy:' + entropyRatio.toFixed(4));
            }

            let zeroCount = 0;
            for (let i = 0; i < data.length; i++) {
                if (data[i] === 0) zeroCount++;
            }
            const zeroRatio = zeroCount / data.length;
            if (zeroRatio > 0.8) {
                score += 25;
                detections.push('high_zero_ratio:' + zeroRatio.toFixed(4));
            }

            this.fingerprintComponents.canvasEntropy = entropyRatio;
            detections.push('entropy:' + entropyRatio.toFixed(4));

        } catch (e) {
            score += 20;
            detections.push('entropy_error: ' + e.message);
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectWebGLEnhanced() {
        let score = 0;
        const detections = [];
        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
            if (!gl) {
                score += 50;
                detections.push('no_webgl');
                return { detected: true, score: Math.min(score, 100), detections };
            }

            const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
            if (debugInfo) {
                const vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
                const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);

                if (!vendor || !renderer) {
                    score += 20;
                    detections.push('webgl_no_vendor_renderer');
                } else {
                    this.fingerprintComponents.webglVendor = vendor;
                    this.fingerprintComponents.webglRenderer = renderer;
                    detections.push('webgl_vendor:' + vendor.substring(0, 30));
                    detections.push('webgl_renderer:' + renderer.substring(0, 50));
                }

                const softwarePatterns = [
                    /swiftshader/i, /llvmpipe/i, /mesa/i, /virtual/i,
                    /google\s*inc/i, /software/i, /microsoft/i,
                    /vmware/i, /parallels/i, /virtualbox/i, /qxl/i
                ];

                for (const pattern of softwarePatterns) {
                    if (pattern.test(renderer)) {
                        score += 35;
                        detections.push('software_renderer_detected');
                        break;
                    }
                }

                const unknownPatterns = [/unknown/i, /generic/i, /default/i];
                for (const pattern of unknownPatterns) {
                    if (pattern.test(renderer)) {
                        score += 20;
                        detections.push('generic_renderer');
                        break;
                    }
                }
            } else {
                score += 25;
                detections.push('no_webgl_debug_info');
            }

            const maxTexSize = gl.getParameter(gl.MAX_TEXTURE_SIZE);
            detections.push('max_texture_size:' + maxTexSize);
            if (maxTexSize <= 1024) {
                score += 20;
                detections.push('small_max_texture');
            }

            const maxVertAttribs = gl.getParameter(gl.MAX_VERTEX_ATTRIBS);
            detections.push('max_vertex_attribs:' + maxVertAttribs);
            if (maxVertAttribs <= 8) {
                score += 15;
                detections.push('few_vertex_attribs');
            }

            const aliasedRange = gl.getParameter(gl.ALIASED_LINE_WIDTH_RANGE);
            if (aliasedRange && aliasedRange[1] <= 1) {
                score += 15;
                detections.push('aliased_only');
            }

            const shaderPrecision = gl.getShaderPrecisionFormat(gl.FRAGMENT_SHADER, gl.HIGH_FLOAT);
            if (shaderPrecision) {
                detections.push('shader_precision:' + shaderPrecision.precision);
                if (shaderPrecision.precision < 16) {
                    score += 20;
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
                    score += 15;
                    detections.push('few_webgl_extensions');
                }
                const criticalExts = ['OES_texture_float', 'WEBGL_debug_renderer_info', 'EXT_texture_filter_anisotropic'];
                for (const extName of criticalExts) {
                    if (!supportedExts.includes(extName) && extName !== 'WEBGL_debug_renderer_info') {
                        score += 5;
                        detections.push('missing_critical_ext:' + extName);
                    }
                }
            }

            const vertexShader = gl.createShader(gl.VERTEX_SHADER);
            const fragmentShader = gl.createShader(gl.FRAGMENT_SHADER);
            gl.shaderSource(vertexShader, 'attribute vec2 position; void main() { gl_Position = vec4(position, 0.0, 1.0); }');
            gl.compileShader(vertexShader);
            const vertCompiled = gl.getShaderParameter(vertexShader, gl.COMPILE_STATUS);
            if (!vertCompiled) {
                score += 10;
                detections.push('vertex_shader_compile_failed');
            }

        } catch (e) {
            score += 40;
            detections.push('webgl_error: ' + e.message);
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
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
                        score += 30;
                        detections.push('webgl2_software_renderer');
                        break;
                    }
                }
            }

            const maxTexSize = gl2.getParameter(gl2.MAX_TEXTURE_SIZE);
            detections.push('webgl2_max_texture:' + maxTexSize);
            if (maxTexSize <= 1024) {
                score += 15;
                detections.push('webgl2_small_texture');
            }

            const supportedExts = gl2.getSupportedExtensions();
            if (supportedExts) {
                detections.push('webgl2_ext_count:' + supportedExts.length);
                if (supportedExts.length < 10) {
                    score += 10;
                    detections.push('few_webgl2_extensions');
                }
            }

            const transformFeedback = gl2.getParameter(gl2.MAX_TRANSFORM_FEEDBACK_SEPARATE_ATTRIBS);
            detections.push('transform_feedback_attrs:' + transformFeedback);

        } catch (e) {
            score += 25;
            detections.push('webgl2_error: ' + e.message);
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectAudioEnhanced() {
        let score = 0;
        const detections = [];
        try {
            const AudioContext = window.OfflineAudioContext || window.webkitOfflineAudioContext;
            if (!AudioContext) {
                score += 35;
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
                score += 25;
                detections.push('audio_render_too_fast');
            }

            const channelData = buffer.getChannelData(0);
            let sumAbs = 0;
            let sumSq = 0;
            let nonZeroCount = 0;
            for (let i = 4500; i < 5000; i++) {
                const abs = Math.abs(channelData[i]);
                sumAbs += abs;
                if (abs > 0) nonZeroCount++;
            }
            for (let i = 0; i < channelData.length; i++) {
                sumSq += channelData[i] * channelData[i];
            }

            detections.push('audio_non_zero_ratio:' + (nonZeroCount / 500).toFixed(4));
            if (nonZeroCount < 100) {
                score += 30;
                detections.push('audio_mostly_silent');
            }

            const variance = sumSq / channelData.length;
            if (variance < 0.0001) {
                score += 20;
                detections.push('audio_no_variance');
            }

            const multipleBuffers = [];
            for (let i = 0; i < 3; i++) {
                const tempCtx = new AudioContext(1, 44100, 44100);
                const tempOsc = tempCtx.createOscillator();
                tempOsc.frequency.setValueAtTime(1000, tempCtx.currentTime);
                tempOsc.connect(tempCtx.destination);
                tempOsc.start(0);
                const tempBuffer = await tempCtx.startRendering();
                multipleBuffers.push(tempBuffer.getChannelData(0).slice(4500, 4550).join(','));
            }

            if (multipleBuffers[0] === multipleBuffers[1]) {
                score += 25;
                detections.push('audio_identical_across_renders');
            }

            const hash = this.hashString(multipleBuffers[0]);
            this.fingerprintComponents.audio = hash;
            detections.push('audio_hash:' + hash.substring(0, 16));

        } catch (e) {
            score += 35;
            detections.push('audio_error: ' + e.message);
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
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
            'Corbel', 'Cambria', 'Bookman', 'Futura', 'Optima',
            'Arial Black', 'Arial Narrow', 'Century Gothic', 'Franklin Gothic Medium',
            'Rockwell', 'Rockwell Extra Bold', 'Courier', 'Constantia'
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
            const fontWidths = {};
            for (const font of testFonts) {
                for (const base of baseFonts) {
                    el.style.fontFamily = `"${font}", ${base}`;
                    const width = el.offsetWidth;
                    if (width !== baseWidths[base]) {
                        detectedFonts.push(font);
                        fontWidths[font] = width;
                        break;
                    }
                }
            }

            document.body.removeChild(el);

            detections.push('detected_font_count:' + detectedFonts.length);
            if (detectedFonts.length < 3) {
                score += 35;
                detections.push('very_few_fonts');
            } else if (detectedFonts.length < 8) {
                score += 20;
                detections.push('few_fonts');
            } else if (detectedFonts.length < 15) {
                score += 10;
                detections.push('limited_fonts');
            }

            this.fingerprintComponents.fonts = detectedFonts;
            detections.push('fonts:' + detectedFonts.slice(0, 10).join(','));

            const suspiciousFonts = ['Keyboard', 'Fake Font', 'Font1', 'TestFont'];
            for (const suspicious of suspiciousFonts) {
                if (detectedFonts.some(f => f.includes(suspicious))) {
                    score += 15;
                    detections.push('suspicious_font');
                    break;
                }
            }

        } catch (e) {
            score += 30;
            detections.push('font_detection_error: ' + e.message);
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
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
                    score += 20;
                    detections.push('no_enumerated_fonts');
                } else if (fontList.length < 5) {
                    score += 10;
                    detections.push('few_enumerated_fonts');
                }

                this.fingerprintComponents.enumeratedFonts = fontList;
            } else {
                score += 15;
                detections.push('font_api_unavailable');
            }

            if (navigator.permissions) {
                try {
                    const result = await navigator.permissions.query({ name: 'font-access' });
                    if (result.state === 'granted') {
                        detections.push('font_access_granted');
                    } else if (result.state === 'denied') {
                        score += 10;
                        detections.push('font_access_denied');
                    }
                } catch (e) {
                    detections.push('font_access_query_failed');
                }
            }

        } catch (e) {
            score += 20;
            detections.push('font_enumeration_error: ' + e.message);
        }
        return { detected: score > 20, score: Math.min(score, 100), detections };
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
                    actualBoundingBoxRight: metrics2.actualBoundingBoxRight,
                    fontBoundingBoxAscent: metrics2.fontBoundingBoxAscent,
                    fontBoundingBoxDescent: metrics2.fontBoundingBoxDescent
                });
            }

            let uniqueWidths = new Set(metrics.map(m => Math.round(m.width)));
            detections.push('unique_widths:' + uniqueWidths.size);

            if (uniqueWidths.size < 3) {
                score += 25;
                detections.push('fonts_have_same_width');
            }

            const avgWidth = metrics.reduce((sum, m) => sum + m.width, 0) / metrics.length;
            const variance = metrics.reduce((sum, m) => sum + Math.pow(m.width - avgWidth, 2), 0) / metrics.length;
            const stdDev = Math.sqrt(variance);

            detections.push('metrics_variance:' + stdDev.toFixed(2));
            if (stdDev < 50) {
                score += 15;
                detections.push('low_font_variance');
            }

        } catch (e) {
            score += 20;
            detections.push('font_metrics_error: ' + e.message);
        }
        return { detected: score > 20, score: Math.min(score, 100), detections };
    }

    async detectPluginsEnhanced() {
        let score = 0;
        const detections = [];
        try {
            const plugins = navigator.plugins;
            if (!plugins || plugins.length === 0) {
                score += 30;
                detections.push('no_plugins');
            } else {
                const pluginNames = Array.from(plugins).map(p => p.name);
                detections.push('plugin_count:' + plugins.length);
                detections.push('plugins:' + pluginNames.slice(0, 5).join(','));

                if (plugins.length < 2) {
                    score += 25;
                    detections.push('very_few_plugins');
                }

                const commonPlugins = ['PDF Viewer', 'Chrome PDF Viewer', 'Chromium PDF Viewer',
                    'Microsoft Edge PDF Viewer', 'WebKit built-in PDF'];
                const hasPDF = pluginNames.some(p =>
                    commonPlugins.some(cp => p.includes(cp))
                );
                if (!hasPDF) {
                    score += 15;
                    detections.push('no_pdf_plugin');
                }

                const chromePlugins = pluginNames.filter(p =>
                    p.includes('Chrome') || p.includes('Chromium')
                );
                if (chromePlugins.length === 0 && /Chrome|Chromium|Edge/i.test(navigator.userAgent || '')) {
                    score += 20;
                    detections.push('missing_chrome_plugins');
                }
            }

            if (navigator.mimeTypes) {
                const mimeTypes = Array.from(navigator.mimeTypes).map(m => m.type);
                detections.push('mime_type_count:' + mimeTypes.length);
                if (mimeTypes.length < 2) {
                    score += 15;
                    detections.push('few_mime_types');
                }
            }

        } catch (e) {
            score += 35;
            detections.push('plugins_error: ' + e.message);
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
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
                        description: p.description,
                        version: p.version
                    };
                    pluginData.plugins.push(pluginInfo);
                    pluginData.totalLength += (p.name + p.description + p.filename).length;

                    for (let j = 0; j < p.length; j++) {
                        const m = p[j];
                        pluginData.mimeTypes.push(m.type);
                        pluginData.totalLength += m.type.length;
                    }
                }
            }

            for (const m of mimeTypes) {
                if (m && m.type) {
                    pluginData.totalLength += m.type.length;
                }
            }

            const hash = this.hashString(JSON.stringify(pluginData.plugins));
            this.fingerprintComponents.pluginFingerprint = hash;
            detections.push('plugin_hash:' + hash.substring(0, 16));
            detections.push('plugin_data_length:' + pluginData.totalLength);

            if (pluginData.totalLength < 50) {
                score += 20;
                detections.push('minimal_plugin_data');
            }

            const chromePattern = /Chrome|Chromium|Edg/i;
            const hasChromePlugin = pluginData.plugins.some(p =>
                chromePattern.test(p.name) || chromePattern.test(p.description)
            );

            if (chromePattern.test(navigator.userAgent || '') && !hasChromePlugin) {
                score += 25;
                detections.push('plugin_ua_mismatch');
            }

        } catch (e) {
            score += 25;
            detections.push('plugin_fingerprint_error: ' + e.message);
        }
        return { detected: score > 20, score: Math.min(score, 100), detections };
    }

    async detectWebRTCEnhanced() {
        let score = 0;
        const detections = [];
        try {
            const RTCPeerConnection = window.RTCPeerConnection ||
                window.webkitRTCPeerConnection ||
                window.mozRTCPeerConnection;

            if (!RTCPeerConnection) {
                score += 20;
                detections.push('no_webrtc');
                return { detected: true, score: Math.min(score, 100), detections };
            }

            const ips = new Set();
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

            await new Promise(resolve => setTimeout(resolve, 1000));

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
            detections.push('ips:' + Array.from(ips).slice(0, 3).join(','));

            if (ips.size === 0) {
                score += 15;
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
                score += 25;
                detections.push('vpn_ip_mismatch');
            }

            this.fingerprintComponents.webrtcIPs = Array.from(ips);

        } catch (e) {
            score += 20;
            detections.push('webrtc_error: ' + e.message);
        }
        return { detected: score > 20, score: Math.min(score, 100), detections };
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

            await new Promise(resolve => setTimeout(resolve, 1500));

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
                score += 30;
                detections.push('ip_leak_detected');
                detections.push('leaked_ips:' + newIPs.join(','));
            }

            const ipv6Pattern = /([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|::/;
            const hasIPv6 = Array.from(newIPs).some(ip => ipv6Pattern.test(ip));
            if (hasIPv6) {
                score += 10;
                detections.push('ipv6_leak');
            }

        } catch (e) {
            score += 15;
            detections.push('webrtc_leak_error: ' + e.message);
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectProxyVPN() {
        let score = 0;
        const detections = [];
        try {
            const xff = this.getHeader('X-Forwarded-For');
            const xri = this.getHeader('X-Real-IP');
            const via = this.getHeader('Via');
            const proxyChain = this.getHeader('X-ProxyChain');
            const forwarded = this.getHeader('Forwarded');

            if (xff) {
                const xffIPs = xff.split(',').map(ip => ip.trim());
                detections.push('xff_count:' + xffIPs.length);
                if (xffIPs.length > 2) {
                    score += 25;
                    detections.push('multi_hop_proxy');
                }
            }

            if (via) {
                score += 15;
                detections.push('via_header_present');
                const proxyKeywords = ['proxy', 'vpn', 'squid', 'nginx', 'apache', ' varnish'];
                for (const keyword of proxyKeywords) {
                    if (via.toLowerCase().includes(keyword)) {
                        score += 15;
                        detections.push('known_proxy_via:' + keyword);
                        break;
                    }
                }
            }

            if (xff && xri && xff !== xri) {
                score += 20;
                detections.push('ip_mismatch');
            }

            const publicIP = this.fingerprintComponents.publicIP;
            if (publicIP) {
                const isDatacenter = this.checkDatacenterIP(publicIP);
                if (isDatacenter) {
                    score += 30;
                    detections.push('datacenter_ip');
                }
            }

        } catch (e) {
            score += 15;
            detections.push('proxy_vpn_error: ' + e.message);
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectVPNIndicators() {
        let score = 0;
        const detections = [];
        try {
            const conn = navigator.connection || navigator.mozConnection || navigator.webkitConnection;

            if (conn) {
                if (conn.type === 'vpn' || conn.type === 'vpn' ) {
                    score += 40;
                    detections.push('vpn_type_detected');
                }

                if (conn.type === 'ethernet' || conn.type === 'wifi') {
                    const effectiveType = conn.effectiveType;
                    detections.push('connection_type:' + effectiveType);

                    if (effectiveType === 'slow-2g' || effectiveType === '2g') {
                        score += 10;
                        detections.push('slow_connection');
                    }
                }

                if (conn.saveData === true) {
                    score += 10;
                    detections.push('data_saver_enabled');
                }
            }

            const latency = conn ? conn.rtt : null;
            if (latency !== null && latency > 500) {
                score += 15;
                detections.push('high_latency:' + latency);
            }

            if (navigator.onLine === false) {
                score += 10;
                detections.push('offline');
            }

        } catch (e) {
            score += 15;
            detections.push('vpn_indicators_error: ' + e.message);
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
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
                    score += 50;
                    detections.push('known_tor_exit_node');
                }

                const torPorts = [9001, 9030, 9050, 9051, 9150];
                detections.push('tor_check_performed');
            }

            const userAgent = navigator.userAgent || '';
            if (/tor|onion/i.test(userAgent)) {
                score += 40;
                detections.push('tor_in_user_agent');
            }

            if (this.fingerprintComponents.webrtcIPs) {
                const hasTorIndicator = this.fingerprintComponents.webrtcIPs.some(ip =>
                    ip.endsWith('.onion') || ip.match(/\.exit$/i)
                );
                if (hasTorIndicator) {
                    score += 35;
                    detections.push('tor_onion_detected');
                }
            }

        } catch (e) {
            score += 15;
            detections.push('tor_check_error: ' + e.message);
        }
        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectHeadless() {
        let score = 0;
        const detections = [];
        if (navigator.webdriver === true || navigator.webdriver === false) {
        } else {
            score += 30;
            detections.push('webdriver_undefined');
        }
        if (navigator.plugins && navigator.plugins.length === 0) {
            score += 20;
            detections.push('no_plugins');
        }
        if (navigator.languages && navigator.languages.length === 0) {
            score += 20;
            detections.push('no_languages');
        }
        if (window.chrome && window.chrome.runtime === undefined) {
            score += 25;
            detections.push('chrome_no_runtime');
        }
        const mimeTypes = navigator.mimeTypes;
        if (mimeTypes && mimeTypes.length === 0) {
            score += 25;
            detections.push('no_mimetypes');
        }
        try {
            const ua = navigator.userAgent || '';
            if (/headless|phantom/i.test(ua)) {
                score += 40;
                detections.push('headless_ua');
            }
        } catch (e) {}
        try {
            if (window.outerHeight === 0 && window.outerWidth === 0) {
                score += 30;
                detections.push('zero_window_size');
            }
        } catch (e) {}

        try {
            const canvas = document.createElement('canvas');
            const ctx = canvas.getContext('2d');
            if (ctx) {
                ctx.fillStyle = 'rgba(255,0,0,0.5)';
                ctx.fillRect(0, 0, 10, 10);
                const imageData = ctx.getImageData(0, 0, 10, 10);
                const hasNonZero = Array.from(imageData.data).some(v => v > 0);
                if (!hasNonZero) {
                    score += 25;
                    detections.push('headless_pixel_pattern');
                }
            }
        } catch (e) {}

        return { detected: score > 35, score: Math.min(score, 100), detections };
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
                score += 20;
                detections.push(prop);
            }
        }
        try {
            if (navigator.webdriver === true) {
                score += 35;
                detections.push('navigator_webdriver_true');
            }
        } catch (e) {}
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectPuppeteer() {
        let score = 0;
        const detections = [];
        try {
            if (navigator.webdriver === true) {
                score += 30;
                detections.push('webdriver_true');
            }
        } catch (e) {}
        try {
            if (document.$cdc_asdjflasutopfhvcZLmcfl_) {
                score += 45;
                detections.push('puppeteer_cdc_marker');
            }
        } catch (e) {}
        try {
            if (document.$chrome_asyncScriptInfo) {
                score += 35;
                detections.push('chrome_async_script');
            }
        } catch (e) {}
        try {
            const userAgent = navigator.userAgent || '';
            if (/headless/i.test(userAgent)) {
                score += 35;
                detections.push('headless_ua');
            }
            if (/puppet/i.test(userAgent)) {
                score += 50;
                detections.push('puppeteer_ua');
            }
        } catch (e) {}
        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectPlaywright() {
        let score = 0;
        const detections = [];
        try {
            if (window.__playwright__ !== undefined ||
                window.__pw_tags !== undefined ||
                window.__pw_resume__ !== undefined) {
                score += 55;
                detections.push('playwright_global');
            }
        } catch (e) {}
        try {
            const el = document.createElement('div');
            el.setAttribute('onfocus', 'return __pw_resume__()');
            if (el.onfocus !== null) {
                score += 40;
                detections.push('playwright_onfocus');
            }
        } catch (e) {}
        try {
            const ua = navigator.userAgent || '';
            if (/playwright/i.test(ua)) {
                score += 55;
                detections.push('playwright_ua');
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
            'Selenium.prototype'
        ];
        for (const prop of selProps) {
            if (window[prop] !== undefined || document[prop] !== undefined) {
                score += 25;
                detections.push(prop);
            }
        }
        try {
            if (document.documentElement.getAttribute('webdriver') !== null) {
                score += 35;
                detections.push('webdriver_attr');
            }
        } catch (e) {}
        try {
            const ua = navigator.userAgent || '';
            if (/selenium/i.test(ua)) {
                score += 50;
                detections.push('selenium_ua');
            }
        } catch (e) {}
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectChromeRuntime() {
        let score = 0;
        const detections = [];
        try {
            if (window.chrome) {
                if (window.chrome.runtime === undefined) {
                    score += 25;
                    detections.push('chrome_runtime_missing');
                }
                if (window.chrome.loadTimes === undefined) {
                    score += 15;
                    detections.push('chrome_loadtimes_missing');
                }
                if (window.chrome.csi === undefined) {
                    score += 15;
                    detections.push('chrome_csi_missing');
                }
                if (window.chrome.app === undefined) {
                    score += 15;
                    detections.push('chrome_app_missing');
                }
            } else {
                if (!/Edge|Edg|Firefox|Safari/i.test(navigator.userAgent || '')) {
                    score += 35;
                    detections.push('no_chrome_no_alt');
                }
            }
        } catch (e) {
            score += 30;
            detections.push('chrome_check_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
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
                        score += 50;
                        detections.push(`${framework}_detected:${marker}`);
                    }
                } else if (window[marker] !== undefined || document[marker] !== undefined) {
                    score += 45;
                    detections.push(`${framework}_detected:${marker}`);
                }
            }
        }

        try {
            const testElement = document.createElement('div');
            testElement.style.cssText = 'display:none';

            const automationFunctions = [
                '__webdriver_script_fn', '__driver_evaluate', '__fxdriver_evaluate',
                '__selenium_evaluate', '__webdriver_unwrapped', '__$webdriverAsyncExecutor'
            ];

            for (const func of automationFunctions) {
                try {
                    testElement.setAttribute('onclick', `return ${func}()`);
                    if (testElement.onclick !== null) {
                        score += 30;
                        detections.push(`automation_fn:${func}`);
                    }
                } catch (e) {}
            }
        } catch (e) {}

        return { detected: score > 35, score: Math.min(score, 100), detections };
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
                    score += 40;
                    detections.push(`${indicator.name}_detected`);
                }
            }

            const cpu = navigator.hardwareConcurrency;
            if (cpu && cpu < 2) {
                score += 20;
                detections.push('low_core_count');
            }

            const mem = navigator.deviceMemory;
            if (mem && mem < 1) {
                score += 25;
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
                            score += 45;
                            detections.push('vm_webgl_renderer');
                        }
                    }
                }
            } catch (e) {}

        } catch (e) {
            score += 20;
            detections.push('virtualization_error: ' + e.message);
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectSandbox() {
        let score = 0;
        const detections = [];
        try {
            try {
                new Worker('data:text/javascript;base64,');
                detections.push('worker_available');
            } catch (e) {
                score += 20;
                detections.push('worker_blocked');
            }

            try {
                const blob = new Blob([''], { type: 'text/plain' });
                const url = URL.createObjectURL(blob);
                const worker = new Worker(url);
                worker.terminate();
                URL.revokeObjectURL(url);
                detections.push('blob_worker_ok');
            } catch (e) {
                score += 15;
                detections.push('blob_worker_blocked');
            }

            try {
                if (typeof SharedArrayBuffer !== 'undefined') {
                    detections.push('shared_array_buffer_available');
                } else {
                    score += 10;
                    detections.push('shared_array_buffer_unavailable');
                }
            } catch (e) {
                score += 10;
                detections.push('shared_array_buffer_error');
            }

            try {
                const testChannel = new MessageChannel();
                testChannel.port1.postMessage('test');
                testChannel.port2.onmessage = () => {};
                testChannel.port1.close();
                testChannel.port2.close();
                detections.push('message_channel_ok');
            } catch (e) {
                score += 15;
                detections.push('message_channel_blocked');
            }

        } catch (e) {
            score += 20;
            detections.push('sandbox_error: ' + e.message);
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
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
                    score += 25;
                    detections.push('all_permissions_denied');
                }
                const deniedCount = permChecks.filter(p => p.state === 'denied').length;
                if (deniedCount >= 4) {
                    score += 15;
                    detections.push('most_permissions_denied');
                }
            } else {
                score += 25;
                detections.push('permissions_api_missing');
            }
        } catch (e) {
            score += 30;
            detections.push('permissions_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectLanguages() {
        let score = 0;
        const detections = [];
        try {
            const langs = navigator.languages;
            if (!langs || langs.length === 0) {
                score += 30;
                detections.push('no_languages');
            }
            const lang = navigator.language;
            if (!lang) {
                score += 25;
                detections.push('no_language');
            }
            if (langs && langs.length > 0 && lang) {
                if (langs[0] !== lang) {
                    score += 20;
                    detections.push('languages_mismatch');
                }
            }
        } catch (e) {
            score += 35;
            detections.push('languages_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectTimezone() {
        let score = 0;
        const detections = [];
        try {
            const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
            if (!tz) {
                score += 35;
                detections.push('no_timezone');
            }
            const offset = new Date().getTimezoneOffset();
            if (offset === 0 && !tz) {
                score += 25;
                detections.push('utc_offset_no_tz');
            }
            const year = new Date().getFullYear();
            if (year < 2000 || year > 2100) {
                score += 30;
                detections.push('unrealistic_date');
            }
            try {
                const matchOffset = /GMT([+-]\d{2}):?(\d{2})/.exec(new Date().toString());
                if (matchOffset) {
                    const strOffset = parseInt(matchOffset[1]) * 60 + parseInt(matchOffset[2]) * (matchOffset[1] > 0 ? 1 : -1);
                    if (Math.abs(strOffset + offset) > 30) {
                        score += 25;
                        detections.push('timezone_offset_mismatch');
                    }
                }
            } catch (e) {}
        } catch (e) {
            score += 40;
            detections.push('timezone_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectScreen() {
        let score = 0;
        const detections = [];
        try {
            const { width, height, colorDepth, pixelDepth, availWidth, availHeight } = screen;
            if (!width || !height) {
                score += 35;
                detections.push('no_screen_size');
            }
            if (colorDepth === 0 || pixelDepth === 0) {
                score += 30;
                detections.push('zero_depth');
            }
            if (width <= 800 || height <= 600) {
                score += 15;
                detections.push('small_screen');
            }
            if (availWidth === 0 || availHeight === 0) {
                score += 20;
                detections.push('zero_avail_size');
            }
        } catch (e) {
            score += 35;
            detections.push('screen_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectHardwareConcurrency() {
        let score = 0;
        const detections = [];
        try {
            const c = navigator.hardwareConcurrency;
            if (c === undefined || c === null) {
                score += 35;
                detections.push('no_concurrency');
            } else if (c <= 1) {
                score += 30;
                detections.push('single_core');
            } else if (c > 64) {
                score += 25;
                detections.push('unrealistic_cores');
            }
        } catch (e) {
            score += 35;
            detections.push('concurrency_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectDeviceMemory() {
        let score = 0;
        const detections = [];
        try {
            const mem = navigator.deviceMemory;
            if (mem === undefined || mem === null) {
                score += 25;
                detections.push('no_device_memory');
            } else if (mem <= 0.25) {
                score += 30;
                detections.push('low_memory');
            } else if (mem > 64) {
                score += 20;
                detections.push('unrealistic_memory');
            }
        } catch (e) {
            score += 25;
            detections.push('memory_error');
        }
        return { detected: score > 20, score: Math.min(score, 100), detections };
    }

    async detectStorage() {
        let score = 0;
        const detections = [];
        try {
            localStorage.setItem('_md_test', '1');
            localStorage.removeItem('_md_test');
        } catch (e) {
            score += 25;
            detections.push('localStorage_denied');
        }
        try {
            sessionStorage.setItem('_md_test', '1');
            sessionStorage.removeItem('_md_test');
        } catch (e) {
            score += 25;
            detections.push('sessionStorage_denied');
        }
        try {
            if (navigator.storage && navigator.storage.estimate) {
                const est = await navigator.storage.estimate();
                if (est.quota === 0) {
                    score += 20;
                    detections.push('zero_storage_quota');
                }
            }
        } catch (e) {
            score += 20;
            detections.push('storage_estimate_error');
        }
        try {
            if (navigator.cookieEnabled === false) {
                score += 20;
                detections.push('cookies_disabled');
            }
        } catch (e) {}
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectNavigatorProps() {
        let score = 0;
        const detections = [];
        try {
            if (navigator.connection) {
                if (navigator.connection.type === 'none' && navigator.onLine === false) {
                    score += 25;
                    detections.push('offline_with_connection');
                }
            }
        } catch (e) {}
        try {
            if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
                const devices = await navigator.mediaDevices.enumerateDevices().catch(() => []);
                if (devices.length === 0) {
                    score += 20;
                    detections.push('no_media_devices');
                }
            }
        } catch (e) {
            score += 20;
            detections.push('media_devices_error');
        }
        try {
            if (!navigator.credentials || !navigator.credentials.preventSilentAccess) {
                score += 10;
                detections.push('no_credentials_api');
            }
        } catch (e) {}
        try {
            if (navigator.serviceWorker === undefined) {
                score += 15;
                detections.push('no_serviceworker');
            }
        } catch (e) {}
        try {
            if (navigator.product === 'Gecko' && !/Firefox/i.test(navigator.userAgent || '')) {
                score += 25;
                detections.push('gecko_no_firefox');
            }
        } catch (e) {}
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectWindowProps() {
        let score = 0;
        const detections = [];
        try {
            const outerW = window.outerWidth;
            const outerH = window.outerHeight;
            const innerW = window.innerWidth;
            const innerH = window.innerHeight;
            if (outerW === 0 || outerH === 0) {
                score += 30;
                detections.push('zero_outer_size');
            }
            if (innerW > outerW || innerH > outerH) {
                score += 20;
                detections.push('inner_larger_than_outer');
            }
        } catch (e) {
            score += 25;
            detections.push('window_size_error');
        }
        try {
            if (window.screenX === undefined || window.screenY === undefined) {
                score += 15;
                detections.push('no_screen_position');
            }
        } catch (e) {}
        try {
            if (window.indexedDB === undefined) {
                score += 15;
                detections.push('no_indexeddb');
            }
        } catch (e) {}
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectIframe() {
        let score = 0;
        const detections = [];
        try {
            if (window.self !== window.top) {
                score += 20;
                detections.push('in_iframe');
            }
        } catch (e) {
            score += 40;
            detections.push('cross_origin_frame');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectNotification() {
        let score = 0;
        const detections = [];
        try {
            if ('Notification' in window) {
                if (Notification.permission === 'denied') {
                    score += 10;
                    detections.push('notification_denied');
                }
            } else {
                score += 20;
                detections.push('no_notification');
            }
        } catch (e) {
            score += 20;
            detections.push('notification_error');
        }
        return { detected: score > 15, score: Math.min(score, 100), detections };
    }

    async detectBattery() {
        let score = 0;
        const detections = [];
        try {
            if (navigator.getBattery) {
                const battery = await navigator.getBattery().catch(() => null);
                if (battery) {
                    if (battery.level === undefined || battery.charging === undefined) {
                        score += 20;
                        detections.push('battery_props_missing');
                    }
                }
            } else {
                score += 15;
                detections.push('no_battery_api');
            }
        } catch (e) {
            score += 20;
            detections.push('battery_error');
        }
        return { detected: score > 15, score: Math.min(score, 100), detections };
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
                    score += 25;
                    detections.push('no_media_inputs');
                }
                const allHaveLabels = devices.every(d => d.label !== '');
                if (!allHaveLabels) {
                    score += 15;
                    detections.push('media_no_labels');
                }
            } else {
                score += 20;
                detections.push('no_media_api');
            }
        } catch (e) {
            score += 25;
            detections.push('media_error');
        }
        return { detected: score > 20, score: Math.min(score, 100), detections };
    }

    async detectConnection() {
        let score = 0;
        const detections = [];
        try {
            const conn = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
            if (!conn) {
                score += 15;
                detections.push('no_connection_api');
            } else {
                if (conn.type === 'vpn') {
                    score += 45;
                    detections.push('vpn_detected');
                }
                if (conn.type === 'proxy') {
                    score += 45;
                    detections.push('proxy_detected');
                }
                if (conn.saveData === true) {
                    score += 15;
                    detections.push('save_data_enabled');
                }
                if (conn.effectiveType === 'slow-2g' || conn.effectiveType === '2g') {
                    score += 15;
                    detections.push('slow_connection');
                }
            }
        } catch (e) {
            score += 15;
            detections.push('connection_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
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
                score += 20;
                detections.push('adblock_detected');
            }
            document.body.removeChild(el);
        } catch (e) {
            score += 15;
            detections.push('adblock_check_error');
        }
        return { detected: score > 15, score: Math.min(score, 100), detections };
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
                    score += 20;
                    detections.push('math_' + key + '_invalid');
                }
            }
        } catch (e) {
            score += 25;
            detections.push('math_error');
        }
        return { detected: score > 15, score: Math.min(score, 100), detections };
    }

    async detectGPUFingerprint() {
        let score = 0;
        const detections = [];
        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
            if (gl) {
                const maxRenderSize = gl.getParameter(gl.MAX_RENDERBUFFER_SIZE);
                const maxViewport = gl.getParameter(gl.MAX_VIEWPORT_DIMS);
                const maxCombinedTexUnits = gl.getParameter(gl.MAX_COMBINED_TEXTURE_IMAGE_UNITS);
                if (maxRenderSize <= 1024) {
                    score += 20;
                    detections.push('small_renderbuffer');
                }
                if (maxCombinedTexUnits <= 8) {
                    score += 15;
                    detections.push('few_texture_units');
                }
            }
        } catch (e) {
            score += 15;
            detections.push('gpu_error');
        }
        return { detected: score > 15, score: Math.min(score, 100), detections };
    }

    async detectSpeech() {
        let score = 0;
        const detections = [];
        try {
            if ('speechSynthesis' in window) {
                const voices = window.speechSynthesis.getVoices();
                if (voices.length === 0) {
                    score += 15;
                    detections.push('no_speech_voices');
                }
            } else {
                score += 20;
                detections.push('no_speech_api');
            }
        } catch (e) {
            score += 15;
            detections.push('speech_error');
        }
        return { detected: score > 10, score: Math.min(score, 100), detections };
    }

    async detectVMFeatures() {
        let score = 0;
        const detections = [];
        try {
            const vmPatterns = {
                vmware: [/vmware/i, /virtualbox/i, /parallels/i, /hyper[- ]?v/i, /qemu/i, /kvm/i, /xen/i],
                cpuFeatures: [/[0-9]+cpu/i, /core\s*\d+/i, /processor/i],
                biosIndicators: [/bios/i, /firmware/i, /uefi/i]
            };

            const ua = navigator.userAgent || '';
            for (const [vmType, patterns] of Object.entries(vmPatterns)) {
                for (const pattern of patterns) {
                    if (pattern.test(ua)) {
                        score += 35;
                        detections.push(`vm_pattern:${vmType}_${pattern}`);
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
                        const vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);

                        const vmRendererPatterns = [
                            /vmware/i, /virtualbox/i, /parallels/i, /qemu/i,
                            /kvm/i, /xen/i, /hyper[ -]?v/i, /virtual/i,
                            /microsoft.*basic/i, /swiftshader/i, /llvmpipe/i
                        ];

                        for (const pattern of vmRendererPatterns) {
                            if (pattern.test(renderer) || pattern.test(vendor)) {
                                score += 40;
                                detections.push(`vm_renderer:${pattern}`);
                            }
                        }

                        this.fingerprintComponents.vmRenderer = renderer;
                        this.fingerprintComponents.vmVendor = vendor;
                    }
                }
            } catch (e) {
                detections.push('vm_webgl_check_error');
            }

            try {
                const cpuCores = navigator.hardwareConcurrency;
                if (cpuCores) {
                    detections.push(`cpu_cores:${cpuCores}`);
                    if (cpuCores === 1) {
                        score += 20;
                        detections.push('vm_single_core');
                    } else if (cpuCores === 2) {
                        score += 10;
                        detections.push('vm_dual_core');
                    }
                }
            } catch (e) {}

            try {
                const memory = navigator.deviceMemory;
                if (memory) {
                    detections.push(`device_memory:${memory}`);
                    if (memory < 2) {
                        score += 25;
                        detections.push('vm_low_memory');
                    }
                }
            } catch (e) {}

            try {
                const startPerf = performance.now();
                let dummy = 0;
                for (let i = 0; i < 1000000; i++) {
                    dummy += Math.sqrt(i);
                }
                const endPerf = performance.now();
                const perfTime = endPerf - startPerf;

                detections.push(`perf_time:${perfTime.toFixed(2)}`);

                if (perfTime < 1) {
                    score += 30;
                    detections.push('vm_unusual_performance');
                } else if (perfTime > 100) {
                    score += 15;
                    detections.push('vm_slow_performance');
                }

                this.fingerprintComponents.perfTime = perfTime;
            } catch (e) {}

            try {
                const platform = navigator.platform || '';
                const vmPlatforms = [/vm/i, /virtual/i, /parallels/i, /virtualbox/i];
                for (const pattern of vmPlatforms) {
                    if (pattern.test(platform)) {
                        score += 35;
                        detections.push(`vm_platform:${pattern}`);
                    }
                }
            } catch (e) {}

            try {
                const screenProps = {
                    width: screen.width,
                    height: screen.height,
                    colorDepth: screen.colorDepth
                };

                const vmScreenPatterns = [
                    { w: 800, h: 600, name: 'vm_800x600' },
                    { w: 1024, h: 768, name: 'vm_1024x768' },
                    { w: 1366, h: 768, name: 'vm_1366x768' }
                ];

                for (const pattern of vmScreenPatterns) {
                    if (screenProps.width === pattern.w && screenProps.height === pattern.h) {
                        score += 15;
                        detections.push(pattern.name);
                    }
                }
            } catch (e) {}

        } catch (e) {
            score += 25;
            detections.push('vm_features_error: ' + e.message);
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectSandboxEscape() {
        let score = 0;
        const detections = [];
        try {
            const sandboxFilePaths = [
                '/system/bin/vboxservice',
                '/system/bin/vboxguest',
                '/usr/bin/vmware-toolbox-cmd',
                '/usr/bin/VBoxControl',
                '/Applications/VMware Fusion.app',
                '/Applications/VirtualBox.app',
                'C:\\Windows\\System32\\drivers\\vboxguest.sys',
                'C:\\Windows\\System32\\drivers\\VBoxMouse.sys',
                '/proc/self/status',
                '/proc/1/cgroup'
            ];

            const sandboxRegistryPaths = [
                'HKLM\\SYSTEM\\CurrentControlSet\\Services\\VBoxGuest',
                'HKLM\\SYSTEM\\CurrentControlSet\\Services\\VBoxMouse',
                'HKLM\\SYSTEM\\CurrentControlSet\\Services\\VBoxService',
                'HKLM\\SOFTWARE\\Oracle\\VirtualBox',
                'HKLM\\SOFTWARE\\VMware, Inc.\\VMware Tools'
            ];

            try {
                if (typeof process !== 'undefined') {
                    detections.push('node_env_detected');
                    score += 30;
                }
            } catch (e) {}

            try {
                const testFunc = new Function('return this')();
                if (testFunc && testFunc.require) {
                    detections.push('require_available');
                    score += 35;
                }
            } catch (e) {}

            try {
                if (typeof require !== 'undefined') {
                    detections.push('require_defined');
                    score += 40;
                }
            } catch (e) {}

            try {
                const testObj = {}.toString.call('test');
                if (testObj === '[object process]') {
                    detections.push('process_object');
                    score += 30;
                }
            } catch (e) {}

            try {
                if (typeof module !== 'undefined' && module.exports) {
                    detections.push('module_exports');
                    score += 25;
                }
            } catch (e) {}

            try {
                if (typeof __dirname !== 'undefined' || typeof __filename !== 'undefined') {
                    detections.push('node_path_vars');
                    score += 35;
                }
            } catch (e) {}

            try {
                if (typeof global !== 'undefined') {
                    detections.push('global_object');
                    score += 20;
                }
            } catch (e) {}

            try {
                const testElement = document.createElement('input');
                testElement.type = 'file';
                if (testElement.webkitdirectory !== undefined) {
                    detections.push('webkitdirectory_available');
                    score += 15;
                }
            } catch (e) {}

            try {
                const canvas = document.createElement('canvas');
                canvas.width = 10;
                canvas.height = 10;
                const ctx = canvas.getContext('2d');
                if (ctx) {
                    const imageData = ctx.getImageData(0, 0, 1, 1);
                    const isZero = Array.from(imageData.data).every(v => v === 0);
                    if (isZero) {
                        score += 25;
                        detections.push('sandbox_read_zero');
                    }
                }
            } catch (e) {}

            try {
                const start = performance.now();
                const testArr = new Array(10000);
                for (let i = 0; i < testArr.length; i++) {
                    testArr[i] = i;
                }
                testArr.sort();
                const end = performance.now();
                const sortTime = end - start;

                if (sortTime < 0.5) {
                    score += 20;
                    detections.push('sandbox_fast_sort');
                }
            } catch (e) {}

            try {
                const testDate = new Date();
                const dateStr = testDate.toString();
                if (/^(Mon|Tue|Wed|Thu|Fri|Sat|Sun)/.test(dateStr) === false) {
                    score += 15;
                    detections.push('sandbox_date_format');
                }
            } catch (e) {}

            try {
                if (typeof navigator !== 'undefined') {
                    const plugins = navigator.plugins || [];
                    const pluginNames = Array.from(plugins).map(p => p.name.toLowerCase());

                    const sandboxPlugins = ['vbox', 'vmware', 'virtual', 'sandbox'];
                    for (const name of pluginNames) {
                        for (const sandbox of sandboxPlugins) {
                            if (name.includes(sandbox)) {
                                score += 30;
                                detections.push(`sandbox_plugin:${sandbox}`);
                            }
                        }
                    }
                }
            } catch (e) {}

        } catch (e) {
            score += 20;
            detections.push('sandbox_escape_error: ' + e.message);
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
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
                },
                breakpoint: () => {
                    const start = Date.now();
                    debugger;
                    const end = Date.now();
                    return end - start > 100;
                },
                devToolsOrientation: () => {
                    return window.outerWidth - window.innerWidth > 150 ||
                           window.outerHeight - window.innerHeight > 150;
                }
            };

            for (const [checkName, checkFn] of Object.entries(devToolsChecks)) {
                try {
                    if (checkFn()) {
                        score += 25;
                        detections.push(`devtools_${checkName}`);
                    }
                } catch (e) {}
            }

            try {
                const getStackTrace = () => {
                    const err = new Error();
                    return err.stack || '';
                };
                const stack = getStackTrace();

                const debuggerPatterns = [
                    /debugger/i,
                    /devtools/i,
                    /__webdriver_evaluate/i,
                    /__selenium_evaluate/i,
                    /__fxdriver_evaluate/i,
                    /__driver_evaluate/i,
                    /__webdriver_script_fn/i
                ];

                for (const pattern of debuggerPatterns) {
                    if (pattern.test(stack)) {
                        score += 30;
                        detections.push(`stack_debugger:${pattern}`);
                    }
                }
            } catch (e) {}

            try {
                const timingSamples = [];
                for (let i = 0; i < 10; i++) {
                    const start = performance.now();
                    const dummy = [];
                    for (let j = 0; j < 1000; j++) {
                        dummy.push(j);
                    }
                    const end = performance.now();
                    timingSamples.push(end - start);
                }

                const avgTime = timingSamples.reduce((a, b) => a + b, 0) / timingSamples.length;
                const variance = timingSamples.reduce((a, b) => a + Math.pow(b - avgTime, 2), 0) / timingSamples.length;
                const stdDev = Math.sqrt(variance);

                detections.push(`timing_avg:${avgTime.toFixed(3)}`);
                detections.push(`timing_std:${stdDev.toFixed(3)}`);

                if (stdDev > 50) {
                    score += 20;
                    detections.push('timing_high_variance');
                }

                if (avgTime > 10) {
                    score += 25;
                    detections.push('timing_too_slow');
                }

                this.fingerprintComponents.timingStdDev = stdDev;
            } catch (e) {}

            try {
                let isDebugger = false;
                const originalLog = console.log;
                console.log = function(...args) {
                    if (args.length === 0) {
                        isDebugger = true;
                    }
                    return originalLog.apply(console, args);
                };

                console.log();
                console.log = originalLog;

                if (isDebugger) {
                    score += 20;
                    detections.push('console_empty_log');
                }
            } catch (e) {}

            try {
                const obj = {};
                const originalDefineProperty = Object.defineProperty;
                Object.defineProperty = function(obj, prop, descriptor) {
                    if (prop === 'devtools' || prop === 'Debugger') {
                        score += 35;
                        detections.push('define_property_debugger');
                    }
                    return originalDefineProperty.apply(this, arguments);
                };
            } catch (e) {}

            try {
                const testFn = () => {
                    const start = performance.now();
                    let result = 0;
                    for (let i = 0; i < 100; i++) {
                        result += Math.sqrt(i);
                    }
                    const end = performance.now();
                    return end - start;
                };

                const normalTime = testFn();
                const threshold = normalTime * 10;

                for (let i = 0; i < 5; i++) {
                    const start = performance.now();
                    void({});
                    const end = performance.now();
                    if (end - start > threshold) {
                        score += 20;
                        detections.push('execution_paused');
                        break;
                    }
                }
            } catch (e) {}

            try {
                if (typeof window !== 'undefined') {
                    const props = ['__webdriver_evaluate', '__selenium_evaluate', '__webdriver_script_fn'];
                    for (const prop of props) {
                        if (window[prop] !== undefined) {
                            score += 30;
                            detections.push(`debugger_prop:${prop}`);
                        }
                    }
                }
            } catch (e) {}

            try {
                const date1 = new Date();
                let counter = 0;
                while (new Date() - date1 < 100) {
                    counter++;
                }

                if (counter < 1000) {
                    score += 15;
                    detections.push('date_loop_inhibited');
                }
            } catch (e) {}

        } catch (e) {
            score += 20;
            detections.push('debugger_enhanced_error: ' + e.message);
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
            '163.', '164.', '165.', '166.', '167.', '168.', '169.', '170.',
            '171.', '172.', '173.', '174.', '175.', '176.', '177.', '178.',
            '179.', '180.', '181.', '182.', '183.', '184.', '185.', '186.',
            '187.', '188.', '189.', '190.', '191.', '192.', '193.', '194.',
            '195.', '196.', '197.', '198.', '199.', '200.', '204.', '207.',
            '208.', '209.', '210.', '211.', '212.', '213.', '214.', '215.',
            '216.', '217.', '218.', '219.', '220.', '221.', '222.', '223.',
            '224.', '225.', '226.', '227.', '228.', '229.', '230.', '231.',
            '232.', '233.', '234.', '235.', '236.', '237.', '238.', '239.'
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
            canvas.width = 100;
            canvas.height = 50;
            const ctx = canvas.getContext('2d');
            if (ctx) {
                ctx.textBaseline = 'top';
                ctx.font = '14px Arial';
                ctx.fillStyle = '#f60';
                ctx.fillRect(0, 0, 50, 50);
                ctx.fillStyle = '#069';
                ctx.fillText('fp', 10, 20);
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
        try {
            if (navigator.languages && navigator.languages.length > 0) {
                components.push('langs:' + navigator.languages.join(','));
            }
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
            timing_data: this.timingData
        };
    }
}

if (typeof window !== 'undefined') {
    window.EnvironmentDetectorEnhanced = EnvironmentDetectorEnhanced;
}
