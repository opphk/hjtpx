(function(globalContext) {
    'use strict';

    var CryptoUtils = (function() {
        var version = '2.0.0';
        var defaultKey = 'hjtpx-obfuscate-key-2024';
        var storagePrefix = '_cry_';
        var debugDetectionEnabled = true;
        var integrityCheckInterval = null;
        var originalHash = null;
        var protectionActive = false;

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

        function hashSHA256Sync(data) {
            var encoder = new TextEncoder();
            var dataBuffer = encoder.encode(data);
            var hashBuffer = null;

            var originalDigest = crypto.subtle.digest;
            crypto.subtle.digest = function() {
                hashBuffer = originalDigest.apply(this, arguments);
                return hashBuffer;
            };

            crypto.subtle.digest('SHA-256', dataBuffer);

            crypto.subtle.digest = originalDigest;

            return hashBuffer ? arrayBufferToBase64(hashBuffer) : null;
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
                var version = data[0];
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

            if (typeof console.clear !== 'undefined') {
                var originalClear = console.clear;
                console.clear = function() {
                    return;
                };
                console.clear = originalClear;
            }

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

        async function generateBlake2b(data, key, bits) {
            var encoder = new TextEncoder();
            var dataBuffer = encoder.encode(data);

            var keyBuffer = encoder.encode(key);

            var hashBuffer = await crypto.subtle.digest(
                { name: 'SHA-256' },
                dataBuffer
            );

            var combined = new Uint8Array(keyBuffer.length + dataBuffer.length);
            combined.set(keyBuffer);
            combined.set(new Uint8Array(hashBuffer), keyBuffer.length);

            var finalHash = await crypto.subtle.digest(
                { name: 'SHA-512' },
                combined
            );

            if (bits === 256) {
                return arrayBufferToHex(finalHash.slice(0, 32));
            } else {
                return arrayBufferToHex(finalHash);
            }
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
                case SignatureAlgorithm.BLAKE2B_256:
                    return generateBlake2b(data, key, 256);
                case SignatureAlgorithm.BLAKE2B_512:
                    return generateBlake2b(data, key, 512);
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

        function signData(data, privateKey) {
            var encoder = new TextEncoder();
            var dataBuffer = encoder.encode(data);

            return crypto.subtle.importKey(
                'pkcs8',
                base64ToArrayBuffer(privateKey),
                { name: 'RSASSA-PKCS1-v1_5', hash: 'SHA-256' },
                false,
                ['sign']
            ).then(function(key) {
                return crypto.subtle.sign(
                    { name: 'RSASSA-PKCS1-v1_5' },
                    key,
                    dataBuffer
                );
            }).then(function(signature) {
                return arrayBufferToBase64(signature);
            });
        }

        function verifySignature(data, signature, publicKey) {
            var encoder = new TextEncoder();
            var dataBuffer = encoder.encode(data);

            return crypto.subtle.importKey(
                'spki',
                base64ToArrayBuffer(publicKey),
                { name: 'RSASSA-PKCS1-v1_5', hash: 'SHA-256' },
                false,
                ['verify']
            ).then(function(key) {
                return crypto.subtle.verify(
                    { name: 'RSASSA-PKCS1-v1_5' },
                    key,
                    base64ToArrayBuffer(signature),
                    dataBuffer
                );
            });
        }

        function generateKeyPair() {
            return crypto.subtle.generateKey(
                {
                    name: 'RSASSA-PKCS1-v1_5',
                    modulusLength: 2048,
                    publicExponent: new Uint8Array([1, 0, 1]),
                    hash: 'SHA-256'
                },
                true,
                ['sign', 'verify']
            );
        }

        function exportPublicKey(keyPair) {
            return crypto.subtle.exportKey('spki', keyPair.publicKey);
        }

        function exportPrivateKey(keyPair) {
            return crypto.subtle.exportKey('pkcs8', keyPair.privateKey);
        }

        function createCodeVerifier() {
            var verifier = generateRandomString(64);
            var hash = null;

            hashSHA256(verifier).then(function(h) {
                hash = h;
            });

            return {
                verifier: verifier,
                getChallenge: function() {
                    return hash || hashSHA256(verifier);
                }
            };
        }

        function generateCodeChallenge(verifier) {
            return hashSHA256(verifier);
        }

        function verifyCodeChallenge(verifier, challenge) {
            return hashSHA256(verifier).then(function(hash) {
                return hash === challenge;
            });
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
            generateBlake2b: generateBlake2b,
            computeSignature: computeSignature,
            generateRequestSignature: generateRequestSignature,
            verifyRequestSignature: verifyRequestSignature,
            verifyHMAC: verifyHMAC,
            signData: signData,
            verifySignature: verifySignature,
            generateKeyPair: generateKeyPair,
            exportPublicKey: exportPublicKey,
            exportPrivateKey: exportPrivateKey,
            createCodeVerifier: createCodeVerifier,
            generateCodeChallenge: generateCodeChallenge,
            verifyCodeChallenge: verifyCodeChallenge,
            protectFunction: protectFunction,
            detectDebugging: detectDebugging,
            initializeProtection: initializeProtection,
            startIntegrityMonitoring: startIntegrityMonitoring,
            stopIntegrityMonitoring: stopIntegrityMonitoring,
            secureStorage: secureStorage,
            createSecureChannel: createSecureChannel,
            CryptoError: CryptoError,
            SignatureAlgorithm: SignatureAlgorithm,
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

(function() {
    'use strict';

    var RuntimeProtection = (function() {
        var originalCode = null;
        var integrityHash = null;
        var protectionActive = false;
        var monitorInterval = null;
        var memorySnapshots = {};
        var selfDestructTriggered = false;

        function computeSHA256(data) {
            var encoder = new TextEncoder();
            var dataBuffer = encoder.encode(data);
            return crypto.subtle.digest('SHA-256', dataBuffer).then(function(hash) {
                var bytes = new Uint8Array(hash);
                var binary = '';
                for (var i = 0; i < bytes.byteLength; i++) {
                    binary += String.fromCharCode(bytes[i]);
                }
                return btoa(binary);
            });
        }

        function takeMemorySnapshot() {
            var snapshot = {
                timestamp: Date.now(),
                functions: [],
                objects: {}
            };

            if (window.Function) {
                var originalToString = Function.prototype.toString;
                var functionKeys = Object.keys(window.Function.prototype);
                functionKeys.forEach(function(key) {
                    try {
                        var desc = Object.getOwnPropertyDescriptor(window.Function.prototype, key);
                        if (desc && desc.value) {
                            snapshot.functions.push({
                                key: key,
                                type: typeof desc.value
                            });
                        }
                    } catch (e) {}
                });
            }

            memorySnapshots[snapshot.timestamp] = snapshot;
            
            var maxSnapshots = 5;
            var timestamps = Object.keys(memorySnapshots).sort();
            if (timestamps.length > maxSnapshots) {
                var oldTimestamps = timestamps.slice(0, timestamps.length - maxSnapshots);
                oldTimestamps.forEach(function(ts) {
                    delete memorySnapshots[ts];
                });
            }

            return snapshot;
        }

        function detectMemoryModification() {
            if (window.Function && Function.prototype.toString) {
                var originalToString = Function.prototype.toString.toString();
                if (originalToString.indexOf('[native code]') === -1) {
                    return true;
                }
            }

            if (window.console && console.log) {
                var originalLog = console.log.toString();
                if (originalLog.indexOf('[native code]') === -1) {
                    return true;
                }
            }

            try {
                var testFunc = function() { return 'test'; };
                var funcString = testFunc.toString();
                if (funcString.indexOf('test') === -1) {
                    return true;
                }
            } catch (e) {
                return true;
            }

            return false;
        }

        function detectCodeTampering() {
            if (!integrityHash) {
                return false;
            }

            var scripts = document.getElementsByTagName('script');
            for (var i = 0; i < scripts.length; i++) {
                var script = scripts[i];
                if (script.src && script.src.indexOf('crypto-utils') !== -1) {
                    var scriptContent = script.innerHTML || script.textContent;
                    if (scriptContent && scriptContent.length > 0) {
                        return computeSHA256(scriptContent).then(function(hash) {
                            return hash !== integrityHash;
                        });
                    }
                }
            }

            return Promise.resolve(false);
        }

        function triggerSelfDestruct() {
            if (selfDestructTriggered) {
                return;
            }
            selfDestructTriggered = true;

            if (monitorInterval) {
                clearInterval(monitorInterval);
                monitorInterval = null;
            }

            if (document.documentElement) {
                document.documentElement.style.display = 'none';
            }

            if (document.body) {
                document.body.innerHTML = '<div style="padding:50px;text-align:center;font-family:sans-serif;background:#000;color:#fff;min-height:100vh;display:flex;flex-direction:column;justify-content:center;align-items:center;">' +
                    '<h1 style="color:#ff0000;">安全警告</h1>' +
                    '<p>检测到代码篡改或异常访问行为</p>' +
                    '<p>系统已自动保护</p>' +
                    '</div>';
            }

            setTimeout(function() {
                var scripts = document.getElementsByTagName('script');
                for (var i = scripts.length - 1; i >= 0; i--) {
                    scripts[i].parentNode.removeChild(scripts[i]);
                }

                if (document.head) {
                    var metas = document.head.getElementsByTagName('meta');
                    for (var j = metas.length - 1; j >= 0; j--) {
                        metas[j].parentNode.removeChild(metas[j]);
                    }
                }

                Object.keys(window).forEach(function(key) {
                    if (key !== 'window' && key !== 'document' && key !== 'location' && key !== 'navigator') {
                        try {
                            if (typeof window[key] === 'function') {
                                (function(k) {
                                    try {
                                        delete window[k];
                                        window[k] = function() {};
                                    } catch (e) {}
                                })(key);
                            }
                        } catch (e) {}
                    }
                });
            }, 100);

            throw new Error('Security violation detected');
        }

        function detectDevTools() {
            var threshold = 160;
            if (window.outerWidth - window.innerWidth > threshold ||
                window.outerHeight - window.innerHeight > threshold) {
                return true;
            }

            var startTime = Date.now();
            debugger;
            var endTime = Date.now();
            if (endTime - startTime > 100) {
                return true;
            }

            var enabled = false;
            (function(x) {
                var d = document.createElement('div');
                d.innerHTML = '<x id="__y"/>';
                d.style.visibility = 'hidden';
                document.head.appendChild(d);
                Object.defineProperty(x, 'inspect', {
                    get: function() {
                        enabled = true;
                        return function() {};
                    }
                });
            })(window);

            if (window.__y && window.__y.id === '__y') {
                enabled = true;
            }

            if (enabled) {
                return true;
            }

            return false;
        }

        function startProtection(options) {
            options = options || {};
            options.checkInterval = options.checkInterval || 2000;
            options.enableSelfDestruct = options.enableSelfDestruct !== false;
            options.enableMemoryProtection = options.enableMemoryProtection !== false;

            var scriptContent = '';
            var scripts = document.getElementsByTagName('script');
            for (var i = 0; i < scripts.length; i++) {
                if (scripts[i].src && scripts[i].src.indexOf('crypto-utils') !== -1) {
                    scriptContent = scripts[i].innerHTML || scripts[i].textContent;
                    break;
                }
            }

            if (scriptContent) {
                computeSHA256(scriptContent).then(function(hash) {
                    integrityHash = hash;
                });
            }

            takeMemorySnapshot();

            monitorInterval = setInterval(function() {
                if (detectMemoryModification()) {
                    console.error('Memory modification detected');
                    if (options.enableSelfDestruct) {
                        triggerSelfDestruct();
                    }
                }

                if (detectDevTools()) {
                    console.error('Developer tools detected');
                    if (options.enableSelfDestruct) {
                        triggerSelfDestruct();
                    }
                }

                if (options.enableMemoryProtection) {
                    takeMemorySnapshot();
                }

                detectCodeTampering().then(function(tampered) {
                    if (tampered) {
                        console.error('Code tampering detected');
                        if (options.enableSelfDestruct) {
                            triggerSelfDestruct();
                        }
                    }
                });
            }, options.checkInterval);

            document.addEventListener('keydown', function(e) {
                if (e.keyCode === 123) {
                    e.preventDefault();
                    if (options.enableSelfDestruct) {
                        triggerSelfDestruct();
                    }
                }
            });

            document.addEventListener('contextmenu', function(e) {
                if (options.preventRightClick) {
                    e.preventDefault();
                }
            });

            protectionActive = true;
        }

        function stopProtection() {
            if (monitorInterval) {
                clearInterval(monitorInterval);
                monitorInterval = null;
            }
            protectionActive = false;
        }

        function getProtectionStatus() {
            return {
                active: protectionActive,
                selfDestructTriggered: selfDestructTriggered,
                hasIntegrityHash: integrityHash !== null,
                snapshotCount: Object.keys(memorySnapshots).length,
                monitorRunning: monitorInterval !== null
            };
        }

        function initializeRuntimeProtection(options) {
            if (document.readyState === 'loading') {
                document.addEventListener('DOMContentLoaded', function() {
                    startProtection(options);
                });
            } else {
                startProtection(options);
            }

            return {
                start: startProtection,
                stop: stopProtection,
                status: getProtectionStatus,
                snapshot: takeMemorySnapshot,
                selfDestruct: triggerSelfDestruct
            };
        }

        return initializeRuntimeProtection;
    })();

    if (typeof window !== 'undefined') {
        window.RuntimeProtection = RuntimeProtection;
    }

    if (typeof CryptoUtils !== 'undefined') {
        CryptoUtils.RuntimeProtection = RuntimeProtection;
    }

    if (typeof CryptoUtils !== 'undefined') {
        CryptoUtils.initializeRuntimeProtection = function(options) {
            return RuntimeProtection(options);
        };

        CryptoUtils.verifyRuntimeIntegrity = function() {
            return detectCodeTampering();
        };

        CryptoUtils.protectMemory = function() {
            return takeMemorySnapshot();
        };

        CryptoUtils.emergencyShutdown = function() {
            triggerSelfDestruct();
        };
    }

    function detectCodeTampering() {
        if (!integrityHash) {
            return Promise.resolve(false);
        }

        var scripts = document.getElementsByTagName('script');
        for (var i = 0; i < scripts.length; i++) {
            var script = scripts[i];
            if (script.src && script.src.indexOf('crypto-utils') !== -1) {
                var scriptContent = script.innerHTML || script.textContent;
                if (scriptContent && scriptContent.length > 0) {
                    return computeSHA256(scriptContent).then(function(hash) {
                        return hash !== integrityHash;
                    });
                }
            }
        }

        return Promise.resolve(false);
    }

    function computeSHA256(data) {
        var encoder = new TextEncoder();
        var dataBuffer = encoder.encode(data);
        return crypto.subtle.digest('SHA-256', dataBuffer).then(function(hash) {
            var bytes = new Uint8Array(hash);
            var binary = '';
            for (var i = 0; i < bytes.byteLength; i++) {
                binary += String.fromCharCode(bytes[i]);
            }
            return btoa(binary);
        });
    }

    function rc4Encrypt(plaintext, key) {
        var s = new Array(256);
        for (var i = 0; i < 256; i++) {
            s[i] = i;
        }

        var j = 0;
        for (var i = 0; i < 256; i++) {
            j = (j + s[i] + key.charCodeAt(i % key.length)) % 256;
            var temp = s[i];
            s[i] = s[j];
            s[j] = temp;
        }

        var i = 0;
        j = 0;
        var result = new Array(plaintext.length);

        for (var k = 0; k < plaintext.length; k++) {
            i = (i + 1) % 256;
            j = (j + s[i]) % 256;
            var temp = s[i];
            s[i] = s[j];
            s[j] = temp;
            result[k] = String.fromCharCode(plaintext.charCodeAt(k) ^ s[(s[i] + s[j]) % 256]);
        }

        return result.join('');
    }

    function rc4Decrypt(ciphertext, key) {
        return rc4Encrypt(ciphertext, key);
    }

    function rc4EncryptBase64(plaintext, key) {
        var encrypted = rc4Encrypt(plaintext, key);
        return btoa(encrypted);
    }

    function rc4DecryptBase64(ciphertextBase64, key) {
        var ciphertext = atob(ciphertextBase64);
        return rc4Decrypt(ciphertext, key);
    }

    function detectHeadlessBrowser() {
        var userAgent = navigator.userAgent.toLowerCase();

        var headlessIndicators = [
            'headlesschrome',
            'phantomjs',
            'selenium',
            'puppeteer',
            'nightmare',
            'slimerjs',
            'ghost',
            'zombie'
        ];

        for (var i = 0; i < headlessIndicators.length; i++) {
            if (userAgent.indexOf(headlessIndicators[i]) !== -1) {
                return true;
            }
        }

        if (window.callPhantom || window._phantom) {
            return true;
        }

        if (navigator.webdriver === true) {
            return true;
        }

        try {
            if (window.Buffer) {
                return true;
            }
        } catch (e) {}

        if (!window.chrome) {
            try {
                var canvas = document.createElement('canvas');
                var gl = canvas.getContext('webgl');
                if (gl && gl.getExtension('WEBGL_debug_renderer_info')) {
                    var debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                    var renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                    if (renderer.indexOf('SwiftShader') !== -1 || renderer.indexOf('llvmpipe') !== -1) {
                        return true;
                    }
                }
            } catch (e) {}
        }

        return false;
    }

    function detectAutomation() {
        if (detectHeadlessBrowser()) {
            return true;
        }

        if (document.documentElement.getAttribute('webdriver')) {
            return true;
        }

        if (navigator.permissions && navigator.permissions.query) {
            return navigator.permissions.query({name: 'notifications'}).then(function(result) {
                return result.state === 'prompt';
            }).catch(function() {
                return false;
            });
        }

        try {
            var testFn = function() {};
            var testStr = testFn.toString();
            if (testStr.indexOf('[native code]') === -1 && testStr.indexOf('testFn') === -1) {
                return true;
            }
        } catch (e) {}

        return false;
    }

    function enhancedSelfDestruct(reason) {
        protectionActive = true;

        var scripts = document.getElementsByTagName('script');
        for (var i = scripts.length - 1; i >= 0; i--) {
            try {
                scripts[i].parentNode.removeChild(scripts[i]);
            } catch (e) {}
        }

        if (document.head) {
            var metas = document.head.getElementsByTagName('meta');
            for (var j = metas.length - 1; j >= 0; j--) {
                try {
                    metas[j].parentNode.removeChild(metas[j]);
                } catch (e) {}
            }
        }

        if (document.documentElement) {
            document.documentElement.style.display = 'none';
        }

        if (document.body) {
            document.body.innerHTML = '<div style="position:fixed;top:0;left:0;right:0;bottom:0;display:flex;flex-direction:column;justify-content:center;align-items:center;background:#000;color:#fff;font-family:sans-serif;z-index:9999999;">' +
                '<h1 style="color:#ff0000;font-size:48px;margin-bottom:20px;">安全警告</h1>' +
                '<p style="font-size:24px;margin-bottom:10px;">检测到异常访问行为</p>' +
                '<p style="font-size:16px;color:#ff6666;">' + (reason || '访问已终止') + '</p>' +
                '<p style="font-size:12px;margin-top:30px;">系统已自动保护</p></div>';
        }

        setTimeout(function() {
            try {
                Object.keys(window).forEach(function(key) {
                    if (key !== 'window' && key !== 'document' && key !== 'location' && key !== 'navigator' && key !== 'history' && key !== 'screen') {
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
    }

    function enhancedAntiDebug() {
        if (!debugDetectionEnabled) return false;

        var threshold = 160;

        if (window.outerWidth - window.innerWidth > threshold ||
            window.outerHeight - window.innerHeight > threshold) {
            enhancedSelfDestruct('开发者工具检测 - 窗口大小异常');
            return true;
        }

        var startTime = performance.now();
        debugger;
        var endTime = performance.now();
        if (endTime - startTime > 100) {
            enhancedSelfDestruct('开发者工具检测 - 调试器中断');
            return true;
        }

        (function(x) {
            Object.defineProperty(x, 'inspect', {
                get: function() {
                    enhancedSelfDestruct('开发者工具检测 - inspect API');
                    return function() {};
                }
            });
        })(window);

        if (typeof window.__inspect !== 'undefined') {
            enhancedSelfDestruct('开发者工具检测 - __inspect变量');
            return true;
        }

        if (window.devtools && window.devtools.isOpen) {
            enhancedSelfDestruct('开发者工具检测 - DevTools已打开');
            return true;
        }

        if (window.Firebug && window.Firebug.chrome && window.Firebug.chrome.isVisible) {
            enhancedSelfDestruct('开发者工具检测 - Firebug已打开');
            return true;
        }

        var consoleMethods = ['log', 'error', 'warn', 'info', 'debug'];
        consoleMethods.forEach(function(method) {
            try {
                if (console[method] && console[method].toString().indexOf('[native code]') === -1) {
                    enhancedSelfDestruct('开发者工具检测 - Console方法被重写');
                    return true;
                }
            } catch (e) {}
        });

        return false;
    }

    function startEnhancedProtection(options) {
        options = options || {};
        options.checkInterval = options.checkInterval || 2000;
        options.enableSelfDestruct = options.enableSelfDestruct !== false;
        options.enableAutomationDetection = options.enableAutomationDetection !== false;
        options.enableHeadlessDetection = options.enableHeadlessDetection !== false;

        if (options.enableAutomationDetection || options.enableHeadlessDetection) {
            if (detectAutomation() || detectHeadlessBrowser()) {
                if (options.enableSelfDestruct) {
                    enhancedSelfDestruct('自动化工具检测');
                    return;
                }
            }
        }

        var checkCounter = 0;
        var originalBodyContent = '';

        try {
            if (document.body) {
                originalBodyContent = document.body.innerHTML;
            }
        } catch (e) {}

        var protectionInterval = setInterval(function() {
            checkCounter++;

            if (options.enableSelfDestruct) {
                if (enhancedAntiDebug()) {
                    return;
                }
            }

            if (options.enableAutomationDetection || options.enableHeadlessDetection) {
                if (checkCounter % 10 === 0) {
                    if (detectAutomation() || detectHeadlessBrowser()) {
                        if (options.enableSelfDestruct) {
                            enhancedSelfDestruct('自动化工具检测');
                            return;
                        }
                    }
                }
            }

            if (checkCounter % 5 === 0 && originalBodyContent) {
                try {
                    if (document.body && document.body.innerHTML !== originalBodyContent) {
                        if (options.enableSelfDestruct) {
                            enhancedSelfDestruct('页面内容被修改');
                            return;
                        }
                    }
                } catch (e) {}
            }

            try {
                var testFunc = function() {};
                var funcStr = testFunc.toString();
                if (funcStr.indexOf('[native code]') === -1 && funcStr.indexOf('testFunc') === -1) {
                    if (options.enableSelfDestruct) {
                        enhancedSelfDestruct('函数toString方法被修改');
                        return;
                    }
                }
            } catch (e) {}

        }, options.checkInterval);

        document.addEventListener('keydown', function(e) {
            if (e.keyCode === 123) {
                e.preventDefault();
                if (options.enableSelfDestruct) {
                    enhancedSelfDestruct('F12键被按下');
                }
            }

            if (e.keyCode === 116) {
                e.preventDefault();
                if (options.enableSelfDestruct) {
                    enhancedSelfDestruct('F5键被按下');
                }
            }

            if (e.ctrlKey && e.shiftKey && e.keyCode === 73) {
                e.preventDefault();
                if (options.enableSelfDestruct) {
                    enhancedSelfDestruct('Ctrl+Shift+I被按下');
                }
            }

            if (e.ctrlKey && e.keyCode === 85) {
                e.preventDefault();
                if (options.enableSelfDestruct) {
                    enhancedSelfDestruct('Ctrl+U被按下');
                }
            }
        });

        document.addEventListener('contextmenu', function(e) {
            if (options.preventRightClick) {
                e.preventDefault();
                return false;
            }
        });

        document.addEventListener('selectstart', function(e) {
            if (options.preventSelection) {
                e.preventDefault();
                return false;
            }
        });

        document.addEventListener('dragstart', function(e) {
            if (options.preventDrag) {
                e.preventDefault();
                return false;
            }
        });

        protectionActive = true;

        return {
            selfDestruct: enhancedSelfDestruct,
            detectHeadless: detectHeadlessBrowser,
            detectAutomation: detectAutomation,
            antiDebug: enhancedAntiDebug
        };
    }

    function generateIntegritySignature(code) {
        var encoder = new TextEncoder();
        var data = encoder.encode(code);

        return crypto.subtle.digest('SHA-256', data).then(function(hash) {
            var binary = '';
            var bytes = new Uint8Array(hash);
            for (var i = 0; i < bytes.byteLength; i++) {
                binary += String.fromCharCode(bytes[i]);
            }
            return btoa(binary);
        });
    }

    function verifyIntegritySignature(code, expectedSignature) {
        return generateIntegritySignature(code).then(function(signature) {
            return signature === expectedSignature;
        });
    }

    function createCodeGuard(code, options) {
        options = options || {};

        var signature = null;

        generateIntegritySignature(code).then(function(sig) {
            signature = sig;
        });

        var guardCode = {
            checkInterval: options.checkInterval || 5000,
            onViolation: options.onViolation || enhancedSelfDestruct,

            check: function() {
                if (signature === null) return;

                var scripts = document.getElementsByTagName('script');
                for (var i = 0; i < scripts.length; i++) {
                    var script = scripts[i];
                    if (script.src && script.src.indexOf('crypto-utils') !== -1) {
                        var content = script.innerHTML || script.textContent;
                        if (content) {
                            generateIntegritySignature(content).then(function(sig) {
                                if (sig !== signature) {
                                    this.onViolation('代码签名验证失败');
                                }
                            }.bind(this));
                        }
                    }
                }
            },

            start: function() {
                var self = this;
                this.interval = setInterval(function() {
                    self.check();
                }, this.checkInterval);
            },

            stop: function() {
                if (this.interval) {
                    clearInterval(this.interval);
                }
            }
        };

        return guardCode;
    }

    return {
        version: version,
        encrypt: encryptAES,
        decrypt: decryptAES,
        encryptString: encryptString,
        decryptString: decryptString,
        encryptRC4: rc4EncryptBase64,
        decryptRC4: rc4DecryptBase64,
        hash: hashSHA256,
        generateRandomBytes: generateRandomBytes,
        generateRandomString: generateRandomString,
        generateHMAC: generateHMAC,
        verifyHMAC: verifyHMAC,
        signData: signData,
        verifySignature: verifySignature,
        generateKeyPair: generateKeyPair,
        exportPublicKey: exportPublicKey,
        exportPrivateKey: exportPrivateKey,
        createCodeVerifier: createCodeVerifier,
        generateCodeChallenge: generateCodeChallenge,
        verifyCodeChallenge: verifyCodeChallenge,
        protectFunction: protectFunction,
        detectDebugging: detectDebugging,
        initializeProtection: initializeProtection,
        startIntegrityMonitoring: startIntegrityMonitoring,
        stopIntegrityMonitoring: stopIntegrityMonitoring,
        secureStorage: secureStorage,
        createSecureChannel: createSecureChannel,
        CryptoError: CryptoError,
        _originalHash: function() { return originalHash; },
        _setDebugDetection: function(enabled) { debugDetectionEnabled = enabled; },
        rc4: {
            encrypt: rc4Encrypt,
            decrypt: rc4Decrypt,
            encryptBase64: rc4EncryptBase64,
            decryptBase64: rc4DecryptBase64
        },
        detect: {
            headlessBrowser: detectHeadlessBrowser,
            automation: detectAutomation,
            devTools: enhancedAntiDebug
        },
        protection: {
            selfDestruct: enhancedSelfDestruct,
            start: startEnhancedProtection,
            guard: createCodeGuard,
            generateSignature: generateIntegritySignature,
            verifySignature: verifyIntegritySignature
        }
    };
})();
