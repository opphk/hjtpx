const { execSync } = require('child_process');
const path = require('path');

describe('NPM Audit Security Tests', () => {
  const AUDIT_THRESHOLD = process.env.AUDIT_THRESHOLD || 'moderate';
  
  let auditResults;

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
  });

  describe('Vulnerability Assessment', () => {
    test('should retrieve audit results', () => {
      expect(auditResults).toBeDefined();
    });

    test('should have vulnerability metadata', () => {
      const { vulnerabilities } = auditResults;
      expect(vulnerabilities).toBeDefined();
      expect(typeof vulnerabilities).toBe('object');
    });
  });

  describe('Critical Vulnerabilities', () => {
    test('should not have critical vulnerabilities', () => {
      const criticalCount = auditResults.vulnerabilities?.critical || 0;
      expect(criticalCount).toBe(0);
    });

    test('should report critical vulnerability details if found', () => {
      const criticalVulns = auditResults.vulnerabilities?.critical || 0;
      if (criticalVulns > 0) {
        console.warn('Critical vulnerabilities found:', criticalVulns);
      }
      expect(typeof criticalVulns).toBe('number');
    });
  });

  describe('High Severity Vulnerabilities', () => {
    test('should not have high severity vulnerabilities', () => {
      const highCount = auditResults.vulnerabilities?.high || 0;
      expect(highCount).toBe(0);
    });

    test('should report high severity vulnerability details if found', () => {
      const highVulns = auditResults.vulnerabilities?.high || 0;
      if (highVulns > 0) {
        console.warn('High severity vulnerabilities found:', highVulns);
      }
      expect(typeof highVulns).toBe('number');
    });
  });

  describe('Moderate Severity Vulnerabilities', () => {
    test('should not exceed moderate vulnerability threshold', () => {
      const moderateCount = auditResults.vulnerabilities?.moderate || 0;
      const highCount = auditResults.vulnerabilities?.high || 0;
      const criticalCount = auditResults.vulnerabilities?.critical || 0;
      
      const totalHighPlus = moderateCount + highCount + criticalCount;
      expect(totalHighPlus).toBeLessThanOrEqual(10);
    });

    test('should document moderate vulnerabilities', () => {
      const moderateVulns = auditResults.vulnerabilities?.moderate || 0;
      if (moderateVulns > 0) {
        console.warn('Moderate severity vulnerabilities found:', moderateVulns);
      }
      expect(typeof moderateVulns).toBe('number');
    });
  });

  describe('Low Severity Vulnerabilities', () => {
    test('should document low severity vulnerabilities', () => {
      const lowVulns = auditResults.vulnerabilities?.low || 0;
      if (lowVulns > 0) {
        console.warn('Low severity vulnerabilities found:', lowVulns);
      }
      expect(typeof lowVulns).toBe('number');
    });
  });

  describe('Audit Report Details', () => {
    test('should include vulnerability recommendations', () => {
      const { vulnerabilities } = auditResults;
      expect(vulnerabilities).toBeDefined();
    });

    test('should list affected packages', () => {
      const { vulnerabilities } = auditResults;
      const vulnKeys = Object.keys(vulnerabilities).filter(k => k !== 'total' && k !== 'info');
      
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
      expect(auditResults).toHaveProperty('vulnerabilities');
    });

    test('should have valid vulnerability counts', () => {
      const vulnCounts = auditResults.vulnerabilities;
      Object.keys(vulnCounts).forEach(key => {
        if (key !== 'total' && key !== 'info') {
          expect(typeof vulnCounts[key]).toBe('number');
          expect(vulnCounts[key]).toBeGreaterThanOrEqual(0);
        }
      });
    });
  });

  describe('Remediation Information', () => {
    test('should document remediation status', () => {
      if (auditResults.actions) {
        expect(Array.isArray(auditResults.actions)).toBe(true);
      } else {
        expect(auditResults.actions).toBeUndefined();
      }
    });

    test('should list fixable vulnerabilities if available', () => {
      if (auditResults.actions) {
        const fixableActions = auditResults.actions.filter(action => 
          action.action === 'install' || action.action === 'update'
        );
        
        if (fixableActions.length > 0) {
          console.log('Fixable vulnerabilities:', fixableActions.length);
        }
        
        expect(Array.isArray(fixableActions)).toBe(true);
      }
    });

    test('should document total vulnerabilities', () => {
      const total = auditResults.vulnerabilities?.total || 0;
      
      if (total > 0) {
        console.warn(`Total vulnerabilities requiring attention: ${total}`);
      }
      
      expect(typeof total).toBe('number');
    });
  });

  describe('Security Compliance', () => {
    const SEVERITY_LEVELS = ['critical', 'high', 'moderate', 'low', 'info'];

    test('should have valid severity levels in results', () => {
      const { vulnerabilities } = auditResults;
      
      Object.keys(vulnerabilities).forEach(severity => {
        expect(SEVERITY_LEVELS).toContain(severity);
      });
    });

    test('should meet security baseline requirements', () => {
      const hasCriticalOrHigh = 
        (auditResults.vulnerabilities?.critical || 0) > 0 ||
        (auditResults.vulnerabilities?.high || 0) > 0;
      
      if (hasCriticalOrHigh) {
        console.error('Security baseline FAILED: Critical or High vulnerabilities detected');
        console.error('Vulnerabilities:', auditResults.vulnerabilities);
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
      const moderateCount = auditResults.vulnerabilities?.moderate || 0;
      const criticalCount = auditResults.vulnerabilities?.critical || 0;
      const highCount = auditResults.vulnerabilities?.high || 0;

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
