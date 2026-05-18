import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';

test.describe('管理端功能测试', () => {
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ request }) => {
    apiHelper = new ApiHelper(request);
  });

  test.describe('API功能测试', () => {
    test('API应该能够获取健康检查状态', async () => {
      const isHealthy = await apiHelper.healthCheck();
      expect(isHealthy).toBeTruthy();
    });

    test('API应该能够获取性能指标', async () => {
      apiHelper.clearPerformanceHistory();
      await apiHelper.healthCheck();
      const metrics = apiHelper.getPerformanceMetrics();
      expect(metrics.length).toBeGreaterThan(0);
    });
  });

  test.describe('管理后台页面加载测试', () => {
    test('管理后台仪表板页面应该能够访问', async ({ page }) => {
      await page.goto('/admin');
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });

    test('管理后台统计页面应该能够访问', async ({ page }) => {
      await page.goto('/admin/stats');
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });

    test('管理后台应用管理页面应该能够访问', async ({ page }) => {
      await page.goto('/admin/applications');
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });
  });
});
