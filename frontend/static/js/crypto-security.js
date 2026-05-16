(function(_0x2e4f, _0x1a9c) {
    'use strict';
    var _0x3d8e = {
        'vKxZm': function(_0x5c7d, _0x3b1a) { return _0x5c7d === _0x3b1a; },
        'wJnYz': function(_0x4e2b, _0x2d9f) { return _0x4e2b + _0x2d9f; },
        'bRtLc': function(_0x7f3a) { return _0x7f3a(); }
    };
    var _0x5b6d = _0x2e4f[_0x1a9c(0x0)];
    if (!_0x5b6d) {
        var _0x8c1a = _0x1a9c;
        var _0x2d7f = {
            'OaQmW': _0x8c1a(0x1),
            'pHgNc': _0x8c1a(0x2),
            'jLxRv': _0x8c1a(0x3),
            'dKfXs': _0x8c1a(0x4),
            'uNwYt': _0x8c1a(0x5),
            'tBqZe': _0x8c1a(0x6),
            'vRrFc': _0x8c1a(0x7),
            'sXhJk': _0x8c1a(0x8),
            'yWqLp': _0x8c1a(0x9),
            'kNmTv': _0x8c1a(0xa)
        };
        var _0x4f8c = {
            'XyZaB': function(_0x6e3d, _0x5a1f) {
                var _0x9d2e = _0x1a9c;
                return _0x3d8e[_0x9d2e(0x0)](_0x6e3d, _0x5a1f);
            },
            'WcDeF': function(_0x7b4a, _0x6c8f) {
                var _0x8f3b = _0x1a9c;
                return _0x3d8e[_0x8f3b(0x1)](_0x7b4a, _0x6c8f);
            },
            'GhIjK': function(_0x3e5d) {
                var _0x7a1c = _0x1a9c;
                return _0x3d8e[_0x7a1c(0x2)](_0x3e5d);
            }
        };
        var _0x6b9e = window[_0x2d7f['OaQmW']] || {};
        var _0x1c3d = window[_0x2d7f['pHgNc']] || {};
        var _0x7e5f = {};
        _0x7e5f[_0x2d7f['jLxRv']] = _0x6b9e;
        _0x7e5f[_0x2d7f['dKfXs']] = _0x1c3d;
        _0x7e5f[_0x2d7f['uNwYt']] = _0x4f8c['XyZaB'];
        _0x7e5f[_0x2d7f['tBqZe']] = _0x4f8c['WcDeF'];
        _0x7e5f[_0x2d7f['vRrFc']] = _0x4f8c['GhIjK'];
        _0x7e5f[_0x2d7f['sXhJk']] = _0x2d7f['yWqLp'];
        _0x7e5f[_0x2d7f['kNmTv']] = _0x2d7f['vRrFc'];
        _0x2e4f[_0x1a9c(0xb)] = _0x7e5f;
    }
    return _0x2e4f[_0x1a9c(0xb)];
})(window, function(_0x3f7a, _0x5d2b, _0x8e4c, _0x1b6d, _0x9a3e, _0x4c8f, _0x7e2a, _0x2d5b, _0x6a9c, _0x3b1d, _0x8f6e, _0x4d2c) {
    var _0x2e8f = {
        'a': 'crypto',
        'b': 'subtle',
        'c': 'XyZaB',
        'd': 'WcDeF',
        'e': 'GhIjK',
        'f': 'csp',
        'g': 'xor',
        'h': 'hash',
        'i': 'monitor'
    };
    return _0x2e8f['a'] + _0x2e8f['b'] + _0x2e8f['c'];
});

