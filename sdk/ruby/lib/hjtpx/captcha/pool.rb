require 'timeout'
require 'logger'

module Hjtpx
  module Captcha
    module Pool
      class PoolConfig
        attr_accessor :pool_size, :max_pool_size, :pool_timeout,
                      :idle_timeout, :connection_timeout, :read_timeout,
                      :write_timeout, :keep_alive, :reuse_connections,
                      :max_retries, :retry_backoff_factor, :retry_status_codes

        def initialize(
          pool_size: 10,
          max_pool_size: 20,
          pool_timeout: 30,
          idle_timeout: 60,
          connection_timeout: 10,
          read_timeout: 30,
          write_timeout: 30,
          keep_alive: true,
          reuse_connections: true,
          max_retries: 3,
          retry_backoff_factor: 0.5,
          retry_status_codes: [429, 500, 502, 503, 504]
        )
          @pool_size = pool_size
          @max_pool_size = max_pool_size
          @pool_timeout = pool_timeout
          @idle_timeout = idle_timeout
          @connection_timeout = connection_timeout
          @read_timeout = read_timeout
          @write_timeout = write_timeout
          @keep_alive = keep_alive
          @reuse_connections = reuse_connections
          @max_retries = max_retries
          @retry_backoff_factor = retry_backoff_factor
          @retry_status_codes = retry_status_codes
        end

        def to_h
          {
            pool_size: pool_size,
            max_pool_size: max_pool_size,
            pool_timeout: pool_timeout,
            idle_timeout: idle_timeout,
            connection_timeout: connection_timeout,
            read_timeout: read_timeout,
            write_timeout: write_timeout,
            keep_alive: keep_alive,
            reuse_connections: reuse_connections,
            max_retries: max_retries,
            retry_backoff_factor: retry_backoff_factor,
            retry_status_codes: retry_status_codes
          }
        end
      end

      class ConnectionPool
        attr_reader :config, :logger

        def initialize(config = nil, logger: nil)
          @config = config || PoolConfig.new
          @logger = logger || Logger.new(STDOUT)
          @mutex = Mutex.new
          @connections = []
          @active_connections = 0
          @total_requests = 0
          @failed_requests = 0
        end

        def acquire(uri, proxy: nil)
          @mutex.synchronize do
            while @connections.empty? && @active_connections >= @config.max_pool_size
              @mutex.wait(@config.pool_timeout)
            end

            @active_connections += 1
          end

          begin
            connection = create_connection(uri, proxy: proxy)
            @total_requests += 1
            connection
          rescue => e
            @mutex.synchronize { @active_connections -= 1 }
            raise e
          end
        end

        def release(connection)
          @mutex.synchronize do
            @active_connections -= 1
            if @config.reuse_connections && connection&.started?
              @connections << connection
            else
              begin
                connection&.finish if connection&.started?
              rescue
                nil
              end
            end
            @mutex.broadcast
          end
        end

        def with_connection(uri, proxy: nil)
          connection = acquire(uri, proxy: proxy)
          begin
            yield connection
          ensure
            release(connection)
          end
        end

        def close
          @mutex.synchronize do
            @connections.each do |conn|
              begin
                conn.finish if conn.started?
              rescue
                nil
              end
            end
            @connections.clear
          end
        end

        def stats
          @mutex.synchronize do
            {
              pool_size: @config.pool_size,
              active_connections: @active_connections,
              idle_connections: @connections.size,
              total_requests: @total_requests,
              failed_requests: @failed_requests,
              success_rate: @total_requests > 0 ? (@total_requests - @failed_requests).to_f / @total_requests : 0.0
            }
          end
        end

        private

        def create_connection(uri, proxy: nil)
          http = Net::HTTP.new(uri.host, uri.port, proxy&.dig(:host), proxy&.dig(:port))
          http.open_timeout = @config.connection_timeout
          http.read_timeout = @config.read_timeout
          http.write_timeout = @config.write_timeout
          http.keep_alive_timeout = @config.idle_timeout if @config.keep_alive

          if uri.scheme == 'https'
            http.use_ssl = true
            http.verify_mode = OpenSSL::SSL::VERIFY_PEER
          end

          http.start unless http.started?
          http
        end
      end
    end
  end
end
