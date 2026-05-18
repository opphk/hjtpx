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
    // Step 1: Get a captcha
    echo "Getting slider captcha...\n";
    $captcha = $client->getSliderCaptcha();

    // Step 2: In a real app, display the captcha and get user input
    // For this example, we'll simulate verification (you would replace this with real verification)

    // Step 3: Verify the captcha (this would happen after user interaction)
    echo "Verifying captcha...\n";
    $verifyResult = $client->verifySliderCaptcha(
        $captcha->sessionId,
        150, // Simulated x coordinate
        $captcha->secretY
    );

    if (!$verifyResult->success) {
        echo "Captcha verification failed: " . $verifyResult->message . "\n";
        exit(1);
    }

    // Step 4: Login with the captcha token (if your system uses tokens)
    // Note: The exact login flow depends on your API implementation
    echo "Captcha verified. Logging in...\n";

    // Some systems might return a token from verification that you can use for login
    // For this example, we'll assume a direct login flow
    $loginResult = $client->login('username', 'password');

    echo "Login successful!\n";
    echo "Access token: " . $loginResult->accessToken . "\n";
    if ($loginResult->user) {
        echo "User: " . json_encode($loginResult->user) . "\n";
    }

    // Now you can use the access token for authenticated requests
    $client->setAccessToken($loginResult->accessToken);

    // When done, you can logout
    echo "\nLogging out...\n";
    $client->logout();

} catch (Exception $e) {
    echo "Error: " . $e->getMessage() . "\n";
}

$client->close();