(function(_0x9f3d) {
    'use strict';
    var _0x4a7e = {
        'encrypt': _0x9f3d[_0x9f3d(0x0)],
        'decrypt': _0x9f3d[_0x9f3d(0x1)],
        'generateKey': _0x9f3d[_0x9f3d(0x2)],
        'generateRSAKeyPair': _0x9f3d[_0x9f3d(0x3)],
        'deriveKey': _0x9f3d[_0x9f3d(0x4)],
        'hash': _0x9f3d[_0x9f3d(0x5)],
        'sign': _0x9f3d[_0x9f3d(0x6)],
        'verify': _0x9f3d[_0x9f3d(0x7)]
    };
    if (typeof module !== 'undefined' && module['exports']) {
        module['exports'] = _0x4a7e;
    } else {
        window['CryptoAPI'] = _0x4a7e;
    }
})(function(_0x5c8d, _0x3a1f, _0x7e4b, _0x2d9c, _0x6f8a, _0x1b5d, _0x8e3c, _0x4d7f) {
    'use strict';
    var _0x2f6e = {
        'a': 'encrypt',
        'b': 'decrypt',
        'c': 'generateKey',
        'd': 'generateRSAKeyPair',
        'e': 'deriveKey',
        'f': 'hash',
        'g': 'sign',
        'h': 'verify',
        'i': 'base64',
        'j': 'utf8',
        'k': 'hex'
    };
    var _0x9c3a = {
        'encodeBase64': function(_0x7b2d) {
            var _0x4e8f = _0x5c8d['call'](this, _0x7b2d);
            if (typeof btoa !== 'undefined') {
                return btoa(_0x4e8f);
            }
            var _0x1d5a = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
            var _0x6b9e = '';
            var _0x3c7f = _0x4e8f['length'];
            var _0x8a2d = 0;
            var _0x5f4b = _0x1d5a['charAt'](_0x6b9e);
            while (_0x8a2d < _0x3c7f) {
                var _0x2e9c = _0x4e8f['charCodeAt'](_0x8a2d++) & 0xff;
                if (_0x8a2d === _0x3c7f) {
                    _0x6b9e += _0x1d5a['charAt'](_0x2e9c >> 2);
                    _0x6b9e += _0x1d5a['charAt']((_0x2e9c & 0x3) << 4);
                    _0x6b9e += '==';
                    break;
                }
                var _0x7f4a = _0x4e8f['charCodeAt'](_0x8a2d++);
                if (_0x8a2d === _0x3c7f) {
                    _0x6b9e += _0x1d5a['charAt'](_0x2e9c >> 2);
                    _0x6b9e += _0x1d5a['charAt'](((_0x2e9c & 0x3) << 4) | ((_0x7f4a & 0xf0) >> 4));
                    _0x6b9e += _0x1d5a['charAt']((_0x7f4a & 0xf) << 2);
                    _0x6b9e += '=';
                    break;
                }
                var _0x3b8f = _0x4e8f['charCodeAt'](_0x8a2d++);
                _0x6b9e += _0x1d5a['charAt'](_0x2e9c >> 2);
                _0x6b9e += _0x1d5a['charAt'](((_0x2e9c & 0x3) << 4) | ((_0x7f4a & 0xf0) >> 4));
                _0x6b9e += _0x1d5a['charAt'](((_0x7f4a & 0xf) << 2) | ((_0x3b8f & 0xc0) >> 6));
                _0x6b9e += _0x1d5a['charAt'](_0x3b8f & 0x3f);
            }
            return _0x6b9e;
        },
        'decodeBase64': function(_0x4d7a) {
            var _0x1f3c = _0x3a1f['call'](this, _0x4d7a);
            if (typeof atob !== 'undefined') {
                return atob(_0x1f3c);
            }
            var _0x8b5e = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
            var _0x6e2d = '';
            var _0x3a9f = _0x1f3c['length'];
            var _0x7c4b = 0;
            var _0x2d8a = [];
            var _0x5f3e = /[^A-Za-z0-9+\/=]/g;
            if (_0x5f3e['exec'](_0x1f3c)) {
                throw new Error('Invalid character in Base64 string');
            }
            while (_0x7c4b < _0x3a9f) {
                var _0x4b9c = _0x8b5e['indexOf'](_0x1f3c['charAt'](_0x7c4b++));
                var _0x9e3f = _0x8b5e['indexOf'](_0x1f3c['charAt'](_0x7c4b++));
                var _0x1d5b = _0x8b5e['indexOf'](_0x1f3c['charAt'](_0x7c4b++));
                var _0x6a8d = _0x8b5e['indexOf'](_0x1f3c['charAt'](_0x7c4b++));
                if (_0x4b9c === -1 || _0x9e3f === -1 || _0x1d5b === -1 || _0x6a8d === -1) {
                    break;
                }
                _0x2d8a[_0x2d8a['length']] = (_0x4b9c << 2) | (_0x9e3f >> 4);
                if (_0x1d5b !== 64) {
                    _0x2d8a[_0x2d8a['length']] = (_0x9e3f & 0xf) << 4 | (_0x1d5b >> 2);
                }
                if (_0x6a8d !== 64) {
                    _0x2d8a[_0x2d8a['length']] = (_0x1d5b & 0x3) << 6 | _0x6a8d;
                }
            }
            for (var _0x3c7d = 0; _0x3c7d < _0x2d8a['length']; _0x3c7d++) {
                _0x6e2d += String['fromCharCode'](_0x2d8a[_0x3c7d]);
            }
            return _0x6e2d;
        },
        'xorEncrypt': function(_0x8d4f, _0x2b6e) {
            var _0x7a3c = '';
            for (var _0x4e9b = 0; _0x4e9b < _0x8d4f['length']; _0x4e9b++) {
                _0x7a3c += String['fromCharCode'](_0x8d4f['charCodeAt'](_0x4e9b) ^ _0x2b6e['charCodeAt'](_0x4e9b % _0x2b6e['length']));
            }
            return _0x7a3c;
        },
        'sha256': function(_0x5d8a) {
            var _0x3f7e = [0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19];
            var _0x2d4a = [0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5, 0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174, 0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da, 0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967, 0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85, 0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070, 0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3, 0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2];
            var _0x6c9d = _0x5d8a['split']('')['map'](function(_0x3e8b) { return _0x3e8b['charCodeAt'](0); });
            var _0x1a5f = _0x6c9d['length'] * 8;
            _0x6c9d[_0x6c9d['length']] = 0x80;
            while ((_0x6c9d['length'] % 64) !== 56) {
                _0x6c9d[_0x6c9d['length']] = 0;
            }
            var _0x8f3a = new DataView(new ArrayBuffer(8));
            _0x8f3a['setUint32'](0, Math['floor'](_0x1a5f / 0x100000000), false);
            _0x8f3a['setUint32'](4, _0x1a5f & 0xffffffff, false);
            for (var _0x7b4d = 0; _0x7b4d < 8; _0x7b4d++) {
                _0x6c9d[_0x6c9d['length']] = _0x8f3a['getUint8'](_0x7b4d);
            }
            var _0x4e2c = new Array(64);
            for (var _0x2a8f = 0; _0x2a8f < _0x6c9d['length'] / 64; _0x2a8f++) {
                var _0x9c5e = new Array(64);
                for (var _0x3b7a = 0; _0x3b7a < 16; _0x3b7a++) {
                    _0x9c5e[_0x3b7a] = (_0x6c9d[_0x2a8f * 64 + _0x3b7a * 4] << 24) | (_0x6c9d[_0x2a8f * 64 + _0x3b7a * 4 + 1] << 16) | (_0x6c9d[_0x2a8f * 64 + _0x3b7a * 4 + 2] << 8) | (_0x6c9d[_0x2a8f * 64 + _0x3b7a * 4 + 3]);
                }
                for (var _0x5f3b = 16; _0x5f3b < 64; _0x5f3b++) {
                    var _0x1d9e = _0x9c5e[_0x5f3b - 15];
                    var _0x7e4d = _0x9c5e[_0x5f3b - 2];
                    var _0x3a8f = ((_0x1d9e >>> 7) | (_0x1d9e << 25)) ^ ((_0x1d9e >>> 18) | (_0x1d9e << 14)) ^ (_0x1d9e >>> 3);
                    var _0x6b3c = ((_0x7e4d >>> 17) | (_0x7e4d << 15)) ^ ((_0x7e4d >>> 19) | (_0x7e4d << 13)) ^ (_0x7e4d >>> 10);
                    _0x9c5e[_0x5f3b] = ((_0x9c5e[_0x5f3b - 16] + _0x3a8f) >>> 0) + ((_0x9c5e[_0x5f3b - 7] + _0x6b3c) >>> 0);
                }
                var _0x2e5a = _0x3f7e[0];
                var _0x8c3f = _0x3f7e[1];
                var _0x4f9b = _0x3f7e[2];
                var _0x1a6d = _0x3f7e[3];
                var _0x9d2e = _0x3f7e[4];
                var _0x6c8a = _0x3f7e[5];
                var _0x3b4f = _0x3f7e[6];
                var _0x7e2b = _0x3f7e[7];
                for (var _0x5a1c = 0; _0x5a1c < 64; _0x5a1c++) {
                    var _0x2d9f = ((_0x9d2e >>> 6) | (_0x9d2e << 26)) ^ ((_0x9d2e >>> 11) | (_0x9d2e << 21)) ^ ((_0x9d2e >>> 25) | (_0x9d2e << 7));
                    var _0x8b4e = ((_0x2e5a >>> 2) | (_0x2e5a << 30)) ^ ((_0x2e5a >>> 13) | (_0x2e5a << 19)) ^ ((_0x2e5a >>> 22) | (_0x2e5a << 10));
                    var _0x4c8d = (_0x2e5a & _0x8c3f) ^ (_0x2e5a & _0x4f9b) ^ (_0x8c3f & _0x4f9b);
                    var _0x1f5e = (_0x2e5a >> 2) ^ (_0x2e5a >> 13) ^ (_0x2e5a >> 22);
                    var _0x9a3f = (_0x9d2e & _0x6c8a) ^ (~_0x9d2e & _0x3b4f);
                    var _0x7b5d = (_0x7e2b + _0x2d9f + _0x9a3f + _0x2d4a[_0x5a1c] + _0x9c5e[_0x5a1c]) >>> 0;
                    var _0x3e8c = (_0x8b4e + _0x4c8d) >>> 0;
                    _0x7e2b = _0x3b4f;
                    _0x3b4f = _0x6c8a;
                    _0x6c8a = _0x9d2e;
                    _0x9d2e = (_0x1a6d + _0x7b5d) >>> 0;
                    _0x1a6d = _0x4f9b;
                    _0x4f9b = _0x8c3f;
                    _0x8c3f = _0x2e5a;
                    _0x2e5a = (_0x7b5d + _0x3e8c) >>> 0;
                }
                _0x3f7e[0] = (_0x3f7e[0] + _0x2e5a) >>> 0;
                _0x3f7e[1] = (_0x3f7e[1] + _0x8c3f) >>> 0;
                _0x3f7e[2] = (_0x3f7e[2] + _0x4f9b) >>> 0;
                _0x3f7e[3] = (_0x3f7e[3] + _0x1a6d) >>> 0;
                _0x3f7e[4] = (_0x3f7e[4] + _0x9d2e) >>> 0;
                _0x3f7e[5] = (_0x3f7e[5] + _0x6c8a) >>> 0;
                _0x3f7e[6] = (_0x3f7e[6] + _0x3b4f) >>> 0;
                _0x3f7e[7] = (_0x3f7e[7] + _0x7e2b) >>> 0;
            }
            var _0x6d4f = '';
            for (var _0x4b8e = 0; _0x4b8e < 8; _0x4b8e++) {
                _0x6d4f += ('00000000' + _0x3f7e[_0x4b8e]['toString'](16))['slice'](-8);
            }
            return _0x6d4f;
        },
        'md5': function(_0x7d9a) {
            function _0x2f8e(_0x3d7c, _0x5a9b) {
                return (_0x3d7c << _0x5a9b) | (_0x3d7c >>> (32 - _0x5a9b));
            }
            function _0x9c4f(_0x4e3a, _0x6b8d, _0x1d5c, _0x8a4e, _0x3b7f, _0x2d9a, _0x5f3d) {
                return _0x2f8e((_0x4e3a + _0x9c4f(_0x6b8d, _0x1d5c, _0x8a4e, _0x3b7f, _0x2d9a, _0x5f3d)) >>> 0, _0x2d9a) + _0x3b7f;
            }
            function _0x6e3b(_0x8f5c, _0x4d9e, _0x1a6f, _0x3b8d, _0x6c9a, _0x2d8f, _0x5a1e) {
                return _0x9c4f((_0x4d9e & _0x1a6f) | ((~_0x4d9e) & _0x3b8d), _0x8f5c, _0x4d9e, _0x6c9a, _0x2d8f, _0x5a1e);
            }
            function _0x1d5a(_0x3e9c, _0x8b4f, _0x4d7e, _0x1a5d, _0x6e8b, _0x2c9f, _0x5f3a) {
                return _0x9c4f((_0x8b4f & _0x4d7e) | (_0x3e9c & (~_0x4d7e)), _0x3e9c, _0x8b4f, _0x6e8b, _0x2c9f, _0x5f3a);
            }
            function _0x8e3f(_0x7b5c, _0x3d8e, _0x4a9f, _0x6c7d, _0x1d8a, _0x2b9e, _0x5f4c) {
                return _0x9c4f(_0x3d8e ^ _0x4a9f ^ _0x6c7d, _0x7b5c, _0x3d8e, _0x1d8a, _0x2b9e, _0x5f4c);
            }
            function _0x4c8a(_0x6e9d, _0x3b7f, _0x8c4e, _0x1d6c, _0x4a8b, _0x2e9f, _0x5d3a) {
                return _0x9c4f(_0x3b7f ^ (_0x6e9d | (~_0x8c4e)), _0x6e9d, _0x3b7f, _0x4a8b, _0x2e9f, _0x5d3a);
            }
            var _0x2d7f = [7, 12, 17, 22, 7, 12, 17, 22, 7, 12, 17, 22, 7, 12, 17, 22, 5, 9, 14, 20, 5, 9, 14, 20, 5, 9, 14, 20, 5, 9, 14, 20, 4, 11, 16, 23, 4, 11, 16, 23, 4, 11, 16, 23, 4, 11, 16, 23, 6, 10, 15, 21, 6, 10, 15, 21, 6, 10, 15, 21, 6, 10, 15, 21];
            var _0x9a3e = [0xd76aa478, 0xe8c7b756, 0x242070db, 0xc1bdceee, 0xf57c0faf, 0x4787c62a, 0xa8304613, 0xfd469501, 0x698098d8, 0x8b44f7af, 0xffff5bb1, 0x895cd7be, 0x6b901122, 0xfd987193, 0xa679438e, 0x49b40821, 0xf61e2562, 0xc040b340, 0x265e5a51, 0xe9b6c7aa, 0xd62f105d, 0x02441453, 0xd8a1e681, 0xe7d3fbc8, 0x21e1cde6, 0xc33707d6, 0xf4d50d87, 0x455a14ed, 0xa9e3e905, 0xfcefa3f8, 0x676f02d9, 0x8d2a4c8a, 0xfffa3942, 0x8771f681, 0x6d9d6122, 0xfde5380c, 0xa4beea44, 0x4bdecfa9, 0xf6bb4b60, 0xbebfbc70, 0x289b7ec6, 0xeaa127fa, 0xd4ef3085, 0x04881d05, 0xd9d4d039, 0xe6db99e5, 0x1fa27cf8, 0xc4ac5665, 0xf4292244, 0x432aff97, 0xab9423a7, 0xfc93a039, 0x655b59c3, 0x8f0ccc92, 0xffeff47d, 0x85845dd1, 0x6fa87e4f, 0xfe2ce6e0, 0xa3014314, 0x4e0811a1, 0xf7537e82, 0xbd3af235, 0x2ad7d2bb, 0xeb86d391];
            var _0x4d7c = 1732584193;
            var _0x8b5e = -271733879;
            var _0x1f3a = -1009589776;
            var _0x6c9f = 1985228508;
            var _0x3e8d = [0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476];
            var _0x5a1d = _0x7d9a['split']('')['map'](function(_0x4e2b) { return _0x4e2b['charCodeAt'](0); });
            var _0x7e4a = _0x5a1d['length'] * 8;
            _0x5a1d[_0x5a1d['length']] = 0x80;
            while ((_0x5a1d['length'] % 64) !== 56) {
                _0x5a1d[_0x5a1d['length']] = 0;
            }
            var _0x2d8b = new DataView(new ArrayBuffer(8));
            _0x2d8b['setUint32'](4, _0x7e4a, false);
            for (var _0x9b4e = 0; _0x9b4e < 8; _0x9b4e++) {
                _0x5a1d[_0x5a1d['length']] = _0x2d8b['getUint8'](_0x9b4e);
            }
            for (var _0x3c7d = 0; _0x3c7d < _0x5a1d['length'] / 64; _0x3c7d++) {
                var _0x6b8e = new Array(16);
                for (var _0x1d5f = 0; _0x1d5f < 16; _0x1d5f++) {
                    _0x6b8e[_0x1d5f] = (_0x5a1d[_0x3c7d * 64 + _0x1d5f * 4] | (_0x5a1d[_0x3c7d * 64 + _0x1d5f * 4 + 1] << 8) | (_0x5a1d[_0x3c7d * 64 + _0x1d5f * 4 + 2] << 16) | (_0x5a1d[_0x3c7d * 64 + _0x1d5f * 4 + 3] << 24)) >>> 0;
                }
                var _0x8f4a = _0x4d7c;
                var _0x2e9d = _0x8b5e;
                var _0x5f3e = _0x1f3a;
                var _0x4c8d = _0x6c9f;
                var _0x7b5a = _0x3e8d[0];
                var _0x1d6e = _0x3e8d[1];
                var _0x9c4b = _0x3e8d[2];
                var _0x3b7f = _0x3e8d[3];
                for (var _0x6e3c = 0; _0x6e3c < 64; _0x6e3c++) {
                    var _0x9a5d, _0x4e3f, _0x7d8e, _0x2b9c;
                    if (_0x6e3c < 16) {
                        _0x9a5d = _0x6e3b;
                        _0x4e3f = _0x8b5e;
                        _0x7d8e = _0x1f3a;
                        _0x2b9c = _0x6c9f;
                    } else if (_0x6e3c < 32) {
                        _0x9a5d = _0x1d5a;
                        _0x4e3f = _0x1f3a;
                        _0x7d8e = _0x6c9f;
                        _0x2b9c = _0x8b5e;
                    } else if (_0x6e3c < 48) {
                        _0x9a5d = _0x8e3f;
                        _0x4e3f = _0x8b5e;
                        _0x7d8e = _0x1f3a;
                        _0x2b9c = _0x6c9f;
                    } else {
                        _0x9a5d = _0x4c8a;
                        _0x4e3f = _0x6c9f;
                        _0x7d8e = _0x1f3a;
                        _0x2b9c = _0x8b5e;
                    }
                    var _0x5d9b = (_0x9a5d(_0x8f4a, _0x2e9d, _0x5f3e, _0x4c8d, _0x6b8e[_0x2d7f[_0x6e3c]], _0x2d7f[_0x6e3c], _0x9a3e[_0x6e3c]) >>> 0) + _0x2b9c;
                    _0x8f4a = _0x2b9c;
                    _0x2e9d = (_0x2f8e(_0x5f3e, 7) + _0x4c8d) >>> 0;
                    _0x5f3e = _0x4c8d;
                    _0x4c8d = _0x2e9d;
                }
                _0x4d7c = (_0x4d7c + _0x8f4a) >>> 0;
                _0x8b5e = (_0x8b5e + _0x2e9d) >>> 0;
                _0x1f3a = (_0x1f3a + _0x5f3e) >>> 0;
                _0x6c9f = (_0x6c9f + _0x4c8d) >>> 0;
                _0x3e8d[0] = (_0x3e8d[0] + _0x7b5a) >>> 0;
                _0x3e8d[1] = (_0x3e8d[1] + _0x1d6e) >>> 0;
                _0x3e8d[2] = (_0x3e8d[2] + _0x9c4b) >>> 0;
                _0x3e8d[3] = (_0x3e8d[3] + _0x3b7f) >>> 0;
            }
            var _0x6a9c = '';
            for (var _0x2d5e = 0; _0x2d5e < 4; _0x2d5e++) {
                _0x6a9c += ('00000000' + _0x3e8d[_0x2d5e]['toString'](16))['slice'](-8);
            }
            return _0x6a9c;
        },
        'utf8ToBytes': function(_0x3f7e) {
            return unescape(encodeURIComponent(_0x3f7e));
        },
        'bytesToUtf8': function(_0x2d9c) {
            return decodeURIComponent(escape(_0x2d9c));
        },
        'hexToBytes': function(_0x4a8f) {
            var _0x1d5b = [];
            for (var _0x3c8e = 0; _0x3c8e < _0x4a8f['length']; _0x3c8e += 2) {
                _0x1d5b['push'](parseInt(_0x4a8f['substr'](_0x3c8e, 2), 16));
            }
            return String['fromCharCode']['apply'](null, _0x1d5b);
        },
        'bytesToHex': function(_0x6f9a) {
            var _0x2e4b = [];
            for (var _0x4d8c = 0; _0x4d8c < _0x6f9a['length']; _0x4d8c++) {
                _0x2e4b['push'](('0' + _0x6f9a['charCodeAt'](_0x4d8c)['toString'](16))['slice'](-2));
            }
            return _0x2e4b['join']('');
        }
    };
    var _0x8c3a = {
        'encrypt': async function(_0x3d7a, _0x6b9e, _0x2f5d) {
            try {
                if (!window[_0x2f6e['a']] || !window[_0x2f6e['a']][_0x2f6e['b']]) {
                    return {
                        'success': false,
                        'error': 'Web Crypto API not available',
                        'fallback': _0x9c3a[_0x2f6e['g']](_0x3d7a, _0x6b9e)
                    };
                }
                var _0x4e8f = _0x9c3a['utf8ToBytes'](_0x3d7a);
                var _0x7a3c = window[_0x2f6e['a']][_0x2f6e['b']]['generateKey']({
                    'name': _0x2f5d || 'AES-GCM',
                    'length': 256
                }, true, ['encrypt', 'decrypt']);
                var _0x1d5a = new Uint8Array(12);
                window[_0x2f6e['a']]['getRandomValues'](_0x1d5a);
                var _0x9e3f = await window[_0x2f6e['a']][_0x2f6e['b']][_0x2f6e['a']]({
                    'name': _0x2f5d || 'AES-GCM',
                    'iv': _0x1d5a
                }, _0x7a3c, _0x4e8f);
                var _0x6a8d = new Uint8Array(_0x1d5a['length'] + _0x9e3f['byteLength']);
                _0x6a8d['set'](_0x1d5a, 0);
                _0x6a8d['set'](new Uint8Array(_0x9e3f), _0x1d5a['length']);
                return {
                    'success': true,
                    'data': _0x9c3a['bytesToHex'](String['fromCharCode']['apply'](null, _0x6a8d)),
                    'key': _0x7a3c
                };
            } catch (_0x3e8b) {
                return {
                    'success': false,
                    'error': _0x3e8b['message'],
                    'fallback': _0x9c3a[_0x2f6e['g']](_0x3d7a, _0x6b9e)
                };
            }
        },
        'decrypt': async function(_0x5d8a, _0x2b6e, _0x4d7f) {
            try {
                if (!window[_0x2f6e['a']] || !window[_0x2f6e['a']][_0x2f6e['b']]) {
                    return {
                        'success': false,
                        'error': 'Web Crypto API not available'
                    };
                }
                var _0x7a3c = _0x9c3a['hexToBytes'](_0x5d8a);
                var _0x1d5a = new Uint8Array(_0x7a3c['slice'](0, 12)['split']('')['map'](function(_0x3e8b) { return _0x3e8b['charCodeAt'](0); }));
                var _0x9e3f = _0x7a3c['slice'](12);
                var _0x1d5b = new Uint8Array(_0x9e3f['split']('')['map'](function(_0x3e8b) { return _0x3e8b['charCodeAt'](0); }));
                var _0x6a8d = await window[_0x2f6e['a']][_0x2f6e['b']]['decrypt']({
                    'name': _0x4d7f || 'AES-GCM',
                    'iv': _0x1d5a
                }, _0x2b6e, _0x1d5b);
                return {
                    'success': true,
                    'data': _0x9c3a['bytesToUtf8'](String['fromCharCode']['apply'](null, new Uint8Array(_0x6a8d)))
                };
            } catch (_0x3e8b) {
                return {
                    'success': false,
                    'error': _0x3e8b['message']
                };
            }
        },
        'generateKey': async function(_0x8d4f, _0x2d8a) {
            try {
                if (!window[_0x2f6e['a']] || !window[_0x2f6e['a']][_0x2f6e['b']]) {
                    throw new Error('Web Crypto API not available');
                }
                var _0x7b4d = await window[_0x2f6e['a']][_0x2f6e['b']][_0x2f6e['c']]({
                    'name': _0x2d8a || 'AES-GCM',
                    'length': 256
                }, true, ['encrypt', 'decrypt']);
                var _0x9c5e = await window[_0x2f6e['a']][_0x2f6e['b']]['exportKey']('raw', _0x7b4d);
                return {
                    'success': true,
                    'key': _0x7b4d,
                    'raw': _0x9c3a['bytesToHex'](String['fromCharCode']['apply'](null, new Uint8Array(_0x9c5e)))
                };
            } catch (_0x3e8b) {
                return {
                    'success': false,
                    'error': _0x3e8b['message']
                };
            }
        },
        'generateRSAKeyPair': async function(_0x3b7a, _0x5f3b) {
            try {
                if (!window[_0x2f6e['a']] || !window[_0x2f6e['a']][_0x2f6e['b']]) {
                    throw new Error('Web Crypto API not available');
                }
                var _0x1d9e = await window[_0x2f6e['a']][_0x2f6e['b']][_0x2f6e['d']]({
                    'name': _0x5f3b || 'RSA-OAEP',
                    'modulusLength': _0x3b7a || 2048,
                    'publicExponent': new Uint8Array([1, 0, 1]),
                    'hash': 'SHA-256'
                }, true, [_0x2f6e['a'], 'decrypt']);
                var _0x7e4d = await window[_0x2f6e['a']][_0x2f6e['b']]['exportKey']('spki', _0x1d9e['publicKey']);
                var _0x3a8f = await window[_0x2f6e['a']][_0x2f6e['b']]['exportKey']('pkcs8', _0x1d9e['privateKey']);
                return {
                    'success': true,
                    'publicKey': _0x9c3a['bytesToHex'](String['fromCharCode']['apply'](null, new Uint8Array(_0x7e4d))),
                    'privateKey': _0x9c3a['bytesToHex'](String['fromCharCode']['apply'](null, new Uint8Array(_0x3a8f)))
                };
            } catch (_0x3e8b) {
                return {
                    'success': false,
                    'error': _0x3e8b['message']
                };
            }
        },
        'encryptRSA': async function(_0x6b3c, _0x9c5e, _0x2a8f) {
            try {
                if (!window[_0x2f6e['a']] || !window[_0x2f6e['a']][_0x2f6e['b']]) {
                    return {
                        'success': false,
                        'error': 'Web Crypto API not available'
                    };
                }
                var _0x3b7a = await window[_0x2f6e['a']][_0x2f6e['b']]['importKey']('spki', _0x9c3a['hexToBytes'](_0x9c5e), {
                    'name': _0x2a8f || 'RSA-OAEP',
                    'hash': 'SHA-256'
                }, false, [_0x2f6e['a']]);
                var _0x7f4a = _0x9c3a['utf8ToBytes'](_0x6b3c);
                var _0x2e9c = await window[_0x2f6e['a']][_0x2f6e['b']][_0x2f6e['a']]({
                    'name': _0x2a8f || 'RSA-OAEP'
                }, _0x3b7a, _0x7f4a);
                return {
                    'success': true,
                    'data': _0x9c3a['bytesToHex'](String['fromCharCode']['apply'](null, new Uint8Array(_0x2e9c)))
                };
            } catch (_0x3e8b) {
                return {
                    'success': false,
                    'error': _0x3e8b['message']
                };
            }
        },
        'decryptRSA': async function(_0x8b4e, _0x9c5e, _0x3b7a) {
            try {
                if (!window[_0x2f6e['a']] || !window[_0x2f6e['a']][_0x2f6e['b']]) {
                    return {
                        'success': false,
                        'error': 'Web Crypto API not available'
                    };
                }
                var _0x1d5b = await window[_0x2f6e['a']][_0x2f6e['b']]['importKey']('pkcs8', _0x9c3a['hexToBytes'](_0x9c5e), {
                    'name': _0x3b7a || 'RSA-OAEP',
                    'hash': 'SHA-256'
                }, false, ['decrypt']);
                var _0x6a8d = _0x9c3a['hexToBytes'](_0x8b4e);
                var _0x4e9b = await window[_0x2f6e['a']][_0x2f6e['b']]['decrypt']({
                    'name': _0x3b7a || 'RSA-OAEP'
                }, _0x1d5b, _0x6a8d);
                return {
                    'success': true,
                    'data': _0x9c3a['bytesToUtf8'](String['fromCharCode']['apply'](null, new Uint8Array(_0x4e9b)))
                };
            } catch (_0x3e8b) {
                return {
                    'success': false,
                    'error': _0x3e8b['message']
                };
            }
        },
        'deriveKey': async function(_0x4f9b, _0x1a6d, _0x9d2e, _0x6c8a) {
            try {
                if (!window[_0x2f6e['a']] || !window[_0x2f6e['a']][_0x2f6e['b']]) {
                    throw new Error('Web Crypto API not available');
                }
                var _0x3b4f = await window[_0x2f6e['a']][_0x2f6e['b']]['importKey']('raw', _0x9c3a['hexToBytes'](_0x1a6d), 'PBKDF2', false, ['deriveBits', 'deriveKey']);
                var _0x7e2b = new Uint8Array(16);
                window[_0x2f6e['a']]['getRandomValues'](_0x7e2b);
                var _0x2d9f = await window[_0x2f6e['a']][_0x2f6e['b']][_0x2f6e['e']]({
                    'name': 'PBKDF2',
                    'salt': _0x7e2b,
                    'iterations': _0x9d2e || 100000,
                    'hash': 'SHA-256'
                }, _0x3b4f, {
                    'name': _0x6c8a || 'AES-GCM',
                    'length': 256
                }, true, [_0x2f6e['a'], 'decrypt']);
                var _0x8c3f = await window[_0x2f6e['a']][_0x2f6e['b']]['exportKey']('raw', _0x2d9f);
                return {
                    'success': true,
                    'key': _0x2d9f,
                    'raw': _0x9c3a['bytesToHex'](String['fromCharCode']['apply'](null, new Uint8Array(_0x8c3f))),
                    'salt': _0x9c3a['bytesToHex'](String['fromCharCode']['apply'](null, _0x7e2b))
                };
            } catch (_0x3e8b) {
                return {
                    'success': false,
                    'error': _0x3e8b['message']
                };
            }
        },
        'hash': function(_0x5a1c, _0x4e2c) {
            var _0x8c3f = _0x4e2c || 'sha256';
            if (_0x8c3f === 'sha256') {
                return _0x9c3a[_0x2f6e['f']](_0x5a1c);
            } else if (_0x8c3f === 'md5') {
                return _0x9c3a[_0x2f6e['h']](_0x5a1c);
            } else {
                return _0x9c3a['sha256'](_0x5a1c);
            }
        },
        'sign': async function(_0x3b7a, _0x1d5e, _0x4f9b) {
            try {
                if (!window[_0x2f6e['a']] || !window[_0x2f6e['a']][_0x2f6e['b']]) {
                    return {
                        'success': false,
                        'error': 'Web Crypto API not available'
                    };
                }
                var _0x8b4e = await window[_0x2f6e['a']][_0x2f6e['b']]['importKey']('pkcs8', _0x9c3a['hexToBytes'](_0x1d5e), {
                    'name': 'RSASSA-PKCS1-v1_5',
                    'hash': 'SHA-256'
                }, false, ['sign']);
                var _0x6a8d = await window[_0x2f6e['a']][_0x2f6e['b']][_0x2f6e['g']]({
                    'name': 'RSASSA-PKCS1-v1_5'
                }, _0x8b4e, new TextEncoder()['encode'](_0x3b7a));
                return {
                    'success': true,
                    'signature': _0x9c3a['bytesToHex'](String['fromCharCode']['apply'](null, new Uint8Array(_0x6a8d)))
                };
            } catch (_0x3e8b) {
                return {
                    'success': false,
                    'error': _0x3e8b['message']
                };
            }
        },
        'verify': async function(_0x8f5c, _0x4d9e, _0x1a6f, _0x3b8d) {
            try {
                if (!window[_0x2f6e['a']] || !window[_0x2f6e['a']][_0x2f6e['b']]) {
                    return {
                        'success': false,
                        'error': 'Web Crypto API not available'
                    };
                }
                var _0x6c9a = await window[_0x2f6e['a']][_0x2f6e['b']]['importKey']('spki', _0x9c3a['hexToBytes'](_0x1a6f), {
                    'name': 'RSASSA-PKCS1-v1_5',
                    'hash': 'SHA-256'
                }, false, ['verify']);
                var _0x2d8f = await window[_0x2f6e['a']][_0x2f6e['b']][_0x2f6e['h']]({
                    'name': 'RSASSA-PKCS1-v1_5'
                }, _0x6c9a, _0x9c3a['hexToBytes'](_0x4d9e), new TextEncoder()['encode'](_0x8f5c));
                return {
                    'success': true,
                    'valid': _0x2d8f
                };
            } catch (_0x3e8b) {
                return {
                    'success': false,
                    'error': _0x3e8b['message']
                };
            }
        },
        'encodeBase64': function(_0x7d9a) {
            return _0x9c3a['encodeBase64'](_0x7d9a);
        },
        'decodeBase64': function(_0x4a8f) {
            return _0x9c3a['decodeBase64'](_0x4a8f);
        },
        'xorEncrypt': function(_0x3f7e, _0x2d9c) {
            return _0x9c3a[_0x2f6e['g']](_0x3f7e, _0x2d9c);
        },
        'randomBytes': function(_0x6f8a) {
            var _0x1b5d = new Uint8Array(_0x6f8a || 32);
            if (window[_0x2f6e['a']]) {
                window[_0x2f6e['a']]['getRandomValues'](_0x1b5d);
            } else {
                for (var _0x8e3c = 0; _0x8e3c < _0x6f8a; _0x8e3c++) {
                    _0x1b5d[_0x8e3c] = Math['floor'](Math['random']() * 256);
                }
            }
            return _0x9c3a['bytesToHex'](String['fromCharCode']['apply'](null, _0x1b5d));
        }
    };
    return _0x8c3a;
});

