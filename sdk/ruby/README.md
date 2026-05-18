# HJTPX Captcha SDK for Ruby

Ruby SDK for HJTPX Captcha service with support for multiple captcha types.

## Installation

Add this to your Gemfile:

```ruby
gem 'hjtpx-captcha', '~> 1.0.0'
```

Or install directly:

```bash
gem install hjtpx-captcha
```

## Requirements

- Ruby 2.7.0 or higher

## Quick Start

```ruby
require 'hjtpx/captcha'

client = Hjtpx::Captcha::Client.new(
  base_url: 'https://your-captcha-server.com',
  api_key: 'your_api_key',
  secret_key: 'your_secret_key'
)

slider_captcha = client.slider.get(width: 320, height: 160)
puts "Session ID: #{slider_captcha.session_id}"
puts "Image URL: #{slider_captcha.image_url}"

result = client.slider.verify(session_id: slider_captcha.session_id, x: 150)
puts "Success: #{result.success}"
```

## Supported Captcha Types

- **Slider Captcha**: Drag-and-drop slider puzzle
- **Click Captcha**: Click on specified positions in order
- **Image Captcha**: Traditional image-based verification
- **Rotation Captcha**: Rotate image to correct angle
- **Gesture Captcha**: Draw gesture patterns
- **Jigsaw Captcha**: Complete jigsaw puzzles
- **Voice Captcha**: Audio-based verification
- **Connect Captcha**: Connect dots in correct order
- **3D Captcha**: 3D object selection

## Configuration

### Client Configuration

```ruby
client = Hjtpx::Captcha::Client.new(
  base_url: 'https://your-captcha-server.com',
  api_key: 'your_api_key',
  secret_key: 'your_secret_key',
  timeout: 30
)
```

### Connection Pool Configuration

```ruby
pool_config = Hjtpx::Captcha::Pool::PoolConfig.new(
  pool_size: 10,
  max_pool_size: 20,
  connection_timeout: 10,
  read_timeout: 30,
  max_retries: 3,
  retry_backoff_factor: 0.5
)

client = Hjtpx::Captcha::Client.new(
  base_url: 'https://your-captcha-server.com',
  pool_config: pool_config
)
```

### Retry Configuration

```ruby
retry_config = Hjtpx::Captcha::Retry::RetryConfig.new(
  max_attempts: 3,
  backoff_factor: 0.5,
  max_backoff: 30,
  retry_status_codes: [429, 500, 502, 503, 504]
)

client = Hjtpx::Captcha::Client.new(
  base_url: 'https://your-captcha-server.com',
  retry_config: retry_config
)
```

## API Reference

### Slider Captcha

```ruby
response = client.slider.get(width: 320, height: 160, tolerance: 8)
result = client.slider.verify(
  session_id: response.session_id,
  x: 150,
  y: 80,
  trajectory: [
    { x: 10, y: 50, t: 100 },
    { x: 20, y: 50, t: 110 }
  ]
)
```

### Click Captcha

```ruby
response = client.click.get(mode: 'number', max_points: 4)
result = client.click.verify(
  session_id: response.session_id,
  points: [[100, 50], [200, 100], [150, 150]],
  click_sequence: [1, 2, 3]
)
```

### Image Captcha

```ruby
response = client.image.get(type: 'number', count: 4)
result = client.image.verify(
  challenge_id: response.challenge_id,
  answer: '1234'
)
```

### Rotation Captcha

```ruby
response = client.rotation.get
result = client.rotation.verify(
  challenge_id: response.challenge_id,
  angle: 45
)
```

### Gesture Captcha

```ruby
response = client.gesture.get
result = client.gesture.verify(
  session_id: response.session_id,
  pattern: [0, 1, 2, 3, 4, 5]
)
```

### Jigsaw Captcha

```ruby
response = client.jigsaw.get(width: 300, height: 300, grid_size: 3)
result = client.jigsaw.verify(
  session_id: response.session_id,
  pieces: [
    { index: 0, originalX: 0, originalY: 0 },
    { index: 1, originalX: 100, originalY: 0 }
  ]
)
```

### Voice Captcha

```ruby
response = client.voice.get(language: 'zh-CN')
result = client.voice.verify(
  session_id: response.session_id,
  answer: '1234'
)
```

### User Authentication

```ruby
result = client.auth.login(
  username: 'user@example.com',
  password: 'password123',
  captcha_token: 'optional_captcha_token'
)
puts "Access Token: #{result.access_token}"

client.auth.logout
```

### Environment Detection

```ruby
script = client.env.get_detection_script
result = client.env.check_environment(data: {
  userAgent: '...',
  platform: 'linux',
  canvasFingerprint: '...'
})
```

## Error Handling

```ruby
begin
  result = client.slider.verify(session_id: '...', x: 100)
rescue Hjtpx::Captcha::Exceptions::ApiError => e
  puts "API Error: #{e.message} (code: #{e.code})"
rescue Hjtpx::Captcha::Exceptions::NetworkError => e
  puts "Network Error: #{e.message}"
rescue Hjtpx::Captcha::Exceptions::TimeoutError => e
  puts "Request timed out"
rescue Hjtpx::Captcha::Exceptions::ValidationError => e
  puts "Validation failed: #{e.message}"
rescue Hjtpx::Captcha::Exceptions::AuthenticationError => e
  puts "Authentication failed: #{e.message}"
rescue Hjtpx::Captcha::Exceptions::SessionExpiredError => e
  puts "Session expired, please refresh captcha"
rescue Hjtpx::Captcha::Exceptions::RateLimitError => e
  puts "Rate limited, retry after #{e.retry_after} seconds"
end
```

## HMAC Signature Verification

```ruby
client = Hjtpx::Captcha::Client.new(
  base_url: 'https://your-captcha-server.com',
  secret_key: 'your_secret_key'
)

signer = Hjtpx::Captcha::Signer::HmacSigner.new('your_secret_key')
signature = signer.sign('data_to_sign')
verified = signer.verify('data_to_sign', signature)
```

## Context Manager Support

```ruby
Hjtpx::Captcha::Client.open(
  base_url: 'https://your-captcha-server.com',
  api_key: 'your_api_key'
) do |client|
  response = client.slider.get
  result = client.slider.verify(session_id: response.session_id, x: 150)
end
```

## Development

Clone the repository and install dependencies:

```bash
cd sdk/ruby
bundle install
```

Run tests:

```bash
bundle exec rspec
```

Run examples:

```bash
ruby examples/basic_example.rb
ruby examples/integration_example.rb
```

## License

MIT License
