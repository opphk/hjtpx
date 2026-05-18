#!/usr/bin/env ruby
require_relative '../lib/hjtpx/captcha'
require 'json'

BASE_URL = ENV['CAPTCHA_BASE_URL'] || 'https://captcha.example.com'
API_KEY = ENV['CAPTCHA_API_KEY'] || 'your_api_key'
SECRET_KEY = ENV['CAPTCHA_SECRET_KEY'] || 'your_secret_key'

class CaptchaService
  def initialize(base_url:, api_key: nil, secret_key: nil)
    @client = Hjtpx::Captcha::Client.new(
      base_url: base_url,
      api_key: api_key,
      secret_key: secret_key
    )
  end

  def generate_slider_captcha(user_id:)
    response = @client.slider.get(width: 320, height: 160, tolerance: 8)
    {
      session_id: response.session_id,
      image_url: response.image_url,
      puzzle_url: response.puzzle_url,
      user_id: user_id,
      created_at: Time.now.to_i
    }
  end

  def verify_slider_captcha(session_id:, x:, y: nil, user_id: nil)
    begin
      result = @client.slider.verify(
        session_id: session_id,
        x: x,
        y: y,
        trajectory: generate_mock_trajectory
      )

      if result.success
        {
          success: true,
          message: 'Verification successful',
          remaining_attempts: result.remaining_attempts,
          risk_score: result.risk_score
        }
      else
        {
          success: false,
          message: result.message,
          remaining_attempts: result.remaining_attempts,
          fail_reason: result.fail_reason
        }
      end
    rescue Hjtpx::Captcha::Exceptions::SessionExpiredError => e
      {
        success: false,
        message: 'Session expired, please refresh captcha',
        error: 'session_expired'
      }
    rescue Hjtpx::Captcha::Exceptions::RateLimitError => e
      {
        success: false,
        message: 'Too many attempts, please wait',
        retry_after: e.retry_after,
        error: 'rate_limited'
      }
    end
  end

  def generate_click_captcha(user_id:, mode: 'number')
    response = @client.click.get(mode: mode, max_points: 4)
    {
      session_id: response.session_id,
      image_url: response.image_url,
      hint: response.hint,
      hint_order: response.hint_order,
      user_id: user_id,
      created_at: Time.now.to_i
    }
  end

  def verify_click_captcha(session_id:, points:, click_sequence: nil)
    begin
      result = @client.click.verify(
        session_id: session_id,
        points: points,
        click_sequence: click_sequence
      )

      {
        success: result.success,
        message: result.message,
        remaining_attempts: result.remaining_attempts
      }
    rescue Hjtpx::Captcha::Exceptions::ValidationError => e
      {
        success: false,
        message: e.message,
        error: 'invalid_points'
      }
    end
  end

  def user_login_flow(username:, password:)
    captcha_token = nil

    begin
      result = @client.auth.login(
        username: username,
        password: password,
        captcha_token: captcha_token
      )

      {
        success: true,
        access_token: result.access_token,
        refresh_token: result.refresh_token,
        expires_in: result.expires_in
      }
    rescue Hjtpx::Captcha::Exceptions::AuthenticationError => e
      {
        success: false,
        message: 'Invalid credentials',
        error: 'auth_failed'
      }
    rescue Hjtpx::Captcha::Exceptions::CaptchaError => e
      {
        success: false,
        message: e.message,
        error: 'captcha_required'
      }
    end
  end

  def detect_environment
    script = @client.env.get_detection_script
    puts "Detection script loaded: #{script.length} bytes"

    detection_data = {
      userAgent: 'Mozilla/5.0 (Test Browser)',
      platform: 'linux',
      language: 'zh-CN',
      screenWidth: 1920,
      screenHeight: 1080,
      timezone: 'Asia/Shanghai',
      plugins: [],
      canvasFingerprint: 'mock_fingerprint_12345',
      webglFingerprint: 'mock_webgl_fingerprint'
    }

    result = @client.env.check_environment(data: detection_data)
    result
  end

  private

  def generate_mock_trajectory
    points = []
    time = 0
    x = 0
    y = 80

    50.times do
      x += rand(5..15)
      y += rand(-2..2)
      time += rand(10..30)
      points << { x: x, y: y, t: time }
    end

    points
  end

  def close
    @client.close
  end
end

if __FILE__ == $PROGRAM_NAME
  service = CaptchaService.new(
    base_url: BASE_URL,
    api_key: API_KEY,
    secret_key: SECRET_KEY
  )

  puts "=" * 60
  puts "Real-world Integration Examples"
  puts "=" * 60

  puts "\n1. Generating slider captcha for user..."
  slider_captcha = service.generate_slider_captcha(user_id: 'user_123')
  puts "Slider captcha generated:"
  puts JSON.pretty_generate(slider_captcha)

  puts "\n2. Verifying slider captcha..."
  verify_result = service.verify_slider_captcha(
    session_id: slider_captcha[:session_id],
    x: 150,
    user_id: 'user_123'
  )
  puts "Verification result:"
  puts JSON.pretty_generate(verify_result)

  puts "\n3. Generating click captcha..."
  click_captcha = service.generate_click_captcha(
    user_id: 'user_456',
    mode: 'number'
  )
  puts "Click captcha generated:"
  puts JSON.pretty_generate(click_captcha)

  puts "\n4. Verifying click captcha..."
  click_verify = service.verify_click_captcha(
    session_id: click_captcha[:session_id],
    points: [[100, 50], [200, 100], [150, 150], [80, 120]]
  )
  puts "Verification result:"
  puts JSON.pretty_generate(click_verify)

  puts "\n5. Detecting user environment..."
  env_result = service.detect_environment
  puts "Environment check completed"

  service.close

  puts "\n" + "=" * 60
  puts "Examples completed"
  puts "=" * 60
end
