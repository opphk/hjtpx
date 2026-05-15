#ifndef CaptchaX_Bridging_Header_h
#define CaptchaX_Bridging_Header_h

#import <Foundation/Foundation.h>

NS_ASSUME_NONNULL_BEGIN

@class CaptchaResult;

@protocol CaptchaXDelegate <NSObject>
- (void)captchaXDidSuccess:(CaptchaResult *)result;
- (void)captchaXDidFailed:(NSError *)error;
- (void)captchaXDidClose;
@end

NS_ASSUME_NONNULL_END

#endif
