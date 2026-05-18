<?php

namespace HJTPX\Captcha\Model;

class VoiceCaptchaResponse
{
    public $sessionId;
    public $audioUrl;
    public $duration;

    public function __construct(array $data)
    {
        $this->sessionId = $data['session_id'] ?? null;
        $this->audioUrl = $data['audio_url'] ?? null;
        $this->duration = $data['duration'] ?? null;
    }
}
