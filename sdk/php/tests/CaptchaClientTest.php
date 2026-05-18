<?php

namespace HJTPX\Captcha\Tests;

use HJTPX\Captcha\CaptchaClient;
use HJTPX\Captcha\Exception\ApiException;
use HJTPX\Captcha\Exception\CaptchaException;
use HJTPX\Captcha\Exception\NetworkException;
use HJTPX\Captcha\Exception\ValidationException;
use HJTPX\Captcha\Exception\AuthenticationException;
use HJTPX\Captcha\Model\SliderCaptchaResponse;
use HJTPX\Captcha\Model\ClickCaptchaResponse;
use HJTPX\Captcha\Model\RotationCaptchaResponse;
use HJTPX\Captcha\Model\GestureCaptchaResponse;
use HJTPX\Captcha\Model\JigsawCaptchaResponse;
use HJTPX\Captcha\Model\VoiceCaptchaResponse;
use HJTPX\Captcha\Model\ConnectCaptchaResponse;
use HJTPX\Captcha\Model\ThreeDCaptchaResponse;
use HJTPX\Captcha\Model\VerifyCaptchaResponse;
use HJTPX\Captcha\Model\LoginResponse;
use HJTPX\Captcha\Model\TrajectoryPoint;
use HJTPX\Captcha\Model\JigsawPiece;
use HJTPX\Captcha\Model\VerifyCaptchaRequest;
use HJTPX\Captcha\Model\LoginRequest;
use HJTPX\Captcha\Pool\ConnectionPoolConfig;
use HJTPX\Captcha\Pool\ConnectionPoolManager;
use HJTPX\Captcha\Retry\RetryConfig;
use HJTPX\Captcha\Retry\RetryManager;
use PHPUnit\Framework\TestCase;

class CaptchaClientTest extends TestCase
{
    protected $client;

    protected function setUp(): void
    {
        $this->client = new CaptchaClient(
            'http://localhost:8080',
            'test-api-key',
            'test-secret-key',
            new ConnectionPoolConfig(5, 10, 10),
            new RetryConfig(2, 50, 1.5)
        );
    }

    public function testClientInitialization(): void
    {
        $this->assertInstanceOf(CaptchaClient::class, $this->client);
    }

    public function testClientConfiguration(): void
    {
        $poolConfig = new ConnectionPoolConfig(10, 30, 30);
        $retryConfig = new RetryConfig(3, 100, 2.0);

        $client = new CaptchaClient(
            'http://test.com',
            'api-key',
            'secret-key',
            $poolConfig,
            $retryConfig
        );

        $this->assertInstanceOf(CaptchaClient::class, $client);
    }

    public function testHmacSigner(): void
    {
        $signer = new \HJTPX\Captcha\Signer\HmacSigner('test-secret');
        $data = 'test-data';
        $signature = $signer->sign($data);

        $this->assertIsString($signature);
        $this->assertEquals(64, strlen($signature));
        $this->assertTrue($signer->verify($data, $signature));
    }

    public function testHmacSignerVerifyFailsWithWrongData(): void
    {
        $signer = new \HJTPX\Captcha\Signer\HmacSigner('test-secret');
        $data = 'test-data';
        $signature = $signer->sign($data);

        $this->assertFalse($signer->verify('wrong-data', $signature));
    }

    public function testRetryManager(): void
    {
        $retryConfig = new RetryConfig(2, 10, 2);
        $retryManager = new RetryManager($retryConfig);

        $attempts = 0;
        $result = $retryManager->execute(function () use (&$attempts) {
            $attempts++;
            if ($attempts < 3) {
                throw new \Exception('Temporary error');
            }
            return 'success';
        });

        $this->assertEquals('success', $result);
        $this->assertEquals(3, $attempts);
    }

