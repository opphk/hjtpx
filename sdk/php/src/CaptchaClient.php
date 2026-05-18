<?php

namespace HJTPX\Captcha;

use GuzzleHttp\Psr7\Request;
use HJTPX\Captcha\Exception\ApiException;
use HJTPX\Captcha\Exception\CaptchaException;
use HJTPX\Captcha\Exception\NetworkException;
use HJTPX\Captcha\Model\ApiResponse;
use HJTPX\Captcha\Model\ClickCaptchaResponse;
use HJTPX\Captcha\Model\ConnectCaptchaResponse;
use HJTPX\Captcha\Model\GestureCaptchaResponse;
use HJTPX\Captcha\Model\JigsawCaptchaResponse;
use HJTPX\Captcha\Model\JigsawPiece;
use HJTPX\Captcha\Model\LoginRequest;
use HJTPX\Captcha\Model\LoginResponse;
use HJTPX\Captcha\Model\RotationCaptchaResponse;
use HJTPX\Captcha\Model\SliderCaptchaResponse;
use HJTPX\Captcha\Model\ThreeDCaptchaResponse;
use HJTPX\Captcha\Model\TrajectoryPoint;
use HJTPX\Captcha\Model\VerifyCaptchaRequest;
use HJTPX\Captcha\Model\VerifyCaptchaResponse;
use HJTPX\Captcha\Model\VoiceCaptchaResponse;
use HJTPX\Captcha\Pool\ConnectionPoolConfig;
use HJTPX\Captcha\Pool\ConnectionPoolManager;
use HJTPX\Captcha\Retry\RetryConfig;
use HJTPX\Captcha\Retry\RetryManager;
use HJTPX\Captcha\Signer\HmacSigner;

class CaptchaClient
{
    protected $baseUrl;
    protected $apiKey;
    protected $secretKey;
    protected $accessToken;
    protected $poolManager;
    protected $retryManager;
    protected $signer;

    public function __construct(
        string $baseUrl,
        string $apiKey = null,
        string $secretKey = null,
        ConnectionPoolConfig $poolConfig = null,
        RetryConfig $retryConfig = null
    ) {
        $this->baseUrl = rtrim($baseUrl, '/');
        $this->apiKey = $apiKey;
        $this->secretKey = $secretKey;

        $this->poolManager = new ConnectionPoolManager($poolConfig ?? new ConnectionPoolConfig());
        $this->retryManager = new RetryManager($retryConfig ?? new RetryConfig());

        if ($secretKey) {
            $this->signer = new HmacSigner($secretKey);
        }
    }

    // ========== Slider Captcha ==========

    public function getSliderCaptcha(
        int $width = null,
        int $height = null,
        int $tolerance = null
    ): SliderCaptchaResponse {
        $params = [];
        if ($width !== null) $params['width'] = $width;
        if ($height !== null) $params['height'] = $height;
        if ($tolerance !== null) $params['tolerance'] = $tolerance;

        $response = $this->get('/api/v1/captcha/slider', $params);
        return new SliderCaptchaResponse($response);
    }

    public function verifySliderCaptcha(
        string $sessionId,
        int $x,
        int $y = null,
        array $trajectory = null
    ): VerifyCaptchaResponse {
        $request = new VerifyCaptchaRequest();
        $request->sessionId = $sessionId;
        $request->type = 'slider';
        $request->x = $x;
        $request->y = $y;
        $request->trajectory = $trajectory;

        return $this->verifyCaptcha($request);
    }

    // ========== Click Captcha ==========

    public function getClickCaptcha(
        string $mode = null,
        bool $shuffle = null,
        int $points = null
    ): ClickCaptchaResponse {
        $params = [];
        if ($mode !== null) $params['mode'] = $mode;
        if ($shuffle !== null) $params['shuffle'] = $shuffle;
        if ($points !== null) $params['points'] = $points;

        $response = $this->get('/api/v1/captcha/click', $params);
        return new ClickCaptchaResponse($response);
    }

    public function verifyClickCaptcha(
        string $sessionId,
        array $points,
        array $clickSequence = null
    ): VerifyCaptchaResponse {
        $request = new VerifyCaptchaRequest();
        $request->sessionId = $sessionId;
        $request->type = 'click';
        $request->points = $points;
        $request->clickSequence = $clickSequence;

        return $this->verifyCaptcha($request);
    }

