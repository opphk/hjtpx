const { ApolloServer } = require('@apollo/server');
const { expressMiddleware } = require('@apollo/server/express4');
const { ApolloServerPluginDrainHttpServer } = require('@apollo/server/plugin/drainHttpServer');
const { makeExecutableSchema } = require('@graphql-tools/schema');
const { WebSocketServer } = require('ws');
const { useServer } = require('graphql-ws/lib/use/ws');
const express = require('express');
const http = require('http');
const cors = require('cors');

const typeDefs = require('./schema');
const resolvers = require('./resolvers');
const { subscriptionResolver, pubsub } = require('./subscriptions');
const { createLoaders } = require('./loaders');

const authService = require('../services/authService');

const createContext = async (contextParams) => {
  let user = null;
  
  if (contextParams.req?.headers?.authorization) {
    const token = contextParams.req.headers.authorization.replace('Bearer ', '');
    try {
      const decoded = authService.verifyToken(token);
      user = { id: decoded.id, email: decoded.email, role: decoded.role };
    } catch (error) {
      user = null;
    }
  }
  
  return {
    user,
    loaders: createLoaders(),
    pubsub,
    req: contextParams.req,
    ...contextParams
  };
};

const createApolloServer = async (app, httpServer) => {
  const schema = makeExecutableSchema({
    typeDefs,
    resolvers: {
      ...resolvers,
      Subscription: subscriptionResolver
    }
  });
  
  const wsServer = new WebSocketServer({
    server: httpServer,
    path: '/graphql'
  });
  
  const serverCleanup = useServer(
    {
      schema,
      context: async (ctx) => {
        return await createContext({ req: ctx.extra.request });
      }
    },
    wsServer
  );
  
  const server = new ApolloServer({
    schema,
    plugins: [
      ApolloServerPluginDrainHttpServer({ httpServer }),
      {
        async serverWillStart() {
          return {
            async drainServer() {
              await serverCleanup.dispose();
            }
          };
        }
      }
    ],
    formatError: (formattedError, error) => {
      console.error('GraphQL Error:', error);
      
      if (error.originalError?.extensions?.code === 'BAD_USER_INPUT') {
        return {
          message: error.message,
          extensions: {
            code: 'BAD_USER_INPUT',
            field: error.path?.[0]
          }
        };
      }
      
      if (error.originalError?.extensions?.code === 'AUTHENTICATION_ERROR') {
        return {
          message: 'Authentication required',
          extensions: {
            code: 'AUTHENTICATION_ERROR'
          }
        };
      }
      
      if (error.originalError?.extensions?.code === 'FORBIDDEN') {
        return {
          message: 'You do not have permission to perform this action',
          extensions: {
            code: 'FORBIDDEN'
          }
        };
      }
      
      return {
        message: formattedError.message,
        locations: formattedError.locations,
        path: formattedError.path
      };
    }
  });
  
  await server.start();
  
  app.use(
    '/graphql',
    cors({
      origin: process.env.NODE_ENV === 'production' 
        ? process.env.ALLOWED_ORIGINS?.split(',') || []
        : '*',
      credentials: true
    }),
    express.json({ limit: '10mb' }),
    expressMiddleware(server, {
      context: async ({ req }) => await createContext({ req })
    })
  );
  
  return server;
};

const setupPlayground = (app) => {
  if (process.env.NODE_ENV !== 'production') {
    app.get('/playground', (req, res) => {
      res.send(`
        <!DOCTYPE html>
        <html>
        <head>
          <title>HJTPX GraphQL Playground</title>
          <link rel="stylesheet" href="https://unpkg.com/graphiql/graphiql.min.css" />
        </head>
        <body style="margin: 0;">
          <div id="graphiql" style="height: 100vh;"></div>
          <script crossorigin src="https://unpkg.com/react/umd/react.production.min.js"></script>
          <script crossorigin src="https://unpkg.com/react-dom/umd/react-dom.production.min.js"></script>
          <script crossorigin src="https://unpkg.com/graphiql/graphiql.min.js"></script>
          <script>
            const fetcher = GraphiQL.createFetcher({
              url: '/graphql',
              wsClient: null
            });
            ReactDOM.render(
              React.createElement(GraphiQL, { fetcher }),
              document.getElementById('graphiql'),
            );
          </script>
        </body>
        </html>
      `);
    });
  }
};

module.exports = { createApolloServer, createContext, setupPlayground };
