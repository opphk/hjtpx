"""
Asyncio Async Performance Examples for Python SDK

展示异步SDK的高性能并发能力。
"""

import asyncio
import aiohttp
import time
from typing import List, Dict, Any
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from async_captcha import AsyncCaptchaClient, AsyncCaptchaType, AsyncClickMode
from async_examples import async_basic_example, async_concurrent_example


class PerformanceMonitor:
    """性能监控器"""

    def __init__(self):
        self.start_time = None
        self.end_time = None
        self.request_count = 0
        self.success_count = 0
        self.failure_count = 0
        self.total_latency = 0.0
        self.latencies: List[float] = []

    def start(self):
        """开始监控"""
        self.start_time = time.time()

    def end(self):
        """结束监控"""
        self.end_time = time.time()

    def record_request(self, latency: float, success: bool):
        """记录请求"""
        self.request_count += 1
        self.total_latency += latency
        self.latencies.append(latency)

        if success:
            self.success_count += 1
        else:
            self.failure_count += 1

    def get_report(self) -> Dict[str, Any]:
        """获取性能报告"""
        duration = self.end_time - self.start_time if self.end_time else 0

        if not self.latencies:
            return {
                "duration": duration,
                "total_requests": 0,
                "success_count": 0,
                "failure_count": 0,
                "success_rate": 0.0,
                "requests_per_second": 0.0,
                "avg_latency": 0.0,
                "min_latency": 0.0,
                "max_latency": 0.0,
                "p50_latency": 0.0,
                "p95_latency": 0.0,
                "p99_latency": 0.0,
            }

        sorted_latencies = sorted(self.latencies)
        count = len(sorted_latencies)

        return {
            "duration": duration,
            "total_requests": self.request_count,
            "success_count": self.success_count,
            "failure_count": self.failure_count,
            "success_rate": self.success_count / self.request_count * 100,
            "requests_per_second": self.request_count / duration if duration > 0 else 0,
            "avg_latency": self.total_latency / self.request_count,
            "min_latency": min(sorted_latencies),
            "max_latency": max(sorted_latencies),
            "p50_latency": sorted_latencies[int(count * 0.5)],
            "p95_latency": sorted_latencies[int(count * 0.95)] if count > 1 else sorted_latencies[0],
            "p99_latency": sorted_latencies[int(count * 0.99)] if count > 1 else sorted_latencies[0],
        }

    def print_report(self):
        """打印性能报告"""
        report = self.get_report()

        print("\n" + "=" * 60)
        print("性能监控报告")
        print("=" * 60)
        print(f"测试时长:        {report['duration']:.2f} 秒")
        print(f"总请求数:        {report['total_requests']}")
        print(f"成功请求:        {report['success_count']}")
        print(f"失败请求:        {report['failure_count']}")
        print(f"成功率:          {report['success_rate']:.2f}%")
        print(f"QPS:             {report['requests_per_second']:.2f} 请求/秒")
        print()
        print("延迟统计:")
        print(f"  平均延迟:      {report['avg_latency']*1000:.2f} ms")
        print(f"  最小延迟:      {report['min_latency']*1000:.2f} ms")
        print(f"  最大延迟:      {report['max_latency']*1000:.2f} ms")
        print(f"  P50延迟:       {report['p50_latency']*1000:.2f} ms")
        print(f"  P95延迟:       {report['p95_latency']*1000:.2f} ms")
        print(f"  P99延迟:       {report['p99_latency']*1000:.2f} ms")
        print("=" * 60)


async def performance_benchmark_example():
    """性能基准测试示例"""
    print("\n" + "="*60)
    print("异步性能基准测试")
    print("="*60)

    monitor = PerformanceMonitor()

    async with AsyncCaptchaClient(base_url="http://localhost:8080") as client:
        monitor.start()

        for i in range(100):
            start = time.time()
            try:
                await client.get_slider_captcha()
                monitor.record_request(time.time() - start, True)
            except Exception as e:
                monitor.record_request(time.time() - start, False)
                print(f"请求 {i+1} 失败: {e}")

            if (i + 1) % 10 == 0:
                report = monitor.get_report()
                print(f"进度: {i+1}/100, 当前QPS: {report['requests_per_second']:.2f}")

        monitor.end()

    monitor.print_report()


