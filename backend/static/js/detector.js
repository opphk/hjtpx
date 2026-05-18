class AdvancedFingerprintDetector {
    constructor() {
        this.canvasHash = '';
        this.webglHash = '';
        this.audioHash = '';
        this.fontHash = '';
        this.cpuidData = {};
        this.vmFeatures = {};
        this.containerFeatures = {};
        this.performanceMetrics = {};
        this.init();
    }

    async init() {
        await this.generateCanvasFingerprint();
        await this.generateWebGLFingerprint();
        await this.generateAudioFingerprint();
        await this.generateFontFingerprint();
        await this.detectCPUID();
        await this.collectPerformanceMetrics();
        this.detectVMFeatures();
        this.detectContainerFeatures();
    }

    async generateCanvasFingerprint() {
        const canvas = document.createElement('canvas');
        canvas.width = 400;
        canvas.height = 200;
        const ctx = canvas.getContext('2d');
        if (!ctx) return '';

        ctx.textBaseline = 'alphabetic';
        ctx.fillStyle = '#f60';
        ctx.fillRect(125, 1, 62, 20);
        ctx.fillStyle = '#069';
        ctx.font = '14px Arial';
        ctx.fillText('Cwm fjordbank glyphs vext quiz, 😀 😎 😮‍💨 👨‍🎤 🤌', 2, 22);
        ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
        ctx.font = '20px Arial';
        ctx.fillText('Cwm fjordbank glyphs vext quiz, 😀 😎 😮‍💨 👨‍🎤 🤌', 4, 55);

        ctx.globalCompositeOperation = 'multiply';
        ctx.fillStyle = 'rgb(255,0,255)';
        ctx.beginPath();
        ctx.arc(80, 80, 60, 0, Math.PI * 2, true);
        ctx.closePath();
        ctx.fill();
        ctx.fillStyle = 'rgb(0,255,255)';
        ctx.beginPath();
        ctx.arc(120, 80, 60, 0, Math.PI * 2 / 3, true);
        ctx.closePath();
        ctx.fill();
        ctx.fillStyle = 'rgb(255,255,0)';
        ctx.beginPath();
        ctx.arc(100, 80, 60, 0, Math.PI * 2 / 3, false);
        ctx.closePath();
        ctx.fill();
        ctx.fillStyle = 'rgb(255,127,0)';
        ctx.beginPath();
        ctx.arc(90, 90, 60, 0, Math.PI * 2 / 3, true);
        ctx.closePath();
        ctx.fill();

        ctx.fillStyle = '#fff';
        ctx.font = 'bold 20px Arial';
        ctx.fillText('abcdefghijklmnopqrstuvwxyz', 4, 95);
        ctx.font = 'italic 18px Georgia';
        ctx.fillText('ABCDEFGHIJKLMNOPQRSTUVWXYZ', 4, 120);
        ctx.font = '14px "Times New Roman"';
        ctx.fillText('0123456789 !@#$%^&*()', 4, 145);

        ctx.globalCompositeOperation = 'source-over';
        ctx.fillStyle = 'rgba(0,0,0,0.5)';
        ctx.fillRect(0, 150, 400, 50);
        ctx.fillStyle = '#fff';
        ctx.font = '16px Consolas, monospace';
        ctx.fillText('Advanced Canvas Fingerprint Test 12345', 10, 175);

        const dataURL = canvas.toDataURL();
        const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
        const pixelHash = this.hashPixelData(imageData);
        const entropy = this.calculateEntropy(imageData);
        const noiseLevel = this.calculateNoiseLevel(imageData);

        const hash = this.sha256(dataURL + pixelHash + entropy + noiseLevel);
        this.canvasHash = hash;
        this.canvasAnalysis = {
            hash,
            entropy,
            noiseLevel,
            pixelHash,
            width: canvas.width,
            height: canvas.height,
            dataURL: dataURL.substring(0, 100)
        };

        return hash;
    }

    hashPixelData(imageData) {
        const data = imageData.data;
        let hash = 0;
        const step = Math.max(1, Math.floor(data.length / 1000));
        for (let i = 0; i < data.length; i += step) {
            hash = ((hash << 5) - hash) + data[i];
            hash = hash & hash;
        }
        return Math.abs(hash).toString(16);
    }

    calculateEntropy(imageData) {
        const data = imageData.data;
        const histogram = new Array(256).fill(0);
        for (let i = 0; i < data.length; i++) {
            histogram[data[i]]++;
        }
        let entropy = 0;
        const total = data.length;
        for (let i = 0; i < 256; i++) {
            if (histogram[i] > 0) {
                const p = histogram[i] / total;
                entropy -= p * Math.log2(p);
            }
        }
        return entropy;
    }

    calculateNoiseLevel(imageData) {
        const data = imageData.data;
        const width = imageData.width;
        const height = imageData.height;
        let totalVariation = 0;
        let count = 0;

        for (let y = 1; y < height - 1; y++) {
            for (let x = 1; x < width - 1; x++) {
                const idx = (y * width + x) * 4;
                const dx = Math.abs(data[idx] - data[idx - 4]) +
                           Math.abs(data[idx + 1] - data[idx - 3]) +
                           Math.abs(data[idx + 2] - data[idx - 2]);
                const dy = Math.abs(data[idx] - data[idx - width * 4]) +
                           Math.abs(data[idx + 1] - data[idx - width * 4 + 1]) +
                           Math.abs(data[idx + 2] - data[idx - width * 4 + 2]);
                totalVariation += dx + dy;
                count++;
            }
        }
        return count > 0 ? totalVariation / count : 0;
    }

    async generateWebGLFingerprint() {
        const canvas = document.createElement('canvas');
        const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
        if (!gl) return '';

        const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
        const vendor = debugInfo ? gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL) : '';
        const renderer = debugInfo ? gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) : '';
        const maxTexSize = gl.getParameter(gl.MAX_TEXTURE_SIZE);
        const maxVertAttribs = gl.getParameter(gl.MAX_VERTEX_ATTRIBS);
        const maxTextureUnits = gl.getParameter(gl.MAX_TEXTURE_IMAGE_UNITS);
        const maxRenderbufferSize = gl.getParameter(gl.MAX_RENDERBUFFER_SIZE);
        const aliasedLineWidth = gl.getParameter(gl.ALIASED_LINE_WIDTH_RANGE);
        const aliasedPointSize = gl.getParameter(gl.ALIASED_POINT_SIZE_RANGE);
        const extensions = gl.getSupportedExtensions() || [];

        const rendererLower = renderer.toLowerCase();
        const softwareRenderer = /swiftshader|llvmpipe|mesa|software|emulated/i.test(renderer);
        const vmRenderer = /vmware|virtualbox|parallels|qemu|kvm|hyperv|xen/i.test(renderer);
        const headlessRenderer = /headless|headlesschrome|headless_chrome/i.test(renderer);

        const precision = gl.getShaderPrecisionFormat(gl.FRAGMENT_SHADER, gl.HIGH_FLOAT);
        const mediumPrecision = gl.getShaderPrecisionFormat(gl.FRAGMENT_SHADER, gl.MEDIUM_FLOAT);
        const lowPrecision = gl.getShaderPrecisionFormat(gl.FRAGMENT_SHADER, gl.LOW_FLOAT);

        const combinedData = `${vendor}~${renderer}~${maxTexSize}~${maxVertAttribs}~${extensions.length}~${precision?.precision}~${rendererLower}`;
        const hash = this.sha256(combinedData);

        this.webglHash = hash;
        this.webglAnalysis = {
            hash,
            vendor,
            renderer,
            maxTexSize,
            maxVertAttribs,
            maxTextureUnits,
            maxRenderbufferSize,
            extensionCount: extensions.length,
            precision: {
                high: precision?.precision || 0,
                medium: mediumPrecision?.precision || 0,
                low: lowPrecision?.precision || 0
            },
            aliasedLineWidth,
            aliasedPointSize,
            softwareRenderer,
            vmRenderer,
            headlessRenderer,
            extensions: extensions.slice(0, 20)
        };

        return hash;
    }

    async generateAudioFingerprint() {
        try {
            const AudioContext = window.OfflineAudioContext || window.webkitOfflineAudioContext;
            if (!AudioContext) return '';

            const ctx = new AudioContext(1, 44100, 44100);
            const osc = ctx.createOscillator();
            osc.type = 'triangle';
            osc.frequency.setValueAtTime(10000, ctx.currentTime);

            const compressor = ctx.createDynamicsCompressor();
            compressor.threshold.setValueAtTime(-50, ctx.currentTime);
            compressor.knee.setValueAtTime(40, ctx.currentTime);
            compressor.ratio.setValueAtTime(12, ctx.currentTime);
            compressor.attack.setValueAtTime(0, ctx.currentTime);
            compressor.release.setValueAtTime(0.25, ctx.currentTime);

            const gain = ctx.createGain();
            gain.gain.setValueAtTime(0.5, ctx.currentTime);

            const filter = ctx.createBiquadFilter();
            filter.type = 'lowpass';
            filter.frequency.setValueAtTime(5000, ctx.currentTime);

            const filter2 = ctx.createBiquadFilter();
            filter2.type = 'highpass';
            filter2.frequency.setValueAtTime(100, ctx.currentTime);

            const convolver = ctx.createConvolver();

            osc.connect(filter);
            filter.connect(filter2);
            filter2.connect(compressor);
            compressor.connect(gain);
            gain.connect(ctx.destination);

            osc.start(0);
            const buffer = await ctx.startRendering();

            const channelData = buffer.getChannelData(0);
            let sumAbs = 0;
            let sumSq = 0;
            let maxAbs = 0;
            let zeroCrossings = 0;
            const histogram = new Array(256).fill(0);

            for (let i = 0; i < channelData.length; i++) {
                const absVal = Math.abs(channelData[i]);
                sumAbs += absVal;
                sumSq += channelData[i] * channelData[i];
                if (absVal > maxAbs) maxAbs = absVal;
                if (i > 0 && ((channelData[i] >= 0 && channelData[i - 1] < 0) || (channelData[i] < 0 && channelData[i - 1] >= 0))) {
                    zeroCrossings++;
                }
                const bucket = Math.floor((channelData[i] + 1) * 128);
                if (bucket >= 0 && bucket < 256) histogram[bucket]++;
            }

            const avgAbs = sumAbs / channelData.length;
            const rms = Math.sqrt(sumSq / channelData.length);
            const crestFactor = maxAbs / rms;

            let entropy = 0;
            const total = channelData.length;
            for (let i = 0; i < 256; i++) {
                if (histogram[i] > 0) {
                    const p = histogram[i] / total;
                    entropy -= p * Math.log2(p);
                }
            }

            const hash = this.sha256(`${sumAbs}~${sumSq}~${maxAbs}~${zeroCrossings}~${entropy}~${channelData.length}`);

            this.audioHash = hash;
            this.audioAnalysis = {
                hash,
                avgAbs,
                rms,
                maxAbs,
                zeroCrossings,
                entropy,
                crestFactor,
                sampleRate: ctx.sampleRate,
                channelCount: buffer.numberOfChannels,
                duration: buffer.duration
            };

            return hash;
        } catch (e) {
            return '';
        }
    }

    async generateFontFingerprint() {
        const baseFonts = ['monospace', 'sans-serif', 'serif'];
        const testFonts = [
            'Arial', 'Arial Black', 'Comic Sans MS', 'Courier New', 'Georgia',
            'Impact', 'Times New Roman', 'Trebuchet MS', 'Verdana', 'Lucida Console',
            'Tahoma', 'Palatino', 'Garamond', 'Bookman', 'Cambria', 'Candara',
            'Century Gothic', 'Consolas', 'Corbel', 'Franklin Gothic', 'Futura',
            'Gill Sans', 'Helvetica', 'Lucida Sans', 'Monaco', 'Optima',
            'Segoe UI', 'Roboto', 'Open Sans', 'Lato', 'Montserrat',
            'Noto Sans', 'Source Sans Pro', 'Ubuntu', 'Fira Sans', 'Nunito',
            'JetBrains Mono', 'SF Mono', 'Menlo', 'Inconsolata', 'Source Code Pro',
            'Microsoft YaHei', 'SimHei', 'SimSun', 'KaiTi', 'Microsoft JhengHei',
            'PingFang SC', 'Hiragino Sans', 'Yu Gothic', 'Meiryo', 'Malgun Gothic',
            'Apple SD Gothic Neo', 'SF Pro Display', 'SF Pro Text'
        ];

        const el = document.createElement('div');
        el.style.cssText = 'position:absolute;left:-9999px;font-size:72px;visibility:hidden;white-space:nowrap;width:auto;height:auto;overflow:visible';
        el.textContent = 'mmmmmmmmmmlli';
        document.body.appendChild(el);

        const baseWidths = {};
        for (const base of baseFonts) {
            el.style.fontFamily = base;
            baseWidths[base] = {
                width: el.offsetWidth,
                height: el.offsetHeight
            };
        }

        const detectedFonts = [];
        const fontMetrics = {};

        for (const font of testFonts) {
            for (const base of baseFonts) {
                el.style.fontFamily = `"${font}", ${base}`;
                const width = el.offsetWidth;
                const height = el.offsetHeight;
                if (width !== baseWidths[base].width || height !== baseWidths[base].height) {
                    detectedFonts.push(font);
                    fontMetrics[font] = { width, height, diff: Math.abs(width - baseWidths[base].width) };
                    break;
                }
            }
        }

        const computedStyle = window.getComputedStyle(el);
        const hasSubpixelRendering = computedStyle.getPropertyValue('-webkit-font-smoothing') !== 'none' ||
                                   computedStyle.getPropertyValue('font-smooth') !== 'never';

        document.body.removeChild(el);

        const fontFamilyCount = new Set(detectedFonts.map(f => f.split(' ')[0])).size;
        const hash = this.sha256(`${detectedFonts.join(',')}~${fontFamilyCount}~${hasSubpixelRendering}`);

        this.fontHash = hash;
        this.fontAnalysis = {
            hash,
            detectedFonts,
            fontCount: detectedFonts.length,
            fontFamilyCount,
            fontMetrics,
            hasSubpixelRendering,
            baseWidths
        };

        return hash;
    }

    async detectCPUID() {
        const hwConcurrency = navigator.hardwareConcurrency || 0;
        const deviceMemory = navigator.deviceMemory || 0;

        const performanceTest = await this.runPerformanceBenchmark();

        const cpuFeatures = {
            cores: hwConcurrency,
            memory: deviceMemory,
            benchmarkScore: performanceTest.score,
            benchmarkTime: performanceTest.time,
            isVirtualCore: hwConcurrency <= 2 || hwConcurrency > 16,
            unusualMemory: deviceMemory < 1 || deviceMemory > 64,
            lowPerformance: performanceTest.score < 50
        };

        const uaLower = navigator.userAgent.toLowerCase();
        if (/hyperv|vmware|virtualbox|kvm|qemu|xen/i.test(uaLower)) {
            cpuFeatures.vmDetected = true;
        }

        this.cpuidData = cpuFeatures;
        return cpuFeatures;
    }

    async runPerformanceBenchmark() {
        const iterations = 100000;
        const startTime = performance.now();

        let result = 0;
        for (let i = 0; i < iterations; i++) {
            result += Math.sqrt(i) * Math.sin(i) * Math.cos(i);
        }

        const endTime = performance.now();
        const time = endTime - startTime;
        const score = Math.max(0, 100 - (time / 100));

        return { score, time };
    }

    async collectPerformanceMetrics() {
        const memory = performance.memory ? {
            jsHeapSizeLimit: performance.memory.jsHeapSizeLimit,
            totalJSHeapSize: performance.memory.totalJSHeapSize,
            usedJSHeapSize: performance.memory.usedJSHeapSize,
            heapUsageRatio: performance.memory.usedJSHeapSize / performance.memory.jsHeapSizeLimit
        } : null;

        const timing = performance.timing ? {
            connectEnd: performance.timing.connectEnd,
            domComplete: performance.timing.domComplete,
            domContentLoaded: performance.timing.domContentLoaded,
            domInteractive: performance.timing.domInteractive,
            loadEventEnd: performance.timing.loadEventEnd,
            navigationStart: performance.timing.navigationStart,
            responseEnd: performance.timing.responseEnd,
            ttfb: performance.timing.responseStart - performance.timing.requestStart
        } : null;

        const memoryTest = await this.testMemoryAccess();

        this.performanceMetrics = {
            memory,
            timing,
            memoryAccessTime: memoryTest.time,
            memoryPattern: memoryTest.pattern,
            isSlowDevice: memoryTest.time > 100
        };

        return this.performanceMetrics;
    }

    async testMemoryAccess() {
        const arraySize = 1000000;
        const array = new Array(arraySize);
        for (let i = 0; i < arraySize; i++) {
            array[i] = i;
        }

        const startTime = performance.now();
        let sum = 0;
        for (let i = 0; i < arraySize; i++) {
            sum += array[i];
        }
        const sequentialTime = performance.now() - startTime;

        const randomIndices = [];
        for (let i = 0; i < 1000; i++) {
            randomIndices.push(Math.floor(Math.random() * arraySize));
        }
        const randomStart = performance.now();
        for (const idx of randomIndices) {
            sum += array[idx];
        }
        const randomTime = performance.now() - randomStart;

        return {
            time: sequentialTime,
            randomTime,
            ratio: randomTime / sequentialTime,
            pattern: randomTime > sequentialTime * 2 ? 'cache_inconsistent' : 'normal'
        };
    }

    detectVMFeatures() {
        const ua = navigator.userAgent.toLowerCase();
        const platform = navigator.platform.toLowerCase();
        const webglRenderer = this.webglAnalysis?.renderer?.toLowerCase() || '';
        const webglVendor = this.webglAnalysis?.vendor?.toLowerCase() || '';

        const vmPatterns = {
            vmware: ['vmware', 'virtualbox', 'parallels', 'hyper-v', 'hyperv'],
            qemu: ['qemu', 'kvm', 'bochs'],
            xen: ['xen', 'hvm'],
            container: ['docker', 'lxc', 'containerd', 'kubernetes', 'k8s'],
            emulator: ['android sdk', 'genymotion', 'bluestacks', 'nox', 'memu', 'ldplayer']
        };

        const detectedVMs = [];
        let vmScore = 0;

        for (const [vmType, patterns] of Object.entries(vmPatterns)) {
            for (const pattern of patterns) {
                if (ua.includes(pattern) || webglRenderer.includes(pattern) || webglVendor.includes(pattern)) {
                    detectedVMs.push(vmType);
                    vmScore += 20;
                }
            }
        }

        if (this.cpuidData?.cores <= 2) vmScore += 15;
        if (this.cpuidData?.memory <= 1) vmScore += 15;
        if (this.webglAnalysis?.softwareRenderer) vmScore += 25;
        if (this.webglAnalysis?.vmRenderer) vmScore += 30;

        const screenMatch = this.detectEmulatorScreen();

        this.vmFeatures = {
            detected: detectedVMs.length > 0,
            types: detectedVMs,
            score: Math.min(vmScore, 100),
            softwareRenderer: this.webglAnalysis?.softwareRenderer || false,
            vmRenderer: this.webglAnalysis?.vmRenderer || false,
            lowCoreCount: this.cpuidData?.cores <= 2,
            lowMemory: this.cpuidData?.memory <= 1,
            screenAnomaly: screenMatch,
            highRisk: vmScore > 50
        };

        return this.vmFeatures;
    }

    detectEmulatorScreen() {
        const { width, height } = screen;
        const commonEmulatorResolutions = [
            { w: 320, h: 480, name: 'iPhone 3GS' },
            { w: 375, h: 667, name: 'iPhone 6/7/8' },
            { w: 414, h: 896, name: 'iPhone XR' },
            { w: 600, h: 1024, name: 'Generic Tablet' },
            { w: 768, h: 1024, name: 'iPad' }
        ];

        for (const res of commonEmulatorResolutions) {
            if (width === res.w && height === res.h) {
                return res.name;
            }
        }

        if (width === height && width > 1000) {
            return 'square_screen_anomaly';
        }

        if (width === screen.availWidth && height === screen.availHeight) {
            return 'fullscreen_emulator';
        }

        return null;
    }

    detectContainerFeatures() {
        const ua = navigator.userAgent.toLowerCase();

        const containerIndicators = {
            docker: ['docker', 'containerd', 'moby'],
            kubernetes: ['kubernetes', 'k8s', 'k3s'],
            lxc: ['lxc', 'linux container'],
            cgroups: []
        };

        const detectedContainers = [];
        let containerScore = 0;

        for (const [containerType, patterns] of Object.entries(containerIndicators)) {
            for (const pattern of patterns) {
                if (ua.includes(pattern)) {
                    detectedContainers.push(containerType);
                    containerScore += 25;
                }
            }
        }

        const storageCheck = this.checkStorageQuota();

        this.containerFeatures = {
            detected: detectedContainers.length > 0,
            types: detectedContainers,
            score: Math.min(containerScore + storageCheck.score, 100),
            zeroQuota: storageCheck.zeroQuota,
            lowQuota: storageCheck.lowQuota,
            highRisk: containerScore > 40
        };

        return this.containerFeatures;
    }

    async checkStorageQuota() {
        try {
            if (navigator.storage && navigator.storage.estimate) {
                const estimate = await navigator.storage.estimate();
                return {
                    quota: estimate.quota,
                    usage: estimate.usage,
                    zeroQuota: estimate.quota === 0,
                    lowQuota: estimate.quota < 100000000,
                    score: estimate.quota === 0 ? 30 : (estimate.quota < 100000000 ? 15 : 0)
                };
            }
        } catch (e) {}
        return { score: 0, zeroQuota: false, lowQuota: false };
    }

    sha256(str) {
        let hash = 0;
        for (let i = 0; i < str.length; i++) {
            const char = str.charCodeAt(i);
            hash = ((hash << 5) - hash) + char;
            hash = hash & hash;
        }
        const hex = Math.abs(hash).toString(16);
        return hex.padStart(16, '0');
    }

    getFullFingerprint() {
        return {
            canvasHash: this.canvasHash,
            webglHash: this.webglHash,
            audioHash: this.audioHash,
            fontHash: this.fontHash,
            canvasAnalysis: this.canvasAnalysis,
            webglAnalysis: this.webglAnalysis,
            audioAnalysis: this.audioAnalysis,
            fontAnalysis: this.fontAnalysis,
            cpuidData: this.cpuidData,
            vmFeatures: this.vmFeatures,
            containerFeatures: this.containerFeatures,
            performanceMetrics: this.performanceMetrics
        };
    }

    calculateRiskScore() {
        let score = 0;

        if (this.vmFeatures?.highRisk) score += 40;
        if (this.vmFeatures?.softwareRenderer) score += 25;
        if (this.containerFeatures?.highRisk) score += 30;

        if (this.canvasAnalysis?.noiseLevel < 0.1) score += 15;
        if (this.webglAnalysis?.headlessRenderer) score += 35;

        if (this.cpuidData?.lowPerformance) score += 20;
        if (this.performanceMetrics?.isSlowDevice) score += 15;

        return Math.min(score, 100);
    }
}

