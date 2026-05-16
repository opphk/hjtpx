"""
hjtpx Python SDK - Examples
"""

import time
from hjtpx import CaptchaClient, Config
from hjtpx import (
    ImageCaptchaRequest,
    SliderCaptchaRequest,
    ClickCaptchaRequest,
    ClickData,
    CaptchaType,
)


def example_image_captcha():
    """Image Captcha Example"""
    print("\n=== Image Captcha Example ===")

    client = CaptchaClient()
    client.set_debug_mode(True)

    request = ImageCaptchaRequest(
        captcha_type=CaptchaType.MIXED,
        count=4,
        noise_mode=2,
        line_mode=1,
    )

    captcha = client.generate_image_captcha(request)
    print(f"✓ Challenge ID: {captcha.challenge_id}")

    result = client.verify_image_captcha(captcha.challenge_id, "1234")
    print(f"✓ Verification success: {result.success}")

    stats = client.get_stats()
    print(f"\n📊 Statistics:")
    print(f"  Total Requests: {stats.total_requests}")
    print(f"  Success Rate: {stats.success_rate:.2f}%")


def example_slider_captcha():
    """Slider Captcha Example"""
    print("\n=== Slider Captcha Example ===")

    client = CaptchaClient()

    request = SliderCaptchaRequest(width=360, height=220)
    slider = client.generate_slider_captcha(request)
    print(f"✓ Challenge ID: {slider.challenge_id}")
    print(f"  Slider Size: {slider.slider_width}x{slider.slider_height}")

    result = client.verify_slider_captcha(slider.challenge_id, "120")
    print(f"✓ Verification success: {result.success}")
    print(f"  Score: {result.score}")
    print(f"  Risk Level: {result.risk_level}")

    stats = client.get_stats()
    print(f"\n📊 Statistics:")
    print(f"  Total Requests: {stats.total_requests}")
    print(f"  Success Rate: {stats.success_rate:.2f}%")


def example_click_captcha():
    """Click Captcha Example"""
    print("\n=== Click Captcha Example ===")

    client = CaptchaClient()

    request = ClickCaptchaRequest(width=360, height=220, icon_count=4)
    click = client.generate_click_captcha(request)
    print(f"✓ Challenge ID: {click.challenge_id}")
    print(f"  Target Index: {click.target_index}")
    print(f"  Icon Positions: {click.icon_positions}")

    clicks = [
        ClickData(
            x=click.target_position[0],
            y=click.target_position[1],
            duration=500,
        )
    ]

    result = client.verify_click_captcha(click.challenge_id, clicks)
    print(f"✓ Verification success: {result.success}")
    print(f"  Score: {result.score}")

    stats = client.get_stats()
    print(f"\n📊 Statistics:")
    print(f"  Total Requests: {stats.total_requests}")
    print(f"  Success Rate: {stats.success_rate:.2f}%")


def main():
    print("=" * 50)
    print("hjtpx Python SDK Examples")
    print("=" * 50)

    try:
        example_image_captcha()
        example_slider_captcha()
        example_click_captcha()

        print("\n" + "=" * 50)
        print("All examples completed successfully!")
        print("=" * 50)

    except Exception as e:
        print(f"\n✗ Error: {e}")
        raise


if __name__ == "__main__":
    main()
