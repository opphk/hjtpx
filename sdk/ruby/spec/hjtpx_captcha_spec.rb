require 'spec_helper'

RSpec.describe Hjtpx::Captcha::Client do
  let(:base_url) { 'https://captcha.example.com' }
  let(:api_key) { 'test_api_key' }
  let(:secret_key) { 'test_secret_key' }

  describe '#initialize' do
    it 'creates a client with required parameters' do
      client = described_class.new(base_url: base_url)
      expect(client.base_url).to eq(base_url.chomp('/'))
    end

    it 'creates a client with api_key' do
      client = described_class.new(base_url: base_url, api_key: api_key)
      expect(client.api_key).to eq(api_key)
    end

    it 'creates a client with signer when secret_key is provided' do
      client = described_class.new(base_url: base_url, secret_key: secret_key)
      expect(client.pool).to be_a(Hjtpx::Captcha::Pool::ConnectionPool)
      expect(client.retry_manager).to be_a(Hjtpx::Captcha::Retry::RetryManager)
    end
  end

  describe '#slider' do
    let(:client) { described_class.new(base_url: base_url) }

    it 'returns a SliderCaptcha instance' do
      expect(client.slider).to be_a(Hjtpx::Captcha::SliderCaptcha)
    end
  end

  describe '#click' do
    let(:client) { described_class.new(base_url: base_url) }

    it 'returns a ClickCaptcha instance' do
      expect(client.click).to be_a(Hjtpx::Captcha::ClickCaptcha)
    end
  end

  describe '#auth' do
    let(:client) { described_class.new(base_url: base_url) }

    it 'returns a UserAuth instance' do
      expect(client.auth).to be_a(Hjtpx::Captcha::UserAuth)
    end
  end

  describe '#close' do
    let(:client) { described_class.new(base_url: base_url) }

    it 'closes the pool without error' do
      expect { client.close }.not_to raise_error
    end
  end
end

RSpec.describe Hjtpx::Captcha::SliderCaptcha do
  let(:client) { Hjtpx::Captcha::Client.new(base_url: 'https://captcha.example.com') }
  let(:slider) { described_class.new(client) }

  describe '#get' do
    it 'returns a SliderCaptchaResponse' do
      WebMock.stub_request(:get, 'https://captcha.example.com/api/v1/captcha/slider')
        .with(query: { 'width' => '320', 'height' => '160', 'tolerance' => '8' })
        .to_return(
          body: { code: 0, data: { sessionId: 'test_session', imageUrl: 'http://example.com/image.jpg', puzzleUrl: 'http://example.com/puzzle.jpg' } }.to_json,
          status: 200,
          headers: { 'Content-Type' => 'application/json' }
        )

      response = slider.get
      expect(response).to be_a(Hjtpx::Captcha::Models::SliderCaptchaResponse)
    end
  end

  describe '#verify' do
    it 'returns a VerifyCaptchaResponse' do
      WebMock.stub_request(:post, 'https://captcha.example.com/api/v1/captcha/verify')
        .with(body: hash_including(type: 'slider', sessionId: 'test_session', x: 100))
        .to_return(
          body: { code: 0, data: { success: true, message: 'Verification successful' } }.to_json,
          status: 200,
          headers: { 'Content-Type' => 'application/json' }
        )

      response = slider.verify(session_id: 'test_session', x: 100)
      expect(response).to be_a(Hjtpx::Captcha::Models::VerifyCaptchaResponse)
      expect(response.success).to be_truthy
    end
  end
end

RSpec.describe Hjtpx::Captcha::ClickCaptcha do
  let(:client) { Hjtpx::Captcha::Client.new(base_url: 'https://captcha.example.com') }
  let(:click) { described_class.new(client) }

  describe '#get' do
    it 'returns a ClickCaptchaResponse' do
      WebMock.stub_request(:get, 'https://captcha.example.com/api/v1/captcha/click')
        .with(query: { 'mode' => 'number', 'points' => '3', 'shuffle' => 'true' })
        .to_return(
          body: { code: 0, data: { sessionId: 'test_session', imageUrl: 'http://example.com/image.jpg', hint: 'Select 1, 2, 3' } }.to_json,
          status: 200,
          headers: { 'Content-Type' => 'application/json' }
        )

      response = click.get
      expect(response).to be_a(Hjtpx::Captcha::Models::ClickCaptchaResponse)
    end
  end
end

RSpec.describe Hjtpx::Captcha::Signer::HmacSigner do
  let(:secret_key) { 'test_secret_key' }
  let(:signer) { described_class.new(secret_key) }

  describe '#sign' do
    it 'returns a Base64 encoded signature' do
      signature = signer.sign('test_data')
      expect(signature).to be_a(String)
      expect(signature).not_to be_empty
    end

    it 'produces consistent signatures for the same input' do
      sig1 = signer.sign('test_data')
      sig2 = signer.sign('test_data')
      expect(sig1).to eq(sig2)
    end

    it 'produces different signatures for different inputs' do
      sig1 = signer.sign('test_data_1')
      sig2 = signer.sign('test_data_2')
      expect(sig1).not_to eq(sig2)
    end
  end

  describe '#verify' do
    it 'returns true for valid signature' do
      data = 'test_data'
      signature = signer.sign(data)
      expect(signer.verify(data, signature)).to be_truthy
    end

    it 'returns false for invalid signature' do
      data = 'test_data'
      expect(signer.verify(data, 'invalid_signature')).to be_falsey
    end
  end

  describe '#sign_request' do
    it 'returns timestamp and signature' do
      result = signer.sign_request('/api/v1/test')
      expect(result[:timestamp]).to be_a(Integer)
      expect(result[:signature]).to be_a(String)
    end
  end
