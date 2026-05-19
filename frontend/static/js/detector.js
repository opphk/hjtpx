class AntiDebug {
    constructor(config = {}) {
        this.enabled = config.enabled !== false;
        this.checkInterval = config.checkInterval || 1000;
        this.threshold = config.threshold || 160;
        this.actions = config.actions || ['hide', 'log', 'block'];
        this.debugDetected = false;
        this.debugAttempts = 0;
        this.maxAttempts = config.maxAttempts || 3;
        this.antiTampering = config.antiTampering !== false;
        this.codeObfuscation = config.codeObfuscation !== false;
        this.init();
    }

    init() {
        if (!this.enabled) return;

        this.detectDevTools();
        this.detectConsole();
        this.obfuscateErrors();
        this.detectDebugging();
        this.protectPrototype();
        
        if (this.antiTampering) {
            this.setupAntiTampering();
        }
        
        if (this.codeObfuscation) {
            this.setupCodeObfuscation();
        }
        
        this.setupDynamicDetection();
    }

    setupDynamicDetection() {
        const self = this;
        
        setInterval(() => {
            self.checkFunctionIntegrity();
            self.detectMemoryInspection();
            self.checkTimingAnomalies();
        }, 3000);
    }
    
    checkFunctionIntegrity() {
        const criticalFunctions = [
            'authenticate',
            'verifyCaptcha',
            'submitChallenge',
            'generateToken'
        ];
        
        criticalFunctions.forEach(fnName => {
            if (typeof window[fnName] === 'function') {
                const originalLength = window[fnName].toString().length;
                if (this.functionLengths[fnName] && 
                    Math.abs(originalLength - this.functionLengths[fnName]) > 5) {
                    this.onDebugDetected('function_tampered');
                }
            }
        });
    }
    
    functionLengths: {},
    
    detectMemoryInspection() {
        const testKey = '__security_check__' + Math.random().toString(36).substring(7);
        const testValue = Date.now();
        
        try {
            localStorage.setItem(testKey, testValue);
            const retrieved = localStorage.getItem(testKey);
            
            if (retrieved !== testValue.toString()) {
                this.onDebugDetected('storage_inspection');
            }
            
            sessionStorage.setItem(testKey, testValue);
            const sessionRetrieved = sessionStorage.getItem(testKey);
            
            if (sessionRetrieved !== testValue.toString()) {
                this.onDebugDetected('session_inspection');
            }
            
            localStorage.removeItem(testKey);
            sessionStorage.removeItem(testKey);
        } catch (e) {
            this.onDebugDetected('storage_protection_bypass');
        }
    }
    
    checkTimingAnomalies() {
        const start = performance.now();
        const iterations = 1000;
        
        for (let i = 0; i < iterations; i++) {
            Math.sqrt(i);
        }
        
        const elapsed = performance.now() - start;
        
        if (elapsed > 100) {
            this.onDebugDetected('timing_anomaly');
        }
        
        const dateNow = Date.now();
        const performanceNow = performance.now();
        const diff = Math.abs(dateNow - performanceNow);
        
        if (diff > 10000) {
            this.onDebugDetected('time_manipulation');
        }
    }
    
    setupAntiTampering() {
        this.detectCodeModification();
        this.monitorVariableChanges();
        this.checkSourceIntegrity();
    }
    
    detectCodeModification() {
        const originalEval = window.eval;
        const self = this;
        
        window.eval = function(code) {
            if (code && typeof code === 'string') {
                const suspiciousPatterns = [
                    /debugger/i,
                    /void\s*0/i,
                    /breakpoint/i
                ];
                
                for (const pattern of suspiciousPatterns) {
                    if (pattern.test(code)) {
                        self.onDebugDetected('eval_injection');
                        return undefined;
                    }
                }
            }
            
            return originalEval.apply(window, arguments);
        };
        
        Object.defineProperty(window, 'eval', {
            get: function() {
                return window.eval;
            },
            configurable: false
        });
    }
    
    monitorVariableChanges() {
        const monitoredVars = ['location', 'navigator', 'document'];
        
        monitoredVars.forEach(varName => {
            const original = window[varName];
            
            if (original && typeof original === 'object') {
                const handler = {
                    get(target, prop) {
                        if (prop === 'toString') {
                            return original.toString.bind(original);
                        }
                        return target[prop];
                    },
                    set(target, prop, value) {
                        self.onDebugDetected('variable_modification');
                        target[prop] = value;
                        return true;
                    }
                };
                
                window[varName] = new Proxy(original, handler);
            }
        });
    }
    
    checkSourceIntegrity() {
        const scripts = document.querySelectorAll('script[src]');
        const scriptHashes = {};
        
        scripts.forEach(script => {
            const src = script.src;
            scriptHashes[src] = true;
        });
        
        const observer = new MutationObserver((mutations) => {
            mutations.forEach(mutation => {
                mutation.addedNodes.forEach(node => {
                    if (node.tagName === 'SCRIPT') {
                        if (!scriptHashes[node.src]) {
                            this.onDebugDetected('script_injection');
                        }
                    }
                    
                    if (node.tagName === 'IFRAME') {
                        this.onDebugDetected('iframe_injection');
                    }
                });
            });
        });
        
        observer.observe(document.documentElement, {
            childList: true,
            subtree: true
        });
    }
    
    setupCodeObfuscation() {
        this.obfuscateStrings();
        this.obfuscateFunctions();
    }
    
    obfuscateStrings() {
        const originalAlert = window.alert;
        const self = this;
        
        window.alert = function(message) {
            if (message && typeof message === 'string') {
                const obfuscated = self.xorEncrypt(message, 0x5A);
                message = obfuscated;
            }
            return originalAlert.apply(window, [message]);
        };
        
        const originalPrompt = window.prompt;
        window.prompt = function(message, defaultValue) {
            if (message && typeof message === 'string') {
                const obfuscated = self.xorEncrypt(message, 0x5A);
                message = obfuscated;
            }
            return originalPrompt.apply(window, [message, defaultValue]);
        };
        
        const originalConfirm = window.confirm;
        window.confirm = function(message) {
            if (message && typeof message === 'string') {
                const obfuscated = self.xorEncrypt(message, 0x5A);
                message = obfuscated;
            }
            return originalConfirm.apply(window, [message]);
        };
    }
    
    xorEncrypt(str, key) {
        let result = '';
        for (let i = 0; i < str.length; i++) {
            result += String.fromCharCode(str.charCodeAt(i) ^ key);
        }
        return result;
    }
    
    obfuscateFunctions() {
        const self = this;
        
        if (typeof window.authenticate === 'function') {
            const originalAuth = window.authenticate;
            window.authenticate = function(...args) {
                if (self.debugDetected) {
                    console.warn('Authentication blocked due to debug detection');
                    return Promise.reject(new Error('Security check failed'));
                }
                return originalAuth.apply(this, args);
            };
        }
        
        if (typeof window.verifyCaptcha === 'function') {
            const originalVerify = window.verifyCaptcha;
            window.verifyCaptcha = function(...args) {
                if (self.debugDetected) {
                    console.warn('Verification blocked due to debug detection');
                    return Promise.reject(new Error('Security check failed'));
                }
                return originalVerify.apply(this, args);
            };
        }
    }

    detectDevTools() {
        const threshold = this.threshold;
        let lastWidth = window.outerWidth;
        let lastHeight = window.outerHeight;

        setInterval(() => {
            const currentWidth = window.outerWidth;
            const currentHeight = window.outerHeight;
            const widthThreshold = currentWidth - window.innerWidth > threshold;
            const heightThreshold = currentHeight - window.innerHeight > threshold;

            if ((widthThreshold && currentWidth !== lastWidth) ||
                (heightThreshold && currentHeight !== lastHeight)) {
                this.onDebugDetected('devtools_size');
            }
            lastWidth = currentWidth;
            lastHeight = currentHeight;
        }, this.checkInterval);

        const originalDefineProperty = Object.defineProperty;
        try {
            Object.defineProperty(document, 'hidden', {
                get: function() {
                    window.__debugDetected = true;
                    return false;
                }
            });
        } catch (e) {}
    }

    detectConsole() {
        const originalConsole = {};
        const methods = ['log', 'warn', 'error', 'info', 'debug', 'table'];

        methods.forEach(method => {
            if (typeof console[method] === 'function') {
                originalConsole[method] = console[method].bind(console);
                console[method] = function(...args) {
                    if (this.isDebugMessage(args)) {
                        return;
                    }
                    originalConsole[method](...args);
                }.bind(this);
            }
        });

        const originalInfo = console.info;
        console.info = function(...args) {
            if (this.isDebugMessage(args)) {
                return;
            }
            originalInfo.apply(console, args);
        }.bind(this);

        Object.defineProperty(console, '__proto__', {
            get: () => {
                this.onDebugDetected('console_access');
                return console.__proto__;
            },
            set: (val) => {
                this.onDebugDetected('console_override');
                return val;
            },
            configurable: true
        });
    }

    isDebugMessage(args) {
        if (!args || args.length === 0) return false;
        const str = args[0]?.toString() || '';
        return str.includes('Developer Tools') ||
               str.includes('devtools') ||
               str.includes('debugger');
    }

    obfuscateErrors() {
        const originalError = window.Error;
        window.Error = function(...args) {
            const error = new originalError(...args);
            Object.defineProperty(error, 'stack', {
                get: () => {
                    return '[obfuscated]';
                }
            });
            return error;
        };
        window.Error.prototype = originalError.prototype;

        window.addEventListener('error', (e) => {
            if (this.debugDetected) {
                e.preventDefault();
                e.stopPropagation();
                return true;
            }
            return false;
        });

        window.addEventListener('unhandledrejection', (e) => {
            if (this.debugDetected) {
                e.preventDefault();
                e.stopPropagation();
            }
        });
    }

    detectDebugging() {
        const startTime = performance.now();
        debugger;
        const endTime = performance.now();

        if (endTime - startTime > 100) {
            this.onDebugDetected('debugger_used');
        }

        setInterval(() => {
            const checkTime = performance.now();
            debugger;
            const checkTime2 = performance.now();
            if (checkTime2 - checkTime > 50) {
                this.onDebugDetected('debugger_detected');
            }
        }, this.checkInterval);

        const originalToString = Function.prototype.toString;
        Function.prototype.toString = function(...args) {
            if (this.name === 'toString' && args.length === 0) {
                return 'function toString() { [native code] }';
            }
            return originalToString.apply(this, args);
        };
    }

    protectPrototype() {
        const protectedObjects = [window, document, navigator];

        protectedObjects.forEach(obj => {
            if (obj && obj.__proto__) {
                Object.defineProperty(obj, '__proto__', {
                    get: () => null,
                    set: (val) => {},
                    configurable: false
                });
            }
        });

        ['constructor', 'toString', 'valueOf'].forEach(prop => {
            try {
                const descriptor = Object.getOwnPropertyDescriptor(Function.prototype, prop);
                if (descriptor) {
                    Object.defineProperty(Function.prototype, prop, {
                        ...descriptor,
                        get: (function(original) {
                            return function() {
                                if (this === Function.prototype && arguments.length === 0) {
                                    return '[native code]';
                                }
                                return original.apply(this, arguments);
                            };
                        })(descriptor.get || descriptor.value)
                    });
                }
            } catch (e) {}
        });
    }

    onDebugDetected(reason) {
        if (this.debugDetected) return;

        this.debugDetected = true;
        console.log(`%c[Security] Debugging detected: ${reason}`, 'color: red; font-size: 14px;');

        if (this.actions.includes('hide')) {
            this.hideContent();
        }

        if (this.actions.includes('log')) {
            this.logDetection(reason);
        }

        if (this.actions.includes('block')) {
            this.blockInteraction();
        }
    }

    hideContent() {
        document.documentElement.style.cssText = 'display: none !important;';
        document.body && (document.body.style.cssText = 'display: none !important;');
    }

    logDetection(reason) {
        const detection = {
            timestamp: new Date().toISOString(),
            reason: reason,
            userAgent: navigator.userAgent,
            platform: navigator.platform,
            url: window.location.href
        };

        try {
            const blob = new Blob([JSON.stringify(detection)], {type: 'application/json'});
            const url = URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.href = url;
            link.download = 'debug-detection-' + Date.now() + '.json';
        } catch (e) {}

        if (typeof navigator.sendBeacon === 'function') {
            navigator.sendBeacon('/api/security/report', JSON.stringify({
                type: 'debug_detection',
                data: detection
            }));
        }
    }

    blockInteraction() {
        document.addEventListener('contextmenu', (e) => {
            if (this.debugDetected) {
                e.preventDefault();
                e.stopPropagation();
                return false;
            }
        }, true);

        document.addEventListener('keydown', (e) => {
            if (this.debugDetected) {
                if (e.keyCode === 123 ||
                    (e.ctrlKey && e.shiftKey && ['I', 'J', 'C'].includes(e.key)) ||
                    (e.ctrlKey && e.shiftKey && e.key === 'i') ||
                    (e.ctrlKey && e.shiftKey && e.key === 'j') ||
                    (e.ctrlKey && e.key === 'u')) {
                    e.preventDefault();
                    e.stopPropagation();
                    return false;
                }
            }
        }, true);

        document.addEventListener('selectstart', (e) => {
            if (this.debugDetected) {
                e.preventDefault();
                e.stopPropagation();
                return false;
            }
        }, true);
    }

    restore() {
        this.debugDetected = false;
        document.documentElement.style.cssText = '';
        document.body && (document.body.style.cssText = '');
    }

    isDebugging() {
        return this.debugDetected;
    }

    getStatus() {
        return {
            enabled: this.enabled,
            debugDetected: this.debugDetected,
            actions: this.actions,
            threshold: this.threshold,
            checkInterval: this.checkInterval
        };
    }
}

