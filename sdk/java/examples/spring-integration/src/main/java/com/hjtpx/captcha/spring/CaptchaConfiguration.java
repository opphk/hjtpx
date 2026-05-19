package com.hjtpx.captcha.spring;

import com.hjtpx.captcha.client.CaptchaClient;
import com.hjtpx.captcha.client.CaptchaClientConfig;
import com.hjtpx.captcha.exception.CaptchaException;
import com.hjtpx.captcha.model.*;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class CaptchaConfiguration {

    @Value("${captcha.base-url:http://localhost:8080}")
    private String baseUrl;

    @Value("${captcha.api-key:}")
    private String apiKey;

    @Value("${captcha.secret-key:}")
    private String secretKey;

    @Value("${captcha.timeout:30000}")
    private int timeout;

    @Value("${captcha.max-retries:3}")
    private int maxRetries;

    @Bean
    public CaptchaClientConfig captchaClientConfig() {
        CaptchaClientConfig config = new CaptchaClientConfig();
        config.setBaseUrl(baseUrl);
        config.setApiKey(apiKey);
        config.setSecretKey(secretKey);
        config.setTimeout(timeout);
        config.setMaxRetries(maxRetries);
        return config;
    }

    @Bean
    public CaptchaClient captchaClient(CaptchaClientConfig config) {
        return new CaptchaClient(config);
    }
}
