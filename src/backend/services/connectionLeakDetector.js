const EventEmitter = require('events');
const fs = require('fs');
const path = require('path');

class ConnectionLeakDetector extends EventEmitter {
  constructor(dbPoolManager) {
    super();
    this.dbPoolManager = dbPoolManager;
    this.checkInterval = null;
    this.leakThreshold = parseInt(process.env.DB_LEAK_THRESHOLD) || 30000;
    this.checkIntervalMs = parseInt(process.env.DB_LEAK_CHECK_INTERVAL) || 10000;
    this.maxLeakRecords = parseInt(process.env.DB_LEAK_MAX_RECORDS) || 100;
    this.enableAutoCleanup = process.env.DB_LEAK_AUTO_CLEANUP === 'true';
    this.autoCleanupTimeout = parseInt(process.env.DB_LEAK_AUTO_CLEANUP_TIMEOUT) || 60000;
    
    this.connectionTracker = new Map();
    this.leakEvents = [];
    this.statistics = {
      totalCheckedOut: 0,
      totalReleased: 0,
      potentialLeaks: 0,
      confirmedLeaks: 0,
      falsePositives: 0,
      autoCleanups: 0
    };
    this.isMonitoring = false;
    this.reportFile = path.join(__dirname, '../../../logs/connection-leaks.json');
    
    this._initializeLogging();
    this._setupPoolEvents();
  }

  _initializeLogging() {
    const logDir = path.dirname(this.reportFile);
    if (!fs.existsSync(logDir)) {
      fs.mkdirSync(logDir, { recursive: true });
    }
  }

  _setupPoolEvents() {
    if (this.dbPoolManager && this.dbPoolManager.checkedOutClients) {
      this._monitorCheckedOutConnections();
    }
  }

  _monitorCheckedOutConnections() {
    setInterval(() => {
      this._checkForLeaks();
    }, this.checkIntervalMs);
  }

  start() {
    if (this.isMonitoring) {
      return;
    }

    this.isMonitoring = true;
    
    if (this.checkInterval) {
      clearInterval(this.checkInterval);
    }

    this.checkInterval = setInterval(() => {
      this._checkForLeaks();
    }, this.checkIntervalMs);

    this.checkInterval.unref();
    console.log(`Connection leak detection started with threshold: ${this.leakThreshold}ms, check interval: ${this.checkIntervalMs}ms`);
  }

  stop() {
    if (this.checkInterval) {
      clearInterval(this.checkInterval);
      this.checkInterval = null;
    }
    this.isMonitoring = false;
    console.log('Connection leak detection stopped');
  }

  trackConnection(clientId, connectionInfo = {}) {
    const trackingInfo = {
      clientId,
      checkedOutAt: Date.now(),
      expectedReleaseTime: Date.now() + this.leakThreshold,
      stackTrace: new Error().stack,
      query: connectionInfo.query || null,
      user: connectionInfo.user || null,
      releaseAttempts: 0,
      lastReleaseAttempt: null,
      status: 'active'
    };

    this.connectionTracker.set(clientId, trackingInfo);
    this.statistics.totalCheckedOut++;

    this.emit('connectionTracked', trackingInfo);
    
    return trackingInfo;
  }

  untrackConnection(clientId) {
    const trackingInfo = this.connectionTracker.get(clientId);
    
    if (trackingInfo) {
      const releaseDuration = Date.now() - trackingInfo.checkedOutAt;
      const wasLeaked = releaseDuration > this.leakThreshold;
      
      trackingInfo.releasedAt = Date.now();
      trackingInfo.releaseDuration = releaseDuration;
      trackingInfo.wasLeaked = wasLeaked;
      trackingInfo.status = 'released';

      if (wasLeaked) {
        this._recordPotentialLeak(trackingInfo);
      } else {
        this.statistics.totalReleased++;
      }

      this.connectionTracker.delete(clientId);
      this.emit('connectionUntracked', trackingInfo);
      
      return trackingInfo;
    }

    return null;
  }

