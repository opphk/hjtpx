<?php

require __DIR__ . '/../vendor/autoload.php';

use HJTPX\Captcha\CaptchaClient;

// Initialize client
$client = new CaptchaClient(
    'http://localhost:8080',
    'your-api-key',
    'your-secret-key'
);

try {
    // Get click captcha
    echo "Getting click captcha...\n";
    $clickCaptcha = $client->getClickCaptcha('icon', true, 3);

    echo "Session ID: " . $clickCaptcha->sessionId . "\n";
    echo "Image URL: " . $clickCaptcha->imageUrl . "\n";
    echo "Hint: " . $clickCaptcha->hint . "\n";
    echo "Max points: " . $clickCaptcha->maxPoints . "\n";

    // In a real application, you would display the captcha to the user
    // and collect their click points. For this example, we'll simulate clicks.

    // Simulated click points (x, y coordinates)
    $clickPoints = [
        [50, 50],
        [150, 50],
        [250, 50],
    ];

    // Verify the captcha
    echo "\nVerifying captcha...\n";
    $result = $client->verifyClickCaptcha(
        $clickCaptcha->sessionId,
        $clickPoints
    );

    if ($result->success) {
        echo "Verification successful!\n";
        echo "Message: " . $result->message . "\n";
    } else {
        echo "Verification failed!\n";
        echo "Message: " . $result->message . "\n";
    }

} catch (Exception $e) {
    echo "Error: " . $e->getMessage() . "\n";
}

$client->close();
