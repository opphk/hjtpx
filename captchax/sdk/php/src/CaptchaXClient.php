<?php

declare(strict_types=1);

namespace CaptchaX\SDK;

class CaptchaXClient
{
    private CaptchaConfig $config;
    private array $headers;
    private int $retryTimes;
    private float $timeout;

    public function __construct(CaptchaConfig $config)
    {
        $this->config = $config;
        $this->headers = [
            'Content-Type' => 'application/json',
            'Accept' => 'application/json',
        ];
        $this->retryTimes = $config->getRetryTimes();
        $this->timeout = (float)$config->getTimeout() / 1000;

        if ($config->getAppId() !== null) {
            $this->headers['X-App-ID'] = $config->getAppId();
        }
    }

    public static function create(string $baseUrl): self
    {
        return new self(new CaptchaConfig($baseUrl));
    }

    public function setAppId(string $appId): void
    {
        $this->headers['X-App-ID'] = $appId;
    }

    public function setApiVersion(string $version): void
    {
        $this->config = $this->config->withApiVersion($version);
    }

    public function getApiVersion(): string
    {
        return $this->config->getApiVersion();
    }

    private function getApiPrefix(): string
    {
        return '/api/' . $this->config->getApiVersion();
    }

    private function getBaseUrl(): string
    {
        return rtrim($this->config->getBaseUrl(), '/');
    }

    private function requireAppId(): void
    {
        if (!isset($this->headers['X-App-ID'])) {
            throw new CaptchaXException('appId is required for captcha generation', 400, 400);
        }
    }

    private function request(string $method, string $endpoint, ?array $body = null, ?string $deduplicationId = null): array
    {
        $url = $this->getBaseUrl() . $endpoint;
        $lastException = null;

        for ($attempt = 0; $attempt <= $this->retryTimes; $attempt++) {
            $ch = curl_init();

            $headers = $this->headers;
            if ($deduplicationId !== null) {
                $headers['X-Deduplication-ID'] = $deduplicationId;
            }

            $headerList = [];
            foreach ($headers as $key => $value) {
                $headerList[] = "$key: $value";
            }

            curl_setopt_array($ch, [
                CURLOPT_URL => $url,
                CURLOPT_RETURNTRANSFER => true,
                CURLOPT_TIMEOUT => $this->timeout,
                CURLOPT_CONNECTTIMEOUT => $this->timeout,
                CURLOPT_HTTPHEADER => $headerList,
                CURLOPT_SSL_VERIFYPEER => true,
                CURLOPT_SSL_VERIFYHOST => 2,
            ]);

            if ($method === 'POST') {
                curl_setopt($ch, CURLOPT_POST, true);
                if ($body !== null) {
                    curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($body));
                }
            } elseif ($method === 'PUT') {
                curl_setopt($ch, CURLOPT_CUSTOMREQUEST, 'PUT');
                if ($body !== null) {
                    curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($body));
                }
            } elseif ($method === 'DELETE') {
                curl_setopt($ch, CURLOPT_CUSTOMREQUEST, 'DELETE');
            } else {
                curl_setopt($ch, CURLOPT_HTTPGET, true);
            }

            $response = curl_exec($ch);
            $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
            $error = curl_error($ch);
            curl_close($ch);

            if ($response === false) {
                $lastException = new CaptchaXException("cURL error: $error", 0, 0);
                if ($attempt < $this->retryTimes) {
                    usleep((int)(pow(2, $attempt) * 1000000));
                    continue;
                }
                throw $lastException;
            }

            $data = json_decode($response, true);

            if ($httpCode >= 500 && $attempt < $this->retryTimes) {
                $lastException = new CaptchaXException("Server error: $httpCode", $httpCode, $httpCode);
                usleep((int)(pow(2, $attempt) * 1000000));
                continue;
            }

            if ($httpCode >= 400) {
                $message = $data['message'] ?? "HTTP error: $httpCode";
                $code = $data['code'] ?? $httpCode;
                throw new CaptchaXException($message, (int)$code, $httpCode, $data);
            }

