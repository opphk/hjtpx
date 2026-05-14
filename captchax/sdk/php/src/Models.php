<?php

declare(strict_types=1);

namespace CaptchaX\SDK;

class ApiVersion
{
    public const V1 = 'v1';
    public const V2 = 'v2';
}

class CaptchaConfig
{
    private string $baseUrl;
    private ?string $appId;
    private int $timeout;
    private int $retryTimes;
    private string $apiVersion;

    public function __construct(
        string $baseUrl,
        ?string $appId = null,
        int $timeout = 10000,
        int $retryTimes = 3,
        string $apiVersion = ApiVersion::V1
    ) {
        $this->baseUrl = $baseUrl;
        $this->appId = $appId;
        $this->timeout = $timeout;
        $this->retryTimes = $retryTimes;
        $this->apiVersion = $apiVersion;
    }

    public static function create(string $baseUrl): self
    {
        return new self($baseUrl);
    }

    public function withAppId(string $appId): self
    {
        $clone = clone $this;
        $clone->appId = $appId;
        return $clone;
    }

    public function withTimeout(int $timeout): self
    {
        $clone = clone $this;
        $clone->timeout = $timeout;
        return $clone;
    }

    public function withRetryTimes(int $retryTimes): self
    {
        $clone = clone $this;
        $clone->retryTimes = $retryTimes;
        return $clone;
    }

    public function withApiVersion(string $apiVersion): self
    {
        $clone = clone $this;
        $clone->apiVersion = $apiVersion;
        return $clone;
    }

    public function getBaseUrl(): string
    {
        return $this->baseUrl;
    }

    public function getAppId(): ?string
    {
        return $this->appId;
    }

    public function getTimeout(): int
    {
        return $this->timeout;
    }

    public function getRetryTimes(): int
    {
        return $this->retryTimes;
    }

    public function getApiVersion(): string
    {
        return $this->apiVersion;
    }
}

class CharPosition
{
    private string $char;
    private int $x;
    private int $y;

    public function __construct(string $char, int $x, int $y)
    {
        $this->char = $char;
        $this->x = $x;
        $this->y = $y;
    }

    public function getChar(): string
    {
        return $this->char;
    }

    public function getX(): int
    {
        return $this->x;
    }

    public function getY(): int
    {
        return $this->y;
    }

    public function toArray(): array
    {
        return [
            'char' => $this->char,
            'x' => $this->x,
            'y' => $this->y,
        ];
    }
}

class SliderCaptchaResult
{
    private string $id;
    private string $backgroundB64;
    private string $sliderB64;
    private int $targetX;
    private int $targetY;

    public function __construct(array $data)
    {
        $this->id = $data['id'] ?? '';
        $this->backgroundB64 = $data['background_b64'] ?? '';
        $this->sliderB64 = $data['slider_b64'] ?? '';
        $this->targetX = $data['target_x'] ?? 0;
        $this->targetY = $data['target_y'] ?? 0;
    }

    public function getId(): string
    {
        return $this->id;
    }

    public function getBackgroundB64(): string
    {
        return $this->backgroundB64;
    }

    public function getSliderB64(): string
    {
        return $this->sliderB64;
    }

    public function getTargetX(): int
    {
        return $this->targetX;
    }

    public function getTargetY(): int
    {
        return $this->targetY;
    }
}

class SliderVerifyResult
{
    private bool $success;
    private string $message;

    public function __construct(array $data)
    {
        $this->success = $data['success'] ?? false;
        $this->message = $data['message'] ?? '';
    }

    public function isSuccess(): bool
    {
        return $this->success;
    }

    public function getMessage(): string
    {
        return $this->message;
    }
}

class ClickCaptchaResult
{
    private string $id;
    private string $image;
    private array $targetChars;
    private array $charPositions;

    public function __construct(array $data)
    {
        $this->id = $data['id'] ?? '';
        $this->image = $data['image'] ?? '';
        $this->targetChars = $data['target_chars'] ?? [];
        $this->charPositions = array_map(
            fn($pos) => new CharPosition($pos['char'] ?? '', $pos['x'] ?? 0, $pos['y'] ?? 0),
            $data['char_positions'] ?? []
        );
    }

