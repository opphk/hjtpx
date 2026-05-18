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

    public function __construct(array $data)
    {
        $this->success = $data['success'] ?? false;
        $this->message = $data['message'] ?? '';
        $this->remainingAttempts = $data['remaining_attempts'] ?? null;
        $this->trajectoryResult = $data['trajectory_result'] ?? null;
        $this->riskScore = $data['risk_score'] ?? null;
        $this->captchaPass = $data['captcha_pass'] ?? null;
        $this->failReason = $data['fail_reason'] ?? null;
    }
}
