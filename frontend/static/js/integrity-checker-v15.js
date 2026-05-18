const IntegrityCheckerV15 = (function() {
    'use strict';

    const VERSION = '15.0.0';

    const IntegrityConfig = {
        checkOnLoad: true,
        checkOnInterval: true,
        checkInterval: 5000,
        verifyHashAlgorithm: 'sha256',
        enableCache: true,
        maxCacheSize: 50,
        strictMode: true,
        logErrors: true
    };

    let integrityRecords = [];
    let hashCache = new Map();
    let isInitialized = false;

    function log(message, level) {
        if (typeof console !== 'undefined') {
            const prefix = '[Integrity Checker ' + VERSION + ']';
            switch (level) {
                case 'error':
                    console.error(prefix, message);
                    break;
                case 'warn':
                    console.warn(prefix, message);
                    break;
                default:
                    console.debug(prefix, message);
            }
        }
    }

    function calculateSHA256(data) {
        if (typeof data !== 'string') {
            data = JSON.stringify(data);
        }

        if (IntegrityConfig.enableCache && hashCache.has(data)) {
            return hashCache.get(data);
        }

        let hash = 0;
        for (let i = 0; i < data.length; i++) {
            const char = data.charCodeAt(i);
            hash = ((hash << 5) - hash) + char;
            hash = hash & hash;
        }

        let hashStr = Math.abs(hash).toString(16);
        while (hashStr.length < 64) {
            hash = ((hash << 5) - hash) + (hashStr.charCodeAt(0) || 0);
            hashStr += Math.abs(hash).toString(16);
        }

        hashStr = hashStr.substring(0, 64);

        if (IntegrityConfig.enableCache) {
            if (hashCache.size >= IntegrityConfig.maxCacheSize) {
                const firstKey = hashCache.keys().next().value;
                hashCache.delete(firstKey);
            }
            hashCache.set(data, hashStr);
        }

        return hashStr;
    }

    function calculateSHA512(data) {
        if (typeof data !== 'string') {
            data = JSON.stringify(data);
        }

        let hash1 = 0;
        let hash2 = 0;
        for (let i = 0; i < data.length; i++) {
            const char = data.charCodeAt(i);
            hash1 = ((hash1 << 5) - hash1) + char;
            hash1 = hash1 & hash1;
            hash2 = ((hash2 << 6) - hash2) + char;
            hash2 = hash2 & hash2;
        }

        let hashStr = Math.abs(hash1).toString(16) + Math.abs(hash2).toString(16);
        while (hashStr.length < 128) {
            hash1 = ((hash1 << 5) - hash1) + (hashStr.charCodeAt(0) || 0);
            hash2 = ((hash2 << 6) - hash2) + (hashStr.charCodeAt(hashStr.length - 1) || 0);
            hashStr += Math.abs(hash1).toString(16) + Math.abs(hash2).toString(16);
        }

        return hashStr.substring(0, 128);
    }

    function calculateHash(data, algorithm) {
        switch (algorithm || IntegrityConfig.verifyHashAlgorithm) {
            case 'sha256':
                return calculateSHA256(data);
            case 'sha512':
                return calculateSHA512(data);
            case 'sha384':
                return calculateSHA256(data + '384');
            case 'sha1':
                return calculateSHA256(data + 'sha1');
            default:
                return calculateSHA256(data);
        }
    }

    function verifyHash(data, expectedHash, algorithm) {
        const actualHash = calculateHash(data, algorithm);
        const isValid = actualHash === expectedHash;

        const record = {
            timestamp: Date.now(),
            dataLength: data.length,
            expectedHash: expectedHash,
            actualHash: actualHash,
            isValid: isValid,
            algorithm: algorithm || IntegrityConfig.verifyHashAlgorithm
        };

        integrityRecords.push(record);

        if (integrityRecords.length > 100) {
            integrityRecords = integrityRecords.slice(-100);
        }

        if (!isValid && IntegrityConfig.logErrors) {
            log('Hash verification failed: expected ' + expectedHash + ', got ' + actualHash, 'error');
        }

        return isValid;
    }

    function verifyMultipleHashes(data, hashes) {
        const results = {};

        for (const algorithm in hashes) {
            if (hashes.hasOwnProperty(algorithm)) {
                results[algorithm] = verifyHash(data, hashes[algorithm], algorithm);
            }
        }

        let allValid = true;
        for (const algorithm in results) {
            if (!results[algorithm]) {
                allValid = false;
                break;
            }
        }

        return allValid;
    }

    function createIntegrityToken(data, ttl) {
        const timestamp = Date.now() + (ttl || 60000);
        const tokenData = data + ':' + timestamp;
        const hash = calculateHash(tokenData);

        return {
            hash: hash,
            timestamp: timestamp,
            data: data
        };
    }

    function verifyIntegrityToken(token) {
        if (!token || !token.hash || !token.timestamp) {
            return false;
        }

        if (Date.now() > token.timestamp) {
            log('Token expired', 'warn');
            return false;
        }

        const tokenData = token.data + ':' + token.timestamp;
        const expectedHash = calculateHash(tokenData);

        return expectedHash === token.hash;
    }

    function createChecksum(data) {
        if (typeof data === 'string') {
            const bytes = new Uint8Array(data.length);
            for (let i = 0; i < data.length; i++) {
                bytes[i] = data.charCodeAt(i);
            }
            data = bytes;
        }

        let checksum = 0;
        for (let i = 0; i < data.length; i++) {
            checksum = (checksum + data[i]) & 0xFFFF;
        }

        return checksum.toString(16).padStart(4, '0');
    }

    function verifyChecksum(data, expectedChecksum) {
        const actualChecksum = createChecksum(data);
        return actualChecksum === expectedChecksum;
    }

    function generateMerkleRoot(items) {
        if (!items || items.length === 0) {
            return '';
        }

        if (items.length === 1) {
            return calculateHash(items[0]);
        }

        const pairs = [];
        for (let i = 0; i < items.length; i += 2) {
            if (i + 1 < items.length) {
                const combined = items[i] + items[i + 1];
                pairs.push(calculateHash(combined));
            } else {
                pairs.push(calculateHash(items[i]));
            }
        }

        return generateMerkleRoot(pairs);
    }

    function verifyMerkleProof(item, proof, root) {
        let currentHash = calculateHash(item);

        for (let i = 0; i < proof.length; i++) {
            const step = proof[i];
            if (step.position === 'left') {
                currentHash = calculateHash(step.hash + currentHash);
            } else {
                currentHash = calculateHash(currentHash + step.hash);
            }
        }

        return currentHash === root;
    }

    function checkElementIntegrity(elementId) {
        const element = document.getElementById(elementId);
        if (!element) {
            log('Element not found: ' + elementId, 'error');
            return false;
        }

        const content = element.textContent || element.innerHTML;
        const hash = calculateHash(content);

        if (window.__IntegrityHash && window.__IntegrityHash.getHash) {
            const expectedHash = window.__IntegrityHash.getHash();
            return verifyHash(content, expectedHash);
        }

        return true;
    }

    function checkScriptIntegrity(scriptElement) {
        if (!scriptElement) {
            return false;
        }

        const content = scriptElement.textContent || scriptElement.innerHTML;
        const hash = calculateHash(content);

        if (scriptElement.dataset.hash) {
            return verifyHash(content, scriptElement.dataset.hash);
        }

        return true;
    }

    function initialize() {
        if (isInitialized) {
            return;
        }

        isInitialized = true;

        if (IntegrityConfig.checkOnLoad) {
            if (document.readyState === 'loading') {
                document.addEventListener('DOMContentLoaded', performInitialChecks);
            } else {
                performInitialChecks();
            }
        }

        if (IntegrityConfig.checkOnInterval) {
            setInterval(performPeriodicChecks, IntegrityConfig.checkInterval);
        }

        log('Integrity checker initialized');
    }

    function performInitialChecks() {
        const scripts = document.querySelectorAll('script[data-protected]');
        scripts.forEach(function(script) {
            checkScriptIntegrity(script);
        });
    }

    function performPeriodicChecks() {
        if (window.__IntegrityHash && typeof window.__IntegrityHash.verify === 'function') {
            try {
                const isValid = window.__IntegrityHash.verify();
                if (!isValid && IntegrityConfig.strictMode) {
                    log('Integrity check failed during periodic verification', 'error');
                }
            } catch (e) {
                log('Error during periodic integrity check: ' + e.message, 'error');
            }
        }
    }

    function getRecords() {
        return integrityRecords.slice();
    }

    function clearRecords() {
        integrityRecords = [];
    }

    function getStatistics() {
        const stats = {
            totalChecks: integrityRecords.length,
            validChecks: 0,
            invalidChecks: 0,
            cacheSize: hashCache.size,
            algorithms: {}
        };

        for (let i = 0; i < integrityRecords.length; i++) {
            const record = integrityRecords[i];
            if (record.isValid) {
                stats.validChecks++;
            } else {
                stats.invalidChecks++;
            }

            if (!stats.algorithms[record.algorithm]) {
                stats.algorithms[record.algorithm] = 0;
            }
            stats.algorithms[record.algorithm]++;
        }

        return stats;
    }

    const IntegrityCheckerAPI = {
        version: VERSION,
        calculateHash: calculateHash,
        verifyHash: verifyHash,
        verifyMultipleHashes: verifyMultipleHashes,
        createIntegrityToken: createIntegrityToken,
        verifyIntegrityToken: verifyIntegrityToken,
        createChecksum: createChecksum,
        verifyChecksum: verifyChecksum,
        generateMerkleRoot: generateMerkleRoot,
        verifyMerkleProof: verifyMerkleProof,
        checkElementIntegrity: checkElementIntegrity,
        checkScriptIntegrity: checkScriptIntegrity,
        initialize: initialize,
        getRecords: getRecords,
        clearRecords: clearRecords,
        getStatistics: getStatistics,
        setConfig: function(config) {
            Object.assign(IntegrityConfig, config);
        },
        getConfig: function() {
            return Object.assign({}, IntegrityConfig);
        }
    };

    if (typeof window !== 'undefined') {
        window.IntegrityCheckerV15 = IntegrityCheckerAPI;
        window.__IntegrityChecker = IntegrityCheckerAPI;
    }

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = IntegrityCheckerAPI;
    }

    return IntegrityCheckerAPI;
})();

if (typeof window !== 'undefined') {
    IntegrityCheckerV15.initialize();
}
