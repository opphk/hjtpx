<?php

declare(strict_types=1);

namespace Hjtpx\Captcha\Exceptions;

class CaptchaNetworkException extends CaptchaException
{
    public function __construct(string $message = '', int $code = 0, ?\Throwable $previous = null)
    {
        parent::__construct($message, $code, $previous);
    }
}
