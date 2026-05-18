#!/usr/bin/env python3
"""
Python SDK 高级完整示例

展示所有功能的使用方法，包括：
- 同步和异步客户端
- 批量操作
- 错误处理
- 性能优化
"""

import asyncio
import sys
import os
from concurrent.futures import ThreadPoolExecutor, as_completed
import time
from dataclasses import dataclass
from typing import List, Dict, Any, Optional

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from captcha import CaptchaClient, CaptchaType, ClickMode
from captcha import TrajectoryPoint, JigsawPiece
from captcha import (
    CaptchaError,
    CaptchaAPIError,
    CaptchaNetworkError,
    CaptchaTimeoutError,
)
from async_captcha import AsyncCaptchaClient, AsyncCaptchaType


@dataclass
class CaptchaTestResult:
    """验证码测试结果"""
    captcha_type: str
    success: bool
    message: str
    duration: float
    error: Optional[str] = None


def generate_realistic_trajectory(secret_y: int, target_x: int, duration_ms: int = 800) -> List[TrajectoryPoint]:
    """
    生成真实的滑动轨迹

    Args:
        secret_y: Y轴位置
        target_x: 目标X坐标
        duration_ms: 滑动总时长（毫秒）

    Returns:
        轨迹点列表
    """
    num_points = 20
    points = []

    for i in range(num_points):
        progress = i / (num_points - 1)
        eased_progress = 0.5 - 0.5 * (1 - progress) ** 3

        x = int(target_x * eased_progress)
        y_offset = int(5 * (1 if i % 2 == 0 else -1) * (1 - progress) * (1 - abs(progress - 0.5) * 2))
        y = secret_y + y_offset

        t = int(duration_ms * progress)

        points.append(TrajectoryPoint(x=x, y=y, t=t))

    points[-1] = TrajectoryPoint(x=target_x, y=secret_y, t=duration_ms)

    return points


def basic_sync_example():
    """基础同步示例"""
    print("\n" + "="*60)
    print("基础同步示例")
    print("="*60)

    with CaptchaClient(
        base_url="http://localhost:8080",
        api_key="demo-key",
        timeout=30,
        max_retries=3,
    ) as client:
        start_time = time.time()

        slider = client.get_slider_captcha(width=320, height=160)
        print(f"✓ 滑块验证码获取成功")
        print(f"  Session ID: {slider.session_id}")
        print(f"  Secret Y: {slider.secret_y}")

        secret_x = 150
        trajectory = generate_realistic_trajectory(slider.secret_y or 80, secret_x)

        result = client.verify_slider_captcha(
            session_id=slider.session_id,
            x=secret_x,
            y=slider.secret_y,
            trajectory=trajectory,
        )

        duration = time.time() - start_time

        print(f"\n✓ 验证结果:")
        print(f"  成功: {result.success}")
        print(f"  消息: {result.message}")
        print(f"  耗时: {duration:.2f}秒")

        if result.trajectory_result:
            print(f"  轨迹得分: {result.trajectory_result.get('score', 'N/A')}")


def click_captcha_example():
    """点击验证码完整示例"""
    print("\n" + "="*60)
    print("点击验证码完整示例")
    print("="*60)

    with CaptchaClient(base_url="http://localhost:8080") as client:
        click = client.get_click_captcha(
            mode=ClickMode.NUMBER,
            max_points=4,
            allow_shuffle=True,
        )

        print(f"✓ 点击验证码获取成功")
        print(f"  Session ID: {click.session_id}")
        print(f"  提示: {click.hint}")
        print(f"  图标数量: {click.max_points}")
        print(f"  模式: {click.mode}")

        if click.points:
            print(f"  目标坐标: {click.points}")

        mock_clicks = [
            [120, 150],
            [200, 150],
            [160, 220],
            [280, 220],
        ]

        result = client.verify_click_captcha(
            session_id=click.session_id,
            points=mock_clicks,
            click_sequence=[0, 1, 2, 3],
        )

        print(f"\n✓ 验证结果:")
        print(f"  成功: {result.success}")
        print(f"  消息: {result.message}")


