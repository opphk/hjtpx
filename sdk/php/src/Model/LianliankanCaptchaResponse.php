<?php

namespace HJTPX\Captcha\Model;

class LianliankanCaptchaResponse
{
    public $sessionId;
    public $gridWidth;
    public $gridHeight;
    public $gridRows;
    public $gridCols;
    public $imageUrl;
    public $pairs;
    public $timeLimit;
    public $maxAttempts;

    public function __construct(array $data)
    {
        $this->sessionId = $data['session_id'] ?? null;
        $this->gridWidth = $data['grid_width'] ?? null;
        $this->gridHeight = $data['grid_height'] ?? null;
        $this->gridRows = $data['grid_rows'] ?? 0;
        $this->gridCols = $data['grid_cols'] ?? 0;
        $this->imageUrl = $data['image_url'] ?? null;
        $this->pairs = $data['pairs'] ?? [];
        $this->timeLimit = $data['time_limit'] ?? 60;
        $this->maxAttempts = $data['max_attempts'] ?? 3;
    }

    public function getTotalCells(): int
    {
        return $this->gridRows * $this->gridCols;
    }

    public function getTotalPairs(): int
    {
        return count($this->pairs);
    }
}
