class EnhancedEnvironmentDetector {
    constructor(options = {}) {
        this.options = Object.assign({
            apiBase: '/api/v1',
            sampleRate: 0.3,
            chainCount: 15,
            enableAll: true,
            sessionId: null,
            enableNetworkAnalysis: true,
            enableEmulatorDetection: true,
            enableAutomationDetection: true,
            fingerprintPrecision: 'high'
        }, options);
        this.results = {};
        this.riskScore = 0;
        this.detectionChain = [];
        this.detectionId = 'det_' + Date.now() + '_' + Math.random().toString(36).substr(2, 6);
        this.weights = {
            canvas: 10,
            webgl: 12,
            webgl2: 10,
            audio: 11,
            fonts: 9,
            webrtc_ip: 12,
            webdriver: 18,
            selenium: 20,
            puppeteer: 20,
            playwright: 20,
            chrome_runtime: 12,
            headless: 15,
            permissions: 8,
            plugins: 7,
            languages: 6,
            timezone: 7,
            screen: 5,
            hardware: 6,
            memory: 5,
            storage: 7,
            navigator: 6,
            window_props: 6,
            iframe: 8,
            notification: 5,
            battery: 5,
            media_devices: 6,
            connection: 7,
            adblock: 6,
            math: 5,
            gpu: 8,
            speech: 5,
            emulator: 18,
            virtualization: 16,
            container: 15,
            vpn: 14,
            tor: 18,
            proxy: 12,
            latency: 8,
            dns: 10,
            automation_framework: 20
        };
    }

    getDetectionMethods() {
        const methods = [
            'detectHeadless',
            'detectWebDriver',
            'detectPuppeteer',
            'detectPlaywright',
            'detectSelenium',
            'detectChromeRuntime',
            'detectPermissions',
            'detectPlugins',
            'detectLanguages',
            'detectTimezone',
            'detectScreen',
            'detectHardwareConcurrency',
            'detectDeviceMemory',
            'detectStorage',
            'detectCanvas',
            'detectWebGL',
            'detectWebGL2',
            'detectAudio',
            'detectFonts',
            'detectNavigatorProps',
            'detectWindowProps',
            'detectIframe',
            'detectNotification',
            'detectBattery',
            'detectMediaDevices',
            'detectWebRTCIP',
            'detectConnection',
            'detectAdBlock',
            'detectMathFingerprint',
            'detectGPUFingerprint',
            'detectSpeech',
            'detectVirtualization',
            'detectEmulator',
            'detectContainer',
            'detectVPNConnection',
            'detectTorNetwork',
            'detectNetworkLatency',
            'detectDNSAnalysis',
            'detectProxy',
            'detectAutomationFramework'
        ];
        return methods;
    }

    generateDetectionChain(count) {
        const allMethods = this.getDetectionMethods();
        const shuffled = [...allMethods].sort(() => Math.random() - 0.5);
        const selected = shuffled.slice(0, Math.min(count, allMethods.length));
        const methodAliases = {};
        selected.forEach((method, i) => {
            methodAliases[method] = 'chk_' + i.toString(36) + '_' + Math.random().toString(36).substr(2, 4);
        });
        return { selected, methodAliases };
    }