async def concurrent_stress_test_example():
    """并发压力测试示例"""
    print("\n" + "="*60)
    print("并发压力测试")
    print("="*60)

    concurrency = 50
    requests_per_worker = 20

    monitor = PerformanceMonitor()

    async def worker(worker_id: int):
        """工作协程"""
        async with AsyncCaptchaClient(base_url="http://localhost:8080") as client:
            for i in range(requests_per_worker):
                start = time.time()
                try:
                    await client.get_slider_captcha()
                    monitor.record_request(time.time() - start, True)
                except Exception as e:
                    monitor.record_request(time.time() - start, False)
                    print(f"Worker-{worker_id} 请求 {i+1} 失败: {e}")

                await asyncio.sleep(0.01)

    monitor.start()
    tasks = [worker(i) for i in range(concurrency)]
    await asyncio.gather(*tasks)
    monitor.end()

    print(f"\n完成 {concurrency} 个并发工作协程，每个执行 {requests_per_worker} 个请求")
    monitor.print_report()


async def mixed_captcha_performance():
    """混合验证码性能测试"""
    print("\n" + "="*60)
    print("混合验证码性能测试")
    print("="*60)

    monitor = PerformanceMonitor()

    captcha_types = [
        ("滑块验证码", lambda c: c.get_slider_captcha()),
        ("点击验证码", lambda c: c.get_click_captcha()),
        ("图形验证码", lambda c: c.get_image_captcha()),
        ("旋转验证码", lambda c: c.get_rotation_captcha()),
        ("手势验证码", lambda c: c.get_gesture_captcha()),
    ]

    async with AsyncCaptchaClient(base_url="http://localhost:8080") as client:
        monitor.start()

        for _ in range(20):
            tasks = []
            for name, captcha_func in captcha_types:
                tasks.append(captcha_func(client))

            results = await asyncio.gather(*tasks, return_exceptions=True)

            for i, result in enumerate(results):
                if isinstance(result, Exception):
                    print(f"{captcha_types[i][0]} 失败: {result}")
                    monitor.record_request(0, False)
                else:
                    monitor.record_request(0, True)

        monitor.end()

    print("\n混合验证码测试完成:")
    print(f"  总请求数: {monitor.request_count}")
    print(f"  成功率: {monitor.success_count}/{monitor.request_count}")


async def latency_distribution_test():
    """延迟分布测试"""
    print("\n" + "="*60)
    print("延迟分布测试")
    print("="*60)

    sample_sizes = [10, 50, 100, 200, 500]
    results = {}

    async with AsyncCaptchaClient(base_url="http://localhost:8080") as client:
        for size in sample_sizes:
            print(f"\n测试样本大小: {size}")
            latencies = []

            for i in range(size):
                start = time.time()
                try:
                    await client.get_slider_captcha()
                    latency = (time.time() - start) * 1000
                    latencies.append(latency)
                except Exception as e:
                    print(f"  请求 {i+1} 失败: {e}")

                await asyncio.sleep(0.01)

            if latencies:
                latencies.sort()
                n = len(latencies)
                results[size] = {
                    "count": n,
                    "avg": sum(latencies) / n,
                    "min": min(latencies),
                    "max": max(latencies),
                    "p50": latencies[int(n * 0.5)],
                    "p95": latencies[int(n * 0.95)],
                    "p99": latencies[int(n * 0.99)],
                }

                print(f"  平均: {results[size]['avg']:.2f} ms")
                print(f"  P50:  {results[size]['p50']:.2f} ms")
                print(f"  P95:  {results[size]['p95']:.2f} ms")
                print(f"  P99:  {results[size]['p99']:.2f} ms")

    print("\n延迟分布对比:")
    print("-" * 60)
    print(f"{'样本数':<10} {'平均ms':<10} {'P50':<10} {'P95':<10} {'P99':<10}")
    print("-" * 60)
    for size, stats in results.items():
        print(f"{size:<10} {stats['avg']:<10.2f} {stats['p50']:<10.2f} {stats['p95']:<10.2f} {stats['p99']:<10.2f}")


async def connection_pool_test():
    """连接池测试"""
    print("\n" + "="*60)
    print("连接池测试")
    print("="*60)

    max_connections_list = [10, 50, 100, 200]

    for max_conn in max_connections_list:
        print(f"\n测试最大连接数: {max_conn}")

        monitor = PerformanceMonitor()

        async with AsyncCaptchaClient(
            base_url="http://localhost:8080",
            max_connections=max_conn
        ) as client:
            monitor.start()

            tasks = [client.get_slider_captcha() for _ in range(100)]
            results = await asyncio.gather(*tasks, return_exceptions=True)

            for result in results:
                if isinstance(result, Exception):
                    monitor.record_request(0, False)
                else:
                    monitor.record_request(0, True)

            monitor.end()

        report = monitor.get_report()
        print(f"  请求数: {report['total_requests']}")
        print(f"  成功率: {report['success_rate']:.2f}%")
        print(f"  QPS: {report['requests_per_second']:.2f}")


