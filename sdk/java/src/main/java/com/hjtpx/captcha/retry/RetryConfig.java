package com.hjtpx.captcha.retry;

import java.util.ArrayList;
import java.util.List;

public class RetryConfig {
    private int maxRetries = 3;
    private long initialDelayMs = 100;
    private long maxDelayMs = 10000;
    private double backoffMultiplier = 2.0;
    private List<Class<? extends Exception>> retryableExceptions = new ArrayList<>();
    private List<Integer> retryableStatusCodes = new ArrayList<>();

    public RetryConfig() {
        retryableStatusCodes.add(429);
        retryableStatusCodes.add(500);
        retryableStatusCodes.add(502);
        retryableStatusCodes.add(503);
        retryableStatusCodes.add(504);
    }

    public int getMaxRetries() {
        return maxRetries;
    }

    public void setMaxRetries(int maxRetries) {
        this.maxRetries = maxRetries;
    }

    public long getInitialDelayMs() {
        return initialDelayMs;
    }

    public void setInitialDelayMs(long initialDelayMs) {
        this.initialDelayMs = initialDelayMs;
    }

    public long getMaxDelayMs() {
        return maxDelayMs;
    }

    public void setMaxDelayMs(long maxDelayMs) {
        this.maxDelayMs = maxDelayMs;
    }

    public double getBackoffMultiplier() {
        return backoffMultiplier;
    }

    public void setBackoffMultiplier(double backoffMultiplier) {
        this.backoffMultiplier = backoffMultiplier;
    }

    public List<Class<? extends Exception>> getRetryableExceptions() {
        return retryableExceptions;
    }

    public void setRetryableExceptions(List<Class<? extends Exception>> retryableExceptions) {
        this.retryableExceptions = retryableExceptions;
    }

    public List<Integer> getRetryableStatusCodes() {
        return retryableStatusCodes;
    }

    public void setRetryableStatusCodes(List<Integer> retryableStatusCodes) {
        this.retryableStatusCodes = retryableStatusCodes;
    }

    public void addRetryableException(Class<? extends Exception> exceptionClass) {
        retryableExceptions.add(exceptionClass);
    }

    public void addRetryableStatusCode(int statusCode) {
        retryableStatusCodes.add(statusCode);
    }

    public boolean isRetryableStatusCode(int statusCode) {
        return retryableStatusCodes.contains(statusCode);
    }

    public boolean isRetryableException(Exception e) {
        for (Class<? extends Exception> exceptionClass : retryableExceptions) {
            if (exceptionClass.isInstance(e)) {
                return true;
            }
        }
        return false;
    }

    public long calculateDelay(int attempt) {
        long delay = (long) (initialDelayMs * Math.pow(backoffMultiplier, attempt));
        return Math.min(delay, maxDelayMs);
    }
}
