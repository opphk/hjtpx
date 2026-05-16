# hjtpx Java SDK

A comprehensive Java SDK for hjtpx captcha services, providing support for image captcha, slider captcha, and click captcha with advanced features like automatic retries, connection pooling, and detailed error handling.

## Features

- **Multiple Captcha Types**: Support for image captcha, slider captcha, and click captcha
- **Automatic Retries**: Built-in retry mechanism with exponential backoff for failed requests
- **Comprehensive Error Handling**: Detailed error types and error code extraction utilities
- **Statistics & Monitoring**: Track request statistics including success rate, retry count, and error tracking
- **Connection Pooling**: Efficient HTTP connection management with configurable pool settings
- **Configurable**: Extensive configuration options for timeouts, pool sizes, and retry behavior
- **Java 8+ Support**: Compatible with Java 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, and 19

## Installation

### Maven

Add this dependency to your `pom.xml`:

```xml
<dependency>
    <groupId>com.hjtpx</groupId>
    <artifactId>hjtpx-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Gradle

Add this dependency to your `build.gradle`:

```groovy
implementation 'com.hjtpx:hjtpx-sdk:1.0.0'
```

### Manual Installation

Download the JAR from the releases page and add it to your classpath.

## Quick Start

```java
import com.hjtpx.sdk.*;

public class Example {
    public static void main(String[] args) {
        Config config = new Config();
        config.setBaseUrl("http://localhost:8080");
        config.setAppId("your-app-id");
        config.setAppSecret("your-app-secret");

        try (CaptchaClient client = new CaptchaClient(config)) {
            ImageCaptchaResponse captcha = client.generateImageCaptcha();
            System.out.println("Challenge ID: " + captcha.getChallengeId());

            VerifyImageCaptchaResponse result = client.verifyImageCaptcha(
                captcha.getChallengeId(),
                "1234"
            );
            System.out.println("Success: " + result.isSuccess());
        } catch (SDKError e) {
            System.err.println("SDK Error " + e.getCode() + ": " + e.getMessage());
        }
    }
}
```

## Configuration

### Basic Configuration

```java
Config config = new Config();
config.setBaseUrl("http://localhost:8080");
config.setAppId("your-app-id");
config.setAppSecret("your-app-secret");
config.setTimeout(30000);
config.setMaxRetries(3);
config.setDebugMode(false);

CaptchaClient client = new CaptchaClient(config);
```

### Using Builder Pattern

```java
Config config = Config.builder()
    .baseUrl("http://localhost:8080")
    .appId("your-app-id")
    .appSecret("your-app-secret")
    .timeout(30000)
    .maxRetries(3)
    .retryDelay(100)
    .debugMode(true)
    .build();

CaptchaClient client = new CaptchaClient(config);
```

### Configuration Options

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| baseUrl | String | http://localhost:8080 | API endpoint |
| appId | String | | Application ID |
| appSecret | String | | Application Secret |
| timeout | int | 30000 | Request timeout in milliseconds |
| maxRetries | int | 3 | Maximum retry attempts |
| retryDelay | long | 100 | Base retry delay in milliseconds |
| maxIdleConns | int | 10 | Maximum idle connections |
| maxOpenConns | int | 100 | Maximum open connections |
| debugMode | boolean | false | Enable debug logging |

## API Reference

### CaptchaClient

The main client for interacting with the captcha service.

#### Constructor

```java
CaptchaClient client = new CaptchaClient(config);
```

Creates a new captcha client with the specified configuration.

#### AutoCloseable

The client implements `AutoCloseable` and should be used with try-with-resources:

```java
try (CaptchaClient client = new CaptchaClient(config)) {
    // Use client
}
```

### Captcha Generation Methods

#### generateImageCaptcha

```java
ImageCaptchaResponse generateImageCaptcha() throws SDKError
ImageCaptchaResponse generateImageCaptcha(ImageCaptchaRequest request) throws SDKError
```

Generates a new image captcha challenge.

```java
ImageCaptchaRequest request = new ImageCaptchaRequest();
request.setType(CaptchaType.NUMBER);
request.setCount(4);
request.setNoiseMode(2);
request.setLineMode(1);

