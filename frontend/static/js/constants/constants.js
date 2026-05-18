(function(globalContext) {
    'use strict';

    var CaptchaConstants = {
        API_BASE: '/api/v1',
        SAMPLE_RATE: 0.3,
        CHAIN_COUNT: 12,

        DETECTION_WEIGHTS: {
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
            speech: 3
        },

        AUTO_TOOLS: [
            'detectWebDriver',
            'detectPuppeteer',
            'detectPlaywright',
            'detectSelenium'
        ],

        PROXY_INDICATORS: [
            'detectWebRTCIP',
            'detectConnection'
        ],

        RISK_THRESHOLDS: {
            AUTO_DETECT_MULTIPLE: 2,
            AUTO_SCORE_MULTIPLIER_HIGH: 1.5,
            AUTO_SCORE_MULTIPLIER_LOW: 1.3,
            PROXY_ANOMALY_MULTIPLE: 2,
            PROXY_SCORE_MULTIPLIER: 1.3,
            MAX_SCORE: 100
        },

        WEBDRIVER_PROPS: [
            'webdriver',
            '__webdriver_evaluate',
            '__selenium_evaluate',
            '__webdriver_script_fn',
            '__driver_evaluate',
            '__fxdriver_evaluate',
            '__webdriver_unwrapped',
            '__lastWatirAlert',
            '__$webdriverAsyncExecutor',
            'callSelenium',
            '__selenium',
            'Selenium'
        ],

        SELENIUM_PROPS: [
            'selenium',
            '_selenium',
            'callSelenium',
            '__selenium',
            'document__selenium',
            'Selenium',
            '__webdriver_script_fn',
            'Selenium.prototype'
        ],

        PUPPETEER_MARKERS: [
            '$cdc_asdjflasutopfhvcZLmcfl_',
            '$chrome_asyncScriptInfo',
            '_puppeteer_globals'
        ],

        PLAYWRIGHT_GLOBALS: [
            '__playwright__',
            '__pw_tags',
            '__pw_resume__'
        ],

        COMMON_PLUGINS: [
            'PDF Viewer',
            'Chrome PDF Viewer',
            'Chromium PDF Viewer',
            'Microsoft Edge PDF Viewer',
            'WebKit built-in PDF'
        ],

        COMMON_FONTS: [
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
        ],

        BASE_FONTS: ['monospace', 'sans-serif', 'serif'],

        STUN_SERVERS: [
            { urls: 'stun:stun.l.google.com:19302' },
            { urls: 'stun:stun1.l.google.com:19302' },
            { urls: 'stun:stun2.l.google.com:19302' }
        ],

        PERMISSION_NAMES: [
            'notifications',
            'geolocation',
            'camera',
            'microphone',
            'midi'
        ],

        MATH_FUNCTIONS: [
            'sin', 'tan', 'log10', 'asin', 'atan2',
            'cos', 'exp', 'sqrt'
        ],

        SOFTWARE_RENDERERS: [
            'swiftshader',
            'llvmpipe',
            'mesa',
            'virtual',
            'google inc'
        ],

        DEFAULT_KEY: 'hjtpx-obfuscate-key-2024',
        STORAGE_PREFIX: '_cry_',

        PBKDF2_ITERATIONS: 100000,
        KEY_LENGTH: 256,
        IV_LENGTH_GCM: 12,
        IV_LENGTH_CBC: 16,

        CAPTCHA_TYPES: {
            SLIDER: 'slider',
            CLICK: 'click',
            VOICE: 'voice',
            THREE_D: '3d',
            LIANLIANKAN: 'lianliankan'
        },

        CAPTCHA_CONFIG: {
            SLIDER: {
                width: 360,
                height: 46,
                puzzleSize: 50,
                maxAttempts: 3
            },
            CLICK: {
                width: 360,
                gridSize: 9,
                requiredClicks: 4
            },
            VOICE: {
                duration: 5000,
                maxRetries: 3
            }
        },

        ERROR_MESSAGES: {
            NETWORK_ERROR: '网络请求失败，请检查网络连接',
            TIMEOUT_ERROR: '请求超时，请稍后重试',
            VALIDATION_ERROR: '验证失败，请重试',
            RATE_LIMIT_ERROR: '请求过于频繁，请稍后重试',
            SESSION_EXPIRED: '会话已过期，请重新登录',
            PERMISSION_DENIED: '权限不足，无法访问该资源',
            SERVER_ERROR: '服务器内部错误，请稍后重试',
            UNKNOWN_ERROR: '发生未知错误，请刷新重试'
        },

        UI_COLORS: {
            PRIMARY: '#c9a96e',
            PRIMARY_GRADIENT: 'linear-gradient(135deg, #c9a96e 0%, #d4b87a 100%)',
            SUCCESS: '#28a745',
            ERROR: '#dc3545',
            WARNING: '#ffc107',
            INFO: '#17a2b8'
        },

        ANIMATION_DURATION: {
            FAST: 150,
            NORMAL: 250,
            SLOW: 400
        },

        DEBUG_THRESHOLD: 160,
        INTEGRITY_CHECK_INTERVAL: 5000
    };

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = CaptchaConstants;
    } else {
        globalContext.CaptchaConstants = CaptchaConstants;
    }

})(typeof window !== 'undefined' ? window : this);
