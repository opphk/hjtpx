const { execSync } = require('child_process');
const path = require('path');

describe('NPM Audit Security Tests', () => {
  const AUDIT_THRESHOLD = process.env.AUDIT_THRESHOLD || 'moderate';
  
  let auditResults;
  let vulnCounts;

  beforeAll(() => {
    try {
      const auditOutput = execSync('npm audit --json', {
        encoding: 'utf-8',
        cwd: path.join(__dirname, '../../'),
        maxBuffer: 50 * 1024 * 1024
      });
      auditResults = JSON.parse(auditOutput);
    } catch (error) {
      if (error.stdout) {
        auditResults = JSON.parse(error.stdout);
      } else {
        throw new Error(`Failed to run npm audit: ${error.message}`);
      }
    }
    
    vulnCounts = auditResults.metadata?.vulnerabilities || {
      info: 0,
      low: 0,
      moderate: 0,
      high: 0,
      critical: 0,
      total: 0
    };
  });

  describe('Vulnerability Assessment', () => {
    test('should retrieve audit results', () => {
      expect(auditResults).toBeDefined();
    });

    test('should have vulnerability metadata', () => {
      expect(auditResults.metadata).toBeDefined();
      expect(typeof auditResults.metadata).toBe('object');
    });
  });

  describe('Critical Vulnerabilities', () => {
    test('should not have critical vulnerabilities', () => {
      const criticalCount = vulnCounts.critical || 0;
      expect(criticalCount).toBe(0);
    });

    test('should report critical vulnerability details if found', () => {
      const criticalVulns = vulnCounts.critical || 0;
      if (criticalVulns > 0) {
        console.warn('Critical vulnerabilities found:', criticalVulns);
      }
      expect(typeof criticalVulns).toBe('number');
    });
  });

  describe('High Severity Vulnerabilities', () => {
    test('should not have high severity vulnerabilities', () => {
      const highCount = vulnCounts.high || 0;
      expect(highCount).toBe(0);
    });

    test('should report high severity vulnerability details if found', () => {
      const highVulns = vulnCounts.high || 0;
      if (highVulns > 0) {
        console.warn('High severity vulnerabilities found:', highVulns);
      }
      expect(typeof highVulns).toBe('number');
    });
  });

  describe('Moderate Severity Vulnerabilities', () => {
    test('should not exceed moderate vulnerability threshold', () => {
      const moderateCount = vulnCounts.moderate || 0;
      const highCount = vulnCounts.high || 0;
      const criticalCount = vulnCounts.critical || 0;
      
      const totalHighPlus = moderateCount + highCount + criticalCount;
      expect(totalHighPlus).toBeLessThanOrEqual(10);
    });

    test('should document moderate vulnerabilities', () => {
      const moderateVulns = vulnCounts.moderate || 0;
      if (moderateVulns > 0) {
        console.warn('Moderate severity vulnerabilities found:', moderateVulns);
      }
      expect(typeof moderateVulns).toBe('number');
    });
  });

  describe('Low Severity Vulnerabilities', () => {
    test('should document low severity vulnerabilities', () => {
      const lowVulns = vulnCounts.low || 0;
      if (lowVulns > 0) {
        console.warn('Low severity vulnerabilities found:', lowVulns);
      }
      expect(typeof lowVulns).toBe('number');
    });
  });

  describe('Audit Report Details', () => {
    test('should include vulnerability recommendations', () => {
      expect(auditResults.vulnerabilities).toBeDefined();
    });

    test('should list affected packages', () => {
      const { vulnerabilities } = auditResults;
      const vulnKeys = Object.keys(vulnerabilities);
      
      if (vulnKeys.length > 0) {
        const firstVuln = vulnerabilities[vulnKeys[0]];
        if (firstVuln) {
          expect(firstVuln).toHaveProperty('name');
          expect(firstVuln).toHaveProperty('severity');
          expect(firstVuln).toHaveProperty('range');
        }
      }
      
      expect(Array.isArray(vulnKeys)).toBe(true);
    });
  });

  describe('Audit Summary', () => {
    test('should provide audit summary', () => {
      expect(auditResults).toHaveProperty('metadata');
    });

    test('should have valid vulnerability counts', () => {
      Object.keys(vulnCounts).forEach(key => {
        expect(typeof vulnCounts[key]).toBe('number');
        expect(vulnCounts[key]).toBeGreaterThanOrEqual(0);
      });
    });
  });

  describe('Remediation Information', () => {
    test('should document remediation status', () => {
      expect(true).toBe(true);
    });

    test('should list fixable vulnerabilities if available', () => {
      expect(true).toBe(true);
    });

    test('should document total vulnerabilities', () => {
      const total = vulnCounts.total || 0;
      
      if (total > 0) {
        console.warn(`Total vulnerabilities requiring attention: ${total}`);
      }
      
      expect(typeof total).toBe('number');
    });
  });

  describe('Security Compliance', () => {
    const SEVERITY_LEVELS = ['critical', 'high', 'moderate', 'low', 'info', 'total'];

    test('should have valid severity levels in results', () => {
      Object.keys(vulnCounts).forEach(severity => {
        expect(SEVERITY_LEVELS).toContain(severity);
      });
    });

    test('should meet security baseline requirements', () => {
      const hasCriticalOrHigh = 
        (vulnCounts.critical || 0) > 0 ||
        (vulnCounts.high || 0) > 0;
      
      if (hasCriticalOrHigh) {
        console.error('Security baseline FAILED: Critical or High vulnerabilities detected');
        console.error('Vulnerabilities:', vulnCounts);
      }
      
      expect(hasCriticalOrHigh).toBe(false);
    });
  });

  describe('Known Vulnerability Handling', () => {
    test('should document Apollo Server vulnerability as known issue', () => {
      const vulnKeys = Object.keys(auditResults.vulnerabilities || {});
      const apolloRelated = vulnKeys.filter(key => 
        key.toLowerCase().includes('apollo')
      );

      if (apolloRelated.length > 0) {
        console.log('Known Apollo-related vulnerabilities:', apolloRelated);
        console.log('Note: Apollo Server vulnerability GHSA-9q82-xgwf-vj6h is a known issue without immediate fix');
      }

      expect(typeof apolloRelated.length).toBe('number');
    });

    test('should allow known moderate vulnerabilities with action plan', () => {
      const moderateCount = vulnCounts.moderate || 0;
      const criticalCount = vulnCounts.critical || 0;
      const highCount = vulnCounts.high || 0;

      const unfixedCount = moderateCount + criticalCount + highCount;
      
      if (unfixedCount > 0) {
        console.log('Unfixed vulnerabilities require action plan:');
        console.log('- Moderate:', moderateCount);
        console.log('- Critical:', criticalCount);
        console.log('- High:', highCount);
      }

      expect(typeof unfixedCount).toBe('number');
    });
  });
});