class CodeProtector {
    constructor(options = {}) {
        this.enabled = options.enabled !== false;
        this.integrityCheckEnabled = options.integrityCheck !== false;
        this.integrityHash = options.integrityHash || null;
        this.init();
    }

    init() {
        if (!this.enabled) return;

        this.addIntegrityCheck();
        this.protectGlobals();
        this.setupMonitoring();
    }

    addIntegrityCheck() {
        if (!this.integrityCheckEnabled) return;

        window.__codeIntegrity = {
            hash: this.integrityHash,
            verified: false,
            checkCount: 0
        };

        window.addEventListener('DOMContentLoaded', () => {
            this.verifyIntegrity();
        });

        setInterval(() => {
            this.verifyIntegrity();
        }, 30000);
    }

    verifyIntegrity() {
        if (!window.__codeIntegrity) return;

        window.__codeIntegrity.checkCount++;

        if (this.integrityHash && window.__codeIntegrity.hash !== this.integrityHash) {
            console.error('[Security] Code integrity check failed');
            this.onIntegrityViolation();
            return false;
        }

        window.__codeIntegrity.verified = true;
        return true;
    }

    onIntegrityViolation() {
        document.documentElement.style.cssText = 'display: none !important;';
        document.body && (document.body.innerHTML = '<div style="display:none"></div>');
    }