class AdvancedProxyDetector {
    constructor() {
        this.proxyIndicators = [];
        this.vpnProviders = this.initVPNProviders();
        this.torPatterns = this.initTorPatterns();
    }

    initVPNProviders() {
        return {
            nordvpn: { ranges: ['45.33.', '45.45.', '45.67.', '45.89.'], asn: [201229, 212502] },
            expressvpn: { ranges: ['23.', '104.', '132.'], asn: [201229] },
            surfshark: { ranges: ['172.104.', '185.220.', '188.172.'], asn: [212502] },
            cyberghost: { ranges: ['37.', '82.', '85.', '89.'], asn: [207083] },
            protonvpn: { ranges: ['185.195.', '185.220.'], asn: [] },
            mullvad: { ranges: ['185.195.', '194.132.'], asn: [] },
            private_internet_access: { ranges: ['104.238.', '107.170.', '172.104.'], asn: [201229] },
            windscribe: { ranges: ['35.182.', '45.33.'], asn: [201229] },
            mullvad_vpn: { ranges: ['185.195.', '194.132.'], asn: [] },
            nordsec: { ranges: ['45.33.', '45.45.'], asn: [201229] },
            surfshark_vpn: { ranges: ['172.104.'], asn: [212502] },
            perfect_privacy: { ranges: ['185.220.'], asn: [] },
            airvpn: { ranges: ['5.', '185.220.'], asn: [] },
            crypto_pa: { ranges: ['5.2.', '185.220.'], asn: [] },
            blackshark: { ranges: ['185.195.'], asn: [] }
        };
    }

