/**
 * @fileoverview 风控监控系统
 * @description 风险趋势统计、异常行为告警、风控效果评估指标
 * @module captchax/internal/risk/monitoring/risk_monitor
 */

'use strict';

class RiskMonitor {
  constructor(config = {}) {
    this.config = {
      alertThreshold: config.alertThreshold || {
        highRiskRate: 0.3,
        criticalRiskRate: 0.1,
        anomalyRate: 0.2,
        blockRate: 0.5
      },
      windowSize: config.windowSize || 3600000,
      checkInterval: config.checkInterval || 60000,
      enableRealTimeAlerts: config.enableRealTimeAlerts !== false,
      trendWindow: config.trendWindow || 24 * 3600000,
      ...config
    };

    this.metrics = {
      totalRequests: 0,
      totalVerifications: 0,
      totalBlocks: 0,
      totalAllows: 0,
      riskScores: [],
      riskLevels: { low: 0, medium: 0, high: 0, critical: 0 },
      anomalyScores: [],
      verificationTimes: [],
      captchaTypes: {},
      alerts: [],
      trend: []
    };

    this.baselineMetrics = null;
    this.alertCallbacks = [];
    this.monitoringTimer = null;
    this.timeSeriesData = new Map();
  }

  startMonitoring() {
    if (this.monitoringTimer) return;

    this.monitoringTimer = setInterval(() => {
      this.performHealthCheck();
    }, this.config.checkInterval);

    return this;
  }

  stopMonitoring() {
    if (this.monitoringTimer) {
      clearInterval(this.monitoringTimer);
      this.monitoringTimer = null;
    }
  }

  recordRequest(data) {
    this.metrics.totalRequests++;

    const timestamp = Date.now();
    this.addTimeSeriesPoint('requests', timestamp);

    if (data.riskScore !== undefined) {
      this.metrics.riskScores.push(data.riskScore);
      this.addTimeSeriesPoint('riskScore', timestamp, data.riskScore);
    }

    if (data.riskLevel) {
      this.metrics.riskLevels[data.riskLevel]++;
    }

    if (data.action) {
      this.updateActionMetrics(data.action);
    }

    if (data.captchaType) {
      this.recordCaptchaType(data.captchaType);
    }

    this.updateTrend();
  }

  recordVerification(data) {
    this.metrics.totalVerifications++;

    if (data.success !== undefined) {
      this.addTimeSeriesPoint(data.success ? 'successes' : 'failures', Date.now());
    }

    if (data.timeSpent) {
      this.metrics.verificationTimes.push(data.timeSpent);
    }

    if (this.config.enableRealTimeAlerts) {
      this.checkForAlerts();
    }
  }

  recordAnomaly(data) {
    this.metrics.anomalyScores.push(data.score || 0);
    this.addTimeSeriesPoint('anomalies', Date.now());

    if (data.severity === 'high' || data.severity === 'critical') {
      this.triggerAlert({
        type: 'anomaly',
        severity: data.severity,
        userId: data.userId,
        score: data.score,
        timestamp: Date.now()
      });
    }
  }

  updateActionMetrics(action) {
    switch (action) {
      case 'allow':
        this.metrics.totalAllows++;
        this.addTimeSeriesPoint('allows', Date.now());
        break;
      case 'verify':
        this.metrics.totalVerifications++;
        this.addTimeSeriesPoint('verifications', Date.now());
        break;
      case 'block':
        this.metrics.totalBlocks++;
        this.addTimeSeriesPoint('blocks', Date.now());
        break;
    }
  }

  recordCaptchaType(type) {
    if (!this.metrics.captchaTypes[type]) {
      this.metrics.captchaTypes[type] = {
        count: 0,
        successRate: 0,
        totalAttempts: 0,
        successfulAttempts: 0,
        avgTimeSpent: 0,
        totalTimeSpent: 0
      };
    }

    this.metrics.captchaTypes[type].count++;
  }

  recordCaptchaResult(type, success, timeSpent) {
    if (!this.metrics.captchaTypes[type]) {
      this.recordCaptchaType(type);
    }

    const stats = this.metrics.captchaTypes[type];
    stats.totalAttempts++;
    stats.totalTimeSpent += timeSpent;

    if (success) {
      stats.successfulAttempts++;
    }

    stats.successRate = stats.successfulAttempts / stats.totalAttempts;
    stats.avgTimeSpent = stats.totalTimeSpent / stats.totalAttempts;
  }

