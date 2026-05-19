(function(globalContext) {
    'use strict';

    const WasmCrypto = (function() {
        const VERSION = '3.1.0';
        const MODULE_NAME = 'WasmCrypto';
        
        let wasmModule = null;
        let wasmExports = null;
        let isWasmLoaded = false;
        let isWasmSupported = false;
        let useWasm = false;
        let memory = null;
        
        let currentKey = null;
        let keyRotationTimer = null;
        const KEY_ROTATION_INTERVAL = 30 * 60 * 1000;

        function checkWasmSupport() {
            if (typeof WebAssembly === 'undefined') {
                console.warn(`${MODULE_NAME}: WebAssembly not supported`);
                return false;
            }
            if (typeof WebAssembly.instantiate === 'undefined') {
                console.warn(`${MODULE_NAME}: WebAssembly.instantiate not available`);
                return false;
            }
            return true;
        }

        function generateRandomBytes(length) {
            const array = new Uint8Array(length);
            crypto.getRandomValues(array);
            return array;
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

        async function loadWasmModule() {
            if (!checkWasmSupport()) {
                isWasmSupported = false;
                return false;
            }

            try {
                const wasmCode = generateWasmBytes();
                const result = await WebAssembly.instantiate(wasmCode, getImports());
                
                if (result.instance && result.instance.exports) {
                    wasmModule = result.instance;
                    wasmExports = result.instance.exports;
                    memory = wasmExports.memory;
                    isWasmLoaded = true;
                    useWasm = true;
                    console.info(`${MODULE_NAME}: WASM module loaded`);
                    return true;
                }
            } catch (error) {
                console.warn(`${MODULE_NAME}: WASM load failed, falling back to JS`);
            }
            
            isWasmLoaded = false;
            useWasm = false;
            return false;
        }

        function getImports() {
            return {
                env: {
                    memory: new WebAssembly.Memory({ initial: 256, maximum: 512 }),
                    seed: () => Date.now() ^ (Math.random() * 0xFFFFFFFF),
                    abort: () => { throw new Error('WASM abort'); }
                }
            };
        }

        function generateWasmBytes() {
            const simpleWasm = new Uint8Array([
                0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00,
                0x01, 0x08, 0x02, 0x60, 0x01, 0x7f, 0x01, 0x7f,
                0x60, 0x00, 0x00, 0x02, 0x17, 0x01, 0x06, 0x6d,
                0x65, 0x6d, 0x6f, 0x72, 0x79, 0x02, 0x00, 0x07,
                0x78, 0x6f, 0x72, 0x5f, 0x62, 0x79, 0x74, 0x65,
                0x73, 0x00, 0x01, 0x0a, 0x13, 0x03, 0x06, 0x66,
                0x75, 0x6e, 0x63, 0x74, 0x69, 0x00, 0x00, 0x0b,
                0x00, 0x00, 0x01, 0x00, 0x41, 0x01, 0x6a
            ]);
            return simpleWasm.buffer;
        }

        async function pbkdf2DeriveKey(password, salt, iterations, keyLength) {
            const encoder = new TextEncoder();
            const keyMaterial = await crypto.subtle.importKey(
                'raw', encoder.encode(password), { name: 'PBKDF2' },
                false, ['deriveBits', 'deriveKey']
            );

            const saltBytes = salt instanceof Uint8Array ? salt : new Uint8Array(salt);
            
            const derivedBits = await crypto.subtle.deriveBits({
                name: 'PBKDF2',
                salt: saltBytes,
                iterations: iterations || 100000,
                hash: 'SHA-256'
            }, keyMaterial, keyLength || 256);

            return new Uint8Array(derivedBits);
        }

        async function aes256GcmEncrypt(plaintext, key, options) {
            options = options || {};

            let keyData;
            if (typeof key === 'string') {
                const salt = options.salt || generateRandomBytes(16);
                keyData = await pbkdf2DeriveKey(key, salt, options.iterations || 100000, 256);
            } else {
                keyData = key instanceof Uint8Array ? key : new Uint8Array(key);
            }

            const iv = options.iv || generateRandomBytes(12);
            const encoder = new TextEncoder();
            const plaintextBuffer = encoder.encode(plaintext);

            const importedKey = await crypto.subtle.importKey(
                'raw', keyData.buffer, { name: 'AES-GCM' }, false, ['encrypt']
            );

            const algorithmOptions = { name: 'AES-GCM', iv: iv, tagLength: 128 };
            if (options.additionalData) {
                algorithmOptions.additionalData = encoder.encode(options.additionalData);
            }

            const ciphertext = await crypto.subtle.encrypt(algorithmOptions, importedKey, plaintextBuffer);

            return {
                ciphertext: arrayBufferToBase64(ciphertext),
                iv: arrayBufferToBase64(iv),
                algorithm: 'AES-256-GCM',
                wasmUsed: useWasm
            };
        }

        async function aes256GcmDecrypt(encryptedData, key, options) {
            options = options || {};

            let keyData;
            if (typeof key === 'string') {
                const salt = encryptedData.salt ? 
                    (typeof encryptedData.salt === 'string' ? base64ToArrayBuffer(encryptedData.salt) : encryptedData.salt) :
                    generateRandomBytes(16);
                keyData = await pbkdf2DeriveKey(key, salt, options.iterations || 100000, 256);
            } else {
                keyData = key instanceof Uint8Array ? key : new Uint8Array(key);
            }

            const iv = new Uint8Array(base64ToArrayBuffer(encryptedData.iv));
            const ciphertext = base64ToArrayBuffer(encryptedData.ciphertext);

            const importedKey = await crypto.subtle.importKey(
                'raw', keyData.buffer, { name: 'AES-GCM' }, false, ['decrypt']
            );

            const algorithmOptions = { name: 'AES-GCM', iv: iv, tagLength: 128 };
            if (encryptedData.additionalData) {
                const encoder = new TextEncoder();
                algorithmOptions.additionalData = encoder.encode(encryptedData.additionalData);
            }

            const plaintext = await crypto.subtle.decrypt(algorithmOptions, importedKey, ciphertext);
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
                'raw', keyBuffer, { name: 'HMAC', hash: 'SHA-256' },
                false, ['sign']
            );

            const signature = await crypto.subtle.sign('HMAC', cryptoKey, dataBuffer);
            return arrayBufferToBase64(signature);
        }

        async function generateNewKey() {
            const key = generateRandomBytes(32);
            return key;
        }

        async function initializeKey(keyMaterial) {
            if (keyMaterial) {
                if (typeof keyMaterial === 'string') {
                    const salt = generateRandomBytes(16);
                    currentKey = await pbkdf2DeriveKey(keyMaterial, salt, 100000, 256);
                } else {
                    currentKey = keyMaterial;
                }
            } else {
                currentKey = await generateNewKey();
            }
            return currentKey;
        }

        function startKeyRotation(interval) {
            if (keyRotationTimer) clearInterval(keyRotationTimer);
            
            keyRotationTimer = setInterval(async () => {
                try {
                    currentKey = await generateNewKey();
                } catch (error) {
                    console.error(`${MODULE_NAME}: Key rotation failed`);
                }
            }, interval || KEY_ROTATION_INTERVAL);
        }

        function stopKeyRotation() {
            if (keyRotationTimer) {
                clearInterval(keyRotationTimer);
                keyRotationTimer = null;
            }
        }

        async function initialize() {
            isWasmSupported = checkWasmSupport();
            
            try {
                await loadWasmModule();
            } catch (e) {
                console.warn(`${MODULE_NAME}: Failed to load WASM`);
            }

            return {
                wasmLoaded: isWasmLoaded,
                wasmSupported: isWasmSupported,
                usingWasm: useWasm,
                version: VERSION
            };
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
            initialize: initialize,
            getStatus: getStatus,
            setUseWasm: setUseWasm,
            encrypt: aes256GcmEncrypt,
            decrypt: aes256GcmDecrypt,
            pbkdf2: pbkdf2DeriveKey,
            generateRandomBytes: generateRandomBytes,
            hashSHA256: hashSHA256,
            hmacSHA256: hmacSHA256,
            initializeKey: initializeKey,
            startKeyRotation: startKeyRotation,
            stopKeyRotation: stopKeyRotation,
            utils: {
                arrayBufferToBase64: arrayBufferToBase64,
                base64ToArrayBuffer: base64ToArrayBuffer
            }
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = WasmCrypto;
    } else {
        globalContext.WasmCrypto = WasmCrypto;
    }

})(typeof window !== 'undefined' ? window : (typeof global !== 'undefined' ? global : this));