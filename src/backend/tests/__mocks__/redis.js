const mockGet = jest.fn();
const mockSet = jest.fn();
const mockDel = jest.fn();
const mockExists = jest.fn();
const mockKeys = jest.fn();
const mockExpire = jest.fn();
const mockTtl = jest.fn();
const mockPing = jest.fn();
const mockQuit = jest.fn();

const redisClient = {
  get: mockGet,
  set: mockSet,
  del: mockDel,
  exists: mockExists,
  keys: mockKeys,
  expire: mockExpire,
  ttl: mockTtl,
  ping: mockPing,
  quit: mockQuit,
  on: jest.fn(),
  connect: jest.fn().mockResolvedValue({}),
  disconnect: jest.fn().mockResolvedValue({})
};

module.exports = {
  mockGet,
  mockSet,
  mockDel,
  mockExists,
  mockKeys,
  mockExpire,
  mockTtl,
  mockPing,
  mockQuit,
  redisClient
};
