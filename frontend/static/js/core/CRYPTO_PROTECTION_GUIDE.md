# 前端代码加密功能完整指南

## 概述

本文档详细介绍 `/workspace/frontend/static/js` 目录下的前端代码加密功能，涵盖 JavaScript 混淆、字符串加密、反调试、完整性校验和代码虚拟化等核心功能。

## 核心模块

### 1. JavaScript 混淆器 (obfuscator-core.js)

#### 功能特性

- **版本**: 5.0.0
- **RC4 字符串加密**: 使用 RC4 流加密算法保护字符串
- **XOR 字符串加密**: 传统的 XOR 加密作为备选
- **变量名混淆**: 将有意义的变量名替换为随机字符
- **控制流扁平化**: 将嵌套的控制流转换为平面结构
- **字符串分段**: 将长字符串分割成多个片段
- **自保护代码**: 检测代码完整性
- **反调试代码**: 内置调试器检测
- **死代码注入**: 插入无用的代码增加分析难度

#### 使用示例

```javascript
// 引入混淆器
<script src="core/obfuscator-core.js"></script>

// 基本使用
const originalCode = `function hello() { return "Hello World"; }`;
const obfuscated = AdvancedObfuscator.obfuscate(originalCode);
console.log(obfuscated);

// 配置选项
AdvancedObfuscator.setConfig({
    enableRC4Encryption: true,      // 启用 RC4 加密
    enableControlFlowFlattening: true,
    enableDeadCodeInjection: true,
    enableSelfDefending: true,
    enableAntiDebug: true,
    controlFlowIterations: 3,       // 控制流迭代次数
    stringChunkSize: 50,           // 字符串分段大小
    maxStringLength: 100           // 最大字符串长度
});

// 获取统计信息
const stats = AdvancedObfuscator.getStats();
console.log(stats);
// {
//   variablesObfuscated: 5,
//   stringsEncrypted: 10,
//   functionsWrapped: 3,
//   version: '5.0.0',
//   rc4Enabled: true,
//   controlFlowIterations: 3,
//   selfDefendingEnabled: true,
//   antiDebugEnabled: true
// }

// 单独使用 RC4 加密
const encrypted = AdvancedObfuscator.rc4Encrypt('Hello World');
const decrypted = AdvancedObfuscator.rc4Decrypt(encrypted);
```

### 2. 反调试增强 (anti-debug-enhanced.js)

#### 功能特性

- **版本**: 5.0.0
- **窗口大小检测**: 检测开发者工具打开
- **调试器语句检测**: 检测 `debugger` 语句
- **控制台篡改检测**: 监控 console 操作
- **自动化检测**: 检测 Selenium、Puppeteer、Playwright
- **性能分析检测**: 监控内存使用
- **堆栈跟踪分析**: 分析异常堆栈
- **Source Map 检测**: 防止源码泄露
- **代理/VPN 检测**: 检测网络环境
- **时间异常检测**: 检测时间操作
- **隐身模式**: 延迟阻断，更难发现

#### 使用示例

```javascript
// 引入反调试模块
<script src="core/anti-debug-enhanced.js"></script>

// 初始化
AntiDebugEnhanced.init({
    enabled: true,
    maxViolations: 3,
    checkInterval: 2000
});

// 启用隐身模式
AntiDebugEnhanced.enableStealthMode();

// 设置保护模式
// 'block': 直接阻止 (默认)
// 'corrupt': 破坏脚本
// 'silent': 静默模式
AntiDebugEnhanced.setProtectionMode('block');

// 设置触发回调
AntiDebugEnhanced.setTriggerCallback((detection) => {
    console.log('检测到调试:', detection);
});

// 单独执行检测
const checks = AntiDebugEnhanced.performChecks();
const detected = checks.filter(c => c.detected);
if (detected.length > 0) {
    console.log('检测到问题:', detected);
}

// 获取状态
const status = AntiDebugEnhanced.getStatus();
console.log(status.enabled, status.violations);

// 启用/禁用特定检测
AntiDebugEnhanced.setCheckEnabled('windowSize', true);
AntiDebugEnhanced.setCheckEnabled('automationDetection', true);
```

### 3. 完整性校验 (integrity-enhanced.js)

