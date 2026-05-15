# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: register.spec.js >> User Registration Flow >> should register successfully with valid data
- Location: tests/e2e/register.spec.js:13:3

# Error details

```
Error: expect(received).not.toMatch(expected)

Expected pattern: not /\/register$/i
Received string:      "http://localhost:3001/register"
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
        - text: Test User 1778825348942
    - generic [ref=e16]:
      - generic [ref=e17]: 邮箱*
      - textbox "邮箱*" [ref=e18]:
        - /placeholder: 请输入邮箱
        - text: testuser1778825348942@example.com
    - generic [ref=e19]:
      - generic [ref=e20]: 密码*
      - textbox "密码*" [ref=e21]:
        - /placeholder: 请输入密码
        - text: ValidPassword123
    - generic [ref=e22]:
      - generic [ref=e23]: 确认密码*
      - textbox "确认密码*" [ref=e24]:
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
  1   | import { test, expect } from '@playwright/test';
  2   | 
  3   | test.describe('User Registration Flow', () => {
  4   |   test.beforeEach(async ({ page }) => {
  5   |     await page.goto('/register');
  6   |     page.on('console', msg => {
  7   |       if (msg.type() === 'error') {
  8   |         console.log(`Console Error: ${msg.text()}`);
  9   |       }
  10  |     });
  11  |   });
  12  | 
  13  |   test('should register successfully with valid data', async ({ page }) => {
  14  |     const timestamp = Date.now();
  15  |     const uniqueEmail = `testuser${timestamp}@example.com`;
  16  |     
  17  |     await page.fill('input[name="name"]', `Test User ${timestamp}`);
  18  |     await page.fill('input[name="email"]', uniqueEmail);
  19  |     await page.fill('input[name="password"]', 'ValidPassword123');
  20  |     await page.fill('input[name="confirmPassword"]', 'ValidPassword123');
  21  |     await page.click('button[type="submit"]');
  22  |     
  23  |     await page.waitForURL(/\/(login|dashboard|verify|success)/i, { timeout: 15000 }).catch(() => {
  24  |       console.log('Registration redirect may not have occurred');
  25  |     });
  26  |     
  27  |     const currentUrl = page.url();
> 28  |     expect(currentUrl).not.toMatch(/\/register$/i);
      |                            ^ Error: expect(received).not.toMatch(expected)
  29  |   });
  30  | 
  31  |   test('should fail registration with duplicate email', async ({ page }) => {
  32  |     await page.fill('input[name="name"]', 'Duplicate User');
  33  |     await page.fill('input[name="email"]', 'admin@example.com');
  34  |     await page.fill('input[name="password"]', 'ValidPassword123');
  35  |     await page.fill('input[name="confirmPassword"]', 'ValidPassword123');
  36  |     await page.click('button[type="submit"]');
  37  |     
  38  |     await page.waitForTimeout(2000);
  39  |     
  40  |     const errorVisible = await page.locator('.error, .alert, [role="alert"], .error-message').first().isVisible().catch(() => false);
  41  |     expect(errorVisible).toBeTruthy();
  42  |   });
  43  | 
  44  |   test.describe('Form Validation', () => {
  45  |     test('should show validation error for empty name field', async ({ page }) => {
  46  |       await page.fill('input[name="email"]', 'newuser@example.com');
  47  |       await page.fill('input[name="password"]', 'ValidPassword123');
  48  |       await page.fill('input[name="confirmPassword"]', 'ValidPassword123');
  49  |       await page.click('button[type="submit"]');
  50  |       
  51  |       await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
  52  |       await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/用户|不能为空/i);
  53  |     });
  54  | 
  55  |     test('should show validation error for short name', async ({ page }) => {
  56  |       await page.fill('input[name="name"]', 'a');
  57  |       await page.fill('input[name="email"]', 'newuser@example.com');
  58  |       await page.fill('input[name="password"]', 'ValidPassword123');
  59  |       await page.fill('input[name="confirmPassword"]', 'ValidPassword123');
  60  |       await page.click('button[type="submit"]');
  61  |       
  62  |       await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
  63  |       await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/用户|2.*字符|至少/i);
  64  |     });
  65  | 
  66  |     test('should show validation error for empty email field', async ({ page }) => {
  67  |       await page.fill('input[name="name"]', 'New User');
  68  |       await page.fill('input[name="password"]', 'ValidPassword123');
  69  |       await page.fill('input[name="confirmPassword"]', 'ValidPassword123');
  70  |       await page.click('button[type="submit"]');
  71  |       
  72  |       await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
  73  |       await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/邮箱|不能为空/i);
  74  |     });
  75  | 
  76  |     test('should show validation error for invalid email format', async ({ page }) => {
  77  |       await page.fill('input[name="name"]', 'New User');
  78  |       await page.fill('input[name="email"]', 'invalid-email');
  79  |       await page.fill('input[name="password"]', 'ValidPassword123');
  80  |       await page.fill('input[name="confirmPassword"]', 'ValidPassword123');
  81  |       await page.click('button[type="submit"]');
  82  |       
  83  |       await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
  84  |       await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/邮箱|有效/i);
  85  |     });
  86  | 
  87  |     test('should show validation error for empty password field', async ({ page }) => {
  88  |       await page.fill('input[name="name"]', 'New User');
  89  |       await page.fill('input[name="email"]', 'newuser@example.com');
  90  |       await page.click('button[type="submit"]');
  91  |       
  92  |       await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
  93  |       await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|不能为空/i);
  94  |     });
  95  | 
  96  |     test('should show validation error for short password', async ({ page }) => {
  97  |       await page.fill('input[name="name"]', 'New User');
  98  |       await page.fill('input[name="email"]', 'newuser@example.com');
  99  |       await page.fill('input[name="password"]', 'short');
  100 |       await page.fill('input[name="confirmPassword"]', 'short');
  101 |       await page.click('button[type="submit"]');
  102 |       
  103 |       await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
  104 |       await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|8.*字符|至少/i);
  105 |     });
  106 | 
  107 |     test('should show validation error for password without uppercase', async ({ page }) => {
  108 |       await page.fill('input[name="name"]', 'New User');
  109 |       await page.fill('input[name="email"]', 'newuser@example.com');
  110 |       await page.fill('input[name="password"]', 'validpassword123');
  111 |       await page.fill('input[name="confirmPassword"]', 'validpassword123');
  112 |       await page.click('button[type="submit"]');
  113 |       
  114 |       await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
  115 |       await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|大写|uppercase/i);
  116 |     });
  117 | 
  118 |     test('should show validation error for password without lowercase', async ({ page }) => {
  119 |       await page.fill('input[name="name"]', 'New User');
  120 |       await page.fill('input[name="email"]', 'newuser@example.com');
  121 |       await page.fill('input[name="password"]', 'VALIDPASSWORD123');
  122 |       await page.fill('input[name="confirmPassword"]', 'VALIDPASSWORD123');
  123 |       await page.click('button[type="submit"]');
  124 |       
  125 |       await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
  126 |       await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|小写|lowercase/i);
  127 |     });
  128 | 
```