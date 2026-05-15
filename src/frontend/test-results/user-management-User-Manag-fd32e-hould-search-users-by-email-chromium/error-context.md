# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: user-management.spec.js >> User Management >> User Search >> should search users by email
- Location: tests/e2e/user-management.spec.js:106:5

# Error details

```
Test timeout of 30000ms exceeded.
```

```
Error: locator.fill: Test timeout of 30000ms exceeded.
Call log:
  - waiting for locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first()

```

# Page snapshot

```yaml
- generic [ref=e4]:
  - generic [ref=e5]:
    - heading "Welcome Back" [level=1] [ref=e6]
    - paragraph [ref=e7]: Please login to your account
  - generic [ref=e8]:
    - generic [ref=e9]:
      - generic [ref=e10]: 邮箱*
      - textbox "邮箱*" [ref=e11]:
        - /placeholder: 请输入邮箱
    - generic [ref=e12]:
      - generic [ref=e13]: 密码*
      - textbox "密码*" [ref=e14]:
        - /placeholder: 请输入密码
    - button "登录" [ref=e15] [cursor=pointer]
  - paragraph [ref=e17]:
    - text: Don't have an account?
    - link "Sign Up Now" [ref=e18] [cursor=pointer]:
      - /url: /register
```

# Test source

```ts
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
  29  |       await expect(page.locator('th').first()).toBeVisible();
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
> 113 |       await searchInput.fill('example');
      |                         ^ Error: locator.fill: Test timeout of 30000ms exceeded.
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
  130 |       const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first();
  131 |       
  132 |       const initialCount = await page.locator('table tbody tr').count();
  133 |       
  134 |       await searchInput.fill('nonexistentsearch');
  135 |       await page.waitForTimeout(500);
  136 |       
  137 |       await searchInput.fill('');
  138 |       await page.waitForTimeout(500);
  139 |       
  140 |       const finalCount = await page.locator('table tbody tr').count();
  141 |       expect(finalCount).toBe(initialCount);
  142 |     });
  143 | 
  144 |     test('should show no results message for non-existent user', async ({ page }) => {
  145 |       await page.goto('/admin/users');
  146 |       
  147 |       await page.waitForLoadState('networkidle');
  148 |       
  149 |       const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first();
  150 |       
  151 |       await searchInput.fill('xyznonexistent123abc');
  152 |       await page.waitForTimeout(1000);
  153 |       
  154 |       const emptyState = page.locator('[class*="empty"], [class*="no-results"], p:has-text("暂无")');
  155 |       const hasEmptyState = await emptyState.isVisible().catch(() => false);
  156 |       
  157 |       if (hasEmptyState) {
  158 |         await expect(emptyState.first()).toBeVisible();
  159 |       } else {
  160 |         const rowCount = await page.locator('table tbody tr').count();
  161 |         expect(rowCount).toBe(0);
  162 |       }
  163 |     });
  164 | 
  165 |     test('should support real-time search filtering', async ({ page }) => {
  166 |       await page.goto('/admin/users');
  167 |       
  168 |       await page.waitForLoadState('networkidle');
  169 |       
  170 |       const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first();
  171 |       
  172 |       await searchInput.type('a', { delay: 100 });
  173 |       await page.waitForTimeout(300);
  174 |       
  175 |       const rowsAfterOneChar = await page.locator('table tbody tr').count();
  176 |       
  177 |       await searchInput.type('dm', { delay: 100 });
  178 |       await page.waitForTimeout(300);
  179 |       
  180 |       const rowsAfterMoreChars = await page.locator('table tbody tr').count();
  181 |       
  182 |       expect(rowsAfterMoreChars).toBeLessThanOrEqual(rowsAfterOneChar);
  183 |     });
  184 |   });
  185 | 
  186 |   test.describe('User Editing', () => {
  187 |     test('should open edit modal when clicking edit button', async ({ page }) => {
  188 |       await page.goto('/admin/users');
  189 |       
  190 |       await page.waitForLoadState('networkidle');
  191 |       await page.waitForSelector('table tbody tr', { timeout: 10000 }).catch(() => {});
  192 |       
  193 |       const editButton = page.locator('button:has-text("编辑"), button:has-text("Edit"), button:has-text("edit")').first();
  194 |       
  195 |       if (await editButton.isVisible().catch(() => false)) {
  196 |         await editButton.click();
  197 |         
  198 |         await page.waitForTimeout(500);
  199 |         const modal = page.locator('[role="dialog"], .modal, .modal-content, [class*="drawer"]');
  200 |         await expect(modal).toBeVisible({ timeout: 5000 });
  201 |       }
  202 |     });
  203 | 
  204 |     test('should pre-fill edit form with user data', async ({ page }) => {
  205 |       await page.goto('/admin/users');
  206 |       
  207 |       await page.waitForLoadState('networkidle');
  208 |       await page.waitForSelector('table tbody tr', { timeout: 10000 }).catch(() => {});
  209 |       
  210 |       const firstRow = page.locator('table tbody tr').first();
  211 |       const userName = await firstRow.locator('td').nth(2).textContent();
  212 |       const userEmail = await firstRow.locator('td').nth(3).textContent();
  213 |       
```