<?php

namespace HJTPX\Captcha\Pool;

class ConnectionPoolConfig
{
    public $maxConnections;
    public $connectionTimeout;
    public $requestTimeout;

    public function __construct(
        int $maxConnections = 10,
        int $connectionTimeout = 30,
        int $requestTimeout = 30
    ) {
        $this->maxConnections = $maxConnections;
        $this->connectionTimeout = $connectionTimeout;
        $this->requestTimeout = $requestTimeout;
    }
}
