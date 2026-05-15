const fs = require('fs');
const path = require('path');
const { LogSanitizer } = require('./log_sanitizer');

class AuditLogger {
  constructor(options = {}) {
    this.logDir = options.logDir || process.env.AUDIT_LOG_DIR || 'logs/audit';
    this.logFile = options.logFile || 'audit.log';
    this.maxFileSize = options.maxFileSize || 100 * 1024 * 1024;
    this.maxFiles = options.maxFiles || 30;
    this.enabled = options.enabled !== false;
    this.buffers = new Map();
    this.flushInterval = options.flushInterval || 60000;
    this.retentionDays = options.retentionDays || 365;
    
    this.init();
    this.startAutoFlush();
    this.startCleanup();
  }

  init() {
    if (!this.enabled) {
      return;
    }

    const fullPath = path.join(this.logDir, this.logFile);
    const dir = path.dirname(fullPath);

    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }

    this.currentLogFile = fullPath;
  }

  log(action, userId, details = {}, req = null) {
    if (!this.enabled) {
      return;
    }

    const entry = {
      timestamp: new Date().toISOString(),
      action,
      userId,
      ip: req?.ip || req?.connection?.remoteAddress || 'unknown',
      userAgent: req?.headers?.['user-agent'] || 'unknown',
      method: req?.method || 'N/A',
      path: req?.path || 'N/A',
      details: LogSanitizer.sanitize(details),
      severity: this.getSeverity(action),
      correlationId: req?.id || this.generateCorrelationId()
    };

    this.writeLog(entry);
  }

  writeLog(entry) {
    const logLine = JSON.stringify(entry) + '\n';
    const buffer = this.buffers.get(this.currentLogFile);

    if (buffer) {
      buffer.push(logLine);
      if (buffer.length >= 100) {
        this.flush(this.currentLogFile);
      }
    } else {
      this.buffers.set(this.currentLogFile, [logLine]);
    }
  }

  flush(logFile = this.currentLogFile) {
    const buffer = this.buffers.get(logFile);
    if (!buffer || buffer.length === 0) {
      return;
    }

    try {
      fs.appendFileSync(logFile, buffer.join(''));
      this.buffers.set(logFile, []);

      const stats = fs.statSync(logFile);
      if (stats.size > this.maxFileSize) {
        this.rotateLog(logFile);
      }
    } catch (error) {
      console.error('Audit log write error:', error);
    }
  }

  rotateLog(logFile) {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const rotatedFile = logFile.replace('.log', `-${timestamp}.log`);
    
    try {
      if (fs.existsSync(logFile)) {
        fs.renameSync(logFile, rotatedFile);
        this.compressOldLogs();
      }
    } catch (error) {
      console.error('Log rotation error:', error);
    }
  }

  compressOldLogs() {
    const files = fs.readdirSync(this.logDir)
      .filter(f => f.endsWith('.log') && !f.endsWith('.gz'))
      .map(f => ({
        name: f,
        path: path.join(this.logDir, f),
        mtime: fs.statSync(path.join(this.logDir, f)).mtime
      }))
      .sort((a, b) => b.mtime - a.mtime);

    if (files.length > this.maxFiles) {
      const toCompress = files.slice(this.maxFiles);
      for (const file of toCompress) {
        try {
          const { execSync } = require('child_process');
          execSync(`gzip -9 "${file.path}"`);
        } catch (error) {
          console.error(`Failed to compress ${file.name}:`, error);
        }
      }
    }
  }

  startAutoFlush() {
    setInterval(() => {
      for (const logFile of this.buffers.keys()) {
        this.flush(logFile);
      }
    }, this.flushInterval);
  }

  startCleanup() {
    setInterval(() => {
      this.cleanupOldLogs();
    }, 24 * 60 * 60 * 1000);
  }

  cleanupOldLogs() {
    const cutoff = Date.now() - (this.retentionDays * 24 * 60 * 60 * 1000);
    
    try {
      const files = fs.readdirSync(this.logDir);
      for (const file of files) {
        const filePath = path.join(this.logDir, file);
        const stats = fs.statSync(filePath);
        
        if (stats.mtimeMs < cutoff) {
          fs.unlinkSync(filePath);
        }
      }
    } catch (error) {
      console.error('Log cleanup error:', error);
    }
  }

  query(filters = {}) {
    const results = [];
    const searchPatterns = this.buildSearchPatterns(filters);
    
    try {
      const content = fs.readFileSync(this.currentLogFile, 'utf-8');
      const lines = content.split('\n').filter(line => line.trim());

      for (const line of lines) {
        try {
          const entry = JSON.parse(line);
          if (this.matchesFilters(entry, filters)) {
            results.push(entry);
          }
        } catch {
          continue;
        }
      }
    } catch (error) {
      console.error('Audit log query error:', error);
    }

    return results;
  }

  matchesFilters(entry, filters) {
    if (filters.action && entry.action !== filters.action) return false;
    if (filters.userId && entry.userId !== filters.userId) return false;
    if (filters.ip && entry.ip !== filters.ip) return false;
    if (filters.severity && entry.severity !== filters.severity) return false;
    if (filters.startDate && new Date(entry.timestamp) < new Date(filters.startDate)) return false;
    if (filters.endDate && new Date(entry.timestamp) > new Date(filters.endDate)) return false;
    
    return true;
  }

  buildSearchPatterns(filters) {
    return [];
  }

  getSeverity(action) {
    const critical = ['DELETE_ACCOUNT', 'PERMISSION_CHANGE', 'SECURITY_BREACH', 'DATA_EXPORT'];
    const high = ['LOGIN', 'PASSWORD_CHANGE', 'API_KEY_CREATE', 'SETTINGS_CHANGE'];
    const medium = ['UPDATE_PROFILE', 'PASSWORD_RESET', 'LOGOUT'];
    const low = ['VIEW', 'LIST', 'SEARCH'];

    if (critical.includes(action)) return 'critical';
    if (high.includes(action)) return 'high';
    if (medium.includes(action)) return 'medium';
    if (low.includes(action)) return 'low';
    return 'info';
  }

  generateCorrelationId() {
    return `audit-${Date.now()}-${Math.random().toString(36).substring(2, 15)}`;
  }

  getStats() {
    return {
      enabled: this.enabled,
      logDir: this.logDir,
      currentLogFile: this.currentLogFile,
      bufferSize: Array.from(this.buffers.values()).reduce((sum, arr) => sum + arr.length, 0),
      retentionDays: this.retentionDays
    };
  }

  close() {
    this.flush();
  }
}

const defaultAuditLogger = new AuditLogger();

module.exports = { AuditLogger, defaultAuditLogger };
