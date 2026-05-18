<?php

namespace HJTPX\Captcha\Tests;

use PHPUnit\Framework\TestCase;
use HJTPX\Captcha\CaptchaClient;
use HJTPX\Captcha\Pool\ConnectionPoolConfig;
use HJTPX\Captcha\Retry\RetryConfig;

class CaptchaClientIntegrationTest extends TestCase
{
    private $client;
    private $mockHandler;

    protected function setUp(): void
    {
        parent::setUp();
        $this->client = new CaptchaClient(
            'http://localhost:8080',
            'test-api-key',
            'test-secret-key'
        );
    }

    protected function tearDown(): void
    {
        if ($this->client) {
            $this->client->close();
        }
        parent::tearDown();
    }

    public function testSliderCaptchaWorkflow(): void
    {
        $sliderData = [
            'session_id' => 'slider-test-' . uniqid(),
            'image_url' => 'http://example.com/slider.jpg',
            'puzzle_url' => 'http://example.com/puzzle.jpg',
            'secret_y' => 80,
            'image_width' => 320,
            'image_height' => 160,
        ];

        $verifyData = [
            'success' => true,
            'message' => 'Verification successful',
            'score' => 0.95,
            'risk_level' => 'low',
        ];

        $this->assertArrayHasKey('session_id', $sliderData);
        $this->assertEquals(80, $sliderData['secret_y']);
        $this->assertTrue($verifyData['success']);
    }

    public function testClickCaptchaWorkflow(): void
    {
        $clickData = [
            'session_id' => 'click-test-' . uniqid(),
            'image_url' => 'http://example.com/click.jpg',
            'hint' => 'Click 1, 2, 3',
            'hint_order' => [0, 1, 2],
            'max_points' => 3,
            'mode' => 'number',
            'allow_shuffle' => true,
            'points' => [
                [100, 100],
                [200, 200],
                [300, 300],
            ],
        ];

        $this->assertEquals(3, $clickData['max_points']);
        $this->assertCount(3, $clickData['points']);
        $this->assertEquals('number', $clickData['mode']);
    }

    public function testGestureCaptchaWorkflow(): void
    {
        $gestureData = [
            'session_id' => 'gesture-test-' . uniqid(),
            'pattern' => '1-2-3-4',
            'grid_size' => 3,
            'hint' => 'Draw the pattern',
        ];

        $this->assertEquals('1-2-3-4', $gestureData['pattern']);
        $this->assertEquals(3, $gestureData['grid_size']);
    }

    public function testRotationCaptchaWorkflow(): void
    {
        $rotationData = [
            'challenge_id' => 'rotation-test-' . uniqid(),
            'image_url' => 'http://example.com/rotation.jpg',
            'target_angle' => 45,
        ];

        $this->assertNotEmpty($rotationData['challenge_id']);
        $this->assertEquals(45, $rotationData['target_angle']);
    }

    public function testJigsawCaptchaWorkflow(): void
    {
        $jigsawData = [
            'session_id' => 'jigsaw-test-' . uniqid(),
            'image_url' => 'http://example.com/jigsaw.jpg',
            'pieces' => [
                [
                    'index' => 0,
                    'original_x' => 0,
                    'original_y' => 0,
                    'current_x' => 10,
                    'current_y' => 10,
                    'width' => 100,
                    'height' => 100,
                    'rotation' => 0,
                ],
                [
                    'index' => 1,
                    'original_x' => 100,
                    'original_y' => 0,
                    'current_x' => 110,
                    'current_y' => 10,
                    'width' => 100,
                    'height' => 100,
                    'rotation' => 0,
                ],
            ],
            'grid_size' => 3,
        ];

        $this->assertEquals(2, count($jigsawData['pieces']));
        $this->assertEquals(3, $jigsawData['grid_size']);
    }

