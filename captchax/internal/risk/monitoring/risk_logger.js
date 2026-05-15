/**
 * @fileoverview 风控日志记录器
 * @description 风控事件记录、验证尝试记录、风险判定记录
 * @module captchax/internal/risk/monitoring/risk_logger
 */

'use strict';

class RiskLogger {
  constructor(config = {}) {
    this.config = {
      maxLogSize: config.maxLogSize || 10000,
      logLevel: config.logLevel || 'info',
      enableConsole: config.enableConsole !== false,
      enableStorage: config.enableStorage !== false,
      storageKey: config.storageKey || 'captchax_risk_logs',
      retentionDays: config.retentionDays || 30,
      batchSize: config.batchSize || 50,
      flushInterval: config.flushInterval || 5000,
      ...config
    };

    this.logs = [];
    this.pendingLogs = [];
    this.eventHandlers = new Map();
    this.statistics = this.initializeStatistics();
    this.flushTimer = null;
    
    if (this.config.enableStorage) {
      this.loadFromStorage();
      this.startFlushTimer();
    }
  }

  initializeStatistics() {
    return {
      totalLogs: 0,
      byLevel: {
        info: 0,
        warning: 0,
        error: 0,
        critical: 0
      },
      byAction: {
        allow: 0,
        verify: 0,
        block: 0
      },
      byRiskLevel: {
        low: 0,
        medium: 0,
        high: 0,
        critical: 0
      },
      averageRiskScore: 0,
      totalRiskScore: 0,
      recentAlerts: []
    };
  }

  log(event) {
    const logEntry = this.formatLogEntry(event);
    
    this.logs.push(logEntry);
    
    if (this.logs.length > this.config.maxLogSize) {
      this.logs.shift();
    }

    this.updateStatistics(logEntry);
    
    this.pendingLogs.push(logEntry);
    
    if (this.pendingLogs.length >= this.config.batchSize) {
      this.flush();
    }

    this.emit('log', logEntry);
    
    if (this.config.enableConsole) {
      this.writeToConsole(logEntry);
    }

    return logEntry;
  }

  logRiskEvent(data) {
    const event = {
      type: 'risk_event',
      timestamp: Date.now(),
      userId: data.userId || null,
      sessionId: data.sessionId || null,
      riskScore: data.riskScore,
      riskLevel: data.riskLevel,
      action: data.action || 'unknown',
      factors: data.factors || [],
      features: data.features || null,
      fingerprint: data.fingerprint || null,
      ip: data.ip || null,
      userAgent: data.userAgent || null,
      metadata: data.metadata || {}
    };

    return this.log(event);
  }

  logVerificationAttempt(data) {
    const event = {
      type: 'verification_attempt',
      timestamp: Date.now(),
      userId: data.userId || null,
      sessionId: data.sessionId || null,
      captchaType: data.captchaType,
      captchaId: data.captchaId,
      success: data.success,
      timeSpent: data.timeSpent,
      attempts: data.attempts || 1,
      riskScore: data.riskScore,
      riskLevel: data.riskLevel,
      metadata: data.metadata || {}
    };

    return this.log(event);
  }

  logAnomaly(data) {
    const event = {
      type: 'anomaly_detected',
      timestamp: Date.now(),
      userId: data.userId || null,
      sessionId: data.sessionId || null,
      anomalyScore: data.anomalyScore,
      anomalyFactors: data.anomalyFactors || [],
      method: data.method,
      severity: data.severity || 'medium',
      metadata: data.metadata || {}
    };

    this.statistics.recentAlerts.push({
      timestamp: event.timestamp,
      userId: event.userId,
      type: event.type,
      severity: event.severity
    });

    if (this.statistics.recentAlerts.length > 100) {
      this.statistics.recentAlerts.shift();
    }

    return this.log(event);
  }

  logSecurityEvent(data) {
    const event = {
      type: 'security_event',
      timestamp: Date.now(),
      eventCategory: data.eventCategory,
      severity: data.severity || 'medium',
      userId: data.userId || null,
      ip: data.ip || null,
      description: data.description,
      details: data.details || {},
      action: data.action || 'logged'
    };

    return this.log(event);
  }

  formatLogEntry(event) {
    return {
      id: this.generateLogId(),
      timestamp: event.timestamp,
      type: event.type,
      level: this.determineLogLevel(event),
      data: event
    };
  }

  determineLogLevel(event) {
    switch (event.type) {
      case 'anomaly_detected':
        if (event.severity === 'critical') return 'critical';
        if (event.severity === 'high') return 'error';
        return 'warning';
      
      case 'security_event':
        if (event.severity === 'critical') return 'critical';
        if (event.severity === 'high') return 'error';
        return 'warning';
      
      case 'risk_event':
        if (event.riskLevel === 'critical') return 'error';
        if (event.riskLevel === 'high') return 'warning';
        return 'info';
      
      case 'verification_attempt':
        return 'info';
      
      default:
        return 'info';
    }
  }

  updateStatistics(logEntry) {
    this.statistics.totalLogs++;

    const level = logEntry.level;
    if (this.statistics.byLevel[level] !== undefined) {
      this.statistics.byLevel[level]++;
    }

    const event = logEntry.data;

    if (event.action && this.statistics.byAction[event.action] !== undefined) {
      this.statistics.byAction[event.action]++;
    }

    if (event.riskLevel && this.statistics.byRiskLevel[event.riskLevel] !== undefined) {
      this.statistics.byRiskLevel[event.riskLevel]++;
    }

    if (typeof event.riskScore === 'number') {
      this.statistics.totalRiskScore += event.riskScore;
      this.statistics.averageRiskScore = 
        this.statistics.totalRiskScore / this.statistics.totalLogs;
    }
  }

