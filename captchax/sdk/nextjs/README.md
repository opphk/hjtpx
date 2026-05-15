# CaptchaX Next.js Integration

CaptchaX 的官方 Next.js App Router 集成包，提供 Server Components、Client Components 和中间件支持。

## 特性

- ✅ **Server Components** - 服务端验证和数据获取
- ✅ **Client Components** - 交互式验证码组件
- ✅ **中间件支持** - 路由保护和 API 验证
- ✅ **TypeScript** - 完整的类型定义
- ✅ **多种验证码类型** - 滑块、点选、拼图、旋转、文字、图标

## 安装

```bash
npm install @captchax/nextjs
```

## 快速开始

### 1. 配置环境变量

创建 `.env.local` 文件：

```bash
NEXT_PUBLIC_CAPTCHA_API_KEY=your_api_key
NEXT_PUBLIC_CAPTCHA_SERVER_URL=https://api.captchax.com
CAPTCHA_API_KEY=your_api_key
CAPTCHA_API_SECRET=your_api_secret
CAPTCHA_SERVER_URL=https://api.captchax.com
```

### 2. 添加 Provider

在 `app/providers.tsx` 中：

```typescript
'use client';

import { CaptchaProvider } from '@captchax/nextjs';

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <CaptchaProvider 
      apiKey={process.env.NEXT_PUBLIC_CAPTCHA_API_KEY!}
      serverUrl={process.env.NEXT_PUBLIC_CAPTCHA_SERVER_URL}
    >
      {children}
    </CaptchaProvider>
  );
}
```

### 3. 在页面中使用

#### 基础按钮验证

```typescript
'use client';

import { CaptchaButton } from '@captchax/nextjs';

export default function HomePage() {
  const handleSuccess = (token: string) => {
    console.log('Verified:', token);
  };
  
  return (
    <CaptchaButton 
      scene="login"
      onSuccess={handleSuccess}
      text="点击验证"
    />
  );
}
```

#### 服务端验证

```typescript
import { verifyCaptcha } from '@captchax/nextjs/server';

export default async function LoginPage({
  searchParams
}: {
  searchParams: { token?: string }
}) {
  if (searchParams.token) {
    const result = await verifyCaptcha(searchParams.token, {
      scene: 'login'
    });
    
    if (result.success) {
      console.log('Verified on server:', result.score);
    }
  }
  
  return <div>登录页面</div>;
}
```

### 4. 中间件保护

创建 `middleware.ts`：

```typescript
import { captchaMiddleware } from '@captchax/nextjs/middleware';

export default captchaMiddleware;

export const config = {
  matcher: ['/login/:path*', '/register/:path*']
};
```

## API 参考

### 组件

#### CaptchaProvider
- `apiKey` - API 密钥（必填）
- `serverUrl` - 服务器地址（可选，默认 https://api.captchax.com）

#### CaptchaButton
- `scene` - 场景标识
- `onSuccess` - 验证成功回调
- `onError` - 验证失败回调
- `text` - 按钮文本
- `disabled` - 是否禁用

#### CaptchaDialog
- `open` - 是否打开
- `onClose` - 关闭回调
- `onSuccess` - 验证成功回调
- `scene` - 场景标识
- `type` - 验证码类型

#### CaptchaSlider
- `onSuccess` - 验证成功回调
- `scene` - 场景标识

### Hooks

#### useCaptcha
```typescript
const { token, loading, error, verify, reset } = useCaptcha({
  scene: 'login',
  onSuccess: (token) => console.log(token)
});
```

#### useCaptchaVerify
```typescript
const { token, loading, error, verify, reset, isVerified } = useCaptchaVerify({
  scene: 'login',
  apiKey: 'your_api_key'
});
```

### 服务端

#### CaptchaXServer
```typescript
import { CaptchaXServer } from '@captchax/nextjs/server';

const client = new CaptchaXServer({
  apiKey: 'your_api_key',
  apiSecret: 'your_api_secret',
  serverUrl: 'https://api.captchax.com'
});

const result = await client.verify({
  token: 'user_token',
  scene: 'login',
  ip: 'user_ip',
  userAgent: 'user_agent'
});
```

## 验证码类型

| 类型 | 说明 |
|------|------|
| slider | 滑块验证码 |
| click | 点选验证码 |
| puzzle | 拼图验证码 |
| rotate | 旋转验证码 |
| text | 文字验证码 |
| icon | 图标验证码 |

## 示例项目

参考 `examples/app-router-example` 目录下的完整示例。

## 开发

```bash
# 安装依赖
npm install

# 构建
npm run build

# 测试
npm test

# 类型检查
npm run lint
```

## 许可证

MIT License
