<?php

namespace HJTPX\Captcha\Pool;

use GuzzleHttp\Client;
use GuzzleHttp\HandlerStack;
use GuzzleHttp\Handler\CurlHandler;
use GuzzleHttp\Middleware;

class ConnectionPoolManager
{
    protected $config;
    protected $client;

    public function __construct(ConnectionPoolConfig $config)
    {
        $this->config = $config;
        $this->initializeClient();
    }

    protected function initializeClient(): void
    {
        $stack = HandlerStack::create(new CurlHandler());

        $this->client = new Client([
            'handler' => $stack,
            'timeout' => $this->config->requestTimeout,
            'connect_timeout' => $this->config->connectionTimeout,
            'verify' => true,
            'http_errors' => false,
        ]);
    }

    public function getClient(): Client
    {
        return $this->client;
    }

    public function close(): void
    {
        // Guzzle handles connection pooling automatically, no explicit close needed
    }
}
