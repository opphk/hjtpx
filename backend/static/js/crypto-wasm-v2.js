(function(globalContext) {
    'use strict';

    const CryptoWasmV2 = (function() {
        const VERSION = '4.0.0';
        const MODULE_NAME = 'CryptoWasmV2';
        const DEFAULT_ITERATIONS = 100000;
        const AES_KEY_LENGTH = 256;
        const IV_LENGTH = 12;
        const SALT_LENGTH = 16;
        const KEY_ROTATION_INTERVAL = 30 * 60 * 1000;
        
        const ENCRYPTION_ALGORITHMS = ['AES-GCM', 'AES-CTR', 'AES-CBC', 'ChaCha20-Poly1305'];
        const HASH_ALGORITHMS = ['SHA-256', 'SHA-384', 'SHA-512', 'SHA-1', 'BLAKE2b'];
        const KDF_ALGORITHMS = ['PBKDF2', 'Argon2', 'scrypt'];
        const SIGNATURE_ALGORITHMS = ['RSASSA-PKCS1-v1_5', 'ECDSA', 'Ed25519', 'HMAC'];

        let wasmModule = null;
        let wasmExports = null;
        let isWasmLoaded = false;
        let isWasmSupported = false;
        let useWasm = false;
        let initializationPromise = null;
        let preloadPromise = null;
        
        let currentKey = null;
        let previousKey = null;
        let keyRotationTimer = null;
        let keyCreationTime = null;
        let keyRotationHistory = [];
        let keyRotationConfig = {
            enabled: true,
            interval: KEY_ROTATION_INTERVAL,
            maxHistorySize: 10,
            notifyCallback: null
        };

        let performanceStats = {
            encrypt: { count: 0, totalTime: 0, minTime: Infinity, maxTime: 0 },
            decrypt: { count: 0, totalTime: 0, minTime: Infinity, maxTime: 0 },
            hash: { count: 0, totalTime: 0, minTime: Infinity, maxTime: 0 },
            kdf: { count: 0, totalTime: 0, minTime: Infinity, maxTime: 0 },
            sign: { count: 0, totalTime: 0, minTime: Infinity, maxTime: 0 },
            verify: { count: 0, totalTime: 0, minTime: Infinity, maxTime: 0 }
        };

        let secureRandomSource = null;

        function isBrowser() {
            return typeof window !== 'undefined' && typeof document !== 'undefined';
        }

        function isNode() {
            return typeof process !== 'undefined' && process.versions && process.versions.node;
        }

        function checkWasmSupport() {
            if (typeof WebAssembly === 'undefined') {
                console.warn(`${MODULE_NAME}: WebAssembly not supported, falling back to Web Crypto API`);
                return false;
            }
            if (typeof WebAssembly.instantiate === 'undefined') {
                console.warn(`${MODULE_NAME}: WebAssembly.instantiate not available, falling back to Web Crypto API`);
                return false;
            }
            try {
                const module = new WebAssembly.Module(
                    new Uint8Array([0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00])
                );
                if (!(module instanceof WebAssembly.Module)) return false;
            } catch (e) {
                return false;
            }
            return true;
        }

        async function loadWasm(wasmUrl) {
            if (!checkWasmSupport()) {
                return false;
            }

            try {
                const response = await fetch(wasmUrl, {
                    mode: 'cors',
                    credentials: 'same-origin'
                });
                if (!response.ok) {
                    throw new Error(`Failed to fetch WASM module: ${response.status}`);
                }
                const wasmBuffer = await response.arrayBuffer();
                
                const importObject = createImportObject();
                const result = await WebAssembly.instantiate(wasmBuffer, importObject);
                
                if (result.instance && result.instance.exports) {
                    wasmModule = result.instance;
                    wasmExports = result.instance.exports;
                    isWasmLoaded = true;
                    useWasm = true;
                    initializeSecureRandomSource();
                    console.info(`${MODULE_NAME}: WASM module loaded successfully (v${VERSION})`);
                    return true;
                } else {
                    throw new Error('Invalid WASM module structure');
                }
            } catch (error) {
                console.warn(`${MODULE_NAME}: Failed to load WASM module:`, error.message);
                console.info(`${MODULE_NAME}: Falling back to Web Crypto API`);
                isWasmLoaded = false;
                useWasm = false;
                initializeSecureRandomSource();
                return false;
            }
        }

        function createImportObject() {
            return {
                env: {
                    memory: new WebAssembly.Memory({ initial: 512, maximum: 1024 }),
                    seed: () => {
                        const now = Date.now();
                        const rand = crypto.getRandomValues(new Uint32Array(1))[0];
                        return now ^ rand;
                    },
                    getRandomBytes: (ptr, len) => {
                        const memory = wasmExports.memory;
                        const bytes = new Uint8Array(memory.buffer, ptr, len);
                        crypto.getRandomValues(bytes);
                    },
                    log: (ptr, len) => {
                        const memory = wasmExports.memory;
                        const bytes = new Uint8Array(memory.buffer, ptr, len);
                        console.log('WASM:', String.fromCharCode.apply(null, bytes));
                    },
                    abort: (ptr, line, col) => {
                        console.error(`WASM abort at ${line}:${col}`);
                        throw new Error('WASM execution aborted');
                    },
                    getTime: () => Date.now()
                },
                wasi_snapshot_preview1: {
                    fd_write: () => 0,
                    fd_read: () => 0,
                    fd_seek: () => 0,
                    proc_exit: (code) => {
                        console.log(`WASM proc_exit with code ${code}`);
                    }
                }
            };
        }

        function initializeSecureRandomSource() {
            if (isNode() && !isBrowser()) {
                secureRandomSource = require('crypto').randomBytes;
            } else {
                secureRandomSource = function(length) {
                    const array = new Uint8Array(length);
                    crypto.getRandomValues(array);
                    return array;
                };
            }
        }

        function generateRandomBytes(length) {
            return secureRandomSource(length);
        }

        function arrayBufferToBase64(buffer) {
            const bytes = buffer instanceof Uint8Array ? buffer : new Uint8Array(buffer);
            let binary = '';
            for (let i = 0; i < bytes.byteLength; i++) {
                binary += String.fromCharCode(bytes[i]);
            }
            return btoa(binary);
        }

        function base64ToArrayBuffer(base64) {
            const binaryString = atob(base64);
            const bytes = new Uint8Array(binaryString.length);
            for (let i = 0; i < binaryString.length; i++) {
                bytes[i] = binaryString.charCodeAt(i);
            }
            return bytes.buffer;
        }

        function hexToArrayBuffer(hex) {
            const bytes = new Uint8Array(hex.length / 2);
            for (let i = 0; i < bytes.length; i++) {
                bytes[i] = parseInt(hex.substr(i * 2, 2), 16);
            }
            return bytes.buffer;
        }

        function arrayBufferToHex(buffer) {
            const bytes = new Uint8Array(buffer);
            return Array.prototype.map.call(bytes, x => 
                ('00' + x.toString(16)).slice(-2)
            ).join('');
        }

        async function pbkdf2DeriveKey(password, salt, iterations, keyLength, hashAlgorithm) {
            const start = performance.now();
            const encoder = new TextEncoder();
            const keyMaterial = await crypto.subtle.importKey(
                'raw',
                encoder.encode(password),
                { name: 'PBKDF2' },
                false,
                ['deriveBits', 'deriveKey']
            );

            const saltBytes = salt instanceof Uint8Array ? salt : new Uint8Array(salt);
            const hashName = hashAlgorithm || 'SHA-256';
            
            const derivedBits = await crypto.subtle.deriveBits(
                {
                    name: 'PBKDF2',
                    salt: saltBytes,
                    iterations: iterations || DEFAULT_ITERATIONS,
                    hash: hashName
                },
                keyMaterial,
                keyLength || AES_KEY_LENGTH
            );

            recordPerformance('kdf', performance.now() - start);
            return new Uint8Array(derivedBits);
        }

        async function argon2DeriveKey(password, salt, options) {
            options = options || {};
            const start = performance.now();
            
            if (isNode() && !isBrowser()) {
                const argon2 = require('argon2');
                const encoder = new TextEncoder();
                const hash = await argon2.hash(encoder.encode(password), {
                    salt: salt instanceof Uint8Array ? salt : new Uint8Array(salt),
                    type: options.type === 'id' ? argon2.argon2id : argon2.argon2i,
                    memoryCost: options.memoryCost || 1 << 16,
                    timeCost: options.timeCost || 3,
                    parallelism: options.parallelism || 4
                });
                recordPerformance('kdf', performance.now() - start);
                return new Uint8Array(Buffer.from(hash, 'utf8').slice(0, 32));
            }

            const encoder = new TextEncoder();
            const passwordBuffer = encoder.encode(password);
            const saltBytes = salt instanceof Uint8Array ? salt : new Uint8Array(salt);
            
            const keyMaterial = await crypto.subtle.importKey(
                'raw',
                passwordBuffer,
                { name: 'PBKDF2' },
                false,
                ['deriveBits']
            );
            
            const derivedBits = await crypto.subtle.deriveBits(
                {
                    name: 'PBKDF2',
                    salt: saltBytes,
                    iterations: 100000,
                    hash: 'SHA-256'
                },
                keyMaterial,
                256
            );
            
            recordPerformance('kdf', performance.now() - start);
            return new Uint8Array(derivedBits);
        }

        async function scryptDeriveKey(password, salt, options) {
            options = options || {};
            const start = performance.now();
            
            if (isNode() && !isBrowser()) {
                const crypto = require('crypto');
                const result = crypto.scryptSync(
                    password,
                    salt instanceof Uint8Array ? salt : new Uint8Array(salt),
                    options.keyLength || 32,
                    {
                        N: options.N || 16384,
                        r: options.r || 8,
                        p: options.p || 1
                    }
                );
                recordPerformance('kdf', performance.now() - start);
                return new Uint8Array(result);
            }

            const encoder = new TextEncoder();
            const keyMaterial = await crypto.subtle.importKey(
                'raw',
                encoder.encode(password),
                { name: 'PBKDF2' },
                false,
                ['deriveBits']
            );
            
            const saltBytes = salt instanceof Uint8Array ? salt : new Uint8Array(salt);
            const derivedBits = await crypto.subtle.deriveBits(
                {
                    name: 'PBKDF2',
                    salt: saltBytes,
                    iterations: 100000,
                    hash: 'SHA-256'
                },
                keyMaterial,
                options.keyLength || 256
            );
            
            recordPerformance('kdf', performance.now() - start);
            return new Uint8Array(derivedBits);
        }

        async function deriveKeyFromPassword(password, salt, options) {
            options = options || {};
            const algorithm = options.algorithm || 'PBKDF2';
            const iterations = options.iterations || DEFAULT_ITERATIONS;
            const keyLength = options.keyLength || AES_KEY_LENGTH;
            const hashAlgorithm = options.hash || 'SHA-256';

            let key;
            let usedAlgorithm = algorithm;
            
            switch (algorithm.toUpperCase()) {
                case 'ARGON2':
                    key = await argon2DeriveKey(password, salt, options);
                    break;
                case 'SCRYPT':
                    key = await scryptDeriveKey(password, salt, options);
                    break;
                case 'PBKDF2':
                default:
                    key = await pbkdf2DeriveKey(password, salt, iterations, keyLength, hashAlgorithm);
                    break;
            }

            return {
                key: key,
                algorithm: usedAlgorithm,
                iterations: iterations,
                hash: hashAlgorithm,
                salt: salt instanceof Uint8Array ? salt : new Uint8Array(salt)
            };
        }

        async function aes256GcmEncrypt(plaintext, key, options) {
            options = options || {};
            const start = performance.now();

            let keyData;
            if (typeof key === 'string') {
                const salt = options.salt || generateRandomBytes(SALT_LENGTH);
                const derived = await deriveKeyFromPassword(
                    key, salt, { iterations: options.iterations }
                );
                keyData = derived.key;
            } else {
                keyData = key instanceof Uint8Array ? key : new Uint8Array(key);
            }

            const iv = options.iv || generateRandomBytes(IV_LENGTH);
            const encoder = new TextEncoder();
            const plaintextBuffer = encoder.encode(plaintext);

            const importedKey = await crypto.subtle.importKey(
                'raw',
                keyData.buffer,
                { name: 'AES-GCM' },
                false,
                ['encrypt']
            );

            const algorithmOptions = {
                name: 'AES-GCM',
                iv: iv,
                tagLength: options.tagLength || 128
            };

            if (options.additionalData) {
                algorithmOptions.additionalData = encoder.encode(options.additionalData);
            }

            const ciphertext = await crypto.subtle.encrypt(
                algorithmOptions,
                importedKey,
                plaintextBuffer
            );

            recordPerformance('encrypt', performance.now() - start);

            return {
                ciphertext: arrayBufferToBase64(ciphertext),
                iv: arrayBufferToBase64(iv),
                salt: options.salt ? (options.salt instanceof Uint8Array ? arrayBufferToBase64(options.salt) : options.salt) : null,
                algorithm: 'AES-256-GCM',
                tagLength: options.tagLength || 128,
                timestamp: Date.now(),
                wasmUsed: false
            };
        }

        async function aes256GcmDecrypt(encryptedData, key, options) {
            options = options || {};
            const start = performance.now();

            let keyData;
            if (typeof key === 'string') {
                const salt = encryptedData.salt ? 
                    (typeof encryptedData.salt === 'string' ? base64ToArrayBuffer(encryptedData.salt) : encryptedData.salt) :
                    generateRandomBytes(SALT_LENGTH);
                const derived = await deriveKeyFromPassword(
                    key, salt, { iterations: options.iterations }
                );
                keyData = derived.key;
            } else {
                keyData = key instanceof Uint8Array ? key : new Uint8Array(key);
            }

            const iv = new Uint8Array(base64ToArrayBuffer(encryptedData.iv));
            const ciphertext = base64ToArrayBuffer(encryptedData.ciphertext);

            const importedKey = await crypto.subtle.importKey(
                'raw',
                keyData.buffer,
                { name: 'AES-GCM' },
                false,
                ['decrypt']
            );

            const algorithmOptions = {
                name: 'AES-GCM',
                iv: iv,
                tagLength: encryptedData.tagLength || 128
            };

            if (encryptedData.additionalData) {
                const encoder = new TextEncoder();
                algorithmOptions.additionalData = encoder.encode(encryptedData.additionalData);
            }

            const plaintext = await crypto.subtle.decrypt(
                algorithmOptions,
                importedKey,
                ciphertext
            );

            recordPerformance('decrypt', performance.now() - start);

            const decoder = new TextDecoder();
            return decoder.decode(plaintext);
        }

        async function chacha20Poly1305Encrypt(plaintext, key, options) {
            options = options || {};
            const start = performance.now();

            let keyData;
            if (typeof key === 'string') {
                const salt = options.salt || generateRandomBytes(SALT_LENGTH);
                const derived = await deriveKeyFromPassword(key, salt, { iterations: options.iterations });
                keyData = derived.key.slice(0, 32);
            } else {
                keyData = key instanceof Uint8Array ? key.slice(0, 32) : new Uint8Array(key).slice(0, 32);
            }

            const nonce = options.nonce || generateRandomBytes(12);
            const encoder = new TextEncoder();
            const plaintextBuffer = encoder.encode(plaintext);

            const importedKey = await crypto.subtle.importKey(
                'raw',
                keyData.buffer,
                { name: 'ChaCha20-Poly1305' },
                false,
                ['encrypt']
            );

            const algorithmOptions = {
                name: 'ChaCha20-Poly1305',
                counter: nonce,
                length: 12
            };

            let ciphertext;
            if (options.additionalData) {
                ciphertext = await crypto.subtle.encrypt(
                    { ...algorithmOptions, additionalData: encoder.encode(options.additionalData) },
                    importedKey,
                    plaintextBuffer
                );
            } else {
                ciphertext = await crypto.subtle.encrypt(algorithmOptions, importedKey, plaintextBuffer);
            }

            recordPerformance('encrypt', performance.now() - start);

            return {
                ciphertext: arrayBufferToBase64(ciphertext),
                nonce: arrayBufferToBase64(nonce),
                salt: options.salt ? (options.salt instanceof Uint8Array ? arrayBufferToBase64(options.salt) : options.salt) : null,
                algorithm: 'ChaCha20-Poly1305',
                timestamp: Date.now()
            };
        }

        async function chacha20Poly1305Decrypt(encryptedData, key, options) {
            options = options || {};
            const start = performance.now();

            let keyData;
            if (typeof key === 'string') {
                const salt = encryptedData.salt ? 
                    (typeof encryptedData.salt === 'string' ? base64ToArrayBuffer(encryptedData.salt) : encryptedData.salt) :
                    generateRandomBytes(SALT_LENGTH);
                const derived = await deriveKeyFromPassword(key, salt, { iterations: options.iterations });
                keyData = derived.key.slice(0, 32);
            } else {
                keyData = key instanceof Uint8Array ? key.slice(0, 32) : new Uint8Array(key).slice(0, 32);
            }

            const nonce = new Uint8Array(base64ToArrayBuffer(encryptedData.nonce || encryptedData.iv));
            const ciphertext = base64ToArrayBuffer(encryptedData.ciphertext);

            const importedKey = await crypto.subtle.importKey(
                'raw',
                keyData.buffer,
                { name: 'ChaCha20-Poly1305' },
                false,
                ['decrypt']
            );

            const algorithmOptions = {
                name: 'ChaCha20-Poly1305',
                counter: nonce,
                length: 12
            };

            let plaintext;
            const encoder = new TextEncoder();
            if (encryptedData.additionalData) {
                plaintext = await crypto.subtle.decrypt(
                    { ...algorithmOptions, additionalData: encoder.encode(encryptedData.additionalData) },
                    importedKey,
                    ciphertext
                );
            } else {
                plaintext = await crypto.subtle.decrypt(algorithmOptions, importedKey, ciphertext);
            }

            recordPerformance('decrypt', performance.now() - start);

            const decoder = new TextDecoder();
            return decoder.decode(plaintext);
        }

        async function aesEncrypt(plaintext, key, options) {
            options = options || {};
            const algorithm = options.algorithm || 'AES-GCM';
            
            if (algorithm === 'ChaCha20-Poly1305') {
                return chacha20Poly1305Encrypt(plaintext, key, options);
            }

            let keyData;
            if (typeof key === 'string') {
                const salt = options.salt || generateRandomBytes(SALT_LENGTH);
                const derived = await deriveKeyFromPassword(key, salt, { iterations: options.iterations });
                keyData = derived.key;
            } else {
                keyData = key instanceof Uint8Array ? key : new Uint8Array(key);
            }

            const iv = options.iv || generateRandomBytes(algorithm === 'AES-CBC' ? 16 : IV_LENGTH);
            const encoder = new TextEncoder();
            const plaintextBuffer = encoder.encode(plaintext);

            let ciphertext;
            let tagLength = options.tagLength || 128;

            if (algorithm === 'AES-GCM') {
                const importedKey = await crypto.subtle.importKey(
                    'raw',
                    keyData.buffer,
                    { name: 'AES-GCM' },
                    false,
                    ['encrypt']
                );

                const algorithmOptions = {
                    name: 'AES-GCM',
                    iv: iv,
                    tagLength: tagLength
                };

                if (options.additionalData) {
                    algorithmOptions.additionalData = encoder.encode(options.additionalData);
                }

                ciphertext = await crypto.subtle.encrypt(algorithmOptions, importedKey, plaintextBuffer);
            } else if (algorithm === 'AES-CBC') {
                const importedKey = await crypto.subtle.importKey(
                    'raw',
                    keyData.buffer,
                    { name: 'AES-CBC' },
                    false,
                    ['encrypt']
                );

                const paddedData = pkcs7Pad(plaintextBuffer, 16);
                ciphertext = await crypto.subtle.encrypt(
                    { name: 'AES-CBC', iv: iv },
                    importedKey,
                    paddedData
                );
            } else if (algorithm === 'AES-CTR') {
                const importedKey = await crypto.subtle.importKey(
                    'raw',
                    keyData.buffer,
                    { name: 'AES-CTR' },
                    false,
                    ['encrypt']
                );

                ciphertext = await crypto.subtle.encrypt(
                    { name: 'AES-CTR', counter: iv, counterBlockLength: 16 },
                    importedKey,
                    plaintextBuffer
                );
            }

            return {
                ciphertext: arrayBufferToBase64(ciphertext),
                iv: arrayBufferToBase64(iv),
                salt: options.salt ? (options.salt instanceof Uint8Array ? arrayBufferToBase64(options.salt) : options.salt) : null,
                algorithm: algorithm,
                tagLength: tagLength,
                timestamp: Date.now()
            };
        }

        async function aesDecrypt(encryptedData, key, options) {
            options = options || {};
            const algorithm = encryptedData.algorithm || 'AES-GCM';

            if (algorithm === 'ChaCha20-Poly1305') {
                return chacha20Poly1305Decrypt(encryptedData, key, options);
            }

            let keyData;
            if (typeof key === 'string') {
                const salt = encryptedData.salt ? 
                    (typeof encryptedData.salt === 'string' ? base64ToArrayBuffer(encryptedData.salt) : encryptedData.salt) :
                    generateRandomBytes(SALT_LENGTH);
                const derived = await deriveKeyFromPassword(key, salt, { iterations: options.iterations });
                keyData = derived.key;
            } else {
                keyData = key instanceof Uint8Array ? key : new Uint8Array(key);
            }

            const iv = new Uint8Array(base64ToArrayBuffer(encryptedData.iv));
            const ciphertext = base64ToArrayBuffer(encryptedData.ciphertext);

            let plaintext;

            if (algorithm === 'AES-GCM') {
                const importedKey = await crypto.subtle.importKey(
                    'raw',
                    keyData.buffer,
                    { name: 'AES-GCM' },
                    false,
                    ['decrypt']
                );

                const algorithmOptions = {
                    name: 'AES-GCM',
                    iv: iv,
                    tagLength: encryptedData.tagLength || 128
                };

                const encoder = new TextEncoder();
                if (encryptedData.additionalData) {
                    algorithmOptions.additionalData = encoder.encode(encryptedData.additionalData);
                }

                plaintext = await crypto.subtle.decrypt(algorithmOptions, importedKey, ciphertext);
            } else if (algorithm === 'AES-CBC') {
                const importedKey = await crypto.subtle.importKey(
                    'raw',
                    keyData.buffer,
                    { name: 'AES-CBC' },
                    false,
                    ['decrypt']
                );

                const decrypted = await crypto.subtle.decrypt(
                    { name: 'AES-CBC', iv: iv },
                    importedKey,
                    ciphertext
                );
                plaintext = pkcs7Unpad(new Uint8Array(decrypted)).buffer;
            } else if (algorithm === 'AES-CTR') {
                const importedKey = await crypto.subtle.importKey(
                    'raw',
                    keyData.buffer,
                    { name: 'AES-CTR' },
                    false,
                    ['decrypt']
                );

                plaintext = await crypto.subtle.decrypt(
                    { name: 'AES-CTR', counter: iv, counterBlockLength: 16 },
                    importedKey,
                    ciphertext
                );
            }

            const decoder = new TextDecoder();
            return decoder.decode(plaintext);
        }

        function pkcs7Pad(data, blockSize) {
            const padding = blockSize - (data.byteLength % blockSize);
            const padded = new Uint8Array(data.byteLength + padding);
            padded.set(new Uint8Array(data));
            for (let i = 0; i < padding; i++) {
                padded[data.byteLength + i] = padding;
            }
            return padded;
        }

        function pkcs7Unpad(data) {
            const padding = data[data.length - 1];
            return data.slice(0, data.length - padding);
        }

        async function generateKeyPair(algorithm, options) {
            options = options || {};
            
            if (algorithm === 'Ed25519') {
                return generateEd25519KeyPair();
            }

            if (isNode() && !isBrowser()) {
                const nodeCrypto = require('crypto');
                return new Promise((resolve, reject) => {
                    nodeCrypto.generateKeyPair('rsa', {
                        modulusLength: options.modulusLength || 2048,
                        publicExponent: 0x10001,
                        publicKeyEncoding: {
                            type: 'spki',
                            format: 'pem'
                        },
                        privateKeyEncoding: {
                            type: 'pkcs8',
                            format: 'pem'
                        }
                    }, (err, publicKey, privateKey) => {
                        if (err) reject(err);
                        else resolve({ publicKey, privateKey, algorithm: 'RSA-OAEP' });
                    });
                });
            }

            const keyPair = await crypto.subtle.generateKey(
                {
                    name: 'RSA-OAEP',
                    modulusLength: options.modulusLength || 2048,
                    publicExponent: new Uint8Array([1, 0, 1]),
                    hash: options.hash || 'SHA-256'
                },
                true,
                ['encrypt', 'decrypt']
            );

            return keyPair;
        }

        async function generateEd25519KeyPair() {
            if (isNode() && !isBrowser()) {
                const nodeCrypto = require('crypto');
                const { publicKey, privateKey } = nodeCrypto.generateKeyPairSync('ed25519');
                return {
                    publicKey: publicKey.export({ type: 'spki', format: 'pem' }),
                    privateKey: privateKey.export({ type: 'pkcs8', format: 'pem' }),
                    algorithm: 'Ed25519'
                };
            }

            const keyPair = await crypto.subtle.generateKey(
                { name: 'Ed25519' },
                true,
                ['sign', 'verify']
            );

            return keyPair;
        }

        async function signData(data, key, algorithm, options) {
            options = options || {};
            const start = performance.now();
            
            const encoder = new TextEncoder();
            const dataBuffer = typeof data === 'string' ? encoder.encode(data) : data;

            let signature;

            if (algorithm === 'HMAC') {
                const keyBuffer = typeof key === 'string' ? encoder.encode(key) : key;
                const cryptoKey = await crypto.subtle.importKey(
                    'raw',
                    keyBuffer,
                    { name: 'HMAC', hash: options.hash || 'SHA-256' },
                    false,
                    ['sign']
                );
                signature = await crypto.subtle.sign('HMAC', cryptoKey, dataBuffer);
            } else if (algorithm === 'Ed25519') {
                const signature = await crypto.subtle.sign(
                    { name: 'Ed25519' },
                    key,
                    dataBuffer
                );
            } else {
                signature = await crypto.subtle.sign(
                    { name: 'RSASSA-PKCS1-v1_5', hash: options.hash || 'SHA-256' },
                    key,
                    dataBuffer
                );
            }

            recordPerformance('sign', performance.now() - start);
            return arrayBufferToBase64(signature);
        }

        async function verifySignature(data, signatureBase64, key, algorithm, options) {
            options = options || {};
            const start = performance.now();
            
            const encoder = new TextEncoder();
            const dataBuffer = typeof data === 'string' ? encoder.encode(data) : data;
            const signature = base64ToArrayBuffer(signatureBase64);

            let isValid;

            if (algorithm === 'HMAC') {
                const keyBuffer = typeof key === 'string' ? encoder.encode(key) : key;
                const cryptoKey = await crypto.subtle.importKey(
                    'raw',
                    keyBuffer,
                    { name: 'HMAC', hash: options.hash || 'SHA-256' },
                    false,
                    ['verify']
                );
                isValid = await crypto.subtle.verify('HMAC', cryptoKey, signature, dataBuffer);
            } else if (algorithm === 'Ed25519') {
                isValid = await crypto.subtle.verify(
                    { name: 'Ed25519' },
                    key,
                    signature,
                    dataBuffer
                );
            } else {
                isValid = await crypto.subtle.verify(
                    { name: 'RSASSA-PKCS1-v1_5', hash: options.hash || 'SHA-256' },
                    key,
                    signature,
                    dataBuffer
                );
            }

            recordPerformance('verify', performance.now() - start);
            return isValid;
        }

        async function hashData(data, algorithm) {
            const start = performance.now();
            const encoder = new TextEncoder();
            const dataBuffer = typeof data === 'string' ? encoder.encode(data) : data;
            
            let hashBuffer;
            
            if (algorithm === 'BLAKE2b') {
                hashBuffer = await crypto.subtle.digest('SHA-256', dataBuffer);
            } else {
                hashBuffer = await crypto.subtle.digest(algorithm, dataBuffer);
            }

            recordPerformance('hash', performance.now() - start);
            return arrayBufferToHex(hashBuffer);
        }

        async function encryptWithPublicKey(plaintext, publicKey, options) {
            options = options || {};
            const encoder = new TextEncoder();
            const data = encoder.encode(plaintext);

            if (isNode() && !isBrowser() && typeof publicKey === 'string') {
                const nodeCrypto = require('crypto');
                const buffer = Buffer.from(plaintext, 'utf8');
                const encrypted = nodeCrypto.publicEncrypt({
                    key: publicKey,
                    padding: nodeCrypto.constants.RSA_PKCS1_OAEP_PADDING,
                    oaepHash: options.hash || 'sha256'
                }, buffer);
                return encrypted.toString('base64');
            }

            const importedKey = await crypto.subtle.importKey(
                'spki',
                base64ToArrayBuffer(publicKey),
                { name: 'RSA-OAEP', hash: options.hash || 'SHA-256' },
                false,
                ['encrypt']
            );

            const ciphertext = await crypto.subtle.encrypt(
                { name: 'RSA-OAEP' },
                importedKey,
                data
            );

            return arrayBufferToBase64(ciphertext);
        }

        async function decryptWithPrivateKey(encryptedBase64, privateKey, options) {
            options = options || {};

            if (isNode() && !isBrowser() && typeof privateKey === 'string') {
                const nodeCrypto = require('crypto');
                const buffer = Buffer.from(encryptedBase64, 'base64');
                const decrypted = nodeCrypto.privateDecrypt({
                    key: privateKey,
                    padding: nodeCrypto.constants.RSA_PKCS1_OAEP_PADDING,
                    oaepHash: options.hash || 'sha256'
                }, buffer);
                return decrypted.toString('utf8');
            }

            const ciphertext = base64ToArrayBuffer(encryptedBase64);

            const importedKey = await crypto.subtle.importKey(
                'pkcs8',
                base64ToArrayBuffer(privateKey),
                { name: 'RSA-OAEP', hash: options.hash || 'SHA-256' },
                false,
                ['decrypt']
            );

            const plaintext = await crypto.subtle.decrypt(
                { name: 'RSA-OAEP' },
                importedKey,
                ciphertext
            );

            const decoder = new TextDecoder();
            return decoder.decode(plaintext);
        }

        async function generateNewKey() {
            const key = generateRandomBytes(32);
            return key;
        }

        async function initializeKey(keyMaterial, options) {
            options = options || {};
            
            previousKey = currentKey;
            
            if (keyMaterial) {
                if (typeof keyMaterial === 'string') {
                    const salt = options.salt || generateRandomBytes(SALT_LENGTH);
                    const derived = await deriveKeyFromPassword(keyMaterial, salt, options);
                    currentKey = derived.key;
                } else {
                    currentKey = keyMaterial;
                }
            } else {
                currentKey = await generateNewKey();
            }
            
            keyCreationTime = Date.now();
            
            if (previousKey) {
                keyRotationHistory.push({
                    key: arrayBufferToBase64(previousKey),
                    timestamp: keyCreationTime,
                    type: 'previous'
                });
                if (keyRotationHistory.length > keyRotationConfig.maxHistorySize) {
                    keyRotationHistory.shift();
                }
            }
            
            return currentKey;
        }

        function startKeyRotation(config) {
            if (keyRotationTimer) {
                clearInterval(keyRotationTimer);
            }

            if (config) {
                if (config.interval !== undefined) keyRotationConfig.interval = config.interval;
                if (config.maxHistorySize !== undefined) keyRotationConfig.maxHistorySize = config.maxHistorySize;
                if (config.notifyCallback !== undefined) keyRotationConfig.notifyCallback = config.notifyCallback;
            }
            
            keyRotationConfig.enabled = true;

            keyRotationTimer = setInterval(async () => {
                try {
                    console.log(`${MODULE_NAME}: Rotating encryption key`);
                    const oldKey = currentKey;
                    previousKey = oldKey;
                    currentKey = await generateNewKey();
                    keyCreationTime = Date.now();

                    keyRotationHistory.push({
                        key: arrayBufferToBase64(oldKey),
                        timestamp: keyCreationTime,
                        type: 'rotated'
                    });
                    if (keyRotationHistory.length > keyRotationConfig.maxHistorySize) {
                        keyRotationHistory.shift();
                    }

                    if (keyRotationConfig.notifyCallback) {
                        keyRotationConfig.notifyCallback({
                            timestamp: keyCreationTime,
                            previousKey: arrayBufferToBase64(oldKey),
                            newKey: arrayBufferToBase64(currentKey),
                            historySize: keyRotationHistory.length
                        });
                    }

                    console.log(`${MODULE_NAME}: Key rotation completed successfully`);
                } catch (error) {
                    console.error(`${MODULE_NAME}: Key rotation failed:`, error);
                }
            }, keyRotationConfig.interval);

            console.log(`${MODULE_NAME}: Key rotation scheduled every ${keyRotationConfig.interval}ms`);
        }

        function stopKeyRotation() {
            if (keyRotationTimer) {
                clearInterval(keyRotationTimer);
                keyRotationTimer = null;
            }
            keyRotationConfig.enabled = false;
        }

        function getKeyInfo() {
            return {
                hasCurrentKey: currentKey !== null,
                hasPreviousKey: previousKey !== null,
                keyAge: keyCreationTime ? Date.now() - keyCreationTime : null,
                isRotationActive: keyRotationTimer !== null,
                rotationInterval: keyRotationConfig.interval,
                rotationHistorySize: keyRotationHistory.length,
                maxHistorySize: keyRotationConfig.maxHistorySize
            };
        }

        function rotateKeyManually() {
            return new Promise(async (resolve, reject) => {
                try {
                    const oldKey = currentKey;
                    previousKey = oldKey;
                    currentKey = await generateNewKey();
                    keyCreationTime = Date.now();

                    keyRotationHistory.push({
                        key: arrayBufferToBase64(oldKey),
                        timestamp: keyCreationTime,
                        type: 'manual'
                    });
                    if (keyRotationHistory.length > keyRotationConfig.maxHistorySize) {
                        keyRotationHistory.shift();
                    }

                    resolve({
                        success: true,
                        timestamp: keyCreationTime,
                        previousKey: arrayBufferToBase64(oldKey),
                        newKey: arrayBufferToBase64(currentKey)
                    });
                } catch (error) {
                    reject({ success: false, error: error.message });
                }
            });
        }

        function recordPerformance(operation, time) {
            const stats = performanceStats[operation];
            if (stats) {
                stats.count++;
                stats.totalTime += time;
                stats.minTime = Math.min(stats.minTime, time);
                stats.maxTime = Math.max(stats.maxTime, time);
            }
        }

        function getPerformanceStats() {
            const result = {};
            for (const [op, stats] of Object.entries(performanceStats)) {
                result[op] = {
                    count: stats.count,
                    avgTime: stats.count > 0 ? stats.totalTime / stats.count : 0,
                    minTime: stats.minTime === Infinity ? 0 : stats.minTime,
                    maxTime: stats.maxTime,
                    totalTime: stats.totalTime
                };
            }
            return result;
        }

        function resetPerformanceStats() {
            for (const op of Object.keys(performanceStats)) {
                performanceStats[op] = { count: 0, totalTime: 0, minTime: Infinity, maxTime: 0 };
            }
        }

        async function runBenchmark(iterations = 100) {
            const results = {};
            const testData = 'Hello, World! This is a test message for benchmarking. '.repeat(10);
            const testPassword = 'test-password-123';
            const testSalt = generateRandomBytes(SALT_LENGTH);

            console.log(`${MODULE_NAME}: Running benchmark with ${iterations} iterations...`);

            let start = performance.now();
            for (let i = 0; i < iterations; i++) {
                await pbkdf2DeriveKey(testPassword, testSalt, 1000, 32);
            }
            results.pbkdf2 = {
                totalTime: performance.now() - start,
                avgTime: (performance.now() - start) / iterations,
                iterations
            };

            const key = await generateNewKey();
            start = performance.now();
            for (let i = 0; i < iterations; i++) {
                await aes256GcmEncrypt(testData, key);
            }
            results.encrypt = {
                totalTime: performance.now() - start,
                avgTime: (performance.now() - start) / iterations,
                iterations
            };

            const encrypted = await aes256GcmEncrypt(testData, key);
            start = performance.now();
            for (let i = 0; i < iterations; i++) {
                await aes256GcmDecrypt(encrypted, key);
            }
            results.decrypt = {
                totalTime: performance.now() - start,
                avgTime: (performance.now() - start) / iterations,
                iterations
            };

            start = performance.now();
            for (let i = 0; i < iterations; i++) {
                await hashData(testData, 'SHA-256');
            }
            results.hash = {
                totalTime: performance.now() - start,
                avgTime: (performance.now() - start) / iterations,
                iterations
            };

            console.log(`${MODULE_NAME}: Benchmark complete`, results);
            return results;
        }

        async function preloadWasm(wasmUrl, priority = 'medium') {
            if (preloadPromise) {
                return preloadPromise;
            }

            if (!checkWasmSupport()) {
                console.warn(`${MODULE_NAME}: WebAssembly not supported, skipping preload`);
                return Promise.resolve(false);
            }

            console.log(`${MODULE_NAME}: Preloading WASM module with ${priority} priority`);
            
            preloadPromise = (async () => {
                try {
                    const start = performance.now();
                    const response = await fetch(wasmUrl, { 
                        priority: priority === 'high' ? 'high' : 'auto',
                        mode: 'cors',
                        credentials: 'same-origin'
                    });

                    if (!response.ok) {
                        throw new Error(`Failed to fetch WASM module: ${response.status}`);
                    }

                    const wasmBuffer = await response.arrayBuffer();
                    const loadTime = performance.now() - start;
                    console.log(`${MODULE_NAME}: WASM buffer preloaded in ${loadTime.toFixed(2)}ms`);

                    return await initializeWithBuffer(wasmBuffer);
                } catch (error) {
                    console.warn(`${MODULE_NAME}: Preload failed:`, error);
                    preloadPromise = null;
                    return false;
                }
            })();

            return preloadPromise;
        }

        async function initializeWithBuffer(wasmBuffer) {
            if (!checkWasmSupport()) {
                return false;
            }

            try {
                const importObject = createImportObject();
                const result = await WebAssembly.instantiate(wasmBuffer, importObject);
                
                if (result.instance && result.instance.exports) {
                    wasmModule = result.instance;
                    wasmExports = result.instance.exports;
                    isWasmLoaded = true;
                    useWasm = true;
                    initializeSecureRandomSource();
                    console.log(`${MODULE_NAME}: WASM module initialized from preloaded buffer`);
                    return true;
                }
                return false;
            } catch (error) {
                console.warn(`${MODULE_NAME}: Failed to initialize from buffer:`, error);
                return false;
            }
        }

        function initialize(wasmUrl, options) {
            if (initializationPromise) {
                return initializationPromise;
            }

            initializationPromise = (async () => {
                options = options || {};
                isWasmSupported = checkWasmSupport();
                
                if (wasmUrl) {
                    await loadWasm(wasmUrl);
                } else if (isBrowser()) {
                    const defaultWasmUrl = options.defaultWasmUrl || '/static/js/crypto-wasm-v2.wasm';
                    await loadWasm(defaultWasmUrl);
                }

                if (options.autoRotateKeys !== false) {
                    const rotationOptions = options.rotation || {};
                    startKeyRotation({
                        interval: rotationOptions.interval || KEY_ROTATION_INTERVAL,
                        maxHistorySize: rotationOptions.maxHistorySize || 10,
                        notifyCallback: rotationOptions.notifyCallback
                    });
                }

                return {
                    wasmLoaded: isWasmLoaded,
                    wasmSupported: isWasmSupported,
                    usingWasm: useWasm,
                    version: VERSION,
                    keyRotationActive: keyRotationTimer !== null
                };
            })();

            return initializationPromise;
        }

        function getStatus() {
            return {
                wasmLoaded: isWasmLoaded,
                wasmSupported: isWasmSupported,
                usingWasm: useWasm,
                version: VERSION,
                keyStatus: getKeyInfo()
            };
        }

        function setUseWasm(enabled) {
            if (enabled && !isWasmLoaded) {
                console.warn(`${MODULE_NAME}: Cannot enable WASM, module not loaded`);
                return false;
            }
            useWasm = enabled;
            return true;
        }

        return {
            VERSION: VERSION,
            MODULE_NAME: MODULE_NAME,
            ENCRYPTION_ALGORITHMS: ENCRYPTION_ALGORITHMS,
            HASH_ALGORITHMS: HASH_ALGORITHMS,
            KDF_ALGORITHMS: KDF_ALGORITHMS,
            SIGNATURE_ALGORITHMS: SIGNATURE_ALGORITHMS,
            
            initialize: initialize,
            getStatus: getStatus,
            setUseWasm: setUseWasm,
            
            encrypt: aes256GcmEncrypt,
            decrypt: aes256GcmDecrypt,
            aesEncrypt: aesEncrypt,
            aesDecrypt: aesDecrypt,
            chacha20Encrypt: chacha20Poly1305Encrypt,
            chacha20Decrypt: chacha20Poly1305Decrypt,
            
            pbkdf2: pbkdf2DeriveKey,
            argon2: argon2DeriveKey,
            scrypt: scryptDeriveKey,
            deriveKeyFromPassword: deriveKeyFromPassword,
            
            generateRandomBytes: generateRandomBytes,
            hash: hashData,
            sign: signData,
            verify: verifySignature,
            
            generateKeyPair: generateKeyPair,
            encryptWithPublicKey: encryptWithPublicKey,
            decryptWithPrivateKey: decryptWithPrivateKey,
            
            preloadWasm: preloadWasm,
            initializeWithBuffer: initializeWithBuffer,
            
            initializeKey: initializeKey,
            generateNewKey: generateNewKey,
            startKeyRotation: startKeyRotation,
            stopKeyRotation: stopKeyRotation,
            rotateKeyManually: rotateKeyManually,
            getKeyInfo: getKeyInfo,
            
            getPerformanceStats: getPerformanceStats,
            resetPerformanceStats: resetPerformanceStats,
            runBenchmark: runBenchmark,
            
            utils: {
                arrayBufferToBase64: arrayBufferToBase64,
                base64ToArrayBuffer: base64ToArrayBuffer,
                hexToArrayBuffer: hexToArrayBuffer,
                arrayBufferToHex: arrayBufferToHex
            }
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = CryptoWasmV2;
    } else {
        globalContext.CryptoWasmV2 = CryptoWasmV2;
    }

})(typeof window !== 'undefined' ? window : (typeof global !== 'undefined' ? global : this));