  addTimeSeriesPoint(metric, timestamp, value = 1) {
    const windowKey = Math.floor(timestamp / this.config.windowSize) * this.config.windowSize;

    if (!this.timeSeriesData.has(windowKey)) {
      this.timeSeriesData.set(windowKey, {
        timestamp: windowKey,
        requests: 0,
        riskScore: [],
        anomalies: 0,
        allows: 0,
        verifications: 0,
        blocks: 0,
        successes: 0,
        failures: 0
      });
    }

    const window = this.timeSeriesData.get(windowKey);

    switch (metric) {
      case 'requests':
        window.requests += value;
        break;
      case 'riskScore':
        window.riskScore.push(value);
        break;
      case 'anomalies':
        window.anomalies += value;
        break;
      case 'allows':
        window.allows += value;
        break;
      case 'verifications':
        window.verifications += value;
        break;
      case 'blocks':
        window.blocks += value;
        break;
      case 'successes':
        window.successes += value;
        break;
      case 'failures':
        window.failures += value;
        break;
    }

    this.cleanupOldTimeSeries();
  }

  cleanupOldTimeSeries() {
    const cutoff = Date.now() - this.config.trendWindow;
    const keysToDelete = [];

    for (const [key] of this.timeSeriesData) {
      if (key < cutoff) {
        keysToDelete.push(key);
      }
    }

    for (const key of keysToDelete) {
      this.timeSeriesData.delete(key);
    }
  }

  updateTrend() {
    if (this.metrics.riskScores.length === 0) return;

    const windowSize = 100;
    const recentScores = this.metrics.riskScores.slice(-windowSize);
    const avgScore = recentScores.reduce((sum, s) => sum + s, 0) / recentScores.length;

    const trend = {
      timestamp: Date.now(),
      averageRiskScore: avgScore,
      highRiskCount: recentScores.filter(s => s > 60).length,
      criticalRiskCount: recentScores.filter(s => s > 80).length,
      sampleSize: recentScores.length
    };

    this.metrics.trend.push(trend);

    if (this.metrics.trend.length > 100) {
      this.metrics.trend.shift();
    }
  }

  checkForAlerts() {
    const now = Date.now();
    const windowStart = now - this.config.windowSize;
    const recentData = this.getRecentMetrics(windowStart);

    if (recentData.totalRequests === 0) return;

    const highRiskRate = recentData.highRiskCount / recentData.totalRequests;
    const criticalRiskRate = recentData.criticalRiskCount / recentData.totalRequests;
    const anomalyRate = recentData.anomalyCount / recentData.totalRequests;
    const blockRate = recentData.blockCount / recentData.totalRequests;

    const alerts = [];

    if (highRiskRate > this.config.alertThreshold.highRiskRate) {
      alerts.push({
        type: 'high_risk_rate',
        severity: 'high',
        message: `High risk rate: ${(highRiskRate * 100).toFixed(2)}% exceeds threshold`,
        currentValue: highRiskRate,
        threshold: this.config.alertThreshold.highRiskRate
      });
    }

    if (criticalRiskRate > this.config.alertThreshold.criticalRiskRate) {
      alerts.push({
        type: 'critical_risk_rate',
        severity: 'critical',
        message: `Critical risk rate: ${(criticalRiskRate * 100).toFixed(2)}% exceeds threshold`,
        currentValue: criticalRiskRate,
        threshold: this.config.alertThreshold.criticalRiskRate
      });
    }

    if (anomalyRate > this.config.alertThreshold.anomalyRate) {
      alerts.push({
        type: 'high_anomaly_rate',
        severity: 'high',
        message: `Anomaly rate: ${(anomalyRate * 100).toFixed(2)}% exceeds threshold`,
        currentValue: anomalyRate,
        threshold: this.config.alertThreshold.anomalyRate
      });
    }

    if (blockRate > this.config.alertThreshold.blockRate) {
      alerts.push({
        type: 'high_block_rate',
        severity: 'warning',
        message: `Block rate: ${(blockRate * 100).toFixed(2)}% exceeds threshold`,
        currentValue: blockRate,
        threshold: this.config.alertThreshold.blockRate
      });
    }

    for (const alert of alerts) {
      this.triggerAlert({
        ...alert,
        timestamp: now,
        recentData
      });
    }
  }

