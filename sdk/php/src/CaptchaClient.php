<?php

namespace HJTPX\Captcha;

use GuzzleHttp\Psr7\Request;
use GuzzleHttp\Exception\TransferException;
use GuzzleHttp\Exception\ConnectException;
use GuzzleHttp\Exception\RequestException;
use HJTPX\Captcha\Exception\ApiException;
use HJTPX\Captcha\Exception\CaptchaException;
use HJTPX\Captcha\Exception\NetworkException;
use HJTPX\Captcha\Exception\ValidationException;
use HJTPX\Captcha\Exception\AuthenticationException;
use HJTPX\Captcha\Model\ApiResponse;
use HJTPX\Captcha\Model\ClickCaptchaResponse;
use HJTPX\Captcha\Model\ConnectCaptchaResponse;
use HJTPX\Captcha\Model\GestureCaptchaResponse;
use HJTPX\Captcha\Model\JigsawCaptchaResponse;
use HJTPX\Captcha\Model\JigsawPiece;
use HJTPX\Captcha\Model\LianliankanCaptchaResponse;
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
    const CAPTCHA_TYPE_SLIDER = 'slider';
    const CAPTCHA_TYPE_CLICK = 'click';
    const CAPTCHA_TYPE_ROTATION = 'rotation';
    const CAPTCHA_TYPE_GESTURE = 'gesture';
    const CAPTCHA_TYPE_JIGSAW = 'jigsaw';
    const CAPTCHA_TYPE_VOICE = 'voice';
    const CAPTCHA_TYPE_CONNECT = 'connect';
    const CAPTCHA_TYPE_3D = '3d';
    const CAPTCHA_TYPE_LIANLIANKAN = 'lianliankan';

    const DEFAULT_TIMEOUT = 30;
    const DEFAULT_CONNECT_TIMEOUT = 10;
    const DEFAULT_MAX_RETRIES = 3;

    protected $baseUrl;
    protected $apiKey;
    protected $secretKey;
    protected $accessToken;
    protected $poolManager;
    protected $retryManager;
    protected $signer;
    protected $logger;
    protected $debugMode;

    public function __construct(
        string $baseUrl,
        string $apiKey = null,
        string $secretKey = null,
        ConnectionPoolConfig $poolConfig = null,
        RetryConfig $retryConfig = null
    ) {
        if (empty($baseUrl)) {
            throw new ValidationException('Base URL cannot be empty');
        }

        $this->baseUrl = rtrim($baseUrl, '/');
        $this->apiKey = $apiKey;
        $this->secretKey = $secretKey;

        $this->poolManager = new ConnectionPoolManager($poolConfig ?? new ConnectionPoolConfig());
        $this->retryManager = new RetryManager($retryConfig ?? new RetryConfig());

        if ($secretKey) {
            $this->signer = new HmacSigner($secretKey);
        }

        $this->debugMode = false;
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
        $this->validateSessionId($sessionId);
        $this->validateX($x);

        $request = new VerifyCaptchaRequest();
        $request->sessionId = $sessionId;
        $request->type = self::CAPTCHA_TYPE_SLIDER;
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
        $this->validateSessionId($sessionId);
        $this->validatePoints($points);

        $request = new VerifyCaptchaRequest();
        $request->sessionId = $sessionId;
        $request->type = self::CAPTCHA_TYPE_CLICK;
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
        $this->validateSessionId($challengeId);
        $this->validateAngle($angle);

        $request = new VerifyCaptchaRequest();
        $request->sessionId = $challengeId;
        $request->type = self::CAPTCHA_TYPE_ROTATION;
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
        $this->validateSessionId($sessionId);
        $this->validatePattern($pattern);

        $request = new VerifyCaptchaRequest();
        $request->sessionId = $sessionId;
        $request->type = self::CAPTCHA_TYPE_GESTURE;
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
        $this->validateSessionId($sessionId);
        $this->validatePieces($pieces);

        $request = new VerifyCaptchaRequest();
        $request->sessionId = $sessionId;
        $request->type = self::CAPTCHA_TYPE_JIGSAW;
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
        $this->validateSessionId($sessionId);
        $this->validateAnswer($answer);

        $request = new VerifyCaptchaRequest();
        $request->sessionId = $sessionId;
        $request->type = self::CAPTCHA_TYPE_VOICE;
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
        $this->validateSessionId($sessionId);
        $this->validateConnections($connections);

        $request = new VerifyCaptchaRequest();
        $request->sessionId = $sessionId;
        $request->type = self::CAPTCHA_TYPE_CONNECT;
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
        $this->validateSessionId($sessionId);
        $this->validateTargetPosition($targetPosition);

        $request = new VerifyCaptchaRequest();
        $request->sessionId = $sessionId;
        $request->type = self::CAPTCHA_TYPE_3D;
        $request->targetPosition = $targetPosition;

        return $this->verifyCaptcha($request);
    }

    // ========== Lianliankan Captcha ==========

    public function getLianliankanCaptcha(
        int $gridRows = 4,
        int $gridCols = 4,
        int $timeLimit = 60
    ): LianliankanCaptchaResponse {
        $params = [
            'grid_rows' => $gridRows,
            'grid_cols' => $gridCols,
            'time_limit' => $timeLimit,
        ];

        $response = $this->get('/api/v1/captcha/lianliankan', $params);
        return new LianliankanCaptchaResponse($response);
    }

    public function verifyLianliankanCaptcha(
        string $sessionId,
        array $connections,
        int $timeSpent = null
    ): VerifyCaptchaResponse {
        $request = new VerifyCaptchaRequest();
        $request->sessionId = $sessionId;
        $request->type = self::CAPTCHA_TYPE_LIANLIANKAN;
        $request->connections = $connections;
        $request->timeSpent = $timeSpent;

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

    // ========== Validation Methods ==========

    protected function validateSessionId(string $sessionId): void
    {
        if (empty($sessionId)) {
            throw new ValidationException('Session ID cannot be empty');
        }
        if (strlen($sessionId) > 128) {
            throw new ValidationException('Session ID is too long (max 128 characters)');
        }
    }

    protected function validateX(int $x): void
    {
        if ($x < 0) {
            throw new ValidationException('X coordinate cannot be negative');
        }
        if ($x > 10000) {
            throw new ValidationException('X coordinate is too large');
        }
    }

    protected function validatePoints(array $points): void
    {
        if (empty($points)) {
            throw new ValidationException('Points array cannot be empty');
        }
        foreach ($points as $index => $point) {
            if (!is_array($point) || count($point) < 2) {
                throw new ValidationException("Invalid point format at index {$index}");
            }
            if (!is_int($point[0]) || !is_int($point[1])) {
                throw new ValidationException("Point coordinates must be integers at index {$index}");
            }
            if ($point[0] < 0 || $point[1] < 0) {
                throw new ValidationException("Point coordinates cannot be negative at index {$index}");
            }
        }
    }

    protected function validateAngle(int $angle): void
    {
        if ($angle < -360 || $angle > 360) {
            throw new ValidationException('Angle must be between -360 and 360 degrees');
        }
    }

    protected function validatePattern(array $pattern): void
    {
        if (empty($pattern)) {
            throw new ValidationException('Pattern array cannot be empty');
        }
        foreach ($pattern as $index => $point) {
            if (!is_int($point)) {
                throw new ValidationException("Pattern points must be integers at index {$index}");
            }
        }
    }

    protected function validatePieces(array $pieces): void
    {
        if (empty($pieces)) {
            throw new ValidationException('Pieces array cannot be empty');
        }
        foreach ($pieces as $index => $piece) {
            if (!is_array($piece)) {
                throw new ValidationException("Invalid piece format at index {$index}");
            }
            if (!isset($piece['index'])) {
                throw new ValidationException("Piece must have index at position {$index}");
            }
        }
    }

    protected function validateAnswer(string $answer): void
    {
        if (empty($answer)) {
            throw new ValidationException('Answer cannot be empty');
        }
        if (strlen($answer) > 100) {
            throw new ValidationException('Answer is too long (max 100 characters)');
        }
    }

    protected function validateConnections(array $connections): void
    {
        if (empty($connections)) {
            throw new ValidationException('Connections array cannot be empty');
        }
        foreach ($connections as $index => $connection) {
            if (!is_array($connection) || count($connection) < 2) {
                throw new ValidationException("Invalid connection format at index {$index}");
            }
        }
    }

    protected function validateTargetPosition(array $position): void
    {
        if (empty($position)) {
            throw new ValidationException('Target position cannot be empty');
        }
        if (count($position) < 2) {
            throw new ValidationException('Target position must have at least 2 coordinates');
        }
        foreach ($position as $index => $coord) {
            if (!is_int($coord) && !is_float($coord)) {
                throw new ValidationException("Target position coordinates must be numeric at index {$index}");
            }
        }
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
            $this->log('error', "HTTP request failed with status {$statusCode}: {$body}");
            throw $this->createApiException($statusCode, $body);
        }

        $data = json_decode($body, true);
        if (json_last_error() !== JSON_ERROR_NONE) {
            throw new CaptchaException('Invalid JSON response: ' . json_last_error_msg());
        }

        if (isset($data['code']) && $data['code'] !== 0) {
            throw new ApiException($data['message'] ?? 'API error', $data['code']);
        }

        return $data['data'] ?? $data;
    }

    protected function createApiException(int $statusCode, string $body): ApiException
    {
        switch ($statusCode) {
            case 401:
            case 403:
                return new AuthenticationException('Authentication failed: ' . $body, $statusCode);
            case 422:
                return new ValidationException('Validation error: ' . $body, $statusCode);
            case 429:
                return new ApiException('Rate limit exceeded', $statusCode);
            default:
                return new ApiException('Request failed: ' . $body, $statusCode);
        }
    }

    protected function handleException(\Exception $e): void
    {
        $this->log('error', get_class($e) . ': ' . $e->getMessage());

        if ($e instanceof ConnectException) {
            throw new NetworkException('Connection failed: ' . $e->getMessage(), 0, $e);
        }

        if ($e instanceof RequestException) {
            $response = $e->getResponse();
            if ($response) {
                throw new ApiException(
                    'Request failed: ' . $e->getMessage(),
                    $response->getStatusCode(),
                    $e
                );
            }
            throw new NetworkException('Network error: ' . $e->getMessage(), 0, $e);
        }

        if ($e instanceof TransferException) {
            throw new NetworkException('Transfer error: ' . $e->getMessage(), 0, $e);
        }
    }

    protected function log(string $level, string $message): void
    {
        if ($this->logger) {
            $this->logger($level, $message);
        }

        if ($this->debugMode && $level === 'error') {
            error_log('[HJTPX Captcha SDK] ' . strtoupper($level) . ': ' . $message);
        }
    }

    public function setLogger(callable $logger): void
    {
        $this->logger = $logger;
    }

    public function setDebugMode(bool $debugMode): void
    {
        $this->debugMode = $debugMode;
    }

    public function getBaseUrl(): string
    {
        return $this->baseUrl;
    }

    public function isConnected(): bool
    {
        try {
            $this->get('/api/v1/health');
            return true;
        } catch (\Exception $e) {
            return false;
        }
    }

    public function __destruct()
    {
        $this->close();
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
