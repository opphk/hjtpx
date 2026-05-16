package com.hjtpx.sdk;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

class SDKErrorTest {
    @Test
    void testSDKErrorConstruction() {
        SDKError error = new SDKError(500, "Test error");
        assertEquals(500, error.getCode());
        assertEquals("Test error", error.getMessage());
    }

    @Test
    void testSDKErrorWithCause() {
        Exception cause = new RuntimeException("Cause error");
        SDKError error = new SDKError(500, "Test error", cause);
        assertEquals(500, error.getCode());
        assertEquals("Test error", error.getMessage());
        assertEquals(cause, error.getCauseException());
    }

    @Test
    void testSDKErrorToString() {
        SDKError error = new SDKError(404, "Not found");
        String str = error.toString();
        assertTrue(str.contains("404"));
        assertTrue(str.contains("Not found"));
    }

    @Test
    void testSDKErrorWithRetryGetRetryAfter() {
        SDKErrorWithRetry error = new SDKErrorWithRetry(429, "Rate limited", 60);
        assertEquals(60, error.getRetryAfter());
        assertTrue(error.isRateLimited());
    }

    @Test
    void testRateLimitedError() {
        RateLimitedError error = new RateLimitedError(30);
        assertEquals(429, error.getCode());
        assertEquals(30, error.getRetryAfter());
    }

    @Test
    void testUnauthorizedError() {
        UnauthorizedError error = new UnauthorizedError();
        assertEquals(401, error.getCode());
        assertTrue(error.isUnauthorized());
    }

    @Test
    void testTimeoutError() {
        TimeoutError error = new TimeoutError("Request timed out");
        assertEquals(408, error.getCode());
    }

    @Test
    void testNetworkError() {
        NetworkError error = new NetworkError("Connection failed");
        assertEquals(0, error.getCode());
    }

    @Test
    void testInvalidParamsError() {
        InvalidParamsError error = new InvalidParamsError("Invalid param");
        assertEquals(400, error.getCode());
        assertTrue(error.isInvalidParams());
    }

    @Test
    void testServerError() {
        ServerError error = new ServerError(500);
        assertEquals(500, error.getCode());
        assertTrue(error.isServerError());
    }
}
