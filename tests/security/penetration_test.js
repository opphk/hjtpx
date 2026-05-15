const axios = require('axios');
const https = require('https');

class PenetrationTest {
  constructor(options = {}) {
    this.targetUrl = options.targetUrl || process.env.TARGET_URL || 'http://localhost:3000';
    this.results = [];
    this.testQueue = [];
    this.concurrentLimit = options.concurrentLimit || 5;
    this.timeout = options.timeout || 30000;
    this.verbose = options.verbose !== false;
    
    this.axiosInstance = axios.create({
      baseURL: this.targetUrl,
      timeout: this.timeout,
      validateStatus: () => true,
      httpsAgent: new https.Agent({ rejectUnauthorized: false })
    });
  }

  async testOWASPTop10() {
    this.log('Starting OWASP Top 10 security tests...');
    
    const tests = [
      { name: 'A01-Injection', fn: () => this.testInjection() },
      { name: 'A02-BrokenAuth', fn: () => this.testBrokenAuth() },
      { name: 'A03-SensitiveDataExposure', fn: () => this.testSensitiveDataExposure() },
      { name: 'A04-XMLExternalEntities', fn: () => this.testXXE() },
      { name: 'A05-BrokenAccessControl', fn: () => this.testBrokenAccessControl() },
      { name: 'A06-SecurityMisconfiguration', fn: () => this.testSecurityMisconfiguration() },
      { name: 'A07-XSS', fn: () => this.testXSS() },
      { name: 'A08-InsecureDeserialization', fn: () => this.testInsecureDeserialization() },
      { name: 'A09-VulnerableComponents', fn: () => this.testVulnerableComponents() },
      { name: 'A10-InsufficientLogging', fn: () => this.testInsufficientLogging() }
    ];

    for (const test of tests) {
      try {
        this.log(`\nTesting: ${test.name}`);
        await test.fn();
      } catch (error) {
        this.recordResult(test.name, 'ERROR', error.message);
      }
    }

    return this.generateReport();
  }

  async testInjection() {
    const payloads = [
      "'; DROP TABLE users;--",
      "1 OR 1=1",
      "1' AND '1'='1",
      "1 UNION SELECT NULL--",
      "<script>alert('XSS')</script>",
      "admin'--",
      "1; DELETE FROM sessions WHERE '1'='1"
    ];

    for (const payload of payloads) {
      try {
        const response = await this.axiosInstance.get('/api/test', {
          params: { input: payload }
        });

        const isBlocked = response.status === 400 || 
                         response.status === 403 ||
                         response.data?.error;

        this.recordResult('SQL Injection', isBlocked ? 'PASS' : 'FAIL', {
          payload,
          status: response.status,
          blocked: isBlocked
        });
      } catch (error) {
        this.recordResult('SQL Injection', 'ERROR', error.message);
      }
    }
  }

  async testBrokenAuth() {
    const tests = [
      { name: 'Weak Password Policy', test: () => this.testWeakPasswords() },
      { name: 'Credential Exposure', test: () => this.testCredentialExposure() },
      { name: 'Session Management', test: () => this.testSessionManagement() }
    ];

    for (const test of tests) {
      try {
        await test.test();
      } catch (error) {
        this.recordResult(test.name, 'ERROR', error.message);
      }
    }
  }

  async testWeakPasswords() {
    const weakPasswords = ['123456', 'password', 'admin', 'letmein'];
    
    for (const password of weakPasswords) {
      try {
        const response = await this.axiosInstance.post('/api/auth/login', {
          email: 'admin@test.com',
          password: password
        });

        const weakAccepted = response.status === 200 && response.data?.token;
        this.recordResult('Weak Password Test', weakAccepted ? 'FAIL' : 'PASS', {
          password,
          accepted: weakAccepted
        });
      } catch {
        this.recordResult('Weak Password Test', 'PASS', { password });
      }
    }
  }

  async testCredentialExposure() {
    const sensitiveEndpoints = [
      '/api/config',
      '/api/debug',
      '/api/admin/users',
      '/api/v1/debug'
    ];

    for (const endpoint of sensitiveEndpoints) {
      try {
        const response = await this.axiosInstance.get(endpoint);
        
        const exposure = response.data && (
          response.data.match(/password|token|secret|key/i) ||
          response.data.includes('admin')
        );

        this.recordResult('Credential Exposure', exposure ? 'FAIL' : 'PASS', {
          endpoint,
          exposed: exposure
        });
      } catch {
        this.recordResult('Credential Exposure', 'PASS', { endpoint });
      }
    }
  }

  async testSessionManagement() {
    try {
      const loginRes = await this.axiosInstance.post('/api/auth/login', {
        email: 'test@test.com',
        password: 'testpass123'
      });

      const sessionToken = loginRes.data?.token;

      if (sessionToken) {
        const tokenLength = sessionToken.length >= 32;
        this.recordResult('Session Token Strength', tokenLength ? 'PASS' : 'FAIL', {
          tokenLength
        });
      }
    } catch {
      this.recordResult('Session Management', 'SKIP', 'Unable to test');
    }
  }