    async runChain() {
        const { selected, methodAliases } = this.generateDetectionChain(
            this.options.chainCount
        );
        this.detectionChain = selected;
        const chainResults = {};
        const startTime = performance.now();

        for (const method of selected) {
            try {
                const alias = methodAliases[method];
                const result = await this[method]();
                chainResults[alias] = result;
                this.results[method] = result;
            } catch (e) {
                const alias = methodAliases[method];
                chainResults[alias] = { detected: false, score: 0, error: e.message };
            }
        }

        const duration = performance.now() - startTime;
        this.riskScore = this.calculateRiskScore();

        return {
            detection_id: this.detectionId,
            chain: chainResults,
            chain_order: Object.values(methodAliases),
            risk_score: this.riskScore,
            duration_ms: Math.round(duration),
            timestamp: Date.now()
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

        const autoTools = ['detectWebDriver', 'detectPuppeteer', 'detectPlaywright', 'detectSelenium', 'detectAutomationFramework'];
        const autoDetected = autoTools.filter(m => {
            const r = this.results[m];
            return r && r.detected === true;
        }).length;

        if (autoDetected >= 3) {
            baseScore = Math.min(baseScore * 1.8 + 30, 100);
        } else if (autoDetected >= 2) {
            baseScore = Math.min(baseScore * 1.5 + 20, 100);
        } else if (autoDetected >= 1) {
            baseScore = Math.min(baseScore * 1.3 + 10, 100);
        }

        const proxyIndicators = ['detectWebRTCIP', 'detectConnection', 'detectVPNConnection', 'detectProxy'];
        const proxyAnomalies = proxyIndicators.filter(m => {
            const r = this.results[m];
            return r && r.score > 30;
        }).length;

        if (proxyAnomalies >= 3) {
            baseScore = Math.min(baseScore * 1.5 + 20, 100);
        } else if (proxyAnomalies >= 2) {
            baseScore = Math.min(baseScore * 1.3 + 15, 100);
        }

        const emulationIndicators = ['detectEmulator', 'detectVirtualization', 'detectContainer'];
        const emulationDetected = emulationIndicators.filter(m => {
            const r = this.results[m];
            return r && r.detected === true;
        }).length;

        if (emulationDetected >= 2) {
            baseScore = Math.min(baseScore * 1.4 + 15, 100);
        } else if (emulationDetected >= 1) {
            baseScore = Math.min(baseScore * 1.2 + 8, 100);
        }

        return Math.round(Math.min(Math.max(baseScore, 0), 100));
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
            if (window.outerWidth && window.outerWidth < 100) {
                score += 25;
                detections.push('abnormal_window_width');
            }
        } catch (e) {}
        
        try {
            if (window.devicePixelRatio && window.devicePixelRatio === 0) {
                score += 30;
                detections.push('zero_device_pixel_ratio');
            }
        } catch (e) {}
        
        try {
            if (typeof navigator.maxTouchPoints === 'number' && navigator.maxTouchPoints === 0) {
                const ua = navigator.userAgent || '';
                if (/mobile|android|iphone/i.test(ua)) {
                    score += 35;
                    detections.push('touch_discrepancy');
                }
            }
        } catch (e) {}
        
        try {
            if (navigator.webgl && !navigator.webgl.getExtension('WEBGL_debug_renderer_info')) {
                score += 30;
                detections.push('webgl_debug_blocked');
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
            'callSelenium', '__selenium', 'Selenium', '_selenium',
            'document.__selenium', ' Selenium', '__webdriver_script_func'
        ];
        
        for (const prop of wdProps) {
            if (window[prop] !== undefined) {
                score += 18;
                detections.push(prop);
            }
        }
        
        try {
            if (navigator.webdriver === true) {
                score += 35;
                detections.push('navigator.webdriver');
            }
        } catch (e) {}
        
        try {
            const el = document.createElement('div');
            el.setAttribute('onclick', 'return __webdriver_script_fn()');
            if (el.onclick !== null) {
                score += 12;
                detections.push('webdriver_script_fn');
            }
        } catch (e) {}
        
        try {
            const el = document.createElement('div');
            el.setAttribute('onmousemove', 'return __driver_evaluate()');
            if (el.onmousemove !== null) {
                score += 12;
                detections.push('driver_evaluate');
            }
        } catch (e) {}
        
        try {
            const keys = Object.keys(window);
            const seleniumKeys = keys.filter(k => /selenium|webdriver|__wd|__sel/i.test(k));
            if (seleniumKeys.length > 0) {
                score += 25;
                detections.push('selenium_keys_found:' + seleniumKeys.length);
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
                detections.push('cdc_marker');
            }
        } catch (e) {}
        
        try {
            if (document.$chrome_asyncScriptInfo) {
                score += 35;
                detections.push('chrome_async_script');
            }
        } catch (e) {}
        
        try {
            if (document.__webdriver_evaluate) {
                score += 40;
                detections.push('webdriver_evaluate');
            }
        } catch (e) {}
        
        try {
            const el = document.createElement('div');
            el.setAttribute('onpaste', 'return function(){throw new Error("puppeteer")}');
            if (el.onpaste !== null) {
                score += 25;
                detections.push('puppeteer_onpaste');
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
            if (/Chrome\/[\d.]+\s+Headless/i.test(userAgent)) {
                score += 55;
                detections.push('chrome_headless_ua');
            }
        } catch (e) {}
        
        try {
            if (window._puppeteer_globals !== undefined) {
                score += 40;
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
        
        try {
            if (window.navigator.permissions !== undefined) {
                const permission = await navigator.permissions.query({ name: 'notifications' }).catch(() => null);
                if (permission && permission.state === 'prompt' && navigator.webdriver) {
                    score += 40;
                    detections.push('puppeteer_permission_state');
                }
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
                window.__pw_resume__ !== undefined ||
                window.__pw_connect__ !== undefined) {
                score += 55;
                detections.push('playwright_global');
            }
        } catch (e) {}
        
        try {
            if (window.__playwright_unstripped__) {
                score += 60;
                detections.push('playwright_unstripped');
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
            const el = document.createElement('div');
            el.setAttribute('onmouseenter', 'return __pw_resume__()');
            if (el.onmouseenter !== null) {
                score += 30;
                detections.push('playwright_mouseenter');
            }
        } catch (e) {}
        
        try {
            const ua = navigator.userAgent || '';
            if (/playwright/i.test(ua)) {
                score += 60;
                detections.push('playwright_ua');
            }
            if (/Mozilla\/5.0.*Windows.*AppleWebKit\/[\d.]+.*\(KHTML, like Gecko\).*Chrome\/[\d.]+\s+Safari\/[\d.]+$/.test(ua)) {
                const browserKeys = Object.keys(window).filter(k => /__pw|pw_|playwright/i.test(k));
                if (browserKeys.length > 0) {
                    score += 65;
                    detections.push('playwright_browser_detected');
                }
            }
        } catch (e) {}
        
        try {
            if (navigator.plugins.length === 0 && navigator.mimeTypes.length === 0) {
                score += 45;
                detections.push('playwright_no_plugins');
            }
        } catch (e) {}
        
        try {
            const langs = navigator.languages;
            if (langs && langs.length === 1 && langs[0] === 'en-US') {
                score += 30;
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
                score += 35;
                detections.push('webdriver_attr');
            }
        } catch (e) {}
        
        try {
            const el = document.createElement('div');
            el.setAttribute('onmouseover', 'return Selenium.prototype.whatever');
            if (el.onmouseover !== null) {
                score += 20;
                detections.push('selenium_prototype');
            }
        } catch (e) {}
        
        try {
            const el = document.createElement('div');
            el.setAttribute('onkeydown', 'return selenium_executor.onkeydown');
            if (el.onkeydown !== null) {
                score += 20;
                detections.push('selenium_executor');
            }
        } catch (e) {}
        
        try {
            const ua = navigator.userAgent || '';
            if (/selenium/i.test(ua)) {
                score += 50;
                detections.push('selenium_ua');
            }
        } catch (e) {}
        
        try {
            if (window.__$webdriverAsyncExecutor !== undefined) {
                score += 30;
                detections.push('webdriver_async_executor');
            }
        } catch (e) {}
        
        try {
            const keys = Object.keys(window);
            const seleniumRelated = keys.filter(k => /selenium|webdriver|__sel/i.test(k));
            if (seleniumRelated.length > 3) {
                score += 45;
                detections.push('selenium_multiple_markers:' + seleniumRelated.length);
            }
        } catch (e) {}
        
        try {
            if (typeof navigator.product === 'string' && 
                navigator.product === 'Gecko' && 
                !/Firefox/i.test(navigator.userAgent || '')) {
                score += 35;
                detections.push('selenium_gecko_mismatch');
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
                    score += 35;
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
                { name: ' WDIO', pattern: /wdio/i, window: '_WDIO' },
                { name: 'splinter', pattern: /splinter/i, window: 'Splinter' },
                { name: 'mechanize', pattern: /mechanize/i, window: 'Mechanize' }
            ];
            
            for (const ap of automationPatterns) {
                if (window[ap.window] !== undefined) {
                    score += 50;
                    detections.push('framework_window:' + ap.name);
                }
                if (ap.pattern.test(navigator.userAgent || '')) {
                    score += 45;
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
            
            if (automationKeys.length > 2) {
                score += 40;
                detections.push('automation_keys:' + automationKeys.length);
            }
        } catch (e) {}
        
        try {
            const ua = navigator.userAgent || '';
            if (/automation|bot|crawler|spider|scraper/i.test(ua)) {
                score += 60;
                detections.push('automation_ua_detected');
            }
        } catch (e) {}
        
        try {
            const startTime = performance.now();
            await new Promise(resolve => setTimeout(resolve, 0));
            const elapsed = performance.now() - startTime;
            if (elapsed < 0.5) {
                score += 30;
                detections.push('unrealistic_timing');
            }
        } catch (e) {}
        
        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectCanvas() {
        let score = 0;
        const detections = [];
        
        try {
            const canvas = document.createElement('canvas');
            canvas.width = 280;
            canvas.height = 80;
            const ctx = canvas.getContext('2d');
            
            if (!ctx) {
                score += 45;
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
            ctx.fillText('abcdefghijklmnopqrstuvwxyz', 4, 70);

            const dataURL = canvas.toDataURL();
            const dataURL2 = canvas.toDataURL();
            
            if (dataURL !== dataURL2) {
                score += 30;
                detections.push('canvas_unstable');
            }

            const imageData = ctx.getImageData(0, 0, 10, 10);
            const pixelSum = Array.from(imageData.data.slice(0, 40)).reduce((a, b) => a + b, 0);
            
            if (pixelSum === 0) {
                score += 25;
                detections.push('canvas_empty_readback');
            }

            const allZeros = Array.from(imageData.data).every(v => v === 0);
            if (allZeros) {
                score += 40;
                detections.push('canvas_all_black');
            }
        } catch (e) {
            score += 40;
            detections.push('canvas_error');
        }
        
        try {
            const canvas2 = document.createElement('canvas');
            canvas2.width = 200;
            canvas2.height = 100;
            const ctx2 = canvas2.getContext('webgl') || canvas2.getContext('experimental-webgl');
            
            if (ctx2) {
                const buffer = ctx2.createBuffer();
                ctx2.bindBuffer(ctx2.ARRAY_BUFFER, buffer);
                ctx2.bufferData(ctx2.ARRAY_BUFFER, new Float32Array([1,2,3,4,5,6,7,8,9,10]), ctx2.STATIC_DRAW);
                
                const ext = ctx2.getExtension('WEBGL_debug_renderer_info');
                if (ext) {
                    const renderer = ctx2.getParameter(ext.UNMASKED_RENDERER_WEBGL) || '';
                    if (/swiftshader|llvmpipe|mesa|software/i.test(renderer)) {
                        score += 35;
                        detections.push('software_rendering_canvas');
                    }
                }
            }
        } catch (e) {}
        
        try {
            const testCanvas = document.createElement('canvas');
            testCanvas.width = 100;
            testCanvas.height = 100;
            const testCtx = testCanvas.getContext('2d');
            
            testCtx.fillStyle = '#ff0000';
            testCtx.fillRect(0, 0, 50, 50);
            testCtx.fillStyle = '#0000ff';
            testCtx.fillRect(50, 0, 50, 50);
            
            const data = testCtx.getImageData(0, 0, 100, 100).data;
            let nonZero = 0;
            for (let i = 0; i < data.length; i += 4) {
                if (data[i] !== 0 || data[i+1] !== 0 || data[i+2] !== 0) {
                    nonZero++;
                }
            }
            
            if (nonZero === 0) {
                score += 50;
                detections.push('canvas_fully_blocked');
            }
        } catch (e) {}
        
        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectWebGL() {
        let score = 0;
        const detections = [];
        
        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
            
            if (!gl) {
                score += 45;
                detections.push('no_webgl');
                return { detected: true, score: Math.min(score, 100), detections };
            }
            
            const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
            if (debugInfo) {
                const vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
                const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                
                if (!vendor || !renderer) {
                    score += 20;
                    detections.push('webgl_no_vendor');
                }
                
                if (/swiftshader|llvmpipe|mesa|virtual|google\s*inc|software/i.test(renderer || '')) {
                    score += 40;
                    detections.push('software_renderer');
                }
                
                if (/vmware|virtualbox|parallels|qemu/i.test(renderer || '')) {
                    score += 50;
                    detections.push('vm_renderer_detected');
                }
            } else {
                score += 25;
                detections.push('no_webgl_debug');
            }
            
            const maxTexSize = gl.getParameter(gl.MAX_TEXTURE_SIZE);
            if (maxTexSize <= 1024) {
                score += 20;
                detections.push('small_tex_size');
            }
            
            const maxVertAttribs = gl.getParameter(gl.MAX_VERTEX_ATTRIBS);
            if (maxVertAttribs <= 8) {
                score += 15;
                detections.push('few_vertex_attribs');
            }
            
            const aliasedRange = gl.getParameter(gl.ALIASED_LINE_WIDTH_RANGE);
            if (aliasedRange && aliasedRange[1] <= 1) {
                score += 15;
                detections.push('aliased_line_only');
            }
            
            const shaderPrecision = gl.getShaderPrecisionFormat(gl.FRAGMENT_SHADER, gl.HIGH_FLOAT);
            if (shaderPrecision && shaderPrecision.precision < 16) {
                score += 20;
                detections.push('low_shader_precision');
            }
            
            const ext = gl.getExtension('EXT_texture_filter_anisotropic');
            if (!ext) {
                score += 8;
                detections.push('no_anisotropic');
            }
            
            const supportedExts = gl.getSupportedExtensions();
            if (supportedExts && supportedExts.length < 10) {
                score += 15;
                detections.push('few_webgl_extensions:' + (supportedExts ? supportedExts.length : 0));
            }
            
            if (supportedExts && supportedExts.length > 50) {
                score += 5;
                detections.push('many_webgl_extensions');
            }
            
            const vertexTextureUnits = gl.getParameter(gl.MAX_VERTEX_TEXTURE_IMAGE_UNITS);
            if (vertexTextureUnits === 0) {
                score += 10;
                detections.push('no_vertex_texture_units');
            }
            
        } catch (e) {
            score += 40;
            detections.push('webgl_error');
        }
        
        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectWebGL2() {
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
                if (/swiftshader|llvmpipe|mesa|virtual|software/i.test(renderer || '')) {
                    score += 35;
                    detections.push('webgl2_software_renderer');
                }
                if (/vmware|virtualbox|parallels|qemu/i.test(renderer || '')) {
                    score += 50;
                    detections.push('webgl2_vm_renderer');
                }
            }
            
            const maxTexSize = gl2.getParameter(gl2.MAX_TEXTURE_SIZE);
            if (maxTexSize <= 1024) {
                score += 15;
                detections.push('webgl2_small_tex');
            }
            
            const supportedExts = gl2.getSupportedExtensions();
            if (supportedExts && supportedExts.length < 5) {
                score += 15;
                detections.push('few_webgl2_extensions:' + (supportedExts ? supportedExts.length : 0));
            }
            
            const max3DTextureSize = gl2.getParameter(gl2.MAX_3D_TEXTURE_SIZE);
            if (max3DTextureSize < 512) {
                score += 20;
                detections.push('webgl2_limited_3d_texture');
            }
            
            const maxRenderbufferSize = gl2.getParameter(gl2.MAX_RENDERBUFFER_SIZE);
            if (maxRenderbufferSize < 4096) {
                score += 15;
                detections.push('webgl2_small_renderbuffer');
            }
            
        } catch (e) {
            score += 25;
            detections.push('webgl2_error');
        }
        
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectAudio() {
        let score = 0;
        const detections = [];
        
        try {
            const AudioContext = window.OfflineAudioContext || window.webkitOfflineAudioContext;
            
            if (!AudioContext) {
                score += 35;
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
            compressor.attack.setValueAtTime(0, ctx.currentTime);
            compressor.release.setValueAtTime(0.25, ctx.currentTime);
            
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

            if (renderTime < 5) {
                score += 25;
                detections.push('audio_render_too_fast');
            }
            
            if (renderTime > 1000) {
                score += 20;
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
                score += 35;
                detections.push('audio_silent');
            }
            
            let uniqueValues = new Set();
            for (let i = 0; i < Math.min(1000, channelData.length); i++) {
                uniqueValues.add(channelData[i].toFixed(6));
            }
            if (uniqueValues.size < 10) {
                score += 30;
                detections.push('audio_low_entropy');
            }
            
            const stats = ctx.sampleRate;
            if (stats !== 44100 && stats !== 48000) {
                score += 15;
                detections.push('audio_unusual_sample_rate:' + stats);
            }
            
        } catch (e) {
            score += 35;
            detections.push('audio_error');
        }
        
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectFonts() {
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
            'Century Gothic', 'Consolas', 'Monaco', 'Menlo'
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
                score += 30;
                detections.push('too_few_fonts:' + fontCount);
            }
            
            if (fontCount < 8) {
                score += 15;
                detections.push('limited_fonts:' + fontCount);
            }
            
            if (detectedFonts.length === 0) {
                score += 45;
                detections.push('no_detected_fonts');
            }
            
            const uncommonFonts = detectedFonts.filter(f => 
                !['Arial', 'Helvetica', 'Times New Roman', 'Courier New', 'Georgia'].includes(f)
            );
            if (uncommonFonts.length === 0 && fontCount > 0) {
                score += 20;
                detections.push('only_system_fonts');
            }
            
        } catch (e) {
            score += 30;
            detections.push('font_detection_error');
        }
        
        return { detected: score > 30, score: Math.min(score, 100), detections };
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
                { pattern: /x86.*android/i, name: 'x86_android' }
            ];
            
            for (const emu of emulatorPatterns) {
                if (emu.pattern.test(ua)) {
                    score += 50;
                    detections.push(emu.name);
                }
            }
            
            if (/iPhone.*CPU.*OS/i.test(ua)) {
                const platform = navigator.platform || '';
                if (!/iPhone|iPad/i.test(platform)) {
                    score += 45;
                    detections.push('ios_platform_mismatch');
                }
            }
            
            if (navigator.maxTouchPoints === 0) {
                const ua = navigator.userAgent || '';
                if (/mobile|android|iphone|ipad/i.test(ua)) {
                    score += 40;
                    detections.push('mobile_no_touch');
                }
            }
            
            if (navigator.maxTouchPoints && navigator.maxTouchPoints > 10) {
                score += 25;
                detections.push('unusual_touch_points:' + navigator.maxTouchPoints);
            }
            
            if (navigator.platform) {
                const platform = navigator.platform.toLowerCase();
                if (platform.includes('linux') && /mobile|android/i.test(ua)) {
                    score += 45;
                    detections.push('linux_android_emulator');
                }
            }
            
            try {
                const width = screen.width;
                const height = screen.height;
                if (width === 320 && height === 480) {
                    score += 35;
                    detections.push('iphone_3gs_resolution');
                }
                if (width === 375 && height === 667) {
                    score += 30;
                    detections.push('iphone_6_resolution');
                }
                if (width === 414 && height === 896) {
                    score += 30;
                    detections.push('iphone_xr_resolution');
                }
            } catch (e) {}
            
            try {
                if (screen.width === screen.availWidth && screen.height === screen.availHeight) {
                    score += 25;
                    detections.push('fullscreen_emulator');
                }
            } catch (e) {}
            
            try {
                const gl = document.createElement('canvas').getContext('webgl');
                if (gl) {
                    const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                    if (debugInfo) {
                        const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) || '';
                        if (/android|adr|goldfish|llvmpipe/i.test(renderer)) {
                            score += 40;
                            detections.push('android_emulator_renderer');
                        }
                    }
                }
            } catch (e) {}
            
        } catch (e) {
            score += 25;
            detections.push('emulator_error');
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
                { pattern: /container/i, name: 'container_marker' }
            ];
            
            for (const vm of vmPatterns) {
                if (vm.pattern.test(ua)) {
                    score += 45;
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
                            { pattern: /virtual|gpu|graphics/i, name: 'virtual_gpu' }
                        ];
                        for (const r of vmRendererPatterns) {
                            if (r.pattern.test(renderer)) {
                                score += 50;
                                detections.push(r.name);
                            }
                        }
                    }
                }
            } catch (e) {}
            
            if (navigator.hardwareConcurrency) {
                if (navigator.hardwareConcurrency === 1) {
                    score += 30;
                    detections.push('single_core_vm');
                }
                if (navigator.hardwareConcurrency === 2) {
                    score += 20;
                    detections.push('dual_core_vm');
                }
                if (navigator.hardwareConcurrency > 16) {
                    score += 20;
                    detections.push('high_cores_vm');
                }
            }
            
            if (navigator.deviceMemory) {
                if (navigator.deviceMemory <= 0.5) {
                    score += 25;
                    detections.push('minimal_memory_vm');
                }
                if (navigator.deviceMemory > 64) {
                    score += 15;
                    detections.push('high_memory_unusual');
                }
            }
            
            try {
                const cpuProps = ['加速', 'CPUID', 'Hypervisor', 'hypervisor'];
                let cpuScore = 0;
                for (const prop of cpuProps) {
                    if (navigator.userAgent.includes(prop)) {
                        cpuScore += 30;
                        detections.push('cpu_' + prop);
                    }
                }
                if (cpuScore > 0) {
                    score += cpuScore;
                }
            } catch (e) {}
            
        } catch (e) {
            score += 30;
            detections.push('virtualization_error');
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
                { pattern: /containerd/i, name: 'containerd' }
            ];
            
            for (const container of containerPatterns) {
                if (container.pattern.test(ua)) {
                    score += 55;
                    detections.push(container.name);
                }
            }
            
            try {
                const response = await fetch('/.dockerenv', { method: 'HEAD' }).catch(() => null);
                if (response && response.ok) {
                    score += 65;
                    detections.push('dockerenv_file');
                }
            } catch (e) {}
            
            try {
                if (navigator.storage && navigator.storage.estimate) {
                    const estimate = await navigator.storage.estimate();
                    if (estimate.quota === 0) {
                        score += 40;
                        detections.push('zero_quota_container');
                    }
                    if (estimate.quota && estimate.quota < 100000000) {
                        score += 30;
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
                        if (data[i] === 0 && data[i+1] === 0 && data[i+2] === 0) {
                            blackCount++;
                        }
                    }
                    if (blackCount > data.length / 4 * 0.8) {
                        score += 35;
                        detections.push('canvas_container_artifact');
                    }
                }
            } catch (e) {}
            
        } catch (e) {
            score += 30;
            detections.push('container_error');
        }
        
        return { detected: score > 45, score: Math.min(score, 100), detections };
    }

    async detectVPNConnection() {
        let score = 0;
        const detections = [];
        
        try {
            const rtcPeerConnection = window.RTCPeerConnection || window.webkitRTCPeerConnection;
            if (rtcPeerConnection) {
                const ips = new Set();
                const pc = new rtcPeerConnection({ 
                    iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] 
                });
                pc.createDataChannel('');
                
                try {
                    const offer = await pc.createOffer();
                    await pc.setLocalDescription(offer);
                    const sdp = pc.localDescription.sdp;
                    
                    const lines = sdp.split('\n');
                    for (const line of lines) {
                        if (line.indexOf('candidate') > -1) {
                            const parts = line.split(' ');
                            if (parts[4] && parts[4] !== '0.0.0.0') {
                                const ip = parts[4];
                                if (!ip.startsWith('192.168.') && !ip.startsWith('10.') && !ip.startsWith('172.')) {
                                    score += 55;
                                    detections.push('external_ip_detected');
                                }
                            }
                        }
                    }
                } catch (e) {}
                
                pc.close();
            }
            
            if (navigator.connection) {
                const conn = navigator.connection;
                if (conn.type === 'vpn' || conn.type === 'pptp' || conn.type === 'tunnel') {
                    score += 60;
                    detections.push('vpn_connection_type');
                }
                if (conn.effectiveType === 'slow-2g' || conn.effectiveType === '2g') {
                    score += 25;
                    detections.push('slow_connection_vpn');
                }
            }
            
            try {
                const startTime = Date.now();
                await fetch('/api/v1/health', { method: 'HEAD', cache: 'no-cache' }).catch(() => null);
                const latency = Date.now() - startTime;
                if (latency > 5000) {
                    score += 30;
                    detections.push('high_latency_vpn');
                }
            } catch (e) {}
            
        } catch (e) {
            score += 20;
            detections.push('vpn_error');
        }
        
        return { detected: score > 45, score: Math.min(score, 100), detections };
    }

    async detectTorNetwork() {
        let score = 0;
        const detections = [];
        
        try {
            const ua = navigator.userAgent || '';
            if (/tor|onion/i.test(ua)) {
                score += 65;
                detections.push('tor_user_agent');
            }
            
            const rtcPeerConnection = window.RTCPeerConnection || window.webkitRTCPeerConnection;
            if (rtcPeerConnection) {
                const pc = new rtcPeerConnection({ 
                    iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] 
                });
                pc.createDataChannel('');
                
                try {
                    const offer = await pc.createOffer();
                    await pc.setLocalDescription(offer);
                    const sdp = pc.localDescription.sdp;
                    
                    if (/tls|inject_host_overwrite/i.test(sdp)) {
                        score += 55;
                        detections.push('tor_sdp_signature');
                    }
                    
                    if (/candidate.*tcp|tcptype/i.test(sdp)) {
                        score += 40;
                        detections.push('tor_tcp_candidate');
                    }
                } catch (e) {}
                
                pc.close();
            }
            
            if (navigator.connection) {
                const conn = navigator.connection;
                if (conn.rtt && conn.rtt > 500) {
                    score += 45;
                    detections.push('high_rtt_tor');
                }
            }
            
        } catch (e) {
            score += 30;
            detections.push('tor_error');
        }
        
        return { detected: score > 50, score: Math.min(score, 100), detections };
    }

