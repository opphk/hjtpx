# hjtpx JavaScript/TypeScript SDK

A comprehensive JavaScript/TypeScript SDK for hjtpx captcha services, providing support for image captcha, slider captcha, and click captcha with advanced features like automatic retries, connection pooling, and detailed error handling.

## Features

- **TypeScript Support**: Full TypeScript support with complete type definitions
- **Multiple Captcha Types**: Support for image captcha, slider captcha, and click captcha
- **Automatic Retries**: Built-in retry mechanism with exponential backoff for failed requests
- **Comprehensive Error Handling**: Detailed error types and error code extraction utilities
- **Statistics & Monitoring**: Track request statistics including success rate, retry count, and error tracking
- **Configurable**: Extensive configuration options for timeouts, pool sizes, and retry behavior
- **Universal**: Works in Node.js and modern browsers

## Installation

### npm

```bash
npm install hjtpx-sdk
```

### yarn

```bash
yarn add hjtpx-sdk
```

### pnpm

```bash
pnpm add hjtpx-sdk
```

### Bun

```bash
bun add hjtpx-sdk
```

## Quick Start

### TypeScript

```typescript
import { CaptchaClient, CaptchaType } from 'hjtpx-sdk';

const client = new CaptchaClient({
  baseUrl: 'http://localhost:8080',
  appId: 'your-app-id',
  appSecret: 'your-app-secret',
});

async function main() {
  const captcha = await client.generateImageCaptcha({
    type: CaptchaType.NUMBER,
    count: 4,
  });
  console.log('Challenge ID:', captcha.challenge_id);

  const result = await client.verifyImageCaptcha(captcha.challenge_id, '1234');
  console.log('Success:', result.success);
}

main().catch(console.error);
```

### JavaScript

```javascript
const { CaptchaClient, CaptchaType } = require('hjtpx-sdk');

const client = new CaptchaClient({
  baseUrl: 'http://localhost:8080',
});

async function main() {
  const captcha = await client.generateImageCaptcha({
    type: CaptchaType.NUMBER,
    count: 4,
  });
  console.log('Challenge ID:', captcha.challenge_id);

  const result = await client.verifyImageCaptcha(captcha.challenge_id, '1234');
  console.log('Success:', result.success);
}

main().catch(console.error);
```

## Configuration

### Basic Configuration

```typescript
import { CaptchaClient } from 'hjtpx-sdk';

const client = new CaptchaClient({
  baseUrl: 'http://localhost:8080',
  appId: 'your-app-id',
  appSecret: 'your-app-secret',
  timeout: 30000,
  maxRetries: 3,
  retryDelay: 100,
  debugMode: false,
});
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| baseUrl | string | http://localhost:8080 | API endpoint |
| appId | string | | Application ID |
| appSecret | string | | Application Secret |
| timeout | number | 30000 | Request timeout in milliseconds |
| maxRetries | number | 3 | Maximum retry attempts |
| retryDelay | number | 100 | Base retry delay in milliseconds |
| maxIdleConns | number | 10 | Maximum idle connections |
| maxOpenConns | number | 100 | Maximum open connections |
| debugMode | boolean | false | Enable debug logging |

## API Reference

### CaptchaClient

The main client for interacting with the captcha service.

#### Constructor

```typescript
const client = new CaptchaClient(options?: SDKOptions);
```

#### Captcha Generation Methods

##### generateImageCaptcha

```typescript
async generateImageCaptcha(request?: ImageCaptchaRequest): Promise<ImageCaptchaResponse>
```

Generates a new image captcha challenge.

```typescript
const captcha = await client.generateImageCaptcha({
  type: CaptchaType.NUMBER,  // number, letter, or mixed
  count: 4,                   // Number of characters (4-6)
  noiseMode: 2,                // Noise level (0-10)
  lineMode: 1,                 // Line interference level (0-10)
});
```

##### generateSliderCaptcha

```typescript
async generateSliderCaptcha(request?: SliderCaptchaRequest): Promise<SliderCaptchaResponse>
```

Generates a new slider captcha challenge.

```typescript
const slider = await client.generateSliderCaptcha({
  width: 360,
  height: 220,
});
```

##### generateClickCaptcha

```typescript
async generateClickCaptcha(request?: ClickCaptchaRequest): Promise<ClickCaptchaResponse>
```

Generates a new click captcha challenge.

```typescript
const click = await client.generateClickCaptcha({
  width: 360,
  height: 220,
  iconCount: 4,
});
```

#### Captcha Verification Methods

##### verifyImageCaptcha

```typescript
async verifyImageCaptcha(challengeId: string, answer: string): Promise<VerifyImageCaptchaResponse>
```

Verifies an image captcha answer.

```typescript
const result = await client.verifyImageCaptcha(captcha.challenge_id, '1234');
console.log('Success:', result.success);
```

##### verifySliderCaptcha

```typescript
async verifySliderCaptcha(challengeId: string, offset: string): Promise<VerifyCaptchaResponse>
```

Verifies a slider captcha solution.

```typescript
const result = await client.verifySliderCaptcha(slider.challenge_id, '120');
console.log('Success:', result.success);
console.log('Risk Score:', result.score);
```

##### verifyClickCaptcha

```typescript
async verifyClickCaptcha(challengeId: string, clicks: ClickData[]): Promise<VerifyCaptchaResponse>
```

Verifies a click captcha solution.

```typescript
const clicks: ClickData[] = [
  { x: 100, y: 120, duration: 500 },
  { x: 200, y: 150, duration: 300 },
];

