"""
Python SDK 高级异步示例

展示高并发、批量处理、错误处理等高级用法
"""

import asyncio
import aiohttp
import time
from async_captcha import (
    AsyncCaptchaClient,
    AsyncCaptchaError,
    AsyncCaptchaAPIError,
    AsyncCaptchaTimeoutError,
    AsyncCaptchaNetworkError,
    AsyncTrajectoryPoint,
)


async def example_concurrent_captchas():
    """并发获取多种验证码示例"""
    print("\n" + "="*60)
    print("示例1: 并发获取多种验证码")
    print("="*60)

    async with AsyncCaptchaClient("http://localhost:8080") as client:
        tasks = [
            client.get_slider_captcha(width=320, height=160, tolerance=8),
            client.get_click_captcha(mode="number", max_points=3),
            client.get_image_captcha(type_="mixed", count=4),
        ]

        results = await asyncio.gather(*tasks, return_exceptions=True)

        for i, result in enumerate(results):
            captcha_type = ["滑块", "点击", "图形"][i]
            if isinstance(result, Exception):
                print(f"  {captcha_type}验证码获取失败: {result}")
            else:
                print(f"  {captcha_type}验证码获取成功: {result.session_id[:20]}...")


async def example_batch_verification():
    """批量验证示例"""
    print("\n" + "="*60)
    print("示例2: 批量验证")
    print("="*60)

    async with AsyncCaptchaClient("http://localhost:8080") as client:
        sliders = await asyncio.gather(
            client.get_slider_captcha() for _ in range(5)
        )

        print(f"  获取了 {len(sliders)} 个滑块验证码")

        verify_tasks = [
            client.verify_slider_captcha(
                session_id=slider.session_id,
                x=150,
                y=slider.secret_y,
            )
            for slider in sliders
        ]

        results = await asyncio.gather(*verify_tasks, return_exceptions=True)

        success_count = sum(
            1 for r in results
            if not isinstance(r, Exception) and r.success
        )

        print(f"  批量验证成功: {success_count}/{len(results)}")
        print(f"  成功率: {success_count/len(results)*100:.1f}%")


async def example_trajectory_verification():
    """轨迹验证示例"""
    print("\n" + "="*60)
    print("示例3: 轨迹验证")
    print("="*60)

    async with AsyncCaptchaClient("http://localhost:8080") as client:
        slider = await client.get_slider_captcha()
        print(f"  获取验证码成功，SecretY: {slider.secret_y}")

        current_time = int(time.time() * 1000)
        trajectory = [
            AsyncTrajectoryPoint(x=0, y=slider.secret_y, t=current_time - 1000),
            AsyncTrajectoryPoint(x=50, y=slider.secret_y + 2, t=current_time - 800),
            AsyncTrajectoryPoint(x=100, y=slider.secret_y - 1, t=current_time - 500),
            AsyncTrajectoryPoint(x=150, y=slider.secret_y + 1, t=current_time - 200),
            AsyncTrajectoryPoint(x=185, y=slider.secret_y, t=current_time),
        ]

        result = await client.verify_slider_captcha(
            session_id=slider.session_id,
            x=185,
            y=slider.secret_y,
            trajectory=trajectory,
        )

        print(f"  验证结果: {'成功' if result.success else '失败'}")
        if result.trajectory_result:
            print(f"  轨迹评分: {result.trajectory_result.get('score', 0):.2f}")
            print(f"  轨迹通过: {result.trajectory_result.get('passed', False)}")


async def example_error_handling():
    """错误处理示例"""
    print("\n" + "="*60)
    print("示例4: 错误处理")
    print("="*60)

    async with AsyncCaptchaClient("http://localhost:8080", max_retries=2) as client:
        test_cases = [
            ("无效会话ID", "invalid-session-id-12345"),
            ("边界值验证", "session-with-empty-data"),
        ]

        for name, session_id in test_cases:
            print(f"\n  测试: {name}")
            try:
                result = await client.verify_slider_captcha(
                    session_id=session_id,
                    x=100,
                )
                print(f"    结果: {result.success}, {result.message}")
            except AsyncCaptchaAPIError as e:
                print(f"    API错误: code={e.code}, message={e.message}")
            except AsyncCaptchaTimeoutError as e:
                print(f"    超时错误: {e}")
            except AsyncCaptchaNetworkError as e:
                print(f"    网络错误: {e}")
            except AsyncCaptchaError as e:
                print(f"    验证码错误: {e}")


