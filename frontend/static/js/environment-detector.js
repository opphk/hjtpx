class EnvironmentDetector {
    constructor(options = {}) {
        this.options = Object.assign({
            apiBase: '/api/v1',
            sampleRate: 0.3,
            chainCount: 12,
            enableAll: true,
            sessionId: null
        }, options);
        this.results = {};
        this.riskScore = 0;
        this.detectionChain = [];
        this.detectionId = 'det_' + Date.now() + '_' + Math.random().toString(36).substr(2, 6);
        this.weights = {
            canvas: 8,
            webgl: 10,
            webgl2: 8,
            audio: 9,
            fonts: 7,
            webrtc_ip: 10,
            webdriver: 15,
            selenium: 18,
            puppeteer: 18,
            playwright: 18,
            chrome_runtime: 10,
            headless: 12,
            permissions: 6,
            plugins: 5,
            languages: 4,
            timezone: 5,
            screen: 3,
            hardware: 4,
            memory: 3,
            storage: 5,
            navigator: 4,
            window_props: 4,
            iframe: 6,
            notification: 3,
            battery: 3,
            media_devices: 4,
            connection: 5,
            adblock: 4,
            math: 3,
            gpu: 6,
            speech: 3,
            proxy_vpn: 20,
            tor_network: 25,
            datacenter_ip: 18,
            timezone_mismatch: 15,
            webrtc_leak: 20
        };
        this.vpnASNPatterns = [
            'AS45090', 'AS42366', 'AS9009', 'AS50611', 'AS48275',
            'AS400052', 'AS400053', 'AS400054', 'AS212883', 'AS212884', 'AS212885',
            'AS400065', 'AS400066', 'AS400067', 'AS62951',
            'AS393398', 'AS393399', 'AS36554', 'AS17451',
            'AS42385', 'AS42386', 'AS42387', 'AS42388',
            'AS157413', 'AS124309', 'AS207243',
            'AS16663', 'AS46844', 'AS202990',
            'AS63040', 'AS63041',
            'AS11426', 'AS11427',
            'AS51659', 'AS62263',
            'AS42073', 'AS212117',
            'AS393125', 'AS393126', 'AS201641',
            'AS51852', 'AS60113',
            'AS397980', 'AS397981',
            'AS209085', 'AS209086',
            'AS45078', 'AS58753',
            'AS51823', 'AS51824',
            'AS51430', 'AS51431',
            'AS61317', 'AS61318',
            'AS49673', 'AS49674',
            'AS47869', 'AS47870'
        ];
        this.datacenterIPRanges = {
            AWS: ['3.', '18.', '23.', '34.', '35.', '44.', '47.', '52.', '54.', '63.', '64.', '65.', '66.', '67.', '68.', '69.', '70.', '71.', '72.', '73.', '74.', '75.', '76.', '77.', '78.', '79.', '80.', '81.', '82.', '83.', '84.', '85.', '86.', '87.', '88.', '89.', '90.', '91.', '92.', '93.', '94.', '95.', '96.', '97.', '98.', '99.', '100.', '104.', '107.', '108.', '130.', '132.', '136.', '142.', '143.', '144.', '146.', '147.', '150.', '152.', '154.', '155.', '157.', '158.', '159.', '160.', '162.', '172.', '174.', '175.', '176.', '177.', '178.', '179.', '180.', '181.', '182.', '183.', '184.', '185.', '186.', '187.', '188.', '189.', '190.', '191.', '192.', '193.', '194.', '195.', '196.', '197.', '198.', '199.', '200.', '201.', '202.', '203.', '204.', '205.', '206.', '207.', '208.', '209.', '210.', '211.', '212.', '213.', '214.', '215.', '216.', '217.', '218.', '219.', '220.', '221.', '222.', '223.'],
            Azure: ['13.64.', '13.65.', '13.66.', '13.67.', '13.68.', '13.69.', '13.70.', '13.71.', '13.72.', '13.73.', '13.74.', '13.75.', '20.', '23.96.', '40.', '51.', '104.208.', '137.116.', '138.91.', '139.217.', '143.161.', '157.56.', '168.'],
            GCP: ['8.', '23.', '34.', '35.192.', '35.196.', '35.200.', '35.208.', '35.224.', '35.240.', '64.15.', '64.233.', '66.22.', '66.102.', '66.249.', '70.32.', '72.14.', '104.154.', '104.196.', '107.167.', '107.178.', '108.59.', '109.107.', '130.211.', '142.', '146.148.', '162.216.', '162.222.', '173.194.', '173.255.', '185.148.', '185.196.', '185.234.', '188.', '192.158.', '199.', '199.192.', '199.223.', '199.232.', '204.', '206.', '207.', '208.', '209.', '210.', '211.', '212.', '213.', '214.', '215.', '216.', '217.', '218.', '219.', '220.', '221.', '222.', '223.'],
            DigitalOcean: ['5.', '10.', '45.', '64.', '67.', '69.', '104.', '107.', '108.', '138.', '143.', '159.', '165.', '167.', '170.', '172.', '185.', '192.', '198.', '199.', '203.', '204.', '205.', '206.', '207.', '208.', '209.', '210.', '211.', '212.', '213.', '214.', '215.', '216.', '217.', '218.', '219.', '220.', '221.', '222.', '223.'],
            Oracle: ['140.', '141.', '144.', '147.', '152.', '157.', '158.', '159.', '160.', '161.', '162.', '164.', '165.', '166.', '167.', '168.', '169.', '170.', '172.', '173.', '192.', '193.', '194.', '195.', '196.', '197.', '198.', '199.', '200.', '201.', '202.', '203.', '204.', '205.', '206.', '207.', '208.', '209.', '210.', '211.'],
            Cloudflare: ['104.16.', '104.17.', '104.18.', '104.19.', '104.20.', '104.21.', '104.22.', '104.23.', '104.24.', '104.25.', '104.26.', '104.27.', '108.162.', '162.158.', '172.64.', '173.245.', '185.45.', '188.114.', '190.93.', '197.234.', '198.41.'],
            Hetzner: ['5.', '13.', '21.', '78.', '81.', '82.', '83.', '84.', '85.', '86.', '87.', '88.', '89.', '90.', '91.', '92.', '93.', '94.', '95.', '96.', '97.', '98.', '99.', '103.', '104.', '106.', '108.', '109.', '116.', '117.', '118.', '119.', '120.', '121.', '122.', '123.', '124.', '125.', '126.', '127.'],
            Linode: ['8.', '12.', '45.', '50.', '64.', '65.', '66.', '67.', '68.', '69.', '70.', '71.', '72.', '73.', '74.', '75.', '76.', '77.', '78.', '79.', '80.', '81.', '82.', '83.', '84.', '85.', '86.', '87.', '88.', '89.', '90.', '91.', '92.', '93.', '94.', '95.', '96.', '97.', '98.', '99.', '104.', '107.', '108.', '109.', '139.', '143.', '144.', '148.', '151.', '158.', '162.', '163.', '164.', '165.', '166.', '167.', '168.', '169.', '170.', '171.', '172.', '173.', '174.', '175.', '176.', '177.', '178.', '179.', '180.', '181.', '182.', '183.', '184.', '185.', '186.', '187.', '188.', '189.', '190.', '191.', '192.', '193.', '194.', '195.', '196.', '197.', '198.', '199.', '200.', '201.', '202.', '203.', '204.', '205.', '206.', '207.', '208.', '209.', '210.', '211.', '212.', '213.', '214.', '215.', '216.', '217.', '218.', '219.', '220.', '221.', '222.', '223.'],
            Vultr: ['45.', '104.', '108.61.', '108.171.', '149.', '155.', '162.', '167.', '172.', '173.', '174.', '175.', '176.', '177.', '178.', '179.', '180.', '181.', '182.', '183.', '184.', '185.', '186.', '187.', '188.', '189.', '190.', '191.', '192.', '193.', '194.', '195.', '196.', '197.', '198.', '199.', '200.', '201.', '202.', '203.', '204.', '205.', '206.', '207.', '208.', '209.', '210.', '211.']
        };
        this.countryTimezones = {
            'CN': 'Asia/Shanghai',
            'JP': 'Asia/Tokyo',
            'KR': 'Asia/Seoul',
            'IN': 'Asia/Kolkata',
            'AU': 'Australia/Sydney',
            'GB': 'Europe/London',
            'DE': 'Europe/Berlin',
            'FR': 'Europe/Paris',
            'IT': 'Europe/Rome',
            'ES': 'Europe/Rome',
            'RU': 'Europe/Moscow',
            'US': 'America/New_York',
            'CA': 'America/New_York',
            'BR': 'America/Sao_Paulo',
            'SA': 'Asia/Dubai',
            'SG': 'Asia/Singapore',
            'HK': 'Asia/Hong_Kong',
            'TW': 'Asia/Taipei'
        };
    }

    getDetectionMethods() {
        return [
            'detectHeadless',
            'detectWebDriver',
            'detectPuppeteer',
            'detectPlaywright',
            'detectSelenium',
            'detectChromeRuntime',
            'detectPermissions',
            'detectPlugins',
            'detectLanguages',
            'detectTimezone',
            'detectScreen',
            'detectHardwareConcurrency',
            'detectDeviceMemory',
            'detectStorage',
            'detectCanvas',
            'detectWebGL',
            'detectWebGL2',
            'detectAudio',
            'detectFonts',
            'detectNavigatorProps',
            'detectWindowProps',
            'detectIframe',
            'detectNotification',
            'detectBattery',
            'detectMediaDevices',
            'detectWebRTCIP',
            'detectConnection',
            'detectAdBlock',
            'detectMathFingerprint',
            'detectGPUFingerprint',
            'detectSpeech',
            'detectVPNConnection',
            'detectTorNetwork',
            'detectProxyVPN',
            'detectDatacenterIP',
            'detectTimezoneMismatch',
            'detectWebRTCLeak',
            'detectEmulators',
            'detectCloudPhones',
            'detectVirtualMachines',
            'detectContainers'
        ];
    }

    generateDetectionChain(count) {
        const allMethods = this.getDetectionMethods();
        const shuffled = [...allMethods].sort(() => Math.random() - 0.5);
        const selected = shuffled.slice(0, Math.min(count, allMethods.length));
        const methodAliases = {};
        selected.forEach((method, i) => {
            methodAliases[method] = 'chk_' + i.toString(36) + '_' + Math.random().toString(36).substr(2, 4);
        });
        return { selected, methodAliases };
    }

    async runChain() {
        const { selected, methodAliases } = this.generateDetectionChain(
            this.options.chainCount
        );
        this.detectionChain = selected;
        const chainResults = {};
        const startTime = performance.now();

        for (const method of selected) {
            try {
                const alias = methodAliases[method];
                const result = await this[method]();
                chainResults[alias] = result;
                this.results[method] = result;
            } catch (e) {
                const alias = methodAliases[method];
                chainResults[alias] = { detected: false, score: 0, error: e.message };
            }
        }

        const duration = performance.now() - startTime;
        this.riskScore = this.calculateRiskScore();

        return {
            detection_id: this.detectionId,
            chain: chainResults,
            chain_order: Object.values(methodAliases),
            risk_score: this.riskScore,
            duration_ms: Math.round(duration),
            timestamp: Date.now()
        };
    }

    calculateRiskScore() {
        let weightedScore = 0;
        let totalWeight = 0;

        for (const key in this.results) {
            const result = this.results[key];
            if (result && typeof result.score === 'number') {
                const weight = this.weights[key] || 5;
                weightedScore += result.score * weight;
                totalWeight += weight;
            }
        }

        if (totalWeight === 0) return 0;

        let baseScore = weightedScore / totalWeight;

        const autoTools = ['detectWebDriver', 'detectPuppeteer', 'detectPlaywright', 'detectSelenium'];
        const autoDetected = autoTools.filter(m => {
            const r = this.results[m];
            return r && r.detected === true;
        }).length;

        if (autoDetected >= 2) {
            baseScore = Math.min(baseScore * 1.5 + 20, 100);
        } else if (autoDetected >= 1) {
            baseScore = Math.min(baseScore * 1.3 + 10, 100);
        }

        const proxyIndicators = ['detectWebRTCIP', 'detectConnection', 'detectVPNConnection', 'detectProxyVPN'];
        const proxyAnomalies = proxyIndicators.filter(m => {
            const r = this.results[m];
            return r && r.score > 30;
        }).length;

        if (proxyAnomalies >= 2) {
            baseScore = Math.min(baseScore * 1.3 + 15, 100);
        }

        return Math.round(Math.min(Math.max(baseScore, 0), 100));
    }

    async detectProxyVPN() {
        let score = 0;
        const detections = [];
        const ipInfo = await this.getClientIPInfo();

        if (ipInfo) {
            if (ipInfo.headers) {
                if (ipInfo.headers['X-Forwarded-For'] || ipInfo.headers['x-forwarded-for']) {
                    score += 35;
                    detections.push('x_forwarded_for_header');
                }
                if (ipInfo.headers['X-Real-IP'] || ipInfo.headers['x-real-ip']) {
                    score += 30;
                    detections.push('x_real_ip_header');
                }
                if (ipInfo.headers['Via'] || ipInfo.headers['via']) {
                    const viaLower = (ipInfo.headers['Via'] || ipInfo.headers['via'] || '').toLowerCase();
                    if (/proxy|vpn|squid|nginx|haproxy|varnish/i.test(viaLower)) {
                        score += 45;
                        detections.push('via_proxy_header');
                    }
                }
                if (ipInfo.headers['X-ProxyChain'] || ipInfo.headers['x-proxychain']) {
                    score += 50;
                    detections.push('proxy_chain_header');
                }
                if (ipInfo.headers['Forwarded'] || ipInfo.headers['forwarded']) {
                    score += 25;
                    detections.push('forwarded_header');
                }
            }

            if (ipInfo.asn) {
                for (const asnPattern of this.vpnASNPatterns) {
                    if (ipInfo.asn.includes(asnPattern)) {
                        score += 50;
                        detections.push('vpn_asn_match:' + asnPattern);
                        break;
                    }
                }
            }

            if (ipInfo.isp) {
                const ispLower = ipInfo.isp.toLowerCase();
                if (/vpn|proxy|tor|hosting|cloud|datacenter/i.test(ispLower)) {
                    score += 40;
                    detections.push('vpn_isp_keyword');
                }
            }
        }

        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectDatacenterIP() {
        let score = 0;
        const detections = [];
        const ipInfo = await this.getClientIPInfo();

        if (ipInfo && ipInfo.ip) {
            for (const [provider, prefixes] of Object.entries(this.datacenterIPRanges)) {
                for (const prefix of prefixes) {
                    if (ipInfo.ip.startsWith(prefix)) {
                        score += 55;
                        detections.push('datacenter_ip:' + provider);
                        break;
                    }
                }
            }

            if (ipInfo.hosting === true || ipInfo.hosting === 'true') {
                score += 40;
                detections.push('hosting_provider');
            }
        }

        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectTimezoneMismatch() {
        let score = 0;
        const detections = [];

        try {
            const clientTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
            const ipInfo = await this.getClientIPInfo();

            if (ipInfo && ipInfo.country) {
                const expectedTimezone = this.countryTimezones[ipInfo.country.toUpperCase()];
                if (expectedTimezone && clientTimezone !== expectedTimezone) {
                    const expectedOffsets = {
                        'Asia/Shanghai': [480],
                        'Asia/Tokyo': [540],
                        'Asia/Seoul': [540],
                        'Asia/Kolkata': [330],
                        'Asia/Dubai': [240],
                        'Asia/Singapore': [480],
                        'Asia/Hong_Kong': [480],
                        'Asia/Taipei': [480],
                        'Europe/London': [0],
                        'Europe/Paris': [60],
                        'Europe/Berlin': [60],
                        'Europe/Moscow': [180],
                        'Europe/Rome': [60],
                        'America/New_York': [-300, -240],
                        'America/Los_Angeles': [-480, -420],
                        'America/Chicago': [-360, -300],
                        'America/Sao_Paulo': [-180],
                        'Australia/Sydney': [600]
                    };

                    const clientOffset = new Date().getTimezoneOffset();
                    const expectedMismatches = expectedOffsets[expectedTimezone] || [];

                    if (!expectedMismatches.includes(clientOffset)) {
                        score += 45;
                        detections.push('timezone_mismatch:' + clientTimezone + '!=' + expectedTimezone);
                    }
                }
            }

            const ua = navigator.userAgent || '';
            if (/headless|phantom|puppeteer|playwright|selenium/i.test(ua)) {
                const year = new Date().getFullYear();
                if (year < 2000 || year > 2100) {
                    score += 25;
                    detections.push('unrealistic_year');
                }
            }
        } catch (e) {
            score += 15;
            detections.push('timezone_check_error');
        }

        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectWebRTCLeak() {
        let score = 0;
        const detections = [];

        try {
            const RTCPeerConnection = window.RTCPeerConnection ||
                window.webkitRTCPeerConnection ||
                window.mozRTCPeerConnection;

            if (!RTCPeerConnection) {
                score += 20;
                detections.push('webrtc_not_available');
                return { detected: true, score: Math.min(score, 100), detections };
            }

            const ips = new Set();
            const pc = new RTCPeerConnection({
                iceServers: [
                    { urls: 'stun:stun.l.google.com:19302' },
                    { urls: 'stun:stun1.l.google.com:19302' },
                    { urls: 'stun:stun2.l.google.com:19302' }
                ]
            });

            pc.createDataChannel('');
            const offer = await pc.createOffer();
            await pc.setLocalDescription(offer);

            const sdp = pc.localDescription.sdp;
            const lines = sdp.split('\n');

            for (const line of lines) {
                if (line.indexOf('candidate') > -1) {
                    const parts = line.split(' ');
                    if (parts[4] && parts[4] !== '0.0.0.0') {
                        ips.add(parts[4]);
                        if (parts[7] !== 'host') {
                            detections.push('relay_candidate:' + parts[4]);
                            score += 35;
                        }
                    }
                }
            }

            pc.close();

            const ipsArr = Array.from(ips);
            const publicIPs = ipsArr.filter(ip =>
                !ip.startsWith('10.') &&
                !ip.startsWith('172.16.') && !ip.startsWith('172.17.') && !ip.startsWith('172.18.') &&
                !ip.startsWith('172.19.') && !ip.startsWith('172.20.') && !ip.startsWith('172.21.') &&
                !ip.startsWith('172.22.') && !ip.startsWith('172.23.') && !ip.startsWith('172.24.') &&
                !ip.startsWith('172.25.') && !ip.startsWith('172.26.') && !ip.startsWith('172.27.') &&
                !ip.startsWith('172.28.') && !ip.startsWith('172.29.') && !ip.startsWith('172.30.') &&
                !ip.startsWith('172.31.') &&
                !ip.startsWith('192.168.')
            );

            if (publicIPs.length > 0) {
                score += 40;
                detections.push('public_ip_leak');
            }

            if (ipsArr.length > 3) {
                score += 25;
                detections.push('multiple_ips');
            }

            for (const provider in this.datacenterIPRanges) {
                for (const ip of publicIPs) {
                    for (const prefix of this.datacenterIPRanges[provider]) {
                        if (ip.startsWith(prefix.replace('/16', '.').replace('/24', '.'))) {
                            score += 45;
                            detections.push('datacenter_webrtc:' + provider);
                            break;
                        }
                    }
                }
            }

        } catch (e) {
            score += 15;
            detections.push('webrtc_leak_error');
        }

        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async getClientIPInfo() {
        try {
            const response = await fetch('/api/v1/ip-info', {
                method: 'GET',
                headers: { 'Content-Type': 'application/json' }
            });
            if (response.ok) {
                return await response.json();
            }
        } catch (e) {}

        try {
            const response = await fetch('https://api.ipapi.co/?format=json', {
                method: 'GET',
                mode: 'cors'
            });
            if (response.ok) {
                const data = await response.json();
                return {
                    ip: data.ip,
                    country: data.country_code,
                    isp: data.connection && data.connection.isp,
                    asn: data.connection && data.connection.asn,
                    hosting: data.hosting || data.colo
                };
            }
        } catch (e) {}

        return { ip: '', headers: {} };
    }

    async detectHeadless() {
        let score = 0;
        const detections = [];
        if (navigator.webdriver === true || navigator.webdriver === false) {
        } else {
            score += 30;
            detections.push('webdriver_undefined');
        }
        if (navigator.plugins && navigator.plugins.length === 0) {
            score += 15;
            detections.push('no_plugins');
        }
        if (navigator.languages && navigator.languages.length === 0) {
            score += 15;
            detections.push('no_languages');
        }
        if (window.chrome && window.chrome.runtime === undefined) {
            score += 20;
            detections.push('chrome_no_runtime');
        }
        const mimeTypes = navigator.mimeTypes;
        if (mimeTypes && mimeTypes.length === 0) {
            score += 20;
            detections.push('no_mimetypes');
        }
        try {
            const ua = navigator.userAgent || '';
            if (/headless|phantom/i.test(ua)) {
                score += 35;
                detections.push('headless_ua');
            }
        } catch (e) {}
        try {
            if (window.outerHeight === 0 && window.outerWidth === 0) {
                score += 25;
                detections.push('zero_window_size');
            }
        } catch (e) {}
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectWebDriver() {
        let score = 0;
        const detections = [];
        const wdProps = [
            'webdriver', '__webdriver_evaluate', '__selenium_evaluate',
            '__webdriver_script_fn', '__driver_evaluate', '__fxdriver_evaluate',
            '__webdriver_unwrapped', '__lastWatirAlert', '__$webdriverAsyncExecutor',
            'callSelenium', '__selenium', 'Selenium'
        ];
        for (const prop of wdProps) {
            if (window[prop] !== undefined) {
                score += 15;
                detections.push(prop);
            }
        }
        try {
            if (navigator.webdriver === true) {
                score += 30;
                detections.push('navigator.webdriver');
            }
        } catch (e) {}
        try {
            const el = document.createElement('div');
            el.setAttribute('onclick', 'return __webdriver_script_fn()');
            if (el.onclick !== null) {
                score += 10;
                detections.push('webdriver_script_fn');
            }
        } catch (e) {}
        try {
            const el = document.createElement('div');
            el.setAttribute('onmousemove', 'return __driver_evaluate()');
            if (el.onmousemove !== null) {
                score += 10;
                detections.push('driver_evaluate');
            }
        } catch (e) {}
        return { detected: score > 20, score: Math.min(score, 100), detections };
    }

    async detectPuppeteer() {
        let score = 0;
        const detections = [];
        try {
            if (navigator.webdriver === true) {
                score += 25;
                detections.push('webdriver_true');
            }
        } catch (e) {}
        try {
            if (document.$cdc_asdjflasutopfhvcZLmcfl_) {
                score += 35;
                detections.push('cdc_marker');
            }
        } catch (e) {}
        try {
            if (document.$chrome_asyncScriptInfo) {
                score += 25;
                detections.push('chrome_async_script');
            }
        } catch (e) {}
        try {
            const el = document.createElement('div');
            el.setAttribute('onpaste', 'return function(){throw new Error("puppeteer")}');
            if (el.onpaste !== null) {
                score += 20;
                detections.push('puppeteer_onpaste');
            }
        } catch (e) {}
        try {
            const userAgent = navigator.userAgent || '';
            if (/headless/i.test(userAgent)) {
                score += 30;
                detections.push('headless_ua');
            }
            if (/puppet/i.test(userAgent)) {
                score += 40;
                detections.push('puppeteer_ua');
            }
        } catch (e) {}
        try {
            if (window._puppeteer_globals !== undefined) {
                score += 30;
                detections.push('puppeteer_globals');
            }
        } catch (e) {}
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectPlaywright() {
        let score = 0;
        const detections = [];
        try {
            if (window.__playwright__ !== undefined ||
                window.__pw_tags !== undefined ||
                window.__pw_resume__ !== undefined) {
                score += 45;
                detections.push('playwright_global');
            }
        } catch (e) {}
        try {
            const el = document.createElement('div');
            el.setAttribute('onfocus', 'return __pw_resume__()');
            if (el.onfocus !== null) {
                score += 35;
                detections.push('playwright_onfocus');
            }
        } catch (e) {}
        try {
            const el = document.createElement('div');
            el.setAttribute('onmouseenter', 'return __pw_resume__()');
            if (el.onmouseenter !== null) {
                score += 25;
                detections.push('playwright_mouseenter');
            }
        } catch (e) {}
        try {
            const ua = navigator.userAgent || '';
            if (/playwright/i.test(ua)) {
                score += 50;
                detections.push('playwright_ua');
            }
        } catch (e) {}
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectSelenium() {
        let score = 0;
        const detections = [];
        const selProps = [
            'selenium', '_selenium', 'callSelenium', '__selenium',
            'document__selenium', 'Selenium', '__webdriver_script_fn',
            'Selenium.prototype'
        ];
        for (const prop of selProps) {
            if (window[prop] !== undefined || document[prop] !== undefined) {
                score += 20;
                detections.push(prop);
            }
        }
        try {
            if (document.documentElement.getAttribute('webdriver') !== null) {
                score += 25;
                detections.push('webdriver_attr');
            }
        } catch (e) {}
        try {
            const el = document.createElement('div');
            el.setAttribute('onmouseover', 'return Selenium.prototype.whatever');
            if (el.onmouseover !== null) {
                score += 15;
                detections.push('selenium_prototype');
            }
        } catch (e) {}
        try {
            const el = document.createElement('div');
            el.setAttribute('onkeydown', 'return selenium_executor.onkeydown');
            if (el.onkeydown !== null) {
                score += 15;
                detections.push('selenium_executor');
            }
        } catch (e) {}
        try {
            const ua = navigator.userAgent || '';
            if (/selenium/i.test(ua)) {
                score += 40;
                detections.push('selenium_ua');
            }
        } catch (e) {}
        try {
            if (window.__$webdriverAsyncExecutor !== undefined) {
                score += 20;
                detections.push('webdriver_async_executor');
            }
        } catch (e) {}
        return { detected: score > 20, score: Math.min(score, 100), detections };
    }

    async detectChromeRuntime() {
        let score = 0;
        const detections = [];
        try {
            if (window.chrome) {
                if (window.chrome.runtime === undefined) {
                    score += 20;
                    detections.push('chrome_runtime_missing');
                } else if (window.chrome.runtime && window.chrome.runtime.id === undefined) {
                    score += 10;
                    detections.push('chrome_runtime_no_id');
                }
                if (window.chrome.loadTimes === undefined) {
                    score += 10;
                    detections.push('chrome_loadtimes_missing');
                }
                if (window.chrome.csi === undefined) {
                    score += 10;
                    detections.push('chrome_csi_missing');
                }
                if (window.chrome.app === undefined) {
                    score += 10;
                    detections.push('chrome_app_missing');
                }
            } else {
                if (!/Edge|Edg|Firefox|Safari/i.test(navigator.userAgent || '')) {
                    score += 30;
                    detections.push('no_chrome_no_alt');
                }
            }
        } catch (e) {
            score += 25;
            detections.push('chrome_check_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectPermissions() {
        let score = 0;
        const detections = [];
        try {
            if (navigator.permissions && navigator.permissions.query) {
                const permNames = ['notifications', 'geolocation', 'camera', 'microphone', 'midi'];
                const permChecks = await Promise.all(
                    permNames.map(name =>
                        navigator.permissions.query({ name }).catch(() => ({ state: 'error' }))
                    )
                );
                const allDenied = permChecks.every(p => p.state === 'denied' || p.state === 'error');
                if (allDenied) {
                    score += 20;
                    detections.push('all_permissions_denied');
                }
                const deniedCount = permChecks.filter(p => p.state === 'denied').length;
                if (deniedCount >= 4) {
                    score += 10;
                    detections.push('most_permissions_denied');
                }
            } else {
                score += 20;
                detections.push('permissions_api_missing');
            }
        } catch (e) {
            score += 25;
            detections.push('permissions_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectPlugins() {
        let score = 0;
        const detections = [];
        try {
            const plugins = navigator.plugins;
            if (!plugins || plugins.length === 0) {
                score += 25;
                detections.push('no_plugins');
            } else {
                const commonPlugins = ['PDF Viewer', 'Chrome PDF Viewer', 'Chromium PDF Viewer',
                    'Microsoft Edge PDF Viewer', 'WebKit built-in PDF'];
                const hasPDF = Array.from(plugins).some(p =>
                    commonPlugins.some(cp => p.name.includes(cp))
                );
                if (!hasPDF) {
                    score += 10;
                    detections.push('no_pdf_plugin');
                }
                if (plugins.length < 3) {
                    score += 10;
                    detections.push('too_few_plugins');
                }
                if (plugins.length > 10) {
                    score += 5;
                    detections.push('too_many_plugins');
                }
            }
        } catch (e) {
            score += 30;
            detections.push('plugins_access_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectLanguages() {
        let score = 0;
        const detections = [];
        try {
            const langs = navigator.languages;
            if (!langs || langs.length === 0) {
                score += 25;
                detections.push('no_languages');
            }
            const lang = navigator.language;
            if (!lang) {
                score += 20;
                detections.push('no_language');
            }
            if (langs && langs.length > 0 && lang) {
                if (langs[0] !== lang) {
                    score += 15;
                    detections.push('languages_mismatch');
                }
            }
        } catch (e) {
            score += 30;
            detections.push('languages_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectTimezone() {
        let score = 0;
        const detections = [];
        try {
            const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
            if (!tz) {
                score += 30;
                detections.push('no_timezone');
            }
            const offset = new Date().getTimezoneOffset();
            if (offset === 0 && !tz) {
                score += 20;
                detections.push('utc_offset_no_tz');
            }
            const year = new Date().getFullYear();
            if (year < 2000 || year > 2100) {
                score += 25;
                detections.push('unrealistic_date');
            }
            try {
                const matchOffset = /GMT([+-]\d{2}):?(\d{2})/.exec(new Date().toString());
                if (matchOffset) {
                    const strOffset = parseInt(matchOffset[1]) * 60 + parseInt(matchOffset[2]) * (matchOffset[1] > 0 ? 1 : -1);
                    if (Math.abs(strOffset + offset) > 30) {
                        score += 20;
                        detections.push('timezone_offset_mismatch');
                    }
                }
            } catch (e) {}
        } catch (e) {
            score += 35;
            detections.push('timezone_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectScreen() {
        let score = 0;
        const detections = [];
        try {
            const { width, height, colorDepth, pixelDepth, availWidth, availHeight } = screen;
            if (!width || !height) {
                score += 30;
                detections.push('no_screen_size');
            }
            if (colorDepth === 0 || pixelDepth === 0) {
                score += 25;
                detections.push('zero_depth');
            }
            if (width <= 800 || height <= 600) {
                score += 10;
                detections.push('small_screen');
            }
            if ('isExtended' in screen && screen.isExtended === undefined) {
                score += 10;
                detections.push('screen_extended_missing');
            }
            if (availWidth === 0 || availHeight === 0) {
                score += 15;
                detections.push('zero_avail_size');
            }
        } catch (e) {
            score += 30;
            detections.push('screen_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectHardwareConcurrency() {
        let score = 0;
        const detections = [];
        try {
            const c = navigator.hardwareConcurrency;
            if (c === undefined || c === null) {
                score += 30;
                detections.push('no_concurrency');
            } else if (c <= 1) {
                score += 25;
                detections.push('single_core');
            } else if (c > 64) {
                score += 20;
                detections.push('unrealistic_cores');
            }
        } catch (e) {
            score += 30;
            detections.push('concurrency_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectDeviceMemory() {
        let score = 0;
        const detections = [];
        try {
            const mem = navigator.deviceMemory;
            if (mem === undefined || mem === null) {
                score += 20;
                detections.push('no_device_memory');
            } else if (mem <= 0.25) {
                score += 25;
                detections.push('low_memory');
            } else if (mem > 64) {
                score += 15;
                detections.push('unrealistic_memory');
            }
        } catch (e) {
            score += 20;
            detections.push('memory_error');
        }
        return { detected: score > 20, score: Math.min(score, 100), detections };
    }

    async detectStorage() {
        let score = 0;
        const detections = [];
        try {
            localStorage.setItem('_md_test', '1');
            localStorage.removeItem('_md_test');
        } catch (e) {
            score += 20;
            detections.push('localStorage_denied');
        }
        try {
            sessionStorage.setItem('_md_test', '1');
            sessionStorage.removeItem('_md_test');
        } catch (e) {
            score += 20;
            detections.push('sessionStorage_denied');
        }
        try {
            if (navigator.storage && navigator.storage.estimate) {
                const est = await navigator.storage.estimate();
                if (est.quota === 0) {
                    score += 15;
                    detections.push('zero_storage_quota');
                }
            } else {
                score += 10;
                detections.push('storage_api_missing');
            }
        } catch (e) {
            score += 15;
            detections.push('storage_estimate_error');
        }
        try {
            if (navigator.cookieEnabled === false) {
                score += 15;
                detections.push('cookies_disabled');
            }
        } catch (e) {}
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectCanvas() {
        let score = 0;
        const detections = [];
        try {
            const canvas = document.createElement('canvas');
            canvas.width = 280;
            canvas.height = 80;
            const ctx = canvas.getContext('2d');
            if (!ctx) {
                score += 40;
                detections.push('no_canvas_context');
                return { detected: true, score: Math.min(score, 100), detections };
            }

            ctx.textBaseline = 'alphabetic';
            ctx.fillStyle = '#f60';
            ctx.fillRect(125, 1, 62, 20);
            ctx.fillStyle = '#069';
            ctx.font = '11pt Arial';
            ctx.fillText('Cwm fjordbank glyphs vext quiz, 😀', 2, 15);
            ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
            ctx.font = '18pt Arial';
            ctx.fillText('Cwm fjordbank glyphs vext quiz, 😀', 4, 45);

            ctx.globalCompositeOperation = 'multiply';
            ctx.fillStyle = 'rgb(255,0,255)';
            ctx.beginPath();
            ctx.arc(50, 50, 50, 0, Math.PI * 2, true);
            ctx.closePath();
            ctx.fill();
            ctx.fillStyle = 'rgb(0,255,255)';
            ctx.beginPath();
            ctx.arc(100, 50, 50, 0, Math.PI * 2 / 3, true);
            ctx.closePath();
            ctx.fill();
            ctx.fillStyle = 'rgb(255,255,0)';
            ctx.beginPath();
            ctx.arc(75, 50, 50, 0, Math.PI * 2 / 3, false);
            ctx.closePath();
            ctx.fill();

            ctx.fillStyle = '#fff';
            ctx.font = 'bold 16pt Arial';
            ctx.fillText('abcdefghijklmnopqrstuvwxyz', 4, 70);

            const dataURL = canvas.toDataURL();
            const dataURL2 = canvas.toDataURL();
            if (dataURL !== dataURL2) {
                score += 25;
                detections.push('canvas_unstable');
            }

            const imageData = ctx.getImageData(0, 0, 10, 10);
            const pixelSum = Array.from(imageData.data.slice(0, 40)).reduce((a, b) => a + b, 0);
            if (pixelSum === 0) {
                score += 20;
                detections.push('canvas_empty_readback');
            }
        } catch (e) {
            score += 35;
            detections.push('canvas_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectWebGL() {
        let score = 0;
        const detections = [];
        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
            if (!gl) {
                score += 40;
                detections.push('no_webgl');
                return { detected: true, score: Math.min(score, 100), detections };
            }
            const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
            if (debugInfo) {
                const vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
                const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                if (!vendor || !renderer) {
                    score += 15;
                    detections.push('webgl_no_vendor');
                }
                if (/swiftshader|llvmpipe|mesa|virtual|google\s*inc/i.test(renderer || '')) {
                    score += 30;
                    detections.push('software_renderer');
                }
            } else {
                score += 20;
                detections.push('no_webgl_debug');
            }
            const maxTexSize = gl.getParameter(gl.MAX_TEXTURE_SIZE);
            if (maxTexSize <= 1024) {
                score += 15;
                detections.push('small_tex_size');
            }
            const maxVertAttribs = gl.getParameter(gl.MAX_VERTEX_ATTRIBS);
            if (maxVertAttribs <= 8) {
                score += 10;
                detections.push('few_vertex_attribs');
            }
            const aliasedRange = gl.getParameter(gl.ALIASED_LINE_WIDTH_RANGE);
            if (aliasedRange && aliasedRange[1] <= 1) {
                score += 10;
                detections.push('aliased_line_only');
            }
            const shaderPrecision = gl.getShaderPrecisionFormat(gl.FRAGMENT_SHADER, gl.HIGH_FLOAT);
            if (shaderPrecision && shaderPrecision.precision < 16) {
                score += 15;
                detections.push('low_shader_precision');
            }
            const ext = gl.getExtension('EXT_texture_filter_anisotropic');
            if (!ext) {
                score += 5;
                detections.push('no_anisotropic');
            }
            const supportedExts = gl.getSupportedExtensions();
            if (supportedExts && supportedExts.length < 10) {
                score += 10;
                detections.push('few_webgl_extensions');
            }
        } catch (e) {
            score += 35;
            detections.push('webgl_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectWebGL2() {
        let score = 0;
        const detections = [];
        try {
            const canvas = document.createElement('canvas');
            const gl2 = canvas.getContext('webgl2');
            if (!gl2) {
                return { detected: false, score: 0, detections: ['no_webgl2'] };
            }
            const debugInfo = gl2.getExtension('WEBGL_debug_renderer_info');
            if (debugInfo) {
                const renderer = gl2.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                if (/swiftshader|llvmpipe|mesa|virtual/i.test(renderer || '')) {
                    score += 25;
                    detections.push('webgl2_software_renderer');
                }
            }
            const maxTexSize = gl2.getParameter(gl2.MAX_TEXTURE_SIZE);
            if (maxTexSize <= 1024) {
                score += 10;
                detections.push('webgl2_small_tex');
            }
            const supportedExts = gl2.getSupportedExtensions();
            if (supportedExts && supportedExts.length < 5) {
                score += 10;
                detections.push('few_webgl2_extensions');
            }
        } catch (e) {
            score += 20;
            detections.push('webgl2_error');
        }
        return { detected: score > 20, score: Math.min(score, 100), detections };
    }

    async detectAudio() {
        let score = 0;
        const detections = [];
        try {
            const AudioContext = window.OfflineAudioContext || window.webkitOfflineAudioContext;
            if (!AudioContext) {
                score += 30;
                detections.push('no_audiocontext');
                return { detected: true, score: Math.min(score, 100), detections };
            }
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
            osc.connect(compressor);
            compressor.connect(ctx.destination);
            osc.start(0);

            const startTime = performance.now();
            const buffer = await ctx.startRendering();
            const renderTime = performance.now() - startTime;

            if (renderTime < 5) {
                score += 20;
                detections.push('audio_render_too_fast');
            }
            const channelData = buffer.getChannelData(0);
            let sumAbs = 0;
            let sumSq = 0;
            for (let i = 4500; i < 5000; i++) {
                sumAbs += Math.abs(channelData[i]);
            }
            for (let i = 0; i < channelData.length; i++) {
                sumSq += channelData[i] * channelData[i];
            }
            if (sumAbs === 0 && sumSq === 0) {
                score += 25;
                detections.push('audio_silent');
            }
        } catch (e) {
            score += 30;
            detections.push('audio_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectFonts() {
        let score = 0;
        const detections = [];
        const baseFonts = ['monospace', 'sans-serif', 'serif'];
        const testFonts = [
            'Arial', 'Helvetica', 'Times New Roman', 'Courier New',
            'Verdana', 'Georgia', 'Palatino', 'Garamond',
            'Impact', 'Comic Sans MS', 'Trebuchet MS', 'Lucida Console',
            'Tahoma', 'Segoe UI', 'Roboto', 'Open Sans',
            'Lato', 'Montserrat', 'Source Sans Pro', 'Raleway',
            'Ubuntu', 'Noto Sans', 'Droid Sans', 'Fira Sans',
            'Merriweather', 'Playfair Display', 'PT Sans', 'Nunito',
            'Quicksand', 'Work Sans', 'Oswald', 'Roboto Condensed',
            'Noto Serif', 'Lora', 'IBM Plex Sans', 'JetBrains Mono',
            'SF Pro Display', 'SF Pro Text', 'Calibri', 'Candara',
            'Corbel', 'Cambria', 'Bookman', 'Futura', 'Optima'
        ];
        try {
            const el = document.createElement('div');
            el.style.cssText = 'position:absolute;left:-9999px;font-size:72px;visibility:hidden;white-space:nowrap';
            el.textContent = 'mmmmmmmmmmlli';
            document.body.appendChild(el);
            const baseWidths = {};
            for (const base of baseFonts) {
                el.style.fontFamily = base;
                baseWidths[base] = el.offsetWidth;
            }
            let fontCount = 0;
            for (const font of testFonts) {
                for (const base of baseFonts) {
                    el.style.fontFamily = `"${font}", ${base}`;
                    if (el.offsetWidth !== baseWidths[base]) {
                        fontCount++;
                        break;
                    }
                }
            }
            document.body.removeChild(el);
            if (fontCount < 3) {
                score += 25;
                detections.push('too_few_fonts');
            }
            if (fontCount < 8) {
                score += 10;
                detections.push('limited_fonts');
            }
        } catch (e) {
            score += 25;
            detections.push('font_detection_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectWebRTCIP() {
        let score = 0;
        const detections = [];
        try {
            const RTCPeerConnection = window.RTCPeerConnection ||
                window.webkitRTCPeerConnection ||
                window.mozRTCPeerConnection;
            if (!RTCPeerConnection) {
                score += 15;
                detections.push('no_webrtc');
                return { detected: true, score: Math.min(score, 100), detections };
            }
            const ips = new Set();
            const pc = new RTCPeerConnection({
                iceServers: [
                    { urls: 'stun:stun.l.google.com:19302' },
                    { urls: 'stun:stun1.l.google.com:19302' },
                    { urls: 'stun:stun2.l.google.com:19302' }
                ]
            });
            pc.createDataChannel('');
            const offer = await pc.createOffer();
            await pc.setLocalDescription(offer);
            const sdp = pc.localDescription.sdp;
            const lines = sdp.split('\n');
            for (const line of lines) {
                if (line.indexOf('candidate') > -1) {
                    const parts = line.split(' ');
                    if (parts[4] && parts[4] !== '0.0.0.0') {
                        ips.add(parts[4]);
                        if (parts[7] !== 'host') {
                            detections.push('relay_ip:' + parts[4]);
                        }
                    }
                }
            }
            pc.close();
            if (ips.size > 1) {
                const ipsArr = Array.from(ips);
                const privateIPs = ipsArr.filter(ip =>
                    ip.startsWith('10.') ||
                    ip.startsWith('172.16.') ||
                    ip.startsWith('192.168.')
                );
                const publicIPs = ipsArr.filter(ip => !privateIPs.includes(ip));
                if (publicIPs.length > 0) {
                    detections.push('public_ip_detected');
                    if (privateIPs.length > 0) {
                        score += 20;
                        detections.push('vpn_possible');
                    }
                }
            }
        } catch (e) {
            score += 15;
            detections.push('webrtc_error');
        }
        return { detected: score > 15, score: Math.min(score, 100), detections };
    }

    async detectNavigatorProps() {
        let score = 0;
        const detections = [];
        try {
            if (navigator.connection) {
                if (navigator.connection.type === 'none' &&
                    navigator.onLine === false) {
                    score += 20;
                    detections.push('offline_with_connection');
                }
            }
        } catch (e) {}
        try {
            if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
                const devices = await navigator.mediaDevices.enumerateDevices().catch(() => []);
                if (devices.length === 0) {
                    score += 15;
                    detections.push('no_media_devices');
                }
            } else {
                score += 10;
                detections.push('no_enumerate_devices');
            }
        } catch (e) {
            score += 15;
            detections.push('media_devices_error');
        }
        try {
            if (!navigator.credentials || !navigator.credentials.preventSilentAccess) {
                score += 5;
                detections.push('no_credentials_api');
            }
        } catch (e) {}
        try {
            if (navigator.serviceWorker === undefined) {
                score += 10;
                detections.push('no_serviceworker');
            }
        } catch (e) {}
        try {
            if (typeof navigator.getBattery === 'function') {
                const battery = await navigator.getBattery().catch(() => null);
                if (battery && battery.charging === undefined) {
                    score += 10;
                    detections.push('battery_no_charging');
                }
            }
        } catch (e) {}
        try {
            if (navigator.product === 'Gecko' && !/Firefox/i.test(navigator.userAgent || '')) {
                score += 20;
                detections.push('gecko_no_firefox');
            }
        } catch (e) {}
        try {
            if (navigator.vendor === '' && navigator.product === 'Gecko') {
            } else if (navigator.vendor === '') {
                score += 10;
                detections.push('empty_vendor');
            }
        } catch (e) {}
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectWindowProps() {
        let score = 0;
        const detections = [];
        try {
            const outerW = window.outerWidth;
            const outerH = window.outerHeight;
            const innerW = window.innerWidth;
            const innerH = window.innerHeight;
            if (outerW === 0 || outerH === 0) {
                score += 25;
                detections.push('zero_outer_size');
            }
            if (innerW > outerW || innerH > outerH) {
                score += 15;
                detections.push('inner_larger_than_outer');
            }
        } catch (e) {
            score += 20;
            detections.push('window_size_error');
        }
        try {
            if (window.screenX === undefined || window.screenY === undefined) {
                score += 10;
                detections.push('no_screen_position');
            }
        } catch (e) {}
        try {
            if (window.openDatabase === undefined) {
                score += 5;
                detections.push('no_opendatabase');
            }
        } catch (e) {}
        try {
            if (window.indexedDB === undefined) {
                score += 10;
                detections.push('no_indexeddb');
            }
        } catch (e) {}
        try {
            if (typeof window.postMessage !== 'function') {
                score += 20;
                detections.push('no_postmessage');
            }
        } catch (e) {}
        try {
            if (window.screenTop === undefined || window.screenLeft === undefined) {
                score += 5;
                detections.push('no_screen_edge');
            }
        } catch (e) {}
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectIframe() {
        let score = 0;
        const detections = [];
        try {
            if (window.self !== window.top) {
                score += 15;
                detections.push('in_iframe');
            }
        } catch (e) {
            score += 35;
            detections.push('cross_origin_frame');
        }
        try {
            const frameEl = document.createElement('iframe');
            frameEl.style.display = 'none';
            frameEl.sandbox = 'allow-scripts';
            document.body.appendChild(frameEl);
            const frameWin = frameEl.contentWindow;
            if (frameWin && frameWin.document) {
            }
            document.body.removeChild(frameEl);
        } catch (e) {
            score += 15;
            detections.push('iframe_access_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    async detectNotification() {
        let score = 0;
        const detections = [];
        try {
            if ('Notification' in window) {
                if (Notification.permission === 'denied') {
                    score += 5;
                    detections.push('notification_denied');
                }
            } else {
                score += 15;
                detections.push('no_notification');
            }
        } catch (e) {
            score += 15;
            detections.push('notification_error');
        }
        return { detected: score > 15, score: Math.min(score, 100), detections };
    }

    async detectBattery() {
        let score = 0;
        const detections = [];
        try {
            if (navigator.getBattery) {
                const battery = await navigator.getBattery().catch(() => null);
                if (battery) {
                    if (battery.level === undefined || battery.charging === undefined) {
                        score += 15;
                        detections.push('battery_props_missing');
                    }
                    if (battery.level === 0 && battery.charging === false) {
                        score += 5;
                        detections.push('battery_dead_not_charging');
                    }
                }
            } else {
                score += 10;
                detections.push('no_battery_api');
            }
        } catch (e) {
            score += 15;
            detections.push('battery_error');
        }
        return { detected: score > 15, score: Math.min(score, 100), detections };
    }

    async detectMediaDevices() {
        let score = 0;
        const detections = [];
        try {
            if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
                const devices = await navigator.mediaDevices.enumerateDevices().catch(() => []);
                const videoInputs = devices.filter(d => d.kind === 'videoinput');
                const audioInputs = devices.filter(d => d.kind === 'audioinput');
                if (videoInputs.length === 0 && audioInputs.length === 0) {
                    score += 20;
                    detections.push('no_media_inputs');
                }
                const allHaveLabels = devices.every(d => d.label !== '');
                if (!allHaveLabels) {
                    score += 10;
                    detections.push('media_no_labels');
                }
            } else {
                score += 15;
                detections.push('no_media_api');
            }
        } catch (e) {
            score += 20;
            detections.push('media_error');
        }
        return { detected: score > 20, score: Math.min(score, 100), detections };
    }

    async detectConnection() {
        let score = 0;
        const detections = [];
        try {
            const conn = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
            if (!conn) {
                score += 10;
                detections.push('no_connection_api');
            } else {
                if (conn.type === 'vpn') {
                    score += 40;
                    detections.push('vpn_detected');
                }
                if (conn.type === 'proxy') {
                    score += 40;
                    detections.push('proxy_detected');
                }
                if (conn.saveData === true) {
                    score += 10;
                    detections.push('save_data_enabled');
                }
                if (conn.effectiveType === 'slow-2g' || conn.effectiveType === '2g') {
                    score += 10;
                    detections.push('slow_connection');
                }
            }
        } catch (e) {
            score += 10;
            detections.push('connection_error');
        }
        return { detected: score > 20, score: Math.min(score, 100), detections };
    }

    async detectAdBlock() {
        let score = 0;
        const detections = [];
        try {
            const el = document.createElement('div');
            el.innerHTML = '&nbsp;';
            el.className = 'adsbox';
            el.style.cssText = 'position:absolute;left:-9999px;top:-9999px;width:1px;height:1px';
            document.body.appendChild(el);
            if (el.offsetHeight === 0) {
                score += 15;
                detections.push('adblock_detected');
            }
            document.body.removeChild(el);
        } catch (e) {
            score += 10;
            detections.push('adblock_check_error');
        }
        return { detected: score > 10, score: Math.min(score, 100), detections };
    }

    async detectMathFingerprint() {
        let score = 0;
        const detections = [];
        try {
            const mathResults = {
                sin: Math.sin(Math.PI / 3),
                tan: Math.tan(1e7),
                log10: Math.log10(100),
                asin: Math.asin(0.5),
                atan2: Math.atan2(1, 2),
                cos: Math.cos(Math.PI / 4),
                exp: Math.exp(1),
                sqrt: Math.sqrt(2)
            };
            for (const key in mathResults) {
                if (!isFinite(mathResults[key]) || isNaN(mathResults[key])) {
                    score += 15;
                    detections.push('math_' + key + '_invalid');
                }
            }
        } catch (e) {
            score += 20;
            detections.push('math_error');
        }
        return { detected: score > 15, score: Math.min(score, 100), detections };
    }

    async detectGPUFingerprint() {
        let score = 0;
        const detections = [];
        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
            if (gl) {
                const maxRenderSize = gl.getParameter(gl.MAX_RENDERBUFFER_SIZE);
                const maxViewport = gl.getParameter(gl.MAX_VIEWPORT_DIMS);
                const maxCombinedTexUnits = gl.getParameter(gl.MAX_COMBINED_TEXTURE_IMAGE_UNITS);
                if (maxRenderSize <= 1024) {
                    score += 15;
                    detections.push('small_renderbuffer');
                }
                if (maxCombinedTexUnits <= 8) {
                    score += 10;
                    detections.push('few_texture_units');
                }
            }
        } catch (e) {
            score += 10;
            detections.push('gpu_error');
        }
        return { detected: score > 15, score: Math.min(score, 100), detections };
    }

    async detectSpeech() {
        let score = 0;
        const detections = [];
        try {
            if ('speechSynthesis' in window) {
                const voices = window.speechSynthesis.getVoices();
                if (voices.length === 0) {
                    score += 10;
                    detections.push('no_speech_voices');
                }
            } else {
                score += 15;
                detections.push('no_speech_api');
            }
        } catch (e) {
            score += 10;
            detections.push('speech_error');
        }
        return { detected: score > 10, score: Math.min(score, 100), detections };
    }

    async detectVPNConnection() {
        let score = 0;
        const detections = [];
        try {
            const rtcPeerConnection = window.RTCPeerConnection || window.webkitRTCPeerConnection;
            if (rtcPeerConnection) {
                const ips = new Set();
                const pc = new rtcPeerConnection({ iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] });
                pc.createDataChannel('');
                try {
                    const offer = await pc.createOffer();
                    await pc.setLocalDescription(offer);
                    const sdp = pc.localDescription.sdp;
                    const lines = sdp.split('\n');
                    for (const line of lines) {
                        if (line.indexOf('candidate') > -1) {
                            const parts = line.split(' ');
                            if (parts[4] && parts[4] !== '0.0.0.0') {
                                const ip = parts[4];
                                if (!ip.startsWith('192.168.') && !ip.startsWith('10.') && !ip.startsWith('172.')) {
                                    score += 50;
                                    detections.push('external_ip_detected');
                                }
                            }
                        }
                    }
                } catch (e) {}
                pc.close();
            }

            if (navigator.connection) {
                const conn = navigator.connection;
                if (conn.type === 'vpn' || conn.type === 'pptp' || conn.type === 'tunnel') {
                    score += 55;
                    detections.push('vpn_connection_type');
                }
                if (conn.effectiveType === 'slow-2g' || conn.effectiveType === '2g') {
                    score += 20;
                    detections.push('slow_connection_vpn');
                }
            }
        } catch (e) {
            score += 15;
            detections.push('vpn_error');
        }
        return { detected: score > 40, score: Math.min(score, 100), detections };
    }

    async detectTorNetwork() {
        let score = 0;
        const detections = [];
        try {
            const ua = navigator.userAgent || '';
            if (/tor|onion/i.test(ua)) {
                score += 60;
                detections.push('tor_user_agent');
            }

            const rtcPeerConnection = window.RTCPeerConnection || window.webkitRTCPeerConnection;
            if (rtcPeerConnection) {
                const pc = new rtcPeerConnection({ iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] });
                pc.createDataChannel('');
                try {
                    const offer = await pc.createOffer();
                    await pc.setLocalDescription(offer);
                    const sdp = pc.localDescription.sdp;
                    if (/tls|inject_host_overwrite/i.test(sdp)) {
                        score += 50;
                        detections.push('tor_sdp_signature');
                    }
                } catch (e) {}
                pc.close();
            }
        } catch (e) {
            score += 25;
            detections.push('tor_error');
        }
        return { detected: score > 45, score: Math.min(score, 100), detections };
    }

    async collectEnhancedEnvironmentData() {
        const data = {};
        try {
            data.canvasHash = await this.generateCanvasHash();
        } catch (e) {
            data.canvasHash = '';
        }
        try {
            data.webglHash = await this.generateWebGLHash();
        } catch (e) {
            data.webglHash = '';
        }
        try {
            data.webglRenderer = this.getWebGLRenderer();
        } catch (e) {
            data.webglRenderer = '';
        }
        try {
            data.webglVendor = this.getWebGLVendor();
        } catch (e) {
            data.webglVendor = '';
        }
        try {
            data.audioHash = await this.generateAudioHash();
        } catch (e) {
            data.audioHash = '';
        }
        try {
            data.fonts = await this.detectFonts();
        } catch (e) {
            data.fonts = [];
        }
        try {
            data.plugins = Array.from(navigator.plugins || []).map(p => p.name);
        } catch (e) {
            data.plugins = [];
        }
        try {
            data.languages = navigator.languages || [navigator.language];
        } catch (e) {
            data.languages = [];
        }
        try {
            data.timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
        } catch (e) {
            data.timezone = '';
        }
        try {
            data.screen = {
                width: screen.width,
                height: screen.height,
                colorDepth: screen.colorDepth,
                pixelRatio: window.devicePixelRatio,
                availWidth: screen.availWidth,
                availHeight: screen.availHeight
            };
        } catch (e) {
            data.screen = {};
        }
        try {
            data.hardware = {
                concurrency: navigator.hardwareConcurrency,
                memory: navigator.deviceMemory,
                maxTouchPoints: navigator.maxTouchPoints
            };
        } catch (e) {
            data.hardware = {};
        }
        try {
            data.connection = navigator.connection ? {
                type: navigator.connection.type,
                effectiveType: navigator.connection.effectiveType,
                rtt: navigator.connection.rtt,
                downlink: navigator.connection.downlink
            } : {};
        } catch (e) {
            data.connection = {};
        }
        try {
            data.ipInfo = await this.getClientIPInfo();
        } catch (e) {
            data.ipInfo = {};
        }
        return data;
    }

    async generateCanvasHash() {
        const canvas = document.createElement('canvas');
        canvas.width = 280;
        canvas.height = 80;
        const ctx = canvas.getContext('2d');
        if (!ctx) return '';

        ctx.textBaseline = 'alphabetic';
        ctx.fillStyle = '#f60';
        ctx.fillRect(125, 1, 62, 20);
        ctx.fillStyle = '#069';
        ctx.font = '11pt Arial';
        ctx.fillText('Cwm fjordbank glyphs vext quiz, 😀', 2, 15);
        ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
        ctx.font = '18pt Arial';
        ctx.fillText('Cwm fjordbank glyphs vext quiz, 😀', 4, 45);

        ctx.globalCompositeOperation = 'multiply';
        ctx.fillStyle = 'rgb(255,0,255)';
        ctx.beginPath();
        ctx.arc(50, 50, 50, 0, Math.PI * 2, true);
        ctx.closePath();
        ctx.fill();

        const dataURL = canvas.toDataURL();
        return this.hashString(dataURL);
    }

    async generateWebGLHash() {
        const canvas = document.createElement('canvas');
        const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
        if (!gl) return '';

        const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
        if (!debugInfo) return '';

        const vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
        const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
        const combined = `${vendor}~${renderer}`;
        return this.hashString(combined);
    }

    getWebGLRenderer() {
        const canvas = document.createElement('canvas');
        const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
        if (!gl) return '';

        const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
        if (!debugInfo) return '';

        return gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) || '';
    }

    getWebGLVendor() {
        const canvas = document.createElement('canvas');
        const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
        if (!gl) return '';

        const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
        if (!debugInfo) return '';

        return gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL) || '';
    }

    async generateAudioHash() {
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
            osc.connect(compressor);
            compressor.connect(ctx.destination);
            osc.start(0);

            const buffer = await ctx.startRendering();
            const channelData = buffer.getChannelData(0);
            let hash = 0;
            for (let i = 0; i < 1000; i++) {
                hash = ((hash << 5) - hash) + channelData[i];
                hash = hash & hash;
            }
            return hash.toString(16);
        } catch (e) {
            return '';
        }
    }

    hashString(str) {
        let hash = 0;
        for (let i = 0; i < str.length; i++) {
            const char = str.charCodeAt(i);
            hash = ((hash << 5) - hash) + char;
            hash = hash & hash;
        }
        return Math.abs(hash).toString(16);
    }

    async runAll() {
        const chainResult = await this.runChain();
        const fingerprint = this.generateFingerprint();
        const enhancedData = await this.collectEnhancedEnvironmentData();
        return Object.assign(chainResult, { fingerprint, enhancedData });
    }

    generateFingerprint() {
        const components = [];
        try {
            components.push('scrn:' + screen.width + 'x' + screen.height + 'x' + screen.colorDepth);
        } catch (e) {}
        try {
            components.push('lang:' + (navigator.language || ''));
        } catch (e) {}
        try {
            components.push('tz:' + (Intl.DateTimeFormat().resolvedOptions().timeZone || ''));
        } catch (e) {}
        try {
            components.push('cpu:' + (navigator.hardwareConcurrency || ''));
        } catch (e) {}
        try {
            components.push('mem:' + (navigator.deviceMemory || ''));
        } catch (e) {}
        try {
            components.push('plat:' + (navigator.platform || ''));
        } catch (e) {}
        try {
            components.push('prod:' + (navigator.product || ''));
        } catch (e) {}
        try {
            components.push('vendor:' + (navigator.vendor || ''));
        } catch (e) {}
        try {
            const canvas = document.createElement('canvas');
            canvas.width = 100;
            canvas.height = 50;
            const ctx = canvas.getContext('2d');
            if (ctx) {
                ctx.textBaseline = 'top';
                ctx.font = '14px Arial';
                ctx.fillStyle = '#f60';
                ctx.fillRect(0, 0, 50, 50);
                ctx.fillStyle = '#069';
                ctx.fillText('fp', 10, 20);
                const dataUrl = canvas.toDataURL();
                const hash = dataUrl.split(',')[1] || dataUrl;
                components.push('cnv:' + hash.substring(0, 32));
            }
        } catch (e) {}
        try {
            const gl = document.createElement('canvas').getContext('webgl');
            if (gl) {
                const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                if (debugInfo) {
                    const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                    components.push('wgl:' + (renderer || '').substring(0, 48));
                }
            }
        } catch (e) {}
        try {
            const offset = new Date().getTimezoneOffset();
            components.push('tzoff:' + offset);
        } catch (e) {}
        try {
            components.push('cookie:' + (navigator.cookieEnabled ? '1' : '0'));
        } catch (e) {}
        try {
            if (navigator.languages && navigator.languages.length > 0) {
                components.push('langs:' + navigator.languages.join(','));
            }
        } catch (e) {}
        return components.join('|');
    }

    async detectEmulators() {
        let score = 0;
        const detections = [];
        const emulatorSignatures = {
            bluestacks: {
                patterns: [/bluestacks/i, /bst.*(?:special|split|instance)/i, /android.*bluestacks/i],
                indicators: ['BlueStacks', 'bst-helper', 'bstsdfsdksandbox', 'Android on x86'],
                weight: 0.95
            },
            nox: {
                patterns: [/nox/i, /noxplayer/i, /noxsandbox/i],
                indicators: ['Nox', 'NoxPlayer', 'NoxApp', 'android-virtualbox'],
                weight: 0.93
            },
            memu: {
                patterns: [/memu/i, /memuplay/i, /memuplayer/i],
                indicators: ['Memu', 'MemuPlayer', 'Microvirt', 'Android on MEmu'],
                weight: 0.92
            },
            ldplayer: {
                patterns: [/ldplayer/i, /ld-play/i, /leidian/i],
                indicators: ['LDPlayer', 'LeiDroid', 'ldlib', 'ldvbox'],
                weight: 0.91
            },
            mumu: {
                patterns: [/mumu/i, /mumux/i, /xiaomu/i],
                indicators: ['MuMu', 'mumu模拟器', 'Netease MuMu', 'mumu_x86'],
                weight: 0.88
            },
            genymotion: {
                patterns: [/genymotion/i, /genyotion/i],
                indicators: ['Genymotion', 'vbox86p', 'vbox86t', 'Genymobile'],
                weight: 0.94
            },
            gameloop: {
                patterns: [/gameloop/i, /tencentgameloop/i, /txgame/i],
                indicators: ['GameLoop', 'Tencent Gaming Buddy', 'android-gameloop'],
                weight: 0.90
            },
            smartgaga: {
                patterns: [/smartgaga/i, /smartga/i],
                indicators: ['SmartGaGa', 'SmartGaga', 'windows-sandbox'],
                weight: 0.87
            },
            windroy: {
                patterns: [/windroy/i, /windroye/i],
                indicators: ['WindRoy', 'WindRoye', 'Microvirt'],
                weight: 0.85
            },
            droid4x: {
                patterns: [/droid4x/i, /macsigner/i],
                indicators: ['Droid4X', 'droid4x-system'],
                weight: 0.84
            }
        };

        try {
            const ua = navigator.userAgent || '';
            const platform = navigator.platform || '';
            
            for (const [name, signature] of Object.entries(emulatorSignatures)) {
                let matched = false;
                
                for (const pattern of signature.patterns) {
                    if (pattern.test(ua)) {
                        score += 40 * signature.weight;
                        detections.push('emulator_pattern:' + name);
                        matched = true;
                        break;
                    }
                }
                
                if (!matched) {
                    for (const indicator of signature.indicators) {
                        if (ua.includes(indicator)) {
                            score += 35 * signature.weight;
                            detections.push('emulator_indicator:' + name);
                            matched = true;
                            break;
                        }
                    }
                }
            }

            const touchPoints = navigator.maxTouchPoints;
            if (touchPoints === 0 && /android|iphone|ipad/i.test(ua)) {
                score += 25;
                detections.push('no_touch_mobile_emulator');
            }

            if (/linux/i.test(platform) && /android|iphone/i.test(ua)) {
                score += 20;
                detections.push('linux_mobile_ua');
            }

        } catch (e) {
            score += 15;
            detections.push('emulator_detection_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectCloudPhones() {
        let score = 0;
        const detections = [];
        const cloudPhoneSignatures = {
            '雷电云': {
                patterns: [/ldyun/i, /ldy/i, /lei(dian)?.*cloud/i],
                indicators: ['LDYun', 'ldy_api', 'android-cloud-ldy', '雷电云手机'],
                weight: 0.96
            },
            '多多云': {
                patterns: [/ddyun/i, /ddy/i, /duoduo.*cloud/i],
                indicators: ['DDYun', 'ddy_api', 'android-cloud-ddy', '多多云手机'],
                weight: 0.95
            },
            '红警': {
                patterns: [/hongji/i, /redalert/i, /red.?alert/i],
                indicators: ['HongJi', 'hongjicloud', 'android-cloud-hj', '红警云'],
                weight: 0.94
            },
            '双子云': {
                patterns: [/shuangzi/i, /gemini.*cloud/i],
                indicators: ['ShuangZi', 'szcloud', 'android-cloud-sz'],
                weight: 0.90
            },
            '蜂窝云': {
                patterns: [/fengwo/i, /fwcloud/i, /beecow/i],
                indicators: ['FengWo', 'FWCloud', 'android-cloud-fw', '蜂窝云'],
                weight: 0.89
            },
            '云帅云': {
                patterns: [/yunshuai/i, /yscloud/i],
                indicators: ['YunShuai', 'YSCloud', 'android-cloud-ys'],
                weight: 0.88
            },
            '蓝光云': {
                patterns: [/languang/i, /lgcloud/i, /bluelight/i],
                indicators: ['LanGuang', 'LGCloud', 'android-cloud-lg'],
                weight: 0.87
            },
            '山寨云': {
                patterns: [/shanzhai/i, /szcloud/i],
                indicators: ['ShanZhai', 'SZCloud', 'android-cloud-sz'],
                weight: 0.86
            }
        };

        try {
            const ua = navigator.userAgent || '';
            
            for (const [name, signature] of Object.entries(cloudPhoneSignatures)) {
                let matched = false;
                
                for (const pattern of signature.patterns) {
                    if (pattern.test(ua)) {
                        score += 45 * signature.weight;
                        detections.push('cloud_phone_pattern:' + name);
                        matched = true;
                        break;
                    }
                }
                
                if (!matched) {
                    for (const indicator of signature.indicators) {
                        if (ua.includes(indicator)) {
                            score += 40 * signature.weight;
                            detections.push('cloud_phone_indicator:' + name);
                            matched = true;
                            break;
                        }
                    }
                }
            }

            const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
            if (timezone && /china|beijing|shanghai/i.test(timezone)) {
                const canvas = document.createElement('canvas');
                const gl = canvas.getContext('webgl');
                if (gl) {
                    const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                    if (debugInfo) {
                        const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                        if (/virtualbox|vbox|vmware|qemu|kvm/i.test(renderer)) {
                            score += 30;
                            detections.push('cloud_phone_webgl');
                        }
                    }
                }
            }

        } catch (e) {
            score += 15;
            detections.push('cloud_phone_detection_error');
        }
        return { detected: score > 35, score: Math.min(score, 100), detections };
    }

    async detectVirtualMachines() {
        let score = 0;
        const detections = [];
        const vmSignatures = {
            vmware: {
                patterns: [/vmware/i, /vmware.*virtual/i, /vmware[_-]?tools/i],
                indicators: ['VMware7,1', 'VMware Virtual Platform', 'VMware Vista', 'VMware7'],
                weight: 0.95
            },
            virtualbox: {
                patterns: [/virtualbox/i, /vbox/i, /vboxclient/i],
                indicators: ['VirtualBox', 'VBOX', 'vbox86p', 'vbox86t', 'VBoxSharedFolders'],
                weight: 0.92
            },
            hyperv: {
                patterns: [/hyper[_-]?v/i, /microsoft.*virtual/i],
                indicators: ['Microsoft Hyper-V', 'Virtual Machine', 'HYPER-V', 'Microsoft Corporation'],
                weight: 0.90
            },
            parallels: {
                patterns: [/parallels/i, /prl_/i],
                indicators: ['Parallels', 'Parallels Virtual Platform', 'prl_vm_app'],
                weight: 0.89
            },
            qemu: {
                patterns: [/qemu/i, /kvm/i, /tcg/i],
                indicators: ['QEMU Virtual CPU', 'KVM', 'Standard PC (Q35 + ICH9', 'TCG'],
                weight: 0.88
            },
            xen: {
                patterns: [/xen/i],
                indicators: ['Xen', 'HVM domU', 'XenSource'],
                weight: 0.87
            }
        };

        try {
            const ua = navigator.userAgent || '';
            
            for (const [name, signature] of Object.entries(vmSignatures)) {
                let matched = false;
                
                for (const pattern of signature.patterns) {
                    if (pattern.test(ua)) {
                        score += 40 * signature.weight;
                        detections.push('vm_pattern:' + name);
                        matched = true;
                        break;
                    }
                }
                
                if (!matched) {
                    for (const indicator of signature.indicators) {
                        if (ua.includes(indicator)) {
                            score += 35 * signature.weight;
                            detections.push('vm_indicator:' + name);
                            matched = true;
                            break;
                        }
                    }
                }
            }

            const cpu = navigator.hardwareConcurrency;
            if (cpu && cpu < 2) {
                score += 20;
                detections.push('low_core_count');
            }

            const mem = navigator.deviceMemory;
            if (mem && mem < 1) {
                score += 25;
                detections.push('low_device_memory');
            }

            try {
                const canvas = document.createElement('canvas');
                const gl = canvas.getContext('webgl');
                if (gl) {
                    const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                    if (debugInfo) {
                        const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                        if (/vmware|virtualbox|parallels|qemu|kvm|hyperv/i.test(renderer)) {
                            score += 45;
                            detections.push('vm_webgl_renderer');
                        }
                    }
                }
            } catch (e) {}

        } catch (e) {
            score += 20;
            detections.push('vm_detection_error');
        }
        return { detected: score > 30, score: Math.min(score, 100), detections };
    }

    async detectContainers() {
        let score = 0;
        const detections = [];
        const containerSignatures = {
            docker: {
                patterns: [/docker/i],
                indicators: ['/.dockerenv', 'container=docker', 'docker-init', 'docker-cluster'],
                weight: 0.85
            },
            kubernetes: {
                patterns: [/kubernetes/i, /k8s/i],
                indicators: ['KUBERNETES_SERVICE_PORT', 'kubernetes.io', 'kube-cluster'],
                weight: 0.82
            },
            lxc: {
                patterns: [/lxc/i],
                indicators: ['/dev/lxd/sock', 'lxc/', 'machine-id'],
                weight: 0.80
            },
            cgroup: {
                patterns: [/container/i],
                indicators: ['1:freezer:/', '1:name=systemd:', '/sys/fs/cgroup/'],
                weight: 0.75
            }
        };

        try {
            const ua = navigator.userAgent || '';
            
            for (const [name, signature] of Object.entries(containerSignatures)) {
                let matched = false;
                
                for (const pattern of signature.patterns) {
                    if (pattern.test(ua)) {
                        score += 35 * signature.weight;
                        detections.push('container_pattern:' + name);
                        matched = true;
                        break;
                    }
                }
                
                if (!matched) {
                    for (const indicator of signature.indicators) {
                        if (ua.includes(indicator)) {
                            score += 30 * signature.weight;
                            detections.push('container_indicator:' + name);
                            matched = true;
                            break;
                        }
                    }
                }
            }

            if (navigator.storage && navigator.storage.estimate) {
                try {
                    const est = await navigator.storage.estimate();
                    if (est.quota === 0) {
                        score += 25;
                        detections.push('zero_storage_quota');
                    }
                } catch (e) {}
            }

            if (navigator.cookieEnabled === false) {
                score += 20;
                detections.push('cookies_disabled');
            }

            if (navigator.hardwareConcurrency && navigator.hardwareConcurrency > 64) {
                score += 15;
                detections.push('unrealistic_cores');
            }

        } catch (e) {
            score += 15;
            detections.push('container_detection_error');
        }
        return { detected: score > 25, score: Math.min(score, 100), detections };
    }

    toJSON() {
        return {
            risk_score: this.riskScore,
            chain_count: this.detectionChain.length,
            results: this.results
        };
    }
}

if (typeof window !== 'undefined') {
    window.EnvironmentDetector = EnvironmentDetector;
}
