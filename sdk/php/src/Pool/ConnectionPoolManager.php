<?php

namespace HJTPX\Captcha\Pool;

use GuzzleHttp\Client;
use GuzzleHttp\HandlerStack;
use GuzzleHttp\Handler\CurlHandler;
use GuzzleHttp\Middleware;
use GuzzleHttp\Exception\ConnectException;

class ConnectionPoolManager
{
    protected $config;
    protected $client;
    protected $stats;

    public function __construct(ConnectionPoolConfig $config)
    {
        $this->config = $config;
        $this->stats = [
            'requests_total' => 0,
            'requests_success' => 0,
            'requests_failed' => 0,
            'connections_created' => 0,
            'last_request_time' => null,
        ];
        $this->initializeClient();
    }

    protected function initializeClient(): void
    {
        $stack = HandlerStack::create(new CurlHandler());

        $stack->push(
            Middleware::mapRequest(function ($request) {
                $this->stats['requests_total']++;
                $this->stats['connections_created']++;
                return $request;
            })
        );

        $clientConfig = [
            'handler' => $stack,
            'timeout' => $this->config->requestTimeout,
            'connect_timeout' => $this->config->connectionTimeout,
            'verify' => $this->config->sslVerify,
            'http_errors' => false,
        ];

        if ($this->config->proxy) {
            $clientConfig['proxy'] = $this->config->proxy;
        }

        if ($this->config->keepAlive) {
            $clientConfig['keepalive'] = true;
        }

        $this->client = new Client($clientConfig);
    }

    public function getClient(): Client
    {
        return $this->client;
    }

    public function getStats(): array
    {
        return $this->stats;
    }

    public function getSuccessRate(): float
    {
        if ($this->stats['requests_total'] === 0) {
            return 0.0;
        }
        return round(
            ($this->stats['requests_success'] / $this->stats['requests_total']) * 100,
            2
        );
    }

    public function recordSuccess(): void
    {
        $this->stats['requests_success']++;
        $this->stats['last_request_time'] = time();
    }

    public function recordFailure(): void
    {
        $this->stats['requests_failed']++;
        $this->stats['last_request_time'] = time();
    }

    public function resetStats(): void
    {
        $this->stats = [
            'requests_total' => 0,
            'requests_success' => 0,
            'requests_failed' => 0,
            'connections_created' => 0,
            'last_request_time' => null,
        ];
    }

    public function getConfig(): ConnectionPoolConfig
    {
        return $this->config;
    }

    public function updateConfig(ConnectionPoolConfig $config): void
    {
        $this->config = $config;
        $this->initializeClient();
    }

    public function isHealthy(): bool
    {
        if ($this->stats['last_request_time'] === null) {
            return true;
        }

        $idleTime = time() - $this->stats['last_request_time'];
        return $idleTime < ($this->config->connectionTimeout * 2);
    }

    public function close(): void
    {
        if ($this->client instanceof Client) {
            $this->client = null;
        }
    }

    public function __destruct()
    {
        $this->close();
    }
}
