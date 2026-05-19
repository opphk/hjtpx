(function(global) {
    'use strict';

    const SDK_VERSION = '18.0.0';
    const DEFAULT_TIMEOUT = 30000;
    const DEFAULT_RETRIES = 3;

    class UnifiedSDK {
        constructor(config) {
            this.config = {
                apiKey: config.apiKey || '',
                baseURL: config.baseURL || 'https://api.hjtpx.com',
                timeout: config.timeout || DEFAULT_TIMEOUT,
                retries: config.retries || DEFAULT_RETRIES,
                platform: config.platform || this.detectPlatform(),
                enableCache: config.enableCache !== false,
                enableMetrics: config.enableMetrics !== false,
                plugins: config.plugins || []
            };

            this.cache = new Map();
            this.metrics = {
                requests: 0,
                failures: 0,
                latency: []
            };

            this.initialized = false;
            this.pluginRegistry = new Map();

            this.registerBuiltinPlugins();
            this.loadPlugins(this.config.plugins);
        }

        detectPlatform() {
            if (typeof window !== 'undefined') {
                return 'browser';
            }
            if (typeof process !== 'undefined' && process.versions?.node) {
                return 'node';
            }
            if (typeof uni !== 'undefined' || typeof wx !== 'undefined') {
                return 'miniapp';
            }
            if (typeof ReactNative !== 'undefined') {
                return 'reactnative';
            }
            return 'unknown';
        }

        async initialize() {
            if (this.initialized) return this;

            for (const [name, plugin] of this.pluginRegistry) {
                if (plugin.onInitialize) {
                    await plugin.onInitialize(this.config);
                }
            }

            if (this.config.enableCache) {
                this.setupCache();
            }

            this.initialized = true;
            return this;
        }

        registerBuiltinPlugins() {
            this.registerPlugin('metrics', {
                onInitialize: () => {},
                beforeRequest: (config) => {
                    config._startTime = Date.now();
                    return config;
                },
                afterResponse: (config, response) => {
                    if (config._startTime) {
                        const latency = Date.now() - config._startTime;
                        this.metrics.latency.push(latency);
                        this.metrics.requests++;
                    }
                    return response;
                },
                onError: (config, error) => {
                    this.metrics.failures++;
                    return error;
                }
            });

            this.registerPlugin('cache', {
                onInitialize: () => {},
                shouldCache: (config) => {
                    return config.method === 'GET' && config.cache !== false;
                },
                getCacheKey: (config) => {
                    return `${config.method}:${config.url}:${JSON.stringify(config.params || {})}`;
                },
                getCached: (key) => {
                    const cached = this.cache.get(key);
                    if (cached && Date.now() - cached.timestamp < (config.cacheTTL || 300000)) {
                        return cached.data;
                    }
                    return null;
                },
                setCache: (key, data) => {
                    this.cache.set(key, {
                        data,
                        timestamp: Date.now()
                    });
                }
            });

            this.registerPlugin('retry', {
                onInitialize: () => {},
                shouldRetry: (error, attempt) => {
                    if (attempt >= this.config.retries) return false;
                    const status = error?.status || error?.response?.status;
                    return status >= 500 || status === 429 || !status;
                },
                getRetryDelay: (attempt) => {
                    return Math.min(1000 * Math.pow(2, attempt), 30000);
                }
            });
        }

        registerPlugin(name, plugin) {
            this.pluginRegistry.set(name, plugin);
        }

        loadPlugins(plugins) {
            plugins.forEach(plugin => {
                if (typeof plugin === 'string') {
                    try {
                        const loadedPlugin = this.loadPluginFromString(plugin);
                        if (loadedPlugin) {
                            this.registerPlugin(plugin, loadedPlugin);
                        }
                    } catch (e) {
                        console.warn(`Failed to load plugin: ${plugin}`, e);
                    }
                } else if (plugin.name && plugin) {
                    this.registerPlugin(plugin.name, plugin);
                }
            });
        }

        loadPluginFromString(name) {
            const plugins = {
                'analytics': this.createAnalyticsPlugin(),
                'compression': this.createCompressionPlugin(),
                'encryption': this.createEncryptionPlugin()
            };
            return plugins[name];
        }

        createAnalyticsPlugin() {
            return {
                onInitialize: () => {},
                trackEvent: (name, data) => {
                    console.log(`[Analytics] Event: ${name}`, data);
                }
            };
        }

        createCompressionPlugin() {
            return {
                onInitialize: () => {},
                compress: (data) => {
                    return data;
                },
                decompress: (data) => {
                    return data;
                }
            };
        }

        createEncryptionPlugin() {
            return {
                onInitialize: () => {},
                encrypt: (data, key) => {
                    return btoa(JSON.stringify(data));
                },
                decrypt: (data, key) => {
                    return JSON.parse(atob(data));
                }
            };
        }

        setupCache() {
            if (this.platform === 'browser') {
                try {
                    const cacheSize = localStorage.getItem('hjtpx_cache_size') || 0;
                    if (parseInt(cacheSize) > 50 * 1024 * 1024) {
                        localStorage.removeItem('hjtpx_cache');
                        localStorage.setItem('hjtpx_cache_size', '0');
                    }
                } catch (e) {
                    console.warn('Cache setup failed:', e);
                }
            }
        }

        async request(config) {
            let processedConfig = { ...config };

            for (const [name, plugin] of this.pluginRegistry) {
                if (plugin.beforeRequest) {
                    processedConfig = plugin.beforeRequest(processedConfig, this.config) || processedConfig;
                }
            }

            try {
                const response = await this.executeRequest(processedConfig);

                for (const [name, plugin] of this.pluginRegistry) {
                    if (plugin.afterResponse) {
                        plugin.afterResponse(processedConfig, response, this.config);
                    }
                }

                return response;
            } catch (error) {
                let processedError = error;

                for (const [name, plugin] of this.pluginRegistry) {
                    if (plugin.onError) {
                        processedError = plugin.onError(processedConfig, processedError, this.config) || processedError;
                    }
                }

                const retryPlugin = this.pluginRegistry.get('retry');
                if (retryPlugin?.shouldRetry?.(processedError, 0)) {
                    return this.executeWithRetry(processedConfig, retryPlugin);
                }

                throw processedError;
            }
        }

        async executeRequest(config) {
            const url = this.buildURL(config);
            const options = this.buildRequestOptions(config);

            if (this.platform === 'node') {
                return this.executeNodeRequest(url, options);
            } else if (this.platform === 'browser' || this.platform === 'reactnative') {
                return this.executeBrowserRequest(url, options);
            } else {
                return this.executeDefaultRequest(url, options);
            }
        }

        buildURL(config) {
            let url = config.url;
            if (!url.startsWith('http')) {
                url = this.config.baseURL + url;
            }

            if (config.params) {
                const queryString = Object.entries(config.params)
                    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
                    .join('&');
                url += (url.includes('?') ? '&' : '?') + queryString;
            }

            return url;
        }

        buildRequestOptions(config) {
            const options = {
                method: config.method || 'GET',
                headers: {
                    'Content-Type': 'application/json',
                    'X-SDK-Version': SDK_VERSION,
                    'X-Platform': this.platform,
                    ...config.headers
                }
            };

            if (this.config.apiKey) {
                options.headers['Authorization'] = `Bearer ${this.config.apiKey}`;
            }

            if (config.body && (config.method === 'POST' || config.method === 'PUT' || config.method === 'PATCH')) {
                options.body = typeof config.body === 'string' ? config.body : JSON.stringify(config.body);
            }

            return options;
        }

        async executeNodeRequest(url, options) {
            const https = require('https');
            const http = require('http');
            const urlModule = require('url');

            return new Promise((resolve, reject) => {
                const parsedUrl = new URL(url);
                const client = parsedUrl.protocol === 'https:' ? https : http;

                const reqOptions = {
                    hostname: parsedUrl.hostname,
                    port: parsedUrl.port || (parsedUrl.protocol === 'https:' ? 443 : 80),
                    path: parsedUrl.pathname + parsedUrl.search,
                    method: options.method,
                    headers: options.headers
                };

                const req = client.request(reqOptions, (res) => {
                    let data = '';
                    res.on('data', chunk => data += chunk);
                    res.on('end', () => {
                        try {
                            resolve({
                                status: res.statusCode,
                                data: JSON.parse(data),
                                headers: res.headers
                            });
                        } catch {
                            resolve({
                                status: res.statusCode,
                                data: data,
                                headers: res.headers
                            });
                        }
                    });
                });

                req.on('error', reject);
                req.setTimeout(this.config.timeout, () => {
                    req.destroy();
                    reject(new Error('Request timeout'));
                });

                if (options.body) {
                    req.write(options.body);
                }
                req.end();
            });
        }

        async executeBrowserRequest(url, options) {
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

            try {
                const response = await fetch(url, {
                    ...options,
                    signal: controller.signal
                });

                clearTimeout(timeoutId);

                const data = await response.json().catch(() => response.text());

                return {
                    status: response.status,
                    data,
                    headers: Object.fromEntries(response.headers.entries())
                };
            } catch (error) {
                clearTimeout(timeoutId);
                throw error;
            }
        }

        async executeDefaultRequest(url, options) {
            return this.executeBrowserRequest(url, options);
        }

        async executeWithRetry(config, retryPlugin, attempt = 0) {
            const delay = retryPlugin.getRetryDelay(attempt);

            await new Promise(resolve => setTimeout(resolve, delay));

            try {
                return await this.executeRequest(config);
            } catch (error) {
                if (retryPlugin.shouldRetry(error, attempt + 1)) {
                    return this.executeWithRetry(config, retryPlugin, attempt + 1);
                }
                throw error;
            }
        }

        async getCaptcha(config) {
            return this.request({
                method: 'GET',
                url: '/api/captcha/create',
                params: config
            });
        }

        async verifyCaptcha(config) {
            return this.request({
                method: 'POST',
                url: '/api/captcha/verify',
                body: config
            });
        }

        async reportRisk(config) {
            return this.request({
                method: 'POST',
                url: '/api/risk/report',
                body: config
            });
        }

        async getDeviceFingerprint() {
            return this.request({
                method: 'GET',
                url: '/api/fingerprint/device'
            });
        }

        async authenticateDevice(config) {
            return this.request({
                method: 'POST',
                url: '/api/iot/device/auth',
                body: config
            });
        }

        async recordBlockchainProof(config) {
            return this.request({
                method: 'POST',
                url: '/api/blockchain/record',
                body: config
            });
        }

        getMetrics() {
            return {
                ...this.metrics,
                cacheSize: this.cache.size,
                avgLatency: this.calculateAvgLatency(),
                successRate: this.calculateSuccessRate()
            };
        }

        calculateAvgLatency() {
            if (this.metrics.latency.length === 0) return 0;
            const sum = this.metrics.latency.reduce((a, b) => a + b, 0);
            return sum / this.metrics.latency.length;
        }

        calculateSuccessRate() {
            if (this.metrics.requests === 0) return 100;
            return ((this.metrics.requests - this.metrics.failures) / this.metrics.requests) * 100;
        }

        clearCache() {
            this.cache.clear();
        }

        destroy() {
            this.clearCache();
            this.pluginRegistry.clear();
            this.initialized = false;
        }
    }

    const createSDK = (config) => {
        return new UnifiedSDK(config);
    };

    createSDK.version = SDK_VERSION;
    createSDK.UnifiedSDK = UnifiedSDK;

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = createSDK;
    } else {
        global.HjtpxSDK = createSDK;
    }

})(typeof window !== 'undefined' ? window : global);
