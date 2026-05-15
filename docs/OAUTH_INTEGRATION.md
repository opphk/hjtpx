# OAuth 2.0 集成文档

## 概述

本文档描述了 HJTPX 项目中 OAuth 2.0 授权服务器的实现细节和使用方法。

## OAuth 2.0 概念

### 什么是 OAuth 2.0

OAuth 2.0 是一个开放标准授权协议，允许用户授权第三方应用访问其在某个网站上的资源，而无需提供用户名和密码。

### 授权流程类型

本项目实现了以下授权流程：

#### 1. 授权码流程 (Authorization Code Flow)

最安全的 OAuth 2.0 流程，适用于有后端的应用程序。

流程图：
```
┌─────────┐                                     ┌─────────────┐
│  用户   │                                     │   客户端    │
└────┬────┘                                     └──────┬──────┘
     │                                                │
     │  1. 点击登录                                    │
     │ ───────────────────────────────────────────────>
     │                                                │
     │  2. 重定向到授权服务器                           │
     │ <───────────────────────────────────────────────
     │                                                │
     │  3. 用户同意授权                                 │
     │ ───────────────────────────────────────────────>
     │                                                │
     │  4. 返回授权码                                   │
     │ <───────────────────────────────────────────────
     │                                                │
     │  5. 使用授权码换取令牌                           │
     │ ────────────────────────────────────────────────
     │                                                │
     │  6. 返回访问令牌                                 │
     │ <───────────────────────────────────────────────
```

#### 2. PKCE 流程 (Proof Key for Code Exchange)

适用于移动应用和单页应用，防止授权码被拦截攻击。

## 端点说明

### 授权端点

**URL**: `GET /oauth/authorize`

**参数**:
| 参数 | 必需 | 说明 |
|------|------|------|
| response_type | 是 | 值必须为 `code` |
| client_id | 是 | 客户端标识 |
| redirect_uri | 是 | 回调地址 |
| scope | 否 | 授权范围，默认 `openid profile email` |
| state | 建议 | 防止 CSRF 攻击的随机字符串 |
| code_challenge | 否 | PKCE 挑战 (使用 S256 方法) |
| code_challenge_method | 否 | 必须是 `S256` |
| nonce | 否 | 用于 ID Token 防重放 |

**响应**: 成功则重定向到 `redirect_uri?code=xxx&state=xxx`

### 令牌端点

**URL**: `POST /oauth/token`

**授权码模式参数**:
```json
{
  "grant_type": "authorization_code",
  "code": "授权码",
  "redirect_uri": "回调地址",
  "client_id": "客户端ID",
  "code_verifier": "PKCE验证器"
}
```

**刷新令牌模式参数**:
```json
{
  "grant_type": "refresh_token",
  "refresh_token": "刷新令牌"
}
```

**客户端凭证模式参数**:
```json
{
  "grant_type": "client_credentials",
  "client_id": "客户端ID",
  "client_secret": "客户端密钥"
}
```

**响应示例**:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "scope": "openid profile email"
}
```

### 撤销端点

**URL**: `POST /oauth/revoke`

**参数**:
```json
{
  "token": "要撤销的令牌",
  "token_type_hint": "access_token 或 refresh_token"
}
```

### 令牌检查端点

**URL**: `POST /oauth/introspect`

**参数**:
```json
{
  "token": "要检查的令牌",
  "token_type_hint": "access_token 或 refresh_token"
}
```

**响应示例**:
```json
{
  "active": true,
  "token_type": "access_token",
  "sub": "user123",
  "scope": "openid profile email",
  "exp": 1625123456
}
```

### 用户信息端点

**URL**: `GET /oauth/userinfo`

**请求头**: `Authorization: Bearer <access_token>`

**响应示例**:
```json
{
  "sub": "user123",
  "email": "user@example.com",
  "name": "张三",
  "role": "user"
}
```

### OpenID Connect 发现端点

**URL**: `GET /.well-known/openid-configuration`

返回 OpenID Connect 提供者配置信息。

## 第三方登录

### GitHub 登录

**初始化 URL**: `GET /auth/github`

**回调 URL**: `GET /auth/github/callback`

**配置环境变量**:
```bash
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GITHUB_CALLBACK_URL=https://your-domain.com/auth/github/callback
```

### Google 登录

**初始化 URL**: `GET /auth/google`

**回调 URL**: `GET /auth/google/callback`

**配置环境变量**:
```bash
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
GOOGLE_CALLBACK_URL=https://your-domain.com/auth/google/callback
```

## PKCE 实现

### 代码验证器生成

```javascript
const PKCE = require('./oauth/pkce');

// 生成随机验证码
const codeVerifier = PKCE.generateCodeVerifier(128);

