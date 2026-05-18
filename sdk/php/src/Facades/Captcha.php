<?php

declare(strict_types=1);

namespace Hjtpx\Captcha\Facades;

use Hjtpx\Captcha\Contracts\CaptchaClientInterface;
use Illuminate\Support\Facades\Facade;

class Captcha extends Facade
{
    protected static function getFacadeAccessor(): string
    {
        return CaptchaClientInterface::class;
    }

    public static function getSliderCaptcha(int $width = 320, int $height = 160, int $tolerance = 8): array
    {
        return static::getFacadeRoot()->getSliderCaptcha($width, $height, $tolerance);
    }

    public static function verifySliderCaptcha(string $sessionId, int $x, ?int $y = null, array $trajectory = []): array
    {
        return static::getFacadeRoot()->verifySliderCaptcha($sessionId, $x, $y, $trajectory);
    }

    public static function getClickCaptcha(string $mode = 'number', int $maxPoints = 3, bool $allowShuffle = true): array
    {
        return static::getFacadeRoot()->getClickCaptcha($mode, $maxPoints, $allowShuffle);
    }

    public static function verifyClickCaptcha(string $sessionId, array $points, ?array $clickSequence = null): array
    {
        return static::getFacadeRoot()->verifyClickCaptcha($sessionId, $points, $clickSequence);
    }

    public static function getImageCaptcha(string $type = 'mixed', int $count = 4): array
    {
        return static::getFacadeRoot()->getImageCaptcha($type, $count);
    }

    public static function verifyImageCaptcha(string $challengeId, string $answer): array
    {
        return static::getFacadeRoot()->verifyImageCaptcha($challengeId, $answer);
    }

    public static function batchVerify(array $requests): array
    {
        return static::getFacadeRoot()->batchVerify($requests);
    }

    public static function asyncVerify(array $request): array
    {
        return static::getFacadeRoot()->asyncVerify($request);
    }

    public static function getAsyncResult(string $taskId): array
    {
        return static::getFacadeRoot()->getAsyncResult($taskId);
    }

    public static function login(string $username, string $password, ?string $captchaToken = null): array
    {
        return static::getFacadeRoot()->login($username, $password, $captchaToken);
    }

    public static function logout(): bool
    {
        return static::getFacadeRoot()->logout();
    }

    public static function getDetectionScript(?string $callback = null): string
    {
        return static::getFacadeRoot()->getDetectionScript($callback);
    }
}
