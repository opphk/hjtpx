class LoadBalancer {
  constructor(options = {}) {
    this.strategy = options.strategy || 'round-robin';
    this.services = new Map();
    this.counters = new Map();
    this.healthThreshold = options.healthThreshold || 0.5;
    this.weights = new Map();
  }

  addService(serviceId, service, weight = 1) {
    this.services.set(serviceId, {
      ...service,
      currentWeight: weight,
      effectiveWeight: weight,
      activeConnections: 0,
      lastUsed: 0,
      requests: 0,
      failures: 0
    });

    this.weights.set(serviceId, weight);
    this.counters.set(serviceId, 0);
  }

  removeService(serviceId) {
    this.services.delete(serviceId);
    this.counters.delete(serviceId);
    this.weights.delete(serviceId);
  }

  updateService(serviceId, updates) {
    const service = this.services.get(serviceId);
    if (service) {
      Object.assign(service, updates);
    }
  }

  getNextService() {
    const activeServices = this.getActiveServices();

    if (activeServices.length === 0) {
      return null;
    }

    switch (this.strategy) {
      case 'round-robin':
        return this.roundRobin(activeServices);

      case 'weighted-round-robin':
        return this.weightedRoundRobin(activeServices);

      case 'least-connections':
        return this.leastConnections(activeServices);

      case 'ip-hash':
        return this.ipHash(activeServices);

      case 'random':
        return this.random(activeServices);

      case 'weighted-random':
        return this.weightedRandom(activeServices);

      case 'health-check':
        return this.healthCheck(activeServices);

      default:
        return this.roundRobin(activeServices);
    }
  }

  getActiveServices() {
    return Array.from(this.services.values()).filter(
      service => service.status === 'healthy'
    );
  }

  roundRobin(services) {
    if (services.length === 1) {
      return services[0];
    }

    let maxCount = -1;
    let selected = services[0];

    for (const service of services) {
      const counter = this.counters.get(service.id) || 0;

      if (counter > maxCount) {
        maxCount = counter;
        selected = service;
      }
    }

    this.counters.set(selected.id, maxCount + 1);
    return selected;
  }

  weightedRoundRobin(services) {
    let totalWeight = 0;

    for (const service of services) {
      service.currentWeight += service.effectiveWeight;
      totalWeight += service.effectiveWeight;
    }

    let maxWeight = 0;
    let selected = services[0];

    for (const service of services) {
      if (service.currentWeight > maxWeight) {
        maxWeight = service.currentWeight;
        selected = service;
      }
    }

    selected.currentWeight -= totalWeight;
    return selected;
  }

  leastConnections(services) {
    let minConnections = Infinity;
    let selected = services[0];

    for (const service of services) {
      if (service.activeConnections < minConnections) {
        minConnections = service.activeConnections;
        selected = service;
      }
    }

    selected.activeConnections++;
    return selected;
  }

  ipHash(services, clientIp = 'default') {
    let hash = 0;

    for (let i = 0; i < clientIp.length; i++) {
      hash = ((hash << 5) - hash) + clientIp.charCodeAt(i);
      hash = hash & hash;
    }

    const index = Math.abs(hash) % services.length;
    return services[index];
  }

  random(services) {
    const index = Math.floor(Math.random() * services.length);
    return services[index];
  }

  weightedRandom(services) {
    const totalWeight = services.reduce((sum, s) => sum + s.effectiveWeight, 0);
    let random = Math.random() * totalWeight;

    for (const service of services) {
      random -= service.effectiveWeight;
      if (random <= 0) {
        return service;
      }
    }

    return services[services.length - 1];
  }

  healthCheck(services) {
    const healthyServices = services.filter(s => {
      const errorRate = s.requests > 0 ? s.failures / s.requests : 0;
      return errorRate < this.healthThreshold;
    });

    if (healthyServices.length === 0) {
      return this.leastConnections(services);
    }

    return this.leastConnections(healthyServices);
  }

  releaseService(serviceId) {
    const service = this.services.get(serviceId);
    if (service && service.activeConnections > 0) {
      service.activeConnections--;
    }
  }

  recordSuccess(serviceId, responseTime) {
    const service = this.services.get(serviceId);
    if (service) {
      service.requests++;
      service.failures = Math.max(0, service.failures - 1);
      service.lastUsed = Date.now();
    }
  }

  recordFailure(serviceId) {
    const service = this.services.get(serviceId);
    if (service) {
      service.requests++;
      service.failures++;
      service.effectiveWeight = Math.max(1, service.effectiveWeight - 1);
    }
  }

  getStats(serviceId) {
    const service = this.services.get(serviceId);
    if (!service) return null;

    const errorRate =
      service.requests > 0
        ? ((service.failures / service.requests) * 100).toFixed(2) + '%'
        : '0%';

    return {
      id: service.id,
      name: service.name,
      url: service.url,
      status: service.status,
      weight: this.weights.get(serviceId),
      effectiveWeight: service.effectiveWeight,
      activeConnections: service.activeConnections,
      requests: service.requests,
      failures: service.failures,
      errorRate,
      lastUsed: service.lastUsed
    };
  }

  getAllStats() {
    const stats = {
      strategy: this.strategy,
      totalServices: this.services.size,
      activeServices: this.getActiveServices().length,
      services: []
    };

    for (const [id] of this.services) {
      stats.services.push(this.getStats(id));
    }

    return stats;
  }

  updateStrategy(newStrategy) {
    const validStrategies = [
      'round-robin',
      'weighted-round-robin',
      'least-connections',
      'ip-hash',
      'random',
      'weighted-random',
      'health-check'
    ];

    if (!validStrategies.includes(newStrategy)) {
      throw new Error(`Invalid strategy: ${newStrategy}`);
    }

    this.strategy = newStrategy;
    console.log(`Load balancer strategy updated to: ${newStrategy}`);
  }

  reset() {
    for (const service of this.services.values()) {
      service.currentWeight = this.weights.get(service.id) || 1;
      service.activeConnections = 0;
      service.requests = 0;
      service.failures = 0;
    }

    for (const [id] of this.counters) {
      this.counters.set(id, 0);
    }
  }
}

const loadBalancer = new LoadBalancer({
  strategy: 'round-robin',
  healthThreshold: 0.3
});

module.exports = loadBalancer;
module.exports.LoadBalancer = LoadBalancer;
