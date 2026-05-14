const os = require('os');

const users = [
  {
    id: '1',
    username: 'admin',
    email: 'admin@hjtpx.example.com',
    role: 'admin',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString()
  },
  {
    id: '2',
    username: 'user1',
    email: 'user1@hjtpx.example.com',
    role: 'user',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString()
  }
];

const resolvers = {
  Query: {
    user: (parent, { id }) => {
      return users.find(user => user.id === id);
    },
    users: (parent, { limit = 10, offset = 0 }) => {
      return users.slice(offset, offset + limit);
    },
    health: () => ({
      status: 'ok',
      timestamp: new Date().toISOString(),
      uptime: process.uptime(),
      environment: process.env.NODE_ENV || 'development',
      version: '1.0.0'
    }),
    detailedHealth: () => {
      const totalMem = os.totalmem();
      const freeMem = os.freemem();
      return {
        status: 'healthy',
        timestamp: new Date().toISOString(),
        services: [
          { name: 'API', status: 'running', version: '1.0.0' },
          { name: 'Database', status: 'connected', version: '1.0.0' },
          { name: 'Cache', status: 'active', version: '1.0.0' }
        ],
        system: {
          nodeVersion: process.version,
          platform: os.platform(),
          environment: process.env.NODE_ENV || 'development',
          totalMemory: totalMem,
          freeMemory: freeMem,
          cpus: os.cpus().length
        },
        database: {
          status: 'connected',
          type: 'PostgreSQL',
          connection: 'active'
        }
      };
    },
    version: () => '1.0.0',
    environment: () => process.env.NODE_ENV || 'development'
  },

  Mutation: {
    createUser: (parent, { input }) => {
      const newUser = {
        id: String(users.length + 1),
        username: input.username,
        email: input.email,
        role: input.role || 'user',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString()
      };
      users.push(newUser);
      return newUser;
    },
    updateUser: (parent, { id, input }) => {
      const userIndex = users.findIndex(user => user.id === id);
      if (userIndex === -1) return null;

      users[userIndex] = {
        ...users[userIndex],
        ...input,
        updatedAt: new Date().toISOString()
      };
      return users[userIndex];
    },
    deleteUser: (parent, { id }) => {
      const userIndex = users.findIndex(user => user.id === id);
      if (userIndex === -1) return false;

      users.splice(userIndex, 1);
      return true;
    }
  }
};

module.exports = resolvers;
