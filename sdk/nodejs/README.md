# HJTpx Node.js SDK

Node.js SDK for the HJTpx captcha verification system.

## Installation

```bash
npm install hjtpx-sdk
```

## Quick Start

```javascript
import { CaptchaClient } from 'hjtpx-sdk';

// Create a client
const client = new CaptchaClient({
  baseUrl: 'http://localhost:8080',
  apiKey: 'your-api-key', // optional
  timeout: 30000,
});

// Get a slider captcha
const captcha = await client.getSliderCaptcha({
  width: 360,
  height: 220,
});

// Verify the captcha
const result = await client.verifyCaptcha({
  session_id: captcha.session_id,
  type: 'slider',
  x: captcha.target_x,
  y: captcha.target_y,
});

console.log('Verification result:', result);
```

## Features

- Multiple captcha types: slider, click, gesture
- Connection pooling for better performance
- Automatic retry mechanism with exponential backoff
- Comprehensive error handling
- TypeScript support
- Async/await API

## API Reference

### CaptchaClient

#### Constructor

```typescript
new CaptchaClient(config: CaptchaClientConfig)
```

**Parameters:**
- `baseUrl` (string, required): The base URL of the HJTpx API
- `apiKey` (string, optional): API key for authentication
- `timeout` (number, optional): Request timeout in milliseconds (default: 30000)
- `maxConnections` (number, optional): Maximum number of concurrent connections (default: 100)
- `retryConfig` (RetryConfig, optional): Retry configuration

#### Methods

##### getSliderCaptcha

Get a slider captcha.

```typescript
async getSliderCaptcha(options?: {
  width?: number;
  height?: number;
  tolerance?: number;
}): Promise<SliderCaptchaResponse>
```

##### getClickCaptcha

Get a click captcha.

```typescript
async getClickCaptcha(options?: {
  mode?: 'number' | 'letter' | 'chinese' | 'mixed' | 'icon';
  shuffle?: boolean;
  points?: number;
}): Promise<ClickCaptchaResponse>
```

##### getGestureCaptcha

Get a gesture captcha.

```typescript
async getGestureCaptcha(): Promise<GestureCaptchaResponse>
```

##### verifyCaptcha

Verify a captcha.

```typescript
async verifyCaptcha(request: VerifyCaptchaRequest): Promise<VerifyCaptchaResponse>
```

##### verifyGestureCaptcha

Verify a gesture captcha.

```typescript
async verifyGestureCaptcha(session_id: string, pattern: number[]): Promise<VerifyCaptchaResponse>
```

##### authLogin

Authenticate a user.

```typescript
async authLogin(request: LoginRequest): Promise<LoginResponse>
```

##### authRegister

Register a new user.

```typescript
async authRegister(request: {
  username: string;
  email: string;
  password: string;
  behavior_data?: string;
}): Promise<UserResponse>
```

##### authRefreshToken

Refresh authentication token.

```typescript
async authRefreshToken(refreshToken: string): Promise<TokenResponse>
```

##### authLogout

Logout current user.

```typescript
async authLogout(): Promise<void>
```

##### getDetectionScript

Get environment detection script.

```typescript
async getDetectionScript(callback?: string): Promise<string>
```

##### submitDetection

Submit detection data.

```typescript
async submitDetection(data: Record<string, unknown>): Promise<DetectionResult>
```

##### checkEnvironment

Check environment security.

```typescript
async checkEnvironment(data: Record<string, unknown>): Promise<EnvironmentResult>
```

##### close

Close the client and release resources.

```typescript
async close(): Promise<void>
```

## Error Handling

The SDK provides specific error classes for different types of errors:

- `CaptchaError`: Base error class
- `ValidationError`: Invalid request parameters
- `AuthenticationError`: Authentication failed
- `NotFoundError`: Resource not found
- `RateLimitError`: Rate limit exceeded
- `ServerError`: Server-side error
- `NetworkError`: Network-related error

```javascript
import { CaptchaClient, RateLimitError } from 'hjtpx-sdk';

try {
  const captcha = await client.getSliderCaptcha();
} catch (error) {
  if (error instanceof RateLimitError) {
    console.log('Rate limit exceeded, retry after:', error.retryAfter);
  } else {
    console.error('Error:', error);
  }
}
```

