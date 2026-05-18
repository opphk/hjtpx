<?php

namespace HJTPX\Captcha\Exception;

class ApiException extends CaptchaException
{
    protected $code;

    public function __construct(string $message, int $code = 0, ?\Throwable $previous = null)
    {
        $this->code = $code;
        parent::__construct($message, $code, $previous);
    }
}