    public function getId(): string
    {
        return $this->id;
    }

    public function getImage(): string
    {
        return $this->image;
    }

    public function getTargetChars(): array
    {
        return $this->targetChars;
    }

    public function getCharPositions(): array
    {
        return $this->charPositions;
    }
}

class ClickVerifyResult
{
    private bool $success;
    private float $score;
    private string $message;

    public function __construct(array $data)
    {
        $this->success = $data['success'] ?? false;
        $this->score = (float)($data['score'] ?? 0.0);
        $this->message = $data['message'] ?? '';
    }

    public function isSuccess(): bool
    {
        return $this->success;
    }

    public function getScore(): float
    {
        return $this->score;
    }

    public function getMessage(): string
    {
        return $this->message;
    }
}

class PuzzleCaptchaResult
{
    private string $id;
    private string $backgroundB64;
    private string $puzzleB64;
    private int $targetX;
    private int $targetY;

    public function __construct(array $data)
    {
        $this->id = $data['id'] ?? '';
        $this->backgroundB64 = $data['background_b64'] ?? '';
        $this->puzzleB64 = $data['puzzle_b64'] ?? '';
        $this->targetX = $data['target_x'] ?? 0;
        $this->targetY = $data['target_y'] ?? 0;
    }

    public function getId(): string
    {
        return $this->id;
    }

    public function getBackgroundB64(): string
    {
        return $this->backgroundB64;
    }

    public function getPuzzleB64(): string
    {
        return $this->puzzleB64;
    }

    public function getTargetX(): int
    {
        return $this->targetX;
    }

    public function getTargetY(): int
    {
        return $this->targetY;
    }
}

class PuzzleVerifyResult
{
    private bool $success;
    private string $message;

    public function __construct(array $data)
    {
        $this->success = $data['success'] ?? false;
        $this->message = $data['message'] ?? '';
    }

    public function isSuccess(): bool
    {
        return $this->success;
    }

    public function getMessage(): string
    {
        return $this->message;
    }
}

class Scenario
{
    private ?string $id;
    private ?string $name;
    private ?string $description;
    private ?string $difficulty;
    private ?array $config;
    private ?string $createdAt;
    private ?string $updatedAt;

    public function __construct(array $data)
    {
        $this->id = $data['id'] ?? null;
        $this->name = $data['name'] ?? null;
        $this->description = $data['description'] ?? null;
        $this->difficulty = $data['difficulty'] ?? null;
        $this->config = $data['config'] ?? null;
        $this->createdAt = $data['created_at'] ?? null;
        $this->updatedAt = $data['updated_at'] ?? null;
    }

    public function getId(): ?string
    {
        return $this->id;
    }

    public function getName(): ?string
    {
        return $this->name;
    }

    public function getDescription(): ?string
    {
        return $this->description;
    }

    public function getDifficulty(): ?string
    {
        return $this->difficulty;
    }

    public function getConfig(): ?array
    {
        return $this->config;
    }

    public function getCreatedAt(): ?string
    {
        return $this->createdAt;
    }

    public function getUpdatedAt(): ?string
    {
        return $this->updatedAt;
    }
}

class Webhook
{
    private ?string $id;
    private ?string $appId;
    private string $url;
    private ?string $secret;
    private array $events;
    private ?array $headers;
    private bool $enabled;
    private ?string $createdAt;
    private ?string $updatedAt;

    public function __construct(array $data)
    {
        $this->id = $data['id'] ?? null;
        $this->appId = $data['app_id'] ?? null;
        $this->url = $data['url'] ?? '';
        $this->secret = $data['secret'] ?? null;
        $this->events = $data['events'] ?? [];
        $this->headers = $data['headers'] ?? null;
        $this->enabled = $data['enabled'] ?? true;
        $this->createdAt = $data['created_at'] ?? null;
        $this->updatedAt = $data['updated_at'] ?? null;
    }

    public function getId(): ?string
    {
        return $this->id;
    }

    public function getAppId(): ?string
    {
        return $this->appId;
    }