    async detectNetworkLatency() {
        let score = 0;
        const detections = [];
        
        try {
            const endpoints = [
                { url: 'https://www.google.com/favicon.ico', name: 'google' },
                { url: 'https://www.cloudflare.com/favicon.ico', name: 'cloudflare' },
                { url: 'https://www.microsoft.com/favicon.ico', name: 'microsoft' }
            ];
            
            const latencies = [];
            
            for (const endpoint of endpoints) {
                try {
                    const startTime = performance.now();
                    await fetch(endpoint.url + '?t=' + Date.now(), { 
                        method: 'HEAD',
                        mode: 'no-cors',
                        cache: 'no-cache' 
                    }).catch(() => null);
                    const latency = performance.now() - startTime;
                    latencies.push(latency);
                } catch (e) {}
            }
            
            if (latencies.length > 0) {
                const avgLatency = latencies.reduce((a, b) => a + b, 0) / latencies.length;
                
                if (avgLatency > 1000) {
                    score += 50;
                    detections.push('very_high_latency:' + Math.round(avgLatency));
                } else if (avgLatency > 500) {
                    score += 35;
                    detections.push('high_latency:' + Math.round(avgLatency));
                } else if (avgLatency > 200) {
                    score += 20;
                    detections.push('moderate_latency:' + Math.round(avgLatency));
                }
                
                const variance = latencies.reduce((sum, lat) => {
                    return sum + Math.pow(lat - avgLatency, 2);
                }, 0) / latencies.length;
                
                if (variance > 100000) {
                    score += 30;
                    detections.push('high_latency_variance');
                }
            } else {
                score += 40;
                detections.push('network_latency_check_failed');
            }
            
            try {
                const startTime = performance.now();
                await new Promise(resolve => setTimeout(resolve, 100));
                const elapsed = performance.now() - startTime;
                
                if (elapsed > 200 || elapsed < 50) {
                    score += 25;
                    detections.push('timing_anomaly:' + Math.round(elapsed));
                }
            } catch (e) {}
            
        } catch (e) {
            score += 20;
            detections.push('network_latency_error');
        }
        
        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectDNSAnalysis() {
        let score = 0;
        const detections = [];
        
        try {
            const dnsProviders = {
                'dns.google': ['8.8.8.8', '8.8.4.4'],
                'dns.cloudflare': ['1.1.1.1', '1.0.0.1'],
                'dns.opendns': ['208.67.222.222', '208.67.220.220'],
                'dns.quad9': ['9.9.9.9']
            };
            
            const rtcPeerConnection = window.RTCPeerConnection || window.webkitRTCPeerConnection;
            if (rtcPeerConnection) {
                const pc = new rtcPeerConnection({
                    iceServers: [
                        { urls: 'stun:dns.google' },
                        { urls: 'stun:dns.cloudflare' }
                    ]
                });
                pc.createDataChannel('');
                
                try {
                    const offer = await pc.createOffer();
                    await pc.setLocalDescription(offer);
                    const sdp = pc.localDescription.sdp;
                    
                    if (/srflx|prflx|relay/i.test(sdp)) {
                        score += 35;
                        detections.push('non_host_candidates');
                    }
                    
                    const candidateCount = (sdp.match(/candidate:/g) || []).length;
                    if (candidateCount < 2) {
                        score += 25;
                        detections.push('minimal_dns_candidates');
                    }
                } catch (e) {}
                
                pc.close();
            }
            
            try {
                const domain = 'cdn.' + Date.now() + '.test.com';
                const startTime = Date.now();
                try {
                    await fetch('https://' + domain, { 
                        method: 'HEAD',
                        mode: 'no-cors' 
                    }).catch(() => null);
                } catch (e) {}
                const resolveTime = Date.now() - startTime;
                
                if (resolveTime > 3000) {
                    score += 40;
                    detections.push('slow_dns_resolution');
                } else if (resolveTime > 1000) {
                    score += 25;
                    detections.push('moderate_dns_delay');
                }
            } catch (e) {}
            
            try {
                if (!navigator.onLine) {
                    score += 30;
                    detections.push('offline_but_online_api');
                }
            } catch (e) {}
            
        } catch (e) {
            score += 20;
            detections.push('dns_analysis_error');
        }
        
        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectProxy() {
        let score = 0;
        const detections = [];
        
        try {
            if (navigator.connection) {
                const conn = navigator.connection;
                if (conn.type === 'proxy' || conn.type === 'socks') {
                    score += 60;
                    detections.push('proxy_connection_type');
                }
                if (conn.type === 'vpn') {
                    score += 50;
                    detections.push('vpn_proxy');
                }
            }
            
            try {
                const rtcPeerConnection = window.RTCPeerConnection || window.webkitRTCPeerConnection;
                if (rtcPeerConnection) {
                    const pc = new rtcPeerConnection({
                        iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
                    });
                    pc.createDataChannel('');
                    
                    try {
                        const offer = await pc.createOffer();
                        await pc.setLocalDescription(offer);
                        const sdp = pc.localDescription.sdp;
                        
                        const candidates = sdp.split('\n').filter(line => line.includes('candidate'));
                        let relayCount = 0;
                        
                        for (const candidate of candidates) {
                            if (/relay|prflx|srflx/i.test(candidate)) {
                                relayCount++;
                            }
                        }
                        
                        if (relayCount > 0) {
                            score += 45;
                            detections.push('relay_candidates:' + relayCount);
                        }
                        
                        if (candidates.length > 10) {
                            score += 30;
                            detections.push('many_ice_candidates');
                        }
                    } catch (e) {}
                    
                    pc.close();
                }
            } catch (e) {}
            
            try {
                const testProxy = await fetch('https://ipinfo.io/json', {
                    method: 'GET',
                    headers: { 'Accept': 'application/json' }
                }).then(r => r.json()).catch(() => null);
                
                if (testProxy) {
                    if (testProxy.proxy === true || testProxy.proxy === 'true') {
                        score += 55;
                        detections.push('proxy_api_detected');
                    }
                    if (testProxy.hosting === true || testProxy.hosting === 'true') {
                        score += 45;
                        detections.push('hosting_detected');
                    }
                }
            } catch (e) {}
            
            try {
                const startTime = performance.now();
                await fetch('/api/v1/health', { method: 'HEAD' }).catch(() => null);
                const elapsed = performance.now() - startTime;
                
                if (elapsed > 3000) {
                    score += 35;
                    detections.push('high_fetch_latency_proxy');
                }
            } catch (e) {}
            
        } catch (e) {
            score += 20;
            detections.push('proxy_error');
        }
        
        return { detected: score > 40, score: Math.min(score, 100), detections };
    }

    async detectChromeRuntime() {
        let score = 0;
        const detections = [];
        
        try {
            if (window.chrome) {
                if (window.chrome.runtime === undefined) {
                    score += 25;
                    detections.push('chrome_runtime_missing');
                } else if (window.chrome.runtime && window.chrome.runtime.id === undefined) {
                    score += 15;
                    detections.push('chrome_runtime_no_id');
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
                if (window.chrome.webstore === undefined) {
                    score += 10;
                    detections.push('chrome_webstore_missing');
                }
            } else {
                if (!/Edge|Edg|Firefox|Safari|iOS|Android/i.test(navigator.userAgent || '')) {
                    score += 40;
                    detections.push('no_chrome_no_alt');
                }
            }
        } catch (e) {
            score += 30;
            detections.push('chrome_check_error');
        }
        
        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectPermissions() {
        let score = 0;
        const detections = [];
        
        try {
            if (navigator.permissions && navigator.permissions.query) {
                const permNames = ['notifications', 'geolocation', 'camera', 'microphone', 'midi', 'persistent-storage'];
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
                    detections.push('most_permissions_denied:' + deniedCount);
                }
                
                const promptCount = permChecks.filter(p => p.state === 'prompt').length;
                if (promptCount >= 5) {
                    score += 10;
                    detections.push('all_permissions_prompt');
                }
            } else {
                score += 25;
                detections.push('permissions_api_missing');
            }
        } catch (e) {
            score += 30;
            detections.push('permissions_error');
        }
        
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectPlugins() {
        let score = 0;
        const detections = [];
        
        try {
            const plugins = navigator.plugins;
            if (!plugins || plugins.length === 0) {
                score += 30;
                detections.push('no_plugins');
            } else {
                const commonPlugins = ['PDF Viewer', 'Chrome PDF Viewer', 'Chromium PDF Viewer',
                    'Microsoft Edge PDF Viewer', 'WebKit built-in PDF'];
                const hasPDF = Array.from(plugins).some(p =>
                    commonPlugins.some(cp => p.name.includes(cp))
                );
                if (!hasPDF) {
                    score += 15;
                    detections.push('no_pdf_plugin');
                }
                if (plugins.length < 3) {
                    score += 15;
                    detections.push('too_few_plugins:' + plugins.length);
                }
                if (plugins.length > 15) {
                    score += 8;
                    detections.push('too_many_plugins:' + plugins.length);
                }
            }
        } catch (e) {
            score += 35;
            detections.push('plugins_access_error');
        }
        
        return { detected: score > 30, score: Math.min(score, 100), detections };
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
                if (langs.length === 1 && langs[0] === 'en-US') {
                    score += 15;
                    detections.push('single_default_language');
                }
            }
        } catch (e) {
            score += 35;
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
        
        return { detected: score > 30, score: Math.min(score, 100), detections };
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
            if ('isExtended' in screen && screen.isExtended === undefined) {
                score += 15;
                detections.push('screen_extended_missing');
            }
            if (availWidth === 0 || availHeight === 0) {
                score += 20;
                detections.push('zero_avail_size');
            }
            if (width === height && width > 1000) {
                score += 20;
                detections.push('unusual_screen_ratio');
            }
        } catch (e) {
            score += 35;
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
                score += 35;
                detections.push('no_concurrency');
            } else if (c <= 1) {
                score += 30;
                detections.push('single_core');
            } else if (c > 64) {
                score += 25;
                detections.push('unrealistic_cores:' + c);
            }
        } catch (e) {
            score += 35;
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
                score += 25;
                detections.push('no_device_memory');
            } else if (mem <= 0.25) {
                score += 30;
                detections.push('low_memory');
            } else if (mem > 64) {
                score += 20;
                detections.push('unrealistic_memory:' + mem);
            }
        } catch (e) {
            score += 25;
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
            } else {
                score += 15;
                detections.push('storage_api_missing');
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
        
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectWebRTCIP() {
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
                    { urls: 'stun:stun2.l.google.com:19302' }
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
                        ips.add(parts[4]);
                        if (parts[7] !== 'host') {
                            detections.push('relay_ip:' + parts[4]);
                        }
                    }
                }
            }
            pc.close();
            
