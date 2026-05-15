class UnifiedSecurityMonitor {
  constructor(options = {}) {
    this.alertThreshold = {
      failedLogin: parseInt(process.env.ALERT_FAILED_LOGIN) || 5,
      suspiciousIP: parseInt(process.env.ALERT_SUSPICIOUS_IP) || 10,
      xssAttempt: parseInt(process.env.ALERT_XSS_ATTEMPT) || 3,
      sqlInjection: parseInt(process.env.ALERT_SQL_INJECTION) || 1,
      bruteForce: parseInt(process.env.ALERT_BRUTE_FORCE) || 10,
      rateLimit: parseInt(process.env.ALERT_RATE_LIMIT) || 50
    };

    this.alerts = [];
    this.threats = [];
    this.suspiciousActivities = [];
    this.counters = new Map();
    this.maxAlerts = options.maxAlerts || 1000;
    this.maxAlertHistory = 1000;
    this.enabled = options.enabled !== false;
    this.alertHandlers = new Set();
    this.alertCallbacks = new Set();
    this.alertMethods = options.alertMethods || ['email', 'slack', 'webhook'];
    this.alertRecipients = options.alertRecipients || [];

    this.threatDetectionRules = this.loadThreatRules();
    this.startCleanupInterval();
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
        patterns: [
          /sqlmap/i,
          /nikto/i,
          /nmap/i,
          /masscan/i,
          /hydra/i,
          /burp/i,
          /scanner/i
        ]
      }
    ];
  }

  async checkAndAlert(event) {
    if (!this.enabled) {
      return;
    }

    const { type, severity, source, details } = this.normalizeEvent(event);

    this.updateCounter(type, source);

    const count = this.getCounter(type, source);
    const threshold = this.alertThreshold[type] || 5;

    if (count >= threshold) {
      await this.sendAlert({
        type,
        severity,
        source,
        count,
        threshold,
        details,
        timestamp: new Date().toISOString()
      });
    }

    return {
      type,
      count,
      threshold,
      thresholdReached: count >= threshold
    };
  }

  normalizeEvent(event) {
    return {
      type: event.type || 'unknown',
      severity: event.severity || this.getDefaultSeverity(event.type),
      source: event.ip || event.userId || event.source || 'unknown',
      details: this.sanitizeForLog(event.details || {})
    };
  }

  getDefaultSeverity(type) {
    const severityMap = {
      sqlInjection: 'critical',
      xssAttempt: 'high',
      bruteForce: 'high',
      failedLogin: 'medium',
      suspiciousIP: 'medium',
      rateLimit: 'low'
    };
    return severityMap[type] || 'low';
  }

  updateCounter(type, source) {
    const key = `${type}:${source}`;
    const current = this.counters.get(key) || { count: 0, firstSeen: Date.now() };
    current.count++;
    current.lastSeen = Date.now();
    this.counters.set(key, current);
  }

  getCounter(type, source) {
    const key = `${type}:${source}`;
    return this.counters.get(key)?.count || 0;
  }

  resetCounter(type, source) {
    const key = `${type}:${source}`;
    this.counters.delete(key);
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

    if (this.threats.length > this.maxAlerts) {
      this.threats.shift();
    }

    this.triggerAlert(record);
    this.checkAndAlert({
      type: rule.id.toLowerCase(),
      severity: rule.severity,
      ip: requestData.ip,
      details: record
    });

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

  async sendAlert(alert) {
    this.alerts.push(alert);
    if (this.alerts.length > this.maxAlertHistory) {
      this.alerts.shift();
    }

    for (const handler of this.alertHandlers) {
      try {
        await handler(alert);
      } catch (error) {
        console.error('Alert handler error:', error);
      }
    }

    for (const callback of this.alertCallbacks) {
      try {
        callback(alert);
      } catch (error) {
        console.error('Alert callback error:', error);
      }
    }

    if (this.alertMethods.includes('email')) {
      await this.sendEmailAlert(alert);
    }

    if (this.alertMethods.includes('slack')) {
      await this.sendSlackAlert(alert);
    }

    if (this.alertMethods.includes('webhook')) {
      await this.sendWebhookAlert(alert);
    }

    console.warn(`[SECURITY ALERT] ${alert.severity?.toUpperCase() || 'UNKNOWN'}: ${alert.type} from ${alert.source} (count: ${alert.count})`);
  }

  async sendEmailAlert(alert) {
    if (!process.env.SMTP_HOST || !this.alertRecipients.length) {
      return;
    }

    const subject = `[Security Alert] ${alert.severity?.toUpperCase()}: ${alert.type}`;
    const body = `
Security Alert Details:
- Type: ${alert.type}
- Severity: ${alert.severity}
- Source: ${alert.source}
- Count: ${alert.count} (threshold: ${alert.threshold})
- Timestamp: ${alert.timestamp}
- Details: ${JSON.stringify(alert.details, null, 2)}
    `.trim();

    console.log(`[Email Alert] Would send to: ${this.alertRecipients.join(', ')}`);
    console.log(`Subject: ${subject}`);
  }

  async sendSlackAlert(alert) {
    if (!process.env.SLACK_WEBHOOK_URL) {
      return;
    }

    console.log('[Slack Alert] Would send to Slack');
  }

  async sendWebhookAlert(alert) {
    if (!process.env.WEBHOOK_URL) {
      return;
    }

    console.log('[Webhook Alert] Would send to webhook');
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

    if (this.alerts.length > this.maxAlertHistory) {
      this.alerts.shift();
    }

    console.error(`🚨 SECURITY ALERT [${threat.severity?.toUpperCase()}]: ${threat.threatName}`);
    console.error(`   IP: ${threat.request?.ip}`);
    console.error(`   Path: ${threat.request?.path}`);
    console.error(`   Time: ${threat.timestamp}`);

    return alert;
  }

  addAlertHandler(handler) {
    this.alertHandlers.add(handler);
  }

  removeAlertHandler(handler) {
    this.alertHandlers.delete(handler);
  }

  onAlert(callback) {
    this.alertCallbacks.add(callback);
  }

  offAlert(callback) {
    this.alertCallbacks.delete(callback);
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

    return filtered
      .sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))
      .slice(0, limit);
  }

  getThreats(options = {}) {
    const { severity, limit = 100 } = options;

    let filtered = [...this.threats];

    if (severity) {
      filtered = filtered.filter(t => t.severity === severity);
    }

    return filtered
      .sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))
      .slice(0, limit);
  }

  getSuspiciousActivities(options = {}) {
    const { type, limit = 100 } = options;

    let filtered = [...this.suspiciousActivities];

    if (type) {
      filtered = filtered.filter(a => a.type === type);
    }

    return filtered
      .sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))
      .slice(0, limit);
  }

  generateSecurityReport(options = {}) {
    const {
      startDate = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000),
      endDate = new Date(),
      includeDetails = true
    } = options;

    const alerts = this.alerts.filter(a => {
      const timestamp = new Date(a.timestamp);
      return timestamp >= startDate && timestamp <= endDate;
    });

    const summary = {
      period: { start: startDate.toISOString(), end: endDate.toISOString() },
      totalAlerts: alerts.length,
      bySeverity: {
        critical: alerts.filter(a => a.severity === 'critical').length,
        high: alerts.filter(a => a.severity === 'high').length,
        medium: alerts.filter(a => a.severity === 'medium').length,
        low: alerts.filter(a => a.severity === 'low').length
      },
      byType: {},
      topSources: {},
      thresholdStats: {}
    };

    alerts.forEach(alert => {
      summary.byType[alert.type] = (summary.byType[alert.type] || 0) + 1;
      summary.topSources[alert.source] = (summary.topSources[alert.source] || 0) + 1;
    });

    const topSources = Object.entries(summary.topSources)
      .sort((a, b) => b[1] - a[1])
      .slice(0, 10);
    summary.topSources = Object.fromEntries(topSources);

    if (includeDetails) {
      summary.recentAlerts = alerts.slice(-50);
    }

    summary.riskScore = this.calculateRiskScore(summary);
    summary.recommendations = this.generateRecommendations(summary);

    return summary;
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
      suspiciousActivities: this.suspiciousActivities.length,
      activeCounters: this.counters.size
    };
  }

  calculateRiskScore(summary) {
    let score = 0;

    score += summary.bySeverity.critical * 10;
    score += summary.bySeverity.high * 5;
    score += summary.bySeverity.medium * 2;
    score += summary.bySeverity.low * 1;

    if (summary.totalAlerts > 100) score += 20;
    else if (summary.totalAlerts > 50) score += 10;

    return Math.min(score, 100);
  }

  generateRecommendations(summary) {
    const recommendations = [];

    if (summary.bySeverity.critical > 0) {
      recommendations.push({
        priority: 'URGENT',
        category: 'Critical Security Events Detected',
        action: 'Investigate critical alerts immediately and consider temporary access restrictions'
      });
    }

    if (summary.byType.sqlInjection > 0 || summary.byType.thr001 > 0) {
      recommendations.push({
        priority: 'HIGH',
        category: 'SQL Injection Attempts',
        action: 'Review and strengthen input validation and parameterized queries'
      });
    }

    if (summary.byType.xssAttempt > 0 || summary.byType.thr002 > 0) {
      recommendations.push({
        priority: 'HIGH',
        category: 'XSS Attempts',
        action: 'Review output encoding and Content-Security-Policy configuration'
      });
    }

    if (summary.byType.bruteForce > 0 || summary.byType.failedLogin > 10) {
      recommendations.push({
        priority: 'MEDIUM',
        category: 'Authentication Attacks',
        action: 'Consider implementing account lockout and CAPTCHA for login attempts'
      });
    }

    return recommendations;
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

  clearOldData(hoursToKeep = 168) {
    const cutoff = Date.now() - hoursToKeep * 3600000;

    const filterByTime = items =>
      items.filter(item => new Date(item.timestamp).getTime() > cutoff);

    this.alerts = filterByTime(this.alerts);
    this.threats = filterByTime(this.threats);
    this.suspiciousActivities = filterByTime(this.suspiciousActivities);

    return {
      alertsRemaining: this.alerts.length,
      threatsRemaining: this.threats.length,
      activitiesRemaining: this.suspiciousActivities.length
    };
  }

  startCleanupInterval() {
    setInterval(() => {
      const oneHourAgo = Date.now() - (60 * 60 * 1000);

      for (const [key, value] of this.counters.entries()) {
        if (value.lastSeen < oneHourAgo) {
          this.counters.delete(key);
        }
      }
    }, 60 * 60 * 1000);
  }

  getStats() {
    return {
      enabled: this.enabled,
      alertHistorySize: this.alerts.length,
      activeCounters: this.counters.size,
      thresholds: this.alertThreshold,
      alertMethods: this.alertMethods,
      threatDetectionRulesCount: this.threatDetectionRules.length
    };
  }

  reset() {
    this.counters.clear();
    this.alerts = [];
    this.threats = [];
    this.suspiciousActivities = [];
  }

  close() {
    this.enabled = false;
    this.reset();
  }
}

const unifiedSecurityMonitor = new UnifiedSecurityMonitor();

module.exports = { UnifiedSecurityMonitor, unifiedSecurityMonitor };
