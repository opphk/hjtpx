const { IPControl } = require('../middleware/security/ip_whitelist');
const { defaultAuditLogger } = require('../utils/security/audit_logger');

const { defaultSecurityMonitor } = require('./security_monitor');

class EmergencyResponse {
  constructor(options = {}) {
    this.incidentHistory = [];
    this.maxHistorySize = 1000;
    this.ipControl = new IPControl();
    this.autoBlockEnabled = options.autoBlock !== false;
    this.notificationCallbacks = new Set();
    this.incidentHandlers = new Map();

    this.severityLevels = {
      LOW: 1,
      MEDIUM: 2,
      HIGH: 3,
      CRITICAL: 4,
      EMERGENCY: 5
    };

    this.initDefaultHandlers();
  }

  initDefaultHandlers() {
    this.registerHandler('sql_injection', async incident => {
      await this.blockIP(incident.source);
      return { action: 'ip_blocked', reason: 'sql_injection_detected' };
    });

    this.registerHandler('xss_attempt', async incident => {
      await this.logIncident(incident);
      return { action: 'logged', reason: 'xss_attempt' };
    });

    this.registerHandler('brute_force', async incident => {
      if (incident.count >= 20) {
        await this.blockIP(incident.source);
        await this.disableAccount(incident.target);
        return { action: 'ip_blocked_account_disabled', reason: 'brute_force_attack' };
      }
      return { action: 'monitoring', reason: 'brute_force_attack' };
    });

    this.registerHandler('suspicious_activity', async incident => {
      await this.logIncident(incident);
      await this.notifySecurityTeam(incident);
      return { action: 'notified', reason: 'suspicious_activity' };
    });

    this.registerHandler('data_breach', async incident => {
      await this.blockIP(incident.source);
      await this.revokeAllSessions();
      await this.notifySecurityTeam(incident, true);
      await this.initiateIncidentResponse(incident);
      return { action: 'full_response', reason: 'data_breach_detected' };
    });

    this.registerHandler('ddos_attack', async incident => {
      await this.enableEmergencyMode();
      await this.activateDDoSProtection(incident);
      return { action: 'ddos_protection_activated', reason: 'ddos_attack_detected' };
    });
  }

  registerHandler(type, handler) {
    this.incidentHandlers.set(type, handler);
  }

  async handleSecurityIncident(incident) {
    const normalizedIncident = this.normalizeIncident(incident);

    this.incidentHistory.push(normalizedIncident);
    if (this.incidentHistory.length > this.maxHistorySize) {
      this.incidentHistory.shift();
    }

    const severity = this.evaluateSeverity(normalizedIncident);
    normalizedIncident.severity = severity;

    if (defaultAuditLogger) {
      defaultAuditLogger.log(
        'SECURITY_INCIDENT',
        normalizedIncident.target || null,
        normalizedIncident
      );
    }

    if (defaultSecurityMonitor) {
      await defaultSecurityMonitor.checkAndAlert({
        type: normalizedIncident.type,
        severity,
        source: normalizedIncident.source,
        details: normalizedIncident
      });
    }

    const handler = this.incidentHandlers.get(normalizedIncident.type);
    let result = { action: 'none', reason: 'no_handler' };

    if (handler) {
      try {
        result = await handler(normalizedIncident);
      } catch (error) {
        console.error('Incident handler error:', error);
        result = { action: 'error', reason: error.message };
      }
    }

    for (const callback of this.notificationCallbacks) {
      try {
        await callback(normalizedIncident, result);
      } catch (error) {
        console.error('Notification callback error:', error);
      }
    }

    return {
      incident: normalizedIncident,
      severity,
      result,
      timestamp: new Date().toISOString()
    };
  }

  normalizeIncident(incident) {
    return {
      id: incident.id || this.generateIncidentId(),
      type: incident.type || 'unknown',
      source: incident.source || incident.ip || 'unknown',
      target: incident.target || incident.userId || null,
      description: incident.description || '',
      details: incident.details || {},
      timestamp: incident.timestamp || new Date().toISOString(),
      status: 'open'
    };
  }

  evaluateSeverity(incident) {
    const baseSeverity = this.severityLevels[incident.severity?.toUpperCase()] || 2;

    const typeSeverity = {
      data_breach: 5,
      ddos_attack: 5,
      sql_injection: 4,
      xss_attempt: 3,
      brute_force: 3,
      suspicious_activity: 2,
      failed_login: 1
    };

    const maxSeverity = typeSeverity[incident.type] || baseSeverity;

    if (incident.count && incident.count > 10) {
      return Math.min(maxSeverity + 1, 5);
    }

    return maxSeverity;
  }

  async blockIP(ip, duration = null) {
    this.ipControl.addBlacklist(ip);

    const blockEntry = {
      ip,
      blockedAt: new Date().toISOString(),
      duration,
      reason: 'security_incident'
    };

    if (defaultAuditLogger) {
      defaultAuditLogger.log('IP_BLOCKED', null, blockEntry);
    }

    console.warn(`[Emergency Response] IP blocked: ${ip}`);

    return blockEntry;
  }

  async unblockIP(ip) {
    this.ipControl.removeBlacklist(ip);

    if (defaultAuditLogger) {
      defaultAuditLogger.log('IP_UNBLOCKED', null, { ip, unblockedAt: new Date().toISOString() });
    }

    console.log(`[Emergency Response] IP unblocked: ${ip}`);
  }

  async disableAccount(userId, reason = 'security_incident') {
    const action = {
      userId,
      action: 'account_disabled',
      reason,
      timestamp: new Date().toISOString()
    };

    if (defaultAuditLogger) {
      defaultAuditLogger.log('ACCOUNT_DISABLED', userId, action);
    }

    console.warn(`[Emergency Response] Account disabled: ${userId}`);

    return action;
  }

