<?php

namespace HJTPX\Captcha\Model;

class LoginRequest
{
    public $username;
    public $password;
    public $captchaToken;

    public function toArray(): array
    {
        $data = [
            'username' => $this->username,
            'password' => $this->password,
        ];

        if ($this->captchaToken !== null) {
            $data['captcha_token'] = $this->captchaToken;
        }

        return $data;
    }
}