    public function testRetryManagerFailsAfterMaxRetries(): void
    {
        $retryConfig = new RetryConfig(2, 10, 1.5);
        $retryManager = new RetryManager($retryConfig);

        $attempts = 0;
        $this->expectException(\Exception::class);
        $retryManager->execute(function () use (&$attempts) {
            $attempts++;
            throw new \Exception('Persistent error');
        });
    }

    public function testModelClasses(): void
    {
        $sliderData = [
            'session_id' => 'test-session',
            'image_url' => 'http://example.com/image.jpg',
            'puzzle_url' => 'http://example.com/puzzle.jpg',
            'secret_y' => 100
        ];
        $sliderResponse = new SliderCaptchaResponse($sliderData);
        $this->assertEquals('test-session', $sliderResponse->sessionId);
        $this->assertEquals('http://example.com/image.jpg', $sliderResponse->imageUrl);
        $this->assertEquals(100, $sliderResponse->secretY);

        $verifyData = [
            'success' => true,
            'message' => 'Verification successful',
            'remaining_attempts' => 3
        ];
        $verifyResponse = new VerifyCaptchaResponse($verifyData);
        $this->assertTrue($verifyResponse->success);
        $this->assertEquals('Verification successful', $verifyResponse->message);
        $this->assertEquals(3, $verifyResponse->remainingAttempts);
    }

    public function testClickCaptchaResponse(): void
    {
        $data = [
            'session_id' => 'click-session',
            'image_url' => 'http://example.com/click.jpg',
            'hint' => 'Click 1, 2, 3',
            'hint_order' => ['1', '2', '3'],
            'max_points' => 3,
            'mode' => 'number',
            'allow_shuffle' => true
        ];

        $response = new ClickCaptchaResponse($data);
        $this->assertEquals('click-session', $response->sessionId);
        $this->assertEquals('Click 1, 2, 3', $response->hint);
        $this->assertEquals(3, $response->maxPoints);
    }

    public function testRotationCaptchaResponse(): void
    {
        $data = [
            'challenge_id' => 'rotation-challenge',
            'image_url' => 'http://example.com/rotation.jpg'
        ];

        $response = new RotationCaptchaResponse($data);
        $this->assertEquals('rotation-challenge', $response->challengeId);
        $this->assertEquals('http://example.com/rotation.jpg', $response->imageUrl);
    }

    public function testGestureCaptchaResponse(): void
    {
        $data = [
            'session_id' => 'gesture-session',
            'pattern' => '1-2-3-4',
            'grid_size' => 3,
            'hint' => 'Draw pattern'
        ];

        $response = new GestureCaptchaResponse($data);
        $this->assertEquals('gesture-session', $response->sessionId);
        $this->assertEquals('1-2-3-4', $response->pattern);
        $this->assertEquals(3, $response->gridSize);
    }

    public function testVoiceCaptchaResponse(): void
    {
        $data = [
            'session_id' => 'voice-session',
            'audio_url' => 'http://example.com/audio.mp3',
            'text' => '123456'
        ];

        $response = new VoiceCaptchaResponse($data);
        $this->assertEquals('voice-session', $response->sessionId);
        $this->assertEquals('123456', $response->text);
    }

    public function testConnectCaptchaResponse(): void
    {
        $data = [
            'session_id' => 'connect-session',
            'image_url' => 'http://example.com/connect.jpg'
        ];

        $response = new ConnectCaptchaResponse($data);
        $this->assertEquals('connect-session', $response->sessionId);
    }

    public function testThreeDCaptchaResponse(): void
    {
        $data = [
            'session_id' => '3d-session',
            'scene_url' => 'http://example.com/scene.glb',
            'target_id' => 'target-123',
            'hint' => 'Find the target'
        ];

        $response = new ThreeDCaptchaResponse($data);
        $this->assertEquals('3d-session', $response->sessionId);
        $this->assertEquals('target-123', $response->targetId);
    }

