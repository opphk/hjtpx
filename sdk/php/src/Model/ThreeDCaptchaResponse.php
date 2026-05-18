<?php

namespace HJTPX\Captcha\Model;

class ThreeDCaptchaResponse
{
    public $sessionId;
    public $sceneUrl;
    public $objectData;

    public function __construct(array $data)
    {
        $this->sessionId = $data['session_id'] ?? null;
        $this->sceneUrl = $data['scene_url'] ?? null;
        $this->objectData = $data['object_data'] ?? null;
    }
}
