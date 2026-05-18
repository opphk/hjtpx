<?php

namespace HJTPX\Captcha\Exception;

class NetworkException extends CaptchaException
{
    protected $host;
    protected $port;
    protected $timeout;

    public function __construct(
        string $message = '',
        int $code = 0,
        ?\Throwable $previous = null,
        string $errorCode = null,
        array $context = [],
        string $host = null,
        int $port = null,
        int $timeout = null
    ) {
        parent::__construct($message, $code, $previous, $errorCode, $context);
        $this->host = $host;
        $this->port = $port;
        $this->timeout = $timeout;
    }

    public function getHost(): ?string
    {
        return $this->host;
    }

    public function getPort(): ?int
    {
        return $this->port;
    }

    public function getTimeout(): ?int
    {
        return $this->timeout;
    }

    public static function connectionTimeout(string $host, int $port, int $timeout): self
    {
        return new self(
            "Connection timeout connecting to {$host}:{$port} (timeout: {$timeout}s)",
            0,
            null,
            'CONNECTION_TIMEOUT',
            ['host' => $host, 'port' => $port, 'timeout' => $timeout],
            $host,
            $port,
            $timeout
        );
    }

    public static function connectionRefused(string $host, int $port): self
    {
        return new self(
            "Connection refused to {$host}:{$port}",
            0,
            null,
            'CONNECTION_REFUSED',
            ['host' => $host, 'port' => $port],
            $host,
            $port
        );
    }

    public static function dnsResolutionFailed(string $host): self
    {
        return new self(
            "DNS resolution failed for host: {$host}",
            0,
            null,
            'DNS_RESOLUTION_FAILED',
            ['host' => $host],
            $host
        );
    }
}
