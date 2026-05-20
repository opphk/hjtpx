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
                automationDetection: true,
                performanceProfiling: true,
                stackTraceAnalysis: true,
                sourceMapDetection: true,
                proxyDetection: true,
                vpnDetection: true
            },
            debugDetected: false,
            detectionHistory: [],
            triggerCallback: null,
            protectionMode: 'block',
            stealthMode: false
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

        function detectAutomation() {
            const indicators = [
                { name: 'selenium', check: () => navigator.webdriver === true },
                { name: 'puppeteer', check: () => navigator.userAgent.includes('HeadlessChrome') },
                { name: 'playwright', check: () => navigator.userAgent.includes('Playwright') },
                { name: 'phantomjs', check: () => navigator.userAgent.includes('PhantomJS') },
                { name: 'automation', check: () => navigator.automation === true }
            ];
            
            for (const indicator of indicators) {
                if (indicator.check()) {
                    return {
                        detected: true,
                        type: 'automation',
                        details: { reason: indicator.name + ' detected' }
                    };
                }
            }
            
            const testDiv = document.createElement('div');
            testDiv.id = 'webdriver-test';
            testDiv.style.display = 'none';
            document.body.appendChild(testDiv);
            
            const driverDiv = document.getElementById('webdriver-test');
            if (driverDiv && driverDiv.style.display === '') {
                return {
                    detected: true,
                    type: 'webdriver',
                    details: { reason: 'WebDriver automation detected' }
                };
            }
            
            return { detected: false };
        }

        function detectPerformanceProfiling() {
            if (window.performance) {
                const memory = window.performance.memory;
                if (memory) {
                    const jsHeapSize = memory.usedJSHeapSize;
                    if (jsHeapSize > 500 * 1024 * 1024) {
                        return {
                            detected: true,
                            type: 'memory_pressure',
                            details: { reason: 'High memory usage detected', heapSize: jsHeapSize }
                        };
                    }
                }
            }
            
            return { detected: false };
        }

        function detectStackTraceAnalysis() {
            try {
                const error = new Error();
                const stack = error.stack;
                
                if (stack) {
                    const suspiciousPatterns = [
                        /at Function\./,
                        /at Object\./,
                        /at \w+\s+\(/,
                        /__proto__/
                    ];
                    
                    for (const pattern of suspiciousPatterns) {
                        if (pattern.test(stack)) {
                            return {
                                detected: true,
                                type: 'stack_trace',
                                details: { reason: 'Suspicious stack trace detected' }
                            };
                        }
                    }
                }
            } catch (e) {
                return { detected: false };
            }
            
            return { detected: false };
        }

        function detectSourceMaps() {
            const scripts = document.querySelectorAll('script');
            
            for (const script of scripts) {
                if (script.src) {
                    try {
                        const url = new URL(script.src);
                        const lastSegment = url.pathname.split('/').pop();
                        if (lastSegment.endsWith('.map')) {
                            return {
                                detected: true,
                                type: 'source_map',
                                details: { reason: 'Source map file detected', url: script.src }
                            };
                        }
                    } catch (e) {
                        continue;
                    }
                }
            }
            
            return { detected: false };
        }

        function detectProxy() {
            const proxyIndicators = [
                typeof navigator.pdfViewerEnabled !== 'undefined',
                navigator.maxTouchPoints > 0 && navigator.maxTouchPoints !== window.navigator.maxTouchPoints,
                typeof navigator.hardwareConcurrency !== 'number' || navigator.hardwareConcurrency > 16
            ];
            
            for (const indicator of proxyIndicators) {
                if (indicator) {
                    return {
                        detected: true,
                        type: 'proxy',
                        details: { reason: 'Proxy or VPN detected' }
                    };
                }
            }
            
            return { detected: false };
        }

        function detectTimeAnomaly() {
            const start1 = Date.now();
            const end1 = Date.now();
            
            const testRuns = 100;
            let totalDiff = 0;
            
            for (let i = 0; i < testRuns; i++) {
                const start = Date.now();
                const end = Date.now();
                totalDiff += (end - start);
            }
            
            const avgDiff = totalDiff / testRuns;
            
            if (avgDiff > 10) {
                return {
                    detected: true,
                    type: 'time_anomaly',
                    details: { reason: 'Time manipulation detected', avgDiff: avgDiff }
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
                results.push(detectTimeAnomaly());
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
            
            if (_0xAD.enabledChecks.automationDetection) {
                results.push(detectAutomation());
            }
            
            if (_0xAD.enabledChecks.performanceProfiling) {
                results.push(detectPerformanceProfiling());
            }
            
            if (_0xAD.enabledChecks.stackTraceAnalysis) {
                results.push(detectStackTraceAnalysis());
            }
            
            if (_0xAD.enabledChecks.sourceMapDetection) {
                results.push(detectSourceMaps());
            }
            
            if (_0xAD.enabledChecks.proxyDetection) {
                results.push(detectProxy());
            }
            
            return results;
        }

        function handleDetection(detection) {
            _0xAD.violations++;
            _0xAD.detectionHistory.push({
                ...detection,
                timestamp: Date.now(),
                violationCount: _0xAD.violations
            });

            if (_0xAD.triggerCallback) {
                _0xAD.triggerCallback(detection);
            }

            if (_0xAD.violations >= _0xAD.maxViolations) {
                triggerProtection();
            }
        }

        function triggerProtection(type, details) {
            _0xAD.enabled = false;
            _0xAD.debugDetected = true;

            if (_0xAD.stealthMode) {
                const randomDelay = Math.random() * 1000 + 500;
                setTimeout(() => {
                    document.documentElement.style.display = 'none';
                }, randomDelay);
            }

            if (_0xAD.protectionMode === 'block') {
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
                        </div>
                    </div>
                `;

                throw new Error('Debug detection triggered - ' + JSON.stringify(_0xAD.detectionHistory));
            } else if (_0xAD.protectionMode === 'corrupt') {
                const originalBody = document.body.innerHTML;
                document.body.innerHTML = originalBody.replace(/<script[^>]*>([\s\S]*?)<\/script>/gi, '<script>void(0);</script>');
            }
        }

        function enableStealthMode() {
            _0xAD.stealthMode = true;
        }

        function disableStealthMode() {
            _0xAD.stealthMode = false;
        }

        function setProtectionMode(mode) {
            if (['block', 'corrupt', 'silent'].includes(mode)) {
                _0xAD.protectionMode = mode;
                return true;
            }
            return false;
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
                version: VERSION
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

        return {
            VERSION: VERSION,
            init: init,
            getStatus: getStatus,
            setTriggerCallback: setTriggerCallback,
            setCheckEnabled: setCheckEnabled,
            performChecks: performAllChecks,
            detectWindowSize: detectWindowSizeAnomaly,
            detectDebugger: detectDebuggerStatement,
            detectConsole: detectConsoleTampering,
            detectTiming: detectTimingAttack,
            detectAutomation: detectAutomation,
            detectPerformance: detectPerformanceProfiling,
            detectStackTrace: detectStackTraceAnalysis,
            detectSourceMaps: detectSourceMaps,
            detectProxy: detectProxy,
            detectTimeAnomaly: detectTimeAnomaly,
            enableStealthMode: enableStealthMode,
            disableStealthMode: disableStealthMode,
            setProtectionMode: setProtectionMode
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = AntiDebugEnhanced;
    } else {
        globalContext.AntiDebugEnhanced = AntiDebugEnhanced;
    }

})(typeof window !== 'undefined' ? window : (typeof global !== 'undefined' ? global : this));