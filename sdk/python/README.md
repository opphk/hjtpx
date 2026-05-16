# hjtpx Python SDK

A comprehensive Python SDK for hjtpx captcha services, providing support for image captcha, slider captcha, and click captcha with advanced features like automatic retries, connection pooling, and detailed error handling.

## Features

- **Multiple Captcha Types**: Support for image captcha, slider captcha, and click captcha
- **Automatic Retries**: Built-in retry mechanism with exponential backoff for failed requests
- **Comprehensive Error Handling**: Detailed error types and error code extraction utilities
- **Statistics & Monitoring**: Track request statistics including success rate, retry count, and error tracking
- **Thread-Safe**: Safe for concurrent use with proper connection management
- **Configurable**: Extensive configuration options for timeouts, pool sizes, and retry behavior
- **Python 3.7+ Support**: Compatible with Python 3.7, 3.8, 3.9, 3.10, 3.11, and 3.12

## Installation

### Using pip

```bash
pip install hjtpx
```

### From source

```bash
git clone https://github.com/hjtpx/hjtpx.git
cd hjtpx/sdk/python
pip install -e .
```

### Development installation

```bash
pip install -e ".[dev]"
```

## Quick Start

```python
from hjtpx import CaptchaClient, ImageCaptchaRequest, CaptchaType

# Create client
client = CaptchaClient(
    base_url="http://localhost:8080",
    app_id="your-app-id",
    app_secret="your-app-secret"
)

# Generate image captcha
captcha = client.generate_image_captcha(
    ImageCaptchaRequest(captcha_type=CaptchaType.NUMBER, count=4)
)
print(f"Challenge ID: {captcha.challenge_id}")

# Verify captcha
result = client.verify_image_captcha(captcha.challenge_id, "1234")
print(f"Success: {result.success}")
```

## Configuration

### Basic Configuration

```python
from hjtpx import CaptchaClient, Config

# Method 1: Using Config class
config = Config(
    base_url="http://localhost:8080",
    app_id="your-app-id",
    app_secret="your-app-secret",
    timeout=30.0,
    max_retries=3,
    retry_delay=0.1,
    debug_mode=False,
)
client = CaptchaClient(config)

# Method 2: Using convenience function
client = CaptchaClient(
    base_url="http://localhost:8080",
    app_id="your-app-id",
    app_secret="your-app-secret"
)
```

### Configuration Options

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| base_url | str | http://localhost:8080 | API endpoint |
| app_id | str | | Application ID |
| app_secret | str | | Application Secret |
| timeout | float | 30.0 | Request timeout in seconds |
| max_retries | int | 3 | Maximum retry attempts |
| retry_delay | float | 0.1 | Base retry delay in seconds |
| max_idle_conns | int | 10 | Maximum idle connections |
| max_open_conns | int | 100 | Maximum open connections |
| debug_mode | bool | False | Enable debug logging |

## API Reference

### CaptchaClient

The main client for interacting with the captcha service.

#### Constructor

```python
client = CaptchaClient(config: Optional[Config] = None)
```

Creates a new captcha client with the specified configuration. If `config` is None, default values will be used.

### Captcha Generation Methods

#### generate_image_captcha

```python
result = client.generate_image_captcha(request: Optional[ImageCaptchaRequest] = None) -> ImageCaptchaResponse
```

Generates a new image captcha challenge.

```python
from hjtpx import CaptchaClient, ImageCaptchaRequest, CaptchaType

client = CaptchaClient()

# With custom parameters
request = ImageCaptchaRequest(
    captcha_type=CaptchaType.NUMBER,  # number, letter, or mixed
    count=4,                            # Number of characters (4-6)
    noise_mode=0,                       # Noise level (0-10)
    line_mode=0,                        # Line interference level (0-10)
)
captcha = client.generate_image_captcha(request)
```

#### generate_slider_captcha

```python
result = client.generate_slider_captcha(request: Optional[SliderCaptchaRequest] = None) -> SliderCaptchaResponse
```

Generates a new slider captcha challenge.

```python
from hjtpx import CaptchaClient, SliderCaptchaRequest

client = CaptchaClient()

request = SliderCaptchaRequest(
    width=360,
    height=220,
)
slider = client.generate_slider_captcha(request)
```

#### generate_click_captcha

```python
result = client.generate_click_captcha(request: Optional[ClickCaptchaRequest] = None) -> ClickCaptchaResponse
```

Generates a new click captcha challenge.

```python
from hjtpx import CaptchaClient, ClickCaptchaRequest

client = CaptchaClient()

request = ClickCaptchaRequest(
    width=360,
    height=220,
    icon_count=4,
)
click = client.generate_click_captcha(request)
```

### Captcha Verification Methods

#### verify_image_captcha

```python
result = client.verify_image_captcha(challenge_id: str, answer: str) -> VerifyImageCaptchaResponse
```

Verifies an image captcha answer.

```python
result = client.verify_image_captcha(captcha.challenge_id, "1234")
print(f"Success: {result.success}")
```

#### verify_slider_captcha

```python
result = client.verify_slider_captcha(challenge_id: str, offset: str) -> VerifyCaptchaResponse
```

Verifies a slider captcha solution.

```python
result = client.verify_slider_captcha(slider.challenge_id, "120")
print(f"Success: {result.success}")
print(f"Risk Score: {result.score}")
```

#### verify_click_captcha

```python
result = client.verify_click_captcha(challenge_id: str, clicks: List[ClickData]) -> VerifyCaptchaResponse
```

Verifies a click captcha solution.

