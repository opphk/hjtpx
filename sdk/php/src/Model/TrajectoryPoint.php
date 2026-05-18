<?php

namespace HJTPX\Captcha\Model;

class TrajectoryPoint
{
    public $x;
    public $y;
    public $t;

    public function __construct(int $x, int $y, int $t)
    {
        $this->x = $x;
        $this->y = $y;
        $this->t = $t;
    }

    public static function fromArray(array $data): self
    {
        return new self($data['x'], $data['y'], $data['t']);
    }

    public function toArray(): array
    {
        return [
            'x' => $this->x,
            'y' => $this->y,
            't' => $this->t,
        ];
    }
}
