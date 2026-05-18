/**
 * HJTPX 加密模块测试指南
 * 
 * 在浏览器控制台测试加密解密功能
 */

(function() {
    console.log('=== HJTPX 加密模块测试指南 ===\n');

    console.log('1. 基础测试 - CryptoWasm 模块:');
    console.log('   // 检查模块是否加载');
    console.log('   if (window.CryptoWasm) {');
    console.log('       console.log("CryptoWasm loaded:", CryptoWasm.VERSION);');
    console.log('   }');

    console.log('\n2. 加密功能测试:');
    console.log('   // 测试加密');
    console.log('   const testData = {');
    console.log('       userId: "user123",');
    console.log('       timestamp: Date.now(),');
    console.log('       action: "click"');
    console.log('   };');
    console.log('   ');
    console.log('   // 初始化并加密');
    console.log('   (async () => {');
    console.log('       await CryptoWasm.initialize();');
    console.log('       const encrypted = await CryptoWasm.encrypt(');
    console.log('           JSON.stringify(testData),');
    console.log('           "your-secret-key"');
    console.log('       );');
    console.log('       console.log("Encrypted:", encrypted);');
    console.log('       ');
    console.log('       // 解密');
    console.log('       const decrypted = await CryptoWasm.decrypt(');
    console.log('           encrypted,');
    console.log('           "your-secret-key",');
    console.log('           { salt: encrypted.salt }');
    console.log('       );');
    console.log('       console.log("Decrypted:", decrypted);');
    console.log('   })();');

    console.log('\n3. 使用 EnhancedTrajectoryEncryptor:');
    console.log('   const encryptor = new EnhancedTrajectoryEncryptor({');
    console.log('       useWasm: true,');
    console.log('       fallbackToLegacy: true');
    console.log('   });');
    console.log('   ');
    console.log('   (async () => {');
    console.log('       await encryptor.initialize();');
    console.log('       console.log("Status:", encryptor.getStatus());');
    console.log('       ');
    console.log('       const trajectory = [');
    console.log('           { x: 100, y: 200, t: 1000 },');
    console.log('           { x: 150, y: 220, t: 1050 },');
    console.log('           { x: 200, y: 250, t: 1100 }');
    console.log('       ];');
    console.log('       ');
    console.log('       const encrypted = await encryptor.encryptTrajectory(trajectory);');
    console.log('       console.log("Encrypted trajectory:", encrypted);');
    console.log('       ');
    console.log('       // 验证解密');
    console.log('       const decrypted = await encryptor.decryptTrajectory(encrypted);');
    console.log('       console.log("Decrypted trajectory:", decrypted);');
    console.log('   })();');

    console.log('\n4. PBKDF2 密钥派生测试:');
    console.log('   (async () => {');
    console.log('       const password = "test-password";');
    console.log('       const salt = CryptoWasm.generateRandomBytes(16);');
    console.log('       const iterations = 100000;');
    console.log('       ');
    console.log('       const derivedKey = await CryptoWasm.pbkdf2(');
    console.log('           password,');
    console.log('           salt,');
    console.log('           iterations,');
    console.log('           256');
    console.log('       );');
    console.log('       console.log("Derived key:", derivedKey);');
    console.log('   })();');

    console.log('\n5. HMAC-SHA256 测试:');
    console.log('   (async () => {');
    console.log('       const data = "Hello, World!";');
    console.log('       const key = "secret-key";');
    console.log('       ');
    console.log('       const hmac = await CryptoWasm.hmacSHA256(data, key);');
    console.log('       console.log("HMAC:", hmac);');
    console.log('   })();');

    console.log('\n6. 完整集成测试:');
    console.log('   // 模拟完整的验证码流程');
    console.log('   (async () => {');
    console.log('       const encryptor = new EnhancedTrajectoryEncryptor();');
    console.log('       await encryptor.initialize();');
    console.log('       ');
    console.log('       // 生成模拟轨迹数据');
    console.log('       const trajectory = [];');
    console.log('       for (let i = 0; i < 10; i++) {');
    console.log('           trajectory.push({');
    console.log('               x: Math.random() * 300,');
    console.log('               y: Math.random() * 150,');
    console.log('               t: Date.now() + i * 50');
    console.log('           });');
    console.log('       }');
    console.log('       ');
    console.log('       // 加密并发送');
    console.log('       const payload = await encryptor.encryptTrajectory(trajectory);');
    console.log('       console.log("Ready to send:", payload);');
    console.log('       ');
    console.log('       // 验证签名和时间戳');
    console.log('       const isValidTimestamp = encryptor.validateTimestamp(payload.timestamp);');
    console.log('       console.log("Valid timestamp:", isValidTimestamp);');
    console.log('   })();');

    console.log('\n=== 测试说明 ===');
    console.log('- 打开浏览器开发者工具 (F12)');
    console.log('- 切换到 Console 标签');
    console.log('- 复制上述测试代码并运行');
    console.log('- 观察控制台输出');
    console.log('- 如果 WASM 模块加载成功，会显示 "wasm_mode"');
    console.log('- 如果 WASM 不可用，会自动降级到 Web Crypto API ("webcrypto_mode")');

    console.log('\n=== 预期输出示例 ===');
    console.log('- CryptoWasm loaded: 1.0.0');
    console.log('- Status: { isInitialized: true, useLegacyFallback: false, wasmAvailable: true, encryptionMode: "AES-256-GCM" }');
    console.log('- Encrypted trajectory: { timestamp: ..., salt: ..., encrypted_data: ..., signature: ..., encryption_mode: "wasm", algorithm: "AES-256-GCM" }');

    console.log('\n=== 故障排除 ===');
    console.log('1. 如果 CryptoWasm 未定义，检查是否已加载 crypto-wasm.js');
    console.log('2. 如果 WASM 加载失败，检查 /static/js/crypto-wasm.wasm 是否存在');
    console.log('3. 降级到 Web Crypto API 是安全的，加密功能不受影响');

})();
