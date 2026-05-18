<?php

namespace HJTPX\Captcha\Model;

class ConnectCaptchaResponse
{
    public $sessionId;
    public $imageUrl;
    public $targets;

    public function __construct(array $data)
    {
        $this->sessionId = $data['session_id'] ?? null;
        $this->imageUrl = $data['image_url'] ?? null;
        $this->targets = $data['targets'] ?? [];
    }
}