ImageCaptchaResponse captcha = client.generateImageCaptcha(request);
```

#### generateSliderCaptcha

```java
SliderCaptchaResponse generateSliderCaptcha() throws SDKError
SliderCaptchaResponse generateSliderCaptcha(SliderCaptchaRequest request) throws SDKError
```

Generates a new slider captcha challenge.

```java
SliderCaptchaRequest request = new SliderCaptchaRequest(360, 220);
SliderCaptchaResponse slider = client.generateSliderCaptcha(request);
```

#### generateClickCaptcha

```java
ClickCaptchaResponse generateClickCaptcha() throws SDKError
ClickCaptchaResponse generateClickCaptcha(ClickCaptchaRequest request) throws SDKError
```

Generates a new click captcha challenge.

```java
ClickCaptchaRequest request = new ClickCaptchaRequest(360, 220, 4);
ClickCaptchaResponse click = client.generateClickCaptcha(request);
```

### Captcha Verification Methods

#### verifyImageCaptcha

```java
VerifyImageCaptchaResponse verifyImageCaptcha(String challengeId, String answer) throws SDKError
```

Verifies an image captcha answer.

```java
VerifyImageCaptchaResponse result = client.verifyImageCaptcha(
    captcha.getChallengeId(),
    "1234"
);
System.out.println("Success: " + result.isSuccess());
```

#### verifySliderCaptcha

```java
VerifyCaptchaResponse verifySliderCaptcha(String challengeId, String offset) throws SDKError
```

Verifies a slider captcha solution.

```java
VerifyCaptchaResponse result = client.verifySliderCaptcha(
    slider.getChallengeId(),
    "120"
);
System.out.println("Success: " + result.isSuccess());
System.out.println("Risk Score: " + result.getScore());
```

#### verifyClickCaptcha

```java
VerifyCaptchaResponse verifyClickCaptcha(String challengeId, List<ClickData> clicks) throws SDKError
```

Verifies a click captcha solution.

```java
List<ClickData> clicks = Arrays.asList(
    new ClickData(100, 120, 500),
    new ClickData(200, 150, 300)
);

VerifyCaptchaResponse result = client.verifyClickCaptcha(
    click.getChallengeId(),
    clicks
);
System.out.println("Success: " + result.isSuccess());
```

### Utility Methods

#### extractBase64Image

```java
byte[] extractBase64Image(String dataUri) throws SDKError
```

Extracts raw image bytes from a base64 data URI.

```java
byte[] imageData = client.extractBase64Image(captcha.getImage());
Files.write(Paths.get("captcha.png"), imageData);
```

#### getStats

```java
PoolStats getStats()
```

Returns current pool statistics.

```java
PoolStats stats = client.getStats();
System.out.println("Total Requests: " + stats.getTotalRequests());
System.out.println("Success Rate: " + stats.getSuccessRate() + "%");
```

### Runtime Configuration

```java
client.setDebugMode(true);
client.setTimeout(60000);
client.setMaxRetries(5);
```

## Error Handling

The SDK provides comprehensive error handling with specific exception types:

### Exception Types

```java
import com.hjtpx.sdk.*;

SDKError              // Base exception
SDKErrorWithRetry    // Error with retry information
NetworkError         // Network connectivity issues
TimeoutError         // Request timeout
InvalidParamsError   // Invalid parameters
RateLimitedError     // Rate limited (includes retry-after)
UnauthorizedError    // Authentication failed
ServerError          // Server-side errors
```

### Example

```java
try (CaptchaClient client = new CaptchaClient(config)) {
    ImageCaptchaResponse captcha = client.generateImageCaptcha();
    VerifyImageCaptchaResponse result = client.verifyImageCaptcha(
        captcha.getChallengeId(),
        "1234"
    );
} catch (RateLimitedError e) {
    System.err.println("Rate limited, retry after " + e.getRetryAfter() + " seconds");
} catch (TimeoutError e) {
    System.err.println("Request timed out");
} catch (UnauthorizedError e) {
    System.err.println("Unauthorized - check credentials");
} catch (SDKError e) {
    System.err.println("SDK Error " + e.getCode() + ": " + e.getMessage());
} catch (Exception e) {
    System.err.println("Unexpected error: " + e.getMessage());
}
```

### Error Checking

```java
try {
    VerifyImageCaptchaResponse result = client.verifyImageCaptcha(challengeId, answer);
} catch (SDKError e) {
    if (e.isRateLimited()) {
        System.out.println("Rate limited");
    } else if (e.isUnauthorized()) {
        System.out.println("Unauthorized");
    } else if (e.isServerError()) {
        System.out.println("Server error");
    } else if (e.isInvalidParams()) {
        System.out.println("Invalid parameters");
    }
}
```

## Complete Examples

### Image Captcha Example

```java
import com.hjtpx.sdk.*;
import java.nio.file.*;

