module Hjtpx
  module Captcha
    module Exceptions
      class CaptchaError < StandardError
        attr_reader :original_error

        def initialize(message = nil, original_error: nil)
          super(message)
          @original_error = original_error
        end
      end

      class ApiError < CaptchaError
        attr_reader :code, :response_data

        def initialize(message = nil, code: nil, response_data: nil)
          super(message)
          @code = code
          @response_data = response_data
        end

        def to_s
          "API Error: #{super} (code: #{@code})"
        end
      end

      class NetworkError < CaptchaError
        def initialize(message = nil, original_error: nil)
          super(message || 'Network error occurred')
          @original_error = original_error
        end
      end

      class TimeoutError < NetworkError
        def initialize(message = nil, original_error: nil)
          super(message || 'Request timeout')
          @original_error = original_error
        end
      end

      class ValidationError < CaptchaError
        def initialize(message = nil, original_error: nil)
          super(message || 'Validation failed')
          @original_error = original_error
        end
      end

      class AuthenticationError < CaptchaError
        def initialize(message = nil, original_error: nil)
          super(message || 'Authentication failed')
          @original_error = original_error
        end
      end

      class SessionExpiredError < CaptchaError
        def initialize(message = nil, original_error: nil)
          super(message || 'Session expired')
          @original_error = original_error
        end
      end

      class RateLimitError < CaptchaError
        attr_reader :retry_after

        def initialize(message = nil, retry_after: nil)
          super(message || 'Rate limit exceeded')
          @retry_after = retry_after
        end
      end
    end
  end
end
