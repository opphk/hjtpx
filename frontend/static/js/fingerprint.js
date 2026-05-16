class FingerprintCollector {
    constructor() {
        this.cache = null;
        this.components = {};
    }

    async collect() {
        if (this.cache) {
            return this.cache;
        }

        const startTime = performance.now();

        const components = await Promise.all([
            this.getUserAgent(),
            this.getScreenInfo(),
            this.getBrowserInfo(),
            this.getPlatformInfo(),
            this.getCanvasFingerprint(),
            this.getWebGLFingerprint(),
            this.getAudioFingerprint(),
            this.getFonts(),
            this.getPlugins(),
            this.getStorageInfo(),
            this.getDoNotTrack(),
            this.getTimezone(),
            this.getLanguage(),
            this.getHardwareInfo()
        ]);

        const [
            userAgent,
            screen,
            browser,
            platform,
            canvas,
            webgl,
            audio,
            fonts,
            plugins,
            storage,
            doNotTrack,
            timezone,
            language,
            hardware
        ] = components;

        this.components = {
            userAgent,
            screen,
            browser,
            platform,
            canvas,
            webgl,
            audio,
            fonts,
            plugins,
            storage,
            doNotTrack,
            timezone,
            language,
            hardware
        };

        const collectionTime = performance.now() - startTime;

        this.cache = {
            user_agent: userAgent,
            screen_width: screen.width,
            screen_height: screen.height,
            color_depth: screen.colorDepth,
            timezone: timezone,
            language: language,
            platform: platform,
            hardware_concurrency: hardware.concurrency,
            device_memory: hardware.memory,
            touch_points: screen.touchPoints,
            webgl_vendor: webgl.vendor,
            webgl_renderer: webgl.renderer,
            canvas_fingerprint: canvas.hash,
            audio_fingerprint: audio.hash,
            fonts: fonts.list,
            plugins: plugins.list,
            do_not_track: doNotTrack,
            cookies_enabled: storage.cookies,
            local_storage: storage.localStorage,
            session_storage: storage.sessionStorage,
            collection_time_ms: collectionTime
        };

        return this.cache;
    }

    async getUserAgent() {
        return navigator.userAgent;
    }

    async getScreenInfo() {
        return {
            width: screen.width,
            height: screen.height,
            colorDepth: screen.colorDepth,
            pixelDepth: screen.pixelDepth,
            availWidth: screen.availWidth,
            availHeight: screen.availHeight,
            touchPoints: navigator.maxTouchPoints || 0
        };
    }

    async getBrowserInfo() {
        const ua = navigator.userAgent;
        let browserName = 'Unknown';
        let browserVersion = '0';

        if (ua.indexOf('Firefox') > -1) {
            browserName = 'Firefox';
            browserVersion = ua.match(/Firefox\/([\d.]+)/)[1];
        } else if (ua.indexOf('Chrome') > -1 && ua.indexOf('Edg') === -1) {
            browserName = 'Chrome';
            browserVersion = ua.match(/Chrome\/([\d.]+)/)[1];
        } else if (ua.indexOf('Safari') > -1 && ua.indexOf('Chrome') === -1) {
            browserName = 'Safari';
            browserVersion = ua.match(/Version\/([\d.]+)/)[1];
        } else if (ua.indexOf('Edg') > -1) {
            browserName = 'Edge';
            browserVersion = ua.match(/Edg\/([\d.]+)/)[1];
        } else if (ua.indexOf('Opera') > -1 || ua.indexOf('OPR') > -1) {
            browserName = 'Opera';
            browserVersion = ua.match(/(?:Opera|OPR)\/([\d.]+)/)[1];
        }

        return {
            name: browserName,
            version: browserVersion,
            language: navigator.language,
            languages: navigator.languages,
            cookiesEnabled: navigator.cookieEnabled,
            javaEnabled: navigator.javaEnabled ? navigator.javaEnabled() : false
        };
    }

    async getPlatformInfo() {
        return {
            platform: navigator.platform || 'Unknown',
            oscpu: navigator.oscpu || 'Unknown',
            vendor: navigator.vendor || 'Unknown'
        };
    }

    async getCanvasFingerprint() {
        try {
            const canvas = document.createElement('canvas');
            canvas.width = 200;
            canvas.height = 50;
            const ctx = canvas.getContext('2d');

            ctx.textBaseline = 'top';
            ctx.font = '14px Arial';
            ctx.fillStyle = '#f60';
            ctx.fillRect(125, 1, 62, 20);
            ctx.fillStyle = '#069';
            ctx.fillText('Fingerprint', 2, 15);
            ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
            ctx.fillText('Canvas', 4, 17);

            ctx.beginPath();
            ctx.moveTo(20, 40);
            ctx.lineTo(180, 40);
            ctx.stroke();

            ctx.beginPath();
            ctx.arc(100, 25, 15, 0, Math.PI * 2);
            ctx.stroke();

            const dataUrl = canvas.toDataURL();
            const hash = await this.hashString(dataUrl);

            return {
                dataUrl: dataUrl,
                hash: hash
            };
        } catch (error) {
            return {
                dataUrl: '',
                hash: await this.hashString('canvas-fallback')
            };
        }
    }

    async getWebGLFingerprint() {
        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');

            if (!gl) {
                return {
                    vendor: 'Unknown',
                    renderer: 'Unknown',
                    hash: await this.hashString('webgl-unavailable')
                };
            }

            const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
            let vendor = 'Unknown';
            let renderer = 'Unknown';

            if (debugInfo) {
                vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
                renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
            }

            const parameters = {
                vendor: vendor,
                renderer: renderer,
                antialias: gl.getParameter(gl.SAMPLE_BUFFERS) && gl.getParameter(gl.SAMPLES) ? true : false,
                alpha: gl.getParameter(gl.ALPHA_BITS),
                depth: gl.getParameter(gl.DEPTH_BITS),
                stencil: gl.getParameter(gl.STENCIL_BITS),
                maxTextureSize: gl.getParameter(gl.MAX_TEXTURE_SIZE),
                maxViewportDims: gl.getParameter(gl.MAX_VIEWPORT_DIMS)
            };

            const hash = await this.hashString(JSON.stringify(parameters));

            return {
                vendor: vendor,
                renderer: renderer,
                parameters: parameters,
                hash: hash
            };
        } catch (error) {
            return {
                vendor: 'Unknown',
                renderer: 'Unknown',
                hash: await this.hashString('webgl-error')
            };
        }
    }

    async getAudioFingerprint() {
        try {
            const audioContext = new (window.AudioContext || window.webkitAudioContext)();
            const oscillator = audioContext.createOscillator();
            const analyser = audioContext.createAnalyser();
            const gainNode = audioContext.createGain();
            const scriptProcessor = audioContext.createScriptProcessor(4096, 1, 1);

            oscillator.type = 'triangle';
            oscillator.frequency.setValueAtTime(10000, audioContext.currentTime);

            gainNode.gain.setValueAtTime(0, audioContext.currentTime);

            oscillator.connect(analyser);
            analyser.connect(scriptProcessor);
            scriptProcessor.connect(gainNode);
            gainNode.connect(audioContext.destination);

            oscillator.start(0);

            return new Promise((resolve) => {
                scriptProcessor.onaudioprocess = (event) => {
                    const output = event.inputBuffer.getChannelData(0);
                    let sum = 0;

                    for (let i = 0; i < output.length; i++) {
                        sum += Math.abs(output[i]);
                    }

                    const average = sum / output.length;

                    oscillator.stop();
                    audioContext.close();

                    const hash = this.hashString(average.toString());

                    resolve({
                        value: average,
                        hash: hash
                    });
                };
            });
        } catch (error) {
            return {
                value: 0,
                hash: await this.hashString('audio-error')
            };
        }
    }

    async getFonts() {
        const baseFonts = ['monospace', 'sans-serif', 'serif'];
        const testString = 'mmmmmmmmmmlli';
        const testSize = '72px';

        const testFonts = [
            'Arial', 'Arial Black', 'Comic Sans MS', 'Courier New', 'Georgia',
            'Impact', 'Times New Roman', 'Trebuchet MS', 'Verdana', 'MS Gothic',
            'MS PGothic', 'MS UI Gothic', 'Meiryo', 'Segoe UI', 'Tahoma',
            'Calibri', 'Cambria', 'Consolas', 'Lucida Console', 'Monaco'
        ];

        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');

        const getWidth = (fontFamily) => {
            ctx.font = `${testSize} ${fontFamily}`;
            return ctx.measureText(testString).width;
        };

        const baseWidths = baseFonts.map(font => getWidth(font));

        const detected = [];

        for (const font of testFonts) {
            let detectedCount = 0;

            for (let i = 0; i < baseFonts.length; i++) {
                const width = getWidth(`'${font}', ${baseFonts[i]}`);
                if (width !== baseWidths[i]) {
                    detectedCount++;
                }
            }

            if (detectedCount > 0) {
                detected.push(font);
            }
        }

        return {
            list: detected,
            count: detected.length
        };
    }

    async getPlugins() {
        const plugins = [];

        if (navigator.plugins) {
            for (let i = 0; i < navigator.plugins.length; i++) {
                const plugin = navigator.plugins[i];
                plugins.push({
                    name: plugin.name,
                    filename: plugin.filename,
                    description: plugin.description
                });
            }
        }

        const hasFlash = (() => {
            try {
                const flash = new ActiveXObject('ShockwaveFlash.ShockwaveFlash');
                return true;
            } catch (e) {
                return navigator.mimeTypes && navigator.mimeTypes['application/x-shockwave-flash'] !== undefined;
            }
        })();

        if (hasFlash) {
            plugins.push({
                name: 'Shockwave Flash',
                filename: 'flash.dll',
                description: 'Adobe Flash Player'
            });
        }

        return {
            list: plugins.map(p => p.name),
            details: plugins
        };
    }

    async getStorageInfo() {
        let localStorageAvailable = false;
        let sessionStorageAvailable = false;
        let cookiesEnabled = false;

        try {
            localStorage.setItem('test', 'test');
            localStorage.removeItem('test');
            localStorageAvailable = true;
        } catch (e) {
            localStorageAvailable = false;
        }

        try {
            sessionStorage.setItem('test', 'test');
            sessionStorage.removeItem('test');
            sessionStorageAvailable = true;
        } catch (e) {
            sessionStorageAvailable = false;
        }

        try {
            cookiesEnabled = navigator.cookieEnabled;
        } catch (e) {
            cookiesEnabled = false;
        }

        return {
            localStorage: localStorageAvailable,
            sessionStorage: sessionStorageAvailable,
            cookies: cookiesEnabled,
            indexedDB: !!window.indexedDB,
            webSQL: !!window.openDatabase
        };
    }

    async getDoNotTrack() {
        try {
            return navigator.doNotTrack === '1' ||
                   window.doNotTrack === '1' ||
                   navigator.msDoNotTrack === '1';
        } catch (error) {
            return false;
        }
    }

    async getTimezone() {
        try {
            const offset = new Date().getTimezoneOffset();
            const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
            return `${tz} (UTC${offset > 0 ? '-' : '+'}${Math.abs(offset / 60)})`;
        } catch (error) {
            return 'Unknown';
        }
    }

    async getLanguage() {
        return {
            language: navigator.language || navigator.userLanguage,
            languages: Array.from(navigator.languages || [navigator.language || navigator.userLanguage])
        };
    }

    async getHardwareInfo() {
        const info = {
            concurrency: 0,
            memory: 0,
            devicePixelRatio: window.devicePixelRatio || 1
        };

        if (navigator.hardwareConcurrency) {
            info.concurrency = navigator.hardwareConcurrency;
        }

        if (navigator.deviceMemory) {
            info.memory = navigator.deviceMemory;
        }

        if (navigator.deviceMemory === undefined) {
            try {
                if (performance.memory) {
                    info.memory = Math.round(performance.memory.jsHeapSizeLimit / (1024 * 1024 * 1024));
                }
            } catch (e) {
                info.memory = 0;
            }
        }

        return info;
    }

    async hashString(str) {
        const encoder = new TextEncoder();
        const data = encoder.encode(str);
        const hashBuffer = await crypto.subtle.digest('SHA-256', data);
        const hashArray = Array.from(new Uint8Array(hashBuffer));
        const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
        return hashHex;
    }

    async generateCombinedHash(fingerprintData) {
        const hashParts = [
            fingerprintData.user_agent,
            `${fingerprintData.screen_width}x${fingerprintData.screen_height}`,
            fingerprintData.canvas_fingerprint,
            fingerprintData.webgl_vendor + fingerprintData.webgl_renderer,
            fingerprintData.audio_fingerprint,
            fingerprintData.platform
        ];

        const combinedString = hashParts.join('|');
        return await this.hashString(combinedString);
    }

    clearCache() {
        this.cache = null;
        this.components = {};
    }

    getComponents() {
        return this.components;
    }
}

