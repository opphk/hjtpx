const { ApolloServer } = require('apollo-server-express');
const typeDefs = require('./schema');
const resolvers = require('./resolvers');

const createApolloServer = (app) => {
  const server = new ApolloServer({
    typeDefs,
    resolvers,
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
    },
    playground: process.env.NODE_ENV !== 'production' ? {
      endpoint: '/graphql',
      settings: {
        'editor.theme': 'dark',
        'editor.fontSize': 14,
        'editor.reuseHeaders': true,
        'general.betaUpdates': false,
        'request.credentials': 'same-origin',
        'request.headers': {
          'Authorization': 'Bearer <your-token>'
        }
      },
      tabs: [
        {
          endpoint: '/graphql',
          name: 'HJTPX GraphQL',
          query: '# Welcome to HJTPX GraphQL API\n# Try out queries and mutations here'
        }
      ]
    } : false,
    introspection: process.env.NODE_ENV !== 'production',
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

module.exports = { createApolloServer };