    // ========== Rotation Captcha ==========

    public function getRotationCaptcha(): RotationCaptchaResponse
    {
        $response = $this->get('/api/v1/captcha/rotation');
        return new RotationCaptchaResponse($response);
    }

    public function verifyRotationCaptcha(
        string $challengeId,
        int $angle
    ): VerifyCaptchaResponse {
        $request = new VerifyCaptchaRequest();
        $request->sessionId = $challengeId;
        $request->type = 'rotation';
        $request->angle = $angle;

        return $this->post('/api/v1/captcha/rotation/verify', $request->toArray());
    }

    // ========== Gesture Captcha ==========

    public function getGestureCaptcha(): GestureCaptchaResponse
    {
        $response = $this->get('/api/v1/captcha/gesture');
        return new GestureCaptchaResponse($response);
    }

    public function verifyGestureCaptcha(
        string $sessionId,
        array $pattern
    ): VerifyCaptchaResponse {
        $request = new VerifyCaptchaRequest();
        $request->sessionId = $sessionId;
        $request->type = 'gesture';
        $request->pattern = $pattern;

        return $this->post('/api/v1/captcha/gesture/verify', $request->toArray());
    }

    // ========== Jigsaw Captcha ==========

    public function getJigsawCaptcha(
        int $width = null,
        int $height = null,
        int $gridSize = null
    ): JigsawCaptchaResponse {
        $params = [];
        if ($width !== null) $params['width'] = $width;
        if ($height !== null) $params['height'] = $height;
        if ($gridSize !== null) $params['grid_size'] = $gridSize;

        $response = $this->get('/api/v1/captcha/jigsaw', $params);
        return new JigsawCaptchaResponse($response);
    }

    public function verifyJigsawCaptcha(
        string $sessionId,
        array $pieces
    ): VerifyCaptchaResponse {
        $request = new VerifyCaptchaRequest();
        $request->sessionId = $sessionId;
        $request->type = 'jigsaw';
        $request->pieces = $pieces;

        return $this->post('/api/v1/captcha/jigsaw/verify', $request->toArray());
    }

    // ========== Voice Captcha ==========

    public function getVoiceCaptcha(string $language = null): VoiceCaptchaResponse
    {
        $params = [];
        if ($language !== null) $params['language'] = $language;

        $response = $this->get('/api/v1/captcha/voice', $params);
        return new VoiceCaptchaResponse($response);
    }

    public function verifyVoiceCaptcha(
        string $sessionId,
        string $answer
    ): VerifyCaptchaResponse {
        $request = new VerifyCaptchaRequest();
        $request->sessionId = $sessionId;
        $request->type = 'voice';
        $request->answer = $answer;

        return $this->verifyCaptcha($request);
    }

    // ========== Connect Captcha ==========

    public function getConnectCaptcha(): ConnectCaptchaResponse
    {
        $response = $this->get('/api/v1/captcha/connect');
        return new ConnectCaptchaResponse($response);
    }

    public function verifyConnectCaptcha(
        string $sessionId,
        array $connections
    ): VerifyCaptchaResponse {
        $request = new VerifyCaptchaRequest();
        $request->sessionId = $sessionId;
        $request->type = 'connect';
        $request->connections = $connections;

        return $this->verifyCaptcha($request);
    }

    // ========== 3D Captcha ==========

    public function getThreeDCaptcha(): ThreeDCaptchaResponse
    {
        $response = $this->get('/api/v1/captcha/3d');
        return new ThreeDCaptchaResponse($response);
    }

    public function verifyThreeDCaptcha(
        string $sessionId,
        array $targetPosition
    ): VerifyCaptchaResponse {
        $request = new VerifyCaptchaRequest();
        $request->sessionId = $sessionId;
        $request->type = '3d';
        $request->targetPosition = $targetPosition;

        return $this->verifyCaptcha($request);
    }

    // ========== Generic Verify ==========

    public function verifyCaptcha(VerifyCaptchaRequest $request): VerifyCaptchaResponse
    {
        $response = $this->post('/api/v1/captcha/verify', $request->toArray());
        return new VerifyCaptchaResponse($response);
    }

