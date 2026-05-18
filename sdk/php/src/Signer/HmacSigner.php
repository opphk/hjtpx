<?php

namespace HJTPX\Captcha\Signer;

class HmacSigner
{
    protected $secretKey;

    public function __construct(string $secretKey)
    {
        $this->secretKey = $secretKey;
    }

    public function sign(string $data): string
    {
        return hash_hmac('sha256', $data, $this->secretKey);
    }

    public function verify(string $data, string $signature): bool
    {
        $expectedSignature = $this->sign($data);
        return hash_equals($expectedSignature, $signature);
    }
}