    initTorPatterns() {
        return {
            relays: ['tor', 'onion', 'torproject', 'exitnode', 'tordnsel'],
            sdpPatterns: ['tcptype', 'tls', 'inject_host_overwrite', 'fingerprint'],
            indicators: ['relay', 'srflx', 'prflx', 'turn']
        };
    }

    async detectProxy() {
        const results = {
            isProxy: false,
            isVPN: false,
            isTor: false,
            isDatacenter: false,
            score: 0,
            confidence: 0,
            indicators: []
        };

        await this.checkWebRTCLeaks(results);
        this.checkConnectionAPI(results);
        this.checkHeaders(results);
        this.checkNetworkLatency(results);

        return results;
    }

    async checkWebRTCLeaks(results) {
        try {
            const RTCPeerConnection = window.RTCPeerConnection ||
                                     window.webkitRTCPeerConnection ||
                                     window.mozRTCPeerConnection;
            if (!RTCPeerConnection) {
                results.indicators.push('webrtc_not_available');
                return;
            }

            const ips = new Set();
            const pc = new RTCPeerConnection({
                iceServers: [
                    { urls: 'stun:stun.l.google.com:19302' },
                    { urls: 'stun:stun1.l.google.com:19302' }
                ]
            });

            pc.createDataChannel('');
            const offer = await pc.createOffer();
            await pc.setLocalDescription(offer);

            const sdp = pc.localDescription.sdp;
            const lines = sdp.split('\n');

            for (const line of lines) {
                if (line.includes('candidate')) {
                    const parts = line.split(' ');
                    if (parts[4] && parts[4] !== '0.0.0.0') {
                        const ip = parts[4];
                        ips.add(ip);

                        if (parts[7] && parts[7] !== 'host') {
                            results.indicators.push(`relay_candidate:${ip}`);
                            results.score += 10;
                        }

                        if (/^(5|23|45|82|85|89|104|128|131|154|171|176|185|192|199|204|209)\./.test(ip)) {
                            results.indicators.push(`tor_vpn_range:${ip}`);
                        }

                        for (const [provider, data] of Object.entries(this.vpnProviders)) {
                            for (const range of data.ranges) {
                                if (ip.startsWith(range)) {
                                    results.isVPN = true;
                                    results.score += 35;
                                    results.indicators.push(`vpn_provider:${provider}`);
                                }
                            }
                        }
                    }
                }
            }

            pc.close();

            if (ips.size > 1) {
                const ipsArr = Array.from(ips);
                const privateIPs = ipsArr.filter(ip =>
                    ip.startsWith('10.') ||
                    ip.startsWith('172.16.') || ip.startsWith('172.31.') ||
                    ip.startsWith('192.168.')
                );
                const publicIPs = ipsArr.filter(ip => !privateIPs.includes(ip));

                if (publicIPs.length > 0 && privateIPs.length > 0) {
                    results.score += 20;
                    results.indicators.push('vpn_ip_mismatch');
                }
            }

        } catch (e) {
            results.indicators.push(`webrtc_error:${e.message}`);
        }
    }