```python
from hjtpx import ClickData

clicks = [
    ClickData(x=100, y=120, duration=500),
    ClickData(x=200, y=150, duration=300),
]
result = client.verify_click_captcha(click.challenge_id, clicks)
```

### Utility Methods

#### extract_base64_image

```python
image_data = client.extract_base64_image(data_uri: str) -> bytes
```

Extracts raw image bytes from a base64 data URI.

```python
image_data = client.extract_base64_image(captcha.image)
with open("captcha.png", "wb") as f:
    f.write(image_data)
```

#### get_stats

```python
stats = client.get_stats() -> PoolStats
```

Returns current pool statistics.

```python
stats = client.get_stats()
print(f"Total Requests: {stats.total_requests}")
print(f"Success Rate: {stats.success_rate:.2f}%")
print(f"Failed Requests: {stats.failed_requests}")
```

### Runtime Configuration

```python
# Set debug mode
client.set_debug_mode(True)

# Set timeout
client.set_timeout(60.0)

# Set max retries
client.set_max_retries(5)

# Set retry delay
client.set_retry_delay(0.2)
```

## Error Handling

The SDK provides comprehensive error handling with specific exception types:

### Exception Types

```python
from hjtpx import (
    SDKError,
    NetworkError,
    TimeoutError,
    InvalidResponseError,
    ServerError,
    InvalidParamsError,
    VerificationFailedError,
    RateLimitedError,
    UnauthorizedError,
)
```

### Example

```python
from hjtpx import CaptchaClient, SDKError, RateLimitedError, TimeoutError

client = CaptchaClient()

try:
    captcha = client.generate_image_captcha()
except RateLimitedError as e:
    print(f"Rate limited, retry after {e.retry_after} seconds")
except TimeoutError as e:
    print(f"Request timed out: {e}")
except SDKError as e:
    print(f"SDK Error {e.code}: {e.message}")
except Exception as e:
    print(f"Unexpected error: {e}")
```

### Error Utilities

```python
from hjtpx import is_sdk_error, get_error_code

try:
    result = client.verify_image_captcha(challenge_id, answer)
except Exception as e:
    if is_sdk_error(e):
        code = get_error_code(e)
        print(f"SDK Error code: {code}")
```

## Complete Examples

### Image Captcha Example

```python
from hjtpx import CaptchaClient, ImageCaptchaRequest, CaptchaType

def main():
    client = CaptchaClient(base_url="http://localhost:8080")

    # Generate captcha
    request = ImageCaptchaRequest(
        captcha_type=CaptchaType.MIXED,
        count=4,
        noise_mode=2,
        line_mode=1,
    )
    captcha = client.generate_image_captcha(request)
    print(f"Challenge ID: {captcha.challenge_id}")

    # Save image
    image_data = client.extract_base64_image(captcha.image)
    with open("captcha.png", "wb") as f:
        f.write(image_data)
    print("Captcha image saved to captcha.png")

    # Verify
    user_input = input("Enter the captcha text: ")
    result = client.verify_image_captcha(captcha.challenge_id, user_input)
    print(f"Verification result: {result.success}")

if __name__ == "__main__":
    main()
```

### Slider Captcha Example

```python
from hjtpx import CaptchaClient, SliderCaptchaRequest

def main():
    client = CaptchaClient(base_url="http://localhost:8080")

    # Generate slider captcha
    request = SliderCaptchaRequest(width=360, height=220)
    slider = client.generate_slider_captcha(request)
    print(f"Challenge ID: {slider.challenge_id}")

    # Save images
    bg_data = client.extract_base64_image(slider.background_image)
    slider_data = client.extract_base64_image(slider.slider_image)

    with open("bg.png", "wb") as f:
        f.write(bg_data)
    with open("slider.png", "wb") as f:
        f.write(slider_data)

    # Simulate slider movement (in real app, track user interaction)
    offset = input("Enter slider offset: ")
    result = client.verify_slider_captcha(slider.challenge_id, offset)
    print(f"Verification result: {result.success}")
    print(f"Risk Score: {result.score}")

if __name__ == "__main__":
    main()
```

### Click Captcha Example

```python
from hjtpx import CaptchaClient, ClickCaptchaRequest, ClickData

def main():
    client = CaptchaClient(base_url="http://localhost:8080")

    # Generate click captcha
    request = ClickCaptchaRequest(width=360, height=220, icon_count=4)
    click = client.generate_click_captcha(request)
    print(f"Challenge ID: {click.challenge_id}")
    print(f"Target Index: {click.target_index}")

    # Save image
    image_data = client.extract_base64_image(click.background_image)
    with open("click_captcha.png", "wb") as f:
        f.write(image_data)

    # Get click position from user or tracking
    clicks = []
    # Example: user clicks on target position
    target_x, target_y = click.target_position[0], click.target_position[1]
    clicks.append(ClickData(x=target_x, y=target_y, duration=500))

    result = client.verify_click_captcha(click.challenge_id, clicks)
    print(f"Verification result: {result.success}")

if __name__ == "__main__":
    main()
```

## Testing

Run tests with pytest:

```bash
cd sdk/python
pip install -e ".[test]"
pytest tests/ -v
```

Run with coverage:

```bash
pytest tests/ -v --cov=hjtpx --cov-report=html
```

## Development

### Setup

```bash
cd sdk/python
pip install -e ".[dev]"
```

### Code Quality

```bash
# Format code
black hjtpx/

# Lint
flake8 hjtpx/

# Type check
mypy hjtpx/
```

## License

MIT License
