const { ApolloServer } = require('@apollo/server');
const { expressMiddleware } = require('@apollo/server/express4');
const typeDefs = require('./schema');
const resolvers = require('./resolvers');

const NODE_ENV = process.env.NODE_ENV || 'development';
const isDevelopment = NODE_ENV === 'development';

const server = new ApolloServer({
  typeDefs,
  resolvers,
  introspection: true,
  playground: isDevelopment
    ? {
        settings: {
          'editor.theme': 'dark',
          'editor.fontSize': 14,
          'editor.fontFamily': "'Fira Code', monospace"
        }
      }
    : false,
  formatError: error => {
    if (isDevelopment) {
      console.error('GraphQL Error:', error);
      return error;
    }
    return {
      message: error.message,
      path: error.path
    };
  }
});

const startServer = async () => {
  await server.start();
  console.log('✅ GraphQL Server initialized');
  return server;
};

const getGraphQLHandler = () => {
  return expressMiddleware(server, {
    context: async ({ req }) => ({
      requestId: req.headers['x-request-id'] || `graphql_${Date.now()}`
    })
  });
};

module.exports = {
  server,
  startServer,
  getGraphQLHandler
};
