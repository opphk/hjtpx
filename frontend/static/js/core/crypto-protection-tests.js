(function(globalContext) {
    'use strict';

    const CryptoProtectionTests = (function() {
        const VERSION = '1.0.0';
        const testResults = [];

        function log(message, type = 'info') {
            const prefix = type === 'success' ? '✅' : type === 'error' ? '❌' : 'ℹ️';
            console.log(`${prefix} ${message}`);
        }

        function assert(condition, message) {
            if (condition) {
                log(message, 'success');
                testResults.push({ test: message, passed: true });
                return true;
            } else {
                log(message, 'error');
                testResults.push({ test: message, passed: false });
                return false;
            }
        }

        function testObfuscator() {
            log('Testing AdvancedObfuscator...', 'info');
            
            if (typeof AdvancedObfuscator === 'undefined') {
                log('AdvancedObfuscator not found', 'error');
                return false;
            }

            assert(AdvancedObfuscator.VERSION === '5.0.0', 'AdvancedObfuscator version is correct');
            
            const testCode = 'function hello() { return "Hello World"; }';
            const obfuscated = AdvancedObfuscator.obfuscate(testCode);
            
            assert(obfuscated.length > testCode.length, 'Obfuscation increases code length');
            assert(!obfuscated.includes('hello'), 'Variable name is obfuscated');
            assert(obfuscated.includes('window.__decrypt'), 'Decoder function is added');
            
            log('AdvancedObfuscator tests completed', 'info');
            return true;
        }

        function testRC4Encryption() {
            log('Testing RC4 Encryption...', 'info');
            
            if (typeof AdvancedObfuscator === 'undefined') {
                log('AdvancedObfuscator not found', 'error');
                return false;
            }

            const testString = 'Hello RC4 World';
            const encrypted = AdvancedObfuscator.rc4Encrypt(testString);
            
            assert(Array.isArray(encrypted), 'RC4 encryption returns array');
            assert(encrypted.length === testString.length, 'Encrypted length matches');
            assert(encrypted[0] !== testString.charCodeAt(0), 'First character is encrypted');
            
            const decrypted = AdvancedObfuscator.rc4Decrypt(encrypted);
            const decryptedString = String.fromCharCode.apply(null, decrypted);
            
            assert(decryptedString === testString, 'RC4 decryption works correctly');
            
            log('RC4 Encryption tests completed', 'info');
            return true;
        }

        function testAntiDebug() {
            log('Testing AntiDebugEnhanced...', 'info');
            
            if (typeof AntiDebugEnhanced === 'undefined') {
                log('AntiDebugEnhanced not found', 'error');
                return false;
            }

            assert(AntiDebugEnhanced.VERSION === '5.0.0', 'AntiDebugEnhanced version is correct');
            
            const status = AntiDebugEnhanced.getStatus();
            assert(typeof status.enabled === 'boolean', 'Status includes enabled flag');
            assert(typeof status.violations === 'number', 'Status includes violations count');
            assert(typeof status.maxViolations === 'number', 'Status includes max violations');
            
            const checks = AntiDebugEnhanced.performChecks();
            assert(Array.isArray(checks), 'performChecks returns array');
            
            log('AntiDebugEnhanced tests completed', 'info');
            return true;
        }

        function testIntegrity() {
            log('Testing IntegrityEnhanced...', 'info');
            
            if (typeof IntegrityEnhanced === 'undefined') {
                log('IntegrityEnhanced not found', 'error');
                return false;
            }

            assert(IntegrityEnhanced.VERSION === '5.0.0', 'IntegrityEnhanced version is correct');
            
            const status = IntegrityEnhanced.getStatus();
            assert(typeof status.version === 'string', 'Status includes version');
            
            assert(typeof IntegrityEnhanced.computeCRC32 === 'function', 'computeCRC32 function exists');
            assert(typeof IntegrityEnhanced.computeSHA256 === 'function', 'computeSHA256 function exists');
            assert(typeof IntegrityEnhanced.computeSHA1 === 'function', 'computeSHA1 function exists');
            assert(typeof IntegrityEnhanced.computeFNV1a === 'function', 'computeFNV1a function exists');
            
            const crc32 = IntegrityEnhanced.computeCRC32('test');
            assert(typeof crc32 === 'number', 'CRC32 returns number');
            
            log('IntegrityEnhanced tests completed', 'info');
            return true;
        }

        function testCodeVirtualization() {
            log('Testing CodeVirtualization...', 'info');
            
            if (typeof CodeVirtualization === 'undefined') {
                log('CodeVirtualization not found', 'error');
                return false;
            }

            assert(CodeVirtualization.VERSION === '3.0.0', 'CodeVirtualization version is correct');
            
            const status = CodeVirtualization.getStatus();
            assert(typeof status.instructionCount === 'number', 'Status includes instruction count');
            assert(typeof status.maxInstructions === 'number', 'Status includes max instructions');
            
            assert(typeof CodeVirtualization.compile === 'function', 'compile function exists');
            assert(typeof CodeVirtualization.run === 'function', 'run function exists');
            
            const testInstructions = [
                [CodeVirtualization.OPCODES.PUSH, 42],
                [CodeVirtualization.OPCODES.HALT]
            ];
            
            const compiled = CodeVirtualization.compile(testInstructions);
            assert(Array.isArray(compiled), 'compile returns array');
            assert(compiled.length > 0, 'Compiled code is not empty');
            
            log('CodeVirtualization tests completed', 'info');
            return true;
        }

        function testCryptoModule() {
            log('Testing CryptoModule...', 'info');
            
            if (typeof CryptoModule === 'undefined') {
                log('CryptoModule not found', 'error');
                return false;
            }

            assert(typeof CryptoModule.hash === 'function', 'hash function exists');
            assert(typeof CryptoModule.encrypt === 'function', 'encrypt function exists');
            assert(typeof CryptoModule.decrypt === 'function', 'decrypt function exists');
            assert(typeof CryptoModule.generateRandomBytes === 'function', 'generateRandomBytes function exists');
            
            const randomBytes = CryptoModule.generateRandomBytes(16);
            assert(randomBytes instanceof Uint8Array, 'generateRandomBytes returns Uint8Array');
            assert(randomBytes.length === 16, 'generateRandomBytes returns correct length');
            
            log('CryptoModule tests completed', 'info');
            return true;
        }

        function testWasmCrypto() {
            log('Testing WasmCrypto...', 'info');
            
            if (typeof WasmCrypto === 'undefined') {
                log('WasmCrypto not found', 'error');
                return false;
            }

            assert(typeof WasmCrypto.hashSHA256 === 'function', 'hashSHA256 function exists');
            assert(typeof WasmCrypto.hmacSHA256 === 'function', 'hmacSHA256 function exists');
            assert(typeof WasmCrypto.encrypt === 'function', 'encrypt function exists');
            assert(typeof WasmCrypto.decrypt === 'function', 'decrypt function exists');
            
            const randomBytes = WasmCrypto.generateRandomBytes(32);
            assert(randomBytes instanceof Uint8Array, 'generateRandomBytes returns Uint8Array');
            assert(randomBytes.length === 32, 'generateRandomBytes returns correct length');
            
            log('WasmCrypto tests completed', 'info');
            return true;
        }

        function testTamperDetection() {
            log('Testing TamperDetection...', 'info');
            
            if (typeof TamperDetection === 'undefined') {
                log('TamperDetection not found', 'error');
                return false;
            }

            assert(TamperDetection.VERSION === '3.0.0', 'TamperDetection version is correct');
            
            const status = TamperDetection.getStatus();
            assert(typeof status.enabled === 'boolean', 'Status includes enabled flag');
            assert(typeof status.violations === 'number', 'Status includes violations');
            
            assert(typeof TamperDetection.protectFunction === 'function', 'protectFunction exists');
            assert(typeof TamperDetection.protectObject === 'function', 'protectObject exists');
            
            log('TamperDetection tests completed', 'info');
            return true;
        }

        async function runAllTests() {
            log('Starting Crypto Protection Module Tests...', 'info');
            log('='.repeat(50), 'info');

            const results = {
                obfuscator: testObfuscator(),
                rc4: testRC4Encryption(),
                antiDebug: testAntiDebug(),
                integrity: testIntegrity(),
                virtualization: testCodeVirtualization(),
                crypto: testCryptoModule(),
                wasm: testWasmCrypto(),
                tamper: testTamperDetection()
            };

            log('='.repeat(50), 'info');
            
            const passed = testResults.filter(r => r.passed).length;
            const failed = testResults.filter(r => !r.passed).length;
            
            log(`Test Results: ${passed} passed, ${failed} failed`, 'info');
            
            if (failed === 0) {
                log('All tests passed!', 'success');
            } else {
                log('Some tests failed!', 'error');
                testResults.filter(r => !r.passed).forEach(r => {
                    log(`Failed: ${r.test}`, 'error');
                });
            }

            return {
                total: testResults.length,
                passed: passed,
                failed: failed,
                results: results
            };
        }

        function getTestResults() {
            return testResults;
        }

        return {
            VERSION: VERSION,
            runAllTests: runAllTests,
            getTestResults: getTestResults,
            testObfuscator: testObfuscator,
            testRC4Encryption: testRC4Encryption,
            testAntiDebug: testAntiDebug,
            testIntegrity: testIntegrity,
            testCodeVirtualization: testCodeVirtualization,
            testCryptoModule: testCryptoModule,
            testWasmCrypto: testWasmCrypto,
            testTamperDetection: testTamperDetection
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = CryptoProtectionTests;
    } else {
        globalContext.CryptoProtectionTests = CryptoProtectionTests;
        
        if (typeof window !== 'undefined') {
            window.addEventListener('load', function() {
                console.log('Crypto Protection Test Suite loaded');
                console.log('Run tests with: CryptoProtectionTests.runAllTests()');
            });
        }
    }

})(typeof window !== 'undefined' ? window : (typeof global !== 'undefined' ? global : this));
