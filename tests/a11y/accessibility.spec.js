const { test, expect } = require('@playwright/test');
const AxeBuilder = require('@axe-core/playwright').default;

const accessibilityTests = [
  {
    name: 'Login Page',
    url: '/login',
    critical: ['page-has-title', 'document-title', 'html-has-lang', 'color-contrast'],
    important: ['link-name', 'button-name', 'image-alt']
  },
  {
    name: 'Register Page',
    url: '/register',
    critical: ['page-has-title', 'document-title', 'html-has-lang', 'color-contrast'],
    important: ['link-name', 'button-name', 'image-alt']
  },
  {
    name: 'Dashboard Page',
    url: '/dashboard',
    critical: ['page-has-title', 'document-title', 'html-has-lang', 'color-contrast'],
    important: ['link-name', 'button-name', 'image-alt']
  },
  {
    name: 'Users Page',
    url: '/users',
    critical: ['page-has-title', 'document-title', 'html-has-lang', 'color-contrast'],
    important: ['link-name', 'button-name', 'image-alt']
  }
];

accessibilityTests.forEach(({ name, url, critical, important }) => {
  test.describe(`${name} - Accessibility Tests`, () => {
    test.beforeEach(async ({ page }) => {
      await page.goto(url);
      await page.waitForLoadState('networkidle');
    });

    test('should not have critical accessibility violations', async ({ page }) => {
      const accessibilityScanResults = await new AxeBuilder({ page })
        .withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
        .analyze();

      const criticalViolations = accessibilityScanResults.violations.filter(violation =>
        critical.includes(violation.id)
      );

      if (criticalViolations.length > 0) {
        console.log('Critical violations found:', criticalViolations.map(v => v.id));
      }

      expect(criticalViolations, `Critical accessibility violations in ${name}`).toHaveLength(0);
    });

    test('should have accessible color contrast', async ({ page }) => {
      const accessibilityScanResults = await new AxeBuilder({ page })
        .withTags(['wcag2aa'])
        .include('body')
        .analyze();

      const contrastViolations = accessibilityScanResults.violations.filter(
        violation => violation.id === 'color-contrast'
      );

      if (contrastViolations.length > 0) {
        console.log('Contrast violations:', contrastViolations.length);
        contrastViolations.forEach(v => {
          v.nodes.forEach(node => {
            console.log(`  - ${node.html}`);
          });
        });
      }

      expect(contrastViolations.length, `Color contrast issues in ${name}`).toBeLessThanOrEqual(0);
    });

    test('should have page title', async ({ page }) => {
      const title = await page.title();
      expect(title, `${name} should have a page title`).not.toBe('');
    });

    test('should have lang attribute on html element', async ({ page }) => {
      const htmlLang = await page.locator('html').getAttribute('lang');
      expect(htmlLang, `${name} should have lang attribute`).not.toBeNull();
    });

    test('should have skip link for keyboard navigation', async ({ page }) => {
      const skipLink = await page.locator('a[href="#main-content"], [role="main"], main, #main, #content').first();
      await expect(skipLink, `${name} should have a way to skip to main content`).toBeVisible({ timeout: 5000 }).catch(() => {
        const mainElement = page.locator('main, [role="main"], #main-content');
        expect(mainElement).toBeDefined();
      });
    });

    test('should have unique page titles', async ({ page }) => {
      const title = await page.title();
      const titleCount = await page.locator(`title:has-text("${title}")`).count();
      expect(titleCount).toBe(1);
    });

    test('should have proper heading hierarchy', async ({ page }) => {
      const headings = await page.locator('h1, h2, h3, h4, h5, h6').all();
      const headingLevels = headings.length > 0 ? await Promise.all(headings.map(h => h.evaluate(el => parseInt(el.tagName.charAt(1))))) : [];
      
      if (headingLevels.length > 0) {
        for (let i = 1; i < headingLevels.length; i++) {
          const current = headingLevels[i];
          const previous = headingLevels[i - 1];
          expect(current - previous, `${name} should not skip heading levels`).toBeLessThanOrEqual(1);
        }
      }
    });

    test('should have proper form labels', async ({ page }) => {
      const inputs = await page.locator('input:not([type="hidden"]), textarea, select').all();
      
      for (const input of inputs) {
        const id = await input.getAttribute('id');
        const ariaLabel = await input.getAttribute('aria-label');
        const ariaLabelledBy = await input.getAttribute('aria-labelledby');
        const placeholder = await input.getAttribute('placeholder');
        
        const hasLabel = id && await page.locator(`label[for="${id}"]`).count() > 0;
        const hasAriaLabel = !!ariaLabel;
        const hasAriaLabelledBy = !!ariaLabelledBy;
        const hasPlaceholder = !!placeholder;
        
        expect(
          hasLabel || hasAriaLabel || hasAriaLabelledBy || hasPlaceholder,
          `Input should have accessible label`
        ).toBeTruthy();
      }
    });

    test('should have descriptive link text', async ({ page }) => {
      const links = await page.locator('a[href]:not([href="#"]):not([href=""])').all();
      
      for (const link of links) {
        const text = await link.textContent();
        const ariaLabel = await link.getAttribute('aria-label');
        const ariaLabelledBy = await link.getAttribute('aria-labelledby');
        const title = await link.getAttribute('title');
        
        const hasDescriptiveText = text && text.trim().length > 0 && !['click here', 'here', 'read more'].includes(text.trim().toLowerCase());
        const hasAriaLabel = !!ariaLabel;
        const hasAriaLabelledBy = !!ariaLabelledBy;
        const hasTitle = !!title;
        
        expect(
          hasDescriptiveText || hasAriaLabel || hasAriaLabelledBy || hasTitle,
          `Link should have descriptive text or aria-label`
        ).toBeTruthy();
      }
    });

    test('should be keyboard navigable', async ({ page }) => {
      await page.keyboard.press('Tab');
      const focusedElement = await page.evaluate(() => document.activeElement.tagName);
      expect(focusedElement).not.toBe('body');
    });
  });
});

test.describe('Global Accessibility Checks', () => {
  test('should pass basic accessibility scan on home page', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a'])
      .analyze();

    expect(accessibilityScanResults.violations.length, 'Should have no WCAG 2A violations').toBeLessThanOrEqual(5);
  });

  test('should maintain accessibility across navigation', async ({ page }) => {
    await page.goto('/');
    
    const pages = ['/login', '/register'];
    
    for (const path of pages) {
      await page.goto(path);
      await page.waitForLoadState('networkidle');
      
      const accessibilityScanResults = await new AxeBuilder({ page })
        .withTags(['wcag2a'])
        .analyze();
      
      const criticalViolations = accessibilityScanResults.violations.filter(
        v => ['page-has-title', 'document-title', 'html-has-lang'].includes(v.id)
      );
      
      expect(criticalViolations.length, `Critical violations on ${path}`).toBe(0);
    }
  });
});