  triggerAlert(alert) {
    alert.id = this.generateAlertId();
    this.metrics.alerts.push(alert);

    if (this.metrics.alerts.length > 100) {
      this.metrics.alerts.shift();
    }

    for (const callback of this.alertCallbacks) {
      try {
        callback(alert);
      } catch (error) {
        console.error('Error in alert callback:', error);
      }
    }
  }

  generateAlertId() {
    return `alert_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  onAlert(callback) {
    this.alertCallbacks.push(callback);
    return this;
  }

  offAlert(callback) {
    const index = this.alertCallbacks.indexOf(callback);
    if (index > -1) {
      this.alertCallbacks.splice(index, 1);
    }
    return this;
  }

  getRecentMetrics(since = Date.now() - this.config.windowSize) {
    let totalRequests = 0;
    let highRiskCount = 0;
    let criticalRiskCount = 0;
    let anomalyCount = 0;
    let blockCount = 0;

    for (const [windowKey, window] of this.timeSeriesData) {
      if (windowKey >= since) {
        totalRequests += window.requests;
        blockCount += window.blocks;
        anomalyCount += window.anomalies;

        for (const score of window.riskScore) {
          if (score > 80) criticalRiskCount++;
          else if (score > 60) highRiskCount++;
        }
      }
    }

    return {
      totalRequests,
      highRiskCount,
      criticalRiskCount,
      anomalyCount,
      blockCount,
      since
    };
  }

  calculateRiskTrend() {
    if (this.metrics.trend.length < 2) {
      return { direction: 'stable', change: 0 };
    }

    const recent = this.metrics.trend.slice(-5);
    const avgRecent = recent.reduce((sum, t) => sum + t.averageRiskScore, 0) / recent.length;

    const previous = this.metrics.trend.slice(-10, -5);
    const avgPrevious = previous.length > 0
      ? previous.reduce((sum, t) => sum + t.averageRiskScore, 0) / previous.length
      : avgRecent;

    const change = avgRecent - avgPrevious;
    const percentChange = avgPrevious > 0 ? (change / avgPrevious) * 100 : 0;

    let direction = 'stable';
    if (change > 5) direction = 'increasing';
    else if (change < -5) direction = 'decreasing';

    return {
      direction,
      change,
      percentChange,
      currentAverage: avgRecent,
      previousAverage: avgPrevious
    };
  }

  calculateEffectivenessMetrics() {
    const total = this.metrics.totalRequests;
    if (total === 0) {
      return this.getEmptyEffectivenessMetrics();
    }

    const allowRate = this.metrics.totalAllows / total;
    const verifyRate = this.metrics.totalVerifications / total;
    const blockRate = this.metrics.totalBlocks / total;

    const avgRiskScore = this.metrics.riskScores.length > 0
      ? this.metrics.riskScores.reduce((sum, s) => sum + s, 0) / this.metrics.riskScores.length
      : 0;

    const avgVerificationTime = this.metrics.verificationTimes.length > 0
      ? this.metrics.verificationTimes.reduce((sum, t) => sum + t, 0) / this.metrics.verificationTimes.length
      : 0;

    const captchaStats = this.calculateCaptchaStats();

    return {
      totalRequests: total,
      allowRate,
      verifyRate,
      blockRate,
      averageRiskScore: avgRiskScore,
      averageVerificationTime: avgVerificationTime,
      captchaStats,
      highRiskRate: this.metrics.riskLevels.high / total,
      criticalRiskRate: this.metrics.riskLevels.critical / total,
      anomalyRate: this.metrics.anomalyScores.length / total
    };
  }

  calculateCaptchaStats() {
    const stats = {};

    for (const [type, data] of Object.entries(this.metrics.captchaTypes)) {
      stats[type] = {
        count: data.count,
        successRate: data.successRate,
        avgTimeSpent: data.avgTimeSpent
      };
    }

    return stats;
  }

  getEmptyEffectivenessMetrics() {
    return {
      totalRequests: 0,
      allowRate: 0,
      verifyRate: 0,
      blockRate: 0,
      averageRiskScore: 0,
      averageVerificationTime: 0,
      captchaStats: {},
      highRiskRate: 0,
      criticalRiskRate: 0,
      anomalyRate: 0
    };
  }

  setBaseline(metrics) {
    this.baselineMetrics = { ...metrics };
  }

  compareToBaseline(current) {
    if (!this.baselineMetrics) {
      return null;
    }

    const comparison = {};

    for (const key of Object.keys(this.baselineMetrics)) {
      if (typeof current[key] === 'number' && typeof this.baselineMetrics[key] === 'number') {
        const change = current[key] - this.baselineMetrics[key];
        const percentChange = this.baselineMetrics[key] !== 0
          ? (change / this.baselineMetrics[key]) * 100
          : 0;

        comparison[key] = {
          current: current[key],
          baseline: this.baselineMetrics[key],
          change,
          percentChange
        };
      }
    }

    return comparison;
  }

  performHealthCheck() {
    const metrics = this.getHealthMetrics();
    
    const alerts = [];

    if (metrics.totalRequestsLastHour === 0) {
      alerts.push({
        type: 'no_traffic',
        severity: 'warning',
        message: 'No traffic in the last hour'
      });
    }

    const criticalRate = metrics.criticalRiskRate;
    if (criticalRate > 0.2) {
      alerts.push({
        type: 'elevated_critical_rate',
        severity: 'critical',
        message: `Critical risk rate elevated: ${(criticalRate * 100).toFixed(2)}%`
      });
    }

    return {
      healthy: alerts.length === 0,
      metrics,
      alerts
    };
  }

  getHealthMetrics() {
    const now = Date.now();
    const oneHourAgo = now - 3600000;
    const recentData = this.getRecentMetrics(oneHourAgo);

    return {
      totalRequestsLastHour: recentData.totalRequests,
      highRiskRate: recentData.totalRequests > 0 
        ? recentData.highRiskCount / recentData.totalRequests 
        : 0,
      criticalRiskRate: recentData.totalRequests > 0
        ? recentData.criticalRiskCount / recentData.totalRequests
        : 0,
      blockRate: recentData.totalRequests > 0
        ? recentData.blockCount / recentData.totalRequests
        : 0,
      anomalyRate: recentData.totalRequests > 0
        ? recentData.anomalyCount / recentData.totalRequests
        : 0
    };
  }

  getTrend() {
    return this.metrics.trend;
  }

  getAlerts(filter = {}) {
    let alerts = [...this.metrics.alerts];

    if (filter.severity) {
      alerts = alerts.filter(a => a.severity === filter.severity);
    }

    if (filter.type) {
      alerts = alerts.filter(a => a.type === filter.type);
    }

    if (filter.since) {
      alerts = alerts.filter(a => a.timestamp >= filter.since);
    }

    return alerts;
  }

  getTimeSeriesData(since = Date.now() - this.config.trendWindow) {
    const data = [];

    for (const [windowKey, window] of this.timeSeriesData) {
      if (windowKey >= since) {
        data.push({
          timestamp: windowKey,
          requests: window.requests,
          avgRiskScore: window.riskScore.length > 0
            ? window.riskScore.reduce((sum, s) => sum + s, 0) / window.riskScore.length
            : 0,
          anomalies: window.anomalies,
          allows: window.allows,
          verifications: window.verifications,
          blocks: window.blocks
        });
      }
    }

    return data.sort((a, b) => a.timestamp - b.timestamp);
  }

  getReport() {
    return {
      metrics: this.getFullMetrics(),
      effectiveness: this.calculateEffectivenessMetrics(),
      trend: this.calculateRiskTrend(),
      health: this.getHealthMetrics(),
      recentAlerts: this.metrics.alerts.slice(-10),
      baseline: this.baselineMetrics
    };
  }

  getFullMetrics() {
    return { ...this.metrics };
  }

  reset() {
    this.metrics = this.initializeMetrics();
    this.timeSeriesData.clear();
    this.metrics.alerts = [];
  }

  initializeMetrics() {
    return {
      totalRequests: 0,
      totalVerifications: 0,
      totalBlocks: 0,
      totalAllows: 0,
      riskScores: [],
      riskLevels: { low: 0, medium: 0, high: 0, critical: 0 },
      anomalyScores: [],
      verificationTimes: [],
      captchaTypes: {},
      alerts: [],
      trend: []
    };
  }

  destroy() {
    this.stopMonitoring();
    this.alertCallbacks = [];
  }
}

module.exports = RiskMonitor;
