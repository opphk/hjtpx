const mockSendCommand = jest.fn();

const redisClient = {
  on: jest.fn((event, callback) => {
    if (event === 'error') {
      callback(new Error('Redis connection error'));
    }
  }),
  connect: jest.fn().mockResolvedValue({}),
  disconnect: jest.fn().mockResolvedValue({}),
  quit: jest.fn().mockResolvedValue({}),
  sendCommand: mockSendCommand,
  get: jest.fn(),
  set: jest.fn(),
  del: jest.fn(),
  exists: jest.fn(),
  keys: jest.fn(),
  expire: jest.fn(),
  ttl: jest.fn(),
  ping: jest.fn(),
  isOpen: false,
  isReady: true
};

module.exports = redisClient;
module.exports.mockSendCommand = mockSendCommand;
