import 'dart:convert';
import 'package:http/http.dart' as http;
import 'captcha_result.dart';
import 'captcha_config.dart';

class CaptchaClient {
  final String baseUrl;
  final String appId;
  final String appSecret;
  final CaptchaConfig config;
  final http.Client _httpClient;

  CaptchaClient({
    required this.baseUrl,
    required this.appId,
    required this.appSecret,
    CaptchaConfig? config,
  })  : this.config = config ?? CaptchaConfig(),
        _httpClient = http.Client();

  Future<SliderCaptchaResult> generateSliderCaptcha({
    int? width,
    int? height,
  }) async {
    final url = Uri.parse('$baseUrl/api/captcha/slider');

    final response = await _httpClient.post(
      url,
      headers: {
        'Content-Type': 'application/json',
        'User-Agent': 'HjtpxCaptcha-Flutter/1.0',
      },
      body: jsonEncode({
        'app_id': appId,
        'captcha_type': 'slider',
        'width': width ?? config.captchaWidth,
        'height': height ?? config.captchaHeight,
      }),
    ).timeout(Duration(seconds: config.timeout));

    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return SliderCaptchaResult.fromJson(data);
    } else {
      throw CaptchaException(
        'Failed to generate captcha: ${response.statusCode}',
        response.statusCode,
      );
    }
  }

  Future<VerifyResult> verifySliderCaptcha({
    required String sessionId,
    required double x,
  }) async {
    final url = Uri.parse('$baseUrl/api/captcha/verify/slider');

    final response = await _httpClient.post(
      url,
      headers: {
        'Content-Type': 'application/json',
        'User-Agent': 'HjtpxCaptcha-Flutter/1.0',
      },
      body: jsonEncode({
        'session_id': sessionId,
        'app_id': appId,
        'x': x,
      }),
    ).timeout(Duration(seconds: config.timeout));

    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return VerifyResult.fromJson(data);
    } else {
      throw CaptchaException(
        'Failed to verify captcha: ${response.statusCode}',
        response.statusCode,
      );
    }
  }

  Future<ClickCaptchaResult> generateClickCaptcha({
    int count = 4,
  }) async {
    final url = Uri.parse('$baseUrl/api/captcha/click');

    final response = await _httpClient.post(
      url,
      headers: {
        'Content-Type': 'application/json',
        'User-Agent': 'HjtpxCaptcha-Flutter/1.0',
      },
      body: jsonEncode({
        'app_id': appId,
        'captcha_type': 'click',
        'count': count,
      }),
    ).timeout(Duration(seconds: config.timeout));

    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return ClickCaptchaResult.fromJson(data);
    } else {
      throw CaptchaException(
        'Failed to generate click captcha: ${response.statusCode}',
        response.statusCode,
      );
    }
  }

  Future<VerifyResult> verifyClickCaptcha({
    required String sessionId,
    required List<int> xCoords,
    required List<int> yCoords,
  }) async {
    final url = Uri.parse('$baseUrl/api/captcha/verify/click');

    final response = await _httpClient.post(
      url,
      headers: {
        'Content-Type': 'application/json',
        'User-Agent': 'HjtpxCaptcha-Flutter/1.0',
      },
      body: jsonEncode({
        'session_id': sessionId,
        'app_id': appId,
        'x_coords': xCoords,
        'y_coords': yCoords,
      }),
    ).timeout(Duration(seconds: config.timeout));

    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return VerifyResult.fromJson(data);
    } else {
      throw CaptchaException(
        'Failed to verify click captcha: ${response.statusCode}',
        response.statusCode,
      );
    }
  }

  void dispose() {
    _httpClient.close();
  }
}

class CaptchaException implements Exception {
  final String message;
  final int statusCode;

  CaptchaException(this.message, this.statusCode);

  @override
  String toString() => 'CaptchaException: $message (status: $statusCode)';
}
