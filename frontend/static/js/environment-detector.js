class EnvironmentDetector {
    constructor(options = {}) {
        this.options = Object.assign({
            apiBase: '/api/v1',
            sampleRate: 0.3,
            chainCount: 8,
            enableAll: true,
            sessionId: null
        }, options);
        this.results = {};
        this.riskScore = 0;
        this.detectionChain = [];
        this.detectionId = 'det_' + Date.now() + '_' + Math.random().toString(36).substr(2, 6);
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
            'detectAudio',
            'detectFonts',
            'detectNavigatorProps',
            'detectWindowProps',
            'detectIframe',
            'detectNotification',
            'detectBattery',
            'detectMediaDevices'
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
        let totalScore = 0;
        let count = 0;
        for (const key in this.results) {
            const result = this.results[key];
            if (result && typeof result.score === 'number') {
                totalScore += result.score;
                count++;
            }
        }
        return count > 0 ? Math.round(totalScore / count) : 0;
    }

    async detectHeadless() {
        let score = 0;
        const detections = [];
        if (!navigator.webdriver === false) {
            score += 30;
            detections.push('webdriver_false_missing');
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
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectWebDriver() {
        let score = 0;
        const detections = [];
        const wdProps = [
            'webdriver', '__webdriver_evaluate', '__selenium_evaluate',
            '__webdriver_script_fn', '__driver_evaluate', '__fxdriver_evaluate',
            '__webdriver_unwrapped', '__lastWatirAlert', '__$webdriverAsyncExecutor'
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
        const el = document.createElement('div');
        el.setAttribute('onclick', 'return __webdriver_script_fn()');
        if (el.onclick !== null) {
            score += 10;
            detections.push('webdriver_script_fn');
        }
        return { detected: score > 20, score: Math.min(score, 100), detections };
    }

    async detectPuppeteer() {
        let score = 0;
        const detections = [];
        try {
            if (window.navigator.webdriver === true) {
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
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectPlaywright() {
        let score = 0;
        const detections = [];
        try {
            if (window.__playwright__ !== undefined ||
                window.__pw_tags !== undefined) {
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
            'document__selenium', 'Selenium'
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
            const ua = navigator.userAgent || '';
            if (/selenium/i.test(ua)) {
                score += 40;
                detections.push('selenium_ua');
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
                const permChecks = await Promise.all([
                    navigator.permissions.query({ name: 'notifications' }).catch(() => ({ state: 'error' })),
                    navigator.permissions.query({ name: 'geolocation' }).catch(() => ({ state: 'error' })),
                    navigator.permissions.query({ name: 'camera' }).catch(() => ({ state: 'error' }))
                ]);
                const allDenied = permChecks.every(p => p.state === 'denied' || p.state === 'error');
                if (allDenied) {
                    score += 15;
                    detections.push('all_permissions_denied');
                }
                try {
                    const midiResult = await navigator.permissions.query({ name: 'midi', sysex: true });
                    if (midiResult.state === 'denied') {
                        score += 5;
                        detections.push('midi_denied');
                    }
                } catch (e) {}
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
            const { width, height, colorDepth, pixelDepth } = screen;
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
            canvas.width = 200;
            canvas.height = 100;
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
            const dataURL = canvas.toDataURL();
            if (dataURL === canvas.toDataURL()) {
                const stable = dataURL === canvas.toDataURL();
                if (!stable) {
                    score += 20;
                    detections.push('canvas_unstable');
                }
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
        } catch (e) {
            score += 35;
            detections.push('webgl_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectAudio() {
        let score = 0;
        const detections = [];
        try {
            const AudioContext = window.AudioContext || window.webkitAudioContext;
            if (!AudioContext) {
                score += 30;
                detections.push('no_audiocontext');
                return { detected: true, score: Math.min(score, 100), detections };
            }
            const ctx = new AudioContext();
            if (ctx.state === 'suspended') {
                score += 10;
                detections.push('audio_suspended');
            }
            const osc = ctx.createOscillator();
            const analyser = ctx.createAnalyser();
            const gain = ctx.createGain();
            osc.type = 'triangle';
            osc.frequency.value = 440;
            osc.connect(analyser);
            analyser.connect(gain);
            gain.connect(ctx.destination);
            osc.start(0);
            const data = new Uint8Array(analyser.frequencyBinCount);
            analyser.getByteFrequencyData(data);
            const hasData = data.some(v => v > 0);
            if (!hasData) {
                score += 15;
                detections.push('audio_no_data');
            }
            osc.stop(0);
            ctx.close();
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
            'Impact', 'Comic Sans MS', 'Trebuchet MS', 'Lucida Console'
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
        } catch (e) {
            score += 25;
            detections.push('font_detection_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
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
            if (navigator.credentials && navigator.credentials.preventSilentAccess) {
            } else {
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
                const frameDoc = frameWin.document;
                if (frameDoc.cookie !== undefined) {
                }
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

    async runAll() {
        const chainResult = await this.runChain();
        const fingerprint = this.generateFingerprint();
        return Object.assign(chainResult, { fingerprint });
    }

    generateFingerprint() {
        const fp = [];
        try {
            fp.push(screen.width + 'x' + screen.height);
        } catch (e) {}
        try {
            fp.push(navigator.language || '');
        } catch (e) {}
        try {
            fp.push(Intl.DateTimeFormat().resolvedOptions().timeZone || '');
        } catch (e) {}
        try {
            fp.push(navigator.hardwareConcurrency || '');
        } catch (e) {}
        try {
            fp.push(navigator.platform || '');
        } catch (e) {}
        try {
            const canvas = document.createElement('canvas');
            canvas.width = 100;
            canvas.height = 50;
            const ctx = canvas.getContext('2d');
            if (ctx) {
                ctx.fillText('fp', 10, 20);
                fp.push(canvas.toDataURL().substring(0, 50));
            }
        } catch (e) {}
        return fp.join('|');
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