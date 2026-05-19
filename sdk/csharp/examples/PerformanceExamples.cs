using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Linq;
using System.Threading.Tasks;
using Hjtpx.Captcha.Client;
using Hjtpx.Captcha.Models;

namespace Hjtpx.Captcha.Examples
{
    public class PerformanceExamples
    {
        private readonly CaptchaClient _client;

        public PerformanceExamples(CaptchaClient client)
        {
            _client = client;
        }

        public async Task RunAllExamples()
        {
            Console.WriteLine("\n" + new string('=', 60));
            Console.WriteLine("性能测试示例");
            Console.WriteLine(new string('=', 60));

            await SingleRequestBenchmark();
            await ConcurrentRequestsBenchmark();
            await BatchProcessingBenchmark();
            await LatencyDistributionBenchmark();
            await ThroughputBenchmark();
        }

        private async Task SingleRequestBenchmark()
        {
            Console.WriteLine("\n[单请求基准测试]");
            Console.WriteLine(new string('-', 40));

            var stopwatch = new Stopwatch();
            var latencies = new List<double>();

            for (int i = 0; i < 100; i++)
            {
                stopwatch.Restart();
                try
                {
                    var captcha = await _client.GetSliderCaptchaAsync();
                    stopwatch.Stop();
                    latencies.Add(stopwatch.Elapsed.TotalMilliseconds);
                }
                catch (Exception ex)
                {
                    Console.WriteLine($"请求 {i + 1} 失败: {ex.Message}");
                }

                if ((i + 1) % 10 == 0)
                {
                    Console.WriteLine($"  进度: {i + 1}/100");
                }
            }

            PrintLatencyStatistics(latencies, "单请求");
        }

        private async Task ConcurrentRequestsBenchmark()
        {
            Console.WriteLine("\n[并发请求基准测试]");
            Console.WriteLine(new string('-', 40));

            var concurrencyLevels = new[] { 5, 10, 20, 50, 100 };

            foreach (var concurrency in concurrencyLevels)
            {
                Console.WriteLine($"\n并发级别: {concurrency}");

                var stopwatch = new Stopwatch();
                var tasks = new List<Task<CaptchaResult>>();
                var latencies = new List<double>();

                stopwatch.Restart();

                for (int i = 0; i < concurrency; i++)
                {
                    tasks.Add(Task.Run(async () =>
                    {
                        var sw = new Stopwatch();
                        sw.Start();
                        try
                        {
                            var captcha = await _client.GetSliderCaptchaAsync();
                            sw.Stop();
                            lock (latencies)
                            {
                                latencies.Add(sw.Elapsed.TotalMilliseconds);
                            }
                            return new CaptchaResult { Success = true, Latency = sw.Elapsed.TotalMilliseconds };
                        }
                        catch (Exception ex)
                        {
                            sw.Stop();
                            lock (latencies)
                            {
                                latencies.Add(sw.Elapsed.TotalMilliseconds);
                            }
                            return new CaptchaResult { Success = false, Error = ex.Message };
                        }
                    }));
                }

                var results = await Task.WhenAll(tasks);
                stopwatch.Stop();

                var successCount = results.Count(r => r.Success);
                Console.WriteLine($"  耗时: {stopwatch.Elapsed.TotalSeconds:F2} 秒");
                Console.WriteLine($"  成功率: {successCount}/{concurrency} ({successCount * 100.0 / concurrency:F1}%)");
                Console.WriteLine($"  QPS: {(successCount / stopwatch.Elapsed.TotalSeconds):F2}");
                PrintLatencyStatistics(latencies, "并发");
            }
        }

        private async Task BatchProcessingBenchmark()
        {
            Console.WriteLine("\n[批量处理基准测试]");
            Console.WriteLine(new string('-', 40));

            var batchSizes = new[] { 10, 50, 100, 500 };

            foreach (var batchSize in batchSizes)
            {
                Console.WriteLine($"\n批次大小: {batchSize}");

                var stopwatch = new Stopwatch();
                var successCount = 0;
                var failCount = 0;

                stopwatch.Restart();

                for (int i = 0; i < batchSize; i++)
                {
                    try
                    {
                        await _client.GetSliderCaptchaAsync();
                        successCount++;
                    }
                    catch
                    {
                        failCount++;
                    }

                    if ((i + 1) % 10 == 0)
                    {
                        await Task.Delay(10);
                    }
                }

                stopwatch.Stop();

                Console.WriteLine($"  耗时: {stopwatch.Elapsed.TotalSeconds:F2} 秒");
                Console.WriteLine($"  成功: {successCount}, 失败: {failCount}");
                Console.WriteLine($"  QPS: {(successCount / stopwatch.Elapsed.TotalSeconds):F2}");
            }
        }

