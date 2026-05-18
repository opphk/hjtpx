<?php

namespace HJTPX\Captcha\Exception;

class ApiException extends CaptchaException
{
    protected $httpStatusCode;
    protected $responseData;

    public function __construct(
        string $message = '',
        int $code = 0,
        ?\Throwable $previous = null,
        string $errorCode = null,
        array $context = [],
        int $httpStatusCode = null,
        array $responseData = []
    ) {
        parent::__construct($message, $code, $previous, $errorCode, $context);
        $this->httpStatusCode = $httpStatusCode;
        $this->responseData = $responseData;
    }

    public function getHttpStatusCode(): ?int
    {
        return $this->httpStatusCode;
    }

    public function getResponseData(): array
    {
        return $this->responseData;
    }

    public function isRateLimitError(): bool
    {
        return $this->httpStatusCode === 429;
    }

    public function isServerError(): bool
    {
        return $this->httpStatusCode !== null && $this->httpStatusCode >= 500;
    }

    public function isClientError(): bool
    {
        return $this->httpStatusCode !== null && $this->httpStatusCode >= 400 && $this->httpStatusCode < 500;
    }

    public static function fromResponse(
        string $message,
        int $httpStatusCode,
        array $responseData = []
    ): self {
        $errorCode = 'API_ERROR';
        $context = [];

        if (isset($responseData['code'])) {
            $errorCode = 'API_CODE_' . $responseData['code'];
        }

        if (isset($responseData['errors'])) {
            $context['errors'] = $responseData['errors'];
        }

        return new self($message, $httpStatusCode, null, $errorCode, $context, $httpStatusCode, $responseData);
    }
}
