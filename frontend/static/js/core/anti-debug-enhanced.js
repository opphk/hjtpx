(function(globalContext) {
    'use strict';

    const AntiDebugEnhanced = (function() {
        const VERSION = '5.0.0';
        
        const _0xAD = {
            enabled: true,
            violations: 0,
            maxViolations: 3,
            checkInterval: 2000,
            lastCheckTime: 0,
            enabledChecks: {
                windowSize: true,
                debuggerStatement: true,
                consoleCheck: true,
                devtoolsCheck: true,
                timingAttack: true,
                breakPointDetection: true,
                memoryCheck: true,
                propertyCheck: true,
                chromeDetection: true,
                firebugDetection: true,
                sourceMapDetection: true,
                functionOverride: true,
                prototypeChain: true,
                stackTrace: true,
                performanceAnalysis: true,
                networkMonitoring: true,
                storageTampering: true
            },
            debugDetected: false,
            detectionHistory: [],
            triggerCallback: null,
            protectionLevel: 'maximum',
            obfuscationSeed: Math.random() * 1000000,
            timestamp: Date.now(),
            integrityHash: null
        };

        function detectWindowSizeAnomaly() {
            const threshold = 160;
            const widthDiff = window.outerWidth - window.innerWidth;
            const heightDiff = window.outerHeight - window.innerHeight;
            
            if (widthDiff > threshold || heightDiff > threshold) {
                return {
                    detected: true,
                    type: 'window_size',
                    details: { widthDiff, heightDiff, threshold }
                };
            }
            return { detected: false };
        }

        function detectDebuggerStatement() {
            const threshold = 50;
            const start = performance.now();
            
            try {
                debugger;
            } catch (e) {
                return {
                    detected: true,
                    type: 'debugger_exception',
                    details: { error: e.message }
                };
            }
            
            const end = performance.now();
            const elapsed = end - start;
            
            if (elapsed > threshold) {
                return {
                    detected: true,
                    type: 'debugger_paused',
                    details: { elapsed, threshold }
                };
            }
            return { detected: false };
        }

        function detectConsoleTampering() {
            const originalClear = console.clear;
            let clearCalled = false;
            
            console.clear = function() {
                clearCalled = true;
            };
            
            console.clear();
            
            console.clear = originalClear;
            
            if (clearCalled) {
                return { detected: false };
            }
            
            if (typeof console._commandLineAPI !== 'undefined') {
                return {
                    detected: true,
                    type: 'console_api',
                    details: { reason: '_commandLineAPI detected' }
                };
            }
            
            if (typeof console.profiles !== 'undefined') {
                return {
                    detected: true,
                    type: 'console_profiles',
                    details: { reason: 'profiles array exists' }
                };
            }
            
            return { detected: false };
        }

        function detectDevtools() {
            const testObj = {};
            Object.defineProperty(testObj, 'test', {
                get: function() {
                    return true;
                },
                configurable: false
            });
            
            try {
                Object.defineProperty(testObj, 'test', {
                    get: function() {
                        return false;
                    },
                    configurable: true
                });
                return {
                    detected: true,
                    type: 'devtools_configurable',
                    details: { reason: 'Property redefinition allowed' }
                };
            } catch (e) {
                return { detected: false };
            }
        }

        function detectTimingAttack() {
            const iterations = 100000;
            const start = Date.now();
            
            for (let i = 0; i < iterations; i++) {
                Math.sin(i) * Math.cos(i);
            }
            
            const elapsed = Date.now() - start;
            
            if (elapsed > 100) {
                return {
                    detected: true,
                    type: 'timing_anomaly',
                    details: { elapsed, expected: '< 100ms' }
                };
            }
            return { detected: false };
        }

        function detectBreakpoints() {
            const debugHandler = function() {};
            debugHandler.toString = function() {
                return 'debug';
            };
            
            let detected = false;
            
            try {
                const fn = new Function('debugger');
                const start = performance.now();
                fn();
                const end = performance.now();
                
                if (end - start > 50) {
                    detected = true;
                }
            } catch (e) {
                detected = true;
            }
            
            return {
                detected: detected,
                type: 'breakpoint',
                details: { reason: detected ? 'Breakpoint triggered' : 'No breakpoint detected' }
            };
        }

        function detectMemoryTampering() {
            const originalMemory = window.performance && window.performance.memory;
            
            if (originalMemory) {
                const usedJSHeapSize = originalMemory.usedJSHeapSize;
                
                try {
                    Object.defineProperty(originalMemory, 'usedJSHeapSize', {
                        get: function() {
                            return usedJSHeapSize + 1000;
                        }
                    });
                    
                    if (originalMemory.usedJSHeapSize !== usedJSHeapSize) {
                        return {
                            detected: true,
                            type: 'memory_tampering',
                            details: { reason: 'Memory properties modified' }
                        };
                    }
                } catch (e) {
                    // Property is read-only, which is normal
                }
            }
            
            return { detected: false };
        }

        function detectPropertyTampering() {
            const props = ['document', 'window', 'location', 'history'];
            
            for (const prop of props) {
                const descriptor = Object.getOwnPropertyDescriptor(window, prop);
                if (descriptor && descriptor.configurable) {
                    return {
                        detected: true,
                        type: 'property_tampering',
                        details: { property: prop, reason: 'Property is configurable' }
                    };
                }
            }
            
            return { detected: false };
        }

        function detectWebkitDebugger() {
            if (window.webkitDebuggerAPI) {
                return {
                    detected: true,
                    type: 'webkit_debugger',
                    details: { reason: 'Webkit debugger API detected' }
                };
            }
            
            if (window.chrome && window.chrome.runtime) {
                return {
                    detected: true,
                    type: 'chrome_runtime',
                    details: { reason: 'Chrome runtime API detected' }
                };
            }
            
            return { detected: false };
        }

        function detectFirebug() {
            if (window.firebug || window.Firebug) {
                return {
                    detected: true,
                    type: 'firebug',
                    details: { reason: 'Firebug detected' }
                };
            }
            return { detected: false };
        }

        function detectSourceMap() {
            const scripts = document.querySelectorAll('script');
            for (const script of scripts) {
                if (script.src && (script.src.indexOf('.map') > -1 || script.hasAttribute('sourceURL'))) {
                    return {
                        detected: true,
                        type: 'source_map',
                        details: { reason: 'Source map detected: ' + script.src }
                    };
                }
            }
            
            if (typeof window.webpackJsonp !== 'undefined' || typeof window.__REACT_DEVTOOLS_GLOBAL_HOOK__ !== 'undefined') {
                return {
                    detected: true,
                    type: 'devtools_hook',
                    details: { reason: 'React DevTools hook detected' }
                };
            }
            
            return { detected: false };
        }

        function detectFunctionOverrides() {
            const functionsToCheck = ['toString', 'valueOf', 'constructor', 'hasOwnProperty'];
            const suspiciousOverrides = [];
            
            for (const fnName of functionsToCheck) {
                const originalFn = Object.prototype[fnName];
                const currentFn = Object.prototype[fnName];
                
                if (originalFn !== currentFn) {
                    suspiciousOverrides.push({
                        function: fnName,
                        original: originalFn.toString(),
                        current: currentFn.toString()
                    });
                }
            }
            
            if (suspiciousOverrides.length > 0) {
                return {
                    detected: true,
                    type: 'function_override',
                    details: { overrides: suspiciousOverrides }
                };
            }
            
            return { detected: false };
        }

        function detectPrototypeChainTampering() {
            const originalObjectProto = Object.getPrototypeOf({});
            const originalArrayProto = Object.getPrototypeOf([]);
            
            try {
                Object.setPrototypeOf({}, null);
                Object.setPrototypeOf([], null);
                
                if (Object.getPrototypeOf({}) !== originalObjectProto) {
                    return {
                        detected: true,
                        type: 'prototype_chain',
                        details: { reason: 'Prototype chain modified' }
                    };
                }
            } catch (e) {
                // Normal behavior
            }
            
            return { detected: false };
        }

        function detectStackTraceAnalysis() {
            const stackTrace = new Error().stack;
            if (stackTrace) {
                const suspiciousPatterns = ['debugger', 'eval', 'Function'];
                for (const pattern of suspiciousPatterns) {
                    if (stackTrace.indexOf(pattern) > -1 && stackTrace.indexOf(pattern) < 20) {
                        return {
                            detected: true,
                            type: 'stack_trace',
                            details: { reason: 'Suspicious stack trace pattern: ' + pattern }
                        };
                    }
                }
            }
            return { detected: false };
        }

        function detectPerformanceAnalysis() {
            const start = performance.now();
            const operations = [];
            
            for (let i = 0; i < 1000; i++) {
                operations.push(i * i);
            }
            
            const elapsed = performance.now() - start;
            
            if (elapsed < 1) {
                return {
                    detected: true,
                    type: 'performance_analysis',
                    details: { reason: 'Suspiciously fast execution: ' + elapsed + 'ms' }
                };
            }
            
            return { detected: false };
        }

        function detectNetworkMonitoring() {
            const xhrOpen = XMLHttpRequest.prototype.open;
            const originalOpen = XMLHttpRequest.prototype.open.toString();
            
            if (XMLHttpRequest.prototype.open.toString() !== originalOpen) {
                return {
                    detected: true,
                    type: 'network_monitoring',
                    details: { reason: 'XMLHttpRequest.open has been modified' }
                };
            }
            
            const fetchOriginal = fetch.toString();
            if (fetch.toString() !== fetchOriginal) {
                return {
                    detected: true,
                    type: 'fetch_monitoring',
                    details: { reason: 'fetch has been modified' }
                };
            }
            
            return { detected: false };
        }

        function detectStorageTampering() {
            const storageEvents = ['localStorage', 'sessionStorage'];
            
            try {
                localStorage.setItem('_0xTest', 'test');
                localStorage.removeItem('_0xTest');
                
                sessionStorage.setItem('_0xTest', 'test');
                sessionStorage.removeItem('_0xTest');
            } catch (e) {
                return {
                    detected: true,
                    type: 'storage_tampering',
                    details: { reason: 'Storage access failed: ' + e.message }
                };
            }
            
            return { detected: false };
        }

        function computeIntegrityHash() {
            const code = document.currentScript ? document.currentScript.textContent : '';
            let hash = 0;
            for (let i = 0; i < code.length; i++) {
                const char = code.charCodeAt(i);
                hash = ((hash << 5) - hash) + char;
                hash = hash & hash;
            }
            _0xAD.integrityHash = Math.abs(hash).toString(16);
            return _0xAD.integrityHash;
        }

        function verifyIntegrity() {
            const currentHash = computeIntegrityHash();
            if (_0xAD.integrityHash && currentHash !== _0xAD.integrityHash) {
                return {
                    detected: true,
                    type: 'integrity_violation',
                    details: { reason: 'Code integrity check failed' }
                };
            }
            return { detected: false };
        }

        function performAllChecks() {
            const results = [];
            
            if (_0xAD.enabledChecks.windowSize) {
                results.push(detectWindowSizeAnomaly());
            }
            
            if (_0xAD.enabledChecks.debuggerStatement) {
                results.push(detectDebuggerStatement());
            }
            
            if (_0xAD.enabledChecks.consoleCheck) {
                results.push(detectConsoleTampering());
            }
            
            if (_0xAD.enabledChecks.devtoolsCheck) {
                results.push(detectDevtools());
                results.push(detectWebkitDebugger());
                results.push(detectFirebug());
            }
            
            if (_0xAD.enabledChecks.timingAttack) {
                results.push(detectTimingAttack());
            }
            
            if (_0xAD.enabledChecks.breakPointDetection) {
                results.push(detectBreakpoints());
            }
            
            if (_0xAD.enabledChecks.memoryCheck) {
                results.push(detectMemoryTampering());
            }
            
            if (_0xAD.enabledChecks.propertyCheck) {
                results.push(detectPropertyTampering());
            }
            
            if (_0xAD.enabledChecks.sourceMapDetection) {
                results.push(detectSourceMap());
            }
            
            if (_0xAD.enabledChecks.functionOverride) {
                results.push(detectFunctionOverrides());
            }
            
            if (_0xAD.enabledChecks.prototypeChain) {
                results.push(detectPrototypeChainTampering());
            }
            
            if (_0xAD.enabledChecks.stackTrace) {
                results.push(detectStackTraceAnalysis());
            }
            
            if (_0xAD.enabledChecks.performanceAnalysis) {
                results.push(detectPerformanceAnalysis());
            }
            
            if (_0xAD.enabledChecks.networkMonitoring) {
                results.push(detectNetworkMonitoring());
            }
            
            if (_0xAD.enabledChecks.storageTampering) {
                results.push(detectStorageTampering());
            }
            
            if (_0xAD.enabledChecks.chromeDetection) {
                results.push(detectChromeSpecificFeatures());
            }
            
            return results;
        }

        function detectChromeSpecificFeatures() {
            if (window.chrome && window.chrome.loadTimes) {
                return {
                    detected: true,
                    type: 'chrome_specific',
                    details: { reason: 'Chrome-specific APIs detected' }
                };
            }
            
            if (window.navigator.userAgent.indexOf('Chrome') > -1) {
                const crypto = window.crypto;
                if (crypto && crypto.subtle) {
                    return { detected: false };
                }
            }
            
            return { detected: false };
        }

        function handleDetection(detection) {
            _0xAD.violations++;
            _0xAD.detectionHistory.push({
                ...detection,
                timestamp: Date.now(),
                violationCount: _0xAD.violations,
                hash: _0xAD.integrityHash
            });

            if (_0xAD.triggerCallback) {
                _0xAD.triggerCallback(detection);
            }

            if (_0xAD.violations >= _0xAD.maxViolations) {
                triggerProtection();
            }
        }

        function triggerProtection() {
            _0xAD.enabled = false;
            _0xAD.debugDetected = true;

            document.documentElement.style.display = 'none';
            document.body.innerHTML = `
                <div style="position:fixed;top:0;left:0;width:100%;height:100%;background:#0a0a0a;color:#e74c3c;font-family:Arial,sans-serif;display:flex;justify-content:center;align-items:center;">
                    <div style="text-align:center;max-width:500px;padding:30px;">
                        <div style="font-size:72px;margin-bottom:20px;">&#9888;</div>
                        <h1 style="font-size:28px;margin:0 0 15px 0;">Security Alert</h1>
                        <p style="font-size:14px;opacity:0.8;margin:0 0 20px 0;">
                            Debugging tools have been detected.<br>
                            Your access has been temporarily restricted for security reasons.
                        </p>
                        <div style="font-size:12px;color:#666;">
                            Violation count: ${_0xAD.violations}/${_0xAD.maxViolations}
                        </div>
                        <div style="margin-top:20px;font-size:10px;color:#888;">
                            Error Code: ${_0xAD.integrityHash || 'UNKNOWN'}
                        </div>
                    </div>
                </div>
            `;

            throw new Error('Debug detection triggered - ' + JSON.stringify(_0xAD.detectionHistory));
        }

        function startDetectionLoop() {
            setInterval(() => {
                if (!_0xAD.enabled || _0xAD.debugDetected) return;

                const now = Date.now();
                if (now - _0xAD.lastCheckTime < _0xAD.checkInterval) return;
                _0xAD.lastCheckTime = now;

                const results = performAllChecks();
                const detected = results.filter(r => r.detected);

                if (detected.length > 0) {
                    handleDetection(detected[0]);
                }
            }, _0xAD.checkInterval);
        }

        function setupKeyboardListeners() {
            document.addEventListener('keydown', (e) => {
                if (!_0xAD.enabled) return;

                const shortcuts = [
                    e.key === 'F12',
                    e.ctrlKey && e.shiftKey && e.key === 'I',
                    e.ctrlKey && e.shiftKey && e.key === 'J',
                    e.ctrlKey && e.shiftKey && e.key === 'C',
                    e.ctrlKey && e.key === 'u'
                ];

                if (shortcuts.some(Boolean)) {
                    e.preventDefault();
                    handleDetection({
                        detected: true,
                        type: 'keyboard_shortcut',
                        details: { key: e.key, ctrl: e.ctrlKey, shift: e.shiftKey }
                    });
                }
            });

            document.addEventListener('contextmenu', (e) => {
                e.preventDefault();
            });
        }

        function setupPropertyProtections() {
            const protectedProps = ['document', 'window', 'location'];
            
            for (const prop of protectedProps) {
                const originalValue = window[prop];
                Object.defineProperty(window, prop, {
                    get: function() {
                        return originalValue;
                    },
                    set: function(value) {
                        handleDetection({
                            detected: true,
                            type: 'property_overwrite',
                            details: { property: prop }
                        });
                    },
                    configurable: false,
                    enumerable: true
                });
            }
        }

        function init(config) {
            if (config) {
                Object.assign(_0xAD, config);
                if (config.enabledChecks) {
                    Object.assign(_0xAD.enabledChecks, config.enabledChecks);
                }
            }

            computeIntegrityHash();

            if (document.readyState === 'loading') {
                document.addEventListener('DOMContentLoaded', () => {
                    startDetectionLoop();
                    setupKeyboardListeners();
                    setupPropertyProtections();
                });
            } else {
                startDetectionLoop();
                setupKeyboardListeners();
                setupPropertyProtections();
            }
        }

        function getStatus() {
            return {
                enabled: _0xAD.enabled,
                violations: _0xAD.violations,
                maxViolations: _0xAD.maxViolations,
                debugDetected: _0xAD.debugDetected,
                detectionHistory: _0xAD.detectionHistory.slice(-5),
                checkInterval: _0xAD.checkInterval,
                enabledChecks: _0xAD.enabledChecks,
                version: VERSION,
                integrityHash: _0xAD.integrityHash,
                protectionLevel: _0xAD.protectionLevel
            };
        }

        function setTriggerCallback(callback) {
            _0xAD.triggerCallback = callback;
        }

        function setCheckEnabled(checkName, enabled) {
            if (_0xAD.enabledChecks.hasOwnProperty(checkName)) {
                _0xAD.enabledChecks[checkName] = enabled;
                return true;
            }
            return false;
        }

        function setProtectionLevel(level) {
            _0xAD.protectionLevel = level;
            
            switch(level) {
                case 'minimum':
                    _0xAD.maxViolations = 5;
                    _0xAD.checkInterval = 5000;
                    break;
                case 'standard':
                    _0xAD.maxViolations = 3;
                    _0xAD.checkInterval = 3000;
                    break;
                case 'maximum':
                    _0xAD.maxViolations = 2;
                    _0xAD.checkInterval = 1500;
                    break;
                case 'paranoid':
                    _0xAD.maxViolations = 1;
                    _0xAD.checkInterval = 1000;
                    break;
            }
        }

        return {
            VERSION: VERSION,
            init: init,
            getStatus: getStatus,
            setTriggerCallback: setTriggerCallback,
            setCheckEnabled: setCheckEnabled,
            setProtectionLevel: setProtectionLevel,
            performChecks: performAllChecks,
            detectWindowSize: detectWindowSizeAnomaly,
            detectDebugger: detectDebuggerStatement,
            detectConsole: detectConsoleTampering,
            detectTiming: detectTimingAttack,
            verifyIntegrity: verifyIntegrity,
            computeHash: computeIntegrityHash
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = AntiDebugEnhanced;
    } else {
        globalContext.AntiDebugEnhanced = AntiDebugEnhanced;
    }

})(typeof window !== 'undefined' ? window : (typeof global !== 'undefined' ? global : this));
