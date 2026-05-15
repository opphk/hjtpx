import { describe, test, expect, vi } from 'vitest';
import { UnifiedSecurityMonitor } from '../../backend/services/unifiedSecurityMonitor';

describe('UnifiedSecurityMonitor', () => {
  let monitor;

  beforeEach(() => {
    monitor = new UnifiedSecurityMonitor({ enabled: true });
  });

  afterEach(() => {
    monitor.close();
  });

  describe('Alert Threshold Management', () => {
    test('should initialize with default thresholds', () => {
      expect(monitor.alertThreshold.failedLogin).toBe(5);
      expect(monitor.alertThreshold.suspiciousIP).toBe(10);
      expect(monitor.alertThreshold.sqlInjection).toBe(1);
    });

    test('should track counter correctly', () => {
      monitor.updateCounter('failedLogin', '192.168.1.1');
      expect(monitor.getCounter('failedLogin', '192.168.1.1')).toBe(1);

      monitor.updateCounter('failedLogin', '192.168.1.1');
      expect(monitor.getCounter('failedLogin', '192.168.1.1')).toBe(2);
    });

    test('should reset counter', () => {
      monitor.updateCounter('failedLogin', '192.168.1.1');
      monitor.updateCounter('failedLogin', '192.168.1.1');
      monitor.resetCounter('failedLogin', '192.168.1.1');
      expect(monitor.getCounter('failedLogin', '192.168.1.1')).toBe(0);
    });

    test('should trigger alert when threshold reached', async () => {
      const alertHandler = vi.fn();
      monitor.addAlertHandler(alertHandler);

      for (let i = 0; i < 5; i++) {
        await monitor.checkAndAlert({
          type: 'failedLogin',
          ip: '192.168.1.1',
          severity: 'medium'
        });
      }

      expect(alertHandler).toHaveBeenCalled();
    });
  });

  describe('Threat Detection', () => {
    test('should detect SQL injection attempts', () => {
      const threat = monitor.detectThreat({
        body: { query: "'; DROP TABLE users; --" },
        query: {},
        headers: {},
        ip: '192.168.1.1',
        path: '/api/users'
      });

      expect(threat).not.toBeNull();
      expect(threat.threatId).toBe('THR001');
      expect(threat.severity).toBe('critical');
    });

    test('should detect XSS attempts', () => {
      const threat = monitor.detectThreat({
        body: { comment: '<script>alert("xss")</script>' },
        query: {},
        headers: {},
        ip: '192.168.1.1',
        path: '/api/comments'
      });

      expect(threat).not.toBeNull();
      expect(threat.threatId).toBe('THR002');
      expect(threat.severity).toBe('high');
    });

    test('should return null for safe input', () => {
      const threat = monitor.detectThreat({
        body: { name: 'John Doe' },
        query: {},
        headers: {},
        ip: '192.168.1.1',
        path: '/api/users'
      });

      expect(threat).toBeNull();
    });

    test('should detect path traversal attempts', () => {
      const threat = monitor.detectThreat({
        body: { file: '../../../etc/passwd' },
        query: {},
        headers: {},
        ip: '192.168.1.1',
        path: '/api/files'
      });

      expect(threat).not.toBeNull();
      expect(threat.threatId).toBe('THR003');
    });
  });

  describe('Suspicious Activity Logging', () => {
    test('should log failed login attempts', () => {
      const activity = monitor.recordFailedLogin('user@example.com', '192.168.1.1', 'Mozilla/5.0');

      expect(activity.type).toBe('FAILED_LOGIN');
      expect(activity.email).toBe('user@example.com');
      expect(activity.ip).toBe('192.168.1.1');
    });

    test('should log failed authentication', () => {
      const activity = monitor.recordFailedAuth('abc123xyz', '192.168.1.1');

      expect(activity.type).toBe('FAILED_AUTH');
      expect(activity.tokenPrefix).toBe('abc123xyz');
    });

    test('should log rate limit exceeded', () => {
      const activity = monitor.recordRateLimitExceeded('192.168.1.1', '/api/users');

      expect(activity.type).toBe('RATE_LIMIT_EXCEEDED');
      expect(activity.ip).toBe('192.168.1.1');
      expect(activity.endpoint).toBe('/api/users');
    });

    test('should log unauthorized access', () => {
      const activity = monitor.recordUnauthorizedAccess('user123', '/admin/settings');

      expect(activity.type).toBe('UNAUTHORIZED_ACCESS');
      expect(activity.userId).toBe('user123');
      expect(activity.resource).toBe('/admin/settings');
    });
  });

  describe('Alert Management', () => {
    test('should acknowledge alerts', () => {
      monitor.alerts.push({
        id: 'ALERT-001',
        acknowledged: false
      });

      const result = monitor.acknowledgeAlert('ALERT-001');

      expect(result).toBe(true);
      expect(monitor.alerts[0].acknowledged).toBe(true);
      expect(monitor.alerts[0].acknowledgedAt).toBeDefined();
    });

    test('should filter alerts by severity', () => {
      monitor.alerts = [
        { id: '1', severity: 'critical', timestamp: new Date().toISOString() },
        { id: '2', severity: 'high', timestamp: new Date().toISOString() },
        { id: '3', severity: 'medium', timestamp: new Date().toISOString() }
      ];

      const criticalAlerts = monitor.getAlerts({ severity: 'critical' });
      expect(criticalAlerts.length).toBe(1);
      expect(criticalAlerts[0].id).toBe('1');
    });

    test('should filter alerts by acknowledged status', () => {
      monitor.alerts = [
        { id: '1', acknowledged: true, timestamp: new Date().toISOString() },
        { id: '2', acknowledged: false, timestamp: new Date().toISOString() }
      ];

      const unacknowledged = monitor.getAlerts({ acknowledged: false });
      expect(unacknowledged.length).toBe(1);
      expect(unacknowledged[0].id).toBe('2');
    });
  });

  describe('Statistics', () => {
    test('should calculate correct statistics', () => {
      monitor.alerts = [
        { severity: 'critical', timestamp: new Date().toISOString() },
        { severity: 'high', timestamp: new Date().toISOString() },
        { severity: 'high', timestamp: new Date().toISOString() },
        { severity: 'medium', timestamp: new Date().toISOString() }
      ];

      monitor.threats = [
        { severity: 'critical', timestamp: new Date().toISOString() },
        { severity: 'high', timestamp: new Date().toISOString() }
      ];

      const stats = monitor.getStatistics();

      expect(stats.totalAlerts).toBe(4);
      expect(stats.threatsBySeverity.critical).toBe(1);
      expect(stats.threatsBySeverity.high).toBe(1);
    });
  });

  describe('Security Report', () => {
    test('should generate security report', () => {
      monitor.alerts = [
        {
          type: 'sqlInjection',
          severity: 'critical',
          source: '192.168.1.1',
          count: 1,
          threshold: 1,
          timestamp: new Date().toISOString()
        }
      ];

      const report = monitor.generateSecurityReport();

      expect(report.totalAlerts).toBe(1);
      expect(report.riskScore).toBeGreaterThan(0);
      expect(report.recommendations.length).toBeGreaterThan(0);
    });

    test('should calculate risk score correctly', () => {
      const summary = {
        bySeverity: {
          critical: 2,
          high: 3,
          medium: 1,
          low: 0
        },
        totalAlerts: 6
      };

      const score = monitor.calculateRiskScore(summary);

      expect(score).toBe(40);
    });
  });

  describe('Data Cleanup', () => {
    test('should clear old data', () => {
      const oldDate = new Date(Date.now() - 200 * 24 * 60 * 60 * 1000);
      const recentDate = new Date();

      monitor.alerts = [
        { timestamp: oldDate.toISOString() },
        { timestamp: recentDate.toISOString() }
      ];

      const result = monitor.clearOldData(168);

      expect(result.alertsRemaining).toBe(1);
    });
  });

  describe('Report Export', () => {
    test('should export complete report', () => {
      const report = monitor.exportReport();

      expect(report.timestamp).toBeDefined();
      expect(report.statistics).toBeDefined();
      expect(report.recentAlerts).toBeDefined();
      expect(report.recentThreats).toBeDefined();
      expect(report.recentActivities).toBeDefined();
      expect(report.threatDetectionRules).toBeDefined();
      expect(report.threatDetectionRules.length).toBeGreaterThan(0);
    });
  });

  describe('Sanitization', () => {
    test('should sanitize sensitive data in logs', () => {
      const result = monitor.sanitizeForLog({
        username: 'testuser',
        password: 'secret123',
        token: 'abc123',
        longField: 'x'.repeat(600)
      });

      expect(result.username).toBe('testuser');
      expect(result.password).toBe('[REDACTED]');
      expect(result.token).toBe('[REDACTED]');
      expect(result.longField.endsWith('...')).toBe(true);
    });
  });

  describe('Alert Callbacks', () => {
    test('should register and call alert callbacks', () => {
      const callback = vi.fn();
      monitor.onAlert(callback);

      monitor.triggerAlert({
        id: 'TEST-001',
        threatName: 'Test Threat',
        severity: 'high',
        request: { ip: '127.0.0.1', path: '/test' },
        timestamp: new Date().toISOString()
      });

      expect(callback).toHaveBeenCalled();
    });

    test('should remove alert callbacks', () => {
      const callback = vi.fn();
      monitor.onAlert(callback);
      monitor.offAlert(callback);

      monitor.triggerAlert({
        id: 'TEST-001',
        threatName: 'Test Threat',
        severity: 'high',
        request: { ip: '127.0.0.1', path: '/test' },
        timestamp: new Date().toISOString()
      });

      expect(callback).not.toHaveBeenCalled();
    });
  });
});
