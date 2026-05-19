import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'captcha_result.dart';

class SliderCaptchaWidget extends StatefulWidget {
  final String backgroundImageUrl;
  final String sliderImageUrl;
  final Function(double progress) onSliderMoved;
  final Function(double progress) onSliderCompleted;

  const SliderCaptchaWidget({
    Key? key,
    required this.backgroundImageUrl,
    required this.sliderImageUrl,
    required this.onSliderMoved,
    required this.onSliderCompleted,
  }) : super(key: key);

  @override
  State<SliderCaptchaWidget> createState() => _SliderCaptchaWidgetState();
}

class _SliderCaptchaWidgetState extends State<SliderCaptchaWidget> {
  double _sliderPosition = 0.0;
  double _maxSliderPosition = 0.0;
  bool _isDragging = false;
  ImageProvider? _backgroundImage;
  ImageProvider? _sliderImage;

  @override
  void initState() {
    super.initState();
    _loadImages();
  }

  void _loadImages() {
    setState(() {
      _backgroundImage = NetworkImage(widget.backgroundImageUrl);
      _sliderImage = NetworkImage(widget.sliderImageUrl);
    });
  }

  void _onPanStart(DragStartDetails details) {
    setState(() {
      _isDragging = true;
    });
    if (widget.onSliderMoved != null) {
      HapticFeedback.lightImpact();
    }
  }

  void _onPanUpdate(DragUpdateDetails details) {
    if (!_isDragging) return;

    setState(() {
      _sliderPosition += details.delta.dx;
      _sliderPosition = _sliderPosition.clamp(0.0, _maxSliderPosition);
    });

    if (widget.onSliderMoved != null) {
      widget.onSliderMoved(_sliderPosition / _maxSliderPosition);
    }
  }

  void _onPanEnd(DragEndDetails details) {
    setState(() {
      _isDragging = false;
    });

    if (widget.onSliderCompleted != null) {
      HapticFeedback.mediumImpact();
      widget.onSliderCompleted(_sliderPosition / _maxSliderPosition);
    }
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final width = constraints.maxWidth;
        final height = width * 0.6;
        _maxSliderPosition = width - 60;

        return Container(
          width: width,
          height: height + 50,
          decoration: BoxDecoration(
            color: Colors.grey[200],
            borderRadius: BorderRadius.circular(8),
          ),
          child: Column(
            children: [
              Container(
                width: width,
                height: height,
                decoration: BoxDecoration(
                  color: Colors.grey[300],
                  borderRadius: const BorderRadius.only(
                    topLeft: Radius.circular(8),
                    topRight: Radius.circular(8),
                  ),
                ),
                child: _backgroundImage != null
                    ? Image(
                        image: _backgroundImage!,
                        fit: BoxFit.cover,
                      )
                    : const Center(child: CircularProgressIndicator()),
              ),
              const SizedBox(height: 10),
              Container(
                width: width - 20,
                height: 30,
                child: Stack(
                  children: [
                    Positioned(
                      left: _sliderPosition,
                      child: GestureDetector(
                        onPanStart: _onPanStart,
                        onPanUpdate: _onPanUpdate,
                        onPanEnd: _onPanEnd,
                        child: Container(
                          width: 50,
                          height: 30,
                          decoration: BoxDecoration(
                            color: Colors.white,
                            borderRadius: BorderRadius.circular(4),
                            boxShadow: [
                              BoxShadow(
                                color: Colors.black.withOpacity(0.2),
                                blurRadius: 4,
                                offset: const Offset(0, 2),
                              ),
                            ],
                          ),
                          child: Center(
                            child: Icon(
                              Icons.chevron_right,
                              color: Colors.grey[600],
                              size: 20,
                            ),
                          ),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}
