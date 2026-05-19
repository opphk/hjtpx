class CaptchaConfig {
  int captchaWidth;
  int captchaHeight;
  bool enableHapticFeedback;
  bool enableSoundEffect;
  int sliderTrackHeight;
  int sliderThumbSize;
  int timeout;

  CaptchaConfig({
    this.captchaWidth = 320,
    this.captchaHeight = 200,
    this.enableHapticFeedback = true,
    this.enableSoundEffect = false,
    this.sliderTrackHeight = 4,
    this.sliderThumbSize = 50,
    this.timeout = 30,
  });

  CaptchaConfig copyWith({
    int? captchaWidth,
    int? captchaHeight,
    bool? enableHapticFeedback,
    bool? enableSoundEffect,
    int? sliderTrackHeight,
    int? sliderThumbSize,
    int? timeout,
  }) {
    return CaptchaConfig(
      captchaWidth: captchaWidth ?? this.captchaWidth,
      captchaHeight: captchaHeight ?? this.captchaHeight,
      enableHapticFeedback: enableHapticFeedback ?? this.enableHapticFeedback,
      enableSoundEffect: enableSoundEffect ?? this.enableSoundEffect,
      sliderTrackHeight: sliderTrackHeight ?? this.sliderTrackHeight,
      sliderThumbSize: sliderThumbSize ?? this.sliderThumbSize,
      timeout: timeout ?? this.timeout,
    );
  }
}