            if (ips.size > 1) {
                const ipsArr = Array.from(ips);
                const privateIPs = ipsArr.filter(ip =>
                    ip.startsWith('10.') ||
                    ip.startsWith('172.16.') || ip.startsWith('172.31.') ||
                    ip.startsWith('192.168.')
                );
                const publicIPs = ipsArr.filter(ip => !privateIPs.includes(ip));
                
                if (publicIPs.length > 0) {
                    detections.push('public_ip_detected');
                    if (privateIPs.length > 0) {
                        score += 25;
                        detections.push('vpn_possible');
                    }
                }
            }
            
            if (ips.size === 0) {
                score += 30;
                detections.push('no_ip_candidates');
            }
            
        } catch (e) {
            score += 20;
            detections.push('webrtc_error');
        }
        
        return { detected: score > 20, score: Math.min(score, 100), detections };
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
            } else {
                score += 15;
                detections.push('no_enumerate_devices');
            }
        } catch (e) {
            score += 20;
            detections.push('media_devices_error');
        }
        
        try {
            if (!navigator.credentials || !navigator.credentials.preventSilentAccess) {
                score += 8;
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
            if (typeof navigator.getBattery === 'function') {
                const battery = await navigator.getBattery().catch(() => null);
                if (battery && battery.charging === undefined) {
                    score += 15;
                    detections.push('battery_no_charging');
                }
            }
        } catch (e) {}
        
        try {
            if (navigator.product === 'Gecko' && !/Firefox/i.test(navigator.userAgent || '')) {
                score += 25;
                detections.push('gecko_no_firefox');
            }
        } catch (e) {}
        
        try {
            if (navigator.vendor === '' && navigator.product === 'Gecko') {
            } else if (navigator.vendor === '') {
                score += 15;
                detections.push('empty_vendor');
            }
        } catch (e) {}
        
        return { detected: score > 30, score: Math.min(score, 100), detections };
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
            if (window.openDatabase === undefined) {
                score += 8;
                detections.push('no_opendatabase');
            }
        } catch (e) {}
        
        try {
            if (window.indexedDB === undefined) {
                score += 15;
                detections.push('no_indexeddb');
            }
        } catch (e) {}
        
        try {
            if (typeof window.postMessage !== 'function') {
                score += 25;
                detections.push('no_postmessage');
            }
        } catch (e) {}
        
        try {
            if (window.screenTop === undefined || window.screenLeft === undefined) {
                score += 8;
                detections.push('no_screen_edge');
            }
        } catch (e) {}
        
        return { detected: score > 30, score: Math.min(score, 100), detections };
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
        
        try {
            const frameEl = document.createElement('iframe');
            frameEl.style.display = 'none';
            frameEl.sandbox = 'allow-scripts';
            document.body.appendChild(frameEl);
            const frameWin = frameEl.contentWindow;
            if (frameWin && frameWin.document) {}
            document.body.removeChild(frameEl);
        } catch (e) {
            score += 20;
            detections.push('iframe_access_error');
        }
        
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectNotification() {
        let score = 0;
        const detections = [];
        
        try {
            if ('Notification' in window) {
                if (Notification.permission === 'denied') {
                    score += 8;
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
                        score += 20;
                        detections.push('battery_props_missing');
                    }
                    if (battery.level === 0 && battery.charging === false) {
                        score += 8;
                        detections.push('battery_dead_not_charging');
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
        
        return { detected: score > 25, score: Math.min(score, 100), detections };
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
                    score += 50;
                    detections.push('vpn_detected');
                }
                if (conn.type === 'proxy') {
                    score += 50;
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
            el.className = 'adsbox adsponsor';
            el.id = 'google_ads_frame';
            el.style.cssText = 'position:absolute;left:-9999px;top:-9999px;width:1px;height:1px';
            document.body.appendChild(el);
            
            if (el.offsetHeight === 0 || el.offsetWidth === 0) {
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
        
        return { detected: score > 20, score: Math.min(score, 100), detections };
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
                if (maxViewport && (maxViewport[0] < 2048 || maxViewport[1] < 2048)) {
                    score += 15;
                    detections.push('limited_viewport');
                }
            }
        } catch (e) {
            score += 15;
            detections.push('gpu_error');
        }
        
        return { detected: score > 20, score: Math.min(score, 100), detections };
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
        
        return { detected: score > 15, score: Math.min(score, 100), detections };
    }

    async collectEnhancedEnvironmentData() {
        const data = {};
        
        try {
            data.canvasHash = await this.generateCanvasHash();
        } catch (e) {
            data.canvasHash = '';
        }
        
        try {
            data.webglHash = await this.generateWebGLHash();
        } catch (e) {
            data.webglHash = '';
        }
        
        try {
            data.webglRenderer = this.getWebGLRenderer();
        } catch (e) {
            data.webglRenderer = '';
        }
        
        try {
            data.webglVendor = this.getWebGLVendor();
        } catch (e) {
            data.webglVendor = '';
        }
        
        try {
            data.audioHash = await this.generateAudioHash();
        } catch (e) {
            data.audioHash = '';
        }
        
        try {
            data.fonts = await this.detectFontsList();
        } catch (e) {
            data.fonts = [];
        }
        
        try {
            data.plugins = Array.from(navigator.plugins || []).map(p => p.name);
        } catch (e) {
            data.plugins = [];
        }
        
        try {
            data.languages = navigator.languages || [navigator.language];
        } catch (e) {
            data.languages = [];
        }
        
        try {
            data.timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
        } catch (e) {
            data.timezone = '';
        }
        
        try {
            data.screen = {
                width: screen.width,
                height: screen.height,
                colorDepth: screen.colorDepth,
                pixelRatio: window.devicePixelRatio,
                availWidth: screen.availWidth,
                availHeight: screen.availHeight
            };
        } catch (e) {
            data.screen = {};
        }
        
        try {
            data.hardware = {
                concurrency: navigator.hardwareConcurrency,
                memory: navigator.deviceMemory,
                maxTouchPoints: navigator.maxTouchPoints
            };
        } catch (e) {
            data.hardware = {};
        }
        
        try {
            data.connection = navigator.connection ? {
                type: navigator.connection.type,
                effectiveType: navigator.connection.effectiveType,
                rtt: navigator.connection.rtt,
                downlink: navigator.connection.downlink
            } : {};
        } catch (e) {
            data.connection = {};
        }
        
        try {
            data.browserInfo = {
                userAgent: navigator.userAgent,
                platform: navigator.platform,
                vendor: navigator.vendor,
                product: navigator.product
            };
        } catch (e) {
            data.browserInfo = {};
        }
        
        return data;
    }

    async generateCanvasHash() {
        const canvas = document.createElement('canvas');
        canvas.width = 280;
        canvas.height = 80;
        const ctx = canvas.getContext('2d');
        if (!ctx) return '';

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

        const dataURL = canvas.toDataURL();
        return this.hashString(dataURL);
    }

    async generateWebGLHash() {
        const canvas = document.createElement('canvas');
        const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
        if (!gl) return '';

        const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
        if (!debugInfo) return '';

        const vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
        const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
        const combined = `${vendor}~${renderer}`;
        return this.hashString(combined);
    }

    getWebGLRenderer() {
        const canvas = document.createElement('canvas');
        const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
        if (!gl) return '';

        const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
        if (!debugInfo) return '';

        return gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) || '';
    }

    getWebGLVendor() {
        const canvas = document.createElement('canvas');
        const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
        if (!gl) return '';

        const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
        if (!debugInfo) return '';

        return gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL) || '';
    }

    async generateAudioHash() {
        try {
            const AudioContext = window.OfflineAudioContext || window.webkitOfflineAudioContext;
            if (!AudioContext) return '';

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
            osc.connect(compressor);
            compressor.connect(ctx.destination);
            osc.start(0);

            const buffer = await ctx.startRendering();
            const channelData = buffer.getChannelData(0);
            let hash = 0;
            for (let i = 0; i < 1000; i++) {
                hash = ((hash << 5) - hash) + channelData[i];
                hash = hash & hash;
            }
            return hash.toString(16);
        } catch (e) {
            return '';
        }
    }

    async detectFontsList() {
        const baseFonts = ['monospace', 'sans-serif', 'serif'];
        const testFonts = [
            'Arial', 'Helvetica', 'Times New Roman', 'Courier New',
            'Verdana', 'Georgia', 'Palatino', 'Garamond',
            'Impact', 'Comic Sans MS', 'Trebuchet MS', 'Lucida Console',
            'Tahoma', 'Segoe UI', 'Roboto', 'Open Sans'
        ];
        const detected = [];

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

            for (const font of testFonts) {
                for (const base of baseFonts) {
                    el.style.fontFamily = `"${font}", ${base}`;
                    if (el.offsetWidth !== baseWidths[base]) {
                        detected.push(font);
                        break;
                    }
                }
            }

            document.body.removeChild(el);
        } catch (e) {}

        return detected;
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

    async runAll() {
        const chainResult = await this.runChain();
        const fingerprint = this.generateFingerprint();
        const enhancedData = await this.collectEnhancedEnvironmentData();
        return Object.assign(chainResult, { fingerprint, enhancedData });
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
            components.push('prod:' + (navigator.product || ''));
        } catch (e) {}
        
        try {
            components.push('vendor:' + (navigator.vendor || ''));
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
        
        try {
            components.push('touch:' + (navigator.maxTouchPoints || 0));
        } catch (e) {}
        
        try {
            components.push('pixel:' + (window.devicePixelRatio || ''));
        } catch (e) {}
        
        return components.join('|');
    }

    toJSON() {
        return {
            risk_score: this.riskScore,
            chain_count: this.detectionChain.length,
            results: this.results
        };
    }
}