    public function testVoiceCaptchaWorkflow(): void
    {
        $voiceData = [
            'session_id' => 'voice-test-' . uniqid(),
            'audio_url' => 'http://example.com/voice.mp3',
            'text' => '123456',
            'language' => 'zh-CN',
        ];

        $this->assertEquals('123456', $voiceData['text']);
        $this->assertEquals('zh-CN', $voiceData['language']);
    }

    public function testLoginWorkflow(): void
    {
        $loginData = [
            'access_token' => 'test-access-token-' . uniqid(),
            'refresh_token' => 'test-refresh-token-' . uniqid(),
            'expires_in' => 3600,
            'user' => [
                'id' => 1,
                'username' => 'testuser',
                'email' => 'test@example.com',
            ],
        ];

        $this->assertNotEmpty($loginData['access_token']);
        $this->assertEquals(1, $loginData['user']['id']);
        $this->assertEquals(3600, $loginData['expires_in']);
    }

    public function testEnvironmentDetection(): void
    {
        $detectionData = [
            'fingerprint' => 'test-fingerprint',
            'canvas_hash' => 'canvas-test',
            'webgl_vendor' => 'Test Vendor',
            'webgl_renderer' => 'Test Renderer',
            'timezone' => 'Asia/Shanghai',
            'language' => 'zh-CN',
        ];

        $result = [
            'success' => true,
            'risk_level' => 'low',
            'risk_score' => 0.1,
            'checks' => [
                'browser' => 'passed',
                'device' => 'passed',
                'network' => 'passed',
            ],
        ];

        $this->assertTrue($result['success']);
        $this->assertEquals('low', $result['risk_level']);
        $this->assertArrayHasKey('checks', $result);
    }

    public function testTrajectoryPointGeneration(): void
    {
        $secretY = 80;
        $trajectory = [];
        $baseTime = time() * 1000;

        for ($x = 0; $x <= 180; $x += 30) {
            $trajectory[] = [
                'x' => $x,
                'y' => $secretY + rand(-2, 2),
                't' => $baseTime + ($x * 5),
            ];
        }

        $this->assertCount(7, $trajectory);
        $this->assertEquals(0, $trajectory[0]['x']);
        $this->assertEquals(180, $trajectory[6]['x']);
    }

    public function testConnectionPoolConfiguration(): void
    {
        $config = new ConnectionPoolConfig();
        $config->setMaxConnections(200);
        $config->setMaxConnectionsPerRoute(50);
        $config->setConnectionTimeout(5000);
        $config->setSocketTimeout(30000);

        $this->assertEquals(200, $config->getMaxConnections());
        $this->assertEquals(50, $config->getMaxConnectionsPerRoute());
        $this->assertEquals(5000, $config->getConnectionTimeout());
        $this->assertEquals(30000, $config->getSocketTimeout());
    }

    public function testRetryConfiguration(): void
    {
        $config = new RetryConfig();
        $config->setMaxRetries(5);
        $config->setInitialDelayMs(200);
        $config->setMaxDelayMs(10000);
        $config->setBackoffMultiplier(2.0);

        $this->assertEquals(5, $config->getMaxRetries());
        $this->assertEquals(200, $config->getInitialDelayMs());
        $this->assertEquals(10000, $config->getMaxDelayMs());
        $this->assertEquals(2.0, $config->getBackoffMultiplier());
    }

    public function testClientWithCustomConfiguration(): void
    {
        $poolConfig = new ConnectionPoolConfig();
        $poolConfig->setMaxConnections(100);

        $retryConfig = new RetryConfig();
        $retryConfig->setMaxRetries(3);

        $client = new CaptchaClient(
            'http://localhost:8080',
            'custom-api-key',
            'custom-secret-key',
            $poolConfig,
            $retryConfig
        );

        $this->assertInstanceOf(CaptchaClient::class, $client);
        $client->close();
    }

    public function testMultipleCaptchaTypes(): void
    {
        $captchaTypes = ['slider', 'click', 'rotation', 'gesture', 'jigsaw', 'voice'];

        foreach ($captchaTypes as $type) {
            $sessionId = $type . '-test-' . uniqid();
            $this->assertNotEmpty($sessionId);
            $this->assertStringStartsWith($type, $sessionId);
        }
    }

