<?php

namespace HJTPX\Captcha\Model;

class JigsawPiece
{
    public $index;
    public $originalX;
    public $originalY;
    public $currentX;
    public $currentY;
    public $width;
    public $height;
    public $rotation;

    public function __construct(array $data)
    {
        $this->index = $data['index'] ?? 0;
        $this->originalX = $data['original_x'] ?? 0;
        $this->originalY = $data['original_y'] ?? 0;
        $this->currentX = $data['current_x'] ?? 0;
        $this->currentY = $data['current_y'] ?? 0;
        $this->width = $data['width'] ?? 0;
        $this->height = $data['height'] ?? 0;
        $this->rotation = $data['rotation'] ?? 0;
    }

    public function toArray(): array
    {
        return [
            'index' => $this->index,
            'original_x' => $this->originalX,
            'original_y' => $this->originalY,
            'current_x' => $this->currentX,
            'current_y' => $this->currentY,
            'width' => $this->width,
            'height' => $this->height,
            'rotation' => $this->rotation,
        ];
    }
}
