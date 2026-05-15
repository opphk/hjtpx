# CaptchaX Android SDK

CaptchaX Android SDK 是 CaptchaX 行为验证系统的官方 Android 客户端 SDK，支持 Kotlin 和 Java，提供简单易用的验证码集成方案。

## 功能特性

### 🎯 支持的验证码类型
- **滑块验证码 (Slider)** - 拖动滑块完成拼图
- **点选验证码 (Click)** - 按顺序点击指定区域
- **拼图验证码 (Puzzle)** - 拖动滑块填充拼图
- **旋转验证码 (Rotate)** - 旋转图片至正确角度
- **文字验证码 (Text)** - 输入图片中的文字
- **图标验证码 (Icon)** - 依次点击对应图标

### ✨ 核心特性
- **Kotlin / Java 双支持** - 完整支持 Kotlin 和 Java 互操作
- **协程异步处理** - 使用 Kotlin 协程处理异步操作
- **智能图片缓存** - LRU 缓存策略，减少网络请求
- **设备指纹识别** - 生成唯一设备标识
- **安全签名验证** - HMAC-SHA256 请求签名
- **预加载机制** - 提前加载验证码，提升用户体验
- **Jetpack Compose 支持** - 完整的 Compose 组件封装

## 项目结构

```
captchax/sdk/android/
├── captchax/                         # SDK 模块
│   ├── src/main/
│   │   ├── java/com/captchax/sdk/
│   │   │   ├── CaptchaX.kt          # 主入口类
│   │   │   ├── CaptchaConfig.kt     # 配置类
│   │   │   ├── CaptchaView.kt       # 验证码视图
│   │   │   ├── CaptchaListener.kt   # 回调接口
│   │   │   ├── CaptchaType.kt       # 验证码类型
│   │   │   ├── NetworkClient.kt     # 网络客户端
│   │   │   ├── DeviceFingerprint.kt # 设备指纹
│   │   │   ├── ImageCache.kt        # 图片缓存
│   │   │   └── util/
│   │   │       ├── Logger.kt         # 日志工具
│   │   │       └── Extensions.kt    # 扩展函数
│   │   └── res/
│   │       ├── layout/              # 布局文件
│   │       ├── drawable/            # Drawable 资源
│   │       └── values/              # 字符串资源
│   └── build.gradle.kts            # SDK 构建配置
├── app/                             # 示例应用
│   ├── src/main/java/com/captchax/example/
│   │   ├── MainActivity.kt         # Kotlin 示例
│   │   ├── JavaExampleActivity.java # Java 示例
│   │   └── ComposeExample.kt       # Compose 示例
│   └── build.gradle.kts            # 示例构建配置
├── build.gradle.kts                # 根构建配置
├── settings.gradle.kts              # 项目设置
├── gradle.properties               # Gradle 属性
└── README.md                       # 本文档
```

## 安装指南

### 方式一：Maven Central

在 `settings.gradle.kts` 中添加仓库：

```kotlin
dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        google()
        mavenCentral()
    }
}
```

在 `build.gradle.kts` 中添加依赖：

```kotlin
dependencies {
    implementation("com.captchax.sdk:captchax:1.0.0")
}
```

### 方式二：本地 Module

将 SDK Module 复制到项目中：

```kotlin
// settings.gradle.kts
include(":captchax")
```

```kotlin
// app/build.gradle.kts
dependencies {
    implementation(project(":captchax"))
}
```

### 方式三：JitPack

```kotlin
// settings.gradle.kts
dependencyResolutionManagement {
    repositories {
        maven { url = uri("https://jitpack.io") }
    }
}

// app/build.gradle.kts
dependencies {
    implementation("com.github.user:CaptchaX-Android:1.0.0")
}
```

## 快速开始

### 1. 在 Application 中初始化 SDK

```kotlin
class MyApplication : Application() {
    override fun onCreate() {
        super.onCreate()
        
        CaptchaX.initialize(
            context = this,
            apiKey = "YOUR_API_KEY",
            apiSecret = "YOUR_API_SECRET",
            serverUrl = "https://api.captchax.com"
        )
    }
}
```

### 2. 设置全局回调（可选）

```kotlin
CaptchaX.listener = object : CaptchaListener {
    override fun onSuccess(token: String) {
        Log.d("CaptchaX", "验证成功: $token")
    }
    
    override fun onError(error: CaptchaError) {
        Log.e("CaptchaX", "验证失败: ${error.message}")
    }
    
    override fun onClose() {
        Log.d("CaptchaX", "验证窗口关闭")
    }
}
```

### 3. 调用验证

#### Kotlin 示例

```kotlin
fun login(view: View) {
    CaptchaX.verify(this, "login") { result ->
        result.onSuccess { token ->
            // 使用 token 进行后续操作
            performLogin(token)
        }.onFailure { error ->
            // 处理错误
            showError(error.message ?: "验证失败")
        }
    }
}
```

