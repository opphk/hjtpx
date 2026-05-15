# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: login.spec.js >> User Login Flow >> Session Management >> should clear form after failed login
- Location: tests/e2e/login.spec.js:158:5

# Error details

```
Error: expect(received).toBe(expected) // Object.is equality

Expected: ""
Received: "WrongPassword123"
```

# Page snapshot

```yaml
- generic [ref=e4]:
  - generic [ref=e5]:
    - heading "Welcome Back" [level=1] [ref=e6]
    - paragraph [ref=e7]: Please login to your account
  - alert "网络错误，请稍后重试" [ref=e8]:
    - generic [ref=e10]: 网络错误，请稍后重试
    - button "关闭提示" [ref=e11] [cursor=pointer]: ×
  - generic [ref=e12]:
    - generic [ref=e13]:
      - generic [ref=e14]: 邮箱*
      - textbox "邮箱*" [ref=e15]:
        - /placeholder: 请输入邮箱
        - text: test@example.com
    - generic [ref=e16]:
      - generic [ref=e17]: 密码*
      - textbox "密码*" [ref=e18]:
        - /placeholder: 请输入密码
        - text: WrongPassword123
    - button "登录" [ref=e19] [cursor=pointer]
  - paragraph [ref=e21]:
    - text: Don't have an account?
    - link "Sign Up Now" [ref=e22] [cursor=pointer]:
      - /url: /register
```

# Test source

```ts
  65  |       await page.click('button[type="submit"]');
  66  | 
  67  |       await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
  68  |       await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|不能为空/i);
  69  |     });
  70  | 
  71  |     test('should show validation error for invalid email format', async ({ page }) => {
  72  |       await page.fill('input[name="email"]', 'invalid-email');
  73  |       await page.fill('input[name="password"]', 'TestPassword123');
  74  |       await page.click('button[type="submit"]');
  75  | 
  76  |       await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
  77  |       await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/邮箱|有效/i);
  78  |     });
  79  | 
  80  |     test('should show validation error for short password', async ({ page }) => {
  81  |       await page.fill('input[name="email"]', 'test@example.com');
  82  |       await page.fill('input[name="password"]', 'short');
  83  |       await page.click('button[type="submit"]');
  84  | 
  85  |       await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
  86  |       await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|8.*字符|至少/i);
  87  |     });
  88  | 
  89  |     test('should show multiple validation errors when all fields are empty', async ({ page }) => {
  90  |       await page.click('button[type="submit"]');
  91  | 
  92  |       await page.waitForTimeout(500);
  93  |       const errorMessages = page.locator('.error-message, [class*="error"]');
  94  |       const errorCount = await errorMessages.count();
  95  | 
  96  |       expect(errorCount).toBeGreaterThanOrEqual(2);
  97  |     });
  98  |   });
  99  | 
  100 |   test.describe('UI Elements', () => {
  101 |     test('should display all required form elements', async ({ page }) => {
  102 |       await expect(page.locator('h1, h2')).toContainText(/登录|welcome|sign in/i, { ignoreCase: true });
  103 |       await expect(page.locator('input[name="email"]')).toBeVisible();
  104 |       await expect(page.locator('input[name="password"]')).toBeVisible();
  105 |       await expect(page.locator('button[type="submit"]')).toBeVisible();
  106 |     });
  107 | 
  108 |     test('should have correct input types for accessibility', async ({ page }) => {
  109 |       await expect(page.locator('input[name="email"]')).toHaveAttribute('type', 'email');
  110 |       await expect(page.locator('input[name="password"]')).toHaveAttribute('type', 'password');
  111 |     });
  112 | 
  113 |     test('should show loading state during submission', async ({ page }) => {
  114 |       await page.fill('input[name="email"]', 'test@example.com');
  115 |       await page.fill('input[name="password"]', 'TestPassword123');
  116 | 
  117 |       const submitButton = page.locator('button[type="submit"]');
  118 |       await submitButton.click();
  119 | 
  120 |       await page.waitForTimeout(100);
  121 |       await expect(submitButton).toBeDisabled();
  122 |     });
  123 | 
  124 |     test('should navigate to registration page', async ({ page }) => {
  125 |       const registerLink = page.locator('a:has-text("注册"), a:has-text("register"), a[href="/register"]').first();
  126 |       if (await registerLink.isVisible().catch(() => false)) {
  127 |         await registerLink.click();
  128 |         await expect(page).toHaveURL(/\/register/i);
  129 |       }
  130 |     });
  131 |   });
  132 | 
  133 |   test.describe('Keyboard Navigation', () => {
  134 |     test('should support tab navigation between fields', async ({ page }) => {
  135 |       await page.locator('input[name="email"]').focus();
  136 |       await expect(page.locator('input[name="email"]')).toBeFocused();
  137 | 
  138 |       await page.keyboard.press('Tab');
  139 |       await expect(page.locator('input[name="password"]')).toBeFocused();
  140 | 
  141 |       await page.keyboard.press('Tab');
  142 |       await expect(page.locator('button[type="submit"]')).toBeFocused();
  143 |     });
  144 | 
  145 |     test('should submit form with Enter key', async ({ page }) => {
  146 |       await page.fill('input[name="email"]', 'test@example.com');
  147 |       await page.fill('input[name="password"]', 'TestPassword123');
  148 | 
  149 |       await page.keyboard.press('Enter');
  150 | 
  151 |       await page.waitForTimeout(100);
  152 |       const isDisabled = await page.locator('button[type="submit"]').isDisabled();
  153 |       expect(isDisabled).toBeTruthy();
  154 |     });
  155 |   });
  156 | 
  157 |   test.describe('Session Management', () => {
  158 |     test('should clear form after failed login', async ({ page }) => {
  159 |       await page.fill('input[name="email"]', 'test@example.com');
  160 |       await page.fill('input[name="password"]', 'WrongPassword123');
  161 |       await page.click('button[type="submit"]');
  162 | 
  163 |       await page.waitForTimeout(1000);
  164 |       expect(await page.locator('input[name="email"]').inputValue()).toBe('test@example.com');
> 165 |       expect(await page.locator('input[name="password"]').inputValue()).toBe('');
      |                                                                         ^ Error: expect(received).toBe(expected) // Object.is equality
  166 |     });
  167 | 
  168 |     test('should clear errors when user starts typing', async ({ page }) => {
  169 |       await page.click('button[type="submit"]');
  170 |       await page.waitForTimeout(300);
  171 | 
  172 |       const errorExists = await page.locator('.error-message, [class*="error"]').first().isVisible();
  173 |       if (errorExists) {
  174 |         await page.fill('input[name="email"]', 'test@example.com');
  175 |         await expect(page.locator('.error-message, [class*="error"]').first()).not.toBeVisible();
  176 |       }
  177 |     });
  178 |   });
  179 | 
  180 |   test.describe('Console Error Monitoring', () => {
  181 |     test('should not have console errors on page load', async ({ page }) => {
  182 |       const errors = [];
  183 |       page.on('console', msg => {
  184 |         if (msg.type() === 'error') {
  185 |           errors.push(msg.text());
  186 |         }
  187 |       });
  188 | 
  189 |       await page.reload();
  190 |       await page.waitForLoadState('networkidle');
  191 | 
  192 |       const criticalErrors = errors.filter(err =>
  193 |         !err.includes('favicon') &&
  194 |         !err.includes('DevTools') &&
  195 |         !err.includes('third-party')
  196 |       );
  197 | 
  198 |       expect(criticalErrors.length).toBe(0);
  199 |     });
  200 |   });
  201 | });
  202 | 
```