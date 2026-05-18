/**
 * Captcha Environment Detector
 * 环境检测脚本 - 用于性能优化
 * 增强版：包含自动化工具检测、浏览器指纹分析、网络环境检测
 */
(function() {
    'use strict';

    var CaptchaEnv = {
        
        // ==================== 基础环境检测 ====================
        
        // 检测是否为移动设备
        isMobile: function() {
            return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
        },

        // 检测是否为触摸设备
        isTouchDevice: function() {
            return ('ontouchstart' in window) || (navigator.maxTouchPoints > 0);
        },

        // ==================== 自动化工具检测 ====================
        
        // 自动化工具检测结果缓存
        _automationCache: null,
        
        // 检测自动化工具
        detectAutomation: function() {
            if (this._automationCache) {
                return this._automationCache;
            }
            
            var results = {
                isHeadless: false,
                isPuppeteer: false,
                isPlaywright: false,
                isSelenium: false,
                isCypress: false,
                isNightmare: false,
                isTestCafe: false,
                isWebDriverIO: false,
                isPhantomJS: false,
                automationScore: 0,
                detections: []
            };
            
            var ua = navigator.userAgent.toLowerCase();
            
            // Headless Chrome 检测
            if (ua.indexOf('headless') > -1 || ua.indexOf('phantom') > -1) {
                results.isHeadless = true;
                results.automationScore += 30;
                results.detections.push('headless_chrome_ua');
            }
            
            // Puppeteer 检测
            if (ua.indexOf('puppeteer') > -1 || document.$cdc_asdjflasutopfhvcZLmcfl_) {
                results.isPuppeteer = true;
                results.automationScore += 40;
                results.detections.push('puppeteer_detected');
            }
            
            // Playwright 检测
            if (window.__playwright__ || window.__pw_tags || window.__pw_resume__) {
                results.isPlaywright = true;
                results.automationScore += 45;
                results.detections.push('playwright_detected');
            }
            
            // Selenium 检测
            if (ua.indexOf('selenium') > -1 || ua.indexOf('webdriver') > -1) {
                results.isSelenium = true;
                results.automationScore += 35;
                results.detections.push('selenium_webdriver_ua');
            }
            
            // Cypress 检测
            if (window.__cypress__ || ua.indexOf('cypress') > -1) {
                results.isCypress = true;
                results.automationScore += 40;
                results.detections.push('cypress_detected');
            }
            
            // Nightmare/Electron 检测
            if (window.Nightmare || ua.indexOf('nightmare') > -1 || ua.indexOf('electron') > -1) {
                results.isNightmare = true;
                results.automationScore += 30;
                results.detections.push('nightmare_electron_detected');
            }
            
            // TestCafe 检测
            if (window.__TESTCAFE || ua.indexOf('testcafe') > -1) {
                results.isTestCafe = true;
                results.automationScore += 35;
                results.detections.push('testcafe_detected');
            }
            
            // WebDriverIO 检测
            if (window.WebDriver || ua.indexOf('webdriverio') > -1 || ua.indexOf('wdio') > -1) {
                results.isWebDriverIO = true;
                results.automationScore += 35;
                results.detections.push('webdriverio_detected');
            }
            
            // PhantomJS 检测
            if (ua.indexOf('phantomjs') > -1) {
                results.isPhantomJS = true;
                results.automationScore += 40;
                results.detections.push('phantomjs_detected');
            }
            
            // navigator.webdriver 检测
            if (navigator.webdriver === true) {
                results.automationScore += 35;
                results.detections.push('navigator_webdriver_true');
            }
            
            // 无插件检测
            if (navigator.plugins && navigator.plugins.length === 0) {
                results.automationScore += 15;
                results.detections.push('no_plugins');
            }
            
            // 无语言设置检测
            if (navigator.languages && navigator.languages.length === 0) {
                results.automationScore += 15;
                results.detections.push('no_languages');
            }
            
            // WebGL 软件渲染器检测
            try {
                var canvas = document.createElement('canvas');
                var gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
                if (gl) {
                    var debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                    if (debugInfo) {
                        var renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                        if (renderer && (renderer.indexOf('SwiftShader') > -1 || 
                            renderer.indexOf('llvmpipe') > -1 || 
                            renderer.indexOf('Software') > -1)) {
                            results.automationScore += 25;
                            results.detections.push('software_rendering_detected');
                        }
                    }
                }
            } catch (e) {}
            
            // 限制分数在0-100之间
            results.automationScore = Math.min(100, Math.max(0, results.automationScore));
            
            this._automationCache = results;
            return results;
        },
        
        // ==================== 浏览器指纹分析 ====================
        
        // 生成浏览器指纹
        generateFingerprint: function() {
            var components = [];
            
            // User Agent
            components.push('ua:' + (navigator.userAgent || ''));
            
            // 语言
            components.push('lang:' + (navigator.language || ''));
            components.push('langs:' + ((navigator.languages || []).join(',')));
            
            // 屏幕信息
            components.push('screen:' + (screen.width || 0) + 'x' + (screen.height || 0) + 'x' + (screen.colorDepth || 0));
            
            // 时区
            components.push('tz:' + (new Date().getTimezoneOffset() || 0));
            
            // 平台
            components.push('plat:' + (navigator.platform || ''));
            
            // 硬件信息
            components.push('cpu:' + (navigator.hardwareConcurrency || 0));
            components.push('mem:' + (navigator.deviceMemory || 0));
            components.push('touch:' + (navigator.maxTouchPoints || 0));
            
            // Canvas 指纹
            try {
                var canvas = document.createElement('canvas');
                canvas.width = 200;
                canvas.height = 50;
                var ctx = canvas.getContext('2d');
                if (ctx) {
                    ctx.textBaseline = 'top';
                    ctx.font = '14px Arial';
                    ctx.fillStyle = '#f60';
                    ctx.fillRect(0, 0, 50, 50);
                    ctx.fillStyle = '#069';
                    ctx.fillText('fingerprint', 10, 20);
                    var dataUrl = canvas.toDataURL();
                    components.push('canvas:' + dataUrl.substring(0, 100));
                }
            } catch (e) {
                components.push('canvas:error');
            }
            
            // WebGL 信息
            try {
                var canvas = document.createElement('canvas');
                var gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
                if (gl) {
                    var debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                    if (debugInfo) {
                        var vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
                        var renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                        components.push('webgl_vendor:' + (vendor || ''));
                        components.push('webgl_renderer:' + (renderer || ''));
                    }
                }
            } catch (e) {
                components.push('webgl:error');
            }
            
            // 插件
            var plugins = [];
            if (navigator.plugins) {
                for (var i = 0; i < navigator.plugins.length; i++) {
                    plugins.push(navigator.plugins[i].name);
                }
            }
            components.push('plugins:' + plugins.join(','));
            
            return this._hashString(components.join('|'));
        },
        
        // 计算字符串哈希
        _hashString: function(str) {
            var hash = 0;
            for (var i = 0; i < str.length; i++) {
                var char = str.charCodeAt(i);
                hash = ((hash << 5) - hash) + char;
                hash = hash & hash;
            }
            return Math.abs(hash).toString(16);
        },
        
        // 分析指纹异常
        analyzeFingerprintAnomalies: function() {
            var anomalies = [];
            
            // Canvas 指纹异常
            try {
                var canvas = document.createElement('canvas');
                var ctx = canvas.getContext('2d');
                if (ctx) {
                    ctx.textBaseline = 'top';
                    ctx.font = '14px Arial';
                    ctx.fillText('test', 2, 2);
                    var dataURL = canvas.toDataURL();
                    if (dataURL.indexOf('data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==') === 0) {
                        anomalies.push('canvas_fingerprint_missing');
                    }
                }
            } catch (e) {
                anomalies.push('canvas_access_blocked');
            }
            
            // WebGL 异常
            try {
                var canvas = document.createElement('canvas');
                var gl = canvas.getContext('webgl');
                if (!gl) {
                    anomalies.push('webgl_not_supported');
                } else {
                    var debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                    if (!debugInfo) {
                        anomalies.push('webgl_debug_blocked');
                    }
                }
            } catch (e) {
                anomalies.push('webgl_access_error');
            }
            
            // 语言异常
            if (!navigator.languages || navigator.languages.length === 0) {
                anomalies.push('no_languages');
            } else if (navigator.languages.length === 1 && navigator.languages[0] === 'en-US') {
                anomalies.push('default_language_only');
            }
            
            // 插件异常
            if (!navigator.plugins || navigator.plugins.length === 0) {
                anomalies.push('no_plugins');
            }
            
            return anomalies;
        },
        
        // ==================== 网络环境检测 ====================
        
        // 检测VPN
        detectVPN: function() {
            var results = {
                isVPN: false,
                confidence: 0,
                evidence: []
            };
            
            // WebRTC 检测
            try {
                var RTCPeerConnection = window.RTCPeerConnection || window.webkitRTCPeerConnection;
                if (RTCPeerConnection) {
                    var pc = new RTCPeerConnection({
                        iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
                    });
                    pc.createDataChannel('');
                    pc.onicecandidate = function(evt) {
                        if (evt.candidate) {
                            var candidate = evt.candidate.candidate;
                            if (candidate.indexOf('srflx') > -1 || candidate.indexOf('relay') > -1) {
                                results.isVPN = true;
                                results.confidence += 30;
                                results.evidence.push('webrtc_candidate_type');
                            }
                        }
                    };
                    pc.createOffer().then(function(offer) {
                        pc.setLocalDescription(offer);
                    }).catch(function() {});
                    setTimeout(function() { pc.close(); }, 1000);
                }
            } catch (e) {}
            
            // 连接类型检测
            if (navigator.connection) {
                var conn = navigator.connection;
                if (conn.type === 'vpn' || conn.type === 'pptp' || conn.type === 'tunnel') {
                    results.isVPN = true;
                    results.confidence += 50;
                    results.evidence.push('connection_type_vpn');
                }
            }
            
            return results;
        },
        
        // 检测代理
        detectProxy: function() {
            var results = {
                isProxy: false,
                confidence: 0,
                evidence: []
            };
            
            // WebRTC 本地IP泄露
            try {
                var RTCPeerConnection = window.RTCPeerConnection || window.webkitRTCPeerConnection;
                if (RTCPeerConnection) {
                    var pc = new RTCPeerConnection({
                        iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
                    });
                    pc.createDataChannel('');
                    pc.onicecandidate = function(evt) {
                        if (evt.candidate) {
                            var candidate = evt.candidate.candidate;
                            if (candidate.indexOf('srflx') > -1 || candidate.indexOf('prflx') > -1) {
                                results.isProxy = true;
                                results.confidence += 25;
                                results.evidence.push('webrtc_srflx_candidate');
                            }
                        }
                    };
                    pc.createOffer().then(function(offer) {
                        pc.setLocalDescription(offer);
                    }).catch(function() {});
                    setTimeout(function() { pc.close(); }, 1000);
                }
            } catch (e) {}
            
            // 连接类型
            if (navigator.connection) {
                var conn = navigator.connection;
                if (conn.type === 'proxy' || conn.type === 'socks') {
                    results.isProxy = true;
                    results.confidence += 60;
                    results.evidence.push('connection_type_proxy');
                }
            }
            
            return results;
        },
        
        // 检测 Tor
        detectTor: function() {
            var results = {
                isTor: false,
                confidence: 0,
                evidence: []
            };
            
            // Tor 检测模式
            var ua = navigator.userAgent.toLowerCase();
            if (ua.indexOf('tor') > -1 || ua.indexOf('onion') > -1) {
                results.isTor = true;
                results.confidence += 70;
                results.evidence.push('tor_ua_signature');
            }
            
            // WebRTC Tor检测
            try {
                var RTCPeerConnection = window.RTCPeerConnection || window.webkitRTCPeerConnection;
                if (RTCPeerConnection) {
                    var pc = new RTCPeerConnection({
                        iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
                    });
                    pc.createDataChannel('');
                    pc.onicecandidate = function(evt) {
                        if (evt.candidate) {
                            var candidate = evt.candidate.candidate;
                            if (candidate.indexOf('tcp') > -1 && candidate.indexOf('typel') > -1) {
                                results.isTor = true;
                                results.confidence += 50;
                                results.evidence.push('tor_tcp_candidate');
                            }
                        }
                    };
                    pc.createOffer().then(function(offer) {
                        pc.setLocalDescription(offer);
                    }).catch(function() {});
                    setTimeout(function() { pc.close(); }, 1000);
                }
            } catch (e) {}
            
            return results;
        },
        
        // 网络延迟分析
        analyzeNetworkLatency: function() {
            var results = {
                avgLatency: 0,
                variance: 0,
                anomalies: []
            };
            
            // 测量延迟
            var latencies = [];
            var startTime = performance.now();
            
            // 简单的延迟测量（使用本地资源）
            try {
                var testStart = Date.now();
                var xhr = new XMLHttpRequest();
                xhr.open('GET', window.location.pathname + '?t=' + testStart, false);
                xhr.send();
                var latency = Date.now() - testStart;
                latencies.push(latency);
            } catch (e) {
                latencies.push(0);
            }
            
            // 计算平均延迟
            if (latencies.length > 0) {
                var sum = 0;
                for (var i = 0; i < latencies.length; i++) {
                    sum += latencies[i];
                }
                results.avgLatency = sum / latencies.length;
                
                // 计算方差
                var sqSum = 0;
                for (var i = 0; i < latencies.length; i++) {
                    sqSum += Math.pow(latencies[i] - results.avgLatency, 2);
                }
                results.variance = sqSum / latencies.length;
                
                // 检测异常
                if (results.avgLatency > 1000) {
                    results.anomalies.push('very_high_latency');
                }
                if (results.variance > 100) {
                    results.anomalies.push('high_variance');
                }
            }
            
            return results;
        },
        
        // ==================== 设备性能检测 ====================
        
        // 检测是否为低性能设备
        isLowPerformance: function() {
            return new Promise(function(resolve) {
                var canvas = document.createElement('canvas');
                var gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
                
                if (!gl) {
                    resolve(true);
                    return;
                }

                var debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                if (debugInfo) {
                    var renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                    var lowPowerGPUs = ['Intel', 'Mali-4', 'Adreno 3', 'PowerVR SGX'];
                    
                    for (var i = 0; i < lowPowerGPUs.length; i++) {
                        if (renderer.indexOf(lowPowerGPUs[i]) !== -1) {
                            resolve(true);
                            return;
                        }
                    }
                }

                var pixels = 0;
                var startTime = Date.now();
                for (var i = 0; i < 100000; i++) {
                    gl.clear(gl.COLOR_BUFFER_BIT);
                    pixels++;
                }
                var duration = Date.now() - startTime;
                
                resolve(duration > 50);
            });
        },

        // 获取网络类型
        getNetworkInfo: function() {
            var connection = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
            
            if (connection) {
                return {
                    type: connection.effectiveType || 'unknown',
                    downlink: connection.downlink || 0,
                    rtt: connection.rtt || 0,
                    saveData: connection.saveData || false
                };
            }
            
            return {
                type: 'unknown',
                downlink: 0,
                rtt: 0,
                saveData: false
            };
        },

        // 获取设备内存信息
        getDeviceMemory: function() {
            if (navigator.deviceMemory) {
                return navigator.deviceMemory;
            }
            return null;
        },

        // 获取CPU核心数
        getHardwareConcurrency: function() {
            return navigator.hardwareConcurrency || 4;
        },

        // 检测是否支持某些功能
        supports: {
            webp: function() {
                var canvas = document.createElement('canvas');
                canvas.width = 1;
                canvas.height = 1;
                return canvas.toDataURL('image/webp').indexOf('data:image/webp') === 0;
            },
            
            IntersectionObserver: 'IntersectionObserver' in window,
            
            MutationObserver: 'MutationObserver' in window,
            
            passiveEventListeners: function() {
                var supportsPassive = false;
                try {
                    var opts = Object.defineProperty({}, 'passive', {
                        get: function() { supportsPassive = true; }
                    });
                    window.addEventListener('test', null, opts);
                } catch (e) {}
                return supportsPassive;
            },
            
            serviceWorker: 'serviceWorker' in navigator,
            
            webWorkers: 'Worker' in window,
            
            performanceObserver: 'PerformanceObserver' in window,
            
            battery: 'getBattery' in navigator
        },

        // 获取浏览器信息
        getBrowserInfo: function() {
            var ua = navigator.userAgent;
            var browser = {};
            
            if (ua.indexOf('Firefox') > -1) {
                browser.name = 'Firefox';
                browser.version = ua.match(/Firefox\/([\d.]+)/)[1];
            } else if (ua.indexOf('Chrome') > -1) {
                browser.name = 'Chrome';
                browser.version = ua.match(/Chrome\/([\d.]+)/)[1];
            } else if (ua.indexOf('Safari') > -1) {
                browser.name = 'Safari';
                browser.version = ua.match(/Version\/([\d.]+)/)[1];
            } else if (ua.indexOf('Edge') > -1) {
                browser.name = 'Edge';
                browser.version = ua.match(/Edge\/([\d.]+)/)[1];
            } else {
                browser.name = 'Unknown';
                browser.version = '0';
            }
            
            return browser;
        },

        // 获取操作系统信息
        getOS: function() {
            var ua = navigator.userAgent;
            var os = {};
            
            if (ua.indexOf('Win') > -1) {
                os.name = 'Windows';
                os.version = ua.indexOf('Windows NT 10') > -1 ? '10' : 
                             ua.indexOf('Windows NT 6.3') > -1 ? '8.1' : '8';
            } else if (ua.indexOf('Mac') > -1) {
                os.name = 'macOS';
                os.version = ua.match(/Mac OS X ([\\d._]+)/)[1].replace(/_/g, '.');
            } else if (ua.indexOf('Linux') > -1) {
                os.name = 'Linux';
                os.version = '';
            } else if (ua.indexOf('Android') > -1) {
                os.name = 'Android';
                os.version = ua.match(/Android ([\\d.]+)/)[1];
            } else if (ua.indexOf('iOS') > -1) {
                os.name = 'iOS';
                os.version = ua.match(/OS ([\\d_]+)/)[1].replace(/_/g, '.');
            } else {
                os.name = 'Unknown';
                os.version = '';
            }
            
            return os;
        },

        // 获取视口信息
        getViewport: function() {
            return {
                width: Math.max(document.documentElement.clientWidth || 0, window.innerWidth || 0),
                height: Math.max(document.documentElement.clientHeight || 0, window.innerHeight || 0),
                pixelRatio: window.devicePixelRatio || 1
            };
        },

        // 获取推荐的质量等级
        getRecommendedQuality: function() {
            var network = this.getNetworkInfo();
            var memory = this.getDeviceMemory();
            var viewport = this.getViewport();
            
            if (network.saveData || network.type === '2g' || network.type === 'slow-2g') {
                return 'low';
            }
            
            if (network.type === '3g' || (memory && memory <= 2)) {
                return 'medium';
            }
            
            if ((memory && memory >= 8) || viewport.pixelRatio >= 2) {
                return 'high';
            }
            
            return 'medium';
        },

        // 获取性能等级
        getPerformanceLevel: function() {
            var viewport = this.getViewport();
            var memory = this.getDeviceMemory();
            var cores = this.getHardwareConcurrency();
            
            var score = 0;
            
            if (viewport.pixelRatio >= 2) score += 2;
            else if (viewport.pixelRatio >= 1.5) score += 1;
            
            if (memory && memory >= 8) score += 2;
            else if (memory && memory >= 4) score += 1;
            
            if (cores >= 8) score += 2;
            else if (cores >= 4) score += 1;
            
            if (this.isMobile()) score -= 1;
            
            if (score >= 5) return 'high';
            if (score >= 3) return 'medium';
            return 'low';
        },

        // 获取完整的性能报告
        getPerformanceReport: function() {
            var self = this;
            return {
                device: {
                    isMobile: this.isMobile(),
                    isTouchDevice: this.isTouchDevice(),
                    browser: this.getBrowserInfo(),
                    os: this.getOS(),
                    viewport: this.getViewport(),
                    memory: this.getDeviceMemory(),
                    cores: this.getHardwareConcurrency()
                },
                network: this.getNetworkInfo(),
                features: this.supports,
                performanceLevel: this.getPerformanceLevel(),
                recommendedQuality: this.getRecommendedQuality()
            };
        },
        
        // ==================== 综合安全检测报告 ====================
        
        // 获取完整的环境安全报告
        getSecurityReport: function() {
            var automation = this.detectAutomation();
            var fingerprint = this.generateFingerprint();
            var fingerprintAnomalies = this.analyzeFingerprintAnomalies();
            var vpn = this.detectVPN();
            var proxy = this.detectProxy();
            var tor = this.detectTor();
            var networkLatency = this.analyzeNetworkLatency();
            
            var riskScore = 0;
            var riskFactors = [];
            
            // 自动化工具风险
            if (automation.automationScore > 0) {
                riskScore += automation.automationScore;
                riskFactors.push({
                    type: 'automation',
                    score: automation.automationScore,
                    details: automation.detections
                });
            }
            
            // 指纹异常风险
            if (fingerprintAnomalies.length > 0) {
                riskScore += fingerprintAnomalies.length * 10;
                riskFactors.push({
                    type: 'fingerprint_anomaly',
                    score: fingerprintAnomalies.length * 10,
                    details: fingerprintAnomalies
                });
            }
            
            // VPN 风险
            if (vpn.isVPN) {
                riskScore += vpn.confidence * 0.5;
                riskFactors.push({
                    type: 'vpn',
                    score: vpn.confidence * 0.5,
                    details: vpn.evidence
                });
            }
            
            // 代理风险
            if (proxy.isProxy) {
                riskScore += proxy.confidence * 0.4;
                riskFactors.push({
                    type: 'proxy',
                    score: proxy.confidence * 0.4,
                    details: proxy.evidence
                });
            }
            
            // Tor 风险
            if (tor.isTor) {
                riskScore += tor.confidence * 0.6;
                riskFactors.push({
                    type: 'tor',
                    score: tor.confidence * 0.6,
                    details: tor.evidence
                });
            }
            
            // 网络延迟异常风险
            if (networkLatency.anomalies.length > 0) {
                riskScore += networkLatency.anomalies.length * 5;
                riskFactors.push({
                    type: 'network_latency',
                    score: networkLatency.anomalies.length * 5,
                    details: networkLatency.anomalies
                });
            }
            
            // 限制风险分数在0-100之间
            riskScore = Math.min(100, Math.max(0, riskScore));
            
            var riskLevel = 'low';
            if (riskScore >= 70) {
                riskLevel = 'critical';
            } else if (riskScore >= 50) {
                riskLevel = 'high';
            } else if (riskScore >= 30) {
                riskLevel = 'medium';
            }
            
            return {
                timestamp: Date.now(),
                fingerprint: fingerprint,
                riskScore: riskScore,
                riskLevel: riskLevel,
                riskFactors: riskFactors,
                automation: automation,
                network: {
                    vpn: vpn,
                    proxy: proxy,
                    tor: tor,
                    latency: networkLatency
                },
                anomalies: fingerprintAnomalies,
                recommendations: this._generateRecommendations(riskLevel, riskFactors)
            };
        },
        
        // 生成安全建议
        _generateRecommendations: function(riskLevel, riskFactors) {
            var recommendations = [];
            
            if (riskLevel === 'critical' || riskLevel === 'high') {
                recommendations.push('检测到高风险自动化工具活动，建议启用额外验证');
            }
            
            for (var i = 0; i < riskFactors.length; i++) {
                var factor = riskFactors[i];
                
                switch (factor.type) {
                    case 'automation':
                        recommendations.push('检测到自动化框架使用，建议进行行为分析');
                        break;
                    case 'fingerprint_anomaly':
                        recommendations.push('检测到指纹异常，可能存在隐私保护或自动化工具');
                        break;
                    case 'vpn':
                        recommendations.push('检测到VPN连接');
                        break;
                    case 'proxy':
                        recommendations.push('检测到代理服务器');
                        break;
                    case 'tor':
                        recommendations.push('检测到Tor网络');
                        break;
                }
            }
            
            if (recommendations.length === 0) {
                recommendations.push('环境检测正常');
            }
            
            return recommendations;
        },

        // 检测首屏渲染完成时间
        measureFirstPaint: function() {
            var timing = performance.timing;
            var result = {};
            
            if (timing) {
                result.domContentLoaded = timing.domContentLoadedEventEnd - timing.navigationStart;
                result.loadComplete = timing.loadEventEnd - timing.navigationStart;
                result.firstPaint = (window.performance && 
                    (window.performance.timing.msFirstPaint || 
                     window.performance.getEntriesByType('paint')[0])) || null;
            }
            
            return result;
        },

        // 检测CLS (Cumulative Layout Shift)
        measureCLS: function(callback) {
            if (!('PerformanceObserver' in window)) {
                callback(0);
                return;
            }
            
            var clsValue = 0;
            var clsEntries = [];
            
            var observer = new PerformanceObserver(function(list) {
                for (var i = 0; i < list.getEntries().length; i++) {
                    var entry = list.getEntries()[i];
                    if (!entry.hadRecentInput) {
                        clsEntries.push(entry);
                        clsValue += entry.value;
                    }
                }
            });
            
            try {
                observer.observe({ type: 'layout-shift', buffered: true });
            } catch (e) {
                callback(0);
                return;
            }
            
            setTimeout(function() {
                observer.disconnect();
                callback(clsValue);
            }, 1000);
        },

        // 检测LCP (Largest Contentful Paint)
        measureLCP: function(callback) {
            if (!('PerformanceObserver' in window)) {
                callback(null);
                return;
            }
            
            var lcpEntry = null;
            
            var observer = new PerformanceObserver(function(list) {
                var entries = list.getEntries();
                if (entries.length > 0) {
                    lcpEntry = entries[entries.length - 1];
                }
            });
            
            try {
                observer.observe({ type: 'largest-contentful-paint', buffered: true });
            } catch (e) {
                callback(null);
                return;
            }
            
            setTimeout(function() {
                observer.disconnect();
                if (lcpEntry) {
                    callback(lcpEntry.startTime);
                } else {
                    callback(null);
                }
            }, 3000);
        },

        // 检测FID (First Input Delay)
        measureFID: function(callback) {
            if (!('PerformanceObserver' in window)) {
                callback(null);
                return;
            }
            
            var fidValue = null;
            
            var observer = new PerformanceObserver(function(list) {
                var entries = list.getEntries();
                if (entries.length > 0) {
                    var entry = entries[0];
                    fidValue = entry.processingStart - entry.startTime;
                }
            });
            
            try {
                observer.observe({ type: 'first-input', buffered: true });
            } catch (e) {
                callback(null);
                return;
            }
            
            setTimeout(function() {
                observer.disconnect();
                callback(fidValue);
            }, 5000);
        },

        // 性能测试
        runPerformanceTests: function(callback) {
            var self = this;
            var results = {
                cls: 0,
                lcp: null,
                fid: null,
                firstPaint: this.measureFirstPaint()
            };
            
            var testsCompleted = 0;
            var totalTests = 3;
            
            function checkComplete() {
                testsCompleted++;
                if (testsCompleted >= totalTests) {
                    callback(results);
                }
            }
            
            this.measureCLS(function(cls) {
                results.cls = cls;
                checkComplete();
            });
            
            this.measureLCP(function(lcp) {
                results.lcp = lcp;
                checkComplete();
            });
            
            this.measureFID(function(fid) {
                results.fid = fid;
                checkComplete();
            });
        }
    };

    // 导出到全局对象
    window.CaptchaEnv = CaptchaEnv;

    // 自动检测并应用优化
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', function() {
            applyOptimizations();
        });
    } else {
        applyOptimizations();
    }

    function applyOptimizations() {
        var report = CaptchaEnv.getPerformanceReport();
        
        // 根据性能等级调整动画
        if (report.performanceLevel === 'low') {
            document.documentElement.classList.add('captcha-reduced-motion');
            document.documentElement.style.setProperty('--animation-duration', '0.01ms');
        }
        
        // 根据网络状况调整图片加载策略
        if (report.network.saveData) {
            document.documentElement.classList.add('captcha-save-data');
        }
        
        // 根据设备类型添加标记
        if (report.device.isMobile) {
            document.documentElement.classList.add('captcha-is-mobile');
        }
        
        if (report.device.isTouchDevice) {
            document.documentElement.classList.add('captcha-is-touch');
        }
        
        // 根据质量推荐应用图片质量
        document.documentElement.setAttribute('data-captcha-quality', report.recommendedQuality);
    }

})();