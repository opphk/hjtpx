(function(globalContext) {
    'use strict';

    var EnvironmentDetectorCore = (function() {
        var version = '2.0.0';

        function EnvironmentDetector(options) {
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

        EnvironmentDetector.prototype.getDetectionMethods = function() {
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
        };

        EnvironmentDetector.prototype.generateDetectionChain = function(count) {
            var allMethods = this.getDetectionMethods();
            var shuffled = [].concat(allMethods).sort(function() { return Math.random() - 0.5; });
            var selected = shuffled.slice(0, Math.min(count, allMethods.length));
            var methodAliases = {};
            selected.forEach(function(method, i) {
                methodAliases[method] = 'chk_' + i.toString(36) + '_' + Math.random().toString(36).substr(2, 4);
            });
            return { selected: selected, methodAliases: methodAliases };
        };

        EnvironmentDetector.prototype.async runChain() {
            var self = this;
            var _ref = this.generateDetectionChain(this.options.chainCount);
            var selected = _ref.selected;
            var methodAliases = _ref.methodAliases;
            this.detectionChain = selected;
            var chainResults = {};
            var startTime = performance.now();

            for (var i = 0; i < selected.length; i++) {
                var method = selected[i];
                try {
                    var alias = methodAliases[method];
                    var result = await this[method]();
                    chainResults[alias] = result;
                    this.results[method] = result;
                } catch (e) {
                    var alias = methodAliases[method];
                    chainResults[alias] = { detected: false, score: 0, error: e.message };
                }
            }

            var duration = performance.now() - startTime;
            this.riskScore = this.calculateRiskScore();

            return {
                detection_id: this.detectionId,
                chain: chainResults,
                chain_order: Object.values(methodAliases),
                risk_score: this.riskScore,
                duration_ms: Math.round(duration),
                timestamp: Date.now()
            };
        };

        EnvironmentDetector.prototype.calculateRiskScore = function() {
            var self = this;
            var weightedScore = 0;
            var totalWeight = 0;

            for (var key in this.results) {
                var result = this.results[key];
                if (result && typeof result.score === 'number') {
                    var weight = this.weights[key] || 5;
                    weightedScore += result.score * weight;
                    totalWeight += weight;
                }
            }

            if (totalWeight === 0) return 0;

            var baseScore = weightedScore / totalWeight;

            var autoTools = ['detectWebDriver', 'detectPuppeteer', 'detectPlaywright', 'detectSelenium'];
            var autoDetected = autoTools.filter(function(m) {
                var r = self.results[m];
                return r && r.detected === true;
            }).length;

            if (autoDetected >= 2) {
                baseScore = Math.min(baseScore * 1.5 + 20, 100);
            } else if (autoDetected >= 1) {
                baseScore = Math.min(baseScore * 1.3 + 10, 100);
            }

            var proxyIndicators = ['detectWebRTCIP', 'detectConnection'];
            var proxyAnomalies = proxyIndicators.filter(function(m) {
                var r = self.results[m];
                return r && r.score > 30;
            }).length;

            if (proxyAnomalies >= 2) {
                baseScore = Math.min(baseScore * 1.3 + 15, 100);
            }

            return Math.round(Math.min(Math.max(baseScore, 0), 100));
        };

        EnvironmentDetector.prototype.detectHeadless = async function() {
            var score = 0;
            var detections = [];
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
            var mimeTypes = navigator.mimeTypes;
            if (mimeTypes && mimeTypes.length === 0) {
                score += 20;
                detections.push('no_mimetypes');
            }
            try {
                var ua = navigator.userAgent || '';
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
            return { detected: score > 30, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectWebDriver = async function() {
            var score = 0;
            var detections = [];
            var wdProps = [
                'webdriver', '__webdriver_evaluate', '__selenium_evaluate',
                '__webdriver_script_fn', '__driver_evaluate', '__fxdriver_evaluate',
                '__webdriver_unwrapped', '__lastWatirAlert', '__$webdriverAsyncExecutor',
                'callSelenium', '__selenium', 'Selenium'
            ];
            for (var i = 0; i < wdProps.length; i++) {
                var prop = wdProps[i];
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
            return { detected: score > 20, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectPuppeteer = async function() {
            var score = 0;
            var detections = [];
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
                var userAgent = navigator.userAgent || '';
                if (/headless/i.test(userAgent)) {
                    score += 30;
                    detections.push('headless_ua');
                }
                if (/puppet/i.test(userAgent)) {
                    score += 40;
                    detections.push('puppeteer_ua');
                }
            } catch (e) {}
            return { detected: score > 30, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectPlaywright = async function() {
            var score = 0;
            var detections = [];
            try {
                if (window.__playwright__ !== undefined ||
                    window.__pw_tags !== undefined ||
                    window.__pw_resume__ !== undefined) {
                    score += 45;
                    detections.push('playwright_global');
                }
            } catch (e) {}
            try {
                var ua = navigator.userAgent || '';
                if (/playwright/i.test(ua)) {
                    score += 50;
                    detections.push('playwright_ua');
                }
            } catch (e) {}
            return { detected: score > 30, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectSelenium = async function() {
            var score = 0;
            var detections = [];
            var selProps = [
                'selenium', '_selenium', 'callSelenium', '__selenium',
                'document__selenium', 'Selenium', '__webdriver_script_fn',
                'Selenium.prototype'
            ];
            for (var i = 0; i < selProps.length; i++) {
                var prop = selProps[i];
                if (window[prop] !== undefined || document[prop] !== undefined) {
                    score += 20;
                    detections.push(prop);
                }
            }
            try {
                var ua = navigator.userAgent || '';
                if (/selenium/i.test(ua)) {
                    score += 40;
                    detections.push('selenium_ua');
                }
            } catch (e) {}
            return { detected: score > 20, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectChromeRuntime = async function() {
            var score = 0;
            var detections = [];
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
            return { detected: score > 30, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectPermissions = async function() {
            var score = 0;
            var detections = [];
            try {
                if (navigator.permissions && navigator.permissions.query) {
                    var permNames = ['notifications', 'geolocation', 'camera', 'microphone', 'midi'];
                    var self = this;
                    var permChecks = await Promise.all(
                        permNames.map(function(name) {
                            return navigator.permissions.query({ name: name }).catch(function() { return { state: 'error' }; });
                        })
                    );
                    var allDenied = permChecks.every(function(p) { return p.state === 'denied' || p.state === 'error'; });
                    if (allDenied) {
                        score += 20;
                        detections.push('all_permissions_denied');
                    }
                    var deniedCount = permChecks.filter(function(p) { return p.state === 'denied'; }).length;
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
            return { detected: score > 25, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectPlugins = async function() {
            var score = 0;
            var detections = [];
            try {
                var plugins = navigator.plugins;
                if (!plugins || plugins.length === 0) {
                    score += 25;
                    detections.push('no_plugins');
                } else {
                    var commonPlugins = ['PDF Viewer', 'Chrome PDF Viewer', 'Chromium PDF Viewer',
                        'Microsoft Edge PDF Viewer', 'WebKit built-in PDF'];
                    var hasPDF = Array.from(plugins).some(function(p) {
                        return commonPlugins.some(function(cp) { return p.name.includes(cp); });
                    });
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
            return { detected: score > 25, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectLanguages = async function() {
            var score = 0;
            var detections = [];
            try {
                var langs = navigator.languages;
                if (!langs || langs.length === 0) {
                    score += 25;
                    detections.push('no_languages');
                }
                var lang = navigator.language;
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
            return { detected: score > 25, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectTimezone = async function() {
            var score = 0;
            var detections = [];
            try {
                var tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
                if (!tz) {
                    score += 30;
                    detections.push('no_timezone');
                }
                var offset = new Date().getTimezoneOffset();
                if (offset === 0 && !tz) {
                    score += 20;
                    detections.push('utc_offset_no_tz');
                }
                var year = new Date().getFullYear();
                if (year < 2000 || year > 2100) {
                    score += 25;
                    detections.push('unrealistic_date');
                }
            } catch (e) {
                score += 35;
                detections.push('timezone_error');
            }
            return { detected: score > 25, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectScreen = async function() {
            var score = 0;
            var detections = [];
            try {
                var width = screen.width;
                var height = screen.height;
                var colorDepth = screen.colorDepth;
                var pixelDepth = screen.pixelDepth;
                var availWidth = screen.availWidth;
                var availHeight = screen.availHeight;
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
                if (availWidth === 0 || availHeight === 0) {
                    score += 15;
                    detections.push('zero_avail_size');
                }
            } catch (e) {
                score += 30;
                detections.push('screen_error');
            }
            return { detected: score > 30, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectHardwareConcurrency = async function() {
            var score = 0;
            var detections = [];
            try {
                var c = navigator.hardwareConcurrency;
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
            return { detected: score > 25, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectDeviceMemory = async function() {
            var score = 0;
            var detections = [];
            try {
                var mem = navigator.deviceMemory;
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
            return { detected: score > 20, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectStorage = async function() {
            var score = 0;
            var detections = [];
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
                    var est = await navigator.storage.estimate();
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
            return { detected: score > 25, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectCanvas = async function() {
            var score = 0;
            var detections = [];
            try {
                var canvas = document.createElement('canvas');
                canvas.width = 280;
                canvas.height = 80;
                var ctx = canvas.getContext('2d');
                if (!ctx) {
                    score += 40;
                    detections.push('no_canvas_context');
                    return { detected: true, score: Math.min(score, 100), detections: detections };
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

                var dataURL = canvas.toDataURL();
                var dataURL2 = canvas.toDataURL();
                if (dataURL !== dataURL2) {
                    score += 25;
                    detections.push('canvas_unstable');
                }

                var imageData = ctx.getImageData(0, 0, 10, 10);
                var pixelSum = Array.from(imageData.data.slice(0, 40)).reduce(function(a, b) { return a + b; }, 0);
                if (pixelSum === 0) {
                    score += 20;
                    detections.push('canvas_empty_readback');
                }
            } catch (e) {
                score += 35;
                detections.push('canvas_error');
            }
            return { detected: score > 30, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectWebGL = async function() {
            var score = 0;
            var detections = [];
            try {
                var canvas = document.createElement('canvas');
                var gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
                if (!gl) {
                    score += 40;
                    detections.push('no_webgl');
                    return { detected: true, score: Math.min(score, 100), detections: detections };
                }
                var debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                if (debugInfo) {
                    var renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                    if (!renderer) {
                        score += 15;
                        detections.push('webgl_no_renderer');
                    }
                    if (/swiftshader|llvmpipe|mesa|virtual|google\s*inc/i.test(renderer || '')) {
                        score += 30;
                        detections.push('software_renderer');
                    }
                } else {
                    score += 20;
                    detections.push('no_webgl_debug');
                }
                var maxTexSize = gl.getParameter(gl.MAX_TEXTURE_SIZE);
                if (maxTexSize <= 1024) {
                    score += 15;
                    detections.push('small_tex_size');
                }
            } catch (e) {
                score += 35;
                detections.push('webgl_error');
            }
            return { detected: score > 30, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectWebGL2 = async function() {
            var score = 0;
            var detections = [];
            try {
                var canvas = document.createElement('canvas');
                var gl2 = canvas.getContext('webgl2');
                if (!gl2) {
                    return { detected: false, score: 0, detections: ['no_webgl2'] };
                }
                var debugInfo = gl2.getExtension('WEBGL_debug_renderer_info');
                if (debugInfo) {
                    var renderer = gl2.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                    if (/swiftshader|llvmpipe|mesa|virtual/i.test(renderer || '')) {
                        score += 25;
                        detections.push('webgl2_software_renderer');
                    }
                }
            } catch (e) {
                score += 20;
                detections.push('webgl2_error');
            }
            return { detected: score > 20, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectAudio = async function() {
            var score = 0;
            var detections = [];
            try {
                var AudioContext = window.OfflineAudioContext || window.webkitOfflineAudioContext;
                if (!AudioContext) {
                    score += 30;
                    detections.push('no_audiocontext');
                    return { detected: true, score: Math.min(score, 100), detections: detections };
                }
                var ctx = new AudioContext(1, 44100, 44100);
                var osc = ctx.createOscillator();
                osc.type = 'triangle';
                osc.frequency.setValueAtTime(10000, ctx.currentTime);
                var compressor = ctx.createDynamicsCompressor();
                compressor.threshold.setValueAtTime(-50, ctx.currentTime);
                compressor.knee.setValueAtTime(40, ctx.currentTime);
                compressor.ratio.setValueAtTime(12, ctx.currentTime);
                compressor.attack.setValueAtTime(0, ctx.currentTime);
                compressor.release.setValueAtTime(0.25, ctx.currentTime);
                osc.connect(compressor);
                compressor.connect(ctx.destination);
                osc.start(0);

                var startTime = performance.now();
                var buffer = await ctx.startRendering();
                var renderTime = performance.now() - startTime;

                if (renderTime < 5) {
                    score += 20;
                    detections.push('audio_render_too_fast');
                }
                var channelData = buffer.getChannelData(0);
                var sumAbs = 0;
                for (var i = 4500; i < 5000; i++) {
                    sumAbs += Math.abs(channelData[i]);
                }
                if (sumAbs === 0) {
                    score += 25;
                    detections.push('audio_silent');
                }
            } catch (e) {
                score += 30;
                detections.push('audio_error');
            }
            return { detected: score > 25, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectFonts = async function() {
            var score = 0;
            var detections = [];
            var baseFonts = ['monospace', 'sans-serif', 'serif'];
            var testFonts = [
                'Arial', 'Helvetica', 'Times New Roman', 'Courier New',
                'Verdana', 'Georgia', 'Palatino', 'Garamond',
                'Impact', 'Comic Sans MS', 'Trebuchet MS', 'Lucida Console',
                'Tahoma', 'Segoe UI', 'Roboto', 'Open Sans',
                'Lato', 'Montserrat', 'Source Sans Pro', 'Raleway',
                'Ubuntu', 'Noto Sans', 'Droid Sans', 'Fira Sans'
            ];
            try {
                var el = document.createElement('div');
                el.style.cssText = 'position:absolute;left:-9999px;font-size:72px;visibility:hidden;white-space:nowrap';
                el.textContent = 'mmmmmmmmmmlli';
                document.body.appendChild(el);
                var baseWidths = {};
                for (var i = 0; i < baseFonts.length; i++) {
                    el.style.fontFamily = baseFonts[i];
                    baseWidths[baseFonts[i]] = el.offsetWidth;
                }
                var fontCount = 0;
                for (var j = 0; j < testFonts.length; j++) {
                    for (var k = 0; k < baseFonts.length; k++) {
                        el.style.fontFamily = '"' + testFonts[j] + '", ' + baseFonts[k];
                        if (el.offsetWidth !== baseWidths[baseFonts[k]]) {
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
            return { detected: score > 25, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectWebRTCIP = async function() {
            var score = 0;
            var detections = [];
            try {
                var RTCPeerConnection = window.RTCPeerConnection ||
                    window.webkitRTCPeerConnection ||
                    window.mozRTCPeerConnection;
                if (!RTCPeerConnection) {
                    score += 15;
                    detections.push('no_webrtc');
                    return { detected: true, score: Math.min(score, 100), detections: detections };
                }
                var ips = {};
                var pc = new RTCPeerConnection({
                    iceServers: [
                        { urls: 'stun:stun.l.google.com:19302' },
                        { urls: 'stun:stun1.l.google.com:19302' },
                        { urls: 'stun:stun2.l.google.com:19302' }
                    ]
                });
                pc.createDataChannel('');
                var offer = await pc.createOffer();
                await pc.setLocalDescription(offer);
                var sdp = pc.localDescription.sdp;
                var lines = sdp.split('\n');
                for (var i = 0; i < lines.length; i++) {
                    var line = lines[i];
                    if (line.indexOf('candidate') > -1) {
                        var parts = line.split(' ');
                        if (parts[4] && parts[4] !== '0.0.0.0') {
                            ips[parts[4]] = true;
                            if (parts[7] !== 'host') {
                                detections.push('relay_ip:' + parts[4]);
                            }
                        }
                    }
                }
                pc.close();
                var ipCount = Object.keys(ips).length;
                if (ipCount > 1) {
                    var ipsArr = Object.keys(ips);
                    var publicIPs = ipsArr.filter(function(ip) {
                        return !ip.startsWith('10.') &&
                               !ip.startsWith('172.16.') &&
                               !ip.startsWith('192.168.');
                    });
                    if (publicIPs.length > 0) {
                        detections.push('public_ip_detected');
                        if (ipsArr.length > publicIPs.length) {
                            score += 20;
                            detections.push('vpn_possible');
                        }
                    }
                }
            } catch (e) {
                score += 15;
                detections.push('webrtc_error');
            }
            return { detected: score > 15, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectNavigatorProps = async function() {
            var score = 0;
            var detections = [];
            try {
                if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
                    var devices = await navigator.mediaDevices.enumerateDevices().catch(function() { return []; });
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
                if (navigator.serviceWorker === undefined) {
                    score += 10;
                    detections.push('no_serviceworker');
                }
            } catch (e) {}
            return { detected: score > 25, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectWindowProps = async function() {
            var score = 0;
            var detections = [];
            try {
                var outerW = window.outerWidth;
                var outerH = window.outerHeight;
                var innerW = window.innerWidth;
                var innerH = window.innerHeight;
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
            return { detected: score > 25, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectIframe = async function() {
            var score = 0;
            var detections = [];
            try {
                if (window.self !== window.top) {
                    score += 15;
                    detections.push('in_iframe');
                }
            } catch (e) {
                score += 35;
                detections.push('cross_origin_frame');
            }
            return { detected: score > 25, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectNotification = async function() {
            var score = 0;
            var detections = [];
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
            return { detected: score > 15, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectBattery = async function() {
            var score = 0;
            var detections = [];
            try {
                if (navigator.getBattery) {
                    var battery = await navigator.getBattery().catch(function() { return null; });
                    if (battery) {
                        if (battery.level === undefined || battery.charging === undefined) {
                            score += 15;
                            detections.push('battery_props_missing');
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
            return { detected: score > 15, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectMediaDevices = async function() {
            var score = 0;
            var detections = [];
            try {
                if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
                    var devices = await navigator.mediaDevices.enumerateDevices().catch(function() { return []; });
                    var videoInputs = devices.filter(function(d) { return d.kind === 'videoinput'; });
                    var audioInputs = devices.filter(function(d) { return d.kind === 'audioinput'; });
                    if (videoInputs.length === 0 && audioInputs.length === 0) {
                        score += 20;
                        detections.push('no_media_inputs');
                    }
                } else {
                    score += 15;
                    detections.push('no_media_api');
                }
            } catch (e) {
                score += 20;
                detections.push('media_error');
            }
            return { detected: score > 20, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectConnection = async function() {
            var score = 0;
            var detections = [];
            try {
                var conn = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
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
                }
            } catch (e) {
                score += 10;
                detections.push('connection_error');
            }
            return { detected: score > 20, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectAdBlock = async function() {
            var score = 0;
            var detections = [];
            try {
                var el = document.createElement('div');
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
            return { detected: score > 10, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectMathFingerprint = async function() {
            var score = 0;
            var detections = [];
            try {
                var mathResults = {
                    sin: Math.sin(Math.PI / 3),
                    tan: Math.tan(1e7),
                    log10: Math.log10(100),
                    asin: Math.asin(0.5),
                    atan2: Math.atan2(1, 2),
                    cos: Math.cos(Math.PI / 4),
                    exp: Math.exp(1),
                    sqrt: Math.sqrt(2)
                };
                for (var key in mathResults) {
                    if (!isFinite(mathResults[key]) || isNaN(mathResults[key])) {
                        score += 15;
                        detections.push('math_' + key + '_invalid');
                    }
                }
            } catch (e) {
                score += 20;
                detections.push('math_error');
            }
            return { detected: score > 15, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectGPUFingerprint = async function() {
            var score = 0;
            var detections = [];
            try {
                var canvas = document.createElement('canvas');
                var gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
                if (gl) {
                    var maxRenderSize = gl.getParameter(gl.MAX_RENDERBUFFER_SIZE);
                    var maxCombinedTexUnits = gl.getParameter(gl.MAX_COMBINED_TEXTURE_IMAGE_UNITS);
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
            return { detected: score > 15, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.detectSpeech = async function() {
            var score = 0;
            var detections = [];
            try {
                if ('speechSynthesis' in window) {
                    var voices = window.speechSynthesis.getVoices();
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
            return { detected: score > 10, score: Math.min(score, 100), detections: detections };
        };

        EnvironmentDetector.prototype.runAll = async function() {
            var chainResult = await this.runChain();
            var fingerprint = this.generateFingerprint();
            return Object.assign(chainResult, { fingerprint: fingerprint });
        };

        EnvironmentDetector.prototype.generateFingerprint = function() {
            var components = [];
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
                var canvas = document.createElement('canvas');
                canvas.width = 100;
                canvas.height = 50;
                var ctx = canvas.getContext('2d');
                if (ctx) {
                    ctx.textBaseline = 'top';
                    ctx.font = '14px Arial';
                    ctx.fillStyle = '#f60';
                    ctx.fillRect(0, 0, 50, 50);
                    ctx.fillStyle = '#069';
                    ctx.fillText('fp', 10, 20);
                    var dataUrl = canvas.toDataURL();
                    var hash = dataUrl.split(',')[1] || dataUrl;
                    components.push('cnv:' + hash.substring(0, 32));
                }
            } catch (e) {}
            try {
                var gl = document.createElement('canvas').getContext('webgl');
                if (gl) {
                    var debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                    if (debugInfo) {
                        var renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                        components.push('wgl:' + (renderer || '').substring(0, 48));
                    }
                }
            } catch (e) {}
            try {
                var offset = new Date().getTimezoneOffset();
                components.push('tzoff:' + offset);
            } catch (e) {}
            try {
                components.push('cookie:' + (navigator.cookieEnabled ? '1' : '0'));
            } catch (e) {}
            return components.join('|');
        };

        EnvironmentDetector.prototype.toJSON = function() {
            return {
                risk_score: this.riskScore,
                chain_count: this.detectionChain.length,
                results: this.results
            };
        };

        return EnvironmentDetector;
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = EnvironmentDetectorCore;
    } else {
        globalContext.EnvironmentDetectorCore = EnvironmentDetectorCore;
    }

})(typeof window !== 'undefined' ? window : this);