end

RSpec.describe Hjtpx::Captcha::Pool::PoolConfig do
  describe '#initialize' do
    it 'sets default values' do
      config = described_class.new
      expect(config.pool_size).to eq(10)
      expect(config.max_pool_size).to eq(20)
      expect(config.max_retries).to eq(3)
    end

    it 'accepts custom values' do
      config = described_class.new(pool_size: 5, max_retries: 5)
      expect(config.pool_size).to eq(5)
      expect(config.max_retries).to eq(5)
    end
  end
end

RSpec.describe Hjtpx::Captcha::Retry::RetryConfig do
  describe '#initialize' do
    it 'sets default values' do
      config = described_class.new
      expect(config.max_attempts).to eq(3)
      expect(config.backoff_factor).to eq(0.5)
      expect(config.retry_status_codes).to eq([429, 500, 502, 503, 504])
    end

    it 'accepts custom values' do
      config = described_class.new(max_attempts: 5, backoff_factor: 1.0)
      expect(config.max_attempts).to eq(5)
      expect(config.backoff_factor).to eq(1.0)
    end
  end
end

RSpec.describe Hjtpx::Captcha::Models::BaseModel do
  describe '#initialize' do
    it 'converts keys to symbols' do
      model = described_class.new('test_key' => 'test_value')
      expect(model[:test_key]).to eq('test_value')
      expect(model.test_key).to eq('test_value')
    end
  end

  describe '#to_h' do
    it 'converts model to hash' do
      model = described_class.new(test: 'value')
      expect(model.to_h).to eq({ test: 'value' })
    end
  end
end

RSpec.describe Hjtpx::Captcha::Models::TrajectoryPoint do
  describe '#initialize' do
    it 'creates a trajectory point' do
      point = described_class.new(x: 10, y: 20, t: 100)
      expect(point.x).to eq(10)
      expect(point.y).to eq(20)
      expect(point.t).to eq(100)
    end
  end

  describe '#to_hash' do
    it 'returns a hash representation' do
      point = described_class.new(x: 10, y: 20, t: 100)
      expect(point.to_hash).to eq({ x: 10, y: 20, t: 100 })
    end
  end
end

RSpec.describe Hjtpx::Captcha::Models::VerifyCaptchaRequest do
  describe '#initialize' do
    it 'creates a request with block syntax' do
      request = described_class.new do |r|
        r.session_id = 'test_session'
        r.type = 'slider'
        r.x = 100
      end

      expect(request.session_id).to eq('test_session')
      expect(request.type).to eq('slider')
      expect(request.x).to eq(100)
    end
  end

  describe '#to_h' do
    it 'converts request to hash with camelCase keys' do
      request = described_class.new do |r|
        r.session_id = 'test_session'
        r.type = 'slider'
        r.x = 100
      end

      hash = request.to_h
      expect(hash[:sessionId]).to eq('test_session')
      expect(hash[:type]).to eq('slider')
      expect(hash[:x]).to eq(100)
    end
  end
end

RSpec.describe Hjtpx::Captcha::Exceptions do
  describe '.CaptchaError' do
    it 'creates an error with message' do
      error = described_class::CaptchaError.new('Test error')
      expect(error.message).to eq('Test error')
    end

    it 'stores original error' do
      original = StandardError.new('Original')
      error = described_class::CaptchaError.new('Error', original_error: original)
      expect(error.original_error).to eq(original)
    end
  end

  describe '.ApiError' do
    it 'creates an API error with code' do
      error = described_class::ApiError.new('API Error', code: 400)
      expect(error.code).to eq(400)
      expect(error.to_s).to include('code: 400')
    end
  end

  describe '.NetworkError' do
    it 'creates a network error' do
      error = described_class::NetworkError.new('Connection failed')
      expect(error.message).to eq('Connection failed')
    end
  end

  describe '.TimeoutError' do
    it 'creates a timeout error' do
      error = described_class::TimeoutError.new('Request timed out')
      expect(error.message).to eq('Request timed out')
    end
  end

  describe '.ValidationError' do
    it 'creates a validation error' do
      error = described_class::ValidationError.new('Invalid input')
      expect(error.message).to eq('Invalid input')
    end
  end

  describe '.AuthenticationError' do
    it 'creates an authentication error' do
      error = described_class::AuthenticationError.new('Auth failed')
      expect(error.message).to eq('Auth failed')
    end
  end

  describe '.SessionExpiredError' do
    it 'creates a session expired error' do
      error = described_class::SessionExpiredError.new('Session expired')
      expect(error.message).to eq('Session expired')
    end
  end

  describe '.RateLimitError' do
    it 'creates a rate limit error with retry_after' do
      error = described_class::RateLimitError.new('Rate limited', retry_after: 60)
      expect(error.retry_after).to eq(60)
    end
  end
end