    public function testTrajectoryPoint(): void
    {
        $point = new TrajectoryPoint(100, 50, 123456789);
        $this->assertEquals(100, $point->x);
        $this->assertEquals(50, $point->y);
        $this->assertEquals(123456789, $point->t);

        $array = $point->toArray();
        $this->assertEquals(['x' => 100, 'y' => 50, 't' => 123456789], $array);
    }

    public function testJigsawPiece(): void
    {
        $pieceData = [
            'index' => 0,
            'original_x' => 0,
            'original_y' => 0,
            'current_x' => 100,
            'current_y' => 100,
            'width' => 100,
            'height' => 100,
            'rotation' => 0
        ];
        $piece = new JigsawPiece($pieceData);
        $this->assertEquals(0, $piece->index);
        $this->assertEquals(100, $piece->currentX);

        $array = $piece->toArray();
        $this->assertEquals($pieceData, $array);
    }

    public function testVerifyCaptchaRequest(): void
    {
        $request = new VerifyCaptchaRequest();
        $request->sessionId = 'test-session';
        $request->type = 'slider';
        $request->x = 150;
        $request->y = 100;

        $array = $request->toArray();
        $this->assertEquals('test-session', $array['session_id']);
        $this->assertEquals('slider', $array['type']);
        $this->assertEquals(150, $array['x']);
        $this->assertEquals(100, $array['y']);
    }

    public function testVerifyCaptchaRequestWithTrajectory(): void
    {
        $request = new VerifyCaptchaRequest();
        $request->sessionId = 'test-session';
        $request->type = 'slider';
        $request->x = 150;
        $request->trajectory = [
            ['x' => 0, 'y' => 100, 't' => 123456],
            ['x' => 50, 'y' => 100, 't' => 123556],
            ['x' => 150, 'y' => 100, 't' => 123656],
        ];

        $array = $request->toArray();
        $this->assertCount(3, $array['trajectory']);
        $this->assertEquals(0, $array['trajectory'][0]['x']);
    }

    public function testVerifyCaptchaRequestWithPoints(): void
    {
        $request = new VerifyCaptchaRequest();
        $request->sessionId = 'test-session';
        $request->type = 'click';
        $request->points = [[100, 100], [200, 200]];

        $array = $request->toArray();
        $this->assertEquals('click', $array['type']);
        $this->assertCount(2, $array['points']);
    }

    public function testLoginRequest(): void
    {
        $request = new LoginRequest();
        $request->username = 'testuser';
        $request->password = 'password123';
        $request->captchaToken = 'captcha-token';

        $array = $request->toArray();
        $this->assertEquals('testuser', $array['username']);
        $this->assertEquals('password123', $array['password']);
        $this->assertEquals('captcha-token', $array['captcha_token']);
    }

    public function testLoginResponse(): void
    {
        $data = [
            'access_token' => 'token123',
            'refresh_token' => 'refresh456',
            'expires_in' => 3600,
            'user' => [
                'id' => 1,
                'username' => 'testuser',
                'email' => 'test@example.com'
            ]
        ];

        $response = new LoginResponse($data);
        $this->assertEquals('token123', $response->accessToken);
        $this->assertEquals('refresh456', $response->refreshToken);
        $this->assertEquals(3600, $response->expiresIn);
        $this->assertEquals('testuser', $response->user['username']);
    }

    public function testConnectionPoolConfig(): void
    {
        $config = new ConnectionPoolConfig(10, 30, 30);
        $this->assertEquals(10, $config->getMaxConnections());
        $this->assertEquals(30, $config->getConnectionTimeout());
        $this->assertEquals(30, $config->getRequestTimeout());
    }

    public function testRetryConfig(): void
    {
        $config = new RetryConfig(5, 200, 2.0, [429, 500, 502, 503, 504]);
        $this->assertEquals(5, $config->getMaxRetries());
        $this->assertEquals(200, $config->getInitialDelayMs());
        $this->assertEquals(2.0, $config->getBackoffMultiplier());
        $this->assertContains(429, $config->getRetryableStatusCodes());
    }

