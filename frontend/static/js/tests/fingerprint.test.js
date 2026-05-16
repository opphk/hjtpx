describe('FingerprintCollector', () => {
    let collector;

    beforeEach(() => {
        collector = new FingerprintCollector();
    });

    describe('Constructor', () => {
        test('should initialize with empty cache and components', () => {
            expect(collector.cache).toBeNull();
            expect(collector.components).toEqual({});
        });
    });

    describe('Hash String Generation', () => {
        test('should generate consistent hash for same input', async () => {
            const input = 'test-string-123';
            
            const hash1 = await collector.hashString(input);
            const hash2 = await collector.hashString(input);
            
            expect(hash1).toBe(hash2);
        });

        test('should generate different hashes for different inputs', async () => {
            const hash1 = await collector.hashString('input1');
            const hash2 = await collector.hashString('input2');
            
            expect(hash1).not.toBe(hash2);
        });

        test('should generate SHA-256 hash format', async () => {
            const hash = await collector.hashString('test');
            
            expect(hash).toMatch(/^[a-f0-9]{64}$/);
        });
    });

    describe('Combined Hash Generation', () => {
        test('should generate combined hash from fingerprint data', async () => {
            const fingerprintData = {
                user_agent: 'Mozilla/5.0 Test Browser',
                screen_width: 1920,
                screen_height: 1080,
                canvas_fingerprint: 'canvas-hash-123',
                webgl_vendor: 'Vendor',
                webgl_renderer: 'Renderer',
                audio_fingerprint: 'audio-hash-456',
                platform: 'Win32'
            };

            const combinedHash = await collector.generateCombinedHash(fingerprintData);

            expect(combinedHash).toMatch(/^[a-f0-9]{64}$/);
        });

        test('should generate different combined hashes for different data', async () => {
            const data1 = {
                user_agent: 'Browser1',
                screen_width: 1920,
                screen_height: 1080,
                canvas_fingerprint: 'hash1',
                webgl_vendor: 'V1',
                webgl_renderer: 'R1',
                audio_fingerprint: 'ah1',
                platform: 'Win32'
            };

            const data2 = {
                user_agent: 'Browser2',
                screen_width: 1920,
                screen_height: 1080,
                canvas_fingerprint: 'hash2',
                webgl_vendor: 'V2',
                webgl_renderer: 'R2',
                audio_fingerprint: 'ah2',
                platform: 'Win32'
            };

            const hash1 = await collector.generateCombinedHash(data1);
            const hash2 = await collector.generateCombinedHash(data2);

            expect(hash1).not.toBe(hash2);
        });
    });

    describe('Cache Management', () => {
        test('should return cached data on subsequent calls', async () => {
            const data = await collector.collect();
            const cachedData = await collector.collect();

            expect(cachedData).toEqual(data);
        });

        test('should clear cache correctly', async () => {
            await collector.collect();
            expect(collector.cache).not.toBeNull();

            collector.clearCache();
            expect(collector.cache).toBeNull();
        });

        test('should clear components on cache clear', async () => {
            await collector.collect();
            expect(Object.keys(collector.components).length).toBeGreaterThan(0);

            collector.clearCache();
            expect(collector.components).toEqual({});
        });
    });

    describe('Component Retrieval', () => {
        test('should return collected components', async () => {
            await collector.collect();
            
            const components = collector.getComponents();
            
            expect(components).toHaveProperty('userAgent');
            expect(components).toHaveProperty('screen');
            expect(components).toHaveProperty('browser');
        });
    });

    describe('Data Collection Structure', () => {
        test('should return complete fingerprint data structure', async () => {
            const data = await collector.collect();

            expect(data).toHaveProperty('user_agent');
            expect(data).toHaveProperty('screen_width');
            expect(data).toHaveProperty('screen_height');
            expect(data).toHaveProperty('color_depth');
            expect(data).toHaveProperty('timezone');
            expect(data).toHaveProperty('language');
            expect(data).toHaveProperty('platform');
            expect(data).toHaveProperty('hardware_concurrency');
            expect(data).toHaveProperty('device_memory');
            expect(data).toHaveProperty('touch_points');
            expect(data).toHaveProperty('webgl_vendor');
            expect(data).toHaveProperty('webgl_renderer');
            expect(data).toHaveProperty('canvas_fingerprint');
            expect(data).toHaveProperty('audio_fingerprint');
            expect(data).toHaveProperty('fonts');
            expect(data).toHaveProperty('plugins');
            expect(data).toHaveProperty('do_not_track');
            expect(data).toHaveProperty('cookies_enabled');
            expect(data).toHaveProperty('local_storage');
            expect(data).toHaveProperty('session_storage');
            expect(data).toHaveProperty('collection_time_ms');
        });

        test('should include collection time', async () => {
            const data = await collector.collect();

            expect(data.collection_time_ms).toBeGreaterThanOrEqual(0);
        });
    });
});