    // ========== Authentication ==========

    public function login(string $username, string $password, string $captchaToken = null): LoginResponse
    {
        $request = new LoginRequest();
        $request->username = $username;
        $request->password = $password;
        $request->captchaToken = $captchaToken;

        $response = $this->post('/api/v1/auth/login', $request->toArray());
        $this->accessToken = $response['access_token'] ?? null;
        return new LoginResponse($response);
    }

    public function logout(): void
    {
        $this->post('/api/v1/auth/logout');
        $this->accessToken = null;
    }

    // ========== Detection ==========

    public function getDetectionScript(string $callback = null): string
    {
        $params = [];
        if ($callback !== null) $params['callback'] = $callback;

        return $this->rawGet('/api/v1/detect/script', $params);
    }

    public function submitDetection(array $data): array
    {
        return $this->post('/api/v1/detect/submit', $data);
    }

    public function checkEnvironment(array $data): array
    {
        return $this->post('/api/v1/detect/check', $data);
    }

    // ========== HTTP Methods ==========

    protected function get(string $path, array $params = []): array
    {
        return $this->retryManager->execute(function () use ($path, $params) {
            $url = $this->baseUrl . $path;
            if (!empty($params)) {
                $url .= '?' . http_build_query($params);
            }

            $request = new Request('GET', $url, $this->buildHeaders($path));
            $response = $this->poolManager->getClient()->send($request);

            return $this->handleResponse($response);
        });
    }

    protected function post(string $path, array $data = []): array
    {
        return $this->retryManager->execute(function () use ($path, $data) {
            $url = $this->baseUrl . $path;
            $body = json_encode($data);

            $request = new Request(
                'POST',
                $url,
                array_merge(['Content-Type' => 'application/json'], $this->buildHeaders($path)),
                $body
            );

            $response = $this->poolManager->getClient()->send($request);

            return $this->handleResponse($response);
        });
    }

    protected function rawGet(string $path, array $params = []): string
    {
        return $this->retryManager->execute(function () use ($path, $params) {
            $url = $this->baseUrl . $path;
            if (!empty($params)) {
                $url .= '?' . http_build_query($params);
            }

            $request = new Request('GET', $url, $this->buildHeaders($path));
            $response = $this->poolManager->getClient()->send($request);

            $statusCode = $response->getStatusCode();
            if ($statusCode < 200 || $statusCode >= 300) {
                throw new ApiException('Request failed', $statusCode);
            }

            return (string) $response->getBody();
        });
    }

    protected function buildHeaders(string $path): array
    {
        $headers = [
            'User-Agent' => 'HJTPX-Captcha-PHP-SDK/1.0.0',
        ];

        if ($this->apiKey) {
            $headers['X-API-Key'] = $this->apiKey;
        }

        if ($this->accessToken) {
            $headers['Authorization'] = 'Bearer ' . $this->accessToken;
        }

        if ($this->signer) {
            $timestamp = time() * 1000;
            $dataToSign = $timestamp . ':' . $path;
            $signature = $this->signer->sign($dataToSign);
            $headers['X-Timestamp'] = (string) $timestamp;
            $headers['X-Signature'] = $signature;
        }

        return $headers;
    }

    protected function handleResponse(\Psr\Http\Message\ResponseInterface $response): array
    {
        $statusCode = $response->getStatusCode();
        $body = (string) $response->getBody();

        if ($statusCode < 200 || $statusCode >= 300) {
            throw new ApiException('Request failed: ' . $body, $statusCode);
        }

        $data = json_decode($body, true);
        if (json_last_error() !== JSON_ERROR_NONE) {
            throw new CaptchaException('Invalid JSON response');
        }

        if (isset($data['code']) && $data['code'] !== 0) {
            throw new ApiException($data['message'] ?? 'API error', $data['code']);
        }

        return $data['data'] ?? $data;
    }

    public function close(): void
    {
        $this->poolManager->close();
    }

    public function getAccessToken(): ?string
    {
        return $this->accessToken;
    }

    public function setAccessToken(?string $accessToken): void
    {
        $this->accessToken = $accessToken;
    }
}
