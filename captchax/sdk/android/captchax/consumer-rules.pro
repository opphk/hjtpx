# Consumer ProGuard Rules for CaptchaX SDK
# This file is used by consumer apps to ensure compatibility with the SDK

# Keep SDK classes
-keep class com.captchax.sdk.** { *; }

# OkHttp
-dontwarn okhttp3.**
-dontwarn okio.**

# Kotlin
-keep class kotlin.** { *; }
-dontwarn kotlin.**

# Coroutines
-keepnames class kotlinx.coroutines.internal.MainDispatcherFactory {}
-keepnames class kotlinx.coroutines.CoroutineExceptionHandler {}

# AndroidX
-keep class androidx.** { *; }
-dontwarn androidx.**