EnhancedEnvironmentDetector.prototype.detectAdvancedAutomation = async function() {
    let score = 0;
    const detections = [];
    
    try {
        const automationIndicators = [
            { name: '__webdriver', weight: 45 },
            { name: '__selenium', weight: 45 },
            { name: '__driver', weight: 40 },
            { name: '__fxdriver', weight: 45 },
            { name: '__webdriver_func', weight: 50 },
            { name: '__selenium_func', weight: 50 },
            { name: '__driver_func', weight: 50 },
            { name: '__webdriver_unwrapped', weight: 50 },
            { name: '__fxdriver_unwrapped', weight: 50 }
        ];
        
        for (const indicator of automationIndicators) {
            if (window[indicator.name] !== undefined) {
                score += indicator.weight;
                detections.push(indicator.name + '_detected');
            }
        }
    } catch (e) {}
    
    try {
        if (window.callSelenium !== undefined) {
            score += 40;
            detections.push('callSelenium_detected');
        }
    } catch (e) {}
    
    try {
        const keys = Object.keys(window);
        const automationKeys = keys.filter(k => 
            /^(?:__)?(?:selenium|webdriver|driver)/i.test(k)
        );
        
        if (automationKeys.length > 0) {
            score += Math.min(automationKeys.length * 15, 60);
            detections.push('automation_keys_' + automationKeys.length);
        }
    } catch (e) {}
    
    try {
        if (document.documentElement.getAttribute('webdriver') !== null) {
            score += 50;
            detections.push('webdriver_attribute');
        }
    } catch (e) {}
    
    try {
        const ua = navigator.userAgent || '';
        const automationPatterns = [
            { pattern: /selenium|webdriver/gi, weight: 55 },
            { pattern: /automation|bot|crawler/gi, weight: 60 },
            { pattern: /headless.*chrome/gi, weight: 50 }
        ];
        
        for (const ap of automationPatterns) {
            if (ap.pattern.test(ua)) {
                score += ap.weight;
                detections.push('ua_automation_pattern');
            }
        }
    } catch (e) {}
    
    try {
        if (navigator.product === 'Gecko' && !/Firefox/i.test(ua)) {
            score += 45;
            detections.push('gecko_mismatch');
        }
    } catch (e) {}
    
    return { detected: score > 40, score: Math.min(score, 100), detections };
};

