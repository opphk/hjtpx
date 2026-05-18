<?php

declare(strict_types=1);

namespace Hjtpx\Captcha\Exceptions;

class CaptchaApiException extends CaptchaException
{
    private ?int $errorCode;

    public function __construct(string $message = '', ?int $errorCode = null, ?\Throwable $previous = null)
    {
        parent::__construct($message, $errorCode ?? 0, $previous);
        $this->errorCode = $errorCode;
    }

    public function getErrorCode(): ?int
    {
        return $this->errorCode;
    }
}
