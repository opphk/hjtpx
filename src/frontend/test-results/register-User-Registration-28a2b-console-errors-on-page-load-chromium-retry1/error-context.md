# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: register.spec.js >> User Registration Flow >> Console Error Monitoring >> should not have console errors on page load
- Location: tests/e2e/register.spec.js:275:5

# Error details

```
Error: expect(received).toBe(expected) // Object.is equality

Expected: 0
Received: 4
```

# Page snapshot

```yaml
- generic [ref=e4]:
  - generic [ref=e5]:
    - heading "Create Account" [level=1] [ref=e6]
    - paragraph [ref=e7]: Join us and start exploring
  - generic [ref=e8]:
    - generic [ref=e9]:
      - generic [ref=e10]: 用户名*
      - textbox "用户名*" [ref=e11]:
        - /placeholder: 请输入用户名
    - generic [ref=e12]:
      - generic [ref=e13]: 邮箱*
      - textbox "邮箱*" [ref=e14]:
        - /placeholder: 请输入邮箱
    - generic [ref=e15]:
      - generic [ref=e16]: 密码*
      - textbox "密码*" [ref=e17]:
        - /placeholder: 请输入密码
    - generic [ref=e18]:
      - generic [ref=e19]: 确认密码*
      - textbox "确认密码*" [ref=e20]:
        - /placeholder: 请再次输入密码
    - button "注册" [ref=e21] [cursor=pointer]
  - paragraph [ref=e23]:
    - text: Already have an account?
    - link "Sign In Now" [ref=e24] [cursor=pointer]:
      - /url: /login
```

# Test source

```ts
  192 |       const loginLink = page.locator('a:has-text("登录"), a:has-text("login"), a[href="/login"]').first();
  193 |       if (await loginLink.isVisible().catch(() => false)) {
  194 |         await loginLink.click();
  195 |         await expect(page).toHaveURL(/\/login/i);
  196 |       }
  197 |     });
  198 |   });
  199 | 
  200 |   test.describe('Keyboard Navigation', () => {
  201 |     test('should support tab navigation between fields', async ({ page }) => {
  202 |       await page.locator('input[name="name"]').focus();
  203 |       await expect(page.locator('input[name="name"]')).toBeFocused();
  204 |       
  205 |       await page.keyboard.press('Tab');
  206 |       await expect(page.locator('input[name="email"]')).toBeFocused();
  207 |       
  208 |       await page.keyboard.press('Tab');
  209 |       await expect(page.locator('input[name="password"]')).toBeFocused();
  210 |       
  211 |       await page.keyboard.press('Tab');
  212 |       await expect(page.locator('input[name="confirmPassword"]')).toBeFocused();
  213 |       
  214 |       await page.keyboard.press('Tab');
  215 |       await expect(page.locator('button[type="submit"]')).toBeFocused();
  216 |     });
  217 | 
  218 |     test('should submit form with Enter key', async ({ page }) => {
  219 |       await page.fill('input[name="name"]', 'New User');
  220 |       await page.fill('input[name="email"]', 'newuser@example.com');
  221 |       await page.fill('input[name="password"]', 'ValidPassword123');
  222 |       await page.fill('input[name="confirmPassword"]', 'ValidPassword123');
  223 |       
  224 |       await page.keyboard.press('Enter');
  225 |       
  226 |       await page.waitForTimeout(100);
  227 |       const isDisabled = await page.locator('button[type="submit"]').isDisabled();
  228 |       expect(isDisabled).toBeTruthy();
  229 |     });
  230 |   });
  231 | 
  232 |   test.describe('Form Data Preservation', () => {
  233 |     test('should preserve name and email on validation error', async ({ page }) => {
  234 |       const testName = 'Valid User';
  235 |       const testEmail = 'valid@example.com';
  236 |       
  237 |       await page.fill('input[name="name"]', testName);
  238 |       await page.fill('input[name="email"]', testEmail);
  239 |       await page.fill('input[name="password"]', 'short');
  240 |       await page.fill('input[name="confirmPassword"]', 'short');
  241 |       await page.click('button[type="submit"]');
  242 |       
  243 |       await page.waitForTimeout(300);
  244 |       
  245 |       expect(await page.locator('input[name="name"]').inputValue()).toBe(testName);
  246 |       expect(await page.locator('input[name="email"]').inputValue()).toBe(testEmail);
  247 |     });
  248 | 
  249 |     test('should clear password fields on validation error', async ({ page }) => {
  250 |       await page.fill('input[name="name"]', 'Valid User');
  251 |       await page.fill('input[name="email"]', 'valid@example.com');
  252 |       await page.fill('input[name="password"]', 'short');
  253 |       await page.fill('input[name="confirmPassword"]', 'short');
  254 |       await page.click('button[type="submit"]');
  255 |       
  256 |       await page.waitForTimeout(300);
  257 |       
  258 |       expect(await page.locator('input[name="password"]').inputValue()).toBe('');
  259 |       expect(await page.locator('input[name="confirmPassword"]').inputValue()).toBe('');
  260 |     });
  261 | 
  262 |     test('should clear errors when user starts typing', async ({ page }) => {
  263 |       await page.click('button[type="submit"]');
  264 |       await page.waitForTimeout(300);
  265 |       
  266 |       const errorExists = await page.locator('.error-message, [class*="error"]').first().isVisible();
  267 |       if (errorExists) {
  268 |         await page.fill('input[name="name"]', 'New User');
  269 |         await expect(page.locator('.error-message, [class*="error"]').first()).not.toBeVisible();
  270 |       }
  271 |     });
  272 |   });
  273 | 
  274 |   test.describe('Console Error Monitoring', () => {
  275 |     test('should not have console errors on page load', async ({ page }) => {
  276 |       const errors = [];
  277 |       page.on('console', msg => {
  278 |         if (msg.type() === 'error') {
  279 |           errors.push(msg.text());
  280 |         }
  281 |       });
  282 |       
  283 |       await page.reload();
  284 |       await page.waitForLoadState('networkidle');
  285 |       
  286 |       const criticalErrors = errors.filter(err => 
  287 |         !err.includes('favicon') && 
  288 |         !err.includes('DevTools') &&
  289 |         !err.includes('third-party')
  290 |       );
  291 |       
> 292 |       expect(criticalErrors.length).toBe(0);
      |                                     ^ Error: expect(received).toBe(expected) // Object.is equality
  293 |     });
  294 |   });
  295 | });
  296 | 
```