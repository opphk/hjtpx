<?php

declare(strict_types=1);

namespace Hjtpx\Captcha\Examples;

use Hjtpx\Captcha\Client\CaptchaClient;
use Hjtpx\Captcha\Exceptions\CaptchaApiException;
use Hjtpx\Captcha\Exceptions\CaptchaException;
use Hjtpx\Captcha\Exceptions\CaptchaNetworkException;
use Hjtpx\Captcha\Exceptions\CaptchaValidationException;

require_once __DIR__ . '/../vendor/autoload.php';

class UsageExample
{
    private CaptchaClient $client;

    public function __construct()
    {
        $this->client = new CaptchaClient(
            baseUrl: 'http://localhost:8080',
            apiKey: 'your-api-key',
            timeout: 30,
            maxRetries: 3
        );
    }

    public function sliderCaptchaExample(): void
    {
        echo "=== Slider Captcha Example ===\n";

        try {
            $captcha = $this->client->getSliderCaptcha(320, 160, 8);
            echo "Session ID: " . ($captcha['session_id'] ?? 'N/A') . "\n";
            echo "Image URL: " . ($captcha['image_url'] ?? 'N/A') . "\n";
            echo "Secret Y: " . ($captcha['secret_y'] ?? 'N/A') . "\n";

            $trajectory = [
                ['x' => 0, 'y' => $captcha['secret_y'] ?? 50, 't' => time() * 1000 - 1000],
                ['x' => 50, 'y' => ($captcha['secret_y'] ?? 50) + 5, 't' => time() * 1000 - 800],
                ['x' => 100, 'y' => ($captcha['secret_y'] ?? 50) - 3, 't' => time() * 1000 - 500],
                ['x' => 150, 'y' => ($captcha['secret_y'] ?? 50) + 2, 't' => time() * 1000 - 200],
                ['x' => 185, 'y' => $captcha['secret_y'] ?? 50, 't' => time() * 1000],
            ];

            $result = $this->client->verifySliderCaptcha(
                $captcha['session_id'],
                185,
                $captcha['secret_y'] ?? null,
                $trajectory
            );

            echo "Verification Success: " . ($result['success'] ? 'Yes' : 'No') . "\n";
            echo "Message: " . ($result['message'] ?? 'N/A') . "\n";
        } catch (CaptchaApiException $e) {
            echo "API Error: " . $e->getMessage() . " (Code: " . $e->getErrorCode() . ")\n";
        } catch (CaptchaNetworkException $e) {
            echo "Network Error: " . $e->getMessage() . "\n";
        } catch (CaptchaValidationException $e) {
            echo "Validation Error: " . $e->getMessage() . "\n";
        } catch (CaptchaException $e) {
            echo "Error: " . $e->getMessage() . "\n";
        }
    }

    public function clickCaptchaExample(): void
    {
        echo "\n=== Click Captcha Example ===\n";

        try {
            $captcha = $this->client->getClickCaptcha('number', 3, true);
            echo "Session ID: " . ($captcha['session_id'] ?? 'N/A') . "\n";
            echo "Image URL: " . ($captcha['image_url'] ?? 'N/A') . "\n";
            echo "Hint: " . ($captcha['hint'] ?? 'N/A') . "\n";
            echo "Hint Order: " . implode(', ', $captcha['hint_order'] ?? []) . "\n";

            $points = [[100, 100], [200, 150], [150, 200]];
            $clickSequence = [1, 2, 3];

            $result = $this->client->verifyClickCaptcha(
                $captcha['session_id'],
                $points,
                $clickSequence
            );

            echo "Verification Success: " . ($result['success'] ? 'Yes' : 'No') . "\n";
        } catch (\Throwable $e) {
            echo "Error: " . $e->getMessage() . "\n";
        }
    }

    public function imageCaptchaExample(): void
    {
        echo "\n=== Image Captcha Example ===\n";

        try {
            $captcha = $this->client->getImageCaptcha('mixed', 4);
            echo "Challenge ID: " . ($captcha['challenge_id'] ?? 'N/A') . "\n";
            echo "Image (base64): " . substr($captcha['image'] ?? '', 0, 50) . "...\n";

            $result = $this->client->verifyImageCaptcha(
                $captcha['challenge_id'],
                'ABCD'
            );

            echo "Verification Success: " . ($result['success'] ? 'Yes' : 'No') . "\n";
        } catch (\Throwable $e) {
            echo "Error: " . $e->getMessage() . "\n";
        }
    }

