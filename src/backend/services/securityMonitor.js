const fs = require('fs');
const path = require('path');

class SecurityMonitor {
  constructor() {
    this.alerts = [];
    this.threats = [];
    this.suspiciousActivities = [];
    this.maxAlerts = 1000;
    this.threatDetectionRules = this.loadThreatRules();
    this.alertCallbacks = [];
  }

  loadThreatRules() {
    return [
      {
        id: 'THR001',
        name: 'SQL Injection Attempt',
        severity: 'critical',
        patterns: [
          /('|(\\')|(;)|(\-\-)|(\/\*)|(\*\/)|(\@))/,
          /(union\s+select|union\s+all\s+select)/i,
          /(exec\s*\(|execute\s*\(|eval\s*\()/i,
          /(\bor\b.*=.*\bor\b)/i,
          /(and\s+1\s*=\s*1|and\s+1\s*=\s*2)/i
        ]
      },
      {
        id: 'THR002',
        name: 'XSS Attempt',
        severity: 'high',
        patterns: [
          /<script[^>]*>/i,
          /<iframe[^>]*>/i,
          /javascript:/i,
          /on\w+\s*=/i,
          /<img[^>]+onerror/i,
          /<svg[^>]+onload/i
        ]
      },
      {
        id: 'THR003',
        name: 'Path Traversal Attempt',
        severity: 'high',
        patterns: [
          /(\.\.\/|\.\.\\|%2e%2e%2f|%2e%2e\/)/i,
          /(\.\.%2f|%2e%2e%5c)/i,
          /(etc\/passwd|windows\/system32)/i
        ]
      },
      {
        id: 'THR004',
        name: 'Command Injection Attempt',
        severity: 'critical',
        patterns: [
          /(\||\;|`|\$\()\s*(cat|ls|dir|whoami|ifconfig|ping)/i,
          /(\$\{.*\}|\$\w+)/,
          /(&&|\|\|)\s*(rm|wget|curl)/i
        ]
      },
      {
        id: 'THR005',
        name: 'Brute Force Attack',
        severity: 'high',
        patterns: []
      },
      {
        id: 'THR006',
        name: 'Suspicious User Agent',
        severity: 'medium',
        patterns: [/sqlmap/i, /nikto/i, /nmap/i, /masscan/i, /hydra/i, /burp/i, /scanner/i]
      },
      {
        id: 'THR007',
        name: 'Rate Limit Exceeded',
        severity: 'medium',
        patterns: []
      },
      {
        id: 'THR008',
        name: 'Invalid Authentication Token',
        severity: 'medium',
        patterns: []
      }
    ];
  }

  detectThreat(requestData) {
    const { body, query, headers, ip, path } = requestData;
    const inputString = JSON.stringify({ body, query, path });

    for (const rule of this.threatDetectionRules) {
      if (rule.patterns.length === 0) continue;

      for (const pattern of rule.patterns) {
        if (pattern.test(inputString) || pattern.test(headers['user-agent'] || '')) {
          return this.createThreatRecord(rule, requestData);
        }
      }
    }

    return null;
  }

  createThreatRecord(rule, requestData) {
    const record = {
      id: `${rule.id}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      threatId: rule.id,
      threatName: rule.name,
      severity: rule.severity,
      timestamp: new Date().toISOString(),
      request: {
        method: requestData.method,
        path: requestData.path,
        ip: requestData.ip,
        userAgent: requestData.headers['user-agent'],
        body: this.sanitizeForLog(requestData.body),
        query: requestData.query
      }
    };

    this.threats.push(record);
    this.triggerAlert(record);

    return record;
  }

  sanitizeForLog(data) {
    if (typeof data === 'object' && data !== null) {
      const sanitized = {};
      for (const [key, value] of Object.entries(data)) {
        if (['password', 'token', 'secret', 'key', 'authorization'].includes(key.toLowerCase())) {
          sanitized[key] = '[REDACTED]';
        } else if (typeof value === 'string' && value.length > 500) {
          sanitized[key] = value.substring(0, 500) + '...';
        } else {
          sanitized[key] = this.sanitizeForLog(value);
        }
      }
      return sanitized;
    }
    return data;
  }

  logSuspiciousActivity(type, data) {
    const activity = {
      id: `ACT-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      type,
      timestamp: new Date().toISOString(),
      ...data
    };

    this.suspiciousActivities.push(activity);

    if (this.suspiciousActivities.length > this.maxAlerts) {
      this.suspiciousActivities.shift();
    }

    return activity;
  }

  recordFailedLogin(email, ip, userAgent) {
    return this.logSuspiciousActivity('FAILED_LOGIN', {
      email,
      ip,
      userAgent,
      attempts: 1
    });
  }

  recordFailedAuth(token, ip) {
    return this.logSuspiciousActivity('FAILED_AUTH', {
      tokenPrefix: token ? token.substring(0, 10) : 'none',
      ip
    });
  }

  recordRateLimitExceeded(ip, endpoint) {
    return this.logSuspiciousActivity('RATE_LIMIT_EXCEEDED', {
      ip,
      endpoint
    });
  }

  recordUnauthorizedAccess(userId, resource) {
    return this.logSuspiciousActivity('UNAUTHORIZED_ACCESS', {
      userId,
      resource
    });
  }

  triggerAlert(threat) {
    const alert = {
      id: `ALERT-${Date.now()}`,
      threatId: threat.id,
      severity: threat.severity,
      message: `Security threat detected: ${threat.threatName}`,
      timestamp: new Date().toISOString(),
      acknowledged: false
    };

    this.alerts.push(alert);

    if (this.alerts.length > this.maxAlerts) {
      this.alerts.shift();
    }

    for (const callback of this.alertCallbacks) {
      try {
        callback(alert);
      } catch (error) {
        console.error('Alert callback error:', error);
      }
    }

    console.error(`🚨 SECURITY ALERT [${threat.severity.toUpperCase()}]: ${threat.threatName}`);
    console.error(`   IP: ${threat.request.ip}`);
    console.error(`   Path: ${threat.request.path}`);
    console.error(`   Time: ${threat.timestamp}`);

    return alert;
  }

  onAlert(callback) {
    this.alertCallbacks.push(callback);
  }

  acknowledgeAlert(alertId) {
    const alert = this.alerts.find(a => a.id === alertId);
    if (alert) {
      alert.acknowledged = true;
      alert.acknowledgedAt = new Date().toISOString();
      return true;
    }
    return false;
  }

  getAlerts(options = {}) {
    const { severity, acknowledged, limit = 100 } = options;

    let filtered = [...this.alerts];

    if (severity) {
      filtered = filtered.filter(a => a.severity === severity);
    }

    if (acknowledged !== undefined) {
      filtered = filtered.filter(a => a.acknowledged === acknowledged);
    }

    return filtered.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp)).slice(0, limit);
  }

  getThreats(options = {}) {
    const { severity, limit = 100 } = options;

    let filtered = [...this.threats];

    if (severity) {
      filtered = filtered.filter(t => t.severity === severity);
    }

    return filtered.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp)).slice(0, limit);
  }

  getSuspiciousActivities(options = {}) {
    const { type, limit = 100 } = options;

    let filtered = [...this.suspiciousActivities];

    if (type) {
      filtered = filtered.filter(a => a.type === type);
    }

    return filtered.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp)).slice(0, limit);
  }

  getStatistics() {
    const now = Date.now();
    const oneHourAgo = now - 3600000;
    const oneDayAgo = now - 86400000;

    const recentAlerts = this.alerts.filter(a => new Date(a.timestamp).getTime() > oneHourAgo);
    const todayAlerts = this.alerts.filter(a => new Date(a.timestamp).getTime() > oneDayAgo);

    const threatCounts = {
      critical: this.threats.filter(t => t.severity === 'critical').length,
      high: this.threats.filter(t => t.severity === 'high').length,
      medium: this.threats.filter(t => t.severity === 'medium').length,
      low: this.threats.filter(t => t.severity === 'low').length
    };

    return {
      totalAlerts: this.alerts.length,
      unacknowledgedAlerts: this.alerts.filter(a => !a.acknowledged).length,
      totalThreats: this.threats.length,
      threatsBySeverity: threatCounts,
      alertsLastHour: recentAlerts.length,
      alertsLastDay: todayAlerts.length,
      suspiciousActivities: this.suspiciousActivities.length
    };
  }

  clearOldData(hoursToKeep = 168) {
    const cutoff = Date.now() - hoursToKeep * 3600000;

    const filterByTime = items => items.filter(item => new Date(item.timestamp).getTime() > cutoff);

    this.alerts = filterByTime(this.alerts);
    this.threats = filterByTime(this.threats);
    this.suspiciousActivities = filterByTime(this.suspiciousActivities);

    return {
      alertsRemaining: this.alerts.length,
      threatsRemaining: this.threats.length,
      activitiesRemaining: this.suspiciousActivities.length
    };
  }

  exportReport() {
    return {
      timestamp: new Date().toISOString(),
      statistics: this.getStatistics(),
      recentAlerts: this.getAlerts({ limit: 50 }),
      recentThreats: this.getThreats({ limit: 50 }),
      recentActivities: this.getSuspiciousActivities({ limit: 50 }),
      threatDetectionRules: this.threatDetectionRules.map(r => ({
        id: r.id,
        name: r.name,
        severity: r.severity,
        patternsCount: r.patterns.length
      }))
    };
  }
}

const securityMonitor = new SecurityMonitor();

module.exports = securityMonitor;
