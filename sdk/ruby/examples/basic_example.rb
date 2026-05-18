#!/usr/bin/env ruby
require_relative '../lib/hjtpx/captcha'

BASE_URL = ENV['CAPTCHA_BASE_URL'] || 'https://captcha.example.com'
API_KEY = ENV['CAPTCHA_API_KEY'] || 'your_api_key'
SECRET_KEY = ENV['CAPTCHA_SECRET_KEY'] || 'your_secret_key'

def print_section(title)
  puts "\n#{'=' * 60}"
  puts title
  puts '=' * 60
end

puts "HJTPX Captcha SDK Ruby Examples"
puts "Base URL: #{BASE_URL}"

client = Hjtpx::Captcha::Client.new(
  base_url: BASE_URL,
  api_key: API_KEY,
  secret_key: SECRET_KEY,
  timeout: 30
)

begin
  print_section("1. Slider Captcha Example")

  puts "\nFetching slider captcha..."
  slider_response = client.slider.get(width: 320, height: 160, tolerance: 8)
  puts "Session ID: #{slider_response.session_id}"
  puts "Image URL: #{slider_response.image_url}"
  puts "Puzzle URL: #{slider_response.puzzle_url}"

  puts "\nSimulating user verification (x=100)..."
  verify_result = client.slider.verify(
    session_id: slider_response.session_id,
    x: 100,
    trajectory: [
      { x: 10, y: 50, t: 100 },
      { x: 20, y: 50, t: 110 },
      { x: 30, y: 50, t: 120 },
    ]
  )
  puts "Verification success: #{verify_result.success}"
  puts "Message: #{verify_result.message}"
  puts "Remaining attempts: #{verify_result.remaining_attempts}"

rescue Hjtpx::Captcha::Exceptions::ApiError => e
  puts "API Error: #{e.message} (code: #{e.code})"
rescue Hjtpx::Captcha::Exceptions::NetworkError => e
  puts "Network Error: #{e.message}"
rescue => e
  puts "Error: #{e.class} - #{e.message}"
end

begin
  print_section("2. Click Captcha Example")

  puts "\nFetching click captcha..."
  click_response = client.click.get(mode: 'number', max_points: 4, allow_shuffle: true)
  puts "Session ID: #{click_response.session_id}"
  puts "Image URL: #{click_response.image_url}"
  puts "Hint: #{click_response.hint}"

  puts "\nSimulating user click verification..."
  points = [[100, 50], [200, 100], [150, 150], [80, 120]]
  verify_result = client.click.verify(
    session_id: click_response.session_id,
    points: points
  )
  puts "Verification success: #{verify_result.success}"
  puts "Message: #{verify_result.message}"

rescue Hjtpx::Captcha::Exceptions::ApiError => e
  puts "API Error: #{e.message} (code: #{e.code})"
rescue Hjtpx::Captcha::Exceptions::NetworkError => e
  puts "Network Error: #{e.message}"
rescue => e
  puts "Error: #{e.class} - #{e.message}"
end

begin
  print_section("3. Image Captcha Example")

  puts "\nFetching image captcha..."
  image_response = client.image.get(type: 'number', count: 4)
  puts "Challenge ID: #{image_response.challenge_id}"
  puts "Image data (first 100 chars): #{image_response.image.to_s[0..100]}..."

  puts "\nSimulating image captcha verification..."
  verify_result = client.image.verify(
    challenge_id: image_response.challenge_id,
    answer: '1234'
  )
  puts "Verification success: #{verify_result.success}"

rescue Hjtpx::Captcha::Exceptions::ApiError => e
  puts "API Error: #{e.message} (code: #{e.code})"
rescue => e
  puts "Error: #{e.class} - #{e.message}"
end

begin
  print_section("4. Gesture Captcha Example")

  puts "\nFetching gesture captcha..."
  gesture_response = client.gesture.get
  puts "Session ID: #{gesture_response.session_id}"
  puts "Pattern: #{gesture_response.pattern}"

  puts "\nSimulating gesture verification..."
  pattern = [0, 1, 2, 3, 4]
  verify_result = client.gesture.verify(
    session_id: gesture_response.session_id,
    pattern: pattern
  )
  puts "Verification success: #{verify_result.success}"

rescue Hjtpx::Captcha::Exceptions::ApiError => e
  puts "API Error: #{e.message} (code: #{e.code})"
rescue => e
  puts "Error: #{e.class} - #{e.message}"
end

begin
  print_section("5. User Authentication Example")

  puts "\nAttempting login..."
  login_result = client.auth.login(
    username: 'test_user',
    password: 'test_password',
    captcha_token: 'optional_captcha_token'
  )
  puts "Login success! Access token: #{login_result.access_token.to_s[0..20]}..."
  puts "Expires in: #{login_result.expires_in} seconds"

rescue Hjtpx::Captcha::Exceptions::AuthenticationError => e
  puts "Authentication failed: #{e.message}"
rescue => e
  puts "Error: #{e.class} - #{e.message}"
end

begin
  print_section("6. Environment Detection Example")

  puts "\nGetting detection script..."
  script = client.env.get_detection_script
  puts "Script length: #{script.length} characters"

rescue => e
  puts "Error: #{e.class} - #{e.message}"
end

print_section("Example Completed")
puts "\nNote: This is a demonstration script. In production,"
puts "you would handle the responses and integrate with your application."

client.close
