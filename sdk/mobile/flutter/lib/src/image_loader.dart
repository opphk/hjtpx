import 'dart:typed_data';
import 'package:http/http.dart' as http;

class ImageLoader {
  final http.Client _httpClient;
  final Map<String, Uint8List> _cache;

  ImageLoader() : _httpClient = http.Client(), _cache = {};

  Future<Uint8List?> loadImage(String url) async {
    if (_cache.containsKey(url)) {
      return _cache[url];
    }

    try {
      final response = await _httpClient.get(
        Uri.parse(url),
        headers: {
          'User-Agent': 'HjtpxCaptcha-Flutter/1.0',
        },
      );

      if (response.statusCode == 200) {
        final bytes = response.bodyBytes;
        _cache[url] = bytes;
        return bytes;
      }
    } catch (e) {
      return null;
    }

    return null;
  }

  Future<void> preloadImage(String url) async {
    await loadImage(url);
  }

  void preloadImages(List<String> urls) async {
    for (final url in urls) {
      await preloadImage(url);
    }
  }

  void clearCache() {
    _cache.clear();
  }

  void dispose() {
    _httpClient.close();
  }
}
