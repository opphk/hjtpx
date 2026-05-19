#import <Foundation/Foundation.h>

NS_ASSUME_NONNULL_BEGIN

@interface HjtpxCaptchaClient : NSObject

@property (nonatomic, copy, nullable) void (^onCaptchaLoaded)(NSString *sessionId, NSString *backgroundImage, NSString *sliderImage);
@property (nonatomic, copy, nullable) void (^onCaptchaVerified)(BOOL success, double score, NSString *message);
@property (nonatomic, copy, nullable) void (^onError)(NSString *error);

- (instancetype)initWithBaseUrl:(NSString *)baseUrl appId:(NSString *)appId appSecret:(NSString *)appSecret;

- (void)generateSliderCaptchaWithWidth:(NSInteger)width height:(NSInteger)height completion:(void (^)(NSString *sessionId, NSString *backgroundImageUrl, NSString *sliderImageUrl, NSError *error))completion;

- (void)verifySliderCaptchaWithSessionId:(NSString *)sessionId x:(float)x completion:(void (^)(BOOL success, double score, NSString *message, NSError *error))completion;

- (void)cancelAllRequests;

@end

NS_ASSUME_NONNULL_END
