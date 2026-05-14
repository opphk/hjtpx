const http = require('http');
const EventEmitter = require('events');

class ServiceDiscovery extends EventEmitter {
  constructor(options = {}) {
    super();
    this.services = new Map();
    this.serviceVersions = new Map();
    this.heartbeatInterval = options.heartbeatInterval || 10000;
    this.healthCheckInterval = options.healthCheckInterval || 30000;
    this.serviceTimeout = options.serviceTimeout || 60000;
    this.versionStrategy = options.versionStrategy || 'semver';
    this.heartbeats = new Map();
    this.healthChecks = new Map();
  }

  registerService(serviceInfo) {
    const { name, version, url, metadata = {} } = serviceInfo;

    if (!name || !url) {
      throw new Error('Service name and URL are required');
    }

    const serviceId = `${name}:${version || '1.0.0'}:${this.generateId()}`;

    const service = {
      id: serviceId,
      name,
      version: version || '1.0.0',
      url,
      metadata,
      status: 'starting',
      registeredAt: new Date().toISOString(),
      lastHeartbeat: Date.now(),
      healthCheckUrl: `${url}/health`,
      requestCount: 0,
      avgResponseTime: 0,
      errorCount: 0,
      consecutiveFailures: 0,
      tags: metadata.tags || []
    };

    this.services.set(serviceId, service);

    if (!this.serviceVersions.has(name)) {
      this.serviceVersions.set(name, []);
    }
    this.serviceVersions.get(name).push(service);

    this.startHeartbeat(serviceId);
    this.scheduleHealthCheck(serviceId);

    this.emit('service:registered', service);

    console.log(`Service registered: ${name}@${version} (${serviceId})`);

    setTimeout(() => {
      const s = this.services.get(serviceId);
      if (s) s.status = 'healthy';
    }, 5000);

    return serviceId;
  }

  unregisterService(serviceId) {
    const service = this.services.get(serviceId);

    if (!service) {
      return false;
    }

    this.stopHeartbeat(serviceId);
    this.stopHealthCheck(serviceId);

    const versionList = this.serviceVersions.get(service.name);
    if (versionList) {
      const index = versionList.findIndex(s => s.id === serviceId);
      if (index !== -1) {
        versionList.splice(index, 1);
      }
    }

    this.services.delete(serviceId);

    this.emit('service:unregistered', service);

    console.log(`Service unregistered: ${service.name} (${serviceId})`);

    return true;
  }

  discoverService(name, options = {}) {
    const { version, tags = [], healthyOnly = true } = options;

    let candidates = this.serviceVersions.get(name) || [];

    if (healthyOnly) {
      candidates = candidates.filter(s => s.status === 'healthy');
    }

    if (version) {
      candidates = candidates.filter(s => this.versionMatches(s.version, version));
    }

    if (tags.length > 0) {
      candidates = candidates.filter(s =>
        tags.every(tag => s.tags.includes(tag))
      );
    }

    if (candidates.length === 0) {
      return null;
    }

    return candidates[Math.floor(Math.random() * candidates.length)];
  }

  discoverAllServices(name, options = {}) {
    const { version, tags = [], healthyOnly = true } = options;

    let candidates = this.serviceVersions.get(name) || [];

    if (healthyOnly) {
      candidates = candidates.filter(s => s.status === 'healthy');
    }

    if (version) {
      candidates = candidates.filter(s => this.versionMatches(s.version, version));
    }

    if (tags.length > 0) {
      candidates = candidates.filter(s =>
        tags.every(tag => s.tags.includes(tag))
      );
    }

    return candidates;
  }

  versionMatches(serviceVersion, requestedVersion) {
    if (this.versionStrategy !== 'semver') {
      return serviceVersion === requestedVersion;
    }

    const [sMajor, sMinor] = serviceVersion.split('.').map(Number);
    const [rMajor, rMinor] = requestedVersion.split('.').map(Number);

    if (sMajor !== rMajor) return false;
    return sMinor >= rMinor;
  }

  startHeartbeat(serviceId) {
    const interval = setInterval(() => {
      const service = this.services.get(serviceId);
      if (!service) {
        clearInterval(interval);
        return;
      }

      service.lastHeartbeat = Date.now();
      this.emit('service:heartbeat', { serviceId, timestamp: service.lastHeartbeat });
    }, this.heartbeatInterval);

    this.heartbeats.set(serviceId, interval);
  }

