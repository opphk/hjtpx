<?php

namespace HJTPX\Captcha\Pool;

class ConnectionPoolConfig
{
    public $maxConnections;
    public $connectionTimeout;
    public $requestTimeout;
    public $proxy;
    public $sslVerify;
    public $maxConcurrentRequests;
    public $keepAlive;

    public function __construct(
        int $maxConnections = 10,
        int $connectionTimeout = 10,
        int $requestTimeout = 30,
        string $proxy = null,
        bool $sslVerify = true,
        int $maxConcurrentRequests = 5,
        bool $keepAlive = true
    ) {
        $this->maxConnections = $maxConnections;
        $this->connectionTimeout = $connectionTimeout;
        $this->requestTimeout = $requestTimeout;
        $this->proxy = $proxy;
        $this->sslVerify = $sslVerify;
        $this->maxConcurrentRequests = $maxConcurrentRequests;
        $this->keepAlive = $keepAlive;
    }

    public function toArray(): array
    {
        return [
            'max_connections' => $this->maxConnections,
            'connection_timeout' => $this->connectionTimeout,
            'request_timeout' => $this->requestTimeout,
            'proxy' => $this->proxy,
            'ssl_verify' => $this->sslVerify,
            'max_concurrent_requests' => $this->maxConcurrentRequests,
            'keep_alive' => $this->keepAlive,
        ];
    }

    public static function fromArray(array $data): self
    {
        return new self(
            $data['max_connections'] ?? 10,
            $data['connection_timeout'] ?? 10,
            $data['request_timeout'] ?? 30,
            $data['proxy'] ?? null,
            $data['ssl_verify'] ?? true,
            $data['max_concurrent_requests'] ?? 5,
            $data['keep_alive'] ?? true
        );
    }

    public function withProxy(string $proxy): self
    {
        $clone = clone $this;
        $clone->proxy = $proxy;
        return $clone;
    }

    public function withSslVerify(bool $verify): self
    {
        $clone = clone $this;
        $clone->sslVerify = $verify;
        return $clone;
    }

    public function withTimeouts(int $connectionTimeout, int $requestTimeout): self
    {
        $clone = clone $this;
        $clone->connectionTimeout = $connectionTimeout;
        $clone->requestTimeout = $requestTimeout;
        return $clone;
    }
}