EnhancedEnvironmentDetector.prototype.detectStealthEvasion = async function() {
    let score = 0;
    const detections = [];
    
    try {
        const stealthIndicators = [
            { name: 'navigator.webdriver override', check: () => Object.getOwnPropertyDescriptor(navigator, 'webdriver')?.get !== undefined, weight: 35 },
            { name: 'chrome.runtime override', check: () => window.chrome?.runtime !== undefined, weight: 30 },
            { name: 'permissions override', check: () => navigator.permissions?.query !== undefined, weight: 25 },
            { name: 'plugin override', check: () => Object.getOwnPropertyDescriptor(HTMLCanvasElement.prototype, 'toDataURL')?.get !== undefined, weight: 40 },
            { name: 'webgl override', check: () => HTMLCanvasElement.prototype.getContext !== undefined, weight: 35 }
        ];
        
        for (const indicator of stealthIndicators) {
            if (indicator.check()) {
                score += indicator.weight;
                detections.push(indicator.name);
            }
        }
    } catch (e) {}
    
    try {
        const originalCanvas = HTMLCanvasElement.prototype.toDataURL;
        const originalGetContext = HTMLCanvasElement.prototype.getContext;
        
        let canvasIntercepted = false;
        let contextIntercepted = false;
        
        HTMLCanvasElement.prototype.toDataURL = function(...args) {
            canvasIntercepted = true;
            return originalCanvas.apply(this, args);
        };
        
        HTMLCanvasElement.prototype.getContext = function(...args) {
            contextIntercepted = true;
            return originalGetContext.apply(this, args);
        };
        
        if (canvasIntercepted || contextIntercepted) {
            score += 50;
            detections.push('canvas_intercepted');
        }
        
        HTMLCanvasElement.prototype.toDataURL = originalCanvas;
        HTMLCanvasElement.prototype.getContext = originalGetContext;
    } catch (e) {}
    
    try {
        const testCanvas = document.createElement('canvas');
        testCanvas.width = 10;
        testCanvas.height = 10;
        const ctx = testCanvas.getContext('2d');
        
        if (ctx) {
            ctx.fillText('test', 0, 0);
            const data1 = testCanvas.toDataURL();
            const data2 = testCanvas.toDataURL();
            
            if (data1 !== data2) {
                score += 45;
                detections.push('dynamic_canvas_fingerprinting');
            }
        }
    } catch (e) {}
    
    try {
        const originalRTCPeerConnection = RTCPeerConnection;
        let rtcIntercepted = false;
        
        window.RTCPeerConnection = function(...args) {
            rtcIntercepted = true;
            return new originalRTCPeerConnection(...args);
        };
        
        if (rtcIntercepted) {
            score += 40;
            detections.push('webrtc_intercepted');
        }
        
        window.RTCPeerConnection = originalRTCPeerConnection;
    } catch (e) {}
    
    return { detected: score > 35, score: Math.min(score, 100), detections };
};

