import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';

test.describe('API性能测试', () => {
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ request }) => {
    apiHelper = new ApiHelper(request);
    apiHelper.clearPerformanceHistory();
  });

  test.describe('延迟指标测试', () => {
    test('健康检查API延迟应该在可接受范围内', async () => {
      for (let i = 0; i < 10; i++) {
        await apiHelper.healthCheck();
      }
      const avgLatency = apiHelper.getAverageLatency();
      const maxLatency = apiHelper.getMaxLatency();

      console.log(`健康检查延迟 - 平均: ${avgLatency}ms, 最大: ${maxLatency}ms`);

      expect(avgLatency).toBeLessThan(1000);
      expect(maxLatency).toBeLessThan(2000);
    });

    test('滑块验证码生成延迟应该在可接受范围内', async () => {
      for (let i = 0; i < 10; i++) {
        const result = await apiHelper.generateSliderCaptcha();
        if (result && result.success) {
          break;
        }
      }
      const avgLatency = apiHelper.getAverageLatency();
      const maxLatency = apiHelper.getMaxLatency();

      console.log(`滑块验证码生成延迟 - 平均: ${avgLatency}ms, 最大: ${maxLatency}ms`);

      expect(avgLatency).toBeLessThan(3000);
    });

    test('点击验证码生成延迟应该在可接受范围内', async () => {
      for (let i = 0; i < 10; i++) {
        const result = await apiHelper.generateClickCaptcha();
        if (result && result.success) {
          break;
        }
      }
      const avgLatency = apiHelper.getAverageLatency();
      const maxLatency = apiHelper.getMaxLatency();

      console.log(`点击验证码生成延迟 - 平均: ${avgLatency}ms, 最大: ${maxLatency}ms`);

      expect(avgLatency).toBeLessThan(3000);
    });

    test('旋转验证码生成延迟应该在可接受范围内', async () => {
      for (let i = 0; i < 10; i++) {
        const result = await apiHelper.generateRotateCaptcha();
        if (result && result.success) {
          break;
        }
      }
      const avgLatency = apiHelper.getAverageLatency();
      const maxLatency = apiHelper.getMaxLatency();

      console.log(`旋转验证码生成延迟 - 平均: ${avgLatency}ms, 最大: ${maxLatency}ms`);

      expect(avgLatency).toBeLessThan(3000);
    });

    test('语音验证码生成延迟应该在可接受范围内', async () => {
      for (let i = 0; i < 5; i++) {
        const result = await apiHelper.generateVoiceCaptcha();
        if (result && result.success) {
          break;
        }
      }
      const avgLatency = apiHelper.getAverageLatency();
      const maxLatency = apiHelper.getMaxLatency();

      console.log(`语音验证码生成延迟 - 平均: ${avgLatency}ms, 最大: ${maxLatency}ms`);

      expect(avgLatency).toBeLessThan(5000);
    });

    test('连连看验证码生成延迟应该在可接受范围内', async () => {
      for (let i = 0; i < 5; i++) {
        const result = await apiHelper.createLianliankanCaptcha();
        if (result && result.success) {
          break;
        }
      }
      const avgLatency = apiHelper.getAverageLatency();
      const maxLatency = apiHelper.getMaxLatency();

      console.log(`连连看验证码生成延迟 - 平均: ${avgLatency}ms, 最大: ${maxLatency}ms`);

      expect(avgLatency).toBeLessThan(5000);
    });
  });

  test.describe('QPS指标测试', () => {
    test('健康检查API应该支持合理QPS', async () => {
      const startTime = Date.now();
      const requests = 20;
      for (let i = 0; i < requests; i++) {
        await apiHelper.healthCheck();
      }
      const duration = Date.now() - startTime;
      const qps = (requests / duration) * 1000;

      console.log(`健康检查QPS: ${qps.toFixed(2)} req/s, 总耗时: ${duration}ms`);

      expect(qps).toBeGreaterThan(1);
    });

    test('滑块验证码生成应该支持合理的QPS', async () => {
      const startTime = Date.now();
      const requests = 10;
      for (let i = 0; i < requests; i++) {
        await apiHelper.generateSliderCaptcha();
      }
      const duration = Date.now() - startTime;
      const qps = (requests / duration) * 1000;

      console.log(`滑块验证码生成QPS: ${qps.toFixed(2)} req/s, 总耗时: ${duration}ms`);

      expect(qps).toBeGreaterThan(0.5);
    });
  });

  test.describe('并发测试', () => {
    test('应该能够处理较高并发验证码生成', async () => {
      const concurrentRequests = 10;
      const promises: Promise<any>[] = [];

      for (let i = 0; i < concurrentRequests; i++) {
        promises.push(apiHelper.generateSliderCaptcha());
      }

      const startTime = Date.now();
      const results = await Promise.all(promises);
      const duration = Date.now() - startTime;

      console.log(`${concurrentRequests}个并发请求完成, 总耗时: ${duration}ms, QPS: ${((concurrentRequests / duration) * 1000).toFixed(2)}`);

      expect(results.length).toBe(concurrentRequests);
      expect(duration).toBeLessThan(60000);
    });
  });

  test.describe('性能稳定性测试', () => {
    test('多次请求应该保持性能稳定', async () => {
      const latencies: number[] = [];
      const rounds = 5;

      for (let r = 0; r < rounds; r++) {
        await apiHelper.healthCheck();
        const metrics = apiHelper.getPerformanceMetrics();
        const lastMetric = metrics[metrics.length - 1];
        latencies.push(lastMetric.latency);
      }

      const avgLatency = apiHelper.getAverageLatency();
      const maxLatency = apiHelper.getMaxLatency();
      const variance = latencies.reduce((sum, lat) => sum + Math.pow(lat - avgLatency, 2), 0) / latencies.length;
      const stdDev = Math.sqrt(variance);

      console.log(`稳定性测试 - 平均延迟: ${avgLatency}ms, 最大延迟: ${maxLatency}ms, 标准差: ${stdDev.toFixed(2)}ms`);

      expect(stdDev).toBeLessThan(avgLatency * 2);
    });
  });
});
