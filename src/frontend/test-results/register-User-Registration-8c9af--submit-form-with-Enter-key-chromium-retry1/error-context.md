# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: register.spec.js >> User Registration Flow >> Keyboard Navigation >> should submit form with Enter key
- Location: tests/e2e/register.spec.js:218:5

# Error details

```
Error: expect(received).toBeTruthy()

Received: false
```

# Page snapshot

```yaml
- generic [ref=e4]:
  - generic [ref=e5]:
    - heading "Create Account" [level=1] [ref=e6]
    - paragraph [ref=e7]: Join us and start exploring
  - alert "网络错误，请稍后重试" [ref=e8]:
    - generic [ref=e10]: 网络错误，请稍后重试
    - button "关闭提示" [ref=e11] [cursor=pointer]: ×
  - generic [ref=e12]:
    - generic [ref=e13]:
      - generic [ref=e14]: 用户名*
      - textbox "用户名*" [ref=e15]:
        - /placeholder: 请输入用户名
        - text: New User
    - generic [ref=e16]:
      - generic [ref=e17]: 邮箱*
      - textbox "邮箱*" [ref=e18]:
        - /placeholder: 请输入邮箱
        - text: newuser@example.com
    - generic [ref=e19]:
      - generic [ref=e20]: 密码*
      - textbox "密码*" [ref=e21]:
        - /placeholder: 请输入密码
        - text: ValidPassword123
    - generic [ref=e22]:
      - generic [ref=e23]: 确认密码*
      - textbox "确认密码*" [active] [ref=e24]:
        - /placeholder: 请再次输入密码
        - text: ValidPassword123
    - button "注册" [ref=e25] [cursor=pointer]
  - paragraph [ref=e27]:
    - text: Already have an account?
    - link "Sign In Now" [ref=e28] [cursor=pointer]:
      - /url: /login
```

# Test source

```ts
  128 | 
  129 |     test('should show validation error for password without number', async ({ page }) => {
  130 |       await page.fill('input[name="name"]', 'New User');
  131 |       await page.fill('input[name="email"]', 'newuser@example.com');
  132 |       await page.fill('input[name="password"]', 'ValidPassword');
  133 |       await page.fill('input[name="confirmPassword"]', 'ValidPassword');
  134 |       await page.click('button[type="submit"]');
  135 |       
  136 |       await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
  137 |       await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|数字|number/i);
  138 |     });
  139 | 
  140 |     test('should show validation error for password mismatch', async ({ page }) => {
  141 |       await page.fill('input[name="name"]', 'New User');
  142 |       await page.fill('input[name="email"]', 'newuser@example.com');
  143 |       await page.fill('input[name="password"]', 'ValidPassword123');
  144 |       await page.fill('input[name="confirmPassword"]', 'DifferentPassword123');
  145 |       await page.click('button[type="submit"]');
  146 |       
  147 |       await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
  148 |       await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|不一致|匹配/i);
  149 |     });
  150 | 
  151 |     test('should show multiple validation errors when all fields are empty', async ({ page }) => {
  152 |       await page.click('button[type="submit"]');
  153 |       
  154 |       await page.waitForTimeout(500);
  155 |       const errorMessages = page.locator('.error-message, [class*="error"]');
  156 |       const errorCount = await errorMessages.count();
  157 |       
  158 |       expect(errorCount).toBeGreaterThanOrEqual(3);
  159 |     });
  160 |   });
  161 | 
  162 |   test.describe('UI Elements', () => {
  163 |     test('should display all required form elements', async ({ page }) => {
  164 |       await expect(page.locator('h1, h2')).toContainText(/注册|register|sign up/i, { ignoreCase: true });
  165 |       await expect(page.locator('input[name="name"]')).toBeVisible();
  166 |       await expect(page.locator('input[name="email"]')).toBeVisible();
  167 |       await expect(page.locator('input[name="password"]')).toBeVisible();
  168 |       await expect(page.locator('input[name="confirmPassword"]')).toBeVisible();
  169 |       await expect(page.locator('button[type="submit"]')).toBeVisible();
  170 |     });
  171 | 
  172 |     test('should have correct input types for accessibility', async ({ page }) => {
  173 |       await expect(page.locator('input[name="email"]')).toHaveAttribute('type', 'email');
  174 |       await expect(page.locator('input[name="password"]')).toHaveAttribute('type', 'password');
  175 |       await expect(page.locator('input[name="confirmPassword"]')).toHaveAttribute('type', 'password');
  176 |     });
  177 | 
  178 |     test('should show loading state during submission', async ({ page }) => {
  179 |       await page.fill('input[name="name"]', 'New User');
  180 |       await page.fill('input[name="email"]', 'newuser@example.com');
  181 |       await page.fill('input[name="password"]', 'ValidPassword123');
  182 |       await page.fill('input[name="confirmPassword"]', 'ValidPassword123');
  183 |       
  184 |       const submitButton = page.locator('button[type="submit"]');
  185 |       await submitButton.click();
  186 |       
  187 |       await page.waitForTimeout(100);
  188 |       await expect(submitButton).toBeDisabled();
  189 |     });
  190 | 
  191 |     test('should navigate to login page', async ({ page }) => {
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
> 228 |       expect(isDisabled).toBeTruthy();
      |                          ^ Error: expect(received).toBeTruthy()
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
  292 |       expect(criticalErrors.length).toBe(0);
  293 |     });
  294 |   });
  295 | });
  296 | 
```