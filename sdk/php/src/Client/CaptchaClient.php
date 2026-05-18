<?php

declare(strict_types=1);

namespace Hjtpx\Captcha\Client;

use GuzzleHttp\Client;
use GuzzleHttp\ClientInterface;
use GuzzleHttp\Exception\GuzzleException;
use GuzzleHttp\Exception\RequestException;
use Hjtpx\Captcha\Exceptions\CaptchaApiException;
use Hjtpx\Captcha\Exceptions\CaptchaException;
use Hjtpx\Captcha\Exceptions\CaptchaNetworkException;
use Hjtpx\Captcha\Exceptions\CaptchaTimeoutException;
use Hjtpx\Captcha\Exceptions\CaptchaValidationException;
use Psr\Http\Message\ResponseInterface;

class CaptchaClient implements \Hjtpx\Captcha\Contracts\CaptchaClientInterface
{
    private string $baseUrl;
    private ?string $apiKey;
    private int $timeout;
    private int $maxRetries;
    private float $retryBackoffFactor;
    private ?string $accessToken = null;
    private ?string $refreshToken = null;
    private ?ClientInterface $httpClient = null;

    public function __construct(
        string $baseUrl,
        ?string $apiKey = null,
        int $timeout = 30,
        int $maxRetries = 3,
        float $retryBackoffFactor = 0.5
    ) {
        $this->baseUrl = rtrim($baseUrl, '/');
        $this->apiKey = $apiKey;
        $this->timeout = $timeout;
        $this->maxRetries = $maxRetries;
        $this->retryBackoffFactor = $retryBackoffFactor;
    }

    public function setHttpClient(ClientInterface $client): void
    {
        $this->httpClient = $client;
    }

    private function getHttpClient(): ClientInterface
    {
        if ($this->httpClient === null) {
            $this->httpClient = new Client([
                'timeout' => $this->timeout,
                'connect_timeout' => 10,
            ]);
        }
        return $this->httpClient;
    }

    private function getHeaders(): array
    {
        $headers = [
            'Content-Type' => 'application/json',
            'User-Agent' => 'Captcha-PHP-SDK/15.0',
            'Accept' => 'application/json',
        ];

        if ($this->apiKey !== null) {
            $headers['X-API-Key'] = $this->apiKey;
        }

        if ($this->accessToken !== null) {
            $headers['Authorization'] = 'Bearer ' . $this->accessToken;
        }

        return $headers;
    }

    private function request(string $method, string $path, array $data = [], array $params = []): array
    {
        $url = $this->baseUrl . $path;
        $options = [
            'headers' => $this->getHeaders(),
        ];

        if (!empty($params)) {
            $options['query'] = $params;
        }

        if (!empty($data) && in_array(strtoupper($method), ['POST', 'PUT', 'PATCH'])) {
            $options['json'] = $data;
        }

        $lastException = null;

        for ($attempt = 0; $attempt <= $this->maxRetries; $attempt++) {
            try {
                $response = $this->getHttpClient()->request($method, $url, $options);
                return $this->parseResponse($response);
            } catch (RequestException $e) {
                $lastException = $e;
                $response = $e->getResponse();

                if ($response !== null) {
                    $statusCode = $response->getStatusCode();

                    if ($statusCode >= 500) {
                        if ($attempt < $this->maxRetries) {
                            $delay = $this->retryBackoffFactor * pow(2, $attempt);
                            usleep((int)($delay * 1000000));
                            continue;
                        }
                    }

                    if ($statusCode === 429) {
                        throw new CaptchaApiException(
                            'Rate limit exceeded',
                            $statusCode
                        );
                    }

                    if ($statusCode === 401) {
                        throw new CaptchaApiException(
                            'Unauthorized - Invalid API key or token',
                            $statusCode
                        );
                    }

                    if ($statusCode === 400) {
                        throw new CaptchaValidationException(
                            'Invalid request parameters',
                            $statusCode
                        );
                    }
                }

                throw new CaptchaApiException(
                    $e->getMessage(),
                    $response?->getStatusCode() ?? 0,
                    $e
                );
            } catch (GuzzleException $e) {
                $lastException = $e;

                if ($attempt < $this->maxRetries) {
                    $delay = $this->retryBackoffFactor * pow(2, $attempt);
                    usleep((int)($delay * 1000000));
                    continue;
                }

                throw new CaptchaNetworkException(
                    'Network error: ' . $e->getMessage(),
                    0,
                    $e
                );
            }
        }

        throw new CaptchaException(
            'Max retries exceeded',
            0,
            $lastException
        );
    }