    protectGlobals() {
        const protectedGlobals = ['__codeIntegrity', '__debugDetected', '__protected'];

        protectedGlobals.forEach(name => {
            try {
                Object.defineProperty(window, name, {
                    get: () => window[name],
                    set: (val) => {
                        if (name === '__codeIntegrity' || name === '__debugDetected') {
                            return;
                        }
                        window[name] = val;
                    },
                    configurable: false,
                    enumerable: true
                });
            } catch (e) {}
        });
    }

    setupMonitoring() {
        const observer = new MutationObserver((mutations) => {
            mutations.forEach((mutation) => {
                mutation.addedNodes.forEach((node) => {
                    if (node.nodeType === 1 && node.tagName === 'SCRIPT') {
                        if (!node.src && node.textContent) {
                            const scriptHash = this.simpleHash(node.textContent);
                            if (scriptHash !== this.expectedScriptHash) {
                                console.warn('[Security] Unsanctioned script injection detected');
                            }
                        }
                    }
                });
            });
        });

        if (document.body) {
            observer.observe(document.body, {
                childList: true,
                subtree: true
            });
        } else {
            document.addEventListener('DOMContentLoaded', () => {
                observer.observe(document.body, {
                    childList: true,
                    subtree: true
                });
            });
        }
    }

    simpleHash(str) {
        let hash = 0;
        for (let i = 0; i < str.length; i++) {
            const char = str.charCodeAt(i);
            hash = ((hash << 5) - hash) + char;
            hash = hash & hash;
        }
        return Math.abs(hash).toString(36);
    }

