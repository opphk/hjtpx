const WebSocket = require('ws');
const http = require('http');
const fs = require('fs');
const path = require('path');

const MONITOR_CONFIG = {
  wsUrl: process.env.WS_MONITOR_URL || 'ws://localhost:3000',
  httpUrl: process.env.HTTP_MONITOR_URL || 'http://localhost:3000',
  checkInterval: parseInt(process.env.MONITOR_INTERVAL || '5000'),
  metricsFile: process.env.METRICS_FILE || '/workspace/hjtpx/monitoring/websocket-metrics.json',
  alertThreshold: {
    connectionFailureRate: 0.1,
    avgLatency: 500,
    memoryUsage: 0.85,
    cpuUsage: 0.90,
  },
};

class WebSocketMonitor {
  constructor() {
    this.metrics = {
      timestamp: new Date().toISOString(),
      connections: {
        total: 0,
        active: 0,
        failed: 0,
        closed: 0,
      },
      messages: {
        sent: 0,
        received: 0,
        failed: 0,
      },
      performance: {
        avgLatency: 0,
        minLatency: 0,
        maxLatency: 0,
        throughput: 0,
      },
      system: {
        memoryUsage: 0,
        cpuUsage: 0,
        uptime: 0,
      },
      alerts: [],
    };

    this.connection = null;
    this.latencies = [];
    this.messageCount = 0;
    this.startTime = Date.now();
  }

  async checkServerHealth() {
    return new Promise((resolve) => {
      const start = Date.now();
      http.get(`${MONITOR_CONFIG.httpUrl}/api/v1/health`, (res) => {
        const latency = Date.now() - start;
        resolve({
          status: res.statusCode === 200 ? 'healthy' : 'unhealthy',
          latency,
          statusCode: res.statusCode,
        });
      }).on('error', (err) => {
        resolve({
          status: 'unreachable',
          latency: Date.now() - start,
          error: err.message,
        });
      });
    });
  }

  connect() {
    return new Promise((resolve, reject) => {
      console.log(`Connecting to WebSocket server: ${MONITOR_CONFIG.wsUrl}`);

      this.connection = new WebSocket(MONITOR_CONFIG.wsUrl, {
        handshakeTimeout: 5000,
      });

      this.connection.on('open', () => {
        console.log('WebSocket connection established');

        this.connection.send(JSON.stringify({
          type: 'auth',
          userId: 'monitor',
          token: process.env.WS_MONITOR_TOKEN || 'monitor-token',
        }));

        this.connection.send(JSON.stringify({
          type: 'subscribe',
          room: 'monitoring',
        }));

        setTimeout(() => {
          this.connection.send(JSON.stringify({
            type: 'ping',
            timestamp: Date.now(),
          }));
        }, 1000);

        resolve();
      });

      this.connection.on('message', (data) => {
        try {
          const message = JSON.parse(data);

          if (message.type === 'pong' && message.timestamp) {
            const latency = Date.now() - message.timestamp;
            this.latencies.push(latency);

            if (this.latencies.length > 100) {
              this.latencies.shift();
            }

            this.metrics.performance.avgLatency =
              this.latencies.reduce((a, b) => a + b, 0) / this.latencies.length;
            this.metrics.performance.minLatency = Math.min(...this.latencies);
            this.metrics.performance.maxLatency = Math.max(...this.latencies);
          }

          if (message.type === 'broadcast') {
            this.messageCount++;
          }

          if (message.metrics) {
            this.metrics.connections = {
              ...this.metrics.connections,
              ...message.metrics.connections,
            };
          }
        } catch (e) {
        }
      });

      this.connection.on('error', (error) => {
        console.error('WebSocket error:', error.message);
        this.metrics.alerts.push({
          type: 'connection_error',
          message: error.message,
          timestamp: new Date().toISOString(),
        });
      });

      this.connection.on('close', () => {
        console.log('WebSocket connection closed');
        this.metrics.connections.closed++;
      });

      setTimeout(() => {
        if (this.connection.readyState !== WebSocket.OPEN) {
          reject(new Error('Connection timeout'));
        }
      }, 5000);
    });
  }

  sendHeartbeat() {
    if (this.connection && this.connection.readyState === WebSocket.OPEN) {
      this.connection.send(JSON.stringify({
        type: 'ping',
        timestamp: Date.now(),
      }));
    }
  }

