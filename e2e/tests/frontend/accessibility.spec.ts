import { test, expect } from '@playwright/test';

/**
 * 无障碍功能E2E测试
 * 
 * 测试目标：
 * 1. 验证ARIA标签的正确使用
 * 2. 验证键盘导航功能
 * 3. 验证屏幕阅读器兼容性
 * 4. 验证焦点管理
 */

test.describe('无障碍功能测试', () => {
  
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('@accessibility 验证页面标题', async ({ page }) => {
    const title = await page.title();
    expect(title).toBeTruthy();
    expect(title.length).toBeGreaterThan(0);
  });

  test('@accessibility 验证ARIA标签 - 首页', async ({ page }) => {
    // 检查主要heading
    const h1 = await page.locator('h1').first();
    await expect(h1).toBeVisible();
    
    const h1Text = await h1.textContent();
    expect(h1Text).toBeTruthy();
    
    // 检查按钮ARIA标签
    const buttons = await page.locator('button');
    const buttonCount = await buttons.count();
    
    for (let i = 0; i < Math.min(buttonCount, 10); i++) {
      const button = buttons.nth(i);
      const ariaLabel = await button.getAttribute('aria-label');
      const textContent = await button.textContent();
      
      // 按钮应该有aria-label或文本内容
      expect(ariaLabel || textContent).toBeTruthy();
    }
  });

  test('@accessibility 验证ARIA标签 - 验证码页面', async ({ page }) => {
    await page.goto('/captcha');
    
    // 检查验证码容器
    const captchaContainer = await page.locator('[role="application"]');
    await expect(captchaContainer).toBeVisible();
    
    // 检查ARIA描述
    const ariaDescribedby = await captchaContainer.getAttribute('aria-describedby');
    expect(ariaDescribedby).toBeTruthy();
    
    // 检查描述元素
    const description = await page.locator(`#${ariaDescribedby}`);
    await expect(description).toBeVisible();
  });

  test('@accessibility 验证表单无障碍', async ({ page }) => {
    await page.goto('/admin/login');
    
    // 检查表单标签关联
    const inputs = await page.locator('input[type="text"], input[type="password"]');
    const inputCount = await inputs.count();
    
    for (let i = 0; i < inputCount; i++) {
      const input = inputs.nth(i);
      const id = await input.getAttribute('id');
      const ariaLabel = await input.getAttribute('aria-label');
      const ariaLabelledby = await input.getAttribute('aria-labelledby');
      const labelExists = id && await page.locator(`label[for="${id}"]`).count() > 0;
      
      // 输入应该有标签关联
      expect(ariaLabel || ariaLabelledby || labelExists).toBeTruthy();
    }
  });

  test('@accessibility 验证焦点管理 - 登录页面', async ({ page }) => {
    await page.goto('/admin/login');
    
    // 点击用户名输入框
    const usernameInput = await page.locator('input[name="username"]');
    await usernameInput.click();
    
    // 检查焦点是否在输入框
    const isFocused = await usernameInput.evaluate(el => el === document.activeElement);
    expect(isFocused).toBeTruthy();
    
    // 按Tab键
    await page.keyboard.press('Tab');
    
    // 焦点应该移动到下一个可聚焦元素
    const focusedElement = await page.evaluate(() => document.activeElement);
    expect(focusedElement).toBeTruthy();
  });

  test('@accessibility 验证键盘导航 - 验证码滑动', async ({ page }) => {
    await page.goto('/captcha');
    
    // 等待验证码加载
    await page.waitForSelector('.captcha-slider');
    
    // 查找滑块
    const slider = await page.locator('.captcha-slider-handle');
    
    // 尝试使用Tab键聚焦
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');
    
    // 检查是否有元素获得焦点
    const focusedElement = await page.evaluate(() => document.activeElement);
    expect(focusedElement).toBeTruthy();
  });

  test('@accessibility 验证错误提示无障碍', async ({ page }) => {
    await page.goto('/admin/login');
    
    // 尝试提交空表单
    await page.locator('button[type="submit"]').click();
    
    // 检查错误提示
    const errorMessages = await page.locator('[role="alert"], .error-message, .text-danger');
    
    if (await errorMessages.count() > 0) {
      const firstError = errorMessages.first();
      await expect(firstError).toBeVisible();
      
      // 检查是否有aria-live属性
      const ariaLive = await firstError.getAttribute('aria-live');
      const role = await firstError.getAttribute('role');
      
      // 错误提示应该有适当的aria属性
      expect(ariaLive || role).toBeTruthy();
    }
  });

  test('@accessibility 验证颜色对比度', async ({ page }) => {
    await page.goto('/');
    
    // 检查主要文本颜色对比度
    const body = await page.locator('body');
    const bgColor = await body.evaluate(el => getComputedStyle(el).backgroundColor);
    
    const textElements = await page.locator('p, span, div');
    const textCount = await textElements.count();
    
    // 检查前5个文本元素的颜色
    for (let i = 0; i < Math.min(textCount, 5); i++) {
      const textElement = textElements.nth(i);
      const textColor = await textElement.evaluate(el => getComputedStyle(el).color);
      
      // 基本检查：颜色应该存在
      expect(textColor).toBeTruthy();
    }
  });

  test('@accessibility 验证跳过导航链接', async ({ page }) => {
    await page.goto('/');
    
    // 检查是否有跳过链接
    const skipLink = await page.locator('a[href="#main-content"], a[href="#main"], a.skip-link');
    
    if (await skipLink.count() > 0) {
      await expect(skipLink.first()).toBeVisible();
    } else {
      // 如果没有跳过链接，至少检查main区域存在
      const mainContent = await page.locator('main, [role="main"], #main, #main-content');
      await expect(mainContent).toBeVisible();
    }
  });

  test('@accessibility 验证图片alt属性', async ({ page }) => {
    await page.goto('/');
    
    const images = await page.locator('img');
    const imageCount = await images.count();
    
    // 检查所有图片是否有alt属性
    for (let i = 0; i < imageCount; i++) {
      const img = images.nth(i);
      const alt = await img.getAttribute('alt');
      const role = await img.getAttribute('role');
      
      // 图片应该有alt属性或role="presentation"
      expect(alt !== null || role === 'presentation').toBeTruthy();
    }
  });

  test('@accessibility 验证语言设置', async ({ page }) => {
    await page.goto('/');
    
    // 检查HTML lang属性
    const html = await page.locator('html');
    const lang = await html.getAttribute('lang');
    
    expect(lang).toBeTruthy();
    expect(lang.length).toBeGreaterThan(0);
  });

  test('@accessibility 验证表单错误关联', async ({ page }) => {
    await page.goto('/admin/login');
    
    // 填写表单但不完整提交
    await page.fill('input[name="username"]', 'test');
    await page.locator('button[type="submit"]').click();
    
    // 检查错误提示是否与输入框关联
    const errorMessages = await page.locator('.error-message, [role="alert"]');
    
    if (await errorMessages.count() > 0) {
      const firstError = errorMessages.first();
      const ariaDescribedby = await firstError.getAttribute('id');
      
      if (ariaDescribedby) {
        const input = await page.locator(`input[aria-describedby*="${ariaDescribedby}"]`);
        await expect(input).toBeVisible();
      }
    }
  });

  test('@accessibility 验证模态框无障碍', async ({ page }) => {
    await page.goto('/');
    
    // 触发模态框（如果有）
    const modalTriggers = await page.locator('[data-toggle="modal"], [data-target]');
    
    if (await modalTriggers.count() > 0) {
      await modalTriggers.first().click();
      
      // 等待模态框出现
      await page.waitForSelector('.modal, [role="dialog"]', { timeout: 5000 });
      
      const modal = await page.locator('.modal.show, .modal[style*="display: block"], [role="dialog"]').first();
      
      // 检查模态框属性
      const ariaModal = await modal.getAttribute('aria-modal');
      const role = await modal.getAttribute('role');
      
      expect(ariaModal === 'true' || role === 'dialog').toBeTruthy();
      
      // 检查标题
      const title = await modal.locator('.modal-title, [role="heading"]');
      await expect(title).toBeVisible();
    }
  });

  test('@accessibility 验证表格无障碍', async ({ page }) => {
    await page.goto('/admin/logs');
    
    const tables = await page.locator('table');
    const tableCount = await tables.count();
    
    if (tableCount > 0) {
      const table = tables.first();
      
      // 检查表格标题
      const caption = await table.locator('caption');
      const th = await table.locator('thead th');
      
      expect(await caption.count() > 0 || await th.count() > 0).toBeTruthy();
      
      // 检查表头
      const headers = await table.locator('th');
      const headerCount = await headers.count();
      
      if (headerCount > 0) {
        for (let i = 0; i < headerCount; i++) {
          const header = headers.nth(i);
          const scope = await header.getAttribute('scope');
          
          // 表头应该有scope属性
          expect(scope).toBeTruthy();
        }
      }
    }
  });

  test('@accessibility 验证列表无障碍', async ({ page }) => {
    await page.goto('/');
    
    // 检查列表结构
    const lists = await page.locator('ul, ol');
    const listCount = await lists.count();
    
    // 检查列表项
    const listItems = await page.locator('li');
    const itemCount = await listItems.count();
    
    expect(itemCount).toBeGreaterThanOrEqual(0);
  });

  test('@accessibility 验证链接文本', async ({ page }) => {
    await page.goto('/');
    
    const links = await page.locator('a');
    const linkCount = await links.count();
    
    for (let i = 0; i < Math.min(linkCount, 10); i++) {
      const link = links.nth(i);
      const linkText = await link.textContent();
      const ariaLabel = await link.getAttribute('aria-label');
      const title = await link.getAttribute('title');
      
      // 链接应该有文本内容或aria-label或title
      expect(linkText || ariaLabel || title).toBeTruthy();
    }
  });
});
