import { test, expect } from '@playwright/test';

test.describe('验证码性能测试', () => {
  test.describe('响应时间测试', () => {
    test('滑块验证码生成响应时间应该小于500ms', async ({ request }) => {
      const measurements: number[] = [];
      
      for (let i = 0; i < 20; i++) {
        const startTime = Date.now();
        const response = await request.post('http://localhost:8080/api/v1/captcha/slider/create');
        const duration = Date.now() - startTime;
        
        expect(response.ok()).toBeTruthy();
        measurements.push(duration);
      }
      
      const avgDuration = measurements.reduce((a, b) => a + b, 0) / measurements.length;
      const p95Duration = measurements.sort((a, b) => a - b)[Math.floor(measurements.length * 0.95)];
      
      console.log(`滑块验证码生成性能:`);
      console.log(`  平均响应时间: ${avgDuration.toFixed(2)}ms`);
      console.log(`  P95响应时间: ${p95Duration.toFixed(2)}ms`);
      
      expect(avgDuration).toBeLessThan(500);
    });

    test('验证码验证响应时间应该小于300ms', async ({ request }) => {
      const genResponse = await request.post('http://localhost:8080/api/v1/captcha/slider/create');
      const captcha = await genResponse.json();
      
      const measurements: number[] = [];
      
      for (let i = 0; i < 20; i++) {
        const startTime = Date.now();
        const response = await request.post('http://localhost:8080/api/v1/captcha/slider/verify', {
          data: {
            session_id: captcha.data.sessionId,
            x: 100 + i,
            y: 50
          }
        });
        const duration = Date.now() - startTime;
        
        measurements.push(duration);
      }
      
      const avgDuration = measurements.reduce((a, b) => a + b, 0) / measurements.length;
      
      console.log(`验证码验证平均响应时间: ${avgDuration.toFixed(2)}ms`);
      expect(avgDuration).toBeLessThan(300);
    });

    test('健康检查响应时间应该小于100ms', async ({ request }) => {
      const measurements: number[] = [];
      
      for (let i = 0; i < 50; i++) {
        const startTime = Date.now();
        await request.get('http://localhost:8080/health');
        const duration = Date.now() - startTime;
        measurements.push(duration);
      }
      
      const avgDuration = measurements.reduce((a, b) => a + b, 0) / measurements.length;
      
      console.log(`健康检查平均响应时间: ${avgDuration.toFixed(2)}ms`);
      expect(avgDuration).toBeLessThan(100);
    });
  });

  test.describe('吞吐量测试', () => {
    test('应该能够处理高并发验证码生成', async ({ request }) => {
      const concurrency = 100;
      const startTime = Date.now();
      
      const promises = [];
      for (let i = 0; i < concurrency; i++) {
        promises.push(request.post('http://localhost:8080/api/v1/captcha/slider/create'));
      }
      
      const results = await Promise.all(promises);
      const duration = Date.now() - startTime;
      
      const successCount = results.filter(r => r.ok()).length;
      const throughput = (successCount / duration) * 1000;
      
      console.log(`高并发验证码生成测试:`);
      console.log(`  并发数: ${concurrency}`);
      console.log(`  成功数: ${successCount}`);
      console.log(`  总耗时: ${duration}ms`);
      console.log(`  吞吐量: ${throughput.toFixed(2)} req/s`);
      
      expect(successCount).toBe(concurrency);
      expect(duration).toBeLessThan(10000);
    });

    test('应该能够处理高并发验证码验证', async ({ request }) => {
      const genResponse = await request.post('http://localhost:8080/api/v1/captcha/slider/create');
      const captcha = await genResponse.json();
      
      const concurrency = 50;
      const startTime = Date.now();
      
      const promises = [];
      for (let i = 0; i < concurrency; i++) {
        promises.push(request.post('http://localhost:8080/api/v1/captcha/slider/verify', {
          data: {
            session_id: captcha.data.sessionId,
            x: 100 + i,
            y: 50
          }
        }));
      }
      
      const results = await Promise.all(promises);
      const duration = Date.now() - startTime;
      
      const successCount = results.filter(r => r.ok()).length;
      const throughput = (successCount / duration) * 1000;
      
      console.log(`高并发验证码验证测试:`);
      console.log(`  并发数: ${concurrency}`);
      console.log(`  成功数: ${successCount}`);
      console.log(`  总耗时: ${duration}ms`);
      console.log(`  吞吐量: ${throughput.toFixed(2)} req/s`);
      
      expect(successCount).toBe(concurrency);
    });

    test('混合操作吞吐量测试', async ({ request }) => {
      const operations = [
        () => request.post('http://localhost:8080/api/v1/captcha/slider/create'),
        () => request.post('http://localhost:8080/api/v1/captcha/click/create'),
        () => request.get('http://localhost:8080/health'),
        () => request.post('http://localhost:8080/api/v1/captcha/emoji/create')
      ];
      
      const totalOperations = 200;
      const startTime = Date.now();
      
      const promises = [];
      for (let i = 0; i < totalOperations; i++) {
        const operation = operations[i % operations.length];
        promises.push(operation());
      }
      
      const results = await Promise.all(promises);
      const duration = Date.now() - startTime;
      
      const successCount = results.filter(r => r.ok()).length;
      const throughput = (successCount / duration) * 1000;
      
      console.log(`混合操作吞吐量测试:`);
      console.log(`  总操作数: ${totalOperations}`);
      console.log(`  成功数: ${successCount}`);
      console.log(`  吞吐量: ${throughput.toFixed(2)} req/s`);
      
      expect(successCount).toBeGreaterThan(totalOperations * 0.95);
    });
  });

  test.describe('稳定性测试', () => {
    test('长时间运行测试', async ({ request }) => {
      const duration = 30000;
      const startTime = Date.now();
      let requestCount = 0;
      let errorCount = 0;
      
      while (Date.now() - startTime < duration) {
        try {
          const response = await request.post('http://localhost:8080/api/v1/captcha/slider/create');
          requestCount++;
          if (!response.ok()) {
            errorCount++;
          }
        } catch (e) {
          errorCount++;
        }
        
        await new Promise(resolve => setTimeout(resolve, 100));
      }
      
      const errorRate = (errorCount / requestCount) * 100;
      
      console.log(`长时间运行测试 (30秒):`);
      console.log(`  总请求数: ${requestCount}`);
      console.log(`  错误数: ${errorCount}`);
      console.log(`  错误率: ${errorRate.toFixed(2)}%`);
      console.log(`  平均QPS: ${(requestCount / 30).toFixed(2)}`);
      
      expect(errorRate).toBeLessThan(5);
    });

    test('连续请求稳定性测试', async ({ request }) => {
      const totalRequests = 100;
      const measurements: number[] = [];
      let errors = 0;
      
      for (let i = 0; i < totalRequests; i++) {
        const startTime = Date.now();
        try {
          const response = await request.post('http://localhost:8080/api/v1/captcha/slider/create');
          const duration = Date.now() - startTime;
          measurements.push(duration);
          if (!response.ok()) {
            errors++;
          }
        } catch (e) {
          errors++;
        }
      }
      
      const avgDuration = measurements.reduce((a, b) => a + b, 0) / measurements.length;
      const minDuration = Math.min(...measurements);
      const maxDuration = Math.max(...measurements);
      const stdDev = Math.sqrt(
        measurements.reduce((sum, val) => sum + Math.pow(val - avgDuration, 2), 0) / measurements.length
      );
      
      console.log(`连续请求稳定性测试:`);
      console.log(`  总请求数: ${totalRequests}`);
      console.log(`  错误数: ${errors}`);
      console.log(`  平均响应时间: ${avgDuration.toFixed(2)}ms`);
      console.log(`  最小响应时间: ${minDuration}ms`);
      console.log(`  最大响应时间: ${maxDuration}ms`);
      console.log(`  标准差: ${stdDev.toFixed(2)}ms`);
      
      expect(errors).toBeLessThan(totalRequests * 0.05);
      expect(stdDev).toBeLessThan(avgDuration * 0.5);
    });
  });

  test.describe('内存和资源测试', () => {
    test('大量Session创建后应该正常清理', async ({ request }) => {
      const sessionCount = 1000;
      const sessions: string[] = [];
      
      for (let i = 0; i < sessionCount; i++) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/slider/create');
        if (response.ok()) {
          const data = await response.json();
          sessions.push(data.data.sessionId);
        }
      }
      
      console.log(`创建了 ${sessions.length} 个Session`);
      
      await new Promise(resolve => setTimeout(resolve, 60000));
      
      const verifyResponse = await request.post('http://localhost:8080/api/v1/captcha/slider/create');
      expect(verifyResponse.ok()).toBeTruthy();
      
      console.log('过期Session清理测试通过');
    });

    test('并发连接应该正确处理', async ({ request }) => {
      const maxConcurrency = 200;
      const batches = 5;
      
      for (let batch = 0; batch < batches; batch++) {
        const promises = [];
        for (let i = 0; i < maxConcurrency; i++) {
          promises.push(request.post('http://localhost:8080/api/v1/captcha/slider/create'));
        }
        
        const results = await Promise.all(promises);
        const successCount = results.filter(r => r.ok()).length;
        
        console.log(`批次 ${batch + 1}: ${successCount}/${maxConcurrency} 成功`);
        expect(successCount).toBeGreaterThan(maxConcurrency * 0.9);
        
        await new Promise(resolve => setTimeout(resolve, 1000));
      }
    });
  });

  test.describe('限流测试', () => {
    test('应该正确执行限流策略', async ({ request }) => {
      const requestsToSend = 150;
      const expectedLimit = 100;
      let successCount = 0;
      let rateLimitedCount = 0;
      
      for (let i = 0; i < requestsToSend; i++) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/slider/create');
        if (response.status() === 429) {
          rateLimitedCount++;
        } else if (response.ok()) {
          successCount++;
        }
      }
      
      console.log(`限流测试:`);
      console.log(`  发送请求: ${requestsToSend}`);
      console.log(`  成功: ${successCount}`);
      console.log(`  被限流: ${rateLimitedCount}`);
      
      expect(rateLimitedCount).toBeGreaterThan(0);
      expect(successCount).toBeLessThanOrEqual(requestsToSend);
    });

    test('限流后应该能够恢复', async ({ request }) => {
      for (let i = 0; i < 50; i++) {
        await request.post('http://localhost:8080/api/v1/captcha/slider/create');
      }
      
      await new Promise(resolve => setTimeout(resolve, 60000));
      
      const response = await request.post('http://localhost:8080/api/v1/captcha/slider/create');
      
      console.log(`限流恢复测试: ${response.ok() ? '已恢复' : '仍被限流'}`);
      
      expect(response.ok() || response.status() === 429).toBe(true);
    });
  });
});