(function(_0x4d2f) {
    'use strict';
    var _0x7e5a = {
        'obfuscate': _0x4d2f[_0x4d2f(0x0)],
        'deobfuscate': _0x4d2f[_0x4d2f(0x1)],
        'detectDebug': _0x4d2f[_0x4d2f(0x2)],
        'integrityCheck': _0x4d2f[_0x4d2f(0x3)],
        'protect': _0x4d2f[_0x4d2f(0x4)],
        'randomize': _0x4d2f[_0x4d2f(0x5)]
    };
    if (typeof module !== 'undefined' && module['exports']) {
        module['exports'] = _0x7e5a;
    } else {
        window['CodeObfuscator'] = _0x7e5a;
    }
})(function(_0x8a3e, _0x5d1f, _0x3e9b, _0x7c5d, _0x2a8f, _0x9b4c) {
    'use strict';
    var _0x4e7a = {
        'a': 'obfuscate',
        'b': 'deobfuscate',
        'c': 'detectDebug',
        'd': 'integrityCheck',
        'e': 'protect',
        'f': 'randomize',
        'g': 'variableNames',
        'h': 'stringEncoding',
        'i': 'codeFlow',
        'j': 'selfDefending',
        'k': 'deadCode'
    };
    var _0x6b3d = {};
    var _0x1d5e = 0;
    var _0x9c8f = ['_0x', '_0x', '_0x', '_0x', '_0x', '_0x', '_0x', '_0x', '_0x', '_0x'];
    var _0x4f6e = {
        'encodeStrings': function(_0x7b2a) {
            var _0x3e5c = arguments['length'] > 1 && arguments[1] !== undefined ? arguments[1] : true;
            var _0x5a9f = arguments['length'] > 2 && arguments[2] !== undefined ? arguments[2] : 2;
            var _0x2d7b = _0x7b2a;
            var _0x8f4d = /"([^"\\]|\\.)*"|'([^'\\]|\\.)*'|`([^`\\]|\\.)*'/g;
            var _0x4e9a = [];
            var _0x1d6b = _0x7b2a['match'](_0x8f4d) || [];
            for (var _0x3c8e = 0; _0x3c8e < _0x1d6b['length']; _0x3c8e++) {
                var _0x6f8c = _0x1d6b[_0x3c8e];
                var _0x9b5a = _0x6f8c['substring'](1, _0x6f8c['length'] - 1);
                var _0x2e9d = void 0;
                if (_0x5a9f === 1) {
                    _0x2e9d = _0x4f6e['toUnicode'](_0x9b5a);
                } else if (_0x5a9f === 2) {
                    _0x2e9d = _0x4f6e['toBase64'](_0x9b5a);
                } else if (_0x5a9f === 3) {
                    _0x2e9d = _0x4f6e['toHex'](_0x9b5a);
                } else {
                    _0x2e9d = _0x4f6e['toRC4'](_0x9b5a);
                }
                _0x2d7b = _0x2d7b['replace'](_0x6f8c, _0x2e9d);
            }
            return _0x2d7b;
        },
        'toUnicode': function(_0x3d8b) {
            var _0x7a4e = '';
            for (var _0x4e9f = 0; _0x4e9f < _0x3d8b['length']; _0x4e9f++) {
                _0x7a4e += '\\u' + ('0000' + _0x3d8b['charCodeAt'](_0x4e9f)['toString'](16))['slice'](-4);
            }
            return '"' + _0x7a4e + '"';
        },
        'toBase64': function(_0x5f3d) {
            var _0x1d5a = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
            var _0x6b9e = '';
            var _0x3c7f = _0x5f3d['length'];
            var _0x8a2d = 0;
            while (_0x8a2d < _0x3c7f) {
                var _0x2e9c = _0x5f3d['charCodeAt'](_0x8a2d++) & 0xff;
                if (_0x8a2d === _0x3c7f) {
                    _0x6b9e += _0x1d5a['charAt'](_0x2e9c >> 2);
                    _0x6b9e += _0x1d5a['charAt']((_0x2e9c & 0x3) << 4);
                    _0x6b9e += '==';
                    break;
                }
                var _0x7f4a = _0x5f3d['charCodeAt'](_0x8a2d++);
                if (_0x8a2d === _0x3c7f) {
                    _0x6b9e += _0x1d5a['charAt'](_0x2e9c >> 2);
                    _0x6b9e += _0x1d5a['charAt'](((_0x2e9c & 0x3) << 4) | ((_0x7f4a & 0xf0) >> 4));
                    _0x6b9e += _0x1d5a['charAt']((_0x7f4a & 0xf) << 2);
                    _0x6b9e += '=';
                    break;
                }
                var _0x3b8f = _0x5f3d['charCodeAt'](_0x8a2d++);
                _0x6b9e += _0x1d5a['charAt'](_0x2e9c >> 2);
                _0x6b9e += _0x1d5a['charAt'](((_0x2e9c & 0x3) << 4) | ((_0x7f4a & 0xf0) >> 4));
                _0x6b9e += _0x1d5a['charAt'](((_0x7f4a & 0xf) << 2) | ((_0x3b8f & 0xc0) >> 6));
                _0x6b9e += _0x1d5a['charAt'](_0x3b8f & 0x3f);
            }
            return 'atob("' + _0x6b9e + '")';
        },
        'toHex': function(_0x4d7a) {
            var _0x1f3c = '';
            for (var _0x8b5e = 0; _0x8b5e < _0x4d7a['length']; _0x8b5e++) {
                _0x1f3c += '\\x' + ('00' + _0x4d7a['charCodeAt'](_0x8b5e)['toString'](16))['slice'](-2);
            }
            return '"' + _0x1f3c + '"';
        },
        'toRC4': function(_0x6e2d) {
            var _0x3a9f = 'hjtpx_secure_key_2024';
            var _0x7c4b = [];
            var _0x2d8a = [];
            for (var _0x5f3e = 0; _0x5f3e < 256; _0x5f3e++) {
                _0x7c4b[_0x5f3e] = _0x5f3e;
                _0x2d8a[_0x5f3e] = _0x3a9f['charCodeAt'](_0x5f3e % _0x3a9f['length']);
            }
            var _0x4b9c = 0;
            for (var _0x9e3f = 0; _0x9e3f < 256; _0x9e3f++) {
                _0x4b9c = (_0x4b9c + _0x7c4b[_0x9e3f] + _0x2d8a[_0x9e3f]) % 256;
                var _0x1d5b = _0x7c4b[_0x9e3f];
                _0x7c4b[_0x9e3f] = _0x7c4b[_0x4b9c];
                _0x7c4b[_0x4b9c] = _0x1d5b;
            }
            var _0x6a8d = [];
            var _0x4e9b = 0;
            var _0x3c7d = 0;
            for (var _0x2d8b = 0; _0x2d8b < _0x6e2d['length']; _0x2d8b++) {
                _0x4e9b = (_0x4e9b + 1) % 256;
                _0x3c7d = (_0x3c7d + _0x7c4b[_0x4e9b]) % 256;
                var _0x2e4b = _0x7c4b[_0x4e9b];
                _0x7c4b[_0x4e9b] = _0x7c4b[_0x3c7d];
                _0x7c4b[_0x3c7d] = _0x2e4b;
                var _0x4d8c = _0x7c4b[(_0x7c4b[_0x4e9b] + _0x7c4b[_0x3c7d]) % 256];
                _0x6a8d['push'](_0x6e2d['charCodeAt'](_0x2d8b) ^ _0x4d8c);
            }
            var _0x6f9a = '';
            for (var _0x3b8e = 0; _0x3b8e < _0x6a8d['length']; _0x3b8e++) {
                _0x6f9a += ('0' + _0x6a8d[_0x3b8e]['toString'](16))['slice'](-2);
            }
            return 'rc4_decode("' + _0x6f9a + '")';
        },
        'generateVariableName': function() {
            var _0x9c3a = arguments['length'] > 0 && arguments[0] !== undefined ? arguments[0] : 6;
            var _0x2e8f = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ';
            var _0x4a7e = _0x9c8f[Math['floor'](Math['random']() * _0x9c8f['length'])];
            for (var _0x5c8d = 0; _0x5c8d < _0x9c3a; _0x5c8d++) {
                _0x4a7e += _0x2e8f['charAt'](Math['floor'](Math['random']() * _0x2e8f['length']));
            }
            return _0x4a7e;
        },
        'transformControlFlow': function(_0x3a1f) {
            var _0x7e4b = /\b(if|else|for|while|switch|case|try|catch|finally)\b/g;
            var _0x2d9c = 0;
            return _0x3a1f['replace'](_0x7e4b, function(_0x6f8a) {
                _0x2d9c++;
                return _0x6f8a;
            });
        },
        'addDeadCode': function(_0x1b5d) {
            var _0x8e3c = ['var _0x1234 = 0;', 'function _0xdead(){return 0;}'];
            var _0x4d7f = Math['floor'](Math['random']() * 2);
            return _0x1b5d + '\n' + _0x8e3c[_0x4d7f];
        },
        'addSelfDefending': function(_0x8e4c) {
            var _0x1b6d = '\n;(function(){var _0xcheck=function(){try{return !!eval("' + _0x8e4c['substring'](0, 50) + '");}catch(e){return true;}};if(!_0xcheck()){throw new Error("Code integrity check failed");}})();';
            return _0x8e4c + _0x1b6d;
        }
    };
    var _0x7a3f = {
        'obfuscate': function(_0x5d2b) {
            var _0x8e4c = arguments['length'] > 1 && arguments[1] !== undefined ? arguments[1] : {};
            var _0x1b6d = {
                'encodeStrings': true,
                'encodeStringsType': 2,
                'obfuscateVariables': true,
                'transformControlFlow': false,
                'addDeadCode': false,
                'addSelfDefending': false
            };
            Object['assign'](_0x1b6d, _0x8e4c);
            var _0x9a3e = _0x5d2b;
            if (_0x1b6d['encodeStrings']) {
                _0x9a3e = _0x4f6e['encodeStrings'](_0x9a3e, true, _0x1b6d['encodeStringsType']);
            }
            if (_0x1b6d['addDeadCode']) {
                _0x9a3e = _0x4f6e['addDeadCode'](_0x9a3e);
            }
            if (_0x1b6d['addSelfDefending']) {
                _0x9a3e = _0x4f6e['addSelfDefending'](_0x5d2b);
            }
            return _0x9a3e;
        },
        'deobfuscate': function(_0x4c8f) {
            return _0x4c8f;
        },
        'detectDebug': function() {
            var _0x7e2a = {
                'devTools': false,
                'debugger': false,
                'consoleOpen': false
            };
            var _0x2d5b = function _0x2d5b() {};
            _0x2d5b['toString'] = function() {
                _0x7e2a['devTools'] = true;
            };
            console['dir'](_0x2d5b);
            console['dir'](_0x2d5b);
            var _0x6a9c = window['outerWidth'] - window['innerWidth'];
            var _0x3b1d = window['outerHeight'] - window['innerHeight'];
            if (Math['abs'](_0x6a9c) > 200 || Math['abs'](_0x3b1d) > 200) {
                _0x7e2a['devTools'] = true;
            }
            var _0x8f6e = Date['now']();
            debugger;
            if (Date['now']() - _0x8f6e > 100) {
                _0x7e2a['debugger'] = true;
            }
            var _0x4d2c = window['console']['open'];
            if (_0x4d2c) {
                _0x7e2a['consoleOpen'] = _0x4d2c();
            }
            var _0x2e8f = setInterval(function() {
                Date['now']();
                debugger;
            }, 1000);
            setTimeout(function() {
                clearInterval(_0x2e8f);
            }, 5000);
            return _0x7e2a;
        },
        'integrityCheck': function(_0x6f8a) {
            var _0x1b5d = 0;
            for (var _0x8e3c = 0; _0x8e3c < _0x6f8a['length']; _0x8e3c++) {
                _0x1b5d = ((_0x1b5d << 5) - _0x1b5d + _0x6f8a['charCodeAt'](_0x8e3c)) | 0;
            }
            return _0x1b5d;
        },
        'protect': function() {
            var _0x4d7f = {
                'protected': true,
                'integrity': _0x7a3f['integrityCheck'](document['documentElement']['innerHTML']['substring'](0, 1000)),
                'timestamp': Date['now']()
            };
            var _0x3e8d = function() {
                var _0x5a1d = false;
                var _0x7e4a = setInterval(function() {
                    var _0x8b5e = window['outerWidth'] - window['innerWidth'];
                    var _0x1f3a = window['outerHeight'] - window['innerHeight'];
                    if (Math['abs'](_0x8b5e) > 160 || Math['abs'](_0x1f3a) > 160) {
                        if (!_0x5a1d) {
                            _0x5a1d = true;
                            if (typeof onDebugDetected === 'function') {
                                onDebugDetected();
                            }
                        }
                    }
                }, 1000);
                return function() {
                    clearInterval(_0x7e4a);
                };
            }();
            return {
                'startProtection': _0x3e8d,
                'info': _0x4d7f
            };
        },
        'randomize': function(_0x3b1d) {
            var _0x8f6e = arguments['length'] > 1 && arguments[1] !== undefined ? arguments[1] : 10;
            var _0x4d2c = [];
            var _0x2e8f = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
            for (var _0x3f7a = 0; _0x3f7a < _0x8f6e; _0x3f7a++) {
                _0x4d2c['push'](_0x2e8f['charAt'](Math['floor'](Math['random']() * _0x2e8f['length'])));
            }
            return _0x4d2c['join']('');
        }
    };
    return _0x7a3f;
});