#### 功能特性

- **版本**: 5.0.0
- **多种哈希算法**:
  - CRC32: 快速校验
  - MD5: 传统哈希
  - SHA-1: 安全哈希
  - SHA-256: 标准哈希
  - SHA-512: 强哈希
  - FNV-1a: 快速哈希
  - Adler-32: 快速校验
- **DOM 标记验证**: 在 DOM 中嵌入标记
- **水印系统**: 版权保护
- **多重哈希**: 综合校验

#### 使用示例

```javascript
// 引入完整性校验模块
<script src="core/integrity-enhanced.js"></script>

// 初始化
IntegrityEnhanced.init({
    checkInterval: 10000,
    maxChecks: 100,
    maxViolations: 3,
    enableWatermark: true
});

// 计算单个哈希
const crc32 = IntegrityEnhanced.computeCRC32('test data');
const sha256 = await IntegrityEnhanced.computeSHA256('test data');
const sha512 = await IntegrityEnhanced.computeSHA512('test data');
const md5 = IntegrityEnhanced.computeMD5('test data');
const sha1 = await IntegrityEnhanced.computeSHA1('test data');
const fnv1a = IntegrityEnhanced.computeFNV1a('test data');
const adler32 = IntegrityEnhanced.computeAdler32('test data');

// 计算多重哈希
const hashes = await IntegrityEnhanced.computeMultipleHashes('test data');
console.log(hashes);
// {
//   crc32: '...',
//   sha1: '...',
//   sha256: '...',
//   sha512: '...',
//   md5: '...',
//   fnv1a: '...',
//   adler32: '...',
//   combined: '...',
//   timestamp: 1234567890
// }

// 验证代码哈希
const verification = await IntegrityEnhanced.verifyCodeHash(code, expectedHashes);
if (!verification.checks.all) {
    console.error('代码完整性验证失败');
}

// 生成签名
const signature = await IntegrityEnhanced.generateSignature(data, key);

// 验证签名
const isValid = await IntegrityEnhanced.verifySignature(signature, data, key);

// 设置验证模式
// 'strict': 严格模式 (默认)
// 'relaxed': 宽松模式
// 'minimal': 最小验证
IntegrityEnhanced.setVerificationMode('strict');

// 启用/禁用水印
IntegrityEnhanced.setWatermarkEnabled(true);

// 获取状态
const status = IntegrityEnhanced.getStatus();
console.log(status);
```

### 4. 代码虚拟化 (code-virtualization.js)

#### 功能特性

- **版本**: 3.0.0
- **48 个操作码**: 丰富的指令集
- **字符串操作**: 内置字符串指令
- **数组操作**: 内置数组指令
- **对象操作**: 内置对象指令
- **时间操作**: 内置时间指令
- **加密操作**: 内置加密指令
- **验证操作**: 内置验证指令
- **调试支持**: 断点支持

#### 操作码列表

```
基础指令: NOP, PUSH, POP, HALT
算术指令: ADD, SUB, MUL, DIV, MOD
逻辑指令: AND, OR, XOR, NOT, SHL, SHR, CMP
控制流: JMP, JZ, JNZ, CALL, RET
内存操作: LOAD, STORE
字符串: STRING_LENGTH, STRING_CHAR_AT, STRING_CONCAT, STRING_EQUALS
数组: ARRAY_CREATE, ARRAY_GET, ARRAY_SET, ARRAY_LENGTH
对象: OBJECT_CREATE, OBJECT_GET, OBJECT_SET
时间: TIME_NOW, TIME_SLEEP
加密: ENCRYPT, DECRYPT, HASH, VALIDATE, CHECKSUM
调试: CONSOLE_LOG, ERROR_THROW, ASSERT
```

#### 使用示例