class FingerprintAPI {
    constructor() {
        this.collector = new FingerprintCollector();
        this.apiBase = '/api/v1/fingerprint';
    }

    async collectAndSend() {
        try {
            const fingerprint = await this.collector.collect();
            const combinedHash = await this.collector.generateCombinedHash(fingerprint);

            const response = await fetch(`${this.apiBase}/collect`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${this.getToken()}`
                },
                body: JSON.stringify({
                    fingerprint: fingerprint,
                    hash: combinedHash
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();

            return {
                success: true,
                fingerprintId: result.data.fingerprint_id,
                hash: result.data.hash,
                riskLevel: result.data.risk_level,
                collectionTime: fingerprint.collection_time_ms
            };
        } catch (error) {
            console.error('Fingerprint collection failed:', error);
            return {
                success: false,
                error: error.message
            };
        }
    }

    async verifyFingerprint(fingerprintId, hash) {
        try {
            const response = await fetch(`${this.apiBase}/verify`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${this.getToken()}`
                },
                body: JSON.stringify({
                    fingerprint_id: fingerprintId,
                    hash: hash
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();

            return {
                valid: result.data.valid,
                riskLevel: result.data.risk_level,
                riskScore: result.data.risk_score,
                riskFactors: result.data.risk_factors,
                isNewDevice: result.data.is_new_device,
                similarDevices: result.data.similar_devices
            };
        } catch (error) {
            console.error('Fingerprint verification failed:', error);
            return {
                valid: false,
                error: error.message
            };
        }
    }

    async getDevices() {
        try {
            const response = await fetch(`${this.apiBase}/devices`, {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${this.getToken()}`
                }
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            return result.data.devices;
        } catch (error) {
            console.error('Failed to get devices:', error);
            return [];
        }
    }

    async trustDevice(deviceId) {
        try {
            const response = await fetch(`${this.apiBase}/devices/${deviceId}/trust`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${this.getToken()}`
                }
            });

            return response.ok;
        } catch (error) {
            console.error('Failed to trust device:', error);
            return false;
        }
    }

    async untrustDevice(deviceId) {
        try {
            const response = await fetch(`${this.apiBase}/devices/${deviceId}/untrust`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${this.getToken()}`
                }
            });

            return response.ok;
        } catch (error) {
            console.error('Failed to untrust device:', error);
            return false;
        }
    }

    async getDeviceHistory(deviceId, limit = 50) {
        try {
            const response = await fetch(`${this.apiBase}/devices/${deviceId}/history?limit=${limit}`, {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${this.getToken()}`
                }
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            return result.data.history;
        } catch (error) {
            console.error('Failed to get device history:', error);
            return [];
        }
    }

    async getAnomalies() {
        try {
            const response = await fetch(`${this.apiBase}/anomalies`, {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${this.getToken()}`
                }
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            return result.data.anomalies;
        } catch (error) {
            console.error('Failed to get anomalies:', error);
            return [];
        }
    }

    async exportData() {
        try {
            const response = await fetch(`${this.apiBase}/export`, {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${this.getToken()}`
                }
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            return result.data.data;
        } catch (error) {
            console.error('Failed to export data:', error);
            return null;
        }
    }

    async deleteData() {
        try {
            const response = await fetch(`${this.apiBase}/data`, {
                method: 'DELETE',
                headers: {
                    'Authorization': `Bearer ${this.getToken()}`
                }
            });

            return response.ok;
        } catch (error) {
            console.error('Failed to delete data:', error);
            return false;
        }
    }

    getToken() {
        return localStorage.getItem('token') || sessionStorage.getItem('token') || '';
    }
}

window.FingerprintCollector = FingerprintCollector;
window.FingerprintAPI = FingerprintAPI;
