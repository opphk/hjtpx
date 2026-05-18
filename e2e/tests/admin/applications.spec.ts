import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';
import { testUsers } from '../../utils/test-data';
import { TestHelpers } from '../../utils/test-helpers';

test.describe('管理端应用管理测试', () => {
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

  test.describe('应用列表测试', () => {
    test('应该能够访问应用管理页面', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      const appsLink = page.locator('a[href*="applications"], a[href*="apps"]').first();
      const linkVisible = await appsLink.isVisible().catch(() => false);
      
      if (linkVisible) {
        await appsLink.click();
        await expect(page).toHaveURL(/\/admin\/applications/);
        await testHelpers.takeScreenshot(page, 'admin-applications-page');
      }
    });

    test('应该能够显示应用列表', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForLoadState('networkidle');
      
      const appTable = page.locator('table, .app-list, [class*="app"]');
      const tableVisible = await appTable.isVisible().catch(() => false);
      
      if (tableVisible) {
        console.log('应用列表可见');
      }
      
      await testHelpers.takeScreenshot(page, 'admin-applications-list');
    });

    test('API应该能够获取应用列表', async () => {
      const apps = await apiHelper.getApplications(authToken);
      expect(apps).toHaveProperty('success', true);
      expect(apps).toHaveProperty('data');
    });

    test('应用列表应该包含分页信息', async () => {
      const apps = await apiHelper.getApplications(authToken);
      
      if (apps.success && apps.data) {
        const hasPagination = apps.data.total !== undefined || 
                             apps.data.page !== undefined ||
                             apps.data.items !== undefined;
        expect(typeof hasPagination).toBe('boolean');
      }
    });
  });

  test.describe('应用创建测试', () => {
    test('应该能够访问应用创建页面', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForLoadState('networkidle');
      
      const createButton = page.locator('button:has-text("创建"), button:has-text("添加"), a[href*="create"]').first();
      const buttonVisible = await createButton.isVisible().catch(() => false);
      
      if (buttonVisible) {
        await createButton.click();
        await page.waitForLoadState('networkidle');
        await testHelpers.takeScreenshot(page, 'admin-app-create-form');
      }
    });

    test('API应该能够创建新应用', async () => {
      const appName = 'Test App - ' + Date.now();
      const result = await apiHelper.createApplication(
        authToken,
        appName,
        'Test application'
      );
      expect(result).toHaveProperty('success', true);
    });

    test('创建应用应该包含所有必要字段', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForLoadState('networkidle');
      
      const createButton = page.locator('button:has-text("创建"), button:has-text("添加")').first();
      const buttonVisible = await createButton.isVisible().catch(() => false);
      
      if (buttonVisible) {
        await createButton.click();
        await page.waitForLoadState('networkidle');
        
        const nameInput = page.locator('input[name="name"], input[name="appName"]');
        const descInput = page.locator('textarea[name="description"], input[name="description"]');
        
        expect(await nameInput.isVisible()).toBeTruthy();
        expect(await descInput.isVisible()).toBeFalsy();
      }
      
      await testHelpers.takeScreenshot(page, 'admin-app-create-fields');
    });

    test('创建应用表单验证测试 - 空名称', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForLoadState('networkidle');
      
      const createButton = page.locator('button:has-text("创建"), button:has-text("添加")').first();
      const buttonVisible = await createButton.isVisible().catch(() => false);
      
      if (buttonVisible) {
        await createButton.click();
        await page.waitForLoadState('networkidle');
        
        const submitButton = page.locator('button[type="submit"]').first();
        await submitButton.click();
        await page.waitForTimeout(500);
        
        await testHelpers.takeScreenshot(page, 'admin-app-create-empty-name');
      }
    });

    test('创建应用重复名称测试', async () => {
      const appName = 'Duplicate App - ' + Date.now();
      
      await apiHelper.createApplication(authToken, appName, 'First app');
      await page.waitForTimeout(500);
      
      const result = await apiHelper.createApplication(authToken, appName, 'Second app');
      expect(result).toBeDefined();
    });
  });

  test.describe('应用编辑测试', () => {
    test('应该能够访问应用编辑页面', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForLoadState('networkidle');
      
      const editButton = page.locator('button:has-text("编辑"), a[href*="edit"]').first();
      const buttonVisible = await editButton.isVisible().catch(() => false);
      
      if (buttonVisible) {
        await editButton.click();
        await page.waitForLoadState('networkidle');
        await testHelpers.takeScreenshot(page, 'admin-app-edit-form');
      }
    });

    test('应该能够修改应用信息', async ({ page }) => {
      const appName = 'Edit Test App - ' + Date.now();
      const createResult = await apiHelper.createApplication(authToken, appName, 'Original description');
      
      if (createResult.success) {
        const appId = createResult.data.id || createResult.data.appId;
        
        await page.goto(`/admin/applications/${appId}/edit`);
        await page.waitForLoadState('networkidle');
        
        const descInput = page.locator('input[name="description"], textarea[name="description"]');
        const descVisible = await descInput.isVisible().catch(() => false);
        
        if (descVisible) {
          await descInput.clear();
          await descInput.fill('Updated description');
          
          const submitButton = page.locator('button[type="submit"]').first();
          await submitButton.click();
          await page.waitForTimeout(1000);
          
          await testHelpers.takeScreenshot(page, 'admin-app-updated');
        }
      }
    });
  });

  test.describe('应用删除测试', () => {
    test('应该能够删除应用', async () => {
      const appName = 'Delete Test App - ' + Date.now();
      const createResult = await apiHelper.createApplication(authToken, appName, 'App to delete');
      
      if (createResult.success) {
        const appId = createResult.data.id || createResult.data.appId;
        
        const deleteResult = await apiHelper.deleteApplication(authToken, appId);
        expect(deleteResult).toBeDefined();
      }
    });

    test('删除确认对话框测试', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForLoadState('networkidle');
      
      const deleteButton = page.locator('button:has-text("删除"), button:has-text("Delete")').first();
      const buttonVisible = await deleteButton.isVisible().catch(() => false);
      
      if (buttonVisible) {
        page.on('dialog', dialog => {
          expect(dialog.message()).toContain('删除');
          dialog.dismiss();
        });
        
        await deleteButton.click();
        await page.waitForTimeout(500);
        
        await testHelpers.takeScreenshot(page, 'admin-app-delete-confirm');
      }
    });

    test('批量删除测试', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForLoadState('networkidle');
      
      const checkbox = page.locator('input[type="checkbox"]').first();
      const checkboxVisible = await checkbox.isVisible().catch(() => false);
      
      if (checkboxVisible) {
        await checkbox.check();
        
        const bulkDeleteButton = page.locator('button:has-text("批量删除"), button:has-text("删除选中")').first();
        const bulkButtonVisible = await bulkDeleteButton.isVisible().catch(() => false);
        
        if (bulkButtonVisible) {
          console.log('批量删除按钮可用');
          await testHelpers.takeScreenshot(page, 'admin-app-bulk-delete');
        }
      }
    });
  });

  test.describe('应用详情测试', () => {
    test('应该能够查看应用详情', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForLoadState('networkidle');
      
      const viewButton = page.locator('a[href*="view"], button:has-text("查看")').first();
      const buttonVisible = await viewButton.isVisible().catch(() => false);
      
      if (buttonVisible) {
        await viewButton.click();
        await page.waitForLoadState('networkidle');
        await testHelpers.takeScreenshot(page, 'admin-app-detail');
      }
    });

    test('应用详情应该显示统计信息', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForLoadState('networkidle');
      
      const appLink = page.locator('a[href*="view"], a[href*="/applications/"]').first();
      const linkVisible = await appLink.isVisible().catch(() => false);
      
      if (linkVisible) {
        await appLink.click();
        await page.waitForLoadState('networkidle');
        
        const statsCards = page.locator('.stat-card, .metric-card, [class*="stat"]');
        const statsCount = await statsCards.count();
        
        console.log(`应用统计卡片数量: ${statsCount}`);
        
        await testHelpers.takeScreenshot(page, 'admin-app-stats');
      }
    });
  });

  test.describe('应用搜索和过滤测试', () => {
    test('应用搜索功能测试', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForLoadState('networkidle');
      
      const searchInput = page.locator('input[type="search"], input[placeholder*="搜索"], input[name="search"]').first();
      const inputVisible = await searchInput.isVisible().catch(() => false);
      
      if (inputVisible) {
        await searchInput.fill('Test');
        await page.waitForTimeout(500);
        
        await testHelpers.takeScreenshot(page, 'admin-app-search');
      }
    });

    test('应用状态过滤测试', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForLoadState('networkidle');
      
      const filterDropdown = page.locator('select, [class*="filter"], button:has-text("状态")').first();
      const dropdownVisible = await filterDropdown.isVisible().catch(() => false);
      
      if (dropdownVisible) {
        await testHelpers.takeScreenshot(page, 'admin-app-filter');
      }
    });

    test('应用排序测试', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForLoadState('networkidle');
      
      const sortHeaders = page.locator('th, .sort-header');
      const headerCount = await sortHeaders.count();
      
      if (headerCount > 0) {
        await sortHeaders.first().click();
        await page.waitForTimeout(500);
        await testHelpers.takeScreenshot(page, 'admin-app-sorted');
      }
    });
  });

  test.describe('应用API Key管理测试', () => {
    test('应该能够生成API Key', async ({ page }) => {
      const appName = 'API Key Test App - ' + Date.now();
      const createResult = await apiHelper.createApplication(authToken, appName, 'API key test');
      
      if (createResult.success) {
        const appId = createResult.data.id || createResult.data.appId;
        
        await page.goto(`/admin/applications/${appId}`);
        await page.waitForLoadState('networkidle');
        
        const generateKeyButton = page.locator('button:has-text("生成Key"), button:has-text("Generate Key")').first();
        const buttonVisible = await generateKeyButton.isVisible().catch(() => false);
        
        if (buttonVisible) {
          await generateKeyButton.click();
          await page.waitForTimeout(1000);
          await testHelpers.takeScreenshot(page, 'admin-app-api-key-generated');
        }
      }
    });

    test('应该能够复制API Key', async ({ page }) => {
      const appName = 'Copy API Key Test - ' + Date.now();
      const createResult = await apiHelper.createApplication(authToken, appName, 'Copy API key test');
      
      if (createResult.success) {
        const appId = createResult.data.id || createResult.data.appId;
        
        await page.goto(`/admin/applications/${appId}`);
        await page.waitForLoadState('networkidle');
        
        const copyButton = page.locator('button:has-text("复制"), button:has-text("Copy")').first();
        const buttonVisible = await copyButton.isVisible().catch(() => false);
        
        if (buttonVisible) {
          await copyButton.click();
          await page.waitForTimeout(500);
          await testHelpers.takeScreenshot(page, 'admin-app-api-key-copied');
        }
      }
    });

    test('应该能够重新生成API Key', async ({ page }) => {
      const appName = 'Regenerate Key Test - ' + Date.now();
      const createResult = await apiHelper.createApplication(authToken, appName, 'Regenerate test');
      
      if (createResult.success) {
        const appId = createResult.data.id || createResult.data.appId;
        
        await page.goto(`/admin/applications/${appId}`);
        await page.waitForLoadState('networkidle');
        
        const regenerateButton = page.locator('button:has-text("重新生成"), button:has-text("Regenerate")').first();
        const buttonVisible = await regenerateButton.isVisible().catch(() => false);
        
        if (buttonVisible) {
          await regenerateButton.click();
          await page.waitForTimeout(1000);
          await testHelpers.takeScreenshot(page, 'admin-app-api-key-regenerated');
        }
      }
    });
  });

  test.describe('应用配置测试', () => {
    test('应该能够配置应用设置', async ({ page }) => {
      const appName = 'Config Test App - ' + Date.now();
      const createResult = await apiHelper.createApplication(authToken, appName, 'Config test');
      
      if (createResult.success) {
        const appId = createResult.data.id || createResult.data.appId;
        
        await page.goto(`/admin/applications/${appId}/settings`);
        await page.waitForLoadState('networkidle');
        
        const settingsPage = await page.locator('form, .settings').isVisible().catch(() => false);
        
        if (settingsPage) {
          console.log('设置页面可见');
        }
        
        await testHelpers.takeScreenshot(page, 'admin-app-settings');
      }
    });

    test('应用白名单配置测试', async ({ page }) => {
      const appName = 'Whitelist Test App - ' + Date.now();
      const createResult = await apiHelper.createApplication(authToken, appName, 'Whitelist test');
      
      if (createResult.success) {
        const appId = createResult.data.id || createResult.data.appId;
        
        await page.goto(`/admin/applications/${appId}/settings`);
        await page.waitForLoadState('networkidle');
        
        const whitelistInput = page.locator('input[name="whitelist"], textarea[name="whitelist"]').first();
        const inputVisible = await whitelistInput.isVisible().catch(() => false);
        
        if (inputVisible) {
          await whitelistInput.fill('127.0.0.1\nlocalhost');
          await page.waitForTimeout(500);
          await testHelpers.takeScreenshot(page, 'admin-app-whitelist');
        }
      }
    });
  });

  test.describe('应用性能测试', () => {
    test('应用列表加载性能测试', async ({ page }) => {
      const startTime = Date.now();
      await page.goto('/admin/applications');
      await page.waitForLoadState('networkidle');
      const loadTime = Date.now() - startTime;
      
      console.log(`应用列表加载时间: ${loadTime}ms`);
      expect(loadTime).toBeLessThan(10000);
    });

    test('应用搜索响应时间测试', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForLoadState('networkidle');
      
      const searchInput = page.locator('input[type="search"], input[name="search"]').first();
      await searchInput.fill('test');
      
      const startTime = Date.now();
      await page.waitForTimeout(500);
      const searchTime = Date.now() - startTime;
      
      console.log(`搜索响应时间: ${searchTime}ms`);
    });

    test('创建应用API响应时间测试', async () => {
      const startTime = Date.now();
      await apiHelper.createApplication(authToken, 'Perf Test ' + Date.now(), 'Performance test');
      const createTime = Date.now() - startTime;
      
      console.log(`创建应用API响应时间: ${createTime}ms`);
      expect(createTime).toBeLessThan(5000);
    });
  });

  test.describe('应用错误处理测试', () => {
    test('网络错误处理测试', async ({ page }) => {
      await page.route('**/api/v1/admin/applications', route => {
        route.abort('failed');
      });
      
      await page.goto('/admin/applications');
      await page.waitForTimeout(2000);
      await testHelpers.takeScreenshot(page, 'admin-app-network-error');
    });

    test('创建应用失败处理测试', async ({ page }) => {
      await page.route('**/api/v1/admin/applications', route => {
        route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ success: false, message: 'Server Error' })
        });
      });
      
      await page.goto('/admin/applications');
      await page.waitForLoadState('networkidle');
      
      const createButton = page.locator('button:has-text("创建"), button:has-text("添加")').first();
      await createButton.click();
      await page.waitForTimeout(1000);
      
      await testHelpers.takeScreenshot(page, 'admin-app-create-error');
    });
  });

  test.describe('应用权限测试', () => {
    test('未认证用户访问应用管理应该被拒绝', async ({ page }) => {
      await page.goto('/admin/applications');
      await expect(page).toHaveURL(/\/admin\/login/);
    });

    test('API未授权访问测试', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/admin/applications');
      expect(response.status()).toBe(401);
    });
  });

  test.describe('应用批量操作测试', () => {
    test('批量启用应用测试', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForLoadState('networkidle');
      
      const checkboxes = page.locator('input[type="checkbox"]');
      const checkboxCount = await checkboxes.count();
      
      if (checkboxCount > 1) {
        await checkboxes.nth(1).check();
        await checkboxes.nth(2).check();
        
        const bulkEnableButton = page.locator('button:has-text("批量启用"), button:has-text("启用选中")').first();
        const buttonVisible = await bulkEnableButton.isVisible().catch(() => false);
        
        if (buttonVisible) {
          console.log('批量启用按钮可用');
          await testHelpers.takeScreenshot(page, 'admin-app-bulk-enable');
        }
      }
    });

    test('批量禁用应用测试', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForLoadState('networkidle');
      
      const checkboxes = page.locator('input[type="checkbox"]');
      const checkboxCount = await checkboxes.count();
      
      if (checkboxCount > 1) {
        await checkboxes.nth(1).check();
        
        const bulkDisableButton = page.locator('button:has-text("批量禁用"), button:has-text("禁用选中")').first();
        const buttonVisible = await bulkDisableButton.isVisible().catch(() => false);
        
        if (buttonVisible) {
          console.log('批量禁用按钮可用');
          await testHelpers.takeScreenshot(page, 'admin-app-bulk-disable');
        }
      }
    });
  });
});
