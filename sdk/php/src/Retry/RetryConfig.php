<?php

namespace HJTPX\Captcha\Retry;

class RetryConfig
{
    public $maxRetries;
    public $retryDelay;
    public $retryMultiplier;
    public $retryableStatusCodes;

    public function __construct(
        int $maxRetries = 3,
        int $retryDelay = 100,
        float $retryMultiplier = 2.0,
        array $retryableStatusCodes = [429, 500, 502, 503, 504]
    ) {
        $this->maxRetries = $maxRetries;
        $this->retryDelay = $retryDelay;
        $this->retryMultiplier = $retryMultiplier;
        $this->retryableStatusCodes = $retryableStatusCodes;
    }
}
