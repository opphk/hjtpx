# 前端代码模块化重构文档

## 概述

本文档说明了前端代码的模块化重构方案，将原有的单体JavaScript文件拆分为多个职责明确的模块，提高代码的可维护性、可扩展性和复用性。

## 目录结构

```
frontend/static/js/
├── constants/
│   └── constants.js      # 全局常量定义
├── utils/
│   └── utils.js         # 通用工具函数
├── core/
│   ├── captcha-core.js              # 验证码核心逻辑
│   ├── crypto-module.js             # 加密解密模块
│   └── environment-detector-core.js # 环境检测核心模块
├── components/
│   └── ui-components.js  # 通用UI组件
└── index.js              # 模块入口文件
```

## 模块说明

### 1. 常量模块 (constants/constants.js)

**功能：**
- 集中管理所有全局常量
- 包括检测权重、阈值、配置参数等

**主要常量：**

- `API_BASE`: API基础路径
- `DETECTION_WEIGHTS`: 各种检测方法的权重配置
- `AUTO_TOOLS`: 自动化工具检测列表
- `PROXY_INDICATORS`: 代理检测指标
- `RISK_THRESHOLDS`: 风险评分阈值
- `DEFAULT_KEY`: 默认加密密钥
- `CAPTCHA_TYPES`: 验证码类型枚举
- `ERROR_MESSAGES`: 错误消息定义
- `UI_COLORS`: UI颜色配置

**使用示例：**

```javascript
// 引入常量模块
<script src="/static/js/constants/constants.js"></script>

// 使用常量
var apiBase = CaptchaConstants.API_BASE;
var weights = CaptchaConstants.DETECTION_WEIGHTS;
```

### 2. 工具函数模块 (utils/utils.js)

**功能：**
- 提供通用工具函数
- 包含数据处理、性能监控、设备检测等功能

**主要功能：**

- **函数工具：**
  - `debounce(func, wait)`: 防抖函数
  - `throttle(func, limit)`: 节流函数

- **ID生成：**
  - `generateUUID()`: 生成UUID
  - `generateRandomString(length)`: 生成随机字符串

- **数据处理：**
  - `deepClone(obj)`: 深拷贝对象
  - `isEmpty(value)`: 检查值是否为空
  - `groupBy(array, key)`: 数组分组
  - `omit(obj, keys)`: 排除对象属性
  - `pick(obj, keys)`: 选择对象属性

- **字符串处理：**
  - `parseQueryString(query)`: 解析查询字符串
  - `buildQueryString(params)`: 构建查询字符串
  - `truncate(str, maxLength, suffix)`: 截断字符串

- **格式转换：**
  - `formatTime(milliseconds)`: 格式化时间
  - `formatFileSize(bytes)`: 格式化文件大小
  - `formatNumber(num)`: 格式化数字
  - `camelToSnake(str)`: 驼峰转蛇形
  - `snakeToCamel(str)`: 蛇形转驼峰

- **验证函数：**
  - `isValidEmail(email)`: 验证邮箱
  - `isValidURL(url)`: 验证URL

- **设备检测：**
  - `getBrowserInfo()`: 获取浏览器信息
  - `getDeviceType()`: 获取设备类型
  - `supportsTouch()`: 检查触摸支持
  - `supportsWebGL()`: 检查WebGL支持
  - `supportsLocalStorage()`: 检查LocalStorage支持
  - `getConnectionType()`: 获取连接类型
  - `getEffectiveType()`: 获取有效连接类型

- **性能监控：**
  - `observePerformance(callback)`: 观察性能指标
  - `getPageLoadTime()`: 获取页面加载时间
  - `getDOMContentLoadedTime()`: 获取DOM加载时间

- **异步工具：**
  - `waitFor(condition, timeout, interval)`: 等待条件满足
  - `retry(fn, maxAttempts, delay)`: 重试机制

**使用示例：**

```javascript
// 使用工具函数
var uuid = CaptchaUtils.generateUUID();
var debouncedFunc = CaptchaUtils.debounce(myFunc, 300);
var browserInfo = CaptchaUtils.getBrowserInfo();

// 性能监控
CaptchaUtils.observePerformance(function(entry) {
    console.log('Performance entry:', entry);
});
```

### 3. 加密模块 (core/crypto-module.js)

**功能：**
- 提供加密解密功能
- 支持AES-GCM、AES-CBC模式
- 提供安全存储、HMAC签名等功能

**主要功能：**

- **加密解密：**
  - `encrypt(plaintext, key, options)`: AES加密
  - `decrypt(encryptedData, key, options)`: AES解密
  - `encryptString(plaintext, key)`: 加密字符串
  - `decryptString(encryptedBase64, key)`: 解密字符串

- **哈希运算：**
  - `hash(data)`: SHA-256哈希
  - `generateHMAC(data, key)`: 生成HMAC
  - `verifyHMAC(data, signature, key)`: 验证HMAC