  async enableAccount(userId) {
    const action = {
      userId,
      action: 'account_enabled',
      timestamp: new Date().toISOString()
    };

    if (defaultAuditLogger) {
      defaultAuditLogger.log('ACCOUNT_ENABLED', userId, action);
    }

    console.log(`[Emergency Response] Account enabled: ${userId}`);
  }

  async revokeSession(sessionId) {
    const action = {
      sessionId,
      action: 'session_revoked',
      timestamp: new Date().toISOString()
    };

    if (defaultAuditLogger) {
      defaultAuditLogger.log('SESSION_REVOKED', null, action);
    }

    console.log(`[Emergency Response] Session revoked: ${sessionId}`);

    return action;
  }

  async revokeAllSessions(userId = null) {
    const action = {
      userId,
      action: 'all_sessions_revoked',
      timestamp: new Date().toISOString()
    };

    if (defaultAuditLogger) {
      defaultAuditLogger.log('ALL_SESSIONS_REVOKED', userId, action);
    }

    console.warn(
      `[Emergency Response] All sessions revoked${userId ? ` for user: ${userId}` : ''}`
    );

    return action;
  }

  async notifySecurityTeam(incident, urgent = false) {
    const notification = {
      incidentId: incident.id,
      type: incident.type,
      severity: incident.severity,
      source: incident.source,
      urgent,
      timestamp: new Date().toISOString()
    };

    for (const callback of this.notificationCallbacks) {
      try {
        await callback(incident, { type: 'security_team_notification' });
      } catch (error) {
        console.error('Security team notification failed:', error);
      }
    }

    console.warn(
      `[Emergency Response] Security team notified${urgent ? ' (URGENT)' : ''}:`,
      notification
    );
  }

  async initiateIncidentResponse(incident) {
    const response = {
      incidentId: incident.id,
      status: 'investigating',
      startedAt: new Date().toISOString(),
      actions: []
    };

    response.actions.push({
      type: 'investigation_started',
      timestamp: new Date().toISOString()
    });

    console.error(`[INCIDENT RESPONSE] Incident #${incident.id} - ${incident.type}`);
    console.error(`  Severity: ${incident.severity}`);
    console.error(`  Source: ${incident.source}`);
    console.error(`  Description: ${incident.description}`);

    return response;
  }

  async enableEmergencyMode() {
    console.warn('[Emergency Response] EMERGENCY MODE ENABLED');

    if (defaultAuditLogger) {
      defaultAuditLogger.log('EMERGENCY_MODE_ENABLED', null, {
        timestamp: new Date().toISOString()
      });
    }
  }

  async disableEmergencyMode() {
    console.log('[Emergency Response] Emergency mode disabled');

    if (defaultAuditLogger) {
      defaultAuditLogger.log('EMERGENCY_MODE_DISABLED', null, {
        timestamp: new Date().toISOString()
      });
    }
  }

  async activateDDoSProtection(incident) {
    console.warn('[Emergency Response] DDoS protection activated');
    console.log(`  Detected from ${incident.count} sources`);
  }

  async logIncident(incident) {
    if (defaultAuditLogger) {
      defaultAuditLogger.log('INCIDENT_LOGGED', incident.target || null, incident);
    }
  }

  addNotificationCallback(callback) {
    this.notificationCallbacks.add(callback);
  }

  removeNotificationCallback(callback) {
    this.notificationCallbacks.delete(callback);
  }

  getIncidentHistory(filters = {}) {
    let history = [...this.incidentHistory];

    if (filters.type) {
      history = history.filter(i => i.type === filters.type);
    }

    if (filters.severity) {
      history = history.filter(i => i.severity >= this.severityLevels[filters.severity]);
    }

    if (filters.source) {
      history = history.filter(i => i.source === filters.source);
    }

    if (filters.startDate) {
      const start = new Date(filters.startDate);
      history = history.filter(i => new Date(i.timestamp) >= start);
    }

    if (filters.endDate) {
      const end = new Date(filters.endDate);
      history = history.filter(i => new Date(i.timestamp) <= end);
    }

    return history;
  }

  getStatistics() {
    const stats = {
      totalIncidents: this.incidentHistory.length,
      byType: {},
      bySeverity: {},
      blockedIPs: Array.from(this.ipControl.blacklist),
      recentIncidents: this.incidentHistory.slice(-10)
    };

    this.incidentHistory.forEach(incident => {
      stats.byType[incident.type] = (stats.byType[incident.type] || 0) + 1;
      stats.bySeverity[incident.severity] = (stats.bySeverity[incident.severity] || 0) + 1;
    });

    return stats;
  }

  generateIncidentId() {
    return `INC-${Date.now()}-${Math.random().toString(36).substring(2, 9).toUpperCase()}`;
  }

  resolveIncident(incidentId, resolution) {
    const incident = this.incidentHistory.find(i => i.id === incidentId);

    if (incident) {
      incident.status = 'resolved';
      incident.resolution = resolution;
      incident.resolvedAt = new Date().toISOString();

      if (defaultAuditLogger) {
        defaultAuditLogger.log('INCIDENT_RESOLVED', incident.target, incident);
      }
    }

    return incident;
  }

  close() {
    this.incidentHandlers.clear();
    this.notificationCallbacks.clear();
    this.incidentHistory = [];
  }
}

const defaultEmergencyResponse = new EmergencyResponse();

module.exports = { EmergencyResponse, defaultEmergencyResponse };
