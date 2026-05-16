package com.hjtpx.sdk;

import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.Mock;
import org.mockito.MockitoAnnotations;

import okhttp3.*;
import java.io.IOException;
import java.util.Arrays;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

class CaptchaClientTest {
    private CaptchaClient client;
    private AutoCloseable closeable;

    @Mock
    private OkHttpClient mockHttpClient;

    @Mock
    private Call mockCall;

    @BeforeEach
    void setUp() {
        closeable = MockitoAnnotations.openMocks(this);
        Config config = new Config();
        config.setBaseUrl("http://localhost:8080");
        config.setDebugMode(true);
        client = new CaptchaClient(config);
    }

    @AfterEach
    void tearDown() throws Exception {
        closeable.close();
        if (client != null) {
            client.close();
        }
    }

    @Test
    void testConstructor() {
        CaptchaClient c = new CaptchaClient();
        assertNotNull(c);
        c.close();
    }

    @Test
    void testConstructorWithConfig() {
        Config config = new Config();
        config.setBaseUrl("http://test.com");
        config.setAppId("test-id");
        config.setAppSecret("test-secret");

        CaptchaClient c = new CaptchaClient(config);
        assertNotNull(c);
        assertEquals("http://test.com", c.getConfig().getBaseUrl());
        assertEquals("test-id", c.getConfig().getAppId());
        c.close();
    }

    @Test
    void testSetDebugMode() {
        client.setDebugMode(true);
        assertTrue(client.getConfig().isDebugMode());

        client.setDebugMode(false);
        assertFalse(client.getConfig().isDebugMode());
    }

    @Test
    void testSetTimeout() {
        client.setTimeout(60000);
        assertEquals(60000, client.getConfig().getTimeout());
    }

    @Test
    void testSetMaxRetries() {
        client.setMaxRetries(5);
        assertEquals(5, client.getConfig().getMaxRetries());
    }

    @Test
    void testGetStats() {
        PoolStats stats = client.getStats();
        assertNotNull(stats);
        assertEquals(0, stats.getTotalRequests());
        assertEquals(0, stats.getSuccessfulRequests());
        assertEquals(0, stats.getFailedRequests());
    }

    @Test
    void testExtractBase64Image() throws SDKError {
        String testData = "data:image/png;base64,SGVsbG8gV29ybGQ=";
        byte[] result = client.extractBase64Image(testData);
        assertEquals("Hello World", new String(result));
    }

    @Test
    void testExtractBase64ImageEmpty() {
        SDKError error = assertThrows(SDKError.class, () -> {
            client.extractBase64Image("");
        });
        assertEquals(400, error.getCode());
    }

    @Test
    void testExtractBase64ImageUnsupportedFormat() {
        SDKError error = assertThrows(SDKError.class, () -> {
            client.extractBase64Image("data:image/gif;base64,abc123");
        });
        assertEquals(400, error.getCode());
    }

    @Test
    void testGenerateImageCaptchaMissingParams() throws SDKError {
        ImageCaptchaResponse response = client.generateImageCaptcha();
        assertNotNull(response);
    }

    @Test
    void testVerifyImageCaptchaMissingChallengeId() {
        SDKError error = assertThrows(SDKError.class, () -> {
            client.verifyImageCaptcha("", "1234");
        });
        assertEquals(400, error.getCode());
        assertTrue(error.getMessage().contains("challenge_id"));
    }

    @Test
    void testVerifyImageCaptchaMissingAnswer() {
        SDKError error = assertThrows(SDKError.class, () -> {
            client.verifyImageCaptcha("test-id", "");
        });
        assertEquals(400, error.getCode());
        assertTrue(error.getMessage().contains("answer"));
    }

    @Test
    void testGenerateSliderCaptchaMissingParams() throws SDKError {
        SliderCaptchaResponse response = client.generateSliderCaptcha();
        assertNotNull(response);
    }

    @Test
    void testVerifySliderCaptchaMissingChallengeId() {
        SDKError error = assertThrows(SDKError.class, () -> {
            client.verifySliderCaptcha("", "120");
        });
        assertEquals(400, error.getCode());
    }

    @Test
    void testVerifySliderCaptchaMissingOffset() {
        SDKError error = assertThrows(SDKError.class, () -> {
            client.verifySliderCaptcha("test-id", "");
        });
        assertEquals(400, error.getCode());
    }

    @Test
    void testGenerateClickCaptchaMissingParams() throws SDKError {
        ClickCaptchaResponse response = client.generateClickCaptcha();
        assertNotNull(response);
    }

    @Test
    void testVerifyClickCaptchaMissingChallengeId() {
        List<ClickData> clicks = Arrays.asList(new ClickData(100, 120, 500));
        SDKError error = assertThrows(SDKError.class, () -> {
            client.verifyClickCaptcha("", clicks);
        });
        assertEquals(400, error.getCode());
    }

    @Test
    void testVerifyClickCaptchaMissingClicks() {
        SDKError error = assertThrows(SDKError.class, () -> {
            client.verifyClickCaptcha("test-id", null);
        });
        assertEquals(400, error.getCode());
    }

    @Test
    void testVerifyClickCaptchaEmptyClicks() {
        SDKError error = assertThrows(SDKError.class, () -> {
            client.verifyClickCaptcha("test-id", Arrays.asList());
        });
        assertEquals(400, error.getCode());
    }

    @Test
    void testStaticCreate() {
        CaptchaClient c = CaptchaClient.create("http://test.com");
        assertNotNull(c);
        c.close();
    }

    @Test
    void testStaticCreateWithCredentials() {
        CaptchaClient c = CaptchaClient.create("http://test.com", "id", "secret");
        assertNotNull(c);
        assertEquals("http://test.com", c.getConfig().getBaseUrl());
        assertEquals("id", c.getConfig().getAppId());
        assertEquals("secret", c.getConfig().getAppSecret());
        c.close();
    }
}
