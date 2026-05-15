require('dotenv').config({
  path: process.env.NODE_ENV === 'production' ? '.env.production' :
        process.env.NODE_ENV === 'staging' ? '.env.staging' : '.env'
});

const http = require('http');
const cluster = require('cluster');
const os = require('os');

const compression = require('compression');
const express = require('express');

const productionConfig = require('./backend/config/production');
const { securityHeaders, additionalSecurityHeaders, helmetMiddleware } = require('./backend/middleware/securityHeaders');
const { initSentry, Sentry } = require('./backend/config/sentry');
const { createApolloServer, startApolloServer } = require('./backend/graphql');

const app = express();

const PORT = process.env.PORT || 3000;
const NODE_ENV = process.env.NODE_ENV || 'development';
const isProduction = NODE_ENV === 'production';

initSentry(app);

if (isProduction && productionConfig.production.trustProxy) {
  app.set('trust proxy', 1);
}

const { corsMiddleware } = require('./backend/middleware/cors');
const { cacheStatsMiddleware } = require('./backend/middleware/cacheMiddleware');
const errorHandler = require('./backend/middleware/errorHandler');
const { logger, logError } = require('./backend/middleware/logger');
const { performanceMiddleware } = require('./backend/middleware/performanceMonitor');
const { ipRateLimiter } = require('./backend/middleware/rateLimiter');
const responseFormatter = require('./backend/middleware/responseFormatter');
const { versionNegotiator, deprecationWarning, SUPPORTED_VERSIONS, DEFAULT_VERSION } = require('./backend/middleware/versionControl');
const { apiVersionNegotiator } = require('./backend/middleware/apiVersionNegotiation');
const { deprecationWarning: deprecationWarningNew } = require('./backend/middleware/deprecationWarning');
const { apiStatsMiddleware } = require('./backend/middleware/apiStats');
const v1Routes = require('./backend/routes/v1');
const v2Routes = require('./backend/routes/v2');
const docsRoutes = require('./backend/routes/docs');
const websocketService = require('./backend/services/websocketService');
const cacheService = require('./backend/services/cacheService');

if (process.env.SENTRY_DSN) {
  app.use(Sentry.Handlers.requestHandler());
  app.use(Sentry.Handlers.tracingHandler());
}