def image_captcha_example():
    """图形验证码完整示例"""
    print("\n" + "="*60)
    print("图形验证码完整示例")
    print("="*60)

    with CaptchaClient(base_url="http://localhost:8080") as client:
        test_cases = [
            ("数字", {"type_": "number", "count": 4}),
            ("字母", {"type_": "letter", "count": 5}),
            ("混合", {"type_": "mixed", "count": 6}),
            ("中文", {"type_": "chinese", "count": 3}),
        ]

        for name, params in test_cases:
            image = client.get_image_captcha(**params)
            print(f"\n{name}验证码:")
            print(f"  Challenge ID: {image.challenge_id}")
            print(f"  图片长度: {len(image.image)} 字符")

            result = client.verify_image_captcha(
                challenge_id=image.challenge_id,
                answer="WRONG",
            )
            print(f"  错误答案验证: {result.success} (预期: False)")


def batch_processing_example():
    """批量处理示例"""
    print("\n" + "="*60)
    print("批量处理示例")
    print("="*60)

    num_requests = 10
    results = []

    def process_single_captcha(index: int) -> CaptchaTestResult:
        """处理单个验证码"""
        start_time = time.time()
        try:
            with CaptchaClient(base_url="http://localhost:8080") as client:
                slider = client.get_slider_captcha()
                secret_x = 150
                trajectory = generate_realistic_trajectory(slider.secret_y or 80, secret_x)

                result = client.verify_slider_captcha(
                    session_id=slider.session_id,
                    x=secret_x,
                    y=slider.secret_y,
                    trajectory=trajectory,
                )

                duration = time.time() - start_time
                return CaptchaTestResult(
                    captcha_type="slider",
                    success=result.success,
                    message=result.message,
                    duration=duration,
                )
        except Exception as e:
            duration = time.time() - start_time
            return CaptchaTestResult(
                captcha_type="slider",
                success=False,
                message="",
                duration=duration,
                error=str(e),
            )

    print(f"启动 {num_requests} 个并发请求...")

    start_time = time.time()
    with ThreadPoolExecutor(max_workers=5) as executor:
        futures = [executor.submit(process_single_captcha, i) for i in range(num_requests)]

        for future in as_completed(futures):
            result = future.result()
            results.append(result)
            status = "✓" if result.success else "✗"
            print(f"  {status} Request completed in {result.duration:.2f}s")

    total_duration = time.time() - start_time

    success_count = sum(1 for r in results if r.success)
    avg_duration = sum(r.duration for r in results) / len(results)

    print(f"\n批量处理统计:")
    print(f"  总请求数: {num_requests}")
    print(f"  成功数: {success_count}")
    print(f"  成功率: {success_count/num_requests*100:.1f}%")
    print(f"  平均耗时: {avg_duration:.2f}秒")
    print(f"  总耗时: {total_duration:.2f}秒")


async def basic_async_example():
    """基础异步示例"""
    print("\n" + "="*60)
    print("基础异步示例")
    print("="*60)

    async with AsyncCaptchaClient(
        base_url="http://localhost:8080",
        timeout=30,
        max_connections=100,
    ) as client:
        start_time = time.time()

        slider = await client.get_slider_captcha(width=320, height=160)
        print(f"✓ 异步获取滑块验证码成功")
        print(f"  Session ID: {slider.session_id}")

        secret_x = 150
        trajectory = generate_realistic_trajectory(slider.secret_y or 80, secret_x)

        result = await client.verify_slider_captcha(
            session_id=slider.session_id,
            x=secret_x,
            y=slider.secret_y,
            trajectory=trajectory,
        )

        duration = time.time() - start_time

        print(f"\n✓ 验证结果:")
        print(f"  成功: {result.success}")
        print(f"  消息: {result.message}")
        print(f"  耗时: {duration:.2f}秒")


async def concurrent_async_example():
    """并发异步请求示例"""
    print("\n" + "="*60)
    print("并发异步请求示例")
    print("="*60)

    num_requests = 20

    async with AsyncCaptchaClient(base_url="http://localhost:8080") as client:
        print(f"启动 {num_requests} 个并发请求...")

        start_time = time.time()

        tasks = [
            client.get_slider_captcha(width=320, height=160)
            for _ in range(num_requests)
        ]

        results = await asyncio.gather(*tasks, return_exceptions=True)

        duration = time.time() - start_time

        success_count = sum(
            1 for r in results
            if not isinstance(r, Exception)
        )

        print(f"\n并发请求统计:")
        print(f"  总请求数: {num_requests}")
        print(f"  成功数: {success_count}")
        print(f"  成功率: {success_count/num_requests*100:.1f}%")
        print(f"  总耗时: {duration:.2f}秒")
        print(f"  平均每个请求: {duration/num_requests*1000:.1f}ms")


