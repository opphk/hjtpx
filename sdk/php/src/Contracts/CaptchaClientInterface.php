<?php

declare(strict_types=1);

namespace Hjtpx\Captcha\Contracts;

interface CaptchaClientInterface
{
    public function getSliderCaptcha(int $width = 320, int $height = 160, int $tolerance = 8): array;

    public function verifySliderCaptcha(string $sessionId, int $x, ?int $y = null, array $trajectory = []): array;

    public function getClickCaptcha(string $mode = 'number', int $maxPoints = 3, bool $allowShuffle = true): array;

    public function verifyClickCaptcha(string $sessionId, array $points, ?array $clickSequence = null): array;

    public function getImageCaptcha(string $type = 'mixed', int $count = 4): array;

    public function verifyImageCaptcha(string $challengeId, string $answer): array;

    public function getRotationCaptcha(): array;

    public function verifyRotationCaptcha(string $challengeId, int $angle): array;

    public function getGestureCaptcha(): array;

    public function verifyGestureCaptcha(string $sessionId, array $pattern): array;

    public function getJigsawCaptcha(int $width = 300, int $height = 300, int $gridSize = 3): array;

    public function verifyJigsawCaptcha(string $sessionId, array $pieces): array;

    public function batchVerify(array $requests): array;

    public function asyncVerify(array $request): array;

    public function getAsyncResult(string $taskId): array;

    public function login(string $username, string $password, ?string $captchaToken = null): array;

    public function logout(): bool;

    public function getDetectionScript(?string $callback = null): string;

    public function submitDetection(array $data): array;

    public function checkEnvironment(array $data): array;
}
