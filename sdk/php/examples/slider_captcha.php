<?php

require __DIR__ . '/../vendor/autoload.php';

use HJTPX\Captcha\CaptchaClient;
use HJTPX\Captcha\Model\TrajectoryPoint;

// Initialize client
$client = new CaptchaClient(
    'http://localhost:8080', // Replace with your actual API endpoint
    'your-api-key',          // Replace with your API key
    'your-secret-key'        // Replace with your secret key (optional)
);

try {
    // Get slider captcha
    echo "Getting slider captcha...\n";
    $sliderCaptcha = $client->getSliderCaptcha(320, 160, 8);

    echo "Session ID: " . $sliderCaptcha->sessionId . "\n";
    echo "Image URL: " . $sliderCaptcha->imageUrl . "\n";
    echo "Puzzle URL: " . $sliderCaptcha->puzzleUrl . "\n";

    // In a real application, you would display the captcha to the user
    // and collect their response. For this example, we'll simulate a response.

    // Create trajectory points (simulated user mouse movement)
    $trajectory = [
        new TrajectoryPoint(0, $sliderCaptcha->secretY ?? 50, time() * 1000 - 1000),
        new TrajectoryPoint(50, $sliderCaptcha->secretY ?? 50, time() * 1000 - 800),
        new TrajectoryPoint(100, $sliderCaptcha->secretY ?? 50, time() * 1000 - 500),
        new TrajectoryPoint(150, $sliderCaptcha->secretY ?? 50, time() * 1000),
    ];

    // Verify the captcha
    echo "\nVerifying captcha...\n";
    $result = $client->verifySliderCaptcha(
        $sliderCaptcha->sessionId,
        150, // x coordinate where the user dragged the puzzle
        $sliderCaptcha->secretY,
        $trajectory
    );

    if ($result->success) {
        echo "Verification successful!\n";
        echo "Message: " . $result->message . "\n";
        if ($result->remainingAttempts !== null) {
            echo "Remaining attempts: " . $result->remainingAttempts . "\n";
        }
    } else {
        echo "Verification failed!\n";
        echo "Message: " . $result->message . "\n";
        if ($result->failReason) {
            echo "Fail reason: " . $result->failReason . "\n";
        }
    }

} catch (Exception $e) {
    echo "Error: " . $e->getMessage() . "\n";
}

// Close the client
$client->close();