const result = await client.verifyClickCaptcha(click.challenge_id, clicks);
console.log('Success:', result.success);
```

#### Utility Methods

##### extractBase64Image

```typescript
extractBase64Image(dataUri: string): Buffer | Uint8Array
```

Extracts raw image bytes from a base64 data URI.

```typescript
const imageData = client.extractBase64Image(captcha.image);
```

##### getStats

```typescript
getStats(): PoolStats
```

Returns current pool statistics.

```typescript
const stats = client.getStats();
console.log('Total Requests:', stats.totalRequests);
console.log('Success Rate:', stats.successRate.toFixed(2) + '%');
```

#### Runtime Configuration

```typescript
client.setDebugMode(true);
client.setTimeout(60000);
client.setMaxRetries(5);
client.setRetryDelay(200);
```

## Error Handling

The SDK provides comprehensive error handling with specific exception types:

### Error Types

```typescript
import { 
  SDKError, 
  NetworkError, 
  TimeoutError, 
  InvalidParamsError, 
  RateLimitedError, 
  UnauthorizedError, 
  ServerError,
  isSDKError,
  getErrorCode,
} from 'hjtpx-sdk';
```

### Example

```typescript
import { CaptchaClient, SDKError, RateLimitedError } from 'hjtpx-sdk';

const client = new CaptchaClient();

try {
  const captcha = await client.generateImageCaptcha();
  const result = await client.verifyImageCaptcha(captcha.challenge_id, '1234');
} catch (error) {
  if (error instanceof RateLimitedError) {
    console.log('Rate limited, retry after', error.retryAfter, 'seconds');
  } else if (error instanceof SDKError) {
    console.log('SDK Error', error.code, ':', error.message);
  } else {
    console.log('Unexpected error:', error);
  }
}
```

### Error Checking

```typescript
try {
  const result = await client.verifyImageCaptcha(challengeId, answer);
} catch (error) {
  if (isSDKError(error)) {
    if (error.isRateLimited()) {
      console.log('Rate limited');
    } else if (error.isUnauthorized()) {
      console.log('Unauthorized');
    } else if (error.isServerError()) {
      console.log('Server error');
    } else if (error.isInvalidParams()) {
      console.log('Invalid parameters');
    }
  }
}
```

## Complete Examples

### Image Captcha Example

```typescript
import { CaptchaClient, CaptchaType } from 'hjtpx-sdk';
import * as fs from 'fs';