    public function getUrl(): string
    {
        return $this->url;
    }

    public function getSecret(): ?string
    {
        return $this->secret;
    }

    public function getEvents(): array
    {
        return $this->events;
    }

    public function getHeaders(): ?array
    {
        return $this->headers;
    }

    public function isEnabled(): bool
    {
        return $this->enabled;
    }

    public function getCreatedAt(): ?string
    {
        return $this->createdAt;
    }

    public function getUpdatedAt(): ?string
    {
        return $this->updatedAt;
    }
}

class BatchVerifyItem
{
    private string $captchaId;
    private string $type;
    private int $targetX;
    private ?int $targetY;
    private ?array $clicks;

    public function __construct(string $captchaId, string $type, int $targetX)
    {
        $this->captchaId = $captchaId;
        $this->type = $type;
        $this->targetX = $targetX;
        $this->targetY = null;
        $this->clicks = null;
    }

    public function withTargetY(int $targetY): self
    {
        $clone = clone $this;
        $clone->targetY = $targetY;
        return $clone;
    }

    public function withClicks(array $clicks): self
    {
        $clone = clone $this;
        $clone->clicks = array_map(fn($c) => $c instanceof CharPosition ? $c->toArray() : $c, $clicks);
        return $clone;
    }

    public function toArray(): array
    {
        $data = [
            'captcha_id' => $this->captchaId,
            'type' => $this->type,
            'target_x' => $this->targetX,
        ];
        if ($this->targetY !== null) {
            $data['target_y'] = $this->targetY;
        }
        if ($this->clicks !== null) {
            $data['clicks'] = $this->clicks;
        }
        return $data;
    }
}

class BatchVerifyResult
{
    private string $captchaId;
    private bool $success;
    private string $message;
    private ?float $score;

    public function __construct(array $data)
    {
        $this->captchaId = $data['captcha_id'] ?? '';
        $this->success = $data['success'] ?? false;
        $this->message = $data['message'] ?? '';
        $this->score = isset($data['score']) ? (float)$data['score'] : null;
    }

    public function getCaptchaId(): string
    {
        return $this->captchaId;
    }

    public function isSuccess(): bool
    {
        return $this->success;
    }

    public function getMessage(): string
    {
        return $this->message;
    }

    public function getScore(): ?float
    {
        return $this->score;
    }
}

class BatchVerifySummary
{
    private int $total;
    private int $successCount;
    private int $failed;
    private int $skipped;

    public function __construct(array $data)
    {
        $this->total = $data['total'] ?? 0;
        $this->successCount = $data['success'] ?? 0;
        $this->failed = $data['failed'] ?? 0;
        $this->skipped = $data['skipped'] ?? 0;
    }

    public function getTotal(): int
    {
        return $this->total;
    }

    public function getSuccessCount(): int
    {
        return $this->successCount;
    }

    public function getFailed(): int
    {
        return $this->failed;
    }

    public function getSkipped(): int
    {
        return $this->skipped;
    }
}

class BatchVerifyResponse
{
    private array $results;
    private BatchVerifySummary $summary;

    public function __construct(array $data)
    {
        $this->results = array_map(fn($r) => new BatchVerifyResult($r), $data['results'] ?? []);
        $this->summary = new BatchVerifySummary($data['summary'] ?? []);
    }

    public function getResults(): array
    {
        return $this->results;
    }

    public function getSummary(): BatchVerifySummary
    {
        return $this->summary;
    }
}

class HealthStatus
{
    private string $status;
    private string $service;
    private string $timestamp;
    private string $version;

    public function __construct(array $data)
    {
        $this->status = $data['status'] ?? '';
        $this->service = $data['service'] ?? '';
        $this->timestamp = $data['timestamp'] ?? '';
        $this->version = $data['version'] ?? '';
    }

    public function getStatus(): string
    {
        return $this->status;
    }

    public function getService(): string
    {
        return $this->service;
    }

    public function getTimestamp(): string
    {
        return $this->timestamp;
    }

    public function getVersion(): string
    {
        return $this->version;
    }
}
