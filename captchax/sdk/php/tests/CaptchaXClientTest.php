<?php

declare(strict_types=1);

namespace CaptchaX\SDK\Tests;

use CaptchaX\SDK\CaptchaConfig;
use CaptchaX\SDK\CaptchaXClient;
use CaptchaX\SDK\CaptchaXException;
use CaptchaX\SDK\ApiVersion;
use CaptchaX\SDK\CharPosition;
use PHPUnit\Framework\TestCase;

class CaptchaXClientTest extends TestCase
{
    public function testCreateClientWithConfig(): void
    {
        $config = new CaptchaConfig('https://captchax.example.com', 'test-app');
        $client = new CaptchaXClient($config);
        $this->assertSame(ApiVersion::V1, $client->getApiVersion());
    }

    public function testCreateClientWithStaticFactory(): void
    {
        $client = CaptchaXClient::create('https://captchax.example.com');
        $this->assertInstanceOf(CaptchaXClient::class, $client);
    }

    public function testSetAppId(): void
    {
        $client = CaptchaXClient::create('https://captchax.example.com');
        $client->setAppId('new-app-id');
        $this->assertTrue(true);
    }

    public function testSetApiVersion(): void
    {
        $client = CaptchaXClient::create('https://captchax.example.com');
        $this->assertSame(ApiVersion::V1, $client->getApiVersion());

        $client->setApiVersion(ApiVersion::V2);
        $this->assertSame(ApiVersion::V2, $client->getApiVersion());
    }

    public function testCreateClientInfo(): void
    {
        $client = CaptchaXClient::create('https://captchax.example.com');
        $info = $client->createClientInfo(['custom' => 'value']);

        $this->assertNotEmpty($info);
        $data = json_decode($info, true);
        $this->assertArrayHasKey('platform', $data);
        $this->assertArrayHasKey('timestamp', $data);
        $this->assertSame('value', $data['custom']);
    }
}

class CaptchaConfigTest extends TestCase
{
    public function testDefaultValues(): void
    {
        $config = new CaptchaConfig('https://captchax.example.com');
        $this->assertSame('https://captchax.example.com', $config->getBaseUrl());
        $this->assertNull($config->getAppId());
        $this->assertSame(10000, $config->getTimeout());
        $this->assertSame(3, $config->getRetryTimes());
        $this->assertSame(ApiVersion::V1, $config->getApiVersion());
    }

    public function testWithMethods(): void
    {
        $config = (new CaptchaConfig('https://captchax.example.com'))
            ->withAppId('test-app')
            ->withTimeout(5000)
            ->withRetryTimes(2)
            ->withApiVersion(ApiVersion::V2);

        $this->assertSame('test-app', $config->getAppId());
        $this->assertSame(5000, $config->getTimeout());
        $this->assertSame(2, $config->getRetryTimes());
        $this->assertSame(ApiVersion::V2, $config->getApiVersion());
    }
}

class CaptchaXExceptionTest extends TestCase
{
    public function testExceptionWithDefaults(): void
    {
        $exception = new CaptchaXException('Test error');
        $this->assertSame('Test error', $exception->getMessage());
        $this->assertSame(500, $exception->getErrorCode());
        $this->assertSame(500, $exception->getStatusCode());
    }

    public function testExceptionWithCustomValues(): void
    {
        $exception = new CaptchaXException('Test error', 400, 400, ['field' => 'value']);
        $this->assertSame(400, $exception->getErrorCode());
        $this->assertSame(400, $exception->getStatusCode());
        $this->assertSame(['field' => 'value'], $exception->getDetails());
    }
}

class CharPositionTest extends TestCase
{
    public function testCharPositionCreation(): void
    {
        $pos = new CharPosition('A', 100, 50);
        $this->assertSame('A', $pos->getChar());
        $this->assertSame(100, $pos->getX());
        $this->assertSame(50, $pos->getY());
    }

    public function testCharPositionToArray(): void
    {
        $pos = new CharPosition('B', 200, 75);
        $arr = $pos->toArray();

        $this->assertSame([
            'char' => 'B',
            'x' => 200,
            'y' => 75,
        ], $arr);
    }
}
