<?php

namespace HJTPX\Captcha\Tests;

use HJTPX\Captcha\CaptchaClient;
use HJTPX\Captcha\Exception\ApiException;
use HJTPX\Captcha\Model\SliderCaptchaResponse;
use HJTPX\Captcha\Model\VerifyCaptchaResponse;
use HJTPX\Captcha\Pool\ConnectionPoolConfig;
use HJTPX\Captcha\Retry\RetryConfig;
use PHPUnit\Framework\TestCase;

class CaptchaClientTest extends TestCase
{
    protected $client;

    protected function setUp(): void
    {
        // This is a test configuration, in real usage you would use your actual API endpoint
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

    public function testHmacSigner(): void
    {
        $signer = new \HJTPX\Captcha\Signer\HmacSigner('test-secret');
        $data = 'test-data';
        $signature = $signer->sign($data);

        $this->assertIsString($signature);
        $this->assertEquals(64, strlen($signature)); // SHA256 produces 64 hex characters
        $this->assertTrue($signer->verify($data, $signature));
    }

    public function testRetryManager(): void
    {
        $retryConfig = new RetryConfig(2, 10, 2);
        $retryManager = new \HJTPX\Captcha\Retry\RetryManager($retryConfig);

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

    public function testModelClasses(): void
    {
        // Test SliderCaptchaResponse
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

        // Test VerifyCaptchaResponse
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

    public function testTrajectoryPoint(): void
    {
        $point = new \HJTPX\Captcha\Model\TrajectoryPoint(100, 50, 123456789);
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
        $piece = new \HJTPX\Captcha\Model\JigsawPiece($pieceData);
        $this->assertEquals(0, $piece->index);
        $this->assertEquals(100, $piece->currentX);

        $array = $piece->toArray();
        $this->assertEquals($pieceData, $array);
    }

    public function testVerifyCaptchaRequest(): void
    {
        $request = new \HJTPX\Captcha\Model\VerifyCaptchaRequest();
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
}