app.use((req, res, next) => {
  req.requestId = `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  res.setHeader('X-Request-ID', req.requestId);
  next();
});

if (productionConfig.production.enableCompression) {
  app.use(
    compression({
      level: productionConfig.production.compressionLevel,
      threshold: productionConfig.production.compressionThreshold,
      filter: (req, res) => {
        if (req.headers['x-no-compression']) {
          return false;
        }
        return compression.filter(req, res);
      }
    })
  );
}

if (isProduction && productionConfig.security.enableHelmet) {
  app.use(helmetMiddleware);
}

app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true, limit: '10mb' }));

app.use(securityHeaders);

if (isProduction && productionConfig.security.enableCors) {
  app.use(corsMiddleware);
} else {
  app.use(corsMiddleware);
}

app.use(additionalSecurityHeaders);
app.use(performanceMiddleware);
app.use(logger);
app.use(apiStatsMiddleware);
app.use(responseFormatter);
app.use(cacheStatsMiddleware);
app.use(versionNegotiator);
app.use(deprecationWarning);
app.use(apiVersionNegotiator);
app.use(deprecationWarningNew);

if (productionConfig.security.enableRateLimit) {
  app.use(ipRateLimiter);
}

app.get('/', (req, res) => {
  res.json({
    success: true,
    data: {
      message: 'Welcome to HJTPX API',
      version: '2.0.0',
      status: 'running',
      timestamp: new Date().toISOString(),
      documentation: '/api-docs',
      healthCheck: `/api/${DEFAULT_VERSION}/health`,
      environment: NODE_ENV,
      apiVersions: SUPPORTED_VERSIONS.map(v => ({
        version: v,
        url: `/api/${v}`,
        health: `/api/${v}/health`
      }))
    }
  });
});

app.use('/api/v1', v1Routes);
app.use('/api/v2', v2Routes);
app.use('/api-docs', docsRoutes);

app.use((req, res) => {
  res.notFound(`Route ${req.method} ${req.path} not found`);
});

app.use((err, req, res, next) => {
  logError(err, req, {
    context: 'Global error handler',
    environment: NODE_ENV
  });

  if (process.env.SENTRY_DSN) {
    Sentry.captureException(err, {
      tags: {
        request_id: req.requestId,
        environment: NODE_ENV
      },
      user: req.user ? { id: req.user.id, email: req.user.email } : undefined,
      extra: {
        path: req.path,
        method: req.method,
        query: req.query,
        body: req.body
      }
    });
  }

  if (err.type === 'entity.parse.failed') {
    return res.badRequest('Invalid JSON payload');
  }

  if (err.name === 'ValidationError') {
    return res.badRequest(err.message);
  }

  res.error(
    NODE_ENV === 'production' ? 'Internal server error' : err.message,
    err.statusCode || 500,
    err.code || 'INTERNAL_ERROR',
    NODE_ENV === 'production' ? undefined : { stack: err.stack }
  );
});

if (process.env.SENTRY_DSN) {
  app.use(Sentry.Handlers.errorHandler());
}

app.use(errorHandler);

const createServer = async () => {
  const apolloServer = createApolloServer();
  await startApolloServer(apolloServer, app);

  const server = app.listen(PORT, () => {
    console.log('🚀 HJTPX API Server');
    console.log('========================================');
    console.log(`Environment: ${NODE_ENV}`);
    console.log(`Port: ${PORT}`);
    console.log(`Default API Version: ${DEFAULT_VERSION}`);
    console.log(`Supported API Versions: ${SUPPORTED_VERSIONS.join(', ')}`);
    console.log(`Health Check: http://localhost:${PORT}/api/${DEFAULT_VERSION}/health`);
    console.log(`GraphQL Playground: http://localhost:${PORT}/graphql`);
    console.log('========================================\n');
  });

  if (isProduction && productionConfig.performance.requestTimeout) {
    server.timeout = productionConfig.performance.requestTimeout;
    server.keepAliveTimeout = productionConfig.performance.keepAliveTimeout;
  }

  websocketService.initialize(server);
  console.log('✅ WebSocket service initialized');

  const gracefulShutdown = signal => {
    console.log(`${signal} signal received: closing HTTP server`);
    websocketService.close();
    server.close(() => {
      console.log('HTTP server closed');
      process.exit(0);
    });
  };

  process.on('SIGTERM', () => gracefulShutdown('SIGTERM'));
  process.on('SIGINT', () => gracefulShutdown('SIGINT'));

  process.on('unhandledRejection', (reason, promise) => {
    logError(new Error(reason), null, { context: 'Unhandled Rejection' });
    console.error('Unhandled Rejection at:', promise, 'reason:', reason);
  });

  process.on('uncaughtException', error => {
    logError(error, null, { context: 'Uncaught Exception' });
    console.error('Uncaught Exception:', error);
    process.exit(1);
  });

  return server;
};

if (isProduction && productionConfig.production.enableCluster && cluster.isMaster) {
  const numCPUs = productionConfig.production.clusterWorkers;
  console.log(`🌐 Master process ${process.pid} is running`);
  console.log(`🔧 Spawning ${numCPUs} worker processes...\n`);

  for (let i = 0; i < numCPUs; i++) {
    cluster.fork();
  }

  cluster.on('exit', (worker, code, signal) => {
    console.log(`Worker ${worker.process.pid} died (${signal || code}). Restarting...`);
    cluster.fork();
  });

  cluster.on('online', (worker) => {
    console.log(`Worker ${worker.process.pid} started`);
  });
} else {
  (async () => {
    try {
      await createServer();
    } catch (error) {
      console.error('Failed to start server:', error);
      process.exit(1);
    }
  })();
}

module.exports = app;
