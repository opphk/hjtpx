<?php

namespace HJTPX\Captcha\Model;

class RotationCaptchaResponse
{
    public $challengeId;
    public $image;

    public function __construct(array $data)
    {
        $this->challengeId = $data['challenge_id'] ?? null;
        $this->image = $data['image'] ?? null;
    }
}
