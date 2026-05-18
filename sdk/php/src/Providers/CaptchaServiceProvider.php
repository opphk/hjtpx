<?php

declare(strict_types=1);

namespace Hjtpx\Captcha\Providers;

use Hjtpx\Captcha\Client\CaptchaClient;
use Hjtpx\Captcha\Contracts\CaptchaClientInterface;
use Illuminate\Support\ServiceProvider;

class CaptchaServiceProvider extends ServiceProvider
{
    public function register(): void
    {
        $this->mergeConfigFrom(
            __DIR__ . '/../../config/captcha.php',
            'captcha'
        );

        $this->app->singleton(CaptchaClientInterface::class, function ($app) {
            $config = $app['config']['captcha'] ?? [];

            $client = new CaptchaClient(
                $config['base_url'] ?? 'http://localhost:8080',
                $config['api_key'] ?? null,
                (int)($config['timeout'] ?? 30),
                (int)($config['max_retries'] ?? 3),
                (float)($config['retry_backoff_factor'] ?? 0.5)
            );

            return $client;
        });

        $this->app->alias(CaptchaClientInterface::class, 'captcha.client');
    }

    public function boot(): void
    {
        if ($this->app->runningInConsole()) {
            $this->publishes([
                __DIR__ . '/../../config/captcha.php' => config_path('captcha.php'),
            ], 'captcha-config');

            $this->commands([]);
        }
    }

    public function provides(): array
    {
        return [
            CaptchaClientInterface::class,
            'captcha.client',
        ];
    }
}