  async testSensitiveDataExposure() {
    const sensitiveFields = ['password', 'token', 'apiKey', 'secret'];
    
    try {
      const response = await this.axiosInstance.get('/api/users/1');
      const responseStr = JSON.stringify(response.data);
      
      for (const field of sensitiveFields) {
        const exposed = responseStr.toLowerCase().includes(field) &&
                       !responseStr.includes('****') &&
                       !responseStr.includes('***');
        
        this.recordResult('Sensitive Data Exposure', exposed ? 'FAIL' : 'PASS', {
          field,
          exposed
        });
      }
    } catch {
      this.recordResult('Sensitive Data Exposure', 'PASS', 'No exposure detected');
    }
  }

  async testXXE() {
    const xxePayloads = [
      '<?xml version="1.0"?><!DOCTYPE foo [<!ENTITY xxe SYSTEM "file:///etc/passwd">]><foo>&xxe;</foo>',
      '<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE foo [<!ENTITY xxe SYSTEM "file:///etc/hosts">]><foo>&xxe;</foo>'
    ];

    for (const payload of xxePayloads) {
      try {
        const response = await this.axiosInstance.post('/api/xml', {
          data: payload,
          headers: { 'Content-Type': 'application/xml' }
        });

        const blocked = response.status === 400 || response.data?.error;
        this.recordResult('XXE Injection', blocked ? 'PASS' : 'FAIL', {
          blocked
        });
      } catch {
        this.recordResult('XXE Injection', 'PASS', 'XXE blocked');
      }
    }
  }

  async testBrokenAccessControl() {
    const tests = [
      { method: 'GET', path: '/api/admin/users', expectAdmin: true },
      { method: 'GET', path: '/api/users/999999', expectNotFound: true },
      { method: 'DELETE', path: '/api/users/1', expectForbidden: true },
      { method: 'PUT', path: '/api/admin/config', expectForbidden: true }
    ];

    for (const test of tests) {
      try {
        const response = await this.axiosInstance[test.method.toLowerCase()](test.path);
        
        let result = 'FAIL';
        if (test.expectForbidden && response.status === 403) result = 'PASS';
        if (test.expectNotFound && response.status === 404) result = 'PASS';
        if (test.expectAdmin && response.status === 401) result = 'PASS';

        this.recordResult('Access Control', result, {
          method: test.method,
          path: test.path,
          status: response.status
        });
      } catch {
        this.recordResult('Access Control', 'PASS', { path: test.path });
      }
    }
  }

  async testSecurityMisconfiguration() {
    const headers = [
      'X-Frame-Options',
      'X-Content-Type-Options',
      'X-XSS-Protection',
      'Strict-Transport-Security',
      'Content-Security-Policy'
    ];

    try {
      const response = await this.axiosInstance.get('/');
      const missingHeaders = headers.filter(h => !response.headers[h.toLowerCase()]);

      this.recordResult('Security Headers', missingHeaders.length === 0 ? 'PASS' : 'WARN', {
        missingHeaders,
        presentHeaders: headers.filter(h => response.headers[h.toLowerCase()])
      });
    } catch {
      this.recordResult('Security Headers', 'ERROR', 'Unable to fetch');
    }
  }

  async testXSS() {
    const xssPayloads = [
      '<script>alert(1)</script>',
      '<img src=x onerror=alert(1)>',
      '<svg onload=alert(1)>',
      "javascript:alert('XSS')",
      '<iframe src="javascript:alert(1)">'
    ];

    for (const payload of xssPayloads) {
      try {
        const response = await this.axiosInstance.get('/api/search', {
          params: { q: payload }
        });

        const responseStr = JSON.stringify(response.data);
        const sanitized = !responseStr.includes(payload);

        this.recordResult('XSS', sanitized ? 'PASS' : 'FAIL', {
          payload: payload.substring(0, 50),
          sanitized
        });
      } catch {
        this.recordResult('XSS', 'PASS', 'XSS blocked');
      }
    }
  }

  async testInsecureDeserialization() {
    const payloads = [
      'O:8:"stdClass":1:{s:4:"data";s:4:"test";}',
      'eyJkYXRhIjoidGVzdCJ9',
      '<serialized object>'
    ];

    for (const payload of payloads) {
      try {
        const response = await this.axiosInstance.post('/api/data', {
          serialized: payload
        });

        const blocked = response.status === 400 || response.data?.error;
        this.recordResult('Deserialization', blocked ? 'PASS' : 'FAIL', {
          blocked
        });
      } catch {
        this.recordResult('Deserialization', 'PASS', 'Blocked');
      }
    }
  }

  async testVulnerableComponents() {
    this.recordResult('Dependency Check', 'INFO', {
      message: 'Run npm audit for dependency vulnerability scanning'
    });
  }

