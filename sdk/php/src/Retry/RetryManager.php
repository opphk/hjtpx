<?php

namespace HJTPX\Captcha\Retry;

class RetryManager
{
    protected $config;

    public function __construct(RetryConfig $config)
    {
        $this->config = $config;
    }

    public function execute(callable $operation)
    {
        $attempt = 0;
        $lastException = null;

        while ($attempt <= $this->config->maxRetries) {
            try {
                return $operation();
            } catch (\Exception $e) {
                $lastException = $e;
                $attempt++;

                if ($attempt > $this->config->maxRetries) {
                    break;
                }

                $delay = $this->calculateDelay($attempt);
                usleep($delay * 1000);
            }
        }

        throw $lastException;
    }

    protected function calculateDelay(int $attempt): int
    {
        return (int) ($this->config->retryDelay * pow($this->config->retryMultiplier, $attempt - 1));
    }
}
