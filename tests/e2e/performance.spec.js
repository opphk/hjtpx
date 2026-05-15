import { test, expect } from '@playwright/test';

test.describe('性能测试', () => {
  test('页面加载时间', async ({ page }) => {
    await page.goto('/');
    
    // 记录关键指标
    const metrics = await page.evaluate(() => {
      return JSON.parse(JSON.stringify(performance.memory));
    });
    
    // 验证页面加载时间在合理范围内
    const loadTime = await page.evaluate(() => {
      const [navigation] = performance.getEntriesByType('navigation');
      return navigation.loadEventEnd - navigation.startTime;
    });
    
    console.log(`页面加载时间: ${loadTime}ms`);
    expect(loadTime).toBeLessThan(3000); // 小于3秒
  });
  
  test('首次内容绘制 (FCP)', async ({ page }) => {
    await page.goto('/');
    
    const fcp = await page.evaluate(() => {
      const entries = performance.getEntriesByType('paint');
      const fcpEntry = entries.find(entry => entry.name === 'first-contentful-paint');
      return fcpEntry ? fcpEntry.startTime : 0;
    });
    
    console.log(`首次内容绘制: ${fcp}ms`);
    expect(fcp).toBeLessThan(2000); // 小于2秒
  });
  
  test('交互响应时间', async ({ page }) => {
    await page.goto('/users');
    
    // 点击用户列表中的编辑按钮
    const startTime = Date.now();
    await page.click('button.edit-user:first-child');
    
    // 等待模态框出现
    await expect(page.locator('[role="dialog"]')).toBeVisible();
    
    const responseTime = Date.now() - startTime;
    console.log(`交互响应时间: ${responseTime}ms`);
    expect(responseTime).toBeLessThan(500); // 小于500毫秒
  });
  
  test('大量数据渲染性能', async ({ page }) => {
    await page.goto('/users');
    
    // 获取1000条用户数据
    const startTime = Date.now();
    
    // 验证表格渲染完成
    await page.waitForSelector('table.users-table tbody tr', { timeout: 5000 });
    
    const renderTime = Date.now() - startTime;
    console.log(`大量数据渲染时间: ${renderTime}ms`);
    expect(renderTime).toBeLessThan(2000); // 小于2秒
  });
  
  test('分页切换性能', async ({ page }) => {
    await page.goto('/users');
    
    const times = [];
    
    // 切换5页并记录时间
    for (let i = 1; i <= 5; i++) {
      const startTime = Date.now();
      await page.click(`button.page-number:text("${i}")`);
      await page.waitForLoadState('networkidle');
      times.push(Date.now() - startTime);
    }
    
    const avgTime = times.reduce((a, b) => a + b, 0) / times.length;
    console.log(`平均分页切换时间: ${avgTime}ms`);
    expect(avgTime).toBeLessThan(1000); // 平均小于1秒
  });
  
  test('内存使用监控', async ({ page }) => {
    await page.goto('/dashboard');
    
    // 记录初始内存
    const initialMemory = await page.evaluate(() => {
      if (performance.memory) {
        return performance.memory.usedJSHeapSize;
      }
      return 0;
    });
    
    // 执行多次导航
    for (let i = 0; i < 10; i++) {
      await page.goto('/users');
      await page.goto('/dashboard');
    }
    
    // 记录最终内存
    const finalMemory = await page.evaluate(() => {
      if (performance.memory) {
        return performance.memory.usedJSHeapSize;
      }
      return 0;
    });
    
    const memoryIncrease = finalMemory - initialMemory;
    console.log(`内存增长: ${memoryIncrease / 1024 / 1024}MB`);
    
    // 内存增长应该小于50MB
    expect(memoryIncrease).toBeLessThan(50 * 1024 * 1024);
  });
});

test.describe('资源加载性能', () => {
  test('静态资源缓存', async ({ page }) => {
    await page.goto('/');
    
    // 第一次访问
    const firstResponse = await page.reload();
    const cacheControl = firstResponse.headers()['cache-control'];
    
    // 验证Cache-Control头存在
    expect(cacheControl).toBeDefined();
    console.log(`Cache-Control: ${cacheControl}`);
  });
  
  test('懒加载组件', async ({ page }) => {
    await page.goto('/');
    
    // 记录初始网络请求数
    const initialRequests = 0;
    
    // 滚动到页面底部，触发懒加载
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    await page.waitForTimeout(1000);
    
    // 验证懒加载内容已加载
    const lazyContent = page.locator('.lazy-loaded-content');
    await expect(lazyContent).toBeVisible();
  });
  
  test('图片优化', async ({ page }) => {
    await page.goto('/');
    
    // 获取所有图片
    const images = await page.$$eval('img', imgs => 
      imgs.map(img => ({
        src: img.src,
        naturalWidth: img.naturalWidth,
        loaded: img.complete && img.naturalWidth > 0
      }))
    );
    
    // 验证图片已正确加载
    for (const img of images) {
      if (img.src && !img.src.includes('data:')) {
        expect(img.loaded).toBe(true);
        console.log(`图片: ${img.src}, 宽度: ${img.naturalWidth}`);
      }
    }
  });
});
