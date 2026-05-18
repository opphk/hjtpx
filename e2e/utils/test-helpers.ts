import { Page } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

interface ConsoleMessage {
  type: 'log' | 'error' | 'warning' | 'info';
  text: string;
  timestamp: Date;
  location?: { url: string; lineNumber: number; columnNumber: number };
}

interface NetworkRequest {
  url: string;
  method: string;
  status: number;
  timestamp: Date;
  duration?: number;
}

interface ConsoleMonitor {
  messages: ConsoleMessage[];
  errors: ConsoleMessage[];
  warnings: ConsoleMessage[];
  startTime: Date;
  endTime?: Date;
}

export class TestHelpers {
  private screenshotDir: string;
  private consoleMonitor: ConsoleMonitor | null = null;
  private networkMonitor: { requests: NetworkRequest[] } | null = null;
  private pageListeners: Map<string, any> = new Map();

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

  async takeElementScreenshot(page: Page, selector: string, name: string) {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const filename = `${timestamp}-${name}.png`;
    const filepath = path.join(this.screenshotDir, filename);
    
    const element = page.locator(selector);
    await element.screenshot({ path: filepath });
    console.log(`Element screenshot saved: ${filepath}`);
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

  startConsoleMonitor(page: Page): void {
    this.consoleMonitor = {
      messages: [],
      errors: [],
      warnings: [],
      startTime: new Date()
    };

    const messageHandler = (msg: any) => {
      if (!this.consoleMonitor) return;
      
      const consoleMsg: ConsoleMessage = {
        type: msg.type(),
        text: msg.text(),
        timestamp: new Date()
      };
      
      this.consoleMonitor.messages.push(consoleMsg);
      
      if (msg.type() === 'error') {
        this.consoleMonitor.errors.push(consoleMsg);
      } else if (msg.type() === 'warning') {
        this.consoleMonitor.warnings.push(consoleMsg);
      }
    };

    page.on('console', messageHandler);
    this.pageListeners.set('console', messageHandler);
  }

  async stopConsoleMonitor(page: Page): Promise<ConsoleMonitor> {
    if (!this.consoleMonitor) {
      this.consoleMonitor = {
        messages: [],
        errors: [],
        warnings: [],
        startTime: new Date()
      };
    }
    
    this.consoleMonitor.endTime = new Date();
    
    const consoleHandler = this.pageListeners.get('console');
    if (consoleHandler) {
      page.off('console', consoleHandler);
      this.pageListeners.delete('console');
    }
    
    return this.consoleMonitor;
  }

  getConsoleErrors(): ConsoleMessage[] {
    return this.consoleMonitor?.errors || [];
  }

  getConsoleWarnings(): ConsoleMessage[] {
    return this.consoleMonitor?.warnings || [];
  }

  getCriticalErrors(): ConsoleMessage[] {
    if (!this.consoleMonitor) return [];
    
    return this.consoleMonitor.errors.filter(err => 
      !err.text.includes('favicon') &&
      !err.text.includes('404') &&
      !err.text.includes('Failed to load resource')
    );
  }

  hasCriticalErrors(): boolean {
    return this.getCriticalErrors().length > 0;
  }

  startNetworkMonitor(page: Page): void {
    this.networkMonitor = {
      requests: []
    };

    const requestHandler = (request: any) => {
      if (!this.networkMonitor) return;
      
      this.networkMonitor.requests.push({
        url: request.url(),
        method: request.method(),
        status: 0,
        timestamp: new Date()
      });
    };

    const responseHandler = (response: any) => {
      if (!this.networkMonitor) return;
      
      const request = this.networkMonitor.requests.find(r => r.url === response.url());
      if (request) {
        request.status = response.status();
      }
    };

    page.on('request', requestHandler);
    page.on('response', responseHandler);
    
    this.pageListeners.set('request', requestHandler);
    this.pageListeners.set('response', responseHandler);
  }

  getNetworkRequests(): NetworkRequest[] {
    return this.networkMonitor?.requests || [];
  }

  getFailedRequests(): NetworkRequest[] {
    if (!this.networkMonitor) return [];
    return this.networkMonitor.requests.filter(r => r.status >= 400);
  }

  generateConsoleReport(): string {
    if (!this.consoleMonitor) {
      return 'No console monitoring data available';
    }

    const lines: string[] = [
      '=== Console Monitoring Report ===',
      `Start Time: ${this.consoleMonitor.startTime.toISOString()}`,
      `End Time: ${this.consoleMonitor.endTime?.toISOString() || 'N/A'}`,
      `Total Messages: ${this.consoleMonitor.messages.length}`,
      `Errors: ${this.consoleMonitor.errors.length}`,
      `Warnings: ${this.consoleMonitor.warnings.length}`,
      ''
    ];

    if (this.consoleMonitor.errors.length > 0) {
      lines.push('=== Errors ===');
      this.consoleMonitor.errors.forEach((err, idx) => {
        lines.push(`${idx + 1}. [${err.type.toUpperCase()}] ${err.text}`);
      });
      lines.push('');
    }

    if (this.consoleMonitor.warnings.length > 0) {
      lines.push('=== Warnings ===');
      this.consoleMonitor.warnings.forEach((warn, idx) => {
        lines.push(`${idx + 1}. [${warn.type.toUpperCase()}] ${warn.text}`);
      });
    }

    return lines.join('\n');
  }

  saveConsoleReport(filepath?: string): string {
    const report = this.generateConsoleReport();
    const filename = filepath || path.join(
      this.screenshotDir,
      `console-report-${new Date().toISOString().replace(/[:.]/g, '-')}.txt`
    );
    fs.writeFileSync(filename, report, 'utf-8');
    console.log(`Console report saved: ${filename}`);
    return filename;
  }

  async capturePageState(page: Page, name: string): Promise<void> {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const stateFile = path.join(this.screenshotDir, `${timestamp}-${name}-state.json`);
    
    const state = {
      url: page.url(),
      title: await page.title(),
      timestamp: new Date().toISOString(),
      consoleErrors: this.getCriticalErrors().map(e => e.text),
      networkRequests: this.getNetworkRequests().length,
      failedRequests: this.getFailedRequests().length
    };
    
    fs.writeFileSync(stateFile, JSON.stringify(state, null, 2), 'utf-8');
    console.log(`Page state saved: ${stateFile}`);
  }
}
