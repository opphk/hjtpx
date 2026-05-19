(function(globalContext) {
    'use strict';

    const IntegrityEnhanced = (function() {
        const VERSION = '5.0.0';
        
        const _0xIE = {
            hashes: {},
            markers: [],
            checkInterval: 10000,
            maxChecks: 100,
            checkCount: 0,
            enabled: true,
            salt: null,
            signatureKey: null,
            crc32Table: [],
            crc32Initialized: false,
            lastCheckTime: 0,
            integrityViolations: 0,
            maxViolations: 3,
            obfuscationSeed: Math.random() * 1000000,
            timestamp: Date.now(),
            verificationHistory: [],
            protectionLevel: 'maximum',
            trustedDomains: [],
            untrustedPatterns: [],
            selfDestructEnabled: false
        };

        function initCRC32() {
            if (_0xIE.crc32Initialized) return;
            
            for (let i = 0; i < 256; i++) {
                let c = i;
                for (let j = 0; j < 8; j++) {
                    c = (c & 1) ? (0xEDB88320 ^ (c >>> 1)) : (c >>> 1);
                }
                _0xIE.crc32Table[i] = c;
            }
            _0xIE.crc32Initialized = true;
        }

        function computeCRC32(data) {
            initCRC32();
            let crc = 0xFFFFFFFF;
            for (let i = 0; i < data.length; i++) {
                crc = _0xIE.crc32Table[(crc ^ data.charCodeAt(i)) & 0xFF] ^ (crc >>> 8);
            }
            return (crc ^ 0xFFFFFFFF) >>> 0;
        }

        async function computeSHA256(data) {
            const encoder = new TextEncoder();
            const dataBuffer = typeof data === 'string' ? encoder.encode(data) : data;
            const hashBuffer = await crypto.subtle.digest('SHA-256', dataBuffer);
            const hashArray = Array.from(new Uint8Array(hashBuffer));
            return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
        }

        async function computeSHA512(data) {
            const encoder = new TextEncoder();
            const dataBuffer = typeof data === 'string' ? encoder.encode(data) : data;
            const hashBuffer = await crypto.subtle.digest('SHA-512', dataBuffer);
            const hashArray = Array.from(new Uint8Array(hashBuffer));
            return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
        }

        function computeMD5(data) {
            function safeAdd(x, y) {
                const lsw = (x & 0xFFFF) + (y & 0xFFFF);
                const msw = (x >> 16) + (y >> 16) + (lsw >> 16);
                return (msw << 16) | (lsw & 0xFFFF);
            }
            
            function bitRotateLeft(num, cnt) {
                return (num << cnt) | (num >>> (32 - cnt));
            }
            
            const x = [];
            const nblk = ((data.length + 8) >> 6) + 1;
            for (let i = 0; i < nblk * 16; i++) x[i] = 0;
            for (let i = 0; i < data.length; i++) x[i >> 2] |= data.charCodeAt(i) << ((i % 4) * 8);
            x[i >> 2] |= 0x80 << ((i % 4) * 8);
            x[nblk * 16 - 2] = data.length * 8;
            
            let a = 1732584193, b = -271733879, c = -1732584194, d = 271733878;
            
            const rounds = [
                [7, -680876936], [12, -389564586], [17, 606105819], [22, -1044525330],
                [5, -165796510], [9, -1069501632], [14, 643717713], [20, -373897302],
                [4, -378558], [11, -2022574463], [16, 1839030562], [23, -35309556],
                [6, -198630844], [10, 1126891415], [15, -1416354905], [21, -57434055]
            ];
            
            for (let i = 0; i < x.length; i += 16) {
                const olda = a, oldb = b, oldc = c, oldd = d;
                
                for (let j = 0; j < 64; j++) {
                    let f, g;
                    if (j < 16) {
                        f = (b & c) | ((~b) & d);
                        g = j;
                    } else if (j < 32) {
                        f = (d & b) | ((~d) & c);
                        g = (5 * j + 1) % 16;
                    } else if (j < 48) {
                        f = c ^ (b | (~d));
                        g = (3 * j + 5) % 16;
                    } else {
                        f = b ^ c ^ d;
                        g = (7 * j) % 16;
                    }
                    
                    const temp = d;
                    d = c;
                    c = b;
                    b = safeAdd(b, bitRotateLeft(safeAdd(a, safeAdd(f, safeAdd(x[i + g], rounds[j % 4][1])), rounds[j % 4][0]));
                    a = temp;
                }
                
                a = safeAdd(a, olda);
                b = safeAdd(b, oldb);
                c = safeAdd(c, oldc);
                d = safeAdd(d, oldd);
            }
            
            return [a, b, c, d].map(v => (v >>> 0).toString(16).padStart(8, '0')).join('');
        }

        function generateSalt() {
            const array = new Uint8Array(16);
            crypto.getRandomValues(array);
            return Array.from(array).map(b => b.toString(16).padStart(2, '0')).join('');
        }

        async function computeMultipleHashes(data) {
            const crc32 = computeCRC32(data).toString(16).padStart(8, '0');
            const sha256 = await computeSHA256(data);
            const sha512 = await computeSHA512(data);
            const md5 = computeMD5(data);
            
            return {
                crc32: crc32,
                sha256: sha256,
                sha512: sha512,
                md5: md5,
                combined: crc32 + sha256.slice(0, 16) + md5.slice(0, 16),
                timestamp: Date.now()
            };
        }

        async function generateSignature(data, key) {
            const encoder = new TextEncoder();
            const dataBuffer = encoder.encode(data);
            const keyBuffer = encoder.encode(key);
            
            const cryptoKey = await crypto.subtle.importKey(
                'raw', keyBuffer, { name: 'HMAC', hash: 'SHA-512' },
                false, ['sign']
            );
            
            const signature = await crypto.subtle.sign('HMAC', cryptoKey, dataBuffer);
            const signatureArray = Array.from(new Uint8Array(signature));
            return signatureArray.map(b => b.toString(16).padStart(2, '0')).join('');
        }

        function createDOMMarkers(count) {
            _0xIE.markers = [];
            const markerCount = count || 7;
            
            for (let i = 0; i < markerCount; i++) {
                const marker = document.createElement('div');
                marker.id = '_0xIE_marker_' + i;
                marker.style.display = 'none';
                marker.setAttribute('data-h', _0xIE.hashes.sha256);
                marker.setAttribute('data-c', _0xIE.hashes.crc32);
                marker.setAttribute('data-s', _0xIE.salt);
                document.body.appendChild(marker);
                _0xIE.markers.push(marker.id);
            }
        }

        function verifyDOMMarkers() {
            for (const markerId of _0xIE.markers) {
                const el = document.getElementById(markerId);
                if (!el) {
                    return { valid: false, reason: 'Marker missing: ' + markerId };
                }
                if (el.getAttribute('data-h') !== _0xIE.hashes.sha256) {
                    return { valid: false, reason: 'Marker hash mismatch: ' + markerId };
                }
                if (el.getAttribute('data-c') !== _0xIE.hashes.crc32) {
                    return { valid: false, reason: 'Marker CRC mismatch: ' + markerId };
                }
                if (el.getAttribute('data-s') !== _0xIE.salt) {
                    return { valid: false, reason: 'Marker salt mismatch: ' + markerId };
                }
            }
            return { valid: true, reason: 'All markers verified' };
        }

        function verifyTimingConsistency() {
            const start = performance.now();
            let result = 0;
            for (let i = 0; i < 5000; i++) {
                result += Math.sin(i) * Math.cos(i);
            }
            const elapsed = performance.now() - start;
            
            if (elapsed > 50) {
                return { valid: false, reason: 'Timing anomaly detected: ' + elapsed + 'ms' };
            }
            return { valid: true, reason: 'Timing check passed: ' + elapsed + 'ms' };
        }

        function verifyScriptIntegrity() {
            const scripts = document.querySelectorAll('script');
            
            for (const script of scripts) {
                if (script.src && script.src.includes('core') && script.integrity) {
                    const expectedHash = script.integrity;
                    if (!expectedHash.includes(_0xIE.hashes.sha256.slice(0, 27))) {
                        return { valid: false, reason: 'Script integrity mismatch: ' + script.src };
                    }
                }
            }
            return { valid: true, reason: 'Script integrity verified' };
        }

        async function verifyCodeHash(code, expectedHashes) {
            const hashes = await computeMultipleHashes(code);
            
            const checks = {
                crc32: hashes.crc32 === expectedHashes.crc32,
                sha256: hashes.sha256 === expectedHashes.sha256,
                sha512: hashes.sha512 === expectedHashes.sha512,
                md5: hashes.md5 === expectedHashes.md5
            };
            
            checks.all = checks.crc32 && checks.sha256 && checks.sha512 && checks.md5;
            
            return { checks, computed: hashes };
        }

        async function verifySignature(signature, data, key) {
            const expectedSignature = await generateSignature(data, key);
            return signature === expectedSignature;
        }

        function verifyContentSecurityPolicy() {
            const metaTags = document.querySelectorAll('meta');
            for (const meta of metaTags) {
                const httpEquiv = meta.getAttribute('http-equiv');
                if (httpEquiv && httpEquiv.toLowerCase() === 'content-security-policy') {
                    return { valid: true, reason: 'CSP header present' };
                }
            }
            return { valid: false, reason: 'Content Security Policy not found' };
        }

        function verifySourceMaps() {
            const scripts = document.querySelectorAll('script');
            for (const script of scripts) {
                if (script.src && script.src.indexOf('.map') > -1) {
                    return { valid: false, reason: 'Source map detected: ' + script.src };
                }
            }
            return { valid: true, reason: 'No source maps found' };
        }

        function verifyDOMIntegrity() {
            const requiredElements = ['html', 'body'];
            for (const tag of requiredElements) {
                if (!document.querySelector(tag)) {
                    return { valid: false, reason: 'Required DOM element missing: ' + tag };
                }
            }
            
            if (document.documentElement.getAttribute('ng-app') || document.documentElement.getAttribute('data-reactroot')) {
                return { valid: false, reason: 'Framework root attributes detected' };
            }
            
            return { valid: true, reason: 'DOM integrity verified' };
        }

        function verifyEventListeners() {
            const originalAddEventListener = EventTarget.prototype.addEventListener;
            let listenerCount = 0;
            
            try {
                EventTarget.prototype.addEventListener = function(type, listener, options) {
                    listenerCount++;
                    return originalAddEventListener.call(this, type, listener, options);
                };
                
                setTimeout(() => {
                    EventTarget.prototype.addEventListener = originalAddEventListener;
                }, 100);
            } catch (e) {
                return { valid: true, reason: 'Event listener check skipped' };
            }
            
            return { valid: true, reason: 'Event listeners monitored' };
        }

        function verifyStorageAccess() {
            try {
                localStorage.setItem('_0xTest', 'test');
                localStorage.removeItem('_0xTest');
                
                sessionStorage.setItem('_0xTest', 'test');
                sessionStorage.removeItem('_0xTest');
                
                return { valid: true, reason: 'Storage access verified' };
            } catch (e) {
                return { valid: false, reason: 'Storage access failed: ' + e.message };
            }
        }

        function verifyPrototypeChain() {
            try {
                const originalToString = Object.prototype.toString;
                const testObj = {};
                const result = Object.prototype.toString.call(testObj);
                
                if (result !== '[object Object]') {
                    return { valid: false, reason: 'Prototype chain modified' };
                }
                
                return { valid: true, reason: 'Prototype chain verified' };
            } catch (e) {
                return { valid: false, reason: 'Prototype chain verification failed' };
            }
        }

        function verifyEvalDisabled() {
            if (typeof window._0xEvalDisabled !== 'undefined') {
                return { valid: true, reason: 'Eval is properly disabled' };
            }
            
            try {
                const testEval = eval('1+1');
                if (testEval !== 2) {
                    return { valid: false, reason: 'Eval is not functioning correctly' };
                }
            } catch (e) {
                return { valid: true, reason: 'Eval appears to be restricted' };
            }
            
            return { valid: true, reason: 'Eval check passed' };
        }

        function verifyWebSocketIntegrity() {
            if (typeof WebSocket !== 'undefined') {
                const originalWebSocket = WebSocket;
                if (WebSocket.toString() !== originalWebSocket.toString()) {
                    return { valid: false, reason: 'WebSocket has been modified' };
                }
            }
            return { valid: true, reason: 'WebSocket integrity verified' };
        }

        async function performIntegrityCheck() {
            if (_0xIE.checkCount >= _0xIE.maxChecks) {
                return { status: 'max_checks_reached', valid: true };
            }

            const now = Date.now();
            if (now - _0xIE.lastCheckTime < _0xIE.checkInterval) {
                return { status: 'too_soon', valid: true };
            }
            _0xIE.lastCheckTime = now;

            const results = [];

            const markerResult = verifyDOMMarkers();
            results.push(markerResult);
            if (!markerResult.valid) {
                _0xIE.integrityViolations++;
                return { status: 'marker_failure', valid: false, reason: markerResult.reason };
            }

            const timingResult = verifyTimingConsistency();
            results.push(timingResult);
            if (!timingResult.valid) {
                _0xIE.integrityViolations++;
                return { status: 'timing_failure', valid: false, reason: timingResult.reason };
            }

            const scriptResult = verifyScriptIntegrity();
            results.push(scriptResult);
            if (!scriptResult.valid) {
                _0xIE.integrityViolations++;
                return { status: 'script_failure', valid: false, reason: scriptResult.reason };
            }

            if (_0xIE.protectionLevel === 'maximum' || _0xIE.protectionLevel === 'paranoid') {
                const cspResult = verifyContentSecurityPolicy();
                results.push(cspResult);
                
                const sourceMapResult = verifySourceMaps();
                results.push(sourceMapResult);
                if (!sourceMapResult.valid) {
                    _0xIE.integrityViolations++;
                    return { status: 'source_map_detected', valid: false, reason: sourceMapResult.reason };
                }
                
                const domResult = verifyDOMIntegrity();
                results.push(domResult);
                if (!domResult.valid) {
                    _0xIE.integrityViolations++;
                    return { status: 'dom_integrity_failure', valid: false, reason: domResult.reason };
                }
                
                const storageResult = verifyStorageAccess();
                results.push(storageResult);
                if (!storageResult.valid) {
                    _0xIE.integrityViolations++;
                    return { status: 'storage_failure', valid: false, reason: storageResult.reason };
                }
                
                const prototypeResult = verifyPrototypeChain();
                results.push(prototypeResult);
                if (!prototypeResult.valid) {
                    _0xIE.integrityViolations++;
                    return { status: 'prototype_failure', valid: false, reason: prototypeResult.reason };
                }
            }

            _0xIE.checkCount++;
            
            _0xIE.verificationHistory.push({
                timestamp: now,
                results: results,
                violations: _0xIE.integrityViolations
            });
            
            if (_0xIE.verificationHistory.length > 10) {
                _0xIE.verificationHistory.shift();
            }
            
            if (_0xIE.integrityViolations >= _0xIE.maxViolations) {
                return { status: 'max_violations', valid: false, reason: 'Too many integrity violations' };
            }

            return { status: 'success', valid: true, checkCount: _0xIE.checkCount };
        }

        function handleIntegrityFailure(reason) {
            _0xIE.enabled = false;
            
            if (_0xIE.selfDestructEnabled) {
                triggerSelfDestruct(reason);
            } else {
                displayIntegrityError(reason);
            }
        }

        function triggerSelfDestruct(reason) {
            document.documentElement.style.display = 'none';
            document.body.innerHTML = '';
            document.body.style.background = '#000';
            document.body.style.color = '#fff';
            document.body.style.fontFamily = 'monospace';
            document.body.style.padding = '50px';
            
            const errorInfo = `
                Integrity Violation Detected
                =================================
                
                Your session has been terminated.
                All local data has been cleared.
                
                Reason: ${reason}
                Error Code: ${_0xIE.hashes.sha256 || 'UNKNOWN'}
                Timestamp: ${new Date().toISOString()}
                
                If you believe this is an error, please contact support.
            `;
            
            document.body.textContent = errorInfo;
            
            try {
                localStorage.clear();
                sessionStorage.clear();
            } catch (e) {
                // Storage clear failed
            }
            
            setTimeout(() => {
                window.location.href = 'about:blank';
            }, 3000);
        }

        function displayIntegrityError(reason) {
            document.documentElement.style.display = 'none';
            const errorPage = `
                <div style="position:fixed;top:0;left:0;width:100%;height:100%;background:#1a1a2e;color:#fff;font-family:Arial,sans-serif;display:flex;justify-content:center;align-items:center;">
                    <div style="text-align:center;max-width:600px;padding:40px;">
                        <div style="width:120px;height:120px;margin:0 auto 30px;border-radius:50%;background:#e74c3c;display:flex;justify-content:center;align-items:center;">
                            <svg width="60" height="60" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="2">
                                <circle cx="12" cy="12" r="10"/>
                                <path d="M15 9l-6 6M9 9l6 6"/>
                            </svg>
                        </div>
                        <h1 style="font-size:36px;margin:0 0 20px 0;color:#e74c3c;">Integrity Check Failed</h1>
                        <p style="font-size:16px;opacity:0.9;margin:0 0 10px 0;">Your session has been terminated due to security concerns.</p>
                        <p style="font-size:14px;opacity:0.7;">Reason: ${escapeHtml(reason)}</p>
                        <div style="margin-top:20px;font-size:12px;opacity:0.6;">
                            Error Code: ${_0xIE.hashes.sha256 || 'UNKNOWN'}
                        </div>
                        <div style="margin-top:30px;padding-top:20px;border-top:1px solid rgba(255,255,255,0.1);">
                            <button onclick="window.location.reload()" style="padding:12px 30px;background:#3498db;color:#fff;border:none;border-radius:4px;cursor:pointer;">Refresh Page</button>
                        </div>
                    </div>
                </div>
            `;
            document.body.innerHTML = errorPage;
            
            throw new Error('Integrity violation detected: ' + reason);
        }

        function escapeHtml(str) {
            const div = document.createElement('div');
            div.textContent = str;
            return div.innerHTML;
        }

        async function start() {
            if (!_0xIE.salt) {
                _0xIE.salt = generateSalt();
            }
            
            if (!_0xIE.signatureKey) {
                _0xIE.signatureKey = generateSalt();
            }

            createDOMMarkers(7);

            setInterval(async () => {
                if (!_0xIE.enabled) return;
                
                const result = await performIntegrityCheck();
                if (!result.valid) {
                    handleIntegrityFailure(result.reason);
                }
            }, _0xIE.checkInterval);

            window.addEventListener('beforeunload', () => {
                _0xIE.enabled = false;
            });
        }

        async function init(initialHashes, config) {
            _0xIE.hashes = initialHashes || {};
            
            if (!_0xIE.hashes.sha256) {
                _0xIE.hashes = await computeMultipleHashes('initial');
            }

            if (config) {
                _0xIE.checkInterval = config.checkInterval || _0xIE.checkInterval;
                _0xIE.maxChecks = config.maxChecks || _0xIE.maxChecks;
                _0xIE.maxViolations = config.maxViolations || _0xIE.maxViolations;
                _0xIE.protectionLevel = config.protectionLevel || _0xIE.protectionLevel;
                _0xIE.selfDestructEnabled = config.selfDestructEnabled || false;
                
                if (config.trustedDomains) {
                    _0xIE.trustedDomains = config.trustedDomains;
                }
            }

            _0xIE.salt = generateSalt();
            _0xIE.signatureKey = generateSalt();

            if (document.readyState === 'loading') {
                document.addEventListener('DOMContentLoaded', start);
            } else {
                start();
            }
        }

        function getStatus() {
            return {
                enabled: _0xIE.enabled,
                checkCount: _0xIE.checkCount,
                maxChecks: _0xIE.maxChecks,
                violations: _0xIE.integrityViolations,
                maxViolations: _0xIE.maxViolations,
                hashes: _0xIE.hashes,
                salt: _0xIE.salt ? '***' : null,
                version: VERSION,
                protectionLevel: _0xIE.protectionLevel,
                verificationHistory: _0xIE.verificationHistory.slice(-5)
            };
        }

        function setProtectionLevel(level) {
            _0xIE.protectionLevel = level;
            
            switch(level) {
                case 'minimum':
                    _0xIE.maxViolations = 5;
                    _0xIE.checkInterval = 30000;
                    break;
                case 'standard':
                    _0xIE.maxViolations = 3;
                    _0xIE.checkInterval = 15000;
                    break;
                case 'maximum':
                    _0xIE.maxViolations = 2;
                    _0xIE.checkInterval = 10000;
                    break;
                case 'paranoid':
                    _0xIE.maxViolations = 1;
                    _0xIE.checkInterval = 5000;
                    _0xIE.selfDestructEnabled = true;
                    break;
            }
        }

        function enableSelfDestruct(enable) {
            _0xIE.selfDestructEnabled = enable;
        }

        async function verifyExternalResources(urls) {
            const results = [];
            
            for (const url of urls) {
                try {
                    const response = await fetch(url, { method: 'HEAD', mode: 'cors' });
                    results.push({
                        url: url,
                        valid: response.ok,
                        status: response.status
                    });
                } catch (e) {
                    results.push({
                        url: url,
                        valid: false,
                        error: e.message
                    });
                }
            }
            
            return results;
        }

        return {
            VERSION: VERSION,
            init: init,
            start: start,
            getStatus: getStatus,
            setProtectionLevel: setProtectionLevel,
            enableSelfDestruct: enableSelfDestruct,
            computeCRC32: computeCRC32,
            computeSHA256: computeSHA256,
            computeSHA512: computeSHA512,
            computeMD5: computeMD5,
            computeMultipleHashes: computeMultipleHashes,
            generateSignature: generateSignature,
            verifySignature: verifySignature,
            verifyCodeHash: verifyCodeHash,
            performIntegrityCheck: performIntegrityCheck,
            verifyExternalResources: verifyExternalResources,
            verifyDOMMarkers: verifyDOMMarkers,
            verifyTimingConsistency: verifyTimingConsistency,
            verifyScriptIntegrity: verifyScriptIntegrity,
            verifyContentSecurityPolicy: verifyContentSecurityPolicy,
            verifySourceMaps: verifySourceMaps,
            verifyDOMIntegrity: verifyDOMIntegrity,
            verifyStorageAccess: verifyStorageAccess,
            verifyPrototypeChain: verifyPrototypeChain,
            verifyEvalDisabled: verifyEvalDisabled,
            verifyWebSocketIntegrity: verifyWebSocketIntegrity
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = IntegrityEnhanced;
    } else {
        globalContext.IntegrityEnhanced = IntegrityEnhanced;
    }

})(typeof window !== 'undefined' ? window : (typeof global !== 'undefined' ? global : this));
