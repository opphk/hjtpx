const IntegrityChecker = (function() {
    'use strict';

    const _0xIC = {
        version: '3.1',
        hashes: {},
        timestamp: null,
        checkInterval: 10000,
        maxChecks: 100,
        checkCount: 0,
        enabled: true,
        markers: [],
        integrityErrors: [],
        lastCheckTime: null,
        checkHistory: [],
        protectionLevel: 'high',
        checksumAlgorithm: 'SHA-256',
        signedScripts: {},
        trustedSources: [],
        violationCallback: null,
        whitelistedElements: [],
        integrityKeys: {},
        performanceThreshold: 100,
        minExecutionTime: 1,
        maxExecutionTime: 1000,
        enableTimingCheck: true,
        enableMemoryCheck: true,
        enableDOMCheck: true,
        enableNetworkCheck: true,
        enableSignatureCheck: true
    };

    const generateHash = function(data) {
        let hash = 0;
        if (data.length === 0) return hash.toString();

        for (let i = 0; i < data.length; i++) {
            const char = data.charCodeAt(i);
            hash = ((hash << 5) - hash) + char;
            hash = hash & hash;
        }

        return Math.abs(hash).toString(16);
    };

    const generateSHA256Hash = async function(data) {
        const encoder = new TextEncoder();
        const dataBuffer = typeof data === 'string' ? encoder.encode(data) : data;
        const hashBuffer = await crypto.subtle.digest('SHA-256', dataBuffer);
        return arrayBufferToBase64(hashBuffer);
    };

    const arrayBufferToBase64 = function(buffer) {
        const bytes = new Uint8Array(buffer);
        let binary = '';
        for (let i = 0; i < bytes.byteLength; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        return btoa(binary);
    };

    const base64ToArrayBuffer = function(base64) {
        const binaryString = atob(base64);
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
            bytes[i] = binaryString.charCodeAt(i);
        }
        return bytes.buffer;
    };

    const createMarkers = function() {
        const markerCount = 7;
        const markerTypes = ['div', 'span', 'script', 'style', 'meta', 'link', 'noscript'];
        
        for (let i = 0; i < markerCount; i++) {
            const marker = document.createElement(markerTypes[i % markerTypes.length]);
            marker.id = '_0xIC_marker_' + i;
            marker.style.display = 'none';
            marker.setAttribute('data-v', _0xIC.hashes.sha256);
            marker.setAttribute('data-t', Date.now().toString(36));
            marker.setAttribute('data-r', Math.random().toString(36).substr(2, 9));
            document.body.appendChild(marker);
            _0xIC.markers.push(marker.id);
        }
    };

    const verifyMarkers = function() {
        for (let i = 0; i < _0xIC.markers.length; i++) {
            const el = document.getElementById(_0xIC.markers[i]);
            if (!el) {
                recordError('marker_missing', 'Marker ' + _0xIC.markers[i] + ' not found');
                return false;
            }
            if (el.getAttribute('data-v') !== _0xIC.hashes.sha256) {
                recordError('marker_tampered', 'Marker ' + _0xIC.markers[i] + ' tampered');
                return false;
            }
        }
        return true;
    };

    const verifyTiming = function() {
        const start = performance.now();
        let sum = 0;
        for (let i = 0; i < 1000; i++) {
            sum += Math.random() * i;
        }
        const end = performance.now();
        const diff = end - start;

        if (diff > 100) {
            recordError('timing_anomaly', 'Execution time anomaly detected: ' + diff + 'ms');
            return false;
        }
        
        if (diff < 1) {
            recordError('timing_suspicious', 'Execution time too fast: ' + diff + 'ms');
            return false;
        }

        return true;
    };

    const verifyDOMIntegrity = function() {
        const criticalElements = [
            'script[src*="main"]',
            'script[src*="core"]',
            'script[src*="crypto"]',
            'link[href*="style"]',
            'link[rel="stylesheet"]'
        ];

        for (let i = 0; i < criticalElements.length; i++) {
            const selector = criticalElements[i];
            const elements = document.querySelectorAll(selector);
            
            for (let j = 0; j < elements.length; j++) {
                const el = elements[j];
                
                if (el.hasAttribute('integrity')) {
                    const expectedIntegrity = el.getAttribute('integrity');
                    if (expectedIntegrity !== _0xIC.hashes.sha256) {
                        recordError('integrity_mismatch', 'Integrity attribute mismatch for ' + selector);
                        return false;
                    }
                }

                if (el.src && !isTrustedSource(el.src)) {
                    recordError('untrusted_source', 'Untrusted source detected: ' + el.src);
                    return false;
                }
            }
        }

        return verifyScriptSignatures();
    };

    const isTrustedSource = function(src) {
        if (_0xIC.trustedSources.length === 0) {
            return true;
        }
        
        for (let i = 0; i < _0xIC.trustedSources.length; i++) {
            const pattern = _0xIC.trustedSources[i];
            if (typeof pattern === 'string') {
                if (src.startsWith(pattern) || src.includes(pattern)) {
                    return true;
                }
            } else if (pattern instanceof RegExp) {
                if (pattern.test(src)) {
                    return true;
                }
            }
        }
        return false;
    };

    const verifyScriptSignatures = function() {
        const scripts = document.querySelectorAll('script');
        
        for (let i = 0; i < scripts.length; i++) {
            const script = scripts[i];
            const signature = script.getAttribute('data-signature');
            
            if (signature) {
                const scriptHash = _0xIC.signedScripts[signature];
                if (!scriptHash) {
                    recordError('signature_not_found', 'Signature not registered: ' + signature);
                    return false;
                }
                
                const content = script.textContent || script.innerText || '';
                if (content) {
                    const currentHash = generateHash(content);
                    if (currentHash !== scriptHash) {
                        recordError('signature_mismatch', 'Script signature mismatch');
                        return false;
                    }
                }
            }
        }
        return true;
    };

    const verifyMemoryIntegrity = function() {
        if (!_0xIC.enableMemoryCheck) return true;
        
        const criticalObjects = ['window', 'document', 'Object', 'Function', 'Array', 'String', 'Number', 'Boolean', 'Date', 'RegExp', 'Promise', 'Map', 'Set'];
        
        for (let i = 0; i < criticalObjects.length; i++) {
            const objName = criticalObjects[i];
            const obj = window[objName];
            
            if (obj && typeof obj.toString === 'function') {
                const originalString = obj.toString();
                if (originalString.indexOf('[native code]') === -1 && objName !== 'window') {
                    recordError('memory_tampered', 'Object ' + objName + ' has been modified');
                    return false;
                }
            }
        }
        
        if (!verifyPrototypeChains()) return false;
        if (!verifyCoreFunctions()) return false;
        if (!verifyPropertyDescriptors()) return false;
        
        return true;
    };

    const verifyPrototypeChains = function() {
        const protoChecks = [
            { obj: Object, prop: 'prototype', expected: '[native code]' },
            { obj: Function, prop: 'prototype', expected: '[native code]' },
            { obj: Array, prop: 'prototype', expected: '[native code]' },
            { obj: String, prop: 'prototype', expected: '[native code]' },
            { obj: Number, prop: 'prototype', expected: '[native code]' },
            { obj: Boolean, prop: 'prototype', expected: '[native code]' },
            { obj: Date, prop: 'prototype', expected: '[native code]' }
        ];

        for (let i = 0; i < protoChecks.length; i++) {
            const check = protoChecks[i];
            if (check.obj[check.prop] && typeof check.obj[check.prop].toString === 'function') {
                const str = check.obj[check.prop].toString();
                if (str.indexOf(check.expected) === -1) {
                    recordError('prototype_tampered', check.obj.name + '.' + check.prop + ' tampered');
                    return false;
                }
            }
        }
        return true;
    };

    const verifyCoreFunctions = function() {
        const coreFunctions = [
            { obj: Object, method: 'defineProperty' },
            { obj: Object, method: 'getOwnPropertyDescriptor' },
            { obj: Object, method: 'getPrototypeOf' },
            { obj: Object, method: 'freeze' },
            { obj: Object, method: 'seal' },
            { obj: Object, method: 'keys' },
            { obj: Array, method: 'isArray' },
            { obj: JSON, method: 'parse' },
            { obj: JSON, method: 'stringify' }
        ];

        for (let i = 0; i < coreFunctions.length; i++) {
            const { obj, method } = coreFunctions[i];
            if (obj && typeof obj[method] === 'function') {
                const funcStr = obj[method].toString();
                if (funcStr.indexOf('[native code]') === -1) {
                    recordError('function_tampered', obj.name + '.' + method + ' has been modified');
                    return false;
                }
            }
        }
        return true;
    };

    const verifyPropertyDescriptors = function() {
        const criticalDescriptors = [
            { obj: window, prop: 'console' },
            { obj: window, prop: 'location' },
            { obj: window, prop: 'document' },
            { obj: Object.prototype, prop: 'hasOwnProperty' },
            { obj: Function.prototype, prop: 'toString' }
        ];

        for (let i = 0; i < criticalDescriptors.length; i++) {
            const { obj, prop } = criticalDescriptors[i];
            try {
                const descriptor = Object.getOwnPropertyDescriptor(obj, prop);
                if (!descriptor) {
                    recordError('descriptor_missing', 'Property descriptor missing for ' + prop);
                    return false;
                }
                if (!descriptor.configurable && !descriptor.writable) {
                    if (obj[prop] === undefined) {
                        recordError('property_undefined', 'Critical property ' + prop + ' is undefined');
                        return false;
                    }
                }
            } catch (e) {
                recordError('descriptor_error', 'Error checking descriptor for ' + prop + ': ' + e.message);
                return false;
            }
        }
        return true;
    };

    const verifyCodeChecksum = async function() {
        const scripts = document.querySelectorAll('script[data-checksum]');
        
        for (let i = 0; i < scripts.length; i++) {
            const script = scripts[i];
            const expectedChecksum = script.getAttribute('data-checksum');
            const content = script.textContent || '';
            
            if (content && expectedChecksum) {
                const actualChecksum = await generateSHA256Hash(content);
                if (actualChecksum !== expectedChecksum) {
                    recordError('checksum_mismatch', 'Script checksum mismatch for element ' + script.id);
                    return false;
                }
            }
        }
        return true;
    };

    const verifyNetworkIntegrity = function() {
        if (!_0xIC.enableNetworkCheck) return true;
        
        const networkAPIs = [
            { obj: navigator, method: 'sendBeacon', name: 'navigator.sendBeacon' },
            { obj: window, method: 'fetch', name: 'fetch' },
            { obj: window, method: 'XMLHttpRequest', name: 'XMLHttpRequest' },
            { obj: window, method: 'WebSocket', name: 'WebSocket' },
            { obj: navigator, method: 'getUserMedia', name: 'navigator.getUserMedia' },
            { obj: navigator, method: 'geolocation', name: 'navigator.geolocation' }
        ];

        for (let i = 0; i < networkAPIs.length; i++) {
            const { obj, method, name } = networkAPIs[i];
            if (obj && typeof obj[method] !== 'undefined') {
                if (typeof obj[method] === 'function') {
                    const funcStr = obj[method].toString();
                    if (funcStr.indexOf('[native code]') === -1 && 
                        funcStr.indexOf('class') === -1 && 
                        funcStr.indexOf('function') === 0) {
                        recordError('network_tampered', name + ' has been modified');
                        return false;
                    }
                } else if (typeof obj[method] === 'object') {
                    if (obj[method] !== null && typeof obj[method].getCurrentPosition === 'function') {
                        const funcStr = obj[method].getCurrentPosition.toString();
                        if (funcStr.indexOf('[native code]') === -1) {
                            recordError('network_tampered', name + '.getCurrentPosition has been modified');
                            return false;
                        }
                    }
                }
            }
        }

        if (!verifyURLIntegrity()) return false;
        
        return true;
    };

    const verifyURLIntegrity = function() {
        const urlParams = new URLSearchParams(window.location.search);
        const suspiciousParams = ['debug', 'debugger', 'test', 'testing', 'dev', 'development'];
        
        for (const param of suspiciousParams) {
            if (urlParams.has(param)) {
                const value = urlParams.get(param);
                if (value === 'true' || value === '1' || value === '') {
                    recordError('suspicious_url_param', 'Suspicious URL parameter detected: ' + param);
                    return false;
                }
            }
        }
        
        if (window.location.hash.indexOf('debug') !== -1 || 
            window.location.hash.indexOf('test') !== -1) {
            recordError('suspicious_hash', 'Suspicious hash fragment detected');
            return false;
        }
        
        return true;
    };

    const recordError = function(code, message) {
        const error = {
            code: code,
            message: message,
            timestamp: Date.now(),
            checkCount: _0xIC.checkCount
        };
        
        _0xIC.integrityErrors.push(error);
        
        if (_0xIC.checkHistory.length > 100) {
            _0xIC.checkHistory.shift();
        }
        _0xIC.checkHistory.push({
            timestamp: Date.now(),
            hasError: true,
            errorCode: code
        });

        if (typeof _0xIC.violationCallback === 'function') {
            try {
                _0xIC.violationCallback(error);
            } catch (e) {
                console.error('Violation callback error:', e);
            }
        }
    };

    const verify = async function() {
        if (_0xIC.checkCount >= _0xIC.maxChecks) {
            stop();
            return true;
        }

        _0xIC.lastCheckTime = Date.now();

        if (!verifyMarkers()) {
            handleFailure('Marker verification failed');
            return false;
        }

        if (!verifyTiming()) {
            handleFailure('Timing verification failed');
            return false;
        }

        if (!verifyDOMIntegrity()) {
            handleFailure('DOM integrity verification failed');
            return false;
        }

        if (!verifyMemoryIntegrity()) {
            handleFailure('Memory integrity verification failed');
            return false;
        }

        if (!verifyNetworkIntegrity()) {
            handleFailure('Network integrity verification failed');
            return false;
        }

        if (_0xIC.protectionLevel === 'high') {
            if (!(await verifyCodeChecksum())) {
                handleFailure('Code checksum verification failed');
                return false;
            }
        }

        _0xIC.checkCount++;
        
        _0xIC.checkHistory.push({
            timestamp: Date.now(),
            hasError: false,
            errorCode: null
        });

        if (_0xIC.checkHistory.length > 100) {
            _0xIC.checkHistory.shift();
        }

        return true;
    };

    const handleFailure = function(reason) {
        _0xIC.enabled = false;
        stop();

        document.documentElement.style.display = 'none';
        
        const errorId = Math.random().toString(36).substr(2, 9);
        const timestamp = new Date().toISOString();
        
        document.body.innerHTML = `
            <div style="position:fixed;top:0;left:0;width:100%;height:100%;background:#000;color:#fff;display:flex;justify-content:center;align-items:center;font-family:Arial,sans-serif;">
                <div style="text-align:center;max-width:600px;padding:40px;">
                    <div style="font-size:48px;margin-bottom:20px;">&#9888;</div>
                    <h1 style="font-size:28px;margin-bottom:16px;">Integrity Check Failed</h1>
                    <p style="font-size:16px;color:#ccc;margin-bottom:8px;">Code has been tampered with</p>
                    <p style="font-size:14px;color:#999;margin-bottom:24px;">${reason}</p>
                    <div style="font-size:12px;color:#666;border-top:1px solid #333;padding-top:16px;">
                        <p>Error ID: ${errorId}</p>
                        <p>Timestamp: ${timestamp}</p>
                    </div>
                </div>
            </div>
        `;

        const event = new CustomEvent('integrityViolation', { detail: { reason, errorId, timestamp } });
        window.dispatchEvent(event);

        throw new Error('Integrity verification failed: ' + reason);
    };

    const start = function() {
        createMarkers();

        _0xIC.timer = setInterval(async function() {
            await verify();
        }, _0xIC.checkInterval);

        window.addEventListener('beforeunload', function() {
            stop();
        });

        window.addEventListener('DOMContentLoaded', function() {
            verify();
        });
    };

    const stop = function() {
        if (_0xIC.timer) {
            clearInterval(_0xIC.timer);
            _0xIC.timer = null;
        }
    };

    const init = function(hashes, config) {
        _0xIC.hashes = hashes || {};
        _0xIC.timestamp = new Date().toISOString();

        if (config) {
            if (typeof config.checkInterval === 'number') {
                _0xIC.checkInterval = config.checkInterval;
            }
            if (typeof config.maxChecks === 'number') {
                _0xIC.maxChecks = config.maxChecks;
            }
            if (typeof config.protectionLevel === 'string') {
                _0xIC.protectionLevel = config.protectionLevel;
            }
            if (typeof config.checksumAlgorithm === 'string') {
                _0xIC.checksumAlgorithm = config.checksumAlgorithm;
            }
            if (Array.isArray(config.trustedSources)) {
                _0xIC.trustedSources = config.trustedSources;
            }
            if (typeof config.violationCallback === 'function') {
                _0xIC.violationCallback = config.violationCallback;
            }
        }

        start();
    };

    const getStatus = function() {
        return {
            enabled: _0xIC.enabled,
            checkCount: _0xIC.checkCount,
            maxChecks: _0xIC.maxChecks,
            hashes: _0xIC.hashes,
            timestamp: _0xIC.timestamp,
            lastCheckTime: _0xIC.lastCheckTime,
            protectionLevel: _0xIC.protectionLevel,
            integrityErrors: _0xIC.integrityErrors,
            checkHistory: _0xIC.checkHistory.slice(-10)
        };
    };

    const verifyManual = function(code) {
        const currentHash = generateHash(code);
        return currentHash === _0xIC.hashes.sha256;
    };

    const verifyManualSHA256 = async function(code) {
        const currentHash = await generateSHA256Hash(code);
        return currentHash === _0xIC.hashes.sha256;
    };

    const registerSignedScript = function(signature, hash) {
        _0xIC.signedScripts[signature] = hash;
    };

    const addTrustedSource = function(source) {
        if (!_0xIC.trustedSources.includes(source)) {
            _0xIC.trustedSources.push(source);
        }
    };

    const removeTrustedSource = function(source) {
        const index = _0xIC.trustedSources.indexOf(source);
        if (index !== -1) {
            _0xIC.trustedSources.splice(index, 1);
        }
    };

    const setViolationCallback = function(callback) {
        if (typeof callback === 'function') {
            _0xIC.violationCallback = callback;
        }
    };

    const clearErrors = function() {
        _0xIC.integrityErrors = [];
    };

    return {
        init: init,
        start: start,
        stop: stop,
        verify: verify,
        getStatus: getStatus,
        verifyManual: verifyManual,
        verifyManualSHA256: verifyManualSHA256,
        registerSignedScript: registerSignedScript,
        addTrustedSource: addTrustedSource,
        removeTrustedSource: removeTrustedSource,
        setViolationCallback: setViolationCallback,
        clearErrors: clearErrors,
        _getInternalState: function() { return _0xIC; }
    };
})();

if (typeof module !== 'undefined' && module.exports) {
    module.exports = IntegrityChecker;
}