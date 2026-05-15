# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: register.spec.js >> User Registration Flow >> UI Elements >> should display all required form elements
- Location: tests/e2e/register.spec.js:163:5

# Error details

```
Error: expect(locator).toContainText(expected) failed

Locator: locator('h1, h2')
Expected pattern: /注册|register|sign up/i
Received string:  "Create Account"
Timeout: 5000ms

Call log:
  - Expect "toContainText" with timeout 5000ms
  - waiting for locator('h1, h2')
    13 × locator resolved to <h1>Create Account</h1>
       - unexpected value "Create Account"

```

```yaml
- heading "Create Account" [level=1]
```

# Test source

```ts
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
> 164 |       await expect(page.locator('h1, h2')).toContainText(/注册|register|sign up/i, { ignoreCase: true });
      |                                            ^ Error: expect(locator).toContainText(expected) failed
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
```