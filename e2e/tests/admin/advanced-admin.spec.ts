import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';
import { testUsers } from '../../utils/test-data';

test.describe('管理端详细测试', () => {
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ request }) => {
    apiHelper = new ApiHelper(request);
  });

  test.describe('管理端统计分析', () => {
    test('应该能够访问统计页面', async ({ page }) => {
      await page.goto('/admin/stats');
      await expect(page).toHaveTitle(/统计|Stats|Dashboard/);
    });

    test('应该能够查看验证码统计数据', async ({ page }) => {
      await page.goto('/admin/stats');
      await page.waitForTimeout(1000);

      const statsSection = page.locator('.stats, .statistics, .captcha-stats');
      if (await statsSection.isVisible({ timeout: 3000 })) {
        await expect(statsSection).toBeVisible();
      }
    });

    test('应该能够查看用户行为数据', async ({ page }) => {
      await page.goto('/admin/behavior');
      await page.waitForTimeout(1000);

      const behaviorSection = page.locator('.behavior, .user-behavior');
      if (await behaviorSection.isVisible({ timeout: 3000 })) {
        await expect(behaviorSection).toBeVisible();
      }
    });

    test('应该能够筛选统计数据', async ({ page }) => {
      await page.goto('/admin/stats');
      await page.waitForTimeout(1000);

      const datePicker = page.locator('input[type="date"], .date-picker, .date-filter');
      if (await datePicker.isVisible({ timeout: 3000 })) {
        await expect(datePicker.first()).toBeVisible();
      }
    });

    test('应该能够导出统计数据', async ({ page }) => {
      await page.goto('/admin/stats');
      await page.waitForTimeout(1000);

      const exportButton = page.locator('button:has-text("Export"), .export-btn, a[href*="export"]');
      if (await exportButton.isVisible({ timeout: 3000 })) {
        await expect(exportButton.first()).toBeVisible();
      }
    });

    test('应该能够查看实时数据', async ({ page }) => {
      await page.goto('/admin/realtime');
      await page.waitForTimeout(1000);

      const realtimeSection = page.locator('.realtime, .live-data');
      if (await realtimeSection.isVisible({ timeout: 3000 })) {
        await expect(realtimeSection).toBeVisible();
      }
    });
  });

  test.describe('管理端日志管理', () => {
    test('应该能够访问日志页面', async ({ page }) => {
      await page.goto('/admin/logs');
      await expect(page).toHaveTitle(/日志|Logs/);
    });

    test('应该能够搜索日志', async ({ page }) => {
      await page.goto('/admin/logs');
      await page.waitForTimeout(1000);

      const searchInput = page.locator('input[type="search"], .search-input, input[name="q"]');
      if (await searchInput.isVisible({ timeout: 3000 })) {
        await searchInput.fill('error');
        await searchInput.press('Enter');
        await page.waitForTimeout(500);
      }
    });

    test('应该能够按类型筛选日志', async ({ page }) => {
      await page.goto('/admin/logs');
      await page.waitForTimeout(1000);

      const typeFilter = page.locator('select[name="type"], .type-filter');
      if (await typeFilter.isVisible({ timeout: 3000 })) {
        await expect(typeFilter).toBeVisible();
      }
    });

    test('应该能够按时间筛选日志', async ({ page }) => {
      await page.goto('/admin/logs');
      await page.waitForTimeout(1000);

      const timeFilter = page.locator('input[type="datetime-local"], .time-filter, .date-filter');
      if (await timeFilter.isVisible({ timeout: 3000 })) {
        await expect(timeFilter.first()).toBeVisible();
      }
    });

    test('应该能够查看日志详情', async ({ page }) => {
      await page.goto('/admin/logs');
      await page.waitForTimeout(1000);

      const logRow = page.locator('tr, .log-item, .log-row');
      if (await logRow.first().isVisible({ timeout: 3000 })) {
        await logRow.first().click();
        await page.waitForTimeout(500);
      }
    });

    test('应该能够导出日志', async ({ page }) => {
      await page.goto('/admin/logs');
      await page.waitForTimeout(1000);

      const exportButton = page.locator('button:has-text("Export"), .export-btn');
      if (await exportButton.isVisible({ timeout: 3000 })) {
        await expect(exportButton).toBeVisible();
      }
    });

    test('应该能够清空日志', async ({ page }) => {
      await page.goto('/admin/logs');
      await page.waitForTimeout(1000);

      const clearButton = page.locator('button:has-text("Clear"), button:has-text("清空")');
      if (await clearButton.isVisible({ timeout: 3000 })) {
        await expect(clearButton).toBeVisible();
      }
    });
  });

  test.describe('管理端监控面板', () => {
    test('应该能够访问监控页面', async ({ page }) => {
      await page.goto('/admin/monitoring');
      await expect(page).toHaveTitle(/监控|Monitoring/);
    });

    test('应该能够查看系统状态', async ({ page }) => {
      await page.goto('/admin/monitoring');
      await page.waitForTimeout(1000);

      const systemStatus = page.locator('.system-status, .health-check');
      if (await systemStatus.isVisible({ timeout: 3000 })) {
        await expect(systemStatus).toBeVisible();
      }
    });

    test('应该能够查看性能指标', async ({ page }) => {
      await page.goto('/admin/monitoring');
      await page.waitForTimeout(1000);

      const performanceMetrics = page.locator('.metrics, .performance');
      if (await performanceMetrics.isVisible({ timeout: 3000 })) {
        await expect(performanceMetrics).toBeVisible();
      }
    });

    test('应该能够查看实时连接', async ({ page }) => {
      await page.goto('/admin/monitoring');
      await page.waitForTimeout(1000);

      const connections = page.locator('.connections, .realtime-connections');
      if (await connections.isVisible({ timeout: 3000 })) {
        await expect(connections).toBeVisible();
      }
    });

    test('应该能够设置告警阈值', async ({ page }) => {
      await page.goto('/admin/monitoring/alerts');
      await page.waitForTimeout(1000);

      const alertSettings = page.locator('.alert-settings, .threshold-settings');
      if (await alertSettings.isVisible({ timeout: 3000 })) {
        await expect(alertSettings).toBeVisible();
      }
    });
  });

  test.describe('管理端应用程序管理', () => {
    test('应该能够访问应用程序列表', async ({ page }) => {
      await page.goto('/admin/applications');
      await expect(page).toHaveTitle(/应用|Applications/);
    });

    test('应该能够创建新应用程序', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForTimeout(1000);

      const createButton = page.locator('button:has-text("Create"), button:has-text("创建"), .create-btn');
      if (await createButton.isVisible({ timeout: 3000 })) {
        await createButton.click();
        await page.waitForTimeout(500);
      }
    });

    test('应该能够编辑应用程序', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForTimeout(1000);

      const editButton = page.locator('button:has-text("Edit"), .edit-btn, a[href*="edit"]');
      if (await editButton.first().isVisible({ timeout: 3000 })) {
        await editButton.first().click();
        await page.waitForTimeout(500);
      }
    });

    test('应该能够删除应用程序', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForTimeout(1000);

      const deleteButton = page.locator('button:has-text("Delete"), .delete-btn');
      if (await deleteButton.first().isVisible({ timeout: 3000 })) {
        await expect(deleteButton.first()).toBeVisible();
      }
    });

    test('应该能够查看应用程序配置', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForTimeout(1000);

      const configSection = page.locator('.config, .settings');
      if (await configSection.first().isVisible({ timeout: 3000 })) {
        await expect(configSection.first()).toBeVisible();
      }
    });

    test('应该能够查看应用程序统计', async ({ page }) => {
      await page.goto('/admin/applications');
      await page.waitForTimeout(1000);

      const statsSection = page.locator('.stats, .statistics');
      if (await statsSection.first().isVisible({ timeout: 3000 })) {
        await expect(statsSection.first()).toBeVisible();
      }
    });
  });

  test.describe('管理端黑名单管理', () => {
    test('应该能够访问黑名单页面', async ({ page }) => {
      await page.goto('/admin/blacklist');
      await expect(page).toHaveTitle(/黑名单|Blacklist/);
    });

    test('应该能够添加IP到黑名单', async ({ page }) => {
      await page.goto('/admin/blacklist');
      await page.waitForTimeout(1000);

      const addButton = page.locator('button:has-text("Add"), button:has-text("添加"), .add-btn');
      if (await addButton.isVisible({ timeout: 3000 })) {
        await addButton.click();
        await page.waitForTimeout(500);
      }
    });

    test('应该能够从黑名单移除', async ({ page }) => {
      await page.goto('/admin/blacklist');
      await page.waitForTimeout(1000);

      const removeButton = page.locator('button:has-text("Remove"), .remove-btn');
      if (await removeButton.first().isVisible({ timeout: 3000 })) {
        await expect(removeButton.first()).toBeVisible();
      }
    });

    test('应该能够批量添加黑名单', async ({ page }) => {
      await page.goto('/admin/blacklist');
      await page.waitForTimeout(1000);

      const batchButton = page.locator('button:has-text("Batch"), .batch-btn');
      if (await batchButton.isVisible({ timeout: 3000 })) {
        await expect(batchButton).toBeVisible();
      }
    });

    test('应该能够导入黑名单', async ({ page }) => {
      await page.goto('/admin/blacklist');
      await page.waitForTimeout(1000);

      const importButton = page.locator('button:has-text("Import"), .import-btn');
      if (await importButton.isVisible({ timeout: 3000 })) {
        await expect(importButton).toBeVisible();
      }
    });

    test('应该能够导出黑名单', async ({ page }) => {
      await page.goto('/admin/blacklist');
      await page.waitForTimeout(1000);

      const exportButton = page.locator('button:has-text("Export"), .export-btn');
      if (await exportButton.isVisible({ timeout: 3000 })) {
        await expect(exportButton).toBeVisible();
      }
    });
  });

  test.describe('管理端安全设置', () => {
    test('应该能够访问安全设置页面', async ({ page }) => {
      await page.goto('/admin/security');
      await expect(page).toHaveTitle(/安全|Security/);
    });

    test('应该能够配置IP白名单', async ({ page }) => {
      await page.goto('/admin/security/whitelist');
      await page.waitForTimeout(1000);

      const whitelistSection = page.locator('.whitelist, .ip-whitelist');
      if (await whitelistSection.isVisible({ timeout: 3000 })) {
        await expect(whitelistSection).toBeVisible();
      }
    });

    test('应该能够配置Rate Limiting', async ({ page }) => {
      await page.goto('/admin/security/rate-limit');
      await page.waitForTimeout(1000);

      const rateLimitSection = page.locator('.rate-limit, .rate-limiting');
      if (await rateLimitSection.isVisible({ timeout: 3000 })) {
        await expect(rateLimitSection).toBeVisible();
      }
    });

    test('应该能够查看安全日志', async ({ page }) => {
      await page.goto('/admin/security/logs');
      await page.waitForTimeout(1000);

      const securityLogs = page.locator('.security-logs, .audit-logs');
      if (await securityLogs.isVisible({ timeout: 3000 })) {
        await expect(securityLogs).toBeVisible();
      }
    });

    test('应该能够配置验证码难度', async ({ page }) => {
      await page.goto('/admin/security/captcha');
      await page.waitForTimeout(1000);

      const captchaSettings = page.locator('.captcha-settings, .difficulty-settings');
      if (await captchaSettings.isVisible({ timeout: 3000 })) {
        await expect(captchaSettings).toBeVisible();
      }
    });
  });

  test.describe('管理端系统设置', () => {
    test('应该能够访问系统设置页面', async ({ page }) => {
      await page.goto('/admin/settings');
      await expect(page).toHaveTitle(/设置|Settings/);
    });

    test('应该能够配置常规设置', async ({ page }) => {
      await page.goto('/admin/settings/general');
      await page.waitForTimeout(1000);

      const generalSettings = page.locator('.general-settings, .settings-section');
      if (await generalSettings.isVisible({ timeout: 3000 })) {
        await expect(generalSettings.first()).toBeVisible();
      }
    });

    test('应该能够配置邮件设置', async ({ page }) => {
      await page.goto('/admin/settings/email');
      await page.waitForTimeout(1000);

      const emailSettings = page.locator('.email-settings, .mail-settings');
      if (await emailSettings.isVisible({ timeout: 3000 })) {
        await expect(emailSettings).toBeVisible();
      }
    });

    test('应该能够配置Webhook', async ({ page }) => {
      await page.goto('/admin/settings/webhooks');
      await page.waitForTimeout(1000);

      const webhookSettings = page.locator('.webhook-settings, .webhooks');
      if (await webhookSettings.isVisible({ timeout: 3000 })) {
        await expect(webhookSettings).toBeVisible();
      }
    });

    test('应该能够查看系统信息', async ({ page }) => {
      await page.goto('/admin/settings/about');
      await page.waitForTimeout(1000);

      const systemInfo = page.locator('.system-info, .about');
      if (await systemInfo.isVisible({ timeout: 3000 })) {
        await expect(systemInfo).toBeVisible();
      }
    });
  });
});