public class ImageCaptchaExample {
    public static void main(String[] args) {
        Config config = new Config();
        config.setBaseUrl("http://localhost:8080");

        try (CaptchaClient client = new CaptchaClient(config)) {
            ImageCaptchaRequest request = new ImageCaptchaRequest();
            request.setType(CaptchaType.MIXED);
            request.setCount(4);
            request.setNoiseMode(2);
            request.setLineMode(1);

            ImageCaptchaResponse captcha = client.generateImageCaptcha(request);
            System.out.println("Challenge ID: " + captcha.getChallengeId());

            byte[] imageData = client.extractBase64Image(captcha.getImage());
            Files.write(Paths.get("captcha.png"), imageData);
            System.out.println("Captcha image saved to captcha.png");

            Scanner scanner = new Scanner(System.in);
            System.out.print("Enter the captcha text: ");
            String userInput = scanner.nextLine();

            VerifyImageCaptchaResponse result = client.verifyImageCaptcha(
                captcha.getChallengeId(),
                userInput
            );
            System.out.println("Verification result: " + result.isSuccess());

        } catch (SDKError e) {
            System.err.println("SDK Error: " + e.getMessage());
        } catch (Exception e) {
            System.err.println("Error: " + e.getMessage());
        }
    }
}
```

### Slider Captcha Example

```java
import com.hjtpx.sdk.*;
import java.nio.file.*;

public class SliderCaptchaExample {
    public static void main(String[] args) {
        Config config = new Config();
        config.setBaseUrl("http://localhost:8080");

        try (CaptchaClient client = new CaptchaClient(config)) {
            SliderCaptchaRequest request = new SliderCaptchaRequest(360, 220);
            SliderCaptchaResponse slider = client.generateSliderCaptcha(request);
            System.out.println("Challenge ID: " + slider.getChallengeId());

            byte[] bgData = client.extractBase64Image(slider.getBackgroundImage());
            byte[] sliderData = client.extractBase64Image(slider.getSliderImage());

            Files.write(Paths.get("bg.png"), bgData);
            Files.write(Paths.get("slider.png"), sliderData);

            Scanner scanner = new Scanner(System.in);
            System.out.print("Enter slider offset: ");
            String offset = scanner.nextLine();

            VerifyCaptchaResponse result = client.verifySliderCaptcha(
                slider.getChallengeId(),
                offset
            );
            System.out.println("Verification result: " + result.isSuccess());
            System.out.println("Risk Score: " + result.getScore());

        } catch (SDKError e) {
            System.err.println("SDK Error: " + e.getMessage());
        } catch (Exception e) {
            System.err.println("Error: " + e.getMessage());
        }
    }
}
```

### Click Captcha Example

```java
import com.hjtpx.sdk.*;
import java.nio.file.*;
import java.util.*;

public class ClickCaptchaExample {
    public static void main(String[] args) {
        Config config = new Config();
        config.setBaseUrl("http://localhost:8080");

        try (CaptchaClient client = new CaptchaClient(config)) {
            ClickCaptchaRequest request = new ClickCaptchaRequest(360, 220, 4);
            ClickCaptchaResponse click = client.generateClickCaptcha(request);
            System.out.println("Challenge ID: " + click.getChallengeId());
            System.out.println("Target Index: " + click.getTargetIndex());

            byte[] imageData = client.extractBase64Image(click.getBackgroundImage());
            Files.write(Paths.get("click_captcha.png"), imageData);

            List<ClickData> clicks = new ArrayList<>();
            List<Integer> targetPos = click.getTargetPosition();
            clicks.add(new ClickData(targetPos.get(0), targetPos.get(1), 500));

            VerifyCaptchaResponse result = client.verifyClickCaptcha(
                click.getChallengeId(),
                clicks
            );
            System.out.println("Verification result: " + result.isSuccess());

        } catch (SDKError e) {
            System.err.println("SDK Error: " + e.getMessage());
        } catch (Exception e) {
            System.err.println("Error: " + e.getMessage());
        }
    }
}
```

## Testing

Run tests with Maven:

```bash
cd sdk/java
mvn test
```

Run with coverage:

```bash
mvn test jacoco:report
```

## Building

Build the SDK:

```bash
mvn clean package
```

Build with Javadoc:

```bash
mvn clean package javadoc:javadoc
```

## Dependencies

- **Jackson** (2.15.2): JSON processing
- **OkHttp** (4.11.0): HTTP client
- **SLF4J** (2.0.7): Logging facade

## License

MIT License