- **安全工具：**
  - `detectDebugging()`: 检测调试模式
  - `secureStorage(key)`: 创建安全存储对象
  - `generateRandomBytes(length)`: 生成随机字节
  - `generateRandomString(length)`: 生成随机字符串

**使用示例：**

```javascript
// 加密数据
var encrypted = await CryptoModule.encryptString('sensitive data');
var decrypted = await CryptoModule.decryptString(encrypted);

// 生成签名
var signature = await CryptoModule.generateHMAC('data', 'secret key');

// 安全存储
var storage = CryptoModule.secureStorage('myKey');
await storage.set({ token: 'abc123' });
var value = await storage.get();
```

### 4. 环境检测模块 (core/environment-detector-core.js)

**功能：**
- 检测浏览器环境特征
- 识别自动化工具（Selenium、Puppeteer、Playwright等）
- 检测代理、VPN等网络环境
- 计算风险评分

**主要检测方法：**

- **自动化工具检测：**
  - `detectHeadless()`: 检测无头浏览器
  - `detectWebDriver()`: 检测WebDriver
  - `detectPuppeteer()`: 检测Puppeteer
  - `detectPlaywright()`: 检测Playwright
  - `detectSelenium()`: 检测Selenium

- **环境特征检测：**
  - `detectCanvas()`: 检测Canvas指纹
  - `detectWebGL()`: 检测WebGL
  - `detectWebGL2()`: 检测WebGL2
  - `detectAudio()`: 检测AudioContext
  - `detectFonts()`: 检测字体列表
  - `detectScreen()`: 检测屏幕信息
  - `detectTimezone()`: 检测时区
  - `detectLanguages()`: 检测语言设置

- **网络环境检测：**
  - `detectWebRTCIP()`: 检测WebRTC泄露的IP
  - `detectConnection()`: 检测网络连接类型
  - `detectAdBlock()`: 检测广告拦截

- **其他检测：**
  - `detectBattery()`: 检测电池状态
  - `detectMediaDevices()`: 检测媒体设备
  - `detectIframe()`: 检测iframe嵌入
  - `detectNotification()`: 检测通知权限

**使用示例：**

```javascript
// 创建检测器实例
var detector = new EnvironmentDetectorCore({
    apiBase: '/api/v1',
    chainCount: 12
});

// 运行所有检测
var result = await detector.runAll();

// 获取风险评分
console.log('Risk score:', result.risk_score);

// 获取指纹
console.log('Fingerprint:', result.fingerprint);

// 运行特定检测链
var chainResult = await detector.runChain();
```

### 5. 验证码核心模块 (core/captcha-core.js)

**功能：**
- 提供多种验证码实现
- 包括滑块验证码、点选验证码、语音验证码

**主要类：**

- **SliderCaptcha**: 滑块验证码
- **ClickCaptcha**: 点选验证码
- **VoiceCaptcha**: 语音验证码

**使用示例：**

```javascript
// 创建滑块验证码
var sliderCaptcha = CaptchaCore.createSliderCaptcha({
    container: '#slider-captcha',
    apiBase: '/api/v1',
    onSuccess: function(data) {
        console.log('验证成功', data);
    },
    onError: function(data) {
        console.log('验证失败', data);
    }
});

// 创建点选验证码
var clickCaptcha = CaptchaCore.createClickCaptcha({
    container: '#click-captcha',
    requiredClicks: 4,
    onSuccess: function(data) {
        console.log('验证成功', data);
    }
});

// 创建语音验证码
var voiceCaptcha = CaptchaCore.createVoiceCaptcha({
    container: '#voice-captcha',
    onSuccess: function() {
        console.log('验证成功');
    }
});

// 刷新验证码
sliderCaptcha.refresh();

// 重置验证码
sliderCaptcha.reset();
```

### 6. UI组件模块 (components/ui-components.js)

**功能：**
- 提供通用UI组件
- 包括Toast提示、加载动画、表单状态等

**主要功能：**

- **样式注入：**
  - `injectBaseStyles()`: 注入基础样式
  - `injectAnimationStyles()`: 注入动画样式

- **提示组件：**
  - `createToast(message, type, options)`: 创建Toast提示

- **加载组件：**
  - `showLoading(container, message)`: 显示加载动画
  - `hideLoading(overlay)`: 隐藏加载动画

- **状态管理：**
  - `setElementState(element, state)`: 设置元素状态
  - `addAccessibilityAttributes(element, type)`: 添加无障碍属性

**使用示例：**

```javascript
// 初始化UI模块
UIModule.injectBaseStyles();

// 显示成功提示
UIModule.createToast('操作成功', 'success');

// 显示错误提示
UIModule.createToast('发生错误', 'error', {
    duration: 8000
});

// 显示加载动画
var overlay = UIModule.showLoading(document.getElementById('container'), '加载中...');

// 隐藏加载动画
UIModule.hideLoading(overlay);

// 设置按钮状态
UIModule.setElementState(button, 'loading');
UIModule.setElementState(button, 'success');
UIModule.setElementState(button, 'error');
UIModule.setElementState(button, 'reset');
```

