require 'faraday'
require 'json'
require 'captchax/error'
require 'captchax/models'

module CaptchaX
  class Client
    API_VERSIONS = { v1: 'v1', v2: 'v2' }.freeze

    def initialize(base_url:, app_id: nil, timeout: 10_000, retry_times: 3, api_version: :v1)
      raise CaptchaX::ConfigurationError, 'base_url is required' unless base_url

      @config = {
        base_url: base_url.chomp('/'),
        app_id: app_id,
        timeout: timeout,
        retry_times: retry_times,
        api_version: api_version
      }
      @connection = nil
    end

    def connection
      @connection ||= Faraday.new(url: @config[:base_url]) do |f|
        f.options.timeout = @config[:timeout] / 1000.0
        f.options.open_timeout = @config[:timeout] / 1000.0
        f.request :json
        f.response :json, content_type: /\bjson\b/
        f.adapter Faraday.default_adapter
      end.tap do |conn|
        conn.headers['Accept'] = 'application/json'
        conn.headers['X-App-Id'] = @config[:app_id] if @config[:app_id]
      end
    end

    def health_check
      execute_with_retry { get('/health') }
    end

    def generate_slider_captcha(scenario_id: nil, extra_data: nil)
      body = {}
      body['scenario_id'] = scenario_id if scenario_id
      body['extra_data'] = extra_data if extra_data
      execute_with_retry { post('/captcha/slider', body) }
    end

    def verify_slider_captcha(captcha_id:, distance:)
      body = { 'captcha_id' => captcha_id, 'distance' => distance }
      execute_with_retry { post('/captcha/slider/verify', body) }
    end

    def generate_click_captcha(scenario_id: nil, extra_data: nil)
      body = {}
      body['scenario_id'] = scenario_id if scenario_id
      body['extra_data'] = extra_data if extra_data
      execute_with_retry { post('/captcha/click', body) }
    end

    def verify_click_captcha(captcha_id:, clicks:)
      body = {
        'captcha_id' => captcha_id,
        'clicks' => clicks.map(&:to_h)
      }
      execute_with_retry { post('/captcha/click/verify', body) }
    end

    def generate_puzzle_captcha(scenario_id: nil, extra_data: nil)
      body = {}
      body['scenario_id'] = scenario_id if scenario_id
      body['extra_data'] = extra_data if extra_data
      execute_with_retry { post('/captcha/puzzle', body) }
    end

    def verify_puzzle_captcha(captcha_id:, distance:)
      body = { 'captcha_id' => captcha_id, 'distance' => distance }
      execute_with_retry { post('/captcha/puzzle/verify', body) }
    end

    def batch_verify(items:)
      body = { 'items' => items.map(&:to_h) }
      execute_with_retry { post('/captcha/batch/verify', body) }
    end

    def list_scenarios
      execute_with_retry { get('/scenarios') }
    end

    def create_scenario(name:, description: nil, difficulty: nil, config: nil)
      body = { 'name' => name }
      body['description'] = description if description
      body['difficulty'] = difficulty if difficulty
      body['config'] = config if config
      execute_with_retry { post('/scenarios', body) }
    end

    def get_scenario(scenario_id)
      execute_with_retry { get("/scenarios/#{scenario_id}") }
    end

    def update_scenario(scenario_id:, name: nil, description: nil, difficulty: nil, config: nil)
      body = {}
      body['name'] = name if name
      body['description'] = description if description
      body['difficulty'] = difficulty if difficulty
      body['config'] = config if config
      execute_with_retry { put("/scenarios/#{scenario_id}", body) }
    end

    def delete_scenario(scenario_id)
      execute_with_retry { delete("/scenarios/#{scenario_id}") }
      true
    end

    def list_webhooks
      execute_with_retry { get('/webhooks') }
    end

    def create_webhook(url:, events:, secret: nil, headers: nil, enabled: true)
      body = { 'url' => url, 'events' => events, 'enabled' => enabled }
      body['secret'] = secret if secret
      body['headers'] = headers if headers
      execute_with_retry { post('/webhooks', body) }
    end

    def get_webhook(webhook_id)
      execute_with_retry { get("/webhooks/#{webhook_id}") }
    end

    def update_webhook(webhook_id:, url: nil, events: nil, secret: nil, headers: nil, enabled: nil)
      body = {}
      body['url'] = url if url
      body['events'] = events if events
      body['secret'] = secret if secret
      body['headers'] = headers if headers
      body['enabled'] = enabled unless enabled.nil?
      execute_with_retry { put("/webhooks/#{webhook_id}", body) }
    end

    def delete_webhook(webhook_id)
      execute_with_retry { delete("/webhooks/#{webhook_id}") }
      true
    end

    private

    def api_prefix
      API_VERSIONS[@config[:api_version]] || 'v1'
    end

    def request(method, path, body = nil)
      response = connection.public_method(method).call("/#{api_prefix}#{path}", body)
      handle_response(response)
    rescue Faraday::TimeoutError, Faraday::ConnectionFailed => e
      raise CaptchaX::TimeoutError, "Request timed out: #{e.message}"
    rescue Faraday::Error => e
      raise CaptchaX::ApiError, e.message
    end

    def handle_response(response)
      raise CaptchaX::ApiError.new(
        "HTTP #{response.status}: #{response.body}",
        response.status
      ) unless response.success?

      data = response.body
      raise CaptchaX::CaptchaXError, 'Empty response received' if data.nil?

      code = data['code']
      if code != 0
        raise CaptchaX::ApiError.new(
          data['message'] || 'API error',
          response.status,
          code
        )
      end

      data['data']
    end

    def execute_with_retry
      attempts = 0
      delay = 0.5

      begin
        yield
      rescue CaptchaX::TimeoutError, CaptchaX::RetryExhaustedError => e
        raise e
      rescue CaptchaX::ApiError => e
        raise unless retryable_error?(e)
        attempts += 1
        if attempts >= @config[:retry_times]
          raise CaptchaX::RetryExhaustedError, "Request failed after #{@config[:retry_times]} attempts"
        end
        sleep(delay)
        delay *= 2
        retry
      end
    end

    def retryable_error?(error)
      error.http_status.nil? ||
        (error.http_status >= 500 && error.http_status < 600) ||
        error.http_status == 429
    end

    def get(path)
      request(:get, path)
    end

    def post(path, body)
      request(:post, path, body)
    end

    def put(path, body)
      request(:put, path, body)
    end

    def delete(path)
      request(:delete, path)
    end
  end
end
