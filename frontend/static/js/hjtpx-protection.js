(function(globalContext) {
    'use strict';

    const HJTPXProtection = (function() {
        const VERSION = '2.0.0';
        
        const modules = {};
        const initialized = false;

        function loadScript(url) {
            return new Promise((resolve, reject) => {
                const script = document.createElement('script');
                script.src = url;
                script.onload = () => resolve();
                script.onerror = () => reject(new Error(`Failed to load ${url}`));
                document.head.appendChild(script);
            });
        }

        async function initializeModules() {
            const coreScripts = [
                '/static/js/core/obfuscator-core.js',
                '/static/js/core/wasm-crypto.js',
                '/static/js/core/integrity-enhanced.js',
                '/static/js/core/anti-debug-enhanced.js',
                '/static/js/core/code-virtualization.js',
                '/static/js/core/tamper-detection.js'
            ];

            for (const script of coreScripts) {
                try {
                    await loadScript(script);
                } catch (e) {
                    console.warn(`Failed to load optional module: ${script}`);
                }
            }

            if (window.AdvancedObfuscator) {
                modules.obfuscator = window.AdvancedObfuscator;
            }
            if (window.WasmCrypto) {
                modules.wasmCrypto = window.WasmCrypto;
            }
            if (window.IntegrityEnhanced) {
                modules.integrity = window.IntegrityEnhanced;
            }
            if (window.AntiDebugEnhanced) {
                modules.antiDebug = window.AntiDebugEnhanced;
            }
            if (window.CodeVirtualization) {
                modules.virtualization = window.CodeVirtualization;
            }
            if (window.TamperDetection) {
                modules.tamperDetection = window.TamperDetection;
            }
        }

        function initAntiDebug() {
            if (modules.antiDebug) {
                modules.antiDebug.init({
                    strictMode: true,
                    maxViolations: 3,
                    onDetection: handleSecurityEvent
                });
            }
        }

        function initTamperDetection() {
            if (modules.tamperDetection) {
                modules.tamperDetection.init({
                    checkInterval: 3000,
                    maxViolations: 3
                });
                modules.tamperDetection.setDetectionCallback(handleSecurityEvent);
            }
        }

        function initIntegrityCheck() {
            if (modules.integrity) {
                modules.integrity.registerCheckpoint('hjtpx-core');
            }
        }

        function handleSecurityEvent(event) {
            console.warn('[HJTPX Security] Event detected:', event);
            
            if (event.type === 'debugger_detected' || event.type === 'tamper_detected') {
                triggerSecurityLockdown(event);
            }
        }

        function triggerSecurityLockdown(event) {
            document.body.innerHTML = `
                <div style="position:fixed;top:0;left:0;width:100%;height:100%;background:#0d0d0d;color:#ff4444;font-family:Arial,sans-serif;display:flex;justify-content:center;align-items:center;">
                    <div style="text-align:center;max-width:600px;padding:40px;">
                        <div style="width:100px;height:100px;margin:0 auto 25px;border-radius:50%;background:#ff4444;display:flex;justify-content:center;align-items:center;">
                            <span style="font-size:48px;">&#x1F512;</span>
                        </div>
                        <h1 style="font-size:32px;margin:0 0 15px 0;">Security Alert</h1>
                        <p style="font-size:14px;opacity:0.9;margin:0 0 20px 0;">
                            Security measures have been triggered due to detected suspicious activity.<br>
                            Please refresh the page to continue.
                        </p>
                        <div style="background:rgba(255,255,255,0.05);padding:15px;border-radius:4px;text-align:left;font-size:12px;">
                            <div><strong>Event:</strong> ${event.type}</div>
                            <div><strong>Time:</strong> ${new Date().toLocaleString()}</div>
                        </div>
                    </div>
                </div>
            `;
        }

        function obfuscateCode(code, options) {
            if (modules.obfuscator) {
                return modules.obfuscator.obfuscate(code, options);
            }
            return code;
        }

        function protectFunction(fn, options) {
            if (modules.virtualization) {
                const protectedFn = modules.virtualization.protectFunction(fn);
                if (modules.tamperDetection) {
                    return modules.tamperDetection.protectFunction(protectedFn, options?.name);
                }
                return protectedFn;
            }
            return fn;
        }

        function protectObject(obj, name) {
            if (modules.tamperDetection) {
                return modules.tamperDetection.protectObject(obj, name);
            }
            return obj;
        }

        function encryptData(data, key) {
            if (modules.wasmCrypto) {
                return modules.wasmCrypto.aes256GcmEncrypt(data, key);
            }
            return Promise.resolve(data);
        }

        function decryptData(data, key) {
            if (modules.wasmCrypto) {
                return modules.wasmCrypto.aes256GcmDecrypt(data, key);
            }
            return Promise.resolve(data);
        }

        function verifyIntegrity() {
            if (modules.integrity) {
                return modules.integrity.performIntegrityCheck();
            }
            return { success: true, checks: [] };
        }

        async function init(config) {
            if (initialized) {
                console.warn('[HJTPX Protection] Already initialized');
                return;
            }

            await initializeModules();

            initAntiDebug();
            initTamperDetection();
            initIntegrityCheck();

            if (modules.wasmCrypto) {
                await modules.wasmCrypto.init();
            }

            console.log('[HJTPX Protection] All modules loaded:', Object.keys(modules));
            
            return {
                success: true,
                modules: Object.keys(modules),
                version: VERSION
            };
        }

        function getStatus() {
            const status = {
                version: VERSION,
                modules: {},
                securityStatus: 'operational'
            };

            if (modules.antiDebug) {
                status.modules.antiDebug = modules.antiDebug.getStatus();
            }
            if (modules.tamperDetection) {
                status.modules.tamperDetection = modules.tamperDetection.getStatus();
                if (status.modules.tamperDetection.tamperDetected) {
                    status.securityStatus = 'compromised';
                }
            }
            if (modules.integrity) {
                status.modules.integrity = 'enabled';
            }
            if (modules.virtualization) {
                status.modules.virtualization = modules.virtualization.getStatus();
            }
            if (modules.wasmCrypto) {
                status.modules.wasmCrypto = modules.wasmCrypto.isInitialized() ? 'initialized' : 'not_initialized';
            }
            if (modules.obfuscator) {
                status.modules.obfuscator = 'available';
            }

            return status;
        }

        return {
            VERSION: VERSION,
            init: init,
            getStatus: getStatus,
            obfuscateCode: obfuscateCode,
            protectFunction: protectFunction,
            protectObject: protectObject,
            encryptData: encryptData,
            decryptData: decryptData,
            verifyIntegrity: verifyIntegrity
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = HJTPXProtection;
    } else {
        globalContext.HJTPXProtection = HJTPXProtection;
    }

})(typeof window !== 'undefined' ? window : (typeof global !== 'undefined' ? global : this));