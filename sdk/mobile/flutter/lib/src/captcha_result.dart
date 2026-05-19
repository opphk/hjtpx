class SliderCaptchaResult {
  final String sessionId;
  final String backgroundImage;
  final String sliderImage;

  SliderCaptchaResult({
    required this.sessionId,
    required this.backgroundImage,
    required this.sliderImage,
  });

  factory SliderCaptchaResult.fromJson(Map<String, dynamic> json) {
    return SliderCaptchaResult(
      sessionId: json['session_id'] ?? '',
      backgroundImage: json['background_image'] ?? '',
      sliderImage: json['slider_image'] ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'session_id': sessionId,
      'background_image': backgroundImage,
      'slider_image': sliderImage,
    };
  }
}

class ClickCaptchaResult {
  final String sessionId;
  final String backgroundImage;
  final int targetCount;

  ClickCaptchaResult({
    required this.sessionId,
    required this.backgroundImage,
    required this.targetCount,
  });

  factory ClickCaptchaResult.fromJson(Map<String, dynamic> json) {
    return ClickCaptchaResult(
      sessionId: json['session_id'] ?? '',
      backgroundImage: json['background_image'] ?? '',
      targetCount: json['target_count'] ?? 0,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'session_id': sessionId,
      'background_image': backgroundImage,
      'target_count': targetCount,
    };
  }
}

class VerifyResult {
  final bool success;
  final double score;
  final String message;

  VerifyResult({
    required this.success,
    required this.score,
    required this.message,
  });

  factory VerifyResult.fromJson(Map<String, dynamic> json) {
    return VerifyResult(
      success: json['success'] ?? false,
      score: (json['score'] ?? 0.0).toDouble(),
      message: json['message'] ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'success': success,
      'score': score,
      'message': message,
    };
  }
}
