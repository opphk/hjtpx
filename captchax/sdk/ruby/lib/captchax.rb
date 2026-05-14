require_relative 'captchax/error'
require_relative 'captchax/models'
require_relative 'captchax/client'

module CaptchaX
  VERSION = '1.0.0'.freeze

  def self.new(config = {})
    Client.new(
      base_url: config[:base_url],
      app_id: config[:app_id],
      timeout: config[:timeout] || 10_000,
      retry_times: config[:retry_times] || 3,
      api_version: config[:api_version] || :v1
    )
  end
end