    private function parseResponse(ResponseInterface $response): array
    {
        $body = (string)$response->getBody();
        $result = json_decode($body, true);

        if (json_last_error() !== JSON_ERROR_NONE) {
            throw new CaptchaApiException('Invalid JSON response');
        }

        if (!isset($result['code'])) {
            throw new CaptchaApiException('Missing code in response');
        }

        $code = (int)$result['code'];
        $message = $result['message'] ?? 'Unknown error';
        $data = $result['data'] ?? null;

        if ($code !== 0) {
            if ($code === 400) {
                throw new CaptchaValidationException($message, $code);
            }
            if ($code === 404 || str_contains(strtolower($message), 'not found') || str_contains(strtolower($message), 'expired')) {
                throw new CaptchaValidationException($message, $code);
            }
            throw new CaptchaApiException($message, $code);
        }

        return $data ?? [];
    }

    public function getSliderCaptcha(int $width = 320, int $height = 160, int $tolerance = 8): array
    {
        return $this->request('GET', '/api/v1/captcha/slider', [], [
            'width' => (string)$width,
            'height' => (string)$height,
            'tolerance' => (string)$tolerance,
        ]);
    }

    public function verifySliderCaptcha(string $sessionId, int $x, ?int $y = null, array $trajectory = []): array
    {
        $data = [
            'session_id' => $sessionId,
            'type' => 'slider',
            'x' => $x,
        ];

        if ($y !== null) {
            $data['y'] = $y;
        }

        if (!empty($trajectory)) {
            $data['trajectory'] = $trajectory;
        }

        return $this->request('POST', '/api/v1/captcha/verify', $data);
    }

    public function getClickCaptcha(string $mode = 'number', int $maxPoints = 3, bool $allowShuffle = true): array
    {
        return $this->request('GET', '/api/v1/captcha/click', [], [
            'mode' => $mode,
            'points' => (string)$maxPoints,
            'shuffle' => $allowShuffle ? 'true' : 'false',
        ]);
    }

    public function verifyClickCaptcha(string $sessionId, array $points, ?array $clickSequence = null): array
    {
        $data = [
            'session_id' => $sessionId,
            'type' => 'click',
            'points' => $points,
        ];

        if ($clickSequence !== null) {
            $data['click_sequence'] = $clickSequence;
        }

        return $this->request('POST', '/api/v1/captcha/verify', $data);
    }

    public function getImageCaptcha(string $type = 'mixed', int $count = 4): array
    {
        return $this->request('GET', '/api/v1/captcha/image', [], [
            'type' => $type,
            'count' => (string)$count,
        ]);
    }

    public function verifyImageCaptcha(string $challengeId, string $answer): array
    {
        return $this->request('POST', '/api/v1/captcha/image/verify', [
            'challenge_id' => $challengeId,
            'answer' => $answer,
        ]);
    }

    public function getRotationCaptcha(): array
    {
        return $this->request('GET', '/api/v1/captcha/rotation');
    }

    public function verifyRotationCaptcha(string $challengeId, int $angle): array
    {
        return $this->request('POST', '/api/v1/captcha/rotation/verify', [
            'challenge_id' => $challengeId,
            'angle' => $angle,
        ]);
    }

    public function getGestureCaptcha(): array
    {
        return $this->request('GET', '/api/v1/captcha/gesture');
    }

    public function verifyGestureCaptcha(string $sessionId, array $pattern): array
    {
        return $this->request('POST', '/api/v1/captcha/gesture/verify', [
            'session_id' => $sessionId,
            'pattern' => $pattern,
        ]);
    }

    public function getJigsawCaptcha(int $width = 300, int $height = 300, int $gridSize = 3): array
    {
        return $this->request('GET', '/api/v1/captcha/jigsaw', [], [
            'width' => (string)$width,
            'height' => (string)$height,
            'grid_size' => (string)$gridSize,
        ]);
    }