    public function batchVerifyExample(): void
    {
        echo "\n=== Batch Verify Example ===\n";

        try {
            $requests = [
                ['session_id' => 'session-1', 'x' => 100, 'y' => 50],
                ['session_id' => 'session-2', 'x' => 150, 'y' => 60],
                ['session_id' => 'session-3', 'x' => 200, 'y' => 70],
            ];

            $result = $this->client->batchVerify($requests);

            echo "Success Count: " . ($result['success_count'] ?? 0) . "\n";
            echo "Failed Count: " . ($result['failed_count'] ?? 0) . "\n";
            echo "Total Time: " . ($result['total_time_ms'] ?? 0) . "ms\n";

            foreach ($result['results'] ?? [] as $r) {
                echo "- Session " . ($r['session_id'] ?? 'N/A') . ": " .
                     ($r['success'] ? 'Success' : 'Failed') . "\n";
            }
        } catch (\Throwable $e) {
            echo "Error: " . $e->getMessage() . "\n";
        }
    }

    public function asyncVerifyExample(): void
    {
        echo "\n=== Async Verify Example ===\n";

        try {
            $asyncResult = $this->client->asyncVerify([
                'session_id' => 'session-async-1',
                'x' => 150,
                'y' => 50,
                'callback_url' => 'https://example.com/callback',
            ]);

            $taskId = $asyncResult['task_id'] ?? 'N/A';
            echo "Task ID: " . $taskId . "\n";
            echo "Status: " . ($asyncResult['status'] ?? 'N/A') . "\n";

            $pollCount = 0;
            while ($pollCount < 10) {
                $result = $this->client->getAsyncResult($taskId);
                $status = $result['status'] ?? '';

                echo "Poll $pollCount: Status = $status\n";

                if ($status === 'completed' || $status === 'failed') {
                    if ($status === 'completed' && isset($result['result'])) {
                        echo "Final Result: " . json_encode($result['result']) . "\n";
                    }
                    break;
                }

                usleep(500000);
                $pollCount++;
            }
        } catch (\Throwable $e) {
            echo "Error: " . $e->getMessage() . "\n";
        }
    }

    public function authenticationExample(): void
    {
        echo "\n=== Authentication Example ===\n";

        try {
            $result = $this->client->login('username', 'password');
            echo "Access Token: " . substr($result['access_token'] ?? '', 0, 20) . "...\n";
            echo "Expires In: " . ($result['expires_in'] ?? 0) . "s\n";
            echo "User: " . ($result['user']['username'] ?? 'N/A') . "\n";

            $this->client->logout();
            echo "Logged out successfully\n";
        } catch (\Throwable $e) {
            echo "Error: " . $e->getMessage() . "\n";
        }
    }

    public function detectionScriptExample(): void
    {
        echo "\n=== Detection Script Example ===\n";

        try {
            $script = $this->client->getDetectionScript('handleDetectionResult');
            echo "Script Length: " . strlen($script) . " bytes\n";
            echo "Script Preview: " . substr($script, 0, 100) . "...\n";
        } catch (\Throwable $e) {
            echo "Error: " . $e->getMessage() . "\n";
        }
    }
}

if (php_sapi_name() === 'cli' && basename(__FILE__) === basename($_SERVER['SCRIPT_NAME'] ?? '')) {
    $example = new UsageExample();

    echo "========================================\n";
    echo "   Hjtpx Captcha PHP SDK v15.0 Examples\n";
    echo "========================================\n\n";

    $example->sliderCaptchaExample();
    $example->clickCaptchaExample();
    $example->imageCaptchaExample();
    $example->batchVerifyExample();
    $example->asyncVerifyExample();
    $example->authenticationExample();
    $example->detectionScriptExample();

    echo "\n========================================\n";
    echo "             Examples Complete\n";
    echo "========================================\n";
}