## Complete Examples

### Slider Captcha Example

```javascript
import { CaptchaClient } from 'hjtpx-sdk';

const client = new CaptchaClient({
  baseUrl: 'http://localhost:8080',
  apiKey: 'your-api-key',
});

// Get slider captcha
const captcha = await client.getSliderCaptcha({
  width: 360,
  height: 200,
  tolerance: 8,
});

console.log('Session ID:', captcha.session_id);
console.log('Image URL:', captcha.image_url);

// User slides to position 150
const result = await client.verifyCaptcha({
  session_id: captcha.session_id,
  type: 'slider',
  x: 150,
  y: captcha.target_y,
});

if (result.success) {
  console.log('Verification passed!');
} else {
  console.log('Verification failed:', result.message);
}

await client.close();
```

### Click Captcha Example

```javascript
import { CaptchaClient } from 'hjtpx-sdk';

const client = new CaptchaClient({
  baseUrl: 'http://localhost:8080',
});

// Get click captcha
const captcha = await client.getClickCaptcha({
  mode: 'number',
  points: 3,
});

console.log('Session ID:', captcha.session_id);
console.log('Hint:', captcha.hint);

// User clicks in correct positions
const result = await client.verifyCaptcha({
  session_id: captcha.session_id,
  type: 'click',
  points: [[100, 100], [200, 100], [150, 200]],
});

console.log('Verification result:', result.success);

await client.close();
```

### Gesture Captcha Example

```javascript
import { CaptchaClient } from 'hjtpx-sdk';

const client = new CaptchaClient({
  baseUrl: 'http://localhost:8080',
});

// Get gesture captcha
const captcha = await client.getGestureCaptcha();

console.log('Session ID:', captcha.session_id);
console.log('Grid Size:', captcha.grid_size);

// User draws gesture pattern (e.g., 1-2-3-4)
const result = await client.verifyGestureCaptcha(
  captcha.session_id,
  [1, 2, 3, 4]
);

console.log('Verification result:', result.success);

await client.close();
```

### User Authentication Example

```javascript
import { CaptchaClient } from 'hjtpx-sdk';

const client = new CaptchaClient({
  baseUrl: 'http://localhost:8080',
});

// Register new user
const registerResult = await client.authRegister({
  username: 'newuser',
  email: 'user@example.com',
  password: 'securepassword',
});

// Login
const loginResult = await client.authLogin({
  username: 'newuser',
  password: 'securepassword',
});

console.log('Access Token:', loginResult.access_token);
console.log('User:', loginResult.user);

// Refresh token
const refreshResult = await client.authRefreshToken(
  loginResult.refresh_token
);

// Logout
await client.authLogout();

await client.close();
```

### Environment Detection Example

```javascript
import { CaptchaClient } from 'hjtpx-sdk';

const client = new CaptchaClient({
  baseUrl: 'http://localhost:8080',
});

// Get detection script for frontend
const script = await client.getDetectionScript('onDetectionComplete');
console.log('Detection script:', script);

// Submit detection data from frontend
const submitResult = await client.submitDetection({
  fingerprint: 'browser-fingerprint-hash',
  canvas_hash: 'canvas-fingerprint',
  webgl_vendor: 'WebGL Vendor',
  timezone: 'Asia/Shanghai',
  language: 'zh-CN',
});

// Check environment security
const checkResult = await client.checkEnvironment({
  fingerprint: 'browser-fingerprint-hash',
  risk_score: 0.1,
});

console.log('Environment check:', checkResult);

await client.close();
```

## Retry Configuration

```typescript
import { CaptchaClient, RetryConfig } from 'hjtpx-sdk';

const retryConfig: RetryConfig = {
  maxRetries: 5,        // Maximum number of retries
  initialDelayMs: 100,  // Initial delay in milliseconds
  maxDelayMs: 5000,     // Maximum delay cap
};

const client = new CaptchaClient({
  baseUrl: 'http://localhost:8080',
  retryConfig,
});
```

## Connection Pool Configuration

```typescript
const client = new CaptchaClient({
  baseUrl: 'http://localhost:8080',
  maxConnections: 50,   // Maximum concurrent connections
  timeout: 30000,       // Request timeout in milliseconds
});
```

## Examples

See the `examples` directory for more examples.

## License

MIT