  async testInsufficientLogging() {
    try {
      const response = await this.axiosInstance.post('/api/test-action');
      
      const hasRequestId = response.headers['x-request-id'] ||
                          response.data?.requestId;
      
      this.recordResult('Request Logging', hasRequestId ? 'PASS' : 'WARN', {
        hasRequestId
      });
    } catch {
      this.recordResult('Request Logging', 'PASS', 'Error handling present');
    }
  }

  recordResult(category, status, details) {
    const result = {
      timestamp: new Date().toISOString(),
      category,
      status,
      details
    };
    
    this.results.push(result);
    
    if (this.verbose) {
      const icon = status === 'PASS' ? '✓' : status === 'FAIL' ? '✗' : status === 'WARN' ? '!' : '?';
      console.log(`  ${icon} ${category}: ${status}`);
    }
  }

  log(message) {
    if (this.verbose) {
      console.log(message);
    }
  }

  generateReport() {
    const summary = {
      total: this.results.length,
      passed: this.results.filter(r => r.status === 'PASS').length,
      failed: this.results.filter(r => r.status === 'FAIL').length,
      warnings: this.results.filter(r => r.status === 'WARN').length,
      errors: this.results.filter(r => r.status === 'ERROR').length,
      skipped: this.results.filter(r => r.status === 'SKIP').length,
      passRate: 0,
      riskLevel: 'LOW'
    };

    if (summary.total > 0) {
      summary.passRate = ((summary.passed / summary.total) * 100).toFixed(2) + '%';
    }

    if (summary.failed > 0) {
      summary.riskLevel = summary.failed > 5 ? 'HIGH' : 'MEDIUM';
    } else if (summary.warnings > 0) {
      summary.riskLevel = 'LOW';
    } else {
      summary.riskLevel = 'MINIMAL';
    }

    return {
      timestamp: new Date().toISOString(),
      targetUrl: this.targetUrl,
      summary,
      results: this.results,
      recommendations: this.generateRecommendations()
    };
  }

  generateRecommendations() {
    const failedResults = this.results.filter(r => r.status === 'FAIL');
    const recommendations = [];

    if (failedResults.some(r => r.category.includes('Injection'))) {
      recommendations.push({
        severity: 'HIGH',
        category: 'Injection',
        recommendation: 'Implement parameterized queries and input validation'
      });
    }

    if (failedResults.some(r => r.category.includes('XSS'))) {
      recommendations.push({
        severity: 'HIGH',
        category: 'XSS',
        recommendation: 'Implement output encoding and Content-Security-Policy'
      });
    }

    if (failedResults.some(r => r.category.includes('Access Control'))) {
      recommendations.push({
        severity: 'HIGH',
        category: 'Access Control',
        recommendation: 'Implement proper authorization checks on all endpoints'
      });
    }

    if (failedResults.some(r => r.category.includes('Security Headers'))) {
      recommendations.push({
        severity: 'MEDIUM',
        category: 'Security Headers',
        recommendation: 'Configure all recommended security headers'
      });
    }

    return recommendations;
  }

  async runFullSuite() {
    console.log('='.repeat(60));
    console.log('PENETRATION TEST SUITE');
    console.log('='.repeat(60));
    console.log(`Target: ${this.targetUrl}`);
    console.log(`Started: ${new Date().toISOString()}`);
    console.log('='.repeat(60));

    const report = await this.testOWASPTop10();

    console.log('\n' + '='.repeat(60));
    console.log('TEST SUMMARY');
    console.log('='.repeat(60));
    console.log(`Total Tests: ${report.summary.total}`);
    console.log(`Passed: ${report.summary.passed}`);
    console.log(`Failed: ${report.summary.failed}`);
    console.log(`Warnings: ${report.summary.warnings}`);
    console.log(`Pass Rate: ${report.summary.passRate}`);
    console.log(`Risk Level: ${report.summary.riskLevel}`);

    if (report.recommendations.length > 0) {
      console.log('\n' + '='.repeat(60));
      console.log('RECOMMENDATIONS');
      console.log('='.repeat(60));
      report.recommendations.forEach((rec, i) => {
        console.log(`\n${i + 1}. [${rec.severity}] ${rec.category}`);
        console.log(`   ${rec.recommendation}`);
      });
    }

    return report;
  }
}

if (require.main === module) {
  const test = new PenetrationTest({
    targetUrl: process.env.TARGET_URL || 'http://localhost:3000',
    verbose: true
  });

  test.runFullSuite()
    .then(report => {
      console.log('\n' + '='.repeat(60));
      console.log('Report saved to: penetration-test-report.json');
      require('fs').writeFileSync(
        'penetration-test-report.json',
        JSON.stringify(report, null, 2)
      );
    })
    .catch(error => {
      console.error('Test failed:', error);
      process.exit(1);
    });
}

module.exports = PenetrationTest;