  generateLogId() {
    const timestamp = Date.now().toString(36);
    const random = Math.random().toString(36).substring(2, 10);
    return `${timestamp}-${random}`;
  }

  writeToConsole(logEntry) {
    const levelColors = {
      info: '\x1b[36m',
      warning: '\x1b[33m',
      error: '\x1b[31m',
      critical: '\x1b[35m'
    };

    const color = levelColors[logEntry.level] || '\x1b[0m';
    const reset = '\x1b[0m';
    
    const time = new Date(logEntry.timestamp).toISOString();
    const message = `[${time}] [${logEntry.level.toUpperCase()}] [${logEntry.type}]`;
    
    console.log(`${color}${message}${reset}`, logEntry.data);
  }

  flush() {
    if (this.pendingLogs.length === 0) return;

    const logsToFlush = [...this.pendingLogs];
    this.pendingLogs = [];

    if (this.config.enableStorage) {
      this.saveToStorage(logsToFlush);
    }

    this.emit('flush', logsToFlush);
  }

  startFlushTimer() {
    if (this.flushTimer) {
      clearInterval(this.flushTimer);
    }

    this.flushTimer = setInterval(() => {
      this.flush();
      this.cleanupOldLogs();
    }, this.config.flushInterval);
  }

  stopFlushTimer() {
    if (this.flushTimer) {
      clearInterval(this.flushTimer);
      this.flushTimer = null;
    }
  }

  cleanupOldLogs() {
    const cutoffTime = Date.now() - (this.config.retentionDays * 24 * 60 * 60 * 1000);
    
    this.logs = this.logs.filter(log => log.timestamp >= cutoffTime);
  }

  saveToStorage(logs) {
    try {
      let storedLogs = [];
      
      try {
        const stored = localStorage.getItem(this.config.storageKey);
        if (stored) {
          storedLogs = JSON.parse(stored);
        }
      } catch (e) {
        console.warn('Failed to load stored logs:', e);
      }

      storedLogs = storedLogs.concat(logs);

      if (storedLogs.length > this.config.maxLogSize) {
        storedLogs = storedLogs.slice(-this.config.maxLogSize);
      }

      try {
        localStorage.setItem(this.config.storageKey, JSON.stringify(storedLogs));
      } catch (e) {
        console.warn('Failed to save logs to storage:', e);
        this.handleStorageOverflow();
      }
    } catch (error) {
      console.error('Error saving logs to storage:', error);
    }
  }

  loadFromStorage() {
    try {
      const stored = localStorage.getItem(this.config.storageKey);
      if (stored) {
        this.logs = JSON.parse(stored);
      }
    } catch (error) {
      console.error('Error loading logs from storage:', error);
      this.logs = [];
    }
  }

  handleStorageOverflow() {
    const halfSize = Math.floor(this.config.maxLogSize / 2);
    this.logs = this.logs.slice(-halfSize);
    
    try {
      localStorage.setItem(this.config.storageKey, JSON.stringify(this.logs));
    } catch (e) {
      console.error('Failed to save even after cleanup:', e);
      localStorage.removeItem(this.config.storageKey);
    }
  }

  getLogs(filter = {}) {
    let result = [...this.logs];

    if (filter.type) {
      result = result.filter(log => log.data.type === filter.type);
    }

    if (filter.level) {
      result = result.filter(log => log.level === filter.level);
    }

    if (filter.userId) {
      result = result.filter(log => log.data.userId === filter.userId);
    }

    if (filter.sessionId) {
      result = result.filter(log => log.data.sessionId === filter.sessionId);
    }

    if (filter.startTime) {
      result = result.filter(log => log.timestamp >= filter.startTime);
    }

    if (filter.endTime) {
      result = result.filter(log => log.timestamp <= filter.endTime);
    }

    if (filter.limit) {
      result = result.slice(-filter.limit);
    }

    return result;
  }

  getStatistics() {
    return {
      ...this.statistics,
      logsInMemory: this.logs.length,
      pendingLogs: this.pendingLogs.length,
      recentLogs: this.logs.slice(-10)
    };
  }

  on(event, handler) {
    if (!this.eventHandlers.has(event)) {
      this.eventHandlers.set(event, []);
    }
    this.eventHandlers.get(event).push(handler);
    return this;
  }

  off(event, handler) {
    if (!this.eventHandlers.has(event)) return this;
    
    const handlers = this.eventHandlers.get(event);
    const index = handlers.indexOf(handler);
    if (index > -1) {
      handlers.splice(index, 1);
    }
    return this;
  }

  emit(event, data) {
    if (!this.eventHandlers.has(event)) return;
    
    const handlers = this.eventHandlers.get(event);
    for (const handler of handlers) {
      try {
        handler(data);
      } catch (error) {
        console.error(`Error in event handler for ${event}:`, error);
      }
    }
  }

  clear() {
    this.logs = [];
    this.pendingLogs = [];
    
    if (this.config.enableStorage) {
      localStorage.removeItem(this.config.storageKey);
    }
    
    this.statistics = this.initializeStatistics();
  }

  exportLogs(format = 'json') {
    if (format === 'json') {
      return JSON.stringify(this.logs, null, 2);
    }
    
    if (format === 'csv') {
      const headers = ['id', 'timestamp', 'type', 'level', 'userId', 'riskScore', 'action'];
      const rows = this.logs.map(log => [
        log.id,
        new Date(log.timestamp).toISOString(),
        log.type,
        log.level,
        log.data.userId || '',
        log.data.riskScore || '',
        log.data.action || ''
      ]);
      
      return [headers.join(','), ...rows.map(row => row.join(','))].join('\n');
    }
    
    return this.logs;
  }

  destroy() {
    this.stopFlushTimer();
    this.flush();
    this.eventHandlers.clear();
  }
}

module.exports = RiskLogger;
