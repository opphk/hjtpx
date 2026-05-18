/**
 * Captcha Environment Detector
 * 环境检测脚本 - 用于性能优化
 */
(function() {
    'use strict';

    var CaptchaEnv = {
        // 检测是否为移动设备
        isMobile: function() {
            return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
        },

        // 检测是否为触摸设备
        isTouchDevice: function() {
            return ('ontouchstart' in window) || (navigator.maxTouchPoints > 0);
        },

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