async def example_custom_timeout():
    """自定义超时示例"""
    print("\n" + "="*60)
    print("示例5: 自定义超时配置")
    print("="*60)

    async with AsyncCaptchaClient(
        "http://localhost:8080",
        timeout=5,
        max_retries=3,
        retry_backoff_factor=0.3,
    ) as client:
        print(f"  超时配置: 5秒")
        print(f"  重试次数: 3")
        print(f"  退避因子: 0.3")

        try:
            slider = await client.get_slider_captcha()
            print(f"  获取成功: {slider.session_id[:20]}...")
        except AsyncCaptchaTimeoutError:
            print(f"  请求超时")


async def example_mixed_captcha_workflow():
    """混合验证码工作流示例"""
    print("\n" + "="*60)
    print("示例6: 混合验证码工作流")
    print("="*60)

    async with AsyncCaptchaClient("http://localhost:8080") as client:
        captcha_types = [
            ("滑块", client.get_slider_captcha),
            ("点击", lambda: client.get_click_captcha(mode="number")),
            ("旋转", client.get_rotation_captcha),
            ("手势", client.get_gesture_captcha),
            ("拼图", client.get_jigsaw_captcha),
        ]

        for name, captcha_func in captcha_types:
            try:
                if asyncio.iscoroutinefunction(captcha_func):
                    result = await captcha_func()
                else:
                    result = captcha_func()

                print(f"  {name}验证码: 获取成功")

                if hasattr(result, 'session_id'):
                    print(f"    会话ID: {result.session_id[:20]}...")

            except Exception as e:
                print(f"  {name}验证码: 获取失败 - {e}")


async def example_concurrent_load():
    """高并发压力测试示例"""
    print("\n" + "="*60)
    print("示例7: 高并发压力测试")
    print("="*60)

    start_time = time.time()
    total_requests = 100
    max_concurrent = 20

    async with AsyncCaptchaClient(
        "http://localhost:8080",
        max_connections=max_concurrent,
    ) as client:
        print(f"  总请求数: {total_requests}")
        print(f"  最大并发: {max_concurrent}")

        semaphore = asyncio.Semaphore(max_concurrent)

        async def bounded_request(i):
            async with semaphore:
                try:
                    return await client.get_slider_captcha()
                except Exception as e:
                    return e

        tasks = [bounded_request(i) for i in range(total_requests)]
        results = await asyncio.gather(*tasks)

        success_count = sum(
            1 for r in results
            if not isinstance(r, Exception)
        )

        elapsed = time.time() - start_time

        print(f"  成功: {success_count}/{total_requests}")
        print(f"  耗时: {elapsed:.2f}秒")
        print(f"  QPS: {total_requests/elapsed:.2f}")


async def example_retry_mechanism():
    """重试机制示例"""
    print("\n" + "="*60)
    print("示例8: 重试机制演示")
    print("="*60)

    async with AsyncCaptchaClient(
        "http://localhost:8080",
        max_retries=3,
        retry_backoff_factor=0.5,
    ) as client:
        print(f"  重试配置:")
        print(f"    最大重试次数: 3")
        print(f"    退避因子: 0.5")

        try:
            result = await client.verify_slider_captcha(
                session_id="test-session",
                x=100,
            )
            print(f"  验证结果: {result.success}")
        except Exception as e:
            print(f"  经过所有重试后失败: {e}")


async def example_context_manager():
    """上下文管理器示例"""
    print("\n" + "="*60)
    print("示例9: 上下文管理器使用")
    print("="*60)

    print("  使用 async with 自动管理资源")

    async with AsyncCaptchaClient("http://localhost:8080") as client:
        slider = await client.get_slider_captcha()
        print(f"  获取验证码成功")

    print("  上下文退出，资源自动释放")


async def main():
    """运行所有示例"""
    print("="*60)
    print("Python SDK 高级异步示例")
    print("="*60)

    examples = [
        example_concurrent_captchas,
        example_batch_verification,
        example_trajectory_verification,
        example_error_handling,
        example_custom_timeout,
        example_mixed_captcha_workflow,
        example_concurrent_load,
        example_retry_mechanism,
        example_context_manager,
    ]

    for i, example in enumerate(examples, 1):
        try:
            await example()
        except Exception as e:
            print(f"\n  示例{i}执行失败: {e}")

    print("\n" + "="*60)
    print("所有示例执行完成")
    print("="*60)


if __name__ == "__main__":
    print("提示: 这些示例需要运行中的验证码服务器")
    print("请确保 http://localhost:8080 可访问\n")

    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\n\n用户中断执行")
    except aiohttp.ClientError as e:
        print(f"\n连接错误: {e}")
        print("请检查服务器是否运行")
    except Exception as e:
        print(f"\n未预期的错误: {e}")
