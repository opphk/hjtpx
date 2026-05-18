<?php

namespace HJTPX\Captcha\Retry;

class RetryConfig
{
    public $maxRetries;
    public $retryDelay;
    public $retryMultiplier;
    public $retryableStatusCodes;
    public $maxRetryDelay;
    public $enableExponentialBackoff;

    public function __construct(
        int $maxRetries = 3,
        int $retryDelay = 100,
        float $retryMultiplier = 2.0,
        array $retryableStatusCodes = [429, 500, 502, 503, 504],
        int $maxRetryDelay = 30000,
        bool $enableExponentialBackoff = true
    ) {
        $this->maxRetries = $maxRetries;
        $this->retryDelay = $retryDelay;
        $this->retryMultiplier = $retryMultiplier;
        $this->retryableStatusCodes = $retryableStatusCodes;
        $this->maxRetryDelay = $maxRetryDelay;
        $this->enableExponentialBackoff = $enableExponentialBackoff;
    }

    public function toArray(): array
    {
        return [
            'max_retries' => $this->maxRetries,
            'retry_delay' => $this->retryDelay,
            'retry_multiplier' => $this->retryMultiplier,
            'retryable_status_codes' => $this->retryableStatusCodes,
            'max_retry_delay' => $this->maxRetryDelay,
            'enable_exponential_backoff' => $this->enableExponentialBackoff,
        ];
    }

    public static function fromArray(array $data): self
    {
        return new self(
            $data['max_retries'] ?? 3,
            $data['retry_delay'] ?? 100,
            $data['retry_multiplier'] ?? 2.0,
            $data['retryable_status_codes'] ?? [429, 500, 502, 503, 504],
            $data['max_retry_delay'] ?? 30000,
            $data['enable_exponential_backoff'] ?? true
        );
    }

    public function withMaxRetries(int $maxRetries): self
    {
        $clone = clone $this;
        $clone->maxRetries = $maxRetries;
        return $clone;
    }

    public function withRetryDelay(int $delay): self
    {
        $clone = clone $this;
        $clone->retryDelay = $delay;
        return $clone;
    }

    public function withRetryMultiplier(float $multiplier): self
    {
        $clone = clone $this;
        $clone->retryMultiplier = $multiplier;
        return $clone;
    }

    public function withRetryableStatusCodes(array $codes): self
    {
        $clone = clone $this;
        $clone->retryableStatusCodes = $codes;
        return $clone;
    }
}
