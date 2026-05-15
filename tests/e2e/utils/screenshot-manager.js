const { test } = require('@playwright/test');
const path = require('path');

class ScreenshotManager {
  constructor(page, testInfo) {
    this.page = page;
    this.testInfo = testInfo;
  }

  async capture(name, options = {}) {
    const screenshotDir = path.join(
      this.testInfo.outputDir,
      'screenshots'
    );

    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const sanitizedName = name.replace(/[^a-zA-Z0-9-_]/g, '-');
    const screenshotName = `${sanitizedName}-${timestamp}.png`;
    const screenshotPath = path.join(screenshotDir, screenshotName);

    await this.page.screenshot({
      path: screenshotPath,
      fullPage: options.fullPage ?? true,
      ...options
    });

    await this.testInfo.attach(screenshotName, {
      path: screenshotPath,
      contentType: 'image/png'
    });

    return screenshotPath;
  }

  async captureElement(selector, name, options = {}) {
    const element = await this.page.locator(selector).first();
    const screenshotDir = path.join(
      this.testInfo.outputDir,
      'screenshots'
    );

    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const sanitizedName = name.replace(/[^a-zA-Z0-9-_]/g, '-');
    const screenshotName = `element-${sanitizedName}-${timestamp}.png`;
    const screenshotPath = path.join(screenshotDir, screenshotName);

    await element.screenshot({
      path: screenshotPath,
      ...options
    });

    await this.testInfo.attach(screenshotName, {
      path: screenshotPath,
      contentType: 'image/png'
    });

    return screenshotPath;
  }

  async captureOnFailure(callback) {
    try {
      await callback();
    } catch (error) {
      await this.capture('failure-screenshot');
      throw error;
    }
  }

  async captureViewport(name, options = {}) {
    return this.capture(name, {
      fullPage: false,
      ...options
    });
  }
}

module.exports = { ScreenshotManager };
