<?php

declare(strict_types=1);

namespace CaptchaX\SDK;

class CaptchaXException extends \Exception
{
    private int $code;
    private int $statusCode;
    private $details;

    public function __construct(
        string $message,
        int $code = 500,
        int $statusCode = 500,
        $details = null,
        ?\Throwable $previous = null
    ) {
        parent::__construct($message, $code, $previous);
        $this->code = $code;
        $this->statusCode = $statusCode;
        $this->details = $details;
    }

    public function getErrorCode(): int
    {
        return $this->code;
    }

    public function getStatusCode(): int
    {
        return $this->statusCode;
    }

    public function getDetails()
    {
        return $this->details;
    }
}