### 7. 模块入口 (index.js)

**功能：**
- 统一管理所有模块
- 提供初始化和加载接口

**主要功能：**

- **初始化：**
  - `CaptchaModules.init()`: 初始化所有已加载的模块
  - `CaptchaModules.quickInit(callback)`: 快速初始化

- **模块加载：**
  - `CaptchaModules.loadAll(callback)`: 加载所有模块
  - `CaptchaModules.loadModule(modulePath, callback)`: 加载单个模块

- **实例创建：**
  - `CaptchaModules.createInstance(options)`: 创建验证码实例

**使用示例：**

```javascript
// 方式一：快速初始化
CaptchaModules.quickInit(function(err) {
    if (err) {
        console.error('初始化失败', err);
        return;
    }
    console.log('初始化成功');
});

// 方式二：手动加载
CaptchaModules.loadAll(function(err) {
    if (err) return;
    // 使用模块
    var instance = CaptchaModules.createInstance({
        apiBase: '/api/v1'
    });
});

// 方式三：创建验证码实例
var captcha = CaptchaModules.createInstance({
    apiBase: '/api/v1',
    captchaType: 'slider'
});

// 使用实例的验证功能
var verification = await captcha.verify({ userInput: 'xxx' });
```

## 加载顺序

模块加载顺序非常重要，需要按照以下依赖顺序：

1. `constants/constants.js` - 常量定义（无依赖）
2. `utils/utils.js` - 工具函数（无依赖）
3. `core/crypto-module.js` - 加密模块（无依赖）
4. `core/environment-detector-core.js` - 环境检测（无依赖）
5. `core/captcha-core.js` - 验证码核心（依赖CryptoModule）
6. `components/ui-components.js` - UI组件（无依赖）
7. `index.js` - 入口文件（依赖以上所有）

## HTML中使用

```html
<!-- 按顺序加载模块 -->
<script src="/static/js/constants/constants.js"></script>
<script src="/static/js/utils/utils.js"></script>
<script src="/static/js/core/crypto-module.js"></script>
<script src="/static/js/core/environment-detector-core.js"></script>
<script src="/static/js/core/captcha-core.js"></script>
<script src="/static/js/components/ui-components.js"></script>
<script src="/static/js/index.js"></script>

<!-- 或者使用入口文件自动加载 -->
<script src="/static/js/index.js"></script>
<script>
    CaptchaModules.quickInit(function(err) {
        if (err) {
            console.error('加载失败', err);
            return;
        }

        // 创建验证码实例
        var captcha = CaptchaCore.createSliderCaptcha({
            container: '#captcha-container',
            onSuccess: function(data) {
                UIModule.createToast('验证成功', 'success');
            }
        });
    });
</script>
```

## 命名规范

- **文件命名：** 使用kebab-case（小写加连字符）
  - `captcha-core.js`
  - `crypto-module.js`
  - `environment-detector-core.js`

- **类命名：** 使用PascalCase（大写字母开头）
  - `class EnvironmentDetector`
  - `class SliderCaptcha`
  - `class ClickCaptcha`

- **函数命名：** 使用camelCase（小写字母开头）
  - `generateUUID()`
  - `detectWebDriver()`
  - `createToast()`

- **常量命名：** 使用UPPER_SNAKE_CASE（全大写下划线分隔）
  - `API_BASE`
  - `MAX_ATTEMPTS`
  - `DEFAULT_KEY`

- **私有变量：** 使用下划线前缀
  - `_isDragging`
  - `_sliderPosition`

## 代码风格

- 使用 `'use strict'` 严格模式
- 使用 IIFE 包装模块
- 使用 JSDoc 风格注释
- 统一的缩进（4空格）
- 分号结尾
- 使用 `var` 声明变量（兼容性考虑）

## 注意事项

1. **浏览器兼容性：**
   - 模块使用了ES6+语法（如async/await）
   - 加密模块依赖Web Crypto API
   - 环境检测依赖多种浏览器API

2. **性能考虑：**
   - 某些检测方法可能较慢，应在需要时调用
   - 使用 `runChain()` 可以只运行部分检测
   - Canvas、WebGL等检测会消耗资源

3. **安全问题：**
   - 默认加密密钥仅用于演示，生产环境应使用安全的密钥管理
   - 不要在前端代码中硬编码敏感信息
   - 环境检测结果仅供参考，不应作为唯一的安全依据

4. **维护建议：**
   - 新增功能时，优先考虑在现有模块中扩展
   - 避免在不同模块中重复相同的代码
   - 保持模块间的低耦合

## 更新日志

### v2.0.0 (2024-05-18)
- 完成模块化重构
- 将单体文件拆分为多个职责明确的模块
- 新增常量模块集中管理配置
- 新增工具函数模块提供通用功能
- 重构加密模块，提供更简洁的API
- 重构环境检测模块，支持链式调用
- 新增UI组件模块提供通用界面元素
- 提供统一的模块入口文件
