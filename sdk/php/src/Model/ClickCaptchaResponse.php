<?php

namespace HJTPX\Captcha\Model;

class ClickCaptchaResponse
{
    public $sessionId;
    public $imageUrl;
    public $hint;
    public $hintOrder;
    public $maxPoints;
    public $mode;
    public $allowShuffle;
    public $points;

    public function __construct(array $data)
    {
        $this->sessionId = $data['session_id'] ?? null;
        $this->imageUrl = $data['image_url'] ?? null;
        $this->hint = $data['hint'] ?? null;
        $this->hintOrder = $data['hint_order'] ?? [];
        $this->maxPoints = $data['max_points'] ?? 0;
        $this->mode = $data['mode'] ?? null;
        $this->allowShuffle = $data['allow_shuffle'] ?? false;
        $this->points = $data['points'] ?? null;
    }
}
