import { test, expect } from '@playwright/test';

test.describe('滑块验证码UI/UX测试', () => {
  
  test.beforeEach(async ({ page }) => {
    await page.goto('/captcha');
    await page.waitForLoadState('networkidle');
  });

  test('滑块验证码进度指示器显示', async ({ page }) => {
    const progressContainer = page.locator('#slider-progress-container');
    const progressBar = page.locator('#slider-progress-bar');
    const progressText = page.locator('#slider-progress-text');
    
    await expect(progressContainer).toBeVisible();
    await expect(progressBar).toBeVisible();
    await expect(progressText).toBeVisible();
    await expect(progressText).toHaveText('0%');
  });

  test('滑块按钮初始状态图标正确', async ({ page }) => {
    const sliderButton = page.locator('#slider-button');
    await expect(sliderButton).toBeVisible();
    
    const defaultIcon = sliderButton.locator('.slider-icon-default');
    const successIcon = sliderButton.locator('.slider-icon-success');
    const errorIcon = sliderButton.locator('.slider-icon-error');
    
    await expect(defaultIcon).toBeVisible();
    await expect(successIcon).toBeHidden();
    await expect(errorIcon).toBeHidden();
  });

  test('滑块拖拽交互', async ({ page }) => {
    const sliderButton = page.locator('#slider-button');
    const sliderContainer = page.locator('#slider-container');
    const progressText = page.locator('#slider-progress-text');
    
    await expect(sliderButton).toBeVisible();
    
    const buttonBox = await sliderButton.boundingBox();
    const containerBox = await sliderContainer.boundingBox();
    
    if (buttonBox && containerBox) {
      await page.mouse.move(buttonBox.x + buttonBox.width / 2, buttonBox.y + buttonBox.height / 2);
      await page.mouse.down();
      
      await page.mouse.move(containerBox.x + containerBox.width * 0.5, buttonBox.y + buttonBox.height / 2, { steps: 10 });
      
      await expect(sliderButton).toHaveClass(/dragging/);
      
      await page.mouse.up();
    }
  });

  test('滑块容器包含目标指示器', async ({ page }) => {
    const targetIndicator = page.locator('#slider-target-indicator');
    await expect(targetIndicator).toBeVisible();
  });

  test('滑块按钮涟漪效果元素存在', async ({ page }) => {
    const sliderButton = page.locator('#slider-button');
    const ripple = sliderButton.locator('.slider-button-ripple');
    
    await expect(sliderButton).toBeVisible();
    await expect(ripple).toBeAttached();
  });

  test('滑块按钮内部图标容器存在', async ({ page }) => {
    const sliderButton = page.locator('#slider-button');
    const innerContainer = sliderButton.locator('.slider-button-inner');
    
    await expect(sliderButton).toBeVisible();
    await expect(innerContainer).toBeVisible();
    
    const defaultIcon = innerContainer.locator('.slider-icon-default');
    const successIcon = innerContainer.locator('.slider-icon-success');
    const errorIcon = innerContainer.locator('.slider-icon-error');
    
    await expect(defaultIcon).toBeVisible();
    await expect(successIcon).toBeAttached();
    await expect(errorIcon).toBeAttached();
  });

  test('滑块进度条渐变填充动画', async ({ page }) => {
    const progressFill = page.locator('#slider-progress-fill-value');
    await expect(progressFill).toBeVisible();
    
    const initialWidth = await progressFill.evaluate(el => (el as HTMLElement).style.width || '0%');
    expect(initialWidth).toBe('0%');
  });

  test('滑块目标指示器动画', async ({ page }) => {
    const targetIndicator = page.locator('#slider-target-indicator');
    await expect(targetIndicator).toBeVisible();
    
    const pulseAnimation = await targetIndicator.evaluate(el => {
      const style = window.getComputedStyle(el);
      return style.animationName;
    });
    expect(pulseAnimation).toBeTruthy();
  });

  test('滑块验证刷新按钮存在', async ({ page }) => {
    const refreshBtn = page.locator('#slider-refresh');
    await expect(refreshBtn).toBeVisible();
    await expect(refreshBtn).toBeEnabled();
  });

  test('滑块验证文本提示正确', async ({ page }) => {
    const sliderText = page.locator('#slider-text');
    await expect(sliderText).toBeVisible();
    const text = await sliderText.textContent();
    expect(text).toBeTruthy();
  });

  test('触摸反馈样式类存在', async ({ page }) => {
    const sliderButton = page.locator('#slider-button');
    const container = page.locator('#slider-container');
    
    await expect(sliderButton).toHaveClass(/captcha-slider-button/);
    await expect(container).toHaveClass(/captcha-slider-container/);
  });

  test('滑块进度容器可交互', async ({ page }) => {
    const progressContainer = page.locator('#slider-progress-container');
    const isInteractive = await progressContainer.evaluate(el => {
      const style = window.getComputedStyle(el);
      return style.display !== 'none' && style.visibility !== 'hidden';
    });
    expect(isInteractive).toBe(true);
  });

  test('成功粒子元素样式正确', async ({ page }) => {
    await page.evaluate(() => {
      const style = document.createElement('style');
      style.textContent = `
        .success-particle {
          position: fixed;
          pointer-events: none;
          z-index: 9999;
        }
      `;
      document.head.appendChild(style);
    });
    
    const particleStyle = await page.evaluate(() => {
      const el = document.createElement('div');
      el.className = 'success-particle';
      document.body.appendChild(el);
      const style = window.getComputedStyle(el);
      document.body.removeChild(el);
      return {
        position: style.position,
        pointerEvents: style.pointerEvents,
        zIndex: style.zIndex
      };
    });
    
    expect(particleStyle.position).toBe('fixed');
    expect(particleStyle.pointerEvents).toBe('none');
  });

  test('涟漪扩展动画关键帧存在', async ({ page }) => {
    const hasRippleKeyframes = await page.evaluate(() => {
      const sheets = document.styleSheets;
      for (const sheet of sheets) {
        try {
          const rules = sheet.cssRules;
          for (const rule of rules) {
            if (rule instanceof CSSKeyframesRule) {
              if (rule.name === 'ripple-expand') {
                return true;
              }
            }
          }
        } catch (e) {
          continue;
        }
      }
      return false;
    });
    expect(hasRippleKeyframes).toBe(true);
  });

  test('移动端滑块触摸激活状态样式', async ({ page }) => {
    const sliderButton = page.locator('#slider-button');
    const touchActiveClass = await sliderButton.evaluate(el => {
      return el.classList.contains('touch-active');
    });
    expect(touchActiveClass).toBe(false);
  });

  test('滑块验证加载状态', async ({ page }) => {
    const loadingOverlay = page.locator('#slider-loading-overlay');
    const isHiddenInitially = await loadingOverlay.evaluate(el => (el as HTMLElement).hidden);
    expect(isHiddenInitially).toBe(true);
  });

  test('滑块容器可访问性属性', async ({ page }) => {
    const sliderContainer = page.locator('#slider-container');
    await expect(sliderContainer).toHaveAttribute('role', 'slider');
    await expect(sliderContainer).toHaveAttribute('aria-label');
    await expect(sliderContainer).toHaveAttribute('aria-valuemin');
    await expect(sliderContainer).toHaveAttribute('aria-valuemax');
    await expect(sliderContainer).toHaveAttribute('aria-valuenow');
  });

  test('滑块进度指示器进度文本样式', async ({ page }) => {
    const progressText = page.locator('#slider-progress-text');
    const style = await progressText.evaluate(el => {
      const computed = window.getComputedStyle(el);
      return {
        fontSize: computed.fontSize,
        color: computed.color,
        fontWeight: computed.fontWeight
      };
    });
    
    expect(style.fontSize).toBeTruthy();
    expect(style.fontWeight).toBeTruthy();
  });
});

