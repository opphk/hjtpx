/**
 * Enhanced Anti-Debug and Reverse Engineering Protection
 * 
 * This module provides multiple layers of anti-debug and anti-tampering
 * protection, including debugger detection, breakpoint detection,
 * timing checks, runtime integrity verification, and enhanced
 * environment fingerprinting.
 * 
 * Note: This is a basic but functional implementation. It may not
 * be 100% effective against all modern reverse engineering techniques,
 * and may have edge cases we haven't discovered.
 */

(function(global) {
    'use strict';

    // Configuration
    const CONFIG = {
        enabled: true,
        checkInterval: 100,
        maxTimeDrift: 100,
        fingerprintRefresh: 60000,
        actions: ['hide', 'log', 'throw'],
        integrityCheck: true,
        fingerprintCheck: true
    };

    // Internal state
    const state = {
        debugDetected: false,
        detectionReason: null,
        lastCheckTime: Date.now(),
        originalFingerprint: null,
        scriptsHashes: new Map()
    };

    // Utility functions
    const utils = {
        randomString(length) {
            const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
            let result = '';
            for (let i = 0; i < length; i++) {
                result += chars.charAt(Math.floor(Math.random() * chars.length));
            }
            return result;
        },

        sha256(str) {
            try {
                let hash = 0;
                for (let i = 0; i < str.length; i++) {
                    const char = str.charCodeAt(i);
                    hash = ((hash << 5) - hash) + char;
                    hash = hash & hash;
                }
                return Math.abs(hash).toString(16);
            } catch (e) {
                return this.randomString(8);
            }
        },

        now() {
            return Date.now();
        },

        performanceNow() {
            try {
                return performance.now();
            } catch (e) {
                return this.now();
            }
        }
    };

    // Debugger Detection Module
    const DebuggerDetector = {
        methods: [],

        init() {
            this.methods = [
                this.detectDebuggerStatement.bind(this),
                this.detectConsoleLog.bind(this),
                this.detectToStringHook.bind(this),
                this.detectDebuggerTrap.bind(this),
                this.detectTimeDrift.bind(this)
            ];
        },

        detectDebuggerStatement() {
            const start = utils.performanceNow();
            try {
                debugger;
            } catch (e) {
            }
            const end = utils.performanceNow();
            if (end - start > 50) {
                return { detected: true, reason: 'debugger_statement', delay: end - start };
            }
            return { detected: false };
        },

        detectConsoleLog() {
            const original = Function.prototype.toString;
            let detected = false;
            try {
                const div = document.createElement('div');
                const id = utils.randomString(8);
                div.id = id;
                div.toString = function() {
                    detected = true;
                    return '[object HTMLDivElement]';
                };
                console.log('%c', div);
            } catch (e) {
            }
            return { detected, reason: 'console_access' };
        },

        detectToStringHook() {
            const original = Function.prototype.toString;
            try {
                let called = false;
                const testObj = {
                    toString: function() {
                        called = true;
                        return 'test';
                    }
                };
                console.log('%c', testObj);
                if (called) {
                    return { detected: true, reason: 'tostring_hook' };
                }
            } catch (e) {
            }
            return { detected: false };
        },

        detectDebuggerTrap() {
            const traps = [
                () => {
                    const start = utils.performanceNow();
                    for (let i = 0; i < 100; i++) {
                        debugger;
                    }
                    const end = utils.performanceNow();
                    return end - start > 200;
                },
                () => {
                    try {
                        const x = new Function('debugger');
                        const start = utils.performanceNow();
                        for (let i = 0; i < 50; i++) x();
                        const end = utils.performanceNow();
                        return end - start > 100;
                    } catch (e) {
                        return false;
                    }
                }
            ];
            for (const trap of traps) {
                if (trap()) {
                    return { detected: true, reason: 'debugger_trap' };
                }
            }
            return { detected: false };
        },

        detectTimeDrift() {
            const now = utils.now();
            const pnow = utils.performanceNow();
            const timeSinceLastCheck = now - state.lastCheckTime;
            
            if (timeSinceLastCheck < 0 || timeSinceLastCheck > 5000) {
                return { detected: true, reason: 'time_drift', drift: timeSinceLastCheck };
            }
            
            state.lastCheckTime = now;
            return { detected: false };
        },

        async checkAll() {
            for (const method of this.methods) {
                try {
                    const result = method();
                    if (result.detected) {
                        return result;
                    }
                } catch (e) {
                }
            }
            return { detected: false };
        }
    };

    // Breakpoint Detection Module
    const BreakpointDetector = {
        init() {
        },

        detectThroughPerformance() {
            const iterations = 1000000;
            const start = utils.performanceNow();
            
            let sum = 0;
            for (let i = 0; i < iterations; i++) {
                sum += Math.sin(i) * Math.cos(i);
            }
            
            const end = utils.performanceNow();
            const duration = end - start;
            const expected = Math.max(iterations / 100000, 50);
            
            if (duration > expected * 3) {
                return { detected: true, reason: 'performance_anomaly', duration, expected };
            }
            return { detected: false };
        },

        detectThroughTimingFunction() {
            const checks = [];
            for (let i = 0; i < 5; i++) {
                const start = utils.performanceNow();
                let x = 0;
                for (let j = 0; j < 100000; j++) {
                    x += j;
                }
                checks.push(utils.performanceNow() - start);
            }
            
            const avg = checks.reduce((a, b) => a + b, 0) / checks.length;
            const max = Math.max(...checks);
            
            if (max > avg * 10) {
                return { detected: true, reason: 'timing_anomaly' };
            }
            return { detected: false };
        },

        async checkAll() {
            const checks = [
                this.detectThroughPerformance.bind(this),
                this.detectThroughTimingFunction.bind(this)
            ];
            
            for (const check of checks) {
                const result = check();
                if (result.detected) {
                    return result;
                }
            }
            return { detected: false };
        }
    };

    // Runtime Integrity Check Module
    const IntegrityChecker = {
        originalFunctions: new Map(),

        init() {
            this.captureState();
            this.watchDOM();
        },

        captureState() {
            const functionsToWatch = [
                'eval', 'setTimeout', 'setInterval', 'Function',
                'document.createElement', 'document.write',
                'console.log', 'console.error'
            ];
            
            for (const name of functionsToWatch) {
                try {
                    let obj = global;
                    let prop = name;
                    
                    if (name.includes('.')) {
                        const parts = name.split('.');
                        obj = global;
                        for (let i = 0; i < parts.length - 1; i++) {
                            obj = obj[parts[i]];
                            if (!obj) break;
                        }
                        prop = parts[parts.length - 1];
                    }
                    
                    if (obj && obj[prop]) {
                        const original = obj[prop];
                        this.originalFunctions.set(name, {
                            obj,
                            prop,
                            original,
                            hash: utils.sha256(original.toString())
                        });
                    }
                } catch (e) {
                }
            }
            
            this.captureScriptHashes();
        },

        captureScriptHashes() {
            try {
                const scripts = document.querySelectorAll('script');
                scripts.forEach((script, index) => {
                    if (script.src) {
                        state.scriptsHashes.set(script.src, 'external');
                    } else if (script.textContent) {
                        state.scriptsHashes.set(`inline-${index}`, utils.sha256(script.textContent));
                    }
                });
            } catch (e) {
            }
        },

        watchDOM() {
            try {
                const observer = new MutationObserver((mutations) => {
                    for (const mutation of mutations) {
                        for (const node of mutation.addedNodes) {
                            if (node.nodeName === 'SCRIPT') {
                                this.onScriptAdded(node);
                            }
                        }
                    }
                });
                
                observer.observe(document.documentElement, {
                    childList: true,
                    subtree: true
                });
            } catch (e) {
            }
        },

        onScriptAdded(script) {
            try {
                let hash;
                if (script.src) {
                    hash = 'external-new';
                } else {
                    hash = utils.sha256(script.textContent);
                }
                
                if (![...state.scriptsHashes.values()].includes(hash)) {
                    this.onIntegrityViolation('unauthorized_script');
                }
            } catch (e) {
            }
        },

        checkFunctionIntegrity() {
            for (const [name, data] of this.originalFunctions) {
                try {
                    const current = data.obj[data.prop];
                    if (!current) continue;
                    
                    const currentHash = utils.sha256(current.toString());
                    if (currentHash !== data.hash) {
                        return {
                            detected: true,
                            reason: 'function_tampered',
                            function: name
                        };
                    }
                } catch (e) {
                }
            }
            return { detected: false };
        },

        checkDOMIntegrity() {
            try {
                const scripts = document.querySelectorAll('script');
                if (scripts.length > state.scriptsHashes.size + 5) {
                    return { detected: true, reason: 'too_many_scripts' };
                }
            } catch (e) {
            }
            return { detected: false };
        },

        checkEnvironmentIntegrity() {
            const expectedProps = ['window', 'document', 'navigator'];
            for (const prop of expectedProps) {
                if (!(prop in global)) {
                    return { detected: true, reason: 'environment_tampered', prop };
                }
            }
            return { detected: false };
        },

        onIntegrityViolation(reason) {
            AntiDebug.onDetection(reason, 'integrity');
        },

        async checkAll() {
            const checks = [
                this.checkFunctionIntegrity.bind(this),
                this.checkDOMIntegrity.bind(this),
                this.checkEnvironmentIntegrity.bind(this)
            ];
            
            for (const check of checks) {
                const result = check();
                if (result.detected) {
                    return result;
                }
            }
            return { detected: false };
        }
    };

    // Enhanced Fingerprint Module
    const FingerprintEnhancer = {
        init() {
            state.originalFingerprint = this.generate();
        },

        generate() {
            const components = [];
            
            try {
                components.push('scr:' + screen.width + 'x' + screen.height + 'x' + screen.colorDepth);
            } catch (e) {}
            
            try {
                components.push('lang:' + (navigator.language || ''));
            } catch (e) {}
            
            try {
                components.push('tz:' + (Intl.DateTimeFormat().resolvedOptions().timeZone || ''));
            } catch (e) {}
            
            try {
                components.push('cpu:' + (navigator.hardwareConcurrency || ''));
            } catch (e) {}
            
            try {
                components.push('mem:' + (navigator.deviceMemory || ''));
            } catch (e) {}
            
            try {
                components.push('plat:' + (navigator.platform || ''));
            } catch (e) {}
            
            try {
                const canvas = document.createElement('canvas');
                canvas.width = 200;
                canvas.height = 50;
                const ctx = canvas.getContext('2d');
                if (ctx) {
                    ctx.textBaseline = 'top';
                    ctx.font = '14px Arial';
                    ctx.fillStyle = '#f60';
                    ctx.fillRect(0, 0, 50, 50);
                    ctx.fillStyle = '#069';
                    ctx.fillText('fp-test', 10, 20);
                    const dataUrl = canvas.toDataURL();
                    components.push('cnv:' + utils.sha256(dataUrl));
                }
            } catch (e) {}
            
            try {
                const gl = document.createElement('canvas').getContext('webgl');
                if (gl) {
                    const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                    if (debugInfo) {
                        const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                        components.push('wgl:' + utils.sha256(renderer || ''));
                    }
                }
            } catch (e) {}
            
            try {
                if (navigator.plugins) {
                    components.push('plg:' + navigator.plugins.length);
                }
            } catch (e) {}
            
            try {
                const audioCtx = new (window.AudioContext || window.webkitAudioContext)();
                const oscillator = audioCtx.createOscillator();
                components.push('aud:' + utils.sha256(oscillator.type));
                audioCtx.close();
            } catch (e) {}
            
            return components.join('|');
        },

        verify() {
            const current = this.generate();
            if (state.originalFingerprint && current !== state.originalFingerprint) {
                return { detected: true, reason: 'fingerprint_changed' };
            }
            return { detected: false };
        }
    };

    // Main Anti-Debug Controller
    const AntiDebug = {
        config: CONFIG,
        state: state,
        checkIntervalId: null,

        init(options = {}) {
            Object.assign(this.config, options);
            
            if (!this.config.enabled) {
                return this;
            }

            DebuggerDetector.init();
            BreakpointDetector.init();
            IntegrityChecker.init();
            FingerprintEnhancer.init();

            this.startMonitoring();
            this.setupKeyboardPrevention();
            this.protectFromTampering();
            
            return this;
        },

        startMonitoring() {
            const runChecks = async () => {
                if (state.debugDetected) return;

                const [debugResult, breakpointResult, integrityResult, fingerprintResult] = await Promise.all([
                    DebuggerDetector.checkAll(),
                    BreakpointDetector.checkAll(),
                    IntegrityChecker.checkAll(),
                    FingerprintEnhancer.verify()
                ]);

                if (debugResult.detected) {
                    this.onDetection(debugResult.reason, 'debugger');
                } else if (breakpointResult.detected) {
                    this.onDetection(breakpointResult.reason, 'breakpoint');
                } else if (integrityResult.detected) {
                    this.onDetection(integrityResult.reason, 'integrity');
                } else if (fingerprintResult.detected) {
                    this.onDetection(fingerprintResult.reason, 'fingerprint');
                }
            };

            runChecks();
            this.checkIntervalId = setInterval(runChecks, this.config.checkInterval);
        },

        setupKeyboardPrevention() {
            document.addEventListener('keydown', (e) => {
                if (state.debugDetected) {
                    e.preventDefault();
                    e.stopPropagation();
                    return false;
                }

                const isF12 = e.key === 'F12';
                const isCtrlShiftI = e.ctrlKey && e.shiftKey && (e.key === 'I' || e.key === 'i');
                const isCtrlShiftJ = e.ctrlKey && e.shiftKey && (e.key === 'J' || e.key === 'j');
                const isCtrlU = e.ctrlKey && (e.key === 'U' || e.key === 'u');
                const isCtrlShiftC = e.ctrlKey && e.shiftKey && (e.key === 'C' || e.key === 'c');

                if (isF12 || isCtrlShiftI || isCtrlShiftJ || isCtrlU || isCtrlShiftC) {
                    e.preventDefault();
                    e.stopPropagation();
                    this.onDetection('devtools_shortcut', 'keyboard');
                    return false;
                }
            }, true);
        },

        protectFromTampering() {
            try {
                const propertiesToProtect = ['debugDetected', 'detectionReason'];
                propertiesToProtect.forEach(prop => {
                    Object.defineProperty(state, prop, {
                        configurable: false,
                        enumerable: true,
                        get: () => state['_' + prop],
                        set: (val) => {
                            if (!state['_' + prop]) {
                                state['_' + prop] = val;
                            }
                        }
                    });
                });
            } catch (e) {
            }
        },

        onDetection(reason, type) {
            if (state.debugDetected) return;

            state.debugDetected = true;
            state.detectionReason = { reason, type, timestamp: Date.now() };

            console.warn('[Security] Suspicious activity detected:', reason);

            if (this.config.actions.includes('log')) {
                this.logDetection(reason, type);
            }

            if (this.config.actions.includes('hide')) {
                this.hideContent();
            }

            if (this.config.actions.includes('throw')) {
                this.throwError();
            }
        },

        logDetection(reason, type) {
            try {
                const data = {
                    type: 'anti_debug_detection',
                    reason: reason,
                    detectionType: type,
                    userAgent: navigator.userAgent,
                    url: window.location.href,
                    timestamp: Date.now()
                };

                if (navigator.sendBeacon) {
                    navigator.sendBeacon('/api/security/log', JSON.stringify(data));
                } else {
                    fetch('/api/security/log', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify(data),
                        keepalive: true
                    }).catch(() => {});
                }
            } catch (e) {
            }
        },

        hideContent() {
            try {
                document.documentElement.style.display = 'none';
                document.body.innerHTML = '<div style="display:none;"></div>';
                
                const protect = () => {
                    document.documentElement.style.display = 'none';
                    if (document.body) {
                        document.body.innerHTML = '<div style="display:none;"></div>';
                    }
                };
                
                setInterval(protect, 100);
            } catch (e) {
            }
        },

        throwError() {
            try {
                const throwInfinite = () => {
                    throw new Error('Security violation');
                };
                setInterval(throwInfinite, 0);
                throwInfinite();
            } catch (e) {
            }
        },

        stop() {
            if (this.checkIntervalId) {
                clearInterval(this.checkIntervalId);
                this.checkIntervalId = null;
            }
        },

        isDebugDetected() {
            return state.debugDetected;
        },

        getState() {
            return { ...state };
        }
    };

    // Export to global
    global.EnhancedAntiDebug = AntiDebug;

    // Auto-initialize
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => AntiDebug.init());
    } else {
        AntiDebug.init();
    }

})(window);
