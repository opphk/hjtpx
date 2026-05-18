(function(globalContext) {
    'use strict';

    var CaptchaUtils = (function() {
        function debounce(func, wait) {
            var timeout;
            return function executedFunction() {
                var context = this;
                var args = arguments;
                var later = function() {
                    timeout = null;
                    func.apply(context, args);
                };
                clearTimeout(timeout);
                timeout = setTimeout(later, wait);
            };
        }

        function throttle(func, limit) {
            var inThrottle;
            return function() {
                var args = arguments;
                var context = this;
                if (!inThrottle) {
                    func.apply(context, args);
                    inThrottle = true;
                    setTimeout(function() {
                        inThrottle = false;
                    }, limit);
                }
            };
        }

        function generateUUID() {
            return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
                var r = Math.random() * 16 | 0;
                var v = c === 'x' ? r : (r & 0x3 | 0x8);
                return v.toString(16);
            });
        }

        function generateRandomString(length) {
            var chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
            var result = '';
            var randomValues = new Uint8Array(length);
            crypto.getRandomValues(randomValues);
            for (var i = 0; i < length; i++) {
                result += chars[randomValues[i] % chars.length];
            }
            return result;
        }

        function deepClone(obj) {
            if (obj === null || typeof obj !== 'object') {
                return obj;
            }
            if (obj instanceof Date) {
                return new Date(obj.getTime());
            }
            if (obj instanceof Array) {
                return obj.map(function(item) {
                    return deepClone(item);
                });
            }
            if (obj instanceof Object) {
                var clonedObj = {};
                for (var key in obj) {
                    if (obj.hasOwnProperty(key)) {
                        clonedObj[key] = deepClone(obj[key]);
                    }
                }
                return clonedObj;
            }
        }

        function isEmpty(value) {
            if (value === null || value === undefined) {
                return true;
            }
            if (typeof value === 'string') {
                return value.trim().length === 0;
            }
            if (Array.isArray(value)) {
                return value.length === 0;
            }
            if (typeof value === 'object') {
                return Object.keys(value).length === 0;
            }
            return false;
        }

        function getTimestamp() {
            return Date.now();
        }

        function formatTime(milliseconds) {
            if (milliseconds < 1000) {
                return milliseconds + 'ms';
            }
            var seconds = Math.floor(milliseconds / 1000);
            if (seconds < 60) {
                return seconds + 's';
            }
            var minutes = Math.floor(seconds / 60);
            var remainingSeconds = seconds % 60;
            return minutes + 'm ' + remainingSeconds + 's';
        }

        function clamp(value, min, max) {
            return Math.min(Math.max(value, min), max);
        }

        function randomInt(min, max) {
            return Math.floor(Math.random() * (max - min + 1)) + min;
        }

        function shuffleArray(array) {
            var shuffled = array.slice();
            for (var i = shuffled.length - 1; i > 0; i--) {
                var j = Math.floor(Math.random() * (i + 1));
                var temp = shuffled[i];
                shuffled[i] = shuffled[j];
                shuffled[j] = temp;
            }
            return shuffled;
        }

        function groupBy(array, key) {
            return array.reduce(function(result, item) {
                var group = typeof key === 'function' ? key(item) : item[key];
                (result[group] = result[group] || []).push(item);
                return result;
            }, {});
        }

        function omit(obj, keys) {
            var result = {};
            Object.keys(obj).forEach(function(key) {
                if (keys.indexOf(key) === -1) {
                    result[key] = obj[key];
                }
            });
            return result;
        }

        function pick(obj, keys) {
            var result = {};
            keys.forEach(function(key) {
                if (key in obj) {
                    result[key] = obj[key];
                }
            });
            return result;
        }

        function parseQueryString(query) {
            var params = {};
            if (!query) return params;
            query.replace(/([^?&=]+)=([^&]*)/g, function(_, key, value) {
                params[decodeURIComponent(key)] = decodeURIComponent(value);
            });
            return params;
        }

        function buildQueryString(params) {
            return Object.keys(params)
                .map(function(key) {
                    return encodeURIComponent(key) + '=' + encodeURIComponent(params[key]);
                })
                .join('&');
        }

        function isValidEmail(email) {
            var regex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
            return regex.test(email);
        }

        function isValidURL(url) {
            try {
                new URL(url);
                return true;
            } catch (e) {
                return false;
            }
        }

        function getBrowserInfo() {
            var ua = navigator.userAgent;
            var browser = 'Unknown';
            var version = '0';

            if (ua.indexOf('Firefox') > -1) {
                browser = 'Firefox';
                version = ua.match(/Firefox\/([\d.]+)/)[1];
            } else if (ua.indexOf('Chrome') > -1 && ua.indexOf('Edg') === -1) {
                browser = 'Chrome';
                version = ua.match(/Chrome\/([\d.]+)/)[1];
            } else if (ua.indexOf('Safari') > -1 && ua.indexOf('Chrome') === -1) {
                browser = 'Safari';
                version = ua.match(/Version\/([\d.]+)/)[1];
            } else if (ua.indexOf('Edg') > -1) {
                browser = 'Edge';
                version = ua.match(/Edg\/([\d.]+)/)[1];
            }

            return {
                browser: browser,
                version: version,
                userAgent: ua,
                platform: navigator.platform,
                language: navigator.language,
                languages: navigator.languages
            };
        }

        function getDeviceType() {
            var ua = navigator.userAgent;
            if (/(tablet|ipad|playbook|silk)|(android(?!.*mobi))/i.test(ua)) {
                return 'tablet';
            }
            if (/Mobile|Android|iP(hone|od)|IEMobile|BlackBerry|Kindle|Silk-Accelerated|(hpw|web)OS|Opera M(obi|ini)/.test(ua)) {
                return 'mobile';
            }
            return 'desktop';
        }

        function supportsTouch() {
            return ('ontouchstart' in window) || (navigator.maxTouchPoints > 0);
        }

        function supportsWebGL() {
            try {
                var canvas = document.createElement('canvas');
                return !!(window.WebGLRenderingContext && 
                    (canvas.getContext('webgl') || canvas.getContext('experimental-webgl')));
            } catch (e) {
                return false;
            }
        }

        function supportsWebGL2() {
            try {
                var canvas = document.createElement('canvas');
                return !!(window.WebGL2RenderingContext && canvas.getContext('webgl2'));
            } catch (e) {
                return false;
            }
        }

        function supportsLocalStorage() {
            try {
                var test = '__storage_test__';
                localStorage.setItem(test, test);
                localStorage.removeItem(test);
                return true;
            } catch (e) {
                return false;
            }
        }

        function supportsSessionStorage() {
            try {
                var test = '__session_test__';
                sessionStorage.setItem(test, test);
                sessionStorage.removeItem(test);
                return true;
            } catch (e) {
                return false;
            }
        }

        function supportsWebSocket() {
            return 'WebSocket' in window || 'MozWebSocket' in window;
        }

        function supportsServiceWorker() {
            return 'serviceWorker' in navigator;
        }

        function getConnectionType() {
            var conn = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
            if (!conn) return 'unknown';
            return conn.type || 'unknown';
        }

        function getEffectiveType() {
            var conn = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
            if (!conn) return 'unknown';
            return conn.effectiveType || 'unknown';
        }

        function observePerformance(callback) {
            if (!window.PerformanceObserver) {
                return null;
            }

            var observer = new PerformanceObserver(function(list) {
                list.getEntries().forEach(function(entry) {
                    callback(entry);
                });
            });

            try {
                observer.observe({ entryTypes: ['navigation', 'resource', 'paint', 'measure'] });
            } catch (e) {
                return null;
            }

            return observer;
        }

        function getPageLoadTime() {
            var perfData = window.performance || window.webkitPerformance;
            if (!perfData) return null;

            var timing = perfData.timing;
            if (!timing) return null;

            return timing.loadEventEnd - timing.navigationStart;
        }

        function getDOMContentLoadedTime() {
            var perfData = window.performance || window.webkitPerformance;
            if (!perfData) return null;

            var timing = perfData.timing;
            if (!timing) return null;

            return timing.domContentLoadedEventEnd - timing.navigationStart;
        }

        function waitFor(condition, timeout, interval) {
            return new Promise(function(resolve, reject) {
                var startTime = Date.now();
                var checkInterval = interval || 100;
                var checkTimeout = timeout || 5000;

                function check() {
                    if (condition()) {
                        resolve(true);
                    } else if (Date.now() - startTime > checkTimeout) {
                        reject(new Error('Timeout waiting for condition'));
                    } else {
                        setTimeout(check, checkInterval);
                    }
                }

                check();
            });
        }

        function retry(fn, maxAttempts, delay) {
            return fn().catch(function(err) {
                if (maxAttempts <= 1) {
                    throw err;
                }
                return new Promise(function(resolve) {
                    setTimeout(function() {
                        resolve(retry(fn, maxAttempts - 1, delay));
                    }, delay || 1000);
                });
            });
        }

        function formatFileSize(bytes) {
            if (bytes === 0) return '0 Bytes';
            var k = 1024;
            var sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
            var i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }

        function formatNumber(num) {
            return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',');
        }

        function truncate(str, maxLength, suffix) {
            suffix = suffix || '...';
            if (str.length <= maxLength) {
                return str;
            }
            return str.substring(0, maxLength - suffix.length) + suffix;
        }

        function capitalizeFirst(str) {
            return str.charAt(0).toUpperCase() + str.slice(1);
        }

        function camelToSnake(str) {
            return str.replace(/[A-Z]/g, function(letter) {
                return '_' + letter.toLowerCase();
            });
        }

        function snakeToCamel(str) {
            return str.replace(/_([a-z])/g, function(letter) {
                return letter[1].toUpperCase();
            });
        }

        return {
            debounce: debounce,
            throttle: throttle,
            generateUUID: generateUUID,
            generateRandomString: generateRandomString,
            deepClone: deepClone,
            isEmpty: isEmpty,
            getTimestamp: getTimestamp,
            formatTime: formatTime,
            clamp: clamp,
            randomInt: randomInt,
            shuffleArray: shuffleArray,
            groupBy: groupBy,
            omit: omit,
            pick: pick,
            parseQueryString: parseQueryString,
            buildQueryString: buildQueryString,
            isValidEmail: isValidEmail,
            isValidURL: isValidURL,
            getBrowserInfo: getBrowserInfo,
            getDeviceType: getDeviceType,
            supportsTouch: supportsTouch,
            supportsWebGL: supportsWebGL,
            supportsWebGL2: supportsWebGL2,
            supportsLocalStorage: supportsLocalStorage,
            supportsSessionStorage: supportsSessionStorage,
            supportsWebSocket: supportsWebSocket,
            supportsServiceWorker: supportsServiceWorker,
            getConnectionType: getConnectionType,
            getEffectiveType: getEffectiveType,
            observePerformance: observePerformance,
            getPageLoadTime: getPageLoadTime,
            getDOMContentLoadedTime: getDOMContentLoadedTime,
            waitFor: waitFor,
            retry: retry,
            formatFileSize: formatFileSize,
            formatNumber: formatNumber,
            truncate: truncate,
            capitalizeFirst: capitalizeFirst,
            camelToSnake: camelToSnake,
            snakeToCamel: snakeToCamel
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = CaptchaUtils;
    } else {
        globalContext.CaptchaUtils = CaptchaUtils;
    }

})(typeof window !== 'undefined' ? window : this);
