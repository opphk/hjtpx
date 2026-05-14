# CaptchaX 示例代码

本目录包含 CaptchaX 的完整使用示例，涵盖前端、后端和各种集成场景。

## 目录结构

```
examples/
├── frontend/              # 前端示例
│   ├── vanilla/          # 原生 JavaScript
│   ├── react/            # React 集成
│   ├── vue/              # Vue 集成
│   └── angular/           # Angular 集成
├── backend/              # 后端示例
│   ├── go/               # Go Gin 框架
│   ├── python/            # Python Flask/Django
│   ├── node/              # Node.js Express
│   ├── java/              # Java Spring Boot
│   └── php/               # PHP Laravel
└── integrations/          # 集成示例
    ├── wordpress/         # WordPress 集成
    ├── django/            # Django 集成
    └── laravel/           # Laravel 集成
```

---

## 前端示例

### 原生 JavaScript 集成

[vanilla/login.html](frontend/vanilla/login.html) - 基础登录表单集成示例

### React 集成

[frontend/react/LoginForm.jsx](frontend/react/LoginForm.jsx) - React 组件示例

### Vue 集成

[frontend/vue/useCaptcha.js](frontend/vue/useCaptcha.js) - Vue 3 Composable 示例

---

## 后端示例

### Go (Gin)

[backend/go/main.go](backend/go/main.go) - 完整后端服务示例

### Python (Flask)

[backend/python/flask_app.py](backend/python/flask_app.py) - Flask 集成示例

### Node.js (Express)

[backend/node/express_app.js](backend/node/express_app.js) - Express 集成示例

---

## 快速开始

### 1. 启动 CaptchaX 服务

```bash
cd /workspace/hjtpx/captchax
docker-compose up -d
```

### 2. 运行前端示例

```bash
# 启动任意 HTTP 服务器
cd examples/frontend/vanilla
python3 -m http.server 8000

# 访问 http://localhost:8000/login.html
```

### 3. 运行后端示例

```bash
# Go 示例
cd examples/backend/go
go run main.go

# Python 示例
cd examples/backend/python
pip install -r requirements.txt
python flask_app.py
```

---

## 示例列表

| 示例 | 语言/框架 | 说明 |
|------|-----------|------|
| [login.html](frontend/vanilla/login.html) | HTML/JS | 基础登录表单 |
| [LoginForm.jsx](frontend/react/LoginForm.jsx) | React | React 登录组件 |
| [useCaptcha.js](frontend/vue/useCaptcha.js) | Vue 3 | Vue Composable |
| [main.go](backend/go/main.go) | Go | Go 后端服务 |
| [flask_app.py](backend/python/flask_app.py) | Python | Flask 后端 |
| [express_app.js](backend/node/express_app.js) | Node.js | Express 后端 |
