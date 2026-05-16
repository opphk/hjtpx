class EnvironmentDetector {
    constructor(options = {}) {
        this.options = Object.assign({
            apiBase: '/api/v1',
            sampleRate: 0.3,
            chainCount: 12,
            enableAll: true,
            sessionId: null
        }, options);
        this.results = {};
        this.riskScore = 0;
        this.detectionChain = [];
        this.detectionId = 'det_' + Date.now() + '_' + Math.random().toString(36).substr(2, 6);
        this.weights = {
            canvas: 8,
            webgl: 10,
            webgl2: 8,
            audio: 9,
            fonts: 7,
            webrtc_ip: 10,
            webdriver: 15,
            selenium: 18,
            puppeteer: 18,
            playwright: 18,
            chrome_runtime: 10,
            headless: 12,
            permissions: 6,
            plugins: 5,
            languages: 4,
            timezone: 5,
            screen: 3,
            hardware: 4,
            memory: 3,
            storage: 5,
            navigator: 4,
            window_props: 4,
            iframe: 6,
            notification: 3,
            battery: 3,
            media_devices: 4,
            connection: 5,
            adblock: 4,
            math: 3,
            gpu: 6,
            speech: 3
        };
    }

    getDetectionMethods() {
        return [
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
            'detectSpeech'
        ];
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

        const autoTools = ['detectWebDriver', 'detectPuppeteer', 'detectPlaywright', 'detectSelenium'];
        const autoDetected = autoTools.filter(m => {
            const r = this.results[m];
            return r && r.detected === true;
        }).length;

        if (autoDetected >= 2) {
            baseScore = Math.min(baseScore * 1.5 + 20, 100);
        } else if (autoDetected >= 1) {
            baseScore = Math.min(baseScore * 1.3 + 10, 100);
        }

        const proxyIndicators = ['detectWebRTCIP', 'detectConnection'];
        const proxyAnomalies = proxyIndicators.filter(m => {
            const r = this.results[m];
            return r && r.score > 30;
        }).length;

        if (proxyAnomalies >= 2) {
            baseScore = Math.min(baseScore * 1.3 + 15, 100);
        }

        return Math.round(Math.min(Math.max(baseScore, 0), 100));
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
            score += 15;
            detections.push('no_plugins');
        }
        if (navigator.languages && navigator.languages.length === 0) {
            score += 15;
            detections.push('no_languages');
        }
        if (window.chrome && window.chrome.runtime === undefined) {
            score += 20;
            detections.push('chrome_no_runtime');
        }
        const mimeTypes = navigator.mimeTypes;
        if (mimeTypes && mimeTypes.length === 0) {
            score += 20;
            detections.push('no_mimetypes');
        }
        try {
            const ua = navigator.userAgent || '';
            if (/headless|phantom/i.test(ua)) {
                score += 35;
                detections.push('headless_ua');
            }
        } catch (e) {}
        try {
            if (window.outerHeight === 0 && window.outerWidth === 0) {
                score += 25;
                detections.push('zero_window_size');
            }
        } catch (e) {}
        return { detected: score > 30, score: Math.min(score, 100), detections };
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
                score += 15;
                detections.push(prop);
            }
        }
        try {
            if (navigator.webdriver === true) {
                score += 30;
                detections.push('navigator.webdriver');
            }
        } catch (e) {}
        try {
            const el = document.createElement('div');
            el.setAttribute('onclick', 'return __webdriver_script_fn()');
            if (el.onclick !== null) {
                score += 10;
                detections.push('webdriver_script_fn');
            }
        } catch (e) {}
        try {
            const el = document.createElement('div');
            el.setAttribute('onmousemove', 'return __driver_evaluate()');
            if (el.onmousemove !== null) {
                score += 10;
                detections.push('driver_evaluate');
            }
        } catch (e) {}
        return { detected: score > 20, score: Math.min(score, 100), detections };
    }

    async detectPuppeteer() {
        let score = 0;
        const detections = [];
        try {
            if (navigator.webdriver === true) {
                score += 25;
                detections.push('webdriver_true');
            }
        } catch (e) {}
        try {
            if (document.$cdc_asdjflasutopfhvcZLmcfl_) {
                score += 35;
                detections.push('cdc_marker');
            }
        } catch (e) {}
        try {
            if (document.$chrome_asyncScriptInfo) {
                score += 25;
                detections.push('chrome_async_script');
            }
        } catch (e) {}
        try {
            const el = document.createElement('div');
            el.setAttribute('onpaste', 'return function(){throw new Error("puppeteer")}');
            if (el.onpaste !== null) {
                score += 20;
                detections.push('puppeteer_onpaste');
            }
        } catch (e) {}
        try {
            const userAgent = navigator.userAgent || '';
            if (/headless/i.test(userAgent)) {
                score += 30;
                detections.push('headless_ua');
            }
            if (/puppet/i.test(userAgent)) {
                score += 40;
                detections.push('puppeteer_ua');
            }
        } catch (e) {}
        try {
            if (window._puppeteer_globals !== undefined) {
                score += 30;
                detections.push('puppeteer_globals');
            }
        } catch (e) {}
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectPlaywright() {
        let score = 0;
        const detections = [];
        try {
            if (window.__playwright__ !== undefined ||
                window.__pw_tags !== undefined ||
                window.__pw_resume__ !== undefined) {
                score += 45;
                detections.push('playwright_global');
            }
        } catch (e) {}
        try {
            const el = document.createElement('div');
            el.setAttribute('onfocus', 'return __pw_resume__()');
            if (el.onfocus !== null) {
                score += 35;
                detections.push('playwright_onfocus');
            }
        } catch (e) {}
        try {
            const el = document.createElement('div');
            el.setAttribute('onmouseenter', 'return __pw_resume__()');
            if (el.onmouseenter !== null) {
                score += 25;
                detections.push('playwright_mouseenter');
            }
        } catch (e) {}
        try {
            const ua = navigator.userAgent || '';
            if (/playwright/i.test(ua)) {
                score += 50;
                detections.push('playwright_ua');
            }
        } catch (e) {}
        return { detected: score > 30, score: Math.min(score, 100), detections };
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
                score += 20;
                detections.push(prop);
            }
        }
        try {
            if (document.documentElement.getAttribute('webdriver') !== null) {
                score += 25;
                detections.push('webdriver_attr');
            }
        } catch (e) {}
        try {
            const el = document.createElement('div');
            el.setAttribute('onmouseover', 'return Selenium.prototype.whatever');
            if (el.onmouseover !== null) {
                score += 15;
                detections.push('selenium_prototype');
            }
        } catch (e) {}
        try {
            const el = document.createElement('div');
            el.setAttribute('onkeydown', 'return selenium_executor.onkeydown');
            if (el.onkeydown !== null) {
                score += 15;
                detections.push('selenium_executor');
            }
        } catch (e) {}
        try {
            const ua = navigator.userAgent || '';
            if (/selenium/i.test(ua)) {
                score += 40;
                detections.push('selenium_ua');
            }
        } catch (e) {}
        try {
            if (window.__$webdriverAsyncExecutor !== undefined) {
                score += 20;
                detections.push('webdriver_async_executor');
            }
        } catch (e) {}
        return { detected: score > 20, score: Math.min(score, 100), detections };
    }

    async detectChromeRuntime() {
        let score = 0;
        const detections = [];
        try {
            if (window.chrome) {
                if (window.chrome.runtime === undefined) {
                    score += 20;
                    detections.push('chrome_runtime_missing');
                } else if (window.chrome.runtime && window.chrome.runtime.id === undefined) {
                    score += 10;
                    detections.push('chrome_runtime_no_id');
                }
                if (window.chrome.loadTimes === undefined) {
                    score += 10;
                    detections.push('chrome_loadtimes_missing');
                }
                if (window.chrome.csi === undefined) {
                    score += 10;
                    detections.push('chrome_csi_missing');
                }
                if (window.chrome.app === undefined) {
                    score += 10;
                    detections.push('chrome_app_missing');
                }
            } else {
                if (!/Edge|Edg|Firefox|Safari/i.test(navigator.userAgent || '')) {
                    score += 30;
                    detections.push('no_chrome_no_alt');
                }
            }
        } catch (e) {
            score += 25;
            detections.push('chrome_check_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
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
                    score += 20;
                    detections.push('all_permissions_denied');
                }
                const deniedCount = permChecks.filter(p => p.state === 'denied').length;
                if (deniedCount >= 4) {
                    score += 10;
                    detections.push('most_permissions_denied');
                }
            } else {
                score += 20;
                detections.push('permissions_api_missing');
            }
        } catch (e) {
            score += 25;
            detections.push('permissions_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectPlugins() {
        let score = 0;
        const detections = [];
        try {
            const plugins = navigator.plugins;
            if (!plugins || plugins.length === 0) {
                score += 25;
                detections.push('no_plugins');
            } else {
                const commonPlugins = ['PDF Viewer', 'Chrome PDF Viewer', 'Chromium PDF Viewer',
                    'Microsoft Edge PDF Viewer', 'WebKit built-in PDF'];
                const hasPDF = Array.from(plugins).some(p =>
                    commonPlugins.some(cp => p.name.includes(cp))
                );
                if (!hasPDF) {
                    score += 10;
                    detections.push('no_pdf_plugin');
                }
                if (plugins.length < 3) {
                    score += 10;
                    detections.push('too_few_plugins');
                }
                if (plugins.length > 10) {
                    score += 5;
                    detections.push('too_many_plugins');
                }
            }
        } catch (e) {
            score += 30;
            detections.push('plugins_access_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectLanguages() {
        let score = 0;
        const detections = [];
        try {
            const langs = navigator.languages;
            if (!langs || langs.length === 0) {
                score += 25;
                detections.push('no_languages');
            }
            const lang = navigator.language;
            if (!lang) {
                score += 20;
                detections.push('no_language');
            }
            if (langs && langs.length > 0 && lang) {
                if (langs[0] !== lang) {
                    score += 15;
                    detections.push('languages_mismatch');
                }
            }
        } catch (e) {
            score += 30;
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
                score += 30;
                detections.push('no_timezone');
            }
            const offset = new Date().getTimezoneOffset();
            if (offset === 0 && !tz) {
                score += 20;
                detections.push('utc_offset_no_tz');
            }
            const year = new Date().getFullYear();
            if (year < 2000 || year > 2100) {
                score += 25;
                detections.push('unrealistic_date');
            }
            try {
                const matchOffset = /GMT([+-]\d{2}):?(\d{2})/.exec(new Date().toString());
                if (matchOffset) {
                    const strOffset = parseInt(matchOffset[1]) * 60 + parseInt(matchOffset[2]) * (matchOffset[1] > 0 ? 1 : -1);
                    if (Math.abs(strOffset + offset) > 30) {
                        score += 20;
                        detections.push('timezone_offset_mismatch');
                    }
                }
            } catch (e) {}
        } catch (e) {
            score += 35;
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
                score += 30;
                detections.push('no_screen_size');
            }
            if (colorDepth === 0 || pixelDepth === 0) {
                score += 25;
                detections.push('zero_depth');
            }
            if (width <= 800 || height <= 600) {
                score += 10;
                detections.push('small_screen');
            }
            if ('isExtended' in screen && screen.isExtended === undefined) {
                score += 10;
                detections.push('screen_extended_missing');
            }
            if (availWidth === 0 || availHeight === 0) {
                score += 15;
                detections.push('zero_avail_size');
            }
        } catch (e) {
            score += 30;
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
                score += 30;
                detections.push('no_concurrency');
            } else if (c <= 1) {
                score += 25;
                detections.push('single_core');
            } else if (c > 64) {
                score += 20;
                detections.push('unrealistic_cores');
            }
        } catch (e) {
            score += 30;
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
                score += 20;
                detections.push('no_device_memory');
            } else if (mem <= 0.25) {
                score += 25;
                detections.push('low_memory');
            } else if (mem > 64) {
                score += 15;
                detections.push('unrealistic_memory');
            }
        } catch (e) {
            score += 20;
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
            score += 20;
            detections.push('localStorage_denied');
        }
        try {
            sessionStorage.setItem('_md_test', '1');
            sessionStorage.removeItem('_md_test');
        } catch (e) {
            score += 20;
            detections.push('sessionStorage_denied');
        }
        try {
            if (navigator.storage && navigator.storage.estimate) {
                const est = await navigator.storage.estimate();
                if (est.quota === 0) {
                    score += 15;
                    detections.push('zero_storage_quota');
                }
            } else {
                score += 10;
                detections.push('storage_api_missing');
            }
        } catch (e) {
            score += 15;
            detections.push('storage_estimate_error');
        }
        try {
            if (navigator.cookieEnabled === false) {
                score += 15;
                detections.push('cookies_disabled');
            }
        } catch (e) {}
        return { detected: score > 25, score: Math.min(score, 100), detections };
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
                score += 40;
                detections.push('no_canvas_context');
                return { detected: true, score: Math.min(score, 100), detections };
            }

            ctx.textBaseline = 'alphabetic';
            ctx.fillStyle = '#f60';
            ctx.fillRect(125, 1, 62, 20);
            ctx.fillStyle = '#069';
            ctx.font = '11pt Arial';
            ctx.fillText('Cwm fjordbank glyphs vext quiz, \ud83d\ude03', 2, 15);
            ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
            ctx.font = '18pt Arial';
            ctx.fillText('Cwm fjordbank glyphs vext quiz, \ud83d\ude03', 4, 45);

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
                score += 25;
                detections.push('canvas_unstable');
            }

            const imageData = ctx.getImageData(0, 0, 10, 10);
            const pixelSum = Array.from(imageData.data.slice(0, 40)).reduce((a, b) => a + b, 0);
            if (pixelSum === 0) {
                score += 20;
                detections.push('canvas_empty_readback');
            }
        } catch (e) {
            score += 35;
            detections.push('canvas_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectWebGL() {
        let score = 0;
        const detections = [];
        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
            if (!gl) {
                score += 40;
                detections.push('no_webgl');
                return { detected: true, score: Math.min(score, 100), detections };
            }
            const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
            if (debugInfo) {
                const vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
                const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                if (!vendor || !renderer) {
                    score += 15;
                    detections.push('webgl_no_vendor');
                }
                if (/swiftshader|llvmpipe|mesa|virtual|google\s*inc/i.test(renderer || '')) {
                    score += 30;
                    detections.push('software_renderer');
                }
            } else {
                score += 20;
                detections.push('no_webgl_debug');
            }
            const maxTexSize = gl.getParameter(gl.MAX_TEXTURE_SIZE);
            if (maxTexSize <= 1024) {
                score += 15;
                detections.push('small_tex_size');
            }
            const maxVertAttribs = gl.getParameter(gl.MAX_VERTEX_ATTRIBS);
            if (maxVertAttribs <= 8) {
                score += 10;
                detections.push('few_vertex_attribs');
            }
            const aliasedRange = gl.getParameter(gl.ALIASED_LINE_WIDTH_RANGE);
            if (aliasedRange && aliasedRange[1] <= 1) {
                score += 10;
                detections.push('aliased_line_only');
            }
            const shaderPrecision = gl.getShaderPrecisionFormat(gl.FRAGMENT_SHADER, gl.HIGH_FLOAT);
            if (shaderPrecision && shaderPrecision.precision < 16) {
                score += 15;
                detections.push('low_shader_precision');
            }
            const ext = gl.getExtension('EXT_texture_filter_anisotropic');
            if (!ext) {
                score += 5;
                detections.push('no_anisotropic');
            }
            const supportedExts = gl.getSupportedExtensions();
            if (supportedExts && supportedExts.length < 10) {
                score += 10;
                detections.push('few_webgl_extensions');
            }
        } catch (e) {
            score += 35;
            detections.push('webgl_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
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
                if (/swiftshader|llvmpipe|mesa|virtual/i.test(renderer || '')) {
                    score += 25;
                    detections.push('webgl2_software_renderer');
                }
            }
            const maxTexSize = gl2.getParameter(gl2.MAX_TEXTURE_SIZE);
            if (maxTexSize <= 1024) {
                score += 10;
                detections.push('webgl2_small_tex');
            }
            const supportedExts = gl2.getSupportedExtensions();
            if (supportedExts && supportedExts.length < 5) {
                score += 10;
                detections.push('few_webgl2_extensions');
            }
        } catch (e) {
            score += 20;
            detections.push('webgl2_error');
        }
        return { detected: score > 20, score: Math.min(score, 100), detections };
    }

    async detectAudio() {
        let score = 0;
        const detections = [];
        try {
            const AudioContext = window.OfflineAudioContext || window.webkitOfflineAudioContext;
            if (!AudioContext) {
                score += 30;
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
            osc.connect(compressor);
            compressor.connect(ctx.destination);
            osc.start(0);

            const startTime = performance.now();
            const buffer = await ctx.startRendering();
            const renderTime = performance.now() - startTime;

            if (renderTime < 5) {
                score += 20;
                detections.push('audio_render_too_fast');
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
                score += 25;
                detections.push('audio_silent');
            }
        } catch (e) {
            score += 30;
            detections.push('audio_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
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
            let fontCount = 0;
            for (const font of testFonts) {
                for (const base of baseFonts) {
                    el.style.fontFamily = `"${font}", ${base}`;
                    if (el.offsetWidth !== baseWidths[base]) {
                        fontCount++;
                        break;
                    }
                }
            }
            document.body.removeChild(el);
            if (fontCount < 3) {
                score += 25;
                detections.push('too_few_fonts');
            }
            if (fontCount < 8) {
                score += 10;
                detections.push('limited_fonts');
            }
        } catch (e) {
            score += 25;
            detections.push('font_detection_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectWebRTCIP() {
        let score = 0;
        const detections = [];
        try {
            const RTCPeerConnection = window.RTCPeerConnection ||
                window.webkitRTCPeerConnection ||
                window.mozRTCPeerConnection;
            if (!RTCPeerConnection) {
                score += 15;
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
                    ip.startsWith('172.16.') ||
                    ip.startsWith('192.168.')
                );
                const publicIPs = ipsArr.filter(ip => !privateIPs.includes(ip));
                if (publicIPs.length > 0) {
                    detections.push('public_ip_detected');
                    if (privateIPs.length > 0) {
                        score += 20;
                        detections.push('vpn_possible');
                    }
                }
            }
        } catch (e) {
            score += 15;
            detections.push('webrtc_error');
        }
        return { detected: score > 15, score: Math.min(score, 100), detections };
    }

    async detectNavigatorProps() {
        let score = 0;
        const detections = [];
        try {
            if (navigator.connection) {
                if (navigator.connection.type === 'none' &&
                    navigator.onLine === false) {
                    score += 20;
                    detections.push('offline_with_connection');
                }
            }
        } catch (e) {}
        try {
            if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
                const devices = await navigator.mediaDevices.enumerateDevices().catch(() => []);
                if (devices.length === 0) {
                    score += 15;
                    detections.push('no_media_devices');
                }
            } else {
                score += 10;
                detections.push('no_enumerate_devices');
            }
        } catch (e) {
            score += 15;
            detections.push('media_devices_error');
        }
        try {
            if (!navigator.credentials || !navigator.credentials.preventSilentAccess) {
                score += 5;
                detections.push('no_credentials_api');
            }
        } catch (e) {}
        try {
            if (navigator.serviceWorker === undefined) {
                score += 10;
                detections.push('no_serviceworker');
            }
        } catch (e) {}
        try {
            if (typeof navigator.getBattery === 'function') {
                const battery = await navigator.getBattery().catch(() => null);
                if (battery && battery.charging === undefined) {
                    score += 10;
                    detections.push('battery_no_charging');
                }
            }
        } catch (e) {}
        try {
            if (navigator.product === 'Gecko' && !/Firefox/i.test(navigator.userAgent || '')) {
                score += 20;
                detections.push('gecko_no_firefox');
            }
        } catch (e) {}
        try {
            if (navigator.vendor === '' && navigator.product === 'Gecko') {
            } else if (navigator.vendor === '') {
                score += 10;
                detections.push('empty_vendor');
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
                score += 25;
                detections.push('zero_outer_size');
            }
            if (innerW > outerW || innerH > outerH) {
                score += 15;
                detections.push('inner_larger_than_outer');
            }
        } catch (e) {
            score += 20;
            detections.push('window_size_error');
        }
        try {
            if (window.screenX === undefined || window.screenY === undefined) {
                score += 10;
                detections.push('no_screen_position');
            }
        } catch (e) {}
        try {
            if (window.openDatabase === undefined) {
                score += 5;
                detections.push('no_opendatabase');
            }
        } catch (e) {}
        try {
            if (window.indexedDB === undefined) {
                score += 10;
                detections.push('no_indexeddb');
            }
        } catch (e) {}
        try {
            if (typeof window.postMessage !== 'function') {
                score += 20;
                detections.push('no_postmessage');
            }
        } catch (e) {}
        try {
            if (window.screenTop === undefined || window.screenLeft === undefined) {
                score += 5;
                detections.push('no_screen_edge');
            }
        } catch (e) {}
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectIframe() {
        let score = 0;
        const detections = [];
        try {
            if (window.self !== window.top) {
                score += 15;
                detections.push('in_iframe');
            }
        } catch (e) {
            score += 35;
            detections.push('cross_origin_frame');
        }
        try {
            const frameEl = document.createElement('iframe');
            frameEl.style.display = 'none';
            frameEl.sandbox = 'allow-scripts';
            document.body.appendChild(frameEl);
            const frameWin = frameEl.contentWindow;
            if (frameWin && frameWin.document) {
            }
            document.body.removeChild(frameEl);
        } catch (e) {
            score += 15;
            detections.push('iframe_access_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectNotification() {
        let score = 0;
        const detections = [];
        try {
            if ('Notification' in window) {
                if (Notification.permission === 'denied') {
                    score += 5;
                    detections.push('notification_denied');
                }
            } else {
                score += 15;
                detections.push('no_notification');
            }
        } catch (e) {
            score += 15;
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
                        score += 15;
                        detections.push('battery_props_missing');
                    }
                    if (battery.level === 0 && battery.charging === false) {
                        score += 5;
                        detections.push('battery_dead_not_charging');
                    }
                }
            } else {
                score += 10;
                detections.push('no_battery_api');
            }
        } catch (e) {
            score += 15;
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
                    score += 20;
                    detections.push('no_media_inputs');
                }
                const allHaveLabels = devices.every(d => d.label !== '');
                if (!allHaveLabels) {
                    score += 10;
                    detections.push('media_no_labels');
                }
            } else {
                score += 15;
                detections.push('no_media_api');
            }
        } catch (e) {
            score += 20;
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
                score += 10;
                detections.push('no_connection_api');
            } else {
                if (conn.type === 'vpn') {
                    score += 40;
                    detections.push('vpn_detected');
                }
                if (conn.type === 'proxy') {
                    score += 40;
                    detections.push('proxy_detected');
                }
                if (conn.saveData === true) {
                    score += 10;
                    detections.push('save_data_enabled');
                }
                if (conn.effectiveType === 'slow-2g' || conn.effectiveType === '2g') {
                    score += 10;
                    detections.push('slow_connection');
                }
            }
        } catch (e) {
            score += 10;
            detections.push('connection_error');
        }
        return { detected: score > 20, score: Math.min(score, 100), detections };
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
                score += 15;
                detections.push('adblock_detected');
            }
            document.body.removeChild(el);
        } catch (e) {
            score += 10;
            detections.push('adblock_check_error');
        }
        return { detected: score > 10, score: Math.min(score, 100), detections };
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
                    score += 15;
                    detections.push('math_' + key + '_invalid');
                }
            }
        } catch (e) {
            score += 20;
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
                    score += 15;
                    detections.push('small_renderbuffer');
                }
                if (maxCombinedTexUnits <= 8) {
                    score += 10;
                    detections.push('few_texture_units');
                }
            }
        } catch (e) {
            score += 10;
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
                    score += 10;
                    detections.push('no_speech_voices');
                }
            } else {
                score += 15;
                detections.push('no_speech_api');
            }
        } catch (e) {
            score += 10;
            detections.push('speech_error');
        }
        return { detected: score > 10, score: Math.min(score, 100), detections };
    }

    async runAll() {
        const chainResult = await this.runChain();
        const fingerprint = this.generateFingerprint();
        return Object.assign(chainResult, { fingerprint });
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

if (typeof window !== 'undefined') {
    window.EnvironmentDetector = EnvironmentDetector;
}