async function main() {
  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
  });

  const captcha = await client.generateImageCaptcha({
    type: CaptchaType.MIXED,
    count: 4,
    noiseMode: 2,
    lineMode: 1,
  });
  console.log('Challenge ID:', captcha.challenge_id);

  const imageData = client.extractBase64Image(captcha.image);
  fs.writeFileSync('captcha.png', imageData);
  console.log('Captcha image saved to captcha.png');

  const userInput = await getUserInput('Enter the captcha text: ');
  const result = await client.verifyImageCaptcha(captcha.challenge_id, userInput);
  console.log('Verification result:', result.success);
}

async function getUserInput(prompt: string): Promise<string> {
  return new Promise((resolve) => {
    process.stdout.write(prompt);
    process.stdin.once('data', (data) => {
      resolve(data.toString().trim());
    });
  });
}

main().catch(console.error);
```

### Slider Captcha Example

```typescript
import { CaptchaClient } from 'hjtpx-sdk';
import * as fs from 'fs';

async function main() {
  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
  });

  const slider = await client.generateSliderCaptcha({
    width: 360,
    height: 220,
  });
  console.log('Challenge ID:', slider.challenge_id);

  const bgData = client.extractBase64Image(slider.background_image);
  const sliderData = client.extractBase64Image(slider.slider_image);

  fs.writeFileSync('bg.png', bgData);
  fs.writeFileSync('slider.png', sliderData);

  const offset = await getUserInput('Enter slider offset: ');
  const result = await client.verifySliderCaptcha(slider.challenge_id, offset);
  console.log('Verification result:', result.success);
  console.log('Risk Score:', result.score);
}

async function getUserInput(prompt: string): Promise<string> {
  return new Promise((resolve) => {
    process.stdout.write(prompt);
    process.stdin.once('data', (data) => {
      resolve(data.toString().trim());
    });
  });
}

main().catch(console.error);
```

### Click Captcha Example

```typescript
import { CaptchaClient, ClickData } from 'hjtpx-sdk';
import * as fs from 'fs';

async function main() {
  const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
  });

  const click = await client.generateClickCaptcha({
    width: 360,
    height: 220,
    iconCount: 4,
  });
  console.log('Challenge ID:', click.challenge_id);
  console.log('Target Index:', click.target_index);

  const imageData = client.extractBase64Image(click.background_image);
  fs.writeFileSync('click_captcha.png', imageData);

  const clicks: ClickData[] = [];
  const targetPos = click.target_position;
  clicks.push(new ClickData(targetPos[0], targetPos[1], 500));

  const result = await client.verifyClickCaptcha(click.challenge_id, clicks);
  console.log('Verification result:', result.success);
}

main().catch(console.error);
```

## Browser Usage

The SDK works in modern browsers with ES modules:

```html
<!DOCTYPE html>
<html>
<head>
  <title>hjtpx Captcha Demo</title>
</head>
<body>
  <h1>hjtpx Captcha Demo</h1>
  <div id="captcha-container"></div>
  <button id="verify-btn">Verify</button>

  <script type="module">
    import { CaptchaClient, CaptchaType } from 'https://unpkg.com/hjtpx-sdk@1.0.0/dist/index.mjs';

    const client = new CaptchaClient({
      baseUrl: 'http://localhost:8080',
    });

    async function init() {
      const captcha = await client.generateImageCaptcha({
        type: CaptchaType.MIXED,
        count: 4,
      });

      const container = document.getElementById('captcha-container');
      container.innerHTML = `
        <img src="${captcha.image}" alt="Captcha" />
        <p>Challenge ID: ${captcha.challenge_id}</p>
      `;

      document.getElementById('verify-btn').addEventListener('click', async () => {
        const answer = prompt('Enter the captcha text:');
        if (answer) {
          const result = await client.verifyImageCaptcha(captcha.challenge_id, answer);
          alert(result.success ? 'Success!' : 'Failed!');
        }
      });
    }

    init().catch(console.error);
  </script>
</body>
</html>
```

## Testing

Run tests with Vitest:

```bash
npm test
```

Run tests in watch mode:

```bash
npm run test:watch
```

## Building

Build the SDK:

```bash
npm run build
```

## TypeScript Configuration

The SDK includes complete TypeScript definitions. No additional setup is required.

```json
{
  "compilerOptions": {
    "moduleResolution": "bundler",
    "strict": true
  }
}
```

## License

MIT License
