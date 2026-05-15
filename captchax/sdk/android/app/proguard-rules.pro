# Add project specific ProGuard rules here.

-keepattributes Signature
-keepattributes *Annotation*

# Keep all CaptchaX SDK classes
-keep class com.captchax.sdk.** { *; }

# OkHttp
-dontwarn okhttp3.**
-dontwarn okio.**
