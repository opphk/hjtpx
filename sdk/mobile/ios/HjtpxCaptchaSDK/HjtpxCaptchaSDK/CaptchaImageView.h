#import <UIKit/UIKit.h>

@interface CaptchaImageView : UIView

@property (nonatomic, copy) void (^onSliderMoved)(float progress);
@property (nonatomic, copy) void (^onSliderCompleted)(float progress);

- (void)setBackgroundImage:(UIImage *)image;
- (void)setSliderThumbImage:(UIImage *)image;
- (void)resetSlider;
- (void)showSuccess;
- (void)showFailure;

@end
