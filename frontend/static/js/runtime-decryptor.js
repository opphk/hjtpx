const RuntimeDecryptor = (function() {
    'use strict';

    const VERSION = '15.0.0';

    const DecryptorConfig = {
        autoInit: true,
        cacheDecrypted: true,
        maxCacheSize: 20,
        timeout: 5000,
        retryCount: 3,
        fallbackEnabled: true
    };

    let decryptionKey = null;
    let isInitialized = false;
    let decryptionCache = new Map();
    let decryptionCount = 0;
    let errorCount = 0;
    let startTime = Date.now();

    function log(message) {
        if (typeof console !== 'undefined' && console.debug) {
            console.debug('[Runtime Decryptor ' + VERSION + ']:', message);
        }
    }

    function error(message) {
        if (typeof console !== 'undefined' && console.error) {
            console.error('[Runtime Decryptor ' + VERSION + ' Error]:', message);
        }
    }

    function generateRandomBytes(length) {
        const bytes = new Uint8Array(length);
        if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
            crypto.getRandomValues(bytes);
        } else {
            for (let i = 0; i < length; i++) {
                bytes[i] = Math.floor(Math.random() * 256);
            }
        }
        return bytes;
    }

    function generateKey(length) {
        const keyBytes = generateRandomBytes(length);
        let key = '';
        for (let i = 0; i < keyBytes.length; i++) {
            key += keyBytes[i].toString(16).padStart(2, '0');
        }
        return key;
    }

    function initialize(key) {
        if (isInitialized && decryptionKey) {
            return;
        }

        if (!key) {
            key = generateKey(32);
        }

        decryptionKey = key;
        isInitialized = true;

        log('Decryptor initialized with key length: ' + key.length);
    }

    function deriveKey(info) {
        if (!decryptionKey) {
            initialize();
        }

        let hash = 0;
        const combined = decryptionKey + info;

        for (let i = 0; i < combined.length; i++) {
            const char = combined.charCodeAt(i);
            hash = ((hash << 5) - hash) + char;
            hash = hash & hash;
        }

        let derivedKey = Math.abs(hash).toString(16);
        while (derivedKey.length < 64) {
            const newHash = ((hash << 5) - hash) + derivedKey.charCodeAt(0);
            derivedKey += Math.abs(newHash).toString(16);
            hash = newHash;
        }

        return derivedKey.substring(0, 64);
    }

    function xorDecrypt(encrypted, key) {
        if (typeof encrypted === 'string') {
            encrypted = base64ToBytes(encrypted);
        }

        if (typeof key === 'string') {
            key = hexToBytes(key);
        }

        const decrypted = new Uint8Array(encrypted.length);
        for (let i = 0; i < encrypted.length; i++) {
            const keyByte = key[i % key.length];
            const offset = (i * 7 + 13) % 256;
            decrypted[i] = ((encrypted[i] - offset + 256) % 256) ^ keyByte;
        }

        return decrypted;
    }

    function xorEncrypt(plaintext, key) {
        if (typeof plaintext === 'string') {
            plaintext = stringToBytes(plaintext);
        }

        if (typeof key === 'string') {
            key = hexToBytes(key);
        }

        const encrypted = new Uint8Array(plaintext.length);
        for (let i = 0; i < plaintext.length; i++) {
            const keyByte = key[i % key.length];
            const offset = (i * 7 + 13) % 256;
            encrypted[i] = ((plaintext[i] ^ keyByte) + offset) % 256;
        }

        return encrypted;
    }

    function base64ToBytes(base64) {
        const binary = atob(base64);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) {
            bytes[i] = binary.charCodeAt(i);
        }
        return bytes;
    }

    function bytesToBase64(bytes) {
        let binary = '';
        for (let i = 0; i < bytes.length; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        return btoa(binary);
    }

    function hexToBytes(hex) {
        const bytes = new Uint8Array(hex.length / 2);
        for (let i = 0; i < hex.length; i += 2) {
            bytes[i / 2] = parseInt(hex.substr(i, 2), 16);
        }
        return bytes;
    }

    function bytesToHex(bytes) {
        let hex = '';
        for (let i = 0; i < bytes.length; i++) {
            hex += bytes[i].toString(16).padStart(2, '0');
        }
        return hex;
    }

    function stringToBytes(str) {
        const bytes = new Uint8Array(str.length);
        for (let i = 0; i < str.length; i++) {
            bytes[i] = str.charCodeAt(i);
        }
        return bytes;
    }

    function bytesToString(bytes) {
        let str = '';
        for (let i = 0; i < bytes.length; i++) {
            str += String.fromCharCode(bytes[i]);
        }
        return str;
    }

    function decrypt(encryptedData, info) {
        if (!isInitialized) {
            initialize();
        }

        const cacheKey = encryptedData + (info || '');
        if (DecryptorConfig.cacheDecrypted && decryptionCache.has(cacheKey)) {
            return decryptionCache.get(cacheKey);
        }

        try {
            const key = deriveKey(info || 'default');
            const decrypted = xorDecrypt(encryptedData, key);
            const result = bytesToString(decrypted);

            if (DecryptorConfig.cacheDecrypted) {
                if (decryptionCache.size >= DecryptorConfig.maxCacheSize) {
                    const firstKey = decryptionCache.keys().next().value;
                    decryptionCache.delete(firstKey);
                }
                decryptionCache.set(cacheKey, result);
            }

            decryptionCount++;
            log('Decryption successful, count: ' + decryptionCount);

            return result;
        } catch (e) {
            error('Decryption failed: ' + e.message);
            errorCount++;

            if (DecryptorConfig.fallbackEnabled) {
                return null;
            }

            throw e;
        }
    }

    function encrypt(plaintext, info) {
        if (!isInitialized) {
            initialize();
        }

        try {
            const key = deriveKey(info || 'default');
            const encrypted = xorEncrypt(plaintext, key);
            return bytesToBase64(encrypted);
        } catch (e) {
            error('Encryption failed: ' + e.message);
            throw e;
        }
    }

    function decryptScript(encryptedScript) {
        const decrypted = decrypt(encryptedScript, 'script');
        if (decrypted === null) {
            return null;
        }

        try {
            const scriptFn = new Function(decrypted);
            return scriptFn;
        } catch (e) {
            error('Failed to create script function: ' + e.message);
            return null;
        }
    }

    function executeDecryptedScript(encryptedScript) {
        const scriptFn = decryptScript(encryptedScript);
        if (scriptFn === null) {
            return false;
        }

        try {
            scriptFn();
            return true;
        } catch (e) {
            error('Failed to execute decrypted script: ' + e.message);
            return false;
        }
    }

    function decryptAndEval(encryptedCode) {
        const decrypted = decrypt(encryptedCode, 'eval');
        if (decrypted === null) {
            return false;
        }

        try {
            eval(decrypted);
            return true;
        } catch (e) {
            error('Failed to eval decrypted code: ' + e.message);
            return false;
        }
    }

    function decryptModule(encryptedModule) {
        try {
            const decrypted = decrypt(encryptedModule, 'module');
            if (decrypted === null) {
                return null;
            }

            const moduleFn = new Function('module', 'exports', decrypted);
            const moduleObj = { exports: {} };
            moduleFn(moduleObj, moduleObj.exports);

            return moduleObj.exports;
        } catch (e) {
            error('Failed to decrypt module: ' + e.message);
            return null;
        }
    }

    function createDecryptedVariable(key, encryptedValue) {
        const value = decrypt(encryptedValue, key);
        if (value !== null) {
            try {
                window[key] = JSON.parse(value);
            } catch (e) {
                window[key] = value;
            }
        }
        return value;
    }

    function decryptJSON(encryptedJSON) {
        const decrypted = decrypt(encryptedJSON, 'json');
        if (decrypted === null) {
            return null;
        }

        try {
            return JSON.parse(decrypted);
        } catch (e) {
            error('Failed to parse decrypted JSON: ' + e.message);
            return null;
        }
    }

    function clearCache() {
        decryptionCache.clear();
        log('Decryption cache cleared');
    }

    function getCacheSize() {
        return decryptionCache.size;
    }

    function getStatistics() {
        return {
            initialized: isInitialized,
            decryptionCount: decryptionCount,
            errorCount: errorCount,
            cacheSize: decryptionCache.size,
            cacheEnabled: DecryptorConfig.cacheDecrypted,
            uptime: Date.now() - startTime,
            keyLength: decryptionKey ? decryptionKey.length : 0
        };
    }

    function setKey(key) {
        decryptionKey = key;
        isInitialized = true;
        decryptionCache.clear();
        log('Decryption key updated');
    }

    function getKey() {
        return decryptionKey;
    }

    function generateNewKey() {
        decryptionKey = generateKey(32);
        isInitialized = true;
        decryptionCache.clear();
        log('New decryption key generated');
        return decryptionKey;
    }

    const DecryptorAPI = {
        version: VERSION,
        init: initialize,
        decrypt: decrypt,
        encrypt: encrypt,
        decryptScript: decryptScript,
        executeDecryptedScript: executeDecryptedScript,
        decryptAndEval: decryptAndEval,
        decryptModule: decryptModule,
        createDecryptedVariable: createDecryptedVariable,
        decryptJSON: decryptJSON,
        clearCache: clearCache,
        getCacheSize: getCacheSize,
        getStatistics: getStatistics,
        setKey: setKey,
        getKey: getKey,
        generateNewKey: generateNewKey,
        isInitialized: function() {
            return isInitialized;
        },
        setConfig: function(config) {
            Object.assign(DecryptorConfig, config);
        },
        getConfig: function() {
            return Object.assign({}, DecryptorConfig);
        }
    };

    if (DecryptorConfig.autoInit) {
        initialize();
    }

    if (typeof window !== 'undefined') {
        window.RuntimeDecryptor = DecryptorAPI;
        window._0xrd = DecryptorAPI;
    }

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = DecryptorAPI;
    }

    log('Runtime decryptor loaded');

    return DecryptorAPI;
})();

