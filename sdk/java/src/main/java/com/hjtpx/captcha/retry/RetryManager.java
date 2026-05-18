package com.hjtpx.captcha.retry;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.concurrent.Callable;

public class RetryManager {
    private static final Logger logger = LoggerFactory.getLogger(RetryManager.class);
    private final RetryConfig config;

    public RetryManager() {
        this(new RetryConfig());
    }

    public RetryManager(RetryConfig config) {
        this.config = config;
    }

    public <T> T execute(Callable<T> callable) throws Exception {
        int attempt = 0;
        Exception lastException = null;

        while (attempt <= config.getMaxRetries()) {
            try {
                return callable.call();
            } catch (Exception e) {
                lastException = e;
                attempt++;

                if (attempt > config.getMaxRetries()) {
                    logger.warn("Max retries ({}) reached, giving up", config.getMaxRetries());
                    break;
                }

                if (!shouldRetry(e)) {
                    logger.warn("Exception not retryable, giving up: {}", e.getMessage());
                    break;
                }

                long delay = config.calculateDelay(attempt - 1);
                logger.info("Retry attempt {}/{}, waiting {}ms", attempt, config.getMaxRetries(), delay);

                try {
                    Thread.sleep(delay);
                } catch (InterruptedException ie) {
                    Thread.currentThread().interrupt();
                    throw e;
                }
            }
        }

        throw lastException;
    }

    private boolean shouldRetry(Exception e) {
        return config.isRetryableException(e) || isNetworkException(e);
    }

    private boolean isNetworkException(Exception e) {
        String message = e.getMessage();
        if (message == null) {
            return false;
        }
        String lowerMessage = message.toLowerCase();
        return lowerMessage.contains("timeout") ||
               lowerMessage.contains("connection reset") ||
               lowerMessage.contains("connection refused") ||
               lowerMessage.contains("socket hang up") ||
               lowerMessage.contains("network");
    }

    public RetryConfig getConfig() {
        return config;
    }
}
