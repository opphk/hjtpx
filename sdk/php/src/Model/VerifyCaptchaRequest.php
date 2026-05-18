<?php

namespace HJTPX\Captcha\Model;

class VerifyCaptchaRequest
{
    public $sessionId;
    public $type;
    public $x;
    public $y;
    public $trajectory;
    public $points;
    public $clickSequence;
    public $angle;
    public $pattern;
    public $pieces;
    public $answer;
    public $connections;
    public $targetPosition;

    public function toArray(): array
    {
        $data = [
            'session_id' => $this->sessionId,
            'type' => $this->type,
        ];

        if ($this->x !== null) {
            $data['x'] = $this->x;
        }
        if ($this->y !== null) {
            $data['y'] = $this->y;
        }
        if ($this->trajectory !== null) {
            $data['trajectory'] = array_map(function ($point) {
                return $point instanceof TrajectoryPoint ? $point->toArray() : $point;
            }, $this->trajectory);
        }
        if ($this->points !== null) {
            $data['points'] = $this->points;
        }
        if ($this->clickSequence !== null) {
            $data['click_sequence'] = $this->clickSequence;
        }
        if ($this->angle !== null) {
            $data['angle'] = $this->angle;
        }
        if ($this->pattern !== null) {
            $data['pattern'] = $this->pattern;
        }
        if ($this->pieces !== null) {
            $data['pieces'] = array_map(function ($piece) {
                return $piece instanceof JigsawPiece ? $piece->toArray() : $piece;
            }, $this->pieces);
        }
        if ($this->answer !== null) {
            $data['answer'] = $this->answer;
        }
        if ($this->connections !== null) {
            $data['connections'] = $this->connections;
        }
        if ($this->targetPosition !== null) {
            $data['target_position'] = $this->targetPosition;
        }

        return $data;
    }
}
