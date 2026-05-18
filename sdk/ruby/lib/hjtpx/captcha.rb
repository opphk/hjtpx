lib = File.expand_path('..', __FILE__)
$LOAD_PATH.unshift(lib) unless $LOAD_PATH.include?(lib)

module Hjtpx
  module Captcha
    autoload :Exceptions, 'hjtpx/captcha/exceptions'
    autoload :Models, 'hjtpx/captcha/models'
    autoload :Signer, 'hjtpx/captcha/signer'
    autoload :Pool, 'hjtpx/captcha/pool'
    autoload :Retry, 'hjtpx/captcha/retry'
    autoload :Client, 'hjtpx/captcha/client'

    VERSION = '1.0.0'.freeze

    def self.new(base_url:, api_key: nil, secret_key: nil, **kwargs)
      Client.new(base_url: base_url, api_key: api_key, secret_key: secret_key, **kwargs)
    end
  end
end
