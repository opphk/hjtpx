const AutomationDetector = (function() {
    'use strict';

    const VERSION = '15.0.0';

    const DetectionConfig = {
        enableSelenium: true,
        enablePlaywright: true,
        enablePuppeteer: true,
        enablePhantomJS: true,
        enableHeadless: true,
        enableHeadlessChrome: true,
        enableWebDriver: true,
        enablePlugins: true,
        enableLanguages: true,
        enableUserAgent: true,
        enableWindowSize: true,
        enableTiming: true,
        enableConsole: true,
        enablePermissions: true,
        checkInterval: 3000,
        maxViolations: 3,
        strictMode: true,
        blockOnDetection: true
    };

    let detectedTypes = [];
    let violations = 0;
    let isActive = true;
    let checkCount = 0;
    let startTime = Date.now();

    const SeleniumPatterns = [
        'webdriver',
        '__webdriver_script_function',
        '__webdriver_script_func',
        '__webdriver_script_fn',
        'selenium',
        'selenium-',
        'SLIMERJS',
        'callSelenium',
        '_selenium'
    ];

    const PlaywrightPatterns = [
        '__playwright',
        'playwright',
        '__pw',
        'pw_api',
        'playwright-replaced'
    ];

    const PuppeteerPatterns = [
        'puppeteer',
        '__puppeteer',
        'puppeteer_replaced',
        'chrome-pdf'
    ];

    const PhantomJSPatterns = [
        'phantomjs',
        '__phantomjs',
        'callPhantom',
        '_phantom',
        'phantom'
    ];

    function log(message) {
        if (typeof console !== 'undefined' && console.debug) {
            console.debug('[Automation Detector ' + VERSION + ']:', message);
        }
    }

    function error(message) {
        if (typeof console !== 'undefined' && console.error) {
            console.error('[Automation Detector ' + VERSION + ' Error]:', message);
        }
    }

    function checkSelenium() {
        if (!DetectionConfig.enableSelenium) {
            return { detected: false };
        }

        for (let i = 0; i < SeleniumPatterns.length; i++) {
            if (window[SeleniumPatterns[i]] !== undefined) {
                return {
                    detected: true,
                    pattern: SeleniumPatterns[i],
                    type: 'selenium',
                    confidence: 0.9
                };
            }
        }

        if (navigator.userAgent.indexOf('webdriver') !== -1) {
            return {
                detected: true,
                pattern: 'userAgent',
                type: 'selenium',
                confidence: 0.85
            };
        }

        return { detected: false };
    }

    function checkPlaywright() {
        if (!DetectionConfig.enablePlaywright) {
            return { detected: false };
        }

        for (let i = 0; i < PlaywrightPatterns.length; i++) {
            if (window[PlaywrightPatterns[i]] !== undefined) {
                return {
                    detected: true,
                    pattern: PlaywrightPatterns[i],
                    type: 'playwright',
                    confidence: 0.9
                };
            }
        }

        return { detected: false };
    }

    function checkPuppeteer() {
        if (!DetectionConfig.enablePuppeteer) {
            return { detected: false };
        }

        for (let i = 0; i < PuppeteerPatterns.length; i++) {
            if (window[PuppeteerPatterns[i]] !== undefined) {
                return {
                    detected: true,
                    pattern: PuppeteerPatterns[i],
                    type: 'puppeteer',
                    confidence: 0.9
                };
            }
        }

        return { detected: false };
    }

    function checkPhantomJS() {
        if (!DetectionConfig.enablePhantomJS) {
            return { detected: false };
        }

        for (let i = 0; i < PhantomJSPatterns.length; i++) {
            if (window[PhantomJSPatterns[i]] !== undefined) {
                return {
                    detected: true,
                    pattern: PhantomJSPatterns[i],
                    type: 'phantomjs',
                    confidence: 0.9
                };
            }
        }

        return { detected: false };
    }

    function checkWebDriver() {
        if (!DetectionConfig.enableWebDriver) {
            return { detected: false };
        }

        if (navigator.webdriver === true) {
            return {
                detected: true,
                pattern: 'navigator.webdriver',
                type: 'webdriver',
                confidence: 0.95
            };
        }

        return { detected: false };
    }

    function checkHeadless() {
        if (!DetectionConfig.enableHeadless && !DetectionConfig.enableHeadlessChrome) {
            return { detected: false };
        }

        if (navigator.userAgent.indexOf('HeadlessChrome') !== -1) {
            return {
                detected: true,
                pattern: 'HeadlessChrome',
                type: 'headless',
                confidence: 0.9
            };
        }

        if (navigator.userAgent.indexOf('PhantomJS') !== -1) {
            return {
                detected: true,
                pattern: 'PhantomJS',
                type: 'headless',
                confidence: 0.85
            };
        }

        return { detected: false };
    }

    function checkPlugins() {
        if (!DetectionConfig.enablePlugins) {
            return { detected: false };
        }

        const plugins = navigator.plugins;
        if (!plugins || plugins.length < 3) {
            return {
                detected: true,
                pattern: 'low_plugin_count',
                type: 'headless',
                confidence: 0.6
            };
        }

        return { detected: false };
    }

    function checkLanguages() {
        if (!DetectionConfig.enableLanguages) {
            return { detected: false };
        }

        const languages = navigator.languages;
        if (!languages || languages.length === 0) {
            return {
                detected: true,
                pattern: 'no_languages',
                type: 'headless',
                confidence: 0.5
            };
        }

        return { detected: false };
    }

    function checkWindowSize() {
        if (!DetectionConfig.enableWindowSize) {
            return { detected: false };
        }

        if (window.outerWidth === 0 && window.outerHeight === 0) {
            return {
                detected: true,
                pattern: 'zero_window_size',
                type: 'headless',
                confidence: 0.8
            };
        }

        const threshold = 160;
        if (window.outerWidth - window.innerWidth > threshold ||
            window.outerHeight - window.innerHeight > threshold) {
            return {
                detected: true,
                pattern: 'devtools_open',
                type: 'devtools',
                confidence: 0.7
            };
        }

        return { detected: false };
    }

    function checkTiming() {
        if (!DetectionConfig.enableTiming) {
            return { detected: false };
        }

        const start = Date.now();
        debugger;
        const end = Date.now();

        if (end - start > 100) {
            return {
                detected: true,
                pattern: 'debugger_timing',
                type: 'debugger',
                confidence: 0.85
            };
        }

        return { detected: false };
    }

    function checkConsole() {
        if (!DetectionConfig.enableConsole) {
            return { detected: false };
        }

        const originalClear = console.clear;
        console.clear = function() {};

        try {
            const testFunc = function() {};
            testFunc.toString = function() {
                return 'function () { [native code] }';
            };
            console.log(testFunc);
        } catch (e) {}

        console.clear = originalClear;

        return { detected: false };
    }

    function checkPermissions() {
        if (!DetectionConfig.enablePermissions) {
            return { detected: false };
        }

        if (navigator.permissions && navigator.permissions.query) {
            return { detected: false };
        }

        return {
            detected: true,
            pattern: 'no_permissions_api',
            type: 'headless',
            confidence: 0.4
        };
    }

    function performDetection() {
        if (!isActive) {
            return { detected: false, types: [] };
        }

        const checks = [
            checkSelenium,
            checkPlaywright,
            checkPuppeteer,
            checkPhantomJS,
            checkWebDriver,
            checkHeadless,
            checkPlugins,
            checkLanguages,
            checkWindowSize,
            checkTiming,
            checkConsole,
            checkPermissions
        ];

        const detections = [];

        for (let i = 0; i < checks.length; i++) {
            try {
                const result = checks[i]();
                if (result.detected) {
                    detections.push(result);
                    if (detectedTypes.indexOf(result.type) === -1) {
                        detectedTypes.push(result.type);
                    }
                }
            } catch (e) {
                continue;
            }
        }

        checkCount++;

        if (detections.length > 0) {
            violations++;
            log('Automation detected: ' + JSON.stringify(detections));

            if (DetectionConfig.blockOnDetection && violations >= DetectionConfig.maxViolations) {
                blockAccess();
            }
        }

        return {
            detected: detections.length > 0,
            types: detectedTypes.slice(),
            detections: detections,
            violations: violations,
            checkCount: checkCount
        };
    }

    function blockAccess() {
        isActive = false;
        document.documentElement.style.display = 'none';
        document.body.innerHTML = '<div style="position:fixed;top:0;left:0;width:100%;height:100%;background:#000;color:#fff;display:flex;justify-content:center;align-items:center;font-family:Arial,sans-serif;z-index:2147483647;"><div style="text-align:center;padding:40px;"><h1 style="margin:0 0 20px 0;font-size:48px;">Access Restricted</h1><p style="margin:0;font-size:18px;opacity:0.8;">Automated access detected and blocked.</p></div></div>';
        throw new Error('Automation detected: Access blocked');
    }

    function setupEventListeners() {
        document.addEventListener('keydown', function(e) {
            if (e.key === 'F12' ||
                (e.ctrlKey && e.shiftKey && e.key === 'I') ||
                (e.ctrlKey && e.shiftKey && e.key === 'J') ||
                (e.ctrlKey && e.shiftKey && e.key === 'C')) {
                e.preventDefault();
                violations++;
                log('Keyboard shortcut blocked: ' + e.key);
            }
        });

        document.addEventListener('contextmenu', function(e) {
            e.preventDefault();
            return false;
        });

        document.addEventListener('dragstart', function(e) {
            e.preventDefault();
            return false;
        });
    }

    function startProtection() {
        setupEventListeners();

        setInterval(function() {
            performDetection();
        }, DetectionConfig.checkInterval);

        log('Automation detection started');
    }

    function getStatus() {
        return {
            active: isActive,
            detectedTypes: detectedTypes.slice(),
            violations: violations,
            checkCount: checkCount,
            startTime: startTime,
            config: Object.assign({}, DetectionConfig)
        };
    }

    function reset() {
        detectedTypes = [];
        violations = 0;
        checkCount = 0;
        startTime = Date.now();
        isActive = true;
    }

    function enable() {
        isActive = true;
    }

    function disable() {
        isActive = false;
    }

    const DetectorAPI = {
        version: VERSION,
        detect: performDetection,
        getStatus: getStatus,
        reset: reset,
        enable: enable,
        disable: disable,
        block: blockAccess,
        checkSelenium: checkSelenium,
        checkPlaywright: checkPlaywright,
        checkPuppeteer: checkPuppeteer,
        checkPhantomJS: checkPhantomJS,
        checkWebDriver: checkWebDriver,
        checkHeadless: checkHeadless,
        checkPlugins: checkPlugins,
        checkLanguages: checkLanguages,
        checkWindowSize: checkWindowSize,
        checkTiming: checkTiming,
        setConfig: function(config) {
            Object.assign(DetectionConfig, config);
        },
        getConfig: function() {
            return Object.assign({}, DetectionConfig);
        }
    };

    if (typeof window !== 'undefined') {
        window.AutomationDetector = DetectorAPI;
        window._0xauto = DetectorAPI;
    }

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = DetectorAPI;
    }

    startProtection();

    return DetectorAPI;
})();