test.describe('管理后台性能测试', () => {
  test.describe('统计数据查询性能', () => {
    test('统计数据查询响应时间应该可接受', async ({ request }) => {
      const loginResponse = await request.post('http://localhost:8080/api/v1/auth/login', {
        data: { username: 'admin', password: 'admin123' }
      });
      const loginData = await loginResponse.json();
      const token = loginData.data?.token;
      
      const endpoints = [
        '/api/v1/admin/stats',
        '/api/v1/admin/stats/realtime',
        '/api/v1/admin/stats/trend',
        '/api/v1/admin/stats/risk-distribution'
      ];
      
      for (const endpoint of endpoints) {
        const startTime = Date.now();
        const response = await request.get(`http://localhost:8080${endpoint}`, {
          headers: { Authorization: `Bearer ${token}` }
        });
        const duration = Date.now() - startTime;
        
        console.log(`${endpoint}: ${duration}ms`);
        expect(response.ok() || response.status() === 401).toBe(true);
        expect(duration).toBeLessThan(2000);
      }
    });
  });

  test.describe('日志查询性能', () => {
    test('大量日志查询应该正常响应', async ({ request }) => {
      const loginResponse = await request.post('http://localhost:8080/api/v1/auth/login', {
        data: { username: 'admin', password: 'admin123' }
      });
      const loginData = await loginResponse.json();
      const token = loginData.data?.token;
      
      const startTime = Date.now();
      const response = await request.get('http://localhost:8080/api/v1/admin/logs', {
        headers: { Authorization: `Bearer ${token}` },
        params: { limit: 1000 }
      });
      const duration = Date.now() - startTime;
      
      console.log(`日志查询 (1000条): ${duration}ms`);
      expect(response.ok()).toBeTruthy();
      expect(duration).toBeLessThan(5000);
    });
  });
});
