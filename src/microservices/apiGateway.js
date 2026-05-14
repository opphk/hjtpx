const express = require('express');
const http = require('http');
const { Server } = require('socket.io');
const helmet = require('helmet');
const cors = require('cors');
const compression = require('compression');

class ApiGateway {
  constructor(options = {}) {
    this.app = express();
    this.server = http.createServer(this.app);
    this.io = new Server(this.server, {
      cors: {
        origin: options.corsOrigin || '*',
        methods: ['GET', 'POST']
      }
    });

    this.port = options.port || 3000;
    this.services = new Map();
    this.routes = new Map();
    this.middleware = [];
    this.rateLimits = new Map();

    this.setupDefaultMiddleware();
    this.setupHealthCheck();
    this.setupWebSocket();
  }

  setupDefaultMiddleware() {
    this.app.use(helmet({
      contentSecurityPolicy: false
    }));

    this.app.use(cors({
      origin: '*',
      methods: ['GET', 'POST', 'PUT', 'DELETE', 'PATCH'],
      allowedHeaders: ['Content-Type', 'Authorization', 'X-Request-ID']
    }));

    this.app.use(compression({ level: 6 }));

    this.app.use(express.json({ limit: '10mb' }));

    this.app.use((req, res, next) => {
      req.startTime = Date.now();
      res.on('finish', () => {
        const duration = Date.now() - req.startTime;
        console.log(`${req.method} ${req.path} ${res.statusCode} ${duration}ms`);
      });
      next();
    });

    this.app.use((req, res, next) => {
      req.requestId = `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      res.setHeader('X-Request-ID', req.requestId);
      next();
    });
  }

  setupHealthCheck() {
    this.app.get('/health', (req, res) => {
      res.json({
        status: 'healthy',
        timestamp: new Date().toISOString(),
        services: this.getServiceStatus(),
        uptime: process.uptime()
      });
    });

    this.app.get('/ready', (req, res) => {
      const healthyServices = Array.from(this.services.values())
        .filter(s => s.status === 'healthy');

      if (healthyServices.length === 0) {
        return res.status(503).json({
          status: 'not ready',
          message: 'No services available'
        });
      }

      res.json({
        status: 'ready',
        services: healthyServices.length
      });
    });
  }

  setupWebSocket() {
    this.io.on('connection', (socket) => {
      console.log('Client connected:', socket.id);

      socket.on('service:register', (data) => {
        this.registerService(data);
        socket.emit('service:registered', { success: true });
      });

      socket.on('service:heartbeat', (data) => {
        this.updateServiceHealth(data.serviceId, 'healthy');
      });

      socket.on('disconnect', () => {
        console.log('Client disconnected:', socket.id);
      });
    });
  }

  registerService(serviceConfig) {
    const { serviceId, name, url, healthCheck } = serviceConfig;

    const service = {
      id: serviceId || name,
      name,
      url,
      healthCheck,
      status: 'unknown',
      lastCheck: null,
      requestCount: 0,
      avgResponseTime: 0,
      consecutiveFailures: 0
    };

    this.services.set(name, service);
    console.log(`Service registered: ${name} at ${url}`);

    if (healthCheck) {
      this.startHealthCheck(name);
    }

    return service;
  }

  startHealthCheck(serviceName) {
    const checkInterval = setInterval(async () => {
      const service = this.services.get(serviceName);
      if (!service) {
        clearInterval(checkInterval);
        return;
      }

      try {
        const start = Date.now();
        const response = await fetch(`${service.url}${service.healthCheck}`);
        const duration = Date.now() - start;

        if (response.ok) {
          service.status = 'healthy';
          service.lastCheck = new Date().toISOString();
          service.consecutiveFailures = 0;
          service.avgResponseTime = (service.avgResponseTime + duration) / 2;
        } else {
          service.status = 'unhealthy';
          service.consecutiveFailures++;
        }
      } catch (error) {
        service.status = 'down';
        service.consecutiveFailures++;
        console.error(`Health check failed for ${serviceName}:`, error.message);
      }

      if (service.consecutiveFailures >= 3) {
        this.notifyServiceFailure(service);
      }
    }, 30000);
  }

  updateServiceHealth(serviceId, status) {
    const service = this.services.get(serviceId);
    if (service) {
      service.status = status;
      service.lastCheck = new Date().toISOString();
    }
  }

  notifyServiceFailure(service) {
    console.error(`Service ${service.name} is failing!`);

    this.io.emit('service:failure', {
      service: service.name,
      failures: service.consecutiveFailures,
      timestamp: new Date().toISOString()
    });
  }

  getServiceStatus() {
    const status = {};
    for (const [name, service] of this.services) {
      status[name] = {
        status: service.status,
        url: service.url,
        lastCheck: service.lastCheck,
        avgResponseTime: service.avgResponseTime
      };
    }
    return status;
  }

  route(method, path, serviceName, servicePath) {
    const key = `${method}:${path}`;
    this.routes.set(key, { serviceName, servicePath });
    return this;
  }

  proxyRequest(req, res, serviceName, servicePath) {
    const service = this.services.get(serviceName);

    if (!service || service.status === 'down') {
      return res.status(503).json({
        success: false,
        error: {
          code: 'SERVICE_UNAVAILABLE',
          message: `Service ${serviceName} is not available`
        }
      });
    }

    const targetPath = servicePath || req.path;
    const targetUrl = `${service.url}${targetPath}${req.url.includes('?') ? req.url.substring(req.path.length) : ''}`;

    service.requestCount++;

    const proxyReq = http.request(targetUrl, {
      method: req.method,
      headers: {
        ...req.headers,
        host: new URL(service.url).host
      }
    }, (proxyRes) => {
      res.status(proxyRes.statusCode);
      proxyRes.headers['x-gateway-service'] = serviceName;
      Object.entries(proxyRes.headers).forEach(([key, value]) => {
        res.setHeader(key, value);
      });
      proxyRes.pipe(res);
    });

    proxyReq.on('error', (error) => {
      console.error(`Proxy error for ${serviceName}:`, error.message);
      service.consecutiveFailures++;
      res.status(502).json({
        success: false,
        error: {
          code: 'BAD_GATEWAY',
          message: `Error connecting to ${serviceName}`
        }
      });
    });

    if (['POST', 'PUT', 'PATCH'].includes(req.method)) {
      req.pipe(proxyReq);
    } else {
      proxyReq.end();
    }
  }

  use(middleware) {
    this.middleware.push(middleware);
    return this;
  }

  applyMiddleware() {
    for (const mw of this.middleware) {
      this.app.use(mw);
    }
  }

  createRouteHandler(method, path, serviceName, servicePath) {
    return (req, res) => {
      this.proxyRequest(req, res, serviceName, servicePath);
    };
  }

  async start() {
    this.applyMiddleware();

    for (const [key, route] of this.routes) {
      const [method, path] = key.split(':');
      this.app[method.toLowerCase()](path, this.createRouteHandler(method, path, route.serviceName, route.servicePath));
    }

    this.app.use((req, res) => {
      res.status(404).json({
        success: false,
        error: {
          code: 'NOT_FOUND',
          message: `Route ${req.method} ${req.path} not found`
        }
      });
    });

    return new Promise((resolve) => {
      this.server.listen(this.port, () => {
        console.log(`API Gateway running on port ${this.port}`);
        resolve(this);
      });
    });
  }

  stop() {
    this.server.close();
    this.io.close();
  }
}

module.exports = ApiGateway;