    setIntegrityHash(hash) {
        this.integrityHash = hash;
        if (window.__codeIntegrity) {
            window.__codeIntegrity.hash = hash;
        }
    }

    isIntegrityVerified() {
        return window.__codeIntegrity?.verified || false;
    }
}

class ParameterEncryptor {
    constructor(publicKey) {
        this.publicKey = publicKey;
        this.encoder = new TextEncoder();
    }

    async encrypt(data) {
        try {
            const jsonStr = JSON.stringify(data);
            const encoded = this.encoder.encode(jsonStr);

            const key = await crypto.subtle.importKey(
                'raw',
                this.deriveKey(this.publicKey),
                { name: 'AES-GCM' },
                false,
                ['encrypt']
            );

            const iv = crypto.getRandomValues(new Uint8Array(12));
            const encrypted = await crypto.subtle.encrypt(
                { name: 'AES-GCM', iv: iv },
                key,
                encoded
            );

            const combined = new Uint8Array(iv.length + encrypted.byteLength);
            combined.set(iv);
            combined.set(new Uint8Array(encrypted), iv.length);

            return this.toBase64(combined);
        } catch (e) {
            console.error('Encryption failed:', e);
            return null;
        }
    }

    async decrypt(encryptedData) {
        try {
            const combined = this.fromBase64(encryptedData);
            const iv = combined.slice(0, 12);
            const data = combined.slice(12);

            const key = await crypto.subtle.importKey(
                'raw',
                this.deriveKey(this.publicKey),
                { name: 'AES-GCM' },
                false,
                ['decrypt']
            );

            const decrypted = await crypto.subtle.decrypt(
                { name: 'AES-GCM', iv: iv },
                key,
                data
            );

            const decoded = new TextDecoder().decode(decrypted);
            return JSON.parse(decoded);
        } catch (e) {
            console.error('Decryption failed:', e);
            return null;
        }
    }

