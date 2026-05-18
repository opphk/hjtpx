<?php

namespace HJTPX\Captcha\Model;

class JigsawCaptchaResponse
{
    public $sessionId;
    public $imageUrl;
    public $pieces;
    public $pieceImages;
    public $gridSize;
    public $pieceWidth;
    public $pieceHeight;
    public $imageWidth;
    public $imageHeight;

    public function __construct(array $data)
    {
        $this->sessionId = $data['session_id'] ?? null;
        $this->imageUrl = $data['image_url'] ?? null;
        $this->pieces = array_map(function ($piece) {
            return new JigsawPiece($piece);
        }, $data['pieces'] ?? []);
        $this->pieceImages = $data['piece_images'] ?? [];
        $this->gridSize = $data['grid_size'] ?? 0;
        $this->pieceWidth = $data['piece_width'] ?? 0;
        $this->pieceHeight = $data['piece_height'] ?? 0;
        $this->imageWidth = $data['image_width'] ?? 0;
        $this->imageHeight = $data['image_height'] ?? 0;
    }
}
