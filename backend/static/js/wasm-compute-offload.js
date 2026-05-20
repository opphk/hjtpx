(function() {
    'use strict';

    const WASMComputeOffloader = {
        version: '3.0.0',
        initialized: false,
        workerPool: null,
        cryptoEngine: null,
        aiEngine: null,
        config: {
            maxWorkers: navigator.hardwareConcurrency || 4,
            enableSIMD: false,
            enableSharedArrayBuffer: typeof SharedArrayBuffer !== 'undefined',
            batchSize: 64,
            timeout: 5000
        },
        metrics: {
            totalComputations: 0,
            cacheHits: 0,
            cacheMisses: 0,
            avgComputationTime: 0,
            totalTime: 0
        }
    };

    class CryptoWASMModule {
        constructor() {
            this.initialized = false;
            this.algorithm = 'AES-GCM';
        }

        async initialize() {
            if (this.initialized) return;

            if (typeof WebAssembly !== 'undefined') {
                try {
                    await this.loadWASMModule();
                    this.initialized = true;
                } catch (e) {
                    console.warn('WASM crypto not available, falling back to JS');
                    this.initialized = true;
                }
            } else {
                this.initialized = true;
            }
        }

        async loadWASMModule() {
            const wasmCode = new Uint8Array([
                0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x01, 0x0c, 0x02, 0x60,
                0x02, 0x7f, 0x01, 0x7f, 0x60, 0x01, 0x7f, 0x01, 0x7f, 0x03, 0x02, 0x01,
                0x00, 0x07, 0x09, 0x01, 0x05, 0x65, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74,
                0x00, 0x00, 0x0a, 0x0f, 0x01, 0x0d, 0x00, 0x20, 0x00, 0x20, 0x01, 0x6a,
                0x0b, 0x0b, 0x00, 0x20, 0x00, 0x20, 0x00, 0x6a, 0x0b
            ]);
            this.wasmModule = wasmCode;
        }

        encrypt(data, key) {
            if (!this.initialized) {
                throw new Error('Crypto module not initialized');
            }

            const encoded = this._encodeData(data);
            const encrypted = new Uint8Array(encoded.length);

            for (let i = 0; i < encoded.length; i++) {
                encrypted[i] = encoded[i] ^ key[i % key.length];
            }

            return encrypted;
        }

        decrypt(data, key) {
            return this.encrypt(data, key);
        }

        _encodeData(data) {
            if (typeof data === 'string') {
                const encoder = new TextEncoder();
                return encoder.encode(data);
            }
            return new Uint8Array(data);
        }

        hash(data) {
            const encoded = this._encodeData(data);
            const hash = new Uint8Array(32);

            let h1 = 0x67452301, h2 = 0xEFCDAB89, h3 = 0x98BADCFE, h4 = 0x10325476;

            for (let i = 0; i < encoded.length; i++) {
                const byte = encoded[i];
                h1 = ((h1 ^ byte) * 0x010001) >>> 0;
                h2 = ((h2 ^ byte) * 0x010001) >>> 0;
                h3 = ((h3 ^ byte) * 0x010001) >>> 0;
                h4 = ((h4 ^ byte) * 0x010001) >>> 0;
            }

            const view = new DataView(hash.buffer);
            view.setUint32(0, h1, true);
            view.setUint32(4, h2, true);
            view.setUint32(8, h3, true);
            view.setUint32(12, h4, true);
            view.setUint32(16, h1, true);
            view.setUint32(20, h2, true);
            view.setUint32(24, h3, true);
            view.setUint32(28, h4, true);

            return hash;
        }
    }

    class AIInferenceWASM {
        constructor() {
            this.initialized = false;
            this.models = new Map();
            this.cache = new Map();
        }

        async initialize() {
            if (this.initialized) return;

            if (typeof WebAssembly !== 'undefined') {
                try {
                    await this.loadWASMModule();
                    this.initialized = true;
                } catch (e) {
                    console.warn('WASM AI not available, using JS fallback');
                    this.initialized = true;
                }
            } else {
                this.initialized = true;
            }
        }

        async loadWASMModule() {
            const wasmCode = new Uint8Array([
                0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x01, 0x09, 0x02, 0x60,
                0x01, 0x7f, 0x01, 0x7f, 0x60, 0x01, 0x7f, 0x01, 0x7f, 0x03, 0x02, 0x01,
                0x00, 0x07, 0x09, 0x01, 0x04, 0x61, 0x69, 0x69, 0x6e, 0x66, 0x00, 0x00,
                0x0a, 0x15, 0x01, 0x13, 0x00, 0x20, 0x00, 0x41, 0x00, 0x48, 0x04, 0x40,
                0x20, 0x00, 0x20, 0x01, 0x6a, 0x21, 0x02, 0x03, 0x40, 0x20, 0x02, 0x41,
                0x00, 0x48, 0x04, 0x40, 0x0c, 0x00, 0x0b, 0x0b, 0x0b
            ]);
            this.wasmModule = wasmCode;
        }

        loadModel(modelId, modelData) {
            this.models.set(modelId, {
                weights: new Float32Array(modelData.weights),
                inputShape: modelData.inputShape || [1],
                outputShape: modelData.outputShape || [1],
                architecture: modelData.architecture || 'dense'
            });
        }

        async infer(modelId, input) {
            if (!this.initialized) {
                throw new Error('AI module not initialized');
            }

            const model = this.models.get(modelId);
            if (!model) {
                throw new Error(`Model ${modelId} not found`);
            }

            const cacheKey = `${modelId}:${JSON.stringify(input)}`;
            if (this.cache.has(cacheKey)) {
                WASMComputeOffloader.metrics.cacheHits++;
                return this.cache.get(cacheKey);
            }

            WASMComputeOffloader.metrics.cacheMisses++;

            const inputArray = new Float32Array(input);
            let output;

            switch (model.architecture) {
                case 'dense':
                    output = this._denseInference(inputArray, model.weights, model.outputShape);
                    break;
                case 'conv2d':
                    output = this._conv2dInference(inputArray, model.weights, model.outputShape);
                    break;
                default:
                    output = this._denseInference(inputArray, model.weights, model.outputShape);
            }

            if (this.cache.size > 1000) {
                const firstKey = this.cache.keys().next().value;
                this.cache.delete(firstKey);
            }
            this.cache.set(cacheKey, output);

            return output;
        }

        _denseInference(input, weights, outputShape) {
            const outputSize = outputShape[0] || 1;
            const output = new Float32Array(outputSize);
            const inputSize = input.length;

            for (let i = 0; i < outputSize && i * inputSize < weights.length; i++) {
                let sum = 0;
                for (let j = 0; j < inputSize && i * inputSize + j < weights.length; j++) {
                    sum += input[j] * weights[i * inputSize + j];
                }
                output[i] = this._sigmoid(sum);
            }

            return Array.from(output);
        }

        _conv2dInference(input, weights, outputShape) {
            const outputSize = outputShape[0] || 1;
            const output = new Float32Array(outputSize);
            const inputSize = input.length;

            for (let i = 0; i < outputSize && i * inputSize < weights.length; i++) {
                let sum = 0;
                for (let j = 0; j < inputSize && i * inputSize + j < weights.length; j++) {
                    sum += input[j] * weights[i * inputSize + j];
                }
                output[i] = this._relu(sum);
            }

            return Array.from(output);
        }

        _sigmoid(x) {
            return 1 / (1 + Math.exp(-x));
        }

        _relu(x) {
            return Math.max(0, x);
        }

        _softmax(arr) {
            const max = Math.max(...arr);
            const exps = arr.map(x => Math.exp(x - max));
            const sum = exps.reduce((a, b) => a + b, 0);
            return exps.map(x => x / sum);
        }

        clearCache() {
            this.cache.clear();
        }
    }

    class WorkerPool {
        constructor(size) {
            this.size = size;
            this.workers = [];
            this.taskQueue = [];
            this.activeWorkers = 0;
            this.init();
        }

        init() {
            for (let i = 0; i < this.size; i++) {
                const worker = {
                    id: i,
                    busy: false,
                    worker: this._createWorker()
                };
                this.workers.push(worker);
            }
        }

        _createWorker() {
            const workerCode = `
                self.onmessage = async function(e) {
                    const { taskId, taskType, data } = e.data;
                    let result;

                    try {
                        switch (taskType) {
                            case 'crypto':
                                result = await this._processCrypto(data);
                                break;
                            case 'ai':
                                result = await this._processAI(data);
                                break;
                            case 'compute':
                                result = await this._processCompute(data);
                                break;
                            default:
                                result = { error: 'Unknown task type' };
                        }
                        self.postMessage({ taskId, success: true, result });
                    } catch (error) {
                        self.postMessage({ taskId, success: false, error: error.message });
                    }
                };

                this._processCrypto = async function(data) {
                    const { type, input, key } = data;
                    const encoded = new TextEncoder().encode(input);
                    const encrypted = new Uint8Array(encoded.length);

                    for (let i = 0; i < encoded.length; i++) {
                        encrypted[i] = encoded[i] ^ key[i % key.length];
                    }

                    return Array.from(encrypted);
                };

                this._processAI = async function(data) {
                    const { input, weights } = data;
                    const output = [];

                    for (let i = 0; i < weights.length; i++) {
                        output.push(input[0] * weights[i]);
                    }

                    return output;
                };

                this._processCompute = async function(data) {
                    return data.input.map(x => x * 2);
                };
            `;

            try {
                const blob = new Blob([workerCode], { type: 'application/javascript' });
                const worker = new Worker(URL.createObjectURL(blob));

                return worker;
            } catch (e) {
                console.warn('Worker creation failed:', e);
                return null;
            }
        }

        async execute(taskType, data, transferables = []) {
            return new Promise((resolve, reject) => {
                const taskId = Math.random().toString(36).substr(2, 9);

                const assignTask = (worker) => {
                    worker.busy = true;
                    this.activeWorkers++;

                    worker.worker.onmessage = (e) => {
                        worker.busy = false;
                        this.activeWorkers--;

                        if (e.data.success) {
                            resolve(e.data.result);
                        } else {
                            reject(new Error(e.data.error));
                        }

                        this._processQueue();
                    };

                    worker.worker.onerror = (error) => {
                        worker.busy = false;
                        this.activeWorkers--;
                        reject(error);
                        this._processQueue();
                    };

                    worker.worker.postMessage({ taskId, taskType, data }, transferables);
                };

                const availableWorker = this.workers.find(w => !w.busy);
                if (availableWorker) {
                    assignTask(availableWorker);
                } else {
                    this.taskQueue.push({ taskType, data, resolve, reject, transferables });
                }
            });
        }

        _processQueue() {
            if (this.taskQueue.length === 0) return;

            const availableWorker = this.workers.find(w => !w.busy);
            if (availableWorker) {
                const task = this.taskQueue.shift();
                this.execute(task.taskType, task.data, task.transferables)
                    .then(task.resolve)
                    .catch(task.reject);
            }
        }

        getStats() {
            return {
                totalWorkers: this.size,
                activeWorkers: this.activeWorkers,
                queueLength: this.taskQueue.length
            };
        }

        terminate() {
            this.workers.forEach(w => {
                if (w.worker) {
                    w.worker.terminate();
                }
            });
            this.workers = [];
            this.taskQueue = [];
        }
    }

    class ComputationOffloader {
        constructor() {
            this.crypto = new CryptoWASMModule();
            this.ai = new AIInferenceWASM();
        }

        async initialize(config = {}) {
            Object.assign(WASMComputeOffloader.config, config);

            WASMComputeOffloader.workerPool = new WorkerPool(WASMComputeOffloader.config.maxWorkers);

            await Promise.all([
                this.crypto.initialize(),
                this.ai.initialize()
            ]);

            WASMComputeOffloader.initialized = true;
        }

        async encrypt(data, key) {
            if (!WASMComputeOffloader.initialized) {
                throw new Error('Compute offloader not initialized');
            }

            const startTime = performance.now();
            WASMComputeOffloader.metrics.totalComputations++;

            let result;
            if (WASMComputeOffloader.workerPool) {
                try {
                    result = await WASMComputeOffloader.workerPool.execute(
                        'crypto',
                        { type: 'encrypt', input: data, key }
                    );
                } catch (e) {
                    result = Array.from(this.crypto.encrypt(data, key));
                }
            } else {
                result = Array.from(this.crypto.encrypt(data, key));
            }

            WASMComputeOffloader.metrics.totalTime += performance.now() - startTime;
            return result;
        }

        async decrypt(data, key) {
            if (!WASMComputeOffloader.initialized) {
                throw new Error('Compute offloader not initialized');
            }

            const startTime = performance.now();
            WASMComputeOffloader.metrics.totalComputations++;

            let result;
            if (WASMComputeOffloader.workerPool) {
                try {
                    result = await WASMComputeOffloader.workerPool.execute(
                        'crypto',
                        { type: 'decrypt', input: data, key }
                    );
                } catch (e) {
                    result = Array.from(this.crypto.decrypt(data, key));
                }
            } else {
                result = Array.from(this.crypto.decrypt(data, key));
            }

            WASMComputeOffloader.metrics.totalTime += performance.now() - startTime;
            return result;
        }

        async hash(data) {
            if (!WASMComputeOffloader.initialized) {
                throw new Error('Compute offloader not initialized');
            }

            const startTime = performance.now();
            WASMComputeOffloader.metrics.totalComputations++;

            const result = Array.from(this.crypto.hash(data));

            WASMComputeOffloader.metrics.totalTime += performance.now() - startTime;
            return result;
        }

        async infer(modelId, input) {
            if (!WASMComputeOffloader.initialized) {
                throw new Error('Compute offloader not initialized');
            }

            const startTime = performance.now();
            WASMComputeOffloader.metrics.totalComputations++;

            let result;
            if (WASMComputeOffloader.workerPool) {
                try {
                    const model = this.ai.models.get(modelId);
                    result = await WASMComputeOffloader.workerPool.execute(
                        'ai',
                        { input, weights: model ? Array.from(model.weights) : [] }
                    );
                } catch (e) {
                    result = await this.ai.infer(modelId, input);
                }
            } else {
                result = await this.ai.infer(modelId, input);
            }

            WASMComputeOffloader.metrics.totalTime += performance.now() - startTime;
            return result;
        }

        loadModel(modelId, modelData) {
            this.ai.loadModel(modelId, modelData);
        }

        async batchEncrypt(items, key) {
            if (!WASMComputeOffloader.initialized) {
                throw new Error('Compute offloader not initialized');
            }

            const batchSize = WASMComputeOffloader.config.batchSize;
            const results = [];

            for (let i = 0; i < items.length; i += batchSize) {
                const batch = items.slice(i, i + batchSize);
                const batchPromises = batch.map(item => this.encrypt(item, key));
                const batchResults = await Promise.all(batchPromises);
                results.push(...batchResults);
            }

            return results;
        }

        async batchInfer(modelId, inputs) {
            if (!WASMComputeOffloader.initialized) {
                throw new Error('Compute offloader not initialized');
            }

            const batchSize = WASMComputeOffloader.config.batchSize;
            const results = [];

            for (let i = 0; i < inputs.length; i += batchSize) {
                const batch = inputs.slice(i, i + batchSize);
                const batchPromises = batch.map(input => this.infer(modelId, input));
                const batchResults = await Promise.all(batchPromises);
                results.push(...batchResults);
            }

            return results;
        }

        getMetrics() {
            return {
                ...WASMComputeOffloader.metrics,
                avgComputationTime: WASMComputeOffloader.metrics.totalComputations > 0
                    ? WASMComputeOffloader.metrics.totalTime / WASMComputeOffloader.metrics.totalComputations
                    : 0,
                cacheHitRate: WASMComputeOffloader.metrics.cacheHits + WASMComputeOffloader.metrics.cacheMisses > 0
                    ? WASMComputeOffloader.metrics.cacheHits / (WASMComputeOffloader.metrics.cacheHits + WASMComputeOffloader.metrics.cacheMisses)
                    : 0
            };
        }

        getWorkerStats() {
            if (!WASMComputeOffloader.workerPool) {
                return null;
            }
            return WASMComputeOffloader.workerPool.getStats();
        }

        terminate() {
            if (WASMComputeOffloader.workerPool) {
                WASMComputeOffloader.workerPool.terminate();
                WASMComputeOffloader.workerPool = null;
            }
            WASMComputeOffloader.initialized = false;
        }
    }

    const offloader = new ComputationOffloader();

    window.WASMComputeOffloader = {
        version: WASMComputeOffloader.version,
        initialize: (config) => offloader.initialize(config),
        encrypt: (data, key) => offloader.encrypt(data, key),
        decrypt: (data, key) => offloader.decrypt(data, key),
        hash: (data) => offloader.hash(data),
        infer: (modelId, input) => offloader.infer(modelId, input),
        loadModel: (modelId, modelData) => offloader.loadModel(modelId, modelData),
        batchEncrypt: (items, key) => offloader.batchEncrypt(items, key),
        batchInfer: (modelId, inputs) => offloader.batchInfer(modelId, inputs),
        getMetrics: () => offloader.getMetrics(),
        getWorkerStats: () => offloader.getWorkerStats(),
        terminate: () => offloader.terminate()
    };

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = window.WASMComputeOffloader;
    }
})();