  _checkForLeaks() {
    if (!this.dbPoolManager || !this.dbPoolManager.checkedOutClients) {
      return;
    }

    const now = Date.now();
    const leakedConnections = [];

    for (const [clientId, trackingInfo] of this.connectionTracker) {
      const holdDuration = now - trackingInfo.checkedOutAt;
      
      if (holdDuration > this.leakThreshold) {
        leakedConnections.push({
          clientId,
          ...trackingInfo,
          holdDuration
        });

        this.statistics.potentialLeaks++;
        
        if (this.enableAutoCleanup && holdDuration > this.autoCleanupTimeout) {
          this._attemptAutoCleanup(clientId, trackingInfo);
        }
      }
    }

    if (leakedConnections.length > 0) {
      this._handleDetectedLeaks(leakedConnections);
    }

    this._updateExpectedReleaseTimes();
  }

  _recordPotentialLeak(trackingInfo) {
    const leakEvent = {
      id: `leak_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      clientId: trackingInfo.clientId,
      checkedOutAt: new Date(trackingInfo.checkedOutAt).toISOString(),
      releasedAt: new Date(trackingInfo.releasedAt).toISOString(),
      holdDuration: trackingInfo.releaseDuration,
      stackTrace: trackingInfo.stackTrace,
      query: trackingInfo.query,
      user: trackingInfo.user,
      severity: this._calculateSeverity(trackingInfo.releaseDuration),
      timestamp: new Date().toISOString()
    };

    this.leakEvents.push(leakEvent);
    
    if (this.leakEvents.length > this.maxLeakRecords) {
      this.leakEvents.shift();
    }

    this.emit('potentialLeak', leakEvent);
    
    return leakEvent;
  }

  _handleDetectedLeaks(leakedConnections) {
    const alert = {
      timestamp: new Date().toISOString(),
      count: leakedConnections.length,
      totalDuration: leakedConnections.reduce((sum, c) => sum + c.holdDuration, 0),
      connections: leakedConnections.map(c => ({
        clientId: c.clientId,
        holdDuration: c.holdDuration,
        severity: this._calculateSeverity(c.holdDuration),
        query: c.query,
        user: c.user
      })),
      averageDuration: leakedConnections.reduce((sum, c) => sum + c.holdDuration, 0) / leakedConnections.length
    };

    console.warn('Connection leaks detected:', JSON.stringify(alert, null, 2));
    
    this.emit('leaksDetected', alert);
    
    if (leakedConnections.length >= 5 || this.leakEvents.length >= 10) {
      this._emitCriticalAlert(alert);
    }

    this._logLeakReport(alert);
  }

  _calculateSeverity(duration) {
    if (duration > 300000) {
      return 'critical';
    } else if (duration > 120000) {
      return 'high';
    } else if (duration > 60000) {
      return 'medium';
    } else {
      return 'low';
    }
  }

  _emitCriticalAlert(alert) {
    console.error('CRITICAL: Multiple connection leaks detected!');
    this.emit('criticalLeakAlert', {
      ...alert,
      message: `Critical: ${alert.count} connection leaks detected`,
      requireImmediateAction: true
    });
  }

  _attemptAutoCleanup(clientId, trackingInfo) {
    if (trackingInfo.status === 'cleaned') {
      return;
    }

    const client = this.dbPoolManager.checkedOutClients.get(clientId);
    
    if (client && client.client && typeof client.client.release === 'function') {
      try {
        client.client.release();
        trackingInfo.status = 'cleaned';
        trackingInfo.cleanedAt = Date.now();
        this.statistics.autoCleanups++;
        
        this.emit('autoCleanup', {
          clientId,
          holdDuration: Date.now() - trackingInfo.checkedOutAt,
          timestamp: new Date().toISOString()
        });
        
        console.warn(`Auto-cleaned leaked connection: ${clientId}`);
      } catch (error) {
        console.error(`Failed to auto-clean connection ${clientId}:`, error);
      }
    }
  }

  _updateExpectedReleaseTimes() {
    for (const [clientId, trackingInfo] of this.connectionTracker) {
      const holdDuration = Date.now() - trackingInfo.checkedOutAt;
      const expectedDuration = this.leakThreshold - holdDuration;
      
      trackingInfo.expectedReleaseTime = Date.now() + expectedDuration;
      trackingInfo.remainingTime = expectedDuration;
      trackingInfo.percentageHeld = (holdDuration / this.leakThreshold) * 100;
    }
  }

  _logLeakReport(alert) {
    try {
      const report = {
        generatedAt: new Date().toISOString(),
        currentLeaks: alert,
        recentLeaks: this.leakEvents.slice(-20),
        statistics: this.statistics,
        threshold: this.leakThreshold,
        activeTrackedConnections: this.connectionTracker.size,
        leakTrend: this._calculateLeakTrend()
      };

      fs.writeFileSync(this.reportFile, JSON.stringify(report, null, 2));
    } catch (error) {
      console.error('Failed to write leak report:', error);
    }
  }

  _calculateLeakTrend() {
    if (this.leakEvents.length < 2) {
      return 'stable';
    }

    const recentLeaks = this.leakEvents.slice(-10);
    const olderLeaks = this.leakEvents.slice(-20, -10);
    
    if (olderLeaks.length === 0) {
      return recentLeaks.length > 0 ? 'increasing' : 'stable';
    }

    const recentCount = recentLeaks.length;
    const olderCount = olderLeaks.length;
    
    if (recentCount > olderCount * 1.5) {
      return 'increasing';
    } else if (recentCount < olderCount * 0.5) {
      return 'decreasing';
    }
    
    return 'stable';
  }

  getStatistics() {
    return {
      ...this.statistics,
      activeTrackedConnections: this.connectionTracker.size,
      leakEventsCount: this.leakEvents.length,
      leakTrend: this._calculateLeakTrend(),
      averageLeakDuration: this.leakEvents.length > 0
        ? this.leakEvents.reduce((sum, e) => sum + e.holdDuration, 0) / this.leakEvents.length
        : 0,
      leakRate: this.statistics.totalCheckedOut > 0
        ? (this.leakEvents.length / this.statistics.totalCheckedOut) * 100
        : 0
    };
  }

  getLeakReport(limit = 50) {
    return {
      summary: this.getStatistics(),
      recentLeaks: this.leakEvents.slice(-limit),
      bySeverity: this._groupLeaksBySeverity(),
      byDuration: this._groupLeaksByDuration(),
      byTimeRange: this._groupLeaksByTimeRange()
    };
  }

  _groupLeaksBySeverity() {
    const grouped = { critical: 0, high: 0, medium: 0, low: 0 };
    
    for (const leak of this.leakEvents) {
      if (grouped[leak.severity] !== undefined) {
        grouped[leak.severity]++;
      }
    }
    
    return grouped;
  }

  _groupLeaksByDuration() {
    const buckets = {
      '< 1min': 0,
      '1-5min': 0,
      '5-15min': 0,
      '15-30min': 0,
      '> 30min': 0
    };

    for (const leak of this.leakEvents) {
      const duration = leak.holdDuration / 1000 / 60;
      
      if (duration < 1) {
        buckets['< 1min']++;
      } else if (duration < 5) {
        buckets['1-5min']++;
      } else if (duration < 15) {
        buckets['5-15min']++;
      } else if (duration < 30) {
        buckets['15-30min']++;
      } else {
        buckets['> 30min']++;
      }
    }

    return buckets;
  }

  _groupLeaksByTimeRange() {
    const now = Date.now();
    const hourAgo = now - 3600000;
    const dayAgo = now - 86400000;
    const weekAgo = now - 604800000;

    const grouped = {
      lastHour: 0,
      lastDay: 0,
      lastWeek: 0,
      older: 0
    };

    for (const leak of this.leakEvents) {
      const timestamp = new Date(leak.timestamp).getTime();
      
      if (timestamp > hourAgo) {
        grouped.lastHour++;
      } else if (timestamp > dayAgo) {
        grouped.lastDay++;
      } else if (timestamp > weekAgo) {
        grouped.lastWeek++;
      } else {
        grouped.older++;
      }
    }

    return grouped;
  }

  getActiveConnections() {
    const connections = [];
    
    for (const [clientId, trackingInfo] of this.connectionTracker) {
      const holdDuration = Date.now() - trackingInfo.checkedOutAt;
      
      connections.push({
        clientId,
        holdDuration,
        checkedOutAt: new Date(trackingInfo.checkedOutAt).toISOString(),
        query: trackingInfo.query,
        user: trackingInfo.user,
        percentageHeld: (holdDuration / this.leakThreshold) * 100,
        status: holdDuration > this.leakThreshold ? 'potential_leak' : 'normal',
        stackTrace: trackingInfo.stackTrace
      });
    }

    return connections.sort((a, b) => b.holdDuration - a.holdDuration);
  }

  getConnectionDetails(clientId) {
    return this.connectionTracker.get(clientId) || null;
  }

  setThreshold(thresholdMs) {
    this.leakThreshold = thresholdMs;
    console.log(`Leak threshold updated to: ${thresholdMs}ms`);
    this.emit('thresholdChanged', { threshold: thresholdMs });
  }

  setCheckInterval(intervalMs) {
    this.checkIntervalMs = intervalMs;
    
    if (this.isMonitoring) {
      this.stop();
      this.start();
    }
    
    console.log(`Leak check interval updated to: ${intervalMs}ms`);
    this.emit('intervalChanged', { interval: intervalMs });
  }

  markAsFalsePositive(clientId) {
    const leakEvent = this.leakEvents.find(e => e.clientId === clientId);
    
    if (leakEvent) {
      leakEvent.falsePositive = true;
      leakEvent.markedAt = new Date().toISOString();
      this.statistics.falsePositives++;
      
      this.emit('falsePositiveMarked', leakEvent);
      return true;
    }
    
    return false;
  }

  clearLeakHistory() {
    this.leakEvents = [];
    console.log('Leak history cleared');
    this.emit('historyCleared');
  }

  resetStatistics() {
    this.statistics = {
      totalCheckedOut: 0,
      totalReleased: 0,
      potentialLeaks: 0,
      confirmedLeaks: 0,
      falsePositives: 0,
      autoCleanups: 0
    };
    console.log('Leak detection statistics reset');
    this.emit('statisticsReset');
  }

  forceCheck() {
    return this._checkForLeaks();
  }

  exportLeakData(format = 'json') {
    const data = {
      exportedAt: new Date().toISOString(),
      leakEvents: this.leakEvents,
      statistics: this.getStatistics(),
      activeConnections: this.getActiveConnections()
    };

    if (format === 'csv') {
      return this._convertToCSV(data.leakEvents);
    }

    return JSON.stringify(data, null, 2);
  }

  _convertToCSV(leakEvents) {
    if (leakEvents.length === 0) {
      return 'No leak events to export';
    }

    const headers = ['ID', 'Client ID', 'Checked Out At', 'Released At', 'Hold Duration (ms)', 'Severity', 'Query', 'User'];
    const rows = leakEvents.map(event => [
      event.id,
      event.clientId,
      event.checkedOutAt,
      event.releasedAt,
      event.holdDuration,
      event.severity,
      event.query || '',
      event.user || ''
    ]);

    return [
      headers.join(','),
      ...rows.map(row => row.map(cell => `"${cell}"`).join(','))
    ].join('\n');
  }
}

module.exports = ConnectionLeakDetector;
