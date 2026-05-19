(function() {
    'use strict';

    const tests = [
        {
            name: 'JavaScript混淆优化测试',
            run: async function() {
                if (!window.AdvancedObfuscator) {
                    return { passed: false, message: 'AdvancedObfuscator模块未加载' };
                }
                
                const originalCode = 'function test() { return "hello"; }';
                const obfuscated = AdvancedObfuscator.obfuscate(originalCode, {
                    renameVariables: true,
                    encryptStrings: true,
                    flattenControlFlow: true,
                    injectDeadCode: true
                });
                
                const isObfuscated = obfuscated !== originalCode && 
                                   !obfuscated.includes('function test') &&
                                   !obfuscated.includes('"hello"');
                
                return {
                    passed: isObfuscated,
                    message: isObfuscated ? '代码混淆成功' : '代码混淆失败'
                };
            }
        },
        {
            name: 'WebAssembly加密模块测试',
            run: async function() {
                if (!window.WasmCrypto) {
                    return { passed: false, message: 'WasmCrypto模块未加载' };
                }
                
                try {
                    await WasmCrypto.init();
                    const data = 'test-data-12345';
                    const key = await WasmCrypto.pbkdf2DeriveKey('password123', 'salt123', 1000, 256);
                    const encrypted = await WasmCrypto.aes256GcmEncrypt(data, key);
                    const decrypted = await WasmCrypto.aes256GcmDecrypt(encrypted, key);
                    
                    return {
                        passed: decrypted === data,
                        message: decrypted === data ? 'WASM加密解密成功' : 'WASM加密解密失败'
                    };
                } catch (e) {
                    return { passed: false, message: 'WASM测试异常: ' + e.message };
                }
            }
        },
        {
            name: '完整性校验增强测试',
            run: async function() {
                if (!window.IntegrityEnhanced) {
                    return { passed: false, message: 'IntegrityEnhanced模块未加载' };
                }
                
                IntegrityEnhanced.registerCheckpoint('test-checkpoint');
                const result = IntegrityEnhanced.performIntegrityCheck();
                
                return {
                    passed: result.success,
                    message: result.success ? '完整性校验通过' : '完整性校验失败'
                };
            }
        },
        {
            name: '反调试机制测试',
            run: async function() {
                if (!window.AntiDebugEnhanced) {
                    return { passed: false, message: 'AntiDebugEnhanced模块未加载' };
                }
                
                AntiDebugEnhanced.init({ strictMode: false });
                const status = AntiDebugEnhanced.getStatus();
                
                return {
                    passed: status.protectionActive,
                    message: status.protectionActive ? '反调试机制已激活' : '反调试机制未激活'
                };
            }
        },
        {
            name: '代码虚拟化保护测试',
            run: async function() {
                if (!window.CodeVirtualization) {
                    return { passed: false, message: 'CodeVirtualization模块未加载' };
                }
                
                const testFn = function(a, b) { return a + b; };
                const protectedFn = CodeVirtualization.protectFunction(testFn);
                const result = protectedFn(2, 3);
                
                const isProtected = protectedFn.toString() === 'function() { [Virtualized Code] }';
                
                return {
                    passed: result === 5 && isProtected,
                    message: result === 5 && isProtected ? '代码虚拟化保护成功' : '代码虚拟化保护失败'
                };
            }
        },
        {
            name: '防篡改检测测试',
            run: async function() {
                if (!window.TamperDetection) {
                    return { passed: false, message: 'TamperDetection模块未加载' };
                }
                
                let tamperDetected = false;
                TamperDetection.setDetectionCallback(function(event) {
                    if (event.type === 'suspicious_log') {
                        tamperDetected = true;
                    }
                });
                
                console.log('debugger test');
                
                return {
                    passed: tamperDetected,
                    message: tamperDetected ? '防篡改检测生效' : '防篡改检测未触发'
                };
            }
        },
        {
            name: '模块整合测试',
            run: async function() {
                if (!window.HJTPXProtection) {
                    return { passed: false, message: 'HJTPXProtection模块未加载' };
                }
                
                const result = await HJTPXProtection.init();
                
                return {
                    passed: result.success && result.modules.length > 0,
                    message: result.success ? '模块整合成功: ' + result.modules.join(', ') : '模块整合失败'
                };
            }
        }
    ];

    async function runTests() {
        console.log('=== HJTPX行为验证系统 - 前端代码加密保护测试 ===');
        
        const results = [];
        let passedCount = 0;
        let failedCount = 0;

        for (const test of tests) {
            console.log(`\n测试: ${test.name}`);
            try {
                const result = await test.run();
                results.push({ name: test.name, ...result });
                
                if (result.passed) {
                    passedCount++;
                    console.log(`  ✓ ${result.message}`);
                } else {
                    failedCount++;
                    console.log(`  ✗ ${result.message}`);
                }
            } catch (e) {
                failedCount++;
                results.push({ name: test.name, passed: false, message: '异常: ' + e.message });
                console.log(`  ✗ 异常: ${e.message}`);
            }
        }

        console.log(`\n=== 测试结果汇总 ===`);
        console.log(`通过: ${passedCount}`);
        console.log(`失败: ${failedCount}`);
        console.log(`成功率: ${((passedCount / tests.length) * 100).toFixed(1)}%`);

        return {
            total: tests.length,
            passed: passedCount,
            failed: failedCount,
            successRate: (passedCount / tests.length) * 100,
            details: results
        };
    }

    function showTestResults(results) {
        const container = document.createElement('div');
        container.style.cssText = `
            position: fixed;
            bottom: 20px;
            right: 20px;
            width: 400px;
            background: #1a1a2e;
            border-radius: 12px;
            padding: 20px;
            color: #fff;
            font-family: monospace;
            font-size: 12px;
            box-shadow: 0 4px 20px rgba(0,0,0,0.5);
            max-height: 500px;
            overflow-y: auto;
            z-index: 9999;
        `;

        let html = `
            <h3 style="margin:0 0 15px 0;color:#4ade80;">HJTPX Protection Test Results</h3>
            <div style="margin-bottom:10px;">
                <span>Total: ${results.total}</span> | 
                <span style="color:#4ade80;">Passed: ${results.passed}</span> | 
                <span style="color:#f87171;">Failed: ${results.failed}</span>
            </div>
            <div style="border-top:1px solid #333;padding-top:10px;">
        `;

        for (const detail of results.details) {
            const color = detail.passed ? '#4ade80' : '#f87171';
            const icon = detail.passed ? '✓' : '✗';
            html += `
                <div style="margin:5px 0;color:${color};">
                    ${icon} ${detail.name}: ${detail.message}
                </div>
            `;
        }

        html += '</div>';
        container.innerHTML = html;
        document.body.appendChild(container);
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', async function() {
            const results = await runTests();
            showTestResults(results);
        });
    } else {
        runTests().then(results => showTestResults(results));
    }

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = { tests, runTests };
    }
})();