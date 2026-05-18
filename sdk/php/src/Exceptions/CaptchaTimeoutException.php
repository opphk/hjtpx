<?php

declare(strict_types=1);

namespace Hjtpx\Captcha\Exceptions;

class CaptchaTimeoutException extends CaptchaNetworkException
{
    public function __construct(string $message = 'Request timed out', int $code = 0, ?\Throwable $previous = null)
    {
        parent::__construct($message, $code, $previous);
    }
}