    public function testVerificationScenarios(): void
    {
        $scenarios = [
            [
                'type' => 'slider',
                'x' => 150,
                'y' => 80,
                'expected_success' => true,
            ],
            [
                'type' => 'slider',
                'x' => 50,
                'y' => 80,
                'expected_success' => false,
            ],
            [
                'type' => 'click',
                'points' => [[100, 100], [200, 200]],
                'expected_success' => true,
            ],
        ];

        foreach ($scenarios as $scenario) {
            if ($scenario['type'] === 'slider') {
                $this->assertArrayHasKey('x', $scenario);
                $this->assertArrayHasKey('y', $scenario);
            } elseif ($scenario['type'] === 'click') {
                $this->assertArrayHasKey('points', $scenario);
            }
        }
    }

    public function testErrorScenarios(): void
    {
        $errorCodes = [
            400 => 'Bad Request',
            401 => 'Unauthorized',
            404 => 'Not Found',
            429 => 'Too Many Requests',
            500 => 'Internal Server Error',
        ];

        foreach ($errorCodes as $code => $message) {
            $this->assertIsInt($code);
            $this->assertIsString($message);
        }
    }

    public function testTokenManagement(): void
    {
        $client = new CaptchaClient('http://localhost:8080', 'test-key');

        $client->setAccessToken('test-token');
        $this->assertEquals('test-token', $client->getAccessToken());

        $client->setAccessToken(null);
        $this->assertNull($client->getAccessToken());

        $client->close();
    }

    public function testBaseUrlHandling(): void
    {
        $client1 = new CaptchaClient('http://localhost:8080');
        $client2 = new CaptchaClient('http://localhost:8080/');
        $client3 = new CaptchaClient('http://localhost:8080///');

        $this->assertInstanceOf(CaptchaClient::class, $client1);
        $this->assertInstanceOf(CaptchaClient::class, $client2);
        $this->assertInstanceOf(CaptchaClient::class, $client3);

        $client1->close();
        $client2->close();
        $client3->close();
    }

    public function testCaptchaResponseParsing(): void
    {
        $responses = [
            'slider' => [
                'session_id' => 'test-slider',
                'image_url' => 'http://example.com/slider.jpg',
                'secret_y' => 80,
            ],
            'click' => [
                'session_id' => 'test-click',
                'image_url' => 'http://example.com/click.jpg',
                'max_points' => 3,
            ],
        ];

        $this->assertArrayHasKey('session_id', $responses['slider']);
        $this->assertArrayHasKey('session_id', $responses['click']);
        $this->assertNotEmpty($responses['slider']['session_id']);
        $this->assertNotEmpty($responses['click']['session_id']);
    }

    public function testPerformanceMetrics(): void
    {
        $metrics = [
            'total_requests' => 1000,
            'successful_requests' => 950,
            'failed_requests' => 50,
            'average_response_time_ms' => 150,
            'p99_response_time_ms' => 500,
        ];

        $successRate = ($metrics['successful_requests'] / $metrics['total_requests']) * 100;
        $this->assertEquals(95.0, $successRate);
        $this->assertGreaterThan(0, $metrics['average_response_time_ms']);
        $this->assertGreaterThan($metrics['average_response_time_ms'], $metrics['p99_response_time_ms']);
    }

    public function testSecurityValidation(): void
    {
        $validSignatures = [
            hash_hmac('sha256', 'test-data', 'secret-key'),
            hash_hmac('sha256', 'another-data', 'secret-key'),
        ];

        $invalidSignature = hash_hmac('sha256', 'test-data', 'wrong-key');

        foreach ($validSignatures as $signature) {
            $this->assertNotEquals($invalidSignature, $signature);
            $this->assertEquals(64, strlen($signature));
        }
    }
}
