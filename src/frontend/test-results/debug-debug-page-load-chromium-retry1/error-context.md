# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: debug.spec.js >> debug page load
- Location: tests/e2e/debug.spec.js:3:1

# Error details

```
Error: expect(received).toBe(expected) // Object.is equality

Expected: 0
Received: 1
```

# Test source

```ts
  1  | import { test, expect } from '@playwright/test';
  2  | 
  3  | test('debug page load', async ({ page }) => {
  4  |   const errors = [];
  5  |   page.on('console', msg => {
  6  |     console.log(`CONSOLE ${msg.type()}: ${msg.text()}`);
  7  |   });
  8  |   page.on('pageerror', error => {
  9  |     console.log(`PAGE ERROR: ${error.message}`);
  10 |     errors.push(error.message);
  11 |   });
  12 | 
  13 |   await page.goto('/login');
  14 |   await page.waitForTimeout(5000);
  15 | 
  16 |   const html = await page.content();
  17 |   console.log('PAGE HTML:', html.substring(0, 500));
  18 | 
  19 |   const rootContent = await page.evaluate(() => {
  20 |     const root = document.getElementById('root');
  21 |     return root ? root.innerHTML : 'root not found';
  22 |   });
  23 |   console.log('ROOT CONTENT:', rootContent.substring(0, 500));
  24 | 
> 25 |   expect(errors.length).toBe(0);
     |                         ^ Error: expect(received).toBe(expected) // Object.is equality
  26 | });
  27 | 
```