    checkConnectionAPI(results) {
        const conn = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
        if (!conn) {
            results.indicators.push('connection_api_unavailable');
            return;
        }

        if (conn.type === 'vpn' || conn.type === 'pptp' || conn.type === 'tunnel') {
            results.isVPN = true;
            results.score += 30;
            results.indicators.push('vpn_connection_type');
        }

        if (conn.type === 'proxy' || conn.type === 'socks') {
            results.isProxy = true;
            results.score += 35;
            results.indicators.push('proxy_connection_type');
        }

        if (conn.effectiveType === 'slow-2g' || conn.effectiveType === '2g') {
            results.score += 10;
            results.indicators.push('slow_connection');
        }

        if (conn.rtt && conn.rtt > 300) {
            results.score += 15;
            results.indicators.push('high_round_trip');
        }
    }

    checkHeaders(results) {
        const xff = document.cookie.includes('X-Forwarded-For') ? this.getCookieValue('X-Forwarded-For') : '';
        const via = document.cookie.includes('Via') ? this.getCookieValue('Via') : '';

        if (xff) {
            const ips = xff.split(',');
            if (ips.length > 1) {
                results.isProxy = true;
                results.score += 25;
                results.indicators.push('multi_hop_proxy');
            }
        }

        if (via) {
            results.isProxy = true;
            results.score += 20;
            results.indicators.push('via_header_present');
        }
    }

