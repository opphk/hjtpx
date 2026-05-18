<?php

namespace HJTPX\Captcha\Exception;

class AuthenticationException extends CaptchaException
{
    protected $username;
    protected $tokenType;

    public function __construct(
        string $message = '',
        int $code = 0,
        ?\Throwable $previous = null,
        string $errorCode = null,
        array $context = [],
        string $username = null,
        string $tokenType = null
    ) {
        parent::__construct($message, $code, $previous, $errorCode ?? 'AUTH_ERROR', $context);
        $this->username = $username;
        $this->tokenType = $tokenType;
    }

    public function getUsername(): ?string
    {
        return $this->username;
    }

    public function getTokenType(): ?string
    {
        return $this->tokenType;
    }

    public function isExpiredToken(): bool
    {
        return $this->errorCode === 'TOKEN_EXPIRED';
    }

    public function isInvalidToken(): bool
    {
        return $this->errorCode === 'INVALID_TOKEN';
    }

    public static function invalidCredentials(string $username = null): self
    {
        return new self(
            'Invalid credentials provided',
            401,
            null,
            'INVALID_CREDENTIALS',
            ['username' => $username],
            $username
        );
    }

    public static function expiredToken(string $tokenType = 'access'): self
    {
        return new self(
            'Token has expired',
            401,
            null,
            'TOKEN_EXPIRED',
            ['token_type' => $tokenType],
            null,
            $tokenType
        );
    }

    public static function invalidToken(string $reason = null): self
    {
        return new self(
            'Token is invalid' . ($reason ? ': ' . $reason : ''),
            401,
            null,
            'INVALID_TOKEN',
            ['reason' => $reason]
        );
    }

    public static function unauthorized(string $reason = null): self
    {
        return new self(
            'Unauthorized access' . ($reason ? ': ' . $reason : ''),
            401,
            null,
            'UNAUTHORIZED',
            ['reason' => $reason]
        );
    }

    public static function forbidden(string $resource = null): self
    {
        return new self(
            'Access forbidden' . ($resource ? ' to ' . $resource : ''),
            403,
            null,
            'FORBIDDEN',
            ['resource' => $resource]
        );
    }
}
