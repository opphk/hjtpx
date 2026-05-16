"""
hjtpx Python SDK - Advanced Examples with Error Handling
"""

import time
from hjtpx import (
    CaptchaClient,
    Config,
    SDKError,
    NetworkError,
    TimeoutError,
    RateLimitedError,
    InvalidParamsError,
    is_sdk_error,
    get_error_code,
)
from hjtpx import ImageCaptchaRequest, CaptchaType, ClickData, ClickCaptchaRequest


def example_with_error_handling():
    """Example demonstrating comprehensive error handling"""
    print("\n=== Error Handling Example ===")

    client = CaptchaClient()
    client.set_debug_mode(True)

    try:
        captcha = client.generate_image_captcha()
        print(f"✓ Challenge ID: {captcha.challenge_id}")

        result = client.verify_image_captcha(captcha.challenge_id, "wrong_answer")
        print(f"✓ Verification: {result.success}")

    except RateLimitedError as e:
        print(f"⚠ Rate limited, retry after: {e.retry_after}s")
    except TimeoutError as e:
        print(f"⚠ Request timed out: {e}")
    except NetworkError as e:
        print(f"⚠ Network error: {e}")
    except InvalidParamsError as e:
        print(f"⚠ Invalid parameters: {e}")
    except SDKError as e:
        code = get_error_code(e)
        print(f"⚠ SDK Error {code}: {e.message}")
    except Exception as e:
        print(f"✗ Unexpected error: {e}")


def example_with_custom_config():
    """Example with custom configuration"""
    print("\n=== Custom Configuration Example ===")

    config = Config(
        base_url="http://localhost:8080",
        app_id="my-app-id",
        app_secret="my-app-secret",
        timeout=60.0,
        max_retries=5,
        retry_delay=0.5,
        debug_mode=True,
    )

    client = CaptchaClient(config)
    print(f"✓ Client created with config:")
    print(f"  Base URL: {client.config.base_url}")
    print(f"  Timeout: {client.config.timeout}s")
    print(f"  Max Retries: {client.config.max_retries}")

    captcha = client.generate_image_captcha()
    print(f"✓ Challenge ID: {captcha.challenge_id}")


def example_with_retry():
    """Example demonstrating retry mechanism"""
    print("\n=== Retry Mechanism Example ===")

    config = Config(
        max_retries=3,
        retry_delay=0.1,
    )

    client = CaptchaClient(config)

    for i in range(5):
        try:
            captcha = client.generate_image_captcha()
            print(f"✓ Attempt {i+1}: Success - {captcha.challenge_id}")
            break
        except SDKError as e:
            print(f"✗ Attempt {i+1}: Failed - {e.message}")
            if i < 4:
                print(f"  Retrying...")
            else:
                print(f"  Max retries reached")


def example_statistics():
    """Example demonstrating statistics tracking"""
    print("\n=== Statistics Example ===")

    client = CaptchaClient()

    for i in range(3):
        try:
            client.generate_image_captcha()
            client.verify_image_captcha(f"test-{i}", "1234")
        except SDKError:
            pass

    stats = client.get_stats()
    print(f"📊 Current Statistics:")
    print(f"  Total Requests: {stats.total_requests}")
    print(f"  Successful: {stats.successful_requests}")
    print(f"  Failed: {stats.failed_requests}")
    print(f"  Retried: {stats.retried_requests}")
    print(f"  Success Rate: {stats.success_rate:.2f}%")
    if stats.last_error:
        print(f"  Last Error: {stats.last_error}")


def example_concurrent_requests():
    """Example demonstrating concurrent requests"""
    import concurrent.futures

    print("\n=== Concurrent Requests Example ===")

    client = CaptchaClient()

    def generate_captcha(index):
        try:
            captcha = client.generate_image_captcha()
            return f"Task {index}: Success - {captcha.challenge_id}"
        except SDKError as e:
            return f"Task {index}: Failed - {e.message}"

    with concurrent.futures.ThreadPoolExecutor(max_workers=3) as executor:
        futures = [executor.submit(generate_captcha, i) for i in range(5)]
        for future in concurrent.futures.as_completed(futures):
            print(f"  {future.result()}")

    stats = client.get_stats()
    print(f"\n📊 Final Statistics:")
    print(f"  Total Requests: {stats.total_requests}")
    print(f"  Success Rate: {stats.success_rate:.2f}%")


def main():
    print("=" * 60)
    print("hjtpx Python SDK - Advanced Examples")
    print("=" * 60)

    try:
        example_with_error_handling()
        example_with_custom_config()
        example_with_retry()
        example_statistics()
        example_concurrent_requests()

        print("\n" + "=" * 60)
        print("All advanced examples completed!")
        print("=" * 60)

    except Exception as e:
        print(f"\n✗ Error: {e}")
        import traceback
        traceback.print_exc()


if __name__ == "__main__":
    main()
