require 'ostruct'

module Hjtpx
  module Captcha
    module Models
      class BaseModel < OpenStruct
        def initialize(hash = {})
          super(hash.transform_keys(&:to_sym))
        end

        def to_h
          @table.transform_values { |v| v.is_a?(BaseModel) ? v.to_h : v }
        end

        def to_json(*args)
          to_h.to_json(*args)
        end

        def [](key)
          @table[key.to_sym]
        end

        def []=(key, value)
          @table[key.to_sym] = value
        end

        def method_missing(method_name, *args, &block)
          if method_name.to_s.end_with?('=')
            key = method_name.to_s.chomp('=').to_sym
            @table[key] = args.first
          else
            @table[method_name.to_sym]
          end
        end

        def respond_to_missing?(method_name, include_private = false)
          @table.key?(method_name.to_sym) || super
        end
      end

      class TrajectoryPoint < BaseModel
        def initialize(x:, y:, t:, **extra)
          super({ x: x, y: y, t: t }.merge(extra))
        end

        def to_hash
          { x: x, y: y, t: t }
        end
      end

      class SliderCaptchaResponse < BaseModel
        def session_id
          @table[:session_id] || @table[:sessionId]
        end

        def image_url
          @table[:image_url] || @table[:imageUrl]
        end

        def puzzle_url
          @table[:puzzle_url] || @table[:puzzleUrl]
        end
      end

      class ClickCaptchaResponse < BaseModel
        def session_id
          @table[:session_id] || @table[:sessionId]
        end

        def image_url
          @table[:image_url] || @table[:imageUrl]
        end
      end

      class ImageCaptchaResponse < BaseModel
        def challenge_id
          @table[:challenge_id] || @table[:challengeId]
        end
      end

      class RotationCaptchaResponse < BaseModel
        def challenge_id
          @table[:challenge_id] || @table[:challengeId]
        end
      end

      class GestureCaptchaResponse < BaseModel
        def session_id
          @table[:session_id] || @table[:sessionId]
        end
      end

      class JigsawPiece < BaseModel
        def initialize(hash = {})
          super(hash)
        end
      end

      class JigsawCaptchaResponse < BaseModel
        def session_id
          @table[:session_id] || @table[:sessionId]
        end

        def image_url
          @table[:image_url] || @table[:imageUrl]
        end

        def pieces
          @table[:pieces] ||= []
          @table[:pieces].map { |p| p.is_a?(JigsawPiece) ? p : JigsawPiece.new(p) }
        end
      end

      class VoiceCaptchaResponse < BaseModel
        def session_id
          @table[:session_id] || @table[:sessionId]
        end

        def audio_url
          @table[:audio_url] || @table[:audioUrl]
        end
      end

      class ConnectCaptchaResponse < BaseModel
        def session_id
          @table[:session_id] || @table[:sessionId]
        end

        def image_url
          @table[:image_url] || @table[:imageUrl]
        end
      end

      class ThreeDCaptchaResponse < BaseModel
        def session_id
          @table[:session_id] || @table[:sessionId]
        end

        def scene_url
          @table[:scene_url] || @table[:sceneUrl]
        end
      end

      class VerifyCaptchaResponse < BaseModel
        def success
          @table[:success] || @table[:captcha_pass] || @table[:captchaPass]
        end

        def message
          @table[:message]
        end

        def remaining_attempts
          @table[:remaining_attempts] || @table[:remainingAttempts]
        end

        def risk_score
          @table[:risk_score] || @table[:riskScore]
        end

        def fail_reason
          @table[:fail_reason] || @table[:failReason]
        end
      end

      class VerifyCaptchaRequest
        attr_accessor :session_id, :type, :x, :y, :points, :angle,
                      :pattern, :pieces, :answer, :connections, :target_position,
                      :trajectory, :click_sequence, :behavior_data

        def initialize
          yield self if block_given?
        end

        def to_h
          hash = { sessionId: session_id, type: type }
          hash[:x] = x if x
          hash[:y] = y if y
          hash[:points] = points if points
          hash[:angle] = angle if angle
          hash[:pattern] = pattern if pattern
          hash[:pieces] = pieces if pieces
          hash[:answer] = answer if answer
          hash[:connections] = connections if connections
          hash[:targetPosition] = target_position if target_position
          hash[:trajectory] = trajectory if trajectory
          hash[:clickSequence] = click_sequence if click_sequence
          hash[:behaviorData] = behavior_data if behavior_data
          hash
        end
      end

      class LoginRequest
        attr_accessor :username, :password, :captcha_token

        def initialize
          yield self if block_given?
        end

        def to_h
          hash = { username: username, password: password }
          hash[:captchaToken] = captcha_token if captcha_token
          hash
        end
      end

      class LoginResponse < BaseModel
        def access_token
          @table[:access_token] || @table[:accessToken]
        end

        def refresh_token
          @table[:refresh_token] || @table[:refreshToken]
        end

        def expires_in
          @table[:expires_in] || @table[:expiresIn]
        end

        def user
          @table[:user]
        end
      end
    end
  end
end