async def timeout_handling_example():
    """超时处理示例"""
    print("\n" + "="*60)
    print("超时处理示例")
    print("="*60)

    timeout_configs = [1, 5, 10, 30]

    for timeout in timeout_configs:
        print(f"\n测试超时时间: {timeout} 秒")

        monitor = PerformanceMonitor()

        async with AsyncCaptchaClient(
            base_url="http://localhost:8080",
            timeout=timeout
        ) as client:
            start = time.time()
            try:
                slider = await client.get_slider_captcha()
                latency = time.time() - start
                monitor.record_request(latency, True)
                print(f"  ✓ 成功: 延迟 {latency*1000:.2f} ms")
            except asyncio.TimeoutError:
                print(f"  ✗ 超时")
                monitor.record_request(time.time() - start, False)
            except Exception as e:
                print(f"  ✗ 错误: {e}")
                monitor.record_request(time.time() - start, False)


async def retry_logic_test():
    """重试逻辑测试"""
    print("\n" + "="*60)
    print("重试逻辑测试")
    print("="*60)

    retry_configs = [(1, 0.1), (3, 0.5), (5, 1.0)]

    for max_retries, backoff in retry_configs:
        print(f"\n测试重试配置: 最大重试={max_retries}, 退避={backoff}秒")

        monitor = PerformanceMonitor()

        async with AsyncCaptchaClient(
            base_url="http://localhost:8080",
            max_retries=max_retries,
            retry_backoff_factor=backoff
        ) as client:
            for i in range(10):
                start = time.time()
                try:
                    await client.get_slider_captcha()
                    monitor.record_request(time.time() - start, True)
                    print(f"  请求 {i+1}: 成功")
                except Exception as e:
                    monitor.record_request(time.time() - start, False)
                    print(f"  请求 {i+1}: 失败 - {e}")

                await asyncio.sleep(0.1)

        report = monitor.get_report()
        print(f"  结果: {report['success_count']}/{report['total_requests']} 成功")


async def batch_processing_example():
    """批量处理示例"""
    print("\n" + "="*60)
    print("批量处理示例")
    print("="*60)

    batch_sizes = [10, 50, 100, 500]

    for batch_size in batch_sizes:
        print(f"\n处理批次大小: {batch_size}")

        monitor = PerformanceMonitor()
        monitor.start()

        async with AsyncCaptchaClient(base_url="http://localhost:8080") as client:
            for batch_start in range(0, batch_size, 10):
                batch_end = min(batch_start + 10, batch_size)
                batch = range(batch_start, batch_end)

                tasks = [client.get_slider_captcha() for _ in batch]
                results = await asyncio.gather(*tasks, return_exceptions=True)

                for result in results:
                    if isinstance(result, Exception):
                        monitor.record_request(0, False)
                    else:
                        monitor.record_request(0, True)

                await asyncio.sleep(0.1)

        monitor.end()

        report = monitor.get_report()
        print(f"  耗时: {report['duration']:.2f} 秒")
        print(f"  QPS: {report['requests_per_second']:.2f}")
        print(f"  成功率: {report['success_rate']:.2f}%")


async def main():
    """主函数"""
    print("="*60)
    print("Python 异步 SDK 性能示例")
    print("="*60)

    examples = [
        ("基础性能测试", performance_benchmark_example),
        ("并发压力测试", concurrent_stress_test_example),
        ("混合验证码测试", mixed_captcha_performance),
        ("延迟分布测试", latency_distribution_test),
        ("连接池测试", connection_pool_test),
        ("超时处理测试", timeout_handling_example),
        ("重试逻辑测试", retry_logic_test),
        ("批量处理测试", batch_processing_example),
    ]

    for name, func in examples:
        try:
            print(f"\n{'='*60}")
            print(f"示例: {name}")
            print('='*60)
            await func()
        except Exception as e:
            print(f"\n示例 '{name}' 执行失败: {e}")
            import traceback
            traceback.print_exc()

    print("\n" + "="*60)
    print("所有性能示例运行完成")
    print("="*60)


if __name__ == "__main__":
    asyncio.run(main())
