<?php

namespace HJTPX\Captcha\Exception;

class ValidationException extends CaptchaException
{
    protected $field;
    protected $errors;

    public function __construct(
        string $message = '',
        int $code = 0,
        ?\Throwable $previous = null,
        string $errorCode = null,
        array $context = [],
        string $field = null,
        array $errors = []
    ) {
        parent::__construct($message, $code, $previous, $errorCode ?? 'VALIDATION_ERROR', $context);
        $this->field = $field;
        $this->errors = $errors;
    }

    public function getField(): ?string
    {
        return $this->field;
    }

    public function getErrors(): array
    {
        return $this->errors;
    }

    public static function invalidField(string $field, string $reason, $value = null): self
    {
        return new self(
            "Invalid value for field '{$field}': {$reason}",
            0,
            null,
            'INVALID_FIELD',
            ['field' => $field, 'reason' => $reason, 'value' => $value],
            $field,
            [$field => $reason]
        );
    }

    public static function missingRequiredField(string $field): self
    {
        return new self(
            "Required field '{$field}' is missing",
            0,
            null,
            'MISSING_FIELD',
            ['field' => $field],
            $field,
            [$field => 'Required field is missing']
        );
    }
}