    deriveKey(publicKey) {
        const encoder = new TextEncoder();
        const data = encoder.encode(publicKey || 'hjtpx-default-key');
        return data.slice(0, 32);
    }

    toBase64(buffer) {
        const bytes = new Uint8Array(buffer);
        let binary = '';
        for (let i = 0; i < bytes.length; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
    }

    fromBase64(str) {
        str = str.replace(/-/g, '+').replace(/_/g, '/');
        while (str.length % 4) str += '=';
        const binary = atob(str);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) {
            bytes[i] = binary.charCodeAt(i);
        }
        return bytes;
    }
}

class RequestSigner {
    constructor(secretKey) {
        this.secretKey = secretKey || 'hjtpx-signing-key';
    }

    async generateSignature(method, path, timestamp, nonce, body) {
        const data = `${method}\n${path}\n${timestamp}\n${nonce}\n${body || ''}`;
        const encoder = new TextEncoder();
        const key = await crypto.subtle.importKey(
            'raw',
            encoder.encode(this.secretKey),
            { name: 'HMAC', hash: 'SHA-256' },
            false,
            ['sign']
        );

        const signature = await crypto.subtle.sign(
            'HMAC',
            key,
            encoder.encode(data)
        );

        return this.toBase64(signature);
    }

    async verifySignature(method, path, timestamp, nonce, body, signature) {
        const expectedSignature = await this.generateSignature(method, path, timestamp, nonce, body);
        return this.constantTimeCompare(signature, expectedSignature);
    }

