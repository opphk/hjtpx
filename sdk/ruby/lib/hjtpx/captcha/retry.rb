require 'timeout'

module Hjtpx
  module Captcha
    module Retry
      class RetryConfig
        attr_accessor :max_attempts, :backoff_factor, :max_backoff,
                      :retry_on_timeout, :retry_on_network_error,
                      :retry_status_codes, :retry_on_exception

        def initialize(
          max_attempts: 3,
          backoff_factor: 0.5,
          max_backoff: 30,
          retry_on_timeout: true,
          retry_on_network_error: true,
          retry_status_codes: [429, 500, 502, 503, 504],
          retry_on_exception: true
        )
          @max_attempts = max_attempts
          @backoff_factor = backoff_factor
          @max_backoff = max_backoff
          @retry_on_timeout = retry_on_timeout
          @retry_on_network_error = retry_on_network_error
          @retry_status_codes = retry_status_codes
          @retry_on_exception = retry_on_exception
        end

        def to_h
          {
            max_attempts: max_attempts,
            backoff_factor: backoff_factor,
            max_backoff: max_backoff,
            retry_on_timeout: retry_on_timeout,
            retry_on_network_error: retry_on_network_error,
            retry_status_codes: retry_status_codes,
            retry_on_exception: retry_on_exception
          }
        end
      end

      class RetryManager
        attr_reader :config, :logger

        def initialize(config = nil, logger: nil)
          @config = config || RetryConfig.new
          @logger = logger || Logger.new(STDOUT)
        end

        def execute
          last_error = nil
          attempt = 0

          while attempt < @config.max_attempts
            attempt += 1

            begin
              return yield
            rescue *network_error_classes => e
              last_error = e

              unless should_retry?(e, attempt)
                raise e
              end

              if attempt < @config.max_attempts
                delay = calculate_backoff(attempt)
                @logger.warn("Retry attempt #{attempt}/#{@config.max_attempts} after #{delay}s: #{e.message}")
                sleep(delay)
              end
            rescue *timeout_error_classes => e
              last_error = e

              unless @config.retry_on_timeout
                raise e
              end

              if attempt < @config.max_attempts
                delay = calculate_backoff(attempt)
                @logger.warn("Retry attempt #{attempt}/#{@config.max_attempts} after #{delay}s (timeout): #{e.message}")
                sleep(delay)
              end
            rescue => e
              last_error = e

              if @config.retry_on_exception && should_retry_on_exception?(e)
                if attempt < @config.max_attempts
                  delay = calculate_backoff(attempt)
                  @logger.warn("Retry attempt #{attempt}/#{@config.max_attempts} after #{delay}s: #{e.message}")
                  sleep(delay)
                end
              else
                raise e
              end
            end
          end

          raise last_error
        end

        def should_retry?(error, attempt)
          return false if attempt >= @config.max_attempts

          true
        end

        def should_retry_on_exception?(error)
          return false unless @config.retry_on_exception
          return false unless @config.retry_on_exception.is_a?(Proc)

          @config.retry_on_exception.call(error)
        end

        def retry_on_status?(status_code)
          @config.retry_status_codes.include?(status_code)
        end

        private

        def calculate_backoff(attempt)
          delay = @config.backoff_factor * (2 ** (attempt - 1))
          [delay, @config.max_backoff].min
        end

        def network_error_classes
          [Errno::ECONNRESET, Errno::ECONNREFUSED, Errno::ENETUNREACH,
           Errno::EHOSTUNREACH, SocketError]
        end

        def timeout_error_classes
          [Timeout::Error, Net::OpenTimeout, Net::ReadTimeout, Net::WriteTimeout]
        end
      end
    end
  end
end
