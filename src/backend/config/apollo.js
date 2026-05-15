const { ApolloServer } = require('@apollo/server');
const { expressMiddleware } = require('@apollo/server/express4');
const typeDefs = require('../graphql/schema');
const resolvers = require('../graphql/resolvers');

const isDevelopment = process.env.NODE_ENV !== 'production';
const isPlaygroundEnabled = process.env.GRAPHQL_PLAYGROUND === 'true' || isDevelopment;

const apolloConfig = {
  playground: isPlaygroundEnabled ? {
    settings: {
      'editor.theme': 'dark',
      'editor.reuseHeaders': true,
      'editor.fontSize': 14,
      'editor.fontFamily': "'Fira Code', 'JetBrains Mono', monospace",
      'request.credentials': 'include',
      'schema.polling.enabled': true,
      'schema.polling.interval': 1000,
      'schema.polling.endpointFilter': '*',
      'general.betaUpdates': false,
      'tracing.hideTracingResponse': false,
      'tracing.supportTracingExtension': true,
      'queries': '# Write your query or mutation here\nquery {\n  users {\n    id\n    email\n    name\n    role\n  }\n}',
      'history': true
    },
    tabs: [
      {
        endpoint: process.env.GRAPHQL_ENDPOINT || '/graphql',
        name: 'HJTPX API',
        query: isDevelopment ? '# Welcome to HJTPX GraphQL Playground\n# Write your queries here\n\nquery {\n  users {\n    id\n    email\n    name\n    role\n    created_at\n  }\n}' : ''
      }
    ]
  } : false,

  introspection: isDevelopment || process.env.GRAPHQL_INTROSPECTION === 'true',

  formatError: (error) => {
    const logError = {
      message: error.message,
      code: error.extensions?.code || 'INTERNAL_SERVER_ERROR',
      locations: error.locations,
      path: error.path,
      stack: isDevelopment ? error.stack : undefined
    };

    if (process.env.NODE_ENV !== 'production') {
      console.error('GraphQL Error:', JSON.stringify(logError, null, 2));
    } else {
      console.error('GraphQL Error:', error.message);
    }

    return {
      message: error.message,
      code: error.extensions?.code || 'INTERNAL_SERVER_ERROR',
      locations: error.locations,
      path: error.path
    };
  },

  formatResponse: (response, { context }) => {
    if (context?.req && process.env.GRAPHQL_LOG_RESPONSES === 'true') {
      console.log('GraphQL Response:', JSON.stringify(response, null, 2));
    }
    return response;
  },

  plugins: [
    {
      async requestDidStart(requestContext) {
        const startTime = Date.now();
        
        return {
          async parsingDidStart() {
            return (error) => {
              if (error) {
                console.error('GraphQL Parsing Error:', error.message);
              }
            };
          },
          async validationDidStart() {
            return (error) => {
              if (error) {
                console.error('GraphQL Validation Error:', error);
              }
            };
          },
          async didResolveOperation(requestContext) {
            const duration = Date.now() - startTime;
            console.log(`GraphQL Operation: ${requestContext.operationName || 'Anonymous'} - ${requestContext.operation?.operation || 'unknown'} (${duration}ms)`);
          },
          async didEncounterErrors(requestContext) {
            console.error('GraphQL Errors:', requestContext.errors);
          },
          async willSendResponse(requestContext) {
            const duration = Date.now() - startTime;
            if (duration > 1000) {
              console.warn(`Slow GraphQL Response: ${duration}ms`);
            }
          }
        };
      }
    }
  ],

  tracing: isDevelopment,
  cache: 'bounded'
};

const createApolloServer = () => {
  const server = new ApolloServer({
    ...apolloConfig,
    typeDefs,
    resolvers
  });

  return server;
};

const startApolloServer = async (server, app) => {
  await server.start();

  app.use('/graphql', expressMiddleware(server, {
    context: async ({ req }) => {
      let user = null;
      let startTime = Date.now();

      if (req.headers.authorization) {
        const token = req.headers.authorization.replace('Bearer ', '');
        try {
          const authService = require('../services/authService');
          const decoded = authService.verifyToken(token);
          user = { 
            id: decoded.id, 
            email: decoded.email, 
            role: decoded.role,
            permissions: decoded.permissions || []
          };
        } catch (error) {
          if (isDevelopment) {
            console.warn('GraphQL Auth Error:', error.message);
          }
          user = null;
        }
      }

      const contextTime = Date.now() - startTime;
      if (contextTime > 100) {
        console.warn(`Slow GraphQL Context Resolution: ${contextTime}ms`);
      }

      return { 
        user, 
        req,
        startTime,
        logger: {
          info: (msg) => console.log(`[GraphQL] ${msg}`),
          warn: (msg) => console.warn(`[GraphQL] ${msg}`),
          error: (msg, err) => console.error(`[GraphQL] ${msg}`, err)
        }
      };
    },
    formatError: apolloConfig.formatError
  }));

  if (isPlaygroundEnabled && isDevelopment) {
    app.get('/graphql/health', (req, res) => {
      res.json({ 
        status: 'healthy', 
        playground: 'enabled',
        endpoint: process.env.GRAPHQL_ENDPOINT || '/graphql'
      });
    });
  }

  return server;
};

module.exports = {
  createApolloServer,
  startApolloServer,
  apolloConfig
};
