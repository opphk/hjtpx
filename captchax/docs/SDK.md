# CaptchaX SDK 接入文档

## 概述

CaptchaX SDK 提供简单易用的前端接入方式，支持滑块验证、点选验证和拼图验证三种模式。通过 SDK，您可以快速将行为验证码集成到您的 Web 应用中。

## 快速接入

### 1. 引入 SDK

#### 方式一：CDN 引入（推荐）

```html
<script src="https://your-captchax-server.com/static/captchax.js"></script>
```

#### 方式二：下载本地

从 Release 页面下载 `captchax.min.js`，然后引入：

```html
<script src="/path/to/captchax.min.js"></script>
```

### 2. HTML 结构

```html
<!-- 验证码容器 -->
<div id="captcha-container"></div>

<!-- 表单 -->
<form id="login-form">
  <input type="text" name="username" required>
  <input type="password" name="password" required>
  <button type="submit">登录</button>
</form>
```

### 3. 初始化并使用

```javascript
// 初始化 CaptchaX
const captcha = new CaptchaX({
  appId: 'your-app-id',
  serverUrl: 'https://your-captchax-server.com',
  type: 'slider', // 可选: slider, click, puzzle
  container: '#captcha-container',

  onSuccess: function(result) {
    console.log('验证成功:', result);
    // 在此处提交表单，附带 token
  },

  onError: function(error) {
    console.error('验证失败:', error);
  },

  onReady: function() {
    console.log('验证码已准备好');
  }
});

// 渲染验证码
captcha.render();
```

---

## 配置选项

### 全局配置

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| appId | string | - | 应用标识（必填） |
| serverUrl | string | - | CaptchaX 服务器地址（必填） |
| type | string | 'slider' | 验证类型：slider / click / puzzle |
| container | string | - | 验证码容器选择器或元素 |
| width | number | 300 | 验证组件宽度 |
| height | number | 200 | 验证组件高度 |
| lang | string | 'zh-CN' | 语言：zh-CN / en |
| theme | string | 'light' | 主题：light / dark |
| timeout | number | 10000 | 请求超时时间（毫秒） |
| retryTimes | number | 3 | 重试次数 |

### 回调函数

| 参数 | 说明 |
|------|------|
| onSuccess | 验证成功回调，参数为验证结果对象 |
| onError | 验证失败回调，参数为错误信息 |
| onReady | 验证码加载完成回调 |
| onClose | 用户关闭验证码回调 |
| onRefresh | 用户刷新验证码回调 |

---

## 验证类型详解

### 滑块验证 (Slider)

用户通过拖动滑块到正确位置完成验证。

```javascript
const captcha = new CaptchaX({
  appId: 'my-app',
  serverUrl: 'https://captchax.example.com',
  type: 'slider',
  container: '#slider-container',

  onSuccess: function(result) {
    // result = { token: 'xxx', captchaId: 'xxx' }
    submitForm(result.token);
  }
});

captcha.render();
```

### 点选验证 (Click)

用户需要按正确顺序点击指定字符。

```javascript
const captcha = new CaptchaX({
  appId: 'my-app',
  serverUrl: 'https://captchax.example.com',
  type: 'click',
  container: '#click-container',
  charCount: 4, // 需要点击的字符数量

  onSuccess: function(result) {
    submitForm(result.token);
  }
});

captcha.render();
```

### 拼图验证 (Puzzle)

用户将拼图块拖动到正确位置。

```javascript
const captcha = new CaptchaX({
  appId: 'my-app',
  serverUrl: 'https://captchax.example.com',
  type: 'puzzle',
  container: '#puzzle-container',

  onSuccess: function(result) {
    submitForm(result.token);
  }
});

captcha.render();
```

---

## 完整示例

### 登录表单集成

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <title>登录示例</title>
  <script src="https://captchax.example.com/static/captchax.js"></script>
  <style>
    body {
      font-family: Arial, sans-serif;
      display: flex;
      justify-content: center;
      align-items: center;
      min-height: 100vh;
      background: #f5f5f5;
    }

    .login-container {
      background: white;
      padding: 30px;
      border-radius: 8px;
      box-shadow: 0 2px 10px rgba(0,0,0,0.1);
      width: 350px;
    }

    .form-group {
      margin-bottom: 15px;
    }

    .form-group label {
      display: block;
      margin-bottom: 5px;
      font-weight: 500;
    }

    .form-group input {
      width: 100%;
      padding: 10px;
      border: 1px solid #ddd;
      border-radius: 4px;
      box-sizing: border-box;
    }

    .btn {
      width: 100%;
      padding: 12px;
      background: #1890ff;
      color: white;
      border: none;
      border-radius: 4px;
      cursor: pointer;
      font-size: 16px;
    }

    .btn:disabled {
      background: #ccc;
      cursor: not-allowed;
    }

    #captcha-container {
      margin-bottom: 15px;
    }

    .error {
      color: #ff4d4f;
      font-size: 12px;
      margin-top: 5px;
    }
  </style>