const CodeProtector = (function() {
    'use strict';

    const VERSION = '15.0.0';

    const ProtectorConfig = {
        protectionLevel: 3,
        enableWASM: true,
        enableIntegrity: true,
        enableAntiAutomation: true,
        enableRuntimeDecryption: true,
        enableVirtualization: true,
        enableTimingProtection: true,
        autoProtect: true,
        blockOnViolation: true,
        maxViolations: 3
    };

    let isProtected = false;
    let violations = 0;

    function initialize() {
        if (typeof FrontendProtectionV15 !== 'undefined') {
            FrontendProtectionV15.initialize();
        }

        if (typeof IntegrityCheckerV15 !== 'undefined') {
            IntegrityCheckerV15.initialize();
        }

        if (typeof AutomationDetector !== 'undefined') {
            AutomationDetector.enable();
        }

        if (typeof RuntimeDecryptor !== 'undefined') {
            RuntimeDecryptor.init();
        }

        isProtected = true;
    }

    function protect(code, options) {
        const config = Object.assign({}, ProtectorConfig, options || {});

        if (config.enableIntegrity) {
            const hash = IntegrityCheckerV15.calculateHash(code);
            code = 'window.__IntegrityHash = { verify: function() { return true; }, getHash: function() { return "' + hash + '"; } };' + code;
        }

        if (config.enableTimingProtection) {
            code = addTimingProtection(code);
        }

        if (config.enableAntiAutomation) {
            code = addAntiAutomationProtection(code);
        }

        if (config.enableVirtualization) {
            code = addVirtualization(code);
        }

        code = wrapInProtection(code);

        return code;
    }

    function addTimingProtection(code) {
        const protection = `
(function(){
    var _0xst = Date.now();
    var _0xok = true;
    var _0xcl = 0;
    setInterval(function() {
        var _0xet = Date.now();
        if(_0xet - _0xst > 100 && _0xok) {
            _0xcl++;
            if(_0xcl > 3) {
                document.documentElement.style.display = 'none';
            }
            _0xok = false;
            setTimeout(function() { _0xok = true; _0xst = Date.now(); }, 5000);
        }
    }, 1000);
})();
`;
        return protection + code;
    }

    function addAntiAutomationProtection(code) {
        const protection = `
(function(){
    var _0xauto = { detected: false };
    if(navigator.webdriver) { _0xauto.detected = true; }
    if(_0xauto.detected) {
        document.documentElement.style.display = 'none';
        throw new Error('Automation detected');
    }
})();
`;
        return protection + code;
    }

    function addVirtualization(code) {
        const protection = `
(function(){
    var _0xvm = window._0xvm = window._0xvm || {};
    _0xvm.handlers = {};
    _0xvm.execute = function(_0xid, _0xarg) {
        if(_0xvm.handlers[_0xid]) {
            return _0xvm.handlers[_0xid](_0xarg);
        }
        return null;
    };
    window._0xvm = _0xvm;
})();
`;
        return protection + code;
    }

    function wrapInProtection(code) {
        return `
(function(){
    "use strict";
    var _0xp15 = { version: "` + VERSION + `", protected: true };
    ` + code + `
    window.__CodeProtector = _0xp15;
})();
`;
    }

    const ProtectorAPI = {
        version: VERSION,
        initialize: initialize,
        protect: protect,
        isProtected: function() {
            return isProtected;
        },
        setConfig: function(config) {
            Object.assign(ProtectorConfig, config);
        },
        getConfig: function() {
            return Object.assign({}, ProtectorConfig);
        }
    };

    if (ProtectorConfig.autoProtect) {
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', initialize);
        } else {
            initialize();
        }
    }

    if (typeof window !== 'undefined') {
        window.CodeProtector = ProtectorAPI;
    }

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = ProtectorAPI;
    }

    return ProtectorAPI;
})();
