# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: user-management.spec.js >> User Management >> User Search >> should show no results message for non-existent user
- Location: tests/e2e/user-management.spec.js:144:5

# Error details

```
Error: Channel closed
```

```
Error: locator.fill: Target page, context or browser has been closed
Call log:
  - waiting for locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first()

```

```
Error: browserContext.close: Target page, context or browser has been closed
```