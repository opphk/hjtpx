import { test, expect } from '@playwright/test';
import { TestHelpers } from '../../utils/test-helpers';

test.describe('管理端新增页面测试', () => {
  let testHelpers: TestHelpers;

  test.beforeEach(async ({ page }) => {
    testHelpers = new TestHelpers();
  });

  test('管理端报表中心页面测试', async ({ page }) => {
    console.log('正在测试管理端报表中心页面...');
    await page.goto('/admin/reports');
    await testHelpers.takeScreenshot(page, 'admin-reports-page');
    
    await expect(page.locator('h1')).toContainText('报表中心');
    console.log('✅ 管理端报表中心页面加载成功');
  });

  test('管理端数据可视化页面测试', async ({ page }) => {
    console.log('正在测试管理端数据可视化页面...');
    await page.goto('/admin/visualization');
    await testHelpers.takeScreenshot(page, 'admin-visualization-page');
    
    await expect(page.locator('h1')).toContainText('数据可视化');
    console.log('✅ 管理端数据可视化页面加载成功');
  });

  test('管理端批量操作页面测试', async ({ page }) => {
    console.log('正在测试管理端批量操作页面...');
    await page.goto('/admin/batch-operations');
    await testHelpers.takeScreenshot(page, 'admin-batch-operations-page');
    
    await expect(page.locator('h1')).toContainText('批量操作');
    console.log('✅ 管理端批量操作页面加载成功');
  });

  test('报表中心自定义报表Tab测试', async ({ page }) => {
    console.log('正在测试报表中心自定义报表Tab...');
    await page.goto('/admin/reports');
    await page.waitForLoadState('networkidle');
    
    const customReportTab = page.locator('[data-bs-target="#custom-report"]');
    await expect(customReportTab).toBeVisible();
    console.log('✅ 报表中心自定义报表Tab正常');
  });

  test('报表中心报表模板Tab测试', async ({ page }) => {
    console.log('正在测试报表中心报表模板Tab...');
    await page.goto('/admin/reports');
    await page.waitForLoadState('networkidle');
    
    const templateTab = page.locator('[data-bs-target="#report-templates"]');
    await templateTab.click();
    await expect(page.locator('#report-templates')).toBeVisible();
    console.log('✅ 报表中心报表模板Tab正常');
  });

  test('报表中心数据导出Tab测试', async ({ page }) => {
    console.log('正在测试报表中心数据导出Tab...');
    await page.goto('/admin/reports');
    await page.waitForLoadState('networkidle');
    
    const exportTab = page.locator('[data-bs-target="#data-export"]');
    await exportTab.click();
    await expect(page.locator('#data-export')).toBeVisible();
    console.log('✅ 报表中心数据导出Tab正常');
  });

  test('数据可视化实时数据流测试', async ({ page }) => {
    console.log('正在测试数据可视化实时数据流...');
    await page.goto('/admin/visualization');
    await page.waitForLoadState('networkidle');
    
    const realtimeChart = page.locator('#realtimeChart');
    await expect(realtimeChart).toBeVisible();
    console.log('✅ 数据可视化实时数据流正常');
  });

  test('数据可视化图表交互测试', async ({ page }) => {
    console.log('正在测试数据可视化图表交互...');
    await page.goto('/admin/visualization');
    await page.waitForLoadState('networkidle');
    
    const chartControls = page.locator('.control-btn[data-type]');
    await expect(chartControls.first()).toBeVisible();
    console.log('✅ 数据可视化图表交互正常');
  });

  test('批量操作选择测试', async ({ page }) => {
    console.log('正在测试批量操作选择功能...');
    await page.goto('/admin/batch-operations');
    await page.waitForLoadState('networkidle');
    
    const itemCard = page.locator('.item-card').first();
    await itemCard.click();
    await expect(itemCard).toHaveClass(/selected/);
    console.log('✅ 批量操作选择功能正常');
  });

  test('批量操作全选测试', async ({ page }) => {
    console.log('正在测试批量操作全选功能...');
    await page.goto('/admin/batch-operations');
    await page.waitForLoadState('networkidle');
    
    const selectAll = page.locator('#selectAllApps');
    await selectAll.check();
    const selectedCount = await page.locator('#selectedCount').textContent();
    expect(parseInt(selectedCount || '0')).toBeGreaterThan(0);
    console.log('✅ 批量操作全选功能正常');
  });
});
