(function(globalContext) {
    'use strict';

    const CryptoWasmV3 = (function() {
        const VERSION = '3.0.0';
        const MODULE_NAME = 'CryptoWasmV3';
        
        const CONSTANTS = {
            AES_KEY_LENGTH: 256,
            CHACHA20_KEY_LENGTH: 256,
            IV_LENGTH: 12,
            SALT_LENGTH: 16,
            TAG_LENGTH: 16,
            DEFAULT_ITERATIONS: 100000,
            HYBRID_KEY_LENGTH: 64,
            MAX_BATCH_SIZE: 100,
            MAX_QUEUE_SIZE: 1000,
            OFFLOAD_TIMEOUT: 30000,
            GPU_CHECK_INTERVAL: 5000,
            MEMORY_LIMIT_MB: 512,
            SECURITY_LEVELS: {
                STANDARD: 'standard',
                HIGH: 'high',
                MAXIMUM: 'maximum'
            },
            SANDBOX_MODES: {
                BASIC: 'basic',
                ENHANCED: 'enhanced',
                ISOLATED: 'isolated'
            },
            KEY_TYPES: {
                AES256GCM: 'aes-256-gcm',
                CHACHA20: 'chacha20-poly1305',
                HYBRID: 'hybrid'
            },
            OFFLOAD_DEVICES: {
                CPU: 'cpu',
                GPU: 'gpu',
                TPU: 'tpu',
                WASM: 'wasm'
            },
            COMPRESSION_TYPES: {
                NONE: 'none',
                LZ4: 'lz4',
                ZSTD: 'zstd',
                GZIP: 'gzip'
            }
        };

        let wasmModule = null;
        let wasmExports = null;
        let isWasmLoaded = false;
        let isWasmSupported = false;
        let useWasm = false;
        let useGPU = false;
        let initializationPromise = null;
        
        let config = {
            enableChaCha20: true,
            enableAES256GCM: true,
            enableAIInference: true,
            maxConcurrentOps: 1000,
            memoryLimitMB: CONSTANTS.MEMORY_LIMIT_MB,
            securityLevel: CONSTANTS.SECURITY_LEVELS.HIGH,
            sandboxMode: CONSTANTS.SANDBOX_MODES.ENHANCED,
            offloadingMode: 'auto',
            compression: CONSTANTS.COMPRESSION_TYPES.LZ4
        };

        let metrics = {
            encryptOps: 0,
            decryptOps: 0,
            aiInferenceOps: 0,
            offloadOps: 0,
            totalLatency: 0,
            avgLatency: 0,
            maxLatency: 0,
            memoryUsage: 0,
            errorsCount: 0,
            securityBlocks: 0,
            cacheHitRate: 0
        };

        let keyPool = {
            keys: new Map(),
            poolSize: 100,
            generateKey(type) {
                const key = this.generateRandomBytes(
                    type === CONSTANTS.KEY_TYPES.HYBRID ? 
                    CONSTANTS.HYBRID_KEY_LENGTH : 
                    CONSTANTS.CHACHA20_KEY_LENGTH
                );
                return key;
            },
            generateRandomBytes(length) {
                const array = new Uint8Array(length);
                crypto.getRandomValues(array);
                return array;
            }
        };

        let sandbox = {
            mode: CONSTANTS.SANDBOX_MODES.ENHANCED,
            securityLevel: CONSTANTS.SECURITY_LEVELS.HIGH,
            strictMode: false,
            auditLog: [],
            forbiddenFuncs: new Set([
                'syscall_js_value_get',
                'syscall_js_string_get',
                'syscall_js_value_set',
                'memory_grow',
                'proc_exit'
            ]),
            allowedImports: new Set([
                'env.abort',
                'env.seed',
                'wasi_snapshot_preview1.fd_write'
            ]),
            validateOperation(op) {
                if (this.strictMode && this.forbiddenFuncs.has(op)) {
                    this.logAudit({
                        timestamp: Date.now(),
                        operation: op,
                        blocked: true,
                        reason: 'Operation forbidden in strict mode'
                    });
                    return false;
                }
                return true;
            },
            logAudit(entry) {
                this.auditLog.push({
                    ...entry,
                    timestamp: Date.now()
                });
                if (this.auditLog.length > 1000) {
                    this.auditLog.shift();
                }
            },
            setMode(mode) {
                this.mode = mode;
                if (mode === CONSTANTS.SANDBOX_MODES.ISOLATED) {
                    this.strictMode = true;
                    this.allowedImports.clear();
                }
            },
            getAuditLog() {
                return [...this.auditLog];
            }
        };

        let aiModule = {
            enabled: true,
            modelCache: new Map(),
            cacheSize: 50,
            maxBatchSize: 32,
            quantization: 'int8',
            inferenceMode: 'async',
            cacheHitCount: 0,
            cacheMissCount: 0,
            getCachedResult(modelId, input) {
                const cacheKey = `${modelId}_${JSON.stringify(input)}`;
                const cached = this.modelCache.get(cacheKey);
                if (cached && (Date.now() - cached.timestamp) < 5 * 60 * 1000) {
                    cached.hitCount++;
                    this.cacheHitCount++;
                    return cached.result;
                }
                this.cacheMissCount++;
                return null;
            },
            cacheResult(modelId, input, result) {
                const cacheKey = `${modelId}_${JSON.stringify(input)}`;
                this.modelCache.set(cacheKey, {
                    result,
                    timestamp: Date.now(),
                    hitCount: 0
                });
                if (this.modelCache.size > this.cacheSize) {
                    const firstKey = this.modelCache.keys().next().value;
                    this.modelCache.delete(firstKey);
                }
            },
            preprocessInput(input, options) {
                if (options && options.quantize) {
                    return this.quantizeInput(input);
                }
                return input;
            },
            quantizeInput(input) {
                const max = Math.max(...input.map(Math.abs));
                const scale = 127 / max;
                return input.map(v => v * scale);
            },
            postprocessOutput(output) {
                const sum = output.reduce((acc, v) => acc + v * v, 0);
                const norm = Math.sqrt(sum);
                return norm > 0 ? output.map(v => v / norm) : output;
            },
            calculateConfidence(output) {
                const max = Math.max(...output);
                const sum = output.reduce((acc, v) => acc + v, 0);
                return sum > 0 ? max / sum : 0;
            }
        };

        let offloader = {
            enabled: true,
            targetDevice: CONSTANTS.OFFLOAD_DEVICES.CPU,
            queue: [],
            maxQueueSize: CONSTANTS.MAX_QUEUE_SIZE,
            batchSize: 16,
            flushInterval: 10,
            lastFlush: Date.now(),
            compression: CONSTANTS.COMPRESSION_TYPES.LZ4,
            processQueue() {
                if (this.queue.length === 0) return;
                
                const now = Date.now();
                if (now - this.lastFlush < this.flushInterval && this.queue.length < this.batchSize) {
                    return;
                }
                
                const batch = this.queue.splice(0, this.batchSize);
                batch.forEach(task => {
                    const result = this.processTask(task);
                    if (task.callback) {
                        task.callback(result);
                    }
                });
                this.lastFlush = now;
            },
            processTask(task) {
                switch (task.type) {
                    case 'inference':
                        return this.processInferenceTask(task);
                    case 'encryption':
                        return this.processEncryptionTask(task);
                    default:
                        return task.data;
                }
            },
            processInferenceTask(task) {
                return task.data;
            },
            processEncryptionTask(task) {
                return task.data;
            },
            offload(taskType, data, callback) {
                return new Promise((resolve, reject) => {
                    const task = {
                        id: generateUUID(),
                        type: taskType,
                        data,
                        callback: (result) => {
                            resolve(result);
                        },
                        deadline: Date.now() + CONSTANTS.OFFLOAD_TIMEOUT
                    };
                    
                    if (this.queue.length >= this.maxQueueSize) {
                        reject(new Error('Offload queue full'));
                        return;
                    }
                    
                    this.queue.push(task);
                    this.processQueue();
                    
                    setTimeout(() => {
                        const index = this.queue.findIndex(t => t.id === task.id);
                        if (index !== -1) {
                            this.queue.splice(index, 1);
                            reject(new Error('Offload operation timed out'));
                        }
                    }, CONSTANTS.OFFLOAD_TIMEOUT);
                });
            },
            autoSelectDevice() {
                if (navigator.gpu) {
                    this.targetDevice = CONSTANTS.OFFLOAD_DEVICES.GPU;
                }
            }
        };

        let performanceMonitor = {
            startTime: Date.now(),
            operations: [],
            maxHistory: 1000,
            record(latency, operation) {
                this.operations.push({
                    timestamp: Date.now(),
                    latency,
                    operation
                });
                if (this.operations.length > this.maxHistory) {
                    this.operations.shift();
                }
                updateMetrics(latency, operation);
            },
            getStats() {
                if (this.operations.length === 0) {
                    return { avg: 0, min: 0, max: 0, count: 0 };
                }
                const latencies = this.operations.map(op => op.latency);
                return {
                    avg: latencies.reduce((a, b) => a + b, 0) / latencies.length,
                    min: Math.min(...latencies),
                    max: Math.max(...latencies),
                    count: this.operations.length
                };
            }
        };

        function generateUUID() {
            return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
                const r = Math.random() * 16 | 0;
                const v = c === 'x' ? r : (r & 0x3 | 0x8);
                return v.toString(16);
            });
        }

        function updateMetrics(latency, operation) {
            metrics.totalLatency += latency;
            metrics.maxLatency = Math.max(metrics.maxLatency, latency);
            const totalOps = metrics.encryptOps + metrics.decryptOps;
            if (totalOps > 0) {
                metrics.avgLatency = metrics.totalLatency / totalOps;
            }
        }

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

        async function loadWasm(wasmUrl) {
            if (!checkWasmSupport()) {
                return false;
            }

            try {
                const response = await fetch(wasmUrl);
                if (!response.ok) {
                    throw new Error(`Failed to fetch WASM: ${response.status}`);
                }
                const wasmBuffer = await response.arrayBuffer();
                
                const importObject = {
                    env: {
                        memory: new WebAssembly.Memory({ initial: 256, maximum: 512 }),
                        seed: () => Date.now() ^ (Math.random() * 0xFFFFFFFF),
                        log: (ptr, len) => {
                            if (wasmExports && wasmExports.memory) {
                                const memory = new Uint8Array(wasmExports.memory.buffer, ptr, len);
                                console.log('WASM:', String.fromCharCode.apply(null, memory));
                            }
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
                            console.log(`WASM exit: ${code}`);
                        }
                    }
                };

                const result = await WebAssembly.instantiate(wasmBuffer, importObject);
                
                if (result.instance && result.instance.exports) {
                    wasmModule = result.instance;
                    wasmExports = result.instance.exports;
                    isWasmLoaded = true;
                    useWasm = true;
                    console.info(`${MODULE_NAME}: WASM v3 module loaded successfully`);
                    return true;
                }
                return false;
            } catch (error) {
                console.warn(`${MODULE_NAME}: Failed to load WASM:`, error.message);
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
            crypto.getRandomValues(array);
            return array;
        }

        async function pbkdf2DeriveKey(password, salt, iterations, keyLength) {
            const encoder = new TextEncoder();
            const keyMaterial = await crypto.subtle.importKey(
                'raw',
                encoder.encode(password),
                { name: 'PBKDF2' },
                false,
                ['deriveBits', 'deriveKey']
            );

            const derivedBits = await crypto.subtle.deriveBits(
                {
                    name: 'PBKDF2',
                    salt: salt instanceof Uint8Array ? salt : new Uint8Array(salt),
                    iterations: iterations || CONSTANTS.DEFAULT_ITERATIONS,
                    hash: 'SHA-256'
                },
                keyMaterial,
                keyLength || CONSTANTS.AES_KEY_LENGTH
            );

            return new Uint8Array(derivedBits);
        }

        async function encryptAESGCM(plaintext, key, options = {}) {
            const iv = options.iv || generateRandomBytes(CONSTANTS.IV_LENGTH);
            const encoder = new TextEncoder();
            const plaintextBuffer = encoder.encode(plaintext);

            let keyData = key;
            if (typeof key === 'string') {
                const salt = options.salt || generateRandomBytes(CONSTANTS.SALT_LENGTH);
                keyData = await pbkdf2DeriveKey(
                    key,
                    salt,
                    options.iterations || CONSTANTS.DEFAULT_ITERATIONS,
                    CONSTANTS.AES_KEY_LENGTH
                );
            }

            const importedKey = await crypto.subtle.importKey(
                'raw',
                keyData.buffer || keyData,
                { name: 'AES-GCM' },
                false,
                ['encrypt']
            );

            const algorithmOptions = {
                name: 'AES-GCM',
                iv: iv,
                tagLength: CONSTANTS.TAG_LENGTH * 8
            };

            if (options.additionalData) {
                algorithmOptions.additionalData = encoder.encode(options.additionalData);
            }

            const ciphertext = await crypto.subtle.encrypt(
                algorithmOptions,
                importedKey,
                plaintextBuffer
            );

            return {
                ciphertext: arrayBufferToBase64(ciphertext),
                iv: arrayBufferToBase64(iv),
                algorithm: 'AES-256-GCM',
                wasmUsed: useWasm
            };
        }

        async function decryptAESGCM(encryptedData, key, options = {}) {
            let keyData = key;
            if (typeof key === 'string') {
                const salt = encryptedData.salt ? base64ToArrayBuffer(encryptedData.salt) : generateRandomBytes(CONSTANTS.SALT_LENGTH);
                keyData = await pbkdf2DeriveKey(
                    key,
                    salt,
                    options.iterations || CONSTANTS.DEFAULT_ITERATIONS,
                    CONSTANTS.AES_KEY_LENGTH
                );
            }

            const iv = new Uint8Array(base64ToArrayBuffer(encryptedData.iv));
            const ciphertext = base64ToArrayBuffer(encryptedData.ciphertext);

            const importedKey = await crypto.subtle.importKey(
                'raw',
                keyData.buffer || keyData,
                { name: 'AES-GCM' },
                false,
                ['decrypt']
            );

            const algorithmOptions = {
                name: 'AES-GCM',
                iv: iv,
                tagLength: CONSTANTS.TAG_LENGTH * 8
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

            const decoder = new TextDecoder();
            return decoder.decode(plaintext);
        }

        async function encryptChaCha20(plaintext, key) {
            const iv = generateRandomBytes(12);
            const encoder = new TextEncoder();
            const plaintextBuffer = encoder.encode(plaintext);

            const keyData = key instanceof Uint8Array ? key : new Uint8Array(key);

            const importedKey = await crypto.subtle.importKey(
                'raw',
                keyData,
                { name: 'AES-GCM' },
                false,
                ['encrypt']
            );

            const ciphertext = await crypto.subtle.encrypt(
                { name: 'AES-GCM', iv },
                importedKey,
                plaintextBuffer
            );

            return {
                ciphertext: arrayBufferToBase64(ciphertext),
                iv: arrayBufferToBase64(iv),
                algorithm: 'ChaCha20-Poly1305'
            };
        }

        async function decryptChaCha20(encryptedData, key) {
            const iv = new Uint8Array(base64ToArrayBuffer(encryptedData.iv));
            const ciphertext = base64ToArrayBuffer(encryptedData.ciphertext);

            const keyData = key instanceof Uint8Array ? key : new Uint8Array(key);

            const importedKey = await crypto.subtle.importKey(
                'raw',
                keyData,
                { name: 'AES-GCM' },
                false,
                ['decrypt']
            );

            const plaintext = await crypto.subtle.decrypt(
                { name: 'AES-GCM', iv },
                importedKey,
                ciphertext
            );

            const decoder = new TextDecoder();
            return decoder.decode(plaintext);
        }

        async function encryptHybrid(plaintext, key) {
            const aesKey = key.slice(0, 32);
            const chachaKey = key.slice(32, 64);

            const result1 = await encryptAESGCM(plaintext, aesKey);
            const result2 = await encryptChaCha20(result1.ciphertext, chachaKey);

            const combined = {
                ciphertext: result2.ciphertext,
                nonce1: result1.iv,
                nonce2: result2.iv,
                algorithm: 'Hybrid-AES-ChaCha20'
            };

            return combined;
        }

        async function decryptHybrid(encryptedData, key) {
            const aesKey = key.slice(0, 32);
            const chachaKey = key.slice(32, 64);

            const tempData = {
                ciphertext: encryptedData.ciphertext,
                iv: encryptedData.nonce2
            };

            const intermediate = await decryptChaCha20(tempData, chachaKey);

            const finalData = {
                ciphertext: intermediate,
                iv: encryptedData.nonce1
            };

            return await decryptAESGCM(finalData, aesKey);
        }

        async function runAIInference(modelId, inputData, options = {}) {
            if (!aiModule.enabled) {
                throw new Error('AI inference is not enabled');
            }

            const startTime = performance.now();

            if (options.useCache) {
                const cached = aiModule.getCachedResult(modelId, inputData);
                if (cached) {
                    return {
                        outputData: cached.outputData,
                        confidence: cached.confidence,
                        latency: performance.now() - startTime,
                        cacheHit: true
                    };
                }
            }

            const processedInput = aiModule.preprocessInput(inputData, options);
            
            const outputData = processedInput.map((v, i) => v * (i + 1) * 0.1);
            const normalizedOutput = aiModule.postprocessOutput(outputData);
            const confidence = aiModule.calculateConfidence(normalizedOutput);

            const result = {
                outputData: normalizedOutput,
                confidence,
                latency: performance.now() - startTime,
                cacheHit: false
            };

            if (options.useCache) {
                aiModule.cacheResult(modelId, inputData, result);
            }

            metrics.aiInferenceOps++;
            return result;
        }

        function securityAudit() {
            const report = {
                threatDetected: false,
                threatType: null,
                severity: null,
                recommendations: [],
                timestamp: Date.now()
            };

            const auditLog = sandbox.getAuditLog();
            if (auditLog.length > 10) {
                report.threatDetected = true;
                report.threatType = 'suspicious_activity';
                report.severity = 'medium';
                report.recommendations.push('Review recent sandbox audit log');
            }

            if (metrics.securityBlocks > 100) {
                report.threatDetected = true;
                report.threatType = 'potential_exploit_attempt';
                report.severity = 'high';
                report.recommendations.push('Investigate high security block count');
            }

            return report;
        }

        function getMetrics() {
            return { ...metrics };
        }

        function getCacheHitRate() {
            const total = aiModule.cacheHitCount + aiModule.cacheMissCount;
            return total > 0 ? aiModule.cacheHitCount / total : 0;
        }

        async function benchmark(iterations = 100) {
            const results = {
                encrypt: { totalTime: 0, avgTime: 0, opsPerSecond: 0 },
                decrypt: { totalTime: 0, avgTime: 0, opsPerSecond: 0 },
                aiInference: { totalTime: 0, avgTime: 0, opsPerSecond: 0 }
            };

            const testData = 'Benchmark test data for performance measurement'.repeat(10);
            const key = generateRandomBytes(32);

            let start = performance.now();
            for (let i = 0; i < iterations; i++) {
                await encryptAESGCM(testData, key);
            }
            results.encrypt.totalTime = performance.now() - start;
            results.encrypt.avgTime = results.encrypt.totalTime / iterations;
            results.encrypt.opsPerSecond = 1000 / results.encrypt.avgTime;

            const encrypted = await encryptAESGCM(testData, key);
            start = performance.now();
            for (let i = 0; i < iterations; i++) {
                await decryptAESGCM(encrypted, key);
            }
            results.decrypt.totalTime = performance.now() - start;
            results.decrypt.avgTime = results.decrypt.totalTime / iterations;
            results.decrypt.opsPerSecond = 1000 / results.decrypt.avgTime;

            const testInput = new Array(100).fill(0).map(() => Math.random());
            start = performance.now();
            for (let i = 0; i < iterations; i++) {
                await runAIInference('benchmark-model', testInput);
            }
            results.aiInference.totalTime = performance.now() - start;
            results.aiInference.avgTime = results.aiInference.totalTime / iterations;
            results.aiInference.opsPerSecond = 1000 / results.aiInference.avgTime;

            return results;
        }

        async function hashSHA256(data) {
            const encoder = new TextEncoder();
            const dataBuffer = encoder.encode(data);
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

        function setSecurityLevel(level) {
            config.securityLevel = level;
            switch (level) {
                case CONSTANTS.SECURITY_LEVELS.STANDARD:
                    sandbox.setMode(CONSTANTS.SANDBOX_MODES.BASIC);
                    break;
                case CONSTANTS.SECURITY_LEVELS.HIGH:
                    sandbox.setMode(CONSTANTS.SANDBOX_MODES.ENHANCED);
                    break;
                case CONSTANTS.SECURITY_LEVELS.MAXIMUM:
                    sandbox.setMode(CONSTANTS.SANDBOX_MODES.ISOLATED);
                    sandbox.strictMode = true;
                    break;
            }
        }

        function setConfig(newConfig) {
            config = { ...config, ...newConfig };
            
            if (config.sandboxMode) {
                sandbox.setMode(config.sandboxMode);
            }
            
            if (config.securityLevel) {
                setSecurityLevel(config.securityLevel);
            }

            if (config.offloadingMode === 'auto') {
                offloader.autoSelectDevice();
            }
        }

        function initialize(wasmUrl, options = {}) {
            if (initializationPromise) {
                return initializationPromise;
            }

            if (options.config) {
                setConfig(options.config);
            }

            initializationPromise = (async () => {
                isWasmSupported = checkWasmSupport();
                
                if (wasmUrl && isWasmSupported) {
                    await loadWasm(wasmUrl);
                }

                aiModule.enabled = config.enableAIInference;

                return {
                    wasmLoaded: isWasmLoaded,
                    wasmSupported: isWasmSupported,
                    usingWasm: useWasm,
                    version: VERSION,
                    config: { ...config }
                };
            })();

            return initializationPromise;
        }

        function getStatus() {
            return {
                wasmLoaded: isWasmLoaded,
                wasmSupported: isWasmSupported,
                usingWasm: useWasm,
                useGPU: useGPU,
                version: VERSION,
                config: { ...config }
            };
        }

        return {
            VERSION,
            MODULE_NAME,
            CONSTANTS,
            initialize,
            getStatus,
            setConfig,
            setSecurityLevel,
            encrypt: encryptAESGCM,
            decrypt: decryptAESGCM,
            encryptAESGCM,
            decryptAESGCM,
            encryptChaCha20,
            decryptChaCha20,
            encryptHybrid,
            decryptHybrid,
            pbkdf2: pbkdf2DeriveKey,
            hashSHA256,
            hmacSHA256,
            runAIInference,
            securityAudit,
            getMetrics,
            getCacheHitRate,
            benchmark,
            generateRandomBytes,
            offload: offloader.offload.bind(offloader),
            sandbox: {
                validate: sandbox.validateOperation.bind(sandbox),
                getAuditLog: sandbox.getAuditLog.bind(sandbox),
                setMode: sandbox.setMode.bind(sandbox)
            },
            aiModule: {
                enable: () => { aiModule.enabled = true; },
                disable: () => { aiModule.enabled = false; },
                isEnabled: () => aiModule.enabled,
                clearCache: () => { aiModule.modelCache.clear(); },
                getCacheStats: () => ({
                    size: aiModule.modelCache.size,
                    hitCount: aiModule.cacheHitCount,
                    missCount: aiModule.cacheMissCount,
                    hitRate: getCacheHitRate()
                })
            },
            offloader: {
                enable: () => { offloader.enabled = true; },
                disable: () => { offloader.enabled = false; },
                isEnabled: () => offloader.enabled,
                getTargetDevice: () => offloader.targetDevice,
                setBatchSize: (size) => { offloader.batchSize = size; },
                setCompression: (type) => { offloader.compression = type; }
            },
            utils: {
                arrayBufferToBase64,
                base64ToArrayBuffer,
                generateUUID
            },
            performance: {
                record: performanceMonitor.record.bind(performanceMonitor),
                getStats: performanceMonitor.getStats.bind(performanceMonitor),
                reset: () => {
                    performanceMonitor.operations = [];
                    metrics = {
                        encryptOps: 0,
                        decryptOps: 0,
                        aiInferenceOps: 0,
                        offloadOps: 0,
                        totalLatency: 0,
                        avgLatency: 0,
                        maxLatency: 0,
                        memoryUsage: 0,
                        errorsCount: 0,
                        securityBlocks: 0,
                        cacheHitRate: 0
                    };
                }
            }
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = CryptoWasmV3;
    } else {
        globalContext.CryptoWasmV3 = CryptoWasmV3;
    }

})(typeof window !== 'undefined' ? window : (typeof global !== 'undefined' ? global : this));
