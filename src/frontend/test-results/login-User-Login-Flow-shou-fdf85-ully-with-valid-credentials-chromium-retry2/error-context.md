# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: login.spec.js >> User Login Flow >> should login successfully with valid credentials
- Location: tests/e2e/login.spec.js:13:3

# Error details

```
Error: expect(received).not.toMatch(expected)

Expected pattern: not /\/login$/i
Received string:      "http://localhost:3001/login"
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
        - text: admin@example.com
    - generic [ref=e16]:
      - generic [ref=e17]: 密码*
      - textbox "密码*" [ref=e18]:
        - /placeholder: 请输入密码
        - text: Admin123!
    - button "登录" [ref=e19] [cursor=pointer]
  - paragraph [ref=e21]:
    - text: Don't have an account?
    - link "Sign Up Now" [ref=e22] [cursor=pointer]:
      - /url: /register
```

# Test source

```ts
  1   | import { test, expect } from '@playwright/test';
  2   | 
  3   | test.describe('User Login Flow', () => {
  4   |   test.beforeEach(async ({ page }) => {
  5   |     await page.goto('/login');
  6   |     page.on('console', msg => {
  7   |       if (msg.type() === 'error') {
  8   |         console.log(`Console Error: ${msg.text()}`);
  9   |       }
  10  |     });
  11  |   });
  12  | 
  13  |   test('should login successfully with valid credentials', async ({ page }) => {
  14  |     await page.goto('/login');
  15  | 
  16  |     await page.fill('input[name="email"]', 'admin@example.com');
  17  |     await page.fill('input[name="password"]', 'Admin123!');
  18  |     await page.click('button[type="submit"]');
  19  | 
  20  |     await page.waitForURL(/\/(dashboard|home)/i, { timeout: 15000 }).catch(() => {
  21  |       console.log('Redirect may not have occurred or login failed');
  22  |     });
  23  | 
  24  |     const currentUrl = page.url();
> 25  |     expect(currentUrl).not.toMatch(/\/login$/i);
      |                            ^ Error: expect(received).not.toMatch(expected)
  26  |   });
  27  | 
  28  |   test('should fail login with incorrect password', async ({ page }) => {
  29  |     await page.fill('input[name="email"]', 'admin@example.com');
  30  |     await page.fill('input[name="password"]', 'WrongPassword123');
  31  |     await page.click('button[type="submit"]');
  32  | 
  33  |     const errorVisible = await page.locator('.error, .alert, [role="alert"]').first().isVisible().catch(() => false);
  34  |     if (errorVisible) {
  35  |       await expect(page.locator('.error, .alert, [role="alert"]').first()).toBeVisible();
  36  |     } else {
  37  |       await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|错误|失败/i);
  38  |     }
  39  | 
  40  |     await expect(page.url()).toMatch(/\/login/i);
  41  |   });
  42  | 
  43  |   test('should fail login with non-existent email', async ({ page }) => {
  44  |     const timestamp = Date.now();
  45  |     await page.fill('input[name="email"]', `nonexistent${timestamp}@example.com`);
  46  |     await page.fill('input[name="password"]', 'SomePassword123');
  47  |     await page.click('button[type="submit"]');
  48  | 
  49  |     await page.waitForTimeout(1000);
  50  |     const errorVisible = await page.locator('.error, .alert, [role="alert"], .error-message').first().isVisible().catch(() => false);
  51  |     expect(errorVisible).toBeTruthy();
  52  |   });
  53  | 
  54  |   test.describe('Form Validation', () => {
  55  |     test('should show validation error for empty email', async ({ page }) => {
  56  |       await page.fill('input[name="password"]', 'TestPassword123');
  57  |       await page.click('button[type="submit"]');
  58  | 
  59  |       await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
  60  |       await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/邮箱|不能为空/i);
  61  |     });
  62  | 
  63  |     test('should show validation error for empty password', async ({ page }) => {
  64  |       await page.fill('input[name="email"]', 'test@example.com');
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
```