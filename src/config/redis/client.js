const isTestEnv = process.env.NODE_ENV === 'test';

if (isTestEnv) {
  const mockFunctions = {
    get: function () {
      return Promise.resolve(null);
    },
    set: function () {
      return Promise.resolve('OK');
    },
    del: function () {
      return Promise.resolve(1);
    },
    exists: function () {
      return Promise.resolve(0);
    },
    keys: function () {
      return Promise.resolve([]);
    },
    expire: function () {
      return Promise.resolve(1);
    },
    ping: function () {
      return Promise.resolve('PONG');
    },
    sendCommand: function () {
      return Promise.resolve(null);
    }
  };

  module.exports = {
    createClient: function () {
      const mockClient = {
        isOpen: true,
        connect: function () {
          return Promise.resolve();
        },
        disconnect: function () {
          return Promise.resolve();
        }
      };

      for (const key in mockFunctions) {
        if (mockFunctions.hasOwnProperty(key)) {
          mockClient[key] = mockFunctions[key];
        }
      }

      return mockClient;
    },
    default: null
  };
} else {
  try {
    const redis = require('redis');
    module.exports = {
      createClient: redis.createClient,
      default: null
    };
  } catch (error) {
    const mockFunctions = {
      get: function () {
        return Promise.resolve(null);
      },
      set: function () {
        return Promise.resolve('OK');
      },
      del: function () {
        return Promise.resolve(1);
      },
      exists: function () {
        return Promise.resolve(0);
      },
      keys: function () {
        return Promise.resolve([]);
      },
      expire: function () {
        return Promise.resolve(1);
      },
      ping: function () {
        return Promise.resolve('PONG');
      },
      sendCommand: function () {
        return Promise.resolve(null);
      }
    };

    module.exports = {
      createClient: function () {
        const mockClient = {
          isOpen: true,
          connect: function () {
            return Promise.resolve();
          },
          disconnect: function () {
            return Promise.resolve();
          }
        };

        for (const key in mockFunctions) {
          if (mockFunctions.hasOwnProperty(key)) {
            mockClient[key] = mockFunctions[key];
          }
        }

        return mockClient;
      },
      default: null
    };
  }
}
