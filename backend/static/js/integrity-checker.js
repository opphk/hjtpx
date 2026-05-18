const IntegrityChecker = (function() {
    'use strict';

    const _0xIC = {
        version: '2.0',
        hashes: {},
        timestamp: null,
        checkInterval: 15000,
        maxChecks: 50,
        checkCount: 0,
        enabled: true,
        markers: []
    };

    const generateHash = function(data) {
        let hash = 0;
        if (data.length === 0) return hash.toString();

        for (let i = 0; i < data.length; i++) {
            const char = data.charCodeAt(i);
            hash = ((hash << 5) - hash) + char;
            hash = hash & hash;
        }

        return Math.abs(hash).toString(16);
    };

    const createMarkers = function() {
        const markerCount = 5;
        for (let i = 0; i < markerCount; i++) {
            const marker = document.createElement('div');
            marker.id = '_0xIC_marker_' + i;
            marker.style.display = 'none';
            marker.setAttribute('data-v', _0xIC.hashes.sha256);
            document.body.appendChild(marker);
            _0xIC.markers.push(marker.id);
        }
    };

    const verifyMarkers = function() {
        for (let i = 0; i < _0xIC.markers.length; i++) {
            const el = document.getElementById(_0xIC.markers[i]);
            if (!el || el.getAttribute('data-v') !== _0xIC.hashes.sha256) {
                return false;
            }
        }
        return true;
    };

    const verifyTiming = function() {
        const start = performance.now();
        let sum = 0;
        for (let i = 0; i < 1000; i++) {
            sum += Math.random() * i;
        }
        const end = performance.now();

        return (end - start) < 100;
    };

    const verifyDOMIntegrity = function() {
        const criticalElements = [
            'script[src*="main"]',
            'script[src*="core"]',
            'link[href*="style"]'
        ];

        criticalElements.forEach(function(selector) {
            const elements = document.querySelectorAll(selector);
            elements.forEach(function(el) {
                if (el integrity && el.integrity !== _0xIC.hashes.sha256) {
                    return false;
                }
            });
        });

        return true;
    };

    const verify = function() {
        if (_0xIC.checkCount >= _0xIC.maxChecks) {
            stop();
            return true;
        }

        if (!verifyMarkers()) {
            handleFailure('Marker verification failed');
            return false;
        }

        if (!verifyTiming()) {
            handleFailure('Timing verification failed');
            return false;
        }

        if (!verifyDOMIntegrity()) {
            handleFailure('DOM integrity verification failed');
            return false;
        }

        _0xIC.checkCount++;
        return true;
    };

    const handleFailure = function(reason) {
        _0xIC.enabled = false;
        stop();

        document.documentElement.style.display = 'none';
        document.body.innerHTML = '<div style="position:fixed;top:0;left:0;width:100%;height:100%;background:#000;color:#fff;display:flex;justify-content:center;align-items:center;font-family:Arial,sans-serif;"><div><h1>Integrity Check Failed</h1><p>Code has been tampered with</p><p>' + reason + '</p></div></div>';

        throw new Error('Integrity verification failed: ' + reason);
    };

    const start = function() {
        createMarkers();

        _0xIC.timer = setInterval(function() {
            verify();
        }, _0xIC.checkInterval);

        window.addEventListener('beforeunload', function() {
            stop();
        });
    };

    const stop = function() {
        if (_0xIC.timer) {
            clearInterval(_0xIC.timer);
            _0xIC.timer = null;
        }
    };

    const init = function(hashes, config) {
        _0xIC.hashes = hashes || {};
        _0xIC.timestamp = new Date().toISOString();

        if (config) {
            _0xIC.checkInterval = config.checkInterval || 15000;
            _0xIC.maxChecks = config.maxChecks || 50;
        }

        start();
    };

    const getStatus = function() {
        return {
            enabled: _0xIC.enabled,
            checkCount: _0xIC.checkCount,
            maxChecks: _0xIC.maxChecks,
            hashes: _0xIC.hashes,
            timestamp: _0xIC.timestamp
        };
    };

    const verifyManual = function(code) {
        const currentHash = generateHash(code);
        return currentHash === _0xIC.hashes.sha256;
    };

    return {
        init: init,
        start: start,
        stop: stop,
        verify: verify,
        getStatus: getStatus,
        verifyManual: verifyManual
    };
})();

if (typeof module !== 'undefined' && module.exports) {
    module.exports = IntegrityChecker;
}