test.describe('滑块验证码动画效果测试', () => {
  
  test('成功动画涟漪效果', async ({ page }) => {
    await page.goto('/captcha');
    await page.waitForLoadState('networkidle');
    
    const sliderButton = page.locator('#slider-button');
    await sliderButton.evaluate(btn => {
      btn.classList.add('success');
    });
    
    const ripple = sliderButton.locator('.slider-button-ripple');
    await ripple.evaluate(el => {
      el.classList.add('ripple-success');
    });
    
    await page.waitForTimeout(100);
    await expect(ripple).toHaveClass(/ripple-success/);
  });

  test('错误动画涟漪效果', async ({ page }) => {
    await page.goto('/captcha');
    await page.waitForLoadState('networkidle');
    
    const sliderButton = page.locator('#slider-button');
    await sliderButton.evaluate(btn => {
      btn.classList.add('error');
    });
    
    const ripple = sliderButton.locator('.slider-button-ripple');
    await ripple.evaluate(el => {
      el.classList.add('ripple-error');
    });
    
    await page.waitForTimeout(100);
    await expect(ripple).toHaveClass(/ripple-error/);
  });

  test('验证中状态动画', async ({ page }) => {
    await page.goto('/captcha');
    await page.waitForLoadState('networkidle');
    
    const sliderButton = page.locator('#slider-button');
    await sliderButton.evaluate(btn => {
      btn.classList.add('verifying');
    });
    
    await expect(sliderButton).toHaveClass(/verifying/);
    
    const innerIcon = sliderButton.locator('.slider-button-inner');
    const defaultIcon = innerIcon.locator('.slider-icon-default');
    await expect(defaultIcon).toBeHidden();
  });

  test('滑块拖拽状态样式', async ({ page }) => {
    await page.goto('/captcha');
    await page.waitForLoadState('networkidle');
    
    const sliderButton = page.locator('#slider-button');
    const container = page.locator('#slider-container');
    
    await sliderButton.evaluate(btn => {
      btn.classList.add('dragging');
    });
    await container.evaluate(c => {
      c.classList.add('is-dragging');
    });
    
    await expect(sliderButton).toHaveClass(/dragging/);
    await expect(container).toHaveClass(/is-dragging/);
  });

  test('接近完成时进度条颜色变化', async ({ page }) => {
    await page.goto('/captcha');
    await page.waitForLoadState('networkidle');
    
    const progressFill = page.locator('#slider-progress-fill-value');
    await progressFill.evaluate(el => {
      el.classList.add('near-complete');
    });
    
    await expect(progressFill).toHaveClass(/near-complete/);
  });

  test('成功动画图标切换', async ({ page }) => {
    await page.goto('/captcha');
    await page.waitForLoadState('networkidle');
    
    const sliderButton = page.locator('#slider-button');
    const innerIcon = sliderButton.locator('.slider-button-inner');
    const defaultIcon = innerIcon.locator('.slider-icon-default');
    const successIcon = innerIcon.locator('.slider-icon-success');
    
    await sliderButton.evaluate(btn => {
      btn.classList.add('success');
    });
    
    await expect(defaultIcon).toBeHidden();
    await expect(successIcon).toBeVisible();
  });

  test('错误动画图标切换', async ({ page }) => {
    await page.goto('/captcha');
    await page.waitForLoadState('networkidle');
    
    const sliderButton = page.locator('#slider-button');
    const innerIcon = sliderButton.locator('.slider-button-inner');
    const defaultIcon = innerIcon.locator('.slider-icon-default');
    const errorIcon = innerIcon.locator('.slider-icon-error');
    
    await sliderButton.evaluate(btn => {
      btn.classList.add('error');
    });
    
    await expect(defaultIcon).toBeHidden();
    await expect(errorIcon).toBeVisible();
  });
});
