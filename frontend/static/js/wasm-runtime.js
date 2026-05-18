const WASMRuntime = (function() {
    'use strict';

    const VERSION = '15.0.0';

    const WASMConfig = {
        enableCaching: true,
        maxCacheSize: 10,
        useWorker: false,
        fallbackEnabled: true
    };

    let wasmModule = null;
    let wasmMemory = null;
    let isInitialized = false;
    let cache = new Map();

    function log(message) {
        if (typeof console !== 'undefined' && console.debug) {
            console.debug('[WASM Runtime ' + VERSION + ']:', message);
        }
    }

    function error(message) {
        if (typeof console !== 'undefined' && console.error) {
            console.error('[WASM Runtime ' + VERSION + ' Error]:', message);
        }
    }

    async function initWASM(wasmBinary) {
        if (isInitialized && wasmModule) {
            return wasmModule;
        }

        if (!wasmBinary) {
            error('WASM binary not provided');
            return null;
        }

        try {
            if (typeof WebAssembly !== 'undefined') {
                const imports = {
                    env: {
                        abort: function(msg, file, line, column) {
                            error('WASM abort: ' + msg + ' at ' + file + ':' + line + ':' + column);
                        },
                        seed: function() {
                            return Date.now() % 1000000;
                        },
                        clock: function() {
                            return Date.now();
                        }
                    }
                };

                wasmModule = await WebAssembly.instantiate(wasmBinary, imports);
                wasmMemory = wasmModule.instance.exports.memory;
                isInitialized = true;
                log('WASM module initialized successfully');

                return wasmModule;
            } else {
                error('WebAssembly not supported');
                return null;
            }
        } catch (e) {
            error('Failed to initialize WASM: ' + e.message);
            return null;
        }
    }

    function decryptBuffer(buffer, key) {
        if (!isInitialized || !wasmMemory) {
            error('WASM not initialized');
            return null;
        }

        try {
            const inputPtr = wasmMemory.buffer.byteLength;
            const outputPtr = inputPtr + buffer.length;

            const inputView = new Uint8Array(wasmMemory.buffer, inputPtr, buffer.length);
            inputView.set(new Uint8Array(buffer));

            if (wasmModule.instance.exports.decrypt) {
                wasmModule.instance.exports.decrypt(inputPtr, buffer.length, outputPtr);
                const result = new Uint8Array(wasmMemory.buffer, outputPtr, buffer.length);
                return result.buffer;
            }

            const decrypted = new Uint8Array(buffer.length);
            for (let i = 0; i < buffer.length; i++) {
                const keyByte = typeof key === 'string' ?
                    key.charCodeAt(i % key.length) :
                    key[i % key.length];
                decrypted[i] = buffer[i] ^ keyByte ^ ((i * 7 + 13) % 256);
            }

            return decrypted.buffer;
        } catch (e) {
            error('Decryption failed: ' + e.message);
            return null;
        }
    }

    function encryptBuffer(buffer, key) {
        if (!isInitialized || !wasmMemory) {
            error('WASM not initialized');
            return null;
        }

        try {
            const encrypted = new Uint8Array(buffer.length);
            for (let i = 0; i < buffer.length; i++) {
                const keyByte = typeof key === 'string' ?
                    key.charCodeAt(i % key.length) :
                    key[i % key.length];
                encrypted[i] = buffer[i] ^ keyByte ^ ((i * 7 + 13) % 256);
            }
            return encrypted.buffer;
        } catch (e) {
            error('Encryption failed: ' + e.message);
            return null;
        }
    }

    function hashData(data) {
        if (typeof data === 'string') {
            data = new TextEncoder().encode(data);
        }

        if (WASMConfig.enableCaching) {
            const cacheKey = arrayToString(data);
            if (cache.has(cacheKey)) {
                return cache.get(cacheKey);
            }
        }

        let hash = 0;
        for (let i = 0; i < data.length; i++) {
            const char = data[i];
            hash = ((hash << 5) - hash) + char;
            hash = hash & hash;
        }

        const result = Math.abs(hash).toString(16);

        if (WASMConfig.enableCaching) {
            if (cache.size >= WASMConfig.maxCacheSize) {
                const firstKey = cache.keys().next().value;
                cache.delete(firstKey);
            }
            cache.set(arrayToString(data), result);
        }

        return result;
    }

    function arrayToString(arr) {
        if (arr instanceof ArrayBuffer) {
            arr = new Uint8Array(arr);
        }
        return Array.from(arr).join(',');
    }

    async function loadWASMFromURL(url) {
        try {
            const response = await fetch(url);
            if (!response.ok) {
                throw new Error('Failed to fetch WASM: ' + response.status);
            }
            const buffer = await response.arrayBuffer();
            return await initWASM(buffer);
        } catch (e) {
            error('Failed to load WASM from URL: ' + e.message);
            return null;
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

    function xorEncrypt(data, key) {
        if (typeof data === 'string') {
            data = new TextEncoder().encode(data);
        }

        if (typeof key === 'string') {
            key = new TextEncoder().encode(key);
        }

        const encrypted = new Uint8Array(data.length);
        for (let i = 0; i < data.length; i++) {
            const keyByte = key[i % key.length];
            const dataByte = data[i];
            const xorByte = dataByte ^ keyByte;
            const offset = (i * 7 + 13) % 256;
            encrypted[i] = (xorByte + offset) % 256;
        }

        return encrypted;
    }

    function xorDecrypt(data, key) {
        if (typeof key === 'string') {
            key = new TextEncoder().encode(key);
        }

        const decrypted = new Uint8Array(data.length);
        for (let i = 0; i < data.length; i++) {
            const keyByte = key[i % key.length];
            const dataByte = data[i];
            const offset = (i * 7 + 13) % 256;
            const xorByte = (dataByte - offset + 256) % 256;
            decrypted[i] = xorByte ^ keyByte;
        }

        return decrypted;
    }

    function createCodeVirtualizationContext() {
        return {
            registers: new Array(16).fill(0),
            stack: [],
            pc: 0,
            running: false,

            execute: function(bytecode) {
                this.running = true;
                this.pc = 0;

                while (this.running && this.pc < bytecode.length) {
                    const op = bytecode[this.pc];
                    this.pc++;

                    switch (op) {
                        case 0x01:
                            this.stack.push(this.registers[bytecode[this.pc++]]);
                            break;
                        case 0x02:
                            this.registers[bytecode[this.pc++]] = this.stack.pop();
                            break;
                        case 0x03:
                            const a = this.stack.pop();
                            const b = this.stack.pop();
                            this.stack.push(b + a);
                            break;
                        case 0x04:
                            const c = this.stack.pop();
                            const d = this.stack.pop();
                            this.stack.push(d - c);
                            break;
                        case 0x05:
                            const e = this.stack.pop();
                            const f = this.stack.pop();
                            this.stack.push(e * f);
                            break;
                        case 0x06:
                            const g = this.stack.pop();
                            const h = this.stack.pop();
                            this.stack.push(Math.floor(h / g));
                            break;
                        case 0x07:
                            this.running = false;
                            break;
                        default:
                            break;
                    }
                }

                return this.stack.pop() || 0;
            },

            reset: function() {
                this.registers.fill(0);
                this.stack = [];
                this.pc = 0;
                this.running = false;
            }
        };
    }

    const WASMAPI = {
        version: VERSION,
        isInitialized: function() {
            return isInitialized;
        },

        init: initWASM,

        initFromURL: loadWASMFromURL,

        decrypt: decryptBuffer,

        encrypt: encryptBuffer,

        hash: hashData,

        getCacheSize: function() {
            return cache.size;
        },

        clearCache: function() {
            cache.clear();
        },

        generateRandomBytes: generateRandomBytes,

        xorEncrypt: xorEncrypt,

        xorDecrypt: xorDecrypt,

        createVM: createCodeVirtualizationContext,

        setConfig: function(config) {
            Object.assign(WASMConfig, config);
        },

        getConfig: function() {
            return Object.assign({}, WASMConfig);
        }
    };

    if (typeof window !== 'undefined') {
        window.WASMRuntime = WASMAPI;
        window._0xw = WASMAPI;
    }

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = WASMAPI;
    }

    return WASMAPI;
})();

const RuntimeDecryptor = (function() {
    'use strict';

    const VERSION = '15.0.0';

    let decryptionKey = null;
    let initialized = false;

    function initialize(key) {
        if (!key) {
            const randomBytes = new Uint8Array(32);
            if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
                crypto.getRandomValues(randomBytes);
            } else {
                for (let i = 0; i < 32; i++) {
                    randomBytes[i] = Math.floor(Math.random() * 256);
                }
            }
            key = Array.from(randomBytes).map(function(b) {
                return b.toString(16).padStart(2, '0');
            }).join('');
        }

        decryptionKey = key;
        initialized = true;
    }

    function decryptString(encrypted) {
        if (!initialized) {
            initialize();
        }

        try {
            const decoded = atob(encrypted);
            const encryptedBytes = new Uint8Array(decoded.length);
            for (let i = 0; i < decoded.length; i++) {
                encryptedBytes[i] = decoded.charCodeAt(i);
            }

            const decrypted = new Uint8Array(encryptedBytes.length);
            for (let i = 0; i < encryptedBytes.length; i++) {
                const keyByte = decryptionKey.charCodeAt(i % decryptionKey.length);
                const offset = (i * 7 + 13) % 256;
                decrypted[i] = ((encryptedBytes[i] - offset + 256) % 256) ^ keyByte;
            }

            return String.fromCharCode.apply(null, decrypted);
        } catch (e) {
            return null;
        }
    }

    function encryptString(plaintext) {
        if (!initialized) {
            initialize();
        }

        try {
            const plaintextBytes = new Uint8Array(plaintext.length);
            for (let i = 0; i < plaintext.length; i++) {
                plaintextBytes[i] = plaintext.charCodeAt(i);
            }

            const encrypted = new Uint8Array(plaintextBytes.length);
            for (let i = 0; i < plaintextBytes.length; i++) {
                const keyByte = decryptionKey.charCodeAt(i % decryptionKey.length);
                const offset = (i * 7 + 13) % 256;
                encrypted[i] = ((plaintextBytes[i] ^ keyByte) + offset) % 256;
            }

            let binary = '';
            for (let i = 0; i < encrypted.length; i++) {
                binary += String.fromCharCode(encrypted[i]);
            }

            return btoa(binary);
        } catch (e) {
            return null;
        }
    }

    function decryptScript(encryptedScript) {
        const decrypted = decryptString(encryptedScript);
        if (decrypted) {
            try {
                return new Function(decrypted);
            } catch (e) {
                return null;
            }
        }
        return null;
    }

    const DecryptorAPI = {
        version: VERSION,
        init: initialize,
        decrypt: decryptString,
        encrypt: encryptString,
        decryptScript: decryptScript,
        isInitialized: function() {
            return initialized;
        }
    };

    if (typeof window !== 'undefined') {
        window.RuntimeDecryptor = DecryptorAPI;
        window._0xrd = DecryptorAPI;
    }

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = DecryptorAPI;
    }

    return DecryptorAPI;
})();
