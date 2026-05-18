const IntegrityChecker = (function() {
    'use strict';

    const _0xIC = {
        version: '3.0',
        hashes: {},
        timestamp: null,
        checkInterval: 15000,
        maxChecks: 50,
        checkCount: 0,
        enabled: true,
        markers: [],
        crc32Table: [],
        crc32Initialized: false
    };

    const initCRC32 = function() {
        if (_0xIC.crc32Initialized) return;
        
        for (let i = 0; i < 256; i++) {
            let c = i;
            for (let j = 0; j < 8; j++) {
                c = (c & 1) ? (0xEDB88320 ^ (c >>> 1)) : (c >>> 1)
            }
            _0xIC.crc32Table[i] = c
        }
        _0xIC.crc32Initialized = true
    }

    const computeCRC32 = function(data) {
        initCRC32()
        let crc = 0xFFFFFFFF
        for (let i = 0; i < data.length; i++) {
            crc = _0xIC.crc32Table[(crc ^ data.charCodeAt(i)) & 0xFF] ^ (crc >>> 8)
        }
        return (crc ^ 0xFFFFFFFF) >>> 0
    }

    const computeMD5 = function(data) {
        function safeAdd(x, y) {
            const lsw = (x & 0xFFFF) + (y & 0xFFFF)
            const msw = (x >> 16) + (y >> 16) + (lsw >> 16)
            return (msw << 16) | (lsw & 0xFFFF)
        }
        
        function bitRotateLeft(num, cnt) {
            return (num << cnt) | (num >>> (32 - cnt))
        }
        
        function md5ff(a, b, c, d, x, s, ac) {
            a = safeAdd(a, safeAdd(safeAdd((b & c) | ((~b) & d), x), ac))
            return safeAdd(bitRotateLeft(a, s), b)
        }
        
        function md5gg(a, b, c, d, x, s, ac) {
            a = safeAdd(a, safeAdd(safeAdd((b & d) | (c & (~d)), x), ac))
            return safeAdd(bitRotateLeft(a, s), b)
        }
        
        function md5hh(a, b, c, d, x, s, ac) {
            a = safeAdd(a, safeAdd(safeAdd(c ^ (b | (~d)), x), ac))
            return safeAdd(bitRotateLeft(a, s), b)
        }
        
        function md5ii(a, b, c, d, x, s, ac) {
            a = safeAdd(a, safeAdd(safeAdd(c ^ (b | (~d)), x), ac))
            return safeAdd(bitRotateLeft(a, s), b)
        }
        
        function md5blks(s) {
            const nblk = ((s.length + 8) >> 6) + 1
            const blks = new Array(nblk * 16)
            for (let i = 0; i < nblk * 16; i++) blks[i] = 0
            for (let i = 0; i < s.length; i++) blks[i >> 2] |= s.charCodeAt(i) << ((i % 4) * 8)
            blks[i >> 2] |= 0x80 << ((i % 4) * 8)
            blks[nblk * 16 - 2] = s.length * 8
            return blks
        }
        
        const x = md5blks(data)
        let a = 1732584193, b = -271733879, c = -1732584194, d = 271733878
        
        for (let i = 0; i < x.length; i += 16) {
            const olda = a, oldb = b, oldc = c, oldd = d
            
            a = md5ff(a, b, c, d, x[i], 7, -680876936)
            d = md5ff(d, a, b, c, x[i + 1], 12, -389564586)
            c = md5ff(c, d, a, b, x[i + 2], 17, 606105819)
            b = md5ff(b, c, d, a, x[i + 3], 22, -1044525330)
            a = md5ff(a, b, c, d, x[i + 4], 7, -176418897)
            d = md5ff(d, a, b, c, x[i + 5], 12, 1200080426)
            c = md5ff(c, d, a, b, x[i + 6], 17, -1473231341)
            b = md5ff(b, c, d, a, x[i + 7], 22, -45705983)
            a = md5ff(a, b, c, d, x[i + 8], 7, 1770035416)
            d = md5ff(d, a, b, c, x[i + 9], 12, -1958414417)
            c = md5ff(c, d, a, b, x[i + 10], 17, -42063)
            b = md5ff(b, c, d, a, x[i + 11], 22, -1990404162)
            a = md5ff(a, b, c, d, x[i + 12], 7, 1804603682)
            d = md5ff(d, a, b, c, x[i + 13], 12, -40341101)
            c = md5ff(c, d, a, b, x[i + 14], 17, -1502002290)
            b = md5ff(b, c, d, a, x[i + 15], 22, 1236535329)
            
            a = md5gg(a, b, c, d, x[i + 1], 5, -165796510)
            d = md5gg(d, a, b, c, x[i + 6], 9, -1069501632)
            c = md5gg(c, d, a, b, x[i + 11], 14, 643717713)
            b = md5gg(b, c, d, a, x[i], 20, -373897302)
            a = md5gg(a, b, c, d, x[i + 5], 5, -701558691)
            d = md5gg(d, a, b, c, x[i + 10], 9, 38016083)
            c = md5gg(c, d, a, b, x[i + 15], 14, -660478335)
            b = md5gg(b, c, d, a, x[i + 4], 20, -405537848)
            a = md5gg(a, b, c, d, x[i + 9], 5, 568446438)
            d = md5gg(d, a, b, c, x[i + 14], 9, -1019803690)
            c = md5gg(c, d, a, b, x[i + 3], 14, -187363961)
            b = md5gg(b, c, d, a, x[i + 8], 20, 1163531501)
            a = md5gg(a, b, c, d, x[i + 13], 5, -1444681467)
            d = md5gg(d, a, b, c, x[i + 2], 9, -51403784)
            c = md5gg(c, d, a, b, x[i + 7], 14, 1735328473)
            b = md5gg(b, c, d, a, x[i + 12], 20, -1926607734)
            
            a = md5hh(a, b, c, d, x[i + 5], 4, -378558)
            d = md5hh(d, a, b, c, x[i + 8], 11, -2022574463)
            c = md5hh(c, d, a, b, x[i + 11], 16, 1839030562)
            b = md5hh(b, c, d, a, x[i + 14], 23, -35309556)
            a = md5hh(a, b, c, d, x[i + 1], 4, -1530992060)
            d = md5hh(d, a, b, c, x[i + 4], 11, 1272893353)
            c = md5hh(c, d, a, b, x[i + 7], 16, -155497632)
            b = md5hh(b, c, d, a, x[i + 10], 23, -1094730640)
            a = md5hh(a, b, c, d, x[i + 13], 4, 681279174)
            d = md5hh(d, a, b, c, x[i + 0], 11, -358537222)
            c = md5hh(c, d, a, b, x[i + 3], 16, -722521979)
            b = md5hh(b, c, d, a, x[i + 6], 23, 76029189)
            a = md5hh(a, b, c, d, x[i + 9], 4, -640364487)
            d = md5hh(d, a, b, c, x[i + 12], 11, -421815835)
            c = md5hh(c, d, a, b, x[i + 15], 16, 530742520)
            b = md5hh(b, c, d, a, x[i + 2], 23, -995338651)
            
            a = md5ii(a, b, c, d, x[i], 6, -198630844)
            d = md5ii(d, a, b, c, x[i + 7], 10, 1126891415)
            c = md5ii(c, d, a, b, x[i + 14], 15, -1416354905)
            b = md5ii(b, c, d, a, x[i + 5], 21, -57434055)
            a = md5ii(a, b, c, d, x[i + 12], 6, 1700485571)
            d = md5ii(d, a, b, c, x[i + 3], 10, -1894986606)
            c = md5ii(c, d, a, b, x[i + 10], 15, -1051523)
            b = md5ii(b, c, d, a, x[i + 1], 21, -2054922799)
            a = md5ii(a, b, c, d, x[i + 8], 6, 1873313359)
            d = md5ii(d, a, b, c, x[i + 15], 10, -30611744)
            c = md5ii(c, d, a, b, x[i + 6], 15, -1560198380)
            b = md5ii(b, c, d, a, x[i + 13], 21, 1309151649)
            a = md5ii(a, b, c, d, x[i + 4], 6, -145523070)
            d = md5ii(d, a, b, c, x[i + 11], 10, -1120210379)
            c = md5ii(c, d, a, b, x[i + 2], 15, 718787259)
            b = md5ii(b, c, d, a, x[i + 9], 21, -343485551)
            
            a = safeAdd(a, olda)
            b = safeAdd(b, oldb)
            c = safeAdd(c, oldc)
            d = safeAdd(d, oldd)
        }
        
        return (a < 0 ? a + 0x100000000 : a).toString(16).padStart(8, '0') +
               (b < 0 ? b + 0x100000000 : b).toString(16).padStart(8, '0') +
               (c < 0 ? c + 0x100000000 : c).toString(16).padStart(8, '0') +
               (d < 0 ? d + 0x100000000 : d).toString(16).padStart(8, '0')
    }

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

    const computeSHA256 = async function(data) {
        if (typeof crypto !== 'undefined' && crypto.subtle) {
            const encoder = new TextEncoder();
            const dataBuffer = encoder.encode(data);
            const hashBuffer = await crypto.subtle.digest('SHA-256', dataBuffer);
            const hashArray = Array.from(new Uint8Array(hashBuffer));
            return hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
        } else {
            return computeMD5(data)
        }
    }

    const computeMultipleHashes = async function(data) {
        const hash = generateHash(data)
        const crc32 = computeCRC32(data).toString(16).padStart(8, '0')
        const sha256 = await computeSHA256(data)
        const md5 = computeMD5(data)
        
        return {
            simple: hash,
            crc32: crc32,
            sha256: sha256,
            md5: md5,
            combined: crc32 + sha256.slice(0, 8)
        }
    }

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
            timestamp: _0xIC.timestamp,
            version: _0xIC.version
        };
    };

    const verifyManual = function(code) {
        const currentHash = generateHash(code);
        return currentHash === _0xIC.hashes.sha256;
    };

    const verifyCRC32 = function(code) {
        const currentCRC = computeCRC32(code);
        return currentCRC === parseInt(_0xIC.hashes.crc32, 16);
    };

    const verifyMultiple = async function(code) {
        const hashes = await computeMultipleHashes(code);
        const results = {
            simple: hashes.simple === _0xIC.hashes.simple,
            crc32: hashes.crc32 === _0xIC.hashes.crc32,
            sha256: hashes.sha256 === _0xIC.hashes.sha256,
            md5: hashes.md5 === _0xIC.hashes.md5
        };
        results.all = results.simple && results.crc32 && results.sha256 && results.md5;
        return results;
    };

    const initAdvanced = async function(hashes, config) {
        _0xIC.hashes = hashes || {};
        _0xIC.timestamp = new Date().toISOString();

        if (config) {
            _0xIC.checkInterval = config.checkInterval || 15000;
            _0xIC.maxChecks = config.maxChecks || 50;
        }

        if (!_0xIC.hashes.sha256 && !_0xIC.hashes.crc32) {
            console.warn('IntegrityChecker: No hashes provided');
        }

        start();
    };

    return {
        init: init,
        initAdvanced: initAdvanced,
        start: start,
        stop: stop,
        verify: verify,
        getStatus: getStatus,
        verifyManual: verifyManual,
        verifyCRC32: verifyCRC32,
        verifyMultiple: verifyMultiple,
        computeHash: generateHash,
        computeCRC32: computeCRC32,
        computeMD5: computeMD5,
        computeSHA256: computeSHA256,
        computeMultipleHashes: computeMultipleHashes
    };
})();

if (typeof module !== 'undefined' && module.exports) {
    module.exports = IntegrityChecker;
}