(function(_0x5c7e) {
    'use strict';
    var _0x3b1a = {
        'encryptRequest': _0x5c7e[_0x5c7e(0x0)],
        'decryptResponse': _0x5c7e[_0x5c7e(0x1)],
        'signRequest': _0x5c7e[_0x5c7e(0x2)],
        'verifyResponse': _0x5c7e[_0x5c7e(0x3)],
        'encryptForm': _0x5c7e[_0x5c7e(0x4)],
        'encryptAjax': _0x5c7e[_0x5c7e(0x5)]
    };
    if (typeof module !== 'undefined' && module['exports']) {
        module['exports'] = _0x3b1a;
    } else {
        window['RequestEncryptor'] = _0x3b1a;
    }
})(function(_0x4e2b, _0x2d9f, _0x7f3a, _0x5c7d, _0x3b1a, _0x8c1a) {
    'use strict';
    var _0x2d7f = {
        'a': 'encryptRequest',
        'b': 'decryptResponse',
        'c': 'signRequest',
        'd': 'verifyResponse',
        'e': 'encryptForm',
        'f': 'encryptAjax',
        'g': 'CryptoAPI',
        'h': 'secretKey',
        'i': 'publicKey',
        'j': 'privateKey',
        'k': 'timestamp',
        'l': 'nonce',
        'm': 'signature',
        'n': 'iv',
        'o': 'salt'
    };
    var _0x6b9e = window[_0x2d7f['g']] || {};
    var _0x1c3d = {
        'secretKey': 'hjtpx_default_key_' + Date['now'](),
        'publicKey': null,
        'privateKey': null,
        'initialized': false
    };
    var _0x7e5f = {
        'init': async function(_0x9d2e) {
            var _0x4f8c = {
                'secretKey': _0x9d2e || 'hjtpx_secure_key_' + Math['random']()['toString'](36)['substring'](2)
            };
            var _0x6b9e = await _0x6b9e['generateRSAKeyPair'](2048, 'RSA-OAEP');
            if (_0x6b9e['success']) {
                _0x4f8c['publicKey'] = _0x6b9e['publicKey'];
                _0x4f8c['privateKey'] = _0x6b9e['privateKey'];
                _0x4f8c['initialized'] = true;
            }
            Object['assign'](_0x1c3d, _0x4f8c);
            return {
                'success': true,
                'publicKey': _0x4f8c['publicKey']
            };
        },
        'generateNonce': function() {
            var _0x8c1a = _0x6b9e['randomBytes'](16);
            return _0x8c1a;
        },
        'generateTimestamp': function() {
            return Date['now']();
        },
        'hashData': function(_0x3b1a) {
            var _0x8c1a = '';
            if (typeof _0x3b1a === 'object') {
                var _0x2d7f = Object['keys'](_0x3b1a)['sort']();
                for (var _0x6b9e = 0; _0x6b9e < _0x2d7f['length']; _0x6b9e++) {
                    var _0x1c3d = _0x2d7f[_0x6b9e];
                    _0x8c1a += _0x1c3d + '=' + JSON['stringify'](_0x3b1a[_0x1c3d]) + '&';
                }
            } else {
                _0x8c1a = String(_0x3b1a);
            }
            return _0x6b9e['hash'](_0x8c1a);
        },
        'encryptRequest': async function(_0x4e2b, _0x2d9f) {
            try {
                var _0x7f3a = _0x7e5f['generateNonce']();
                var _0x5c7d = _0x7e5f['generateTimestamp']();
                var _0x3b1a = _0x7e5f['hashData'](_0x4e2b);
                var _0x8c1a = _0x7e5f['hashData'](_0x3b1a + _0x5c7d + _0x7f3a);
                var _0x2d7f = _0x7e5f['hashData'](_0x1c3d['secretKey']);
                var _0x6b9e = await _0x6b9e['encrypt'](JSON['stringify'](_0x4e2b), _0x2d7f, 'AES-GCM');
                var _0x1c3d = null;
                if (_0x1c3d['publicKey'] && _0x6b9e['success']) {
                    _0x1c3d = await _0x6b9e['encryptRSA'](_0x6b9e['raw'], _0x1c3d['publicKey'], 'RSA-OAEP');
                }
                var _0x4f8c = {
                    'success': true,
                    'data': _0x6b9e['data'],
                    'encryptedKey': _0x1c3d ? _0x1c3d['data'] : null,
                    'nonce': _0x7f3a,
                    'timestamp': _0x5c7d,
                    'signature': _0x8c1a,
                    'hash': _0x3b1a
                };
                if (_0x2d9f) {
                    Object['assign'](_0x2d9f, _0x4f8c);
                }
                return _0x4f8c;
            } catch (_0x6b9e) {
                return {
                    'success': false,
                    'error': _0x6b9e['message']
                };
            }
        },
        'decryptResponse': async function(_0x3b1a, _0x8c1a) {
            try {
                if (!_0x3b1a || !_0x3b1a['data']) {
                    return {
                        'success': false,
                        'error': 'Invalid response data'
                    };
                }
                var _0x2d7f = _0x7e5f['hashData'](_0x1c3d['secretKey']);
                var _0x6b9e = await _0x6b9e['decrypt'](_0x3b1a['data'], _0x2d7f, 'AES-GCM');
                if (_0x6b9e['success']) {
                    var _0x1c3d = JSON['parse'](_0x6b9e['data']);
                    if (_0x3b1a['signature']) {
                        var _0x4f8c = _0x7e5f['hashData'](_0x6b9e['data'] + _0x3b1a['timestamp'] + _0x3b1a['nonce']);
                        if (_0x4f8c !== _0x3b1a['signature']) {
                            return {
                                'success': false,
                                'error': 'Signature verification failed'
                            };
                        }
                    }
                    return {
                        'success': true,
                        'data': _0x1c3d
                    };
                }
                return {
                    'success': false,
                    'error': 'Decryption failed'
                };
            } catch (_0x6b9e) {
                return {
                    'success': false,
                    'error': _0x6b9e['message']
                };
            }
        },
        'signRequest': function(_0x4e2b) {
            try {
                var _0x2d9f = _0x7e5f['generateNonce']();
                var _0x7f3a = _0x7e5f['generateTimestamp']();
                var _0x5c7d = _0x7e5f['hashData'](_0x4e2b);
                var _0x3b1a = _0x7e5f['hashData'](_0x5c7d + _0x7f3a + _0x2d9f + _0x1c3d['secretKey']);
                return {
                    'success': true,
                    'data': _0x4e2b,
                    'signature': _0x3b1a,
                    'nonce': _0x2d9f,
                    'timestamp': _0x7f3a,
                    'hash': _0x5c7d
                };
            } catch (_0x8c1a) {
                return {
                    'success': false,
                    'error': _0x8c1a['message']
                };
            }
        },
        'verifyResponse': function(_0x2d9f) {
            try {
                if (!_0x2d9f || !_0x2d9f['data'] || !_0x2d9f['signature']) {
                    return {
                        'success': false,
                        'valid': false,
                        'error': 'Invalid response format'
                    };
                }
                var _0x7f3a = _0x7e5f['hashData'](_0x2d9f['data']);
                var _0x5c7d = _0x7e5f['hashData'](_0x7f3a + _0x2d9f['timestamp'] + _0x2d9f['nonce'] + _0x1c3d['secretKey']);
                var _0x3b1a = _0x5c7d === _0x2d9f['signature'];
                if (!_0x3b1a) {
                    return {
                        'success': true,
                        'valid': false,
                        'error': 'Signature mismatch'
                    };
                }
                if (Date['now']() - _0x2d9f['timestamp'] > 300000) {
                    return {
                        'success': true,
                        'valid': false,
                        'error': 'Request expired'
                    };
                }
                return {
                    'success': true,
                    'valid': true
                };
            } catch (_0x8c1a) {
                return {
                    'success': false,
                    'valid': false,
                    'error': _0x8c1a['message']
                };
            }
        },
        'encryptForm': function(_0x2d9f) {
            var _0x7f3a = arguments['length'] > 1 && arguments[1] !== undefined ? arguments[1] : {};
            try {
                var _0x5c7d = document['getElementById'](_0x2d9f);
                if (!_0x5c7d) {
                    throw new Error('Form not found: ' + _0x2d9f);
                }
                var _0x3b1a = {};
                var _0x8c1a = _0x5c7d['querySelectorAll']('input, select, textarea');
                for (var _0x2d7f = 0; _0x2d7f < _0x8c1a['length']; _0x2d7f++) {
                    var _0x6b9e = _0x8c1a[_0x2d7f];
                    if (_0x6b9e['name'] && !_0x6b9e['disabled']) {
                        var _0x1c3d = _0x6b9e['value'] || '';
                        if (_0x6b9e['type'] === 'checkbox' || _0x6b9e['type'] === 'radio') {
                            if (_0x6b9e['checked']) {
                                _0x3b1a[_0x6b9e['name']] = _0x1c3d;
                            }
                        } else {
                            _0x3b1a[_0x6b9e['name']] = _0x1c3d;
                        }
                    }
                }
                var _0x4f8c = _0x7e5f['encryptRequest'](_0x3b1a, _0x7f3a);
                return _0x4f8c;
            } catch (_0x6b9e) {
                return {
                    'success': false,
                    'error': _0x6b9e['message']
                };
            }
        },
        'encryptAjax': function(_0x2d9f) {
            var _0x7f3a = arguments['length'] > 1 && arguments[1] !== undefined ? arguments[1] : {};
            var _0x5c7d = arguments['length'] > 2 && arguments[2] !== undefined ? arguments[2] : {};
            try {
                var _0x3b1a = _0x7e5f['encryptRequest'](_0x2d9f, _0x5c7d);
                if (!_0x3b1a['success']) {
                    return Promise['resolve'](_0x3b1a);
                }
                var _0x8c1a = {
                    'method': _0x5c7d['method'] || 'POST',
                    'headers': {
                        'Content-Type': 'application/json',
                        'X-Encrypted': '1',
                        'X-Timestamp': _0x3b1a['timestamp'],
                        'X-Nonce': _0x3b1a['nonce'],
                        'X-Signature': _0x3b1a['signature']
                    },
                    'body': JSON['stringify']({
                        'data': _0x3b1a['data'],
                        'hash': _0x3b1a['hash']
                    })
                };
                return fetch(_0x5c7d['url'] || '/api/encrypt', _0x8c1a)['then'](function(_0x6b9e) {
                    return _0x6b9e['json']();
                })['then'](function(_0x1c3d) {
                    return _0x7e5f['decryptResponse'](_0x1c3d);
                })['catch'](function(_0x4f8c) {
                    return {
                        'success': false,
                        'error': _0x4f8c['message']
                    };
                });
            } catch (_0x6b9e) {
                return Promise['resolve']({
                    'success': false,
                    'error': _0x6b9e['message']
                });
            }
        }
    };
    return _0x7e5f;
});

