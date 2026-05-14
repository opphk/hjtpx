require 'spec_helper'
require 'captchax'
require 'captchax/client'

RSpec.describe CaptchaX::Client do
  let(:client) { described_class.new(base_url: 'https://api.example.com', app_id: 'test-app', retry_times: 3) }

  describe '#initialize' do
    it 'creates a client with configuration' do
      expect(client).to be_a(CaptchaX::Client)
    end

    it 'raises error when base_url is missing' do
      expect { described_class.new }.to raise_error(CaptchaX::ConfigurationError)
    end
  end

  describe '#health_check' do
    it 'returns health status' do
      stub_request(:get, 'https://api.example.com/v1/health')
        .to_return(
          status: 200,
          body: {
            code: 0,
            message: 'success',
            data: {
              status: 'healthy',
              service: 'captchax',
              timestamp: '2024-01-01T00:00:00Z',
              version: '1.0.0'
            }
          }.to_json
        )

      result = client.health_check
      expect(result[:status]).to eq('healthy')
    end
  end

  describe '#generate_slider_captcha' do
    it 'generates slider captcha' do
      stub_request(:post, 'https://api.example.com/v1/captcha/slider')
        .to_return(
          status: 200,
          body: {
            code: 0,
            message: 'success',
            data: {
              id: 'captcha-123',
              background_b64: 'base64bg',
              slider_b64: 'base64slider',
              target_x: 100,
              target_y: 50
            }
          }.to_json
        )

      result = client.generate_slider_captcha
      expect(result[:id]).to eq('captcha-123')
      expect(result[:target_x]).to eq(100)
    end

    it 'passes scenario_id when provided' do
      stub_request(:post, 'https://api.example.com/v1/captcha/slider')
        .with(body: hash_including('scenario_id' => 'scenario-1'))
        .to_return(
          status: 200,
          body: {
            code: 0,
            message: 'success',
            data: { id: 'captcha-123' }
          }.to_json
        )

      result = client.generate_slider_captcha(scenario_id: 'scenario-1')
      expect(result[:id]).to eq('captcha-123')
    end
  end

  describe '#verify_slider_captcha' do
    it 'verifies slider captcha' do
      stub_request(:post, 'https://api.example.com/v1/captcha/slider/verify')
        .to_return(
          status: 200,
          body: {
            code: 0,
            message: 'success',
            data: { success: true, message: 'Verification passed' }
          }.to_json
        )

      result = client.verify_slider_captcha(captcha_id: 'captcha-123', distance: 0.95)
      expect(result[:success]).to be true
    end
  end

  describe '#generate_click_captcha' do
    it 'generates click captcha' do
      stub_request(:post, 'https://api.example.com/v1/captcha/click')
        .to_return(
          status: 200,
          body: {
            code: 0,
            message: 'success',
            data: {
              id: 'click-456',
              image: 'base64image',
              target_chars: %w[A B C],
              char_positions: [
                { 'char' => 'A', 'x' => 100, 'y' => 50 },
                { 'char' => 'B', 'x' => 200, 'y' => 100 }
              ]
            }
          }.to_json
        )

      result = client.generate_click_captcha
      expect(result[:id]).to eq('click-456')
      expect(result[:target_chars]).to eq(%w[A B C])
    end
  end

  describe '#verify_click_captcha' do
    it 'verifies click captcha' do
      stub_request(:post, 'https://api.example.com/v1/captcha/click/verify')
        .to_return(
          status: 200,
          body: {
            code: 0,
            message: 'success',
            data: { success: true, score: 0.95, message: 'OK' }
          }.to_json
        )

      clicks = [CaptchaX::CharPosition.new(char: 'A', x: 100, y: 50)]
      result = client.verify_click_captcha(captcha_id: 'click-456', clicks: clicks)
      expect(result[:success]).to be true
      expect(result[:score]).to eq(0.95)
    end
  end

  describe '#generate_puzzle_captcha' do
    it 'generates puzzle captcha' do
      stub_request(:post, 'https://api.example.com/v1/captcha/puzzle')
        .to_return(
          status: 200,
          body: {
            code: 0,
            message: 'success',
            data: {
              id: 'puzzle-789',
              background_b64: 'base64bg',
              puzzle_b64: 'base64puzzle',
              target_x: 150,
              target_y: 75
            }
          }.to_json
        )

      result = client.generate_puzzle_captcha
      expect(result[:id]).to eq('puzzle-789')
    end
  end

  describe '#verify_puzzle_captcha' do
    it 'verifies puzzle captcha' do
      stub_request(:post, 'https://api.example.com/v1/captcha/puzzle/verify')
        .to_return(
          status: 200,
          body: {
            code: 0,
            message: 'success',
            data: { success: true, message: 'Verification passed' }
          }.to_json
        )

      result = client.verify_puzzle_captcha(captcha_id: 'puzzle-789', distance: 0.98)
      expect(result[:success]).to be true
    end
  end

  describe '#batch_verify' do
    it 'verifies multiple captchas' do
      stub_request(:post, 'https://api.example.com/v1/captcha/batch/verify')
        .to_return(
          status: 200,
          body: {
            code: 0,
            message: 'success',
            data: {
              results: [
                { captcha_id: 'captcha-1', success: true, message: 'OK' },
                { captcha_id: 'captcha-2', success: false, message: 'Invalid' }
              ],
              summary: { total: 2, success: 1, failed: 1, skipped: 0 }
            }
          }.to_json
        )

      items = [
        CaptchaX::BatchVerifyItem.new(captcha_id: 'captcha-1', type: 'slider', target_x: 100),
        CaptchaX::BatchVerifyItem.new(captcha_id: 'captcha-2', type: 'slider', target_x: 50)
      ]
      result = client.batch_verify(items: items)
      expect(result[:summary][:total]).to eq(2)
    end
  end

  describe '#list_scenarios' do
    it 'returns scenario list' do
      stub_request(:get, 'https://api.example.com/v1/scenarios')
        .to_return(
          status: 200,
          body: {
            code: 0,
            message: 'success',
            data: [
              { id: 'scenario-1', name: 'Test Scenario', description: 'Test', difficulty: 'medium' }
            ]
          }.to_json
        )

      result = client.list_scenarios
      expect(result).to be_an(Array)
      expect(result.first[:name]).to eq('Test Scenario')
    end
  end

  describe '#create_scenario' do
    it 'creates a scenario' do
      stub_request(:post, 'https://api.example.com/v1/scenarios')
        .to_return(
          status: 200,
          body: {
            code: 0,
            message: 'success',
            data: { id: 'scenario-new', name: 'New Scenario', description: 'New', difficulty: 'easy' }
          }.to_json
        )

      result = client.create_scenario(name: 'New Scenario', description: 'New', difficulty: 'easy')
      expect(result[:name]).to eq('New Scenario')
    end
  end

  describe '#get_scenario' do
    it 'returns a scenario' do
      stub_request(:get, 'https://api.example.com/v1/scenarios/scenario-123')
        .to_return(
          status: 200,
          body: {
            code: 0,
            message: 'success',
            data: { id: 'scenario-123', name: 'Test Scenario' }
          }.to_json
        )

      result = client.get_scenario('scenario-123')
      expect(result[:id]).to eq('scenario-123')
    end
  end

  describe '#update_scenario' do
    it 'updates a scenario' do
      stub_request(:put, 'https://api.example.com/v1/scenarios/scenario-123')
        .to_return(
          status: 200,
          body: {
            code: 0,
            message: 'success',
            data: { id: 'scenario-123', name: 'Updated Scenario' }
          }.to_json
        )

      result = client.update_scenario(scenario_id: 'scenario-123', name: 'Updated Scenario')
      expect(result[:name]).to eq('Updated Scenario')
    end
  end

  describe '#delete_scenario' do
    it 'deletes a scenario and returns true' do
      stub_request(:delete, 'https://api.example.com/v1/scenarios/scenario-123')
        .to_return(status: 200, body: { code: 0, message: 'success' }.to_json)

      result = client.delete_scenario('scenario-123')
      expect(result).to be true
    end
  end

  describe '#list_webhooks' do
    it 'returns webhook list' do
      stub_request(:get, 'https://api.example.com/v1/webhooks')
        .to_return(
          status: 200,
          body: {
            code: 0,
            message: 'success',
            data: [
              { id: 'webhook-1', url: 'https://example.com/webhook', events: ['verify.success'] }
            ]
          }.to_json
        )

      result = client.list_webhooks
      expect(result).to be_an(Array)
      expect(result.first[:url]).to eq('https://example.com/webhook')
    end
  end

  describe '#create_webhook' do
    it 'creates a webhook' do
      stub_request(:post, 'https://api.example.com/v1/webhooks')
        .to_return(
          status: 200,
          body: {
            code: 0,
            message: 'success',
            data: { id: 'webhook-new', url: 'https://example.com/newhook', events: ['verify.success'] }
          }.to_json
        )

      result = client.create_webhook(url: 'https://example.com/newhook', events: ['verify.success'])
      expect(result[:url]).to eq('https://example.com/newhook')
    end
  end

  describe '#get_webhook' do
    it 'returns a webhook' do
      stub_request(:get, 'https://api.example.com/v1/webhooks/webhook-123')
        .to_return(
          status: 200,
          body: {
            code: 0,
            message: 'success',
            data: { id: 'webhook-123', url: 'https://example.com/webhook' }
          }.to_json
        )

      result = client.get_webhook('webhook-123')
      expect(result[:id]).to eq('webhook-123')
    end
  end

  describe '#update_webhook' do
    it 'updates a webhook' do
      stub_request(:put, 'https://api.example.com/v1/webhooks/webhook-123')
        .to_return(
          status: 200,
          body: {
            code: 0,
            message: 'success',
            data: { id: 'webhook-123', url: 'https://example.com/updated' }
          }.to_json
        )

      result = client.update_webhook(webhook_id: 'webhook-123', url: 'https://example.com/updated')
      expect(result[:url]).to eq('https://example.com/updated')
    end
  end

  describe '#delete_webhook' do
    it 'deletes a webhook and returns true' do
      stub_request(:delete, 'https://api.example.com/v1/webhooks/webhook-123')
        .to_return(status: 200, body: { code: 0, message: 'success' }.to_json)

      result = client.delete_webhook('webhook-123')
      expect(result).to be true
    end
  end

  describe 'retry mechanism' do
    it 'retries on timeout' do
      counter = 0
      stub_request(:get, 'https://api.example.com/v1/health')
        .to_return do
          counter += 1
          if counter < 2
            raise Faraday::TimeoutError, 'Connection timeout'
          else
            { status: 200, body: { code: 0, message: 'success', data: { status: 'healthy' } }.to_json }
          end
        end

      result = client.health_check
      expect(result[:status]).to eq('healthy')
      expect(counter).to eq(2)
    end
  end

  describe 'error handling' do
    it 'raises ApiError on non-zero code' do
      stub_request(:get, 'https://api.example.com/v1/health')
        .to_return(
          status: 200,
          body: { code: 400, message: 'Bad request' }.to_json
        )

      expect { client.health_check }.to raise_error(CaptchaX::ApiError)
    end

    it 'raises ApiError on HTTP error' do
      stub_request(:get, 'https://api.example.com/v1/health')
        .to_return(status: 500, body: 'Internal Server Error')

      expect { client.health_check }.to raise_error(CaptchaX::ApiError)
    end
  end
end