        private async Task LatencyDistributionBenchmark()
        {
            Console.WriteLine("\n[延迟分布基准测试]");
            Console.WriteLine(new string('-', 40));

            var sampleSize = 1000;
            var latencies = new List<double>();

            Console.WriteLine($"收集 {sampleSize} 个样本...");

            for (int i = 0; i < sampleSize; i++)
            {
                var stopwatch = new Stopwatch();
                stopwatch.Start();

                try
                {
                    await _client.GetSliderCaptchaAsync();
                    stopwatch.Stop();
                    latencies.Add(stopwatch.Elapsed.TotalMilliseconds);
                }
                catch
                {
                    stopwatch.Stop();
                    latencies.Add(stopwatch.Elapsed.TotalMilliseconds);
                }

                if ((i + 1) % 100 == 0)
                {
                    Console.WriteLine($"  进度: {i + 1}/{sampleSize}");
                }
            }

            latencies.Sort();

            Console.WriteLine("\n延迟分布统计:");
            Console.WriteLine($"  样本数: {latencies.Count}");
            Console.WriteLine($"  平均值: {latencies.Average():F2} ms");
            Console.WriteLine($"  最小值: {latencies.Min():F2} ms");
            Console.WriteLine($"  最大值: {latencies.Max():F2} ms");
            Console.WriteLine($"  中位数 (P50): {GetPercentile(latencies, 0.5):F2} ms");
            Console.WriteLine($"  P90: {GetPercentile(latencies, 0.9):F2} ms");
            Console.WriteLine($"  P95: {GetPercentile(latencies, 0.95):F2} ms");
            Console.WriteLine($"  P99: {GetPercentile(latencies, 0.99):F2} ms");
            Console.WriteLine($"  P99.9: {GetPercentile(latencies, 0.999):F2} ms");
        }

        private async Task ThroughputBenchmark()
        {
            Console.WriteLine("\n[吞吐量基准测试]");
            Console.WriteLine(new string('-', 40));

            var durationSeconds = 10;
            var requestCount = 0;
            var successCount = 0;
            var failCount = 0;

            Console.WriteLine($"测试时长: {durationSeconds} 秒");

            var startTime = DateTime.UtcNow;
            var tasks = new List<Task>();

            while ((DateTime.UtcNow - startTime).TotalSeconds < durationSeconds)
            {
                tasks.Add(Task.Run(async () =>
                {
                    try
                    {
                        await _client.GetSliderCaptchaAsync();
                        Interlocked.Increment(ref successCount);
                    }
                    catch
                    {
                        Interlocked.Increment(ref failCount);
                    }
                    Interlocked.Increment(ref requestCount);
                }));

                if (tasks.Count >= 100)
                {
                    await Task.WhenAll(tasks);
                    tasks.Clear();
                }
            }

            if (tasks.Count > 0)
            {
                await Task.WhenAll(tasks);
            }

            var elapsed = (DateTime.UtcNow - startTime).TotalSeconds;

            Console.WriteLine($"\n测试结果:");
            Console.WriteLine($"  总请求数: {requestCount}");
            Console.WriteLine($"  成功: {successCount}, 失败: {failCount}");
            Console.WriteLine($"  成功率: {successCount * 100.0 / requestCount:F2}%");
            Console.WriteLine($"  QPS: {requestCount / elapsed:F2}");
        }

        private void PrintLatencyStatistics(List<double> latencies, string label)
        {
            if (latencies.Count == 0)
            {
                Console.WriteLine("  无数据");
                return;
            }

            latencies.Sort();
            Console.WriteLine($"\n  {label}延迟统计:");
            Console.WriteLine($"    样本数: {latencies.Count}");
            Console.WriteLine($"    平均: {latencies.Average():F2} ms");
            Console.WriteLine($"    最小: {latencies.Min():F2} ms");
            Console.WriteLine($"    最大: {latencies.Max():F2} ms");
            Console.WriteLine($"    P50: {GetPercentile(latencies, 0.5):F2} ms");
            Console.WriteLine($"    P95: {GetPercentile(latencies, 0.95):F2} ms");
            Console.WriteLine($"    P99: {GetPercentile(latencies, 0.99):F2} ms");
        }

        private double GetPercentile(List<double> sortedData, double percentile)
        {
            if (sortedData.Count == 0) return 0;

            var index = (int)Math.Ceiling(percentile * sortedData.Count) - 1;
            return sortedData[Math.Max(0, Math.Min(index, sortedData.Count - 1))];
        }

        private class CaptchaResult
        {
            public bool Success { get; set; }
            public double Latency { get; set; }
            public string Error { get; set; }
        }
    }
}
