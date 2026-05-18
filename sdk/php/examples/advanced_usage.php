<?php

require __DIR__ . '/../vendor/autoload.php';

use HJTPX\Captcha\CaptchaClient;
use HJTPX\Captcha\Pool\ConnectionPoolConfig;
use HJTPX\Captcha\Retry\RetryConfig;

$poolConfig = new ConnectionPoolConfig(
    10,
    10,
    30,
    null,
    true,
    5,
    true
);

$retryConfig = new RetryConfig(
    3,
    100,
    2.0,
    [429, 500, 502, 503, 504],
    30000,
    true
);

$client = new CaptchaClient(
    'http://localhost:8080',
    'your-api-key',
    'your-secret-key',
    $poolConfig,
    $retryConfig
);

$client->setDebugMode(true);
$client->setLogger(function ($level, $message) {
    echo "[" . strtoupper($level) . "] $message\n";
});

try {
    echo "=== Slider Captcha ===\n";
    $slider = $client->getSliderCaptcha(320, 160, 8);
    echo "Session ID: " . $slider->sessionId . "\n";
    echo "Image URL: " . $slider->imageUrl . "\n";
    echo "Secret Y: " . $slider->secretY . "\n";

    $sliderResult = $client->verifySliderCaptcha(
        $slider->sessionId,
        150,
        $slider->secretY
    );
    echo "Verification: " . ($sliderResult->success ? "SUCCESS" : "FAILED") . "\n";

    echo "\n=== Click Captcha ===\n";
    $click = $client->getClickCaptcha('icon', true, 3);
    echo "Session ID: " . $click->sessionId . "\n";
    echo "Image URL: " . $click->imageUrl . "\n";
    echo "Hint: " . $click->hint . "\n";

    $clickResult = $client->verifyClickCaptcha(
        $click->sessionId,
        [[100, 100], [200, 100], [300, 100]]
    );
    echo "Verification: " . ($clickResult->success ? "SUCCESS" : "FAILED") . "\n";

    echo "\n=== Rotation Captcha ===\n";
    $rotation = $client->getRotationCaptcha();
    echo "Challenge ID: " . $rotation->challengeId . "\n";
    echo "Image URL: " . $rotation->imageUrl . "\n";

    $rotationResult = $client->verifyRotationCaptcha(
        $rotation->challengeId,
        90
    );
    echo "Verification: " . ($rotationResult->success ? "SUCCESS" : "FAILED") . "\n";

    echo "\n=== Gesture Captcha ===\n";
    $gesture = $client->getGestureCaptcha();
    echo "Session ID: " . $gesture->sessionId . "\n";
    echo "Pattern: " . implode(', ', $gesture->pattern) . "\n";

    $gestureResult = $client->verifyGestureCaptcha(
        $gesture->sessionId,
        [0, 1, 2, 5, 8]
    );
    echo "Verification: " . ($gestureResult->success ? "SUCCESS" : "FAILED") . "\n";

    echo "\n=== Voice Captcha ===\n";
    $voice = $client->getVoiceCaptcha('zh-CN');
    echo "Session ID: " . $voice->sessionId . "\n";
    echo "Audio URL: " . $voice->audioUrl . "\n";

    $voiceResult = $client->verifyVoiceCaptcha(
        $voice->sessionId,
        '123456'
    );
    echo "Verification: " . ($voiceResult->success ? "SUCCESS" : "FAILED") . "\n";

    echo "\n=== Lianliankan Captcha ===\n";
    $lianliankan = $client->getLianliankanCaptcha(4, 4, 60);
    echo "Session ID: " . $lianliankan->sessionId . "\n";
    echo "Grid: " . $lianliankan->gridRows . "x" . $lianliankan->gridCols . "\n";
    echo "Pairs: " . $lianliankan->getTotalPairs() . "\n";

    $connections = [
        [[0, 0], [0, 1]],
        [[1, 0], [1, 1]]
    ];
    $lianliankanResult = $client->verifyLianliankanCaptcha(
        $lianliankan->sessionId,
        $connections,
        15
    );
    echo "Verification: " . ($lianliankanResult->success ? "SUCCESS" : "FAILED") . "\n";

    echo "\n=== 3D Captcha ===\n";
    $threeD = $client->getThreeDCaptcha();
    echo "Session ID: " . $threeD->sessionId . "\n";
    echo "Scene URL: " . $threeD->sceneUrl . "\n";

    $threeDResult = $client->verifyThreeDCaptcha(
        $threeD->sessionId,
        [100, 50, 0]
    );
    echo "Verification: " . ($threeDResult->success ? "SUCCESS" : "FAILED") . "\n";

    echo "\n=== Connection Pool Stats ===\n";
    $stats = $client->poolManager->getStats();
    print_r($stats);
    echo "Success Rate: " . $client->poolManager->getSuccessRate() . "%\n";

    echo "\n=== Health Check ===\n";
    echo "Is Connected: " . ($client->isConnected() ? "YES" : "NO") . "\n";

} catch (Exception $e) {
    echo "Error: " . $e->getMessage() . "\n";
    echo "Class: " . get_class($e) . "\n";

    if ($e instanceof \HJTPX\Captcha\Exception\ApiException) {
        echo "HTTP Status: " . $e->getHttpStatusCode() . "\n";
        echo "Error Code: " . $e->getErrorCode() . "\n";
    }
}

$client->close();
