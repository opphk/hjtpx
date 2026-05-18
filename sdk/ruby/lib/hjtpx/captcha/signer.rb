require 'openssl'
require 'base64'

module Hjtpx
  module Captcha
    module Signer
      class HmacSigner
        attr_reader :secret_key, :algorithm

        def initialize(secret_key, algorithm: 'SHA256')
          @secret_key = secret_key
          @algorithm = algorithm
        end

        def sign(data)
          digest = OpenSSL::Digest.new(algorithm.downcase)
          hmac = OpenSSL::HMAC.digest(digest, secret_key, data)
          Base64.strict_encode64(hmac)
        end

        def verify(data, signature)
          expected = sign(data)
          secure_compare(expected, signature)
        end

        def sign_request(path, timestamp = nil)
          timestamp ||= (Time.now.to_f * 1000).to_i
          data_to_sign = "#{timestamp}:#{path}"
          signature = sign(data_to_sign)
          { timestamp: timestamp, signature: signature }
        end

        private

        def secure_compare(a, b)
          return false unless a.bytesize == b.bytesize

          l = a.unpack("C*")
          r, i = 0, -1

          b.each_byte do |v|
            i += 1
            r |= v ^ l[i]
          end

          r == 0
        end
      end
    end
  end
end