// 生成挑战
const codeChallenge = await PKCE.generateCodeChallenge(codeVerifier);
```

### 验证流程

1. 客户端生成 `code_verifier` (43-128字符的随机字符串)
2. 客户端使用 S256 方法计算 `code_challenge`
3. 授权请求中发送 `code_challenge` 和 `code_challenge_method=S256`
4. 令牌请求中发送原始 `code_verifier`
5. 服务器验证 `code_verifier` 与 `code_challenge` 匹配

## 令牌管理

### 访问令牌

- 默认过期时间: 1小时
- 用于访问受保护资源
- 包含用户信息和权限范围

### 刷新令牌

- 默认过期时间: 7天
- 用于获取新的访问令牌
- 支持令牌轮换

### 令牌撤销

- 支持撤销访问令牌和刷新令牌
- 撤销后，令牌将被加入黑名单
- 撤销操作会记录到数据库

## 安全最佳实践

### 1. 使用 PKCE

所有公共客户端（移动应用、单页应用）都必须使用 PKCE。

### 2. 验证重定向 URI

只允许预注册的重定向 URI。

### 3. 使用 HTTPS

生产环境必须使用 HTTPS。

### 4. 令牌存储

- 访问令牌存储在内存中
- 刷新令牌使用安全存储（如 Keychain）
- 避免 XSS 攻击

### 5. CSRF 防护

使用 `state` 参数防止 CSRF 攻击。

## 错误处理

### OAuth 错误码

| 错误码 | 说明 |
|--------|------|
| invalid_request | 请求参数缺失或无效 |
| invalid_client | 客户端认证失败 |
| invalid_grant | 授权码或刷新令牌无效 |
| unauthorized_client | 客户端无权使用此授权类型 |
| unsupported_grant_type | 不支持的授权类型 |
| invalid_scope | 请求的权限范围无效 |

### 错误响应格式

```json
{
  "error": "invalid_request",
  "error_description": "code_verifier is required for PKCE flow"
}
```

## 环境变量配置

```bash
# JWT 配置
JWT_SECRET=your-jwt-secret-key
JWT_EXPIRES_IN=1h
REFRESH_TOKEN_EXPIRES_IN=7d

# OAuth 配置
OAUTH_CLIENT_ID=hjtpx-client
OAUTH_CLIENT_SECRET=your-client-secret
OAUTH_ISSUER=https://your-domain.com
OAUTH_BASE_URL=https://your-domain.com
OAUTH_REDIRECT_URIS=https://your-app.com/callback

# GitHub OAuth
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
GITHUB_CALLBACK_URL=https://your-domain.com/auth/github/callback

# Google OAuth
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GOOGLE_CALLBACK_URL=https://your-domain.com/auth/google/callback

# 成功登录后重定向
OAUTH_SUCCESS_REDIRECT_URL=/dashboard
```

## 数据库表

### oauth_token_revocations

用于记录令牌撤销历史：

```sql
CREATE TABLE oauth_token_revocations (
  id SERIAL PRIMARY KEY,
  token_jti VARCHAR(255) UNIQUE NOT NULL,
  token_type VARCHAR(50) NOT NULL,
  revoked_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth_revocations_jti ON oauth_token_revocations(token_jti);
```

### users 表扩展

需要为 `users` 表添加以下字段：

```sql
ALTER TABLE users ADD COLUMN oauth_provider VARCHAR(50);
ALTER TABLE users ADD COLUMN oauth_provider_id VARCHAR(255);
ALTER TABLE users ADD COLUMN oauth_token TEXT;
ALTER TABLE users ADD COLUMN oauth_refresh_token TEXT;
ALTER TABLE users ADD COLUMN oauth_token_expires TIMESTAMP;
```

## 示例代码

### 授权请求示例

```javascript
// 使用 PKCE 的授权请求
const codeVerifier = PKCE.generateCodeVerifier();
const codeChallenge = await PKCE.generateCodeChallenge(codeVerifier);
const state = crypto.randomBytes(16).toString('hex');

// 存储 code_verifier 以便后续使用
sessionStorage.setItem('code_verifier', codeVerifier);

const authUrl = new URL('https://your-domain.com/oauth/authorize');
authUrl.searchParams.set('response_type', 'code');
authUrl.searchParams.set('client_id', 'your-client-id');
authUrl.searchParams.set('redirect_uri', 'https://your-app.com/callback');
authUrl.searchParams.set('scope', 'openid profile email');
authUrl.searchParams.set('state', state);
authUrl.searchParams.set('code_challenge', codeChallenge);
authUrl.searchParams.set('code_challenge_method', 'S256');

window.location.href = authUrl.toString();
```

### 令牌交换示例

```javascript
// 令牌交换请求
const codeVerifier = sessionStorage.getItem('code_verifier');

const response = await fetch('https://your-domain.com/oauth/token', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    grant_type: 'authorization_code',
    code: urlParams.get('code'),
    redirect_uri: 'https://your-app.com/callback',
    client_id: 'your-client-id',
    code_verifier: codeVerifier
  })
});

const tokens = await response.json();
// 存储令牌
localStorage.setItem('access_token', tokens.access_token);
localStorage.setItem('refresh_token', tokens.refresh_token);
```

### 刷新令牌示例

```javascript
async function refreshAccessToken() {
  const refreshToken = localStorage.getItem('refresh_token');

  const response = await fetch('https://your-domain.com/oauth/token', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      grant_type: 'refresh_token',
      refresh_token: refreshToken
    })
  });

  const tokens = await response.json();
  localStorage.setItem('access_token', tokens.access_token);
  localStorage.setItem('refresh_token', tokens.refresh_token);

  return tokens;
}
```

## 故障排除

### 常见问题

1. **授权码已过期**
   - 授权码有效期为10分钟
   - 需要重新发起授权请求

2. **PKCE 验证失败**
   - 确保使用相同的 `code_verifier`
   - 确认 `code_challenge_method` 为 `S256`

3. **令牌验证失败**
   - 检查令牌是否已过期
   - 确认令牌是否已被撤销

4. **重定向 URI 不匹配**
   - 确认 `redirect_uri` 与注册时完全一致

## 参考资料

- [OAuth 2.0 RFC 6749](https://tools.ietf.org/html/rfc6749)
- [PKCE RFC 7636](https://tools.ietf.org/html/rfc7636)
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [GitHub OAuth Apps](https://docs.github.com/en/developers/apps/authorizing-oauth-apps)
- [Google OAuth 2.0](https://developers.google.com/identity/protocols/oauth2)
