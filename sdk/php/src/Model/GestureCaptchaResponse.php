<?php

namespace HJTPX\Captcha\Model;

class GestureCaptchaResponse
{
    public $sessionId;
    public $pattern;
    public $gridSize;
    public $hint;

    public function __construct(array $data)
    {
        $this->sessionId = $data['session_id'] ?? null;
        $this->pattern = $data['pattern'] ?? null;
        $this->gridSize = $data['grid_size'] ?? null;
        $this->hint = $data['hint'] ?? null;
    }
}