```javascript
// 引入代码虚拟化模块
<script src="core/code-virtualization.js"></script>

// 基本使用
const instructions = [
    [CodeVirtualization.OPCODES.PUSH, 42],
    [CodeVirtualization.OPCODES.HALT]
];

const compiled = CodeVirtualization.compile(instructions);
const result = CodeVirtualization.run(compiled);

console.log(result.stack);  // [42]
console.log(result.completed);  // true

// 保护函数
function myFunction(x, y) {
    return x + y;
}

const protected = CodeVirtualization.protectFunction(myFunction);
const result = protected(5, 3);  // 8

// 虚拟化代码生成
const virtualCode = CodeVirtualization.generateVirtualizedCode('test data');
const vmResult = CodeVirtualization.run(virtualCode);

// VM 状态管理
CodeVirtualization.setMaxInstructions(100000);
CodeVirtualization.enableStringOperations(true);
CodeVirtualization.enableArrayOperations(true);
CodeVirtualization.enableTimeOperations(true);

// 断点管理
CodeVirtualization.addBreakpoint(10);
CodeVirtualization.removeBreakpoint(10);
CodeVirtualization.clearBreakpoints();

// 获取 VM 状态
const status = CodeVirtualization.getVMStatus();
console.log(status);
// {
//   running: false,
//   instructionCount: 10,
//   stackDepth: 1,
//   memoryUsed: 0,
//   stringsCount: 0,
//   arraysCount: 0,
//   objectsCount: 0,
//   maxInstructions: 100000
// }
```

### 5. 加密模块 (crypto-module.js)

#### 功能特性

- **版本**: 3.0.0
- **AES-GCM/AES-CBC 加密**: 标准对称加密
- **PBKDF2 密钥派生**: 安全密钥生成
- **SHA-256/SHA-512 哈希**: 密码学哈希
- **HMAC**: 消息认证
- **安全存储**: 加密的 localStorage
- **轨迹数据加密**: 专用于轨迹数据

#### 使用示例

```javascript
// 引入加密模块
<script src="core/crypto-module.js"></script>

// 字符串加密
const encrypted = await CryptoModule.encryptString('Hello World');
const decrypted = await CryptoModule.decryptString(encrypted);

// AES 加密
const aesEncrypted = await CryptoModule.encrypt('plaintext', 'key');
const aesDecrypted = await CryptoModule.decrypt(aesEncrypted, 'key');

// 哈希计算
const hash = await CryptoModule.hash('data to hash');

// HMAC
const hmac = await CryptoModule.generateHMAC('message', 'key');
const isValid = await CryptoModule.verifyHMAC('message', hmac, 'key');

// 生成随机数据
const randomBytes = CryptoModule.generateRandomBytes(32);
const randomString = CryptoModule.generateRandomString(16);

// 安全存储
const storage = CryptoModule.secureStorage('myKey');
await storage.set({ data: 'value' });
const value = await storage.get();
await storage.remove();

// 轨迹数据加密
const trajectoryEncrypted = await CryptoModule.encryptTrajectoryData(trajectoryData, sessionKey);
const trajectoryDecrypted = await CryptoModule.decryptTrajectoryData(trajectoryEncrypted, sessionKey);
```

### 6. WebAssembly 加密 (wasm-crypto.js)

#### 功能特性

- **版本**: 3.1.0
- **WebAssembly 支持**: 高性能加密
- **AES-256-GCM**: 强加密
- **PBKDF2**: 密钥派生
- **密钥轮换**: 自动密钥更新

#### 使用示例

```javascript
// 引入 WASM 加密模块
<script src="core/wasm-crypto.js"></script>

// 初始化
const status = await WasmCrypto.initialize();
console.log(status.wasmLoaded, status.wasmSupported);

// 加密
const encrypted = await WasmCrypto.encrypt('plaintext', 'key');
const decrypted = await WasmCrypto.decrypt(encrypted, 'key');

// 哈希
const hash = await WasmCrypto.hashSHA256('data');

// HMAC
const hmac = await WasmCrypto.hmacSHA256('message', 'key');

// 密钥管理
const key = await WasmCrypto.initializeKey('password');
WasmCrypto.startKeyRotation(30 * 60 * 1000);  // 30分钟
WasmCrypto.stopKeyRotation();

// 随机数生成
const randomBytes = WasmCrypto.generateRandomBytes(32);

// 切换 WASM/JS 模式
WasmCrypto.setUseWasm(true);
```

### 7. 篡改检测 (tamper-detection.js)

#### 功能特性

