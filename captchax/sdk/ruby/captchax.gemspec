Gem::Specification.new do |spec|
  spec.name          = 'captchax'
  spec.version       = '1.0.0'
  spec.authors       = ['CaptchaX Team']
  spec.email         = ['team@captchax.io']
  spec.summary       = 'CaptchaX SDK for Ruby - Multi-language captcha verification service'
  spec.description   = 'Official Ruby SDK for CaptchaX captcha verification service. Supports slider, click, and puzzle captchas with retry mechanism and timeout handling.'
  spec.homepage      = 'https://github.com/captchax/captchax-sdk-ruby'
  spec.license       = 'MIT'
  spec.required_ruby_version = '>= 3.0'

  spec.files         = Dir['lib/**/*.rb']
  spec.require_paths = ['lib']

  spec.add_dependency 'faraday', '~> 2.7'
  spec.add_dependency 'json', '>= 2.3'

  spec.add_development_dependency 'rspec', '~> 3.12'
  spec.add_development_dependency 'webmock', '~> 3.18'
  spec.add_development_dependency 'simplecov', '~> 0.22'
end