    public function testExceptionClasses(): void
    {
        $apiException = new ApiException('API Error', 400);
        $this->assertEquals('API Error', $apiException->getMessage());
        $this->assertEquals(400, $apiException->getCode());

        $captchaException = new CaptchaException('Captcha Error');
        $this->assertEquals('Captcha Error', $captchaException->getMessage());

        $networkException = new NetworkException('Network Error');
        $this->assertEquals('Network Error', $networkException->getMessage());

        $validationException = new ValidationException('Validation Error');
        $this->assertEquals('Validation Error', $validationException->getMessage());

        $authException = new AuthenticationException('Auth Error');
        $this->assertEquals('Auth Error', $authException->getMessage());
    }

    public function testAccessTokenManagement(): void
    {
        $this->assertNull($this->client->getAccessToken());

        $this->client->setAccessToken('test-token');
        $this->assertEquals('test-token', $this->client->getAccessToken());
    }

    public function testCloseMethod(): void
    {
        $this->client->close();
        $this->assertTrue(true);
    }
}

class CaptchaClientIntegrationTest extends TestCase
{
    public function testAllCaptchaTypeResponses(): void
    {
        $sliderData = [
            'session_id' => 'slider-123',
            'image_url' => 'http://example.com/slider.jpg',
            'puzzle_url' => 'http://example.com/puzzle.jpg',
            'secret_y' => 80
        ];
        $slider = new SliderCaptchaResponse($sliderData);
        $this->assertInstanceOf(SliderCaptchaResponse::class, $slider);

        $clickData = [
            'session_id' => 'click-123',
            'image_url' => 'http://example.com/click.jpg',
            'hint' => 'Click icons',
            'hint_order' => ['1', '2'],
            'max_points' => 2
        ];
        $click = new ClickCaptchaResponse($clickData);
        $this->assertInstanceOf(ClickCaptchaResponse::class, $click);

        $rotationData = [
            'challenge_id' => 'rotation-123',
            'image_url' => 'http://example.com/rotation.jpg'
        ];
        $rotation = new RotationCaptchaResponse($rotationData);
        $this->assertInstanceOf(RotationCaptchaResponse::class, $rotation);

        $gestureData = [
            'session_id' => 'gesture-123',
            'pattern' => '1-2-3',
            'grid_size' => 3
        ];
        $gesture = new GestureCaptchaResponse($gestureData);
        $this->assertInstanceOf(GestureCaptchaResponse::class, $gesture);

        $voiceData = [
            'session_id' => 'voice-123',
            'audio_url' => 'http://example.com/voice.mp3',
            'text' => '1234'
        ];
        $voice = new VoiceCaptchaResponse($voiceData);
        $this->assertInstanceOf(VoiceCaptchaResponse::class, $voice);

        $threeDData = [
            'session_id' => '3d-123',
            'scene_url' => 'http://example.com/scene.glb',
            'target_id' => 'target-1'
        ];
        $threeD = new ThreeDCaptchaResponse($threeDData);
        $this->assertInstanceOf(ThreeDCaptchaResponse::class, $threeD);
    }

    public function testVerifyResultWithAllFields(): void
    {
        $data = [
            'success' => true,
            'message' => 'Verification passed',
            'remaining_attempts' => 3,
            'trajectory_result' => [
                'score' => 0.95,
                'passed' => true,
                'reasons' => []
            ],
            'risk_score' => 0.1,
            'captcha_pass' => true,
            'fail_reason' => null
        ];

        $response = new VerifyCaptchaResponse($data);
        $this->assertTrue($response->success);
        $this->assertEquals('Verification passed', $response->message);
        $this->assertEquals(3, $response->remainingAttempts);
        $this->assertEquals(0.95, $response->trajectoryResult['score']);
        $this->assertEquals(0.1, $response->riskScore);
    }
}
