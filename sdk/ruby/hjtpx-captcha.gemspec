Gem::Specification.new do |spec|
  spec.name          = 'hjtpx-captcha'
  spec.version       = '1.0.0'
  spec.authors       = ['HJTPX Team']
  spec.email         = ['support@hjtpx.com']
  spec.summary       = 'HJTPX Captcha SDK for Ruby'
  spec.description   = 'Ruby SDK for HJTPX Captcha service with support for multiple captcha types'
  spec.homepage      = 'https://github.com/hjtpx/sdk-ruby'
  spec.license       = 'MIT'

  spec.files         = Dir['lib/**/*.rb']
  spec.require_paths = ['lib']

  spec.required_ruby_version = '>= 2.7.0'

  spec.add_dependency 'net-http', '>= 0.4.0'

  spec.add_development_dependency 'bundler', '~> 2.0'
  spec.add_development_dependency 'rake', '~> 13.0'
  spec.add_development_dependency 'rspec', '~> 3.10'
  spec.add_development_dependency 'webmock', '~> 3.14'
  spec.add_development_dependency 'simplecov', '~> 0.21'
end