(function(_0x9a3e) {
    'use strict';
    var _0x4c8f = {
        'init': _0x9a3e[_0x9a3e(0x0)],
        'log': _0x9a3e[_0x9a3e(0x1)],
        'track': _0x9a3e[_0x9a3e(0x2)],
        'report': _0x9a3e[_0x9a3e(0x3)],
        'onError': _0x9a3e[_0x9a3e(0x4)],
        'onWarning': _0x9a3e[_0x9a3e(0x5)],
        'getStats': _0x9a3e[_0x9a3e(0x6)]
    };
    if (typeof module !== 'undefined' && module['exports']) {
        module['exports'] = _0x4c8f;
    } else {
        window['SecurityMonitor'] = _0x4c8f;
    }
})(function(_0x7e2a, _0x2d5b, _0x6a9c, _0x3b1d, _0x8f6e, _0x4d2c, _0x2e8f) {
    'use strict';
    var _0x4a7e = {
        'a': 'init',
        'b': 'log',
        'c': 'track',
        'd': 'report',
        'e': 'onError',
        'f': 'onWarning',
        'g': 'getStats',
        'h': 'events',
        'i': 'errors',
        'j': 'warnings',
        'k': 'performance',
        'l': 'security'
    };
    var _0x5c8d = {
        'events': [],
        'errors': [],
        'warnings': [],
        'performance': {},
        'security': {},
        'startTime': Date['now'](),
        'maxEvents': 1000,
        'enabled': false
    };
    var _0x3a1f = {
        'init': function(_0x8e4c) {
            var _0x1b6d = {
                'enabled': true,
                'reportUrl': '/api/security/report',
                'maxEvents': 1000,
                'captureErrors': true,
                'captureWarnings': true,
                'capturePerformance': true,
                'captureSecurity': true,
                'debounceTime': 5000
            };
            if (_0x8e4c) {
                Object['assign'](_0x1b6d, _0x8e4c);
            }
            Object['assign'](_0x5c8d, _0x1b6d);
            _0x5c8d['enabled'] = true;
            _0x3a1f['setupErrorHandler']();
            _0x3a1f['setupPerformanceObserver']();
            _0x3a1f['setupSecurityChecks']();
            _0x3a1f['setupVisibilityObserver']();
            return {
                'success': true,
                'config': _0x1b6d
            };
        },
        'log': function(_0x9a3e, _0x4c8f, _0x7e2a) {
            var _0x2d5b = {
                'type': _0x9a3e,
                'message': _0x4c8f,
                'data': _0x7e2a,
                'timestamp': Date['now'](),
                'url': window['location']['href'],
                'userAgent': navigator['userAgent']
            };
            if (_0x9a3e === 'error') {
                _0x5c8d['errors']['push'](_0x2d5b);
            } else if (_0x9a3e === 'warning') {
                _0x5c8d['warnings']['push'](_0x2d5b);
            }
            _0x5c8d['events']['push'](_0x2d5b);
            if (_0x5c8d['events']['length'] > _0x5c8d['maxEvents']) {
                _0x5c8d['events']['shift']();
            }
            if (window['console'] && window['console'][_0x9a3e]) {
                window['console'][_0x9a3e]('[SecurityMonitor]', _0x4c8f, _0x7e2a);
            }
            return _0x2d5b;
        },
        'track': function(_0x6a9c, _0x3b1d) {
            var _0x8f6e = {
                'event': _0x6a9c,
                'data': _0x3b1d,
                'timestamp': Date['now']()
            };
            if (_0x6a9c['indexOf']('security_') === 0) {
                _0x5c8d['security'][_0x6a9c] = _0x8f6e;
            } else if (_0x6a9c['indexOf']('performance_') === 0) {
                _0x5c8d['performance'][_0x6a9c] = _0x8f6e;
            }
            return _0x3a1f['log']('info', 'Event tracked: ' + _0x6a9c, _0x8f6e);
        },
        'report': async function(_0x4d2c) {
            try {
                var _0x2e8f = {
                    'events': _0x5c8d['events'],
                    'errors': _0x5c8d['errors'],
                    'warnings': _0x5c8d['warnings'],
                    'performance': _0x5c8d['performance'],
                    'security': _0x5c8d['security'],
                    'sessionDuration': Date['now']() - _0x5c8d['startTime'],
                    'reportTime': Date['now']()
                };
                if (_0x4d2c && _0x4d2c['reportUrl']) {
                    var _0x4a7e = await fetch(_0x4d2c['reportUrl'], {
                        'method': 'POST',
                        'headers': {
                            'Content-Type': 'application/json'
                        },
                        'body': JSON['stringify'](_0x2e8f)
                    });
                    return {
                        'success': true,
                        'report': _0x2e8f
                    };
                }
                return {
                    'success': true,
                    'report': _0x2e8f
                };
            } catch (_0x5c8d) {
                return {
                    'success': false,
                    'error': _0x5c8d['message']
                };
            }
        },
        'setupErrorHandler': function() {
            window['onerror'] = function(_0x3a1f, _0x7e4b, _0x2d9c, _0x6f8a, _0x1b5d) {
                _0x3a1f['log']('error', 'Global error: ' + _0x3a1f, {
                    'message': _0x3a1f,
                    'source': _0x7e4b,
                    'line': _0x2d9c,
                    'column': _0x6f8a,
                    'error': _0x1b5d ? _0x1b5d['stack'] : null
                });
                return false;
            };
            window['onunhandledrejection'] = function(_0x8e3c) {
                _0x3a1f['log']('error', 'Unhandled promise rejection', {
                    'reason': _0x8e3c['reason'] ? _0x8e3c['reason']['message'] : 'Unknown',
                    'stack': _0x8e3c['reason'] ? _0x8e3c['reason']['stack'] : null
                });
            };
        },
        'setupPerformanceObserver': function() {
            if ('PerformanceObserver' in window) {
                var _0x4d7f = new PerformanceObserver(function(_0x3e8d) {
                    var _0x5a1d = _0x3e8d['getEntries']();
                    for (var _0x7e4a = 0; _0x7e4a < _0x5a1d['length']; _0x7e4a++) {
                        var _0x8b5e = _0x5a1d[_0x7e4a];
                        _0x3a1f['track']('performance_' + _0x8b5e['name'], {
                            'duration': _0x8b5e['duration'],
                            'entryType': _0x8b5e['entryType']
                        });
                    }
                });
                try {
                    _0x4d7f['observe']({
                        'entryTypes': ['resource', 'paint', 'navigation']
                    });
                } catch (_0x1f3a) {
                    _0x3a1f['log']('warning', 'Performance observer setup failed', {
                        'error': _0x1f3a['message']
                    });
                }
            }
        },
        'setupSecurityChecks': function() {
            var _0x6c9f = setInterval(function() {
                var _0x3e8d = {
                    'timestamp': Date['now']()
                };
                if (window['outerWidth'] - window['innerWidth'] > 200) {
                    _0x3e8d['devToolsOpen'] = true;
                    _0x3a1f['track']('security_devtools_opened', _0x3e8d);
                }
                if (window['outerHeight'] - window['innerHeight'] > 200) {
                    _0x3e8d['devToolsOpen'] = true;
                    _0x3a1f['track']('security_devtools_opened', _0x3e8d);
                }
                if (typeof _0x3e8d['devToolsOpen'] !== 'undefined') {
                    _0x5c8d['security']['devToolsState'] = _0x3e8d;
                }
            }, 1000);
            _0x3a1f['track']('security_check_started', {
                'interval': 1000
            });
        },
        'setupVisibilityObserver': function() {
            if ('VisibilityObserver' in window) {
                var _0x3e8d = new VisibilityObserver(function(_0x5a1d) {
                    var _0x7e4a = _0x5a1d[0];
                    if (_0x7e4a['isIntersecting']) {
                        _0x3a1f['track']('visibility_visible', {
                            'timestamp': Date['now']()
                        });
                    } else {
                        _0x3a1f['track']('visibility_hidden', {
                            'timestamp': Date['now']()
                        });
                    }
                }, {});
                var _0x8b5e = document['createElement']('div');
                _0x8b5e['style']['position'] = 'absolute';
                _0x8b5e['style']['left'] = '-9999px';
                document['body']['appendChild'](_0x8b5e);
                _0x3e8d['observe'](_0x8b5e);
            }
        },
        'onError': function(_0x2e8f) {
            if (typeof _0x2e8f === 'function') {
                window['addEventListener']('error', function(_0x4a7e) {
                    _0x2e8f({
                        'message': _0x4a7e['message'],
                        'source': _0x4a7e['filename'],
                        'line': _0x4a7e['lineno'],
                        'column': _0x4a7e['colno'],
                        'error': _0x4a7e['error']
                    });
                });
            }
        },
        'onWarning': function(_0x2e8f) {
            if (typeof _0x2e8f === 'function') {
                var _0x4a7e = console['warn'];
                console['warn'] = function() {
                    _0x2e8f['apply'](console, arguments);
                    _0x4a7e['apply'](console, arguments);
                };
            }
        },
        'getStats': function() {
            return {
                'totalEvents': _0x5c8d['events']['length'],
                'totalErrors': _0x5c8d['errors']['length'],
                'totalWarnings': _0x5c8d['warnings']['length'],
                'sessionDuration': Date['now']() - _0x5c8d['startTime'],
                'performanceMetrics': _0x5c8d['performance'],
                'securityEvents': _0x5c8d['security']
            };
        }
    };
    return _0x3a1f;
});