EnhancedEnvironmentDetector.prototype.detectAdvancedEmulator = async function() {
    let score = 0;
    const detections = [];
    
    try {
        const emulatorPatterns = [
            { pattern: /android.*emulator/i, name: 'android_emulator_ua', weight: 55 },
            { pattern: /genymotion/i, name: 'genymotion', weight: 60 },
            { pattern: /bluestacks/i, name: 'bluestacks', weight: 55 },
            { pattern: /nox.*player/i, name: 'nox_player', weight: 60 },
            { pattern: /meMu/i, name: 'memu', weight: 60 },
            { pattern: /LDPlayer/i, name: 'ldplayer', weight: 55 },
            { pattern: /droid4x/i, name: 'droid4x', weight: 55 },
            { pattern: /koplayer/i, name: 'koplayer', weight: 55 },
            { pattern: /iPhone.*Simulator/i, name: 'ios_simulator', weight: 50 },
            { pattern: /iPad.*Simulator/i, name: 'ios_simulator', weight: 50 },
            { pattern: /x86_64.*android/i, name: 'android_x86_emulator', weight: 50 }
        ];
        
        const ua = navigator.userAgent || '';
        
        for (const ep of emulatorPatterns) {
            if (ep.pattern.test(ua)) {
                score += ep.weight;
                detections.push(ep.name);
            }
        }
    } catch (e) {}
    
    try {
        if (navigator.platform) {
            const platform = navigator.platform.toLowerCase();
            
            if (/linux/i.test(platform) && /android/i.test(ua)) {
                score += 50;
                detections.push('linux_android_mismatch');
            }
            
            if (/win/i.test(platform) && /android/i.test(ua)) {
                score += 45;
                detections.push('windows_android_mismatch');
            }
        }
    } catch (e) {}
    
    try {
        const screenWidth = screen.width;
        const screenHeight = screen.height;
        
        const emulatorResolutions = [
            { w: 320, h: 480, name: 'iphone_3gs', weight: 40 },
            { w: 375, h: 667, name: 'iphone_6', weight: 35 },
            { w: 414, h: 896, name: 'iphone_xr', weight: 35 },
            { w: 768, h: 1024, name: 'ipad', weight: 40 },
            { w: 600, h: 1024, name: 'android_tablet', weight: 40 }
        ];
        
        for (const res of emulatorResolutions) {
            if (screenWidth === res.w && screenHeight === res.h) {
                if (!/mobile|android|iphone|ipad/i.test(ua)) {
                    score += res.weight;
                    detections.push(res.name + '_resolution_no_mobile_ua');
                }
            }
        }
    } catch (e) {}
    
    try {
        if (screen.width === screen.availWidth && screen.height === screen.availHeight) {
            if (/mobile|android/i.test(ua)) {
                score += 40;
                detections.push('fullscreen_mobile_emulator');
            }
        }
    } catch (e) {}
    
    try {
        if (navigator.maxTouchPoints === 0) {
            const ua = navigator.userAgent || '';
            if (/mobile|android|iphone|ipad/i.test(ua)) {
                score += 50;
                detections.push('mobile_no_touch_support');
            }
        }
    } catch (e) {}
    
    try {
        if (navigator.hardwareConcurrency) {
            if (navigator.hardwareConcurrency > 16) {
                if (/android|iphone|ipad/i.test(ua)) {
                    score += 45;
                    detections.push('mobile_high_cpu_cores');
                }
            }
        }
    } catch (e) {}
    
    try {
        const canvas = document.createElement('canvas');
        const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
        
        if (gl) {
            const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
            if (debugInfo) {
                const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) || '';
                const rendererLower = renderer.toLowerCase();
                
                const emulatorRenderers = [
                    { pattern: /android|adr|goldfish/i, name: 'android_emulator_gpu', weight: 50 },
                    { pattern: /llvmpipe|swiftshader|software/i, name: 'software_rendering', weight: 45 },
                    { pattern: /virtual/i, name: 'virtual_gpu', weight: 40 }
                ];
                
                for (const er of emulatorRenderers) {
                    if (er.pattern.test(rendererLower)) {
                        score += er.weight;
                        detections.push(er.name);
                    }
                }
            }
        }
    } catch (e) {}
    
    return { detected: score > 45, score: Math.min(score, 100), detections };
};

EnhancedEnvironmentDetector.prototype.detectCanvasAdvanced = async function() {
    let score = 0;
    const detections = [];
    
    try {
        const canvas = document.createElement('canvas');
        canvas.width = 300;
        canvas.height = 100;
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
        
        const dataURL = canvas.toDataURL();
        const dataURL2 = canvas.toDataURL();
        
        if (dataURL !== dataURL2) {
            score += 35;
            detections.push('canvas_unstable');
        }
        
        const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
        const pixelValues = imageData.data;
        
        let nonZeroPixels = 0;
        for (let i = 0; i < pixelValues.length; i += 4) {
            if (pixelValues[i] !== 0 || pixelValues[i+1] !== 0 || pixelValues[i+2] !== 0) {
                nonZeroPixels++;
            }
        }
        
        const totalPixels = pixelValues.length / 4;
        const nonZeroRatio = nonZeroPixels / totalPixels;
        
        if (nonZeroRatio < 0.1) {
            score += 40;
            detections.push('canvas_mostly_black');
        }
        
        const entropy = this.calculateCanvasEntropy(pixelValues);
        if (entropy < 2.0) {
            score += 30;
            detections.push('low_canvas_entropy');
        }
        
        const hash = await this.hashCanvasDataURL(dataURL);
        if (this.isCommonCanvasHash(hash)) {
            score += 25;
            detections.push('common_canvas_hash');
        }
    } catch (e) {
        score += 45;
        detections.push('canvas_error');
    }
    
    try {
        const testCanvas = document.createElement('canvas');
        testCanvas.width = 100;
        testCanvas.height = 100;
        const testCtx = testCanvas.getContext('2d');
        
        testCtx.fillStyle = '#ff0000';
        testCtx.fillRect(0, 0, 50, 50);
        testCtx.fillStyle = '#0000ff';
        testCtx.fillRect(50, 0, 50, 50);
        
        const imageData = testCtx.getImageData(0, 0, 100, 100).data;
        
        let anomalyDetected = 0;
        for (let i = 0; i < imageData.length; i += 4) {
            const r = imageData[i];
            const g = imageData[i+1];
            const b = imageData[i+2];
            
            if (r === 0 && g === 0 && b === 0) {
                anomalyDetected++;
            }
        }
        
        if (anomalyDetected > imageData.length / 8) {
            score += 45;
            detections.push('canvas_color_anomaly');
        }
    } catch (e) {}
    
    return { detected: score > 40, score: Math.min(score, 100), detections };
};