    async checkNetworkLatency(results) {
        try {
            const startTime = performance.now();
            await fetch('/api/v1/health', { method: 'HEAD', cache: 'no-cache' }).catch(() => null);
            const latency = performance.now() - startTime;

            if (latency > 3000) {
                results.score += 25;
                results.indicators.push(`high_latency:${Math.round(latency)}`);
            } else if (latency > 1000) {
                results.score += 10;
                results.indicators.push(`moderate_latency:${Math.round(latency)}`);
            }
        } catch (e) {
            results.indicators.push('latency_check_failed');
        }
    }

    getCookieValue(name) {
        const match = document.cookie.match(new RegExp('(^| )' + name + '=([^;]+)'));
        return match ? match[2] : '';
    }

    async detectTor() {
        const results = {
            isTor: false,
            score: 0,
            confidence: 0,
            indicators: []
        };

        const ua = navigator.userAgent.toLowerCase();
        if (/tor|onion/i.test(ua)) {
            results.isTor = true;
            results.score += 40;
            results.confidence += 0.5;
            results.indicators.push('tor_user_agent');
        }

        try {
            const RTCPeerConnection = window.RTCPeerConnection ||
                                     window.webkitRTCPeerConnection;
            if (RTCPeerConnection) {
                const pc = new RTCPeerConnection({
                    iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
                });
                pc.createDataChannel('');
                const offer = await pc.createOffer();
                await pc.setLocalDescription(offer);
                const sdp = pc.localDescription.sdp.toLowerCase();

                for (const pattern of this.torPatterns.sdpPatterns) {
                    if (sdp.includes(pattern)) {
                        results.isTor = true;
                        results.score += 30;
                        results.confidence += 0.4;
                        results.indicators.push(`tor_sdp:${pattern}`);
                    }
                }

                for (const indicator of this.torPatterns.indicators) {
                    if (sdp.includes(indicator)) {
                        results.score += 15;
                        results.indicators.push(`tor_indicator:${indicator}`);
                    }
                }

                pc.close();
            }
        } catch (e) {
            results.indicators.push(`tor_check_error:${e.message}`);
        }

        return results;
    }

