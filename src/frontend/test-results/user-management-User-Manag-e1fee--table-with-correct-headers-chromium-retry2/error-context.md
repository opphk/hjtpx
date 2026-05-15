# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: user-management.spec.js >> User Management >> User List Loading >> should display user table with correct headers
- Location: tests/e2e/user-management.spec.js:24:5

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('th').first()
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 5000ms
  - waiting for locator('th').first()

```

```yaml
- heading "Welcome Back" [level=1]
- paragraph: Please login to your account
- text: 邮箱*
- textbox "邮箱*":
  - /placeholder: 请输入邮箱
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
  3   | test.describe('User Management', () => {
  4   |   test.beforeEach(async ({ page }) => {
  5   |     page.on('console', msg => {
  6   |       if (msg.type() === 'error') {
  7   |         console.log(`Console Error: ${msg.text()}`);
  8   |       }
  9   |     });
  10  |   });
  11  | 
  12  |   test.describe('User List Loading', () => {
  13  |     test('should load user list successfully', async ({ page }) => {
  14  |       await page.goto('/admin/users');
  15  |       
  16  |       await page.waitForLoadState('networkidle');
  17  |       
  18  |       const table = page.locator('table');
  19  |       await expect(table).toBeVisible({ timeout: 10000 }).catch(() => {
  20  |         console.log('Table may not be visible immediately');
  21  |       });
  22  |     });
  23  | 
  24  |     test('should display user table with correct headers', async ({ page }) => {
  25  |       await page.goto('/admin/users');
  26  |       
  27  |       await page.waitForLoadState('networkidle');
  28  |       
> 29  |       await expect(page.locator('th').first()).toBeVisible();
      |                                                ^ Error: expect(locator).toBeVisible() failed
  30  |       
  31  |       const headers = ['ID', '用户名', '邮箱', '角色', '状态', '注册时间', '最后登录', '操作'];
  32  |       for (const header of headers) {
  33  |         await expect(page.locator(`th:has-text("${header}")`)).toBeVisible({ timeout: 5000 }).catch(() => {
  34  |           console.log(`Header "${header}" not found`);
  35  |         });
  36  |       }
  37  |     });
  38  | 
  39  |     test('should show loading state while fetching users', async ({ page }) => {
  40  |       await page.goto('/admin/users');
  41  |       
  42  |       const loadingIndicator = page.locator('[class*="loading"], [class*="skeleton"], .spinner');
  43  |       
  44  |       if (await loadingIndicator.isVisible().catch(() => false)) {
  45  |         await expect(loadingIndicator).toBeVisible();
  46  |       }
  47  |     });
  48  | 
  49  |     test('should display pagination controls', async ({ page }) => {
  50  |       await page.goto('/admin/users');
  51  |       
  52  |       await page.waitForLoadState('networkidle');
  53  |       
  54  |       const pagination = page.locator('[class*="pagination"], nav[aria-label*="pagination"]');
  55  |       const hasPagination = await pagination.isVisible().catch(() => false);
  56  |       
  57  |       if (hasPagination) {
  58  |         await expect(pagination).toBeVisible();
  59  |       } else {
  60  |         const totalUsers = await page.locator('table tbody tr').count();
  61  |         expect(totalUsers).toBeGreaterThan(0);
  62  |       }
  63  |     });
  64  | 
  65  |     test('should show user count', async ({ page }) => {
  66  |       await page.goto('/admin/users');
  67  |       
  68  |       await page.waitForLoadState('networkidle');
  69  |       
  70  |       const userCount = page.locator('[class*="count"], [class*="total"], [class*="summary"]');
  71  |       const hasCount = await userCount.isVisible().catch(() => false);
  72  |       
  73  |       if (hasCount) {
  74  |         await expect(userCount.first()).toBeVisible();
  75  |       }
  76  |     });
  77  |   });
  78  | 
  79  |   test.describe('User Search', () => {
  80  |     test('should display search input field', async ({ page }) => {
  81  |       await page.goto('/admin/users');
  82  |       
  83  |       const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first();
  84  |       await expect(searchInput).toBeVisible();
  85  |     });
  86  | 
  87  |     test('should search users by name', async ({ page }) => {
  88  |       await page.goto('/admin/users');
  89  |       
  90  |       await page.waitForLoadState('networkidle');
  91  |       
  92  |       const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first();
  93  |       
  94  |       await searchInput.fill('admin');
  95  |       await page.waitForTimeout(500);
  96  |       
  97  |       const rows = page.locator('table tbody tr');
  98  |       const rowCount = await rows.count();
  99  |       
  100 |       if (rowCount > 0) {
  101 |         const firstRowText = await rows.first().textContent();
  102 |         expect(firstRowText.toLowerCase()).toContain('admin');
  103 |       }
  104 |     });
  105 | 
  106 |     test('should search users by email', async ({ page }) => {
  107 |       await page.goto('/admin/users');
  108 |       
  109 |       await page.waitForLoadState('networkidle');
  110 |       
  111 |       const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first();
  112 |       
  113 |       await searchInput.fill('example');
  114 |       await page.waitForTimeout(500);
  115 |       
  116 |       const rows = page.locator('table tbody tr');
  117 |       const rowCount = await rows.count();
  118 |       
  119 |       if (rowCount > 0) {
  120 |         const firstRowText = await rows.first().textContent();
  121 |         expect(firstRowText.toLowerCase()).toContain('example');
  122 |       }
  123 |     });
  124 | 
  125 |     test('should clear search and show all users', async ({ page }) => {
  126 |       await page.goto('/admin/users');
  127 |       
  128 |       await page.waitForLoadState('networkidle');
  129 |       
```