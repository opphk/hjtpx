(function() {
    'use strict';

    const SDK = require('./unified-sdk');

    describe('Unified SDK Tests', function() {
        this.timeout(10000);

        let sdk;

        beforeEach(function() {
            sdk = SDK({
                apiKey: 'test-api-key',
                baseURL: 'https://api.test.com',
                timeout: 5000,
                retries: 2,
                enableCache: true,
                enableMetrics: true
            });
        });

        afterEach(function() {
            if (sdk) {
                sdk.destroy();
            }
        });

        describe('Initialization', function() {
            it('should create SDK instance', function() {
                sdk.should.exist;
                sdk.config.should.have.property('apiKey', 'test-api-key');
                sdk.config.should.have.property('baseURL', 'https://api.test.com');
            });

            it('should detect platform', function() {
                sdk.config.platform.should.be.oneOf(['browser', 'node', 'reactnative', 'miniapp', 'unknown']);
            });

            it('should initialize successfully', async function() {
                await sdk.initialize();
                sdk.initialized.should.be.true;
            });

            it('should initialize only once', async function() {
                await sdk.initialize();
                const firstInit = sdk.initialized;
                await sdk.initialize();
                sdk.initialized.should.equal(firstInit);
            });
        });

        describe('Plugin System', function() {
            it('should register plugins', function() {
                sdk.registerPlugin('test', {
                    onInitialize: () => {},
                    testMethod: () => 'test'
                });
                sdk.pluginRegistry.should.have.property('test');
            });

            it('should load plugins from array', function() {
                const testSdk = SDK({
                    plugins: []
                });
                testSdk.pluginRegistry.should.have.property('metrics');
                testSdk.pluginRegistry.should.have.property('cache');
                testSdk.pluginRegistry.should.have.property('retry');
                testSdk.destroy();
            });

            it('should call plugin lifecycle methods', async function() {
                let initialized = false;
                sdk.registerPlugin('lifecycle-test', {
                    onInitialize: () => { initialized = true; }
                });
                await sdk.initialize();
                initialized.should.be.true;
            });
        });

        describe('Request Building', function() {
            it('should build correct URL', function() {
                const url = sdk.buildURL({
                    url: '/api/test',
                    params: { id: 123, name: 'test' }
                });
                url.should.include('https://api.test.com/api/test');
                url.should.include('id=123');
                url.should.include('name=test');
            });

            it('should build request options', function() {
                const options = sdk.buildRequestOptions({
                    method: 'POST',
                    body: { data: 'test' }
                });
                options.method.should.equal('POST');
                options.headers.should.have.property('Content-Type', 'application/json');
                options.headers.should.have.property('X-SDK-Version');
            });

            it('should include authorization header', function() {
                const options = sdk.buildRequestOptions({ method: 'GET' });
                options.headers.should.have.property('Authorization', 'Bearer test-api-key');
            });
        });

        describe('Cache System', function() {
            it('should cache GET requests', function() {
                sdk.cache.set('test-key', { data: 'test-data', timestamp: Date.now() });
                sdk.cache.should.have.property('test-key');
            });

            it('should clear cache', function() {
                sdk.cache.set('key1', { data: 'data1' });
                sdk.cache.set('key2', { data: 'data2' });
                sdk.clearCache();
                sdk.cache.size.should.equal(0);
            });
        });

        describe('Metrics', function() {
            it('should track requests', function() {
                const metrics = sdk.getMetrics();
                metrics.should.have.property('requests');
                metrics.should.have.property('failures');
                metrics.should.have.property('latency');
            });

            it('should calculate success rate', function() {
                sdk.metrics.requests = 10;
                sdk.metrics.failures = 2;
                const rate = sdk.calculateSuccessRate();
                rate.should.equal(80);
            });

            it('should calculate average latency', function() {
                sdk.metrics.latency = [100, 200, 300];
                const avg = sdk.calculateAvgLatency();
                avg.should.equal(200);
            });
        });

        describe('URL Building', function() {
            it('should handle absolute URLs', function() {
                const url = sdk.buildURL({
                    url: 'https://other.com/api',
                    params: { key: 'value' }
                });
                url.should.equal('https://other.com/api?key=value');
            });

            it('should handle URLs without params', function() {
                const url = sdk.buildURL({
                    url: '/api/simple'
                });
                url.should.equal('https://api.test.com/api/simple');
            });

            it('should handle URLs with existing query params', function() {
                const url = sdk.buildURL({
                    url: '/api/test?existing=1',
                    params: { new: '2' }
                });
                url.should.include('existing=1');
                url.should.include('new=2');
            });
        });

        describe('Config Merging', function() {
            it('should use default timeout', function() {
                const defaultSdk = SDK({});
                defaultSdk.config.timeout.should.equal(30000);
                defaultSdk.destroy();
            });

            it('should allow custom timeout', function() {
                const customSdk = SDK({ timeout: 60000 });
                customSdk.config.timeout.should.equal(60000);
                customSdk.destroy();
            });

            it('should disable cache when specified', function() {
                const noCacheSdk = SDK({ enableCache: false });
                noCacheSdk.config.enableCache.should.be.false;
                noCacheSdk.destroy();
            });
        });

        describe('Plugin Lifecycle', function() {
            it('should call beforeRequest hook', async function() {
                let called = false;
                sdk.registerPlugin('before-test', {
                    beforeRequest: (config) => {
                        called = true;
                        config.customHeader = 'test';
                        return config;
                    }
                });
                await sdk.initialize();
                called.should.be.true;
            });

            it('should call afterResponse hook', async function() {
                let called = false;
                sdk.registerPlugin('after-test', {
                    afterResponse: () => { called = true; }
                });
                await sdk.initialize();
                called.should.be.true;
            });
        });

        describe('SDK Version', function() {
            it('should export version', function() {
                SDK.version.should.be.a('string');
                SDK.version.should.equal('18.0.0');
            });

            it('should export UnifiedSDK class', function() {
                SDK.UnifiedSDK.should.be.a('function');
            });
        });

        describe('Destroy', function() {
            it('should clear all data on destroy', async function() {
                sdk.cache.set('key', { data: 'value' });
                await sdk.initialize();
                sdk.destroy();
                sdk.cache.size.should.equal(0);
                sdk.initialized.should.be.false;
            });
        });
    });

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = SDK;
    }
})();
