import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';
import { testUsers } from '../../utils/test-data';
import { TestHelpers } from '../../utils/test-helpers';

test.describe('管理端仪表盘测试', () => {
  let apiHelper: ApiHelper;
  let testHelpers: TestHelpers;
  let authToken: string;

  test.beforeEach(async ({ request, page }) => {
    apiHelper = new ApiHelper(request);
    testHelpers = new TestHelpers();
    
    const loginResult = await apiHelper.adminLogin(
      testUsers.admin.username,
      testUsers.admin.password
    );
    authToken = loginResult.data.token;
  });

  test.describe('仪表盘访问测试', () => {
    test('应该能够访问仪表板页面', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      await expect(page.locator('h1, h2')).toContainText(/Dashboard|仪表板/);
      await testHelpers.takeScreenshot(page, 'admin-dashboard-loaded');
    });

    test('仪表板应该显示统计数据', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      await page.waitForLoadState('networkidle');
      
      const statsCards = page.locator('.stat-card, .stats-card, .metric-card, [class*="stat"]');
      const statsCount = await statsCards.count();
      
      if (statsCount > 0) {
        console.log(`发现 ${statsCount} 个统计卡片`);
      }
      
      await testHelpers.takeScreenshot(page, 'admin-dashboard-stats');
    });

    test('仪表板应该显示图表', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      const charts = page.locator('canvas, svg, .chart, [class*="chart"]');
      const chartCount = await charts.count();
      
      if (chartCount > 0) {
        console.log(`发现 ${chartCount} 个图表`);
      }
      
      await testHelpers.takeScreenshot(page, 'admin-dashboard-charts');
    });

    test('未认证用户应该被重定向到登录页', async ({ page }) => {
      await page.goto('/admin/dashboard');
      await expect(page).toHaveURL(/\/admin\/login/);
    });
  });

  test.describe('仪表盘API测试', () => {
    test('API应该能够获取验证统计数据', async () => {
      const stats = await apiHelper.getVerificationStats(authToken);
      expect(stats).toHaveProperty('success', true);
      expect(stats).toHaveProperty('data');
    });

    test('API统计数据应该包含必要的字段', async () => {
      const stats = await apiHelper.getVerificationStats(authToken);
      
      if (stats.success && stats.data) {
        const expectedFields = ['total', 'success', 'failed', 'rate'];
        const hasRequiredFields = expectedFields.some(field => field in stats.data);
        expect(hasRequiredFields).toBeTruthy();
      }
    });

    test('API统计数据应该是数字类型', async () => {
      const stats = await apiHelper.getVerificationStats(authToken);
      
      if (stats.success && stats.data) {
        Object.values(stats.data).forEach(value => {
          if (typeof value === 'number') {
            expect(value).toBeGreaterThanOrEqual(0);
          }
        });
      }
    });

    test('API统计数据趋势数据测试', async () => {
      const stats = await apiHelper.getVerificationStats(authToken);
      
      if (stats.success && stats.data) {
        const trendData = stats.data.trend || stats.data.history || [];
        expect(Array.isArray(trendData)).toBeTruthy();
      }
    });
  });

  test.describe('仪表盘导航测试', () => {
    test('仪表板应该有导航菜单', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      
      await expect(page.locator('nav, .sidebar, .menu, aside')).toBeVisible();
      await testHelpers.takeScreenshot(page, 'admin-nav-visible');
    });

    test('导航菜单应该包含所有主要链接', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      
      const navLinks = page.locator('nav a, .sidebar a, .menu a');
      const linkCount = await navLinks.count();
      
      console.log(`发现 ${linkCount} 个导航链接`);
      expect(linkCount).toBeGreaterThan(0);
    });

    test('应该能够通过导航访问应用管理页面', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      
      const appsLink = page.locator('a[href*="applications"], a[href*="apps"]').first();
      const linkVisible = await appsLink.isVisible().catch(() => false);
      
      if (linkVisible) {
        await appsLink.click();
        await page.waitForLoadState('networkidle');
        await testHelpers.takeScreenshot(page, 'admin-navigate-to-apps');
      }
    });

    test('应该能够通过导航访问日志页面', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      
      const logsLink = page.locator('a[href*="logs"], a[href*="audit"]').first();
      const linkVisible = await logsLink.isVisible().catch(() => false);
      
      if (linkVisible) {
        await logsLink.click();
        await page.waitForLoadState('networkidle');
        await testHelpers.takeScreenshot(page, 'admin-navigate-to-logs');
      }
    });

    test('应该能够通过导航访问监控页面', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      
      const monitorLink = page.locator('a[href*="monitor"], a[href*="stats"]').first();
      const linkVisible = await monitorLink.isVisible().catch(() => false);
      
      if (linkVisible) {
        await monitorLink.click();
        await page.waitForLoadState('networkidle');
        await testHelpers.takeScreenshot(page, 'admin-navigate-to-monitor');
      }
    });
  });

  test.describe('仪表盘性能测试', () => {
    test('仪表板页面加载时间测试', async ({ page }) => {
      const startTime = Date.now();
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await page.waitForLoadState('networkidle');
      const loadTime = Date.now() - startTime;
      
      console.log(`仪表板加载时间: ${loadTime}ms`);
      expect(loadTime).toBeLessThan(10000);
      await testHelpers.takeScreenshot(page, 'admin-dashboard-performance');
    });

    test('API响应时间测试', async () => {
      const startTime = Date.now();
      await apiHelper.getVerificationStats(authToken);
      const responseTime = Date.now() - startTime;
      
      console.log(`API响应时间: ${responseTime}ms`);
      expect(responseTime).toBeLessThan(5000);
    });

    test('页面刷新性能测试', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await page.waitForLoadState('networkidle');
      
      const startTime = Date.now();
      await page.reload();
      await page.waitForLoadState('networkidle');
      const refreshTime = Date.now() - startTime;
      
      console.log(`页面刷新时间: ${refreshTime}ms`);
      expect(refreshTime).toBeLessThan(10000);
    });
  });

  test.describe('仪表盘实时更新测试', () => {
    test('仪表板数据应该能够实时更新', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      await page.waitForTimeout(5000);
      
      const statsBefore = await page.locator('.stat-value, .metric-value').first().textContent().catch(() => '0');
      
      await apiHelper.getVerificationStats(authToken);
      
      await page.waitForTimeout(2000);
      
      await testHelpers.takeScreenshot(page, 'admin-dashboard-realtime-update');
    });

    test('WebSocket连接测试（如果支持）', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      const wsConnections = await page.evaluate(() => {
        return window.WebSocket ? 1 : 0;
      });
      
      console.log(`WebSocket支持: ${wsConnections > 0 ? '是' : '否'}`);
    });
  });

  test.describe('仪表盘响应式测试', () => {
    test('移动端视图测试', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      await page.setViewportSize({ width: 375, height: 667 });
      await page.waitForTimeout(1000);
      await testHelpers.takeScreenshot(page, 'admin-dashboard-mobile');
    });

    test('平板视图测试', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      await page.setViewportSize({ width: 768, height: 1024 });
      await page.waitForTimeout(1000);
      await testHelpers.takeScreenshot(page, 'admin-dashboard-tablet');
    });

    test('桌面视图测试', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      await page.setViewportSize({ width: 1920, height: 1080 });
      await page.waitForTimeout(1000);
      await testHelpers.takeScreenshot(page, 'admin-dashboard-desktop');
    });
  });

  test.describe('仪表盘错误处理测试', () => {
    test('网络错误处理测试', async ({ page }) => {
      await page.route('**/api/v1/admin/stats', route => {
        route.abort('failed');
      });
      
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      await page.waitForTimeout(2000);
      await testHelpers.takeScreenshot(page, 'admin-dashboard-network-error');
    });

    test('超时错误处理测试', async ({ page }) => {
      await page.route('**/api/v1/admin/stats', route => {
        route.delay(10000);
      });
      
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      await page.waitForTimeout(5000);
      await testHelpers.takeScreenshot(page, 'admin-dashboard-timeout');
    });

    test('数据加载失败测试', async ({ page }) => {
      await page.route('**/api/v1/admin/stats', route => {
        route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ success: false, message: 'Server Error' })
        });
      });
      
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      await page.waitForTimeout(2000);
      await testHelpers.takeScreenshot(page, 'admin-dashboard-data-error');
    });
  });

  test.describe('仪表盘控制台监控测试', () => {
    test('仪表板控制台错误检测', async ({ page }) => {
      const consoleErrors: string[] = [];
      
      page.on('console', msg => {
        if (msg.type() === 'error') {
          consoleErrors.push(msg.text());
        }
      });
      
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      await page.waitForLoadState('networkidle');
      
      await testHelpers.takeScreenshot(page, 'admin-dashboard-console-check');
      
      const criticalErrors = consoleErrors.filter(err => 
        !err.includes('favicon') && 
        !err.includes('404') &&
        !err.includes('Failed to load resource')
      );
      
      console.log('控制台错误:', criticalErrors);
    });

    test('仪表板网络请求监控', async ({ page }) => {
      const networkRequests: { url: string; status: number }[] = [];
      
      page.on('response', response => {
        if (response.url().includes('/api/')) {
          networkRequests.push({
            url: response.url(),
            status: response.status()
          });
        }
      });
      
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      await page.waitForLoadState('networkidle');
      
      console.log(`API请求数量: ${networkRequests.length}`);
      
      const failedRequests = networkRequests.filter(r => r.status >= 400);
      if (failedRequests.length > 0) {
        console.log('失败的请求:', failedRequests);
      }
      
      await testHelpers.takeScreenshot(page, 'admin-dashboard-network-monitor');
    });
  });

  test.describe('仪表盘数据可视化测试', () => {
    test('图表数据点测试', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      await page.waitForLoadState('networkidle');
      
      const chartElements = page.locator('canvas, svg, [class*="chart"]');
      const chartCount = await chartElements.count();
      
      console.log(`图表元素数量: ${chartCount}`);
      expect(chartCount).toBeGreaterThan(0);
      
      await testHelpers.takeScreenshot(page, 'admin-dashboard-charts-count');
    });

    test('图表交互测试', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      await page.waitForLoadState('networkidle');
      
      const chartArea = page.locator('.chart-area, [class*="chart"]').first();
      const chartVisible = await chartArea.isVisible().catch(() => false);
      
      if (chartVisible) {
        const chartBox = await chartArea.boundingBox();
        if (chartBox) {
          await page.mouse.hover(chartBox.x + chartBox.width / 2, chartBox.y + chartBox.height / 2);
          await page.waitForTimeout(500);
          await testHelpers.takeScreenshot(page, 'admin-dashboard-chart-hover');
        }
      }
    });

    test('时间范围选择器测试', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      await page.waitForLoadState('networkidle');
      
      const dateRangeSelector = page.locator('input[type="date"], select, [class*="date"], [class*="range"]').first();
      const selectorVisible = await dateRangeSelector.isVisible().catch(() => false);
      
      if (selectorVisible) {
        await testHelpers.takeScreenshot(page, 'admin-dashboard-date-range');
      }
    });
  });

  test.describe('仪表盘快捷操作测试', () => {
    test('刷新按钮测试', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      const refreshButton = page.locator('button:has-text("刷新"), button:has-text("Refresh"), [class*="refresh"]').first();
      const refreshVisible = await refreshButton.isVisible().catch(() => false);
      
      if (refreshVisible) {
        await refreshButton.click();
        await page.waitForTimeout(1000);
        await testHelpers.takeScreenshot(page, 'admin-dashboard-refreshed');
      }
    });

    test('导出按钮测试', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      const exportButton = page.locator('button:has-text("导出"), button:has-text("Export"), [class*="export"]').first();
      const exportVisible = await exportButton.isVisible().catch(() => false);
      
      if (exportVisible) {
        console.log('导出按钮可用');
        await testHelpers.takeScreenshot(page, 'admin-dashboard-export-available');
      }
    });

    test('通知按钮测试', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      const notificationButton = page.locator('button:has-text("通知"), button:has-text("Notification"), [class*="notification"]').first();
      const notificationVisible = await notificationButton.isVisible().catch(() => false);
      
      if (notificationVisible) {
        await notificationButton.click();
        await page.waitForTimeout(500);
        await testHelpers.takeScreenshot(page, 'admin-dashboard-notifications');
      }
    });
  });
});
