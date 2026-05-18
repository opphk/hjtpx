<?php

namespace HJTPX\Captcha\Model;

class SliderCaptchaResponse
{
    public $sessionId;
    public $imageUrl;
    public $puzzleUrl;
    public $hintUrl;
    public $shape;
    public $secretY;
    public $imageWidth;
    public $imageHeight;
    public $tolerance;

    public function __construct(array $data)
    {
        $this->sessionId = $data['session_id'] ?? null;
        $this->imageUrl = $data['image_url'] ?? null;
        $this->puzzleUrl = $data['puzzle_url'] ?? null;
        $this->hintUrl = $data['hint_url'] ?? null;
        $this->shape = $data['shape'] ?? null;
        $this->secretY = $data['secret_y'] ?? null;
        $this->imageWidth = $data['image_width'] ?? null;
        $this->imageHeight = $data['image_height'] ?? null;
        $this->tolerance = $data['tolerance'] ?? null;
    }
}
