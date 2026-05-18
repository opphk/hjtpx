<?php

namespace HJTPX\Captcha\Model;

class LoginResponse
{
    public $accessToken;
    public $refreshToken;
    public $expiresIn;
    public $user;

    public function __construct(array $data)
    {
        $this->accessToken = $data['access_token'] ?? null;
        $this->refreshToken = $data['refresh_token'] ?? null;
        $this->expiresIn = $data['expires_in'] ?? null;
        $this->user = $data['user'] ?? null;
    }
}
