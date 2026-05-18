<?php

return [
    'base_url' => env('CAPTCHA_BASE_URL', 'http://localhost:8080'),

    'api_key' => env('CAPTCHA_API_KEY', null),

    'timeout' => env('CAPTCHA_TIMEOUT', 30),

    'max_retries' => env('CAPTCHA_MAX_RETRIES', 3),

    'retry_backoff_factor' => env('CAPTCHA_RETRY_BACKOFF_FACTOR', 0.5),

    'default_width' => env('CAPTCHA_DEFAULT_WIDTH', 320),

    'default_height' => env('CAPTCHA_DEFAULT_HEIGHT', 160),

    'default_tolerance' => env('CAPTCHA_DEFAULT_TOLERANCE', 8),
];
