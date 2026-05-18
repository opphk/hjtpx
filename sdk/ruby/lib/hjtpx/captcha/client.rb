require 'uri'
require 'net/http'
require 'json'
require 'timeout'
require_relative 'exceptions'
require_relative 'models'
require_relative 'signer'
require_relative 'pool'
require_relative 'retry'

module Hjtpx
  module Captcha
    class Client
      attr_reader :base_url, :api_key, :access_token, :pool, :retry_manager

      def initialize(
        base_url:,
        api_key: nil,
        secret_key: nil,
        pool_config: nil,
        retry_config: nil,
        logger: nil,
        timeout: 30
      )
        @base_url = base_url.chomp('/')
        @api_key = api_key
        @timeout = timeout
        @logger = logger || Logger.new(STDOUT)
        @access_token = nil
        @refresh_token = nil

        @pool = Pool::ConnectionPool.new(pool_config, logger: @logger)
        @retry_manager = Retry::RetryManager.new(retry_config, logger: @logger)

        if secret_key
          @signer = Signer::HmacSigner.new(secret_key)
        else
          @signer = nil
        end
      end

      def access_token=(token)
        @access_token = token
      end

      def get_access_token
        @access_token
      end

      def slider
        SliderCaptcha.new(self)
      end

      def click
        ClickCaptcha.new(self)
      end

      def image
        ImageCaptcha.new(self)
      end

      def rotation
        RotationCaptcha.new(self)
      end

      def gesture
        GestureCaptcha.new(self)
      end

      def jigsaw
        JigsawCaptcha.new(self)
      end

      def voice
        VoiceCaptcha.new(self)
      end

      def connect
        ConnectCaptcha.new(self)
      end

      def three_d
        ThreeDCaptcha.new(self)
      end

      def auth
        UserAuth.new(self)
      end

      def env
        Environment.new(self)
      end

      def verify_captcha(session_id:, type:, **kwargs)
        request = Models::VerifyCaptchaRequest.new
        request.session_id = session_id
        request.type = type
        kwargs.each { |k, v| request.public_send("#{k}=", v) if request.respond_to?("#{k}=") }

        response = post('/api/v1/captcha/verify', request.to_h)
        Models::VerifyCaptchaResponse.new(response)
      end

      def get(path, params: {})
        execute_request(:get, path, params: params)
      end

      def post(path, data = {})
        execute_request(:post, path, data: data)
      end

      def close
        @pool.close
      end

      def with_connection
        uri = URI.parse(@base_url)
        @pool.with_connection(uri) do |http|
          yield http
        end
      end

      private

      def execute_request(method, path, params: {}, data: nil)
        @retry_manager.execute do
          perform_request(method, path, params: params, data: data)
        end
      end

      def perform_request(method, path, params: {}, data: nil)
        uri = URI.parse(@base_url + path)
        uri.query = URI.encode_www_form(params) unless params.empty?

        headers = build_headers(path)

        @pool.with_connection(uri) do |http|
          case method
          when :get
            request = Net::HTTP::Get.new(uri.request_uri, headers)
          when :post
            request = Net::HTTP::Post.new(uri.request_uri, headers)
            request.body = data.to_json unless data.nil?
          when :put
            request = Net::HTTP::Put.new(uri.request_uri, headers)
            request.body = data.to_json unless data.nil?
          when :delete
            request = Net::HTTP::Delete.new(uri.request_uri, headers)
          else
            raise ArgumentError, "Unsupported HTTP method: #{method}"
          end

          response = http.request(request)
          handle_response(response)
        end
      end

      def build_headers(path)
        headers = {
          'Content-Type' => 'application/json',
          'Accept' => 'application/json',
          'User-Agent' => "HJTPX-Captcha-Ruby-SDK/#{VERSION}"
        }

        headers['X-API-Key'] = @api_key if @api_key
        headers['Authorization'] = "Bearer #{@access_token}" if @access_token

        if @signer
          timestamp = (Time.now.to_f * 1000).to_i
          data_to_sign = "#{timestamp}:#{path}"
          signature = @signer.sign(data_to_sign)
          headers['X-Timestamp'] = timestamp.to_s
          headers['X-Signature'] = signature
        end

        headers
      end

      def handle_response(response)
        raise Exceptions::NetworkError, "Empty response" if response.body.nil? || response.body.empty?

        status_code = response.code.to_i

        case status_code
        when 200..299
          begin
            data = JSON.parse(response.body)
          rescue JSON::ParserError
            raise Exceptions::ApiError, "Invalid JSON response: #{response.body[0..200]}"
          end

          if data.is_a?(Hash)
            if data['code'] && data['code'] != 0
              raise Exceptions::ApiError.new(
                data['message'] || 'Unknown error',
                code: data['code'],
                response_data: data['data']
              )
            end
            return data['data'] || data
          end

          data
        when 400
          raise Exceptions::ValidationError, "Validation error: #{response.body}"
        when 401
          raise Exceptions::AuthenticationError, "Authentication failed: #{response.body}"
        when 404
          raise Exceptions::SessionExpiredError, "Resource not found or session expired: #{response.body}"
        when 429
          retry_after = response['Retry-After'].to_i
          raise Exceptions::RateLimitError, "Rate limit exceeded", retry_after: retry_after
        when 500..599
          raise Exceptions::ApiError, "Server error: #{response.body}", code: status_code
        else
          raise Exceptions::ApiError, "Unexpected status code: #{status_code}", code: status_code
        end
      end

      class VERSION
        def self.to_s
          '1.0.0'
        end
      end
    end

    class SliderCaptcha
      def initialize(client)
        @client = client
      end

      def get(width: 320, height: 160, tolerance: 8)
        params = { width: width, height: height, tolerance: tolerance }
        response = @client.get('/api/v1/captcha/slider', params: params)
        Models::SliderCaptchaResponse.new(response)
      end

      def verify(session_id:, x:, y: nil, trajectory: nil, behavior_data: nil)
        request = Models::VerifyCaptchaRequest.new
        request.session_id = session_id
        request.type = 'slider'
        request.x = x
        request.y = y if y
        request.trajectory = trajectory if trajectory
        request.behavior_data = behavior_data if behavior_data

        response = @client.post('/api/v1/captcha/verify', request.to_h)
        Models::VerifyCaptchaResponse.new(response)
      end
    end

    class ClickCaptcha
      def initialize(client)
        @client = client
      end

      def get(mode: 'number', max_points: 3, allow_shuffle: true)
        params = { mode: mode, points: max_points, shuffle: allow_shuffle }
        response = @client.get('/api/v1/captcha/click', params: params)
        Models::ClickCaptchaResponse.new(response)
      end

      def verify(session_id:, points:, click_sequence: nil, behavior_data: nil)
        request = Models::VerifyCaptchaRequest.new
        request.session_id = session_id
        request.type = 'click'
        request.points = points
        request.click_sequence = click_sequence if click_sequence
        request.behavior_data = behavior_data if behavior_data

        response = @client.post('/api/v1/captcha/verify', request.to_h)
        Models::VerifyCaptchaResponse.new(response)
      end
    end

    class ImageCaptcha
      def initialize(client)
        @client = client
      end

      def get(type: 'mixed', count: 4, noise_mode: 0, line_mode: 0)
        params = { type: type, count: count, noise_mode: noise_mode, line_mode: line_mode }
        response = @client.get('/api/v1/captcha/image', params: params)
        Models::ImageCaptchaResponse.new(response)
      end

      def verify(challenge_id:, answer:)
        request = Models::VerifyCaptchaRequest.new
        request.session_id = challenge_id
        request.type = 'image'
        request.answer = answer

        response = @client.post('/api/v1/captcha/image/verify', request.to_h)
        Models::VerifyCaptchaResponse.new(response)
      end
    end

    class RotationCaptcha
      def initialize(client)
        @client = client
      end

      def get
        response = @client.get('/api/v1/captcha/rotation')
        Models::RotationCaptchaResponse.new(response)
      end

      def verify(challenge_id:, angle:)
        request = Models::VerifyCaptchaRequest.new
        request.session_id = challenge_id
        request.type = 'rotation'
        request.angle = angle

        response = @client.post('/api/v1/captcha/rotation/verify', request.to_h)
        Models::VerifyCaptchaResponse.new(response)
      end
    end

    class GestureCaptcha
      def initialize(client)
        @client = client
      end

      def get
        response = @client.get('/api/v1/captcha/gesture')
        Models::GestureCaptchaResponse.new(response)
      end

      def verify(session_id:, pattern:)
        request = Models::VerifyCaptchaRequest.new
        request.session_id = session_id
        request.type = 'gesture'
        request.pattern = pattern

        response = @client.post('/api/v1/captcha/gesture/verify', request.to_h)
        Models::VerifyCaptchaResponse.new(response)
      end
    end

    class JigsawCaptcha
      def initialize(client)
        @client = client
      end

      def get(width: 300, height: 300, grid_size: 3)
        params = { width: width, height: height, grid_size: grid_size }
        response = @client.get('/api/v1/captcha/jigsaw', params: params)
        Models::JigsawCaptchaResponse.new(response)
      end

      def verify(session_id:, pieces:)
        request = Models::VerifyCaptchaRequest.new
        request.session_id = session_id
        request.type = 'jigsaw'
        request.pieces = pieces

        response = @client.post('/api/v1/captcha/jigsaw/verify', request.to_h)
        Models::VerifyCaptchaResponse.new(response)
      end
    end

    class VoiceCaptcha
      def initialize(client)
        @client = client
      end

      def get(language: 'zh-CN')
        params = { language: language }
        response = @client.get('/api/v1/captcha/voice', params: params)
        Models::VoiceCaptchaResponse.new(response)
      end

      def verify(session_id:, answer:)
        request = Models::VerifyCaptchaRequest.new
        request.session_id = session_id
        request.type = 'voice'
        request.answer = answer

        response = @client.post('/api/v1/captcha/verify', request.to_h)
        Models::VerifyCaptchaResponse.new(response)
      end
    end

    class ConnectCaptcha
      def initialize(client)
        @client = client
      end

      def get
        response = @client.get('/api/v1/captcha/connect')
        Models::ConnectCaptchaResponse.new(response)
      end

      def verify(session_id:, connections:)
        request = Models::VerifyCaptchaRequest.new
        request.session_id = session_id
        request.type = 'connect'
        request.connections = connections

        response = @client.post('/api/v1/captcha/verify', request.to_h)
        Models::VerifyCaptchaResponse.new(response)
      end
    end

    class ThreeDCaptcha
      def initialize(client)
        @client = client
      end

      def get
        response = @client.get('/api/v1/captcha/3d')
        Models::ThreeDCaptchaResponse.new(response)
      end

      def verify(session_id:, target_position:)
        request = Models::VerifyCaptchaRequest.new
        request.session_id = session_id
        request.type = '3d'
        request.target_position = target_position

        response = @client.post('/api/v1/captcha/verify', request.to_h)
        Models::VerifyCaptchaResponse.new(response)
      end
    end

    class UserAuth
      def initialize(client)
        @client = client
      end

      def register(username:, email:, password:, behavior_data: nil)
        data = { username: username, email: email, password: password }
        data[:behaviorData] = behavior_data if behavior_data
        @client.post('/api/v1/auth/register', data)
      end

      def login(username:, password:, captcha_token: nil)
        data = Models::LoginRequest.new
        data.username = username
        data.password = password
        data.captcha_token = captcha_token if captcha_token

        response = @client.post('/api/v1/auth/login', data.to_h)
        @client.access_token = response['access_token'] || response['accessToken']
        @client.instance_variable_set(:@refresh_token, response['refresh_token'] || response['refreshToken'])

        Models::LoginResponse.new(response)
      end

      def refresh_token(refresh_token: nil)
        token = refresh_token || @client.instance_variable_get(:@refresh_token)
        raise Exceptions::CaptchaError, "No refresh token available" unless token

        response = @client.post('/api/v1/auth/refresh', { refreshToken: token })
        @client.access_token = response['access_token'] || response['accessToken']
        @client.instance_variable_set(:@refresh_token, response['refresh_token'] || response['refreshToken']) if response['refresh_token'] || response['refreshToken']

        response
      end

      def logout
        @client.post('/api/v1/auth/logout')
        @client.access_token = nil
        @client.instance_variable_set(:@refresh_token, nil)
        nil
      end

      def verify_email(token:)
        @client.get('/api/v1/auth/verify-email', params: { token: token })
      end

      def resend_verification(email:)
        @client.post('/api/v1/auth/resend-verification', { email: email })
      end

      def request_password_reset(email:)
        @client.post('/api/v1/auth/request-password-reset', { email: email })
      end

      def reset_password(token:, new_password:)
        @client.post('/api/v1/auth/reset-password', { token: token, newPassword: new_password })
      end
    end

    class Environment
      def initialize(client)
        @client = client
      end

      def get_detection_script(callback: nil)
        params = { callback: callback } if callback
        @client.get('/api/v1/detect/script', params: params || {})
      end

      def submit_detection(data:)
        @client.post('/api/v1/detect/submit', data)
      end

      def check_environment(data:)
        @client.post('/api/v1/detect/check', data)
      end
    end
  end
end