            if (isset($data['code']) && $data['code'] !== 200) {
                throw new CaptchaXException(
                    $data['message'] ?? 'Unknown error',
                    (int)$data['code'],
                    $httpCode,
                    $data
                );
            }

            return $data;
        }

        throw new CaptchaXException(
            "Request failed after " . ($this->retryTimes + 1) . " attempts",
            0,
            0,
            $lastException
        );
    }

    private function get(string $endpoint): array
    {
        return $this->request('GET', $endpoint);
    }

    private function post(string $endpoint, ?array $body = null, ?string $deduplicationId = null): array
    {
        return $this->request('POST', $endpoint, $body, $deduplicationId);
    }

    private function put(string $endpoint, ?array $body = null): array
    {
        return $this->request('PUT', $endpoint, $body);
    }

    private function delete(string $endpoint): array
    {
        return $this->request('DELETE', $endpoint);
    }

    public function healthCheck(): HealthStatus
    {
        $data = $this->get('/health');
        return new HealthStatus($data['data'] ?? []);
    }

    public function generateSliderCaptcha(
        ?int $width = null,
        ?int $height = null,
        ?string $clientInfo = null,
        ?string $scenarioId = null
    ): SliderCaptchaResult {
        $this->requireAppId();

        $body = [
            'app_id' => $this->headers['X-App-ID'],
        ];

        if ($width !== null) {
            $body['width'] = $width;
        }
        if ($height !== null) {
            $body['height'] = $height;
        }
        if ($clientInfo !== null) {
            $body['client_info'] = $clientInfo;
        }
        if ($scenarioId !== null) {
            $body['scenario_id'] = $scenarioId;
        }

        $data = $this->post($this->getApiPrefix() . '/captcha/slider', $body);
        return new SliderCaptchaResult($data['data'] ?? []);
    }

    public function verifySliderCaptcha(string $captchaId, int $targetX, ?int $targetY = null): SliderVerifyResult
    {
        $body = [
            'captcha_id' => $captchaId,
            'target_x' => $targetX,
        ];

        if ($targetY !== null) {
            $body['target_y'] = $targetY;
        }

        $data = $this->post($this->getApiPrefix() . '/captcha/slider/verify', $body);
        return new SliderVerifyResult($data['data'] ?? []);
    }

    public function generateClickCaptcha(
        ?int $charCount = null,
        ?string $clientInfo = null,
        ?string $scenarioId = null
    ): ClickCaptchaResult {
        $this->requireAppId();

        $body = [
            'app_id' => $this->headers['X-App-ID'],
        ];

        if ($charCount !== null) {
            $body['char_count'] = $charCount;
        }
        if ($clientInfo !== null) {
            $body['client_info'] = $clientInfo;
        }
        if ($scenarioId !== null) {
            $body['scenario_id'] = $scenarioId;
        }

        $data = $this->post($this->getApiPrefix() . '/captcha/click', $body);
        return new ClickCaptchaResult($data['data'] ?? []);
    }

    public function verifyClickCaptcha(string $captchaId, array $clicks): ClickVerifyResult
    {
        $body = [
            'captcha_id' => $captchaId,
            'clicks' => array_map(
                fn($click) => $click instanceof CharPosition ? $click->toArray() : $click,
                $clicks
            ),
        ];

        $data = $this->post($this->getApiPrefix() . '/captcha/click/verify', $body);
        return new ClickVerifyResult($data['data'] ?? []);
    }

    public function generatePuzzleCaptcha(
        ?int $width = null,
        ?int $height = null,
        ?string $clientInfo = null,
        ?string $scenarioId = null
    ): PuzzleCaptchaResult {
        $this->requireAppId();

        $body = [
            'app_id' => $this->headers['X-App-ID'],
        ];

        if ($width !== null) {
            $body['width'] = $width;
        }
        if ($height !== null) {
            $body['height'] = $height;
        }
        if ($clientInfo !== null) {
            $body['client_info'] = $clientInfo;
        }
        if ($scenarioId !== null) {
            $body['scenario_id'] = $scenarioId;
        }

        $data = $this->post($this->getApiPrefix() . '/captcha/puzzle', $body);
        return new PuzzleCaptchaResult($data['data'] ?? []);
    }

    public function verifyPuzzleCaptcha(string $captchaId, int $targetX, ?int $targetY = null): PuzzleVerifyResult
    {
        $body = [
            'captcha_id' => $captchaId,
            'target_x' => $targetX,
        ];

        if ($targetY !== null) {
            $body['target_y'] = $targetY;
        }

        $data = $this->post($this->getApiPrefix() . '/captcha/puzzle/verify', $body);
        return new PuzzleVerifyResult($data['data'] ?? []);
    }

    public function batchVerify(array $items, ?string $deduplicationId = null): BatchVerifyResponse
    {
        $body = [
            'items' => array_map(
                fn($item) => $item instanceof BatchVerifyItem ? $item->toArray() : $item,
                $items
            ),
        ];

        $data = $this->post($this->getApiPrefix() . '/captcha/batch/verify', $body, $deduplicationId);
        return new BatchVerifyResponse($data['data'] ?? []);
    }

    public function listScenarios(): array
    {
        $data = $this->get($this->getApiPrefix() . '/captcha/scenarios');
        return [
            'scenarios' => array_map(fn($s) => new Scenario($s), $data['data']['scenarios'] ?? []),
            'total' => $data['data']['total'] ?? 0,
        ];
    }

    public function createScenario(string $name, ?string $description = null, ?string $difficulty = null, ?array $config = null): Scenario
    {
        $body = ['name' => $name];

        if ($description !== null) {
            $body['description'] = $description;
        }
        if ($difficulty !== null) {
            $body['difficulty'] = $difficulty;
        }
        if ($config !== null) {
            $body['config'] = $config;
        }

        $data = $this->post($this->getApiPrefix() . '/captcha/scenarios', $body);
        return new Scenario($data['data'] ?? []);
    }

    public function getScenario(string $scenarioId): Scenario
    {
        $data = $this->get($this->getApiPrefix() . '/captcha/scenarios/' . $scenarioId);
        return new Scenario($data['data'] ?? []);
    }

    public function updateScenario(string $scenarioId, array $updates): Scenario
    {
        $data = $this->put($this->getApiPrefix() . '/captcha/scenarios/' . $scenarioId, $updates);
        return new Scenario($data['data'] ?? []);
    }

    public function deleteScenario(string $scenarioId): array
    {
        $data = $this->delete($this->getApiPrefix() . '/captcha/scenarios/' . $scenarioId);
        return $data['data'] ?? [];
    }

    public function registerWebhook(
        string $appId,
        string $url,
        array $events,
        ?string $secret = null,
        ?array $headers = null
    ): Webhook {
        $body = [
            'app_id' => $appId,
            'url' => $url,
            'events' => $events,
        ];

        if ($secret !== null) {
            $body['secret'] = $secret;
        }
        if ($headers !== null) {
            $body['headers'] = $headers;
        }

        $data = $this->post($this->getApiPrefix() . '/captcha/webhook/register', $body);
        return new Webhook($data['data'] ?? []);
    }

    public function listWebhooks(?string $appId = null): array
    {
        $endpoint = $this->getApiPrefix() . '/captcha/webhook';
        if ($appId !== null) {
            $endpoint .= '?app_id=' . urlencode($appId);
        }

        $data = $this->get($endpoint);
        return [
            'webhooks' => array_map(fn($w) => new Webhook($w), $data['data']['webhooks'] ?? []),
            'total' => $data['data']['total'] ?? 0,
        ];
    }

    public function updateWebhook(string $webhookId, array $updates): Webhook
    {
        $data = $this->put($this->getApiPrefix() . '/captcha/webhook/' . $webhookId, $updates);
        return new Webhook($data['data'] ?? []);
    }

    public function unregisterWebhook(string $webhookId): array
    {
        $data = $this->delete($this->getApiPrefix() . '/captcha/webhook/' . $webhookId);
        return $data['data'] ?? [];
    }

    public function createClientInfo(?array $extra = null): string
    {
        $info = [
            'platform' => 'PHP ' . PHP_VERSION,
            'timestamp' => (int)(microtime(true) * 1000),
        ];

        if ($extra !== null) {
            $info = array_merge($info, $extra);
        }

        return json_encode($info);
    }
}
