(function(globalContext) {
    'use strict';

    var CryptoModule = (function() {
        var version = '2.0.0';
        var defaultKey = 'hjtpx-obfuscate-key-2024';
        var storagePrefix = '_cry_';
        var debugDetectionEnabled = true;
        var integrityCheckInterval = null;
        var originalHash = null;
        var protectionActive = false;

        function CryptoError(message, code) {
            this.name = 'CryptoError';
            this.message = message;
            this.code = code || 'UNKNOWN_ERROR';
        }
        CryptoError.prototype = Object.create(Error.prototype);
        CryptoError.prototype.constructor = CryptoError;

        function arrayBufferToBase64(buffer) {
            var bytes = new Uint8Array(buffer);
            var binary = '';
            for (var i = 0; i < bytes.byteLength; i++) {
                binary += String.fromCharCode(bytes[i]);
            }
            return btoa(binary);
        }

        function base64ToArrayBuffer(base64) {
            var binaryString = atob(base64);
            var bytes = new Uint8Array(binaryString.length);
            for (var i = 0; i < binaryString.length; i++) {
                bytes[i] = binaryString.charCodeAt(i);
            }
            return bytes.buffer;
        }

        function deriveKey(password, salt, iterations, keyLength) {
            var encoder = new TextEncoder();
            var passwordBuffer = encoder.encode(password);
            var saltBuffer = salt instanceof ArrayBuffer ? new Uint8Array(salt) : salt;

            var importedKey = crypto.subtle.importKey(
                'raw',
                passwordBuffer,
                { name: 'PBKDF2' },
                false,
                ['deriveBits']
            );

            return importedKey.then(function(key) {
                return crypto.subtle.deriveBits(
                    {
                        name: 'PBKDF2',
                        salt: saltBuffer,
                        iterations: iterations || 100000,
                        hash: 'SHA-256'
                    },
                    key,
                    (keyLength || 256)
                );
            });
        }

        function generateRandomBytes(length) {
            var array = new Uint8Array(length);
            crypto.getRandomValues(array);
            return array;
        }

        function generateRandomString(length) {
            var chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
            var randomBytes = generateRandomBytes(length);
            var result = '';
            for (var i = 0; i < length; i++) {
                result += chars[randomBytes[i] % chars.length];
            }
            return result;
        }

        function hashSHA256(data) {
            var encoder = new TextEncoder();
            var dataBuffer = typeof data === 'string' ? encoder.encode(data) : data;
            return crypto.subtle.digest('SHA-256', dataBuffer).then(function(hash) {
                return arrayBufferToBase64(hash);
            });
        }

        async function encryptAES(plaintext, key, options) {
            options = options || {};
            var useGCM = options.mode !== 'CBC';

            var keyData = typeof key === 'string' ?
                await deriveKey(key, new Uint8Array([1, 2, 3, 4, 5, 6, 7, 8]), 100000, 256) :
                key;

            var importedKey = await crypto.subtle.importKey(
                'raw',
                keyData,
                { name: useGCM ? 'AES-GCM' : 'AES-CBC' },
                false,
                ['encrypt']
            );

            var iv = generateRandomBytes(useGCM ? 12 : 16);
            var encoder = new TextEncoder();
            var plaintextBuffer = encoder.encode(plaintext);

            var additionalData = options.aad ? encoder.encode(options.aad) : null;
            var algorithmOptions = useGCM ?
                { name: 'AES-GCM', iv: iv, tagLength: 128 } :
                { name: 'AES-CBC', iv: iv };

            if (additionalData && useGCM) {
                algorithmOptions.additionalData = additionalData;
            }

            var ciphertext = await crypto.subtle.encrypt(
                algorithmOptions,
                importedKey,
                plaintextBuffer
            );

            var result = {
                ciphertext: arrayBufferToBase64(ciphertext),
                iv: arrayBufferToBase64(iv),
                mode: useGCM ? 'GCM' : 'CBC'
            };

            if (additionalData) {
                result.aad = options.aad;
            }

            return result;
        }

        async function decryptAES(encryptedData, key, options) {
            options = options || {};
            var useGCM = (encryptedData.mode === 'GCM') || (options.mode === 'GCM');

            var keyData = typeof key === 'string' ?
                await deriveKey(key, new Uint8Array([1, 2, 3, 4, 5, 6, 7, 8]), 100000, 256) :
                key;

            var importedKey = await crypto.subtle.importKey(
                'raw',
                keyData,
                { name: useGCM ? 'AES-GCM' : 'AES-CBC' },
                false,
                ['decrypt']
            );

            var iv = new Uint8Array(base64ToArrayBuffer(encryptedData.iv));
            var ciphertext = base64ToArrayBuffer(encryptedData.ciphertext);

            var algorithmOptions = useGCM ?
                { name: 'AES-GCM', iv: iv, tagLength: 128 } :
                { name: 'AES-CBC', iv: iv };

            if (encryptedData.aad && useGCM) {
                var encoder = new TextEncoder();
                algorithmOptions.additionalData = encoder.encode(encryptedData.aad);
            }

            var plaintext = await crypto.subtle.decrypt(
                algorithmOptions,
                importedKey,
                ciphertext
            );

            var decoder = new TextDecoder();
            return decoder.decode(plaintext);
        }

        async function encryptString(plaintext, key) {
            if (!plaintext) {
                throw new CryptoError('Plaintext cannot be empty', 'EMPTY_INPUT');
            }

            var actualKey = key || defaultKey;
            var encrypted = await encryptAES(plaintext, actualKey);

            return arrayBufferToBase64(new Uint8Array([
                ...new Uint8Array([1]),
                ...new Uint8Array(base64ToArrayBuffer(encrypted.iv)),
                ...new Uint8Array(base64ToArrayBuffer(encrypted.ciphertext))
            ]).buffer);
        }

        async function decryptString(encryptedBase64, key) {
            if (!encryptedBase64) {
                throw new CryptoError('Encrypted data cannot be empty', 'EMPTY_INPUT');
            }

            try {
                var actualKey = key || defaultKey;
                var data = new Uint8Array(base64ToArrayBuffer(encryptedBase64));
                var version = data[0];
                var iv = data.slice(1, 17);
                var ciphertext = data.slice(17);

                var encryptedData = {
                    ciphertext: arrayBufferToBase64(ciphertext.buffer),
                    iv: arrayBufferToBase64(iv.buffer),
                    mode: 'CBC'
                };

                return await decryptAES(encryptedData, actualKey);
            } catch (error) {
                throw new CryptoError('Decryption failed: ' + error.message, 'DECRYPTION_ERROR');
            }
        }

        function detectDebugging() {
            if (!debugDetectionEnabled) return false;

            var threshold = 160;
            if (window.outerWidth - window.innerWidth > threshold ||
                window.outerHeight - window.innerHeight > threshold) {
                return true;
            }

            var start = performance.now();
            debugger;
            var end = performance.now();
            if (end - start > 100) {
                return true;
            }

            if (typeof console.clear !== 'undefined') {
                var originalClear = console.clear;
                console.clear = function() {
                    return;
                };
                console.clear = originalClear;
            }

            return false;
        }

        function generateHMAC(data, key) {
            var encoder = new TextEncoder();
            var dataBuffer = encoder.encode(data);

            return crypto.subtle.importKey(
                'raw',
                encoder.encode(key),
                { name: 'HMAC', hash: 'SHA-256' },
                false,
                ['sign']
            ).then(function(importedKey) {
                return crypto.subtle.sign('HMAC', importedKey, dataBuffer);
            }).then(function(signature) {
                return arrayBufferToBase64(signature);
            });
        }

        async function verifyHMAC(data, signature, key) {
            var expectedSignature = await generateHMAC(data, key);
            return signature === expectedSignature;
        }

        function secureStorage(key) {
            var storageKey = storagePrefix + key;
            var sessionData = {};

            return {
                set: async function(value, options) {
                    options = options || {};
                    var dataToStore;

                    if (options.encrypt !== false) {
                        var encrypted = await encryptString(JSON.stringify(value));
                        dataToStore = encrypted;
                    } else {
                        dataToStore = JSON.stringify(value);
                    }

                    if (options.sessionOnly) {
                        sessionData[key] = dataToStore;
                    } else {
                        try {
                            localStorage.setItem(storageKey, dataToStore);
                        } catch (e) {
                            console.error('Storage error:', e);
                        }
                    }

                    return true;
                },

                get: async function(options) {
                    options = options || {};
                    var storedData;

                    if (options.sessionOnly) {
                        storedData = sessionData[key];
                    } else {
                        try {
                            storedData = localStorage.getItem(storageKey);
                        } catch (e) {
                            console.error('Storage error:', e);
                        }
                    }

                    if (!storedData) {
                        return null;
                    }

                    if (options.decrypt !== false) {
                        try {
                            var decrypted = await decryptString(storedData);
                            return JSON.parse(decrypted);
                        } catch (e) {
                            try {
                                return JSON.parse(storedData);
                            } catch (e2) {
                                return storedData;
                            }
                        }
                    }

                    try {
                        return JSON.parse(storedData);
                    } catch (e) {
                        return storedData;
                    }
                },

                remove: function(options) {
                    options = options || {};

                    if (options.sessionOnly) {
                        delete sessionData[key];
                    } else {
                        try {
                            localStorage.removeItem(storageKey);
                        } catch (e) {
                            console.error('Storage error:', e);
                        }
                    }

                    return true;
                },

                exists: function(options) {
                    options = options || {};

                    if (options.sessionOnly) {
                        return key in sessionData;
                    }

                    try {
                        return localStorage.getItem(storageKey) !== null;
                    } catch (e) {
                        return false;
                    }
                }
            };
        }

        return {
            version: version,
            encrypt: encryptAES,
            decrypt: decryptAES,
            encryptString: encryptString,
            decryptString: decryptString,
            hash: hashSHA256,
            generateRandomBytes: generateRandomBytes,
            generateRandomString: generateRandomString,
            generateHMAC: generateHMAC,
            verifyHMAC: verifyHMAC,
            detectDebugging: detectDebugging,
            secureStorage: secureStorage,
            CryptoError: CryptoError,
            _setDebugDetection: function(enabled) { debugDetectionEnabled = enabled; }
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = CryptoModule;
    } else {
        globalContext.CryptoModule = CryptoModule;
    }

})(typeof window !== 'undefined' ? window : this);