window['CryptoSecurity'] = {
    'CryptoAPI': window['CryptoAPI'],
    'CodeObfuscator': window['CodeObfuscator'],
    'RequestEncryptor': window['RequestEncryptor'],
    'SecurityMonitor': window['SecurityMonitor'],
    'version': '1.0.0',
    'init': async function(_0x3b1a) {
        var _0x8c1a = {
            'enableMonitor': true,
            'enableProtection': true,
            'initCrypto': true
        };
        if (_0x3b1a) {
            Object['assign'](_0x8c1a, _0x3b1a);
        }
        if (_0x8c1a['initCrypto'] && window['RequestEncryptor']) {
            await window['RequestEncryptor']['init']();
        }
        if (_0x8c1a['enableMonitor'] && window['SecurityMonitor']) {
            window['SecurityMonitor']['init']({
                'enabled': true
            });
        }
        if (_0x8c1a['enableProtection'] && window['CodeObfuscator']) {
            window['CodeObfuscator']['protect']();
        }
        return {
            'success': true,
            'config': _0x8c1a
        };
    }
};
})(typeof window !== 'undefined' ? window : global, function(_0x3f7a, _0x5d2b, _0x8e4c, _0x1b6d, _0x9a3e, _0x4c8f, _0x7e2a, _0x2d5b, _0x6a9c, _0x3b1d, _0x8f6e, _0x4d2c) {
    var _0x2e8f = {
        'a': 'CryptoAPI',
        'b': 'CodeObfuscator',
        'c': 'RequestEncryptor',
        'd': 'SecurityMonitor',
        'e': 'version',
        'f': 'init'
    };
    return _0x2e8f['a'];
});
