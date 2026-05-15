require('dotenv').config({
  path:
    process.env.NODE_ENV === 'production'
      ? '.env.production'
      : process.env.NODE_ENV === 'staging'
        ? '.env.staging'
        : '.env'
});

const http = require('http');
const cluster = require('cluster');
const os = require('os');

const compression = require('compression');
const express = require('express');

const productionConfig = require('./backend/config/production');
const {
  securityHeaders,
  additionalSecurityHeaders
} = require('./backend/middleware/securityHeaders');

const app = express();

const PORT = process.env.PORT || 3000;
const NODE_ENV = process.env.NODE_ENV || 'development';
const isProduction = NODE_ENV === 'production';

if (isProduction && productionConfig.production.trustProxy) {
  app.set('trust proxy', 1);
}

const { corsMiddleware } = require('./backend/middleware/cors');
const errorHandler = require('./backend/middleware/errorHandler');
const { logger, logError } = require('./backend/middleware/logger');
const { performanceMiddleware } = require('./backend/middleware/performanceMonitor');
const { ipRateLimiter } = require('./backend/middleware/rateLimiter');
const responseFormatter = require('./backend/middleware/responseFormatter');
const v1Routes = require('./backend/routes/v1');
const v2Routes = require('./backend/routes/v2');
const docsRoutes = require('./backend/routes/docs');
const { versionNegotiationMiddleware } = require('./backend/middleware/apiVersionNegotiation');
const websocketService = require('./backend/services/websocketService');
const { startServer: startGraphQL, getGraphQLHandler } = require('./graphql/server');

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

app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true, limit: '10mb' }));

if (isProduction && productionConfig.security.enableHelmet) {
  app.use(securityHeaders);
}

if (isProduction && productionConfig.security.enableCors) {
  app.use(corsMiddleware);
} else {
  app.use(corsMiddleware);
}

app.use(additionalSecurityHeaders);
app.use(performanceMiddleware);
app.use(logger);
app.use(responseFormatter);

if (productionConfig.security.enableRateLimit) {
  app.use(ipRateLimiter);
}

app.get('/', (req, res) => {
  res.json({
    success: true,
    data: {
      message: 'Welcome to HJTPX API',
      version: '1.0.0',
      status: 'running',
      timestamp: new Date().toISOString(),
      documentation: '/api/v1',
      healthCheck: '/api/v1/health',
      environment: NODE_ENV
    }
  });
});

app.use('/api/v1', v1Routes);
app.use('/api/v2', v2Routes);
app.use('/api-docs', docsRoutes);

app.use('/graphql', getGraphQLHandler());
if (NODE_ENV === 'development') {
  app.get('/playground', (req, res) => {
    res.send(`
      <!DOCTYPE html>
      <html>
        <head>
          <title>GraphQL Playground</title>
          <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
          <script src="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
        </head>
        <body>
          <div id="root"></div>
          <script>
            window.addEventListener('load', function() {
              GraphQLPlayground.init(document.getElementById('root'), {
                endpoint: '/graphql'
              });
            });
          </script>
        </body>
      </html>
    `);
  });
}

app.use((req, res) => {
  res.notFound(`Route ${req.method} ${req.path} not found`);
});

app.use((err, req, res, next) => {
  logError(err, req, {
    context: 'Global error handler',
    environment: NODE_ENV
  });

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

app.use(errorHandler);

const createServer = async () => {
  await startGraphQL();

  const server = app.listen(PORT, () => {
    console.log('🚀 HJTPX API Server');
    console.log('========================================');
    console.log(`Environment: ${NODE_ENV}`);
    console.log(`Port: ${PORT}`);
    console.log(`API Version: v1`);
    console.log(`Health Check: http://localhost:${PORT}/api/v1/health`);
    console.log(`Detailed Health: http://localhost:${PORT}/api/v1/health/detailed`);
    console.log(`GraphQL Endpoint: http://localhost:${PORT}/graphql`);
    if (NODE_ENV === 'development') {
      console.log(`GraphQL Playground: http://localhost:${PORT}/playground`);
    }
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

  cluster.on('online', worker => {
    console.log(`Worker ${worker.process.pid} started`);
  });
} else {
  (async () => {
    await createServer();
  })();
}

module.exports = app;