</head>
<body>
  <div class="login-container">
    <h2 style="text-align:center;margin-bottom:20px;">用户登录</h2>

    <form id="login-form">
      <div class="form-group">
        <label>用户名</label>
        <input type="text" name="username" required>
      </div>

      <div class="form-group">
        <label>密码</label>
        <input type="password" name="password" required>
      </div>

      <div id="captcha-container"></div>

      <button type="submit" class="btn" id="submit-btn" disabled>登录</button>
    </form>
  </div>

  <script>
    let captchaToken = null;

    const captcha = new CaptchaX({
      appId: 'login-app',
      serverUrl: 'https://captchax.example.com',
      container: '#captcha-container',

      onSuccess: function(result) {
        captchaToken = result.token;
        document.getElementById('submit-btn').disabled = false;
        console.log('验证成功，token:', captchaToken);
      },

      onError: function(error) {
        console.error('验证失败:', error);
        document.getElementById('submit-btn').disabled = true;
        captchaToken = null;
      },

      onReady: function() {
        console.log('验证码已就绪');
      }
    });

    captcha.render();

    document.getElementById('login-form').addEventListener('submit', function(e) {
      e.preventDefault();

      if (!captchaToken) {
        alert('请先完成验证');
        return;
      }

      const formData = new FormData(this);
      const data = Object.fromEntries(formData);
      data.captchaToken = captchaToken;

      console.log('提交数据:', data);

      // 发送登录请求到后端
      fetch('/api/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(data)
      })
      .then(res => res.json())
      .then(result => {
        if (result.success) {
          alert('登录成功');
          location.href = '/dashboard';
        } else {
          alert('登录失败: ' + result.message);
          captcha.reset();
        }
      })
      .catch(err => {
        console.error('请求失败:', err);
        alert('网络错误');
      });
    });
  </script>
</body>
</html>
```

---

## 后端验证

前端获取 token 后，需要在后端验证 token 的有效性。

### 验证流程

1. 前端完成验证，获取 token
2. 前端将 token 伴随表单提交到后端
3. 后端调用 CaptchaX 验证接口确认 token 有效
4. 验证通过后处理业务逻辑

### 后端验证示例

```python
# Python Flask 示例
from flask import Flask, request, jsonify
import httpx

app = Flask(__name__)

@app.route('/api/login', methods=['POST'])
def login():
    data = request.json
    captcha_token = data.get('captchaToken')

    if not captcha_token:
        return jsonify({'success': False, 'message': '请先完成验证'}), 400

    # 验证 captcha token
    is_valid = verify_captcha_token(captcha_token)
    if not is_valid:
        return jsonify({'success': False, 'message': '验证失败'}), 400

    # 验证通过，处理登录逻辑
    username = data.get('username')
    password = data.get('password')

    # ... 验证用户名密码 ...

    return jsonify({'success': True, 'message': '登录成功'})

def verify_captcha_token(token):
    # 调用 CaptchaX 验证接口
    # 这里需要实现 token 验证逻辑
    # 可以缓存验证结果避免重复请求
    return True
```

```javascript
// Node.js Express 示例
const express = require('express');
const app = express();

app.post('/api/login', async (req, res) => {
  const { username, password, captchaToken } = req.body;

  if (!captchaToken) {
    return res.status(400).json({ success: false, message: '请先完成验证' });
  }

  // 验证 captcha token
  const isValid = await verifyCaptchaToken(captchaToken);
  if (!isValid) {
    return res.status(400).json({ success: false, message: '验证失败' });
  }

  // 处理登录逻辑
  // ...

  res.json({ success: true, message: '登录成功' });
});

async function verifyCaptchaToken(token) {
  // 实现 token 验证逻辑
  return true;
}
```

---

## 方法 API

CaptchaX 实例提供以下方法：

### render()

渲染验证码到页面。

```javascript
captcha.render();
```

### reset()

重置验证码，重新生成。

```javascript
captcha.reset();
```

### destroy()

销毁验证码实例。

```javascript
captcha.destroy();
```

### show()

显示验证码。

```javascript
captcha.show();
```

### hide()

隐藏验证码。

```javascript
captcha.hide();
```

### verify()

手动触发验证。

```javascript
captcha.verify().then(result => {
  console.log(result);
});
```

---

## 事件监听

```javascript
const captcha = new CaptchaX({ ... });

captcha.on('ready', function() {
  console.log('验证码已就绪');
});

captcha.on('success', function(result) {
  console.log('验证成功', result);
});

captcha.on('error', function(error) {
  console.log('验证失败', error);
});

captcha.on('close', function() {
  console.log('验证码已关闭');
});

captcha.on('refresh', function() {
  console.log('验证码已刷新');
});

captcha.render();
```

---

## 样式定制

通过 CSS 自定义验证码样式：

```css
/* 自定义容器样式 */
.captchax-container {
  border-radius: 8px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
}

/* 自定义滑块样式 */
.captchax-slider {
  background: #f0f0f0;
}

/* 自定义成功状态 */
.captchax-success {
  background: #52c41a;
}

/* 自定义错误状态 */
.captchax-error {
  background: #ff4d4f;
}
```

---

## 常见问题

### Q: 验证码加载缓慢？

A: 检查网络连接，确保 serverUrl 配置正确。可以使用 CDN 加速静态资源。

### Q: 验证总是失败？

A: 检查：
1. 验证码是否过期（默认5分钟）
2. 服务端是否正常运行
3. 坐标容差是否设置合理

### Q: 如何处理移动端适配？

A: CaptchaX 自动适配移动端。确保容器宽度足够（建议至少 280px）。

### Q: 如何实现无感知验证？

A: 可以在页面加载完成后预加载验证码，但暂时隐藏，等用户提交时再显示。

### Q: 验证码图片显示异常？

A: 检查浏览器是否支持 Base64 图片格式，或配置服务端返回完整图片 URL。

---

## 浏览器兼容性

- Chrome 50+
- Firefox 45+
- Safari 11+
- Edge 79+
- IE 不支持（需要 polyfill）