EnhancedEnvironmentDetector.prototype.calculateCanvasEntropy = function(pixelData) {
    const frequency = {};
    for (let i = 0; i < pixelData.length; i++) {
        const value = pixelData[i];
        frequency[value] = (frequency[value] || 0) + 1;
    }
    
    let entropy = 0;
    const total = pixelData.length;
    
    for (const value in frequency) {
        const p = frequency[value] / total;
        if (p > 0) {
            entropy -= p * Math.log2(p);
        }
    }
    
    return entropy;
};

EnhancedEnvironmentDetector.prototype.hashCanvasDataURL = async function(dataURL) {
    const encoder = new TextEncoder();
    const data = encoder.encode(dataURL);
    const hashBuffer = await crypto.subtle.digest('SHA-256', data);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
};

EnhancedEnvironmentDetector.prototype.isCommonCanvasHash = function(hash) {
    const commonHashes = [
        'a1b2c3d4e5f6',
        '1234567890ab',
        'ffffffffffff',
        '000000000000',
        'deadbeef1234'
    ];
    
    const prefix = hash.substring(0, 12);
    return commonHashes.some(common => prefix === common);
};

EnhancedEnvironmentDetector.prototype.detectWebGLAdvanced = async function() {
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
            
            const rendererLower = renderer.toLowerCase();
            
            const softwareIndicators = [
                'swiftshader', 'llvmpipe', 'mesa', 'software', 'emulated'
            ];
            
            for (const indicator of softwareIndicators) {
                if (rendererLower.includes(indicator)) {
                    score += 45;
                    detections.push('software_renderer_' + indicator);
                }
            }
            
            const vmIndicators = [
                'vmware', 'virtualbox', 'parallels', 'qemu', 'kvm'
            ];
            
            for (const indicator of vmIndicators) {
                if (rendererLower.includes(indicator)) {
                    score += 50;
                    detections.push('vm_renderer_' + indicator);
                }
            }
            
            const anonymizedPatterns = ['generic', 'unknown', 'default'];
            let anonymizedCount = 0;
            for (const pattern of anonymizedPatterns) {
                if (rendererLower.includes(pattern)) {
                    anonymizedCount++;
                }
            }
            
            if (anonymizedCount >= 2) {
                score += 35;
                detections.push('anonymized_webgl');
            }
            
            if (!vendor || !renderer) {
                score += 30;
                detections.push('missing_webgl_info');
            }
        } else {
            score += 30;
            detections.push('webgl_debug_blocked');
        }
        
        const maxTexSize = gl.getParameter(gl.MAX_TEXTURE_SIZE);
        if (maxTexSize < 2048) {
            score += 25;
            detections.push('low_max_texture_size');
        }
        
        const extensions = gl.getSupportedExtensions();
        if (!extensions || extensions.length < 10) {
            score += 20;
            detections.push('few_webgl_extensions');
        }
        
        const maxVertAttribs = gl.getParameter(gl.MAX_VERTEX_ATTRIBS);
        if (maxVertAttribs < 8) {
            score += 20;
            detections.push('low_vertex_attribs');
        }
        
    } catch (e) {
        score += 40;
        detections.push('webgl_error');
    }
    
    return { detected: score > 40, score: Math.min(score, 100), detections };
};

EnhancedEnvironmentDetector.prototype.collectEnhancedFingerprints = async function() {
    const fingerprints = {};
    
    try {
        fingerprints.canvasHash = await this.generateCanvasFingerprint();
    } catch (e) {
        fingerprints.canvasHash = '';
    }
    
    try {
        fingerprints.webglHash = await this.generateWebGLFingerprint();
    } catch (e) {
        fingerprints.webglHash = '';
    }
    
    try {
        fingerprints.audioHash = await this.generateAudioFingerprint();
    } catch (e) {
        fingerprints.audioHash = '';
    }
    
    try {
        fingerprints.fontHash = await this.generateFontFingerprint();
    } catch (e) {
        fingerprints.fontHash = '';
    }
    
    try {
        fingerprints.behavioralHash = this.generateBehavioralFingerprint();
    } catch (e) {
        fingerprints.behavioralHash = '';
    }
    
    try {
        fingerprints.timingHash = await this.generateTimingFingerprint();
    } catch (e) {
        fingerprints.timingHash = '';
    }
    
    return fingerprints;
};

EnhancedEnvironmentDetector.prototype.generateCanvasFingerprint = async function() {
    const canvas = document.createElement('canvas');
    canvas.width = 300;
    canvas.height = 100;
    const ctx = canvas.getContext('2d');
    
    if (!ctx) return '';
    
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
    
    const dataURL = canvas.toDataURL();
    return await this.hashCanvasDataURL(dataURL);
};

EnhancedEnvironmentDetector.prototype.generateWebGLFingerprint = async function() {
    const canvas = document.createElement('canvas');
    const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
    
    if (!gl) return '';
    
    const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
    if (!debugInfo) return '';
    
    const vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
    const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
    
    const combined = `${vendor}~${renderer}`;
    return await this.hashCanvasDataURL(combined);
};

EnhancedEnvironmentDetector.prototype.generateAudioFingerprint = async function() {
    try {
        const AudioContext = window.OfflineAudioContext || window.webkitOfflineAudioContext;
        if (!AudioContext) return '';
        
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
        
        osc.connect(compressor);
        compressor.connect(ctx.destination);
        osc.start(0);
        
        const buffer = await ctx.startRendering();
        const channelData = buffer.getChannelData(0);
        
        let hash = 0;
        for (let i = 0; i < 1000; i++) {
            hash = ((hash << 5) - hash) + channelData[i];
            hash = hash & hash;
        }
        
        return Math.abs(hash).toString(16);
    } catch (e) {
        return '';
    }
};

EnhancedEnvironmentDetector.prototype.generateFontFingerprint = async function() {
    const baseFonts = ['monospace', 'sans-serif', 'serif'];
    const testFonts = [
        'Arial', 'Helvetica', 'Times New Roman', 'Courier New',
        'Verdana', 'Georgia', 'Palatino', 'Garamond'
    ];
    
    const el = document.createElement('div');
    el.style.cssText = 'position:absolute;left:-9999px;font-size:72px;visibility:hidden';
    el.textContent = 'mmmmmmmmmmlli';
    document.body.appendChild(el);
    
    const baseWidths = {};
    for (const base of baseFonts) {
        el.style.fontFamily = base;
        baseWidths[base] = el.offsetWidth;
    }
    
    const detected = [];
    for (const font of testFonts) {
        for (const base of baseFonts) {
            el.style.fontFamily = `"${font}", ${base}`;
            if (el.offsetWidth !== baseWidths[base]) {
                detected.push(font);
                break;
            }
        }
    }
    
    document.body.removeChild(el);
    
    return detected.join(',');
};

EnhancedEnvironmentDetector.prototype.generateBehavioralFingerprint = function() {
    const components = [];
    
    try {
        components.push('touch:' + (navigator.maxTouchPoints || 0));
    } catch (e) {}
    
    try {
        components.push('pixel:' + (window.devicePixelRatio || ''));
    } catch (e) {}
    
    try {
        components.push('cpu:' + (navigator.hardwareConcurrency || ''));
    } catch (e) {}
    
    try {
        components.push('mem:' + (navigator.deviceMemory || ''));
    } catch (e) {}
    
    return components.join('|');
};

EnhancedEnvironmentDetector.prototype.generateTimingFingerprint = async function() {
    const samples = [];
    
    for (let i = 0; i < 5; i++) {
        const start = performance.now();
        await new Promise(resolve => setTimeout(resolve, 1));
        const end = performance.now();
        samples.push(end - start);
    }
    
    const avg = samples.reduce((a, b) => a + b, 0) / samples.length;
    const variance = samples.reduce((sum, val) => sum + Math.pow(val - avg, 2), 0) / samples.length;
    
    return `${avg.toFixed(2)}~${variance.toFixed(2)}`;
};

if (typeof window !== 'undefined') {
    window.EnhancedEnvironmentDetector = EnhancedEnvironmentDetector;
}