    public function verifyJigsawCaptcha(string $sessionId, array $pieces): array
    {
        return $this->request('POST', '/api/v1/captcha/jigsaw/verify', [
            'session_id' => $sessionId,
            'pieces' => $pieces,
        ]);
    }

    public function batchVerify(array $requests): array
    {
        if (empty($requests)) {
            return [
                'results' => [],
                'success_count' => 0,
                'failed_count' => 0,
                'total_time_ms' => 0,
            ];
        }

        $results = [];
        $successCount = 0;
        $failedCount = 0;
        $startTime = microtime(true);

        foreach ($requests as $request) {
            try {
                $result = $this->verifySliderCaptcha(
                    $request['session_id'] ?? '',
                    $request['x'] ?? 0,
                    $request['y'] ?? null,
                    $request['trajectory'] ?? []
                );

                $success = $result['success'] ?? false;
                $results[] = [
                    'session_id' => $request['session_id'] ?? '',
                    'success' => $success,
                    'message' => $result['message'] ?? '',
                    'remaining_attempts' => $result['remaining_attempts'] ?? null,
                ];

                if ($success) {
                    $successCount++;
                } else {
                    $failedCount++;
                }
            } catch (\Throwable $e) {
                $results[] = [
                    'session_id' => $request['session_id'] ?? '',
                    'success' => false,
                    'message' => $e->getMessage(),
                ];
                $failedCount++;
            }
        }

        $totalTime = (int)((microtime(true) - $startTime) * 1000);

        return [
            'results' => $results,
            'success_count' => $successCount,
            'failed_count' => $failedCount,
            'total_time_ms' => $totalTime,
        ];
    }

    public function asyncVerify(array $request): array
    {
        $data = [
            'session_id' => $request['session_id'] ?? '',
            'x' => $request['x'] ?? 0,
            'y' => $request['y'] ?? null,
            'trajectory' => $request['trajectory'] ?? [],
        ];

        if (isset($request['callback_url'])) {
            $data['callback_url'] = $request['callback_url'];
        }

        return $this->request('POST', '/api/v1/captcha/async/verify', $data);
    }

    public function getAsyncResult(string $taskId): array
    {
        return $this->request('GET', "/api/v1/captcha/async/result/{$taskId}");
    }

    public function login(string $username, string $password, ?string $captchaToken = null): array
    {
        $data = [
            'username' => $username,
            'password' => $password,
        ];

        if ($captchaToken !== null) {
            $data['captcha_token'] = $captchaToken;
        }

        $result = $this->request('POST', '/api/v1/auth/login', $data);

        $this->accessToken = $result['access_token'] ?? null;
        $this->refreshToken = $result['refresh_token'] ?? null;

        return $result;
    }

    public function logout(): bool
    {
        try {
            $this->request('POST', '/api/v1/auth/logout');
            $this->accessToken = null;
            $this->refreshToken = null;
            return true;
        } catch (\Throwable $e) {
            $this->accessToken = null;
            $this->refreshToken = null;
            return false;
        }
    }

    public function getDetectionScript(?string $callback = null): string
    {
        $params = [];
        if ($callback !== null) {
            $params['callback'] = $callback;
        }

        $url = $this->baseUrl . '/api/v1/detect/script';

        $options = [
            'headers' => $this->getHeaders(),
            'timeout' => $this->timeout,
        ];

        if (!empty($params)) {
            $options['query'] = $params;
        }

        try {
            $response = $this->getHttpClient()->request('GET', $url, $options);
            return (string)$response->getBody();
        } catch (GuzzleException $e) {
            throw new CaptchaNetworkException(
                'Failed to fetch detection script: ' . $e->getMessage(),
                0,
                $e
            );
        }
    }

    public function submitDetection(array $data): array
    {
        return $this->request('POST', '/api/v1/detect/submit', $data);
    }

    public function checkEnvironment(array $data): array
    {
        return $this->request('POST', '/api/v1/detect/check', $data);
    }

    public function setAccessToken(string $token): void
    {
        $this->accessToken = $token;
    }

    public function setRefreshToken(string $token): void
    {
        $this->refreshToken = $token;
    }

    public function getAccessToken(): ?string
    {
        return $this->accessToken;
    }

    public function getRefreshToken(): ?string
    {
        return $this->refreshToken;
    }
}