    constantTimeCompare(a, b) {
        if (a.length !== b.length) return false;
        let result = 0;
        for (let i = 0; i < a.length; i++) {
            result |= a.charCodeAt(i) ^ b.charCodeAt(i);
        }
        return result === 0;
    }

    toBase64(buffer) {
        const bytes = new Uint8Array(buffer);
        let binary = '';
        for (let i = 0; i < bytes.length; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
    }
}

class SecurityManager {
    constructor(config = {}) {
        this.config = {
            antiDebug: config.antiDebug !== false,
            codeProtection: config.codeProtection !== false,
            parameterEncryption: config.parameterEncryption !== false,
            requestSigning: config.requestSigning !== false,
            publicKey: config.publicKey || 'hjtpx-public-key',
            secretKey: config.secretKey || 'hjtpx-secret-key'
        };

        this.antiDebug = null;
        this.codeProtector = null;
        this.parameterEncryptor = null;
        this.requestSigner = null;

        this.init();
    }

    init() {
        if (this.config.antiDebug) {
            this.antiDebug = new AntiDebug();
        }

        if (this.config.codeProtection) {
            this.codeProtector = new CodeProtector();
        }

        if (this.config.parameterEncryption) {
            this.parameterEncryptor = new ParameterEncryptor(this.config.publicKey);
        }

        if (this.config.requestSigning) {
            this.requestSigner = new RequestSigner(this.config.secretKey);
        }

        this.setupSecureRequest();
    }

    async setupSecureRequest() {
        if (!this.requestSigner) return;

        const originalFetch = window.fetch;
        const self = this;

        window.fetch = async function(url, options = {}) {
            const method = options.method || 'GET';
            const timestamp = Math.floor(Date.now() / 1000).toString();
            const nonce = await self.generateNonce();
            const body = options.body || '';

            let signature;
            if (typeof url === 'string') {
                signature = await self.requestSigner.generateSignature(
                    method,
                    url,
                    timestamp,
                    nonce,
                    typeof body === 'string' ? body : ''
                );
            } else if (url instanceof Request) {
                signature = await self.requestSigner.generateSignature(
                    url.method,
                    url.url,
                    timestamp,
                    nonce,
                    ''
                );
            }

            options.headers = {
                ...options.headers,
                'X-Timestamp': timestamp,
                'X-Nonce': nonce,
                'X-Signature': signature
            };

            return originalFetch.call(window, url, options);
        };
    }

    async generateNonce() {
        const array = new Uint8Array(16);
        crypto.getRandomValues(array);
        return Array.from(array, b => b.toString(16).padStart(2, '0')).join('');
    }

    async encryptParams(params) {
        if (!this.parameterEncryptor) {
            return params;
        }
        return this.parameterEncryptor.encrypt(params);
    }

    async decryptParams(encryptedData) {
        if (!this.parameterEncryptor) {
            return null;
        }
        return this.parameterEncryptor.decrypt(encryptedData);
    }

    isDebugging() {
        return this.antiDebug?.isDebugging() || false;
    }

    getStatus() {
        return {
            antiDebug: this.antiDebug?.getStatus() || null,
            debugging: this.isDebugging(),
            codeProtection: this.codeProtector?.isIntegrityVerified() || false,
            parameterEncryption: !!this.parameterEncryptor,
            requestSigning: !!this.requestSigner
        };
    }
}

window.AntiDebug = AntiDebug;
window.CodeProtector = CodeProtector;
window.ParameterEncryptor = ParameterEncryptor;
window.RequestSigner = RequestSigner;
window.SecurityManager = SecurityManager;
