<?php

namespace HJTPX\Captcha\Model;

class VerifyCaptchaResponse
{
    public $success;
    public $message;
    public $remainingAttempts;
    public $trajectoryResult;
    public $riskScore;
    public $captchaPass;
    public $failReason;
    public $score;
    public $token;
    public $expiresAt;

    public function __construct(array $data)
    {
        $this->success = $data['success'] ?? false;
        $this->message = $data['message'] ?? '';
        $this->remainingAttempts = $data['remaining_attempts'] ?? null;
        $this->trajectoryResult = $data['trajectory_result'] ?? null;
        $this->riskScore = $data['risk_score'] ?? null;
        $this->captchaPass = $data['captcha_pass'] ?? null;
        $this->failReason = $data['fail_reason'] ?? null;
        $this->score = $data['score'] ?? null;
        $this->token = $data['token'] ?? null;
        $this->expiresAt = $data['expires_at'] ?? null;
    }

    public function isValid(): bool
    {
        return $this->success === true;
    }

    public function hasRemainingAttempts(): bool
    {
        return $this->remainingAttempts !== null && $this->remainingAttempts > 0;
    }

    public function getRiskLevel(): string
    {
        if ($this->riskScore === null) {
            return 'unknown';
        }

        if ($this->riskScore >= 80) {
            return 'low';
        } elseif ($this->riskScore >= 50) {
            return 'medium';
        } else {
            return 'high';
        }
    }

    public function toArray(): array
    {
        return [
            'success' => $this->success,
            'message' => $this->message,
            'remaining_attempts' => $this->remainingAttempts,
            'trajectory_result' => $this->trajectoryResult,
            'risk_score' => $this->riskScore,
            'captcha_pass' => $this->captchaPass,
            'fail_reason' => $this->failReason,
            'score' => $this->score,
            'token' => $this->token,
            'expires_at' => $this->expiresAt,
        ];
    }
}