async def mixed_captcha_async_example():
    """混合验证码异步示例"""
    print("\n" + "="*60)
    print("混合验证码异步示例")
    print("="*60)

    async with AsyncCaptchaClient(base_url="http://localhost:8080") as client:
        tasks = [
            ("滑块", client.get_slider_captcha()),
            ("点击", client.get_click_captcha()),
            ("图形", client.get_image_captcha(type_='mixed', count=4)),
            ("手势", client.get_gesture_captcha()),
        ]

        print("并发获取多种验证码...")

        for name, task in tasks:
            try:
                result = await task
                print(f"  ✓ {name}验证码: {getattr(result, 'session_id', getattr(result, 'challenge_id', 'N/A'))}")
            except Exception as e:
                print(f"  ✗ {name}验证码失败: {e}")


def error_handling_example():
    """高级错误处理示例"""
    print("\n" + "="*60)
    print("高级错误处理示例")
    print("="*60)

    test_cases = [
        ("网络错误", "http://nonexistent.example.com"),
        ("超短超时", "http://localhost:8080"),
    ]

    for name, url in test_cases:
        print(f"\n测试场景: {name}")
        try:
            with CaptchaClient(
                base_url=url,
                timeout=2 if name == "超短超时" else 5,
            ) as client:
                slider = client.get_slider_captcha()
                print(f"  ✓ 成功: {slider.session_id}")

        except CaptchaTimeoutError as e:
            print(f"  ✗ 超时错误: {e}")

        except CaptchaNetworkError as e:
            print(f"  ✗ 网络错误: {e}")

        except CaptchaAPIError as e:
            print(f"  ✗ API错误: code={e.code}, message={e.message}")

        except CaptchaError as e:
            print(f"  ✗ 验证码错误: {e}")


def performance_comparison_example():
    """性能对比示例"""
    print("\n" + "="*60)
    print("性能对比示例")
    print("="*60)

    num_requests = 5

    print("\n1. 串行请求:")
    start_time = time.time()
    with CaptchaClient(base_url="http://localhost:8080") as client:
        for i in range(num_requests):
            slider = client.get_slider_captcha()
            result = client.verify_slider_captcha(
                session_id=slider.session_id,
                x=150,
            )
    serial_duration = time.time() - start_time
    print(f"  耗时: {serial_duration:.2f}秒")

    print("\n2. 批量并发请求:")
    start_time = time.time()

    def process(i):
        with CaptchaClient(base_url="http://localhost:8080") as client:
            slider = client.get_slider_captcha()
            return client.verify_slider_captcha(
                session_id=slider.session_id,
                x=150,
            )

    with ThreadPoolExecutor(max_workers=5) as executor:
        futures = [executor.submit(process, i) for i in range(num_requests)]
        results = [f.result() for f in as_completed(futures)]

    concurrent_duration = time.time() - start_time
    print(f"  耗时: {concurrent_duration:.2f}秒")

    print(f"\n性能对比:")
    print(f"  串行: {serial_duration:.2f}秒")
    print(f"  并发: {concurrent_duration:.2f}秒")
    print(f"  加速比: {serial_duration/concurrent_duration:.2f}x")


def main():
    """主函数"""
    print("="*60)
    print("Python SDK 高级完整示例")
    print("="*60)

    examples = [
        ("基础同步示例", basic_sync_example),
        ("点击验证码示例", click_captcha_example),
        ("图形验证码示例", image_captcha_example),
        ("批量处理示例", batch_processing_example),
        ("错误处理示例", error_handling_example),
        ("性能对比示例", performance_comparison_example),
        ("基础异步示例", lambda: asyncio.run(basic_async_example())),
        ("并发异步示例", lambda: asyncio.run(concurrent_async_example())),
        ("混合验证码异步示例", lambda: asyncio.run(mixed_captcha_async_example())),
    ]

    print("\n可用示例:")
    for i, (name, _) in enumerate(examples, 1):
        print(f"  {i}. {name}")
    print("  0. 运行全部")

    choice = input("\n请选择示例 (0-{}): ".format(len(examples))).strip()

    if choice == "0":
        for name, func in examples:
            try:
                func()
            except Exception as e:
                print(f"\n✗ 示例 '{name}' 执行失败: {e}")
    else:
        try:
            index = int(choice) - 1
            if 0 <= index < len(examples):
                examples[index][1]()
            else:
                print("无效的选择")
        except ValueError:
            print("无效的输入")
        except Exception as e:
            print(f"执行失败: {e}")

    print("\n" + "="*60)
    print("示例完成")
    print("="*60)


if __name__ == "__main__":
    main()