- **版本**: 3.0.0
- **函数保护**: 保护关键函数
- **对象保护**: 保护重要对象
- **内存快照**: 监控内存变化
- **DOM 监控**: 监控 DOM 修改
- **控制台监控**: 监控控制台操作
- **网络监控**: 监控网络请求
- **Eval 监控**: 监控 eval 使用

#### 使用示例

```javascript
// 引入篡改检测模块
<script src="core/tamper-detection.js"></script>

// 初始化
TamperDetection.init({
    enabled: true,
    maxViolations: 3,
    checkInterval: 3000
});

// 保护函数
function myFunction() {
    return 'result';
}

const protected = TamperDetection.protectFunction(myFunction, 'myFunction');

// 保护对象
const protectedObj = TamperDetection.protectObject({ key: 'value' }, 'myObject');

// 创建内存快照
TamperDetection.createMemorySnapshot('initial');
setTimeout(() => {
    const result = TamperDetection.compareMemorySnapshots('initial');
    if (result.changed) {
        console.log('内存异常:', result.details);
    }
}, 5000);

// 设置检测回调
TamperDetection.setDetectionCallback((detection) => {
    console.log('检测到篡改:', detection);
});

// 获取状态
const status = TamperDetection.getStatus();
console.log(status);

// 清理
TamperDetection.cleanup();
```

## 集成使用示例

### 完整集成

```javascript
// 1. 初始化所有模块
<script src="core/obfuscator-core.js"></script>
<script src="core/anti-debug-enhanced.js"></script>
<script src="core/integrity-enhanced.js"></script>
<script src="core/crypto-module.js"></script>

<script>
// 2. 初始化反调试
AntiDebugEnhanced.init({
    maxViolations: 3,
    checkInterval: 2000
});

// 3. 初始化完整性校验
IntegrityEnhanced.init({
    enableWatermark: true
});

// 4. 混淆代码
function mySecretFunction() {
    return 'secret';
}

const protectedCode = AdvancedObfuscator.obfuscate(mySecretFunction.toString());
console.log('混淆后的代码长度:', protectedCode.length);

// 5. 加密数据
async function secureData(data) {
    const encrypted = await CryptoModule.encryptString(JSON.stringify(data));
    return encrypted;
}

// 6. 完整性检查
async function verifyIntegrity(data) {
    const hashes = await IntegrityEnhanced.computeMultipleHashes(data);
    return hashes;
}
</script>
```

## 测试

### 运行测试

```javascript
// 引入测试模块
<script src="core/crypto-protection-tests.js"></script>

<script>
// 运行所有测试
const results = await CryptoProtectionTests.runAllTests();
console.log(results);
// {
//   total: 15,
//   passed: 15,
//   failed: 0,
//   results: { ... }
// }

// 运行特定测试
CryptoProtectionTests.testObfuscator();
CryptoProtectionTests.testRC4Encryption();
CryptoProtectionTests.testAntiDebug();
CryptoProtectionTests.testIntegrity();
CryptoProtectionTests.testCodeVirtualization();
CryptoProtectionTests.testCryptoModule();
CryptoProtectionTests.testWasmCrypto();
CryptoProtectionTests.testTamperDetection();
</script>
```

### 测试页面

打开 `core/crypto-protection-test.html` 可以通过图形界面测试所有功能。

## 性能考虑

1. **混淆级别**: 高级混淆会增加代码体积和执行时间
2. **加密算法**: RC4 比 XOR 慢但更安全
3. **检查间隔**: 反调试检查过于频繁可能影响性能
4. **虚拟化**: 代码虚拟化会显著降低执行速度，仅用于关键代码

## 安全建议

1. 不要在客户端存储敏感密钥
2. 混淆不能完全阻止逆向工程
3. 结合后端验证提高安全性
4. 定期更新混淆策略
5. 使用 HTTPS 传输
6. 结合 CSP (Content Security Policy)

## 版本历史

- **5.0.0**: 添加 RC4 加密、控制流迭代、自保护代码、隐身模式、多重哈希
- **4.0.0**: 添加水印、多种哈希算法、更多反调试检测
- **3.0.0**: 添加代码虚拟化、WASM 加密、篡改检测
- **2.0.0**: 基础混淆和控制流扁平化
- **1.0.0**: 初始版本

## 许可证

MIT License

## 支持

如有问题，请检查浏览器控制台的错误信息。
