(function(globalContext) {
    'use strict';

    const CryptoWasm = (function() {
        const VERSION = '3.0.0';
    const MODULE_NAME = 'CryptoWasm';
    const DEFAULT_ITERATIONS = 100000;
    const AES_KEY_LENGTH = 256;
    const IV_LENGTH = 12;
    const SALT_LENGTH = 16;
    const KEY_ROTATION_INTERVAL = 30 * 60 * 1000;
    const PRELOAD_PRIORITY = ['high', 'medium', 'low'];
    const ENCRYPTION_ALGORITHMS = ['AES-GCM', 'AES-CTR', 'AES-CBC'];
    const HASH_ALGORITHMS = ['SHA-256', 'SHA-384', 'SHA-512', 'SHA-1'];

        let wasmModule = null;
        let wasmExports = null;
        let isWasmLoaded = false;
        let isWasmSupported = false;
        let useWasm = false;
        let initializationPromise = null;
        let preloadPromise = null;
        
        // Key rotation management
        let currentKey = null;
        let previousKey = null;
        let keyRotationTimer = null;
        let keyCreationTime = null;
        
        // Performance benchmarking
        let performanceStats = {
            encrypt: { count: 0, totalTime: 0, minTime: Infinity, maxTime: 0 },
            decrypt: { count: 0, totalTime: 0, minTime: Infinity, maxTime: 0 },
            hash: { count: 0, totalTime: 0, minTime: Infinity, maxTime: 0 },
            pbkdf2: { count: 0, totalTime: 0, minTime: Infinity, maxTime: 0 }
        };

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
            return true;
        }

        async function loadWasm(wasmUrl) {
            if (!checkWasmSupport()) {
                return false;
            }

            try {
                const response = await fetch(wasmUrl);
                if (!response.ok) {
                    throw new Error(`Failed to fetch WASM module: ${response.status}`);
                }
                const wasmBuffer = await response.arrayBuffer();
                
                const importObject = {
                    env: {
                        memory: new WebAssembly.Memory({ initial: 256, maximum: 512 }),
                        seed: () => Date.now() ^ (Math.random() * 0xFFFFFFFF),
                        log: (ptr, len) => {
                            const memory = wasmExports.memory;
                            const bytes = new Uint8Array(memory.buffer, ptr, len);
                            console.log('WASM:', String.fromCharCode.apply(null, bytes));
                        },
                        abort: (ptr, line, col) => {
                            console.error(`WASM abort at ${line}:${col}`);
                            throw new Error('WASM execution aborted');
                        }
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

                const result = await WebAssembly.instantiate(wasmBuffer, importObject);
                
                if (result.instance && result.instance.exports) {
                    wasmModule = result.instance;
                    wasmExports = result.instance.exports;
                    isWasmLoaded = true;
                    useWasm = true;
                    console.info(`${MODULE_NAME}: WASM module loaded successfully`);
                    return true;
                } else {
                    throw new Error('Invalid WASM module structure');
                }
            } catch (error) {
                console.warn(`${MODULE_NAME}: Failed to load WASM module:`, error.message);
                console.info(`${MODULE_NAME}: Falling back to Web Crypto API`);
                isWasmLoaded = false;
                useWasm = false;
                return false;
            }
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

        function generateRandomBytes(length) {
            const array = new Uint8Array(length);
            if (isNode() && !isBrowser()) {
                const nodeCrypto = require('crypto');
                nodeCrypto.randomFillSync(array);
            } else {
                crypto.getRandomValues(array);
            }
            return array;
        }

        async function pbkdf2DeriveKey(password, salt, iterations, keyLength) {
            if (useWasm && isWasmLoaded && wasmExports && wasmExports.pbkdf2) {
                try {
                    const encoder = new TextEncoder();
                    const passwordBuffer = encoder.encode(password);
                    const saltBuffer = salt instanceof Uint8Array ? salt : new Uint8Array(salt);
                    
                    const passwordPtr = allocateBuffer(passwordBuffer);
                    const saltPtr = allocateBuffer(saltBuffer);
                    const keyPtr = allocateBuffer(keyLength / 8);
                    
                    const result = wasmExports.pbkdf2(
                        passwordPtr, passwordBuffer.length,
                        saltPtr, saltBuffer.length,
                        iterations || DEFAULT_ITERATIONS,
                        keyPtr, keyLength / 8
                    );
                    
                    if (result === 0) {
                        const keyBuffer = new Uint8Array(wasmExports.memory.buffer, keyPtr, keyLength / 8);
                        return new Uint8Array(keyBuffer);
                    } else {
                        throw new Error('WASM PBKDF2 failed');
                    }
                } catch (error) {
                    console.warn(`${MODULE_NAME}: WASM PBKDF2 failed, falling back to Web Crypto API`);
                }
            }

            const encoder = new TextEncoder();
            const keyMaterial = await crypto.subtle.importKey(
                'raw',
                encoder.encode(password),
                { name: 'PBKDF2' },
                false,
                ['deriveBits', 'deriveKey']
            );

            const saltBytes = salt instanceof Uint8Array ? salt : new Uint8Array(salt);
            
            const iterationsToUse = iterations || DEFAULT_ITERATIONS;
            const hashName = 'SHA-256';
            
            const derivedBits = await crypto.subtle.deriveBits(
                {
                    name: 'PBKDF2',
                    salt: saltBytes,
                    iterations: iterationsToUse,
                    hash: hashName
                },
                keyMaterial,
                keyLength || AES_KEY_LENGTH
            );

            return new Uint8Array(derivedBits);
        }

        async function pbkdf2DeriveKeyAdvanced(password, salt, iterations, keyLength, hashAlgorithm) {
            const encoder = new TextEncoder();
            const keyMaterial = await crypto.subtle.importKey(
                'raw',
                encoder.encode(password),
                { name: 'PBKDF2' },
                false,
                ['deriveBits', 'deriveKey']
            );

            const saltBytes = salt instanceof Uint8Array ? salt : new Uint8Array(salt);
            
            const derivedBits = await crypto.subtle.deriveBits(
                {
                    name: 'PBKDF2',
                    salt: saltBytes,
                    iterations: iterations || DEFAULT_ITERATIONS,
                    hash: hashAlgorithm || 'SHA-256'
                },
                keyMaterial,
                keyLength || AES_KEY_LENGTH
            );

            return new Uint8Array(derivedBits);
        }

        async function deriveKeyFromPassword(password, salt, options) {
            options = options || {};
            const iterations = options.iterations || DEFAULT_ITERATIONS;
            const keyLength = options.keyLength || AES_KEY_LENGTH;
            const hashAlgorithm = options.hash || 'SHA-256';
            
            const key = await pbkdf2DeriveKeyAdvanced(password, salt, iterations, keyLength, hashAlgorithm);
            
            return {
                key: key,
                algorithm: 'PBKDF2',
                iterations: iterations,
                hash: hashAlgorithm,
                salt: salt instanceof Uint8Array ? salt : new Uint8Array(salt)
            };
        }

        function allocateBuffer(size) {
            if (wasmExports && wasmExports.allocate) {
                return wasmExports.allocate(size);
            }
            const buffer = new Uint8Array(size);
            const ptr = wasmExports.memory.buffer.byteLength;
            return ptr;
        }

        async function aes256GcmEncrypt(plaintext, key, options) {
            options = options || {};
            const useWasmEncryption = useWasm && isWasmLoaded && wasmExports && wasmExports.aes_gcm_encrypt;

            let keyData;
            if (typeof key === 'string') {
                const salt = options.salt || generateRandomBytes(SALT_LENGTH);
                keyData = await pbkdf2DeriveKey(
                    key,
                    salt,
                    options.iterations || DEFAULT_ITERATIONS,
                    AES_KEY_LENGTH
                );
            } else {
                keyData = key instanceof Uint8Array ? key : new Uint8Array(key);
            }

            const iv = options.iv || generateRandomBytes(IV_LENGTH);
            const encoder = new TextEncoder();
            const plaintextBuffer = encoder.encode(plaintext);

            let ciphertext;
            if (useWasmEncryption) {
                try {
                    const keyPtr = allocateBuffer(keyData.length);
                    const ivPtr = allocateBuffer(iv.length);
                    const plaintextPtr = allocateBuffer(plaintextBuffer.length);
                    
                    wasmExports.aes_gcm_encrypt(
                        keyPtr, keyData.length,
                        ivPtr, iv.length,
                        plaintextPtr, plaintextBuffer.length
                    );
                    
                    const outputPtr = allocateBuffer(plaintextBuffer.length + 16);
                    const outputLen = wasmExports.get_encrypted_length();
                    
                    const encryptedData = new Uint8Array(wasmExports.memory.buffer, outputPtr, outputLen);
                    ciphertext = encryptedData.buffer;
                } catch (error) {
                    console.warn(`${MODULE_NAME}: WASM AES-GCM encryption failed, falling back to Web Crypto API`);
                    useWasmEncryption = false;
                }
            }

            if (!useWasmEncryption || !useWasm) {
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
                    tagLength: 128
                };

                if (options.additionalData) {
                    algorithmOptions.additionalData = encoder.encode(options.additionalData);
                }

                ciphertext = await crypto.subtle.encrypt(
                    algorithmOptions,
                    importedKey,
                    plaintextBuffer
                );
            }

            const salt = typeof options.salt !== 'undefined' ? 
                (options.salt instanceof Uint8Array ? arrayBufferToBase64(options.salt) : options.salt) : 
                null;

            return {
                ciphertext: arrayBufferToBase64(ciphertext),
                iv: arrayBufferToBase64(iv),
                salt: salt,
                algorithm: 'AES-256-GCM',
                wasmUsed: useWasmEncryption
            };
        }

        async function aes256GcmDecrypt(encryptedData, key, options) {
            options = options || {};
            const useWasmDecryption = useWasm && isWasmLoaded && wasmExports && wasmExports.aes_gcm_decrypt;

            let keyData;
            if (typeof key === 'string') {
                const salt = options.salt ? 
                    (typeof options.salt === 'string' ? base64ToArrayBuffer(options.salt) : options.salt) :
                    generateRandomBytes(SALT_LENGTH);
                keyData = await pbkdf2DeriveKey(
                    key,
                    salt,
                    options.iterations || DEFAULT_ITERATIONS,
                    AES_KEY_LENGTH
                );
            } else {
                keyData = key instanceof Uint8Array ? key : new Uint8Array(key);
            }

            const iv = new Uint8Array(base64ToArrayBuffer(encryptedData.iv));
            const ciphertext = base64ToArrayBuffer(encryptedData.ciphertext);

            let plaintext;
            if (useWasmDecryption) {
                try {
                    const keyPtr = allocateBuffer(keyData.length);
                    const ivPtr = allocateBuffer(iv.length);
                    const ciphertextPtr = allocateBuffer(ciphertext.byteLength);
                    
                    wasmExports.aes_gcm_decrypt(
                        keyPtr, keyData.length,
                        ivPtr, iv.length,
                        ciphertextPtr, ciphertext.byteLength
                    );
                    
                    const decryptedData = new Uint8Array(wasmExports.memory.buffer, ciphertextPtr, ciphertext.byteLength - 16);
                    plaintext = decryptedData.buffer;
                } catch (error) {
                    console.warn(`${MODULE_NAME}: WASM AES-GCM decryption failed, falling back to Web Crypto API`);
                    useWasmDecryption = false;
                }
            }

            if (!useWasmDecryption || !useWasm) {
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
                    tagLength: 128
                };

                if (encryptedData.additionalData) {
                    const encoder = new TextEncoder();
                    algorithmOptions.additionalData = encoder.encode(encryptedData.additionalData);
                }

                plaintext = await crypto.subtle.decrypt(
                    algorithmOptions,
                    importedKey,
                    ciphertext
                );
            }

            const decoder = new TextDecoder();
            return decoder.decode(plaintext);
        }

        async function aesEncrypt(plaintext, key, options) {
            options = options || {};
            const algorithm = options.algorithm || 'AES-GCM';
            
            let keyData;
            if (typeof key === 'string') {
                const salt = options.salt || generateRandomBytes(SALT_LENGTH);
                keyData = await pbkdf2DeriveKey(
                    key,
                    salt,
                    options.iterations || DEFAULT_ITERATIONS,
                    AES_KEY_LENGTH
                );
            } else {
                keyData = key instanceof Uint8Array ? key : new Uint8Array(key);
            }

            const iv = options.iv || generateRandomBytes(IV_LENGTH);
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

                ciphertext = await crypto.subtle.encrypt(
                    algorithmOptions,
                    importedKey,
                    plaintextBuffer
                );
            } else if (algorithm === 'AES-CBC') {
                const importedKey = await crypto.subtle.importKey(
                    'raw',
                    keyData.buffer,
                    { name: 'AES-CBC' },
                    false,
                    ['encrypt']
                );

                ciphertext = await crypto.subtle.encrypt(
                    {
                        name: 'AES-CBC',
                        iv: iv
                    },
                    importedKey,
                    plaintextBuffer
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
                    {
                        name: 'AES-CTR',
                        counter: iv,
                        counterBlockLength: 16
                    },
                    importedKey,
                    plaintextBuffer
                );
            }

            const salt = typeof options.salt !== 'undefined' ? 
                (options.salt instanceof Uint8Array ? arrayBufferToBase64(options.salt) : options.salt) : 
                null;

            return {
                ciphertext: arrayBufferToBase64(ciphertext),
                iv: arrayBufferToBase64(iv),
                salt: salt,
                algorithm: algorithm,
                tagLength: tagLength,
                wasmUsed: false
            };
        }

        async function aesDecrypt(encryptedData, key, options) {
            options = options || {};
            const algorithm = encryptedData.algorithm || 'AES-GCM';

            let keyData;
            if (typeof key === 'string') {
                const salt = encryptedData.salt ? 
                    (typeof encryptedData.salt === 'string' ? base64ToArrayBuffer(encryptedData.salt) : encryptedData.salt) :
                    generateRandomBytes(SALT_LENGTH);
                keyData = await pbkdf2DeriveKey(
                    key,
                    salt,
                    options.iterations || DEFAULT_ITERATIONS,
                    AES_KEY_LENGTH
                );
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

                if (encryptedData.additionalData) {
                    const encoder = new TextEncoder();
                    algorithmOptions.additionalData = encoder.encode(encryptedData.additionalData);
                }

                plaintext = await crypto.subtle.decrypt(
                    algorithmOptions,
                    importedKey,
                    ciphertext
                );
            } else if (algorithm === 'AES-CBC') {
                const importedKey = await crypto.subtle.importKey(
                    'raw',
                    keyData.buffer,
                    { name: 'AES-CBC' },
                    false,
                    ['decrypt']
                );

                plaintext = await crypto.subtle.decrypt(
                    {
                        name: 'AES-CBC',
                        iv: iv
                    },
                    importedKey,
                    ciphertext
                );
            } else if (algorithm === 'AES-CTR') {
                const importedKey = await crypto.subtle.importKey(
                    'raw',
                    keyData.buffer,
                    { name: 'AES-CTR' },
                    false,
                    ['decrypt']
                );

                plaintext = await crypto.subtle.decrypt(
                    {
                        name: 'AES-CTR',
                        counter: iv,
                        counterBlockLength: 16
                    },
                    importedKey,
                    ciphertext
                );
            }

            const decoder = new TextDecoder();
            return decoder.decode(plaintext);
        }

        async function generateKeyPair() {
            if (isNode() && !isBrowser()) {
                const nodeCrypto = require('crypto');
                return new Promise((resolve, reject) => {
                    nodeCrypto.generateKeyPair('rsa', {
                        modulusLength: 2048,
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
                        else resolve({ publicKey, privateKey });
                    });
                });
            }

            return crypto.subtle.generateKey(
                {
                    name: 'RSA-OAEP',
                    modulusLength: 2048,
                    publicExponent: new Uint8Array([1, 0, 1]),
                    hash: 'SHA-256'
                },
                true,
                ['encrypt', 'decrypt']
            );
        }

        async function encryptWithPublicKey(plaintext, publicKey) {
            if (isNode() && !isBrowser() && typeof publicKey === 'string') {
                const nodeCrypto = require('crypto');
                const buffer = Buffer.from(plaintext, 'utf8');
                const encrypted = nodeCrypto.publicEncrypt({
                    key: publicKey,
                    padding: nodeCrypto.constants.RSA_PKCS1_OAEP_PADDING,
                    oaepHash: 'sha256'
                }, buffer);
                return encrypted.toString('base64');
            }

            const encoder = new TextEncoder();
            const data = encoder.encode(plaintext);

            const importedKey = await crypto.subtle.importKey(
                'spki',
                base64ToArrayBuffer(publicKey),
                { name: 'RSA-OAEP' },
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

        async function decryptWithPrivateKey(encryptedBase64, privateKey) {
            if (isNode() && !isBrowser() && typeof privateKey === 'string') {
                const nodeCrypto = require('crypto');
                const buffer = Buffer.from(encryptedBase64, 'base64');
                const decrypted = nodeCrypto.privateDecrypt({
                    key: privateKey,
                    padding: nodeCrypto.constants.RSA_PKCS1_OAEP_PADDING,
                    oaepHash: 'sha256'
                }, buffer);
                return decrypted.toString('utf8');
            }

            const ciphertext = base64ToArrayBuffer(encryptedBase64);

            const importedKey = await crypto.subtle.importKey(
                'pkcs8',
                base64ToArrayBuffer(privateKey),
                { name: 'RSA-OAEP' },
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

        async function hashSHA256(data) {
            const encoder = new TextEncoder();
            const dataBuffer = typeof data === 'string' ? encoder.encode(data) : data;
            
            const hashBuffer = await crypto.subtle.digest('SHA-256', dataBuffer);
            return arrayBufferToBase64(hashBuffer);
        }

        async function hmacSHA256(data, key) {
            const encoder = new TextEncoder();
            const dataBuffer = encoder.encode(data);
            const keyBuffer = encoder.encode(key);

            const cryptoKey = await crypto.subtle.importKey(
                'raw',
                keyBuffer,
                { name: 'HMAC', hash: 'SHA-256' },
                false,
                ['sign']
            );

            const signature = await crypto.subtle.sign('HMAC', cryptoKey, dataBuffer);
            return arrayBufferToBase64(signature);
        }

        // WASM预加载优化
        function preloadWasm(wasmUrl, priority = 'medium') {
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
                    let response;
                    
                    if (priority === 'high' && 'createObjectURL' in URL) {
                        // High priority: use fetch with stream
                        response = await fetch(wasmUrl, { 
                            priority: 'high',
                            mode: 'cors',
                            credentials: 'same-origin'
                        });
                    } else {
                        response = await fetch(wasmUrl);
                    }

                    if (!response.ok) {
                        throw new Error(`Failed to fetch WASM module: ${response.status}`);
                    }

                    const wasmBuffer = await response.arrayBuffer();
                    const loadTime = performance.now() - start;
                    console.log(`${MODULE_NAME}: WASM buffer preloaded in ${loadTime.toFixed(2)}ms`);

                    // Initialize with preloaded buffer
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
                const importObject = {
                    env: {
                        memory: new WebAssembly.Memory({ initial: 256, maximum: 512 }),
                        seed: () => Date.now() ^ (Math.random() * 0xFFFFFFFF),
                        log: (ptr, len) => {
                            const memory = wasmExports.memory;
                            const bytes = new Uint8Array(memory.buffer, ptr, len);
                            console.log('WASM:', String.fromCharCode.apply(null, bytes));
                        },
                        abort: (ptr, line, col) => {
                            console.error(`WASM abort at ${line}:${col}`);
                            throw new Error('WASM execution aborted');
                        }
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

                const result = await WebAssembly.instantiate(wasmBuffer, importObject);
                
                if (result.instance && result.instance.exports) {
                    wasmModule = result.instance;
                    wasmExports = result.instance.exports;
                    isWasmLoaded = true;
                    useWasm = true;
                    console.log(`${MODULE_NAME}: WASM module initialized from preloaded buffer`);
                    return true;
                }
                return false;
            } catch (error) {
                console.warn(`${MODULE_NAME}: Failed to initialize from buffer:`, error);
                return false;
            }
        }

        // 密钥轮换机制
        async function generateNewKey() {
            const key = generateRandomBytes(32);
            keyCreationTime = Date.now();
            return key;
        }

        async function initializeKey(keyMaterial) {
            previousKey = currentKey;
            if (keyMaterial) {
                if (typeof keyMaterial === 'string') {
                    const salt = generateRandomBytes(SALT_LENGTH);
                    currentKey = await pbkdf2DeriveKey(keyMaterial, salt, DEFAULT_ITERATIONS, AES_KEY_LENGTH / 8);
                } else {
                    currentKey = keyMaterial;
                }
            } else {
                currentKey = await generateNewKey();
            }
            keyCreationTime = Date.now();
            return currentKey;
        }

        function startKeyRotation(interval = KEY_ROTATION_INTERVAL) {
            if (keyRotationTimer) {
                clearInterval(keyRotationTimer);
            }
            
            keyRotationTimer = setInterval(async () => {
                try {
                    console.log(`${MODULE_NAME}: Rotating encryption key`);
                    previousKey = currentKey;
                    currentKey = await generateNewKey();
                    keyCreationTime = Date.now();
                } catch (error) {
                    console.error(`${MODULE_NAME}: Key rotation failed:`, error);
                }
            }, interval);
            
            console.log(`${MODULE_NAME}: Key rotation scheduled every ${interval}ms`);
        }

        function stopKeyRotation() {
            if (keyRotationTimer) {
                clearInterval(keyRotationTimer);
                keyRotationTimer = null;
            }
        }

        function getKeyInfo() {
            return {
                hasCurrentKey: currentKey !== null,
                hasPreviousKey: previousKey !== null,
                keyAge: keyCreationTime ? Date.now() - keyCreationTime : null,
                isRotationActive: keyRotationTimer !== null
            };
        }

        // 性能基准测试
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

            // PBKDF2 benchmark
            let start = performance.now();
            for (let i = 0; i < iterations; i++) {
                await pbkdf2DeriveKey(testPassword, testSalt, 1000, 32);
            }
            results.pbkdf2 = {
                totalTime: performance.now() - start,
                avgTime: (performance.now() - start) / iterations,
                iterations
            };

            // Encryption benchmark
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

            // Decryption benchmark
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

            // Hash benchmark
            start = performance.now();
            for (let i = 0; i < iterations; i++) {
                await hashSHA256(testData);
            }
            results.hash = {
                totalTime: performance.now() - start,
                avgTime: (performance.now() - start) / iterations,
                iterations
            };

            console.log(`${MODULE_NAME}: Benchmark complete`, results);
            return results;
        }

        function initialize(wasmUrl) {
            if (initializationPromise) {
                return initializationPromise;
            }

            initializationPromise = (async () => {
                isWasmSupported = checkWasmSupport();
                
                if (wasmUrl) {
                    await loadWasm(wasmUrl);
                } else if (isBrowser()) {
                    const defaultWasmUrl = '/static/js/crypto-wasm.wasm';
                    await loadWasm(defaultWasmUrl);
                }

                return {
                    wasmLoaded: isWasmLoaded,
                    wasmSupported: isWasmSupported,
                    usingWasm: useWasm,
                    version: VERSION
                };
            })();

            return initializationPromise;
        }

        function getStatus() {
            return {
                wasmLoaded: isWasmLoaded,
                wasmSupported: isWasmSupported,
                usingWasm: useWasm,
                version: VERSION
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
            initialize: initialize,
            getStatus: getStatus,
            setUseWasm: setUseWasm,
            encrypt: aes256GcmEncrypt,
            decrypt: aes256GcmDecrypt,
            aesEncrypt: aesEncrypt,
            aesDecrypt: aesDecrypt,
            pbkdf2: pbkdf2DeriveKey,
            pbkdf2Advanced: pbkdf2DeriveKeyAdvanced,
            deriveKeyFromPassword: deriveKeyFromPassword,
            generateRandomBytes: generateRandomBytes,
            hashSHA256: hashSHA256,
            hmacSHA256: hmacSHA256,
            generateKeyPair: generateKeyPair,
            encryptWithPublicKey: encryptWithPublicKey,
            decryptWithPrivateKey: decryptWithPrivateKey,
            preloadWasm: preloadWasm,
            initializeWithBuffer: initializeWithBuffer,
            initializeKey: initializeKey,
            startKeyRotation: startKeyRotation,
            stopKeyRotation: stopKeyRotation,
            getKeyInfo: getKeyInfo,
            getPerformanceStats: getPerformanceStats,
            resetPerformanceStats: resetPerformanceStats,
            runBenchmark: runBenchmark,
            utils: {
                arrayBufferToBase64: arrayBufferToBase64,
                base64ToArrayBuffer: base64ToArrayBuffer
            }
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = CryptoWasm;
    } else {
        globalContext.CryptoWasm = CryptoWasm;
    }

})(typeof window !== 'undefined' ? window : (typeof global !== 'undefined' ? global : this));