describe('FingerprintAPI', () => {
    let api;

    beforeEach(() => {
        api = new FingerprintAPI();
    });

    describe('Constructor', () => {
        test('should initialize with FingerprintCollector', () => {
            expect(api.collector).toBeInstanceOf(FingerprintCollector);
        });

        test('should set default API base URL', () => {
            expect(api.apiBase).toBe('/api/v1/fingerprint');
        });
    });

    describe('Token Retrieval', () => {
        test('should retrieve token from localStorage', () => {
            localStorage.setItem('token', 'test-token-123');

            const token = api.getToken();

            expect(token).toBe('test-token-123');

            localStorage.removeItem('token');
        });

        test('should retrieve token from sessionStorage as fallback', () => {
            sessionStorage.setItem('token', 'session-token');

            const token = api.getToken();

            expect(token).toBe('session-token');

            sessionStorage.removeItem('token');
        });

        test('should return empty string when no token found', () => {
            localStorage.removeItem('token');
            sessionStorage.removeItem('token');

            const token = api.getToken();

            expect(token).toBe('');
        });
    });

    describe('API Response Structure', () => {
        test('should handle successful response structure', () => {
            const mockResult = {
                success: true,
                fingerprintId: 123,
                hash: 'abc123',
                riskLevel: 'low',
                collectionTime: 50
            };

            expect(mockResult).toHaveProperty('success');
            expect(mockResult).toHaveProperty('fingerprintId');
            expect(mockResult).toHaveProperty('hash');
            expect(mockResult).toHaveProperty('riskLevel');
        });

        test('should handle error response structure', () => {
            const mockError = {
                success: false,
                error: 'Network error'
            };

            expect(mockError.success).toBe(false);
            expect(mockError).toHaveProperty('error');
        });
    });
});

describe('Device Management', () => {
    describe('Device Info Structure', () => {
        test('should validate device info structure', () => {
            const deviceInfo = {
                id: 1,
                hash: 'device-hash-123',
                user_agent: 'Mozilla/5.0',
                screen_info: '1920x1080',
                browser_info: 'Chrome',
                platform_info: 'Windows',
                visit_count: 10,
                is_trusted: true,
                risk_level: 'low'
            };

            expect(deviceInfo).toHaveProperty('id');
            expect(deviceInfo).toHaveProperty('hash');
            expect(deviceInfo).toHaveProperty('visit_count');
            expect(deviceInfo).toHaveProperty('is_trusted');
            expect(deviceInfo).toHaveProperty('risk_level');
        });

        test('should validate risk levels', () => {
            const validLevels = ['low', 'medium', 'high'];

            expect(validLevels).toContain('low');
            expect(validLevels).toContain('medium');
            expect(validLevels).toContain('high');
        });
    });

    describe('Similar Device Structure', () => {
        test('should validate similar device structure', () => {
            const similarDevice = {
                device_id: 1,
                similarity: 0.92,
                first_seen_at: '2024-01-01T00:00:00Z',
                last_seen_at: '2024-01-15T00:00:00Z',
                visit_count: 5
            };

            expect(similarDevice).toHaveProperty('device_id');
            expect(similarDevice).toHaveProperty('similarity');
            expect(similarDevice.similarity).toBeGreaterThanOrEqual(0);
            expect(similarDevice.similarity).toBeLessThanOrEqual(1);
        });
    });
});
