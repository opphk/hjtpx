const fs = require('fs');
const path = require('path');

const { defaultAuditLogger } = require('../utils/security/audit_logger');
const { LogSanitizer } = require('../utils/security/log_sanitizer');

class SecurityMonitor {
  constructor(options = {}) {
    this.alertThreshold = {
      failedLogin: parseInt(process.env.ALERT_FAILED_LOGIN) || 5,
      suspiciousIP: parseInt(process.env.ALERT_SUSPICIOUS_IP) || 10,
      xssAttempt: parseInt(process.env.ALERT_XSS_ATTEMPT) || 3,
      sqlInjection: parseInt(process.env.ALERT_SQL_INJECTION) || 1,
      bruteForce: parseInt(process.env.ALERT_BRUTE_FORCE) || 10,
      rateLimit: parseInt(process.env.ALERT_RATE_LIMIT) || 50
    };

    this.counters = new Map();
    this.alertHistory = [];
    this.maxAlertHistory = 1000;
    this.enabled = options.enabled !== false;
    this.alertHandlers = new Set();
    this.alertMethods = options.alertMethods || ['email', 'slack', 'webhook'];
    this.alertRecipients = options.alertRecipients || [];

    this.startCleanupInterval();
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

      if (defaultAuditLogger) {
        defaultAuditLogger.log('SECURITY_ALERT', null, {
          type,
          severity,
          source,
          count,
          threshold
        });
      }
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
      details: LogSanitizer.sanitize(event.details || {})
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

  async sendAlert(alert) {
    this.alertHistory.push(alert);
    if (this.alertHistory.length > this.maxAlertHistory) {
      this.alertHistory.shift();
    }

    for (const handler of this.alertHandlers) {
      try {
        await handler(alert);
      } catch (error) {
        console.error('Alert handler error:', error);
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

    console.warn(
      `[SECURITY ALERT] ${alert.severity.toUpperCase()}: ${alert.type} from ${alert.source} (count: ${alert.count})`
    );
  }

  async sendEmailAlert(alert) {
    if (!process.env.SMTP_HOST || !this.alertRecipients.length) {
      return;
    }

    const subject = `[Security Alert] ${alert.severity.toUpperCase()}: ${alert.type}`;
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

    const severityEmoji = {
      critical: ':rotating_light:',
      high: ':warning:',
      medium: ':large_yellow_circle:',
      low: ':information_source:'
    };

    const payload = {
      text: `${severityEmoji[alert.severity]} Security Alert: ${alert.type}`,
      attachments: [
        {
          color: this.getSeverityColor(alert.severity),
          fields: [
            { title: 'Severity', value: alert.severity.toUpperCase(), short: true },
            { title: 'Source', value: alert.source, short: true },
            { title: 'Count', value: String(alert.count), short: true },
            { title: 'Threshold', value: String(alert.threshold), short: true },
            { title: 'Timestamp', value: alert.timestamp, short: false }
          ]
        }
      ]
    };

    console.log('[Slack Alert] Would send to Slack');
  }

  async sendWebhookAlert(alert) {
    if (!process.env.WEBHOOK_URL) {
      return;
    }

    console.log('[Webhook Alert] Would send to webhook');
  }

  getSeverityColor(severity) {
    const colors = {
      critical: '#FF0000',
      high: '#FFA500',
      medium: '#FFFF00',
      low: '#00FF00'
    };
    return colors[severity] || '#00FF00';
  }

  addAlertHandler(handler) {
    this.alertHandlers.add(handler);
  }

  removeAlertHandler(handler) {
    this.alertHandlers.delete(handler);
  }

  generateSecurityReport(options = {}) {
    const {
      startDate = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000),
      endDate = new Date(),
      includeDetails = true
    } = options;

    const alerts = this.alertHistory.filter(a => {
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

    if (summary.byType.sqlInjection > 0) {
      recommendations.push({
        priority: 'HIGH',
        category: 'SQL Injection Attempts',
        action: 'Review and strengthen input validation and parameterized queries'
      });
    }

    if (summary.byType.xssAttempt > 0) {
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

  startCleanupInterval() {
    setInterval(
      () => {
        const oneHourAgo = Date.now() - 60 * 60 * 1000;

        for (const [key, value] of this.counters.entries()) {
          if (value.lastSeen < oneHourAgo) {
            this.counters.delete(key);
          }
        }
      },
      60 * 60 * 1000
    );
  }

  getStats() {
    return {
      enabled: this.enabled,
      alertHistorySize: this.alertHistory.length,
      activeCounters: this.counters.size,
      thresholds: this.alertThreshold,
      alertMethods: this.alertMethods
    };
  }

  reset() {
    this.counters.clear();
    this.alertHistory = [];
  }

  close() {
    this.enabled = false;
    this.reset();
  }
}

const defaultSecurityMonitor = new SecurityMonitor();

module.exports = { SecurityMonitor, defaultSecurityMonitor };