    async analyzeIPHistory(ip) {
        const historyScore = 0;
        const patterns = [];

        return {
            ip,
            historyScore,
            patterns,
            riskLevel: historyScore > 50 ? 'high' : (historyScore > 25 ? 'medium' : 'low')
        };
    }
}

class EnhancedEnvironmentDetector {
    constructor(options = {}) {
        this.options = Object.assign({
            apiBase: '/api/v1',
            sampleRate: 0.3,
            chainCount: 20,
            enableAll: true
        }, options);

        this.fingerprintDetector = new AdvancedFingerprintDetector();
        this.proxyDetector = new AdvancedProxyDetector();
        this.results = {};
        this.riskScore = 0;
    }

    async runFullDetection() {
        await this.fingerprintDetector.init();

        const proxyResult = await this.proxyDetector.detectProxy();
        const torResult = await this.proxyDetector.detectTor();

        const vmRisk = this.calculateVMRisk();
        const containerRisk = this.calculateContainerRisk();
        const performanceRisk = this.calculatePerformanceRisk();

        this.riskScore = Math.min(
            vmRisk * 0.35 +
            containerRisk * 0.25 +
            proxyResult.score * 0.25 +
            torResult.score * 0.15,
            100
        );

        this.results = {
            fingerprint: this.fingerprintDetector.getFullFingerprint(),
            proxy: proxyResult,
            tor: torResult,
            vmRisk,
            containerRisk,
            performanceRisk,
            overallRisk: this.riskScore,
            timestamp: Date.now()
        };

        return this.results;
    }

