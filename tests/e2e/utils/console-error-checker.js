const { test } = require('@playwright/test');
const fs = require('fs').promises;
const path = require('path');

class ConsoleErrorChecker {
  constructor(page, testInfo) {
    this.page = page;
    this.testInfo = testInfo;
    this.errors = [];
    this.warnings = [];
    this.messages = [];
  }

  setup() {
    this.page.on('console', (msg) => {
      const type = msg.type();
      const text = msg.text();
      const location = msg.location();
      
      const entry = {
        type,
        text,
        url: location?.url,
        lineNumber: location?.lineNumber,
        columnNumber: location?.columnNumber,
        timestamp: new Date().toISOString()
      };

      if (type === 'error') {
        this.errors.push(entry);
      } else if (type === 'warning') {
        this.warnings.push(entry);
      } else {
        this.messages.push(entry);
      }
    });

    this.page.on('pageerror', (error) => {
      this.errors.push({
        type: 'exception',
        text: error.message,
        stack: error.stack,
        timestamp: new Date().toISOString()
      });
    });
  }

  getErrors() {
    return this.errors;
  }

  getWarnings() {
    return this.warnings;
  }

  getMessages() {
    return this.messages;
  }

  hasErrors() {
    return this.errors.length > 0;
  }

  hasWarnings() {
    return this.warnings.length > 0;
  }

  async saveLogs() {
    const logsDir = path.join(
      this.testInfo.outputDir,
      'logs'
    );
    
    await fs.mkdir(logsDir, { recursive: true });

    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    
    const errorLogPath = path.join(logsDir, `errors-${timestamp}.json`);
    await fs.writeFile(errorLogPath, JSON.stringify(this.errors, null, 2));
    
    const warningLogPath = path.join(logsDir, `warnings-${timestamp}.json`);
    await fs.writeFile(warningLogPath, JSON.stringify(this.warnings, null, 2));
    
    const messageLogPath = path.join(logsDir, `messages-${timestamp}.json`);
    await fs.writeFile(messageLogPath, JSON.stringify(this.messages, null, 2));

    return {
      errors: errorLogPath,
      warnings: warningLogPath,
      messages: messageLogPath
    };
  }

  reset() {
    this.errors = [];
    this.warnings = [];
    this.messages = [];
  }
}

module.exports = { ConsoleErrorChecker };
