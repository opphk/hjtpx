import { Page } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

export class TestHelpers {
  private screenshotDir: string;

  constructor(screenshotBaseDir: string = 'test-screenshots') {
    this.screenshotDir = path.join(process.cwd(), screenshotBaseDir);
    this.ensureScreenshotDir();
  }

  private ensureScreenshotDir() {
    if (!fs.existsSync(this.screenshotDir)) {
      fs.mkdirSync(this.screenshotDir, { recursive: true });
    }
  }

  async takeScreenshot(page: Page, name: string, fullPage: boolean = true) {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const filename = `${timestamp}-${name}.png`;
    const filepath = path.join(this.screenshotDir, filename);
    await page.screenshot({ path: filepath, fullPage });
    console.log(`Screenshot saved: ${filepath}`);
    return filepath;
  }

  async checkConsoleErrors(page: Page): Promise<string[]> {
    const errors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });
    return errors;
  }

  async checkNetworkRequests(page: Page) {
    const requests: { url: string; status: number; method: string }[] = [];
    page.on('response', response => {
      requests.push({
        url: response.url(),
        status: response.status(),
        method: response.request().method()
      });
    });
    return requests;
  }
}
