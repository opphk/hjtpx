<?php

namespace HJTPX\Captcha\Retry;

use GuzzleHttp\Exception\ConnectException;
use GuzzleHttp\Exception\RequestException;

class RetryManager
{
    protected $config;
    protected $retryCount;

    public function __construct(RetryConfig $config)
    {
        $this->config = $config;
        $this->retryCount = 0;
    }

    public function execute(callable $operation, callable $onRetry = null)
    {
        $attempt = 0;
        $lastException = null;

        while ($attempt <= $this->config->maxRetries) {
            try {
                $result = $operation();
                $this->retryCount = $attempt;
                return $result;
            } catch (\Exception $e) {
                $lastException = $e;
                $attempt++;

                if ($attempt > $this->config->maxRetries) {
                    break;
                }

                if (!$this->isRetryable($e)) {
                    break;
                }

                $delay = $this->calculateDelay($attempt);
                
                if ($onRetry) {
                    $onRetry($attempt, $e, $delay);
                }

                usleep($delay * 1000);
            }
        }

        throw $lastException;
    }

    protected function isRetryable(\Exception $e): bool
    {
        if ($e instanceof ConnectException) {
            return true;
        }

        if ($e instanceof RequestException) {
            $response = $e->getResponse();
            if ($response) {
                $statusCode = $response->getStatusCode();
                return in_array($statusCode, $this->config->retryableStatusCodes, true);
            }
            return true;
        }

        if ($e instanceof \RuntimeException) {
            return true;
        }

        return false;
    }

    protected function calculateDelay(int $attempt): int
    {
        $delay = (int) ($this->config->retryDelay * pow($this->config->retryMultiplier, $attempt - 1));
        return min($delay, $this->config->maxRetryDelay);
    }

    public function getRetryCount(): int
    {
        return $this->retryCount;
    }

    public function resetRetryCount(): void
    {
        $this->retryCount = 0;
    }

    public function getConfig(): RetryConfig
    {
        return $this->config;
    }
}