  updateSystemMetrics() {
    const memUsage = process.memoryUsage();
    this.metrics.system.memoryUsage = memUsage.heapUsed / memUsage.heapTotal;
    this.metrics.system.uptime = Date.now() - this.startTime;

    const os = require('os');
    this.metrics.system.cpuUsage = os.loadavg()[0] / os.cpus().length;
  }

  checkAlerts() {
    const alerts = [];

    if (this.metrics.connections.total > 0) {
      const failureRate = this.metrics.connections.failed / this.metrics.connections.total;
      if (failureRate > MONITOR_CONFIG.alertThreshold.connectionFailureRate) {
        alerts.push({
          type: 'high_failure_rate',
          severity: 'warning',
          message: `Connection failure rate: ${(failureRate * 100).toFixed(2)}%`,
          threshold: MONITOR_CONFIG.alertThreshold.connectionFailureRate,
          current: failureRate,
        });
      }
    }

    if (this.metrics.performance.avgLatency > MONITOR_CONFIG.alertThreshold.avgLatency) {
      alerts.push({
        type: 'high_latency',
        severity: 'warning',
        message: `Average latency: ${this.metrics.performance.avgLatency.toFixed(2)}ms`,
        threshold: MONITOR_CONFIG.alertThreshold.avgLatency,
        current: this.metrics.performance.avgLatency,
      });
    }

    if (this.metrics.system.memoryUsage > MONITOR_CONFIG.alertThreshold.memoryUsage) {
      alerts.push({
        type: 'high_memory_usage',
        severity: 'critical',
        message: `Memory usage: ${(this.metrics.system.memoryUsage * 100).toFixed(2)}%`,
        threshold: MONITOR_CONFIG.alertThreshold.memoryUsage,
        current: this.metrics.system.memoryUsage,
      });
    }

    if (alerts.length > 0) {
      this.metrics.alerts = [...this.metrics.alerts.slice(-50), ...alerts];

      if (process.env.SLACK_WEBHOOK_URL) {
        this.sendSlackNotification(alerts);
      }
    }

    return alerts;
  }

  sendSlackNotification(alerts) {
    const webhookUrl = process.env.SLACK_WEBHOOK_URL;
    const message = alerts.map(a => `⚠️ *${a.severity.toUpperCase()}*: ${a.message}`).join('\n');

    http.request({
      method: 'POST',
      hostname: new URL(webhookUrl).hostname,
      path: new URL(webhookUrl).pathname,
      headers: { 'Content-Type': 'application/json' },
    }, () => {}).end(JSON.stringify({ text: message }));
  }

  saveMetrics() {
    const metricsDir = path.dirname(MONITOR_CONFIG.metricsFile);
    if (!fs.existsSync(metricsDir)) {
      fs.mkdirSync(metricsDir, { recursive: true });
    }

    this.metrics.timestamp = new Date().toISOString();
    fs.writeFileSync(
      MONITOR_CONFIG.metricsFile,
      JSON.stringify(this.metrics, null, 2)
    );
  }

  async start() {
    console.log('Starting WebSocket Monitor...');
    console.log(`Check interval: ${MONITOR_CONFIG.checkInterval}ms`);

    try {
      await this.connect();

      const heartbeatInterval = setInterval(() => this.sendHeartbeat(), 30000);
      const metricsInterval = setInterval(() => {
        this.updateSystemMetrics();
        this.checkAlerts();
        this.saveMetrics();

        console.log(`[${new Date().toISOString()}] Metrics updated`);
        console.log(`  Active connections: ${this.metrics.connections.active}`);
        console.log(`  Avg latency: ${this.metrics.performance.avgLatency.toFixed(2)}ms`);
        console.log(`  Memory: ${(this.metrics.system.memoryUsage * 100).toFixed(1)}%`);
      }, MONITOR_CONFIG.checkInterval);

      process.on('SIGINT', () => {
        console.log('\nShutting down monitor...');
        clearInterval(heartbeatInterval);
        clearInterval(metricsInterval);
        if (this.connection) {
          this.connection.close();
        }
        process.exit(0);
      });

    } catch (error) {
      console.error('Failed to start monitor:', error.message);
      process.exit(1);
    }
  }

  getMetrics() {
    return this.metrics;
  }
}

if (require.main === module) {
  const monitor = new WebSocketMonitor();
  monitor.start();
}

module.exports = WebSocketMonitor;