  stopHeartbeat(serviceId) {
    const interval = this.heartbeats.get(serviceId);
    if (interval) {
      clearInterval(interval);
      this.heartbeats.delete(serviceId);
    }
  }

  scheduleHealthCheck(serviceId) {
    const check = async () => {
      const service = this.services.get(serviceId);
      if (!service) return;

      try {
        await this.performHealthCheck(service);
        service.consecutiveFailures = 0;
        service.status = 'healthy';
        this.emit('service:healthy', service);
      } catch (error) {
        service.consecutiveFailures++;
        service.errorCount++;

        if (service.consecutiveFailures >= 3) {
          service.status = 'unhealthy';
          this.emit('service:unhealthy', { service, error: error.message });
        }

        if (service.consecutiveFailures >= 5) {
          this.unregisterService(serviceId);
          this.emit('service:failed', { service, error: error.message });
        }
      }
    };

    const interval = setInterval(check, this.healthCheckInterval);
    this.healthChecks.set(serviceId, interval);

    check();
  }

  stopHealthCheck(serviceId) {
    const interval = this.healthChecks.get(serviceId);
    if (interval) {
      clearInterval(interval);
      this.healthChecks.delete(serviceId);
    }
  }

  async performHealthCheck(service) {
    return new Promise((resolve, reject) => {
      const url = new URL(service.healthCheckUrl);
      const req = http.request(url, { method: 'GET', timeout: 5000 }, (res) => {
        if (res.statusCode >= 200 && res.statusCode < 300) {
          resolve(res);
        } else {
          reject(new Error(`Health check failed with status ${res.statusCode}`));
        }
      });

      req.on('error', reject);
      req.on('timeout', () => {
        req.destroy();
        reject(new Error('Health check timeout'));
      });

      req.end();
    });
  }

  updateServiceMetrics(serviceId, responseTime, success = true) {
    const service = this.services.get(serviceId);
    if (!service) return;

    service.requestCount++;
    service.avgResponseTime =
      (service.avgResponseTime * (service.requestCount - 1) + responseTime) /
      service.requestCount;

    if (!success) {
      service.errorCount++;
    }
  }

  getService(serviceId) {
    return this.services.get(serviceId);
  }

  getServicesByName(name) {
    return this.serviceVersions.get(name) || [];
  }

  getAllServices() {
    return Array.from(this.services.values());
  }

  getHealthyServices() {
    return Array.from(this.services.values()).filter(s => s.status === 'healthy');
  }

  getServiceStats(serviceId) {
    const service = this.services.get(serviceId);
    if (!service) return null;

    return {
      id: service.id,
      name: service.name,
      version: service.version,
      status: service.status,
      requestCount: service.requestCount,
      avgResponseTime: service.avgResponseTime,
      errorCount: service.errorCount,
      errorRate:
        service.requestCount > 0
          ? ((service.errorCount / service.requestCount) * 100).toFixed(2) + '%'
          : '0%',
      uptime: Date.now() - new Date(service.registeredAt).getTime()
    };
  }

  getDiscoveryStats() {
    const services = Array.from(this.services.values());
    const byStatus = {
      healthy: services.filter(s => s.status === 'healthy').length,
      unhealthy: services.filter(s => s.status === 'unhealthy').length,
      starting: services.filter(s => s.status === 'starting').length
    };

    const byName = {};
    for (const [name, versionList] of this.serviceVersions) {
      byName[name] = versionList.length;
    }

    return {
      totalServices: services.length,
      byStatus,
      byName,
      totalRequests: services.reduce((sum, s) => sum + s.requestCount, 0),
      totalErrors: services.reduce((sum, s) => sum + s.errorCount, 0)
    };
  }

  generateId() {
    return Math.random().toString(36).substring(2, 15);
  }

  shutdown() {
    for (const interval of this.heartbeats.values()) {
      clearInterval(interval);
    }

    for (const interval of this.healthChecks.values()) {
      clearInterval(interval);
    }

    this.heartbeats.clear();
    this.healthChecks.clear();
    this.services.clear();
    this.serviceVersions.clear();

    console.log('Service discovery shut down');
  }
}

const serviceDiscovery = new ServiceDiscovery();

module.exports = serviceDiscovery;
module.exports.ServiceDiscovery = ServiceDiscovery;
