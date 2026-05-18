const FrontendProtection = (function() {
    'use strict';

    const _0xProtection = {
        version: '2.0',
        enabled: true,
        config: {
            enableDebugDetection: true,
            enableIntegrityCheck: true,
            enableTimingProtection: true,
            enableMemoryProtection: true,
            enableBreakpointDetection: true,
            checkInterval: 3000,
            maxViolations: 3
        },
        violations: 0,
        startTime: Date.now(),
        baselineTiming: 0,
        timingSamples: []
    };

    const debugDetectors = [
        function() {
            const threshold = 160;
            return window.outerWidth - window.innerWidth > threshold ||
                   window.outerHeight - window.innerHeight > threshold;
        },
        function() {
            const start = performance.now();
            debugger;
            const end = performance.now();
            return end - start > 50;
        },
        function() {
            const testFunc = function() {};
            testFunc.toString = function() {
                return window.devtools && window.devtools.isOpen ? 'true' : 'false';
            };
            console.log(testFunc);
            return false;
        },
        function() {
            return typeof console._commandLineAPI !== 'undefined' ||
                   typeof console.profiles !== 'undefined' ||
                   window.firebug;
        },
        function() {
            if (window.webkitDebuggerAPI) return true;
            if (window.chrome && window.chrome.runtime) return true;
            return false;
        },
        function() {
            const start = Date.now();
            try {
                eval('debugger;');
            } catch (e) {
                return true;
            }
            const end = Date.now();
            return end - start > 50;
        }
    ];

    const detectDevTools = function() {
        for (let i = 0; i < debugDetectors.length; i++) {
            try {
                if (debugDetectors[i]()) {
                    return true;
                }
            } catch (e) {
                continue;
            }
        }
        return false;
    };

    const recordTiming = function() {
        const now = Date.now();
        const elapsed = now - _0xProtection.startTime;
        _0xProtection.timingSamples.push(elapsed);

        if (_0xProtection.timingSamples.length > 10) {
            _0xProtection.timingSamples.shift();
        }

        if (_0xProtection.timingSamples.length === 10) {
            const sum = _0xProtection.timingSamples.reduce((a, b) => a + b, 0);
            _0xProtection.baselineTiming = sum / 10;
        }

        return elapsed;
    };

    const checkTimingAnomaly = function() {
        const currentTiming = recordTiming();

        if (_0xProtection.baselineTiming > 0) {
            const deviation = Math.abs(currentTiming - _0xProtection.baselineTiming);
            const threshold = _0xProtection.baselineTiming * 3;

            if (deviation > threshold) {
                return true;
            }
        }

        return false;
    };

    const checkMemoryIntegrity = function() {
        const markers = ['__protection_marker_1', '__protection_marker_2'];
        const markerValue = 'integrity_check_' + Date.now();

        markers.forEach(function(marker) {
            const el = document.getElementById(marker);
            if (el && el.getAttribute('data-v') !== markerValue) {
                throw new Error('Memory integrity compromised');
            }
        });

        return true;
    };

    const createProtectionMarkers = function() {
        const markerValue = 'integrity_check_' + Date.now();
        const markers = ['__protection_marker_1', '__protection_marker_2', '__protection_marker_3'];

        markers.forEach(function(marker) {
            const el = document.createElement('div');
            el.id = marker;
            el.style.display = 'none';
            el.setAttribute('data-v', markerValue);
            document.body.appendChild(el);
        });

        return markerValue;
    };

    const blockAccess = function() {
        _0xProtection.enabled = false;
        document.documentElement.style.display = 'none';
        document.body.innerHTML = '<div style="position:fixed;top:0;left:0;width:100%;height:100%;background:#000;color:#fff;display:flex;justify-content:center;align-items:center;font-family:Arial,sans-serif;z-index:2147483647;"><div style="text-align:center;padding:40px;"><h1 style="margin:0 0 20px 0;font-size:48px;">Access Restricted</h1><p style="margin:0;font-size:18px;opacity:0.8;">Security violation detected</p></div></div>';
        throw new Error('Protection triggered');
    };

    const handleViolation = function() {
        _0xProtection.violations++;

        if (_0xProtection.violations >= _0xProtection.config.maxViolations) {
            blockAccess();
            return;
        }

        console.warn('Security violation detected: ' + _0xProtection.violations);
    };

    const performChecks = function() {
        if (!_0xProtection.enabled) return;

        if (_0xProtection.config.enableDebugDetection && detectDevTools()) {
            handleViolation();
            return;
        }

        if (_0xProtection.config.enableTimingProtection && checkTimingAnomaly()) {
            handleViolation();
            return;
        }

        if (_0xProtection.config.enableMemoryProtection) {
            try {
                checkMemoryIntegrity();
            } catch (e) {
                handleViolation();
                return;
            }
        }
    };

    const init = function() {
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', function() {
                createProtectionMarkers();
                startProtection();
            });
        } else {
            createProtectionMarkers();
            startProtection();
        }
    };

    const startProtection = function() {
        setInterval(performChecks, _0xProtection.config.checkInterval);

        document.addEventListener('keydown', function(e) {
            if (e.key === 'F12' ||
                (e.ctrlKey && e.shiftKey && e.key === 'I') ||
                (e.ctrlKey && e.shiftKey && e.key === 'J') ||
                (e.ctrlKey && e.shiftKey && e.key === 'C') ||
                (e.ctrlKey && e.key === 'u')) {
                e.preventDefault();
                handleViolation();
            }
        });

        document.addEventListener('contextmenu', function(e) {
            e.preventDefault();
        });

        Object.defineProperty(document, 'hidden', {
            get: function() {
                handleViolation();
                return false;
            },
            configurable: false
        });

        Object.defineProperty(document, 'visibilityState', {
            get: function() {
                return 'visible';
            },
            configurable: false
        });
    };

    const getProtectionStatus = function() {
        return {
            enabled: _0xProtection.enabled,
            violations: _0xProtection.violations,
            startTime: _0xProtection.startTime,
            config: _0xProtection.config
        };
    };

    const setConfig = function(config) {
        Object.assign(_0xProtection.config, config);
    };

    init();

    return {
        getStatus: getProtectionStatus,
        setConfig: setConfig,
        performCheck: performChecks,
        block: blockAccess
    };
})();

if (typeof module !== 'undefined' && module.exports) {
    module.exports = FrontendProtection;
}
