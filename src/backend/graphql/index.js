const { ApolloServer } = require('@apollo/server');
const { expressMiddleware } = require('@apollo/server/express4');
const typeDefs = require('./schema');
const resolvers = require('./resolvers');

const createApolloServer = (app) => {
  const server = new ApolloServer({
    typeDefs,
    resolvers,
    formatError: (error) => {
      if (process.env.NODE_ENV !== 'production') {
        console.error('GraphQL Error:', error);
      }
      return {
        message: error.message,
        code: error.extensions?.code,
        locations: error.locations,
        path: error.path
      };
    }
  });

  return server;
};

const startApolloServer = async (server, app) => {
  await server.start();

  app.use('/graphql', expressMiddleware(server, {
    context: async ({ req }) => {
      let user = null;
      if (req.headers.authorization) {
        const token = req.headers.authorization.replace('Bearer ', '');
        try {
          const authService = require('../services/authService');
          const decoded = authService.verifyToken(token);
          user = { id: decoded.id, email: decoded.email, role: decoded.role };
        } catch (error) {
          user = null;
        }
      }
      return { user, req };
    }
  }));
};

module.exports = { createApolloServer, startApolloServer };
