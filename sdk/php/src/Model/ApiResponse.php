<?php

namespace HJTPX\Captcha\Model;

class ApiResponse
{
    public $code;
    public $message;
    public $data;

    public function __construct(int $code, string $message, $data = null)
    {
        $this->code = $code;
        $this->message = $message;
        $this->data = $data;
    }

    public static function fromArray(array $data): self
    {
        return new self(
            $data['code'] ?? 0,
            $data['message'] ?? '',
            $data['data'] ?? null
        );
    }
}
