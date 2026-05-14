const crypto = require('crypto');
const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

class SecurityAudit {
  constructor() {
    this.findings = [];
    this.severityLevels = ['critical', 'high', 'medium', 'low', 'info'];
    this.rules = this.loadRules();
  }

  loadRules() {
    return [
      {
        id: 'SEC001',
        name: 'Hardcoded Secrets Detection',
        severity: 'critical',
        check: (code) => {
          const patterns = [
            /password\s*=\s*['"][^'"]+['"]/i,
            /api[_-]?key\s*=\s*['"][^'"]+['"]/i,
            /secret\s*=\s*['"][^'"]+['"]/i,
            /token\s*=\s*['"][^'"]+['"]/i
          ];
          return patterns.some(p => p.test(code));
        }
      },
      {
        id: 'SEC002',
        name: 'SQL Injection Vulnerability',
        severity: 'critical',
        check: (code) => {
          const patterns = [
            /query\s*\(\s*['"`]\s*\$/,
            /execute\s*\(\s*['"`]\s*\+/,
            /pool\.query.*\+\s*req/
          ];
          return patterns.some(p => p.test(code));
        }
      },
      {
        id: 'SEC003',
        name: 'XSS Vulnerability',
        severity: 'high',
        check: (code) => {
          const patterns = [
            /innerHTML\s*=\s*(?!.*sanitize)/,
            /dangerouslySetInnerHTML/i,
            /document\.write/
          ];
          return patterns.some(p => p.test(code));
        }
      },
      {
        id: 'SEC004',
        name: 'Weak Cryptography',
        severity: 'high',
        check: (code) => {
          const weakAlg = ['md5', 'sha1', 'des', 'rc4'];
          return weakAlg.some(alg => code.includes(alg));
        }
      },
      {
        id: 'SEC005',
        name: 'Insecure Direct Object Reference',
        severity: 'medium',
        check: (code) => {
          const patterns = [
            /req\.params\.id(?!\s*(?:===|!==|==|!=))/,
            /req\.body\.id(?!\s*(?:===|!==|==|!=))/
          ];
          return patterns.some(p => p.test(code));
        }
      },
      {
        id: 'SEC006',
        name: 'Missing Rate Limiting',
        severity: 'medium',
        check: (code, filePath) => {
          return !code.includes('rateLimit') && !code.includes('rate-limiter');
        }
      },
      {
        id: 'SEC007',
        name: 'Missing CSRF Protection',
        severity: 'medium',
        check: (code) => {
          return !code.includes('csrf') && !code.includes('csurf');
        }
      },
      {
        id: 'SEC008',
        name: 'Weak Password Policy',
        severity: 'medium',
        check: (code) => {
          const hasWeakPassword = /password.*min.*[0-4]/.test(code);
          const hasNoComplexity = !/pattern.*(?=.*[A-Z])(?=.*[a-z])(?=.*\d])/.test(code);
          return hasWeakPassword || hasNoComplexity;
        }
      },
      {
        id: 'SEC009',
        name: 'Missing Input Validation',
        severity: 'medium',
        check: (code) => {
          return !code.includes('joi') && !code.includes('validate') && !code.includes('schema');
        }
      },
      {
        id: 'SEC010',
        name: 'Missing Security Headers',
        severity: 'low',
        check: (code, filePath) => {
          return filePath.includes('middleware') && !code.includes('helmet');
        }
      },
      {
        id: 'SEC011',
        name: 'Verbose Error Messages',
        severity: 'low',
        check: (code) => {
          const patterns = [
            /console\.log.*error/i,
            /res\.json.*error.*stack/
          ];
          return patterns.some(p => p.test(code));
        }
      },
      {
        id: 'SEC012',
        name: 'Missing HTTPS Enforcement',
        severity: 'medium',
        check: (code) => {
          return !code.includes('https') && !code.includes('ssl') && !code.includes('TLS');
        }
      }
    ];
  }

  async auditCode(code, filePath = 'unknown') {
    const fileFindings = [];

    for (const rule of this.rules) {
      try {
        if (rule.check(code, filePath)) {
          fileFindings.push({
            ruleId: rule.id,
            ruleName: rule.name,
            severity: rule.severity,
            file: filePath,
            recommendation: this.getRecommendation(rule.id)
          });
        }
      } catch (error) {
        console.error(`Error checking rule ${rule.id}:`, error);
      }
    }

    this.findings.push(...fileFindings);
    return fileFindings;
  }

  getRecommendation(ruleId) {
    const recommendations = {
      SEC001: 'Move secrets to environment variables. Never commit credentials to version control.',
      SEC002: 'Use parameterized queries or ORM methods. Never concatenate user input into SQL queries.',
      SEC003: 'Use React\'s default escaping or sanitize HTML with DOMPurify before setting innerHTML.',
      SEC004: 'Use strong algorithms like AES-256-GCM, bcrypt, or Argon2 for cryptography.',
      SEC005: 'Always verify user ownership of requested resources before returning data.',
      SEC006: 'Implement rate limiting on sensitive endpoints to prevent brute force attacks.',
      SEC007: 'Implement CSRF tokens for state-changing operations, especially for authenticated users.',
      SEC008: 'Enforce strong password policies: minimum 8 chars, mixed case, numbers, and special chars.',
      SEC009: 'Validate all user input using a validation library like Joi or express-validator.',
      SEC010: 'Add security headers using helmet.js: X-Frame-Options, CSP, HSTS, etc.',
      SEC011: 'Log errors internally but return generic messages to users in production.',
      SEC012: 'Enforce HTTPS and redirect HTTP to HTTPS. Configure HSTS header.'
    };
    return recommendations[ruleId] || 'Review and fix the security issue.';
  }

  getReport() {
    const report = {
      timestamp: new Date().toISOString(),
      totalFindings: this.findings.length,
      bySeverity: {},
      findings: this.findings,
      summary: {
        critical: 0,
        high: 0,
        medium: 0,
        low: 0,
        info: 0
      },
      recommendations: []
    };

    for (const finding of this.findings) {
      report.summary[finding.severity]++;
      if (!report.bySeverity[finding.severity]) {
        report.bySeverity[finding.severity] = [];
      }
      report.bySeverity[finding.severity].push(finding);
    }

    const uniqueRules = [...new Set(this.findings.map(f => f.ruleId))];
    for (const ruleId of uniqueRules) {
      report.recommendations.push({
        ruleId,
        recommendation: this.getRecommendation(ruleId),
        count: this.findings.filter(f => f.ruleId === ruleId).length
      });
    }

    return report;
  }

  clear() {
    this.findings = [];
  }

  generateSecurityScore() {
    const weights = {
      critical: 40,
      high: 20,
      medium: 10,
      low: 5,
      info: 1
    };

    let deduction = 0;
    for (const finding of this.findings) {
      deduction += weights[finding.severity] || 0;
    }

    const score = Math.max(0, 100 - deduction);
    return {
      score,
      grade: score >= 90 ? 'A' : score >= 80 ? 'B' : score >= 70 ? 'C' : score >= 60 ? 'D' : 'F',
      maxDeduction: 100
    };
  }

  async runVulnerabilityScan() {
    console.log('🔍 Starting comprehensive security vulnerability scan...\n');

    const report = {
      timestamp: new Date().toISOString(),
      npmAuditResults: null,
      codeAuditResults: null,
      overallRisk: 'Low',
      recommendations: []
    };

    try {
      console.log('📦 Running npm audit...');
      const npmAuditOutput = execSync('npm audit --json', { encoding: 'utf-8', maxBuffer: 50 * 1024 * 1024 });
      const npmAuditData = JSON.parse(npmAuditOutput);
      report.npmAuditResults = this.parseNpmAuditResults(npmAuditData);
      console.log(`   Found ${report.npmAuditResults.totalVulnerabilities} vulnerabilities`);
    } catch (error) {
      if (error.status === 1 && error.stdout) {
        const npmAuditData = JSON.parse(error.stdout);
        report.npmAuditResults = this.parseNpmAuditResults(npmAuditData);
        console.log(`   Found ${report.npmAuditResults.totalVulnerabilities} vulnerabilities`);
      } else {
        console.log('   ⚠️ npm audit completed with warnings');
        report.npmAuditResults = { error: 'Unable to complete npm audit', vulnerabilities: [], totalVulnerabilities: 0 };
      }
    }

    try {
      console.log('🔎 Scanning code for security issues...');
      const codeResults = await this.scanCodeForVulnerabilities();
      report.codeAuditResults = codeResults;
      console.log(`   Found ${codeResults.totalIssues} code security issues`);
    } catch (error) {
      console.error('   ❌ Error during code scanning:', error.message);
      report.codeAuditResults = { error: 'Code scan failed', issues: [], totalIssues: 0 };
    }

    const totalIssues = (report.npmAuditResults?.totalVulnerabilities || 0) +
                        (report.codeAuditResults?.totalIssues || 0);
    report.totalIssues = totalIssues;

    if (totalIssues > 20) {
      report.overallRisk = 'Critical';
    } else if (totalIssues > 10) {
      report.overallRisk = 'High';
    } else if (totalIssues > 5) {
      report.overallRisk = 'Medium';
    } else if (totalIssues > 0) {
      report.overallRisk = 'Low';
    } else {
      report.overallRisk = 'Minimal';
    }

    report.recommendations = this.generateRecommendations(report);

    console.log('\n📊 Security Scan Summary:');
    console.log(`   Total Issues: ${totalIssues}`);
    console.log(`   Overall Risk: ${report.overallRisk}`);
    console.log(`   NPM Vulnerabilities: ${report.npmAuditResults?.totalVulnerabilities || 0}`);
    console.log(`   Code Issues: ${report.codeAuditResults?.totalIssues || 0}`);

    if (report.recommendations.length > 0) {
      console.log('\n🔧 Recommendations:');
      report.recommendations.forEach((rec, i) => {
        console.log(`   ${i + 1}. ${rec}`);
      });
    }

    const reportPath = path.join(process.cwd(), 'security-scan-report.json');
    fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
    console.log(`\n📄 Detailed report saved to: ${reportPath}`);

    return report;
  }

  parseNpmAuditResults(data) {
    const vulnerabilities = [];

    if (data.advisories) {
      for (const [id, advisory] of Object.entries(data.advisories)) {
        vulnerabilities.push({
          id: id,
          name: advisory.module_name,
          severity: advisory.severity,
          title: advisory.title,
          url: advisory.url,
          description: advisory.overview,
          recommendedUpdate: advisory.fix_available ? advisory.fix_available.version : null,
          vulnerableVersions: advisory.vulnerable_versions
        });
      }
    }

    const bySeverity = { critical: [], high: [], medium: [], low: [], info: [] };
    vulnerabilities.forEach(v => {
      if (bySeverity[v.severity]) {
        bySeverity[v.severity].push(v);
      }
    });

    return {
      totalVulnerabilities: vulnerabilities.length,
      bySeverity,
      vulnerabilities,
      metadata: data.metadata || {}
    };
  }

  async scanCodeForVulnerabilities() {
    const issues = [];
    const srcPath = path.join(process.cwd(), 'src');

    if (!fs.existsSync(srcPath)) {
      return { totalIssues: 0, issues: [] };
    }

    const scanDirectory = (dir, relativePath = '') => {
      try {
        const files = fs.readdirSync(dir);

        for (const file of files) {
          const fullPath = path.join(dir, file);
          const fileRelativePath = path.join(relativePath, file);
          const stat = fs.statSync(fullPath);

          if (stat.isDirectory()) {
            if (!file.startsWith('.') && file !== 'node_modules' && file !== 'coverage') {
              scanDirectory(fullPath, fileRelativePath);
            }
          } else if (/\.(js|jsx|ts|tsx)$/.test(file)) {
            try {
              const content = fs.readFileSync(fullPath, 'utf-8');
              const fileIssues = this.auditCode(content, fileRelativePath);
              issues.push(...fileIssues);
            } catch (e) {
              console.warn(`   Warning: Could not read ${fileRelativePath}`);
            }
          }
        }
      } catch (e) {
        console.warn(`   Warning: Could not scan ${relativePath}`);
      }
    };

    scanDirectory(srcPath);

    const bySeverity = { critical: [], high: [], medium: [], low: [], info: [] };
    issues.forEach(issue => {
      if (bySeverity[issue.severity]) {
        bySeverity[issue.severity].push(issue);
      }
    });

    return {
      totalIssues: issues.length,
      bySeverity,
      issues
    };
  }

  generateRecommendations(report) {
    const recommendations = [];

    if (report.npmAuditResults?.totalVulnerabilities > 0) {
      const critical = report.npmAuditResults.bySeverity.critical?.length || 0;
      const high = report.npmAuditResults.bySeverity.high?.length || 0;

      if (critical > 0) {
        recommendations.push('URGENT: Update packages with critical vulnerabilities immediately using "npm audit fix"');
      }
      if (high > 0) {
        recommendations.push('Update packages with high-severity vulnerabilities to prevent potential attacks');
      }
    }

    if (report.codeAuditResults?.totalIssues > 0) {
      const codeIssues = report.codeAuditResults.issues;
      const criticalCode = codeIssues.filter(i => i.severity === 'critical');

      if (criticalCode.length > 0) {
        recommendations.push('Fix critical security issues in code immediately: hardcoded secrets, SQL injection, etc.');
      }

      const highCode = codeIssues.filter(i => i.severity === 'high');
      if (highCode.length > 0) {
        recommendations.push('Review and fix high-severity code issues (XSS, weak cryptography, etc.)');
      }
    }

    if (report.overallRisk === 'Critical' || report.overallRisk === 'High') {
      recommendations.push('Consider implementing automated security scanning in CI/CD pipeline');
      recommendations.push('Schedule regular security audits and dependency updates');
    }

    return recommendations;
  }
}

const securityAudit = new SecurityAudit();

module.exports = securityAudit;
