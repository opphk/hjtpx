(function(globalContext) {
    'use strict';

    var CryptoUtils = (function() {
        var version = '3.0.0';
        var defaultKey = 'hjtpx-obfuscate-key-2024';
        var storagePrefix = '_cry_';
        var debugDetectionEnabled = true;
        var integrityCheckInterval = null;
        var originalHash = null;
        var protectionActive = false;
        var selfDestructTriggered = false;
        var monitorInterval = null;

        var SignatureAlgorithm = {
            HMAC_SHA256: 'HMAC-SHA256',
            HMAC_SHA512: 'HMAC-SHA512',
            BLAKE2B_256: 'BLAKE2B-256',
            BLAKE2B_512: 'BLAKE2B-512'
        };

        var currentSignatureAlgorithm = SignatureAlgorithm.HMAC_SHA256;

        function CryptoError(message, code) {
            this.name = 'CryptoError';
            this.message = message;
            this.code = code || 'UNKNOWN_ERROR';
        }
        CryptoError.prototype = Object.create(Error.prototype);
        CryptoError.prototype.constructor = CryptoError;

        var AntiDebugManager = (function() {
            var instance = null;
            var detectors = [];
            var checkInterval = 2000;
            var compromised = false;
            var detectionCount = 0;
            var lastCheckTime = 0;

            function AntiDebugManager() {
                if (instance) return instance;
                instance = this;
                this.init();
            }

            AntiDebugManager.prototype.init = function() {
                this.registerDefaultDetectors();
                this.startMonitoring();
                this.preventDebugKeys();
                this.preventContextMenu();
            };

            AntiDebugManager.prototype.registerDefaultDetectors = function() {
                var self = this;

                this.registerDetector(function() {
                    var threshold = 160;
                    if (window.outerWidth - window.innerWidth > threshold ||
                        window.outerHeight - window.innerHeight > threshold) {
                        return true;
                    }
                    return false;
                });

                this.registerDetector(function() {
                    var start = performance.now();
                    debugger;
                    var end = performance.now();
                    if (end - start > 100) {
                        return true;
                    }
                    return false;
                });

                this.registerDetector(function() {
                    var result = false;
                    (function(x) {
                        var d = document.createElement('div');
                        d.innerHTML = '<x id="_detect"/>';
                        d.style.cssText = 'position:absolute;left:-9999px;top:-9999px;';
                        Object.defineProperty(x, 'inspect', {
                            get: function() {
                                result = true;
                                return function() {};
                            }
                        });
                        document.body.appendChild(d);
                        setTimeout(function() {
                            document.body.removeChild(d);
                        }, 100);
                    })(window);

                    if (result) return true;

                    try {
                        if (window.devtools && window.devtools.isOpen) {
                            return true;
                        }
                    } catch (e) {}

                    return false;
                });

                this.registerDetector(function() {
                    try {
                        if (Function.prototype.toString.toString().indexOf('[native code]') === -1) {
                            return true;
                        }
                    } catch (e) {}
                    return false;
                });

                this.registerDetector(function() {
                    try {
                        if (console.log.toString().indexOf('[native code]') === -1) {
                            return true;
                        }
                    } catch (e) {}
                    return false;
                });

                this.registerDetector(function() {
                    var ua = navigator.userAgent.toLowerCase();
                    var headlessIndicators = [
                        'headlesschrome', 'phantomjs', 'selenium', 'puppeteer',
                        'nightmare', 'slimerjs', 'zombie', 'ghost'
                    ];
                    for (var i = 0; i < headlessIndicators.length; i++) {
                        if (ua.indexOf(headlessIndicators[i]) !== -1) {
                            return true;
                        }
                    }

                    if (navigator.webdriver === true) return true;
                    if (window.callPhantom || window._phantom) return true;

                    try {
                        if (window.Buffer) return true;
                    } catch (e) {}

                    try {
                        var canvas = document.createElement('canvas');
                        var gl = canvas.getContext('webgl');
                        if (gl && gl.getExtension('WEBGL_debug_renderer_info')) {
                            var renderer = gl.getParameter(gl.getExtension('WEBGL_debug_renderer_info').UNMASKED_RENDERER_WEBGL);
                            if (renderer.indexOf('SwiftShader') !== -1 || renderer.indexOf('llvmpipe') !== -1) {
                                return true;
                            }
                        }
                    } catch (e) {}

                    return false;
                });

                this.registerDetector(function() {
                    try {
                        var testFunc = function() {};
                        var funcStr = testFunc.toString();
                        if (funcStr.indexOf('[native code]') === -1 && funcStr.indexOf('testFunc') === -1) {
                            return true;
                        }
                    } catch (e) {}
                    return false;
                });
            };

            AntiDebugManager.prototype.registerDetector = function(fn) {
                if (typeof fn === 'function') {
                    detectors.push(fn);
                }
            };

            AntiDebugManager.prototype.check = function() {
                if (compromised) return true;

                var now = Date.now();
                if (now - lastCheckTime < 500) return false;
                lastCheckTime = now;

                for (var i = 0; i < detectors.length; i++) {
                    try {
                        if (detectors[i]()) {
                            detectionCount++;
                            this.onDetected();
                            return true;
                        }
                    } catch (e) {}
                }
                return false;
            };

            AntiDebugManager.prototype.onDetected = function() {
                if (compromised) return;
                compromised = true;
                this.triggerSelfDestruct('debug_detected');
            };

            AntiDebugManager.prototype.triggerSelfDestruct = function(reason) {
                if (selfDestructTriggered) return;
                selfDestructTriggered = true;

                if (monitorInterval) {
                    clearInterval(monitorInterval);
                    monitorInterval = null;
                }

                if (document.documentElement) {
                    document.documentElement.style.display = 'none';
                }

                if (document.body) {
                    document.body.innerHTML = '<div style="position:fixed;top:0;left:0;right:0;bottom:0;display:flex;flex-direction:column;justify-content:center;align-items:center;background:#000;color:#fff;font-family:sans-serif;z-index:2147483647;min-height:100vh;">' +
                        '<h1 style="color:#ff0000;font-size:48px;margin-bottom:20px;">安全警告</h1>' +
                        '<p style="font-size:24px;margin-bottom:10px;">检测到异常访问行为</p>' +
                        '<p style="font-size:16px;color:#ff6666;">' + (reason || '访问已终止') + '</p>' +
                        '<p style="font-size:12px;margin-top:30px;">系统已自动保护</p></div>';
                }

                setTimeout(function() {
                    try {
                        var scripts = document.getElementsByTagName('script');
                        for (var i = scripts.length - 1; i >= 0; i--) {
                            try {
                                if (scripts[i].parentNode) {
                                    scripts[i].parentNode.removeChild(scripts[i]);
                                }
                            } catch (e) {}
                        }

                        Object.keys(window).forEach(function(key) {
                            if (key !== 'window' && key !== 'document' && key !== 'location' &&
                                key !== 'navigator' && key !== 'history' && key !== 'screen') {
                                try {
                                    if (typeof window[key] === 'function') {
                                        window[key] = function() {};
                                    }
                                } catch (e) {}
                            }
                        });
                    } catch (e) {}
                }, 100);

                throw new Error('Security violation: ' + (reason || 'Access denied'));
            };

            AntiDebugManager.prototype.startMonitoring = function() {
                var self = this;
                if (monitorInterval) {
                    clearInterval(monitorInterval);
                }

                monitorInterval = setInterval(function() {
                    self.check();
                }, checkInterval);
            };

            AntiDebugManager.prototype.preventDebugKeys = function() {
                var self = this;
                document.addEventListener('keydown', function(e) {
                    if (e.keyCode === 123) {
                        e.preventDefault();
                        self.triggerSelfDestruct('F12 key pressed');
                    }

                    if (e.keyCode === 116) {
                        e.preventDefault();
                    }

                    if (e.ctrlKey && e.shiftKey && e.keyCode === 73) {
                        e.preventDefault();
                        self.triggerSelfDestruct('Ctrl+Shift+I pressed');
                    }

                    if (e.ctrlKey && e.keyCode === 85) {
                        e.preventDefault();
                        self.triggerSelfDestruct('Ctrl+U pressed');
                    }

                    if (e.ctrlKey && e.shiftKey && e.keyCode === 74) {
                        e.preventDefault();
                        self.triggerSelfDestruct('Ctrl+Shift+J pressed');
                    }
                });
            };

            AntiDebugManager.prototype.preventContextMenu = function() {
                document.addEventListener('contextmenu', function(e) {
                    e.preventDefault();
                    return false;
                });
            };

            AntiDebugManager.prototype.getStatus = function() {
                return {
                    compromised: compromised,
                    detectionCount: detectionCount,
                    detectorCount: detectors.length,
                    monitoring: monitorInterval !== null
                };
            };

            AntiDebugManager.prototype.disable = function() {
                if (monitorInterval) {
                    clearInterval(monitorInterval);
                    monitorInterval = null;
                }
                detectors = [];
                compromised = false;
            };

            return AntiDebugManager;
        })();

        var MemoryGuard = (function() {
            var instance = null;
            var protectedObjects = {};
            var originalDescriptors = {};
            var originalToStrings = {};
            var checkInterval = null;

            function MemoryGuard() {
                if (instance) return instance;
                instance = this;
            }

            MemoryGuard.prototype.protect = function(obj, prop) {
                var key = obj + '::' + prop;
                if (protectedObjects[key]) return;

                try {
                    var descriptor = Object.getOwnPropertyDescriptor(window[obj], prop);
                    if (descriptor) {
                        originalDescriptors[key] = {
                            value: descriptor.value,
                            writable: descriptor.writable,
                            enumerable: descriptor.enumerable,
                            configurable: descriptor.configurable
                        };

                        if (descriptor.value) {
                            originalToStrings[key] = descriptor.value.toString();
                        }

                        var self = this;
                        Object.defineProperty(window[obj], prop, {
                            get: function() {
                                return originalDescriptors[key].value;
                            },
                            set: function(v) {
                                if (v && v.toString && v.toString().indexOf('[native code]') === -1) {
                                    throw new Error('Memory modification detected');
                                }
                                originalDescriptors[key].value = v;
                            },
                            enumerable: descriptor.enumerable,
                            configurable: false
                        });

                        protectedObjects[key] = true;
                    }
                } catch (e) {}
            };

            MemoryGuard.prototype.check = function() {
                for (var key in originalDescriptors) {
                    try {
                        var parts = key.split('::');
                        var obj = parts[0];
                        var prop = parts[1];

                        if (originalToStrings[key]) {
                            var current = window[obj][prop];
                            if (current && current.toString && current.toString().indexOf('[native code]') === -1) {
                                return true;
                            }
                        }
                    } catch (e) {
                        return true;
                    }
                }
                return false;
            };

            MemoryGuard.prototype.startProtection = function() {
                this.protect('Function', 'prototype');
                this.protect('console', 'log');
                this.protect('console', 'error');
                this.protect('console', 'warn');
                this.protect('console', 'info');

                var self = this;
                if (checkInterval) {
                    clearInterval(checkInterval);
                }

                checkInterval = setInterval(function() {
                    if (self.check()) {
                        if (typeof AntiDebugManager !== 'undefined') {
                            var ad = new AntiDebugManager();
                            ad.triggerSelfDestruct('Memory modification detected');
                        }
                    }
                }, 3000);
            };

            MemoryGuard.prototype.stopProtection = function() {
                if (checkInterval) {
                    clearInterval(checkInterval);
                    checkInterval = null;
                }
            };

            return MemoryGuard;
        })();

        var IntegrityChecker = (function() {
            var instance = null;
            var scriptHashes = {};
            var integrityHash = null;
            var checkInterval = null;

            function IntegrityChecker() {
                if (instance) return instance;
                instance = this;
            }

            IntegrityChecker.prototype.hashCode = function(str) {
                var hash = 0;
                for (var i = 0; i < str.length; i++) {
                    var char = str.charCodeAt(i);
                    hash = ((hash << 5) - hash) + char;
                    hash = hash & hash;
                }
                return hash.toString();
            };

            IntegrityChecker.prototype.computeSHA256 = async function(data) {
                var encoder = new TextEncoder();
                var dataBuffer = encoder.encode(data);
                var hashBuffer = await crypto.subtle.digest('SHA-256', dataBuffer);
                var bytes = new Uint8Array(hashBuffer);
                var binary = '';
                for (var i = 0; i < bytes.byteLength; i++) {
                    binary += String.fromCharCode(bytes[i]);
                }
                return btoa(binary);
            };

            IntegrityChecker.prototype.registerScript = function(script) {
                try {
                    var content = script.innerHTML || script.textContent;
                    if (content) {
                        var hash = this.hashCode(content);
                        scriptHashes[script.src || 'inline_' + Math.random()] = hash;
                        return hash;
                    }
                } catch (e) {}
                return null;
            };

            IntegrityChecker.prototype.checkScripts = function() {
                try {
                    var scripts = document.getElementsByTagName('script');
                    for (var i = 0; i < scripts.length; i++) {
                        var s = scripts[i];
                        if (s.src && s.src.indexOf('crypto-utils') !== -1) {
                            var content = s.innerHTML || s.textContent;
                            if (content) {
                                var currentHash = this.hashCode(content);
                                var storedHash = scriptHashes[s.src];

                                if (storedHash && storedHash !== currentHash) {
                                    return true;
                                }
                            }
                        }
                    }
                } catch (e) {}
                return false;
            };

            IntegrityChecker.prototype.setIntegrityHash = function(hash) {
                integrityHash = hash;
            };

            IntegrityChecker.prototype.verifyIntegrity = async function() {
                if (!integrityHash) return true;

                try {
                    var scripts = document.getElementsByTagName('script');
                    for (var i = 0; i < scripts.length; i++) {
                        var s = scripts[i];
                        if (s.src && s.src.indexOf('crypto-utils') !== -1) {
                            var content = s.innerHTML || s.textContent;
                            if (content) {
                                var currentHash = await this.computeSHA256(content);
                                if (currentHash !== integrityHash) {
                                    return false;
                                }
                            }
                        }
                    }
                } catch (e) {}
                return true;
            };

            IntegrityChecker.prototype.startMonitoring = function(interval) {
                var self = this;
                var scripts = document.getElementsByTagName('script');
                for (var i = 0; i < scripts.length; i++) {
                    this.registerScript(scripts[i]);
                }

                if (checkInterval) {
                    clearInterval(checkInterval);
                }

                checkInterval = setInterval(function() {
                    if (self.checkScripts()) {
                        if (typeof AntiDebugManager !== 'undefined') {
                            var ad = new AntiDebugManager();
                            ad.triggerSelfDestruct('Script tampering detected');
                        }
                    }
                }, interval || 5000);
            };

            IntegrityChecker.prototype.stopMonitoring = function() {
                if (checkInterval) {
                    clearInterval(checkInterval);
                    checkInterval = null;
                }
            };

            return IntegrityChecker;
        })();

        function arrayBufferToBase64(buffer) {
            var bytes = new Uint8Array(buffer);
            var binary = '';
            for (var i = 0; i < bytes.byteLength; i++) {
                binary += String.fromCharCode(bytes[i]);
            }
            return btoa(binary);
        }

        function base64ToArrayBuffer(base64) {
            var binaryString = atob(base64);
            var bytes = new Uint8Array(binaryString.length);
            for (var i = 0; i < binaryString.length; i++) {
                bytes[i] = binaryString.charCodeAt(i);
            }
            return bytes.buffer;
        }

        function deriveKey(password, salt, iterations, keyLength) {
            var encoder = new TextEncoder();
            var passwordBuffer = encoder.encode(password);
            var saltBuffer = salt instanceof ArrayBuffer ? new Uint8Array(salt) : salt;

            var importedKey = crypto.subtle.importKey(
                'raw',
                passwordBuffer,
                { name: 'PBKDF2' },
                false,
                ['deriveBits']
            );

            return importedKey.then(function(key) {
                return crypto.subtle.deriveBits(
                    {
                        name: 'PBKDF2',
                        salt: saltBuffer,
                        iterations: iterations || 100000,
                        hash: 'SHA-256'
                    },
                    key,
                    (keyLength || 256)
                );
            });
        }

        function generateRandomBytes(length) {
            var array = new Uint8Array(length);
            crypto.getRandomValues(array);
            return array;
        }

        function generateRandomString(length) {
            var chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
            var randomBytes = generateRandomBytes(length);
            var result = '';
            for (var i = 0; i < length; i++) {
                result += chars[randomBytes[i] % chars.length];
            }
            return result;
        }

        function hashSHA256(data) {
            var encoder = new TextEncoder();
            var dataBuffer = typeof data === 'string' ? encoder.encode(data) : data;
            return crypto.subtle.digest('SHA-256', dataBuffer).then(function(hash) {
                return arrayBufferToBase64(hash);
            });
        }

        async function encryptAES(plaintext, key, options) {
            options = options || {};
            var useGCM = options.mode !== 'CBC';

            var keyData = typeof key === 'string' ?
                await deriveKey(key, new Uint8Array([1, 2, 3, 4, 5, 6, 7, 8]), 100000, 256) :
                key;

            var importedKey = await crypto.subtle.importKey(
                'raw',
                keyData,
                { name: useGCM ? 'AES-GCM' : 'AES-CBC' },
                false,
                ['encrypt']
            );

            var iv = generateRandomBytes(useGCM ? 12 : 16);
            var encoder = new TextEncoder();
            var plaintextBuffer = encoder.encode(plaintext);

            var additionalData = options.aad ? encoder.encode(options.aad) : null;
            var algorithmOptions = useGCM ?
                { name: 'AES-GCM', iv: iv, tagLength: 128 } :
                { name: 'AES-CBC', iv: iv };

            if (additionalData && useGCM) {
                algorithmOptions.additionalData = additionalData;
            }

            var ciphertext = await crypto.subtle.encrypt(
                algorithmOptions,
                importedKey,
                plaintextBuffer
            );

            var result = {
                ciphertext: arrayBufferToBase64(ciphertext),
                iv: arrayBufferToBase64(iv),
                mode: useGCM ? 'GCM' : 'CBC'
            };

            if (additionalData) {
                result.aad = options.aad;
            }

            return result;
        }

        async function decryptAES(encryptedData, key, options) {
            options = options || {};
            var useGCM = (encryptedData.mode === 'GCM') || (options.mode === 'GCM');

            var keyData = typeof key === 'string' ?
                await deriveKey(key, new Uint8Array([1, 2, 3, 4, 5, 6, 7, 8]), 100000, 256) :
                key;

            var importedKey = await crypto.subtle.importKey(
                'raw',
                keyData,
                { name: useGCM ? 'AES-GCM' : 'AES-CBC' },
                false,
                ['decrypt']
            );

            var iv = new Uint8Array(base64ToArrayBuffer(encryptedData.iv));
            var ciphertext = base64ToArrayBuffer(encryptedData.ciphertext);

            var algorithmOptions = useGCM ?
                { name: 'AES-GCM', iv: iv, tagLength: 128 } :
                { name: 'AES-CBC', iv: iv };

            if (encryptedData.aad && useGCM) {
                var encoder = new TextEncoder();
                algorithmOptions.additionalData = encoder.encode(encryptedData.aad);
            }

            var plaintext = await crypto.subtle.decrypt(
                algorithmOptions,
                importedKey,
                ciphertext
            );

            var decoder = new TextDecoder();
            return decoder.decode(plaintext);
        }

        async function encryptString(plaintext, key) {
            if (!plaintext) {
                throw new CryptoError('Plaintext cannot be empty', 'EMPTY_INPUT');
            }

            var actualKey = key || defaultKey;
            var encrypted = await encryptAES(plaintext, actualKey);

            return arrayBufferToBase64(new Uint8Array([
                ...new Uint8Array([1]),
                ...new Uint8Array(base64ToArrayBuffer(encrypted.iv)),
                ...new Uint8Array(base64ToArrayBuffer(encrypted.ciphertext))
            ]).buffer);
        }

        async function decryptString(encryptedBase64, key) {
            if (!encryptedBase64) {
                throw new CryptoError('Encrypted data cannot be empty', 'EMPTY_INPUT');
            }

            try {
                var actualKey = key || defaultKey;
                var data = new Uint8Array(base64ToArrayBuffer(encryptedBase64));
                var iv = data.slice(1, 17);
                var ciphertext = data.slice(17);

                var encryptedData = {
                    ciphertext: arrayBufferToBase64(ciphertext.buffer),
                    iv: arrayBufferToBase64(iv.buffer),
                    mode: 'CBC'
                };

                return await decryptAES(encryptedData, actualKey);
            } catch (error) {
                throw new CryptoError('Decryption failed: ' + error.message, 'DECRYPTION_ERROR');
            }
        }

        function protectFunction(func, options) {
            options = options || {};
            var functionName = options.name || func.name || 'anonymous';
            var wrapCalls = options.wrapCalls !== false;
            var addIntegrityCheck = options.integrityCheck !== false;

            var callCount = 0;
            var lastCallTime = 0;
            var originalFunc = func;

            var protectedFunc = function() {
                if (debugDetectionEnabled && detectDebugging()) {
                    handleDebugDetected();
                    return undefined;
                }

                callCount++;
                var now = Date.now();

                if (options.rateLimit) {
                    var timeDiff = now - lastCallTime;
                    if (timeDiff < options.rateLimit) {
                        throw new CryptoError('Rate limit exceeded', 'RATE_LIMIT');
                    }
                }

                lastCallTime = now;

                if (addIntegrityCheck && originalHash) {
                    var currentCode = protectedFunc.toString();
                    if (!verifyIntegrityHash(currentCode)) {
                        handleIntegrityViolation();
                        return undefined;
                    }
                }

                try {
                    var args = Array.prototype.slice.call(arguments);
                    if (wrapCalls) {
                        return executeWithProtection(function() {
                            return originalFunc.apply(this, args);
                        });
                    }
                    return originalFunc.apply(this, arguments);
                } catch (error) {
                    if (options.errorHandler) {
                        options.errorHandler(error);
                    }
                    throw error;
                }
            };

            protectedFunc._isProtected = true;
            protectedFunc._originalFunction = originalFunc;
            protectedFunc._callCount = function() { return callCount; };
            protectedFunc._resetCount = function() { callCount = 0; };

            Object.defineProperty(protectedFunc, 'name', {
                value: functionName,
                configurable: true
            });

            return protectedFunc;
        }

        function executeWithProtection(fn) {
            try {
                var result = fn();
                if (result && typeof result.then === 'function') {
                    return result.catch(function(error) {
                        console.error('Protected async function error:', error);
                        throw error;
                    });
                }
                return result;
            } catch (error) {
                console.error('Protected function error:', error);
                throw error;
            }
        }

        function detectDebugging() {
            if (!debugDetectionEnabled) return false;

            var threshold = 160;
            if (window.outerWidth - window.innerWidth > threshold ||
                window.outerHeight - window.innerHeight > threshold) {
                return true;
            }

            var start = performance.now();
            debugger;
            var end = performance.now();
            if (end - start > 100) {
                return true;
            }

            try {
                if (Function.prototype.toString.toString().indexOf('[native code]') === -1) {
                    return true;
                }
            } catch (e) {}

            return false;
        }

        function handleDebugDetected() {
            protectionActive = true;

            if (document.documentElement) {
                document.documentElement.style.display = 'none';
            }

            if (document.body) {
                document.body.innerHTML = '<div style="padding:50px;text-align:center;font-family:sans-serif;">' +
                    '<h1>访问受限</h1>' +
                    '<p>检测到异常访问行为</p>' +
                    '</div>';
            }

            if (typeof onDebugDetected === 'function') {
                onDebugDetected();
            }
        }

        function handleIntegrityViolation() {
            protectionActive = true;

            if (document.documentElement) {
                document.documentElement.style.display = 'none';
            }

            if (document.body) {
                document.body.innerHTML = '<div style="padding:50px;text-align:center;font-family:sans-serif;">' +
                    '<h1>安全警告</h1>' +
                    '<p>代码完整性验证失败</p>' +
                    '</div>';
            }

            if (integrityCheckInterval) {
                clearInterval(integrityCheckInterval);
                integrityCheckInterval = null;
            }
        }

        function computeIntegrityHash(code) {
            var encoder = new TextEncoder();
            var dataBuffer = encoder.encode(code);
            var hashBuffer = null;

            crypto.subtle.digest('SHA-256', dataBuffer).then(function(hash) {
                hashBuffer = arrayBufferToBase64(hash);
                originalHash = hashBuffer;
            });

            return originalHash;
        }

        function verifyIntegrityHash(code) {
            if (!originalHash) return true;

            var currentHash = computeIntegrityHash(code);
            return originalHash === currentHash;
        }

        function startIntegrityMonitoring(interval) {
            if (integrityCheckInterval) {
                clearInterval(integrityCheckInterval);
            }

            var checkInterval = interval || 5000;

            integrityCheckInterval = setInterval(function() {
                if (debugDetectionEnabled && detectDebugging()) {
                    handleDebugDetected();
                }
            }, checkInterval);
        }

        function stopIntegrityMonitoring() {
            if (integrityCheckInterval) {
                clearInterval(integrityCheckInterval);
                integrityCheckInterval = null;
            }
        }

        function initializeProtection(options) {
            options = options || {};

            if (options.enableDebugDetection !== false) {
                debugDetectionEnabled = true;
            }

            if (options.initialIntegrityCheck !== false) {
                var code = document.currentScript ? document.currentScript.innerHTML : '';
                computeIntegrityHash(code);
            }

            if (options.monitorInterval) {
                startIntegrityMonitoring(options.monitorInterval);
            }

            if (options.addSecurityHeaders) {
                addSecurityHeaders();
            }

            if (options.disableEval) {
                disableEval();
            }

            if (options.preventRightClick) {
                preventRightClick();
            }

            return {
                enabled: true,
                debugDetection: debugDetectionEnabled,
                timestamp: Date.now()
            };
        }

        function addSecurityHeaders() {
            var metaTags = [
                { name: 'X-Content-Type-Options', content: 'nosniff' },
                { name: 'X-Frame-Options', content: 'DENY' },
                { name: 'X-XSS-Protection', content: '1; mode=block' }
            ];

            metaTags.forEach(function(meta) {
                var existing = document.querySelector('meta[name="' + meta.name + '"]');
                if (!existing) {
                    var newMeta = document.createElement('meta');
                    newMeta.name = meta.name;
                    newMeta.content = meta.content;
                    document.head.appendChild(newMeta);
                }
            });
        }

        function disableEval() {
            try {
                var originalEval = window.eval;
                window.eval = function(code) {
                    if (code && code.indexOf('debugger') !== -1) {
                        throw new CryptoError('Eval with debugger detected', 'EVAL_BLOCKED');
                    }
                    return originalEval.apply(window, arguments);
                };
            } catch (e) {
                console.warn('Could not disable eval:', e);
            }
        }

        function preventRightClick() {
            document.addEventListener('contextmenu', function(e) {
                e.preventDefault();
                return false;
            });
        }

        function secureStorage(key) {
            var storageKey = storagePrefix + key;
            var sessionData = {};

            return {
                set: async function(value, options) {
                    options = options || {};
                    var dataToStore;

                    if (options.encrypt !== false) {
                        var encrypted = await encryptString(JSON.stringify(value));
                        dataToStore = encrypted;
                    } else {
                        dataToStore = JSON.stringify(value);
                    }

                    if (options.sessionOnly) {
                        sessionData[key] = dataToStore;
                    } else {
                        try {
                            localStorage.setItem(storageKey, dataToStore);
                        } catch (e) {
                            console.error('Storage error:', e);
                        }
                    }

                    return true;
                },

                get: async function(options) {
                    options = options || {};
                    var storedData;

                    if (options.sessionOnly) {
                        storedData = sessionData[key];
                    } else {
                        try {
                            storedData = localStorage.getItem(storageKey);
                        } catch (e) {
                            console.error('Storage error:', e);
                        }
                    }

                    if (!storedData) {
                        return null;
                    }

                    if (options.decrypt !== false) {
                        try {
                            var decrypted = await decryptString(storedData);
                            return JSON.parse(decrypted);
                        } catch (e) {
                            try {
                                return JSON.parse(storedData);
                            } catch (e2) {
                                return storedData;
                            }
                        }
                    }

                    try {
                        return JSON.parse(storedData);
                    } catch (e) {
                        return storedData;
                    }
                },

                remove: function(options) {
                    options = options || {};

                    if (options.sessionOnly) {
                        delete sessionData[key];
                    } else {
                        try {
                            localStorage.removeItem(storageKey);
                        } catch (e) {
                            console.error('Storage error:', e);
                        }
                    }

                    return true;
                },

                exists: function(options) {
                    options = options || {};

                    if (options.sessionOnly) {
                        return key in sessionData;
                    }

                    try {
                        return localStorage.getItem(storageKey) !== null;
                    } catch (e) {
                        return false;
                    }
                }
            };
        }

        function createSecureChannel(targetWindow, options) {
            options = options || {};
            var channelId = generateRandomString(16);
            var messageQueue = [];
            var handlers = {};
            var isOpen = false;

            function postSecureMessage(message, transfer) {
                var wrappedMessage = {
                    _channelId: channelId,
                    _timestamp: Date.now(),
                    _data: message
                };

                var encryptedPayload = null;

                if (options.encrypt) {
                    return encryptString(JSON.stringify(wrappedMessage)).then(function(encrypted) {
                        return {
                            _encrypted: true,
                            _payload: encrypted
                        };
                    });
                }

                return Promise.resolve(wrappedMessage);
            }

            function handleMessage(event) {
                var data = event.data;

                if (options.validateOrigin && event.origin !== options.validateOrigin) {
                    return;
                }

                if (data._encrypted) {
                    decryptString(data._payload).then(function(decrypted) {
                        processMessage(JSON.parse(decrypted));
                    });
                } else if (data._channelId === channelId) {
                    processMessage(data);
                }
            }

            function processMessage(data) {
                if (data._type && handlers[data._type]) {
                    handlers[data._type](data._data);
                }

                if (data._requestId && handlers.response) {
                    handlers.response(data);
                }
            }

            function open() {
                if (typeof targetWindow !== 'undefined' && targetWindow.addEventListener) {
                    targetWindow.addEventListener('message', handleMessage);
                    isOpen = true;
                }
                return this;
            }

            function close() {
                if (typeof targetWindow !== 'undefined' && targetWindow.removeEventListener) {
                    targetWindow.removeEventListener('message', handleMessage);
                }
                isOpen = false;
                return this;
            }

            function send(type, data) {
                var requestId = generateRandomString(8);
                return postSecureMessage({
                    _type: type,
                    _data: data,
                    _requestId: requestId
                }).then(function(message) {
                    if (typeof targetWindow !== 'undefined' && targetWindow.postMessage) {
                        targetWindow.postMessage(message, options.targetOrigin || '*');
                    }
                    return requestId;
                });
            }

            function on(type, handler) {
                handlers[type] = handler;
                return this;
            }

            function off(type) {
                delete handlers[type];
                return this;
            }

            return {
                open: open,
                close: close,
                send: send,
                on: on,
                off: off,
                isOpen: function() { return isOpen; }
            };
        }

        function generateHMAC(data, key) {
            var encoder = new TextEncoder();
            var dataBuffer = encoder.encode(data);

            return crypto.subtle.importKey(
                'raw',
                encoder.encode(key),
                { name: 'HMAC', hash: 'SHA-256' },
                false,
                ['sign']
            ).then(function(importedKey) {
                return crypto.subtle.sign('HMAC', importedKey, dataBuffer);
            }).then(function(signature) {
                return arrayBufferToBase64(signature);
            });
        }

        async function generateHMACHex(data, key, algorithm) {
            var encoder = new TextEncoder();
            var dataBuffer = encoder.encode(data);
            var hashName = algorithm === 'HMAC-SHA512' ? 'SHA-512' : 'SHA-256';

            var importedKey = await crypto.subtle.importKey(
                'raw',
                encoder.encode(key),
                { name: 'HMAC', hash: hashName },
                false,
                ['sign']
            );

            var signature = await crypto.subtle.sign('HMAC', importedKey, dataBuffer);
            return arrayBufferToHex(signature);
        }

        function arrayBufferToHex(buffer) {
            var bytes = new Uint8Array(buffer);
            var hex = '';
            for (var i = 0; i < bytes.byteLength; i++) {
                var hexByte = bytes[i].toString(16);
                if (hexByte.length === 1) {
                    hexByte = '0' + hexByte;
                }
                hex += hexByte;
            }
            return hex;
        }

        async function computeSignature(data, key, algorithm) {
            switch (algorithm) {
                case SignatureAlgorithm.HMAC_SHA256:
                    return generateHMACHex(data, key, 'HMAC-SHA256');
                case SignatureAlgorithm.HMAC_SHA512:
                    return generateHMACHex(data, key, 'HMAC-SHA512');
                default:
                    return generateHMACHex(data, key, 'HMAC-SHA256');
            }
        }

        function buildStringToSign(method, path, query, timestamp, nonce, bodyHash, additionalData) {
            var parts = [];
            parts.push(method.toUpperCase());
            parts.push(path);

            if (query && query !== '') {
                var sortedQuery = sortQueryString(query);
                parts.push(sortedQuery);
            }

            parts.push(timestamp.toString());

            if (nonce && nonce !== '') {
                parts.push(nonce);
            }

            if (bodyHash && bodyHash !== '') {
                parts.push(bodyHash);
            }

            if (additionalData && additionalData.length > 0) {
                for (var i = 0; i < additionalData.length; i++) {
                    if (additionalData[i] && additionalData[i] !== '') {
                        parts.push(additionalData[i]);
                    }
                }
            }

            return parts.join('\n');
        }

        function sortQueryString(query) {
            if (!query || query === '') {
                return '';
            }

            var params = query.split('&');
            var paramMap = {};

            for (var i = 0; i < params.length; i++) {
                var pair = params[i].split('=');
                var key = pair[0];
                var value = pair.length > 1 ? pair[1] : '';
                if (!paramMap[key]) {
                    paramMap[key] = [];
                }
                paramMap[key].push(value);
            }

            var keys = Object.keys(paramMap).sort();
            var resultParts = [];

            for (var j = 0; j < keys.length; j++) {
                var key = keys[j];
                var values = paramMap[key];
                for (var k = 0; k < values.length; k++) {
                    resultParts.push(key + '=' + values[k]);
                }
            }

            return resultParts.join('&');
        }

        async function hashBody(body) {
            if (!body || body.length === 0) {
                return '';
            }

            var encoder = new TextEncoder();
            var dataBuffer = encoder.encode(body);
            var hashBuffer = await crypto.subtle.digest('SHA-256', dataBuffer);
            return arrayBufferToHex(hashBuffer);
        }

        async function generateRequestSignature(method, path, query, timestamp, nonce, body, key, algorithm, additionalData) {
            var bodyHash = await hashBody(body);
            var stringToSign = buildStringToSign(method, path, query, timestamp, nonce, bodyHash, additionalData);
            return computeSignature(stringToSign, key, algorithm);
        }

        async function verifyRequestSignature(method, path, query, timestamp, nonce, body, signature, key, algorithm, additionalData) {
            var expectedSignature = await generateRequestSignature(method, path, query, timestamp, nonce, body, key, algorithm, additionalData);
            return signature === expectedSignature;
        }

        async function verifyHMAC(data, signature, key) {
            var expectedSignature = await generateHMAC(data, key);
            return signature === expectedSignature;
        }

        return {
            version: version,
            encrypt: encryptAES,
            decrypt: decryptAES,
            encryptString: encryptString,
            decryptString: decryptString,
            hash: hashSHA256,
            generateRandomBytes: generateRandomBytes,
            generateRandomString: generateRandomString,
            generateHMAC: generateHMAC,
            generateHMACHex: generateHMACHex,
            computeSignature: computeSignature,
            generateRequestSignature: generateRequestSignature,
            verifyRequestSignature: verifyRequestSignature,
            verifyHMAC: verifyHMAC,
            protectFunction: protectFunction,
            detectDebugging: detectDebugging,
            initializeProtection: initializeProtection,
            startIntegrityMonitoring: startIntegrityMonitoring,
            stopIntegrityMonitoring: stopIntegrityMonitoring,
            secureStorage: secureStorage,
            createSecureChannel: createSecureChannel,
            CryptoError: CryptoError,
            SignatureAlgorithm: SignatureAlgorithm,
            AntiDebugManager: AntiDebugManager,
            MemoryGuard: MemoryGuard,
            IntegrityChecker: IntegrityChecker,
            setSignatureAlgorithm: function(algorithm) {
                if (SignatureAlgorithm[algorithm] || Object.values(SignatureAlgorithm).indexOf(algorithm) !== -1) {
                    currentSignatureAlgorithm = SignatureAlgorithm[algorithm] || algorithm;
                    return true;
                }
                return false;
            },
            getSignatureAlgorithm: function() {
                return currentSignatureAlgorithm;
            },
            buildStringToSign: buildStringToSign,
            sortQueryString: sortQueryString,
            hashBody: hashBody,
            _originalHash: function() { return originalHash; },
            _setDebugDetection: function(enabled) { debugDetectionEnabled = enabled; }
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = CryptoUtils;
    } else {
        globalContext.CryptoUtils = CryptoUtils;
    }

})(typeof window !== 'undefined' ? window : this);