#### Java 示例

```java
public void login(View view) {
    CaptchaX.INSTANCE.verify(this, "login", result -> {
        if (result.isSuccess()) {
            String token = result.getOrNull();
            performLogin(token);
        } else {
            Exception error = result.getExceptionOrNull();
            showError(error != null ? error.getMessage() : "验证失败");
        }
    });
}
```

### 4. 使用 CaptchaView 自定义UI

```kotlin
val captchaView = CaptchaView(context)
captchaView.listener = object : CaptchaViewListener {
    override fun onSuccess(token: String) {
        Log.d("CaptchaX", "验证成功: $token")
    }
    
    override fun onError(error: CaptchaError) {
        Log.e("CaptchaX", "错误: ${error.message}")
    }
    
    override fun onClose() {
        Log.d("CaptchaX", "关闭")
    }
    
    override fun onReady() {
        Log.d("CaptchaX", "就绪")
    }
    
    override fun onLoading() {
        Log.d("CaptchaX", "加载中")
    }
    
    override fun onLoaded() {
        Log.d("CaptchaX", "已加载")
    }
}

captchaView.load(CaptchaType.SLIDER)
```

## Jetpack Compose 集成

### Compose 按钮组件

```kotlin
@Composable
fun LoginScreen() {
    CaptchaButton(
        scene = "login",
        onSuccess = { token ->
            performLogin(token)
        },
        onError = { error ->
            showError(error.message ?: "验证失败")
        }
    )
}
```

### 自定义验证码对话框

```kotlin
@Composable
fun CustomCaptchaDialog(
    onDismiss: () -> Unit,
    onSuccess: (String) -> Unit,
    onError: (CaptchaError) -> Unit
) {
    var selectedType by remember { mutableStateOf(CaptchaType.SLIDER) }
    
    CaptchaDialog(
        onDismiss = onDismiss,
        onSuccess = onSuccess,
        onError = onError
    )
}
```

## API 文档

### CaptchaX

SDK 主入口类，提供全局配置和验证方法。

#### 方法

| 方法 | 说明 |
|------|------|
| `initialize(context, apiKey, apiSecret)` | 初始化 SDK |
| `verify(activity, scene, callback)` | 请求验证码验证 |
| `preload(scene)` | 预加载验证码 |
| `destroy()` | 销毁 SDK，释放资源 |

### CaptchaConfig

SDK 配置类，使用 Builder 模式。

```kotlin
val config = CaptchaConfig.builder()
    .apiKey("YOUR_API_KEY")
    .apiSecret("YOUR_API_SECRET")
    .serverUrl("https://api.captchax.com")
    .timeout(30000L)
    .cacheEnabled(true)
    .preloadEnabled(true)
    .build()

CaptchaX.initialize(this, config)
```

### CaptchaType

验证码类型枚举。

| 类型 | 说明 |
|------|------|
| `SLIDER` | 滑块验证码 |
| `CLICK` | 点选验证码 |
| `ROTATE` | 旋转验证码 |
| `PUZZLE` | 拼图验证码 |
| `TEXT` | 文字验证码 |
| `ICON` | 图标验证码 |

### CaptchaListener

全局验证回调接口。

```kotlin
interface CaptchaListener {
    fun onSuccess(token: String)      // 验证成功
    fun onError(error: CaptchaError)  // 验证失败
    fun onClose()                     // 用户关闭验证
}
```

### CaptchaViewListener

CaptchaView 视图回调接口。

```kotlin
interface CaptchaViewListener {
    fun onSuccess(token: String)      // 验证成功
    fun onError(error: CaptchaError)  // 验证失败
    fun onClose()                     // 用户关闭验证
    fun onReady()                     // 视图就绪
    fun onLoading()                   // 开始加载
    fun onLoaded()                    // 加载完成
}
```

### CaptchaError

验证错误类型。

| 错误类型 | 说明 |
|---------|------|
| `NetworkError` | 网络错误 |
| `ServerError` | 服务器错误 |
| `ValidationError` | 验证失败 |
| `TimeoutError` | 请求超时 |
| `CancelledError` | 用户取消 |
| `UnknownError` | 未知错误 |

## 示例代码

### 完整登录流程

