# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: login.spec.js >> User Login Flow >> Form Validation >> should show validation error for empty password
- Location: tests/e2e/login.spec.js:63:5

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.error-message, [class*="error"]').first()
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 5000ms
  - waiting for locator('.error-message, [class*="error"]').first()

```

```yaml
- heading "Welcome Back" [level=1]
- paragraph: Please login to your account
- text: 邮箱*
- textbox "邮箱*":
  - /placeholder: 请输入邮箱
  - text: test@example.com
- text: 密码*
- textbox "密码*":
  - /placeholder: 请输入密码
- button "登录"
- paragraph:
  - text: Don't have an account?
  - link "Sign Up Now":
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
  25  |     expect(currentUrl).not.toMatch(/\/login$/i);
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
> 67  |       await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
      |                                                                              ^ Error: expect(locator).toBeVisible() failed
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
  165 |       expect(await page.locator('input[name="password"]').inputValue()).toBe('');
  166 |     });
  167 | 
```