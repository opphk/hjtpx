(function(globalContext) {
    'use strict';

    const TamperDetection = (function() {
        const VERSION = '3.0.0';
        
        const _0xTD = {
            enabled: true,
            checkInterval: 3000,
            maxViolations: 3,
            violations: 0,
            protectedFunctions: new Map(),
            protectedObjects: new Map(),
            memorySnapshots: new Map(),
            hashCache: new Map(),
            eventListeners: [],
            tamperDetected: false,
            detectionCallback: null
        };

        function generateRandomToken(length) {
            const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
            let token = '';
            const array = new Uint8Array(length);
            crypto.getRandomValues(array);
            for (let i = 0; i < length; i++) {
                token += chars[array[i] % chars.length];
            }
            return token;
        }

        function computeObjectHash(obj) {
            let hash = 0;
            const str = JSON.stringify(obj, (key, value) => {
                if (typeof value === 'function') return 'function';
                return value;
            });
            for (let i = 0; i < str.length; i++) {
                hash = ((hash << 5) - hash) + str.charCodeAt(i);
                hash = hash & hash;
            }
            return Math.abs(hash).toString(16);
        }

        function protectFunction(fn, name) {
            const originalHash = computeObjectHash(fn);
            const wrapper = function(...args) {
                const currentHash = computeObjectHash(fn);
                if (currentHash !== originalHash) {
                    handleTampering('function_tampered', { name, expected: originalHash, actual: currentHash });
                    throw new Error('Function integrity compromised');
                }
                return fn.apply(this, args);
            };

            _0xTD.protectedFunctions.set(name || generateRandomToken(16), {
                original: fn,
                wrapper: wrapper,
                hash: originalHash
            });

            return wrapper;
        }

        function protectObject(obj, name) {
            const originalHash = computeObjectHash(obj);
            const proxy = new Proxy(obj, {
                get(target, prop) {
                    if (prop === '_isProtected') return true;
                    const currentHash = computeObjectHash(target);
                    if (currentHash !== originalHash) {
                        handleTampering('object_tampered', { name, prop, expected: originalHash, actual: currentHash });
                    }
                    return target[prop];
                },
                set(target, prop, value) {
                    handleTampering('object_modified', { name, prop, value });
                    return false;
                },
                deleteProperty(target, prop) {
                    handleTampering('object_property_deleted', { name, prop });
                    return false;
                }
            });

            _0xTD.protectedObjects.set(name || generateRandomToken(16), {
                original: obj,
                proxy: proxy,
                hash: originalHash
            });

            return proxy;
        }

        function createMemorySnapshot(name) {
            const snapshot = {
                timestamp: Date.now(),
                memoryUsage: window.performance && window.performance.memory ? 
                    window.performance.memory.usedJSHeapSize : 0,
                variables: {},
                checksum: 0
            };

            _0xTD.memorySnapshots.set(name, snapshot);
            return snapshot;
        }

        function compareMemorySnapshots(name) {
            const snapshot = _0xTD.memorySnapshots.get(name);
            if (!snapshot) return { changed: false };

            const currentMemory = window.performance && window.performance.memory ? 
                window.performance.memory.usedJSHeapSize : 0;

            const threshold = 1024 * 1024;
            const diff = Math.abs(currentMemory - snapshot.memoryUsage);

            if (diff > threshold) {
                return {
                    changed: true,
                    type: 'memory_anomaly',
                    details: {
                        expected: snapshot.memoryUsage,
                        actual: currentMemory,
                        diff: diff,
                        threshold: threshold
                    }
                };
            }

            return { changed: false };
        }

        function monitorDOMChanges() {
            const observer = new MutationObserver((mutations) => {
                for (const mutation of mutations) {
                    if (mutation.type === 'childList') {
                        for (const added of mutation.addedNodes) {
                            if (added.nodeType === Node.ELEMENT_NODE && added.tagName === 'SCRIPT') {
                                handleTampering('unauthorized_script', {
                                    src: added.src,
                                    text: added.textContent ? added.textContent.length : 0
                                });
                            }
                        }
                    }
                    
                    if (mutation.type === 'attributes') {
                        if (mutation.attributeName === 'src' && mutation.target.tagName === 'SCRIPT') {
                            handleTampering('script_src_changed', {
                                element: mutation.target,
                                oldValue: mutation.oldValue,
                                newValue: mutation.target.src
                            });
                        }
                    }
                }
            });

            observer.observe(document.body, {
                childList: true,
                attributes: true,
                subtree: true,
                attributeOldValue: true
            });

            _0xTD.eventListeners.push({ type: 'mutation', listener: observer });
        }

        function monitorConsole() {
            const originalLog = console.log;
            const originalError = console.error;
            const originalWarn = console.warn;
            const originalDebug = console.debug;

            const checkForTampering = (method, args) => {
                for (const arg of args) {
                    if (typeof arg === 'string') {
                        if (arg.includes('debugger') || 
                            arg.includes('breakpoint') ||
                            arg.includes('tamper') ||
                            arg.includes('hook')) {
                            handleTampering('suspicious_log', { method, message: arg });
                        }
                    }
                }
            };

            console.log = function(...args) {
                checkForTampering('log', args);
                return originalLog.apply(this, args);
            };

            console.error = function(...args) {
                checkForTampering('error', args);
                return originalError.apply(this, args);
            };

            console.warn = function(...args) {
                checkForTampering('warn', args);
                return originalWarn.apply(this, args);
            };

            console.debug = function(...args) {
                checkForTampering('debug', args);
                return originalDebug.apply(this, args);
            };

            _0xTD.eventListeners.push({ type: 'console', originalLog, originalError, originalWarn, originalDebug });
        }

        function monitorNetworkRequests() {
            const originalFetch = window.fetch;
            const originalXHR = XMLHttpRequest.prototype.open;

            window.fetch = function(resource, options) {
                checkRequest(resource, options);
                return originalFetch.apply(this, arguments);
            };

            XMLHttpRequest.prototype.open = function(method, url) {
                checkRequest(url, { method });
                return originalXHR.apply(this, arguments);
            };

            _0xTD.eventListeners.push({ type: 'network', originalFetch, originalXHR });
        }

        function checkRequest(resource, options) {
            const suspiciousPatterns = [
                /debugger/, /breakpoint/, /eval/, /unsafe-eval/,
                /data:text\/javascript/, /blob:/, /about:/
            ];

            const url = typeof resource === 'string' ? resource : resource.url;
            
            for (const pattern of suspiciousPatterns) {
                if (url && pattern.test(url)) {
                    handleTampering('suspicious_request', { url, method: options?.method });
                    break;
                }
            }
        }

        function monitorEval() {
            const originalEval = window.eval;
            const originalFunction = window.Function;

            window.eval = function(code) {
                if (isSuspiciousCode(code)) {
                    handleTampering('suspicious_eval', { code: code.substring(0, 100) });
                }
                return originalEval(code);
            };

            window.Function = function(...args) {
                const code = args[args.length - 1];
                if (isSuspiciousCode(code)) {
                    handleTampering('suspicious_function', { code: code.substring(0, 100) });
                }
                return originalFunction.apply(this, args);
            };

            _0xTD.eventListeners.push({ type: 'eval', originalEval, originalFunction });
        }

        function isSuspiciousCode(code) {
            const patterns = [
                /debugger\s*;/,
                /\beval\s*\(/,
                /\bFunction\s*\(/,
                /document\.write\s*\(/,
                /location\.href\s*=/,
                /window\.location\s*=/,
                /__proto__/,
                /Object\.defineProperty/,
                /Object\.setPrototypeOf/
            ];

            for (const pattern of patterns) {
                if (pattern.test(code)) {
                    return true;
                }
            }
            return false;
        }

        function handleTampering(type, details) {
            _0xTD.violations++;
            
            if (_0xTD.detectionCallback) {
                _0xTD.detectionCallback({
                    type,
                    details,
                    violations: _0xTD.violations,
                    timestamp: Date.now()
                });
            }

            if (_0xTD.violations >= _0xTD.maxViolations) {
                triggerProtection(type, details);
            }
        }

        function triggerProtection(type, details) {
            _0xTD.enabled = false;
            _0xTD.tamperDetected = true;

            document.documentElement.style.display = 'none';
            document.body.innerHTML = `
                <div style="position:fixed;top:0;left:0;width:100%;height:100%;background:#0d0d0d;color:#ff4444;font-family:Arial,sans-serif;display:flex;justify-content:center;align-items:center;">
                    <div style="text-align:center;max-width:600px;padding:40px;">
                        <div style="width:100px;height:100px;margin:0 auto 25px;border-radius:50%;background:#ff4444;display:flex;justify-content:center;align-items:center;">
                            <span style="font-size:48px;">&#128276;</span>
                        </div>
                        <h1 style="font-size:32px;margin:0 0 15px 0;">Tampering Detected</h1>
                        <p style="font-size:14px;opacity:0.9;margin:0 0 20px 0;">
                            Unauthorized modification or suspicious activity has been detected.<br>
                            For security reasons, this session has been terminated.
                        </p>
                        <div style="background:rgba(255,255,255,0.05);padding:15px;border-radius:4px;text-align:left;font-size:12px;">
                            <div><strong>Type:</strong> ${type}</div>
                            <div><strong>Violations:</strong> ${_0xTD.violations}/${_0xTD.maxViolations}</div>
                            <div><strong>Time:</strong> ${new Date().toLocaleString()}</div>
                        </div>
                    </div>
                </div>
            `;

            throw new Error(`Tampering detected: ${type} - ${JSON.stringify(details)}`);
        }

        function startMonitoring() {
            monitorDOMChanges();
            monitorConsole();
            monitorNetworkRequests();
            monitorEval();

            setInterval(() => {
                if (!_0xTD.enabled) return;

                for (const [name] of _0xTD.memorySnapshots) {
                    const result = compareMemorySnapshots(name);
                    if (result.changed) {
                        handleTampering(result.type, result.details);
                    }
                }

                for (const [name, data] of _0xTD.protectedFunctions) {
                    const currentHash = computeObjectHash(data.original);
                    if (currentHash !== data.hash) {
                        handleTampering('function_hash_mismatch', { name, expected: data.hash, actual: currentHash });
                    }
                }
            }, _0xTD.checkInterval);
        }

        function init(config) {
            if (config) {
                Object.assign(_0xTD, config);
            }

            if (document.readyState === 'loading') {
                document.addEventListener('DOMContentLoaded', startMonitoring);
            } else {
                startMonitoring();
            }
        }

        function getStatus() {
            return {
                enabled: _0xTD.enabled,
                violations: _0xTD.violations,
                maxViolations: _0xTD.maxViolations,
                tamperDetected: _0xTD.tamperDetected,
                protectedFunctionsCount: _0xTD.protectedFunctions.size,
                protectedObjectsCount: _0xTD.protectedObjects.size,
                memorySnapshotsCount: _0xTD.memorySnapshots.size,
                version: VERSION
            };
        }

        function setDetectionCallback(callback) {
            _0xTD.detectionCallback = callback;
        }

        function cleanup() {
            for (const listener of _0xTD.eventListeners) {
                if (listener.type === 'mutation' && listener.listener.disconnect) {
                    listener.listener.disconnect();
                } else if (listener.type === 'console') {
                    console.log = listener.originalLog;
                    console.error = listener.originalError;
                    console.warn = listener.originalWarn;
                    console.debug = listener.originalDebug;
                } else if (listener.type === 'network') {
                    window.fetch = listener.originalFetch;
                    XMLHttpRequest.prototype.open = listener.originalXHR;
                } else if (listener.type === 'eval') {
                    window.eval = listener.originalEval;
                    window.Function = listener.originalFunction;
                }
            }
            _0xTD.eventListeners = [];
        }

        return {
            VERSION: VERSION,
            init: init,
            getStatus: getStatus,
            protectFunction: protectFunction,
            protectObject: protectObject,
            createMemorySnapshot: createMemorySnapshot,
            compareMemorySnapshots: compareMemorySnapshots,
            setDetectionCallback: setDetectionCallback,
            cleanup: cleanup
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = TamperDetection;
    } else {
        globalContext.TamperDetection = TamperDetection;
    }

})(typeof window !== 'undefined' ? window : (typeof global !== 'undefined' ? global : this));