```kotlin
class LoginActivity : AppCompatActivity() {
    
    private lateinit var usernameInput: EditText
    private lateinit var passwordInput: EditText
    private lateinit var captchaButton: Button
    
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_login)
        
        // 初始化 CaptchaX
        CaptchaX.initialize(this, "API_KEY", "API_SECRET")
        
        // 设置回调
        CaptchaX.listener = createCaptchaListener()
        
        // 预加载验证码
        CaptchaX.preload("login")
        
        // 绑定视图
        usernameInput = findViewById(R.id.username)
        passwordInput = findViewById(R.id.password)
        captchaButton = findViewById(R.id.btnLogin)
        
        captchaButton.setOnClickListener {
            showCaptchaAndLogin()
        }
    }
    
    private fun showCaptchaAndLogin() {
        val username = usernameInput.text.toString()
        val password = passwordInput.text.toString()
        
        if (!validateInput(username, password)) {
            return
        }
        
        CaptchaX.verify(this, "login") { result ->
            result.onSuccess { token ->
                performLogin(username, password, token)
            }.onFailure { error ->
                showError("验证失败: ${error.message}")
            }
        }
    }
    
    private fun performLogin(username: String, password: String, captchaToken: String) {
        // 调用登录 API
        api.login(username, password, captchaToken) { response ->
            if (response.isSuccess) {
                navigateToMain()
            } else {
                showError("登录失败")
            }
        }
    }
    
    private fun createCaptchaListener() = object : CaptchaListener {
        override fun onSuccess(token: String) {
            Log.d("Login", "验证成功: $token")
        }
        
        override fun onError(error: CaptchaError) {
            Log.e("Login", "验证失败: ${error.message}")
        }
        
        override fun onClose() {
            Log.d("Login", "用户关闭验证")
        }
    }
}
```

### Fragment 中使用

```kotlin
class RegisterFragment : Fragment() {
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        val btnRegister = view.findViewById<Button>(R.id.btnRegister)
        btnRegister.setOnClickListener {
            CaptchaX.verify(requireActivity(), "register") { result ->
                result.onSuccess { token ->
                    registerUser(token)
                }.onFailure { error ->
                    showError(error.message ?: "验证失败")
                }
            }
        }
    }
    
    private fun registerUser(captchaToken: String) {
        // 注册逻辑
    }
}
```

### Dialog 中使用

```kotlin
fun showCaptchaDialog(activity: Activity) {
    val dialog = AlertDialog.Builder(activity)
        .setTitle("安全验证")
        .setMessage("请完成验证后继续")
        .setView(CaptchaView(activity).apply {
            load(CaptchaType.SLIDER)
        })
        .setNegativeButton("取消") { d, _ -> d.dismiss() }
        .create()
    
    dialog.show()
}
```

## ProGuard 配置

如果启用 ProGuard，请添加以下规则：

```proguard
# CaptchaX SDK
-keep class com.captchax.sdk.** { *; }

# OkHttp
-dontwarn okhttp3.**
-dontwarn okio.**
-keepnames class okhttp3.internal.publicsuffix.PublicSuffixDatabase

# Kotlin
-keep class kotlin.** { *; }
-dontwarn kotlin.**

# Coroutines
-keepnames class kotlinx.coroutines.internal.MainDispatcherFactory {}
-keepnames class kotlinx.coroutines.CoroutineExceptionHandler {}
```

## 错误处理

```kotlin
CaptchaX.verify(this, "login") { result ->
    result.fold(
        onSuccess = { token ->
            // 使用 token
        },
        onFailure = { error ->
            when (error) {
                is CaptchaError.NetworkError -> {
                    showToast("网络连接失败")
                }
                is CaptchaError.TimeoutError -> {
                    showToast("请求超时，请重试")
                }
                is CaptchaError.ValidationError -> {
                    showToast("验证失败，请重试")
                }
                is CaptchaError.ServerError -> {
                    showToast("服务器错误")
                }
                is CaptchaError.CancelledError -> {
                    // 用户取消，无需处理
                }
                else -> {
                    showToast("未知错误")
                }
            }
        }
    )
}
```

## 注意事项

1. **网络权限**：确保在 `AndroidManifest.xml` 中添加网络权限
   ```xml
   <uses-permission android:name="android.permission.INTERNET" />
   ```

2. **Cleartext Traffic**：如果使用 HTTP，请配置
   ```xml
   <application android:usesCleartextTraffic="true">
   ```

3. **Context 泄漏**：在 Activity/Fragment 销毁时调用 `CaptchaX.destroy()`

4. **线程安全**：SDK 方法可在主线程调用，内部使用协程处理异步操作

## 常见问题

### Q: 如何获取 API Key 和 Secret？
A: 请访问 CaptchaX 管理后台 (https://captchax.com/admin) 注册并创建应用。

### Q: 验证失败怎么办？
A: 请检查：
1. API Key 和 Secret 是否正确
2. 网络连接是否正常
3. 服务器地址是否正确

### Q: 如何自定义验证码样式？
A: 可以通过修改 `captcha_view.xml` 布局文件来自定义样式。

### Q: 支持哪些 Android 版本？
A: SDK 支持 Android 5.0 (API 21) 及以上版本。

## 版本历史

### v1.0.0 (2026-05-15)
- 初始版本发布
- 支持 6 种验证码类型
- 完整的 Kotlin/Java 互操作
- Jetpack Compose 支持

## License

Copyright © 2026 CaptchaX. All rights reserved.
