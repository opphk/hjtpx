package com.captchax.sdk;

import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.Assert.*;

public class CaptchaXClientTest {

    @Test(expected = IllegalArgumentException.class)
    public void testConstructorWithNullBaseUrl() {
        new CaptchaXClient((String) null);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testConstructorWithEmptyBaseUrl() {
        new CaptchaXClient("");
    }

    @Test
    public void testConstructorWithValidConfig() {
        CaptchaConfig config = new CaptchaConfig.Builder()
                .baseUrl("https://captchax.example.com")
                .appId("test-app")
                .timeout(5000)
                .retryTimes(2)
                .apiVersion(ApiVersion.V2)
                .build();

        CaptchaXClient client = new CaptchaXClient(config);
        assertEquals(ApiVersion.V2, client.getApiVersion());
    }

    @Test
    public void testSetAppId() {
        CaptchaXClient client = new CaptchaXClient("https://captchax.example.com");
        client.setAppId("new-app-id");
    }

    @Test
    public void testSetApiVersion() {
        CaptchaXClient client = new CaptchaXClient("https://captchax.example.com");
        assertEquals(ApiVersion.V1, client.getApiVersion());

        client.setApiVersion(ApiVersion.V2);
        assertEquals(ApiVersion.V2, client.getApiVersion());
    }

    @Test
    public void testCreateClientInfo() {
        CaptchaXClient client = new CaptchaXClient("https://captchax.example.com");
        String info = client.createClientInfo(null);
        assertNotNull(info);
        assertTrue(info.contains("platform"));
        assertTrue(info.contains("timestamp"));

        Map<String, Object> extra = new HashMap<>();
        extra.put("customField", "value");
        info = client.createClientInfo(extra);
        assertTrue(info.contains("customField"));
        assertTrue(info.contains("value"));
    }
}

class CaptchaXExceptionTest {

    @Test
    public void testExceptionWithMessage() {
        CaptchaXException exception = new CaptchaXException("Test error");
        assertEquals("Test error", exception.getMessage());
        assertEquals(500, exception.getCode());
        assertEquals(500, exception.getStatusCode());
    }

    @Test
    public void testExceptionWithCode() {
        CaptchaXException exception = new CaptchaXException("Test error", 400, 400);
        assertEquals("Test error", exception.getMessage());
        assertEquals(400, exception.getCode());
        assertEquals(400, exception.getStatusCode());
    }

    @Test
    public void testExceptionWithDetails() {
        CaptchaXException exception = new CaptchaXException("Test error", 400, 400, "details");
        assertEquals("details", exception.getDetails());
    }

    @Test
    public void testExceptionWithCause() {
        Exception cause = new RuntimeException("cause");
        CaptchaXException exception = new CaptchaXException("Test error", cause);
        assertEquals(cause, exception.getCause());
    }
}

class ApiVersionTest {

    @Test
    public void testApiVersionValues() {
        assertEquals("v1", ApiVersion.V1.getValue());
        assertEquals("v2", ApiVersion.V2.getValue());
    }
}

class CaptchaConfigTest {

    @Test
    public void testConfigBuilder() {
        CaptchaConfig config = new CaptchaConfig.Builder()
                .baseUrl("https://captchax.example.com")
                .appId("test-app")
                .timeout(5000)
                .retryTimes(2)
                .apiVersion(ApiVersion.V2)
                .build();

        assertEquals("https://captchax.example.com", config.getBaseUrl());
        assertEquals("test-app", config.getAppId());
        assertEquals(5000, config.getTimeout());
        assertEquals(2, config.getRetryTimes());
        assertEquals(ApiVersion.V2, config.getApiVersion());
    }

    @Test
    public void testConfigDefaults() {
        CaptchaConfig config = new CaptchaConfig("https://example.com");
        assertEquals(10000, config.getTimeout());
        assertEquals(3, config.getRetryTimes());
        assertEquals(ApiVersion.V1, config.getApiVersion());
    }
}
