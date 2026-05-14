module CaptchaX
  class CharPosition
    attr_accessor :char, :x, :y

    def initialize(char: nil, x: nil, y: nil)
      @char = char
      @x = x
      @y = y
    end

    def to_h
      { 'char' => @char, 'x' => @x, 'y' => @y }
    end

    def self.from_h(hash)
      new(char: hash['char'], x: hash['x'], y: hash['y'])
    end
  end

  class SliderCaptchaResult
    attr_accessor :id, :background_b64, :slider_b64, :target_x, :target_y

    def initialize(id: nil, background_b64: nil, slider_b64: nil, target_x: nil, target_y: nil)
      @id = id
      @background_b64 = background_b64
      @slider_b64 = slider_b64
      @target_x = target_x
      @target_y = target_y
    end

    def self.from_h(hash)
      new(
        id: hash['id'],
        background_b64: hash['background_b64'],
        slider_b64: hash['slider_b64'],
        target_x: hash['target_x'],
        target_y: hash['target_y']
      )
    end
  end

  class SliderVerifyResult
    attr_accessor :success, :message

    def initialize(success:, message: nil)
      @success = success
      @message = message
    end

    def self.from_h(hash)
      new(success: hash['success'], message: hash['message'])
    end
  end

  class ClickCaptchaResult
    attr_accessor :id, :image, :target_chars, :char_positions

    def initialize(id: nil, image: nil, target_chars: nil, char_positions: nil)
      @id = id
      @image = image
      @target_chars = target_chars || []
      @char_positions = char_positions || []
    end

    def self.from_h(hash)
      new(
        id: hash['id'],
        image: hash['image'],
        target_chars: hash['target_chars'] || [],
        char_positions: (hash['char_positions'] || []).map { |pos| CharPosition.from_h(pos) }
      )
    end
  end

  class ClickVerifyResult
    attr_accessor :success, :score, :message

    def initialize(success:, score: nil, message: nil)
      @success = success
      @score = score
      @message = message
    end

    def self.from_h(hash)
      new(success: hash['success'], score: hash['score'], message: hash['message'])
    end
  end

  class PuzzleCaptchaResult
    attr_accessor :id, :background_b64, :puzzle_b64, :target_x, :target_y

    def initialize(id: nil, background_b64: nil, puzzle_b64: nil, target_x: nil, target_y: nil)
      @id = id
      @background_b64 = background_b64
      @puzzle_b64 = puzzle_b64
      @target_x = target_x
      @target_y = target_y
    end

    def self.from_h(hash)
      new(
        id: hash['id'],
        background_b64: hash['background_b64'],
        puzzle_b64: hash['puzzle_b64'],
        target_x: hash['target_x'],
        target_y: hash['target_y']
      )
    end
  end

  class PuzzleVerifyResult
    attr_accessor :success, :message

    def initialize(success:, message: nil)
      @success = success
      @message = message
    end

    def self.from_h(hash)
      new(success: hash['success'], message: hash['message'])
    end
  end

  class Scenario
    attr_accessor :id, :name, :description, :difficulty, :config, :created_at, :updated_at

    def initialize(id: nil, name: nil, description: nil, difficulty: nil, config: nil, created_at: nil, updated_at: nil)
      @id = id
      @name = name
      @description = description
      @difficulty = difficulty
      @config = config
      @created_at = created_at
      @updated_at = updated_at
    end

    def to_h
      hash = { 'name' => @name }
      hash['description'] = @description if @description
      hash['difficulty'] = @difficulty if @difficulty
      hash['config'] = @config if @config
      hash
    end

    def self.from_h(hash)
      new(
        id: hash['id'],
        name: hash['name'],
        description: hash['description'],
        difficulty: hash['difficulty'],
        config: hash['config'],
        created_at: hash['created_at'],
        updated_at: hash['updated_at']
      )
    end
  end

  class Webhook
    attr_accessor :id, :app_id, :url, :secret, :events, :headers, :enabled, :created_at, :updated_at

    def initialize(id: nil, app_id: nil, url: nil, secret: nil, events: nil, headers: nil, enabled: true, created_at: nil, updated_at: nil)
      @id = id
      @app_id = app_id
      @url = url
      @secret = secret
      @events = events || []
      @headers = headers
      @enabled = enabled
      @created_at = created_at
      @updated_at = updated_at
    end

    def to_h
      hash = { 'url' => @url, 'events' => @events, 'enabled' => @enabled }
      hash['secret'] = @secret if @secret
      hash['headers'] = @headers if @headers
      hash
    end

    def self.from_h(hash)
      new(
        id: hash['id'],
        app_id: hash['app_id'],
        url: hash['url'],
        secret: hash['secret'],
        events: hash['events'] || [],
        headers: hash['headers'],
        enabled: hash.fetch('enabled', true),
        created_at: hash['created_at'],
        updated_at: hash['updated_at']
      )
    end
  end

  class BatchVerifyItem
    attr_accessor :captcha_id, :type, :target_x, :target_y, :clicks

    def initialize(captcha_id:, type:, target_x: nil, target_y: nil, clicks: nil)
      @captcha_id = captcha_id
      @type = type
      @target_x = target_x
      @target_y = target_y
      @clicks = clicks
    end

    def to_h
      hash = { 'captcha_id' => @captcha_id, 'type' => @type }
      hash['target_x'] = @target_x if @target_x
      hash['target_y'] = @target_y if @target_y
      hash['clicks'] = @clicks.map(&:to_h) if @clicks
      hash
    end

    def self.from_h(hash)
      new(
        captcha_id: hash['captcha_id'],
        type: hash['type'],
        target_x: hash['target_x'],
        target_y: hash['target_y'],
        clicks: (hash['clicks'] || []).map { |pos| CharPosition.from_h(pos) }
      )
    end
  end

  class BatchVerifyResult
    attr_accessor :captcha_id, :success, :message, :score

    def initialize(captcha_id:, success:, message: nil, score: nil)
      @captcha_id = captcha_id
      @success = success
      @message = message
      @score = score
    end

    def self.from_h(hash)
      new(
        captcha_id: hash['captcha_id'],
        success: hash['success'],
        message: hash['message'],
        score: hash['score']
      )
    end
  end

  class BatchVerifySummary
    attr_accessor :total, :success, :failed, :skipped

    def initialize(total:, success:, failed:, skipped:)
      @total = total
      @success = success
      @failed = failed
      @skipped = skipped
    end

    def self.from_h(hash)
      new(
        total: hash['total'],
        success: hash['success'],
        failed: hash['failed'],
        skipped: hash['skipped']
      )
    end
  end

  class BatchVerifyResponse
    attr_accessor :results, :summary

    def initialize(results:, summary:)
      @results = results
      @summary = summary
    end

    def self.from_h(hash)
      new(
        results: (hash['results'] || []).map { |r| BatchVerifyResult.from_h(r) },
        summary: BatchVerifySummary.from_h(hash['summary'] || {})
      )
    end
  end

  class HealthStatus
    attr_accessor :status, :service, :timestamp, :version

    def initialize(status:, service: nil, timestamp: nil, version: nil)
      @status = status
      @service = service
      @timestamp = timestamp
      @version = version
    end

    def self.from_h(hash)
      new(
        status: hash['status'],
        service: hash['service'],
        timestamp: hash['timestamp'],
        version: hash['version']
      )
    end
  end
end
