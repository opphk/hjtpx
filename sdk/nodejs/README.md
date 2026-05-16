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

##### verifyCaptcha

Verify a captcha.

```typescript
async verifyCaptcha(request: VerifyCaptchaRequest): Promise<VerifyCaptchaResponse>
```

##### authLogin

Authenticate a user.

```typescript
async authLogin(request: LoginRequest): Promise<LoginResponse>
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

## Examples

See the `examples` directory for more examples.

## License

MIT
