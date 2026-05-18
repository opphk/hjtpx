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
        const CHACHA20_KEY_LENGTH = 32;
        const CHACHA20_NONCE_LENGTH = 12;
        const POLY1305_TAG_LENGTH = 16;

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
        
        let performanceStats = {
            encrypt: { count: 0, totalTime: 0, minTime: Infinity, maxTime: 0 },
            decrypt: { count: 0, totalTime: 0, minTime: Infinity, maxTime: 0 },
            hash: { count: 0, totalTime: 0, minTime: Infinity, maxTime: 0 },
            pbkdf2: { count: 0, totalTime: 0, minTime: Infinity, maxTime: 0 },
            chacha20: { count: 0, totalTime: 0, minTime: Infinity, maxTime: 0 }
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
            const startTime = performance.now();
            
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
                        recordPerformance('pbkdf2', performance.now() - startTime);
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
            
            const derivedBits = await crypto.subtle.deriveBits(
                {
                    name: 'PBKDF2',
                    salt: saltBytes,
                    iterations: iterations || DEFAULT_ITERATIONS,
                    hash: 'SHA-256'
                },
                keyMaterial,
                keyLength || AES_KEY_LENGTH
            );

            recordPerformance('pbkdf2', performance.now() - startTime);
            return new Uint8Array(derivedBits);
        }

        function allocateBuffer(size) {
            if (wasmExports && wasmExports.allocate) {
                return wasmExports.allocate(size);
            }
            const buffer = new Uint8Array(size);
            const ptr = wasmExports.memory.buffer.byteLength;
            return ptr;
        }

        function chacha20Block(key, counter, nonce) {
            const state = new Uint32Array(16);
            const constants = [0x61707865, 0x3320646E, 0x79622D32, 0x6B206574];
            for (let i = 0; i < 4; i++) {
                state[i] = constants[i];
            }
            
            for (let i = 0; i < 8; i++) {
                state[i + 4] = (key[i * 4 + 3] << 24) | (key[i * 4 + 2] << 16) | (key[i * 4 + 1] << 8) | key[i * 4];
            }
            
            state[12] = counter;
            state[13] = (nonce[3] << 24) | (nonce[2] << 16) | (nonce[1] << 8) | nonce[0];
            state[14] = (nonce[7] << 24) | (nonce[6] << 16) | (nonce[5] << 8) | nonce[4];
            state[15] = (nonce[11] << 24) | (nonce[10] << 16) | (nonce[9] << 8) | nonce[8];

            const x = new Uint32Array(state);
            
            for (let round = 0; round < 20; round += 2) {
                x[0] ^= x[4]; x[4] ^= x[0]; x[0] ^= x[4];
                x[12] = (x[12] << 16) | (x[12] >>> 16);
                x[0] += x[12];
                
                x[1] ^= x[5]; x[5] ^= x[1]; x[1] ^= x[5];
                x[13] = (x[13] << 16) | (x[13] >>> 16);
                x[1] += x[13];
                
                x[2] ^= x[6]; x[6] ^= x[2]; x[2] ^= x[6];
                x[14] = (x[14] << 16) | (x[14] >>> 16);
                x[2] += x[14];
                
                x[3] ^= x[7]; x[7] ^= x[3]; x[3] ^= x[7];
                x[15] = (x[15] << 16) | (x[15] >>> 16);
                x[3] += x[15];
                
                x[0] ^= x[8]; x[8] ^= x[0]; x[0] ^= x[8];
                x[12] = (x[12] << 12) | (x[12] >>> 20);
                x[0] += x[12];
                
                x[1] ^= x[9]; x[9] ^= x[1]; x[1] ^= x[9];
                x[13] = (x[13] << 12) | (x[13] >>> 20);
                x[1] += x[13];
                
                x[2] ^= x[10]; x[10] ^= x[2]; x[2] ^= x[10];
                x[14] = (x[14] << 12) | (x[14] >>> 20);
                x[2] += x[14];
                
                x[3] ^= x[11]; x[11] ^= x[3]; x[3] ^= x[11];
                x[15] = (x[15] << 12) | (x[15] >>> 20);
                x[3] += x[15];
                
                x[0] ^= x[12]; x[12] ^= x[0]; x[0] ^= x[12];
                x[4] = (x[4] << 8) | (x[4] >>> 24);
                x[0] += x[4];
                
                x[1] ^= x[13]; x[13] ^= x[1]; x[1] ^= x[13];
                x[5] = (x[5] << 8) | (x[5] >>> 24);
                x[1] += x[5];
                
                x[2] ^= x[14]; x[14] ^= x[2]; x[2] ^= x[14];
                x[6] = (x[6] << 8) | (x[6] >>> 24);
                x[2] += x[6];
                
                x[3] ^= x[15]; x[15] ^= x[3]; x[3] ^= x[15];
                x[7] = (x[7] << 8) | (x[7] >>> 24);
                x[3] += x[7];
                
                x[0] ^= x[12]; x[12] ^= x[0]; x[0] ^= x[12];
                x[8] = (x[8] << 7) | (x[8] >>> 25);
                x[0] += x[8];
                
                x[1] ^= x[13]; x[13] ^= x[1]; x[1] ^= x[13];
                x[9] = (x[9] << 7) | (x[9] >>> 25);
                x[1] += x[9];
                
                x[2] ^= x[14]; x[14] ^= x[2]; x[2] ^= x[14];
                x[10] = (x[10] << 7) | (x[10] >>> 25);
                x[2] += x[10];
                
                x[3] ^= x[15]; x[15] ^= x[3]; x[3] ^= x[15];
                x[11] = (x[11] << 7) | (x[11] >>> 25);
                x[3] += x[11];
            }

            const result = new Uint8Array(64);
            for (let i = 0; i < 16; i++) {
                result[i * 4] = x[i] & 0xff;
                result[i * 4 + 1] = (x[i] >> 8) & 0xff;
                result[i * 4 + 2] = (x[i] >> 16) & 0xff;
                result[i * 4 + 3] = (x[i] >> 24) & 0xff;
            }
            
            return result;
        }

        function poly1305Mac(key, data) {
            const r0 = (key[0] & 0xff) | ((key[1] & 0xff) << 8) | ((key[2] & 0xff) << 16) | ((key[3] & 0x0f) << 24);
            const r1 = ((key[3] >> 4) & 0xff) | ((key[4] & 0xff) << 8) | ((key[5] & 0xff) << 16) | ((key[6] & 0x0f) << 24);
            const r2 = ((key[6] >> 4) & 0xff) | ((key[7] & 0xff) << 8) | ((key[8] & 0xff) << 16) | ((key[9] & 0x0f) << 24);
            const r3 = ((key[9] >> 4) & 0xff) | ((key[10] & 0xff) << 8) | ((key[11] & 0xff) << 16) | ((key[12] & 0x0f) << 24);
            const r4 = ((key[12] >> 4) & 0xff) | ((key[13] & 0xff) << 8) | ((key[14] & 0xff) << 16) | ((key[15] & 0x0f) << 24);

            const s0 = (key[16] & 0xff) | ((key[17] & 0xff) << 8) | ((key[18] & 0xff) << 16) | ((key[19] & 0xff) << 24);
            const s1 = (key[20] & 0xff) | ((key[21] & 0xff) << 8) | ((key[22] & 0xff) << 16) | ((key[23] & 0xff) << 24);
            const s2 = (key[24] & 0xff) | ((key[25] & 0xff) << 8) | ((key[26] & 0xff) << 16) | ((key[27] & 0xff) << 24);
            const s3 = (key[28] & 0xff) | ((key[29] & 0xff) << 8) | ((key[30] & 0xff) << 16) | ((key[31] & 0xff) << 24);

            let h0 = 0, h1 = 0, h2 = 0, h3 = 0, h4 = 0;

            for (let i = 0; i < data.length; i += 16) {
                let d0 = 0, d1 = 0, d2 = 0, d3 = 0, d4 = 0;
                
                for (let j = 0; j < 16 && i + j < data.length; j++) {
                    const shift = (j % 4) * 8;
                    const idx = Math.floor(j / 4);
                    const val = data[i + j] << shift;
                    
                    if (idx === 0) d0 |= val;
                    else if (idx === 1) d1 |= val;
                    else if (idx === 2) d2 |= val;
                    else if (idx === 3) d3 |= val;
                    else d4 |= val;
                }
                
                if (i + 16 <= data.length) {
                    d4 |= 1 << 24;
                }

                h0 += d0; h1 += d1; h2 += d2; h3 += d3; h4 += d4;

                let c = Math.floor(h0 / 0x100000000); h0 %= 0x100000000;
                h1 += c; c = Math.floor(h1 / 0x100000000); h1 %= 0x100000000;
                h2 += c; c = Math.floor(h2 / 0x100000000); h2 %= 0x100000000;
                h3 += c; c = Math.floor(h3 / 0x100000000); h3 %= 0x100000000;
                h4 += c; c = Math.floor(h4 / 0x100000000); h4 %= 0x100000000;

                const t0 = h0, t1 = h1, t2 = h2, t3 = h3, t4 = h4;

                function mul(r, t) {
                    const a = Math.floor(t / 0x100000000), b = t % 0x100000000;
                    const c = Math.floor(r / 0x100000000), d = r % 0x100000000;
                    
                    let ac = a * c, ad = a * d, bc = b * c, bd = b * d;
                    
                    let carry = Math.floor(bd / 0x100000000);
                    bd %= 0x100000000;
                    
                    carry += Math.floor(bc / 0x100000000);
                    bc %= 0x100000000;
                    
                    carry += Math.floor(ad / 0x100000000);
                    ad %= 0x100000000;
                    
                    carry += ac;
                    
                    return [bd, bc + ad, carry];
                }

                const m0 = mul(r0, t0);
                const m1 = mul(r1, t1);
                const m2 = mul(r2, t2);
                const m3 = mul(r3, t3);
                const m4 = mul(r4, t4);

                h0 = m0[0] + (m1[2] << 1) + (m2[2] << 2) + (m3[2] << 3) + (m4[2] << 4);
                h1 = m0[1] + m1[0] + (m1[1] << 1) + (m2[1] << 2) + (m3[1] << 3) + (m4[1] << 4);
                h2 = m1[1] + m2[0] + (m2[1] << 1) + (m3[1] << 2) + (m4[1] << 3);
                h3 = m2[1] + m3[0] + (m3[1] << 1) + (m4[1] << 2);
                h4 = m3[1] + m4[0];

                c = Math.floor(h0 / 0x100000000); h0 %= 0x100000000;
                h1 += c; c = Math.floor(h1 / 0x100000000); h1 %= 0x100000000;
                h2 += c; c = Math.floor(h2 / 0x100000000); h2 %= 0x100000000;
                h3 += c; c = Math.floor(h3 / 0x100000000); h3 %= 0x100000000;
                h4 += c; c = Math.floor(h4 / 0x100000000); h4 %= 0x100000000;

                h2 += (h4 << 2) + 1;
                h4 = 0;

                c = Math.floor(h2 / 0x100000000); h2 %= 0x100000000;
                h3 += c; c = Math.floor(h3 / 0x100000000); h3 %= 0x100000000;
                h4 += c; c = Math.floor(h4 / 0x100000000); h4 %= 0x100000000;
            }

            h0 += s0; h1 += s1; h2 += s2; h3 += s3;

            if (h0 >= 0x100000000) h0 -= 0x100000000;
            if (h1 >= 0x100000000) h1 -= 0x100000000;
            if (h2 >= 0x100000000) h2 -= 0x100000000;
            if (h3 >= 0x100000000) h3 -= 0x100000000;

            const mac = new Uint8Array(16);
            mac[0] = h0 & 0xff;
            mac[1] = (h0 >> 8) & 0xff;
            mac[2] = (h0 >> 16) & 0xff;
            mac[3] = (h0 >> 24) & 0xff;
            mac[4] = h1 & 0xff;
            mac[5] = (h1 >> 8) & 0xff;
            mac[6] = (h1 >> 16) & 0xff;
            mac[7] = (h1 >> 24) & 0xff;
            mac[8] = h2 & 0xff;
            mac[9] = (h2 >> 8) & 0xff;
            mac[10] = (h2 >> 16) & 0xff;
            mac[11] = (h2 >> 24) & 0xff;
            mac[12] = h3 & 0xff;
            mac[13] = (h3 >> 8) & 0xff;
            mac[14] = (h3 >> 16) & 0xff;
            mac[15] = (h3 >> 24) & 0xff;

            return mac;
        }

        async function chacha20Poly1305Encrypt(plaintext, key, options) {
            const startTime = performance.now();
            options = options || {};
            
            let keyData;
            if (typeof key === 'string') {
                const salt = options.salt || generateRandomBytes(SALT_LENGTH);
                keyData = await pbkdf2DeriveKey(key, salt, options.iterations || DEFAULT_ITERATIONS, CHACHA20_KEY_LENGTH * 8);
            } else {
                keyData = key instanceof Uint8Array ? key : new Uint8Array(key);
            }

            if (keyData.length !== 32) {
                throw new Error('ChaCha20-Poly1305 requires 256-bit key');
            }

            const nonce = options.nonce || generateRandomBytes(CHACHA20_NONCE_LENGTH);
            const encoder = new TextEncoder();
            const plaintextBuffer = encoder.encode(plaintext);
            const additionalData = options.additionalData || new Uint8Array(0);

            const block0 = chacha20Block(keyData, 0, nonce);
            const poly1305Key = block0.slice(0, 32);

            const encrypted = new Uint8Array(plaintextBuffer.length);
            let remaining = plaintextBuffer.length;
            let offset = 0;
            let currentCounter = 1;

            while (remaining > 0) {
                const block = chacha20Block(keyData, currentCounter, nonce);
                const blockSize = Math.min(remaining, 64);
                
                for (let i = 0; i < blockSize; i++) {
                    encrypted[offset + i] = plaintextBuffer[offset + i] ^ block[i];
                }
                
                offset += blockSize;
                remaining -= blockSize;
                currentCounter++;
            }

            const paddedAdLength = (additionalData.length + 15) & ~15;
            const paddedCtLength = (encrypted.length + 15) & ~15;
            
            const authData = new Uint8Array(paddedAdLength + paddedCtLength + 16);
            
            authData.set(additionalData);
            authData.set(encrypted, paddedAdLength);
            
            const adLen64 = new Uint8Array(8);
            const ctLen64 = new Uint8Array(8);
            
            for (let i = 0; i < 8; i++) {
                adLen64[i] = (additionalData.length >> (8 * i)) & 0xff;
                ctLen64[i] = (encrypted.length >> (8 * i)) & 0xff;
            }
            
            authData.set(adLen64, paddedAdLength + paddedCtLength);
            authData.set(ctLen64, paddedAdLength + paddedCtLength + 8);

            const tag = poly1305Mac(poly1305Key, authData);

            const combined = new Uint8Array(nonce.length + encrypted.length + tag.length);
            combined.set(nonce);
            combined.set(encrypted, nonce.length);
            combined.set(tag, nonce.length + encrypted.length);

            recordPerformance('chacha20', performance.now() - startTime);

            const salt = typeof options.salt !== 'undefined' ? 
                (options.salt instanceof Uint8Array ? arrayBufferToBase64(options.salt) : options.salt) : 
                null;

            return {
                ciphertext: arrayBufferToBase64(combined),
                nonce: arrayBufferToBase64(nonce),
                salt: salt,
                algorithm: 'ChaCha20-Poly1305',
                wasmUsed: false
            };
        }

        async function chacha20Poly1305Decrypt(encryptedData, key, options) {
            const startTime = performance.now();
            options = options || {};
            
            let keyData;
            if (typeof key === 'string') {
                const salt = options.salt ? 
                    (typeof options.salt === 'string' ? base64ToArrayBuffer(options.salt) : options.salt) :
                    generateRandomBytes(SALT_LENGTH);
                keyData = await pbkdf2DeriveKey(key, salt, options.iterations || DEFAULT_ITERATIONS, CHACHA20_KEY_LENGTH * 8);
            } else {
                keyData = key instanceof Uint8Array ? key : new Uint8Array(key);
            }

            if (keyData.length !== 32) {
                throw new Error('ChaCha20-Poly1305 requires 256-bit key');
            }

            const combined = new Uint8Array(base64ToArrayBuffer(encryptedData.ciphertext));
            const nonce = combined.slice(0, CHACHA20_NONCE_LENGTH);
            const ciphertext = combined.slice(CHACHA20_NONCE_LENGTH, combined.length - POLY1305_TAG_LENGTH);
            const tag = combined.slice(combined.length - POLY1305_TAG_LENGTH);

            const additionalData = options.additionalData || new Uint8Array(0);

            const block0 = chacha20Block(keyData, 0, nonce);
            const poly1305Key = block0.slice(0, 32);

            const paddedAdLength = (additionalData.length + 15) & ~15;
            const paddedCtLength = (ciphertext.length + 15) & ~15;
            
            const authData = new Uint8Array(paddedAdLength + paddedCtLength + 16);
            
            authData.set(additionalData);
            authData.set(ciphertext, paddedAdLength);
            
            const adLen64 = new Uint8Array(8);
            const ctLen64 = new Uint8Array(8);
            
            for (let i = 0; i < 8; i++) {
                adLen64[i] = (additionalData.length >> (8 * i)) & 0xff;
                ctLen64[i] = (ciphertext.length >> (8 * i)) & 0xff;
            }
            
            authData.set(adLen64, paddedAdLength + paddedCtLength);
            authData.set(ctLen64, paddedAdLength + paddedCtLength + 8);

            const computedTag = poly1305Mac(poly1305Key, authData);

            for (let i = 0; i < POLY1305_TAG_LENGTH; i++) {
                if (computedTag[i] !== tag[i]) {
                    throw new Error('Authentication tag mismatch');
                }
            }

            const decrypted = new Uint8Array(ciphertext.length);
            let remaining = ciphertext.length;
            let offset = 0;
            let currentCounter = 1;

            while (remaining > 0) {
                const block = chacha20Block(keyData, currentCounter, nonce);
                const blockSize = Math.min(remaining, 64);
                
                for (let i = 0; i < blockSize; i++) {
                    decrypted[offset + i] = ciphertext[offset + i] ^ block[i];
                }
                
                offset += blockSize;
                remaining -= blockSize;
                currentCounter++;
            }

            recordPerformance('chacha20', performance.now() - startTime);

            const decoder = new TextDecoder();
            return decoder.decode(decrypted);
        }

        async function aes256GcmEncrypt(plaintext, key, options) {
            options = options || {};
            const useWasmEncryption = useWasm && isWasmLoaded && wasmExports && wasmExports.aes_gcm_encrypt;
            const startTime = performance.now();

            let keyData;
            if (typeof key === 'string') {
                const salt = options.salt || generateRandomBytes(SALT_LENGTH);
                keyData = await pbkdf2DeriveKey(key, salt, options.iterations || DEFAULT_ITERATIONS, AES_KEY_LENGTH);
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
                    
                    wasmExports.aes_gcm_encrypt(keyPtr, keyData.length, ivPtr, iv.length, plaintextPtr, plaintextBuffer.length);
                    
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
                const importedKey = await crypto.subtle.importKey('raw', keyData.buffer, { name: 'AES-GCM' }, false, ['encrypt']);

                const algorithmOptions = { name: 'AES-GCM', iv: iv, tagLength: 128 };

                if (options.additionalData) {
                    algorithmOptions.additionalData = encoder.encode(options.additionalData);
                }

                ciphertext = await crypto.subtle.encrypt(algorithmOptions, importedKey, plaintextBuffer);
            }

            recordPerformance('encrypt', performance.now() - startTime);

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
            const startTime = performance.now();

            let keyData;
            if (typeof key === 'string') {
                const salt = options.salt ? 
                    (typeof options.salt === 'string' ? base64ToArrayBuffer(options.salt) : options.salt) :
                    generateRandomBytes(SALT_LENGTH);
                keyData = await pbkdf2DeriveKey(key, salt, options.iterations || DEFAULT_ITERATIONS, AES_KEY_LENGTH);
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
                    
                    wasmExports.aes_gcm_decrypt(keyPtr, keyData.length, ivPtr, iv.length, ciphertextPtr, ciphertext.byteLength);
                    
                    const decryptedData = new Uint8Array(wasmExports.memory.buffer, ciphertextPtr, ciphertext.byteLength - 16);
                    plaintext = decryptedData.buffer;
                } catch (error) {
                    console.warn(`${MODULE_NAME}: WASM AES-GCM decryption failed, falling back to Web Crypto API`);
                    useWasmDecryption = false;
                }
            }

            if (!useWasmDecryption || !useWasm) {
                const importedKey = await crypto.subtle.importKey('raw', keyData.buffer, { name: 'AES-GCM' }, false, ['decrypt']);

                const algorithmOptions = { name: 'AES-GCM', iv: iv, tagLength: 128 };

                if (encryptedData.additionalData) {
                    const encoder = new TextEncoder();
                    algorithmOptions.additionalData = encoder.encode(encryptedData.additionalData);
                }

                plaintext = await crypto.subtle.decrypt(algorithmOptions, importedKey, ciphertext);
            }

            recordPerformance('decrypt', performance.now() - startTime);

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
                        publicKeyEncoding: { type: 'spki', format: 'pem' },
                        privateKeyEncoding: { type: 'pkcs8', format: 'pem' }
                    }, (err, publicKey, privateKey) => {
                        if (err) reject(err);
                        else resolve({ publicKey, privateKey });
                    });
                });
            }

            return crypto.subtle.generateKey(
                { name: 'RSA-OAEP', modulusLength: 2048, publicExponent: new Uint8Array([1, 0, 1]), hash: 'SHA-256' },
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

            const ciphertext = await crypto.subtle.encrypt({ name: 'RSA-OAEP' }, importedKey, data);
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

            const plaintext = await crypto.subtle.decrypt({ name: 'RSA-OAEP' }, importedKey, ciphertext);

            const decoder = new TextDecoder();
            return decoder.decode(plaintext);
        }

        async function hashSHA256(data) {
            const startTime = performance.now();
            const encoder = new TextEncoder();
            const dataBuffer = typeof data === 'string' ? encoder.encode(data) : data;
            
            const hashBuffer = await crypto.subtle.digest('SHA-256', dataBuffer);
            recordPerformance('hash', performance.now() - startTime);
            return arrayBufferToBase64(hashBuffer);
        }

        async function hmacSHA256(data, key) {
            const encoder = new TextEncoder();
            const dataBuffer = encoder.encode(data);
            const keyBuffer = encoder.encode(key);

            const cryptoKey = await crypto.subtle.importKey('raw', keyBuffer, { name: 'HMAC', hash: 'SHA-256' }, false, ['sign']);

            const signature = await crypto.subtle.sign('HMAC', cryptoKey, dataBuffer);
            return arrayBufferToBase64(signature);
        }

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
                        response = await fetch(wasmUrl, { priority: 'high', mode: 'cors', credentials: 'same-origin' });
                    } else {
                        response = await fetch(wasmUrl);
                    }

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
            results.pbkdf2 = { totalTime: performance.now() - start, avgTime: (performance.now() - start) / iterations, iterations };

            const key = await generateNewKey();
            start = performance.now();
            for (let i = 0; i < iterations; i++) {
                await aes256GcmEncrypt(testData, key);
            }
            results.encrypt = { totalTime: performance.now() - start, avgTime: (performance.now() - start) / iterations, iterations };

            const encrypted = await aes256GcmEncrypt(testData, key);
            start = performance.now();
            for (let i = 0; i < iterations; i++) {
                await aes256GcmDecrypt(encrypted, key);
            }
            results.decrypt = { totalTime: performance.now() - start, avgTime: (performance.now() - start) / iterations, iterations };

            start = performance.now();
            for (let i = 0; i < iterations; i++) {
                await hashSHA256(testData);
            }
            results.hash = { totalTime: performance.now() - start, avgTime: (performance.now() - start) / iterations, iterations };

            start = performance.now();
            for (let i = 0; i < iterations; i++) {
                await chacha20Poly1305Encrypt(testData, key);
            }
            results.chacha20 = { totalTime: performance.now() - start, avgTime: (performance.now() - start) / iterations, iterations };

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
            initialize: initialize,
            getStatus: getStatus,
            setUseWasm: setUseWasm,
            encrypt: aes256GcmEncrypt,
            decrypt: aes256GcmDecrypt,
            encryptChaCha20: chacha20Poly1305Encrypt,
            decryptChaCha20: chacha20Poly1305Decrypt,
            pbkdf2: pbkdf2DeriveKey,
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