    calculateVMRisk() {
        const vm = this.fingerprintDetector.vmFeatures;
        if (!vm) return 0;

        let risk = 0;
        if (vm.highRisk) risk += 40;
        if (vm.softwareRenderer) risk += 25;
        if (vm.vmRenderer) risk += 30;
        if (vm.lowCoreCount) risk += 15;
        if (vm.lowMemory) risk += 15;

        return Math.min(risk, 100);
    }

    calculateContainerRisk() {
        const container = this.fingerprintDetector.containerFeatures;
        if (!container) return 0;

        let risk = 0;
        if (container.highRisk) risk += 40;
        if (container.detected) risk += 30;
        if (container.zeroQuota) risk += 25;
        if (container.lowQuota) risk += 15;

        return Math.min(risk, 100);
    }

    calculatePerformanceRisk() {
        const perf = this.fingerprintDetector.performanceMetrics;
        if (!perf) return 0;

        let risk = 0;
        if (perf.isSlowDevice) risk += 25;
        if (perf.memoryAccessTime > 100) risk += 20;
        if (perf.memoryPattern === 'cache_inconsistent') risk += 15;

        return Math.min(risk, 100);
    }

    getRiskLevel() {
        if (this.riskScore >= 80) return 'critical';
        if (this.riskScore >= 60) return 'high';
        if (this.riskScore >= 40) return 'medium';
        if (this.riskScore >= 20) return 'low';
        return 'minimal';
    }

    getRecommendations() {
        const recommendations = [];
        const level = this.getRiskLevel();

        if (level === 'critical' || level === 'high') {
            recommendations.push('block_access');
            recommendations.push('require_additional_verification');
        } else if (level === 'medium') {
            recommendations.push('require_captcha');
            recommendations.push('enhanced_monitoring');
        } else if (level === 'low') {
            recommendations.push('standard_verification');
        } else {
            recommendations.push('allow_with_logging');
        }

        return recommendations;
    }

    toJSON() {
        return {
            results: this.results,
            riskScore: this.riskScore,
            riskLevel: this.getRiskLevel(),
            recommendations: this.getRecommendations()
        };
    }
}

window.AdvancedFingerprintDetector = AdvancedFingerprintDetector;
window.AdvancedProxyDetector = AdvancedProxyDetector;
window.EnhancedEnvironmentDetector = EnhancedEnvironmentDetector;
