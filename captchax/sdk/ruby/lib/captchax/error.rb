module CaptchaX
  class CaptchaXError < StandardError
    attr_reader :code

    def initialize(message, code = nil)
      super(message)
      @code = code
    end
  end

  class ConfigurationError < CaptchaXError
    def initialize(message)
      super(message, 'CONFIG_ERROR')
    end
  end

  class ApiError < CaptchaXError
    attr_reader :http_status

    def initialize(message, http_status = nil, code = nil)
      super(message, code)
      @http_status = http_status
    end
  end

  class TimeoutError < CaptchaXError
    def initialize(message = 'Request timed out')
      super(message, 'TIMEOUT')
    end
  end

  class RetryExhaustedError < CaptchaXError
    def initialize(message = 'Retry attempts exhausted')
      super(message, 'RETRY_EXHAUSTED')
    end
  end
end
