# HJTPX Captcha SDK for Rust

Rust SDK for HJTPX Captcha Service.

## Features

- Support for multiple captcha types: Slider, Click, Image, Rotation, Gesture, Jigsaw, Voice
- Built-in retry mechanism with exponential backoff
- Connection pooling and timeout configuration
- HMAC signature support for API authentication
- Comprehensive error handling

## Installation

Add this to your Cargo.toml:

```toml
[dependencies]
hjtpx-captcha-sdk = "1.0.0"
```

## Usage

### Basic Example

```rust
use hjtpx_captcha_sdk::CaptchaClient;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Create client
    let mut client = CaptchaClient::new("http://localhost:8080")
        .with_api_key("your-api-key")
        .with_timeout(std::time::Duration::from_secs(30));

    // Get slider captcha
    let slider = client.get_slider_captcha(Some(320), Some(160), Some(8)).await?;
    println!("Session ID: {}", slider.session_id);

    // Verify slider captcha
    let result = client.verify_slider_captcha(&slider.session_id, 150, None, None).await?;
    println!("Success: {}", result.success);

    Ok(())
}
```

### Client Configuration

```rust
let client = CaptchaClient::new("http://localhost:8080")
    .with_api_key("your-api-key")
    .with_app_credentials("app-id", "app-secret")
    .with_timeout(std::time::Duration::from_secs(30))
    .with_retries(3, std::time::Duration::from_millis(100));
```

### Authentication

```rust
let mut client = CaptchaClient::new("http://localhost:8080");

// Login
let response = client.login("username", "password", None).await?;
println!("Access Token: {}", response.access_token);

// Use authenticated client
let result = client.get_slider_captcha(None, None, None).await?;

// Logout
client.logout().await?;
```

## Supported Captcha Types

1. **Slider Captcha** - Slider verification
2. **Click Captcha** - Click verification
3. **Image Captcha** - Character/image verification
4. **Rotation Captcha** - Rotation verification
5. **Gesture Captcha** - Pattern gesture verification
6. **Jigsaw Captcha** - Jigsaw puzzle verification
7. **Voice Captcha** - Audio verification

## API Reference

### Captcha Methods

- `get_slider_captcha(width, height, tolerance)` - Get slider captcha
- `get_click_captcha(mode, max_points, allow_shuffle)` - Get click captcha
- `get_image_captcha(type, count)` - Get image captcha
- `get_rotation_captcha()` - Get rotation captcha
- `get_gesture_captcha()` - Get gesture captcha
- `get_jigsaw_captcha(width, height, grid_size)` - Get jigsaw captcha
- `get_voice_captcha(language)` - Get voice captcha

### Verification Methods

- `verify_slider_captcha(session_id, x, y, trajectory)` - Verify slider
- `verify_click_captcha(session_id, points, click_sequence)` - Verify click
- `verify_image_captcha(challenge_id, answer)` - Verify image
- `verify_rotation_captcha(challenge_id, angle)` - Verify rotation
- `verify_gesture_captcha(session_id, pattern)` - Verify gesture
- `verify_jigsaw_captcha(session_id, pieces)` - Verify jigsaw

### Auth Methods

- `login(username, password, captcha_token)` - User login
- `logout()` - User logout
- `register(username, email, password, behavior_data)` - User registration
- `refresh_token(refresh_token)` - Refresh access token

### Detection Methods

- `get_detection_script(callback)` - Get environment detection script
- `submit_detection(data)` - Submit detection data
- `check_environment(data)` - Check environment

## Error Handling

The SDK returns `Result<T, CaptchaError>` where `CaptchaError` can be:

- `CaptchaError::Network` - Network errors
- `CaptchaError::Timeout` - Request timeout
- `CaptchaError::ApiError` - API errors with code and message
- `CaptchaError::Authentication` - Authentication failures
- `CaptchaError::RateLimited` - Rate limiting
- `CaptchaError::CaptchaExpired` - Captcha expired
- `CaptchaError::VerificationFailed` - Verification failed
- `CaptchaError::InvalidParameter` - Invalid parameters

## License

MIT
